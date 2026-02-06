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

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestGetPipelineDetailsIncludesQueuedRuns(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("name", "queued-pipeline")
	record.Set("description", "queued pipeline description")
	record.Set("yaml", "name: queued-pipeline\n")
	require.NoError(t, app.Save(record))

	stubQueuedRuns(t, "usera-s-organization/queued-pipeline")

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/pipeline/list-workflows", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string][]pipelineWorkflowSummary
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

		summaries := response[record.Id]
		require.Len(t, summaries, 1)
		require.NotNil(t, summaries[0].Queue)
		require.Equal(t, "ticket-queued", summaries[0].Queue.TicketID)
		require.Equal(t, 1, summaries[0].Queue.Position)
		require.Equal(t, 3, summaries[0].Queue.LineLen)
		require.Equal(t, []string{"runner-1"}, summaries[0].Queue.RunnerIDs)
		require.Equal(t, "Queued", summaries[0].Status)
		require.Equal(t, "queued-pipeline", summaries[0].DisplayName)
		require.NotNil(t, summaries[0].Execution)
		require.Equal(t, "queue/ticket-queued", summaries[0].Execution.WorkflowID)
		require.Equal(t, "ticket-queued", summaries[0].Execution.RunID)
		require.Equal(t, []string{"runner-1"}, summaries[0].RunnerIDs)

		return nil
	})
	require.NoError(t, serveErr)
}

func TestGetPipelineSpecificDetailsIncludesQueuedRuns(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	app := setupPipelineStartApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("name", "queued-pipeline-specific")
	record.Set("description", "queued pipeline specific description")
	record.Set("yaml", "name: queued-pipeline-specific\n")
	require.NoError(t, app.Save(record))

	stubQueuedRuns(t, "usera-s-organization/queued-pipeline-specific")

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodGet,
			"/api/pipeline/list-workflows/"+record.Id,
			nil,
		)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var response []pipelineWorkflowSummary
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		require.Len(t, response, 1)
		require.NotNil(t, response[0].Queue)
		require.Equal(t, "ticket-queued", response[0].Queue.TicketID)
		require.Equal(t, "Queued", response[0].Status)
		require.Equal(t, "queued-pipeline-specific", response[0].DisplayName)
		require.Equal(t, []string{"runner-1"}, response[0].RunnerIDs)

		return nil
	})
	require.NoError(t, serveErr)
}

func stubQueuedRuns(t *testing.T, pipelineID string) {
	originalList := listMobileRunnerSemaphoreWorkflows
	originalQuery := queryMobileRunnerSemaphoreQueuedRuns

	t.Cleanup(func() {
		listMobileRunnerSemaphoreWorkflows = originalList
		queryMobileRunnerSemaphoreQueuedRuns = originalQuery
	})

	listMobileRunnerSemaphoreWorkflows = func(_ context.Context) ([]string, error) {
		return []string{"runner-1"}, nil
	}

	enqueuedAt := time.Date(2026, 2, 5, 9, 0, 0, 0, time.UTC)
	queryMobileRunnerSemaphoreQueuedRuns = func(
		_ context.Context,
		_ string,
		_ string,
	) ([]workflows.MobileRunnerSemaphoreQueuedRunView, error) {
		return []workflows.MobileRunnerSemaphoreQueuedRunView{
			{
				TicketID:           "ticket-queued",
				OwnerNamespace:     "usera-s-organization",
				PipelineIdentifier: pipelineID,
				EnqueuedAt:         enqueuedAt,
				LeaderRunnerID:     "runner-1",
				RequiredRunnerIDs:  []string{"runner-1"},
				Status:             workflowengine.MobileRunnerSemaphoreRunQueued,
				Position:           1,
				LineLen:            3,
			},
		}, nil
	}
}
