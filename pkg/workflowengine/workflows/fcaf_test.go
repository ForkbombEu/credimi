// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestFCAFAssessmentWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	act := activities.NewFCAFAssessmentActivity()
	resolveAct := activities.NewFCAFResolveExecutionPlanActivity()
	env.RegisterActivityWithOptions(
		resolveAct.Execute,
		activity.RegisterOptions{Name: resolveAct.Name()},
	)
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
	env.OnActivity(resolveAct.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: activities.FCAFResolveExecutionPlanActivityOutput{
				SelectedTests:         []string{"test-1"},
				PipelinePreconditions: nil,
			},
		}, nil)
	env.OnActivity(act.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: activities.FCAFAssessmentActivityOutput{
				Report: engine.Report{
					Tests: []engine.TestResult{{
						ID:     "test-1",
						Status: validators.StatusPass,
						Preconditions: []engine.NodeResult{
							{
								ID:          "pipeline.pid_sdjwt",
								Kind:        "pipeline",
								Status:      validators.StatusPass,
								Message:     "pipeline outputs extracted",
								WorkflowID:  "wf-pipeline",
								RunID:       "run-pipeline",
								PipelineURL: "https://app.example.test/my/tests/runs/wf-pipeline/run-pipeline",
								Outputs: map[string]any{
									"pid_sdjwt": map[string]any{
										"claims": map[string]any{"email": "person@example.test"},
									},
								},
								Evidence: []engine.EvidenceResult{{
									Name:       "pid_sdjwt",
									SourceNode: "pipeline.pid_sdjwt",
									Path:       "$.output.pid_sdjwt",
								}},
							},
						},
						Assertions: []engine.AssertionResult{{
							ID:           "email-present",
							Status:       validators.StatusPass,
							EvidenceKeys: []string{"pid_sdjwt"},
						}},
					}},
					Summary: engine.Summary{Pass: 1},
				},
			},
		}, nil)

	w := NewFCAFAssessmentWorkflow()
	registerFCAFTestWorkflow(t, env)
	env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
		Payload: FCAFAssessmentWorkflowPayload{
			TestIDs: []string{"test-1"},
			Suite:   "wallet_solution/relying_party",
			Evidence: evidence.Bundle{
				DecodedSDJWT: map[string]any{"email": "person@example.test"},
			},
		},
		Config: map[string]any{"app_url": "http://app.example.test"},
	})

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "FCAF assessment completed", result.Message)
	var report engine.Report
	data, err := json.Marshal(result.Output)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &report))
	require.Empty(t, report.Tests)
	require.Len(t, report.ExecutedTests, 1)
	require.Equal(t, "passed", report.ExecutedTests[0].Outcome.Status)
	require.Contains(t, report.Evidence, "pid_sdjwt")
	require.Nil(t, report.Evidence["pid_sdjwt"].Value)
	require.Contains(t, report.Evidence, "pipeline.pid_sdjwt.run")
	runEvidence, ok := report.Evidence["pipeline.pid_sdjwt.run"].Value.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "wf-pipeline", runEvidence["workflow_id"])
	require.Equal(t, "run-pipeline", runEvidence["run_id"])
	require.Equal(
		t,
		"https://app.example.test/my/tests/runs/wf-pipeline/run-pipeline",
		runEvidence["pipeline_url"],
	)
	require.Equal(
		t,
		[]string{"pipeline.pid_sdjwt.run", "pid_sdjwt"},
		report.ExecutedTests[0].Preconditions[0].EvidenceKeys,
	)
}

func TestFCAFAssessmentWorkflowAllowsAllApplicableTests(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	act := activities.NewFCAFAssessmentActivity()
	resolveAct := activities.NewFCAFResolveExecutionPlanActivity()
	env.RegisterActivityWithOptions(
		resolveAct.Execute,
		activity.RegisterOptions{Name: resolveAct.Name()},
	)
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
	env.OnActivity(resolveAct.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: activities.FCAFResolveExecutionPlanActivityOutput{
				SelectedTests:         nil,
				PipelinePreconditions: nil,
			},
		}, nil)
	env.OnActivity(act.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: activities.FCAFAssessmentActivityOutput{
				Report: engine.Report{Summary: engine.Summary{}},
			},
		}, nil)

	w := NewFCAFAssessmentWorkflow()
	registerFCAFTestWorkflow(t, env)
	env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
		Payload: FCAFAssessmentWorkflowPayload{Suite: "wallet_solution/relying_party"},
		Config:  map[string]any{"app_url": "http://app.example.test"},
	})

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
}

func TestFCAFAssessmentWorkflowFailsWhenAnyTestFails(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	act := activities.NewFCAFAssessmentActivity()
	resolveAct := activities.NewFCAFResolveExecutionPlanActivity()
	env.RegisterActivityWithOptions(
		resolveAct.Execute,
		activity.RegisterOptions{Name: resolveAct.Name()},
	)
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
	env.OnActivity(resolveAct.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: activities.FCAFResolveExecutionPlanActivityOutput{
				SelectedTests: []string{"test-1"},
			},
		}, nil)
	env.OnActivity(act.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: activities.FCAFAssessmentActivityOutput{
				Report: engine.Report{
					Tests: []engine.TestResult{{
						ID:      "test-1",
						Title:   "Email present",
						Status:  validators.StatusFail,
						Message: "one or more assertions failed",
						Preconditions: []engine.NodeResult{{
							ID:      "pipeline.pid.presentation.sdjwt.all-claims",
							Kind:    "pipeline",
							Status:  validators.StatusFail,
							Message: "missing key in path",
						}},
					}},
					Summary: engine.Summary{Fail: 1},
				},
			},
		}, nil)

	w := NewFCAFAssessmentWorkflow()
	registerFCAFTestWorkflow(t, env)
	env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
		Payload: FCAFAssessmentWorkflowPayload{
			TestIDs: []string{"test-1"},
			Suite:   "wallet_solution/relying_party",
		},
		Config: map[string]any{"app_url": "http://app.example.test"},
	})

	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	require.Error(t, err)
	failure := workflowengine.ParseWorkflowError(err)
	require.Equal(t, "CRE229", failure.Code)
	require.Equal(t, "FCAF assessment failed", failure.Summary)
	require.Contains(t, failure.Message, "test-1")
	require.Contains(t, failure.Message, "precondition pipeline.pid.presentation.sdjwt.all-claims")
	require.NotNil(t, failure.Details["output"])
	require.Nil(t, failure.Details["executed_tests"])
}

func TestFCAFTestWorkflowRunsOneLeafChildPerTest(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	resultActivity := activities.NewGetWorkflowResultActivity()
	assessmentActivity := activities.NewFCAFAssessmentActivity()
	env.RegisterActivityWithOptions(
		resultActivity.Execute,
		activity.RegisterOptions{Name: resultActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		assessmentActivity.Execute,
		activity.RegisterOptions{Name: assessmentActivity.Name()},
	)
	env.OnActivity(resultActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: workflowengine.WorkflowResult{
			WorkflowID:    "pipeline-workflow",
			WorkflowRunID: "pipeline-run",
			Output: map[string]any{
				"step-1": map[string]any{"outputs": "credential"},
			},
		}}, nil).
		Twice()
	env.OnActivity(assessmentActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: activities.FCAFAssessmentActivityOutput{Report: engine.Report{
				Status: "passed",
				ExecutedTests: []engine.ExecutedTest{{
					TestID: "test-1",
					Status: "passed",
					Outcome: engine.TestOutcome{
						Status: "passed",
					},
				}},
				Evidence: engine.EvidenceMap{
					"pid_mdoc": {
						Type:  "mdoc",
						Value: "large credential",
					},
					"pipeline.run": {
						Type:  "pipeline.run",
						Value: "run URL",
					},
				},
				Summary: engine.Summary{Pass: 1},
			}},
		}, nil).
		Once()
	env.OnActivity(assessmentActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: activities.FCAFAssessmentActivityOutput{Report: engine.Report{
				Status: "passed",
				ExecutedTests: []engine.ExecutedTest{{
					TestID: "test-2",
					Status: "passed",
					Outcome: engine.TestOutcome{
						Status: "passed",
					},
				}},
				Evidence: engine.EvidenceMap{
					"pid_mdoc": {
						Type:  "mdoc",
						Value: "large credential repeated",
					},
					"pipeline.run": {
						Type:  "pipeline.run",
						Value: "run URL",
					},
				},
				Summary: engine.Summary{Pass: 1},
			}},
		}, nil).
		Once()

	registerFCAFTestWorkflow(t, env)
	w := NewFCAFTestWorkflow()
	env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
		Payload: FCAFTestWorkflowPayload{
			TestIDs: []string{"test-1", "test-2"},
			Suite:   "wallet_solution/relying_party",
			PipelineReferences: []FCAFPipelineResultReference{{
				Aliases:    []string{"pipeline.pid.presentation.mdoc.all-claims-elements"},
				WorkflowID: "pipeline-workflow",
				RunID:      "pipeline-run",
				Namespace:  "organization",
			}},
		},
		Config: map[string]any{"app_url": "http://app.example.test"},
	})

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	var report engine.Report
	data, err := json.Marshal(result.Output)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &report))
	require.Len(t, report.ExecutedTests, 2)
	require.Contains(t, report.Evidence, "pid_mdoc")
	require.Nil(t, report.Evidence["pid_mdoc"].Value)
	require.Contains(t, report.Evidence, "pipeline.run")
	env.AssertExpectations(t)
}

func TestFCAFSingleTestWorkflowKeepsOnlySelectedTest(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()
	assessmentActivity := activities.NewFCAFAssessmentActivity()
	env.RegisterActivityWithOptions(
		assessmentActivity.Execute,
		activity.RegisterOptions{Name: assessmentActivity.Name()},
	)
	env.OnActivity(assessmentActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: activities.FCAFAssessmentActivityOutput{Report: engine.Report{
				Status: "passed",
				ExecutedTests: []engine.ExecutedTest{
					{
						TestID: "selected-test",
						Status: "passed",
						Outcome: engine.TestOutcome{
							Status: "passed",
						},
					},
					{
						TestID: "precondition-test",
						Status: "passed",
						Outcome: engine.TestOutcome{
							Status: "passed",
						},
					},
				},
				Summary: engine.Summary{Pass: 2},
			}},
		}, nil).
		Once()

	w := NewFCAFSingleTestWorkflow()
	env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
		Payload: FCAFSingleTestWorkflowPayload{
			TestID: "selected-test",
			Suite:  "wallet_solution/relying_party",
		},
		Config: map[string]any{"app_url": "http://app.example.test"},
	})

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	var report engine.Report
	data, err := json.Marshal(result.Output)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &report))
	require.Len(t, report.ExecutedTests, 1)
	require.Equal(t, "selected-test", report.ExecutedTests[0].TestID)
	require.Equal(t, 1, report.Summary.Pass)
	env.AssertExpectations(t)
}

func registerFCAFTestWorkflow(
	t *testing.T,
	env *testsuite.TestWorkflowEnvironment,
) {
	t.Helper()
	testWorkflow := NewFCAFTestWorkflow()
	singleTestWorkflow := NewFCAFSingleTestWorkflow()
	env.RegisterWorkflowWithOptions(
		testWorkflow.WorkflowFunc,
		workflow.RegisterOptions{Name: testWorkflow.Name()},
	)
	env.RegisterWorkflowWithOptions(
		singleTestWorkflow.WorkflowFunc,
		workflow.RegisterOptions{Name: singleTestWorkflow.Name()},
	)
}

func TestDecodePipelineYAMLHTTPOutput(t *testing.T) {
	t.Parallel()

	got, err := decodePipelineYAMLHTTPOutput(map[string]any{
		"status": http.StatusOK,
		"body":   "steps:\n  - use: http-request\n",
	})
	require.NoError(t, err)
	require.Equal(t, "steps:\n  - use: http-request\n", got)
}

func TestPrepareFCAFPreconditionPipelineYAMLPreservesStepIDs(t *testing.T) {
	t.Parallel()

	rewritten, info, err := prepareFCAFPreconditionPipelineYAML(`name: test
steps:
  - id: issuer-0001
    use: credential-offer
    continue_on_error: false
    with:
      credential_id: example/credential
  - id: mobile-0002
    use: mobile-automation
    with:
      action_id: example/action
      parameters:
        deeplink: ${{issuer-0001.outputs}}
`, map[string]any{"runner_id": "org/runner"})
	require.NoError(t, err)
	require.True(t, info.NeedsGlobalRunner)

	def, err := pipelineinternal.ParseWorkflow(rewritten)
	require.NoError(t, err)
	require.Len(t, def.Steps, 2)
	require.Equal(t, "issuer-0001", def.Steps[0].ID)
	require.Equal(t, "mobile-0002", def.Steps[1].ID)
	require.Equal(t, "org/runner", def.Runtime.GlobalRunnerID)
	require.Equal(
		t,
		"${{issuer-0001.outputs}}",
		def.Steps[1].With.Payload["parameters"].(map[string]any)["deeplink"],
	)
}

func TestFCAFPreconditionWorkflowIDUsesCanonicalSuffix(t *testing.T) {
	t.Parallel()

	got := fcafPreconditionWorkflowID(
		"FCAFAssessment-1234",
		"pipeline.pid.presentation.sdjwt.all-claims",
	)
	require.Equal(t, "FCAFAssessment-1234-pid-presentation-sdjwt-all-claims", got)
}

func TestFCAFTestsWorkflowID(t *testing.T) {
	got := fcafTestsWorkflowID("FCAFAssessment-1234")
	require.Equal(t, "FCAFAssessment-1234-tests", got)
}

func TestFCAFSingleTestWorkflowID(t *testing.T) {
	got := fcafSingleTestWorkflowID(
		"FCAFAssessment-1234-tests",
		"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001",
	)
	require.Equal(
		t,
		"FCAFAssessment-1234-tests-test-ws-rp-dm-addressdata-emailaddress-pid-ietf-sd-jwt-vc-001",
		got,
	)
}

func TestExistingFCAFPipelineOutputReusesPipelineID(t *testing.T) {
	t.Parallel()

	shared := evidence.PipelineExecutionResult{WorkflowID: "wf-shared"}
	output, found := existingFCAFPipelineOutput(
		map[string]any{"owner/long-pipeline": shared},
		dsl.PreconditionDefinition{
			ID:         "pipeline.test-specific-view",
			PipelineID: "owner/long-pipeline",
		},
	)

	require.True(t, found)
	require.Equal(t, shared, output)
}

func TestFailedPipelineWorkflowResultPreservesOutputAndStepFailures(t *testing.T) {
	t.Parallel()

	err := workflowengine.NewWorkflowError(
		workflowengine.NewAppError(workflowengine.WorkflowError{
			Code:    "CRE229",
			Summary: "Pipeline failed",
			Details: map[string]any{
				"output": map[string]any{
					"passed-step": map[string]any{"outputs": "ok"},
				},
				"errors": []workflowengine.WorkflowError{{
					Code:    "CRE228",
					Summary: "assertion failed",
					Details: map[string]any{"step_id": "failed-step"},
				}},
			},
		}),
		&workflowengine.WorkflowRunMetadata{
			WorkflowID: "wf-1",
			RunID:      "run-1",
		},
	)

	failed, recovered := failedPipelineWorkflowResult(err)
	require.True(t, recovered)
	result, decodeErr := fcafPipelineExecutionResult(failed, "", "")
	require.NoError(t, decodeErr)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, "run-1", result.WorkflowRunID)
	require.Len(t, result.StepFailures, 1)
	require.Equal(t, "failed-step", result.StepFailures[0].StepID)
	require.Equal(
		t,
		"ok",
		result.Output.(map[string]any)["passed-step"].(map[string]any)["outputs"],
	)
}

func TestFCAFQueueTicketIDAvoidsPathSeparators(t *testing.T) {
	t.Parallel()

	got := fcafQueueTicketID("FCAFAssessment-1234/child")
	require.NotContains(t, got, "/")
	require.Contains(t, got, "fcaf-")
}

func TestEnqueueRunTicketStatusToSemaphoreStatus(t *testing.T) {
	t.Parallel()

	got := enqueueRunTicketStatusToSemaphoreStatus(
		"ticket-1",
		[]string{"runner-1", "runner-2"},
		activities.EnqueuePipelineRunTicketActivityOutput{
			Status:            "running",
			Position:          0,
			LineLen:           1,
			WorkflowID:        "wf-1",
			RunID:             "run-1",
			WorkflowNamespace: "tenant-a",
			ErrorMessage:      "boom",
		},
	)
	require.Equal(t, "ticket-1", got.TicketID)
	require.Equal(t, "runner-1", got.LeaderRunnerID)
	require.Equal(t, []string{"runner-1", "runner-2"}, got.RequiredRunnerIDs)
	require.Equal(t, "wf-1", got.WorkflowID)
	require.Equal(t, "run-1", got.RunID)
	require.Equal(t, "tenant-a", got.WorkflowNamespace)
	require.Equal(t, "boom", got.ErrorMessage)
}

func TestMergeSemaphoreStatusPreservesWorkflowIdentity(t *testing.T) {
	t.Parallel()

	got := mergeSemaphoreStatus(
		mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView{
			TicketID:          "ticket-1",
			Status:            mobilerunnersemaphore.MobileRunnerSemaphoreRunRunning,
			WorkflowID:        "wf-1",
			RunID:             "run-1",
			WorkflowNamespace: "tenant-a",
		},
		mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView{
			TicketID: "ticket-1",
			Status:   mobilerunnersemaphore.MobileRunnerSemaphoreRunNotFound,
		},
	)
	require.Equal(t, mobilerunnersemaphore.MobileRunnerSemaphoreRunNotFound, got.Status)
	require.Equal(t, "wf-1", got.WorkflowID)
	require.Equal(t, "run-1", got.RunID)
	require.Equal(t, "tenant-a", got.WorkflowNamespace)
}

func TestRunQueuedPipelineWaitsForClosedTerminalWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	enqueueAct := activities.NewEnqueuePipelineRunTicketActivity()
	queryAct := activities.NewQueryMobileRunnerSemaphoreRunStatusActivity()
	checkAct := activities.NewCheckWorkflowClosedActivity()
	getResultAct := activities.NewGetWorkflowResultActivity()

	env.RegisterActivityWithOptions(func(
		_ context.Context,
		_ workflowengine.ActivityInput,
	) (workflowengine.ActivityResult, error) {
		return workflowengine.ActivityResult{
			Output: activities.EnqueuePipelineRunTicketActivityOutput{
				Status:            mobilerunnersemaphore.MobileRunnerSemaphoreRunFailed,
				WorkflowID:        "wf-1",
				RunID:             "run-1",
				WorkflowNamespace: "tenant-a",
			},
		}, nil
	}, activity.RegisterOptions{Name: enqueueAct.Name()})

	env.RegisterActivityWithOptions(func(
		_ context.Context,
		_ workflowengine.ActivityInput,
	) (workflowengine.ActivityResult, error) {
		t.Fatal("query activity should not run for already-terminal queued status")
		return workflowengine.ActivityResult{}, nil
	}, activity.RegisterOptions{Name: queryAct.Name()})

	checkCalls := 0
	env.RegisterActivityWithOptions(func(
		_ context.Context,
		_ workflowengine.ActivityInput,
	) (workflowengine.ActivityResult, error) {
		checkCalls++
		if checkCalls == 1 {
			return workflowengine.ActivityResult{
				Output: activities.CheckWorkflowClosedActivityOutput{
					Closed: false,
					Status: "RUNNING",
				},
			}, nil
		}
		return workflowengine.ActivityResult{
			Output: activities.CheckWorkflowClosedActivityOutput{Closed: true, Status: "FAILED"},
		}, nil
	}, activity.RegisterOptions{Name: checkAct.Name()})

	getCalls := 0
	env.RegisterActivityWithOptions(func(
		_ context.Context,
		_ workflowengine.ActivityInput,
	) (workflowengine.ActivityResult, error) {
		getCalls++
		return workflowengine.ActivityResult{
			Output: workflowengine.WorkflowResult{
				WorkflowID:    "wf-1",
				WorkflowRunID: "run-1",
				Output:        map[string]any{"done": true},
			},
		}, nil
	}, activity.RegisterOptions{Name: getResultAct.Name()})

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (workflowengine.WorkflowResult, error) {
			ctx = workflow.WithActivityOptions(ctx, DefaultActivityOptions)
			return NewFCAFPreconditionPipelineWorkflow().runQueuedPipeline(
				ctx,
				workflowengine.WorkflowInput{
					Config: map[string]any{
						"app_url": "http://app.example.test",
					},
				},
				"tenant-a/pipeline-a",
				"tenant-a",
				"name: test\nsteps: []\n",
				[]string{"runner-1"},
			)
		},
		workflow.RegisterOptions{Name: "test-run-queued-pipeline-waits-for-close"},
	)

	env.ExecuteWorkflow("test-run-queued-pipeline-waits-for-close")

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 2, checkCalls)
	require.Equal(t, 1, getCalls)

	pipelineResult, err := evidence.DecodePipelineExecutionResult(result.Output)
	require.NoError(t, err)
	require.Equal(t, "wf-1", pipelineResult.WorkflowID)
	require.Equal(t, "run-1", pipelineResult.WorkflowRunID)
}

func TestSummarizeFCAFFailures(t *testing.T) {
	report := engine.Report{
		Failures: []engine.TestFailure{{
			TestID: "test-1",
			Status: validators.StatusFail,
			Reasons: []engine.FailureReason{{
				Scope:   "assertion",
				ID:      "email_present",
				Status:  validators.StatusFail,
				Message: "claim \"email\" is missing",
			}},
		}},
	}

	got := summarizeFCAFFailures(report)
	require.Contains(t, got, "1 FCAF test(s) did not pass")
	require.Contains(t, got, "test-1 (fail)")
	require.Contains(t, got, "assertion email_present: claim \"email\" is missing")
}
