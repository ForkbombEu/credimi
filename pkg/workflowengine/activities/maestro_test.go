// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestMaestroFlowActivity_Execute(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()
	activity := NewMaestroFlowActivity()
	env.RegisterActivity(activity.Execute)

	tmpBinDir := t.TempDir()
	binPath := filepath.Join(tmpBinDir, "maestro", "bin", "maestro")

	// Create dummy binary that prints version
	err := os.MkdirAll(filepath.Dir(binPath), 0755)
	require.NoError(t, err)
	err = os.WriteFile(binPath, []byte("#!/bin/sh\necho \"Maestro CLI v1.2.3\""), 0755)
	require.NoError(t, err)

	// Set BIN environment variable to point to tmp directory
	os.Setenv("BIN", tmpBinDir)

	tests := []struct {
		name             string
		payload          map[string]interface{}
		expectedError    bool
		expectedErrorMsg errorcodes.Code
		expectedOutput   string
	}{
		{
			name: "Success - prints version",
			payload: map[string]interface{}{
				"yaml": "some: yaml",
			},
			expectedOutput: "Maestro CLI v1.2.3",
		},
		{
			name:             "Failure - missing payload",
			payload:          map[string]interface{}{},
			expectedError:    true,
			expectedErrorMsg: errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "Failure - missing binary",
			payload: map[string]interface{}{
				"yaml": "some: yaml",
			},
			expectedError:    true,
			expectedErrorMsg: errorcodes.Codes[errorcodes.CommandExecutionFailed],
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := workflowengine.ActivityInput{
				Payload: tc.payload,
			}

			// Remove binary for the missing binary test
			if tc.name == "Failure - missing binary" {
				os.Remove(binPath)
			}

			var result workflowengine.ActivityResult
			future, err := env.ExecuteActivity(activity.Execute, input)

			if tc.expectedError {
				require.Error(t, err)
				if tc.expectedErrorMsg != (errorcodes.Code{}) {
					require.Contains(t, err.Error(), tc.expectedErrorMsg.Code)
					require.Contains(t, err.Error(), tc.expectedErrorMsg.Description)
				}
			} else {
				require.NoError(t, err)
				future.Get(&result)
				require.Contains(t, result.Output.(string), tc.expectedOutput)
			}
		})
	}
}
