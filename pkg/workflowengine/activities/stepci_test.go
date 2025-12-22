// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//go:build !unit

package activities

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestStepCIActivity_Configure(t *testing.T) {
	activity := NewStepCIWorkflowActivity()

	tests := []struct {
		name             string
		config           map[string]string
		payload          StepCIWorkflowActivityPayload
		expectedYAML     string
		expectError      bool
		expectedErrorMsg errorcodes.Code
	}{
		{
			name: "Success - valid template",
			config: map[string]string{
				"template": `hello: [[ .name ]]`,
			},
			payload: StepCIWorkflowActivityPayload{
				Data: map[string]any{
					"name": "world",
				},
			},
			expectedYAML: "hello: world",
		},
		{
			name:             "Failure - missing template",
			config:           map[string]string{},
			expectError:      true,
			expectedErrorMsg: errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
		},
		{
			name: "Failure - invalid template syntax",
			config: map[string]string{
				"template": `[[ .name ]`},
			payload: StepCIWorkflowActivityPayload{
				Data: map[string]any{
					"name": "bad",
				},
			},
			expectError:      true,
			expectedErrorMsg: errorcodes.Codes[errorcodes.TemplateRenderFailed],
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := &workflowengine.ActivityInput{
				Config:  tc.config,
				Payload: tc.payload,
			}

			err := activity.Configure(input)

			if tc.expectError {
				require.Error(t, err)
				if tc.expectedErrorMsg != (errorcodes.Code{}) {
					require.Contains(t, err.Error(), tc.expectedErrorMsg.Code)
					require.Contains(t, err.Error(), tc.expectedErrorMsg.Description)
				}
			} else {
				require.NoError(t, err)
				payload, err := workflowengine.DecodePayload[StepCIWorkflowActivityPayload](input.Payload)
				require.NoError(t, err)
				require.Equal(t, tc.expectedYAML, strings.TrimSpace(payload.Yaml))
			}
		})
	}
}

func TestStepCIActivity_Execute(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()

	activity := &StepCIWorkflowActivity{}
	env.RegisterActivity(activity.Execute)

	tmpBinDir := t.TempDir()
	t.Setenv("BIN", tmpBinDir)

	tests := []struct {
		name             string
		mockScript       string
		expectError      bool
		expectedCaptures any
	}{
		{
			name: "success",
			mockScript: `#!/bin/sh
echo '{"passed":true,"captures":{"test":1}}'
`,
			expectError:      false,
			expectedCaptures: map[string]any{"test": float64(1)},
		},
		{
			name: "failure",
			mockScript: `#!/bin/sh
echo '{"passed":false}'
exit 1
`,
			expectError: true,
		},
	}

	writeMockRunner := func(script string) string {
		binPath := filepath.Join(tmpBinDir, "stepci-captured-runner")
		err := os.WriteFile(binPath, []byte(script), 0755)
		require.NoError(t, err)
		return binPath
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			writeMockRunner(tc.mockScript)

			input := workflowengine.ActivityInput{
				Payload: StepCIWorkflowActivityPayload{
					Yaml: "dummy",
				},
			}

			future, err := env.ExecuteActivity(activity.Execute, input)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				var result workflowengine.ActivityResult
				err = future.Get(&result)
				require.NoError(t, err)
				outputMap, ok := result.Output.(map[string]any)
				require.True(t, ok)
				require.Equal(t, tc.expectedCaptures, outputMap["captures"])
			}
		})
	}
}
