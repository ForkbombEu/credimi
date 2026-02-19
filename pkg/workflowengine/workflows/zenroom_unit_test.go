// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type fakeDockerClient struct {
	removed []string
}

func (f *fakeDockerClient) ContainerRemove(
	_ context.Context,
	containerID string,
	_ container.RemoveOptions,
) error {
	f.removed = append(f.removed, containerID)
	return nil
}

func TestZenroomWorkflowSuccess(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	w := NewZenroomWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{
		Name: w.Name(),
	})

	dockerActivity := activities.NewDockerActivity()
	env.RegisterActivityWithOptions(dockerActivity.Execute, activity.RegisterOptions{
		Name: dockerActivity.Name(),
	})

	fake := &fakeDockerClient{}
	origNewDockerClient := newDockerClient
	t.Cleanup(func() {
		newDockerClient = origNewDockerClient
	})
	newDockerClient = func() (dockerClient, error) {
		return fake, nil
	}

	env.OnActivity(dockerActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: map[string]any{
				"containerID": "cont-1",
				"exitCode":    float64(0),
				"stderr":      "",
				"stdout":      `{"ok":true}`,
			},
			Log: []string{"log-1"},
		}, nil)

	input := workflowengine.WorkflowInput{
		Payload: ZenroomWorkflowPayload{
			Contract: "contract",
		},
		Config: map[string]any{
			"app_url": "http://app.test",
		},
	}
	env.ExecuteWorkflow(w.Name(), input)

	var result workflowengine.WorkflowResult
	err := env.GetWorkflowResult(&result)
	require.NoError(t, err)
	require.Equal(t, "Zenroom execution successful", result.Message)
	require.Equal(t, true, result.Output.(map[string]any)["ok"])
	require.Equal(t, []string{"cont-1"}, fake.removed)
}

func TestZenroomWorkflowDockerClientFailure(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	w := NewZenroomWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{
		Name: w.Name(),
	})

	dockerActivity := activities.NewDockerActivity()
	env.RegisterActivityWithOptions(dockerActivity.Execute, activity.RegisterOptions{
		Name: dockerActivity.Name(),
	})

	origNewDockerClient := newDockerClient
	t.Cleanup(func() {
		newDockerClient = origNewDockerClient
	})
	newDockerClient = func() (dockerClient, error) {
		return nil, errors.New("boom")
	}

	env.OnActivity(dockerActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: map[string]any{
				"containerID": "cont-2",
				"exitCode":    float64(0),
				"stderr":      "",
				"stdout":      `{"ok":true}`,
			},
		}, nil)

	input := workflowengine.WorkflowInput{
		Payload: ZenroomWorkflowPayload{
			Contract: "contract",
		},
		Config: map[string]any{
			"app_url": "http://app.test",
		},
	}
	env.ExecuteWorkflow(w.Name(), input)

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.DockerClientCreationFailed].Code)
}

func TestZenroomWorkflowStart(t *testing.T) {
	origClient := zenroomTemporalClient
	t.Cleanup(func() {
		zenroomTemporalClient = origClient
	})

	mockClient := &temporalmocks.Client{}
	mockRun := &temporalmocks.WorkflowRun{}
	mockClient.
		On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mockRun, nil).
		Once()

	zenroomTemporalClient = func(namespace string) (client.Client, error) {
		require.Equal(t, "ns-1", namespace)
		return mockClient, nil
	}

	w := NewZenroomWorkflow()
	input := workflowengine.WorkflowInput{
		Config: map[string]any{
			"namespace": "ns-1",
			"Memo":      map[string]any{"test": "memo"},
		},
	}

	_, err := w.Start(input)
	require.NoError(t, err)

	args := mockClient.Calls[0].Arguments
	options := args.Get(1).(client.StartWorkflowOptions)
	require.Equal(t, ZenroomTaskQueue, options.TaskQueue)
	require.True(t, strings.HasPrefix(options.ID, "Zenroom-Workflow-"))
	require.Equal(t, map[string]any{"test": "memo"}, options.Memo)
}

func TestZenroomWorkflowStartClientError(t *testing.T) {
	origClient := zenroomTemporalClient
	t.Cleanup(func() {
		zenroomTemporalClient = origClient
	})

	zenroomTemporalClient = func(namespace string) (client.Client, error) {
		require.Equal(t, DefaultNamespace, namespace)
		return nil, errors.New("no client")
	}

	w := NewZenroomWorkflow()
	_, err := w.Start(workflowengine.WorkflowInput{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to create client")
}

func TestZenroomWorkflowOutputErrors(t *testing.T) {
	tests := []struct {
		name       string
		output     any
		errCode    string
		errMessage string
	}{
		{
			name:    "invalid output type",
			output:  "nope",
			errCode: errorcodes.Codes[errorcodes.UnexpectedDockerOutput].Code,
		},
		{
			name: "invalid exit code type",
			output: map[string]any{
				"containerID": "cont-1",
				"exitCode":    "bad",
			},
			errCode: errorcodes.Codes[errorcodes.UnexpectedDockerOutput].Code,
		},
		{
			name: "invalid stderr type",
			output: map[string]any{
				"containerID": "cont-1",
				"exitCode":    float64(0),
				"stderr":      123,
			},
			errCode: errorcodes.Codes[errorcodes.UnexpectedDockerOutput].Code,
		},
		{
			name: "invalid stdout type",
			output: map[string]any{
				"containerID": "cont-1",
				"exitCode":    float64(0),
				"stderr":      "",
				"stdout":      123,
				"stout":       "bad-output",
			},
			errCode: errorcodes.Codes[errorcodes.UnexpectedDockerOutput].Code,
		},
		{
			name: "non-zero exit code",
			output: map[string]any{
				"containerID": "cont-1",
				"exitCode":    float64(2),
				"stderr":      "boom",
				"stdout":      "out",
			},
			errCode: errorcodes.Codes[errorcodes.ZenroomExecutionFailed].Code,
		},
		{
			name: "invalid stdout JSON",
			output: map[string]any{
				"containerID": "cont-1",
				"exitCode":    float64(0),
				"stderr":      "",
				"stdout":      "not-json",
			},
			errCode: errorcodes.Codes[errorcodes.JSONUnmarshalFailed].Code,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testSuite := &testsuite.WorkflowTestSuite{}
			env := testSuite.NewTestWorkflowEnvironment()

			w := NewZenroomWorkflow()
			env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{
				Name: w.Name(),
			})

			dockerActivity := activities.NewDockerActivity()
			env.RegisterActivityWithOptions(dockerActivity.Execute, activity.RegisterOptions{
				Name: dockerActivity.Name(),
			})

			fake := &fakeDockerClient{}
			origNewDockerClient := newDockerClient
			t.Cleanup(func() {
				newDockerClient = origNewDockerClient
			})
			newDockerClient = func() (dockerClient, error) {
				return fake, nil
			}

			env.OnActivity(dockerActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{
					Output: tc.output,
				}, nil)

			input := workflowengine.WorkflowInput{
				Payload: ZenroomWorkflowPayload{
					Contract: "contract",
				},
				Config: map[string]any{
					"app_url": "http://app.test",
				},
			}
			env.ExecuteWorkflow(w.Name(), input)

			err := env.GetWorkflowError()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errCode)
			if tc.errMessage != "" {
				require.Contains(t, err.Error(), tc.errMessage)
			}
		})
	}
}
