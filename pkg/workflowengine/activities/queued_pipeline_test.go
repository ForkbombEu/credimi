// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

type fakeWorkflowRun struct {
	id    string
	runID string
}

func (f fakeWorkflowRun) GetID() string {
	return f.id
}

func (f fakeWorkflowRun) GetRunID() string {
	return f.runID
}

func (f fakeWorkflowRun) Get(ctx context.Context, valuePtr interface{}) error {
	return nil
}

func (f fakeWorkflowRun) GetWithOptions(
	ctx context.Context,
	valuePtr interface{},
	options client.WorkflowRunGetOptions,
) error {
	return nil
}

type fakeTemporalClient struct {
	run client.WorkflowRun
}

func (f fakeTemporalClient) ExecuteWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow interface{},
	args ...interface{},
) (client.WorkflowRun, error) {
	return f.run, nil
}

type failingDoer struct {
	err error
}

func (f failingDoer) Do(*http.Request) (*http.Response, error) {
	return nil, f.err
}

type countingDoer struct {
	attempts int
	err      error
}

func (c *countingDoer) Do(*http.Request) (*http.Response, error) {
	c.attempts++
	return nil, c.err
}

func TestStartQueuedPipelineActivityNonFatalResultFailure(t *testing.T) {
	act := NewStartQueuedPipelineActivity()
	act.temporalClientFactory = func(namespace string) (temporalWorkflowStarter, error) {
		return fakeTemporalClient{
			run: fakeWorkflowRun{
				id:    "wf-1",
				runID: "run-1",
			},
		}, nil
	}
	act.httpDoer = failingDoer{err: errors.New("boom")}

	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: StartQueuedPipelineActivityInput{
			TicketID:           "ticket-1",
			OwnerNamespace:     "tenant-1",
			PipelineIdentifier: "tenant-1/pipeline",
			YAML:               "name: test\nsteps: []\n",
			PipelineConfig: map[string]any{
				"app_url": "https://example.com",
			},
		},
	})
	require.NoError(t, err)

	output, ok := result.Output.(StartQueuedPipelineActivityOutput)
	require.True(t, ok)
	require.Equal(t, "wf-1", output.WorkflowID)
	require.Equal(t, "run-1", output.RunID)
	require.Equal(t, "tenant-1", output.WorkflowNamespace)
	require.False(t, output.PipelineResultCreated)
	require.NotEmpty(t, output.PipelineResultError)
	require.NotEmpty(t, result.Log)
}

func TestStartQueuedPipelineActivityRetriesPipelineResult(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	act := NewStartQueuedPipelineActivity()
	act.temporalClientFactory = func(namespace string) (temporalWorkflowStarter, error) {
		return fakeTemporalClient{
			run: fakeWorkflowRun{
				id:    "wf-2",
				runID: "run-2",
			},
		}, nil
	}
	act.httpDoer = server.Client()

	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: StartQueuedPipelineActivityInput{
			TicketID:           "ticket-2",
			OwnerNamespace:     "tenant-2",
			PipelineIdentifier: "tenant-2/pipeline",
			YAML:               "name: test\nsteps: []\n",
			PipelineConfig: map[string]any{
				"app_url": server.URL,
			},
		},
	})
	require.NoError(t, err)

	output, ok := result.Output.(StartQueuedPipelineActivityOutput)
	require.True(t, ok)
	require.True(t, output.PipelineResultCreated)
	require.Empty(t, output.PipelineResultError)
	require.Empty(t, result.Log)
	require.Equal(t, 3, attempts)
}

func TestCreatePipelineExecutionResultWithRetryAttempts(t *testing.T) {
	doer := &countingDoer{err: errors.New("boom")}

	err := createPipelineExecutionResultWithRetry(
		context.Background(),
		doer,
		"https://example.com",
		"tenant-1",
		"pipeline-1",
		"wf-1",
		"run-1",
	)

	require.Error(t, err)
	require.Equal(t, 4, doer.attempts)
}
