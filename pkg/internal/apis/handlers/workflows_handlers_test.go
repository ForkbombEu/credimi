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
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func decodeAPIError(t testing.TB, rec *httptest.ResponseRecorder) apierror.APIError {
	t.Helper()
	body := decodeHandlerErrorResponse(t, rec)
	return apierror.APIError{
		Code:    body.Error.Code,
		Domain:  body.Error.Domain,
		Reason:  body.Error.Reason,
		Message: body.Error.Message,
	}
}

func TestWorkflowListingRoutesRegistered(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	WorkflowListingRoutes.Add(app)
	ConformanceRoutes.Add(app)

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	err = app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/list-workflows", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)

		req = httptest.NewRequest(http.MethodGet, "/api/compliance/checks", nil)
		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNotFound, rec.Code)

		return nil
	})
	require.NoError(t, err)
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
	req.SetPathValue("workflowId", "checks-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleGetMyWorkflowRun()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
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
	req.SetPathValue("workflowId", "checks-1")
	rec := httptest.NewRecorder()

	err = HandleGetMyWorkflowRun()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "runId is required", apiErr.Reason)
}

func TestListChecksWorkflowsTemporalPages(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	first := &workflow.WorkflowExecutionInfo{
		Execution: &common.WorkflowExecution{WorkflowId: "wf-1"},
	}
	second := &workflow.WorkflowExecutionInfo{
		Execution: &common.WorkflowExecution{WorkflowId: "wf-2"},
	}

	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return req.GetNamespace() == "tenant-1" && req.GetQuery() == "query" &&
					req.GetPageSize() == 1000 && len(req.GetNextPageToken()) == 0
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions:    []*workflow.WorkflowExecutionInfo{first},
			NextPageToken: []byte("page-2"),
		}, nil).
		Once()
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return string(req.GetNextPageToken()) == "page-2"
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{second},
		}, nil).
		Once()

	resp, err := listWorkflowsTemporal(context.Background(), mockClient, "tenant-1", "query")
	require.NoError(t, err)
	require.Equal(t, []*workflow.WorkflowExecutionInfo{first, second}, resp.GetExecutions())
	mockClient.AssertExpectations(t)
}

func TestHandleGetMyCheckRunSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := workflowTemporalClient
	t.Cleanup(func() {
		workflowTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	workflowTemporalClient = func(namespace string) (client.Client, error) {
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
	req.SetPathValue("workflowId", "check-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleGetMyWorkflowRun()(&core.RequestEvent{
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

// A run opened from another account's link (e.g. its push notification) is not
// in the caller's namespace: the run page must tell the caller it belongs to
// another organization, not that it is missing.
func TestHandleGetMyWorkflowRunMissingFromNamespace(t *testing.T) {
	cases := []struct {
		name       string
		ownerEmail string
		wantStatus int
		wantReason string
	}{
		{
			name:       "pipeline run of another organization",
			ownerEmail: "userB@example.org",
			wantStatus: http.StatusForbidden,
			wantReason: "workflow belongs to another organization",
		},
		{
			name:       "pipeline run of the caller's organization",
			ownerEmail: "userA@example.org",
			wantStatus: http.StatusNotFound,
			wantReason: "workflow not found",
		},
		{
			name:       "unrecorded run",
			wantStatus: http.StatusNotFound,
			wantReason: "workflow not found",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, err := tests.NewTestApp(testDataDir)
			require.NoError(t, err)
			defer app.Cleanup()

			authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)

			if tc.ownerEmail != "" {
				owner, err := app.FindAuthRecordByEmail("users", tc.ownerEmail)
				require.NoError(t, err)
				ownerOrgID, err := pbutils.GetUserOrganizationID(app, owner.Id)
				require.NoError(t, err)

				pipelineColl, err := app.FindCollectionByNameOrId("pipelines")
				require.NoError(t, err)
				pipelineRecord := core.NewRecord(pipelineColl)
				pipelineRecord.Set("owner", ownerOrgID)
				pipelineRecord.Set("canonified_name", "pipeline123")
				pipelineRecord.Set("name", "pipeline123")
				pipelineRecord.Set("description", "demo pipeline")
				pipelineRecord.Set("yaml", "name: demo")
				require.NoError(t, app.Save(pipelineRecord))

				resultsColl, err := app.FindCollectionByNameOrId("pipeline_results")
				require.NoError(t, err)
				resultRecord := core.NewRecord(resultsColl)
				resultRecord.Set("owner", ownerOrgID)
				resultRecord.Set("pipeline", pipelineRecord.Id)
				resultRecord.Set("workflow_id", "pipeline-1")
				resultRecord.Set("run_id", "run-1")
				require.NoError(t, app.Save(resultRecord))
			}

			origClient := workflowTemporalClient
			t.Cleanup(func() { workflowTemporalClient = origClient })
			mockClient := &temporalmocks.Client{}
			workflowTemporalClient = func(string) (client.Client, error) {
				return mockClient, nil
			}
			mockClient.
				On("DescribeWorkflowExecution", mock.Anything, "pipeline-1", "run-1").
				Return(nil, serviceerror.NewNotFound("workflow not found"))

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/my/workflows/pipeline-1/runs/run-1",
				nil,
			)
			req.SetPathValue("workflowId", "pipeline-1")
			req.SetPathValue("runId", "run-1")
			rec := httptest.NewRecorder()

			err = HandleGetMyWorkflowRun()(&core.RequestEvent{
				App:  app,
				Auth: authRecord,
				Event: router.Event{
					Request:  req,
					Response: rec,
				},
			})
			requireHandlerErrorHandled(t, rec, err)
			require.Equal(t, tc.wantStatus, rec.Code)
			require.Equal(t, tc.wantReason, decodeAPIError(t, rec).Reason)
		})
	}
}

// The run detail page reads its device list (and the live view button) from
// this endpoint, so pipeline runs must carry their devices here.
func TestHandleGetMyWorkflowRunIncludesPipelineDevices(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := workflowTemporalClient
	t.Cleanup(func() { workflowTemporalClient = origClient })
	mockClient := &temporalmocks.Client{}
	workflowTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}

	deviceIDs, err := temporalcrypto.DataConverter().ToPayload([]string{"tenant/runner-1/pixel"})
	require.NoError(t, err)
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "pipeline-1", "run-1").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Execution: &common.WorkflowExecution{WorkflowId: "pipeline-1", RunId: "run-1"},
				Type:      &common.WorkflowType{Name: "Dynamic Pipeline Workflow"},
				Status:    enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
				SearchAttributes: &common.SearchAttributes{
					IndexedFields: map[string]*common.Payload{
						workflowengine.DeviceIdentifiersSearchAttribute: deviceIDs,
					},
				},
			},
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/my/workflows/pipeline-1/runs/run-1", nil)
	req.SetPathValue("workflowId", "pipeline-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleGetMyWorkflowRun()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Devices []PipelineExecutionDevice `json:"devices"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, []PipelineExecutionDevice{
		{DeviceID: "tenant/runner-1/pixel", Name: "pixel", LiveView: false},
	}, body.Devices)
}

func TestHandleGetMyCheckRunIncludesFailureReason(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := workflowTemporalClient
	t.Cleanup(func() {
		workflowTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	workflowTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}
	payload, err := json.Marshal(workflowengine.WorkflowError{
		Code:         "CRE302",
		Summary:      "StepCI checks failed",
		Message:      "One or more StepCI assertions failed.",
		ActivityName: "Run an automation workflow of API calls",
	})
	require.NoError(t, err)

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
				Status:    enums.WORKFLOW_EXECUTION_STATUS_FAILED,
			},
		}, nil)
	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"check-1",
			"run-1",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT,
		).
		Return(&fakeHistoryIterator{
			events: []*historypb.HistoryEvent{
				{
					Attributes: &historypb.HistoryEvent_WorkflowExecutionFailedEventAttributes{
						WorkflowExecutionFailedEventAttributes: &historypb.WorkflowExecutionFailedEventAttributes{
							Failure: &failure.Failure{
								Message: "StepCI checks failed: One or more StepCI assertions failed.",
								FailureInfo: &failure.Failure_ApplicationFailureInfo{
									ApplicationFailureInfo: &failure.ApplicationFailureInfo{
										Type: "CRE302",
										Details: &common.Payloads{
											Payloads: []*common.Payload{{Data: payload}},
										},
									},
								},
							},
						},
					},
				},
			},
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/check-1/runs/run-1", nil)
	req.SetPathValue("workflowId", "check-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleGetMyWorkflowRun()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Equal(
		t,
		"CRE302: [Run an automation workflow of API calls] StepCI checks failed: One or more StepCI assertions failed.",
		response["failure_reason"],
	)
}

func TestHandleGetMyCheckRunInvalidArgument(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := workflowTemporalClient
	t.Cleanup(func() {
		workflowTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	workflowTemporalClient = func(namespace string) (client.Client, error) {
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
	req.SetPathValue("workflowId", "check-1")
	req.SetPathValue("runId", "bad-run")
	rec := httptest.NewRecorder()

	err = HandleGetMyWorkflowRun()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "invalid workflow ID", apiErr.Reason)
}

func TestComputeChildDisplayNameAdditional(t *testing.T) {
	require.Equal(
		t,
		"View logs workflow",
		computeChildDisplayName("OpenID4VPWalletCheckWorkflow-1"),
	)
	require.Equal(t, "View logs workflow", computeChildDisplayName("EWCWorkflow-1"))

	uuid := "123e4567-e89b-12d3-a456-426614174000"
	require.Equal(t, "child", computeChildDisplayName("parent-"+uuid+"-child"))
	require.Equal(t, "plain", computeChildDisplayName("plain"))
}

func TestSortExecutionSummariesAdditional(t *testing.T) {
	list := []*WorkflowExecutionSummary{
		{StartTime: "2025-01-02T15:04:05Z", EndTime: "2025-01-02T16:04:05Z"},
		{StartTime: "2025-01-01T15:04:05Z", EndTime: "2025-01-01T16:04:05Z"},
	}
	sortExecutionSummaries(list, false)
	require.Equal(t, "2025-01-02T15:04:05Z", list[0].StartTime)
	require.Equal(t, "2025-01-01T15:04:05Z", list[1].StartTime)
}

func TestBuildWorkflowExecutionHierarchyWithChildAdditional(t *testing.T) {
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

	roots := buildWorkflowExecutionHierarchy(
		context.Background(),
		nil,
		executions,
		"owner",
		nil,
	)
	require.Len(t, roots, 1)
	require.Equal(t, "Parent Name", roots[0].DisplayName)
	require.Len(t, roots[0].Children, 1)
}

func TestResolveQueuedPipelineDisplayNameFallbackAdditional(t *testing.T) {
	require.Equal(t, "pipeline-run", resolveQueuedPipelineDisplayName(nil, ""))
	require.Equal(t, "id", resolveQueuedPipelineDisplayName(nil, "id"))
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

	origClient := workflowTemporalClient
	t.Cleanup(func() {
		workflowTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("CancelWorkflow", mock.Anything, "check-1", "run-1").
		Return(&serviceerror.NotFound{}).
		Once()

	workflowTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/my/checks/check-1/runs/run-1", nil)
	req.SetPathValue("workflowId", "check-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleCancelMyWorkflowRun()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleListMyWorkflowsStatusFilterQuery(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := listWorkflowsTemporalClient
	origList := listWorkflows
	t.Cleanup(func() {
		listWorkflowsTemporalClient = origClient
		listWorkflows = origList
	})

	mockClient := &temporalmocks.Client{}
	listWorkflowsTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	var capturedQuery string
	listWorkflows = func(ctx context.Context, c client.Client, namespace string, query string) (*workflowservice.ListWorkflowExecutionsResponse, error) {
		capturedQuery = query
		return &workflowservice.ListWorkflowExecutionsResponse{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks?status=Completed,FAILED", nil)
	rec := httptest.NewRecorder()

	err = HandleListMyWorkflows()(&core.RequestEvent{
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

func TestHandleListMyWorkflowsPagination(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := listWorkflowsTemporalClient
	origList := listWorkflows
	t.Cleanup(func() {
		listWorkflowsTemporalClient = origClient
		listWorkflows = origList
	})

	mockClient := &temporalmocks.Client{}
	listWorkflowsTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	now := time.Now()
	listWorkflows = func(ctx context.Context, c client.Client, namespace string, query string) (*workflowservice.ListWorkflowExecutionsResponse, error) {
		return &workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				{
					Execution: &common.WorkflowExecution{WorkflowId: "wf-1", RunId: "run-1"},
					Type:      &common.WorkflowType{Name: "CheckWorkflow"},
					Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					StartTime: timestamppb.New(now.Add(-1 * time.Minute)),
					CloseTime: timestamppb.New(now),
				},
				{
					Execution: &common.WorkflowExecution{WorkflowId: "wf-2", RunId: "run-2"},
					Type:      &common.WorkflowType{Name: "CheckWorkflow"},
					Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					StartTime: timestamppb.New(now.Add(-2 * time.Minute)),
					CloseTime: timestamppb.New(now.Add(-time.Minute)),
				},
				{
					Execution: &common.WorkflowExecution{WorkflowId: "wf-3", RunId: "run-3"},
					Type:      &common.WorkflowType{Name: "CheckWorkflow"},
					Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					StartTime: timestamppb.New(now.Add(-3 * time.Minute)),
					CloseTime: timestamppb.New(now.Add(-2 * time.Minute)),
				},
			},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/list-workflows?limit=1&offset=1", nil)
	rec := httptest.NewRecorder()

	err = HandleListMyWorkflows()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp ListMyWorkflowsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Executions, 1)
	require.Equal(t, "wf-2", resp.Executions[0].Execution.WorkflowID)
}

func TestHandleListMyWorkflowsTemporalClientError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := listWorkflowsTemporalClient
	t.Cleanup(func() {
		listWorkflowsTemporalClient = origClient
	})

	listWorkflowsTemporalClient = func(_ string) (client.Client, error) {
		return nil, errors.New("no client")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks", nil)
	rec := httptest.NewRecorder()

	err = HandleListMyWorkflows()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusServiceUnavailable, apiErr.Code)
	require.Equal(t, "temporal_unavailable", apiErr.Reason)
}

func TestHandleListMyWorkflowsQueuedStatusReturnsEmpty(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := listWorkflowsTemporalClient
	origList := listWorkflows
	t.Cleanup(func() {
		listWorkflowsTemporalClient = origClient
		listWorkflows = origList
	})

	listWorkflowsTemporalClient = func(_ string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	listWorkflows = func(_ context.Context, _ client.Client, _ string, _ string) (*workflowservice.ListWorkflowExecutionsResponse, error) {
		t.Fatalf("listWorkflows should not be called for queued-only filter")
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/list-workflows?status=queued", nil)
	rec := httptest.NewRecorder()

	err = HandleListMyWorkflows()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp ListMyWorkflowsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Empty(t, resp.Executions)
}

func TestHandleListMyWorkflowsFiltersDynamicPipeline(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := listWorkflowsTemporalClient
	origList := listWorkflows
	t.Cleanup(func() {
		listWorkflowsTemporalClient = origClient
		listWorkflows = origList
	})

	mockClient := &temporalmocks.Client{}
	listWorkflowsTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	listWorkflows = func(ctx context.Context, c client.Client, namespace string, query string) (*workflowservice.ListWorkflowExecutionsResponse, error) {
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
						WorkflowId: "Pipeline-Execute-test-pipeline-123",
						RunId:      "run-execute",
					},
					Type:      &common.WorkflowType{Name: "Dynamic Pipeline Workflow"},
					Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					StartTime: timestamppb.New(time.Now().Add(-4 * time.Minute)),
					CloseTime: timestamppb.New(time.Now().Add(-3 * time.Minute)),
				},
				{
					Execution: &common.WorkflowExecution{
						WorkflowId: "wf-execute-child",
						RunId:      "run-execute-child",
					},
					Type: &common.WorkflowType{Name: "CheckWorkflow"},
					ParentExecution: &common.WorkflowExecution{
						WorkflowId: "Pipeline-Execute-test-pipeline-123",
						RunId:      "run-execute",
					},
					Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					StartTime: timestamppb.New(time.Now().Add(-230 * time.Second)),
					CloseTime: timestamppb.New(time.Now().Add(-190 * time.Second)),
				},
				{
					Execution: &common.WorkflowExecution{
						WorkflowId: "wf-dyn-child",
						RunId:      "run-dyn-child",
					},
					Type: &common.WorkflowType{Name: "CheckWorkflow"},
					ParentExecution: &common.WorkflowExecution{
						WorkflowId: "wf-dyn",
						RunId:      "run-dyn",
					},
					Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					StartTime: timestamppb.New(time.Now().Add(-90 * time.Second)),
					CloseTime: timestamppb.New(time.Now().Add(-30 * time.Second)),
				},
				{
					Execution: &common.WorkflowExecution{
						WorkflowId: "wf-dyn-missing-child",
						RunId:      "run-dyn-missing-child",
					},
					Type: &common.WorkflowType{Name: "CheckWorkflow"},
					ParentExecution: &common.WorkflowExecution{
						WorkflowId: "wf-dyn-missing",
						RunId:      "run-dyn-missing",
					},
					Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					StartTime: timestamppb.New(time.Now().Add(-85 * time.Second)),
					CloseTime: timestamppb.New(time.Now().Add(-25 * time.Second)),
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
				{
					Execution: &common.WorkflowExecution{
						WorkflowId: "wf-check-child",
						RunId:      "run-check-child",
					},
					Type: &common.WorkflowType{Name: "CheckWorkflow"},
					ParentExecution: &common.WorkflowExecution{
						WorkflowId: "wf-check",
						RunId:      "run-check",
					},
					Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					StartTime: timestamppb.New(time.Now().Add(-160 * time.Second)),
					CloseTime: timestamppb.New(time.Now().Add(-150 * time.Second)),
				},
			},
		}, nil
	}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-dyn-missing", "run-dyn-missing").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Type: &common.WorkflowType{Name: "Dynamic Pipeline Workflow"},
			},
		}, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks?status=completed", nil)
	rec := httptest.NewRecorder()

	err = HandleListMyWorkflows()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp ListMyWorkflowsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Executions, 2)
	require.Equal(t, "wf-check", resp.Executions[0].Execution.WorkflowID)
	require.Len(t, resp.Executions[0].Children, 1)
	require.Equal(t, "wf-check-child", resp.Executions[0].Children[0].Execution.WorkflowID)
	require.Equal(t, "Pipeline-Execute-test-pipeline-123", resp.Executions[1].Execution.WorkflowID)
	require.Len(t, resp.Executions[1].Children, 1)
	require.Equal(t, "wf-execute-child", resp.Executions[1].Children[0].Execution.WorkflowID)
}

func TestHandleListMyWorkflowsHidesActionAuthorSessions(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := listWorkflowsTemporalClient
	origList := listWorkflows
	t.Cleanup(func() {
		listWorkflowsTemporalClient = origClient
		listWorkflows = origList
	})
	listWorkflowsTemporalClient = func(string) (client.Client, error) {
		return &temporalmocks.Client{}, nil
	}
	session := "ActionAuthorSession-0f8fad5b-d9cb-469f-a165-70867728950e"
	execution := func(id, runID, workflowType string, parent *common.WorkflowExecution) *workflow.WorkflowExecutionInfo {
		return &workflow.WorkflowExecutionInfo{
			Execution:       &common.WorkflowExecution{WorkflowId: id, RunId: runID},
			Type:            &common.WorkflowType{Name: workflowType},
			ParentExecution: parent,
			Status:          enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
			StartTime:       timestamppb.New(time.Now().Add(-time.Minute)),
		}
	}
	listWorkflows = func(context.Context, client.Client, string, string) (*workflowservice.ListWorkflowExecutionsResponse, error) {
		return &workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				execution(session, "run-session", "Action author session", nil),
				execution(
					session+"-author",
					"run-chat",
					"ActionAuthor",
					&common.WorkflowExecution{WorkflowId: session, RunId: "run-session"},
				),
				execution("wf-check", "run-check", "CheckWorkflow", nil),
			},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/list-workflows", nil)
	rec := httptest.NewRecorder()
	err = HandleListMyWorkflows()(&core.RequestEvent{
		App:   app,
		Auth:  authRecord,
		Event: router.Event{Request: req, Response: rec},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp ListMyWorkflowsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Executions, 1)
	require.Equal(t, "wf-check", resp.Executions[0].Execution.WorkflowID)
	require.Empty(t, resp.Executions[0].Children)
}

func TestWorkflowRunHandlersHideActionAuthorSessions(t *testing.T) {
	for name, handler := range map[string]func() func(*core.RequestEvent) error{
		"get":    HandleGetMyWorkflowRun,
		"cancel": HandleCancelMyWorkflowRun,
	} {
		for _, workflowID := range []string{
			"ActionAuthorSession-x",
			"ActionAuthorSession-x-author",
		} {
			t.Run(name+" "+workflowID, func(t *testing.T) {
				app, err := tests.NewTestApp(testDataDir)
				require.NoError(t, err)
				defer app.Cleanup()
				authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
				require.NoError(t, err)

				origClient := workflowTemporalClient
				t.Cleanup(func() { workflowTemporalClient = origClient })
				workflowTemporalClient = func(string) (client.Client, error) {
					t.Fatal("temporal client must not be created")
					return nil, nil
				}

				req := httptest.NewRequest(http.MethodGet, "/api/my/workflows/x/runs/run-1", nil)
				req.SetPathValue("workflowId", workflowID)
				req.SetPathValue("runId", "run-1")
				rec := httptest.NewRecorder()
				err = handler()(&core.RequestEvent{
					App:   app,
					Auth:  authRecord,
					Event: router.Event{Request: req, Response: rec},
				})
				requireHandlerErrorHandled(t, rec, err)
				require.Equal(t, http.StatusNotFound, rec.Code)
			})
		}
	}
}

func TestBuildWorkflowExecutionHierarchy(t *testing.T) {
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
		Execution: &WorkflowIdentifier{
			WorkflowID: "child-123e4567-e89b-12d3-a456-426614174000-suffix",
			RunID:      "run-child",
		},
		Type:            WorkflowType{Name: "CustomCheck"},
		StartTime:       time.Now().Add(-90 * time.Second).UTC().Format(time.RFC3339),
		CloseTime:       time.Now().Add(-80 * time.Second).UTC().Format(time.RFC3339),
		Status:          "COMPLETED",
		ParentExecution: &WorkflowIdentifier{RunID: "run-parent"},
	}

	mockClient := &temporalmocks.Client{}
	results := buildWorkflowExecutionHierarchy(
		context.Background(),
		app,
		[]*WorkflowExecution{child, parent},
		"owner",
		mockClient,
	)
	require.Len(t, results, 1)
	require.Equal(t, "Parent Display", results[0].DisplayName)
	require.Len(t, results[0].Children, 1)
	require.Equal(t, "suffix", results[0].Children[0].DisplayName)
	require.Equal(t, parent.StartTime, results[0].StartTime)
	require.Equal(t, parent.CloseTime, results[0].EndTime)
}

func TestHandleListMyWorkflowsListError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := listWorkflowsTemporalClient
	origList := listWorkflows
	t.Cleanup(func() {
		listWorkflowsTemporalClient = origClient
		listWorkflows = origList
	})

	mockClient := &temporalmocks.Client{}
	listWorkflowsTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}
	listWorkflows = func(ctx context.Context, c client.Client, namespace string, query string) (*workflowservice.ListWorkflowExecutionsResponse, error) {
		return nil, &serviceerror.Internal{Message: "list failed"}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks", nil)
	rec := httptest.NewRecorder()

	err = HandleListMyWorkflows()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusInternalServerError, apiErr.Code)
	require.Equal(t, "failed to list workflows", apiErr.Reason)
}

func TestHandleCancelMyCheckRunSuccessAdditional(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origClient := workflowTemporalClient
	t.Cleanup(func() {
		workflowTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("CancelWorkflow", mock.Anything, "check-1", "run-1").
		Return(nil)

	workflowTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/my/checks/check-1/runs/run-1", nil)
	req.SetPathValue("workflowId", "check-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleCancelMyWorkflowRun()(&core.RequestEvent{
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
