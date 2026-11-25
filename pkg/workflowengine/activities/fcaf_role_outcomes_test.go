// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"encoding/json"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/aggregate"
	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevice"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
)

const (
	outcomeWallet   = "acme/wallet/"
	outcomeScenario = "fcaf-wallet-solution-relying-party-engagement-haip-vp.yaml"
)

func outcomePayload(t *testing.T, value any) *commonpb.Payload {
	t.Helper()
	payload, err := temporalcrypto.DataConverter().ToPayload(value)
	require.NoError(t, err)
	return payload
}

func outcomeStepScheduled(t *testing.T, id int64, stepID string) *historypb.HistoryEvent {
	t.Helper()
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED,
		Attributes: &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{
			NexusOperationScheduledEventAttributes: &historypb.NexusOperationScheduledEventAttributes{
				Endpoint:  "runner-test",
				Service:   mobiledevice.ServiceName,
				Operation: mobiledevice.RunStepOperationName,
				Input: outcomePayload(t, mobiledevice.RunStepInput{
					WorkflowID: fcafTestWorkflowID + "-" + stepID,
					Input: workflowengine.WorkflowInput{
						Config: map[string]any{workflowengine.StepIDConfigKey: stepID},
					},
				}),
			},
		},
	}
}

func outcomeStepCompleted(t *testing.T, id, scheduledID int64) *historypb.HistoryEvent {
	t.Helper()
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
		Attributes: &historypb.HistoryEvent_NexusOperationCompletedEventAttributes{
			NexusOperationCompletedEventAttributes: &historypb.NexusOperationCompletedEventAttributes{
				ScheduledEventId: scheduledID,
				Result:           outcomePayload(t, workflowengine.WorkflowResult{}),
			},
		},
	}
}

func outcomeFailureConverter() converter.FailureConverter {
	return temporal.NewDefaultFailureConverter(
		temporal.DefaultFailureConverterOptions{DataConverter: temporalcrypto.DataConverter()},
	)
}

func outcomeStepFailed(id, scheduledID int64, cause error) *historypb.HistoryEvent {
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
							Service:          mobiledevice.ServiceName,
							Operation:        mobiledevice.RunStepOperationName,
						},
					},
					Cause: outcomeFailureConverter().ErrorToFailure(cause),
				},
			},
		},
	}
}

// outcomeHistory is a wallet pipeline run: the prelude completed, a
// composed flow failed (CRE315) in its second action, and a linked block
// failed at its producer, before its wallet step ran.
func outcomeHistory(t *testing.T, pidSDJWT string) []*historypb.HistoryEvent {
	t.Helper()
	prefix := aggregate.ScenarioPrefix(DefaultFCAFValidationSuite, outcomeScenario)
	mobile := func(id, linkedTo string, with map[string]any) pipelineinternal.StepDefinition {
		return pipelineinternal.StepDefinition{
			StepSpec: pipelineinternal.StepSpec{
				ID:   id,
				Use:  "mobile-automation",
				With: pipelineinternal.StepInputs{Payload: with},
			},
			LinkedTo: linkedTo,
		}
	}
	def := &pipelineinternal.WorkflowDefinition{
		Name: "fcaf",
		Steps: []pipelineinternal.StepDefinition{
			mobile(aggregate.PreludeStepID, "", map[string]any{
				"action_id":  outcomeWallet + "fcaf-reset-and-onboard",
				"version_id": outcomeWallet + "1-0",
			}),
			{StepSpec: pipelineinternal.StepSpec{ID: "obtain", Use: "http-request"}},
			mobile(prefix+"-present", "", map[string]any{"actions": []any{
				map[string]any{"id": outcomeWallet + "fcaf-issue-pid"},
				map[string]any{"id": outcomeWallet + "fcaf-engagement-haip-vp"},
			}}),
			{StepSpec: pipelineinternal.StepSpec{ID: prefix + "-create", Use: "http-request"}},
			mobile(prefix+"-again", prefix+"-create", map[string]any{
				"action_id": outcomeWallet + "fcaf-expect-request-rejected",
			}),
		},
	}
	evidence := fcafHistory(t, pidSDJWT)
	stdout := "action " + outcomeWallet + "fcaf-issue-pid... COMPLETED\n" +
		"  Tap on \"Share\"... FAILED\n" +
		"action " + outcomeWallet + "fcaf-engagement-haip-vp... FAILED\n"
	return []*historypb.HistoryEvent{
		{
			EventId:   1,
			EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
				WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
					Input: fcafPayloads(t, map[string]any{"workflow_definition": def}),
				},
			},
		},
		outcomeStepScheduled(t, 2, aggregate.PreludeStepID),
		outcomeStepCompleted(t, 3, 2),
		evidence[1],
		evidence[2],
		outcomeStepScheduled(t, 10, prefix+"-present"),
		outcomeStepFailed(12, 10, workflowengine.NewAppError(workflowengine.WorkflowError{
			Code: errorcodes.MobileAutomationStepError,
			Details: map[string]any{
				"stdout":  stdout,
				"step_id": prefix + "-present",
			},
		})),
		{
			EventId:   13,
			EventType: enums.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED,
			Attributes: &historypb.HistoryEvent_StartChildWorkflowExecutionInitiatedEventAttributes{
				StartChildWorkflowExecutionInitiatedEventAttributes: &historypb.StartChildWorkflowExecutionInitiatedEventAttributes{
					WorkflowType: &commonpb.WorkflowType{
						Name: pipelineinternal.LinkedStepsWorkflowName,
					},
				},
			},
		},
		{
			EventId:   15,
			EventType: enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_FAILED,
			Attributes: &historypb.HistoryEvent_ChildWorkflowExecutionFailedEventAttributes{
				ChildWorkflowExecutionFailedEventAttributes: &historypb.ChildWorkflowExecutionFailedEventAttributes{
					InitiatedEventId: 13,
					Failure: outcomeFailureConverter().ErrorToFailure(
						workflowengine.NewAppError(workflowengine.WorkflowError{
							Code: "CRE228",
							Details: map[string]any{
								"output": map[string]any{},
								"errors": []any{map[string]any{
									"code":    "CRE101",
									"details": map[string]any{"step_id": prefix + "-create"},
								}},
							},
						}),
					),
				},
			},
		},
	}
}

func TestFCAFValidationActivityStoresRoleOutcomes(t *testing.T) {
	setFCAFTestEnv(t)
	pidSDJWT := `{"query_0":["` + testPIDSDJWTPresentation(t) + `"]}`
	historyClient := mockFCAFHistory(t, outcomeHistory(t, pidSDJWT))
	app := newPipelineResultsTestApp(t)
	ensureFCAFReportFields(t, app)
	record := createTestPipelineResult(t, app, fcafTestWorkflowID, fcafTestRunID)

	output, err := executeFCAFValidation(t, app, workflowengine.ActivityInput{
		Payload: FCAFValidationActivityInput{
			TestIDs: []string{fcafTestTestID},
			Pipeline: map[string]any{
				fcafTestSource: map[string]any{
					"output": map[string]any{
						"pid_sdjwt": "${{ obtain.outputs.body.pid_sdjwt }}",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	historyClient.AssertExpectations(t)
	compact, err := json.Marshal(output)
	require.NoError(t, err)
	require.NotContains(t, string(compact), "role_outcomes", "the compact output stays as it was")

	reloaded, err := app.FindRecordById("pipeline_results", record.Id)
	require.NoError(t, err)
	var stored struct {
		Summary      map[string]any                 `json:"summary"`
		RoleOutcomes *actionauthor.FCAFRoleOutcomes `json:"role_outcomes"`
	}
	require.NoError(
		t,
		json.Unmarshal(readPipelineResultFile(t, app, reloaded, "fcaf_report"), &stored),
	)
	require.NotNil(t, stored.Summary)
	require.NotNil(t, stored.RoleOutcomes)
	prefix := aggregate.ScenarioPrefix(DefaultFCAFValidationSuite, outcomeScenario)
	fixture := outcomeScenario + "#"
	require.Equal(t, outcomeWallet, stored.RoleOutcomes.WalletPrefix)
	require.Equal(t, []actionauthor.FCAFStepOutcome{
		{
			StepID: aggregate.PreludeStepID, Action: outcomeWallet + "fcaf-reset-and-onboard",
			Role: "fcaf-reset-and-onboard", Status: actionauthor.FCAFRunOK,
		},
		{
			StepID: prefix + "-present", Action: outcomeWallet + "fcaf-issue-pid",
			Role: "fcaf-issue-pid", FixtureID: fixture + "present", Status: actionauthor.FCAFRunOK,
		},
		{
			StepID: prefix + "-present", Action: outcomeWallet + "fcaf-engagement-haip-vp",
			Role: "fcaf-engagement-haip-vp", FixtureID: fixture + "present",
			Status: actionauthor.FCAFRunFailing, FailedCommand: `Tap on "Share"`,
		},
		{
			StepID: prefix + "-again", Action: outcomeWallet + "fcaf-expect-request-rejected",
			Role: "fcaf-expect-request-rejected", FixtureID: fixture + "again",
			Status: actionauthor.FCAFRunNotRun,
		},
	}, stored.RoleOutcomes.Steps)
	require.Equal(t, []actionauthor.FCAFRoleOutcome{
		{
			Action: outcomeWallet + "fcaf-engagement-haip-vp",
			Role:   "fcaf-engagement-haip-vp", Status: actionauthor.FCAFRunFailing, Failing: 1,
			FailedCommand: `Tap on "Share"`, FixtureID: fixture + "present",
			StepID: prefix + "-present",
		},
		{
			Action: outcomeWallet + "fcaf-expect-request-rejected",
			Role:   "fcaf-expect-request-rejected", Status: actionauthor.FCAFRunNotRun, NotRun: 1,
		},
		{
			Action: outcomeWallet + "fcaf-issue-pid",
			Role:   "fcaf-issue-pid", Status: actionauthor.FCAFRunOK, OK: 1,
		},
		{
			Action: outcomeWallet + "fcaf-reset-and-onboard",
			Role:   "fcaf-reset-and-onboard", Status: actionauthor.FCAFRunOK, OK: 1,
		},
	}, stored.RoleOutcomes.Roles)
}

func TestFCAFRunStatusIsTheLowercaseTemporalStatus(t *testing.T) {
	for status, want := range map[enums.WorkflowExecutionStatus]string{
		enums.WORKFLOW_EXECUTION_STATUS_RUNNING:          "running",
		enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:        "completed",
		enums.WORKFLOW_EXECUTION_STATUS_FAILED:           "failed",
		enums.WORKFLOW_EXECUTION_STATUS_CANCELED:         "canceled",
		enums.WORKFLOW_EXECUTION_STATUS_TERMINATED:       "terminated",
		enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:        "timed_out",
		enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW: "continued_as_new",
	} {
		require.Equal(t, want, fcafRunStatus(status))
	}
}
