// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"errors"
	"os"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func Test_StartCheckWorkflow(t *testing.T) {
	os.Setenv("OPENIDNET_TOKEN", "test_token")

	testCases := []struct {
		name           string
		suite          string
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectErr      bool
		errorContains  string
	}{
		{
			name:  "OpenID suite succeeds",
			suite: "openid_conformance_suite",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				sendMail := activities.NewSendMailActivity()

				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
				env.RegisterActivityWithOptions(sendMail.Execute, activity.RegisterOptions{Name: sendMail.Name()})
				childOpenID := OpenIDNetLogsWorkflow{}
				env.RegisterWorkflowWithOptions(childOpenID.Workflow, workflow.RegisterOptions{Name: childOpenID.Name()})

				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"captures": map[string]any{
								"deeplink": "https://openid-link",
								"rid":      "12345",
							},
						},
					}, nil)

				env.OnActivity(sendMail.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)

				env.OnWorkflow(childOpenID.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.WorkflowResult{
						Output: map[string]any{},
						Log:    nil,
					}, nil).Maybe()
			},
			expectErr: false,
		},
		{
			name:  "EWC suite succeeds (fire-and-forget child)",
			suite: "ewc",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				sendMail := activities.NewSendMailActivity()

				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
				env.RegisterActivityWithOptions(sendMail.Execute, activity.RegisterOptions{Name: sendMail.Name()})

				childEWC := EWCStatusWorkflow{}
				env.RegisterWorkflowWithOptions(childEWC.Workflow, workflow.RegisterOptions{Name: childEWC.Name()})

				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"captures": map[string]any{
								"deeplink":   "https://ewc-link",
								"session_id": "sess-123",
							},
						},
					}, nil)

				env.OnActivity(sendMail.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)

				env.OnWorkflow(childEWC.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.WorkflowResult{
						Output: map[string]any{},
						Log:    nil,
					}, nil).Maybe()
			},
			expectErr: false,
		},
		{
			name:  "Unsupported suite fails",
			suite: "invalid_suite",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
			},
			expectErr:     true,
			errorContains: "unsupported suite",
		},
		{
			name:  "StepCI activity fails - OpenID",
			suite: "openid_conformance_suite",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				sendMail := activities.NewSendMailActivity()

				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
				env.RegisterActivityWithOptions(sendMail.Execute, activity.RegisterOptions{Name: sendMail.Name()})

				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, errors.New("StepCI execution failed"))
			},
			expectErr:     true,
			errorContains: "StepCI execution failed",
		},
		{
			name:  "StepCI activity fails - EWC",
			suite: "ewc",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				sendMail := activities.NewSendMailActivity()

				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
				env.RegisterActivityWithOptions(sendMail.Execute, activity.RegisterOptions{Name: sendMail.Name()})

				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, errors.New("StepCI connection timeout"))
			},
			expectErr:     true,
			errorContains: "StepCI connection timeout",
		},
		{
			name:  "StepCI returns invalid output - missing captures",
			suite: "openid_conformance_suite",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				sendMail := activities.NewSendMailActivity()

				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
				env.RegisterActivityWithOptions(sendMail.Execute, activity.RegisterOptions{Name: sendMail.Name()})

				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"some_field": "value",
						},
					}, nil)
			},
			expectErr:     true,
			errorContains: "StepCI unexpected output",
		},
		{
			name:  "StepCI returns invalid output - missing deeplink",
			suite: "ewc",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				sendMail := activities.NewSendMailActivity()

				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
				env.RegisterActivityWithOptions(sendMail.Execute, activity.RegisterOptions{Name: sendMail.Name()})

				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"captures": map[string]any{
								"session_id": "sess-123",
							},
						},
					}, nil)
			},
			expectErr:     true,
			errorContains: "missing deeplink",
		},
		{
			name:  "Missing rid in captures - OpenID",
			suite: "openid_conformance_suite",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				sendMail := activities.NewSendMailActivity()

				env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
				env.RegisterActivityWithOptions(sendMail.Execute, activity.RegisterOptions{Name: sendMail.Name()})

				env.OnActivity(stepCI.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{
							"captures": map[string]any{
								"deeplink": "https://openid-link",
							},
						},
					}, nil)

				env.OnActivity(sendMail.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
			},
			expectErr:     true,
			errorContains: "rid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()

			w := StartCheckWorkflow{}
			env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

			tc.mockActivities(env)

			payload := StartCheckWorkflowPayload{
				UserMail: "test@example.org",
				Suite:    tc.suite,
			}

			config := map[string]any{
				"app_url":   "https://test-app.com",
				"template":  "test-template",
				"namespace": "test-namespace",
				"app_name":  "Credimi",
				"app_logo":  "https://logo.png",
				"user_name": "John Doe",
				"memo":      map[string]any{"standard": "openid4vp_wallet"},
			}

			if tc.suite == "openid_conformance_suite" {
				payload.Variant = "test-variant"
				payload.Form = &Form{Alias: "test-alias"}
				payload.TestName = "test-name"
			}

			if tc.suite == "ewc" {
				payload.SessionID = "test-session-id"
				config["check_endpoint"] = "https://test-ewc.com"
			}

			env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
				Payload:         payload,
				Config:          config,
				ActivityOptions: &DefaultActivityOptions,
			})

			require.True(t, env.IsWorkflowCompleted(), "Workflow should complete")

			if tc.expectErr {
				require.Error(t, env.GetWorkflowError(), "Expected workflow to fail")
				errMsg := env.GetWorkflowError().Error()
				require.Contains(t, errMsg, tc.errorContains, "Error message should contain expected text")
			} else {
				require.NoError(t, env.GetWorkflowError(), "Expected workflow to succeed")

				var result workflowengine.WorkflowResult
				require.NoError(t, env.GetWorkflowResult(&result), "Should be able to get workflow result")

				require.NotNil(t, result.Output, "Result output should not be nil")
				require.Contains(t, result.Output, "deeplink", "Result should contain deeplink")

				deeplink, ok := result.Output.(map[string]any)["deeplink"].(string)
				require.True(t, ok, "Deeplink should be a string")
				require.NotEmpty(t, deeplink, "Deeplink should not be empty")

				require.Equal(t, "Check completed successfully", result.Message)
			}
		})
	}
}
