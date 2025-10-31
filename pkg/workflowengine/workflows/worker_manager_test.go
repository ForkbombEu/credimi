// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func Test_WorkerManagerWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		inputPayload   map[string]any
		inputConfig    map[string]any
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectedErr    bool
		expectedResult any
	}{
		{
			name: "Workflow succeeds with valid namespace and old_namespace",
			inputPayload: map[string]any{
				"namespace":     "test-namespace",
				"old_namespace": "old-test-namespace",
			},
			inputConfig: map[string]any{
				"server_url": "https://test-server.com",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(httpAct.Execute, activity.RegisterOptions{
					Name: httpAct.Name(),
				})

				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{"status": "ok"},
					}, nil)
			},
			expectedResult: "Send namespace 'test-namespace' to start workers successfully",
		},
		{
			name: "Workflow fails when namespace missing",
			inputPayload: map[string]any{
				"old_namespace": "old-test-namespace",
			},
			inputConfig: map[string]any{
				"server_url": "https://test-server.com",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectedErr:    true,
		},
		{
			name: "Workflow fails when old_namespace missing",
			inputPayload: map[string]any{
				"namespace": "test-namespace",
			},
			inputConfig: map[string]any{
				"server_url": "https://test-server.com",
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectedErr:    true,
		},
		{
			name: "Workflow fails when server_url missing in config",
			inputPayload: map[string]any{
				"namespace":     "test-namespace",
				"old_namespace": "old-test-namespace",
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

			var w WorkerManagerWorkflow
			env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
				Payload: tc.inputPayload,
				Config:  tc.inputConfig,
			})

			if tc.expectedErr {
				var result workflowengine.WorkflowResult
				err := env.GetWorkflowResult(&result)
				require.Error(t, err)
			} else {
				var result workflowengine.WorkflowResult
				err := env.GetWorkflowResult(&result)
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, result.Message)
			}
		})
	}
}
