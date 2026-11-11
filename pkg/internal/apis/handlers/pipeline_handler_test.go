// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func setupPipelineApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	ensureScoreboardExpandedDataField(t, app)
	ensureScoreboardMinRunningTimeSecondsField(t, app)
	canonify.RegisterCanonifyHooks(app)
	PipelineInternalRoutes.Add(app)
	seedInternalAdminKey(t, app)

	return app
}

// The test fixture predates the expanded_data migration. Keep its schema aligned
// with the production migration while tests run against the fixture database.
func ensureScoreboardExpandedDataField(t testing.TB, app *tests.TestApp) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
	require.NoError(t, err)
	if collection.Fields.GetByName("expanded_data") != nil {
		return
	}
	collection.Fields.Add(&core.JSONField{Name: "expanded_data"})
	require.NoError(t, app.Save(collection))
}

func ensureScoreboardMinRunningTimeSecondsField(t testing.TB, app *tests.TestApp) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
	require.NoError(t, err)
	if collection.Fields.GetByName("minimum_running_time_seconds") != nil {
		return
	}
	collection.Fields.Add(&core.NumberField{Name: "minimum_running_time_seconds", OnlyInt: true})
	require.NoError(t, app.Save(collection))
}

// setupPipelineStartApp builds a test app with pipeline start routes.
func setupPipelineStartApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	PipelineRoutes.Add(app)

	return app
}

func TestHandleListPipelineExecutionOverviewReturnsResults(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineStartApp(t)
	defer app.Cleanup()
	ensurePipelineRetentionEvidenceFields(t, app)

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
	setPipelineResultFiles(
		t,
		app,
		resultRecord.Id,
		[]string{"abc_result_video_main.mp4"},
		[]string{"abc_screenshot_main.png"},
		[]string{"abc_logfile_main.zip"},
		nil,
	)
	setPipelineResultReport(t, app, resultRecord.Id, "run_report.md")
	_, err = app.DB().NewQuery(
		`UPDATE pipeline_results SET canonified_identifier = '' WHERE id = {:id}`,
	).Bind(dbx.Params{"id": resultRecord.Id}).Execute()
	require.NoError(t, err)

	pipelinePath, err := canonify.BuildPath(
		app,
		pipelineRecord,
		canonify.CanonifyPaths["pipelines"],
		"",
	)
	require.NoError(t, err)
	pipelineIdentifier := strings.Trim(pipelinePath, "/")

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
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return strings.Contains(req.GetQuery(), "WorkflowType")
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				buildPipelineExecutionInfo(
					"wf-1",
					"run-1",
					pipelineIdentifier,
				),
			},
		}, nil).
		Once()
	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-1",
			"run-1",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(&fakeHistoryIterator{events: []*historypb.HistoryEvent{}}, nil).
		Maybe()

	pipelineTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/list-executions", nil)
	rec := httptest.NewRecorder()

	err = HandleListPipelineExecutionOverview()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var response map[string][]pipelineWorkflowSummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response[pipelineRecord.Id], 1)
	require.Equal(t, pipelineIdentifier, response[pipelineRecord.Id][0].PipelineIdentifier)

	summary := response[pipelineRecord.Id][0]
	require.Len(t, summary.Results, 1)
	require.NotEmpty(t, summary.Results[0].Video)
	require.NotEmpty(t, summary.Results[0].Log)
	require.Contains(t, summary.Report, "run_report.md")
	require.Empty(t, summary.Children)
}

func TestHandleListPipelineExecutionOverviewListError(t *testing.T) {
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
		On(
			"ListWorkflow",
			mock.Anything,
			mock.AnythingOfType("*workflowservice.ListWorkflowExecutionsRequest"),
		).
		Return((*workflowservice.ListWorkflowExecutionsResponse)(nil), errors.New("boom")).
		Once()

	pipelineTemporalClient = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/list-executions", nil)
	rec := httptest.NewRecorder()

	err = HandleListPipelineExecutionOverview()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleListPipelineExecutionOverviewTemporalClientError(t *testing.T) {
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

	pipelineTemporalClient = func(namespace string) (client.Client, error) {
		return nil, errors.New("no temporal")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/list-executions", nil)
	rec := httptest.NewRecorder()

	err = HandleListPipelineExecutionOverview()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestHandleListPipelineExecutionOverviewQueuedRunsError(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/list-executions", nil)
	rec := httptest.NewRecorder()

	err = HandleListPipelineExecutionOverview()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleListPipelineExecutionHistoryFiltersAndPaginates(t *testing.T) {
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

	originalTemporalClient := pipelineTemporalClient
	originalListQueued := pipelineListQueuedRuns
	t.Cleanup(func() {
		pipelineTemporalClient = originalTemporalClient
		pipelineListQueuedRuns = originalListQueued
	})

	pipelineListQueuedRuns = func(context.Context, string) (map[string]QueuedPipelineRunAggregate, error) {
		return map[string]QueuedPipelineRunAggregate{}, nil
	}

	resultsColl, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	resultRecord := core.NewRecord(resultsColl)
	resultRecord.Set("owner", orgID)
	resultRecord.Set("pipeline", pipelineRecord.Id)
	resultRecord.Set("workflow_id", "wf-2")
	resultRecord.Set("run_id", "run-2")
	require.NoError(t, app.Save(resultRecord))

	pipelinePath, err := canonify.BuildPath(
		app,
		pipelineRecord,
		canonify.CanonifyPaths["pipelines"],
		"",
	)
	require.NoError(t, err)
	pipelineIdentifier := strings.Trim(pipelinePath, "/")

	mockClient := &temporalmocks.Client{}
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return req.GetNamespace() == "usera-s-organization" &&
					strings.Contains(
						req.GetQuery(),
						fmt.Sprintf(
							"ExecutionStatus=%d",
							enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
						),
					) &&
					strings.Contains(
						req.GetQuery(),
						fmt.Sprintf(`PipelineIdentifier="%s"`, pipelineIdentifier),
					) &&
					req.GetPageSize() == int32(1) &&
					len(req.GetNextPageToken()) == 0
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				buildPipelineExecutionInfo("wf-1", "run-1", pipelineIdentifier),
			},
			NextPageToken: []byte("next"),
		}, nil).
		Once()
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return req.GetNamespace() == "usera-s-organization" &&
					strings.Contains(
						req.GetQuery(),
						fmt.Sprintf(
							"ExecutionStatus=%d",
							enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
						),
					) &&
					strings.Contains(
						req.GetQuery(),
						fmt.Sprintf(`PipelineIdentifier="%s"`, pipelineIdentifier),
					) &&
					req.GetPageSize() == int32(1) &&
					string(req.GetNextPageToken()) == "next"
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				buildPipelineExecutionInfo("wf-2", "run-2", pipelineIdentifier),
			},
		}, nil).
		Once()
	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-2",
			"run-2",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(&fakeHistoryIterator{events: []*historypb.HistoryEvent{}}, nil).
		Maybe()
	pipelineTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/pipeline/list-executions/"+pipelineRecord.Id+"?status=completed&limit=1&page=1",
		nil,
	)
	req.SetPathValue("id", pipelineRecord.Id)
	rec := httptest.NewRecorder()

	err = HandleListPipelineExecutionHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status=%d body=%s", rec.Code, rec.Body.String())
	}

	var response []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	execution, ok := response[0]["execution"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "wf-2", execution["workflowId"])
	require.Equal(t, pipelineIdentifier, response[0]["pipeline_identifier"])
	require.Empty(t, response[0]["children"])
}

func TestHandleListPipelineExecutionHistoryQueuedOnly(t *testing.T) {
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
	originalTemporalClient := pipelineTemporalClient
	t.Cleanup(func() {
		pipelineListQueuedRuns = originalListQueued
		pipelineTemporalClient = originalTemporalClient
	})

	pipelineListQueuedRuns = func(ctx context.Context, namespace string) (map[string]QueuedPipelineRunAggregate, error) {
		return map[string]QueuedPipelineRunAggregate{
			"ticket-1": {
				TicketID:           "ticket-1",
				PipelineIdentifier: "usera-s-organization/pipeline123",
				EnqueuedAt:         time.Now().Add(-1 * time.Minute),
				LeaderDeviceID:     "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				DeviceIDs:          []string{"runner-1"},
			},
		}, nil
	}
	pipelineTemporalClient = func(string) (client.Client, error) {
		t.Fatalf("temporal client should not be used for queued-only status")
		return nil, nil
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/pipeline/list-executions/"+pipelineRecord.Id+"?status=queued",
		nil,
	)
	req.SetPathValue("id", pipelineRecord.Id)
	rec := httptest.NewRecorder()

	err = HandleListPipelineExecutionHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status=%d body=%s", rec.Code, rec.Body.String())
	}

	var response []pipelineWorkflowSummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.NotNil(t, response[0].Queue)
	require.Equal(t, "ticket-1", response[0].Queue.TicketID)
	require.Equal(t, string(WorkflowStatusQueued), response[0].Status)
	require.Equal(t, "usera-s-organization/pipeline123", response[0].PipelineIdentifier)
}

func TestHandleListPipelineExecutionHistoryIncludesQueuedInPagination(t *testing.T) {
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
		return map[string]QueuedPipelineRunAggregate{
			"ticket-1": {
				TicketID:           "ticket-1",
				PipelineIdentifier: "usera-s-organization/pipeline123",
				EnqueuedAt:         time.Now().Add(-1 * time.Minute),
				LeaderDeviceID:     "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				DeviceIDs:          []string{"runner-1"},
			},
		}, nil
	}

	pipelinePath, err := canonify.BuildPath(
		app,
		pipelineRecord,
		canonify.CanonifyPaths["pipelines"],
		"",
	)
	require.NoError(t, err)
	pipelineIdentifier := strings.Trim(pipelinePath, "/")

	mockClient := &temporalmocks.Client{}
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return req.GetNamespace() == "usera-s-organization" &&
					strings.Contains(
						req.GetQuery(),
						fmt.Sprintf(`PipelineIdentifier="%s"`, pipelineIdentifier),
					) &&
					req.GetPageSize() == int32(1) &&
					len(req.GetNextPageToken()) == 0
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				buildPipelineExecutionInfo("wf-1", "run-1", pipelineIdentifier),
			},
		}, nil).
		Once()
	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-1",
			"run-1",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(&fakeHistoryIterator{events: []*historypb.HistoryEvent{}}, nil).
		Maybe()
	pipelineTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/pipeline/list-executions/"+pipelineRecord.Id+"?limit=1&offset=1",
		nil,
	)
	req.SetPathValue("id", pipelineRecord.Id)
	rec := httptest.NewRecorder()

	err = HandleListPipelineExecutionHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status=%d body=%s", rec.Code, rec.Body.String())
	}

	var response []pipelineWorkflowSummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, "wf-1", response[0].Execution.WorkflowID)
	require.Equal(t, pipelineIdentifier, response[0].PipelineIdentifier)
}

func TestHandleListPipelineExecutionHistoryMissingAuth(t *testing.T) {
	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/list-executions/any", nil)
	req.SetPathValue("id", "any")
	rec := httptest.NewRecorder()

	err := HandleListPipelineExecutionHistory()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleListPipelineExecutionHistoryMissingID(t *testing.T) {
	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/list-executions", nil)
	rec := httptest.NewRecorder()

	err = HandleListPipelineExecutionHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleListPipelineExecutionHistoryNoPipelines(t *testing.T) {
	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/list-executions/missing", nil)
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()

	err = HandleListPipelineExecutionHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status=%d body=%s", rec.Code, rec.Body.String())
	}

	var response []pipelineWorkflowSummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Empty(t, response)
}

func TestHandleListPipelineExecutionHistoryPublishedPipelineShowsOnlyMyRuns(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	orgColl, err := app.FindCollectionByNameOrId("organizations")
	require.NoError(t, err)
	otherOrg := core.NewRecord(orgColl)
	otherOrg.Set("name", "Other Org")
	otherOrg.Set("canonified_name", "other-org")
	require.NoError(t, app.Save(otherOrg))

	pipelineColl, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelineColl)
	pipelineRecord.Set("owner", otherOrg.Id)
	pipelineRecord.Set("name", "pipeline123")
	pipelineRecord.Set("canonified_name", "pipeline123")
	pipelineRecord.Set("description", "published demo pipeline")
	pipelineRecord.Set("yaml", "name: demo")
	pipelineRecord.Set("published", true)
	require.NoError(t, app.Save(pipelineRecord))

	resultsColl, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	resultRecord := core.NewRecord(resultsColl)
	resultRecord.Set("owner", orgID)
	resultRecord.Set("pipeline", pipelineRecord.Id)
	resultRecord.Set("workflow_id", "wf-foreign")
	resultRecord.Set("run_id", "run-foreign")
	require.NoError(t, app.Save(resultRecord))

	pipelinePath, err := canonify.BuildPath(
		app,
		pipelineRecord,
		canonify.CanonifyPaths["pipelines"],
		"",
	)
	require.NoError(t, err)
	pipelineIdentifier := strings.Trim(pipelinePath, "/")

	originalTemporalClient := pipelineTemporalClient
	originalListQueued := pipelineListQueuedRuns
	t.Cleanup(func() {
		pipelineTemporalClient = originalTemporalClient
		pipelineListQueuedRuns = originalListQueued
	})

	pipelineListQueuedRuns = func(context.Context, string) (map[string]QueuedPipelineRunAggregate, error) {
		return map[string]QueuedPipelineRunAggregate{}, nil
	}

	mockClient := &temporalmocks.Client{}
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return req.GetNamespace() == "usera-s-organization" &&
					strings.Contains(
						req.GetQuery(),
						fmt.Sprintf(`PipelineIdentifier="%s"`, pipelineIdentifier),
					)
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				buildPipelineExecutionInfo("wf-foreign", "run-foreign", pipelineIdentifier),
			},
		}, nil).
		Once()
	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-foreign",
			"run-foreign",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(&fakeHistoryIterator{events: []*historypb.HistoryEvent{}}, nil).
		Maybe()
	pipelineTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/pipeline/list-executions/"+pipelineRecord.Id,
		nil,
	)
	req.SetPathValue("id", pipelineRecord.Id)
	rec := httptest.NewRecorder()

	err = HandleListPipelineExecutionHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response []pipelineWorkflowSummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response, 1)
	require.Equal(t, "wf-foreign", response[0].Execution.WorkflowID)
	require.Equal(t, pipelineIdentifier, response[0].PipelineIdentifier)
}

func TestHandleListPipelineExecutionHistoryListError(t *testing.T) {
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

	originalTemporalClient := pipelineTemporalClient
	t.Cleanup(func() {
		pipelineTemporalClient = originalTemporalClient
	})

	mockClient := &temporalmocks.Client{}
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.AnythingOfType("*workflowservice.ListWorkflowExecutionsRequest"),
		).
		Return((*workflowservice.ListWorkflowExecutionsResponse)(nil), errors.New("boom")).
		Once()
	pipelineTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/pipeline/list-executions/"+pipelineRecord.Id,
		nil,
	)
	req.SetPathValue("id", pipelineRecord.Id)
	rec := httptest.NewRecorder()

	err = HandleListPipelineExecutionHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleGetPipelineExecutionReturnsOneRun(t *testing.T) {
	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	organization, err := pbutils.GetUserOrganization(app, authRecord.Id)
	require.NoError(t, err)
	pipelineRecord := createPipelineExecutionTestPipeline(t, app, organization.Id)
	pipelineIdentifier := pipelineIdentifierForTest(t, app, pipelineRecord)

	root := buildPipelineExecutionInfo("wf-1", "run-1", pipelineIdentifier)

	// The single-run summary counts no steps, so it never reads the run's history.
	mockClient := &temporalmocks.Client{}
	mockClient.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: root,
		}, nil).
		Once()
	originalTemporalClient := pipelineTemporalClient
	t.Cleanup(func() { pipelineTemporalClient = originalTemporalClient })
	pipelineTemporalClient = func(string) (client.Client, error) { return mockClient, nil }

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("id", pipelineRecord.Id)
	req.SetPathValue("workflow_id", "wf-1")
	req.SetPathValue("run_id", "run-1")
	rec := httptest.NewRecorder()
	err = HandleGetPipelineExecution()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response pipelineWorkflowSummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, "wf-1", response.Execution.WorkflowID)
	require.Equal(t, pipelineIdentifier, response.PipelineIdentifier)
	require.Empty(t, response.Children)
	require.Nil(t, response.StepCount)
	mockClient.AssertExpectations(t)
}

func TestCountStartedStepsCachesClosedRunsOnly(t *testing.T) {
	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	history := func() *fakeHistoryIterator {
		return &fakeHistoryIterator{events: []*historypb.HistoryEvent{
			stepsStartedEvent(t, pipelineinternal.StepDefinition{
				StepSpec: pipelineinternal.StepSpec{ID: "fetch", Use: "http-request"},
			}),
			{
				EventId:   5,
				EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
				Attributes: &historypb.HistoryEvent_ActivityTaskScheduledEventAttributes{
					ActivityTaskScheduledEventAttributes: &historypb.ActivityTaskScheduledEventAttributes{
						Input: stepsPayloads(t, map[string]any{
							"config": map[string]any{workflowengine.StepIDConfigKey: "fetch"},
						}),
					},
				},
			},
		}}
	}
	mockClient := &temporalmocks.Client{}
	// The closed run is read once; the running run on every call.
	for _, tc := range []struct {
		runID string
		reads int
	}{{"closed", 1}, {"running", 2}} {
		mockClient.On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-1",
			tc.runID,
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).Return(func(context.Context, string, string, bool, enums.HistoryEventFilterType) client.HistoryEventIterator {
			return history()
		}).Times(tc.reads)
	}
	originalStepCounts := pipelineStepCounts
	t.Cleanup(func() { pipelineStepCounts = originalStepCounts })
	pipelineStepCounts = &stepCountCache{counts: map[string]int{}}

	for range 2 {
		for _, tc := range []struct {
			runID  string
			closed bool
		}{{"closed", true}, {"running", false}} {
			count := countStartedSteps(
				context.Background(),
				app,
				mockClient,
				"org",
				workflowExecutionRef{WorkflowID: "wf-1", RunID: tc.runID},
				tc.closed,
			)
			require.NotNil(t, count)
			require.Equal(t, 1, *count)
		}
	}
	mockClient.AssertExpectations(t)
}

func createPipelineExecutionTestPipeline(
	t testing.TB,
	app core.App,
	ownerID string,
) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("owner", ownerID)
	record.Set("name", "Pipeline execution test")
	record.Set("canonified_name", "pipeline-execution-test")
	record.Set("description", "Pipeline execution handler test")
	record.Set("yaml", "name: Pipeline execution test")
	require.NoError(t, app.Save(record))
	return record
}

func pipelineIdentifierForTest(
	t testing.TB,
	app core.App,
	pipelineRecord *core.Record,
) string {
	t.Helper()
	path, err := canonify.BuildPath(
		app,
		pipelineRecord,
		canonify.CanonifyPaths["pipelines"],
		"",
	)
	require.NoError(t, err)
	return strings.Trim(path, "/")
}

func TestSelectPipelineExecutionOverview(t *testing.T) {
	executions := map[string][]*WorkflowExecution{
		"pipeline-1": {
			{
				Status:    "WORKFLOW_EXECUTION_STATUS_COMPLETED",
				StartTime: time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			},
			{
				Status:    "WORKFLOW_EXECUTION_STATUS_RUNNING",
				StartTime: time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
			},
			{
				Status:    "WORKFLOW_EXECUTION_STATUS_COMPLETED",
				StartTime: time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
			},
		},
	}

	selected := selectPipelineExecutionOverview(executions, 2)
	require.Len(t, selected["pipeline-1"], 2)
	require.ElementsMatch(
		t,
		[]string{"WORKFLOW_EXECUTION_STATUS_RUNNING", "WORKFLOW_EXECUTION_STATUS_COMPLETED"},
		[]string{selected["pipeline-1"][0].Status, selected["pipeline-1"][1].Status},
	)
}

func TestSelectPipelineExecutionOverviewOrdersMixedTimestampPrecision(t *testing.T) {
	executions := map[string][]*WorkflowExecution{
		"pipeline-1": {
			{
				Status:    "WORKFLOW_EXECUTION_STATUS_COMPLETED",
				StartTime: "2026-04-21T10:00:00Z",
			},
			{
				Status:    "WORKFLOW_EXECUTION_STATUS_COMPLETED",
				StartTime: "2026-04-21T10:00:00.1Z",
			},
		},
	}

	selected := selectPipelineExecutionOverview(executions, 1)

	require.Len(t, selected["pipeline-1"], 1)
	require.Equal(t, "2026-04-21T10:00:00.1Z", selected["pipeline-1"][0].StartTime)
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

func ensurePipelineRetentionEvidenceFields(t testing.TB, app *tests.TestApp) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	if collection.Fields.GetByName("credential_well_knowns") == nil {
		collection.Fields.Add(&core.JSONField{Name: "credential_well_knowns"})
	}
	if collection.Fields.GetByName("presentation_results") == nil {
		collection.Fields.Add(&core.JSONField{Name: "presentation_results"})
	}
	if collection.Fields.GetByName("report") == nil {
		collection.Fields.Add(&core.FileField{Name: "report", MaxSelect: 1})
	}
	if collection.Fields.GetByName("fcaf_report") == nil {
		collection.Fields.Add(&core.FileField{Name: "fcaf_report", MaxSelect: 1})
	}
	if collection.Fields.GetByName("fcaf_report_pdf") == nil {
		collection.Fields.Add(&core.FileField{Name: "fcaf_report_pdf", MaxSelect: 1})
	}
	require.NoError(t, app.Save(collection))
}

func setPipelineResultFiles(
	t testing.TB,
	app *tests.TestApp,
	recordID string,
	videoResults []string,
	screenshots []string,
	logcats []string,
	iosLogstreams []string,
) {
	t.Helper()

	_, err := app.DB().NewQuery(
		`UPDATE pipeline_results
		SET video_results = {:video_results},
		    screenshots = {:screenshots},
		    logcats = {:logcats},
		    ios_logstreams = {:ios_logstreams}
		WHERE id = {:id}`,
	).Bind(dbx.Params{
		"video_results":  mustMarshalJSONStringArray(t, videoResults),
		"screenshots":    mustMarshalJSONStringArray(t, screenshots),
		"logcats":        mustMarshalJSONStringArray(t, logcats),
		"ios_logstreams": mustMarshalJSONStringArray(t, iosLogstreams),
		"id":             recordID,
	}).Execute()
	require.NoError(t, err)
}

func setPipelineResultReport(t testing.TB, app *tests.TestApp, recordID string, report string) {
	t.Helper()

	_, err := app.DB().NewQuery(
		`UPDATE pipeline_results
		SET report = {:report}
		WHERE id = {:id}`,
	).Bind(dbx.Params{
		"report": mustMarshalJSONStringArray(t, []string{report}),
		"id":     recordID,
	}).Execute()
	require.NoError(t, err)
}

func mustMarshalJSONStringArray(t testing.TB, values []string) string {
	t.Helper()

	if values == nil {
		values = []string{}
	}

	data, err := json.Marshal(values)
	require.NoError(t, err)

	return string(data)
}

type pipelineStepsFixture struct {
	app          *tests.TestApp
	authRecord   *core.Record
	pipelineID   string
	identifier   string
	mockClient   *temporalmocks.Client
	historyStart time.Time
}

func newPipelineStepsFixture(t *testing.T) *pipelineStepsFixture {
	t.Helper()
	app := setupPipelineStartApp(t)
	t.Cleanup(app.Cleanup)
	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	organization, err := pbutils.GetUserOrganization(app, authRecord.Id)
	require.NoError(t, err)
	pipelineRecord := createPipelineExecutionTestPipeline(t, app, organization.Id)

	mockClient := temporalmocks.NewClient(t)
	originalTemporalClient := pipelineTemporalClient
	t.Cleanup(func() { pipelineTemporalClient = originalTemporalClient })
	pipelineTemporalClient = func(string) (client.Client, error) { return mockClient, nil }

	return &pipelineStepsFixture{
		app:          app,
		authRecord:   authRecord,
		pipelineID:   pipelineRecord.Id,
		identifier:   pipelineIdentifierForTest(t, app, pipelineRecord),
		mockClient:   mockClient,
		historyStart: time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC),
	}
}

func (f *pipelineStepsFixture) describePipelineRun(identifier string) {
	f.mockClient.On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: buildPipelineExecutionInfo("wf-1", "run-1", identifier),
		}, nil).
		Once()
}

func (f *pipelineStepsFixture) history(
	workflowID, runID string,
	events ...*historypb.HistoryEvent,
) {
	for i, event := range events {
		event.EventTime = timestamppb.New(f.historyStart.Add(time.Duration(i) * time.Second))
	}
	f.mockClient.On(
		"GetWorkflowHistory",
		mock.Anything,
		workflowID,
		runID,
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(&fakeHistoryIterator{events: events}).Once()
}

func (f *pipelineStepsFixture) children(
	parentWorkflowID string,
	executions ...*workflow.WorkflowExecutionInfo,
) {
	f.mockClient.On(
		"ListWorkflow",
		mock.Anything,
		mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
			return strings.Contains(req.GetQuery(), `ParentWorkflowId="`+parentWorkflowID+`"`)
		}),
	).Return(&workflowservice.ListWorkflowExecutionsResponse{Executions: executions}, nil).Once()
}

func (f *pipelineStepsFixture) serve(t *testing.T, query string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/steps"+query, nil)
	req.SetPathValue("id", f.pipelineID)
	req.SetPathValue("workflow_id", "wf-1")
	req.SetPathValue("run_id", "run-1")
	rec := httptest.NewRecorder()
	err := HandleListPipelineExecutionSteps()(&core.RequestEvent{
		App:   f.app,
		Auth:  f.authRecord,
		Event: router.Event{Request: req, Response: rec},
	})
	requireHandlerErrorHandled(t, rec, err)
	return rec
}

func stepsPayloads(t *testing.T, value any) *common.Payloads {
	t.Helper()
	payloads, err := temporalcrypto.DataConverter().ToPayloads(value)
	require.NoError(t, err)
	return payloads
}

func stepsStartedEvent(
	t *testing.T,
	steps ...pipelineinternal.StepDefinition,
) *historypb.HistoryEvent {
	t.Helper()
	return &historypb.HistoryEvent{
		EventId:   1,
		EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
			WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
				Input: stepsPayloads(t, map[string]any{
					"workflow_definition": pipelineinternal.WorkflowDefinition{
						Name:  "steps",
						Steps: steps,
					},
				}),
			},
		},
	}
}

func stepsChildInitiatedEvent(
	t *testing.T,
	id int64,
	typeName, workflowID string,
	input any,
) *historypb.HistoryEvent {
	t.Helper()
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED,
		Attributes: &historypb.HistoryEvent_StartChildWorkflowExecutionInitiatedEventAttributes{
			StartChildWorkflowExecutionInitiatedEventAttributes: &historypb.StartChildWorkflowExecutionInitiatedEventAttributes{
				WorkflowId:   workflowID,
				WorkflowType: &common.WorkflowType{Name: typeName},
				Input:        stepsPayloads(t, input),
			},
		},
	}
}

func stepsChildStartedEvent(
	id, initiatedID int64,
	workflowID, runID string,
) *historypb.HistoryEvent {
	return &historypb.HistoryEvent{
		EventId:   id,
		EventType: enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_STARTED,
		Attributes: &historypb.HistoryEvent_ChildWorkflowExecutionStartedEventAttributes{
			ChildWorkflowExecutionStartedEventAttributes: &historypb.ChildWorkflowExecutionStartedEventAttributes{
				InitiatedEventId:  initiatedID,
				WorkflowExecution: &common.WorkflowExecution{WorkflowId: workflowID, RunId: runID},
			},
		},
	}
}

func stepsChildInfo(
	workflowID, runID, typeName string,
	memo *common.Memo,
	start time.Time,
) *workflow.WorkflowExecutionInfo {
	return &workflow.WorkflowExecutionInfo{
		Execution:       &common.WorkflowExecution{WorkflowId: workflowID, RunId: runID},
		ParentExecution: &common.WorkflowExecution{WorkflowId: "wf-1", RunId: "run-1"},
		Type:            &common.WorkflowType{Name: typeName},
		Memo:            memo,
		Status:          enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
		StartTime:       timestamppb.New(start),
	}
}

func TestHandleListPipelineExecutionStepsListsActivitiesAndWorkflows(t *testing.T) {
	f := newPipelineStepsFixture(t)
	f.describePipelineRun(f.identifier)
	f.history("wf-1", "run-1",
		stepsStartedEvent(t,
			pipelineinternal.StepDefinition{
				StepSpec: pipelineinternal.StepSpec{ID: "fetch", Use: "http-request"},
			},
			pipelineinternal.StepDefinition{
				StepSpec: pipelineinternal.StepSpec{ID: "check", Use: "conformance-check"},
			},
		),
		&historypb.HistoryEvent{
			EventId:   5,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: &historypb.HistoryEvent_ActivityTaskScheduledEventAttributes{
				ActivityTaskScheduledEventAttributes: &historypb.ActivityTaskScheduledEventAttributes{
					ActivityType: &common.ActivityType{Name: "Make an HTTP request"},
					Input: stepsPayloads(t, map[string]any{
						"config": map[string]any{workflowengine.StepIDConfigKey: "fetch"},
					}),
				},
			},
		},
		&historypb.HistoryEvent{
			EventId:   7,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
			Attributes: &historypb.HistoryEvent_ActivityTaskCompletedEventAttributes{
				ActivityTaskCompletedEventAttributes: &historypb.ActivityTaskCompletedEventAttributes{
					ScheduledEventId: 5,
				},
			},
		},
		stepsChildInitiatedEvent(t, 8, "Conformance check", "wf-1-check", map[string]any{
			"config": map[string]any{workflowengine.StepIDConfigKey: "check"},
		}),
		stepsChildStartedEvent(9, 8, "wf-1-check", "check-run"),
		&historypb.HistoryEvent{
			EventId:   10,
			EventType: enums.EVENT_TYPE_CHILD_WORKFLOW_EXECUTION_FAILED,
			Attributes: &historypb.HistoryEvent_ChildWorkflowExecutionFailedEventAttributes{
				ChildWorkflowExecutionFailedEventAttributes: &historypb.ChildWorkflowExecutionFailedEventAttributes{
					InitiatedEventId: 8,
					WorkflowExecution: &common.WorkflowExecution{
						WorkflowId: "wf-1-check",
						RunId:      "check-run",
					},
					Failure: &failurepb.Failure{Message: "check failed"},
				},
			},
		},
	)
	f.children("wf-1", stepsChildInfo(
		"wf-1-check",
		"check-run",
		"Conformance check",
		temporalMemoWithLogsCapability(t, true),
		f.historyStart,
	))

	rec := f.serve(t, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var response PipelineExecutionStepsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, []PipelineExecutionStep{
		{
			StepID:      "fetch",
			Kind:        "activity",
			Type:        "Make an HTTP request",
			DisplayName: "fetch",
			Status:      "Completed",
			StartTime:   "2026-11-11T10:00:01Z",
			EndTime:     "2026-11-11T10:00:02Z",
			Duration:    calculateDuration("2026-11-11T10:00:01Z", "2026-11-11T10:00:02Z"),
			HistoryEvent: PipelineStepHistoryEvent{
				WorkflowID: "wf-1",
				RunID:      "run-1",
				EventID:    5,
			},
		},
		{
			StepID:        "check",
			Kind:          "workflow",
			Type:          "Conformance check",
			DisplayName:   "check",
			Status:        "Failed",
			StartTime:     "2026-11-11T10:00:03Z",
			EndTime:       "2026-11-11T10:00:05Z",
			Duration:      calculateDuration("2026-11-11T10:00:03Z", "2026-11-11T10:00:05Z"),
			FailureReason: "check failed",
			HistoryEvent: PipelineStepHistoryEvent{
				WorkflowID: "wf-1",
				RunID:      "run-1",
				EventID:    8,
			},
			Execution: &WorkflowIdentifier{WorkflowID: "wf-1-check", RunID: "check-run"},
			HasLogs:   true,
		},
	}, response.Steps)
}

func TestHandleListPipelineExecutionStepsCountsBlockAttempts(t *testing.T) {
	f := newPipelineStepsFixture(t)
	f.describePipelineRun(f.identifier)
	f.history("wf-1", "run-1",
		stepsStartedEvent(t,
			pipelineinternal.StepDefinition{
				StepSpec: pipelineinternal.StepSpec{ID: "offer", Use: "credential-offer"},
			},
			pipelineinternal.StepDefinition{
				StepSpec: pipelineinternal.StepSpec{ID: "wallet", Use: "mobile-automation"},
				LinkedTo: "offer",
			},
		),
		stepsChildInitiatedEvent(
			t,
			5,
			pipelineinternal.LinkedStepsWorkflowName,
			"wf-1-offer",
			map[string]any{},
		),
		stepsChildStartedEvent(6, 5, "wf-1-offer", "block-run-1"),
	)
	payload, err := converter.GetDefaultDataConverter().ToPayload(workflowengine.LinkedStepsMemo{
		Steps:           []string{"offer", "wallet"},
		MaximumAttempts: 3,
	})
	require.NoError(t, err)
	memo := &common.Memo{Fields: map[string]*common.Payload{
		workflowengine.LinkedStepsMemoKey: payload,
	}}
	f.children("wf-1",
		stepsChildInfo(
			"wf-1-offer",
			"block-run-2",
			pipelineinternal.LinkedStepsWorkflowName,
			memo,
			f.historyStart.Add(time.Minute),
		),
		stepsChildInfo(
			"wf-1-offer",
			"block-run-1",
			pipelineinternal.LinkedStepsWorkflowName,
			memo,
			f.historyStart,
		),
	)

	// The latest block run has started its first step only.
	f.history("wf-1-offer", "block-run-2",
		&historypb.HistoryEvent{
			EventId:   1,
			EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
				WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
					Input: stepsPayloads(t, map[string]any{
						"steps": []pipelineinternal.StepDefinition{
							{
								StepSpec: pipelineinternal.StepSpec{
									ID:  "offer",
									Use: "credential-offer",
								},
							},
							{
								StepSpec: pipelineinternal.StepSpec{
									ID:  "wallet",
									Use: "mobile-automation",
								},
								LinkedTo: "offer",
							},
						},
					}),
				},
			},
		},
		&historypb.HistoryEvent{
			EventId:   5,
			EventType: enums.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
			Attributes: &historypb.HistoryEvent_ActivityTaskScheduledEventAttributes{
				ActivityTaskScheduledEventAttributes: &historypb.ActivityTaskScheduledEventAttributes{
					Input: stepsPayloads(t, map[string]any{
						"config": map[string]any{workflowengine.StepIDConfigKey: "offer"},
					}),
				},
			},
		},
	)

	rec := f.serve(t, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var response PipelineExecutionStepsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Steps, 1)
	block := response.Steps[0]
	require.Equal(t, "block", block.Kind)
	require.Equal(t, "offer → wallet", block.DisplayName)
	require.Equal(t, "Running", block.Status)
	require.Equal(t, int32(2), block.Attempt)
	require.Equal(t, int32(3), block.MaxAttempts)
	require.Equal(
		t,
		&WorkflowIdentifier{WorkflowID: "wf-1-offer", RunID: "block-run-2"},
		block.Execution,
	)
	require.NotNil(t, block.StepCount)
	require.Equal(t, 1, *block.StepCount)
}

func TestHandleListPipelineExecutionStepsRejectsABlockOfAnotherRun(t *testing.T) {
	f := newPipelineStepsFixture(t)
	f.describePipelineRun(f.identifier)
	f.mockClient.On("DescribeWorkflowExecution", mock.Anything, "wf-2-offer", "block-run").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Execution: &common.WorkflowExecution{WorkflowId: "wf-2-offer", RunId: "block-run"},
				Type:      &common.WorkflowType{Name: pipelineinternal.LinkedStepsWorkflowName},
				ParentExecution: &common.WorkflowExecution{
					WorkflowId: "wf-2",
					RunId:      "run-2",
				},
			},
		}, nil).
		Once()

	rec := f.serve(t, "?block_workflow_id=wf-2-offer&block_run_id=block-run")
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleListPipelineExecutionStepsNeedsBothBlockIDs(t *testing.T) {
	f := newPipelineStepsFixture(t)
	f.describePipelineRun(f.identifier)

	rec := f.serve(t, "?block_workflow_id=wf-1-offer")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleListPipelineExecutionStepsRejectsARunOfAnotherPipeline(t *testing.T) {
	f := newPipelineStepsFixture(t)
	f.describePipelineRun("someone/another-pipeline")

	rec := f.serve(t, "")
	require.Equal(t, http.StatusNotFound, rec.Code)
}
