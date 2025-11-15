// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func Test_EWCWorkflow(t *testing.T) {
	var callCount int
	testCases := []struct {
		name           string
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectRunning  bool
		expectedErr    bool
		errorCode      errorcodes.Code
	}{
		{
			name: "Workflow completes when status is success",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				StepCIActivity := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(StepCIActivity.Execute, activity.RegisterOptions{
					Name: StepCIActivity.Name(),
				})
				MailActivity := activities.NewSendMailActivity()
				env.RegisterActivityWithOptions(MailActivity.Execute, activity.RegisterOptions{
					Name: MailActivity.Name(),
				})
				HTTPActivity := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
					Name: HTTPActivity.Name(),
				})

				env.OnActivity(StepCIActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"captures": map[string]any{"deeplink": "test_content", "session_id": "12345"}}}, nil)
				env.OnActivity(MailActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Run(func(_ mock.Arguments) {
						callCount++
					}).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]string{"status": "success"},
					}}, nil)
			},
			expectRunning: false,
		},
		{
			name: "Workflow loops when status is pending",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				StepCIActivity := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(StepCIActivity.Execute, activity.RegisterOptions{
					Name: StepCIActivity.Name(),
				})
				MailActivity := activities.NewSendMailActivity()
				env.RegisterActivityWithOptions(MailActivity.Execute, activity.RegisterOptions{
					Name: MailActivity.Name(),
				})
				HTTPActivity := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
					Name: HTTPActivity.Name(),
				})

				env.OnActivity(StepCIActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"captures": map[string]any{"deeplink": "test_content", "session_id": "12345"}}}, nil)
				env.OnActivity(MailActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Run(func(_ mock.Arguments) {
						callCount++
					}).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]string{"status": "pending", "reason": "ok"},
					}}, nil)
			},
			expectRunning: true,
		},
		{
			name: "Workflow fails when status is failed",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				StepCIActivity := activities.NewStepCIWorkflowActivity()
				env.RegisterActivityWithOptions(StepCIActivity.Execute, activity.RegisterOptions{
					Name: StepCIActivity.Name(),
				})
				MailActivity := activities.NewSendMailActivity()
				env.RegisterActivityWithOptions(MailActivity.Execute, activity.RegisterOptions{
					Name: MailActivity.Name(),
				})
				HTTPActivity := activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
					Name: HTTPActivity.Name(),
				})

				env.OnActivity(StepCIActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"captures": map[string]any{"deeplink": "test_content", "session_id": "12345"}}}, nil)
				env.OnActivity(MailActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Run(func(_ mock.Arguments) {
						callCount++
					}).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]string{"status": "failed", "reason": "fail test reason"},
					}}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.EWCCheckFailed],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()

			callCount = 0
			var w EWCWorkflow
			tc.mockActivities(env)
			done := make(chan struct{})
			go func() {
				env.RegisterDelayedCallback(func() {
					env.SignalWorkflow(EwcStartCheckSignal, nil)
				}, time.Second*30)
				env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
					Payload: EWCWorkflowPayload{
						SessionID: "12345",
						UserMail:  "test@example.org",
					},
					Config: map[string]any{
						"app_url":        "https://test-app.com",
						"template":       "test-template",
						"check_endpoint": "test/endpoint",
						"namespace":      "test-namespace",
						"app_name":       "Credimi",
						"app_logo":       "https://logo.png",
						"user_name":      "John Doe",
						"memo": map[string]any{
							"standard": "openid4vp_wallet",
							"author":   "ewc",
						},
					},
				})

				close(done)
			}()
			if !tc.expectedErr {
				if tc.expectRunning {
					env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*90)

					<-done
					require.Greater(t, callCount, 1) // Expecting multiple activity calls
				} else {
					<-done
					var result workflowengine.WorkflowResult
					require.NoError(t, env.GetWorkflowResult(&result))
					require.Equal(t, 1, callCount) // Only two activity call (no looping)
				}
			} else {
				<-done
				var result workflowengine.WorkflowResult
				require.Error(t, env.GetWorkflowResult(&result))
				require.Contains(t, env.GetWorkflowResult(&result).Error(), tc.errorCode.Code)
				require.Contains(t, env.GetWorkflowResult(&result).Error(), tc.errorCode.Description)
			}
		})
	}
}

func Test_EWCStatusWorkflow(t *testing.T) {
	testCases := []struct {
		name          string
		mockResponse  workflowengine.ActivityResult
		expectRunning bool
		expectedErr   bool
	}{
		{
			name: "Workflow completes when status is success",
			mockResponse: workflowengine.ActivityResult{Output: map[string]any{
				"body": map[string]any{
					"status": "success",
					"claims": []string{"claim1", "claim2"},
				},
			}},
			expectRunning: false,
			expectedErr:   false,
		},
		{
			name: "Workflow keeps polling when status is pending",
			mockResponse: workflowengine.ActivityResult{Output: map[string]any{
				"body": map[string]any{
					"status": "pending",
					"reason": "ok",
				},
			}},
			expectRunning: true,
			expectedErr:   false,
		},
		{
			name: "Workflow fails when status is failed",
			mockResponse: workflowengine.ActivityResult{Output: map[string]any{
				"body": map[string]any{
					"status": "failed",
					"reason": "failure reason",
				},
			}},
			expectRunning: false,
			expectedErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()

			callCount := 0
			HTTPActivity := activities.NewHTTPActivity()
			env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
				Name: HTTPActivity.Name(),
			})

			env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
				Run(func(_ mock.Arguments) {
					callCount++
				}).
				Return(tc.mockResponse, nil)

			var w EWCStatusWorkflow

			done := make(chan struct{})
			go func() {
				env.RegisterDelayedCallback(func() {
					env.SignalWorkflow(EwcStartCheckSignal, nil)
				}, time.Second*30)

				env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
					Payload: EWCStatusWorkflowPayload{
						SessionID: "12345",
					},
					Config: map[string]any{
						"app_url":        "https://test-app.com",
						"check_endpoint": "https://api.test/ewc",
						"interval":       float64(time.Second * 10),
					},
				})

				close(done)
			}()

			if tc.expectRunning {
				env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*45)

				<-done
				require.Greater(t, callCount, 1, "Expected multiple HTTP calls for ongoing polling")
			} else {
				<-done
				var result workflowengine.WorkflowResult
				if tc.expectedErr {
					require.Error(t, env.GetWorkflowResult(&result))
				} else {
					require.NoError(t, env.GetWorkflowResult(&result))
					require.NotEmpty(t, result.Message)
				}
				require.GreaterOrEqual(t, callCount, 1)
			}
		})
	}
}
