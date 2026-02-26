// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func Test_OpenIDNETWorkflows(t *testing.T) {
	var callCount int
	testCases := []struct {
		name           string
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectRunning  bool
		expectedErr    bool
		errorCode      errorcodes.Code
	}{
		{
			name: "Workflow loops when result is RUNNING",
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
					Return(workflowengine.ActivityResult{Output: map[string]any{"captures": map[string]any{"rid": "12345", "deeplink": "test"}}}, nil)
				env.OnActivity(MailActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Run(func(_ mock.Arguments) {
						callCount++
					}).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": []map[string]any{{"result": "RUNNING"}},
					}}, nil)
			},
			expectRunning: true,
		},
		{
			name: "Workflow completes when result is FINISHED",
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
					Return(workflowengine.ActivityResult{Output: map[string]any{"captures": map[string]any{"rid": "12345", "deeplink": "test"}}}, nil)
				env.OnActivity(MailActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Run(func(_ mock.Arguments) {
						callCount++
					}).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": []map[string]any{{"result": "FINISHED"}},
					}}, nil)
			},
		},
		{
			name: "Workflow fails when result is FAILURE",
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
					Return(workflowengine.ActivityResult{Output: map[string]any{"captures": map[string]any{"rid": "12345", "deeplink": "test"}}}, nil)
				env.OnActivity(MailActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Run(func(_ mock.Arguments) {
						callCount++
					}).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": []map[string]any{{"result": "FAILURE"}},
					}}, nil)
			},
			expectedErr: true,
			errorCode:   errorcodes.Codes[errorcodes.OpenIDnetCheckFailed],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			callCount = 0
			w := NewOpenIDNetWorkflow()
			env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{
				Name: w.Name(),
			})

			child := NewOpenIDNetLogsWorkflow()
			env.RegisterWorkflowWithOptions(child.Workflow, workflow.RegisterOptions{
				Name: child.Name(),
			})
			// Set environment variables
			os.Setenv("OPENIDNET_TOKEN", "test_token")

			tc.mockActivities(env)
			done := make(chan struct{})
			go func() {
				env.RegisterDelayedCallback(func() {
					env.SignalWorkflowByID(
						"default-test-workflow-id-log",
						OpenIDNetStartCheckSignal,
						nil,
					)
				}, time.Second*30)
				env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
					Payload: OpenIDNetWorkflowPayload{
						Variant:  "test-variant",
						Form:     Form{Alias: "test-alias"},
						TestName: "test-name",
						UserMail: "user@test.org",
					},
					Config: map[string]any{
						"app_url":   "https://test-app.com",
						"template":  "test-template",
						"namespace": "test-namespace",
						"app_name":  "Credimi",
						"app_logo":  "https://logo.png",
						"user_name": "John Doe",
						"memo": map[string]any{
							"standard": "openid4vp_wallet",
							"author":   "openid_conformance_suite",
						},
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
					require.Equal(t, 2, callCount) // Only two activity call (no looping)
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

func Test_LogSubWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		mockResponse   workflowengine.ActivityResult
		expectRunning  bool
		expectedCancel bool
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
		{
			name: "Workflow stops when pipeline cancel signal is received",
			mockResponse: workflowengine.ActivityResult{Output: map[string]any{
				"body": []map[string]any{{"result": "RUNNING"}},
			}},
			expectRunning:  false,
			expectedCancel: true,
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
			w := NewOpenIDNetLogsWorkflow()
			env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
				Run(func(_ mock.Arguments) {
					callCount++
				}).
				Return(tc.mockResponse, nil)
			done := make(chan struct{})
			go func() {
				env.RegisterDelayedCallback(func() {
					env.SignalWorkflow(OpenIDNetStartCheckSignal, nil)
				}, time.Second*30)
				if tc.expectedCancel {
					env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*45)
				}
				env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
					Payload: OpenIDNetLogsWorkflowPayload{
						Rid:   "12345",
						Token: "test-token",
					},
					Config: map[string]any{
						"app_url":  "https://test-app.com",
						"interval": time.Second * 10,
					},
				})

				close(done)
			}()

			if tc.expectRunning {
				env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*45)

				<-done
				require.Greater(t, callCount, 1) // Expecting multiple activity calls
			} else {
				<-done
				var result workflowengine.WorkflowResult
				err := env.GetWorkflowResult(&result)

				if tc.expectedCancel {
					require.Error(t, err)
					require.Contains(t, err.Error(), "canceled")
				} else {
					require.NoError(t, err)

					require.NotEmpty(t, result.Log)
					require.Equal(t, 2, callCount) // Only two activity call (no looping)
				}
			}
		})
	}
}

func TestOpenIDNetWorkflowStart(t *testing.T) {
	origStart := openidnetStartWorkflowWithOptions
	t.Cleanup(func() {
		openidnetStartWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string
	var capturedInput workflowengine.WorkflowInput

	openidnetStartWorkflowWithOptions = func(
		namespace string,
		options client.StartWorkflowOptions,
		name string,
		input workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedOptions = options
		capturedName = name
		capturedInput = input
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}

	w := NewOpenIDNetWorkflow()
	input := workflowengine.WorkflowInput{
		Config: map[string]any{
			"namespace": "ns-1",
		},
	}
	result, err := w.Start(input)
	require.NoError(t, err)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, "run-1", result.WorkflowRunID)
	require.Equal(t, "ns-1", capturedNamespace)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, input, capturedInput)
	require.Equal(t, OpenIDNetTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "OpenIDNetCheckWorkflow"))
	require.Equal(t, 24*time.Hour, capturedOptions.WorkflowExecutionTimeout)
}
