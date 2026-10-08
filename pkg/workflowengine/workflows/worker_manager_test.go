// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"context"
	"errors"
	"net/http"
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
				RunnerIDs:    []string{"org/runner-1", "org/runner-2"},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				runnerHTTPAct := activities.NewMobileRunnerHTTPActivity(nil)
				env.RegisterActivityWithOptions(runnerHTTPAct.Execute, activity.RegisterOptions{
					Name: runnerHTTPAct.Name(),
				})
				env.OnActivity(runnerHTTPAct.Name(), mock.Anything, mock.Anything).Return(
					func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
						payload, err := workflowengine.DecodePayload[activities.MobileRunnerHTTPActivityPayload](
							input.Payload,
						)
						require.NoError(t, err)
						require.Equal(t, http.MethodPost, payload.Method)
						require.Contains(
							t,
							[]string{"org/runner-1", "org/runner-2"},
							payload.RunnerID,
						)
						require.Equal(t, "/worker/test-namespace", payload.Path)
						require.Equal(t, 202, payload.ExpectedStatus)
						body, ok := payload.Body.(map[string]any)
						require.True(t, ok)
						require.Equal(t, "old-test-namespace", body["old_namespace"])
						return workflowengine.ActivityResult{}, nil
					},
				).
					Times(2)
			},
			assertResult: func(t *testing.T, result workflowengine.WorkflowResult) {
				require.Equal(
					t,
					"Send namespace 'test-namespace' to start workers finished: 2/2 succeeded (0 failed)",
					result.Message,
				)
				runnerResults := assertWorkerManagerOutput(t, result.Output, 2, 2, 0)
				require.Equal(t, true, runnerResults[0]["success"])
				require.Equal(t, "org/runner-1", runnerResults[0]["runner_id"])
				require.Equal(t, true, runnerResults[1]["success"])
				require.Equal(t, "org/runner-2", runnerResults[1]["runner_id"])
			},
		},
		{
			name: "Workflow normalizes and deduplicates runner IDs",
			inputPayload: WorkerManagerWorkflowPayload{
				Namespace:    "test-namespace",
				OldNamespace: "old-test-namespace",
				RunnerIDs: []string{
					" org/runner-1 ",
					"",
					"org/runner-1",
					"org/runner-2",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				runnerHTTPAct := activities.NewMobileRunnerHTTPActivity(nil)
				env.RegisterActivityWithOptions(runnerHTTPAct.Execute, activity.RegisterOptions{
					Name: runnerHTTPAct.Name(),
				})
				env.OnActivity(runnerHTTPAct.Name(), mock.Anything, mock.Anything).Return(
					func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
						payload, err := workflowengine.DecodePayload[activities.MobileRunnerHTTPActivityPayload](
							input.Payload,
						)
						require.NoError(t, err)
						require.Contains(
							t,
							[]string{"org/runner-1", "org/runner-2"},
							payload.RunnerID,
						)
						return workflowengine.ActivityResult{}, nil
					},
				).
					Times(2)
			},
			assertResult: func(t *testing.T, result workflowengine.WorkflowResult) {
				require.Equal(
					t,
					"Send namespace 'test-namespace' to start workers finished: 2/2 succeeded (0 failed)",
					result.Message,
				)
			},
		},
		{
			name: "Workflow with explicit empty runner IDs starts no runner",
			inputPayload: WorkerManagerWorkflowPayload{
				Namespace:    "test-namespace",
				OldNamespace: "old-test-namespace",
				RunnerIDs:    []string{},
			},
			mockActivities: func(_ *testsuite.TestWorkflowEnvironment) {},
			assertResult: func(t *testing.T, result workflowengine.WorkflowResult) {
				require.Equal(
					t,
					"Send namespace 'test-namespace' to start workers finished: 0/0 succeeded (0 failed)",
					result.Message,
				)
				assertWorkerManagerOutput(t, result.Output, 0, 0, 0)
			},
		},
		{
			name: "Workflow treats nil runner IDs as empty",
			inputPayload: WorkerManagerWorkflowPayload{
				Namespace:    "test-namespace",
				OldNamespace: "old-test-namespace",
			},
			mockActivities: func(_ *testsuite.TestWorkflowEnvironment) {},
			assertResult: func(t *testing.T, result workflowengine.WorkflowResult) {
				require.Equal(
					t,
					"Send namespace 'test-namespace' to start workers finished: 0/0 succeeded (0 failed)",
					result.Message,
				)
				assertWorkerManagerOutput(t, result.Output, 0, 0, 0)
			},
		},
		{
			name: "Workflow keeps running when one runner fails",
			inputPayload: WorkerManagerWorkflowPayload{
				Namespace:    "test-namespace",
				OldNamespace: "old-test-namespace",
				RunnerIDs: []string{
					"org/runner-1",
					"org/runner-2",
					"org/runner-3",
				},
			},
			inputOptions: &workflow.ActivityOptions{
				ScheduleToCloseTimeout: DefaultActivityOptions.ScheduleToCloseTimeout,
				StartToCloseTimeout:    DefaultActivityOptions.StartToCloseTimeout,
				RetryPolicy: &temporal.RetryPolicy{
					MaximumAttempts: 1,
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				runnerHTTPAct := activities.NewMobileRunnerHTTPActivity(nil)
				env.RegisterActivityWithOptions(runnerHTTPAct.Execute, activity.RegisterOptions{
					Name: runnerHTTPAct.Name(),
				})
				env.OnActivity(runnerHTTPAct.Name(), mock.Anything, mock.Anything).Return(
					func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
						payload, err := workflowengine.DecodePayload[activities.MobileRunnerHTTPActivityPayload](
							input.Payload,
						)
						require.NoError(t, err)
						require.Equal(t, http.MethodPost, payload.Method)
						require.Equal(t, 202, payload.ExpectedStatus)
						body, ok := payload.Body.(map[string]any)
						require.True(t, ok)
						require.Equal(t, "old-test-namespace", body["old_namespace"])

						if payload.RunnerID == "org/runner-2" {
							return workflowengine.ActivityResult{}, errors.New("runner timeout")
						}

						return workflowengine.ActivityResult{}, nil
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
			mockActivities: func(_ *testsuite.TestWorkflowEnvironment) {},
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
				ActivityOptions: tc.inputOptions,
			})

			var result workflowengine.WorkflowResult
			err := env.GetWorkflowResult(&result)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tc.assertResult(t, result)
			env.AssertExpectations(t)
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
