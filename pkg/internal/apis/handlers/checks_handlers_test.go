// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/dbx"
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

func TestHandleListMyCheckRunsUnauthorized(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/checks-1/runs", nil)
	req.SetPathValue("checkId", "checks-1")
	rec := httptest.NewRecorder()

	err = HandleListMyCheckRuns()(&core.RequestEvent{
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

func TestHandleListMyCheckRunsTemporalClientError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := checksTemporalClient
	t.Cleanup(func() { checksTemporalClient = origClient })

	checksTemporalClient = func(string) (client.Client, error) {
		return nil, errors.New("no client")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/checks-1/runs", nil)
	req.SetPathValue("checkId", "checks-1")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusInternalServerError, apiErr.Code)
	require.Equal(t, "unable to create client", apiErr.Reason)
}

func TestHandleListMyCheckRunsListError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := checksTemporalClient
	t.Cleanup(func() { checksTemporalClient = origClient })

	mockClient := &temporalmocks.Client{}
	checksTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}
	mockClient.On("ListWorkflow", mock.Anything, mock.Anything).
		Return((*workflowservice.ListWorkflowExecutionsResponse)(nil), &serviceerror.Internal{Message: "boom"})

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/checks-1/runs", nil)
	req.SetPathValue("checkId", "checks-1")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusInternalServerError, apiErr.Code)
	require.Equal(t, "failed to list workflow executions", apiErr.Reason)
}

func TestListChecksWorkflowsTemporal(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	expected := &workflowservice.ListWorkflowExecutionsResponse{}

	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return req.GetNamespace() == "tenant-1" && req.GetQuery() == "query"
			}),
		).
		Return(expected, nil)

	resp, err := listChecksWorkflowsTemporal(context.Background(), mockClient, "tenant-1", "query")
	require.NoError(t, err)
	require.Same(t, expected, resp)
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

func TestHandleGetMyCheckRunSuccess(t *testing.T) {
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
	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	mockClient.
		On(
			"DescribeWorkflowExecution",
			mock.Anything,
			"check-1",
			"run-1",
		).
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Execution: &common.WorkflowExecution{WorkflowId: "check-1", RunId: "run-1"},
				Type:      &common.WorkflowType{Name: "CustomCheck"},
			},
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-1/runs/run-1", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
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
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "workflowExecutionInfo")
}

func TestHandleGetMyCheckRunInvalidArgument(t *testing.T) {
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
	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	mockClient.
		On(
			"DescribeWorkflowExecution",
			mock.Anything,
			"check-1",
			"bad-run",
		).
		Return(nil, &serviceerror.InvalidArgument{Message: "invalid"})

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-1/runs/bad-run", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "bad-run")
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
	require.Equal(t, "invalid workflow ID", apiErr.Reason)
}

func TestComputeChildDisplayNameAdditional(t *testing.T) {
	require.Equal(t, "View logs workflow", computeChildDisplayName("OpenIDNetCheckWorkflow-1"))
	require.Equal(t, "View logs workflow", computeChildDisplayName("EWCWorkflow-1"))

	uuid := "123e4567-e89b-12d3-a456-426614174000"
	require.Equal(t, "child", computeChildDisplayName("parent-"+uuid+"-child"))
	require.Equal(t, "plain", computeChildDisplayName("plain"))
}

func TestShouldIncludeQueuedRunsAdditional(t *testing.T) {
	require.True(t, shouldIncludeQueuedRuns(""))
	require.True(t, shouldIncludeQueuedRuns("running"))
	require.True(t, shouldIncludeQueuedRuns("completed, running"))
	require.False(t, shouldIncludeQueuedRuns("completed"))
}

func TestSortExecutionSummariesAdditional(t *testing.T) {
	loc, err := time.LoadLocation("UTC")
	require.NoError(t, err)

	list := []*WorkflowExecutionSummary{
		{StartTime: "2025-01-02T15:04:05Z", EndTime: "2025-01-02T16:04:05Z"},
		{StartTime: "2025-01-01T15:04:05Z", EndTime: "2025-01-01T16:04:05Z"},
	}
	sortExecutionSummaries(list, loc, false)
	require.Equal(t, "02/01/2025, 15:04:05", list[0].StartTime)
	require.Equal(t, "01/01/2025, 15:04:05", list[1].StartTime)
}

func TestBuildExecutionHierarchyWithChildAdditional(t *testing.T) {
	memoValue := base64.StdEncoding.EncodeToString([]byte(`"Parent Name"`))
	executions := []*WorkflowExecution{
		{
			Execution: &WorkflowIdentifier{WorkflowID: "parent", RunID: "run-parent"},
			Type:      WorkflowType{Name: "Other"},
			StartTime: "2025-01-01T10:00:00Z",
			CloseTime: "2025-01-01T10:05:00Z",
			Status:    "WORKFLOW_EXECUTION_STATUS_COMPLETED",
			Memo: &Memo{
				Fields: map[string]*Payload{
					"test": {Data: &memoValue},
				},
			},
		},
		{
			Execution: &WorkflowIdentifier{WorkflowID: "child", RunID: "run-child"},
			Type:      WorkflowType{Name: "Child"},
			StartTime: "2025-01-01T10:01:00Z",
			CloseTime: "2025-01-01T10:02:00Z",
			Status:    "WORKFLOW_EXECUTION_STATUS_COMPLETED",
			ParentExecution: &WorkflowIdentifier{
				WorkflowID: "parent",
				RunID:      "run-parent",
			},
		},
	}

	roots := buildExecutionHierarchy(nil, executions, "owner", "UTC", nil)
	require.Len(t, roots, 1)
	require.Equal(t, "Parent Name", roots[0].DisplayName)
	require.Len(t, roots[0].Children, 1)
}

func TestBuildQueuedWorkflowSummariesAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	canonify.RegisterCanonifyHooks(app)

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	pipelineRecord := createPipelineRecord(t, app, orgID, "My Pipeline")

	path, err := canonify.BuildPath(app, pipelineRecord, canonify.CanonifyPaths["pipelines"], "")
	require.NoError(t, err)

	queued := map[string]QueuedPipelineRunAggregate{
		"ticket-2": {
			TicketID:           "ticket-2",
			PipelineIdentifier: strings.Trim(path, "/"),
			EnqueuedAt:         time.Date(2025, 1, 2, 3, 0, 0, 0, time.UTC),
			Position:           1,
			LineLen:            2,
		},
		"ticket-1": {
			TicketID:           "ticket-1",
			PipelineIdentifier: strings.Trim(path, "/"),
			EnqueuedAt:         time.Date(2025, 1, 2, 4, 0, 0, 0, time.UTC),
			Position:           0,
			LineLen:            2,
		},
	}

	summaries := buildQueuedWorkflowSummaries(app, queued, "UTC")
	require.Len(t, summaries, 2)
	require.Equal(t, string(WorkflowStatusQueued), summaries[0].Status)
	require.Equal(t, "My Pipeline", summaries[0].DisplayName)
}

func TestResolveQueuedPipelineDisplayNameFallbackAdditional(t *testing.T) {
	require.Equal(t, "pipeline-run", resolveQueuedPipelineDisplayName(nil, ""))
	require.Equal(t, "id", resolveQueuedPipelineDisplayName(nil, "id"))
}

func TestHandleGetMyCheckRunHistorySuccess(t *testing.T) {
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
	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	iter := &fakeHistoryIterator{
		events: []*historypb.HistoryEvent{
			{EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED},
		},
	}

	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"check-1",
			"run-1",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(iter)

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-1/runs/run-1/history", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleGetMyCheckRunHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	require.Equal(t, float64(1), payload["count"])
	require.Equal(t, "check-1", payload["checkId"])
	require.Equal(t, "run-1", payload["runId"])
	history, ok := payload["history"].([]any)
	require.True(t, ok)
	require.Len(t, history, 1)
}

func TestHandleListMyCheckRunsSuccess(t *testing.T) {
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
	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return req.GetNamespace() != "" && req.GetQuery() != ""
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				{
					Execution: &common.WorkflowExecution{WorkflowId: "check-1", RunId: "run-1"},
					Type:      &common.WorkflowType{Name: "CustomCheck"},
					Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					StartTime: timestamppb.New(time.Now().Add(-2 * time.Minute)),
					CloseTime: timestamppb.New(time.Now().Add(-1 * time.Minute)),
				},
			},
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-1/runs", nil)
	req.SetPathValue("checkId", "check-1")
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
	require.Equal(t, http.StatusOK, rec.Code)

	var resp ListMyChecksResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Executions, 1)
	require.Equal(t, "check-1", resp.Executions[0].Execution.WorkflowID)
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

func TestHandleTerminateMyCheckRunMissingAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/my/checks/check-1/runs/run-1/terminate", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err := HandleTerminateMyCheckRun()(&core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleTerminateMyCheckRunMissingParams(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks//runs//terminate", nil)
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
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleTerminateMyCheckRunMissingOrganization(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	usersCollection, err := app.FindCollectionByNameOrId("users")
	require.NoError(t, err)
	authRecord := core.NewRecord(usersCollection)
	authRecord.Id = "missing-user"

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks/check-1/runs/run-1/terminate", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleTerminateMyCheckRunClientError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := checksTemporalClient
	t.Cleanup(func() {
		checksTemporalClient = origClient
	})

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return nil, errors.New("client error")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks/check-1/runs/run-1/terminate", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleTerminateMyCheckRunNotFound(t *testing.T) {
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
		On(
			"TerminateWorkflow",
			mock.Anything,
			"check-3",
			"run-3",
			"Terminated by user",
			mock.Anything,
		).
		Return(&serviceerror.NotFound{}).
		Once()

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks/check-3/runs/run-3/terminate", nil)
	req.SetPathValue("checkId", "check-3")
	req.SetPathValue("runId", "run-3")
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
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleTerminateMyCheckRunError(t *testing.T) {
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
		On(
			"TerminateWorkflow",
			mock.Anything,
			"check-4",
			"run-4",
			"Terminated by user",
			mock.Anything,
		).
		Return(errors.New("terminate error")).
		Once()

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks/check-4/runs/run-4/terminate", nil)
	req.SetPathValue("checkId", "check-4")
	req.SetPathValue("runId", "run-4")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)
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
		On(
			"TerminateWorkflow",
			mock.Anything,
			"check-2",
			"run-2",
			"Terminated by user",
			mock.Anything,
		).
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

func TestHandleMyCheckLogsMissingAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-1/runs/run-1/logs", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err := HandleMyCheckLogs()(&core.RequestEvent{
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleMyCheckLogsMissingParams(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks//runs//logs", nil)
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
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleMyCheckLogsMissingOrganization(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	orgAuth, err := app.FindFirstRecordByFilter(
		"orgAuthorizations",
		"user={:user}",
		dbx.Params{"user": authRecord.Id},
	)
	require.NoError(t, err)

	orgRecord, err := app.FindRecordById("organizations", orgAuth.GetString("organization"))
	require.NoError(t, err)

	orgRecord.Set("canonified_name", "")
	require.NoError(t, app.Save(orgRecord))

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-2/runs/run-2/logs", nil)
	req.SetPathValue("checkId", "check-2")
	req.SetPathValue("runId", "run-2")
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
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleMyCheckLogsClientError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := checksTemporalClient
	t.Cleanup(func() {
		checksTemporalClient = origClient
	})

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return nil, errors.New("client error")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-3/runs/run-3/logs", nil)
	req.SetPathValue("checkId", "check-3")
	req.SetPathValue("runId", "run-3")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleMyCheckLogsDescribeNotFound(t *testing.T) {
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
		On("DescribeWorkflowExecution", mock.Anything, "check-5", "run-5").
		Return(nil, &serviceerror.NotFound{}).
		Once()

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-5/runs/run-5/logs", nil)
	req.SetPathValue("checkId", "check-5")
	req.SetPathValue("runId", "run-5")
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
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleMyCheckLogsDescribeError(t *testing.T) {
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
		On("DescribeWorkflowExecution", mock.Anything, "check-6", "run-6").
		Return(nil, errors.New("describe error")).
		Once()

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-6/runs/run-6/logs", nil)
	req.SetPathValue("checkId", "check-6")
	req.SetPathValue("runId", "run-6")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleMyCheckLogsSignalStartError(t *testing.T) {
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
		On("DescribeWorkflowExecution", mock.Anything, "check-7", "run-7").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{}, nil).
		Once()
	mockClient.
		On("SignalWorkflow", mock.Anything, "check-7", "run-7", "start-logs", struct{}{}).
		Return(errors.New("signal error")).
		Once()

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/my/checks/check-7/runs/run-7/logs?action=start",
		nil,
	)
	req.SetPathValue("checkId", "check-7")
	req.SetPathValue("runId", "run-7")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleMyCheckLogsSignalStopError(t *testing.T) {
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
		On("DescribeWorkflowExecution", mock.Anything, "check-8", "run-8").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{}, nil).
		Once()
	mockClient.
		On("SignalWorkflow", mock.Anything, "check-8", "run-8", "stop-logs", struct{}{}).
		Return(errors.New("signal error")).
		Once()

	checksTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/my/checks/check-8/runs/run-8/logs?action=stop",
		nil,
	)
	req.SetPathValue("checkId", "check-8")
	req.SetPathValue("runId", "run-8")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)
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

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/my/checks/check-4/runs/run-4/logs?action=start",
		nil,
	)
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

func TestHandleListMyChecksTemporalClientError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := listChecksTemporalClient
	t.Cleanup(func() {
		listChecksTemporalClient = origClient
	})

	listChecksTemporalClient = func(_ string) (client.Client, error) {
		return nil, errors.New("no client")
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
	require.Equal(t, "unable to create client", apiErr.Reason)
}

func TestHandleListMyChecksQueuedRunsError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := listChecksTemporalClient
	origList := listMobileRunnerSemaphoreWorkflows
	t.Cleanup(func() {
		listChecksTemporalClient = origClient
		listMobileRunnerSemaphoreWorkflows = origList
	})

	listChecksTemporalClient = func(_ string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	listMobileRunnerSemaphoreWorkflows = func(_ context.Context) ([]string, error) {
		return nil, errors.New("list failed")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks?status=queued", nil)
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
	require.Equal(t, "failed to list queued runs", apiErr.Reason)
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
					Execution: &common.WorkflowExecution{
						WorkflowId: "wf-check",
						RunId:      "run-check",
					},
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

func TestBuildExecutionHierarchy(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	encoded := base64.StdEncoding.EncodeToString([]byte("Parent Display"))
	parentMemo := &Memo{Fields: map[string]*Payload{
		"test": {Data: &encoded},
	}}

	parent := &WorkflowExecution{
		Execution: &WorkflowIdentifier{WorkflowID: "parent", RunID: "run-parent"},
		Type:      WorkflowType{Name: "CustomCheck"},
		StartTime: time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339),
		CloseTime: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
		Status:    "COMPLETED",
		Memo:      parentMemo,
	}

	child := &WorkflowExecution{
		Execution:       &WorkflowIdentifier{WorkflowID: "child-123e4567-e89b-12d3-a456-426614174000-suffix", RunID: "run-child"},
		Type:            WorkflowType{Name: "CustomCheck"},
		StartTime:       time.Now().Add(-90 * time.Second).UTC().Format(time.RFC3339),
		CloseTime:       time.Now().Add(-80 * time.Second).UTC().Format(time.RFC3339),
		Status:          "COMPLETED",
		ParentExecution: &WorkflowIdentifier{RunID: "run-parent"},
	}

	mockClient := &temporalmocks.Client{}
	results := buildExecutionHierarchy(app, []*WorkflowExecution{child, parent}, "owner", "UTC", mockClient)
	require.Len(t, results, 1)
	require.Equal(t, "Parent Display", results[0].DisplayName)
	require.Len(t, results[0].Children, 1)
	require.Equal(t, "suffix", results[0].Children[0].DisplayName)
	require.Contains(t, results[0].StartTime, "/")
	require.Contains(t, results[0].EndTime, "/")
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

func TestHandleRerunMyCheckUnauthorizedAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks/check-1/runs/run-1/rerun", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleRerunMyCheck()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleRerunMyCheckMissingParamsAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks//runs//rerun", nil)
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
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleRerunMyCheckDescribeNotFoundAdditional(t *testing.T) {
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
		On("DescribeWorkflowExecution", mock.Anything, "check-1", "run-1").
		Return(nil, &serviceerror.NotFound{})

	checksTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks/check-1/runs/run-1/rerun", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
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
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleRerunMyCheckTemporalClientErrorAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := checksTemporalClient
	t.Cleanup(func() {
		checksTemporalClient = origClient
	})
	checksTemporalClient = func(string) (client.Client, error) {
		return nil, errors.New("no client")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks/check-1/runs/run-1/rerun", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleRerunMyCheckGetInputErrorAdditional(t *testing.T) {
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

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "check-1", "run-1").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Type:      &common.WorkflowType{Name: "wf-type"},
				TaskQueue: "task-queue",
			},
		}, nil)

	checksTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}
	checksGetWorkflowInput = func(string, string, client.Client) (workflowengine.WorkflowInput, error) {
		return workflowengine.WorkflowInput{}, errors.New("input error")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks/check-1/runs/run-1/rerun", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleRerunMyCheckStartErrorAdditional(t *testing.T) {
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
		On("DescribeWorkflowExecution", mock.Anything, "check-1", "run-1").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Type:      &common.WorkflowType{Name: "wf-type"},
				TaskQueue: "task-queue",
			},
		}, nil)

	checksTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}
	checksGetWorkflowInput = func(string, string, client.Client) (workflowengine.WorkflowInput, error) {
		return workflowengine.WorkflowInput{Config: map[string]any{"app_url": "https://app"}}, nil
	}
	checksStartWorkflowWithOptions = func(
		string,
		client.StartWorkflowOptions,
		string,
		workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("start failed")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/my/checks/check-1/runs/run-1/rerun", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleCancelMyCheckRunSuccessAdditional(t *testing.T) {
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
		Return(nil)

	checksTemporalClient = func(string) (client.Client, error) {
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
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "\"status\":\"canceled\"")
}

func TestHandleExportMyCheckRunMissingParamsAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks//runs/", nil)
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
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleExportMyCheckRunGetInputErrorAdditional(t *testing.T) {
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

	checksTemporalClient = func(string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	checksGetWorkflowInput = func(string, string, client.Client) (workflowengine.WorkflowInput, error) {
		return workflowengine.WorkflowInput{}, errors.New("input error")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-1/runs/run-1/export", nil)
	req.SetPathValue("checkId", "check-1")
	req.SetPathValue("runId", "run-1")
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
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetWorkflowInputIteratorErrorAdditional(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	iter := &fakeHistoryIterator{
		nextErr: errors.New("boom"),
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

	_, err := getWorkflowInput("wf-1", "run-1", mockClient)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get workflow history")
}
