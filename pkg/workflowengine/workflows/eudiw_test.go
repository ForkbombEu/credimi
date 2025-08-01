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

func Test_EudiwWorkflow(t *testing.T) {
	var callCount int
	testCases := []struct {
		name           string
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectRunning  bool
		expectedErr    bool
		errorCode      errorcodes.Code
	}{
		{
			name: "Workflow completes when status is 200",
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
					Return(workflowengine.ActivityResult{Output: map[string]any{"captures": map[string]any{"client_id": "test_client_id", "transaction_id": "12345", "request_uri": "test_uri"}}}, nil)
				env.OnActivity(MailActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Run(func(_ mock.Arguments) {
						callCount++
					}).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"status": 200,
						"body":   map[string]any{"events": []map[string]any{{"logs": "test_logs"}}},
					}}, nil)
			},
			expectRunning: false,
		},
		{
			name: "Workflow loops when status is 400",
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
					Return(workflowengine.ActivityResult{Output: map[string]any{"captures": map[string]any{"client_id": "test_client_id", "transaction_id": "12345", "request_uri": "test_uri"}}}, nil)
				env.OnActivity(MailActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Run(func(_ mock.Arguments) {
						callCount++
					}).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"status": 400,
						"body":   map[string]any{"events": []map[string]any{{"logs": "test_logs"}}},
					}}, nil)
			},
			expectRunning: true,
		},
		{
			name: "Workflow fails when status is 500",
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
					Return(workflowengine.ActivityResult{Output: map[string]any{"captures": map[string]any{"client_id": "test_client_id", "transaction_id": "12345", "request_uri": "test_uri"}}}, nil)
				env.OnActivity(MailActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Run(func(_ mock.Arguments) {
						callCount++
					}).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"status": 500,
						"body":   map[string]any{"events": []map[string]any{{"logs": "test_logs"}}},
					}}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.EudiwCheckFailed],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()

			callCount = 0
			var w EudiwWorkflow
			tc.mockActivities(env)
			done := make(chan struct{})
			go func() {
				env.RegisterDelayedCallback(func() {
					env.SignalWorkflow(EudiwStartCheckSignal, nil)
				}, time.Second*30)
				env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
					Payload: map[string]any{
						"nonce":     "12345",
						"id":        "12345",
						"user_mail": "test@example.org",
					},
					Config: map[string]any{
						"app_url":   "https://test-app.com",
						"template":  "test-template",
						"namespace": "test-namespace",
					},
				})

				close(done)
			}()
			if !tc.expectedErr {
				if tc.expectRunning {
					env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*90)

					<-done
					require.Greater(t, callCount, 3) // Expecting multiple activity calls
				} else {
					<-done
					var result workflowengine.WorkflowResult
					require.NoError(t, env.GetWorkflowResult(&result))
					require.Equal(t, 3, callCount) // Only two activity call (no looping)
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
