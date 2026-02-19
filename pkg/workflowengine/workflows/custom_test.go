// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
)

func Test_CustomCheckWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		inputPayload   CustomCheckWorkflowPayload
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectedErr    bool
		errorCode      errorcodes.Code
		expectedResult any
	}{
		{
			name: "Workflow succeeds when yaml is provided in payload",
			inputPayload: CustomCheckWorkflowPayload{
				Yaml: "test-yaml-content",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{
					Name: stepCI.Name(),
				})

				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"tests": []any{"test1", "test2"},
					}}, nil)
			},
			expectedResult: []any{"test1", "test2"},
		},
		{
			name: "Workflow fetches yaml via HTTP when only id is provided",
			inputPayload: CustomCheckWorkflowPayload{
				CheckID: "custom-check-id",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{
					Name: stepCI.Name(),
				})
				httpAct := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(httpAct.Execute, activity.RegisterOptions{
					Name: httpAct.Name(),
				})

				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]any{
							"record": map[string]any{
								"yaml": "fetched-yaml",
							},
						},
					}}, nil)

				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"tests": []any{"ok"},
					}}, nil)
			},
			expectedResult: []any{"ok"},
		},
		{
			name:         "Workflow fails when yaml missing in HTTP response",
			inputPayload: CustomCheckWorkflowPayload{CheckID: "broken-id"},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{
					Name: stepCI.Name(),
				})
				httpAct := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(httpAct.Execute, activity.RegisterOptions{
					Name: httpAct.Name(),
				})

				// Return response without yaml
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]any{
							"record": map[string]any{},
						},
					}}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()

			tc.mockActivities(env)

			w := NewCustomCheckWorkflow()
			env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
				Payload: tc.inputPayload,
				Config: map[string]any{
					"app_url": "https://test-app.com",
					"env":     "test",
				},
			})

			if tc.expectedErr {
				var result workflowengine.WorkflowResult
				err := env.GetWorkflowResult(&result)
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorCode.Code)
				require.Contains(t, err.Error(), tc.errorCode.Description)
			} else {
				var result workflowengine.WorkflowResult
				err := env.GetWorkflowResult(&result)
				require.NoError(t, err)
				require.NotNil(t, result.Output)
				require.Equal(t, tc.expectedResult, result.Output)
			}
		})
	}
}

func TestCustomCheckWorkflowStart(t *testing.T) {
	origStart := customStartWorkflowWithOptions
	t.Cleanup(func() {
		customStartWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string
	var capturedInput workflowengine.WorkflowInput

	customStartWorkflowWithOptions = func(
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

	w := NewCustomCheckWorkflow()
	input := workflowengine.WorkflowInput{Payload: CustomCheckWorkflowPayload{Yaml: "steps: []"}}
	result, err := w.Start("ns-1", input)
	require.NoError(t, err)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, "run-1", result.WorkflowRunID)
	require.Equal(t, "ns-1", capturedNamespace)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, input, capturedInput)
	require.Equal(t, CustomCheckTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "custom-"))
}
