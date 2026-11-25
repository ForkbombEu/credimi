// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package pipelinehistory rebuilds a pipeline run's step outputs from its Temporal history,
// so activities can reference step outputs instead of receiving copies of them.
package pipelinehistory

import (
	"context"
	"errors"
	"fmt"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevice"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
)

const (
	childPipelineUse = "child-pipeline"
	heartbeatEvery   = 1000
)

var errNoDefinition = errors.New("pipeline history has no workflow_definition")

// OutputKindFunc reports the registry OutputKind of a step-usable activity `use`.
type OutputKindFunc func(use string) (workflowengine.OutputKind, bool)

// Run is a pipeline run's definition and main step outputs as recorded in its history.
type Run struct {
	// Definition is the workflow_definition from the WorkflowExecutionStarted input,
	// with fixtures applied as the workflow applies them.
	Definition *pipelineinternal.WorkflowDefinition
	// Outputs maps a main step ID to the value the workflow stores as
	// finalOutput[id]["outputs"].
	Outputs map[string]any
	// EventIDs maps a main step ID to the ID of the completion or failure event that
	// recorded its output. Steps run inside a linked block are recorded at the block's
	// completion or failure event.
	EventIDs map[string]int64
	// Failures maps a main step ID to the failure that ended it. A step with an
	// output and no failure completed; a step of a failed linked block records
	// the block's failure only when the block names it (details.errors[0]).
	Failures map[string]workflowengine.WorkflowError
}

// DataContext returns {stepID: {"outputs": value}}, the shape pipeline expressions resolve
// against.
func (r *Run) DataContext() map[string]any {
	dataCtx := make(map[string]any, len(r.Outputs))
	for id, value := range r.Outputs {
		dataCtx[id] = map[string]any{"outputs": value}
	}
	return dataCtx
}

type mainStep struct {
	use             string
	continueOnError bool
}

type loader struct {
	dc         converter.DataConverter
	fc         converter.FailureConverter
	outputKind OutputKindFunc
	run        *Run
	steps      map[string]mainStep
	scheduled  map[int64]string
	// nexusSteps holds the scheduled events of the Run step operations of
	// main mobile steps, whose workflow ran in a runner namespace.
	nexusSteps map[int64]string
	initiated  map[int64]string
	blocks     map[int64]struct{}
}

// Load reads the history of one pipeline run and rebuilds the outputs of its main steps,
// following the same rules the pipeline workflow uses to fill its in-memory finalOutput.
// Steps are identified by the step_id carried in their activity or child workflow input
// config; hook, finally and internal events are ignored. A mobile step's output is the
// result of its workflow, read from the Nexus operation that ran it in a runner namespace. Steps
// run inside a linked block are recorded at the block's completion or failure event, from
// the block's output map.
func Load(
	ctx context.Context,
	c client.Client,
	workflowID, runID string,
	outputKind OutputKindFunc,
) (*Run, error) {
	dc := temporalcrypto.DataConverter()
	l := &loader{
		dc: dc,
		fc: temporal.NewDefaultFailureConverter(
			temporal.DefaultFailureConverterOptions{DataConverter: dc},
		),
		outputKind: outputKind,
		run: &Run{
			Outputs:  map[string]any{},
			EventIDs: map[string]int64{},
			Failures: map[string]workflowengine.WorkflowError{},
		},
		steps:      map[string]mainStep{},
		scheduled:  map[int64]string{},
		nexusSteps: map[int64]string{},
		initiated:  map[int64]string{},
		blocks:     map[int64]struct{}{},
	}

	iter := c.GetWorkflowHistory(
		ctx,
		workflowID,
		runID,
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	events := 0
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("read pipeline history: %w", err)
		}
		events++
		if events%heartbeatEvery == 0 && activity.IsActivity(ctx) {
			activity.RecordHeartbeat(ctx, events)
		}
		if err := l.apply(event); err != nil {
			return nil, fmt.Errorf("read pipeline history: %w", err)
		}
	}
	if l.run.Definition == nil {
		return nil, errNoDefinition
	}
	return l.run, nil
}

func (l *loader) apply(event *historypb.HistoryEvent) error {
	eventID := event.GetEventId()
	switch event.GetEventType() {
	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:
		return l.applyStarted(event.GetWorkflowExecutionStartedEventAttributes().GetInput())

	case enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
		var input struct {
			Config map[string]any `json:"config"`
		}
		// An input that does not decode as an activity input belongs to no step.
		if l.decodeFirst(
			event.GetActivityTaskScheduledEventAttributes().GetInput(),
			&input,
		) == nil {
			if id, ok := l.mainStepID(input.Config); ok {
				l.scheduled[eventID] = id
			}
		}

	case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
		return l.applyActivityCompleted(eventID, event.GetActivityTaskCompletedEventAttributes())
	case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
		attrs := event.GetActivityTaskFailedEventAttributes()
		l.recordActivityFailure(attrs.GetScheduledEventId(), eventID, attrs.GetFailure())
	case enums.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT:
		attrs := event.GetActivityTaskTimedOutEventAttributes()
		l.recordActivityFailure(attrs.GetScheduledEventId(), eventID, attrs.GetFailure())
	case enums.EVENT_TYPE_ACTIVITY_TASK_CANCELED:
		attrs := event.GetActivityTaskCanceledEventAttributes()
		l.recordActivityFailure(attrs.GetScheduledEventId(), eventID, nil)

	case enums.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED,
		enums.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
		enums.EVENT_TYPE_NEXUS_OPERATION_FAILED,
		enums.EVENT_TYPE_NEXUS_OPERATION_CANCELED,
		enums.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT:
		return l.applyNexus(event)

	case enums.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED:
		var input struct {
			Config        map[string]any `json:"config"`
			WorkflowInput *struct {
				Config map[string]any `json:"config"`
			} `json:"workflow_input"`
		}
		attrs := event.GetStartChildWorkflowExecutionInitiatedEventAttributes()
		if attrs.GetWorkflowType().GetName() == pipelineinternal.LinkedStepsWorkflowName {
			l.blocks[eventID] = struct{}{}
			return nil
		}
		// An input that does not decode as a workflow input belongs to no step.
		if l.decodeFirst(attrs.GetInput(), &input) == nil {
			cfg := input.Config
			if input.WorkflowInput != nil && input.WorkflowInput.Config != nil {
				cfg = input.WorkflowInput.Config
			}
			if id, ok := l.mainStepID(cfg); ok {
				l.initiated[eventID] = id
			}
		}

	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED:
		attrs := event.GetChildWorkflowExecutionCompletedEventAttributes()
		if l.isBlock(attrs.GetInitiatedEventId()) {
			var result workflowengine.WorkflowResult
			if err := l.decodeFirst(attrs.GetResult(), &result); err != nil {
				return fmt.Errorf("decode result of linked steps block: %w", err)
			}
			l.recordBlock(eventID, workflowengine.AsMap(result.Output))
			return nil
		}
		id, ok := l.initiated[attrs.GetInitiatedEventId()]
		if !ok {
			return nil
		}
		var result workflowengine.WorkflowResult
		if err := l.decodeFirst(attrs.GetResult(), &result); err != nil {
			return fmt.Errorf("decode result of step %s: %w", id, err)
		}
		l.record(id, eventID, result.Output)

	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_FAILED:
		attrs := event.GetChildWorkflowExecutionFailedEventAttributes()
		err := l.fc.FailureToError(attrs.GetFailure())
		if l.isBlock(attrs.GetInitiatedEventId()) {
			l.recordBlock(eventID, workflowengine.ExtractOutputFromError(err))
			l.recordBlockFailure(err)
			return nil
		}
		l.recordChildFailure(
			attrs.GetInitiatedEventId(),
			eventID,
			workflowengine.ExtractOutputFromError(err),
			temporal.IsTimeoutError(err) || temporal.IsCanceledError(err),
			stepFailure(err, "failed"),
		)
	// Blocks are never in l.initiated, so a terminated, unstarted, timed out or canceled
	// block records nothing.
	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TERMINATED:
		attrs := event.GetChildWorkflowExecutionTerminatedEventAttributes()
		l.recordChildFailure(
			attrs.GetInitiatedEventId(), eventID, nil, false, stepFailure(nil, "terminated"),
		)
	case enums.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_FAILED:
		attrs := event.GetStartChildWorkflowExecutionFailedEventAttributes()
		l.recordChildFailure(
			attrs.GetInitiatedEventId(), eventID, nil, false, stepFailure(nil, "did not start"),
		)
	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TIMED_OUT:
		attrs := event.GetChildWorkflowExecutionTimedOutEventAttributes()
		l.recordChildFailure(
			attrs.GetInitiatedEventId(), eventID, nil, true, stepFailure(nil, "timed out"),
		)
	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED:
		attrs := event.GetChildWorkflowExecutionCanceledEventAttributes()
		l.recordChildFailure(
			attrs.GetInitiatedEventId(), eventID, nil, true, stepFailure(nil, "canceled"),
		)
	}
	return nil
}

// applyNexus records the outputs of main mobile steps from their Run step operations.
func (l *loader) applyNexus(event *historypb.HistoryEvent) error {
	eventID := event.GetEventId()
	switch event.GetEventType() {
	case enums.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED:
		attrs := event.GetNexusOperationScheduledEventAttributes()
		if attrs.GetOperation() != mobiledevice.RunStepOperationName {
			return nil
		}
		var input mobiledevice.RunStepInput
		// An input that does not decode as a step input belongs to no step.
		if attrs.GetInput() != nil && l.dc.FromPayload(attrs.GetInput(), &input) == nil {
			if id, ok := l.mainStepID(input.Input.Config); ok {
				l.nexusSteps[eventID] = id
			}
		}
	case enums.EVENT_TYPE_NEXUS_OPERATION_COMPLETED:
		attrs := event.GetNexusOperationCompletedEventAttributes()
		id, ok := l.nexusSteps[attrs.GetScheduledEventId()]
		if !ok {
			return nil
		}
		var result workflowengine.WorkflowResult
		if attrs.GetResult() != nil {
			if err := l.dc.FromPayload(attrs.GetResult(), &result); err != nil {
				return fmt.Errorf("decode result of step %s: %w", id, err)
			}
		}
		l.record(id, eventID, result.Output)
	case enums.EVENT_TYPE_NEXUS_OPERATION_FAILED:
		attrs := event.GetNexusOperationFailedEventAttributes()
		l.recordNexusFailure(attrs.GetScheduledEventId(), eventID, attrs.GetFailure())
	case enums.EVENT_TYPE_NEXUS_OPERATION_CANCELED:
		attrs := event.GetNexusOperationCanceledEventAttributes()
		l.recordNexusFailure(attrs.GetScheduledEventId(), eventID, nil)
	case enums.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT:
		attrs := event.GetNexusOperationTimedOutEventAttributes()
		l.recordNexusFailure(attrs.GetScheduledEventId(), eventID, nil)
	}
	return nil
}

func (l *loader) applyStarted(input *commonpb.Payloads) error {
	var start struct {
		WorkflowDefinition *pipelineinternal.WorkflowDefinition `json:"workflow_definition"`
	}
	if err := l.decodeFirst(input, &start); err != nil {
		return fmt.Errorf("decode workflow input: %w", err)
	}
	if start.WorkflowDefinition == nil {
		return errNoDefinition
	}
	if err := pipelineinternal.ApplyFixture(start.WorkflowDefinition); err != nil {
		return err
	}
	l.run.Definition = start.WorkflowDefinition
	for _, step := range start.WorkflowDefinition.Steps {
		if step.ID == "" {
			continue
		}
		l.steps[step.ID] = mainStep{use: step.Use, continueOnError: step.ContinueOnError}
	}
	return nil
}

func (l *loader) mainStepID(cfg map[string]any) (string, bool) {
	id, ok := cfg[workflowengine.StepIDConfigKey].(string)
	if !ok {
		return "", false
	}
	_, isMain := l.steps[id]
	return id, isMain
}

func (l *loader) applyActivityCompleted(
	eventID int64,
	attrs *historypb.ActivityTaskCompletedEventAttributes,
) error {
	id, ok := l.scheduled[attrs.GetScheduledEventId()]
	if !ok {
		return nil
	}
	var result workflowengine.ActivityResult
	if err := l.decodeFirst(attrs.GetResult(), &result); err != nil {
		return fmt.Errorf("decode result of step %s: %w", id, err)
	}
	kind, known := l.outputKind(l.steps[id].use)
	if !known {
		kind = workflowengine.OutputMap
	}
	l.record(id, eventID, workflowengine.StepOutputFromActivityResult(kind, result))
	return nil
}

// recordActivityFailure mirrors ExecuteStep: an activity step stores the zero
// ActivityResult returned with the error, because it is non-nil.
func (l *loader) recordActivityFailure(
	scheduledEventID, eventID int64,
	failure *failurepb.Failure,
) {
	id, ok := l.scheduled[scheduledEventID]
	if !ok {
		return
	}
	l.record(id, eventID, workflowengine.ActivityResult{})
	var err error
	if failure != nil {
		err = l.fc.FailureToError(failure)
	}
	l.run.Failures[id] = stepFailure(err, "canceled")
}

// recordNexusFailure mirrors runMobileStep: a mobile step stores the
// failure's Details.output when it carries one, and the zero WorkflowResult
// otherwise, also when it timed out or was canceled (nil failure).
func (l *loader) recordNexusFailure(
	scheduledEventID, eventID int64,
	failure *failurepb.Failure,
) {
	id, ok := l.nexusSteps[scheduledEventID]
	if !ok {
		return
	}
	var err error
	if failure != nil {
		err = l.fc.FailureToError(failure)
	}
	if output := workflowengine.ExtractOutputFromError(err); output != nil {
		l.record(id, eventID, output)
	} else {
		l.record(id, eventID, workflowengine.WorkflowResult{})
	}
	l.run.Failures[id] = stepFailure(err, "canceled or timed out")
}

// recordChildFailure mirrors executeRegularStep for task workflows, which store the failure's
// Details.output when it carries one and the zero WorkflowResult otherwise, and
// handleChildPipelineStepError for child pipelines, which store the failure's Details.output
// only for continue_on_error steps that neither timed out nor were canceled.
func (l *loader) recordChildFailure(
	initiatedEventID, eventID int64,
	failureOutput map[string]any,
	timedOutOrCanceled bool,
	failure workflowengine.WorkflowError,
) {
	id, ok := l.initiated[initiatedEventID]
	if !ok {
		return
	}
	defer func() { l.run.Failures[id] = failure }()
	step := l.steps[id]
	if step.use != childPipelineUse {
		if failureOutput != nil {
			l.record(id, eventID, failureOutput)
			return
		}
		l.record(id, eventID, workflowengine.WorkflowResult{})
		return
	}
	if !step.continueOnError || timedOutOrCanceled {
		return
	}
	var output any
	if failureOutput != nil {
		output = failureOutput
	}
	l.record(id, eventID, output)
}

// stepFailure is the structured failure of err, or a failure with only
// fallback as its message when err carries none.
func stepFailure(err error, fallback string) workflowengine.WorkflowError {
	if failure := workflowengine.ParseWorkflowError(err); failure.Code != "" ||
		failure.Summary != "" || failure.Message != "" {
		return failure
	}
	if err != nil {
		return workflowengine.WorkflowError{Message: err.Error()}
	}
	return workflowengine.WorkflowError{Message: fallback}
}

// recordBlockFailure records the failure of the block step that stopped a
// failed linked block: the first step failure of the block failure's
// details.errors (workflowengine.StepFailures).
func (l *loader) recordBlockFailure(err error) {
	failures := workflowengine.StepFailures(workflowengine.ParseWorkflowError(err))
	if len(failures) == 0 {
		return
	}
	stepID, _ := failures[0].Details["step_id"].(string)
	if _, main := l.steps[stepID]; main {
		l.run.Failures[stepID] = failures[0]
	}
}

// isBlock reports whether a child initiated event started a Linked Steps Workflow.
func (l *loader) isBlock(initiatedEventID int64) bool {
	_, ok := l.blocks[initiatedEventID]
	return ok
}

// recordBlock unpacks a Linked Steps Workflow output, {stepID: {"outputs": value}}, into the
// outputs of the main steps it ran.
func (l *loader) recordBlock(eventID int64, output map[string]any) {
	for id, entry := range output {
		if _, ok := l.steps[id]; !ok {
			continue
		}
		l.record(id, eventID, workflowengine.AsMap(entry)["outputs"])
	}
}

func (l *loader) record(id string, eventID int64, output any) {
	l.run.Outputs[id] = output
	l.run.EventIDs[id] = eventID
	delete(l.run.Failures, id)
}

func (l *loader) decodeFirst(payloads *commonpb.Payloads, valuePtr any) error {
	if len(payloads.GetPayloads()) == 0 {
		return nil
	}
	return l.dc.FromPayload(payloads.GetPayloads()[0], valuePtr)
}
