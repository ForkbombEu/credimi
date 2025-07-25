// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"encoding/json"
	"testing"

	"reflect"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func Test_VLEIValidationWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		rawJSON        string
		schema         string
		expectError    bool
		errorCode      errorcodes.Code
		expectedMsg    string
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
	}{
		{
			name: "Valid JSON matches schema",
			rawJSON: `{
				"name": "Alice",
				"age": 30
			}`,
			schema: `{
				"type": "object",
				"properties": {
					"name": { "type": "string" },
					"age": { "type": "number" }
				},
				"required": ["name", "age"]
			}`,
			expectError: false,
			expectedMsg: "vLEI is valid according to the schema for test",
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				jsonActivity := activities.NewJSONActivity(map[string]reflect.Type{
					"map": reflect.TypeOf(map[string]any{}),
				})
				validateActivity := activities.NewSchemaValidationActivity()

				env.RegisterActivityWithOptions(jsonActivity.Execute, activity.RegisterOptions{
					Name: jsonActivity.Name(),
				})
				env.RegisterActivityWithOptions(validateActivity.Execute, activity.RegisterOptions{
					Name: validateActivity.Name(),
				})

				var parsed map[string]any
				_ = json.Unmarshal([]byte(`{"name": "Alice", "age": 30}`), &parsed)

				env.OnActivity(jsonActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: parsed}, nil)

				env.OnActivity(validateActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: map[string]any{
						"status":  "success",
						"message": "JSON is valid against the schema",
					}}, nil)
			},
		},
		{
			name: "Invalid JSON fails schema validation",
			rawJSON: `{
				"name": "Alice"
			}`,
			schema: `{
				"type": "object",
				"properties": {
					"name": { "type": "string" },
					"age": { "type": "number" }
				},
				"required": ["name", "age"]
			}`,
			expectError: true,
			errorCode:   errorcodes.Codes[errorcodes.SchemaValidationFailed],
			mockActivities: func(env *testsuite.TestWorkflowEnvironment) {
				jsonActivity := activities.NewJSONActivity(map[string]reflect.Type{
					"map": reflect.TypeOf(map[string]any{}),
				})
				validateActivity := activities.NewSchemaValidationActivity()

				env.RegisterActivityWithOptions(jsonActivity.Execute, activity.RegisterOptions{
					Name: jsonActivity.Name(),
				})
				env.RegisterActivityWithOptions(validateActivity.Execute, activity.RegisterOptions{
					Name: validateActivity.Name(),
				})

				var parsed map[string]any
				_ = json.Unmarshal([]byte(`{"name": "Alice"}`), &parsed)

				env.OnActivity(jsonActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{Output: parsed}, nil)

				env.OnActivity(validateActivity.Name(), mock.Anything, mock.Anything).
					Return(workflowengine.ActivityResult{}, workflowengine.NewAppError(errorcodes.Codes[errorcodes.SchemaValidationFailed], "Missing required field: age"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()
			tc.mockActivities(env)

			w := &VLEIValidationWorkflow{}
			env.ExecuteWorkflow(w.Workflow, workflowengine.WorkflowInput{
				Config: map[string]any{
					"schema": tc.schema,
				},
				Payload: map[string]any{
					"rawJSON":   tc.rawJSON,
					"vLEI_type": "test",
				},
			})

			if tc.expectError {
				var result workflowengine.WorkflowResult
				require.Error(t, env.GetWorkflowResult(&result))
				require.Contains(t, env.GetWorkflowResult(&result).Error(), tc.errorCode.Code)
				require.Contains(t, env.GetWorkflowResult(&result).Error(), tc.errorCode.Description)
			} else {
				var result workflowengine.WorkflowResult
				require.NoError(t, env.GetWorkflowResult(&result))
				require.Equal(t, tc.expectedMsg, result.Message)
			}
		})
	}
}
