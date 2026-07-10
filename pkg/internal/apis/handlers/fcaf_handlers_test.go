// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
)

func TestHandleRunFCAFStartsWorkflow(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	app.Settings().Meta.AppURL = "https://credimi.test"

	origStart := fcafWorkflowStart
	origTemporalClient := fcafTemporalClient
	origWait := fcafWorkflowWaitForResult
	t.Cleanup(func() {
		fcafWorkflowStart = origStart
		fcafTemporalClient = origTemporalClient
		fcafWorkflowWaitForResult = origWait
	})

	var capturedNamespace string
	var capturedInput workflowengine.WorkflowInput
	fcafWorkflowStart = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedInput = input
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-1",
			WorkflowRunID: "run-1",
		}, nil
	}

	body, err := json.Marshal(RunFCAFInput{
		TestIDs:  []string{"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001"},
		Runtime:  map[string]any{"credential_format": "sd-jwt-vc"},
		RunnerID: "org-owner/runner-1",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/fcaf/run", bytes.NewBuffer(body))
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, RunFCAFInput{
		TestIDs:  []string{"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001"},
		Runtime:  map[string]any{"credential_format": "sd-jwt-vc"},
		RunnerID: "org-owner/runner-1",
	}))
	rec := httptest.NewRecorder()

	err = HandleRunFCAF()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "usera-s-organization", capturedNamespace)
	require.Equal(t, "wallet_solution/relying_party", capturedInput.Payload.(workflows.FCAFAssessmentWorkflowPayload).Suite)
	require.Equal(t, "org-owner/runner-1", capturedInput.Payload.(workflows.FCAFAssessmentWorkflowPayload).RunnerID)
	require.Equal(t, "org-owner/runner-1", capturedInput.Payload.(workflows.FCAFAssessmentWorkflowPayload).Runtime["runner_id"])
	require.Equal(t, "https://credimi.test", capturedInput.Config["app_url"])
	require.Contains(t, rec.Body.String(), `"workflow_id":"wf-1"`)
}

func TestHandleRunFCAFWaitForCompletionReturnsReport(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	app.Settings().Meta.AppURL = "https://credimi.test"

	origStart := fcafWorkflowStart
	origTemporalClient := fcafTemporalClient
	origWait := fcafWorkflowWaitForResult
	t.Cleanup(func() {
		fcafWorkflowStart = origStart
		fcafTemporalClient = origTemporalClient
		fcafWorkflowWaitForResult = origWait
	})

	fcafWorkflowStart = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-2",
			WorkflowRunID: "run-2",
		}, nil
	}

	mockClient := &temporalmocks.Client{}
	fcafTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}
	fcafWorkflowWaitForResult = func(c client.Client, workflowID, runID string) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			Output: engine.Report{
				Tests: []engine.TestResult{{
					ID:     "test-1",
					Status: validators.StatusPass,
				}},
				ExecutedTests: []engine.ExecutedTest{{
					TestID: "test-1",
					Status: "passed",
					Outcome: engine.TestOutcome{
						Status: "passed",
					},
				}},
				Summary: engine.Summary{
					Pass: 1,
				},
			},
		}, nil
	}

	body, err := json.Marshal(RunFCAFInput{
		TestIDs:           []string{"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001"},
		WaitForCompletion: true,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/fcaf/run", bytes.NewBuffer(body))
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, RunFCAFInput{
		TestIDs:           []string{"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001"},
		WaitForCompletion: true,
	}))
	rec := httptest.NewRecorder()

	err = HandleRunFCAF()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"workflow_id":"wf-2"`)
	require.Contains(t, rec.Body.String(), `"pass":1`)
	require.Contains(t, rec.Body.String(), `"status":"pass"`)
	require.Contains(t, rec.Body.String(), `"executed_tests"`)
	require.Contains(t, rec.Body.String(), `"status":"passed"`)
	mockClient.AssertNotCalled(t, "Close")
}

func TestHandleRunFCAFWaitForCompletionReturnsFailureReport(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	app.Settings().Meta.AppURL = "https://credimi.test"

	origStart := fcafWorkflowStart
	origTemporalClient := fcafTemporalClient
	origWait := fcafWorkflowWaitForResult
	t.Cleanup(func() {
		fcafWorkflowStart = origStart
		fcafTemporalClient = origTemporalClient
		fcafWorkflowWaitForResult = origWait
	})

	fcafWorkflowStart = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-3",
			WorkflowRunID: "run-3",
		}, nil
	}

	mockClient := &temporalmocks.Client{}
	fcafTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}
	report := engine.Report{
		Tests: []engine.TestResult{{
			ID:      "test-1",
			Status:  validators.StatusFail,
			Message: "one or more assertions failed",
		}},
		Summary: engine.Summary{Fail: 1},
		ExecutedTests: []engine.ExecutedTest{{
			TestID: "test-1",
			Status: "failed",
			Preconditions: []engine.ExecutedCheck{{
				ID:      "pipeline.pid.presentation.sdjwt.all-ics-claims",
				Kind:    "pipeline",
				Status:  "passed",
				Message: "pipeline outputs extracted",
			}},
			Assertions: []engine.ExecutedCheck{{
				ID:      "email_present",
				Kind:    "assertion",
				Status:  "failed",
				Message: `claim "email" is missing`,
			}},
			Outcome: engine.TestOutcome{
				Status: "failed",
				Reason: `claim "email" is missing`,
			},
		}},
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
	fcafWorkflowWaitForResult = func(c client.Client, workflowID, runID string) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, temporal.NewApplicationError(
			"FCAF assessment failed",
			"CRE229",
			workflowengine.WorkflowError{
				Code:    "CRE229",
				Summary: "FCAF assessment failed",
				Message: "1 FCAF test(s) did not pass",
				Details: map[string]any{
					"output":   report,
					"summary":  report.Summary,
					"failures": report.Failures,
				},
			},
		)
	}

	body, err := json.Marshal(RunFCAFInput{
		TestIDs:           []string{"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001"},
		WaitForCompletion: true,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/fcaf/run", bytes.NewBuffer(body))
	req = req.WithContext(context.WithValue(req.Context(), middlewares.ValidatedInputKey, RunFCAFInput{
		TestIDs:           []string{"WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001"},
		WaitForCompletion: true,
	}))
	rec := httptest.NewRecorder()

	err = HandleRunFCAF()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), `"workflow_id":"wf-3"`)
	require.Contains(t, rec.Body.String(), `"fail":1`)
	require.Contains(t, rec.Body.String(), `"executed_tests"`)
	require.Contains(t, rec.Body.String(), `"status":"failed"`)
	require.Contains(t, rec.Body.String(), `"claim \"email\" is missing"`)
	mockClient.AssertNotCalled(t, "Close")
}
