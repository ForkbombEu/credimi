// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func decodeAPIError(t testing.TB, rec *httptest.ResponseRecorder) apierror.APIError {
	t.Helper()
	var apiErr apierror.APIError
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&apiErr))
	return apiErr
}

func TestHandleGetMyCheckRunRequiresAuth(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/my/checks/checks-1/runs/run-1",
		nil,
	)
	req.SetPathValue("checkId", "checks-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleGetMyCheckRun()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusUnauthorized, apiErr.Code)
	require.Equal(t, "authentication required", apiErr.Reason)
}

func TestHandleGetMyCheckRunMissingRunID(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/my/checks/checks-1/runs/",
		nil,
	)
	req.SetPathValue("checkId", "checks-1")
	rec := httptest.NewRecorder()

	err = HandleGetMyCheckRun()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "runId is required", apiErr.Reason)
}

func TestHandleListMyCheckRunsMissingCheckID(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks//runs", nil)
	rec := httptest.NewRecorder()

	err = HandleListMyCheckRuns()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "checkId is required", apiErr.Reason)
}

func TestHandleGetMyCheckRunHistoryMissingParams(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/checks-1/runs/", nil)
	req.SetPathValue("checkId", "")
	req.SetPathValue("runId", "")
	ar := httptest.NewRecorder()

	err = HandleGetMyCheckRunHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: ar,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, ar.Code)

	apiErr := decodeAPIError(t, ar)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "checkId and runId are required", apiErr.Reason)
}

func TestShouldIncludeQueuedRuns(t *testing.T) {
	require.True(t, shouldIncludeQueuedRuns(""))
	require.True(t, shouldIncludeQueuedRuns("running"))
	require.True(t, shouldIncludeQueuedRuns("Completed, RUNNING"))
	require.False(t, shouldIncludeQueuedRuns("completed"))
}

func TestBuildQueuedWorkflowSummary(t *testing.T) {
	now := time.Date(2026, 2, 17, 12, 0, 0, 0, time.UTC)
	queued := QueuedPipelineRunAggregate{
		TicketID:   "ticket-1",
		Position:   0,
		LineLen:    3,
		RunnerIDs:  []string{"runner-1", "runner-2"},
		EnqueuedAt: now,
	}

	summary := buildQueuedWorkflowSummary(queued, "UTC", "display-name")
	require.Equal(t, "queue/ticket-1", summary.Execution.WorkflowID)
	require.Equal(t, "ticket-1", summary.Execution.RunID)
	require.Equal(t, "display-name", summary.DisplayName)
	require.Equal(t, string(WorkflowStatusQueued), summary.Status)
	require.Equal(t, 1, summary.Queue.Position)
	require.Equal(t, 3, summary.Queue.LineLen)
	require.Equal(t, []string{"runner-1", "runner-2"}, summary.Queue.RunnerIDs)
}

func TestResolveQueuedPipelineDisplayNameFallback(t *testing.T) {
	require.Equal(t, "pipeline-run", resolveQueuedPipelineDisplayName(nil, ""))
	require.Equal(t, "custom-id", resolveQueuedPipelineDisplayName(nil, "custom-id"))
}

func TestHandleCancelMyCheckRunNotFound(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := checksTemporalClient
	t.Cleanup(func() {
		checksTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("CancelWorkflow", mock.Anything, "check-1", "run-1").
		Return(&serviceerror.NotFound{}).
		Once()

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/my/checks/check-1/runs/run-1", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleCancelMyCheckRun()(&core.RequestEvent{
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

func TestHandleTerminateMyCheckRunSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := checksTemporalClient
	t.Cleanup(func() {
		checksTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("TerminateWorkflow", mock.Anything, "check-2", "run-2", "Terminated by user", mock.Anything).
		Return(nil).
		Once()

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks/check-2/runs/run-2/terminate", nil)
	req.SetPathValue("checkId", "check-2")
	req.SetPathValue("runId", "run-2")
	rec := httptest.NewRecorder()

	err = HandleTerminateMyCheckRun()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "\"status\":\"terminated\"")
}

func TestHandleExportMyCheckRunSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := checksTemporalClient
	origInput := checksGetWorkflowInput
	t.Cleanup(func() {
		checksTemporalClient = origClient
		checksGetWorkflowInput = origInput
	})

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	checksGetWorkflowInput = func(checkID string, runID string, c client.Client) (workflowengine.WorkflowInput, error) {
		return workflowengine.WorkflowInput{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-3/runs/run-3/export", nil)
	req.SetPathValue("checkId", "check-3")
	req.SetPathValue("runId", "run-3")
	rec := httptest.NewRecorder()

	err = HandleExportMyCheckRun()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "\"export\"")
}

func TestHandleMyCheckLogsStart(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := checksTemporalClient
	t.Cleanup(func() {
		checksTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "check-4", "run-4").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{}, nil).
		Once()
	mockClient.
		On("SignalWorkflow", mock.Anything, "check-4", "run-4", "start-logs", struct{}{}).
		Return(nil).
		Once()

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-4/runs/run-4/logs?action=start", nil)
	req.SetPathValue("checkId", "check-4")
	req.SetPathValue("runId", "run-4")
	rec := httptest.NewRecorder()

	err = HandleMyCheckLogs()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "\"status\":\"started\"")
}

func TestHandleRerunMyCheckSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := checksTemporalClient
	origInput := checksGetWorkflowInput
	origStart := checksStartWorkflowWithOptions
	t.Cleanup(func() {
		checksTemporalClient = origClient
		checksGetWorkflowInput = origInput
		checksStartWorkflowWithOptions = origStart
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "check-5", "run-5").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Type:      &common.WorkflowType{Name: "wf-type"},
				TaskQueue: "task-queue",
			},
			ExecutionConfig: &workflow.WorkflowExecutionConfig{
				WorkflowRunTimeout:       durationpb.New(2 * time.Minute),
				WorkflowExecutionTimeout: durationpb.New(5 * time.Minute),
			},
		}, nil).
		Once()

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}
	checksGetWorkflowInput = func(checkID string, runID string, c client.Client) (workflowengine.WorkflowInput, error) {
		return workflowengine.WorkflowInput{
			Payload: map[string]any{"foo": "bar"},
			Config:  map[string]any{"app_url": "https://app"},
		}, nil
	}
	checksStartWorkflowWithOptions = func(
		namespace string,
		options client.StartWorkflowOptions,
		workflowName string,
		input workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-new",
			WorkflowRunID: "run-new",
		}, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks/check-5/runs/run-5/rerun", nil)
	req.SetPathValue("checkId", "check-5")
	req.SetPathValue("runId", "run-5")
	rec := httptest.NewRecorder()

	err = HandleRerunMyCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "\"workflow_id\":\"wf-new\"")
	require.Contains(t, rec.Body.String(), "\"run_id\":\"run-new\"")
}

func TestGetWorkflowInputSuccess(t *testing.T) {
	mockClient := &temporalmocks.Client{}

	inputData := map[string]any{
		"Payload": map[string]any{"foo": "bar"},
		"Config":  map[string]any{"app_url": "https://app"},
	}
	raw, err := json.Marshal(inputData)
	require.NoError(t, err)
	payloads := &common.Payloads{
		Payloads: []*common.Payload{
			{Data: raw},
		},
	}

	iter := &fakeHistoryIterator{
		events: []*historypb.HistoryEvent{
			{
				EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
				Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
					WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
						Input: payloads,
					},
				},
			},
		},
	}

	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-1",
			"run-1",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(iter).
		Once()

	got, err := getWorkflowInput("wf-1", "run-1", mockClient)
	require.NoError(t, err)
	payload, ok := got.Payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "bar", payload["foo"])
	require.Equal(t, "https://app", got.Config["app_url"])
}

func TestHandleListMyChecksStatusFilterQuery(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := listChecksTemporalClient
	origList := listChecksWorkflows
	t.Cleanup(func() {
		listChecksTemporalClient = origClient
		listChecksWorkflows = origList
	})

	mockClient := &temporalmocks.Client{}
	listChecksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	var capturedQuery string
	listChecksWorkflows = func(ctx context.Context, c client.Client, namespace string, query string) (*workflowservice.ListWorkflowExecutionsResponse, error) {
		capturedQuery = query
		return &workflowservice.ListWorkflowExecutionsResponse{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks?status=Completed,FAILED", nil)
	rec := httptest.NewRecorder()

	err = HandleListMyChecks()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	expected := []string{
		fmt.Sprintf("ExecutionStatus=%d", enums.WORKFLOW_EXECUTION_STATUS_COMPLETED),
		fmt.Sprintf("ExecutionStatus=%d", enums.WORKFLOW_EXECUTION_STATUS_FAILED),
	}
	for _, clause := range expected {
		require.Contains(t, capturedQuery, clause)
	}
	require.Contains(t, capturedQuery, "or")
}

func TestHandleListMyChecksFiltersDynamicPipeline(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := listChecksTemporalClient
	origList := listChecksWorkflows
	t.Cleanup(func() {
		listChecksTemporalClient = origClient
		listChecksWorkflows = origList
	})

	mockClient := &temporalmocks.Client{}
	listChecksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	listChecksWorkflows = func(ctx context.Context, c client.Client, namespace string, query string) (*workflowservice.ListWorkflowExecutionsResponse, error) {
		return &workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				{
					Execution: &common.WorkflowExecution{WorkflowId: "wf-dyn", RunId: "run-dyn"},
					Type:      &common.WorkflowType{Name: "Dynamic Pipeline Workflow"},
					Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					StartTime: timestamppb.New(time.Now().Add(-2 * time.Minute)),
					CloseTime: timestamppb.New(time.Now().Add(-time.Minute)),
				},
				{
					Execution: &common.WorkflowExecution{WorkflowId: "wf-check", RunId: "run-check"},
					Type:      &common.WorkflowType{Name: "CheckWorkflow"},
					Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					StartTime: timestamppb.New(time.Now().Add(-3 * time.Minute)),
					CloseTime: timestamppb.New(time.Now().Add(-2 * time.Minute)),
				},
			},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks?status=completed", nil)
	rec := httptest.NewRecorder()

	err = HandleListMyChecks()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp ListMyChecksResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Executions, 1)
	require.Equal(t, "wf-check", resp.Executions[0].Execution.WorkflowID)
}

func TestHandleListMyChecksListError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := listChecksTemporalClient
	origList := listChecksWorkflows
	t.Cleanup(func() {
		listChecksTemporalClient = origClient
		listChecksWorkflows = origList
	})

	mockClient := &temporalmocks.Client{}
	listChecksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}
	listChecksWorkflows = func(ctx context.Context, c client.Client, namespace string, query string) (*workflowservice.ListWorkflowExecutionsResponse, error) {
		return nil, &serviceerror.Internal{Message: "list failed"}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks", nil)
	rec := httptest.NewRecorder()

	err = HandleListMyChecks()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusInternalServerError, apiErr.Code)
	require.Equal(t, "failed to list workflows", apiErr.Reason)
}
