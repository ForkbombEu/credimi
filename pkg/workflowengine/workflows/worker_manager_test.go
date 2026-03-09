// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func Test_WorkerManagerWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		inputPayload   WorkerManagerWorkflowPayload
		inputConfig    map[string]any
		inputOptions   *workflow.ActivityOptions
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectedErr    bool
		assertResult   func(t *testing.T, result workflowengine.WorkflowResult)
	}{
		{
			name: "Workflow succeeds with valid namespace and old_namespace",
			inputPayload: WorkerManagerWorkflowPayload{
				Namespace:    "test-namespace",
				OldNamespace: "old-test-namespace",
			},
			inputConfig: map[string]any{
				"app_url": "https://test-server.com",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(httpAct.Execute, activity.RegisterOptions{
					Name: httpAct.Name(),
				})

				callCount := 0
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).Return(
					func(_ context.Context, _ workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
						callCount++
						if callCount == 1 {
							return workflowengine.ActivityResult{
								Output: map[string]any{
									"status": "ok",
									"body":   map[string]any{"runners": []any{"runner1", "runner2"}},
								},
							}, nil
						}
						return workflowengine.ActivityResult{}, nil
					},
				)
			},
			assertResult: func(t *testing.T, result workflowengine.WorkflowResult) {
				require.Equal(
					t,
					"Send namespace 'test-namespace' to start workers finished: 2/2 succeeded (0 failed)",
					result.Message,
				)
				runnerResults := assertWorkerManagerOutput(t, result.Output, 2, 2, 0)
				require.Equal(t, true, runnerResults[0]["success"])
				require.Equal(t, true, runnerResults[1]["success"])
			},
		},
		{
			name: "Workflow keeps running when one runner fails",
			inputPayload: WorkerManagerWorkflowPayload{
				Namespace:    "test-namespace",
				OldNamespace: "old-test-namespace",
			},
			inputConfig: map[string]any{
				"app_url": "https://test-server.com",
			},
			inputOptions: &workflow.ActivityOptions{
				ScheduleToCloseTimeout: DefaultActivityOptions.ScheduleToCloseTimeout,
				StartToCloseTimeout:    DefaultActivityOptions.StartToCloseTimeout,
				RetryPolicy: &temporal.RetryPolicy{
					MaximumAttempts: 1,
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(httpAct.Execute, activity.RegisterOptions{
					Name: httpAct.Name(),
				})

				callCount := 0
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).Return(
					func(_ context.Context, _ workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
						callCount++
						switch callCount {
						case 1:
							return workflowengine.ActivityResult{
								Output: map[string]any{
									"status": "ok",
									"body":   map[string]any{"runners": []any{"runner1", "runner2", "runner3"}},
								},
							}, nil
						case 3:
							return workflowengine.ActivityResult{}, errors.New("runner timeout")
						default:
							return workflowengine.ActivityResult{}, nil
						}
					},
				)
			},
			assertResult: func(t *testing.T, result workflowengine.WorkflowResult) {
				require.Equal(
					t,
					"Send namespace 'test-namespace' to start workers finished: 2/3 succeeded (1 failed)",
					result.Message,
				)

				runnerResults := assertWorkerManagerOutput(t, result.Output, 3, 2, 1)
				require.Equal(t, true, runnerResults[0]["success"])
				require.Equal(t, false, runnerResults[1]["success"])
				require.NotEmpty(t, runnerResults[1]["error"])
				require.Equal(t, true, runnerResults[2]["success"])
			},
		},
		{
			name: "Workflow fails when namespace missing",
			inputPayload: WorkerManagerWorkflowPayload{
				OldNamespace: "old-test-namespace",
			},
			inputConfig: map[string]any{
				"app_url": "https://test-server.com",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectedErr:    true,
		},
		{
			name: "Workflow fails when list API returns invalid response body",
			inputPayload: WorkerManagerWorkflowPayload{
				Namespace:    "test-namespace",
				OldNamespace: "old-test-namespace",
			},
			inputConfig: map[string]any{
				"app_url": "https://test-server.com",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewHTTPActivity()

				env.RegisterActivityWithOptions(httpAct.Execute, activity.RegisterOptions{
					Name: httpAct.Name(),
				})
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"status": "ok",
							"body": map[string]any{
								"runners": "not-an-array",
							},
						},
					}, nil)
			},
			expectedErr: true,
		},
		{
			name: "Workflow fails when app_url missing in config",
			inputPayload: WorkerManagerWorkflowPayload{
				Namespace:    "test-namespace",
				OldNamespace: "old-test-namespace",
			},
			inputConfig:    map[string]any{},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectedErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()

			tc.mockActivities(env)

			w := NewWorkerManagerWorkflow()
			env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
				Payload:         tc.inputPayload,
				Config:          tc.inputConfig,
				ActivityOptions: tc.inputOptions,
			})

			if tc.expectedErr {
				var result workflowengine.WorkflowResult
				err := env.GetWorkflowResult(&result)
				require.Error(t, err)
			} else {
				var result workflowengine.WorkflowResult
				err := env.GetWorkflowResult(&result)
				require.NoError(t, err)
				tc.assertResult(t, result)
			}
		})
	}
}

func assertWorkerManagerOutput(
	t *testing.T,
	output any,
	expectedTotal int,
	expectedSuccess int,
	expectedFailed int,
) []map[string]any {
	t.Helper()

	outputMap, ok := output.(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, expectedTotal, outputMap["total_runners"])
	require.EqualValues(t, expectedSuccess, outputMap["successful_runners"])
	require.EqualValues(t, expectedFailed, outputMap["failed_runners"])

	runnerResultsRaw, ok := outputMap["runner_results"].([]any)
	require.True(t, ok)
	require.Len(t, runnerResultsRaw, expectedTotal)

	runnerResults := make([]map[string]any, 0, len(runnerResultsRaw))
	for _, result := range runnerResultsRaw {
		runnerResult, ok := result.(map[string]any)
		require.True(t, ok)
		runnerResults = append(runnerResults, runnerResult)
	}

	return runnerResults
}

func TestWorkerManagerWorkflowStart(t *testing.T) {
	origStart := workerManagerStartWorkflowWithOptions
	t.Cleanup(func() {
		workerManagerStartWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string
	var capturedInput workflowengine.WorkflowInput

	workerManagerStartWorkflowWithOptions = func(
		namespace string,
		options client.StartWorkflowOptions,
		name string,
		input workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedOptions = options
		capturedName = name
		capturedInput = input
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}

	w := NewWorkerManagerWorkflow()
	input := workflowengine.WorkflowInput{
		Payload: WorkerManagerWorkflowPayload{Namespace: "org-1"},
	}
	result, err := w.Start("ns-1", input)
	require.NoError(t, err)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, "run-1", result.WorkflowRunID)
	require.Equal(t, "ns-1", capturedNamespace)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, input, capturedInput)
	require.Equal(t, WorkerManagerTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "worker-manager-"))
}
