// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/stretchr/testify/require"
)

func TestListQueuedPipelineRunsAggregatesTickets(t *testing.T) {
	originalList := listMobileRunnerSemaphoreWorkflows
	originalQuery := queryMobileRunnerSemaphoreQueuedRuns
	t.Cleanup(func() {
		listMobileRunnerSemaphoreWorkflows = originalList
		queryMobileRunnerSemaphoreQueuedRuns = originalQuery
	})

	orgNamespace := "org-1"
	enqueuedAt := time.Date(2026, 2, 5, 9, 0, 0, 0, time.UTC)

	listMobileRunnerSemaphoreWorkflows = func(_ context.Context) ([]string, error) {
		return []string{"runner-1", "runner-2"}, nil
	}

	queryMobileRunnerSemaphoreQueuedRuns = func(
		_ context.Context,
		runnerID string,
		ownerNamespace string,
	) ([]workflows.MobileRunnerSemaphoreQueuedRunView, error) {
		require.Equal(t, orgNamespace, ownerNamespace)

		switch runnerID {
		case "runner-1":
			return []workflows.MobileRunnerSemaphoreQueuedRunView{
				{
					TicketID:           "ticket-1",
					OwnerNamespace:     orgNamespace,
					PipelineIdentifier: "org-1/pipeline-a",
					EnqueuedAt:         enqueuedAt,
					LeaderRunnerID:     "runner-1",
					RequiredRunnerIDs:  []string{"runner-1", "runner-2"},
					Status:             workflowengine.MobileRunnerSemaphoreRunQueued,
					Position:           0,
					LineLen:            2,
				},
				{
					TicketID:           "ticket-2",
					OwnerNamespace:     orgNamespace,
					PipelineIdentifier: "org-1/pipeline-b",
					EnqueuedAt:         enqueuedAt,
					LeaderRunnerID:     "runner-1",
					RequiredRunnerIDs:  []string{"runner-1"},
					Status:             workflowengine.MobileRunnerSemaphoreRunRunning,
					Position:           0,
					LineLen:            1,
				},
			}, nil
		case "runner-2":
			return []workflows.MobileRunnerSemaphoreQueuedRunView{
				{
					TicketID:           "ticket-1",
					OwnerNamespace:     orgNamespace,
					PipelineIdentifier: "org-1/pipeline-a",
					EnqueuedAt:         enqueuedAt,
					LeaderRunnerID:     "runner-1",
					RequiredRunnerIDs:  []string{"runner-1", "runner-2"},
					Status:             workflowengine.MobileRunnerSemaphoreRunQueued,
					Position:           1,
					LineLen:            3,
				},
			}, nil
		default:
			return nil, nil
		}
	}

	queued, err := listQueuedPipelineRuns(context.Background(), orgNamespace)
	require.NoError(t, err)
	require.Len(t, queued, 1)

	agg, ok := queued["ticket-1"]
	require.True(t, ok)
	require.Equal(t, "org-1/pipeline-a", agg.PipelineIdentifier)
	require.Equal(t, enqueuedAt, agg.EnqueuedAt)
	require.Equal(t, "runner-1", agg.LeaderRunnerID)
	require.Equal(t, []string{"runner-1", "runner-2"}, agg.RequiredRunnerIDs)
	require.Equal(t, []string{"runner-1", "runner-2"}, agg.RunnerIDs)
	require.Equal(t, workflowengine.MobileRunnerSemaphoreRunQueued, agg.Status)
	require.Equal(t, 1, agg.Position)
	require.Equal(t, 3, agg.LineLen)
}
