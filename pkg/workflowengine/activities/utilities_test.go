// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"os"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestCheckFileExistsActivity_Execute(t *testing.T) {
	activity := NewCheckFileExistsActivity()
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	env.RegisterActivity(activity.Execute)

	// Prepare a temporary file to test positive case
	tmpFile, err := os.CreateTemp("", "checkfile_test_*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tests := []struct {
		name            string
		input           workflowengine.ActivityInput
		expectExists    bool
		expectError     bool
		expectedErrCode errorcodes.Code
	}{
		{
			name: "Success - file exists",
			input: workflowengine.ActivityInput{
				Payload: CheckFileExistsActivityPayload{
					Path: tmpFile.Name(),
				},
			},
			expectExists: true,
		},
		{
			name: "Success - file does not exist",
			input: workflowengine.ActivityInput{
				Payload: CheckFileExistsActivityPayload{
					Path: "/nonexistent/path/to/file.txt",
				},
			},
			expectExists: false,
		},
		{
			name: "Failure - missing path key",
			input: workflowengine.ActivityInput{
				Payload: CheckFileExistsActivityPayload{},
			},
			expectError:     true,
			expectedErrCode: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result workflowengine.ActivityResult
			future, err := env.ExecuteActivity(activity.Execute, tt.input)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErrCode.Code)
				require.Contains(t, err.Error(), tt.expectedErrCode.Description)
			} else {
				require.NoError(t, err)
				future.Get(&result)
				exists, ok := result.Output.(bool)
				require.True(t, ok, "output should be a boolean")
				require.Equal(t, tt.expectExists, exists)
			}
		})
	}
}
