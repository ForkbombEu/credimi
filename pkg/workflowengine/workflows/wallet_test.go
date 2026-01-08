// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"reflect"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func Test_WalletWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		payload        WalletWorkflowPayload
		config         map[string]any
		expectError    bool
		errorCode      errorcodes.Code
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
	}{
		{
			name: "Valid Workflow Run (Google URL)",
			payload: WalletWorkflowPayload{
				URL: "https://com.example.wallet",
			},
			config: map[string]any{
				"namespace": "namespace",
				"app_url":   "http://app.example.com",
			},
			expectError: false,
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				parseActivity := activities.NewParseWalletURLActivity()
				dockerActivity := activities.NewDockerActivity()
				jsonActivity := activities.NewJSONActivity(map[string]reflect.Type{
					"map": reflect.TypeOf(map[string]any{}),
				})

				env.RegisterActivityWithOptions(parseActivity.Execute, activity.RegisterOptions{
					Name: parseActivity.Name(),
				})
				env.RegisterActivityWithOptions(dockerActivity.Execute, activity.RegisterOptions{
					Name: dockerActivity.Name(),
				})
				env.RegisterActivityWithOptions(jsonActivity.Execute, activity.RegisterOptions{
					Name: jsonActivity.Name(),
				})

				testdata := `{"test": "test", "id": "A12345"}`
				env.OnActivity(parseActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"api_input": "http://example.com", "store_type": "google"}}, nil)
				env.OnActivity(dockerActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"stdout": testdata}}, nil)
				env.OnActivity(jsonActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"status": 200, "body": "test_result"}}, nil)
			},
		},
		{
			name: "Valid Workflow Run (Apple URL)",
			payload: WalletWorkflowPayload{
				URL: "https://com.example.wallet",
			},
			config: map[string]any{
				"namespace": "namespace",
				"app_url":   "http://app.example.com",
			},
			expectError: false,
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				HTTPActivity := activities.NewHTTPActivity()
				parseActivity := activities.NewParseWalletURLActivity()
				env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
					Name: HTTPActivity.Name(),
				})
				env.RegisterActivityWithOptions(parseActivity.Execute, activity.RegisterOptions{
					Name: parseActivity.Name(),
				})

				testdata := []map[string]any{{"test": "test", "id": "A12345"}}
				env.OnActivity(parseActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"api_input": "http://example.com", "store_type": "apple"}}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"status": 200, "body": map[string]any{"results": testdata}}}, nil)
			},
		},
		{
			name: "Invalid Workflow Run",
			payload: WalletWorkflowPayload{
				URL: "https://com.example.wallet",
			},
			config: map[string]any{
				"namespace": "namespace",
				"app_url":   "http://app.example.com",
			},
			expectError: true,
			errorCode:   errorcodes.Codes[errorcodes.ParseURLFailed],
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				HTTPActivity := activities.NewHTTPActivity()
				parseActivity := activities.NewParseWalletURLActivity()
				dockerActivity := activities.NewDockerActivity()
				jsonActivity := activities.NewJSONActivity(map[string]reflect.Type{
					"map": reflect.TypeOf(map[string]any{}),
				})

				env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
					Name: HTTPActivity.Name(),
				})
				env.RegisterActivityWithOptions(parseActivity.Execute, activity.RegisterOptions{
					Name: parseActivity.Name(),
				})
				env.RegisterActivityWithOptions(dockerActivity.Execute, activity.RegisterOptions{
					Name: dockerActivity.Name(),
				})
				env.RegisterActivityWithOptions(dockerActivity.Execute, activity.RegisterOptions{
					Name: jsonActivity.Name(),
				})
				env.OnActivity(parseActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, workflowengine.NewAppError(errorcodes.Codes[errorcodes.ParseURLFailed], ""))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			tc.mockActivities(env)

			w := NewWalletWorkflow()
			env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
				Payload: tc.payload,
				Config:  tc.config,
			})

			if tc.expectError {
				var result workflowengine.WorkflowResult
				require.Error(t, env.GetWorkflowResult(&result))
				require.Contains(t, env.GetWorkflowResult(&result).Error(), tc.errorCode.Code)
				require.Contains(
					t,
					env.GetWorkflowResult(&result).Error(),
					tc.errorCode.Description,
				)
			} else {
				var result workflowengine.WorkflowResult
				require.NoError(t, env.GetWorkflowResult(&result))
			}
		})
	}
}
