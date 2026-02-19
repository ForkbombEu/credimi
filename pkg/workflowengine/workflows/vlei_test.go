// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"encoding/json"
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

func Test_VLEIValidationWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		payload        VLEIValidationWorkflowPayload
		config         map[string]any
		expectError    bool
		errorCode      errorcodes.Code
		expectedMsg    string
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
	}{
		{
			name: "Valid Workflow Run",
			payload: VLEIValidationWorkflowPayload{
				CredentialID: "12345",
			},
			config: map[string]any{
				"server_url": "http://example.com",
				"app_url":    "http://app.example.com",
			},
			expectError: false,
			expectedMsg: "validated for credential: '12345'",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				HTTPActivity := activities.NewHTTPActivity()
				parseActivity := activities.NewCESRParsingActivity()
				validateActivity := activities.NewCESRValidateActivity()

				env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
					Name: HTTPActivity.Name(),
				})
				env.RegisterActivityWithOptions(parseActivity.Execute, activity.RegisterOptions{
					Name: parseActivity.Name(),
				})
				env.RegisterActivityWithOptions(validateActivity.Execute, activity.RegisterOptions{
					Name: validateActivity.Name(),
				})

				var parsed map[string]any
				_ = json.Unmarshal([]byte(`[{"test": "test", "v": "A12345"}]`), &parsed)

				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"status": 200, "body": "test_result"}}, nil)
				env.OnActivity(parseActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: parsed}, nil)
				env.OnActivity(validateActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: "validated"}, nil)
			},
		},
		{
			name: "Invalid Workflow Run",
			payload: VLEIValidationWorkflowPayload{
				CredentialID: "12345",
			},
			config: map[string]any{
				"server_url": "http://example.com",
				"app_url":    "http://app.example.com",
			},
			expectError: true,
			errorCode:   errorcodes.Codes[errorcodes.SchemaValidationFailed],
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				HTTPActivity := activities.NewHTTPActivity()
				parseActivity := activities.NewCESRParsingActivity()
				validateActivity := activities.NewCESRValidateActivity()

				env.RegisterActivityWithOptions(HTTPActivity.Execute, activity.RegisterOptions{
					Name: HTTPActivity.Name(),
				})
				env.RegisterActivityWithOptions(parseActivity.Execute, activity.RegisterOptions{
					Name: parseActivity.Name(),
				})
				env.RegisterActivityWithOptions(validateActivity.Execute, activity.RegisterOptions{
					Name: validateActivity.Name(),
				})

				var parsed map[string]any
				_ = json.Unmarshal([]byte(`[{"test": "test", "v": "A12345"}]`), &parsed)

				env.OnActivity(HTTPActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{"status": 200, "body": "test_result"}}, nil)
				env.OnActivity(parseActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: parsed}, nil)
				env.OnActivity(validateActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, workflowengine.NewAppError(errorcodes.Codes[errorcodes.SchemaValidationFailed], ""))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			tc.mockActivities(env)

			w := NewVLEIValidationWorkflow()
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
				require.Equal(t, tc.expectedMsg, result.Message)
			}
		})
	}
}

func Test_VLEIValidationLocalWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		payload        VLEIValidationLocalWorkflowPayload
		config         map[string]any
		schema         string
		expectError    bool
		errorCode      errorcodes.Code
		expectedMsg    string
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
	}{
		{
			name: "Valid Workflow Run",
			payload: VLEIValidationLocalWorkflowPayload{
				CESR: `[{"test": "test", "v": "A12345"}]`,
			},
			config: map[string]any{
				"app_url": "http://app.example.com",
			},
			expectError: false,
			expectedMsg: "validated from file",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				parseActivity := activities.NewCESRParsingActivity()
				validateActivity := activities.NewCESRValidateActivity()
				env.RegisterActivityWithOptions(parseActivity.Execute, activity.RegisterOptions{
					Name: parseActivity.Name(),
				})
				env.RegisterActivityWithOptions(validateActivity.Execute, activity.RegisterOptions{
					Name: validateActivity.Name(),
				})

				var parsed map[string]any
				_ = json.Unmarshal([]byte(`[{"test": "test", "v": "A12345"}]`), &parsed)
				env.OnActivity(parseActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: parsed}, nil)
				env.OnActivity(validateActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: "validated from file"}, nil)
			},
		},
		{
			name: "Invalid Workflow Run",
			payload: VLEIValidationLocalWorkflowPayload{
				CESR: `[{"test": "test", "v": "A12345"}]`,
			},
			config: map[string]any{
				"server_url": "http://example.com",
				"app_url":    "http://app.example.com",
			},
			expectError: true,
			errorCode:   errorcodes.Codes[errorcodes.SchemaValidationFailed],
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				parseActivity := activities.NewCESRParsingActivity()
				validateActivity := activities.NewCESRValidateActivity()
				env.RegisterActivityWithOptions(parseActivity.Execute, activity.RegisterOptions{
					Name: parseActivity.Name(),
				})
				env.RegisterActivityWithOptions(validateActivity.Execute, activity.RegisterOptions{
					Name: validateActivity.Name(),
				})

				var parsed map[string]any
				_ = json.Unmarshal([]byte(`[{"test": "test", "v": "A12345"}]`), &parsed)
				env.OnActivity(parseActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: parsed}, nil)
				env.OnActivity(validateActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, workflowengine.NewAppError(errorcodes.Codes[errorcodes.SchemaValidationFailed], ""))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			tc.mockActivities(env)

			w := NewVLEIValidationLocalWorkflow()
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
				require.Equal(t, tc.expectedMsg, result.Message)
			}
		})
	}
}

func TestVLEIValidationWorkflowStart(t *testing.T) {
	origStart := vleiStartWorkflowWithOptions
	t.Cleanup(func() {
		vleiStartWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string
	var capturedInput workflowengine.WorkflowInput

	vleiStartWorkflowWithOptions = func(
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

	w := NewVLEIValidationWorkflow()
	input := workflowengine.WorkflowInput{
		Payload: VLEIValidationWorkflowPayload{CredentialID: "cred-1"},
	}
	result, err := w.Start("ns-1", input)
	require.NoError(t, err)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, "run-1", result.WorkflowRunID)
	require.Equal(t, "ns-1", capturedNamespace)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, input, capturedInput)
	require.Equal(t, VLEIValidationTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "VLEIValidation-"))
	require.Equal(t, 24*time.Hour, capturedOptions.WorkflowExecutionTimeout)
}

func TestVLEIValidationLocalWorkflowStart(t *testing.T) {
	origStart := vleiStartWorkflowWithOptions
	t.Cleanup(func() {
		vleiStartWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string
	var capturedInput workflowengine.WorkflowInput

	vleiStartWorkflowWithOptions = func(
		namespace string,
		options client.StartWorkflowOptions,
		name string,
		input workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedOptions = options
		capturedName = name
		capturedInput = input
		return workflowengine.WorkflowResult{WorkflowID: "wf-2", WorkflowRunID: "run-2"}, nil
	}

	w := NewVLEIValidationLocalWorkflow()
	input := workflowengine.WorkflowInput{
		Payload: VLEIValidationLocalWorkflowPayload{CESR: "test"},
	}
	result, err := w.Start("ns-2", input)
	require.NoError(t, err)
	require.Equal(t, "wf-2", result.WorkflowID)
	require.Equal(t, "run-2", result.WorkflowRunID)
	require.Equal(t, "ns-2", capturedNamespace)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, input, capturedInput)
	require.Equal(t, VLEIValidationLocalTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "VLEILocalValidation-"))
	require.Equal(t, 24*time.Hour, capturedOptions.WorkflowExecutionTimeout)
}
