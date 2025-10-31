// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func Test_GetUseCaseVerificationDeeplinkWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		input          workflowengine.WorkflowInput
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectedErr    bool
		expectedOutput string
		errorCode      errorcodes.Code
	}{
		{
			name: "Success: retrieves use case verification deeplink",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url": "https://example.com",
				},
				Payload: map[string]any{
					"use_case_id": "test_use_case",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(httpAct.Execute, activity.RegisterOptions{Name: httpAct.Name()})
				stepCIAct := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(stepCIAct.Execute, activity.RegisterOptions{Name: stepCIAct.Name()})
				env.OnActivity(stepCIAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"captures": map[string]any{"deeplink": "test-deeplink"},
						},
					}, nil)
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"body": map[string]any{"code": "yaml-test-code"},
						},
					}, nil)
			},
			expectedOutput: "test-deeplink",
		},
		{
			name: "Failure: missing use_case_id",
			input: workflowengine.WorkflowInput{
				Config:  map[string]any{"app_url": "https://example.com"},
				Payload: map[string]any{},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectedErr:    true,
			errorCode:      errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "Failure: missing app_url",
			input: workflowengine.WorkflowInput{
				Config:  map[string]any{},
				Payload: map[string]any{"use_case_id": "test_use_case"},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectedErr:    true,
			errorCode:      errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
		},
		{
			name: "Failure: invalid HTTP output (body not a map)",
			input: workflowengine.WorkflowInput{
				Config:  map[string]any{"app_url": "https://example.com"},
				Payload: map[string]any{"use_case_id": "test_use_case"},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(httpAct.Execute, activity.RegisterOptions{Name: httpAct.Name()})
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"body": "not-a-map"}}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
		},
		{
			name: "Failure: StepCI activity fails",
			input: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url": "https://example.com",
				},
				Payload: map[string]any{
					"use_case_id": "test_use_case",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				httpAct := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(httpAct.Execute, activity.RegisterOptions{Name: httpAct.Name()})
				stepCIAct := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(stepCIAct.Execute, activity.RegisterOptions{Name: stepCIAct.Name()})
				env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{"body": map[string]any{"code": "valid-yaml"}},
					}, nil)
				env.OnActivity(stepCIAct.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, fmt.Errorf("CRE301: stepCI execution failed"))
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.CommandExecutionFailed],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			tc.mockActivities(env)

			var wf GetUseCaseVerificationDeeplinkWorkflow
			tc.input.ActivityOptions = &DefaultActivityOptions
			env.ExecuteWorkflow(wf.Workflow, tc.input)

			require.True(t, env.IsWorkflowCompleted())

			if tc.expectedErr {
				err := env.GetWorkflowError()
				require.Error(t, err)
				if tc.errorCode.Code != "" {
					require.Contains(t, err.Error(), tc.errorCode.Code)
				}
			} else {
				require.NoError(t, env.GetWorkflowError())

				var result workflowengine.WorkflowResult
				require.NoError(t, env.GetWorkflowResult(&result))
				require.Equal(t, "Successfully retrieved  use case verification deeplink", result.Message)
				require.Equal(t, tc.expectedOutput, result.Output)
			}
		})
	}
}
