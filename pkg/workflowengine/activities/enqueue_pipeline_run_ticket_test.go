// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
)

// TestEnqueuePipelineRunTicketActivityCallsTemporalUpdates asserts each runner gets an
// update-with-start enqueue and the runner statuses are aggregated.
func TestEnqueuePipelineRunTicketActivityCallsTemporalUpdates(t *testing.T) {
	responses := map[string]mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse{
		"runner-b": {
			TicketID: "ticket-1",
			Status:   mobiledevicesemaphore.MobileDeviceSemaphoreRunQueued,
			Position: 2,
			LineLen:  3,
		},
		"runner-a": {
			TicketID: "ticket-1",
			Status:   mobiledevicesemaphore.MobileDeviceSemaphoreRunRunning,
			Position: 1,
			LineLen:  2,
		},
	}

	mockClient := temporalmocks.NewClient(t)
	var startIDs []string
	mockClient.
		On(
			"NewWithStartWorkflowOperation",
			mock.Anything,
			mobiledevicesemaphore.WorkflowName,
			mock.Anything,
		).
		Run(func(args mock.Arguments) {
			opts := args.Get(0).(client.StartWorkflowOptions)
			require.Equal(t, mobiledevicesemaphore.TaskQueue, opts.TaskQueue)
			startIDs = append(startIDs, opts.ID)
		}).
		Return(nil).
		Times(2)

	var updates []client.UpdateWorkflowOptions
	for _, deviceID := range []string{"runner-b", "runner-a"} {
		resp := responses[deviceID]
		handle := temporalmocks.NewWorkflowUpdateHandle(t)
		handle.On("Get", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				*args.Get(1).(*mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse) = resp
			}).
			Return(nil).
			Once()
		mockClient.
			On("UpdateWithStartWorkflow", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				opts := args.Get(1).(client.UpdateWithStartWorkflowOptions)
				updates = append(updates, opts.UpdateOptions)
			}).
			Return(handle, nil).
			Once()
	}

	act := NewEnqueuePipelineRunTicketActivity()
	act.temporalClientFactory = func(namespace string) (client.Client, error) {
		require.Equal(t, workflowengine.MobileDeviceSemaphoreDefaultNamespace, namespace)
		return mockClient, nil
	}

	payload := EnqueuePipelineRunTicketActivityInput{
		TicketID:           "ticket-1",
		OwnerNamespace:     "tenant-1",
		EnqueuedAt:         time.Now().UTC(),
		DeviceIDs:          []string{"runner-b", "runner-a"},
		PipelineIdentifier: "tenant-1/pipeline",
		YAML:               "name: test\nsteps: []\n",
		PipelineConfig: map[string]any{
			"app_url": "https://example.test",
		},
		Memo: map[string]any{
			"test": "pipeline-run",
		},
		MaxPipelinesInQueue: 4,
	}

	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{Payload: payload})
	require.NoError(t, err)

	output, ok := result.Output.(EnqueuePipelineRunTicketActivityOutput)
	require.True(t, ok)
	require.Equal(t, mobiledevicesemaphore.MobileDeviceSemaphoreRunRunning, output.Status)
	require.Equal(t, 2, output.Position)
	require.Equal(t, 3, output.LineLen)
	require.Len(t, output.Runners, 2)

	require.Equal(t, []string{
		mobiledevicesemaphore.WorkflowID("runner-b"),
		mobiledevicesemaphore.WorkflowID("runner-a"),
	}, startIDs)

	require.Len(t, updates, 2)
	require.Equal(t, mobiledevicesemaphore.EnqueueRunUpdate, updates[0].UpdateName)
	require.Equal(t, "enqueue/runner-b/ticket-1", updates[0].UpdateID)
	require.Len(t, updates[0].Args, 1)
	req, ok := updates[0].Args[0].(mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest)
	require.True(t, ok)
	require.Equal(t, "runner-b", req.DeviceID)
	require.ElementsMatch(t, []string{"runner-b", "runner-a"}, req.RequiredDeviceIDs)
	require.Equal(t, "runner-b", req.LeaderDeviceID)
	require.Equal(t, 4, req.MaxPipelinesInQueue)
}

// TestEnqueuePipelineRunTicketActivityQueueLimitError keeps the queue-limit error type stable.
func TestEnqueuePipelineRunTicketActivityQueueLimitError(t *testing.T) {
	queueErr := temporal.NewApplicationError(
		"queue limit exceeded",
		mobiledevicesemaphore.ErrQueueLimitExceeded,
	)

	mockClient := temporalmocks.NewClient(t)
	mockClient.On("NewWithStartWorkflowOperation", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	mockClient.On("UpdateWithStartWorkflow", mock.Anything, mock.Anything).
		Return(nil, queueErr)
	mockClient.On("UpdateWorkflow", mock.Anything, mock.Anything).
		Return(nil, errors.New("rollback failed"))

	act := NewEnqueuePipelineRunTicketActivity()
	act.temporalClientFactory = func(namespace string) (client.Client, error) {
		return mockClient, nil
	}

	payload := EnqueuePipelineRunTicketActivityInput{
		TicketID:           "ticket-2",
		OwnerNamespace:     "tenant-2",
		EnqueuedAt:         time.Now().UTC(),
		DeviceIDs:          []string{"runner-1"},
		PipelineIdentifier: "tenant-2/pipeline",
		YAML:               "name: test\nsteps: []\n",
	}

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{Payload: payload})
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, mobiledevicesemaphore.ErrQueueLimitExceeded, appErr.Type())
}

func TestEnqueuePipelineRunTicketActivityValidationErrors(t *testing.T) {
	act := NewEnqueuePipelineRunTicketActivity()

	tests := []struct {
		name        string
		payload     EnqueuePipelineRunTicketActivityInput
		errContains string
	}{
		{
			name: "missing ticket id",
			payload: EnqueuePipelineRunTicketActivityInput{
				OwnerNamespace:     "tenant",
				PipelineIdentifier: "tenant/pipeline",
				YAML:               "name: test\nsteps: []\n",
				DeviceIDs:          []string{"runner-1"},
			},
			errContains: "ticket_id",
		},
		{
			name: "missing owner namespace",
			payload: EnqueuePipelineRunTicketActivityInput{
				TicketID:           "ticket-1",
				PipelineIdentifier: "tenant/pipeline",
				YAML:               "name: test\nsteps: []\n",
				DeviceIDs:          []string{"runner-1"},
			},
			errContains: "owner_namespace",
		},
		{
			name: "missing pipeline identifier",
			payload: EnqueuePipelineRunTicketActivityInput{
				TicketID:       "ticket-1",
				OwnerNamespace: "tenant",
				YAML:           "name: test\nsteps: []\n",
				DeviceIDs:      []string{"runner-1"},
			},
			errContains: "pipeline_identifier",
		},
		{
			name: "missing yaml",
			payload: EnqueuePipelineRunTicketActivityInput{
				TicketID:           "ticket-1",
				OwnerNamespace:     "tenant",
				PipelineIdentifier: "tenant/pipeline",
				DeviceIDs:          []string{"runner-1"},
			},
			errContains: "yaml is required",
		},
		{
			name: "missing runner ids",
			payload: EnqueuePipelineRunTicketActivityInput{
				TicketID:           "ticket-1",
				OwnerNamespace:     "tenant",
				PipelineIdentifier: "tenant/pipeline",
				YAML:               "name: test\nsteps: []\n",
			},
			errContains: "device_ids",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := act.Execute(
				context.Background(),
				workflowengine.ActivityInput{Payload: tc.payload},
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errContains)
		})
	}
}

func TestEnqueuePipelineRunTicketActivityTemporalClientError(t *testing.T) {
	act := NewEnqueuePipelineRunTicketActivity()
	act.temporalClientFactory = func(string) (client.Client, error) {
		return nil, errors.New("no client")
	}

	payload := EnqueuePipelineRunTicketActivityInput{
		TicketID:           "ticket-1",
		OwnerNamespace:     "tenant",
		PipelineIdentifier: "tenant/pipeline",
		YAML:               "name: test\nsteps: []\n",
		DeviceIDs:          []string{"runner-1"},
	}

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{Payload: payload})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no client")
}
