// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestHandleGetPipelineResultsQueuedOnly(t *testing.T) {
	app, authRecord, pipelineRecord := setupPipelineResultsApp(t)
	defer app.Cleanup()

	restore := installPipelineResultsSeams(t)
	defer restore()

	pipelineResultsListQueuedRuns = func(_ context.Context, _ string) (map[string]QueuedPipelineRunAggregate, error) {
		return map[string]QueuedPipelineRunAggregate{
			"ticket-1": {
				TicketID:           "ticket-1",
				PipelineIdentifier: pipelineRecord.Id,
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

	err := HandleGetPipelineResults()(&core.RequestEvent{
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
}

func TestHandleGetPipelineResultsCompletedOnly(t *testing.T) {
	app, authRecord, pipelineRecord := setupPipelineResultsApp(t)
	defer app.Cleanup()

	orgID := pipelineRecord.GetString("owner")
	createPipelineResult(t, app, orgID, pipelineRecord.Id, "wf-1", "run-1")

	restore := installPipelineResultsSeams(t)
	defer restore()

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-1", "run-1").
		Return(buildWorkflowExecutionResponse("wf-1", "run-1"), nil).
		Once()
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.AnythingOfType("*workflowservice.ListWorkflowExecutionsRequest"),
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

	err := HandleGetPipelineResults()(&core.RequestEvent{
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

	mockClient.AssertExpectations(t)
}

func TestHandleGetPipelineResultsMixed(t *testing.T) {
	app, authRecord, pipelineRecord := setupPipelineResultsApp(t)
	defer app.Cleanup()

	orgID := pipelineRecord.GetString("owner")
	createPipelineResult(t, app, orgID, pipelineRecord.Id, "wf-2", "run-2")

	restore := installPipelineResultsSeams(t)
	defer restore()

	mockClient := &temporalmocks.Client{}
	mockClient.
		On("DescribeWorkflowExecution", mock.Anything, "wf-2", "run-2").
		Return(buildWorkflowExecutionResponse("wf-2", "run-2"), nil).
		Once()
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.AnythingOfType("*workflowservice.ListWorkflowExecutionsRequest"),
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
				PipelineIdentifier: pipelineRecord.Id,
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

	err := HandleGetPipelineResults()(&core.RequestEvent{
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
		On("GetWorkflowHistory", mock.Anything, "wf-1", "run-1", false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT).
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

	appendQueuedPipelineSummaries(nil, response, queuedByPipeline, "UTC", map[string]map[string]any{})
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
		On("GetWorkflowHistory", mock.Anything, "wf-1", "run-1", false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT).
		Return(iter, nil)
	value, err := readGlobalRunnerIDFromTemporalHistory(context.Background(), mockClient, "wf-1", "run-1")
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
		On("GetWorkflowHistory", mock.Anything, "wf-2", "run-2", false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT).
		Return(iter, nil)
	value, err = readGlobalRunnerIDFromTemporalHistory(context.Background(), mockClient, "wf-2", "run-2")
	require.NoError(t, err)
	require.Equal(t, "", value)
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

func installPipelineResultsSeams(t testing.TB) func() {
	t.Helper()

	origClient := pipelineResultsTemporalClient
	origList := pipelineResultsListQueuedRuns

	return func() {
		pipelineResultsTemporalClient = origClient
		pipelineResultsListQueuedRuns = origList
	}
}
