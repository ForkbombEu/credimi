// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
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

type queuedRunsEncodedValue struct {
	value []workflows.MobileRunnerSemaphoreQueuedRunView
}

func (q queuedRunsEncodedValue) HasValue() bool { return true }

func (q queuedRunsEncodedValue) Get(valuePtr interface{}) error {
	ptr, ok := valuePtr.(*[]workflows.MobileRunnerSemaphoreQueuedRunView)
	if !ok {
		return errors.New("unexpected type")
	}
	*ptr = q.value
	return nil
}

type errorEncodedValue struct{}

func (e errorEncodedValue) HasValue() bool { return true }

func (e errorEncodedValue) Get(interface{}) error { return errors.New("decode failed") }

func TestListMobileRunnerSemaphoreWorkflowsTemporal(t *testing.T) {
	origClient := queuedRunsTemporalClient
	t.Cleanup(func() { queuedRunsTemporalClient = origClient })

	mockClient := temporalmocks.NewClient(t)
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.AnythingOfType("*workflowservice.ListWorkflowExecutionsRequest"),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				{
					Execution: &commonpb.WorkflowExecution{
						WorkflowId: workflows.MobileRunnerSemaphoreWorkflowName + "/runner-1",
					},
				},
				{Execution: &commonpb.WorkflowExecution{WorkflowId: "unrelated"}},
			},
			NextPageToken: []byte("next"),
		}, nil).
		Once()
	mockClient.
		On(
			"ListWorkflow",
			mock.Anything,
			mock.AnythingOfType("*workflowservice.ListWorkflowExecutionsRequest"),
		).
		Return(&workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				{
					Execution: &commonpb.WorkflowExecution{
						WorkflowId: workflows.MobileRunnerSemaphoreWorkflowName + "/runner-2",
					},
				},
				{},
			},
		}, nil).
		Once()

	queuedRunsTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	runnerIDs, err := listMobileRunnerSemaphoreWorkflowsTemporal(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"runner-1", "runner-2"}, runnerIDs)
}

func TestQueryMobileRunnerSemaphoreQueuedRunsTemporal(t *testing.T) {
	t.Run("not found returns nil", func(t *testing.T) {
		origClient := queuedRunsTemporalClient
		t.Cleanup(func() { queuedRunsTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		mockClient.
			On(
				"QueryWorkflow",
				mock.Anything,
				workflows.MobileRunnerSemaphoreWorkflowID("runner-1"),
				"",
				workflows.MobileRunnerSemaphoreListQueuedRunsQuery,
				"org-1",
			).
			Return(converter.EncodedValue(nil), &serviceerror.NotFound{Message: "missing"})

		queuedRunsTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		views, err := queryMobileRunnerSemaphoreQueuedRunsTemporal(
			context.Background(),
			"runner-1",
			"org-1",
		)
		require.NoError(t, err)
		require.Nil(t, views)
	})

	t.Run("decode error bubbles", func(t *testing.T) {
		origClient := queuedRunsTemporalClient
		t.Cleanup(func() { queuedRunsTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		mockClient.
			On(
				"QueryWorkflow",
				mock.Anything,
				workflows.MobileRunnerSemaphoreWorkflowID("runner-2"),
				"",
				workflows.MobileRunnerSemaphoreListQueuedRunsQuery,
				"org-1",
			).
			Return(converter.EncodedValue(errorEncodedValue{}), nil)

		queuedRunsTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		_, err := queryMobileRunnerSemaphoreQueuedRunsTemporal(
			context.Background(),
			"runner-2",
			"org-1",
		)
		require.ErrorContains(t, err, "decode failed")
	})

	t.Run("returns queued runs", func(t *testing.T) {
		origClient := queuedRunsTemporalClient
		t.Cleanup(func() { queuedRunsTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		expected := []workflows.MobileRunnerSemaphoreQueuedRunView{
			{TicketID: "ticket-1", OwnerNamespace: "org-1"},
		}
		mockClient.
			On(
				"QueryWorkflow",
				mock.Anything,
				workflows.MobileRunnerSemaphoreWorkflowID("runner-3"),
				"",
				workflows.MobileRunnerSemaphoreListQueuedRunsQuery,
				"org-1",
			).
			Return(converter.EncodedValue(queuedRunsEncodedValue{value: expected}), nil)

		queuedRunsTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		views, err := queryMobileRunnerSemaphoreQueuedRunsTemporal(
			context.Background(),
			"runner-3",
			"org-1",
		)
		require.NoError(t, err)
		require.Len(t, views, 1)
		require.Equal(t, "ticket-1", views[0].TicketID)
	})
}
