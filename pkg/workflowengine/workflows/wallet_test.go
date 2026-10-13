// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"reflect"
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
)

func Test_WalletWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		payload        WalletWorkflowPayload
		config         map[string]any
		expectError    bool
		errorCode      errorcodes.Code
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
	}{
		{
			name: "Valid Workflow Run (Google URL)",
			payload: WalletWorkflowPayload{
				URL: "https://com.example.wallet",
			},
			config: map[string]any{
				"namespace": "namespace",
				"app_url":   "http://app.example.com",
			},
			expectError: false,
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				parseActivity := activities.NewParseWalletURLActivity()
				dockerActivity := activities.NewDockerActivity()
				jsonActivity := activities.NewJSONActivity(map[string]reflect.Type{
					"map": reflect.TypeOf(map[string]any{}),
				})

				env.RegisterActivityWithOptions(parseActivity.Execute, activity.RegisterOptions{
					Name: parseActivity.Name(),
				})
				env.RegisterActivityWithOptions(dockerActivity.Execute, activity.RegisterOptions{
					Name: dockerActivity.Name(),
				})
				env.RegisterActivityWithOptions(jsonActivity.Execute, activity.RegisterOptions{
					Name: jsonActivity.Name(),
				})

				testdata := `{"test": "test", "id": "A12345"}`
				env.OnActivity(parseActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"api_input": "http://example.com", "store_type": "google"}}, nil)
				env.OnActivity(dockerActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"stdout": testdata}}, nil)
				env.OnActivity(jsonActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"status": 200, "body": "test_result"}}, nil)
			},
		},
		{
			name: "Valid Workflow Run (Apple URL)",
			payload: WalletWorkflowPayload{
				URL: "https://com.example.wallet",
			},
			config: map[string]any{
				"namespace": "namespace",
				"app_url":   "http://app.example.com",
			},
			expectError: false,
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				HTTPActivity := activities.NewHTTPActivity()
				parseActivity := activities.NewParseWalletURLActivity()
				env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
					Name: HTTPActivity.Name(),
				})
				env.RegisterActivityWithOptions(parseActivity.Execute, activity.RegisterOptions{
					Name: parseActivity.Name(),
				})

				testdata := []map[string]any{{"test": "test", "id": "A12345"}}
				env.OnActivity(parseActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"api_input": "http://example.com", "store_type": "apple"}}, nil)
				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"status": 200, "body": map[string]any{"results": testdata}}}, nil)
			},
		},
		{
			name: "Invalid Workflow Run",
			payload: WalletWorkflowPayload{
				URL: "https://com.example.wallet",
			},
			config: map[string]any{
				"namespace": "namespace",
				"app_url":   "http://app.example.com",
			},
			expectError: true,
			errorCode:   errorcodes.Codes[errorcodes.ParseURLFailed],
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				HTTPActivity := activities.NewHTTPActivity()
				parseActivity := activities.NewParseWalletURLActivity()
				dockerActivity := activities.NewDockerActivity()
				jsonActivity := activities.NewJSONActivity(map[string]reflect.Type{
					"map": reflect.TypeOf(map[string]any{}),
				})

				env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
					Name: HTTPActivity.Name(),
				})
				env.RegisterActivityWithOptions(parseActivity.Execute, activity.RegisterOptions{
					Name: parseActivity.Name(),
				})
				env.RegisterActivityWithOptions(dockerActivity.Execute, activity.RegisterOptions{
					Name: dockerActivity.Name(),
				})
				env.RegisterActivityWithOptions(dockerActivity.Execute, activity.RegisterOptions{
					Name: jsonActivity.Name(),
				})
				env.OnActivity(parseActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, workflowengine.NewAppError(workflowengine.WorkflowError{Code: errorcodes.Codes[errorcodes.ParseURLFailed].Code, Summary: errorcodes.Codes[errorcodes.ParseURLFailed].Description, Message: ""}))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			tc.mockActivities(env)

			w := NewWalletWorkflow()
			env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
				Payload: tc.payload,
				Config:  tc.config,
			})

			if tc.expectError {
				var result workflowengine.WorkflowResult
				require.Error(t, env.GetWorkflowResult(&result))
				require.Contains(t, env.GetWorkflowResult(&result).Error(), tc.errorCode.Code)
				require.Contains(
					t,
					env.GetWorkflowResult(&result).Error(),
					tc.errorCode.Description,
				)
			} else {
				var result workflowengine.WorkflowResult
				require.NoError(t, env.GetWorkflowResult(&result))
			}
		})
	}
}

func TestWalletWorkflowAppMetadataUpdate(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	parseActivity := activities.NewParseWalletURLActivity()
	env.RegisterActivityWithOptions(httpActivity.Execute, activity.RegisterOptions{
		Name: httpActivity.Name(),
	})
	env.RegisterActivityWithOptions(parseActivity.Execute, activity.RegisterOptions{
		Name: parseActivity.Name(),
	})
	// The parse step takes a while, so the update arrives before the metadata
	// is ready and must wait for it.
	env.OnActivity(parseActivity.Name(), mock.Anything, mock.Anything).
		After(time.Minute).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"api_input":  "123",
			"store_type": "apple",
		}}, nil)
	env.OnActivity(httpActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"body": map[string]any{"results": []map[string]any{{"trackName": "Wallet"}}},
		}}, nil)

	var updateResult map[string]any
	var updateErr error
	completed := false
	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflow(AppMetadataUpdate, "metadata-1", &testsuite.TestUpdateCallback{
			OnAccept: func() {},
			OnReject: func(err error) { updateErr = err },
			OnComplete: func(result any, err error) {
				completed = true
				updateErr = err
				if m, ok := result.(map[string]any); ok {
					updateResult = m
				}
			},
		})
	}, time.Second)

	env.ExecuteWorkflow(NewWalletWorkflow().Workflow, workflowengine.WorkflowInput{
		Payload: WalletWorkflowPayload{URL: "https://apps.apple.com/app/id123"},
		Config:  map[string]any{"namespace": "namespace", "app_url": "http://app.example.com"},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.True(t, completed)
	require.NoError(t, updateErr)
	require.Equal(t, map[string]any{
		"metadata":  map[string]any{"trackName": "Wallet"},
		"storeType": "apple",
	}, updateResult)
}

func TestWalletWorkflowStart(t *testing.T) {
	origStart := walletStartWorkflowWithOptions
	t.Cleanup(func() {
		walletStartWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string
	var capturedInput workflowengine.WorkflowInput

	walletStartWorkflowWithOptions = func(
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

	w := NewWalletWorkflow()
	input := workflowengine.WorkflowInput{
		Payload: WalletWorkflowPayload{URL: "https://example.com"},
	}
	result, err := w.Start("ns-1", input)
	require.NoError(t, err)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, "run-1", result.WorkflowRunID)
	require.Equal(t, "ns-1", capturedNamespace)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, input, capturedInput)
	require.Equal(t, WalletTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "Wallet-Workflow-"))
	require.Equal(t, 24*time.Hour, capturedOptions.WorkflowExecutionTimeout)
}
