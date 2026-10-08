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
				Payload: GetUseCaseVerificationDeeplinkWorkflowPayload{
					UseCaseIdentifier: "test_use_case",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				env.RegisterActivityWithOptions(
					activities.NewGetUseCaseVerificationDeeplinkActivity(nil).Execute,
					activity.RegisterOptions{
						Name: activities.GetUseCaseVerificationDeeplinkActivityName,
					},
				)
				stepCIAct := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(
					stepCIAct.Execute,
					activity.RegisterOptions{Name: stepCIAct.Name()},
				)
				env.OnActivity(
					activities.GetUseCaseVerificationDeeplinkActivityName,
					mock.Anything,
					mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
						payload, err := workflowengine.DecodePayload[activities.GetUseCaseVerificationDeeplinkInput](
							input.Payload,
						)
						return err == nil && payload.UseCaseIdentifier == "test_use_case"
					}),
				).
					Return(workflowengine.ActivityResult{
						Output:  map[string]any{"code": "yaml-test-code"},
						Secrets: map[string]any{"pin": "1234"},
					}, nil)
				env.OnActivity(
					stepCIAct.Name(),
					mock.Anything,
					mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
						payload, ok := input.Payload.(map[string]any)
						return ok &&
							payload["yaml"] == "yaml-test-code" &&
							requireSecrets(t, input.Secrets, map[string]string{
								"pin": "1234",
							})
					}),
				).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"captures": map[string]any{"deeplink": "test-deeplink"},
						},
					}, nil)
			},
			expectedOutput: "test-deeplink",
		},
		{
			name: "Failure: missing use_case_id",
			input: workflowengine.WorkflowInput{
				Payload: GetUseCaseVerificationDeeplinkWorkflowPayload{},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {},
			expectedErr:    true,
			errorCode:      errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "Failure: invalid activity output (not a map)",
			input: workflowengine.WorkflowInput{
				Payload: GetUseCaseVerificationDeeplinkWorkflowPayload{
					UseCaseIdentifier: "test_use_case",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				env.RegisterActivityWithOptions(
					activities.NewGetUseCaseVerificationDeeplinkActivity(nil).Execute,
					activity.RegisterOptions{
						Name: activities.GetUseCaseVerificationDeeplinkActivityName,
					},
				)
				env.OnActivity(
					activities.GetUseCaseVerificationDeeplinkActivityName,
					mock.Anything,
					mock.Anything,
				).
					Return(workflowengine.ActivityResult{Output: "not-a-map"}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
		},
		{
			name: "Failure: StepCI activity fails",
			input: workflowengine.WorkflowInput{
				Payload: GetUseCaseVerificationDeeplinkWorkflowPayload{
					UseCaseIdentifier: "test_use_case",
				},
			},
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				env.RegisterActivityWithOptions(
					activities.NewGetUseCaseVerificationDeeplinkActivity(nil).Execute,
					activity.RegisterOptions{
						Name: activities.GetUseCaseVerificationDeeplinkActivityName,
					},
				)
				stepCIAct := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(
					stepCIAct.Execute,
					activity.RegisterOptions{Name: stepCIAct.Name()},
				)
				env.OnActivity(
					activities.GetUseCaseVerificationDeeplinkActivityName,
					mock.Anything,
					mock.Anything,
				).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{"code": "valid-yaml"},
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

			w := NewGetUseCaseVerificationDeeplinkWorkflow()
			tc.input.ActivityOptions = &DefaultActivityOptions
			tc.input.Config = map[string]any{"app_url": "https://example.com"}
			env.ExecuteWorkflow(w.Workflow, tc.input)

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
				require.Equal(
					t,
					"Successfully retrieved  use case verification deeplink",
					result.Message,
				)
				require.Equal(t, tc.expectedOutput, result.Output)
			}
		})
	}
}
