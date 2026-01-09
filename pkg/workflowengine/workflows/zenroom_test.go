// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
//go:build !unit

package workflows

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type fakeDockerClient struct{}

func (fakeDockerClient) ContainerRemove(
	_ context.Context,
	_ string,
	_ container.RemoveOptions,
) error {
	return nil
}

func Test_ZenroomWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}

	tests := []struct {
		name            string
		contract        string
		keys            string
		data            string
		expectError     bool
		expectOutputs   []string
		expectErrorCode errorcodes.Code
	}{
		{
			name: "Successful execution",
			contract: `
Given I have a 'string' named 'message'
Given I have a 'string' named 'keys'
Then print the data
`,
			keys:          `{"keys": "hello from keys"}`,
			data:          `{"message": "hello from data"}`,
			expectError:   false,
			expectOutputs: []string{"keys", "message"},
		},
		{
			name: "Failure due to broken contract",
			contract: `
Given I have a 'string' named 'broken'
`,
			expectError:     true,
			expectErrorCode: errorcodes.Codes[errorcodes.ZenroomExecutionFailed],
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := testSuite.NewTestWorkflowEnvironment()
			env.SetTestTimeout(10 * time.Minute)

			originalFactory := dockerClientFactory
			dockerClientFactory = func() (dockerContainerRemover, error) {
				return fakeDockerClient{}, nil
			}
			t.Cleanup(func() {
				dockerClientFactory = originalFactory
			})

			w := NewZenroomWorkflow()
			env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{
				Name: w.Name(),
			})

			zenroomActivity := activities.NewDockerActivity()
			env.RegisterActivityWithOptions(
				zenroomActivity.Execute,
				activity.RegisterOptions{Name: zenroomActivity.Name()},
			)
			exitCode := float64(0)
			stdout := `{"keys":"hello from keys","message":"hello from data"}`
			if tc.expectError {
				exitCode = float64(1)
				stdout = `{"error":"broken contract"}`
			}
			env.OnActivity(
				zenroomActivity.Name(),
				mock.Anything,
				mock.Anything,
			).Return(workflowengine.ActivityResult{
				Output: map[string]any{
					"containerID": "container-test",
					"exitCode":    exitCode,
					"stderr":      "stderr output",
					"stdout":      stdout,
				},
				Log: []string{"stderr output"},
			}, nil)

			payload := ZenroomWorkflowPayload{
				Contract: tc.contract,
			}
			if tc.contract != "" {
				payload.Contract = tc.contract
			}

			if tc.keys != "" {
				payload.Keys = tc.keys
			}
			if tc.data != "" {
				payload.Data = tc.data
			}
			input := workflowengine.WorkflowInput{
				Payload: payload,
				Config: map[string]any{
					"app_url": "http://app.example.com",
				},
			}
			env.ExecuteWorkflow(w.Name(), input)

			var result workflowengine.WorkflowResult
			err := env.GetWorkflowResult(&result)

			if tc.expectError {
				require.Error(t, err, "Expected an error but got none")
				require.Contains(t, err.Error(), tc.expectErrorCode.Code)
			} else {
				require.NoError(t, err, "Expected no error but got one")
				for _, key := range tc.expectOutputs {
					require.Contains(t, result.Output, key, "Output should contain expected key")
				}
			}
		})
	}
}
