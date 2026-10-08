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

func Test_OpenID4VPWalletWorkflows(t *testing.T) {
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
				registerOpenIDNetLogPushActivity(env, func() { callCount++ })

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
				registerOpenIDNetLogPushActivity(env, func() { callCount++ })

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
				registerOpenIDNetLogPushActivity(env, func() { callCount++ })

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
			w := NewOpenID4VPWalletWorkflow()
			env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{
				Name: w.Name(),
			})

			child := NewOpenID4VPWalletLogsWorkflow()
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
						OpenID4VPWalletStartCheckSignal,
						nil,
					)
				}, time.Second*30)
				env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
					Payload: OpenID4VPWalletWorkflowPayload{
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
					require.Equal(t, 2, callCount) // Logs poll and log push (no looping)
				}
			} else {
				<-done
				var result workflowengine.WorkflowResult
				require.Error(t, env.GetWorkflowResult(&result))
				require.Contains(t, env.GetWorkflowResult(&result).Error(), tc.errorCode.Code)
				require.Contains(
					t,
					env.GetWorkflowResult(&result).Error(),
					tc.errorCode.Description,
				)
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
			registerOpenIDNetLogPushActivity(env, func() { callCount++ })
			w := NewOpenID4VPWalletLogsWorkflow()
			env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
				Run(func(_ mock.Arguments) {
					callCount++
				}).
				Return(tc.mockResponse, nil)
			done := make(chan struct{})
			go func() {
				env.RegisterDelayedCallback(func() {
					env.SignalWorkflow(OpenID4VPWalletStartCheckSignal, nil)
				}, time.Second*30)
				if tc.expectedCancel {
					env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*45)
				}
				env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
					Payload: OpenID4VPWalletLogsWorkflowPayload{
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
					require.Equal(t, 2, callCount) // Logs poll and log push (no looping)
				}
			}
		})
	}
}

// registerOpenIDNetLogPushActivity mocks the openidnet realtime logs push,
// invoking onPush for each call.
func registerOpenIDNetLogPushActivity(env *testsuite.TestWorkflowEnvironment, onPush func()) {
	registerRealtimeLogsActivity(env)
	onRealtimeLogsActivity(env, testWorkflowID+OpenID4VPWalletSubscription, onPush)
}

func TestOpenID4VPWalletWorkflowStart(t *testing.T) {
	origStart := openID4VPWalletStartWorkflowWithOptions
	t.Cleanup(func() {
		openID4VPWalletStartWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string
	var capturedInput workflowengine.WorkflowInput

	openID4VPWalletStartWorkflowWithOptions = func(
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

	w := NewOpenID4VPWalletWorkflow()
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
	require.Equal(t, "ns-1", capturedInput.Config["namespace"])
	requireWorkflowLogsCapability(t, capturedInput, true)
	require.Equal(t, OpenID4VPWalletTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "OpenID4VPWalletCheckWorkflow"))
	require.Equal(t, 24*time.Hour, capturedOptions.WorkflowExecutionTimeout)
}

// runRunningOpenID4VPLogsWorkflow runs the logs workflow against a run that
// stays RUNNING with identical logs, applying signals before cancelling at
// cancelAt. It returns the number of log polls and realtime log pushes.
func runRunningOpenID4VPLogsWorkflow(
	t *testing.T,
	signals func(env *testsuite.TestWorkflowEnvironment),
	cancelAt time.Duration,
) (polls int, pushes int) {
	t.Helper()
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(httpActivity.Execute, activity.RegisterOptions{
		Name: httpActivity.Name(),
	})
	registerOpenIDNetLogPushActivity(env, func() { pushes++ })
	env.OnActivity(httpActivity.Name(), mock.Anything, mock.Anything).
		Run(func(_ mock.Arguments) { polls++ }).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"body": []map[string]any{{"result": "RUNNING"}},
		}}, nil)

	signals(env)
	env.RegisterDelayedCallback(env.CancelWorkflow, cancelAt)
	env.ExecuteWorkflow(NewOpenID4VPWalletLogsWorkflow().Workflow, workflowengine.WorkflowInput{
		Payload: OpenID4VPWalletLogsWorkflowPayload{Rid: "12345", Token: "test-token"},
		Config: map[string]any{
			"app_url":  "https://test-app.com",
			"interval": 10 * time.Second,
		},
	})
	require.True(t, env.IsWorkflowCompleted())
	return polls, pushes
}

func TestOpenID4VPLogsRestartWithinIntervalKeepsOneTimerChain(t *testing.T) {
	polls, _ := runRunningOpenID4VPLogsWorkflow(t, func(env *testsuite.TestWorkflowEnvironment) {
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(OpenID4VPWalletStartCheckSignal, nil)
		}, time.Second)
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(OpenID4VPWalletStopCheckSignal, nil)
		}, 2*time.Second)
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(OpenID4VPWalletStartCheckSignal, nil)
		}, 5*time.Second)
	}, 34*time.Second)

	// Polls at 1s and 5s (start signals), then 15s and 25s; a leaked timer
	// from the first start would add polls at 11s, 21s and 31s.
	require.Equal(t, 4, polls)
}

func TestOpenID4VPLogsSendsUnchangedLogsOnce(t *testing.T) {
	polls, pushes := runRunningOpenID4VPLogsWorkflow(
		t,
		func(env *testsuite.TestWorkflowEnvironment) {
			env.RegisterDelayedCallback(func() {
				env.SignalWorkflow(OpenID4VPWalletStartCheckSignal, nil)
			}, time.Second)
		},
		15*time.Second,
	)

	require.Equal(t, 2, polls)
	require.Equal(t, 1, pushes)
}
