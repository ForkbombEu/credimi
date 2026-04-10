// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
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
			workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url": "https://credimi.test",
				},
			},
			capturedInput,
		)

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

func TestHandleGetPipelineScoreboardMissingNamespace(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/pipeline/scoreboard/",
		nil,
	)
	rec := httptest.NewRecorder()

	err := HandleGetPipelineScoreboard()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Contains(t, resp["message"], "please provide a namespace in the path")
}
func TestHandleGetPipelineScoreboard(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	createRunnerRecord(t, app, orgID, "runner-android", "")
	createRunnerRecord(t, app, orgID, "runner-ios", "")
	createRunnerRecord(t, app, orgID, "runner-default", "")

	pipeline1 := createPipelineRecord(t, app, orgID, "Android E2E Tests")
	pipeline2 := createPipelineRecord(t, app, orgID, "iOS E2E Tests")
	pipeline3 := createPipelineRecord(t, app, orgID, "iOS E3E Tests")

	pipeline1.Set("published", true)
	require.NoError(t, app.Save(pipeline1))
	pipeline2.Set("published", true)
	require.NoError(t, app.Save(pipeline2))
	pipeline3.Set("published", true)
	require.NoError(t, app.Save(pipeline3))

	pipeline1Canonified := pipeline1.GetString("canonified_name")
	pipeline2Canonified := pipeline2.GetString("canonified_name")
	pipeline3Canonified := pipeline3.GetString("canonified_name")

	namespace := "usera-s-organization"

	mockClient := &temporalmocks.Client{}

	now := time.Now()
	exec1 := buildPipelineExecutionInfoWithRunner(
		t,
		"Pipeline-Sched-wf-1",
		"run-1",
		fmt.Sprintf("%s/%s", namespace, pipeline1Canonified),
		"Completed",
		[]string{"usera-s-organization/runner-android"},
		now.Add(-2*time.Hour).Add(-2*time.Minute).Add(-33*time.Second),
		now.Add(-2*time.Hour),
	)
	exec2 := buildPipelineExecutionInfoWithRunner(
		t,
		"wf-2",
		"run-2",
		fmt.Sprintf("%s/%s", namespace, pipeline1Canonified),
		"Completed",
		[]string{"usera-s-organization/runner-android"},
		now.Add(-1*time.Hour).Add(-1*time.Minute).Add(-45*time.Second),
		now.Add(-1*time.Hour),
	)
	exec3 := buildPipelineExecutionInfoWithRunner(
		t,
		"wf-3",
		"run-3",
		fmt.Sprintf("%s/%s", namespace, pipeline1Canonified),
		"Failed",
		[]string{"usera-s-organization/runner-ios"},
		now.Add(-30*time.Minute).Add(-30*time.Second),
		now.Add(-30*time.Minute),
	)

	exec4 := buildPipelineExecutionInfoWithRunner(
		t,
		"Pipeline-Sched-wf-4",
		"run-4",
		fmt.Sprintf("%s/%s", namespace, pipeline2Canonified),
		"Completed",
		[]string{"usera-s-organization/runner-ios", "usera-s-organization/runner-default"},
		now.Add(-5*time.Minute).Add(-10*time.Second),
		now.Add(-1*time.Minute),
	)
	exec5 := buildPipelineExecutionInfoWithRunner(
		t,
		"wf-5",
		"run-5",
		fmt.Sprintf("%s/%s", namespace, pipeline3Canonified),
		"Completed",
		[]string{"usera-s-organization/runner-ios", "usera-s-organization/runner-default"},
		now.Add(-2*time.Hour).Add(-5*time.Minute).Add(-10*time.Second),
		now.Add(-1*time.Minute),
	)
	mockClient.
		On("ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return !strings.Contains(req.GetQuery(), "ParentWorkflowId")
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				exec1.Info, exec2.Info, exec3.Info, exec4.Info, exec5.Info,
			},
		}, nil).
		Once()

	originalClient := pipelineResultsTemporalClient
	defer func() { pipelineResultsTemporalClient = originalClient }()
	pipelineResultsTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/pipeline/scoreboard/"+namespace,
		nil,
	)
	req.SetPathValue("namespace", namespace)
	rec := httptest.NewRecorder()

	err = HandleGetPipelineScoreboard()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response []PipelineStatsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))

	require.Len(t, response, 3)

	var stats1, stats2, stats3 *PipelineStatsResponse
	for i := range response {
		if response[i].PipelineName == "Android E2E Tests" {
			stats1 = &response[i]
		}
		if response[i].PipelineName == "iOS E2E Tests" {
			stats2 = &response[i]
		}
		if response[i].PipelineName == "iOS E3E Tests" {
			stats3 = &response[i]
		}
	}

	require.NotNil(t, stats1)
	require.Equal(t, 3, stats1.TotalRuns)
	require.Equal(t, 2, stats1.TotalSuccesses)
	require.Equal(t, 1, stats1.ScheduledExecutions)
	require.Equal(t, 2, stats1.ManualExecutions)
	require.ElementsMatch(
		t,
		[]string{"usera-s-organization/runner-android", "usera-s-organization/runner-ios"},
		stats1.Runners,
	)
	require.Equal(t, "1m45s", stats1.MinExecutionTime)
	expectedFirstTime := exec1.Info.GetStartTime().AsTime()
	actualFirstTime, err := time.Parse(time.RFC3339Nano, stats1.FirstExecutionDate)
	require.NoError(t, err)
	require.WithinDuration(t, expectedFirstTime, actualFirstTime, time.Second)
	expectedLastTime := exec3.Info.GetStartTime().AsTime()
	actualLastTime, err := time.Parse(time.RFC3339Nano, stats1.LastExecutionDate)
	require.NoError(t, err)
	require.WithinDuration(t, expectedLastTime, actualLastTime, time.Second)
	require.Equal(t, 66.67, stats1.SuccessRate)

	require.NotNil(t, stats1.LastSuccessfulRun, "LastSuccessfulRun should not be nil")
	require.Equal(t, "wf-2", stats1.LastSuccessfulRun.WorkflowID)
	require.Equal(t, "run-2", stats1.LastSuccessfulRun.RunID)
	require.NotEmpty(t, stats1.LastSuccessfulRun.StartTime)

	require.NotNil(t, stats2)
	require.Equal(t, 1, stats2.TotalRuns)
	require.Equal(t, 1, stats2.TotalSuccesses)
	require.Equal(t, 1, stats2.ScheduledExecutions)
	require.Equal(t, 0, stats2.ManualExecutions)
	require.ElementsMatch(
		t,
		[]string{"usera-s-organization/runner-ios", "usera-s-organization/runner-default"},
		stats2.Runners,
	)
	require.Equal(t, "4m10s", stats2.MinExecutionTime)
	expectedTime2 := exec4.Info.GetStartTime().AsTime()
	actualTime2, err := time.Parse(time.RFC3339Nano, stats2.FirstExecutionDate)
	require.NoError(t, err)
	require.WithinDuration(t, expectedTime2, actualTime2, time.Second)
	require.Equal(t, stats2.FirstExecutionDate, stats2.LastExecutionDate)
	require.Equal(t, 100.00, stats2.SuccessRate)
	require.NotNil(t, stats2.LastSuccessfulRun, "LastSuccessfulRun should not be nil")
	require.Equal(t, "Pipeline-Sched-wf-4", stats2.LastSuccessfulRun.WorkflowID)
	require.Equal(t, "run-4", stats2.LastSuccessfulRun.RunID)

	require.Equal(t, "2h4m10s", stats3.MinExecutionTime)
	require.NotNil(t, stats3.LastSuccessfulRun, "LastSuccessfulRun should not be nil")
	require.Equal(t, "wf-5", stats3.LastSuccessfulRun.WorkflowID)
	require.Equal(t, "run-5", stats3.LastSuccessfulRun.RunID)

	mockClient.AssertExpectations(t)
}

func TestHandleGetExecutionDetails(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	createRunnerRecord(t, app, orgID, "runner-android", "")
	createRunnerRecord(t, app, orgID, "runner-ios", "")
	createRunnerRecord(t, app, orgID, "runner-default", "")

	pipeline1 := createPipelineRecord(t, app, orgID, "Android E2E Tests")
	pipeline2 := createPipelineRecord(t, app, orgID, "iOS E2E Tests")

	pipeline1.Set("published", true)
	require.NoError(t, app.Save(pipeline1))
	pipeline2.Set("published", true)
	require.NoError(t, app.Save(pipeline2))

	pipeline1Canonified := pipeline1.GetString("canonified_name")
	pipeline2Canonified := pipeline2.GetString("canonified_name")

	namespace := "usera-s-organization"

	now := time.Now()

	exec2 := buildPipelineExecutionInfoWithRunner(
		t,
		"wf-2",
		"run-2",
		fmt.Sprintf("%s/%s", namespace, pipeline1Canonified),
		"Completed",
		[]string{"usera-s-organization/runner-android"},
		now.Add(-1*time.Hour).Add(-1*time.Minute).Add(-45*time.Second),
		now.Add(-1*time.Hour),
	)
	addEntitySearchAttributes(exec2.Info, map[string]any{
		workflowengine.VersionsSearchAttribute: []string{"org/wallet/v1-0-0"},
		workflowengine.ActionsSearchAttribute: []string{
			"org/action/maestro-1",
			"org/action/maestro-2",
		},
		workflowengine.CredentialsSearchAttribute: []string{
			"org/issuer/credential-1",
			"org/issuer/credential-2",
		},
		workflowengine.UseCaseSearchAttribute: []string{
			"org/verifier/uc-1",
			"org/verifier/uc-2",
		},
		workflowengine.ConformanceCheckSearchAttribute: []string{"conformance/check-1"},
		workflowengine.CustomCheckSearchAttribute:      []string{"custom/check-1"},
	})

	exec4 := buildPipelineExecutionInfoWithRunner(
		t,
		"Pipeline-Sched-wf-4",
		"run-4",
		fmt.Sprintf("%s/%s", namespace, pipeline2Canonified),
		"Completed",
		[]string{"usera-s-organization/runner-ios", "usera-s-organization/runner-default"},
		now.Add(-5*time.Minute).Add(-10*time.Second),
		now.Add(-1*time.Minute),
	)
	addEntitySearchAttributes(exec4.Info, map[string]any{
		workflowengine.VersionsSearchAttribute: []string{
			"org/wallet/v2-0-0",
			"org/wallet/v3-0-0",
		},
		workflowengine.CredentialsSearchAttribute: []string{"org/issuer/credential-3"},
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("ListWorkflow",
			mock.Anything,
			mock.Anything,
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil).
		Maybe()

	mockClient.
		On("DescribeWorkflowExecution",
			mock.Anything,
			"wf-2",
			"run-2",
		).
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: exec2.Info,
		}, nil).
		Once()

	mockClient.
		On("DescribeWorkflowExecution",
			mock.Anything,
			"Pipeline-Sched-wf-4",
			"run-4",
		).
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: exec4.Info,
		}, nil).
		Once()

	mockClient.
		On("DescribeWorkflowExecution",
			mock.Anything,
			"non-existent",
			"non-existent",
		).
		Return(nil, fmt.Errorf("workflow not found")).
		Once()

	originalClient := pipelineResultsTemporalClient
	defer func() { pipelineResultsTemporalClient = originalClient }()
	pipelineResultsTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	t.Run("execution details for exec2", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/pipeline/execution-details/"+namespace+"/wf-2/run-2",
			nil,
		)
		req.SetPathValue("namespace", namespace)
		req.SetPathValue("workflow_id", "wf-2")
		req.SetPathValue("run_id", "run-2")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err = HandleGetExecutionDetails()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)

		var details LastExecutionDetails
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&details))

		require.Equal(t, "android-e2e-tests", details.PipelineName)
		require.Empty(t, details.OrgLogo)

		require.Empty(t, details.Video)
		require.Empty(t, details.Screenshots)
		require.Empty(t, details.Logs)

		require.ElementsMatch(t, []string{"org/wallet"}, details.WalletUsed)
		require.ElementsMatch(t, []string{"org/wallet/v1-0-0"}, details.WalletVersionUsed)
		require.ElementsMatch(
			t,
			[]string{"org/action/maestro-1", "org/action/maestro-2"},
			details.MaestroScripts,
		)
		require.ElementsMatch(
			t,
			[]string{"org/issuer/credential-1", "org/issuer/credential-2"},
			details.Credentials,
		)
		require.ElementsMatch(t, []string{"org/issuer"}, details.Issuers)
		require.ElementsMatch(
			t,
			[]string{"org/verifier/uc-1", "org/verifier/uc-2"},
			details.UseCaseVerifications,
		)
		require.ElementsMatch(t, []string{"org/verifier"}, details.Verifiers)
		require.ElementsMatch(t, []string{"conformance/check-1"}, details.ConformanceTests)
		require.ElementsMatch(t, []string{"custom/check-1"}, details.CustomChecks)
	})

	t.Run("execution details for exec4", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/pipeline/execution-details/"+namespace+"/Pipeline-Sched-wf-4/run-4",
			nil,
		)
		req.SetPathValue("namespace", namespace)
		req.SetPathValue("workflow_id", "Pipeline-Sched-wf-4")
		req.SetPathValue("run_id", "run-4")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err = HandleGetExecutionDetails()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)

		var details LastExecutionDetails
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&details))

		require.Equal(t, "ios-e2e-tests", details.PipelineName)

		require.ElementsMatch(t, []string{"org/wallet"}, details.WalletUsed)
		require.ElementsMatch(
			t,
			[]string{"org/wallet/v2-0-0", "org/wallet/v3-0-0"},
			details.WalletVersionUsed,
		)
		require.ElementsMatch(t, []string{"org/issuer/credential-3"}, details.Credentials)
		require.ElementsMatch(t, []string{"org/issuer"}, details.Issuers)
	})
	t.Run("missing namespace", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/pipeline/execution-details///",
			nil,
		)
		rec := httptest.NewRecorder()

		err = HandleGetExecutionDetails()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("workflow not found", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/pipeline/execution-details/"+namespace+"/non-existent/non-existent",
			nil,
		)
		req.SetPathValue("namespace", namespace)
		req.SetPathValue("workflow_id", "non-existent")
		req.SetPathValue("run_id", "non-existent")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()

		err = HandleGetExecutionDetails()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	mockClient.AssertExpectations(t)
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
	runnerIDs []string,
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

	if len(runnerIDs) > 0 {
		payload, err := converter.GetDefaultDataConverter().ToPayload(runnerIDs)
		require.NoError(t, err)
		indexedFields[workflowengine.RunnerIdentifiersSearchAttribute] = payload
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

func createRunnerRecord(t testing.TB, app *tests.TestApp, orgID, name, runnerType string) {
	runnersColl, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)

	runner := core.NewRecord(runnersColl)
	runner.Set("name", name)
	runner.Set("type", runnerType)
	runner.Set("owner", orgID)
	runner.Set("ip", "my_ip")
	require.NoError(t, app.Save(runner))
}

func TestSaveScoreboardResults(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	
	pipeline := createPipelineRecord(t, app, orgID, "Test Pipeline")
	pipeline.Set("published", true)
	require.NoError(t, app.Save(pipeline))
	
	createRunnerRecord(t, app, orgID, "test-runner", "")
	createPipelineResult(t, app, orgID, pipeline.Id, "wf-new", "run-new")

	t.Run("success - saves results correctly", func(t *testing.T) {
		aggregatedPipelines := []workflows.AggregatedPipelineStats{
			{
				PipelineID:          pipeline.Id,
				PipelineName:        "Test Pipeline",
				RunnerTypes:         []string{},
				Runners:             []string{"usera-s-organization/test-runner"},
				TotalRuns:           10,
				TotalSuccesses:      8,
				SuccessRate:         80.0,
				ManualExecutions:    5,
				ScheduledExecutions: 5,
				MinExecutionTime:    "1m30s",
				FirstExecutionDate:  "2024-01-01T00:00:00Z",
				LastExecutionDate:   "2024-01-02T00:00:00Z",
				LastExecution: &workflows.LatestExecutionDetails{
					PipelineName: "Test Pipeline",
					WorkflowID: "wf-new",
					RunID:      "run-new",
				},
			},
		}

		requestBody := SaveScoreboardResultsRequest{
			AggregatedPipelines: aggregatedPipelines,
		}
		bodyBytes, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/scoreboard/save-results",
			strings.NewReader(string(bodyBytes)),
		)
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		err = HandleSaveScoreboardResults()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)

		var response SaveScoreboardResultsResponse
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
		require.True(t, response.Success)
		require.Equal(t, 1, response.RecordsCount)
		
		collection, err := app.FindCollectionByNameOrId("pipeline_results_aggegrates")
		require.NoError(t, err)

		records, err := app.FindRecordsByFilter(collection.Id, "", "", -1, 0)
		require.NoError(t, err)
		require.Len(t, records, 1)

		record := records[0]
		require.Equal(t, pipeline.Id, record.GetString("pipeline"))
		require.Equal(t, 10, record.GetInt("total_runs"))
		require.Equal(t, 8, record.GetInt("total_successes"))
		require.Equal(t, 5, record.GetInt("manually_executed_runs"))
		require.Equal(t, 5, record.GetInt("scheduled_runs"))
		require.Equal(t, "1m30s", record.GetString("minimum_running_time"))

		runnerIDs := record.GetStringSlice("mobile_runners")
		require.Len(t, runnerIDs, 1)

		runnerRecord, err := app.FindRecordById("mobile_runners", runnerIDs[0])
		require.NoError(t, err)
		require.Equal(t, "test-runner", runnerRecord.GetString("name"))

		latestExecutionID := record.GetString("latest_execution")
		require.NotEmpty(t, latestExecutionID, "latest_execution should not be empty")

		executionRecord, err := app.FindRecordById("pipeline_results", latestExecutionID)
		require.NoError(t, err)
		require.Equal(t, "wf-new", executionRecord.GetString("workflow_id"))
		require.Equal(t, "run-new", executionRecord.GetString("run_id"))
	})
	t.Run("fail - invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/scoreboard/save-results",
			strings.NewReader("invalid json {{{{"),
		)
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		err := HandleSaveScoreboardResults()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("fail - empty aggregated pipelines", func(t *testing.T) {
		requestBody := SaveScoreboardResultsRequest{
			AggregatedPipelines: []workflows.AggregatedPipelineStats{},
		}
		bodyBytes, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/scoreboard/save-results",
			strings.NewReader(string(bodyBytes)),
		)
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		err = HandleSaveScoreboardResults()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("fail - missing API key", func(t *testing.T) {
		requestBody := SaveScoreboardResultsRequest{
			AggregatedPipelines: []workflows.AggregatedPipelineStats{
				{PipelineID: "test", PipelineName: "Test"},
			},
		}
		bodyBytes, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/scoreboard/save-results",
			strings.NewReader(string(bodyBytes)),
		)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		err = HandleSaveScoreboardResults()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("fail - pipeline not found", func(t *testing.T) {
		aggregatedPipelines := []workflows.AggregatedPipelineStats{
			{
				PipelineID:   "non-existent-pipeline-id",
				PipelineName: "Non Existent Pipeline",
				TotalRuns:    10,
			},
		}

		requestBody := SaveScoreboardResultsRequest{
			AggregatedPipelines: aggregatedPipelines,
		}
		bodyBytes, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/scoreboard/save-results",
			strings.NewReader(string(bodyBytes)),
		)
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		err = HandleSaveScoreboardResults()(&core.RequestEvent{
			App: app,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestFindRunners(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	
	createRunnerRecord(t, app, orgID, "existing-runner", "")

	t.Run("success - existing runners", func(t *testing.T) {
		runnerNames := []string{
			"usera-s-organization/existing-runner",
		}
		ids, err := findRunners(app, runnerNames)
		require.NoError(t, err)
		require.Len(t, ids, 1)
	})

	t.Run("fail - invalid runner format (no slash)", func(t *testing.T) {
		runnerNames := []string{
			"invalid-format-no-slash",
		}
		ids, err := findRunners(app, runnerNames)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid runner format")
		require.Empty(t, ids)
	})

	t.Run("fail - invalid runner format (multiple slashes)", func(t *testing.T) {
		runnerNames := []string{
			"owner/name/extra",
		}
		ids, err := findRunners(app, runnerNames)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid runner format")
		require.Empty(t, ids)
	})

	t.Run("success - empty runner list", func(t *testing.T) {
		ids, err := findRunners(app, []string{})
		require.NoError(t, err)
		require.Empty(t, ids)
	})

	t.Run("success - non-existent runners (not created)", func(t *testing.T) {
		runnerNames := []string{
			"usera-s-organization/non-existent-runner-1",
			"usera-s-organization/non-existent-runner-2",
		}
		ids, err := findRunners(app, runnerNames)
		require.NoError(t, err)
		require.Empty(t, ids)
	})
}
