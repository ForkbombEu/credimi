// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"strings"
	"sync/atomic"
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
)

func Test_EWCWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		mockActivities func(env *testsuite.TestWorkflowEnvironment, callCount *atomic.Int32)
		expectRunning  bool
		expectedErr    bool
		errorCode      errorcodes.Code
	}{
		{
			name: "Workflow completes when status is success",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment, callCount *atomic.Int32) {
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
						callCount.Add(1)
					}).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]string{"status": "success"},
					}}, nil)
			},
			expectRunning: false,
		},
		{
			name: "Workflow loops when status is pending",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment, callCount *atomic.Int32) {
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
						callCount.Add(1)
					}).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"body": map[string]string{"status": "pending", "reason": "ok"},
					}}, nil)
			},
			expectRunning: true,
		},
		{
			name: "Workflow fails when status is failed",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment, callCount *atomic.Int32) {
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
						callCount.Add(1)
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

			var callCount atomic.Int32
			w := NewEWCWorkflow()
			tc.mockActivities(env, &callCount)
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
					require.Greater(
						t,
						callCount.Load(),
						int32(1),
					) // Expecting multiple activity calls
				} else {
					<-done
					var result workflowengine.WorkflowResult
					require.NoError(t, env.GetWorkflowResult(&result))
					require.Equal(t, int32(1), callCount.Load()) // Only two activity call (no looping)
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
		name           string
		mockResponse   workflowengine.ActivityResult
		expectRunning  bool
		expectedErr    bool
		expectedCancel bool
		expectedLog    map[string]any
	}{
		{
			name: "Workflow completes when status is success",
			mockResponse: workflowengine.ActivityResult{Output: map[string]any{
				"body": map[string]any{
					"status": "success",
					"claims": map[string]any{
						"claim1": "value-1",
						"claim2": "value-2",
					},
				},
			}},
			expectRunning: false,
			expectedErr:   false,
			expectedLog: map[string]any{
				"claim1": "value-1",
				"claim2": "value-2",
			},
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
		{
			name: "Workflow stops when pipeline cancel signal is received",
			mockResponse: workflowengine.ActivityResult{Output: map[string]any{
				"body": map[string]any{
					"status": "pending",
					"reason": "ok",
				},
			}},
			expectRunning:  false,
			expectedErr:    true,
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

			env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
				Run(func(_ mock.Arguments) {
					callCount++
				}).
				Return(tc.mockResponse, nil)

			w := NewEWCStatusWorkflow()

			done := make(chan struct{})
			go func() {
				env.RegisterDelayedCallback(func() {
					env.SignalWorkflow(EwcStartCheckSignal, nil)
				}, time.Second*30)

				if tc.expectedCancel {
					env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*45)
				}

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
					err := env.GetWorkflowResult(&result)
					require.Error(t, err)

					if tc.expectedCancel {
						require.Contains(t, err.Error(), "canceled")
					}
				} else {
					require.NoError(t, env.GetWorkflowResult(&result))
					require.NotEmpty(t, result.Message)
					if tc.expectedLog != nil {
						require.Equal(t, tc.expectedLog, result.Log)
					}
				}
				require.GreaterOrEqual(t, callCount, 1)
			}
		})
	}
}

func TestEWCWorkflowStart(t *testing.T) {
	origStart := ewcStartWorkflowWithOptions
	t.Cleanup(func() {
		ewcStartWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string
	var capturedInput workflowengine.WorkflowInput

	ewcStartWorkflowWithOptions = func(
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

	w := NewEWCWorkflow()
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
	require.Equal(t, EWCTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "EWCWorkflow"))
	require.Equal(t, 24*time.Hour, capturedOptions.WorkflowExecutionTimeout)
}

func TestResolveEWCLikeCheckEndpoint(t *testing.T) {
	testCases := []struct {
		name      string
		suite     string
		standard  string
		want      string
		expectErr bool
	}{
		{
			name:     "ewc openid4vp endpoint",
			suite:    EWCSuite,
			standard: "openid4vp_wallet",
			want:     "https://ewc.api.forkbomb.eu/verificationStatus",
		},
		{
			name:     "ewc openid4vci endpoint",
			suite:    EWCSuite,
			standard: "openid4vci_wallet",
			want:     "https://ewc.api.forkbomb.eu/issueStatus",
		},
		{
			name:     "webuild openid4vp endpoint",
			suite:    WebuildSuite,
			standard: "openid4vp_wallet",
			want:     "https://webuild.api.forkbomb.eu/verificationStatus",
		},
		{
			name:     "webuild openid4vci endpoint",
			suite:    WebuildSuite,
			standard: "openid4vci_wallet",
			want:     "https://webuild.api.forkbomb.eu/issueStatus",
		},
		{
			name:      "unsupported suite fails",
			suite:     "invalid",
			standard:  "openid4vp_wallet",
			expectErr: true,
		},
		{
			name:      "unsupported standard fails",
			suite:     EWCSuite,
			standard:  "invalid",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveEWCLikeCheckEndpoint(tc.suite, tc.standard)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
