// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

const covWEGlobalRunnerYAML = `
name: needs global runner
steps:
  - id: step-1
    use: mobile-automation
    with:
      action_id: action-1
`

func TestCovWEScheduledPipelineEnqueueWorkflowOutcomes(t *testing.T) {
	invalidPayload := errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code
	missingConfig := errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code
	unexpectedOutput := errorcodes.Codes[errorcodes.UnexpectedActivityOutput].Code

	tests := []struct {
		name           string
		payload        ScheduledPipelineEnqueueWorkflowInput
		resolveOutput  any
		resolveErr     error
		accessErr      error
		enqueueErr     error
		wantCode       string
		wantText       string
		wantDeviceIDs  []string
		wantGlobalID   string
		wantNoEnqueued bool
	}{
		{
			name: "blank pipeline identifier",
			payload: ScheduledPipelineEnqueueWorkflowInput{
				PipelineIdentifier: "  ",
				OwnerNamespace:     "org",
			},
			wantCode: invalidPayload,
			wantText: "pipeline_identifier is required",
		},
		{
			name: "blank owner namespace",
			payload: ScheduledPipelineEnqueueWorkflowInput{
				PipelineIdentifier: "p",
				OwnerNamespace:     " ",
			},
			wantCode: invalidPayload,
			wantText: "owner_namespace is required",
		},
		{
			name:       "pipeline cannot be resolved",
			resolveErr: covWENonRetryable("pipelines record p not found"),
			wantText:   "pipelines record p not found",
		},
		{
			name:          "resolve output is not a record",
			resolveOutput: []any{"x"},
			wantCode:      unexpectedOutput,
			wantText:      "invalid output format",
		},
		{
			name:          "record without yaml",
			resolveOutput: map[string]any{"name": "p"},
			wantCode:      unexpectedOutput,
			wantText:      "missing yaml in record",
		},
		{
			name:          "invalid yaml",
			resolveOutput: map[string]any{"yaml": "steps: [unclosed"},
			wantCode:      errorcodes.Codes[errorcodes.PipelineParsingError].Code,
		},
		{
			name:          "global runner required but not configured",
			resolveOutput: map[string]any{"yaml": covWEGlobalRunnerYAML},
			wantCode:      missingConfig,
			wantText:      "global_device_id",
		},
		{
			name:          "pipeline without any runner",
			resolveOutput: map[string]any{"yaml": "name: nothing\nsteps:\n  - use: http-request\n"},
			wantCode:      missingConfig,
			wantText:      "device_ids",
		},
		{
			name: "device access denied",
			payload: ScheduledPipelineEnqueueWorkflowInput{
				GlobalDeviceID: "runner/device",
			},
			resolveOutput: map[string]any{"yaml": covWEGlobalRunnerYAML},
			accessErr:     covWENonRetryable("mobile device is not accessible"),
			wantText:      "mobile device is not accessible",
		},
		{
			name: "enqueue fails",
			payload: ScheduledPipelineEnqueueWorkflowInput{
				GlobalDeviceID: "runner/device",
			},
			resolveOutput: map[string]any{"yaml": covWEGlobalRunnerYAML},
			enqueueErr:    covWENonRetryable("queue full"),
			wantText:      "queue full",
		},
		{
			name: "global device id from payload is enqueued",
			payload: ScheduledPipelineEnqueueWorkflowInput{
				GlobalDeviceID: " runner/device ",
			},
			resolveOutput: map[string]any{"yaml": covWEGlobalRunnerYAML},
			wantDeviceIDs: []string{"runner/device"},
			wantGlobalID:  "runner/device",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
			env.RegisterActivityWithOptions(
				activities.NewResolveRecordActivity(nil).Execute,
				activity.RegisterOptions{Name: activities.ResolveRecordActivityName},
			)
			env.RegisterActivityWithOptions(
				activities.NewValidateDeviceAccessActivity(nil).Execute,
				activity.RegisterOptions{Name: activities.ValidateDeviceAccessActivityName},
			)
			var enqueued *activities.EnqueuePipelineRunTicketActivityInput
			env.RegisterActivityWithOptions(
				func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
					if tc.enqueueErr != nil {
						return workflowengine.ActivityResult{}, tc.enqueueErr
					}
					payload, err := workflowengine.DecodePayload[activities.EnqueuePipelineRunTicketActivityInput](
						input.Payload,
					)
					if err != nil {
						return workflowengine.ActivityResult{}, err
					}
					enqueued = &payload
					return workflowengine.ActivityResult{
						Output: map[string]any{"status": "queued"},
					}, nil
				},
				activity.RegisterOptions{Name: activities.EnqueuePipelineRunTicketActivityName},
			)
			env.OnActivity(activities.ResolveRecordActivityName, mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: tc.resolveOutput}, tc.resolveErr).
				Maybe()
			env.OnActivity(activities.ValidateDeviceAccessActivityName, mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{}, tc.accessErr).
				Maybe()

			payload := tc.payload
			if payload.PipelineIdentifier == "" {
				payload.PipelineIdentifier = "pipeline-1"
			}
			if payload.OwnerNamespace == "" {
				payload.OwnerNamespace = "org-1"
			}
			env.ExecuteWorkflow(
				NewScheduledPipelineEnqueueWorkflow().Workflow,
				workflowengine.WorkflowInput{
					Payload: payload,
					Config:  map[string]any{"app_url": "https://credimi.test"},
				},
			)

			require.True(t, env.IsWorkflowCompleted())
			err := env.GetWorkflowError()
			if tc.wantDeviceIDs == nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantCode)
				assert.Contains(t, err.Error(), tc.wantText)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, enqueued)
			assert.Equal(t, tc.wantDeviceIDs, enqueued.DeviceIDs)
			assert.Equal(t, tc.wantGlobalID, enqueued.PipelineConfig["global_device_id"])
			assert.Equal(t, "org-1", enqueued.PipelineConfig["namespace"])
		})
	}
}

func TestCovWEParseScheduledPipelineDefinition(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		wantErr     bool
		wantIDs     []string
		wantGlobal  bool
		wantRuntime string
	}{
		{name: "blank yaml has no runners", yaml: "  \n"},
		{name: "malformed yaml", yaml: "steps: [", wantErr: true},
		{
			name: "nested success steps and runtime global device",
			yaml: `
runtime:
  global_device_id: global/dev
steps:
  - use: http-request
    on_success:
      - use: mobile-automation
        with:
          device_id: " runner/b "
      - use: mobile-automation
        with:
          device_id: runner/a
`,
			wantIDs:     []string{"runner/a", "runner/b"},
			wantRuntime: "global/dev",
		},
		{
			name:       "mobile step without device needs the global runner",
			yaml:       covWEGlobalRunnerYAML,
			wantGlobal: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			def, info, err := parseScheduledPipelineDefinition(tc.yaml)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantIDs, info.DeviceIDs)
			assert.Equal(t, tc.wantGlobal, info.NeedsGlobalRunner)
			assert.Equal(t, tc.wantRuntime, def.Runtime.GlobalDeviceID)
		})
	}
}
