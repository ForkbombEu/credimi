// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"strings"
	"testing"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/dbx"
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

func (f fakeWorkflowRun) GetFirstExecutionRunID() string {
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

type capturingTemporalClient struct {
	run         client.WorkflowRun
	lastOptions client.StartWorkflowOptions
	lastArgs    []interface{}
}

// ExecuteWorkflow records workflow start options for assertions and returns the stubbed run.
func (c *capturingTemporalClient) ExecuteWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow interface{},
	args ...interface{},
) (client.WorkflowRun, error) {
	c.lastOptions = options
	c.lastArgs = append([]interface{}(nil), args...)
	return c.run, nil
}

func newCapturingQueuedPipelineActivity(
	t *testing.T,
	workflowID string,
	runID string,
) (*StartQueuedPipelineActivity, *capturingTemporalClient) {
	t.Helper()
	captured := &capturingTemporalClient{
		run: fakeWorkflowRun{id: workflowID, runID: runID},
	}
	act := NewStartQueuedPipelineActivity(newPipelineResultsTestApp(t))
	act.temporalClientFactory = func(namespace string) (temporalWorkflowStarter, error) {
		return captured, nil
	}
	return act, captured
}

func capturedWorkflowConfig(t *testing.T, captured *capturingTemporalClient) map[string]any {
	t.Helper()
	require.Len(t, captured.lastArgs, 1)
	workflowInput, ok := captured.lastArgs[0].(map[string]any)
	require.True(t, ok)
	rawInput, ok := workflowInput["workflow_input"].(workflowengine.WorkflowInput)
	require.True(t, ok)
	return rawInput.Config
}

func TestStartQueuedPipelineActivityNonFatalResultFailure(t *testing.T) {
	tests := []struct {
		name           string
		ownerNamespace string
		pipelineID     string
		errContains    string
	}{
		{
			name:           "unknown pipeline",
			ownerNamespace: testOrgANamespace,
			pipelineID:     testOrgANamespace + "/unknown-pipeline",
			errContains:    "resolve pipeline",
		},
		{
			name:           "unknown owner",
			ownerNamespace: "missing-org",
			pipelineID:     "missing-org/pipeline",
			errContains:    "lookup owner organization: organization missing-org not found",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			act, _ := newCapturingQueuedPipelineActivity(t, "wf-1", "run-1")

			result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
				Payload: StartQueuedPipelineActivityInput{
					TicketID:           "ticket-1",
					OwnerNamespace:     tc.ownerNamespace,
					PipelineIdentifier: tc.pipelineID,
					YAML:               "name: test\nsteps: []\n",
				},
			})
			require.NoError(t, err)

			output, ok := result.Output.(StartQueuedPipelineActivityOutput)
			require.True(t, ok)
			require.Equal(t, "wf-1", output.WorkflowID)
			require.Equal(t, "run-1", output.RunID)
			require.Equal(t, tc.ownerNamespace, output.WorkflowNamespace)
			require.False(t, output.PipelineResultCreated)
			require.Contains(t, output.PipelineResultError, tc.errContains)
			require.Len(t, result.Log, 1)
			require.Contains(t, result.Log[0], "pipeline execution result not created: ")
		})
	}
}

func TestStartQueuedPipelineActivityCreatesPipelineResult(t *testing.T) {
	act, _ := newCapturingQueuedPipelineActivity(t, "wf-ci", "run-ci")
	app := act.app
	pipeline, identifier := createTestPipeline(t, app, "queued-ci")

	payload := StartQueuedPipelineActivityInput{
		TicketID:           "ticket-ci",
		OwnerNamespace:     testOrgANamespace,
		PipelineIdentifier: identifier,
		YAML:               "name: test\nsteps: []\n",
		Memo: map[string]any{
			pipelineinternal.RunTypeMemoKey: pipelineinternal.RunTypeCI,
		},
	}
	for range 2 {
		result, err := act.Execute(
			context.Background(),
			workflowengine.ActivityInput{Payload: payload},
		)
		require.NoError(t, err)
		output, ok := result.Output.(StartQueuedPipelineActivityOutput)
		require.True(t, ok)
		require.True(t, output.PipelineResultCreated)
		require.Empty(t, output.PipelineResultError)
		require.Empty(t, result.Log)
	}

	records, err := app.FindAllRecords(
		"pipeline_results",
		dbx.HashExp{"workflow_id": "wf-ci", "run_id": "run-ci"},
	)
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, pipeline.Id, records[0].GetString("pipeline"))
	require.Equal(t, testOrgAID, records[0].GetString("owner"))
	require.Equal(t, pipelineinternal.RunTypeCI, records[0].GetString("type"))
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
			act, captured := newCapturingQueuedPipelineActivity(t, "wf-3", "run-3")

			_, err := act.Execute(context.Background(), workflowengine.ActivityInput{
				Payload: StartQueuedPipelineActivityInput{
					TicketID:           test.ticketID,
					OwnerNamespace:     "tenant-1",
					PipelineIdentifier: "tenant-1/pipeline",
					YAML:               "name: test\nsteps: []\n",
				},
			})
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(captured.lastOptions.ID, test.wantPrefix))
			if test.blockedPrefix != "" {
				require.False(t, strings.HasPrefix(captured.lastOptions.ID, test.blockedPrefix))
			}
			key := temporal.NewSearchAttributeKeyKeyword(
				workflowengine.PipelineIdentifierSearchAttribute,
			)
			value, ok := captured.lastOptions.TypedSearchAttributes.GetKeyword(key)
			require.True(t, ok)
			require.Equal(t, "tenant-1/pipeline", value)
		})
	}
}

func TestStartQueuedPipelineActivityPropagatesDisableAndroidPlayStore(t *testing.T) {
	act, captured := newCapturingQueuedPipelineActivity(t, "wf-4", "run-4")

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: StartQueuedPipelineActivityInput{
			TicketID:           "ticket-4",
			OwnerNamespace:     "tenant-1",
			PipelineIdentifier: "tenant-1/pipeline",
			YAML: `name: test
runtime:
  disable_android_play_store: true
steps: []
`,
		},
	})
	require.NoError(t, err)
	require.Equal(t, true, capturedWorkflowConfig(t, captured)["disable_android_play_store"])
}

func TestStartQueuedPipelineActivitySkipsReservedYAMLConfig(t *testing.T) {
	act, captured := newCapturingQueuedPipelineActivity(t, "wf-5", "run-5")

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: StartQueuedPipelineActivityInput{
			TicketID:           "ticket-5",
			OwnerNamespace:     "tenant-1",
			PipelineIdentifier: "tenant-1/pipeline",
			YAML: `name: test
config:
  keep: value
  temp_wallet_version:
    record_id: malicious
  temp_credentials:
    - record_id: malicious
  temp_use_case_verifications:
    - record_id: malicious
  github_pr_comment:
    repository: attacker/repo
    pull_request_number: 17
  app_url: https://attacker.example
steps: []
`,
			PipelineConfig: map[string]any{
				"app_url": "https://stale.example",
			},
		},
	})
	require.NoError(t, err)

	config := capturedWorkflowConfig(t, captured)
	require.Equal(t, "value", config["keep"])
	require.NotContains(t, config, queuedTempWalletVersionConfigKey)
	require.NotContains(t, config, queuedTempCredentialsConfigKey)
	require.NotContains(t, config, queuedTempUseCaseVerificationsConfigKey)
	require.NotContains(t, config, queuedGitHubPRCommentConfigKey)
	require.Equal(t, pipelineResultsTestAppURL, config[workflowengine.AppURLConfigKey])
	require.Equal(t, "Credimi", config[workflowengine.AppNameConfigKey])
	require.Contains(t, config, workflowengine.AppLogoConfigKey)
}

func TestStartQueuedPipelineActivityValidationErrors(t *testing.T) {
	act := NewStartQueuedPipelineActivity(newPipelineResultsTestApp(t))

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
			name: "invalid yaml",
			payload: StartQueuedPipelineActivityInput{
				OwnerNamespace:     "ns",
				PipelineIdentifier: "p",
				YAML:               "name: [",
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
	act := NewStartQueuedPipelineActivity(nil)
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
	require.Equal(t, 30*time.Second, opts.ActivityOptions.HeartbeatTimeout)
	require.NotNil(t, opts.ActivityOptions.RetryPolicy)
	require.Equal(t, int32(3), opts.ActivityOptions.RetryPolicy.MaximumAttempts)
	require.Equal(t, 3*time.Second, opts.ActivityOptions.RetryPolicy.InitialInterval)
	require.Equal(t, 9*time.Second, opts.ActivityOptions.RetryPolicy.MaximumInterval)
	require.Equal(t, 1.5, opts.ActivityOptions.RetryPolicy.BackoffCoefficient)
}

func TestPrepareQueuedWorkflowOptionsHeartbeatOverride(t *testing.T) {
	rc := queuedRuntime{}
	rc.Temporal.ActivityOptions.HeartbeatTimeout = "2m"

	opts := prepareQueuedWorkflowOptions(rc)
	require.Equal(t, 2*time.Minute, opts.ActivityOptions.HeartbeatTimeout)
}

func TestApplySemaphoreTicketMetadata(t *testing.T) {
	payload := StartQueuedPipelineActivityInput{
		TicketID:          "ticket-1",
		RequiredDeviceIDs: []string{"runner-1"},
		LeaderDeviceID:    "runner-1",
		OwnerNamespace:    "ns-1",
	}

	applySemaphoreTicketMetadata(nil, payload)

	config := map[string]any{}
	applySemaphoreTicketMetadata(config, payload)
	require.Equal(t, "ticket-1", config[mobileDeviceSemaphoreTicketIDConfigKey])
	require.Equal(t, []string{"runner-1"}, config[mobileDeviceSemaphoreDeviceIDsConfigKey])
	require.Equal(t, "runner-1", config[mobileDeviceSemaphoreLeaderDeviceIDConfigKey])
	require.Equal(t, "ns-1", config[mobileDeviceSemaphoreOwnerNamespaceConfigKey])
}

func TestParseQueuedWorkflowDefinitionError(t *testing.T) {
	_, _, err := parseQueuedWorkflowDefinition("name: [")
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse workflow definition")
}

func TestValidateQueuedWorkflowDefinitionReferencesRequiresStepIDs(t *testing.T) {
	err := validateQueuedWorkflowDefinitionReferences(`name: test
steps:
  - use: credential-offer
  - use: mobile-automation
    with:
      parameters:
        deeplink: ${{issuer-step.outputs}}
`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "pipeline uses inter-step references")
	require.Contains(t, err.Error(), "steps[0]")
	require.Contains(t, err.Error(), "steps[1]")
}

func TestValidateQueuedWorkflowDefinitionReferencesAllowsIDs(t *testing.T) {
	err := validateQueuedWorkflowDefinitionReferences(`name: test
steps:
  - id: issuer-step
    use: credential-offer
  - id: present-step
    use: mobile-automation
    with:
      parameters:
        deeplink: ${{issuer-step.outputs}}
`)
	require.NoError(t, err)
}

func TestValidateQueuedWorkflowDefinitionReferencesAllowsMissingIDsWithoutRefs(t *testing.T) {
	err := validateQueuedWorkflowDefinitionReferences(`name: test
steps:
  - use: credential-offer
  - use: mobile-automation
    with:
      parameters:
        deeplink: static-value
`)
	require.NoError(t, err)
}
