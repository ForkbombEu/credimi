// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"os"
	"testing"
	"time"

	"github.com/forkbombeu/didimo/pkg/workflowengine"

	"github.com/forkbombeu/didimo/pkg/workflowengine/activities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func Test_OpenIDNETWorkflows(t *testing.T) {
	testCases := []struct {
		name                string
		mockActivities      func(env *testsuite.TestWorkflowEnvironment)
		completeSignalDelay time.Duration
		signalData          SignalData
		startLogsDelay      time.Duration
		expectedMsg         string
	}{
		{
			name: "Signal before child completes",
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
					Return(workflowengine.ActivityResult{Output: map[string]any{"rid": 12345}}, nil)
				env.OnActivity(MailActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": []map[string]any{{"result": "RUNNING"}},
					}}, nil)
			},
			completeSignalDelay: time.Minute,
			signalData:          SignalData{Success: false, Reason: "Test failure"},
			startLogsDelay:      time.Second * 30,
			expectedMsg:         "Workflow terminated with a failure message: Test failure",
		},
		{
			name: "Child terminates before signal",
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
					Return(workflowengine.ActivityResult{Output: map[string]any{"rid": 12345}}, nil)
				env.OnActivity(MailActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": []map[string]any{{"result": "FINISHED"}},
					}}, nil)
			},
			completeSignalDelay: 2 * time.Minute,
			signalData:          SignalData{Success: true},
			startLogsDelay:      time.Second * 30,
			expectedMsg:         "Workflow completed successfully",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			env.SetTestTimeout(100 * time.Minute)
			var w OpenIDNetWorkflow
			env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{
				Name: w.Name(),
			})
			var child OpenIDNetLogsWorkflow
			env.RegisterWorkflowWithOptions(child.Workflow, workflow.RegisterOptions{
				Name: child.Name(),
			})
			// Set environment variables
			os.Setenv("OPENIDNET_TOKEN", "test_token")
			os.Setenv("MAIL_SENDER", "test@example.org")

			tc.mockActivities(env)

			env.RegisterDelayedCallback(func() {
				env.SignalWorkflow("wallet-test-signal", tc.signalData)
			}, tc.completeSignalDelay)
			env.RegisterDelayedCallback(func() {
				env.SignalWorkflowByID("default-test-workflow-id-log", "wallet-test-start-log-update", nil)
			}, tc.completeSignalDelay)
			// Execute workflow
			env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
				Payload: map[string]any{
					"variant":   "test-variant",
					"form":      mock.Anything,
					"user_mail": "user@test.org",
					"app_url":   "https://test-app.com",
				},
				Config: map[string]any{
					"template": "test-template",
				},
			})
			var result workflowengine.WorkflowResult
			require.NoError(t, env.GetWorkflowResult(&result))
			require.Equal(t, tc.expectedMsg, result.Message)
		})
	}
}

func Test_LogSubWorkflow(t *testing.T) {
	testCases := []struct {
		name          string
		mockResponse  workflowengine.ActivityResult
		expectRunning bool
	}{
		{
			name: "Workflow completes when result is FINISHED",
			mockResponse: workflowengine.ActivityResult{Output: map[string]any{
				"body": []map[string]any{{"result": "FINISHED"}},
			}},
			expectRunning: false,
		},
		{
			name: "Workflow runs indefinitely when result is RUNNING",
			mockResponse: workflowengine.ActivityResult{Output: map[string]any{
				"body": []map[string]any{{"result": "RUNNING"}},
			}},
			expectRunning: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()

			callCount := 0
			var HTTPActivity activities.HTTPActivity
			env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
				Name: HTTPActivity.Name(),
			})
			var logsWorkflow OpenIDNetLogsWorkflow
			env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
				Run(func(_ mock.Arguments) {
					callCount++
				}).
				Return(tc.mockResponse, nil)
			done := make(chan struct{})
			go func() {
				env.RegisterDelayedCallback(func() {
					env.SignalWorkflow("wallet-test-start-log-update", nil)
				}, time.Second*30)
				env.ExecuteWorkflow(logsWorkflow.Workflow, workflowengine.WorkflowInput{
					Payload: map[string]any{
						"rid":     "12345",
						"token":   "test-token",
						"app_url": "https://test-app.com",
					},
					Config: map[string]any{
						"interval": time.Second * 10,
					},
				})

				close(done)
			}()

			if tc.expectRunning {
				env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*45)

				<-done
				assert.Greater(t, callCount, 1) // Expecting multiple activity calls
			} else {
				<-done
				var result workflowengine.WorkflowResult
				assert.NoError(t, env.GetWorkflowResult(&result))
				assert.NotEmpty(t, result.Log)
				assert.Equal(t, callCount, 1) // Only one activity call (no looping)
			}
		})
	}
}
