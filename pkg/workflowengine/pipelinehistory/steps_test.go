// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipelinehistory

import (
	"context"
	"errors"
	"testing"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var stepsBaseTime = time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)

func at(event *historypb.HistoryEvent, second int) *historypb.HistoryEvent {
	event.EventTime = timestamppb.New(stepsBaseTime.Add(time.Duration(second) * time.Second))
	return event
}

func secondsAfterBase(second int) time.Time {
	return stepsBaseTime.Add(time.Duration(second) * time.Second)
}

func named(event *historypb.HistoryEvent, typeName string) *historypb.HistoryEvent {
	if attrs := event.GetActivityTaskScheduledEventAttributes(); attrs != nil {
		attrs.ActivityType = &commonpb.ActivityType{Name: typeName}
	}
	if attrs := event.GetStartChildWorkflowExecutionInitiatedEventAttributes(); attrs != nil {
		attrs.WorkflowType = &commonpb.WorkflowType{Name: typeName}
	}
	return event
}

func withChildID(event *historypb.HistoryEvent, workflowID string) *historypb.HistoryEvent {
	event.GetStartChildWorkflowExecutionInitiatedEventAttributes().WorkflowId = workflowID
	return event
}

func activityStartedEvent(id, scheduledID int64, attempt int32) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_ACTIVITY_TASK_STARTED,
		Attributes: &historypb.HistoryEvent_ActivityTaskStartedEventAttributes{
			ActivityTaskStartedEventAttributes: &historypb.ActivityTaskStartedEventAttributes{
				ScheduledEventId: scheduledID,
				Attempt:          attempt,
			},
		},
	}
}

func childStartedEvent(id, initiatedID int64, workflowID, runID string) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_STARTED,
		Attributes: &historypb.HistoryEvent_ChildWorkflowExecutionStartedEventAttributes{
			ChildWorkflowExecutionStartedEventAttributes: &historypb.ChildWorkflowExecutionStartedEventAttributes{
				InitiatedEventId: initiatedID,
				WorkflowExecution: &commonpb.WorkflowExecution{
					WorkflowId: workflowID,
					RunId:      runID,
				},
			},
		},
	}
}

func withChildRun(event *historypb.HistoryEvent, workflowID, runID string) *historypb.HistoryEvent {
	execution := &commonpb.WorkflowExecution{WorkflowId: workflowID, RunId: runID}
	if attrs := event.GetChildWorkflowExecutionCompletedEventAttributes(); attrs != nil {
		attrs.WorkflowExecution = execution
	}
	if attrs := event.GetChildWorkflowExecutionFailedEventAttributes(); attrs != nil {
		attrs.WorkflowExecution = execution
	}
	return event
}

func workflowTerminatedEvent(id int64) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionTerminatedEventAttributes{
			WorkflowExecutionTerminatedEventAttributes: &historypb.WorkflowExecutionTerminatedEventAttributes{},
		},
	}
}

func TestLoadStepsListsStartedMainSteps(t *testing.T) {
	def := &pipelineinternal.WorkflowDefinition{
		Name: "history",
		Steps: []pipelineinternal.StepDefinition{
			{StepSpec: pipelineinternal.StepSpec{ID: "fetch", Use: "http-request"}},
			{StepSpec: pipelineinternal.StepSpec{ID: "check", Use: "conformance-check"}},
			{StepSpec: pipelineinternal.StepSpec{ID: "never", Use: "email"}},
		},
	}
	events := []*historypb.HistoryEvent{
		at(startedEvent(t, def), 0),
		// A setup activity carries no step_id.
		at(named(scheduledEvent(t, 3, map[string]any{"config": map[string]any{}}), "Setup"), 1),
		at(named(scheduledEvent(t, 5, stepConfig("fetch")), "Make an HTTP request"), 2),
		at(activityStartedEvent(6, 5, 2), 4),
		at(activityCompletedEvent(t, 7, 5, workflowengine.ActivityResult{}), 5),
		// A hook step carries a step_id that is not a main step.
		at(named(scheduledEvent(t, 8, stepConfig("hook")), "Make an HTTP request"), 6),
		at(withChildID(named(childInitiatedEvent(t, 10, map[string]any{
			"workflow_input": stepConfig("check"),
		}), "Conformance check"), "wf-1-check"), 7),
		at(childStartedEvent(11, 10, "wf-1-check", "check-run"), 8),
		at(
			withChildRun(
				childFailedEvent(12, 10, errors.New("check failed")),
				"wf-1-check",
				"check-run",
			),
			9,
		),
	}

	steps, err := LoadSteps(context.Background(), mockHistory(events), "wf-1", "run-1")
	require.NoError(t, err)
	require.Equal(t, []Step{
		{
			ID:        "fetch",
			Kind:      StepKindActivity,
			TypeName:  "Make an HTTP request",
			Status:    "Completed",
			StartTime: secondsAfterBase(4),
			EndTime:   secondsAfterBase(5),
			Attempt:   2,
			EventID:   5,
		},
		{
			ID:              "check",
			Kind:            StepKindWorkflow,
			TypeName:        "Conformance check",
			Status:          "Failed",
			StartTime:       secondsAfterBase(7),
			EndTime:         secondsAfterBase(9),
			EventID:         10,
			ChildWorkflowID: "wf-1-check",
			ChildRunID:      "check-run",
			FailureMessage:  "check failed",
		},
	}, steps)
}

func TestLoadStepsListsLinkedStepsBlocks(t *testing.T) {
	def := &pipelineinternal.WorkflowDefinition{
		Name: "history",
		Steps: []pipelineinternal.StepDefinition{
			{StepSpec: pipelineinternal.StepSpec{ID: "Get Offer", Use: "credential-offer"}},
			{
				StepSpec: pipelineinternal.StepSpec{ID: "wallet", Use: "mobile-automation"},
				LinkedTo: "Get Offer",
			},
		},
	}
	blockID := "wf-1-get-offer"
	events := []*historypb.HistoryEvent{
		at(startedEvent(t, def), 0),
		at(withChildID(blockInitiatedEvent(t, 4), blockID), 1),
		at(childStartedEvent(6, 4, blockID, "block-run-1"), 2),
		at(withChildRun(
			childCompletedEvent(t, 9, 4, workflowengine.WorkflowResult{}),
			blockID,
			"block-run-2",
		), 3),
	}

	steps, err := LoadSteps(context.Background(), mockHistory(events), "wf-1", "run-1")
	require.NoError(t, err)
	require.Equal(t, []Step{{
		ID:              "Get Offer",
		Kind:            StepKindBlock,
		TypeName:        pipelineinternal.LinkedStepsWorkflowName,
		Status:          "Completed",
		StartTime:       secondsAfterBase(1),
		EndTime:         secondsAfterBase(3),
		EventID:         4,
		ChildWorkflowID: blockID,
		ChildRunID:      "block-run-2",
		BlockStepIDs:    []string{"Get Offer", "wallet"},
	}}, steps)
}

func TestLoadStepsReadsALinkedStepsBlockRun(t *testing.T) {
	started := &historypb.HistoryEvent{
		EventId:   1,
		EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
			WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
				Input: toPayloads(t, map[string]any{
					"steps": []pipelineinternal.StepDefinition{
						{StepSpec: pipelineinternal.StepSpec{ID: "offer", Use: "credential-offer"}},
						{
							StepSpec: pipelineinternal.StepSpec{ID: "fetch", Use: "http-request"},
							LinkedTo: "offer",
						},
					},
				}),
			},
		},
	}
	events := []*historypb.HistoryEvent{
		started,
		named(scheduledEvent(t, 5, stepConfig("offer")), "Get a credential offer"),
		activityCompletedEvent(t, 7, 5, workflowengine.ActivityResult{}),
		named(scheduledEvent(t, 8, stepConfig("fetch")), "Make an HTTP request"),
		activityFailedEvent(10, 8),
	}

	steps, err := LoadSteps(context.Background(), mockHistory(events), "wf-1", "run-1")
	require.NoError(t, err)
	require.Len(t, steps, 2)
	require.Equal(t, "offer", steps[0].ID)
	require.Equal(t, StepKindActivity, steps[0].Kind)
	require.Equal(t, "Completed", steps[0].Status)
	require.Equal(t, "fetch", steps[1].ID)
	require.Equal(t, "Failed", steps[1].Status)
	require.Equal(t, int64(8), steps[1].EventID)
}

func TestLoadStepsEndsRunningStepsWithTheRun(t *testing.T) {
	def := &pipelineinternal.WorkflowDefinition{
		Name: "history",
		Steps: []pipelineinternal.StepDefinition{
			{StepSpec: pipelineinternal.StepSpec{ID: "fetch", Use: "http-request"}},
		},
	}
	events := []*historypb.HistoryEvent{
		at(startedEvent(t, def), 0),
		at(scheduledEvent(t, 5, stepConfig("fetch")), 1),
		at(workflowTerminatedEvent(6), 30),
	}

	steps, err := LoadSteps(context.Background(), mockHistory(events), "wf-1", "run-1")
	require.NoError(t, err)
	require.Len(t, steps, 1)
	require.Equal(t, "Terminated", steps[0].Status)
	require.Equal(t, secondsAfterBase(30), steps[0].EndTime)
}

func TestLoadStepsWithoutDefinition(t *testing.T) {
	events := []*historypb.HistoryEvent{startedEvent(t, nil)}
	_, err := LoadSteps(context.Background(), mockHistory(events), "wf-1", "run-1")
	require.ErrorIs(t, err, errNoDefinition)
}
