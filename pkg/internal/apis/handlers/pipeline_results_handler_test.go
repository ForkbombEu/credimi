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
	"github.com/forkbombeu/credimi/pkg/internal/runners"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestParsePaginationParamsPipelineResults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?limit=5&offset=2", nil)
	limit, offset := parsePaginationParams(&core.RequestEvent{
		Event: router.Event{Request: req},
	}, 20, 0)
	require.Equal(t, 5, limit)
	require.Equal(t, 2, offset)

	req = httptest.NewRequest(http.MethodGet, "/?limit=-1&offset=bad", nil)
	limit, offset = parsePaginationParams(&core.RequestEvent{
		Event: router.Event{Request: req},
	}, 20, 0)
	require.Equal(t, 20, limit)
	require.Equal(t, 0, offset)
}

func TestEscapeTemporalQueryValue(t *testing.T) {
	got := escapeTemporalQueryValue(`foo"bar\baz`)
	require.Equal(t, `foo\"bar\\baz`, got)
}

func TestBuildPipelineWorkflowsQuery(t *testing.T) {
	query := buildPipelineWorkflowsQuery(
		[]enums.WorkflowExecutionStatus{
			enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
			enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
		},
		"/tenant-1/pipeline/",
	)

	expected := fmt.Sprintf(
		`WorkflowType="%s" and (ExecutionStatus=%d or ExecutionStatus=%d) and %s="%s"`,
		pipeline.NewPipelineWorkflow().Name(),
		enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
		enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
		workflowengine.PipelineIdentifierSearchAttribute,
		"tenant-1/pipeline",
	)
	require.Equal(t, expected, query)
}

func TestListPipelineWorkflowExecutionsPagination(t *testing.T) {
	mockClient := &temporalmocks.Client{}

	page1 := &workflowservice.ListWorkflowExecutionsResponse{
		Executions: []*workflow.WorkflowExecutionInfo{
			{
				Execution: &common.WorkflowExecution{WorkflowId: "wf-1", RunId: "run-1"},
				Type:      &common.WorkflowType{Name: pipeline.NewPipelineWorkflow().Name()},
				Status:    enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
			},
			{
				Execution: &common.WorkflowExecution{WorkflowId: "wf-2", RunId: "run-2"},
				Type:      &common.WorkflowType{Name: pipeline.NewPipelineWorkflow().Name()},
				Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			},
		},
		NextPageToken: []byte("next"),
	}
	page2 := &workflowservice.ListWorkflowExecutionsResponse{
		Executions: []*workflow.WorkflowExecutionInfo{
			{
				Execution: &common.WorkflowExecution{WorkflowId: "wf-3", RunId: "run-3"},
				Type:      &common.WorkflowType{Name: pipeline.NewPipelineWorkflow().Name()},
				Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			},
		},
	}

	statusFilters := []enums.WorkflowExecutionStatus{
		enums.WORKFLOW_EXECUTION_STATUS_RUNNING,
	}
	pipelineIdentifier := "tenant-1/pipeline"
	expectedQuery := buildPipelineWorkflowsQuery(statusFilters, pipelineIdentifier)

	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return req.GetNamespace() == "default" &&
					req.GetQuery() == expectedQuery &&
					req.GetPageSize() == int32(2) &&
					len(req.GetNextPageToken()) == 0
			}),
		).
		Return(page1, nil).
		Once()
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return req.GetNamespace() == "default" &&
					req.GetQuery() == expectedQuery &&
					req.GetPageSize() == int32(2) &&
					string(req.GetNextPageToken()) == "next"
			}),
		).
		Return(page2, nil).
		Once()

	results, err := listPipelineWorkflowExecutions(
		context.Background(),
		mockClient,
		"default",
		statusFilters,
		pipelineIdentifier,
		2,
		1,
	)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, "wf-2", results[0].Execution.WorkflowID)
	require.Equal(t, "wf-3", results[1].Execution.WorkflowID)

	mockClient.AssertExpectations(t)
}

func TestListPipelineWorkflowExecutionsError(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.AnythingOfType("*workflowservice.ListWorkflowExecutionsRequest"),
		).
		Return((*workflowservice.ListWorkflowExecutionsResponse)(nil), errors.New("boom")).
		Once()

	_, err := listPipelineWorkflowExecutions(
		context.Background(),
		mockClient,
		"default",
		nil,
		"",
		1,
		0,
	)
	require.Error(t, err)

	mockClient.AssertExpectations(t)
}

func TestResolvePipelineIdentifiersForExecutionsFallback(t *testing.T) {
	app, _, pipelineRecord := setupPipelineResultsApp(t)
	defer app.Cleanup()

	orgID := pipelineRecord.GetString("owner")
	createPipelineResult(t, app, orgID, pipelineRecord.Id, "wf-1", "run-1")

	exec := &WorkflowExecution{
		Execution: &WorkflowIdentifier{
			WorkflowID: "wf-1",
			RunID:      "run-1",
		},
	}

	identifiers, err := resolvePipelineIdentifiersForExecutions(
		app,
		[]*WorkflowExecution{exec},
		orgID,
	)
	require.NoError(t, err)

	expectedPath, err := canonify.BuildPath(
		app,
		pipelineRecord,
		canonify.CanonifyPaths["pipelines"],
		"",
	)
	require.NoError(t, err)

	ref := workflowExecutionRef{WorkflowID: "wf-1", RunID: "run-1"}
	require.Equal(t, strings.Trim(expectedPath, "/"), identifiers[ref])
}

func TestBuildWorkflowExecutionSummaryDuration(t *testing.T) {
	start := "2025-01-01T00:00:00Z"
	end := "2025-01-01T01:02:03Z"
	exec := &WorkflowExecution{
		Execution: &WorkflowIdentifier{WorkflowID: "wf-1", RunID: "run-1"},
		Type:      WorkflowType{Name: "example"},
		StartTime: start,
		CloseTime: end,
		Status:    "WORKFLOW_EXECUTION_STATUS_COMPLETED",
	}

	summary := buildWorkflowExecutionSummary(exec, nil)
	require.NotNil(t, summary)
	require.Equal(t, "Completed", summary.Status)
	require.Equal(t, "1h 2m 3s", summary.Duration)
}

func TestBuildWorkflowExecutionSummaryFailureReason(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	iter := &fakeHistoryIterator{
		events: []*historypb.HistoryEvent{
			{
				EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED,
				Attributes: &historypb.HistoryEvent_WorkflowExecutionFailedEventAttributes{
					WorkflowExecutionFailedEventAttributes: &historypb.WorkflowExecutionFailedEventAttributes{
						Failure: &failurepb.Failure{
							Cause: &failurepb.Failure{Message: "boom"},
						},
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
			enums.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT,
		).
		Return(iter, nil).
		Once()

	exec := &WorkflowExecution{
		Execution: &WorkflowIdentifier{WorkflowID: "wf-1", RunID: "run-1"},
		Type:      WorkflowType{Name: "example"},
		StartTime: "2025-01-01T00:00:00Z",
		CloseTime: "2025-01-01T00:01:00Z",
		Status:    "WORKFLOW_EXECUTION_STATUS_FAILED",
	}

	summary := buildWorkflowExecutionSummary(exec, mockClient)
	require.NotNil(t, summary)
	require.NotNil(t, summary.FailureReason)
	require.Equal(t, "boom", *summary.FailureReason)
}

func TestFetchCompletedWorkflowsWithPaginationLimitZero(t *testing.T) {
	app, authRecord, pipelineRecord := setupPipelineResultsApp(t)
	defer app.Cleanup()

	pipelineMap := map[string]*core.Record{pipelineRecord.Id: pipelineRecord}

	summaries, apiErr := fetchCompletedWorkflowsWithPagination(
		&core.RequestEvent{App: app, Auth: authRecord},
		pipelineMap,
		"ns-1",
		authRecord,
		pipelineRecord.GetString("owner"),
		"",
		0,
		0,
	)
	require.Nil(t, apiErr)
	require.Empty(t, summaries)
}

func TestFetchCompletedWorkflowsWithPaginationClientError(t *testing.T) {
	app, authRecord, pipelineRecord := setupPipelineResultsApp(t)
	defer app.Cleanup()

	restore := installPipelineResultsSeams(t)
	t.Cleanup(restore)

	pipelineResultsTemporalClient = func(string) (client.Client, error) {
		return nil, errors.New("no client")
	}

	pipelineMap := map[string]*core.Record{pipelineRecord.Id: pipelineRecord}

	_, apiErr := fetchCompletedWorkflowsWithPagination(
		&core.RequestEvent{App: app, Auth: authRecord},
		pipelineMap,
		"ns-1",
		authRecord,
		pipelineRecord.GetString("owner"),
		"",
		1,
		0,
	)
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusInternalServerError, apiErr.Code)
}

func TestFetchCompletedWorkflowsWithPaginationSkipsAndFilters(t *testing.T) {
	app, authRecord, pipelineRecord := setupPipelineResultsApp(t)
	defer app.Cleanup()

	orgID := pipelineRecord.GetString("owner")
	createPipelineResult(t, app, orgID, pipelineRecord.Id, "wf-1", "run-1")
	createPipelineResult(t, app, orgID, pipelineRecord.Id, "wf-2", "run-2")

	restore := installPipelineResultsSeams(t)
	t.Cleanup(restore)

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
				return strings.Contains(req.GetQuery(), "WorkflowType") &&
					!strings.Contains(req.GetQuery(), "ParentWorkflowId")
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				buildPipelineExecutionInfo(
					t,
					"wf-1",
					"run-1",
					pipelineIdentifier,
					enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
				),
				buildPipelineExecutionInfo(
					t,
					"wf-2",
					"run-2",
					pipelineIdentifier,
					enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
				),
			},
		}, nil).
		Once()
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return strings.Contains(req.GetQuery(), "ParentWorkflowId")
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil).
		Maybe()
	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).
		Return(&fakeHistoryIterator{events: []*historypb.HistoryEvent{}}).
		Maybe()

	pipelineResultsTemporalClient = func(string) (client.Client, error) {
		return mockClient, nil
	}

	pipelineMap := map[string]*core.Record{pipelineRecord.Id: pipelineRecord}

	summaries, apiErr := fetchCompletedWorkflowsWithPagination(
		&core.RequestEvent{App: app, Auth: authRecord},
		pipelineMap,
		"ns-1",
		authRecord,
		orgID,
		"Completed",
		2,
		0,
	)
	require.Nil(t, apiErr)
	require.Len(t, summaries, 2)
	for _, summary := range summaries {
		require.Equal(t, pipelineIdentifier, summary.PipelineIdentifier)
	}
}

func TestBuildPipelineExecutionHierarchyFromResult(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	app.Settings().Meta.AppURL = "https://example.test"

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Id = "result-1"
	record.Set("video_results", []string{"sample_result_video_1.mp4"})
	record.Set("screenshots", []string{"sample_screenshot_1.png"})

	pipelineWf := pipeline.PipelineWorkflow{}
	root := &WorkflowExecution{
		Execution: &WorkflowIdentifier{WorkflowID: "wf-1", RunID: "run-1"},
		Type:      WorkflowType{Name: pipelineWf.Name()},
		StartTime: "2025-01-01T00:00:00Z",
		CloseTime: "2025-01-01T00:01:00Z",
		Status:    "WORKFLOW_EXECUTION_STATUS_COMPLETED",
	}
	child := &WorkflowExecution{
		Execution: &WorkflowIdentifier{WorkflowID: "child-1", RunID: "run-2"},
		Type:      WorkflowType{Name: "ChildWorkflow"},
		StartTime: "2025-01-01T00:02:00Z",
		CloseTime: "2025-01-01T00:03:00Z",
		Status:    "WORKFLOW_EXECUTION_STATUS_COMPLETED",
	}

	summaries := buildPipelineExecutionHierarchyFromResult(
		app,
		record,
		root,
		[]*WorkflowExecution{child},
		"default",
		"UTC",
		nil,
	)
	require.Len(t, summaries, 1)
	require.NotEmpty(t, summaries[0].Results)
	require.Len(t, summaries[0].Children, 1)
}

func TestHandleGetPipelineResultsQueuedOnly(t *testing.T) {
	app, authRecord, pipelineRecord := setupPipelineResultsApp(t)
	defer app.Cleanup()

	restore := installPipelineResultsSeams(t)
	defer restore()

	pipelinePath, err := canonify.BuildPath(
		app,
		pipelineRecord,
		canonify.CanonifyPaths["pipelines"],
		"",
	)
	require.NoError(t, err)
	pipelineIdentifier := strings.Trim(pipelinePath, "/")

	pipelineResultsListQueuedRuns = func(_ context.Context, _ string) (map[string]QueuedPipelineRunAggregate, error) {
		return map[string]QueuedPipelineRunAggregate{
			"ticket-1": {
				TicketID:           "ticket-1",
				PipelineIdentifier: pipelineIdentifier,
				EnqueuedAt:         time.Now().Add(-time.Minute),
				LeaderRunnerID:     "runner-1",
				RequiredRunnerIDs:  []string{"runner-1"},
				RunnerIDs:          []string{"runner-1"},
				Status:             workflowengine.MobileRunnerSemaphoreRunQueued,
				Position:           0,
				LineLen:            1,
			},
		}, nil
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/pipeline/results?status=queued",
		nil,
	)
	rec := httptest.NewRecorder()

	err = HandleGetPipelineResults()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var summaries []pipelineWorkflowSummary
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&summaries))
	require.Len(t, summaries, 1)
	require.Equal(t, string(WorkflowStatusQueued), summaries[0].Status)
	require.Equal(t, pipelineIdentifier, summaries[0].PipelineIdentifier)
}

func TestHandleGetPipelineResultsCompletedOnly(t *testing.T) {
	app, authRecord, pipelineRecord := setupPipelineResultsApp(t)
	defer app.Cleanup()

	orgID := pipelineRecord.GetString("owner")
	createPipelineResult(t, app, orgID, pipelineRecord.Id, "wf-1", "run-1")

	restore := installPipelineResultsSeams(t)
	defer restore()

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
				return strings.Contains(req.GetQuery(), "WorkflowType") &&
					!strings.Contains(req.GetQuery(), "ParentWorkflowId")
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				buildPipelineExecutionInfo(
					t,
					"wf-1",
					"run-1",
					pipelineIdentifier,
					enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
				),
			},
		}, nil).
		Once()
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return strings.Contains(req.GetQuery(), "ParentWorkflowId")
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil).
		Maybe()
	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-1",
			"run-1",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(&fakeHistoryIterator{
			events: []*historypb.HistoryEvent{
				{
					EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
					Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
						WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{},
					},
				},
			},
		}, nil).
		Once()

	pipelineResultsTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}
	pipelineResultsListQueuedRuns = func(_ context.Context, _ string) (map[string]QueuedPipelineRunAggregate, error) {
		t.Fatalf("queued runs should not be requested for status filter")
		return nil, nil
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/pipeline/results?status=completed",
		nil,
	)
	rec := httptest.NewRecorder()

	err = HandleGetPipelineResults()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var summaries []pipelineWorkflowSummary
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&summaries))
	require.Len(t, summaries, 1)
	require.Equal(t, string(WorkflowStatusCompleted), summaries[0].Status)
	require.Equal(t, pipelineIdentifier, summaries[0].PipelineIdentifier)

	mockClient.AssertExpectations(t)
}

func TestHandleGetPipelineResultsMixed(t *testing.T) {
	app, authRecord, pipelineRecord := setupPipelineResultsApp(t)
	defer app.Cleanup()

	orgID := pipelineRecord.GetString("owner")
	createPipelineResult(t, app, orgID, pipelineRecord.Id, "wf-2", "run-2")

	restore := installPipelineResultsSeams(t)
	defer restore()

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
				return strings.Contains(req.GetQuery(), "WorkflowType") &&
					!strings.Contains(req.GetQuery(), "ParentWorkflowId")
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				buildPipelineExecutionInfo(
					t,
					"wf-2",
					"run-2",
					pipelineIdentifier,
					enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
				),
			},
		}, nil).
		Once()
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.MatchedBy(func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
				return strings.Contains(req.GetQuery(), "ParentWorkflowId")
			}),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil).
		Maybe()
	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-2",
			"run-2",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(&fakeHistoryIterator{
			events: []*historypb.HistoryEvent{
				{
					EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
					Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
						WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{},
					},
				},
			},
		}, nil).
		Once()

	pipelineResultsTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}
	pipelineResultsListQueuedRuns = func(_ context.Context, _ string) (map[string]QueuedPipelineRunAggregate, error) {
		return map[string]QueuedPipelineRunAggregate{
			"ticket-2": {
				TicketID:           "ticket-2",
				PipelineIdentifier: pipelineIdentifier,
				EnqueuedAt:         time.Now().Add(-2 * time.Minute),
				LeaderRunnerID:     "runner-2",
				RequiredRunnerIDs:  []string{"runner-2"},
				RunnerIDs:          []string{"runner-2"},
				Status:             workflowengine.MobileRunnerSemaphoreRunQueued,
				Position:           0,
				LineLen:            1,
			},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/results", nil)
	rec := httptest.NewRecorder()

	err = HandleGetPipelineResults()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var summaries []pipelineWorkflowSummary
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&summaries))
	require.Len(t, summaries, 2)

	statuses := []string{summaries[0].Status, summaries[1].Status}
	require.Contains(t, statuses, string(WorkflowStatusQueued))
	require.Contains(t, statuses, string(WorkflowStatusCompleted))
	for _, summary := range summaries {
		if summary.Status == string(WorkflowStatusQueued) ||
			summary.Status == string(WorkflowStatusCompleted) {
			require.Equal(t, pipelineIdentifier, summary.PipelineIdentifier)
		}
	}

	mockClient.AssertExpectations(t)
}

func TestBuildChildWorkflowParentQueryPipelineResults(t *testing.T) {
	query := buildChildWorkflowParentQuery([]workflowExecutionRef{
		{WorkflowID: "wf-1", RunID: "run-1"},
		{WorkflowID: "wf\"2", RunID: "run\\2"},
		{WorkflowID: "", RunID: "skip"},
	})
	require.Contains(t, query, `ParentWorkflowId="wf-1"`)
	require.Contains(t, query, `ParentRunId="run-1"`)
	require.Contains(t, query, `ParentWorkflowId="wf\"2"`)
	require.Contains(t, query, `ParentRunId="run\\2"`)
}

func TestFormatQueuedRunTimePipelineResults(t *testing.T) {
	require.Equal(t, "", formatQueuedRunTime(time.Time{}, "UTC"))

	ts := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	formatted := formatQueuedRunTime(ts, "UTC")
	require.Equal(t, "02/01/2025, 03:04:05", formatted)

	invalid := formatQueuedRunTime(ts, "bad/timezone")
	require.NotEmpty(t, invalid)
}

func TestMapQueuedRunsToPipelinesPipelineResults(t *testing.T) {
	app, _, pipelineRecord := setupPipelineResultsApp(t)
	defer app.Cleanup()

	org, err := app.FindRecordById("organizations", pipelineRecord.GetString("owner"))
	require.NoError(t, err)

	pipelineRecord.Set("canonified_name", "pipeline-1")
	require.NoError(t, app.Save(pipelineRecord))

	orgName := org.GetString("canonified_name")
	if orgName == "" {
		orgName = "org-1"
		org.Set("canonified_name", orgName)
		require.NoError(t, app.Save(org))
	}
	queuedRuns := map[string]QueuedPipelineRunAggregate{
		"ticket-1": {PipelineIdentifier: pipelineRecord.Id},
		"ticket-2": {PipelineIdentifier: pipelineRecord.Id},
		"ticket-3": {PipelineIdentifier: "missing"},
	}

	result := mapQueuedRunsToPipelines(app, []*core.Record{pipelineRecord}, queuedRuns)
	require.Len(t, result[pipelineRecord.Id], 2)
}

func TestAttachPipelineRunnerInfoPipelineResults(t *testing.T) {
	app, _, _ := setupPipelineResultsApp(t)
	defer app.Cleanup()

	execs := []*WorkflowExecutionSummary{
		{
			Execution: &WorkflowIdentifier{WorkflowID: "wf-1", RunID: "run-1"},
			Type:      WorkflowType{Name: "Pipeline"},
			Status:    "COMPLETED",
		},
	}

	info := runners.PipelineRunnerInfo{NeedsGlobalRunner: true}
	out := attachPipelineRunnerInfo(app, execs, "runner-1", info, map[string]map[string]any{})
	require.Len(t, out, 1)
	require.Equal(t, "runner-1", out[0].GlobalRunnerID)
	require.Equal(t, []string{"runner-1"}, out[0].RunnerIDs)
}

func TestAttachRunnerInfoFromTemporalStartInputPipelineResults(t *testing.T) {
	payloads, err := converter.GetDefaultDataConverter().ToPayloads(
		pipeline.PipelineWorkflowInput{
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{"global_runner_id": "runner-1"},
			},
		},
	)
	require.NoError(t, err)

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

	mockClient := &temporalmocks.Client{}
	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-1",
			"run-1",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(iter, nil)

	execs := []*WorkflowExecutionSummary{
		{
			Execution: &WorkflowIdentifier{WorkflowID: "wf-1", RunID: "run-1"},
			Type:      WorkflowType{Name: "Pipeline"},
			Status:    "COMPLETED",
		},
	}

	app, _, _ := setupPipelineResultsApp(t)
	defer app.Cleanup()

	out, err := attachRunnerInfoFromTemporalStartInput(attachRunnerInfoFromTemporalInputArgs{
		App:         app,
		Ctx:         context.Background(),
		Client:      mockClient,
		Executions:  execs,
		Info:        runners.PipelineRunnerInfo{NeedsGlobalRunner: true},
		RunnerCache: map[string]map[string]any{},
	})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "runner-1", out[0].GlobalRunnerID)
	require.Equal(t, []string{"runner-1"}, out[0].RunnerIDs)
}

func TestDescribeWorkflowExecutionErrors(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return(nil, &serviceerror.NotFound{Message: "missing"})

	_, apiErr := describeWorkflowExecution(mockClient, "wf-1", "run-1")
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusNotFound, apiErr.Code)

	mockClient = &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-2", "run-2").
		Return(nil, &serviceerror.InvalidArgument{Message: "bad"})
	_, apiErr = describeWorkflowExecution(mockClient, "wf-2", "run-2")
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
}

func TestDescribeWorkflowExecutionWithParent(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-3", "run-3").
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
				Execution: &common.WorkflowExecution{WorkflowId: "wf-3", RunId: "run-3"},
				ParentExecution: &common.WorkflowExecution{
					WorkflowId: "parent",
					RunId:      "run-parent",
				},
				Type: &common.WorkflowType{Name: "Pipeline"},
			},
		}, nil)

	exec, apiErr := describeWorkflowExecution(mockClient, "wf-3", "run-3")
	require.Nil(t, apiErr)
	require.NotNil(t, exec.ParentExecution)
	require.Equal(t, "parent", exec.ParentExecution.WorkflowID)
}

func TestBuildQueuedPipelineSummaries(t *testing.T) {
	queued := []QueuedPipelineRunAggregate{
		{
			TicketID:           "t1",
			PipelineIdentifier: "pipe-1",
			Position:           0,
			LineLen:            2,
			EnqueuedAt:         time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		},
	}

	summaries := buildQueuedPipelineSummaries(nil, queued, "UTC", map[string]map[string]any{})
	require.Len(t, summaries, 1)
	require.Equal(t, "pipe-1", summaries[0].DisplayName)
	require.Equal(t, "t1", summaries[0].Queue.TicketID)
	require.Equal(t, 1, summaries[0].Queue.Position)
}

func TestAppendQueuedPipelineSummaries(t *testing.T) {
	response := map[string][]*pipelineWorkflowSummary{
		"pipe-1": {{WorkflowExecutionSummary: WorkflowExecutionSummary{Status: "COMPLETED"}}},
	}
	queuedByPipeline := map[string][]QueuedPipelineRunAggregate{
		"pipe-1": {{
			TicketID:           "t1",
			PipelineIdentifier: "pipe-1",
			Position:           0,
			LineLen:            1,
		}},
	}

	appendQueuedPipelineSummaries(
		nil,
		response,
		queuedByPipeline,
		"UTC",
		map[string]map[string]any{},
	)
	require.Len(t, response["pipe-1"], 2)
	require.Equal(t, string(WorkflowStatusQueued), response["pipe-1"][0].Status)
}

func TestReadGlobalRunnerIDFromTemporalHistory(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	iter := &fakeHistoryIterator{
		events: []*historypb.HistoryEvent{
			{EventType: enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED},
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
		Return(iter, nil)
	value, err := readGlobalRunnerIDFromTemporalHistory(
		context.Background(),
		mockClient,
		"wf-1",
		"run-1",
	)
	require.NoError(t, err)
	require.Equal(t, "", value)

	mockClient = &temporalmocks.Client{}
	iter = &fakeHistoryIterator{
		events: []*historypb.HistoryEvent{
			{
				EventType: enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
				Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
					WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
						Input: nil,
					},
				},
			},
		},
	}
	mockClient.
		On(
			"GetWorkflowHistory",
			mock.Anything,
			"wf-2",
			"run-2",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).
		Return(iter, nil)
	value, err = readGlobalRunnerIDFromTemporalHistory(
		context.Background(),
		mockClient,
		"wf-2",
		"run-2",
	)
	require.NoError(t, err)
	require.Equal(t, "", value)
}

func TestComputePipelineResultsFromRecordPipelineResultsHandler(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	app.Settings().Meta.AppURL = "https://example.test"

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Id = "result-1"
	record.Set("video_results", []string{"sample_result_video_1.mp4", "orphan.mp4"})
	record.Set("screenshots", []string{"sample_screenshot_1.png", "extra.png"})

	results := computePipelineResultsFromRecord(app, record)
	require.Len(t, results, 1)
	require.Contains(t, results[0].Video, "sample_result_video_1.mp4")
	require.Contains(t, results[0].Screenshot, "sample_screenshot_1.png")
}

func TestGetChildWorkflowsByParents(t *testing.T) {
	mockClient := &temporalmocks.Client{}

	parentA := workflowExecutionRef{WorkflowID: "wf-1", RunID: "run-1"}
	parentB := workflowExecutionRef{WorkflowID: "wf-2", RunID: "run-2"}

	page1 := &workflowservice.ListWorkflowExecutionsResponse{
		Executions: []*workflow.WorkflowExecutionInfo{
			{
				Execution: &common.WorkflowExecution{
					WorkflowId: "child-1",
					RunId:      "run-child-1",
				},
				ParentExecution: &common.WorkflowExecution{WorkflowId: "wf-1", RunId: "run-1"},
				Type:            &common.WorkflowType{Name: "Pipeline"},
				Status:          enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			},
			{
				Execution: &common.WorkflowExecution{
					WorkflowId: "child-skip",
					RunId:      "run-skip",
				},
				ParentExecution: &common.WorkflowExecution{
					WorkflowId: "missing",
					RunId:      "run-missing",
				},
				Type:   &common.WorkflowType{Name: "Pipeline"},
				Status: enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			},
		},
		NextPageToken: []byte("next"),
	}

	page2 := &workflowservice.ListWorkflowExecutionsResponse{
		Executions: []*workflow.WorkflowExecutionInfo{
			{
				Execution: &common.WorkflowExecution{
					WorkflowId: "child-2",
					RunId:      "run-child-2",
				},
				ParentExecution: &common.WorkflowExecution{WorkflowId: "wf-2", RunId: "run-2"},
				Type:            &common.WorkflowType{Name: "Pipeline"},
				Status:          enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			},
		},
	}

	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.AnythingOfType("*workflowservice.ListWorkflowExecutionsRequest"),
		).
		Return(page1, nil).
		Once()
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.AnythingOfType("*workflowservice.ListWorkflowExecutionsRequest"),
		).
		Return(page2, nil).
		Once()

	children, err := getChildWorkflowsByParents(
		context.Background(),
		mockClient,
		"default",
		[]workflowExecutionRef{parentA, parentB, parentA, {}},
	)
	require.NoError(t, err)
	require.Len(t, children[parentA], 1)
	require.Len(t, children[parentB], 1)

	mockClient.AssertExpectations(t)
}

func TestGetChildWorkflowsByParentsError(t *testing.T) {
	mockClient := &temporalmocks.Client{}
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.AnythingOfType("*workflowservice.ListWorkflowExecutionsRequest"),
		).
		Return((*workflowservice.ListWorkflowExecutionsResponse)(nil), errors.New("boom")).
		Once()

	parent := workflowExecutionRef{WorkflowID: "wf-1", RunID: "run-1"}
	children, err := getChildWorkflowsByParents(
		context.Background(),
		mockClient,
		"default",
		[]workflowExecutionRef{parent},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
	require.Contains(t, children, parent)

	mockClient.AssertExpectations(t)
}

func setupPipelineResultsApp(t testing.TB) (*tests.TestApp, *core.Record, *core.Record) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	pipelineRecord := createPipelineRecord(t, app, orgID, "pipeline123")

	return app, authRecord, pipelineRecord
}

func createPipelineRecord(t testing.TB, app *tests.TestApp, orgID, name string) *core.Record {
	coll, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("name", name)
	record.Set("description", "test-description")
	yaml := "name: " + name + "\nsteps:\n  - name: step1\n    use: rest\n"
	record.Set("steps", map[string]any{"rest-chain": map[string]any{"yaml": yaml}})
	record.Set("yaml", yaml)
	require.NoError(t, app.Save(record))

	return record
}

func createPipelineResult(
	t testing.TB,
	app *tests.TestApp,
	orgID, pipelineID, workflowID, runID string,
) {
	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("pipeline", pipelineID)
	record.Set("workflow_id", workflowID)
	record.Set("run_id", runID)
	require.NoError(t, app.Save(record))
}

func buildWorkflowExecutionResponse(
	workflowID, runID string,
) *workflowservice.DescribeWorkflowExecutionResponse {
	return &workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &workflow.WorkflowExecutionInfo{
			Execution: &common.WorkflowExecution{
				WorkflowId: workflowID,
				RunId:      runID,
			},
			Type: &common.WorkflowType{
				Name: pipeline.NewPipelineWorkflow().Name(),
			},
			Status:    enums.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			StartTime: timestamppb.New(time.Now().Add(-2 * time.Minute)),
			CloseTime: timestamppb.New(time.Now().Add(-time.Minute)),
		},
	}
}

func buildPipelineExecutionInfo(
	t testing.TB,
	workflowID, runID, pipelineIdentifier string,
	status enums.WorkflowExecutionStatus,
) *workflow.WorkflowExecutionInfo {
	info := &workflow.WorkflowExecutionInfo{
		Execution: &common.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      runID,
		},
		Type: &common.WorkflowType{
			Name: pipeline.NewPipelineWorkflow().Name(),
		},
		Status:    status,
		StartTime: timestamppb.New(time.Now().Add(-2 * time.Minute)),
		CloseTime: timestamppb.New(time.Now().Add(-time.Minute)),
	}

	if pipelineIdentifier == "" {
		return info
	}

	payload, err := converter.GetDefaultDataConverter().ToPayload(pipelineIdentifier)
	require.NoError(t, err)
	info.SearchAttributes = &common.SearchAttributes{
		IndexedFields: map[string]*common.Payload{
			workflowengine.PipelineIdentifierSearchAttribute: payload,
		},
	}
	return info
}

func installPipelineResultsSeams(t testing.TB) func() {
	t.Helper()

	origClient := pipelineResultsTemporalClient
	origList := pipelineResultsListQueuedRuns

	return func() {
		pipelineResultsTemporalClient = origClient
		pipelineResultsListQueuedRuns = origList
	}
}
