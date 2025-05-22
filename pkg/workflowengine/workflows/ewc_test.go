// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"testing"
	"time"

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
		errorMessage   string
	}{
		{
			name: "Workflow completes when status is success",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				var StepCIActivity activities.StepCIWorkflowActivity
				env.RegisterActivityWithOptions(StepCIActivity.Execute, activity.RegisterOptions{
					Name: StepCIActivity.Name(),
				})
				var MailActivity activities.SendMailActivity
				env.RegisterActivityWithOptions(MailActivity.Execute, activity.RegisterOptions{
					Name: MailActivity.Name(),
				})
				var HTTPActivity activities.HTTPActivity
				env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
					Name: HTTPActivity.Name(),
				})

				env.OnActivity(StepCIActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"captures": map[string]any{"deep_link": "test_content", "session_id": "12345"}}}, nil)
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
				var StepCIActivity activities.StepCIWorkflowActivity
				env.RegisterActivityWithOptions(StepCIActivity.Execute, activity.RegisterOptions{
					Name: StepCIActivity.Name(),
				})
				var MailActivity activities.SendMailActivity
				env.RegisterActivityWithOptions(MailActivity.Execute, activity.RegisterOptions{
					Name: MailActivity.Name(),
				})
				var HTTPActivity activities.HTTPActivity
				env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
					Name: HTTPActivity.Name(),
				})

				env.OnActivity(StepCIActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"captures": map[string]any{"deep_link": "test_content", "session_id": "12345"}}}, nil)
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
				var StepCIActivity activities.StepCIWorkflowActivity
				env.RegisterActivityWithOptions(StepCIActivity.Execute, activity.RegisterOptions{
					Name: StepCIActivity.Name(),
				})
				var MailActivity activities.SendMailActivity
				env.RegisterActivityWithOptions(MailActivity.Execute, activity.RegisterOptions{
					Name: MailActivity.Name(),
				})
				var HTTPActivity activities.HTTPActivity
				env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
					Name: HTTPActivity.Name(),
				})

				env.OnActivity(StepCIActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"captures": map[string]any{"deep_link": "test_content", "session_id": "12345"}}}, nil)
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
			expectedErr:  true,
			errorMessage: "EWC check failed: fail test reason",
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
					Payload: map[string]any{
						"session_id": "12345",
						"app_url":    "https://test-app.com",
						"user_mail":  "test@example.org",
					},
					Config: map[string]any{
						"template":       "test-template",
						"check_endpoint": "test/endpoint",
						"namespace":      "test-namespace",
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
				require.Contains(t, env.GetWorkflowResult(&result).Error(), tc.errorMessage)
			}
		})
	}
}
