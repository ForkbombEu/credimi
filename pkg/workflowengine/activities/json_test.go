// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

type DummyStruct struct {
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

func TestParseJSONActivity_Execute(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()

	activity := NewJSONActivity(
		map[string]reflect.Type{
			"DummyStruct": reflect.TypeOf(DummyStruct{}),
		},
	)
	env.RegisterActivity(activity.Execute)

	tests := []struct {
		name           string
		rawJSON        string
		expectErr      bool
		expectedErrMsg errorcodes.Code
		expectValue    DummyStruct
	}{
		{
			name:    "Success - valid JSON",
			rawJSON: `{"name":"Alice","age":30,"email":"alice@example.com"}`,
			expectValue: DummyStruct{
				Name:  "Alice",
				Age:   30,
				Email: "alice@example.com",
			},
		},
		{
			name:           "Failure - missing rawJSON",
			expectErr:      true,
			expectedErrMsg: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name:           "Failure - malformed JSON",
			rawJSON:        `{"name": "Charlie", "age": "oops"`,
			expectErr:      true,
			expectedErrMsg: errorcodes.Codes[errorcodes.JSONUnmarshalFailed],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := JSONActivityPayload{
				StructType: "DummyStruct",
			}
			if tt.rawJSON != "" {
				payload.RawJSON = tt.rawJSON
			}

			input := workflowengine.ActivityInput{
				Payload: payload,
			}

			future, err := env.ExecuteActivity(activity.Execute, input)

			if tt.expectErr {
				require.Error(t, err)
				if tt.expectedErrMsg != (errorcodes.Code{}) {
					require.Contains(t, err.Error(), tt.expectedErrMsg.Code)
					require.Contains(t, err.Error(), tt.expectedErrMsg.Description)
				}
			} else {
				require.NoError(t, err)
				var result workflowengine.ActivityResult
				err := future.Get(&result)
				require.NoError(t, err)
				jsonBytes, err := json.Marshal(result.Output)
				require.NoError(t, err)
				var actual DummyStruct
				json.Unmarshal(jsonBytes, &actual)
				require.Equal(t, tt.expectValue, actual)
			}
		})
	}
}
