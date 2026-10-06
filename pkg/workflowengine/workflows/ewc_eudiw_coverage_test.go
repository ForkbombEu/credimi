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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

const (
	covWEEWCStatusURL = "https://api.test/ewc/session-1"
	covWEEWCLogsURL   = "https://api.test/ewc/logs/session-1"
)

// covWEIsHTTPURL matches an HTTP activity input targeting url.
func covWEIsHTTPURL(url string) any {
	return mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
		payload, err := workflowengine.DecodePayload[activities.HTTPActivityPayload](input.Payload)
		return err == nil && payload.URL == url
	})
}

func covWERegisterEWCActivities(env *testsuite.TestWorkflowEnvironment) {
	stepCI := activities.NewStepCIWorkflowActivity()
	env.RegisterActivityWithOptions(stepCI.Execute, activity.RegisterOptions{Name: stepCI.Name()})
	mail := activities.NewSendMailActivity()
	env.RegisterActivityWithOptions(mail.Execute, activity.RegisterOptions{Name: mail.Name()})
	httpAct := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(httpAct.Execute, activity.RegisterOptions{Name: httpAct.Name()})
	registerRealtimeLogsActivity(env)
}

func TestCovWEEWCStatusWorkflowFailures(t *testing.T) {
	httpName := activities.NewHTTPActivity().Name()
	checkFailed := errorcodes.Codes[errorcodes.EWCCheckFailed].Code
	missingConfig := errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code
	validConfig := func() map[string]any {
		return map[string]any{
			"app_url":        "https://credimi.test",
			"check_endpoint": "https://api.test/ewc/{{ sessionId }}",
			"logs_endpoint":  "https://api.test/ewc/logs/{{ sessionId }}",
		}
	}
	statusBody := func(body any) workflowengine.ActivityResult {
		return workflowengine.ActivityResult{Output: map[string]any{"body": body}}
	}

	tests := []struct {
		name         string
		config       func() map[string]any
		setup        func(env *testsuite.TestWorkflowEnvironment)
		wantCode     string
		wantContains string
	}{
		{
			name: "missing check endpoint",
			config: func() map[string]any {
				c := validConfig()
				delete(c, "check_endpoint")
				return c
			},
			wantCode:     missingConfig,
			wantContains: "check_endpoint",
		},
		{
			name: "missing logs endpoint",
			config: func() map[string]any {
				c := validConfig()
				c["logs_endpoint"] = ""
				return c
			},
			wantCode:     missingConfig,
			wantContains: "logs_endpoint",
		},
		{
			name: "status request fails",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(covWEEWCStatusURL)).
					Return(workflowengine.ActivityResult{}, covWENonRetryable("status unreachable"))
			},
			wantContains: "status unreachable",
		},
		{
			name: "status body is not an object",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(covWEEWCStatusURL)).
					Return(statusBody("plain text"), nil)
			},
			wantCode: errorcodes.Codes[errorcodes.JSONUnmarshalFailed].Code,
		},
		{
			name: "logs request fails",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(covWEEWCStatusURL)).
					Return(statusBody(map[string]any{"status": "success"}), nil)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(covWEEWCLogsURL)).
					Return(workflowengine.ActivityResult{}, covWENonRetryable("logs unreachable"))
			},
			wantContains: "logs unreachable",
		},
		{
			name: "realtime log push fails",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(covWEEWCStatusURL)).
					Return(statusBody(map[string]any{"status": "success"}), nil)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(covWEEWCLogsURL)).
					Return(statusBody([]any{map[string]any{"message": "hi"}}), nil)
				env.OnActivity(activities.SendRealtimeLogsActivityName, mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, covWENonRetryable("broker down"))
			},
			wantContains: "broker down",
		},
		{
			name: "pending with a non-ok reason fails the check",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(covWEEWCStatusURL)).
					Return(statusBody(map[string]any{"status": "pending", "reason": "session expired"}), nil)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(covWEEWCLogsURL)).
					Return(statusBody(map[string]any{}), nil)
			},
			wantCode:     checkFailed,
			wantContains: "session expired",
		},
		{
			name: "unknown status fails the check",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(covWEEWCStatusURL)).
					Return(statusBody(map[string]any{"status": "exploded"}), nil)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(covWEEWCLogsURL)).
					Return(statusBody(map[string]any{}), nil)
			},
			wantCode:     checkFailed,
			wantContains: "unexpected status from 'https://api.test/ewc/{{ sessionId }}': exploded",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
			covWERegisterEWCActivities(env)
			if tc.setup != nil {
				tc.setup(env)
			}
			config := validConfig()
			if tc.config != nil {
				config = tc.config()
			}

			env.ExecuteWorkflow(NewEWCStatusWorkflow().Workflow, workflowengine.WorkflowInput{
				Payload: EWCStatusWorkflowPayload{SessionID: "session-1"},
				Config:  config,
			})

			require.True(t, env.IsWorkflowCompleted())
			err := env.GetWorkflowError()
			require.Error(t, err)
			if tc.wantCode != "" {
				assert.Contains(t, err.Error(), tc.wantCode)
			}
			assert.Contains(t, err.Error(), tc.wantContains)
		})
	}
}

func TestCovWEEWCStatusWorkflowStopAndPipelineCancelSignals(t *testing.T) {
	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	covWERegisterEWCActivities(env)
	httpName := activities.NewHTTPActivity().Name()
	var statusCalls atomic.Int32
	env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(covWEEWCStatusURL)).
		Run(func(mock.Arguments) { statusCalls.Add(1) }).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"body": map[string]any{"status": "pending", "reason": "ok"},
		}}, nil)
	env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(covWEEWCLogsURL)).
		Return(workflowengine.ActivityResult{Output: map[string]any{"body": map[string]any{}}}, nil)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(EwcStopCheckSignal, nil)
	}, 2500*time.Millisecond)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(PipelineCancelSignal, nil)
	}, 30*time.Second)

	env.ExecuteWorkflow(NewEWCStatusWorkflow().Workflow, workflowengine.WorkflowInput{
		Payload: EWCStatusWorkflowPayload{SessionID: "session-1"},
		Config: map[string]any{
			"app_url":        "https://credimi.test",
			"check_endpoint": "https://api.test/ewc/{{ sessionId }}",
			"logs_endpoint":  "https://api.test/ewc/logs/{{ sessionId }}",
			"interval":       float64(time.Second),
		},
	})

	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "canceled")
	// Polls at 1s and 2s; the stop signal at 2.5s halts polling until the cancel.
	assert.Equal(t, int32(2), statusCalls.Load())
}

func TestCovWEEWCWorkflowRejectsMissingConfiguration(t *testing.T) {
	missingConfig := errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code
	validConfig := func() map[string]any {
		return map[string]any{
			"app_url":        "https://credimi.test",
			"template":       "tpl",
			"check_endpoint": "https://api.test/ewc",
			"logs_endpoint":  "https://api.test/ewc/logs/{{ sessionId }}",
			"namespace":      "ns",
			"app_name":       "Credimi",
			"app_logo":       "https://logo",
			"user_name":      "User",
			"memo":           map[string]any{"author": "ewc"},
		}
	}
	tests := []struct {
		name     string
		payload  any
		mutate   func(map[string]any)
		captures map[string]any
		wantCode string
		wantText string
	}{
		{
			name:     "payload without user mail",
			payload:  EWCWorkflowPayload{},
			wantCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
		},
		{
			name:     "missing template",
			mutate:   func(c map[string]any) { delete(c, "template") },
			wantCode: missingConfig,
			wantText: "template",
		},
		{
			name:     "missing app url",
			mutate:   func(c map[string]any) { c["app_url"] = "" },
			wantCode: missingConfig,
			wantText: "app_url",
		},
		{
			name:     "missing check endpoint",
			mutate:   func(c map[string]any) { delete(c, "check_endpoint") },
			wantCode: missingConfig,
			wantText: "check_endpoint",
		},
		{
			name:     "missing logs endpoint",
			mutate:   func(c map[string]any) { delete(c, "logs_endpoint") },
			wantCode: missingConfig,
			wantText: "logs_endpoint",
		},
		{
			name:     "memo without author",
			mutate:   func(c map[string]any) { c["memo"] = map[string]any{} },
			wantCode: missingConfig,
			wantText: "author",
		},
		{
			name:     "StepCI captures without session id",
			captures: map[string]any{"deeplink": "openid://x"},
			wantCode: errorcodes.Codes[errorcodes.UnexpectedStepCIOutput].Code,
			wantText: "session_id",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
			covWERegisterEWCActivities(env)
			if tc.captures != nil {
				env.OnActivity(activities.NewStepCIWorkflowActivity().Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{
						Output: map[string]any{"captures": tc.captures},
					}, nil)
			}
			config := validConfig()
			if tc.mutate != nil {
				tc.mutate(config)
			}
			payload := tc.payload
			if payload == nil {
				payload = EWCWorkflowPayload{UserMail: "user@example.org"}
			}

			env.ExecuteWorkflow(NewEWCWorkflow().Workflow, workflowengine.WorkflowInput{
				Payload: payload,
				Config:  config,
			})

			require.True(t, env.IsWorkflowCompleted())
			err := env.GetWorkflowError()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantCode)
			assert.Contains(t, err.Error(), tc.wantText)
		})
	}
}

func TestCovWEEudiwWorkflowFailures(t *testing.T) {
	stepCIName := activities.NewStepCIWorkflowActivity().Name()
	mailName := activities.NewSendMailActivity().Name()
	httpName := activities.NewHTTPActivity().Name()
	statusURL := "https://verifier-backend.eudiw.dev/ui/presentations/tx-1"
	eventsURL := statusURL + "/events"
	stepCIOutput := errorcodes.Codes[errorcodes.UnexpectedStepCIOutput].Code
	unexpectedHTTP := errorcodes.Codes[errorcodes.UnexpectedHTTPResponse].Code
	validCaptures := map[string]any{
		"client_id":      "client",
		"request_uri":    "https://request",
		"transaction_id": "tx-1",
	}
	captures := func(drop string) workflowengine.ActivityResult {
		c := map[string]any{}
		for k, v := range validCaptures {
			if k != drop {
				c[k] = v
			}
		}
		return workflowengine.ActivityResult{Output: map[string]any{"captures": c}}
	}
	upToPolling := func(env *testsuite.TestWorkflowEnvironment) {
		env.OnActivity(stepCIName, mock.Anything, mock.Anything).Return(captures(""), nil)
		env.OnActivity(mailName, mock.Anything, mock.Anything).
			Return(workflowengine.ActivityResult{}, nil)
	}
	events := workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{"events": []any{map[string]any{"event": "e"}}},
	}}

	tests := []struct {
		name     string
		payload  any
		config   map[string]any
		setup    func(env *testsuite.TestWorkflowEnvironment)
		wantCode string
		wantText string
	}{
		{
			name:     "payload without user mail",
			payload:  EudiwWorkflowPayload{ID: "1", Nonce: "n"},
			wantCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
		},
		{
			name:     "missing template",
			config:   map[string]any{"app_url": "https://credimi.test", "namespace": "ns"},
			wantCode: errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
			wantText: "template",
		},
		{
			name: "StepCI fails",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(stepCIName, mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, covWENonRetryable("stepci broke"))
			},
			wantText: "stepci broke",
		},
		{
			name: "StepCI output without captures",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(stepCIName, mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{}}, nil)
			},
			wantCode: stepCIOutput,
		},
		{
			name: "captures without client id",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(stepCIName, mock.Anything, mock.Anything).
					Return(captures("client_id"), nil)
			},
			wantCode: stepCIOutput,
			wantText: "client_id",
		},
		{
			name: "captures without request uri",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(stepCIName, mock.Anything, mock.Anything).
					Return(captures("request_uri"), nil)
			},
			wantCode: stepCIOutput,
			wantText: "request_uri",
		},
		{
			name: "captures without transaction id",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(stepCIName, mock.Anything, mock.Anything).
					Return(captures("transaction_id"), nil)
			},
			wantCode: stepCIOutput,
			wantText: "transaction_id",
		},
		{
			name: "verification mail fails",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(stepCIName, mock.Anything, mock.Anything).Return(captures(""), nil)
				env.OnActivity(mailName, mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, covWENonRetryable("smtp down"))
			},
			wantText: "smtp down",
		},
		{
			name: "status poll fails",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				upToPolling(env)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(statusURL)).
					Return(workflowengine.ActivityResult{}, covWENonRetryable("verifier down"))
			},
			wantText: "verifier down",
		},
		{
			name: "status output is not a map",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				upToPolling(env)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(statusURL)).
					Return(workflowengine.ActivityResult{Output: "nope"}, nil)
			},
			wantCode: unexpectedHTTP,
			wantText: "unexpected output type: string",
		},
		{
			name: "status output without status code",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				upToPolling(env)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(statusURL)).
					Return(workflowengine.ActivityResult{Output: map[string]any{"body": "x"}}, nil)
			},
			wantCode: unexpectedHTTP,
			wantText: "missing or invalid status code",
		},
		{
			name: "events fetch fails",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				upToPolling(env)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(statusURL)).
					Return(workflowengine.ActivityResult{Output: map[string]any{"status": 200}}, nil)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(eventsURL)).
					Return(workflowengine.ActivityResult{}, covWENonRetryable("events down"))
			},
			wantText: "events down",
		},
		{
			name: "realtime log push fails",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				upToPolling(env)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(statusURL)).
					Return(workflowengine.ActivityResult{Output: map[string]any{"status": 200}}, nil)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(eventsURL)).
					Return(events, nil)
				env.OnActivity(activities.SendRealtimeLogsActivityName, mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, covWENonRetryable("broker down"))
			},
			wantText: "broker down",
		},
		{
			name: "unexpected status code fails the check",
			setup: func(env *testsuite.TestWorkflowEnvironment) {
				upToPolling(env)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(statusURL)).
					Return(workflowengine.ActivityResult{Output: map[string]any{"status": 302}}, nil)
				env.OnActivity(httpName, mock.Anything, covWEIsHTTPURL(eventsURL)).
					Return(events, nil)
				onRealtimeLogsActivity(env, testWorkflowID+EudiwSubscription, func() {})
			},
			wantCode: errorcodes.Codes[errorcodes.EudiwCheckFailed].Code,
			wantText: "unexpected status code: 302",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
			covWERegisterEWCActivities(env)
			if tc.setup != nil {
				tc.setup(env)
			}
			payload := tc.payload
			if payload == nil {
				payload = EudiwWorkflowPayload{ID: "1", Nonce: "n", UserMail: "u@example.org"}
			}
			config := tc.config
			if config == nil {
				config = map[string]any{
					"app_url":   "https://credimi.test",
					"template":  "tpl",
					"namespace": "ns",
				}
			}
			env.RegisterDelayedCallback(func() {
				env.SignalWorkflow(EudiwStartCheckSignal, nil)
			}, time.Second)

			env.ExecuteWorkflow(NewEudiwWorkflow().Workflow, workflowengine.WorkflowInput{
				Payload: payload,
				Config:  config,
			})

			require.True(t, env.IsWorkflowCompleted())
			err := env.GetWorkflowError()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantCode)
			assert.Contains(t, err.Error(), tc.wantText)
			assert.False(t, strings.Contains(err.Error(), "panic"))
		})
	}
}
