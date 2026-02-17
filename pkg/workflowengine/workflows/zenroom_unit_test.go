// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"context"
	"errors"
	"testing"

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
