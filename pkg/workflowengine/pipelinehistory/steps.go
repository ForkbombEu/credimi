// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipelinehistory

import (
	"context"
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

// StepKind tells how a main step ran in its run.
type StepKind string

const (
	StepKindActivity StepKind = "activity"
	StepKindWorkflow StepKind = "workflow"
	StepKindBlock    StepKind = "block"
)

// Step statuses, equal to the workflow status names the API reports.
const (
	statusRunning    = "Running"
	statusCompleted  = "Completed"
	statusFailed     = "Failed"
	statusCanceled   = "Canceled"
	statusTerminated = "Terminated"
	statusTimedOut   = "TimedOut"
)

// Step is one main step that started in a run, in the order its first event was recorded.
type Step struct {
	ID       string
	Kind     StepKind
	TypeName string // activity or workflow type, e.g. "Make an HTTP request"
	// Status is Running, Completed, Failed, Canceled, Terminated or TimedOut.
	Status    string
	StartTime time.Time
	EndTime   time.Time // zero while running
	// Attempt is the attempt of the last start of an activity; 0 otherwise.
	Attempt int32
	// EventID is the ActivityTaskScheduled or StartChildWorkflowExecutionInitiated event.
	EventID         int64
	ChildWorkflowID string // workflow and block steps
	ChildRunID      string // last run known from this history
	BlockStepIDs    []string
	FailureMessage  string
}

type stepsLoader struct {
	dc         converter.DataConverter
	workflowID string
	mainSteps  map[string]struct{}
	// blocks maps a linked steps block's child workflow ID to its step IDs.
	blocks map[string][]string
	steps  []*Step
	// byEvent indexes steps by their scheduled or initiated event ID.
	byEvent map[int64]*Step
	defined bool
}

// LoadSteps reads the history of one pipeline run, or of one linked steps block run, and
// lists the main steps that started in it. Steps are identified by the step_id carried in
// their activity or child workflow input config, and blocks by their child workflow ID;
// hook, finally and internal events are ignored.
func LoadSteps(ctx context.Context, c client.Client, workflowID, runID string) ([]Step, error) {
	l := &stepsLoader{
		dc:         temporalcrypto.DataConverter(),
		workflowID: workflowID,
		mainSteps:  map[string]struct{}{},
		blocks:     map[string][]string{},
		byEvent:    map[int64]*Step{},
	}
	iter := c.GetWorkflowHistory(
		ctx,
		workflowID,
		runID,
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("read pipeline steps: %w", err)
		}
		if err := l.apply(event); err != nil {
			return nil, fmt.Errorf("read pipeline steps: %w", err)
		}
	}
	if !l.defined {
		return nil, errNoDefinition
	}
	steps := make([]Step, len(l.steps))
	for i, step := range l.steps {
		steps[i] = *step
	}
	return steps, nil
}

func (l *stepsLoader) apply(event *historypb.HistoryEvent) error {
	eventID := event.GetEventId()
	var eventTime time.Time
	if ts := event.GetEventTime(); ts != nil {
		eventTime = ts.AsTime()
	}
	switch event.GetEventType() {
	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:
		return l.applyStarted(event.GetWorkflowExecutionStartedEventAttributes().GetInput())

	case enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
		attrs := event.GetActivityTaskScheduledEventAttributes()
		var input struct {
			Config map[string]any `json:"config"`
		}
		// An input that does not decode as an activity input belongs to no step.
		if l.decodeFirst(attrs.GetInput(), &input) == nil {
			if id, ok := l.mainStepID(input.Config); ok {
				l.add(eventID, &Step{
					ID:        id,
					Kind:      StepKindActivity,
					TypeName:  attrs.GetActivityType().GetName(),
					Status:    statusRunning,
					StartTime: eventTime,
					EventID:   eventID,
				})
			}
		}

	case enums.EVENT_TYPE_ACTIVITY_TASK_STARTED:
		attrs := event.GetActivityTaskStartedEventAttributes()
		if step, ok := l.byEvent[attrs.GetScheduledEventId()]; ok {
			step.Attempt = attrs.GetAttempt()
			step.StartTime = eventTime
		}
	case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
		attrs := event.GetActivityTaskCompletedEventAttributes()
		l.close(attrs.GetScheduledEventId(), statusCompleted, eventTime, nil)
	case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
		attrs := event.GetActivityTaskFailedEventAttributes()
		l.close(attrs.GetScheduledEventId(), statusFailed, eventTime, attrs.GetFailure())
	case enums.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT:
		attrs := event.GetActivityTaskTimedOutEventAttributes()
		l.close(attrs.GetScheduledEventId(), statusTimedOut, eventTime, attrs.GetFailure())
	case enums.EVENT_TYPE_ACTIVITY_TASK_CANCELED:
		attrs := event.GetActivityTaskCanceledEventAttributes()
		l.close(attrs.GetScheduledEventId(), statusCanceled, eventTime, nil)

	case enums.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED:
		l.applyChildInitiated(
			eventID,
			eventTime,
			event.GetStartChildWorkflowExecutionInitiatedEventAttributes(),
		)
	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_STARTED:
		attrs := event.GetChildWorkflowExecutionStartedEventAttributes()
		if step, ok := l.byEvent[attrs.GetInitiatedEventId()]; ok {
			step.ChildRunID = attrs.GetWorkflowExecution().GetRunId()
		}
	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED:
		attrs := event.GetChildWorkflowExecutionCompletedEventAttributes()
		l.closeChild(
			attrs.GetInitiatedEventId(),
			statusCompleted,
			eventTime,
			attrs.GetWorkflowExecution(),
			nil,
		)
	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_FAILED:
		attrs := event.GetChildWorkflowExecutionFailedEventAttributes()
		l.closeChild(
			attrs.GetInitiatedEventId(),
			statusFailed,
			eventTime,
			attrs.GetWorkflowExecution(),
			attrs.GetFailure(),
		)
	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_CANCELED:
		attrs := event.GetChildWorkflowExecutionCanceledEventAttributes()
		l.closeChild(
			attrs.GetInitiatedEventId(),
			statusCanceled,
			eventTime,
			attrs.GetWorkflowExecution(),
			nil,
		)
	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TIMED_OUT:
		attrs := event.GetChildWorkflowExecutionTimedOutEventAttributes()
		l.closeChild(
			attrs.GetInitiatedEventId(),
			statusTimedOut,
			eventTime,
			attrs.GetWorkflowExecution(),
			nil,
		)
	case enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TERMINATED:
		attrs := event.GetChildWorkflowExecutionTerminatedEventAttributes()
		l.closeChild(
			attrs.GetInitiatedEventId(),
			statusTerminated,
			eventTime,
			attrs.GetWorkflowExecution(),
			nil,
		)
	case enums.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_FAILED:
		attrs := event.GetStartChildWorkflowExecutionFailedEventAttributes()
		if step, ok := l.byEvent[attrs.GetInitiatedEventId()]; ok {
			step.Status = statusFailed
			step.EndTime = eventTime
			step.FailureMessage = "start failed: " + attrs.GetCause().String()
		}

	// A closed run leaves no step running: steps it did not close end with the run.
	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED,
		enums.EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW:
		l.closeRunning(statusCompleted, eventTime)
	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
		l.closeRunning(statusFailed, eventTime)
	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_CANCELED:
		l.closeRunning(statusCanceled, eventTime)
	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED:
		l.closeRunning(statusTerminated, eventTime)
	case enums.EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT:
		l.closeRunning(statusTimedOut, eventTime)
	}
	return nil
}

// applyStarted reads the main steps from a pipeline run's workflow_definition, or from the
// steps of a Linked Steps Workflow run.
func (l *stepsLoader) applyStarted(input *commonpb.Payloads) error {
	var start struct {
		WorkflowDefinition *pipelineinternal.WorkflowDefinition `json:"workflow_definition"`
		Steps              []pipelineinternal.StepDefinition    `json:"steps"`
	}
	if err := l.decodeFirst(input, &start); err != nil {
		return fmt.Errorf("decode workflow input: %w", err)
	}
	steps := start.Steps
	if start.WorkflowDefinition != nil {
		if err := pipelineinternal.ApplyFixture(start.WorkflowDefinition); err != nil {
			return err
		}
		steps = start.WorkflowDefinition.Steps
		for _, block := range pipelineinternal.LinkedStepBlocks(steps) {
			ids := make([]string, 0, block[1]-block[0]+1)
			for _, step := range steps[block[0] : block[1]+1] {
				ids = append(ids, step.ID)
			}
			// The child workflow ID executeLinkedSteps gives the block.
			childID := l.workflowID + "-" + canonify.CanonifyPlain(steps[block[0]].ID)
			l.blocks[childID] = ids
		}
	} else if start.Steps == nil {
		return errNoDefinition
	}
	l.defined = true
	for _, step := range steps {
		if step.ID != "" {
			l.mainSteps[step.ID] = struct{}{}
		}
	}
	return nil
}

func (l *stepsLoader) applyChildInitiated(
	eventID int64,
	eventTime time.Time,
	attrs *historypb.StartChildWorkflowExecutionInitiatedEventAttributes,
) {
	typeName := attrs.GetWorkflowType().GetName()
	if typeName == pipelineinternal.LinkedStepsWorkflowName {
		ids, ok := l.blocks[attrs.GetWorkflowId()]
		if !ok {
			return
		}
		l.add(eventID, &Step{
			ID:              ids[0],
			Kind:            StepKindBlock,
			TypeName:        typeName,
			Status:          statusRunning,
			StartTime:       eventTime,
			EventID:         eventID,
			ChildWorkflowID: attrs.GetWorkflowId(),
			BlockStepIDs:    ids,
		})
		return
	}
	var input struct {
		Config        map[string]any `json:"config"`
		WorkflowInput *struct {
			Config map[string]any `json:"config"`
		} `json:"workflow_input"`
	}
	// An input that does not decode as a workflow input belongs to no step.
	if l.decodeFirst(attrs.GetInput(), &input) != nil {
		return
	}
	cfg := input.Config
	if input.WorkflowInput != nil && input.WorkflowInput.Config != nil {
		cfg = input.WorkflowInput.Config
	}
	if id, ok := l.mainStepID(cfg); ok {
		l.add(eventID, &Step{
			ID:              id,
			Kind:            StepKindWorkflow,
			TypeName:        typeName,
			Status:          statusRunning,
			StartTime:       eventTime,
			EventID:         eventID,
			ChildWorkflowID: attrs.GetWorkflowId(),
		})
	}
}

func (l *stepsLoader) add(eventID int64, step *Step) {
	l.steps = append(l.steps, step)
	l.byEvent[eventID] = step
}

func (l *stepsLoader) close(
	eventID int64,
	status string,
	eventTime time.Time,
	failure *failurepb.Failure,
) *Step {
	step, ok := l.byEvent[eventID]
	if !ok {
		return nil
	}
	step.Status = status
	step.EndTime = eventTime
	step.FailureMessage = workflowengine.WorkflowFailureMessageFromHistory(failure)
	return step
}

func (l *stepsLoader) closeChild(
	initiatedEventID int64,
	status string,
	eventTime time.Time,
	execution *commonpb.WorkflowExecution,
	failure *failurepb.Failure,
) {
	if step := l.close(initiatedEventID, status, eventTime, failure); step != nil {
		if runID := execution.GetRunId(); runID != "" {
			step.ChildRunID = runID
		}
	}
}

func (l *stepsLoader) closeRunning(status string, eventTime time.Time) {
	for _, step := range l.steps {
		if step.Status == statusRunning {
			step.Status = status
			step.EndTime = eventTime
		}
	}
}

func (l *stepsLoader) mainStepID(cfg map[string]any) (string, bool) {
	id, ok := cfg[workflowengine.StepIDConfigKey].(string)
	if !ok {
		return "", false
	}
	_, isMain := l.mainSteps[id]
	return id, isMain
}

func (l *stepsLoader) decodeFirst(payloads *commonpb.Payloads, valuePtr any) error {
	if len(payloads.GetPayloads()) == 0 {
		return nil
	}
	return l.dc.FromPayload(payloads.GetPayloads()[0], valuePtr)
}
