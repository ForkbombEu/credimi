// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

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

				env.RegisterActivityWithOptions(
					stepCI.Execute,
					activity.RegisterOptions{Name: stepCI.Name()},
				)
				env.RegisterActivityWithOptions(
					sendMail.Execute,
					activity.RegisterOptions{Name: sendMail.Name()},
				)
				childOpenID := NewOpenIDNetLogsWorkflow()
				env.RegisterWorkflowWithOptions(
					childOpenID.Workflow,
					workflow.RegisterOptions{Name: childOpenID.Name()},
				)

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

				env.RegisterActivityWithOptions(
					stepCI.Execute,
					activity.RegisterOptions{Name: stepCI.Name()},
				)
				env.RegisterActivityWithOptions(
					sendMail.Execute,
					activity.RegisterOptions{Name: sendMail.Name()},
				)

				childEWC := NewEWCStatusWorkflow()
				env.RegisterWorkflowWithOptions(
					childEWC.Workflow,
					workflow.RegisterOptions{Name: childEWC.Name()},
				)

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

				env.RegisterActivityWithOptions(
					stepCI.Execute,
					activity.RegisterOptions{Name: stepCI.Name()},
				)
				env.RegisterActivityWithOptions(
					sendMail.Execute,
					activity.RegisterOptions{Name: sendMail.Name()},
				)

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

				env.RegisterActivityWithOptions(
					stepCI.Execute,
					activity.RegisterOptions{Name: stepCI.Name()},
				)
				env.RegisterActivityWithOptions(
					sendMail.Execute,
					activity.RegisterOptions{Name: sendMail.Name()},
				)

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

				env.RegisterActivityWithOptions(
					stepCI.Execute,
					activity.RegisterOptions{Name: stepCI.Name()},
				)
				env.RegisterActivityWithOptions(
					sendMail.Execute,
					activity.RegisterOptions{Name: sendMail.Name()},
				)

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

				env.RegisterActivityWithOptions(
					stepCI.Execute,
					activity.RegisterOptions{Name: stepCI.Name()},
				)
				env.RegisterActivityWithOptions(
					sendMail.Execute,
					activity.RegisterOptions{Name: sendMail.Name()},
				)
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
			errorContains: "deeplink",
		},
		{
			name:  "Missing rid in captures - OpenID",
			suite: "openid_conformance_suite",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				stepCI := activities.NewStepCIWorkflowActivity()
				sendMail := activities.NewSendMailActivity()

				env.RegisterActivityWithOptions(
					stepCI.Execute,
					activity.RegisterOptions{Name: stepCI.Name()},
				)
				env.RegisterActivityWithOptions(
					sendMail.Execute,
					activity.RegisterOptions{Name: sendMail.Name()},
				)

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

			w := NewStartCheckWorkflow()
			env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

			tc.mockActivities(env)

			payload := StartCheckWorkflowPayload{
				UserMail: "test@example.org",
				Suite:    tc.suite,
				CheckID:  "test-check-id",
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
				require.Contains(
					t,
					errMsg,
					tc.errorContains,
					"Error message should contain expected text",
				)
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

func TestRunStepCIAndSendMailNoMail(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	env.RegisterActivityWithOptions(
		stepCIActivity.Execute,
		activity.RegisterOptions{Name: stepCIActivity.Name()},
	)
	env.OnActivity(stepCIActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: map[string]any{
				"captures": map[string]any{
					"deeplink": "link",
				},
			},
		}, nil).
		Once()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (StepCIAndEmailResult, error) {
			ctx = workflow.WithActivityOptions(ctx, activityOptionsNoRetry())
			cfg := StepCIAndEmailConfig{
				AppURL:        "https://app.example",
				Template:      "steps: []",
				StepCIPayload: activities.StepCIWorkflowActivityPayload{Data: map[string]any{}},
				RunMetadata:   &workflowengine.WorkflowErrorMetadata{},
				SendMail:      false,
			}
			return RunStepCIAndSendMail(ctx, cfg)
		},
		workflow.RegisterOptions{Name: "test-stepci-no-mail"},
	)

	env.ExecuteWorkflow("test-stepci-no-mail")
	require.NoError(t, env.GetWorkflowError())

	var result StepCIAndEmailResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "link", result.Captures["deeplink"])
}

func TestRunStepCIAndSendMailMissingCaptures(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	env.RegisterActivityWithOptions(
		stepCIActivity.Execute,
		activity.RegisterOptions{Name: stepCIActivity.Name()},
	)
	env.OnActivity(stepCIActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{}}, nil).
		Once()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (StepCIAndEmailResult, error) {
			ctx = workflow.WithActivityOptions(ctx, activityOptionsNoRetry())
			cfg := StepCIAndEmailConfig{
				AppURL:        "https://app.example",
				Template:      "steps: []",
				StepCIPayload: activities.StepCIWorkflowActivityPayload{Data: map[string]any{}},
				RunMetadata:   &workflowengine.WorkflowErrorMetadata{},
				SendMail:      false,
			}
			return RunStepCIAndSendMail(ctx, cfg)
		},
		workflow.RegisterOptions{Name: "test-stepci-missing-captures"},
	)

	env.ExecuteWorkflow("test-stepci-missing-captures")
	require.Error(t, env.GetWorkflowError())
}

func TestRunStepCIAndSendMailMissingDeeplink(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	env.RegisterActivityWithOptions(
		stepCIActivity.Execute,
		activity.RegisterOptions{Name: stepCIActivity.Name()},
	)
	env.OnActivity(stepCIActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: map[string]any{
				"captures": map[string]any{
					"no_deeplink": "missing",
				},
			},
		}, nil).
		Once()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (StepCIAndEmailResult, error) {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
			})
			cfg := StepCIAndEmailConfig{
				AppURL:        "https://app.example",
				AppName:       "Credimi",
				AppLogo:       "logo.png",
				UserName:      "User",
				UserMail:      "user@example.org",
				Namespace:     "ns-1",
				Template:      "steps: []",
				StepCIPayload: activities.StepCIWorkflowActivityPayload{Data: map[string]any{}},
				RunMetadata:   &workflowengine.WorkflowErrorMetadata{},
				Suite:         OpenIDConformanceSuite,
				SendMail:      true,
			}
			return RunStepCIAndSendMail(ctx, cfg)
		},
		workflow.RegisterOptions{Name: "test-stepci-missing-deeplink"},
	)

	env.ExecuteWorkflow("test-stepci-missing-deeplink")
	require.Error(t, env.GetWorkflowError())
}

func TestRunStepCIAndSendMailConfigureError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (StepCIAndEmailResult, error) {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
			})
			cfg := StepCIAndEmailConfig{
				AppURL:        "https://app.example",
				Template:      "",
				StepCIPayload: activities.StepCIWorkflowActivityPayload{Data: map[string]any{}},
				RunMetadata:   &workflowengine.WorkflowErrorMetadata{},
				SendMail:      false,
			}
			return RunStepCIAndSendMail(ctx, cfg)
		},
		workflow.RegisterOptions{Name: "test-stepci-config-error"},
	)

	env.ExecuteWorkflow("test-stepci-config-error")
	require.Error(t, env.GetWorkflowError())
}

func TestRunStepCIAndSendMailStepCIExecuteError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	env.RegisterActivityWithOptions(
		stepCIActivity.Execute,
		activity.RegisterOptions{Name: stepCIActivity.Name()},
	)
	env.OnActivity(stepCIActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, errors.New("stepci failed")).
		Once()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (StepCIAndEmailResult, error) {
			ctx = workflow.WithActivityOptions(ctx, activityOptionsNoRetry())
			cfg := StepCIAndEmailConfig{
				AppURL:        "https://app.example",
				Template:      "steps: []",
				StepCIPayload: activities.StepCIWorkflowActivityPayload{Data: map[string]any{}},
				RunMetadata:   &workflowengine.WorkflowErrorMetadata{},
				SendMail:      false,
			}
			return RunStepCIAndSendMail(ctx, cfg)
		},
		workflow.RegisterOptions{Name: "test-stepci-exec-error"},
	)

	env.ExecuteWorkflow("test-stepci-exec-error")
	require.Error(t, env.GetWorkflowError())
}

func TestRunStepCIAndSendMailInvalidAppURL(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	env.RegisterActivityWithOptions(
		stepCIActivity.Execute,
		activity.RegisterOptions{Name: stepCIActivity.Name()},
	)
	env.OnActivity(stepCIActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: map[string]any{
				"captures": map[string]any{
					"deeplink": "link",
				},
			},
		}, nil).
		Once()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (StepCIAndEmailResult, error) {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
			})
			cfg := StepCIAndEmailConfig{
				AppURL:        "http://[::1",
				Template:      "steps: []",
				StepCIPayload: activities.StepCIWorkflowActivityPayload{Data: map[string]any{}},
				RunMetadata:   &workflowengine.WorkflowErrorMetadata{},
				AppName:       "Credimi",
				AppLogo:       "logo.png",
				UserName:      "User",
				UserMail:      "user@example.org",
				Namespace:     "ns-1",
				Suite:         OpenIDConformanceSuite,
				SendMail:      true,
			}
			return RunStepCIAndSendMail(ctx, cfg)
		},
		workflow.RegisterOptions{Name: "test-stepci-invalid-url"},
	)

	env.ExecuteWorkflow("test-stepci-invalid-url")
	require.Error(t, env.GetWorkflowError())
}

func TestRunStepCIAndSendMailEmailConfigureError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	env.RegisterActivityWithOptions(
		stepCIActivity.Execute,
		activity.RegisterOptions{Name: stepCIActivity.Name()},
	)
	env.OnActivity(stepCIActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: map[string]any{
				"captures": map[string]any{
					"deeplink": "link",
				},
			},
		}, nil).
		Once()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (StepCIAndEmailResult, error) {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
			})
			cfg := StepCIAndEmailConfig{
				AppURL:        "https://app.example",
				Template:      "steps: []",
				StepCIPayload: activities.StepCIWorkflowActivityPayload{Data: map[string]any{}},
				RunMetadata:   &workflowengine.WorkflowErrorMetadata{},
				AppName:       "Credimi",
				AppLogo:       "logo.png",
				UserName:      "User",
				UserMail:      "",
				Namespace:     "ns-1",
				Suite:         OpenIDConformanceSuite,
				SendMail:      true,
			}
			return RunStepCIAndSendMail(ctx, cfg)
		},
		workflow.RegisterOptions{Name: "test-stepci-email-config-error"},
	)

	env.ExecuteWorkflow("test-stepci-email-config-error")
	require.Error(t, env.GetWorkflowError())
}

func TestRunStepCIAndSendMailEmailExecuteError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	env.RegisterActivityWithOptions(
		stepCIActivity.Execute,
		activity.RegisterOptions{Name: stepCIActivity.Name()},
	)

	emailActivity := activities.NewSendMailActivity()
	env.RegisterActivityWithOptions(
		emailActivity.Execute,
		activity.RegisterOptions{Name: emailActivity.Name()},
	)

	env.OnActivity(stepCIActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: map[string]any{
				"captures": map[string]any{
					"deeplink": "link",
				},
			},
		}, nil).
		Once()
	env.OnActivity(emailActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, errors.New("send failed")).
		Once()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (StepCIAndEmailResult, error) {
			ctx = workflow.WithActivityOptions(ctx, activityOptionsNoRetry())
			cfg := StepCIAndEmailConfig{
				AppURL:        "https://app.example",
				Template:      "steps: []",
				StepCIPayload: activities.StepCIWorkflowActivityPayload{Data: map[string]any{}},
				RunMetadata:   &workflowengine.WorkflowErrorMetadata{},
				AppName:       "Credimi",
				AppLogo:       "logo.png",
				UserName:      "User",
				UserMail:      "user@example.org",
				Namespace:     "ns-1",
				Suite:         OpenIDConformanceSuite,
				SendMail:      true,
			}
			return RunStepCIAndSendMail(ctx, cfg)
		},
		workflow.RegisterOptions{Name: "test-stepci-email-exec-error"},
	)

	env.ExecuteWorkflow("test-stepci-email-exec-error")
	require.Error(t, env.GetWorkflowError())
}

func TestStartCheckWorkflowStart(t *testing.T) {
	origStart := startCheckWorkflowWithOptions
	t.Cleanup(func() {
		startCheckWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string

	startCheckWorkflowWithOptions = func(
		namespace string,
		options client.StartWorkflowOptions,
		name string,
		_ workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedOptions = options
		capturedName = name
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}

	w := NewStartCheckWorkflow()
	result, err := w.Start("ns-1", workflowengine.WorkflowInput{})
	require.NoError(t, err)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, "run-1", result.WorkflowRunID)
	require.Equal(t, "ns-1", capturedNamespace)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, ConformanceCheckTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "conformance-check-"))
}

// activityOptionsNoRetry returns activity options for single-attempt activities in tests.
func activityOptionsNoRetry() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
}
