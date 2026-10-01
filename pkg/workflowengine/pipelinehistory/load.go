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
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
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
	// recorded its output.
	EventIDs map[string]int64
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
	initiated  map[int64]string
}

// Load reads the history of one pipeline run and rebuilds the outputs of its main steps,
// following the same rules the pipeline workflow uses to fill its in-memory finalOutput.
// Steps are identified by the step_id carried in their activity or child workflow input
// config; hook, finally and internal events are ignored.
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
		},
		steps:     map[string]mainStep{},
		scheduled: map[int64]string{},
		initiated: map[int64]string{},
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
		attrs := event.GetActivityTaskCompletedEventAttributes()
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

	case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
		l.recordActivityFailure(
			event.GetActivityTaskFailedEventAttributes().GetScheduledEventId(),
			eventID,
		)
	case enums.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT:
		l.recordActivityFailure(
			event.GetActivityTaskTimedOutEventAttributes().GetScheduledEventId(),
			eventID,
		)
	case enums.EVENT_TYPE_ACTIVITY_TASK_CANCELED:
		l.recordActivityFailure(
			event.GetActivityTaskCanceledEventAttributes().GetScheduledEventId(),
			eventID,
		)

	case enums.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED:
		var input struct {
			Config        map[string]any `json:"config"`
			WorkflowInput *struct {
				Config map[string]any `json:"config"`
			} `json:"workflow_input"`
		}
		attrs := event.GetStartChildWorkflowExecutionInitiatedEventAttributes()
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
		l.recordChildFailure(
			attrs.GetInitiatedEventId(),
			eventID,
			workflowengine.ExtractOutputFromError(err),
			temporal.IsTimeoutError(err) || temporal.IsCanceledError(err),
		)
	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TERMINATED:
		attrs := event.GetChildWorkflowExecutionTerminatedEventAttributes()
		l.recordChildFailure(attrs.GetInitiatedEventId(), eventID, nil, false)
	case enums.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_FAILED:
		attrs := event.GetStartChildWorkflowExecutionFailedEventAttributes()
		l.recordChildFailure(attrs.GetInitiatedEventId(), eventID, nil, false)
	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TIMED_OUT:
		attrs := event.GetChildWorkflowExecutionTimedOutEventAttributes()
		l.recordChildFailure(attrs.GetInitiatedEventId(), eventID, nil, true)
	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED:
		attrs := event.GetChildWorkflowExecutionCanceledEventAttributes()
		l.recordChildFailure(attrs.GetInitiatedEventId(), eventID, nil, true)
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

// recordActivityFailure mirrors ExecuteStep, which returns a zero ActivityResult with the
// error; the pipeline stores it because it is non-nil.
func (l *loader) recordActivityFailure(scheduledEventID, eventID int64) {
	if id, ok := l.scheduled[scheduledEventID]; ok {
		l.record(id, eventID, workflowengine.ActivityResult{})
	}
}

// recordChildFailure mirrors executeRegularStep for task workflows, which store the zero
// WorkflowResult, and handleChildPipelineStepError for child pipelines, which store the
// failure's Details.output only for continue_on_error steps that neither timed out nor
// were canceled.
func (l *loader) recordChildFailure(
	initiatedEventID, eventID int64,
	failureOutput map[string]any,
	timedOutOrCanceled bool,
) {
	id, ok := l.initiated[initiatedEventID]
	if !ok {
		return
	}
	step := l.steps[id]
	if step.use != childPipelineUse {
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

func (l *loader) record(id string, eventID int64, output any) {
	l.run.Outputs[id] = output
	l.run.EventIDs[id] = eventID
}

func (l *loader) decodeFirst(payloads *commonpb.Payloads, valuePtr any) error {
	if len(payloads.GetPayloads()) == 0 {
		return nil
	}
	return l.dc.FromPayload(payloads.GetPayloads()[0], valuePtr)
}
