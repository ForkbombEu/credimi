// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func setupPipelineApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	PipelineTemporalInternalRoutes.Add(app)

	return app
}

// setupPipelineStartApp builds a test app with pipeline start routes.
func setupPipelineStartApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	PipelineRoutes.Add(app)

	return app
}

func TestGetPipelineYAML(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing pipeline_identifier parameter",
			Method:         http.MethodGet,
			URL:            "/api/pipeline/get-yaml",
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"pipeline_identifier"`,
				`"pipeline_identifier is required"`,
			},
			TestAppFactory: setupPipelineApp,
		},
		{
			Name:           "nonexistent pipeline identifier",
			Method:         http.MethodGet,
			URL:            "/api/pipeline/get-yaml?pipeline_identifier=does-not-exist",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"pipeline not found"`,
			},
			TestAppFactory: setupPipelineApp,
		},
		{
			Name:           "valid pipeline identifier",
			Method:         http.MethodGet,
			URL:            "/api/pipeline/get-yaml?pipeline_identifier=usera-s-organization/pipeline123",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`example-yaml-content`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineApp(t)

				coll, err := app.FindCollectionByNameOrId("pipelines")
				require.NoError(t, err)

				record := core.NewRecord(coll)
				record.Set("owner", orgID)
				record.Set("name", "pipeline123")
				record.Set("description", "test-description")
				record.Set(
					"steps",
					map[string]any{"rest-chain": map[string]any{"yaml": "example-yaml-content"}},
				)
				record.Set("yaml", "example-yaml-content")
				require.NoError(t, app.Save(record))

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestSetPipelineExecutionResults(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing request body",
			Method:         http.MethodPost,
			URL:            "/api/pipeline/pipeline-execution-results",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				"pipeline not found",
			},
			TestAppFactory: setupPipelineApp,
		},
		{
			Name:   "valid pipeline execution result",
			Method: http.MethodPost,
			URL:    "/api/pipeline/pipeline-execution-results",
			Body: jsonBody(map[string]any{
				"owner":       "usera-s-organization",
				"pipeline_id": "usera-s-organization/pipeline123",
				"workflow_id": "workflow-xyz",
				"run_id":      "run-001",
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"owner"`,
				`"pipeline"`,
				`"workflow_id"`,
				`"run_id"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineApp(t)

				coll, err := app.FindCollectionByNameOrId("pipelines")
				require.NoError(t, err)

				record := core.NewRecord(coll)
				record.Set("id", "pipeline1234567")
				record.Set("owner", orgID)
				record.Set("name", "pipeline123")
				record.Set("description", "test-description")
				record.Set(
					"steps",
					map[string]any{"rest-chain": map[string]any{"yaml": "example-yaml-content"}},
				)
				record.Set("yaml", "example-yaml-content")
				require.NoError(t, app.Save(record))

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestSetPipelineExecutionResultsIdempotent(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("id", "pipeline1234567")
	record.Set("owner", orgID)
	record.Set("name", "pipeline123")
	record.Set("description", "test-description")
	record.Set(
		"steps",
		map[string]any{"rest-chain": map[string]any{"yaml": "example-yaml-content"}},
	)
	record.Set("yaml", "example-yaml-content")
	require.NoError(t, app.Save(record))

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		body := `{"owner":"usera-s-organization","pipeline_id":"usera-s-organization/pipeline123","workflow_id":"workflow-xyz","run_id":"run-001"}`
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(
				http.MethodPost,
				"/api/pipeline/pipeline-execution-results",
				strings.NewReader(body),
			)
			req.Header.Set("content-type", "application/json")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)
		}

		records, err := app.FindRecordsByFilter(
			"pipeline_results",
			"workflow_id = {:workflow_id} && run_id = {:run_id}",
			"",
			-1,
			0,
			dbx.Params{
				"workflow_id": "workflow-xyz",
				"run_id":      "run-001",
			},
		)
		require.NoError(t, err)
		require.Len(t, records, 1)

		return nil
	})
	require.NoError(t, serveErr)
}

func TestHandleGetPipelineDetailsReturnsResults(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	pipelineColl, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelineColl)
	pipelineRecord.Set("owner", orgID)
	pipelineRecord.Set("name", "pipeline123")
	pipelineRecord.Set("canonified_name", "pipeline123")
	pipelineRecord.Set("description", "demo pipeline")
	pipelineRecord.Set("yaml", "name: demo")
	require.NoError(t, app.Save(pipelineRecord))

	resultsColl, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	resultRecord := core.NewRecord(resultsColl)
	resultRecord.Set("owner", orgID)
	resultRecord.Set("pipeline", pipelineRecord.Id)
	resultRecord.Set("workflow_id", "wf-1")
	resultRecord.Set("run_id", "run-1")
	require.NoError(t, app.Save(resultRecord))

	originalListQueued := pipelineListQueuedRuns
	originalTemporalClient := pipelineTemporalClient
	t.Cleanup(func() {
		pipelineListQueuedRuns = originalListQueued
		pipelineTemporalClient = originalTemporalClient
	})

	pipelineListQueuedRuns = func(ctx context.Context, namespace string) (map[string]QueuedPipelineRunAggregate, error) {
		return map[string]QueuedPipelineRunAggregate{}, nil
	}

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Execution: &common.WorkflowExecution{
					WorkflowId: "wf-1",
					RunId:      "run-1",
				},
				Type:      &common.WorkflowType{Name: "custom-workflow"},
				Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
				StartTime: timestamppb.New(time.Now().Add(-2 * time.Minute)),
				CloseTime: timestamppb.New(time.Now().Add(-1 * time.Minute)),
			},
		}, nil).
		Once()

	callCount := 0
	pipelineTemporalClient = func(namespace string) (client.Client, error) {
		callCount++
		if callCount <= 2 {
			return mockClient, nil
		}
		return nil, errors.New("temporal unavailable")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/list-workflows", nil)
	rec := httptest.NewRecorder()

	err = HandleGetPipelineDetails()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), pipelineRecord.Id)
}

func TestHandleGetPipelineDetailsDescribeNotFound(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	pipelineColl, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelineColl)
	pipelineRecord.Set("owner", orgID)
	pipelineRecord.Set("name", "pipeline123")
	pipelineRecord.Set("canonified_name", "pipeline123")
	pipelineRecord.Set("description", "demo pipeline")
	pipelineRecord.Set("yaml", "name: demo")
	require.NoError(t, app.Save(pipelineRecord))

	resultsColl, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	resultRecord := core.NewRecord(resultsColl)
	resultRecord.Set("owner", orgID)
	resultRecord.Set("pipeline", pipelineRecord.Id)
	resultRecord.Set("workflow_id", "wf-1")
	resultRecord.Set("run_id", "run-1")
	require.NoError(t, app.Save(resultRecord))

	originalListQueued := pipelineListQueuedRuns
	originalTemporalClient := pipelineTemporalClient
	t.Cleanup(func() {
		pipelineListQueuedRuns = originalListQueued
		pipelineTemporalClient = originalTemporalClient
	})

	pipelineListQueuedRuns = func(ctx context.Context, namespace string) (map[string]QueuedPipelineRunAggregate, error) {
		return map[string]QueuedPipelineRunAggregate{}, nil
	}

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return((*workflowservice.DescribeWorkflowExecutionResponse)(nil), serviceerror.NewNotFound("missing")).
		Once()

	pipelineTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/list-workflows", nil)
	rec := httptest.NewRecorder()

	err = HandleGetPipelineDetails()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleGetPipelineDetailsDescribeInvalidArgument(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	pipelineColl, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelineColl)
	pipelineRecord.Set("owner", orgID)
	pipelineRecord.Set("name", "pipeline123")
	pipelineRecord.Set("canonified_name", "pipeline123")
	pipelineRecord.Set("description", "demo pipeline")
	pipelineRecord.Set("yaml", "name: demo")
	require.NoError(t, app.Save(pipelineRecord))

	resultsColl, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	resultRecord := core.NewRecord(resultsColl)
	resultRecord.Set("owner", orgID)
	resultRecord.Set("pipeline", pipelineRecord.Id)
	resultRecord.Set("workflow_id", "wf-1")
	resultRecord.Set("run_id", "run-1")
	require.NoError(t, app.Save(resultRecord))

	originalListQueued := pipelineListQueuedRuns
	originalTemporalClient := pipelineTemporalClient
	t.Cleanup(func() {
		pipelineListQueuedRuns = originalListQueued
		pipelineTemporalClient = originalTemporalClient
	})

	pipelineListQueuedRuns = func(ctx context.Context, namespace string) (map[string]QueuedPipelineRunAggregate, error) {
		return map[string]QueuedPipelineRunAggregate{}, nil
	}

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return((*workflowservice.DescribeWorkflowExecutionResponse)(nil), serviceerror.NewInvalidArgument("bad id")).
		Once()

	pipelineTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/list-workflows", nil)
	rec := httptest.NewRecorder()

	err = HandleGetPipelineDetails()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleGetPipelineDetailsQueuedRunsError(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	pipelineColl, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelineColl)
	pipelineRecord.Set("owner", orgID)
	pipelineRecord.Set("name", "pipeline123")
	pipelineRecord.Set("canonified_name", "pipeline123")
	pipelineRecord.Set("description", "demo pipeline")
	pipelineRecord.Set("yaml", "name: demo")
	require.NoError(t, app.Save(pipelineRecord))

	originalListQueued := pipelineListQueuedRuns
	t.Cleanup(func() {
		pipelineListQueuedRuns = originalListQueued
	})

	pipelineListQueuedRuns = func(ctx context.Context, namespace string) (map[string]QueuedPipelineRunAggregate, error) {
		return nil, errors.New("boom")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/list-workflows", nil)
	rec := httptest.NewRecorder()

	err = HandleGetPipelineDetails()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleGetPipelineSpecificDetailsQueuedOnly(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	pipelineColl, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelineColl)
	pipelineRecord.Set("owner", orgID)
	pipelineRecord.Set("name", "pipeline123")
	pipelineRecord.Set("canonified_name", "pipeline123")
	pipelineRecord.Set("description", "demo pipeline")
	pipelineRecord.Set("yaml", "name: demo")
	require.NoError(t, app.Save(pipelineRecord))

	originalListQueued := pipelineListQueuedRuns
	t.Cleanup(func() {
		pipelineListQueuedRuns = originalListQueued
	})

	pipelineListQueuedRuns = func(ctx context.Context, namespace string) (map[string]QueuedPipelineRunAggregate, error) {
		return map[string]QueuedPipelineRunAggregate{
			"ticket-1": {
				TicketID:           "ticket-1",
				PipelineIdentifier: "usera-s-organization/pipeline123",
				EnqueuedAt:         time.Now().Add(-1 * time.Minute),
				LeaderRunnerID:     "runner-1",
				RequiredRunnerIDs:  []string{"runner-1"},
				RunnerIDs:          []string{"runner-1"},
			},
		}, nil
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/pipeline/list-workflows/"+pipelineRecord.Id,
		nil,
	)
	req.SetPathValue("id", pipelineRecord.Id)
	rec := httptest.NewRecorder()

	err = HandleGetPipelineSpecificDetails()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.NotEmpty(t, response)
	queue, ok := response[0]["queue"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "ticket-1", queue["ticket_id"])
}

func TestProcessPipelineResultsClientError(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	origClient := pipelineTemporalClient
	t.Cleanup(func() { pipelineTemporalClient = origClient })

	pipelineTemporalClient = func(_ string) (client.Client, error) {
		return nil, errors.New("no client")
	}

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	record.Set("workflow_id", "wf-1")
	record.Set("run_id", "run-1")

	var tc client.Client
	_, err = processPipelineResults("ns", []*core.Record{record}, &tc)
	require.Error(t, err)

	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusInternalServerError, apiErr.Code)
}

func TestProcessPipelineResultsDescribeErrors(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	record.Set("workflow_id", "wf-1")
	record.Set("run_id", "run-1")

	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{
			name:     "not found",
			err:      &serviceerror.NotFound{Message: "missing"},
			wantCode: http.StatusNotFound,
		},
		{
			name:     "invalid argument",
			err:      &serviceerror.InvalidArgument{Message: "bad"},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "generic",
			err:      errors.New("boom"),
			wantCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origClient := pipelineTemporalClient
			t.Cleanup(func() { pipelineTemporalClient = origClient })

			mockClient := temporalmocks.NewClient(t)
			mockClient.
				On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
				Return(nil, tc.err)

			pipelineTemporalClient = func(_ string) (client.Client, error) {
				return mockClient, nil
			}

			var temporalClient client.Client
			_, err := processPipelineResults("ns", []*core.Record{record}, &temporalClient)
			require.Error(t, err)
			var apiErr *apierror.APIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, tc.wantCode, apiErr.Code)
		})
	}
}

func TestProcessPipelineResultsParentExecution(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	origClient := pipelineTemporalClient
	t.Cleanup(func() { pipelineTemporalClient = origClient })

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	record.Set("workflow_id", "wf-1")
	record.Set("run_id", "run-1")

	mockClient := temporalmocks.NewClient(t)
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Execution: &common.WorkflowExecution{WorkflowId: "wf-1", RunId: "run-1"},
				ParentExecution: &common.WorkflowExecution{
					WorkflowId: "parent-wf",
					RunId:      "parent-run",
				},
				Type: &common.WorkflowType{Name: "Pipeline"},
			},
		}, nil)

	pipelineTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	var temporalClient client.Client
	execs, err := processPipelineResults("ns", []*core.Record{record}, &temporalClient)
	require.NoError(t, err)
	require.Len(t, execs, 1)
	require.NotNil(t, execs[0].ParentExecution)
	require.Equal(t, "parent-wf", execs[0].ParentExecution.WorkflowID)
	require.Equal(t, "parent-run", execs[0].ParentExecution.RunID)
}

func TestProcessPipelineResultsSetsTemporalClient(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	origClient := pipelineTemporalClient
	t.Cleanup(func() { pipelineTemporalClient = origClient })

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	record.Set("workflow_id", "wf-1")
	record.Set("run_id", "run-1")

	mockClient := temporalmocks.NewClient(t)
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Execution: &common.WorkflowExecution{WorkflowId: "wf-1", RunId: "run-1"},
				Type:      &common.WorkflowType{Name: "Pipeline"},
			},
		}, nil)

	pipelineTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	var temporalClient client.Client
	_, err = processPipelineResults("ns", []*core.Record{record}, &temporalClient)
	require.NoError(t, err)
	require.NotNil(t, temporalClient)
}

func TestSelectTopExecutionsByPipeline(t *testing.T) {
	executions := []struct {
		pipelineID string
		execution  *WorkflowExecutionSummary
	}{
		{
			pipelineID: "pipeline-1",
			execution: &WorkflowExecutionSummary{
				Status:    string(WorkflowStatusCompleted),
				StartTime: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			pipelineID: "pipeline-1",
			execution: &WorkflowExecutionSummary{
				Status:    string(WorkflowStatusRunning),
				StartTime: time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
			},
		},
		{
			pipelineID: "pipeline-1",
			execution: &WorkflowExecutionSummary{
				Status:    string(WorkflowStatusCompleted),
				StartTime: time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
			},
		},
	}

	selected := selectTopExecutionsByPipeline(executions, 2)
	require.Len(t, selected["pipeline-1"], 2)
	require.ElementsMatch(
		t,
		[]string{string(WorkflowStatusRunning), string(WorkflowStatusCompleted)},
		[]string{selected["pipeline-1"][0].Status, selected["pipeline-1"][1].Status},
	)
}

func TestBuildChildWorkflowParentQuery(t *testing.T) {
	require.Equal(t, "", buildChildWorkflowParentQuery(nil))

	query := buildChildWorkflowParentQuery([]workflowExecutionRef{
		{
			WorkflowID: "parent-workflow-1",
			RunID:      "run-1",
		},
		{
			WorkflowID: `parent"workflow\2`,
			RunID:      `run"2\`,
		},
	})

	require.Equal(
		t,
		`(ParentWorkflowId="parent-workflow-1" AND ParentRunId="run-1") OR `+
			`(ParentWorkflowId="parent\"workflow\\2" AND ParentRunId="run\"2\\")`,
		query,
	)
}
