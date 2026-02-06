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
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

func TestHandleListMyChecksIncludesQueuedRunsForRunningFilter(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	restoreChecks, restoreQueued := stubChecksQueuedDependencies(t)
	t.Cleanup(restoreChecks)
	t.Cleanup(restoreQueued)

	runRequest := func(url string) ListMyChecksResponse {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		err := HandleListMyChecks()(&core.RequestEvent{
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
		return resp
	}

	resp := runRequest("/api/my/checks")
	require.Len(t, resp.Executions, 1)
	require.NotNil(t, resp.Executions[0].Queue)
	require.Equal(t, "ticket-queued", resp.Executions[0].Queue.TicketID)
	require.Equal(t, "queued", resp.Executions[0].Status)

	respRunning := runRequest("/api/my/checks?status=" + statusStringRunning)
	require.Len(t, respRunning.Executions, 1)
	require.NotNil(t, respRunning.Executions[0].Queue)

	respCompleted := runRequest("/api/my/checks?status=completed")
	require.Empty(t, respCompleted.Executions)
}

func stubChecksQueuedDependencies(t *testing.T) (func(), func()) {
	t.Helper()

	originalChecksClient := listChecksTemporalClient
	originalChecksWorkflows := listChecksWorkflows
	originalListSemaphores := listMobileRunnerSemaphoreWorkflows
	originalQueryQueued := queryMobileRunnerSemaphoreQueuedRuns

	listChecksTemporalClient = func(_ string) (client.Client, error) {
		return nil, nil
	}
	listChecksWorkflows = func(
		_ context.Context,
		_ client.Client,
		_ string,
		_ string,
	) (*workflowservice.ListWorkflowExecutionsResponse, error) {
		return &workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{},
		}, nil
	}

	listMobileRunnerSemaphoreWorkflows = func(_ context.Context) ([]string, error) {
		return []string{"runner-1"}, nil
	}
	queryMobileRunnerSemaphoreQueuedRuns = func(
		_ context.Context,
		_ string,
		_ string,
	) ([]workflows.MobileRunnerSemaphoreQueuedRunView, error) {
		return []workflows.MobileRunnerSemaphoreQueuedRunView{
			{
				TicketID:           "ticket-queued",
				OwnerNamespace:     "usera-s-organization",
				PipelineIdentifier: "usera-s-organization/pipeline-queued",
				EnqueuedAt:         time.Date(2026, 2, 5, 9, 0, 0, 0, time.UTC),
				LeaderRunnerID:     "runner-1",
				RequiredRunnerIDs:  []string{"runner-1"},
				Status:             workflowengine.MobileRunnerSemaphoreRunQueued,
				Position:           0,
				LineLen:            1,
			},
		}, nil
	}

	restoreChecks := func() {
		listChecksTemporalClient = originalChecksClient
		listChecksWorkflows = originalChecksWorkflows
	}
	restoreQueued := func() {
		listMobileRunnerSemaphoreWorkflows = originalListSemaphores
		queryMobileRunnerSemaphoreQueuedRuns = originalQueryQueued
	}

	return restoreChecks, restoreQueued
}
