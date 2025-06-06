// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestSchemaValidationActivity_Execute(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()

	act := NewSchemaValidationActivity()
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{
		Name: act.Name(),
	})

	tests := []struct {
		name            string
		payload         map[string]any
		expectErr       bool
		expectedErrCode errorcodes.Code
		expectedErrMsg  string
	}{
		{
			name: "Success - valid schema and data",
			payload: map[string]any{
				"schema": `{
					"type": "object",
					"properties": {
						"name": { "type": "string" }
					},
					"required": ["name"]
				}`,
				"data": map[string]any{
					"name": "Credimi",
				},
			},
			expectErr: false,
		},
		{
			name: "Success - valid schema with subschema",
			payload: map[string]any{
				"schema": `{
					"$schema": "http://json-schema.org/draft-07/schema#",
					"type": "object",
					"properties": {
						"person": { "$ref": "subschema1.json" }
					},
					"required": ["person"]
				}`,
				"subschema": []any{
					`{
						"$id": "person.json",
						"type": "object",
						"properties": {
							"name": { "type": "string" },
							"age": { "type": "integer" }
						},
						"required": ["name", "age"]
					}`,
				},
				"data": map[string]any{
					"person": map[string]any{
						"name": "Alice",
						"age":  30,
					},
				},
			},
			expectErr: false,
		},
		{
			name: "Failure - missing schema",
			payload: map[string]any{
				"data": map[string]any{
					"name": "Credimi",
				},
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "Failure - schema not a string",
			payload: map[string]any{
				"schema": 123,
				"data": map[string]any{
					"name": "Credimi",
				},
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "Failure - missing data",
			payload: map[string]any{
				"schema": `{"type":"object"}`,
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "Failure - data not a map",
			payload: map[string]any{
				"schema": `{"type":"object"}`,
				"data":   "string-not-map",
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "Failure - invalid schema JSON",
			payload: map[string]any{
				"schema": `{"type":`,
				"data": map[string]any{
					"name": "Credimi",
				},
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.JSONUnmarshalFailed],
		},
		{
			name: "Failure - schema validation fails",
			payload: map[string]any{
				"schema": `{
					"type": "object",
					"properties": {
						"age": { "type": "integer" }
					},
					"required": ["age"]
				}`,
				"data": map[string]any{
					"age": "not-an-integer",
				},
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.SchemaValidationFailed],
			expectedErrMsg:  "jsonschema validation failed with 'file:///schema.json#'\n- at '/age' [S#/properties/age/type]: got string, want integer",
		},
		{
			name: "Failure - invalid subschema type",
			payload: map[string]any{
				"schema":    `{"type":"object"}`,
				"data":      map[string]any{},
				"subschema": 123, // invalid
			},
			expectErr:       true,
			expectedErrCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := workflowengine.ActivityInput{
				Payload: tt.payload,
			}
			future, err := env.ExecuteActivity(act.Execute, input)

			if tt.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrCode.Code)
				if tt.expectedErrMsg == "" {
					require.Contains(t, err.Error(), tt.expectedErrCode.Description)
				}
				require.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
				var result workflowengine.ActivityResult
				require.NoError(t, future.Get(&result))
			}
		})
	}
}
