// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
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

type capturingTemporalClient struct {
	run         client.WorkflowRun
	lastOptions client.StartWorkflowOptions
}

// ExecuteWorkflow records workflow start options for assertions and returns the stubbed run.
func (c *capturingTemporalClient) ExecuteWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow interface{},
	args ...interface{},
) (client.WorkflowRun, error) {
	c.lastOptions = options
	return c.run, nil
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

// TestStartQueuedPipelineActivityWorkflowIDPrefix verifies scheduled tickets get a distinct ID prefix.
func TestStartQueuedPipelineActivityWorkflowIDPrefix(t *testing.T) {
	tests := []struct {
		name          string
		ticketID      string
		wantPrefix    string
		blockedPrefix string
	}{
		{
			name:       "scheduled ticket",
			ticketID:   "sched/wf/run",
			wantPrefix: "Pipeline-Sched-",
		},
		{
			name:          "non scheduled ticket",
			ticketID:      "ticket-1",
			wantPrefix:    "Pipeline-",
			blockedPrefix: "Pipeline-Sched-",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			captured := &capturingTemporalClient{
				run: fakeWorkflowRun{
					id:    "wf-3",
					runID: "run-3",
				},
			}
			act := NewStartQueuedPipelineActivity()
			act.temporalClientFactory = func(namespace string) (temporalWorkflowStarter, error) {
				return captured, nil
			}
			act.httpDoer = failingDoer{err: errors.New("boom")}

			_, err := act.Execute(context.Background(), workflowengine.ActivityInput{
				Payload: StartQueuedPipelineActivityInput{
					TicketID:           test.ticketID,
					OwnerNamespace:     "tenant-1",
					PipelineIdentifier: "tenant-1/pipeline",
					YAML:               "name: test\nsteps: []\n",
					PipelineConfig: map[string]any{
						"app_url": "https://example.com",
					},
				},
			})
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(captured.lastOptions.ID, test.wantPrefix))
			if test.blockedPrefix != "" {
				require.False(t, strings.HasPrefix(captured.lastOptions.ID, test.blockedPrefix))
			}
		})
	}
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

func TestStartQueuedPipelineActivityValidationErrors(t *testing.T) {
	act := NewStartQueuedPipelineActivity()

	tests := []struct {
		name        string
		payload     StartQueuedPipelineActivityInput
		errContains string
	}{
		{
			name: "missing owner namespace",
			payload: StartQueuedPipelineActivityInput{
				PipelineIdentifier: "p",
				YAML:               "name: test\n",
			},
			errContains: "owner_namespace",
		},
		{
			name: "missing pipeline identifier",
			payload: StartQueuedPipelineActivityInput{
				OwnerNamespace: "ns",
				YAML:           "name: test\n",
			},
			errContains: "pipeline_identifier",
		},
		{
			name: "missing yaml",
			payload: StartQueuedPipelineActivityInput{
				OwnerNamespace:     "ns",
				PipelineIdentifier: "p",
			},
			errContains: "yaml is required",
		},
		{
			name: "missing app_url",
			payload: StartQueuedPipelineActivityInput{
				OwnerNamespace:     "ns",
				PipelineIdentifier: "p",
				YAML:               "name: test\nsteps: []\n",
			},
			errContains: "app_url",
		},
		{
			name: "invalid yaml",
			payload: StartQueuedPipelineActivityInput{
				OwnerNamespace:     "ns",
				PipelineIdentifier: "p",
				YAML:               "name: [",
				PipelineConfig: map[string]any{
					"app_url": "https://example.com",
				},
			},
			errContains: "parse workflow definition",
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
			require.True(t, temporal.IsApplicationError(err))
		})
	}
}

func TestStartQueuedPipelineActivityName(t *testing.T) {
	act := NewStartQueuedPipelineActivity()
	require.Equal(t, "Start queued pipeline", act.Name())
}

func TestCopyStringSlice(t *testing.T) {
	require.Equal(t, []string{}, copyStringSlice(nil))

	original := []string{"a", "b"}
	copied := copyStringSlice(original)
	require.Equal(t, original, copied)
	original[0] = "changed"
	require.Equal(t, []string{"a", "b"}, copied)
}

func TestParseDurationOrDefaultInvalid(t *testing.T) {
	dur := parseDurationOrDefault("not-a-duration", defaultActivityStartTimeout)
	require.Equal(t, 5*time.Minute, dur)
}

func TestPrepareQueuedWorkflowOptionsOverrides(t *testing.T) {
	rc := queuedRuntime{}
	rc.Temporal.ExecutionTimeout = "2h"
	rc.Temporal.ActivityOptions.ScheduleToCloseTimeout = "15m"
	rc.Temporal.ActivityOptions.StartToCloseTimeout = "7m"
	rc.Temporal.ActivityOptions.RetryPolicy.MaximumAttempts = 3
	rc.Temporal.ActivityOptions.RetryPolicy.InitialInterval = "3s"
	rc.Temporal.ActivityOptions.RetryPolicy.MaximumInterval = "9s"
	rc.Temporal.ActivityOptions.RetryPolicy.BackoffCoefficient = 1.5

	opts := prepareQueuedWorkflowOptions(rc)
	require.Equal(t, 2*time.Hour, opts.Options.WorkflowExecutionTimeout)
	require.Equal(t, 15*time.Minute, opts.ActivityOptions.ScheduleToCloseTimeout)
	require.Equal(t, 7*time.Minute, opts.ActivityOptions.StartToCloseTimeout)
	require.NotNil(t, opts.ActivityOptions.RetryPolicy)
	require.Equal(t, int32(3), opts.ActivityOptions.RetryPolicy.MaximumAttempts)
	require.Equal(t, 3*time.Second, opts.ActivityOptions.RetryPolicy.InitialInterval)
	require.Equal(t, 9*time.Second, opts.ActivityOptions.RetryPolicy.MaximumInterval)
	require.Equal(t, 1.5, opts.ActivityOptions.RetryPolicy.BackoffCoefficient)
}

func TestApplySemaphoreTicketMetadata(t *testing.T) {
	payload := StartQueuedPipelineActivityInput{
		TicketID:          "ticket-1",
		RequiredRunnerIDs: []string{"runner-1"},
		LeaderRunnerID:    "runner-1",
		OwnerNamespace:    "ns-1",
	}

	applySemaphoreTicketMetadata(nil, payload)

	config := map[string]any{}
	applySemaphoreTicketMetadata(config, payload)
	require.Equal(t, "ticket-1", config[mobileRunnerSemaphoreTicketIDConfigKey])
	require.Equal(t, []string{"runner-1"}, config[mobileRunnerSemaphoreRunnerIDsConfigKey])
	require.Equal(t, "runner-1", config[mobileRunnerSemaphoreLeaderRunnerIDConfigKey])
	require.Equal(t, "ns-1", config[mobileRunnerSemaphoreOwnerNamespaceConfigKey])
}

func TestParseQueuedWorkflowDefinitionError(t *testing.T) {
	_, _, err := parseQueuedWorkflowDefinition("name: [")
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse workflow definition")
}
