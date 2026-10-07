// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestStartAggregateScoreboardRequiresAPIKey(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			Name:           "missing API key",
			Method:         http.MethodPost,
			URL:            "/api/pipeline/scoreboard/aggregate/start",
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				"api_key_required",
			},
			TestAppFactory: setupPipelineApp,
		},
		{
			Name:           "invalid API key",
			Method:         http.MethodPost,
			URL:            "/api/pipeline/scoreboard/aggregate/start",
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				"invalid_api_key",
			},
			Headers: map[string]string{
				"Credimi-Api-Key": "wrong-key",
			},
			TestAppFactory: setupPipelineApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestStartAggregateScoreboard(t *testing.T) {
	origStart := aggregateScoreboardWorkflowStart
	t.Cleanup(func() {
		aggregateScoreboardWorkflowStart = origStart
	})

	app := setupPipelineApp(t)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://credimi.test"

	var capturedNamespace string
	var capturedInput workflowengine.WorkflowInput

	aggregateScoreboardWorkflowStart = func(
		namespace string,
		input workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedInput = input
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-123",
			WorkflowRunID: "run-456",
			Message:       "started",
		}, nil
	}

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	err = app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/pipeline/scoreboard/aggregate/start", nil)
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var response StartAggregateScoreboardResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
		require.Equal(t, "wf-123", response.WorkflowID)
		require.Equal(t, "run-456", response.WorkflowRunID)
		require.Equal(t, "started", response.Message)
		require.Equal(t, "default", response.WorkflowNamespace)
		require.Equal(t, "default", capturedNamespace)
		require.Equal(
			t,
			workflowengine.WorkflowInput{Config: workflowengine.WithAppConfig(app, nil)},
			capturedInput,
		)
		require.Equal(t, "https://credimi.test", capturedInput.Config["app_url"])

		return nil
	})
	require.NoError(t, err)
}

func TestStartAggregateScoreboardWorkflowStartFailure(t *testing.T) {
	origStart := aggregateScoreboardWorkflowStart
	t.Cleanup(func() {
		aggregateScoreboardWorkflowStart = origStart
	})

	app := setupPipelineApp(t)
	defer app.Cleanup()

	aggregateScoreboardWorkflowStart = func(
		_ string,
		_ workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("temporal unavailable")
	}

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	err = app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/pipeline/scoreboard/aggregate/start", nil)
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Contains(t, rec.Body.String(), "failed to start aggregate scoreboard workflow")

		return nil
	})
	require.NoError(t, err)
}

func createPipelineResultWithType(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	pipelineID string,
	workflowID string,
	runID string,
	runType string,
) {
	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("pipeline", pipelineID)
	record.Set("workflow_id", workflowID)
	record.Set("run_id", runID)
	record.Set("type", runType)
	require.NoError(t, app.Save(record))
}

func addEntitySearchAttributes(info *workflow.WorkflowExecutionInfo, attrs map[string]any) {
	if info.GetSearchAttributes() == nil {
		info.SearchAttributes = &common.SearchAttributes{
			IndexedFields: make(map[string]*common.Payload),
		}
	}

	for key, value := range attrs {
		payload, err := converter.GetDefaultDataConverter().ToPayload(value)
		if err != nil {
			continue
		}
		info.SearchAttributes.IndexedFields[key] = payload
	}
}

type ExecutionInfo struct {
	Info     *workflow.WorkflowExecutionInfo
	Duration time.Duration
}

func buildPipelineExecutionInfoWithRunner(
	t testing.TB,
	workflowID, runID, pipelineIdentifier, status string,
	deviceIDs []string,
	startTime, closeTime time.Time,
) ExecutionInfo {
	info := &workflow.WorkflowExecutionInfo{
		Execution: &common.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
		Type: &common.WorkflowType{
			Name: pipeline.NewPipelineWorkflow().Name(),
		},
		Status:    parseStatus(status),
		StartTime: timestamppb.New(startTime),
		CloseTime: timestamppb.New(closeTime),
	}

	duration := closeTime.Sub(startTime)

	indexedFields := make(map[string]*common.Payload)

	if pipelineIdentifier != "" {
		payload, err := converter.GetDefaultDataConverter().ToPayload(pipelineIdentifier)
		require.NoError(t, err)
		indexedFields[workflowengine.PipelineIdentifierSearchAttribute] = payload
	}

	if len(deviceIDs) > 0 {
		payload, err := converter.GetDefaultDataConverter().ToPayload(deviceIDs)
		require.NoError(t, err)
		indexedFields[workflowengine.DeviceIdentifiersSearchAttribute] = payload
	}

	if len(indexedFields) > 0 {
		info.SearchAttributes = &common.SearchAttributes{
			IndexedFields: indexedFields,
		}
	}

	return ExecutionInfo{
		Info:     info,
		Duration: duration,
	}
}

func parseStatus(status string) enums.WorkflowExecutionStatus {
	switch status {
	case "Completed":
		return enums.WORKFLOW_EXECUTION_STATUS_COMPLETED
	case "Failed":
		return enums.WORKFLOW_EXECUTION_STATUS_FAILED
	case "Running":
		return enums.WORKFLOW_EXECUTION_STATUS_RUNNING
	case "Canceled":
		return enums.WORKFLOW_EXECUTION_STATUS_CANCELED
	case "Terminated":
		return enums.WORKFLOW_EXECUTION_STATUS_TERMINATED
	default:
		return enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED
	}
}

func TestExtractCompletionStatus(t *testing.T) {
	testCases := []struct {
		name string
		exec *WorkflowExecution
		want bool
	}{
		{
			name: "completed normalized status",
			exec: &WorkflowExecution{Status: "Completed"},
			want: true,
		},
		{
			name: "completed temporal enum status",
			exec: &WorkflowExecution{Status: "WORKFLOW_EXECUTION_STATUS_COMPLETED"},
			want: true,
		},
		{
			name: "failed status",
			exec: &WorkflowExecution{Status: "Failed"},
			want: false,
		},
		{
			name: "nil execution",
			exec: nil,
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, extractCompletionStatus(tc.exec))
		})
	}
}

func TestCalculateStatsFromExecutionsOrdersMixedTimestampPrecision(t *testing.T) {
	attrs := DecodedWorkflowSearchAttributes{}
	executions := []*WorkflowExecution{
		{
			Execution:        &WorkflowIdentifier{WorkflowID: "whole-second", RunID: "run-1"},
			StartTime:        "2026-04-21T10:00:00Z",
			CloseTime:        "2026-04-21T10:01:00Z",
			Status:           "Completed",
			SearchAttributes: &attrs,
		},
		{
			Execution:        &WorkflowIdentifier{WorkflowID: "fractional-second", RunID: "run-2"},
			StartTime:        "2026-04-21T10:00:00.1Z",
			CloseTime:        "2026-04-21T10:01:00.1Z",
			Status:           "Completed",
			SearchAttributes: &attrs,
		},
		{
			Execution:        &WorkflowIdentifier{WorkflowID: "earliest", RunID: "run-3"},
			StartTime:        "2026-04-21T09:59:59.999999999Z",
			CloseTime:        "2026-04-21T10:00:59.999999999Z",
			Status:           "Completed",
			SearchAttributes: &attrs,
		},
	}

	stats, lastRun := calculateStatsFromExecutions(executions, nil, nil, nil)

	require.Equal(t, "2026-04-21T09:59:59.999999999Z", stats.FirstExecutionDate)
	require.Equal(t, "2026-04-21T10:00:00.1Z", stats.LastExecutionDate)
	require.NotNil(t, lastRun)
	require.Equal(t, "fractional-second", lastRun.WorkflowID)
}

func TestCalculateStatsFromExecutionsTracksLatestFailedRun(t *testing.T) {
	attrs := DecodedWorkflowSearchAttributes{}
	executions := []*WorkflowExecution{
		{
			Execution:        &WorkflowIdentifier{WorkflowID: "older-success", RunID: "run-1"},
			StartTime:        "2026-04-21T10:00:00Z",
			CloseTime:        "2026-04-21T10:01:00Z",
			Status:           "Completed",
			SearchAttributes: &attrs,
		},
		{
			Execution:        &WorkflowIdentifier{WorkflowID: "latest-failure", RunID: "run-2"},
			StartTime:        "2026-04-21T11:00:00Z",
			CloseTime:        "2026-04-21T11:01:00Z",
			Status:           "Failed",
			SearchAttributes: &attrs,
		},
	}

	stats, lastRun := calculateStatsFromExecutions(executions, nil, nil, nil)

	require.Equal(t, 2, stats.TotalRuns)
	require.Equal(t, 1, stats.TotalSuccesses)
	require.NotNil(t, lastRun, "failed latest run must be tracked for scoreboard evidence")
	require.Equal(t, "latest-failure", lastRun.WorkflowID)
	require.Equal(t, "run-2", lastRun.RunID)
}

func createRunnerRecord(t testing.TB, app *tests.TestApp, orgID, name string) *core.Record {
	runnersColl, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)

	runner := core.NewRecord(runnersColl)
	runner.Set("name", name)
	runner.Set("owner", orgID)
	runner.Set("ip", "my_ip")
	runner.Set("type", "android_emulator")
	require.NoError(t, app.Save(runner))
	return runner
}

func createDeviceRecord(t testing.TB, app *tests.TestApp, orgID, runnerID, name string) {
	devicesColl, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(devicesColl)
	device.Set("name", name)
	device.Set("owner", orgID)
	device.Set("runner", runnerID)
	device.Set("type", "android_emulator")
	device.Set("serial", name+"-serial")
	require.NoError(t, app.Save(device))
}

func createWalletRecord(t testing.TB, app *tests.TestApp, orgID, name string) *core.Record {
	walletsColl, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	wallet := core.NewRecord(walletsColl)
	wallet.Set("owner", orgID)
	wallet.Set("name", name)
	wallet.Set("published", true)
	require.NoError(t, app.Save(wallet))

	walletVersionColl, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)
	versionRecord := core.NewRecord(walletVersionColl)
	versionRecord.Set("wallet", wallet.Id)
	versionRecord.Set("tag", "1.0.0")
	versionRecord.Set("owner", orgID)
	apkFile := NewTestFile("app.apk", []byte("dummy apk content"))
	versionRecord.Set("android_installer", []*filesystem.File{apkFile})
	require.NoError(t, app.Save(versionRecord))

	walletActionColl, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	actionRecord := core.NewRecord(walletActionColl)
	actionRecord.Set("wallet", wallet.Id)
	actionRecord.Set("name", "my-action")
	actionRecord.Set("category", "onboarding")
	actionRecord.Set("owner", orgID)
	actionRecord.Set("code", "my-code")
	require.NoError(t, app.Save(actionRecord))
	require.NoError(t, app.Save(versionRecord))
	return wallet
}

func createVerifierRecord(t testing.TB, app *tests.TestApp, orgID, name string) {
	verifiersColl, err := app.FindCollectionByNameOrId("verifiers")
	require.NoError(t, err)

	verifier := core.NewRecord(verifiersColl)
	verifier.Set("owner", orgID)
	verifier.Set("name", name)
	verifier.Set("published", true)
	verifier.Set("url", "https://verifier.example")
	verifier.Set("standard_and_version", "testsuite/draft-01")
	verifier.Set("format", []string{"SD-JWT"})
	verifier.Set("signing_algorithms", []string{"ES256"})
	verifier.Set("cryptographic_binding_methods", []string{"jwk"})
	verifier.Set("description", "example description")
	require.NoError(t, app.Save(verifier))

	coll, err := app.FindCollectionByNameOrId("use_cases_verifications")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	record.Set("name", "usecase123")
	record.Set("owner", orgID)
	record.Set("published", true)
	record.Set("verifier", verifier.Id)
	record.Set("yaml", "example code")
	require.NoError(t, app.Save(record))
}

func createIssuerRecord(t testing.TB, app *tests.TestApp, orgID, name string) {
	issuersColl, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)
	issuer := core.NewRecord(issuersColl)
	issuer.Set("url", "https://test-issuer.example.com")
	issuer.Set("name", name)
	issuer.Set("owner", orgID)
	issuer.Set("published", true)
	issuer.Set("imported", true)
	require.NoError(t, app.Save(issuer))

	credColl, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)
	cred := core.NewRecord(credColl)
	cred.Set("credential_issuer", issuer.Id)
	cred.Set("name", "cred-3")
	cred.Set("display_name", "Old Name")
	cred.Set("logo_url", "https://old.logo")
	cred.Set("json", `not-json`)
	cred.Set("owner", orgID)
	cred.Set("published", true)
	require.NoError(t, app.Save(cred))
}

func createCustomCheckRecord(t testing.TB, app *tests.TestApp, orgID, name string) {
	customChecksColl, err := app.FindCollectionByNameOrId("custom_checks")
	require.NoError(t, err)
	check := core.NewRecord(customChecksColl)
	check.Set("name", name)
	check.Set("yaml", "example code")
	check.Set("owner", orgID)
	check.Set("published", true)
	require.NoError(t, app.Save(check))
}

func TestFindRunners(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	createRunnerRecord(t, app, orgID, "existing-runner")

	t.Run("success - existing runners", func(t *testing.T) {
		runnerNames := []string{
			"usera-s-organization/existing-runner",
		}
		ids, err := findRecords(app, runnerNames)
		require.NoError(t, err)
		require.Len(t, ids, 1)
	})

	t.Run("fail - invalid runner format (no slash)", func(t *testing.T) {
		runnerNames := []string{
			"invalid-format-no-slash",
		}
		ids, err := findRecords(app, runnerNames)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid-format-no-slash")
		require.Empty(t, ids)
	})

	t.Run("fail - invalid runner format (multiple slashes)", func(t *testing.T) {
		runnerNames := []string{
			"owner/name/extra",
		}
		ids, err := findRecords(app, runnerNames)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to resolve path owner/name/extra")
		require.Empty(t, ids)
	})

	t.Run("success - empty runner list", func(t *testing.T) {
		ids, err := findRecords(app, []string{})
		require.NoError(t, err)
		require.Empty(t, ids)
	})

	t.Run("fail - non-existent runners (not created)", func(t *testing.T) {
		runnerNames := []string{
			"usera-s-organization/non-existent-runner-1",
			"usera-s-organization/non-existent-runner-2",
		}
		ids, err := findRecords(app, runnerNames)
		require.Error(t, err)
		require.Contains(
			t,
			err.Error(),
			"failed to resolve path usera-s-organization/non-existent-runner-1",
		)
		require.Empty(t, ids)
	})
}

func TestHandleScheduleAggregateScoreboard(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://example.test"

	originalScheduleTemporalClient := scheduleTemporalClient
	defer func() { scheduleTemporalClient = originalScheduleTemporalClient }()

	t.Run("success - schedule with valid interval", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockScheduleClient := &fakeScheduleClient{}
		mockClient.On("ScheduleClient").Return(mockScheduleClient)
		mockClient.On("Close").Return()

		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return mockClient, nil
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/scoreboard/aggregate/start?schedule=300",
			nil,
		)
		req.SetPathValue("schedule", "300")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err := HandleStartAggregateScoreboard()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
		require.Contains(t, response["message"].(string), "scheduled every 300 seconds")
		require.NotEmpty(t, response["schedule_id"])

		require.Len(t, mockScheduleClient.createdOptions, 1)
		opts := mockScheduleClient.createdOptions[0]
		require.Len(t, opts.Spec.Intervals, 1)
		require.Equal(t, 300*time.Second, opts.Spec.Intervals[0].Every)
		action, ok := opts.Action.(*client.ScheduleWorkflowAction)
		require.True(t, ok)
		require.True(t, strings.HasPrefix(action.ID, "aggregate-scoreboard-"))
	})

	t.Run("fail - invalid schedule parameter (negative)", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/scoreboard/aggregate/start?schedule=-100",
			nil,
		)
		req.SetPathValue("schedule", "-100")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err := HandleStartAggregateScoreboard()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		requireHandlerErrorHandled(t, rec, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("fail - invalid schedule parameter (zero)", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/scoreboard/aggregate/start?schedule=0",
			nil,
		)
		req.SetPathValue("schedule", "0")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err := HandleStartAggregateScoreboard()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		requireHandlerErrorHandled(t, rec, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("fail - invalid schedule parameter (not a number)", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/scoreboard/aggregate/start?schedule=abc",
			nil,
		)
		req.SetPathValue("schedule", "abc")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err := HandleStartAggregateScoreboard()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		requireHandlerErrorHandled(t, rec, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
	t.Run("fail - temporal client error", func(t *testing.T) {
		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return nil, errors.New("temporal connection failed")
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/scoreboard/aggregate/start?schedule=300",
			nil,
		)
		req.SetPathValue("schedule", "300")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err := HandleStartAggregateScoreboard()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		requireHandlerErrorHandled(t, rec, err)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("fail - schedule creation fails", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockScheduleClient := &fakeScheduleClient{createErr: errors.New("create failed")}
		mockClient.On("ScheduleClient").Return(mockScheduleClient)
		mockClient.On("Close").Return()

		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return mockClient, nil
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/scoreboard/aggregate/start?schedule=300",
			nil,
		)
		req.SetPathValue("schedule", "300")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err := HandleStartAggregateScoreboard()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		requireHandlerErrorHandled(t, rec, err)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestHandleCancelAggregateScoreboardSchedule(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	originalScheduleTemporalClient := scheduleTemporalClient
	defer func() { scheduleTemporalClient = originalScheduleTemporalClient }()

	t.Run("success - cancel existing schedule", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockScheduleClient := &fakeScheduleClient{}
		mockClient.On("ScheduleClient").Return(mockScheduleClient)
		mockClient.On("Close").Return()

		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return mockClient, nil
		}

		req := httptest.NewRequest(
			http.MethodDelete,
			"/api/scoreboard/aggregate/schedule/test-schedule-123",
			nil,
		)
		req.SetPathValue("schedule_id", "test-schedule-123")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err := HandleCancelAggregateScoreboardSchedule()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
		require.True(t, response["success"].(bool))
		require.Equal(t, "Schedule cancelled successfully", response["message"])
		require.Equal(t, "test-schedule-123", response["schedule_id"])
	})

	t.Run("fail - missing schedule_id in path", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodDelete,
			"/api/scoreboard/aggregate/schedule/",
			nil,
		)
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err := HandleCancelAggregateScoreboardSchedule()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		requireHandlerErrorHandled(t, rec, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("fail - schedule not found", func(t *testing.T) {
		mockHandle := &temporalmocks.ScheduleHandle{}
		mockHandle.On("Delete", mock.Anything).
			Return(&serviceerror.NotFound{Message: "schedule not found"})

		mockScheduleClient := &fakeScheduleClient{
			handle: mockHandle,
		}
		mockClient := &temporalmocks.Client{}
		mockClient.On("ScheduleClient").Return(mockScheduleClient)
		mockClient.On("Close").Return()

		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return mockClient, nil
		}

		req := httptest.NewRequest(
			http.MethodDelete,
			"/api/scoreboard/aggregate/schedule/non-existent",
			nil,
		)
		req.SetPathValue("schedule_id", "non-existent")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err := HandleCancelAggregateScoreboardSchedule()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		requireHandlerErrorHandled(t, rec, err)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("fail - temporal client error", func(t *testing.T) {
		scheduleTemporalClient = func(namespace string) (client.Client, error) {
			return nil, errors.New("temporal connection failed")
		}

		req := httptest.NewRequest(
			http.MethodDelete,
			"/api/scoreboard/aggregate/schedule/test-schedule-123",
			nil,
		)
		req.SetPathValue("schedule_id", "test-schedule-123")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err := HandleCancelAggregateScoreboardSchedule()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		requireHandlerErrorHandled(t, rec, err)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
