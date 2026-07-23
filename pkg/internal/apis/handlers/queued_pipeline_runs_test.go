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
	originalList := listMobileDeviceSemaphoreWorkflows
	originalQuery := queryMobileDeviceSemaphoreQueuedRuns
	t.Cleanup(func() {
		listMobileDeviceSemaphoreWorkflows = originalList
		queryMobileDeviceSemaphoreQueuedRuns = originalQuery
	})

	orgNamespace := "org-1"
	enqueuedAt := time.Date(2026, 2, 5, 9, 0, 0, 0, time.UTC)

	listMobileDeviceSemaphoreWorkflows = func(_ context.Context) ([]string, error) {
		return []string{"runner-1", "runner-2"}, nil
	}

	queryMobileDeviceSemaphoreQueuedRuns = func(
		_ context.Context,
		runnerID string,
		ownerNamespace string,
	) ([]workflows.MobileDeviceSemaphoreQueuedRunView, error) {
		require.Equal(t, orgNamespace, ownerNamespace)

		switch runnerID {
		case "runner-1":
			return []workflows.MobileDeviceSemaphoreQueuedRunView{
				{
					TicketID:           "ticket-1",
					OwnerNamespace:     orgNamespace,
					PipelineIdentifier: "org-1/pipeline-a",
					EnqueuedAt:         enqueuedAt,
					LeaderDeviceID:     "runner-1",
					RequiredDeviceIDs:  []string{"runner-1", "runner-2"},
					Status:             workflowengine.MobileDeviceSemaphoreRunQueued,
					Position:           0,
					LineLen:            2,
				},
				{
					TicketID:           "ticket-2",
					OwnerNamespace:     orgNamespace,
					PipelineIdentifier: "org-1/pipeline-b",
					EnqueuedAt:         enqueuedAt,
					LeaderDeviceID:     "runner-1",
					RequiredDeviceIDs:  []string{"runner-1"},
					Status:             workflowengine.MobileDeviceSemaphoreRunRunning,
					Position:           0,
					LineLen:            1,
				},
			}, nil
		case "runner-2":
			return []workflows.MobileDeviceSemaphoreQueuedRunView{
				{
					TicketID:           "ticket-1",
					OwnerNamespace:     orgNamespace,
					PipelineIdentifier: "org-1/pipeline-a",
					EnqueuedAt:         enqueuedAt,
					LeaderDeviceID:     "runner-1",
					RequiredDeviceIDs:  []string{"runner-1", "runner-2"},
					Status:             workflowengine.MobileDeviceSemaphoreRunQueued,
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
	require.Equal(t, "runner-1", agg.LeaderDeviceID)
	require.Equal(t, []string{"runner-1", "runner-2"}, agg.RequiredDeviceIDs)
	require.Equal(t, []string{"runner-1", "runner-2"}, agg.DeviceIDs)
	require.Equal(t, workflowengine.MobileDeviceSemaphoreRunQueued, agg.Status)
	require.Equal(t, 1, agg.Position)
	require.Equal(t, 3, agg.LineLen)
}

type queuedRunsEncodedValue struct {
	value []workflows.MobileDeviceSemaphoreQueuedRunView
}

func (q queuedRunsEncodedValue) HasValue() bool { return true }

func (q queuedRunsEncodedValue) Get(valuePtr interface{}) error {
	ptr, ok := valuePtr.(*[]workflows.MobileDeviceSemaphoreQueuedRunView)
	if !ok {
		return errors.New("unexpected type")
	}
	*ptr = q.value
	return nil
}

type errorEncodedValue struct{}

func (e errorEncodedValue) HasValue() bool { return true }

func (e errorEncodedValue) Get(interface{}) error { return errors.New("decode failed") }

func TestListMobileDeviceSemaphoreWorkflowsTemporal(t *testing.T) {
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
						WorkflowId: workflows.MobileDeviceSemaphoreWorkflowName + "/runner-1",
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
						WorkflowId: workflows.MobileDeviceSemaphoreWorkflowName + "/runner-2",
					},
				},
				{},
			},
		}, nil).
		Once()

	queuedRunsTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	runnerIDs, err := listMobileDeviceSemaphoreWorkflowsTemporal(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"runner-1", "runner-2"}, runnerIDs)
}

func TestQueryMobileDeviceSemaphoreQueuedRunsTemporal(t *testing.T) {
	t.Run("not found returns nil", func(t *testing.T) {
		origClient := queuedRunsTemporalClient
		t.Cleanup(func() { queuedRunsTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		mockClient.
			On(
				"QueryWorkflow",
				mock.Anything,
				workflows.MobileDeviceSemaphoreWorkflowID("runner-1"),
				"",
				workflows.MobileDeviceSemaphoreListQueuedRunsQuery,
				"org-1",
			).
			Return(converter.EncodedValue(nil), &serviceerror.NotFound{Message: "missing"})

		queuedRunsTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		views, err := queryMobileDeviceSemaphoreQueuedRunsTemporal(
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
				workflows.MobileDeviceSemaphoreWorkflowID("runner-2"),
				"",
				workflows.MobileDeviceSemaphoreListQueuedRunsQuery,
				"org-1",
			).
			Return(converter.EncodedValue(errorEncodedValue{}), nil)

		queuedRunsTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		_, err := queryMobileDeviceSemaphoreQueuedRunsTemporal(
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
		expected := []workflows.MobileDeviceSemaphoreQueuedRunView{
			{TicketID: "ticket-1", OwnerNamespace: "org-1"},
		}
		mockClient.
			On(
				"QueryWorkflow",
				mock.Anything,
				workflows.MobileDeviceSemaphoreWorkflowID("runner-3"),
				"",
				workflows.MobileDeviceSemaphoreListQueuedRunsQuery,
				"org-1",
			).
			Return(converter.EncodedValue(queuedRunsEncodedValue{value: expected}), nil)

		queuedRunsTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		views, err := queryMobileDeviceSemaphoreQueuedRunsTemporal(
			context.Background(),
			"runner-3",
			"org-1",
		)
		require.NoError(t, err)
		require.Len(t, views, 1)
		require.Equal(t, "ticket-1", views[0].TicketID)
	})
}
