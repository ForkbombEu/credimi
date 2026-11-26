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
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevice"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
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
	case "custom-activity":
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

func activityTimedOutEvent(id, scheduledID int64) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT,
		Attributes: &historypb.HistoryEvent_ActivityTaskTimedOutEventAttributes{
			ActivityTaskTimedOutEventAttributes: &historypb.ActivityTaskTimedOutEventAttributes{
				ScheduledEventId: scheduledID,
			},
		},
	}
}

func activityCanceledEvent(id, scheduledID int64) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_ACTIVITY_TASK_CANCELED,
		Attributes: &historypb.HistoryEvent_ActivityTaskCanceledEventAttributes{
			ActivityTaskCanceledEventAttributes: &historypb.ActivityTaskCanceledEventAttributes{
				ScheduledEventId: scheduledID,
			},
		},
	}
}

const testRunnerNamespace = "runner-r1"

func toPayload(t *testing.T, value any) *commonpb.Payload {
	t.Helper()
	payload, err := temporalcrypto.DataConverter().ToPayload(value)
	require.NoError(t, err)
	return payload
}

func nexusScheduledEvent(
	t *testing.T,
	id int64,
	operation string,
	input any,
) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED,
		Attributes: &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{
			NexusOperationScheduledEventAttributes: &historypb.NexusOperationScheduledEventAttributes{
				Endpoint:  testRunnerNamespace,
				Service:   mobiledevice.ServiceName,
				Operation: operation,
				Input:     toPayload(t, input),
			},
		},
	}
}

// runStepScheduledEvent schedules the Run step operation that runs a mobile step's
// workflow in a runner namespace.
func runStepScheduledEvent(t *testing.T, id int64, stepID string) *historypb.HistoryEvent {
	return nexusScheduledEvent(t, id, mobiledevice.RunStepOperationName, mobiledevice.RunStepInput{
		WorkflowID: "wf-1-" + stepID,
		Pipeline: mobiledevice.PipelineRun{
			Namespace:  "org-ns",
			WorkflowID: "wf-1",
			RunID:      "run-1",
		},
		Input: workflowengine.WorkflowInput{
			Config: map[string]any{workflowengine.StepIDConfigKey: stepID},
		},
	})
}

func nexusStartedEvent(id, scheduledID int64, workflowID, runID string) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_NEXUS_OPERATION_STARTED,
		Links: []*commonpb.Link{{
			Variant: &commonpb.Link_WorkflowEvent_{
				WorkflowEvent: &commonpb.Link_WorkflowEvent{
					Namespace:  testRunnerNamespace,
					WorkflowId: workflowID,
					RunId:      runID,
					Reference: &commonpb.Link_WorkflowEvent_EventRef{
						EventRef: &commonpb.Link_WorkflowEvent_EventReference{
							EventId:   1,
							EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
						},
					},
				},
			},
		}},
		Attributes: &historypb.HistoryEvent_NexusOperationStartedEventAttributes{
			NexusOperationStartedEventAttributes: &historypb.NexusOperationStartedEventAttributes{
				ScheduledEventId: scheduledID,
				OperationToken:   "token-" + workflowID,
			},
		},
	}
}

func nexusCompletedEvent(
	t *testing.T,
	id, scheduledID int64,
	result any,
) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
		Attributes: &historypb.HistoryEvent_NexusOperationCompletedEventAttributes{
			NexusOperationCompletedEventAttributes: &historypb.NexusOperationCompletedEventAttributes{
				ScheduledEventId: scheduledID,
				Result:           toPayload(t, result),
			},
		},
	}
}

// nexusFailedEvent records a failed operation as the server does: a generic Nexus
// operation failure caused by the handler's error.
func nexusFailedEvent(id, scheduledID int64, cause error) *historypb.HistoryEvent {
	fc := temporal.NewDefaultFailureConverter(
		temporal.DefaultFailureConverterOptions{DataConverter: temporalcrypto.DataConverter()},
	)
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_NEXUS_OPERATION_FAILED,
		Attributes: &historypb.HistoryEvent_NexusOperationFailedEventAttributes{
			NexusOperationFailedEventAttributes: &historypb.NexusOperationFailedEventAttributes{
				ScheduledEventId: scheduledID,
				Failure: &failurepb.Failure{
					Message: "nexus operation completed unsuccessfully",
					FailureInfo: &failurepb.Failure_NexusOperationExecutionFailureInfo{
						NexusOperationExecutionFailureInfo: &failurepb.NexusOperationFailureInfo{
							ScheduledEventId: scheduledID,
							Endpoint:         testRunnerNamespace,
							Service:          mobiledevice.ServiceName,
							Operation:        mobiledevice.RunStepOperationName,
						},
					},
					Cause: fc.ErrorToFailure(cause),
				},
			},
		},
	}
}

func nexusTimedOutEvent(id, scheduledID int64) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_NEXUS_OPERATION_TIMED_OUT,
		Attributes: &historypb.HistoryEvent_NexusOperationTimedOutEventAttributes{
			NexusOperationTimedOutEventAttributes: &historypb.NexusOperationTimedOutEventAttributes{
				ScheduledEventId: scheduledID,
			},
		},
	}
}

func nexusCanceledEvent(id, scheduledID int64) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_NEXUS_OPERATION_CANCELED,
		Attributes: &historypb.HistoryEvent_NexusOperationCanceledEventAttributes{
			NexusOperationCanceledEventAttributes: &historypb.NexusOperationCanceledEventAttributes{
				ScheduledEventId: scheduledID,
			},
		},
	}
}

// deviceOperationEvents schedules and completes Prepare and Clean up for a device used by
// stepID; neither is a step.
func deviceOperationEvents(t *testing.T, firstID int64, stepID string) []*historypb.HistoryEvent {
	cfg := workflowengine.WorkflowInput{
		Config: map[string]any{workflowengine.StepIDConfigKey: stepID},
	}
	return []*historypb.HistoryEvent{
		nexusScheduledEvent(
			t,
			firstID,
			mobiledevice.PrepareOperationName,
			mobiledevice.PrepareInput{
				WorkflowID: "wf-1-device-pixel-prepare",
				DeviceID:   "pixel",
				Requests:   []mobiledevice.MobileDeviceRequest{{DeviceID: "pixel", StepID: stepID}},
				Workflow:   cfg,
			},
		),
		nexusStartedEvent(firstID+1, firstID, "wf-1-device-pixel-prepare", "prepare-run"),
		nexusCompletedEvent(t, firstID+2, firstID, mobiledevice.PrepareResult{
			Device: map[string]any{"serial": "s1"},
		}),
		nexusScheduledEvent(
			t,
			firstID+3,
			mobiledevice.CleanUpOperationName,
			mobiledevice.CleanUpInput{
				WorkflowID: "wf-1-device-pixel-cleanup",
				DeviceID:   "pixel",
				Workflow:   cfg,
			},
		),
		nexusFailedEvent(
			firstID+4,
			firstID+3,
			workflowengine.NewAppError(workflowengine.WorkflowError{
				Code:    "CRE201",
				Details: map[string]any{"output": map[string]any{"cleanup": true}},
			}),
		),
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
			step("custom", "custom-activity"),
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
		scheduledEvent(t, 14, stepConfig("custom")),
		activityCompletedEvent(t, 16, 14, workflowengine.ActivityResult{
			Output: "valid",
			Log:    []string{"checked"},
		}),
		runStepScheduledEvent(t, 17, "phone"),
		nexusStartedEvent(18, 17, "wf-1-phone", "phone-run"),
		nexusCompletedEvent(t, 19, 17, workflowengine.WorkflowResult{
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
		"custom": workflowengine.ActivityResult{
			Output: "valid",
			Log:    []string{"checked"},
		},
		"phone": map[string]any{"video": "v.mp4"},
		"child": map[string]any{"inner": map[string]any{"outputs": float64(1)}},
	}, run.Outputs)
	require.Equal(t, map[string]int64{
		"fetch":  7,
		"broken": 10,
		"custom": 16,
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

func TestLoadKeepsTheOutputAFailedTaskWorkflowReports(t *testing.T) {
	def := &pipelineinternal.WorkflowDefinition{
		Name: "history",
		Steps: []pipelineinternal.StepDefinition{
			{StepSpec: pipelineinternal.StepSpec{ID: "engage", Use: "conformance-check"}},
			{StepSpec: pipelineinternal.StepSpec{ID: "silent", Use: "conformance-check"}},
		},
	}
	events := []*historypb.HistoryEvent{
		startedEvent(t, def),
		childInitiatedEvent(t, 5, stepConfig("engage")),
		childFailedEvent(7, 5, workflowengine.NewAppError(workflowengine.WorkflowError{
			Code: "CRE201",
			Details: map[string]any{"output": map[string]any{
				"flow_output": map[string]any{"maestro_screenshot_urls": []any{"not-invoked.png"}},
			}},
		})),
		childInitiatedEvent(t, 8, stepConfig("silent")),
		childFailedEvent(10, 8, workflowengine.NewAppError(workflowengine.WorkflowError{
			Code: "CRE201",
		})),
	}

	run, err := Load(context.Background(), mockHistory(events), "wf-1", "run-1", testOutputKind)
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"engage": map[string]any{
			"flow_output": map[string]any{"maestro_screenshot_urls": []any{"not-invoked.png"}},
		},
		"silent": workflowengine.WorkflowResult{},
	}, run.Outputs)
}

func TestLoadStoresTheZeroResultOfAFailedActivity(t *testing.T) {
	step := func(id string) pipelineinternal.StepDefinition {
		return pipelineinternal.StepDefinition{
			StepSpec: pipelineinternal.StepSpec{ID: id, Use: "http-request"},
		}
	}
	def := &pipelineinternal.WorkflowDefinition{
		Name:  "history",
		Steps: []pipelineinternal.StepDefinition{step("failed"), step("slow"), step("stopped")},
	}
	events := []*historypb.HistoryEvent{
		startedEvent(t, def),
		scheduledEvent(t, 5, stepConfig("failed")),
		activityFailedEvent(7, 5),
		scheduledEvent(t, 8, stepConfig("slow")),
		activityTimedOutEvent(10, 8),
		scheduledEvent(t, 11, stepConfig("stopped")),
		activityCanceledEvent(13, 11),
	}

	run, err := Load(context.Background(), mockHistory(events), "wf-1", "run-1", testOutputKind)
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"failed":  workflowengine.ActivityResult{},
		"slow":    workflowengine.ActivityResult{},
		"stopped": workflowengine.ActivityResult{},
	}, run.Outputs)
	require.Equal(t, map[string]int64{"failed": 7, "slow": 10, "stopped": 13}, run.EventIDs)
}

func TestLoadReadsRunStepOperationFailures(t *testing.T) {
	step := func(id string) pipelineinternal.StepDefinition {
		return pipelineinternal.StepDefinition{
			StepSpec: pipelineinternal.StepSpec{ID: id, Use: "mobile-automation"},
		}
	}
	def := &pipelineinternal.WorkflowDefinition{
		Name: "history",
		Steps: []pipelineinternal.StepDefinition{
			step("engage"), step("silent"), step("slow"), step("stopped"),
		},
	}
	events := []*historypb.HistoryEvent{
		startedEvent(t, def),
		runStepScheduledEvent(t, 5, "engage"),
		nexusFailedEvent(7, 5, workflowengine.NewNonRetryableAppError(
			workflowengine.WorkflowError{
				Code: "CRE201",
				Details: map[string]any{"output": map[string]any{
					"flow_output": map[string]any{"maestro_screenshot_urls": []any{"fail.png"}},
				}},
			},
		)),
		runStepScheduledEvent(t, 8, "silent"),
		nexusFailedEvent(10, 8, workflowengine.NewNonRetryableAppError(
			workflowengine.WorkflowError{Code: "CRE201"},
		)),
		runStepScheduledEvent(t, 11, "slow"),
		nexusTimedOutEvent(13, 11),
		runStepScheduledEvent(t, 14, "stopped"),
		nexusCanceledEvent(16, 14),
	}

	run, err := Load(context.Background(), mockHistory(events), "wf-1", "run-1", testOutputKind)
	require.NoError(t, err)
	require.Equal(t, map[string]any{
		"engage": map[string]any{
			"flow_output": map[string]any{"maestro_screenshot_urls": []any{"fail.png"}},
		},
		"silent":  workflowengine.WorkflowResult{},
		"slow":    workflowengine.WorkflowResult{},
		"stopped": workflowengine.WorkflowResult{},
	}, run.Outputs)
	require.Equal(t, map[string]int64{
		"engage":  7,
		"silent":  10,
		"slow":    13,
		"stopped": 16,
	}, run.EventIDs)
	require.Equal(t, "CRE201", run.Failures["engage"].Code)
	require.Equal(t, "CRE201", run.Failures["silent"].Code)
	require.Equal(t, "canceled or timed out", run.Failures["slow"].Message)
	require.Equal(t, "canceled or timed out", run.Failures["stopped"].Message)
	require.Len(t, run.Failures, 4)
}

func TestLoadIgnoresDeviceOperations(t *testing.T) {
	def := &pipelineinternal.WorkflowDefinition{
		Name: "history",
		Steps: []pipelineinternal.StepDefinition{
			{StepSpec: pipelineinternal.StepSpec{ID: "phone", Use: "mobile-automation"}},
		},
	}
	events := append(
		[]*historypb.HistoryEvent{startedEvent(t, def)},
		deviceOperationEvents(t, 5, "phone")...,
	)

	run, err := Load(context.Background(), mockHistory(events), "wf-1", "run-1", testOutputKind)
	require.NoError(t, err)
	require.Empty(t, run.Outputs)
	require.Empty(t, run.EventIDs)
}

func blockInitiatedEvent(t *testing.T, id int64) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED,
		Attributes: &historypb.HistoryEvent_StartChildWorkflowExecutionInitiatedEventAttributes{
			StartChildWorkflowExecutionInitiatedEventAttributes: &historypb.StartChildWorkflowExecutionInitiatedEventAttributes{
				WorkflowType: &commonpb.WorkflowType{
					Name: pipelineinternal.LinkedStepsWorkflowName,
				},
				// A block input carries pipeline_config, never a step_id config.
				Input: toPayloads(t, map[string]any{
					"pipeline_config": map[string]any{workflowengine.StepIDConfigKey: "offer"},
				}),
			},
		},
	}
}

func TestLoadUnpacksLinkedStepsBlocks(t *testing.T) {
	def := &pipelineinternal.WorkflowDefinition{
		Name: "history",
		Steps: []pipelineinternal.StepDefinition{
			{StepSpec: pipelineinternal.StepSpec{ID: "offer", Use: "credential-offer"}},
			{
				StepSpec: pipelineinternal.StepSpec{ID: "wallet", Use: "mobile-automation"},
				LinkedTo: "offer",
			},
		},
	}
	blockOutput := map[string]any{
		"offer": map[string]any{"outputs": map[string]any{"deeplink": "x"}},
		"wallet": map[string]any{"outputs": map[string]any{
			"flow_output": map[string]any{"maestro_screenshot_urls": []any{"u"}},
		}},
		"unknown": map[string]any{"outputs": "ignored"},
	}
	wantOutputs := map[string]any{
		"offer": map[string]any{"deeplink": "x"},
		"wallet": map[string]any{
			"flow_output": map[string]any{"maestro_screenshot_urls": []any{"u"}},
		},
	}

	t.Run("completed", func(t *testing.T) {
		events := []*historypb.HistoryEvent{
			startedEvent(t, def),
			blockInitiatedEvent(t, 5),
			childCompletedEvent(t, 9, 5, workflowengine.WorkflowResult{Output: blockOutput}),
		}
		run, err := Load(context.Background(), mockHistory(events), "wf-1", "run-1", testOutputKind)
		require.NoError(t, err)
		require.Equal(t, wantOutputs, run.Outputs)
		require.Equal(t, map[string]int64{"offer": 9, "wallet": 9}, run.EventIDs)
	})

	t.Run("failed", func(t *testing.T) {
		events := []*historypb.HistoryEvent{
			startedEvent(t, def),
			blockInitiatedEvent(t, 5),
			childFailedEvent(9, 5, workflowengine.NewAppError(workflowengine.WorkflowError{
				Code: "CRE228",
				Details: map[string]any{
					"output": blockOutput,
					"errors": []any{map[string]any{
						"code":    "CRE315",
						"details": map[string]any{"step_id": "wallet", "stdout": "x... FAILED"},
					}},
				},
			})),
		}
		run, err := Load(context.Background(), mockHistory(events), "wf-1", "run-1", testOutputKind)
		require.NoError(t, err)
		require.Equal(t, wantOutputs, run.Outputs)
		require.Equal(t, map[string]int64{"offer": 9, "wallet": 9}, run.EventIDs)
		// The step the block names failed; the other one completed.
		require.Equal(t, map[string]workflowengine.WorkflowError{"wallet": {
			Code:    "CRE315",
			Details: map[string]any{"step_id": "wallet", "stdout": "x... FAILED"},
		}}, run.Failures)
	})

	t.Run("terminated", func(t *testing.T) {
		events := []*historypb.HistoryEvent{
			startedEvent(t, def),
			blockInitiatedEvent(t, 5),
			{
				EventId:   9,
				EventType: enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_TERMINATED,
				Attributes: &historypb.HistoryEvent_ChildWorkflowExecutionTerminatedEventAttributes{
					ChildWorkflowExecutionTerminatedEventAttributes: &historypb.ChildWorkflowExecutionTerminatedEventAttributes{
						InitiatedEventId: 5,
					},
				},
			},
		}
		run, err := Load(context.Background(), mockHistory(events), "wf-1", "run-1", testOutputKind)
		require.NoError(t, err)
		require.Empty(t, run.Outputs)
		require.Empty(t, run.EventIDs)
	})
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
