// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
)

func TestPipelineRetentionWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		input          workflowengine.WorkflowInput
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectError    bool
		assertResult   func(t *testing.T, result workflowengine.WorkflowResult)
	}{
		{
			name: "success",
			input: workflowengine.WorkflowInput{
				Payload: PipelineRetentionWorkflowInput{
					OlderThanDays: 30,
					DryRun:        true,
					BatchSize:     100,
				},
				Config: map[string]any{
					"app_url": "https://credimi.test",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				act := activities.NewInternalHTTPActivity()
				env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
				env.OnActivity(
					act.Name(),
					mock.Anything,
					mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
						payload, ok := input.Payload.(map[string]any)
						if !ok {
							return false
						}
						url, _ := payload["url"].(string)
						body, _ := payload["body"].(map[string]any)
						return strings.Contains(url, "/api/pipeline/retention/delete-files") &&
							body["batch_size"] == float64(PipelineRetentionDefaultBatchSize)
					}),
				).Return(
					workflowengine.ActivityResult{
						Output: map[string]any{
							"body": map[string]any{
								"older_than_days": 30.0,
								"matched_records": 12.0,
								"updated_records": 0.0,
							},
						},
					},
					nil,
				).Once()
			},
			assertResult: func(t *testing.T, result workflowengine.WorkflowResult) {
				require.Equal(t, "Pipeline retention dry run completed", result.Message)
				body, ok := result.Output.(map[string]any)
				require.True(t, ok)
				require.Equal(t, 30.0, body["older_than_days"])
				require.Equal(t, 12.0, body["matched_records"])
			},
		},
		{
			name: "missing app_url",
			input: workflowengine.WorkflowInput{
				Payload: PipelineRetentionWorkflowInput{
					OlderThanDays: 30,
				},
				Config: map[string]any{},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectError:    true,
		},
		{
			name: "invalid activity output",
			input: workflowengine.WorkflowInput{
				Payload: PipelineRetentionWorkflowInput{
					OlderThanDays: 30,
				},
				Config: map[string]any{
					"app_url": "https://credimi.test",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				act := activities.NewInternalHTTPActivity()
				env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
				env.OnActivity(
					act.Name(),
					mock.Anything,
					mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
						payload, ok := input.Payload.(map[string]any)
						if !ok {
							return false
						}
						body, _ := payload["body"].(map[string]any)
						return body["batch_size"] == float64(PipelineRetentionDefaultBatchSize)
					}),
				).Return(
					workflowengine.ActivityResult{Output: "bad-output"},
					nil,
				).Once()
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suite := &testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			tc.mockActivities(env)

			w := NewPipelineRetentionWorkflow()
			env.ExecuteWorkflow(w.Workflow, tc.input)

			var result workflowengine.WorkflowResult
			err := env.GetWorkflowResult(&result)
			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			tc.assertResult(t, result)
		})
	}
}

func TestPipelineRetentionWorkflowStart(t *testing.T) {
	orig := pipelineRetentionStartWorkflowWithOptions
	t.Cleanup(func() {
		pipelineRetentionStartWorkflowWithOptions = orig
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string
	var capturedInput workflowengine.WorkflowInput

	pipelineRetentionStartWorkflowWithOptions = func(
		namespace string,
		options client.StartWorkflowOptions,
		name string,
		input workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedOptions = options
		capturedName = name
		capturedInput = input

		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-1",
			WorkflowRunID: "run-1",
			Message:       "started",
		}, nil
	}

	w := NewPipelineRetentionWorkflow()
	input := workflowengine.WorkflowInput{
		Payload: PipelineRetentionWorkflowInput{OlderThanDays: 30},
	}

	result, err := w.Start(DefaultNamespace, input)
	require.NoError(t, err)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, DefaultNamespace, capturedNamespace)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "pipeline-retention-"))
	require.Equal(t, PipelineRetentionTaskQueue, capturedOptions.TaskQueue)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, input, capturedInput)
}

func TestPipelineRetentionWorkflowName(t *testing.T) {
	require.Equal(t, "PipelineRetentionWorkflow", NewPipelineRetentionWorkflow().Name())
}

func TestPipelineRetentionWorkflowOptions(t *testing.T) {
	require.Equal(t, DefaultActivityOptions, NewPipelineRetentionWorkflow().GetOptions())
}
