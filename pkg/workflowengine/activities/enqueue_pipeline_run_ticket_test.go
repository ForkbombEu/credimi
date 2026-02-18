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
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

// enqueueFakeWorkflowRun is a lightweight workflow run stub for enqueue tests.
type enqueueFakeWorkflowRun struct {
	id    string
	runID string
}

// GetID returns the workflow ID for the fake run.
func (f enqueueFakeWorkflowRun) GetID() string {
	return f.id
}

// GetRunID returns the workflow run ID for the fake run.
func (f enqueueFakeWorkflowRun) GetRunID() string {
	return f.runID
}

// Get is a no-op workflow run getter for tests.
func (f enqueueFakeWorkflowRun) Get(ctx context.Context, valuePtr interface{}) error {
	return nil
}

// GetWithOptions is a no-op workflow run getter for tests.
func (f enqueueFakeWorkflowRun) GetWithOptions(
	ctx context.Context,
	valuePtr interface{},
	options client.WorkflowRunGetOptions,
) error {
	return nil
}

// fakeUpdateHandle returns a predefined enqueue response.
type fakeUpdateHandle struct {
	response   mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunResponse
	err        error
	workflowID string
	runID      string
	updateID   string
}

// Get writes the predefined response into the provided pointer.
func (f fakeUpdateHandle) Get(ctx context.Context, valuePtr interface{}) error {
	if f.err != nil {
		return f.err
	}
	respPtr, ok := valuePtr.(*mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunResponse)
	if !ok {
		return errors.New("unexpected response type")
	}
	*respPtr = f.response
	return nil
}

// WorkflowID returns the workflow ID for the update handle.
func (f fakeUpdateHandle) WorkflowID() string {
	return f.workflowID
}

// RunID returns the run ID for the update handle.
func (f fakeUpdateHandle) RunID() string {
	return f.runID
}

// UpdateID returns the update ID for the update handle.
func (f fakeUpdateHandle) UpdateID() string {
	return f.updateID
}

// executeCall captures arguments sent to ExecuteWorkflow.
type executeCall struct {
	options  client.StartWorkflowOptions
	workflow interface{}
}

// fakeEnqueueClient captures Temporal client calls for assertions.
type fakeEnqueueClient struct {
	executeCalls    []executeCall
	updateCalls     []client.UpdateWorkflowOptions
	updateResponses map[string]mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunResponse
	updateErr       error
}

// ExecuteWorkflow records workflow start calls for assertions.
func (f *fakeEnqueueClient) ExecuteWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow interface{},
	args ...interface{},
) (client.WorkflowRun, error) {
	f.executeCalls = append(f.executeCalls, executeCall{options: options, workflow: workflow})
	return enqueueFakeWorkflowRun{id: options.ID, runID: options.ID + "-run"}, nil
}

// UpdateWorkflow records update calls and returns preset responses.
func (f *fakeEnqueueClient) UpdateWorkflow(
	ctx context.Context,
	options client.UpdateWorkflowOptions,
) (client.WorkflowUpdateHandle, error) {
	f.updateCalls = append(f.updateCalls, options)
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	resp := f.updateResponses[options.WorkflowID]
	return fakeUpdateHandle{
		response:   resp,
		workflowID: options.WorkflowID,
		updateID:   options.UpdateID,
	}, nil
}

// TestEnqueuePipelineRunTicketActivityCallsTemporalUpdates asserts workflow IDs and update names.
func TestEnqueuePipelineRunTicketActivityCallsTemporalUpdates(t *testing.T) {
	act := NewEnqueuePipelineRunTicketActivity()
	fakeClient := &fakeEnqueueClient{
		updateResponses: map[string]mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunResponse{
			mobilerunnersemaphore.WorkflowID("runner-b"): {
				TicketID: "ticket-1",
				Status:   mobilerunnersemaphore.MobileRunnerSemaphoreRunQueued,
				Position: 2,
				LineLen:  3,
			},
			mobilerunnersemaphore.WorkflowID("runner-a"): {
				TicketID: "ticket-1",
				Status:   mobilerunnersemaphore.MobileRunnerSemaphoreRunRunning,
				Position: 1,
				LineLen:  2,
			},
		},
	}

	act.temporalClientFactory = func(namespace string) (temporalWorkflowUpdater, error) {
		require.Equal(t, workflowengine.MobileRunnerSemaphoreDefaultNamespace, namespace)
		return fakeClient, nil
	}

	payload := EnqueuePipelineRunTicketActivityInput{
		TicketID:           "ticket-1",
		OwnerNamespace:     "tenant-1",
		EnqueuedAt:         time.Now().UTC(),
		RunnerIDs:          []string{"runner-b", "runner-a"},
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
	require.Equal(t, mobilerunnersemaphore.MobileRunnerSemaphoreRunRunning, output.Status)
	require.Equal(t, 2, output.Position)
	require.Equal(t, 3, output.LineLen)
	require.Len(t, output.Runners, 2)

	require.Len(t, fakeClient.executeCalls, 2)
	require.Equal(t, mobilerunnersemaphore.WorkflowName, fakeClient.executeCalls[0].workflow)
	require.Equal(
		t,
		mobilerunnersemaphore.WorkflowID("runner-b"),
		fakeClient.executeCalls[0].options.ID,
	)
	require.Equal(t, mobilerunnersemaphore.TaskQueue, fakeClient.executeCalls[0].options.TaskQueue)

	require.Len(t, fakeClient.updateCalls, 2)
	require.Equal(
		t,
		mobilerunnersemaphore.WorkflowID("runner-b"),
		fakeClient.updateCalls[0].WorkflowID,
	)
	require.Equal(t, mobilerunnersemaphore.EnqueueRunUpdate, fakeClient.updateCalls[0].UpdateName)
	require.Equal(
		t,
		"enqueue/runner-b/ticket-1",
		fakeClient.updateCalls[0].UpdateID,
	)
	require.Len(t, fakeClient.updateCalls[0].Args, 1)
	req, ok := fakeClient.updateCalls[0].Args[0].(mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunRequest)
	require.True(t, ok)
	require.Equal(t, "runner-b", req.RunnerID)
	require.Equal(t, []string{"runner-b", "runner-a"}, req.RequiredRunnerIDs)
	require.Equal(t, "runner-b", req.LeaderRunnerID)
}

// TestEnqueuePipelineRunTicketActivityQueueLimitError keeps the queue-limit error type stable.
func TestEnqueuePipelineRunTicketActivityQueueLimitError(t *testing.T) {
	act := NewEnqueuePipelineRunTicketActivity()
	queueErr := temporal.NewApplicationError(
		"queue limit exceeded",
		mobilerunnersemaphore.ErrQueueLimitExceeded,
	)

	fakeClient := &fakeEnqueueClient{updateErr: queueErr}
	act.temporalClientFactory = func(namespace string) (temporalWorkflowUpdater, error) {
		return fakeClient, nil
	}

	payload := EnqueuePipelineRunTicketActivityInput{
		TicketID:           "ticket-2",
		OwnerNamespace:     "tenant-2",
		EnqueuedAt:         time.Now().UTC(),
		RunnerIDs:          []string{"runner-1"},
		PipelineIdentifier: "tenant-2/pipeline",
		YAML:               "name: test\nsteps: []\n",
	}

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{Payload: payload})
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, mobilerunnersemaphore.ErrQueueLimitExceeded, appErr.Type())
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
				RunnerIDs:          []string{"runner-1"},
			},
			errContains: "ticket_id",
		},
		{
			name: "missing owner namespace",
			payload: EnqueuePipelineRunTicketActivityInput{
				TicketID:           "ticket-1",
				PipelineIdentifier: "tenant/pipeline",
				YAML:               "name: test\nsteps: []\n",
				RunnerIDs:          []string{"runner-1"},
			},
			errContains: "owner_namespace",
		},
		{
			name: "missing pipeline identifier",
			payload: EnqueuePipelineRunTicketActivityInput{
				TicketID:       "ticket-1",
				OwnerNamespace: "tenant",
				YAML:           "name: test\nsteps: []\n",
				RunnerIDs:      []string{"runner-1"},
			},
			errContains: "pipeline_identifier",
		},
		{
			name: "missing yaml",
			payload: EnqueuePipelineRunTicketActivityInput{
				TicketID:           "ticket-1",
				OwnerNamespace:     "tenant",
				PipelineIdentifier: "tenant/pipeline",
				RunnerIDs:          []string{"runner-1"},
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
			errContains: "runner_ids",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := act.Execute(context.Background(), workflowengine.ActivityInput{Payload: tc.payload})
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errContains)
		})
	}
}

func TestEnqueuePipelineRunTicketActivityTemporalClientError(t *testing.T) {
	act := NewEnqueuePipelineRunTicketActivity()
	act.temporalClientFactory = func(string) (temporalWorkflowUpdater, error) {
		return nil, errors.New("no client")
	}

	payload := EnqueuePipelineRunTicketActivityInput{
		TicketID:           "ticket-1",
		OwnerNamespace:     "tenant",
		PipelineIdentifier: "tenant/pipeline",
		YAML:               "name: test\nsteps: []\n",
		RunnerIDs:          []string{"runner-1"},
	}

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{Payload: payload})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no client")
}
