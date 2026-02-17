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
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
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
		On("ListWorkflow", mock.Anything, mock.AnythingOfType("*workflowservice.ListWorkflowExecutionsRequest")).
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
		On("ListWorkflow", mock.Anything, mock.AnythingOfType("*workflowservice.ListWorkflowExecutionsRequest")).
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
) *core.Record {
	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("pipeline", pipelineID)
	record.Set("workflow_id", workflowID)
	record.Set("run_id", runID)
	require.NoError(t, app.Save(record))

	return record
}

func buildWorkflowExecutionResponse(workflowID, runID string) *workflowservice.DescribeWorkflowExecutionResponse {
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
