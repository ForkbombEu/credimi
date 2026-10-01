// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipelinehistory

import (
	"context"
	"errors"
	"testing"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
)

type fakeHistoryIterator struct {
	events  []*historypb.HistoryEvent
	nextErr error
	index   int
}

func (f *fakeHistoryIterator) HasNext() bool {
	return f.nextErr != nil || f.index < len(f.events)
}

func (f *fakeHistoryIterator) Next() (*historypb.HistoryEvent, error) {
	if f.nextErr != nil {
		err := f.nextErr
		f.nextErr = nil
		return nil, err
	}
	if f.index >= len(f.events) {
		return nil, errors.New("no more events")
	}
	event := f.events[f.index]
	f.index++
	return event, nil
}

func testOutputKind(use string) (workflowengine.OutputKind, bool) {
	switch use {
	case "http-request":
		return workflowengine.OutputMap, true
	case "cesr-validate":
		return workflowengine.OutputAny, true
	}
	return 0, false
}

func toPayloads(t *testing.T, value any) *commonpb.Payloads {
	t.Helper()
	payloads, err := temporalcrypto.DataConverter().ToPayloads(value)
	require.NoError(t, err)
	return payloads
}

func startedEvent(t *testing.T, def *pipelineinternal.WorkflowDefinition) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   1,
		EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
			WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
				Input: toPayloads(t, map[string]any{"workflow_definition": def}),
			},
		},
	}
}

func scheduledEvent(t *testing.T, id int64, input any) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
		Attributes: &historypb.HistoryEvent_ActivityTaskScheduledEventAttributes{
			ActivityTaskScheduledEventAttributes: &historypb.ActivityTaskScheduledEventAttributes{
				Input: toPayloads(t, input),
			},
		},
	}
}

func activityCompletedEvent(
	t *testing.T,
	id, scheduledID int64,
	result workflowengine.ActivityResult,
) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
		Attributes: &historypb.HistoryEvent_ActivityTaskCompletedEventAttributes{
			ActivityTaskCompletedEventAttributes: &historypb.ActivityTaskCompletedEventAttributes{
				ScheduledEventId: scheduledID,
				Result:           toPayloads(t, result),
			},
		},
	}
}

func activityFailedEvent(id, scheduledID int64) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_ACTIVITY_TASK_FAILED,
		Attributes: &historypb.HistoryEvent_ActivityTaskFailedEventAttributes{
			ActivityTaskFailedEventAttributes: &historypb.ActivityTaskFailedEventAttributes{
				ScheduledEventId: scheduledID,
			},
		},
	}
}

func childInitiatedEvent(t *testing.T, id int64, input any) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED,
		Attributes: &historypb.HistoryEvent_StartChildWorkflowExecutionInitiatedEventAttributes{
			StartChildWorkflowExecutionInitiatedEventAttributes: &historypb.StartChildWorkflowExecutionInitiatedEventAttributes{
				Input: toPayloads(t, input),
			},
		},
	}
}

func childCompletedEvent(
	t *testing.T,
	id, initiatedID int64,
	result workflowengine.WorkflowResult,
) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_COMPLETED,
		Attributes: &historypb.HistoryEvent_ChildWorkflowExecutionCompletedEventAttributes{
			ChildWorkflowExecutionCompletedEventAttributes: &historypb.ChildWorkflowExecutionCompletedEventAttributes{
				InitiatedEventId: initiatedID,
				Result:           toPayloads(t, result),
			},
		},
	}
}

func childFailedEvent(id, initiatedID int64, err error) *historypb.HistoryEvent {
	fc := temporal.NewDefaultFailureConverter(
		temporal.DefaultFailureConverterOptions{DataConverter: temporalcrypto.DataConverter()},
	)
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_FAILED,
		Attributes: &historypb.HistoryEvent_ChildWorkflowExecutionFailedEventAttributes{
			ChildWorkflowExecutionFailedEventAttributes: &historypb.ChildWorkflowExecutionFailedEventAttributes{
				InitiatedEventId: initiatedID,
				Failure:          fc.ErrorToFailure(err),
			},
		},
	}
}

func stepConfig(id string) map[string]any {
	return map[string]any{"config": map[string]any{workflowengine.StepIDConfigKey: id}}
}

func mockHistory(events []*historypb.HistoryEvent) *temporalmocks.Client {
	c := &temporalmocks.Client{}
	c.On(
		"GetWorkflowHistory",
		mock.Anything,
		"wf-1",
		"run-1",
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(&fakeHistoryIterator{events: events})
	return c
}

func TestLoadRebuildsMainStepOutputs(t *testing.T) {
	step := func(id, use string) pipelineinternal.StepDefinition {
		return pipelineinternal.StepDefinition{
			StepSpec: pipelineinternal.StepSpec{ID: id, Use: use},
		}
	}
	broken := step("broken", "http-request")
	broken.OnError = []*pipelineinternal.OnErrorStepDefinition{
		{StepSpec: pipelineinternal.StepSpec{ID: "record-error", Use: "mobile-automation"}},
	}
	child := step("child", "child-pipeline")
	child.ContinueOnError = true
	def := &pipelineinternal.WorkflowDefinition{
		Name: "history",
		Steps: []pipelineinternal.StepDefinition{
			step("fetch", "http-request"),
			broken,
			step("cesr", "cesr-validate"),
			step("phone", "mobile-automation"),
			child,
		},
	}
	childErr := workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    "CRE229",
		Summary: "child failed",
		Details: map[string]any{"output": map[string]any{"inner": map[string]any{"outputs": 1}}},
	})

	events := []*historypb.HistoryEvent{
		startedEvent(t, def),
		scheduledEvent(t, 5, stepConfig("fetch")),
		activityCompletedEvent(t, 7, 5, workflowengine.ActivityResult{
			Output: map[string]any{"body": "ok"},
		}),
		scheduledEvent(t, 8, stepConfig("broken")),
		activityFailedEvent(10, 8),
		childInitiatedEvent(t, 11, stepConfig("record-error")),
		childCompletedEvent(t, 13, 11, workflowengine.WorkflowResult{Output: "hook output"}),
		scheduledEvent(t, 14, stepConfig("cesr")),
		activityCompletedEvent(t, 16, 14, workflowengine.ActivityResult{
			Output: "valid",
			Log:    []string{"checked"},
		}),
		childInitiatedEvent(t, 17, stepConfig("phone")),
		childCompletedEvent(t, 19, 17, workflowengine.WorkflowResult{
			Output: map[string]any{"video": "v.mp4"},
		}),
		childInitiatedEvent(t, 20, map[string]any{
			"workflow_definition": map[string]any{"name": "inner"},
			"workflow_input":      stepConfig("child"),
		}),
		childFailedEvent(22, 20, childErr),
		scheduledEvent(t, 23, map[string]any{"config": map[string]any{"app_url": "x"}}),
		activityCompletedEvent(t, 25, 23, workflowengine.ActivityResult{Output: "internal"}),
		scheduledEvent(t, 26, "not an activity input"),
	}

	run, err := Load(context.Background(), mockHistory(events), "wf-1", "run-1", testOutputKind)
	require.NoError(t, err)
	require.Equal(t, "history", run.Definition.Name)
	require.Equal(t, map[string]any{
		"fetch":  map[string]any{"body": "ok"},
		"broken": workflowengine.ActivityResult{},
		"cesr": workflowengine.ActivityResult{
			Output: "valid",
			Log:    []string{"checked"},
		},
		"phone": map[string]any{"video": "v.mp4"},
		"child": map[string]any{"inner": map[string]any{"outputs": float64(1)}},
	}, run.Outputs)
	require.Equal(t, map[string]int64{
		"fetch":  7,
		"broken": 10,
		"cesr":   16,
		"phone":  19,
		"child":  22,
	}, run.EventIDs)
	require.Equal(t, map[string]any{"outputs": map[string]any{"body": "ok"}},
		run.DataContext()["fetch"])
}

func TestLoadSkipsFailedChildPipelineWithoutContinueOnError(t *testing.T) {
	def := &pipelineinternal.WorkflowDefinition{
		Name: "history",
		Steps: []pipelineinternal.StepDefinition{
			{StepSpec: pipelineinternal.StepSpec{ID: "child", Use: "child-pipeline"}},
		},
	}
	events := []*historypb.HistoryEvent{
		startedEvent(t, def),
		childInitiatedEvent(t, 5, map[string]any{"workflow_input": stepConfig("child")}),
		childFailedEvent(7, 5, workflowengine.NewAppError(workflowengine.WorkflowError{
			Code:    "CRE229",
			Details: map[string]any{"output": map[string]any{"inner": 1}},
		})),
	}

	run, err := Load(context.Background(), mockHistory(events), "wf-1", "run-1", testOutputKind)
	require.NoError(t, err)
	require.Empty(t, run.Outputs)
}

func TestLoadErrors(t *testing.T) {
	t.Run("missing definition", func(t *testing.T) {
		events := []*historypb.HistoryEvent{startedEvent(t, nil)}
		_, err := Load(context.Background(), mockHistory(events), "wf-1", "run-1", testOutputKind)
		require.ErrorContains(t, err, "pipeline history has no workflow_definition")
	})
	t.Run("iterator error", func(t *testing.T) {
		c := &temporalmocks.Client{}
		c.On("GetWorkflowHistory", mock.Anything, "wf-1", "run-1", false, mock.Anything).
			Return(&fakeHistoryIterator{nextErr: errors.New("unavailable")})
		_, err := Load(context.Background(), c, "wf-1", "run-1", testOutputKind)
		require.EqualError(t, err, "read pipeline history: unavailable")
	})
}
