// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestPipelineStartMissingNamespace(t *testing.T) {
	pipelineWf := NewPipelineWorkflow()
	result, err := pipelineWf.Start(
		"name: test-pipeline\nsteps: []\n",
		map[string]any{},
		map[string]any{},
	)
	require.Error(t, err)
	require.Empty(t, result.WorkflowID)
	require.Contains(t, err.Error(), "namespace is required")
}

func TestPipelineStartInvalidYAML(t *testing.T) {
	pipelineWf := NewPipelineWorkflow()
	_, err := pipelineWf.Start("name: [", map[string]any{}, map[string]any{})
	require.Error(t, err)
}

func TestPipelineStartScheduled(t *testing.T) {
	pipelineWf := NewPipelineWorkflow()

	originalClient := pipelineTemporalClient
	defer func() {
		pipelineTemporalClient = originalClient
	}()

	mockClient := temporalmocks.NewClient(t)
	scheduleClient := temporalmocks.NewScheduleClient(t)
	scheduleHandle := temporalmocks.NewScheduleHandle(t)

	scheduleHandle.On("Describe", mock.Anything).Return(&client.ScheduleDescription{}, nil)
	scheduleHandle.On("GetID").Return("schedule-123")
	scheduleClient.On("Create", mock.Anything, mock.Anything).Return(scheduleHandle, nil)
	mockClient.On("ScheduleClient").Return(scheduleClient)

	pipelineTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	result, err := pipelineWf.Start(
		"name: scheduled-pipeline\nruntime:\n  schedule:\n    interval: 1m\nsteps: []\n",
		map[string]any{"namespace": "default"},
		map[string]any{},
	)
	require.NoError(t, err)
	require.Equal(t, "schedule-123", result.WorkflowID)
	require.Contains(t, result.Message, "scheduled successfully")
}

func TestPipelineStartImmediate(t *testing.T) {
	pipelineWf := NewPipelineWorkflow()

	originalClient := pipelineTemporalClient
	defer func() {
		pipelineTemporalClient = originalClient
	}()

	mockClient := temporalmocks.NewClient(t)
	workflowRun := temporalmocks.NewWorkflowRun(t)

	workflowRun.On("GetID").Return("workflow-123")
	workflowRun.On("GetRunID").Return("run-456")
	mockClient.On(
		"ExecuteWorkflow",
		mock.Anything,
		mock.Anything,
		pipelineWf.Name(),
		mock.Anything,
	).Return(workflowRun, nil)

	pipelineTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	result, err := pipelineWf.Start(
		"name: immediate-pipeline\nsteps: []\n",
		map[string]any{"namespace": "default"},
		map[string]any{},
	)
	require.NoError(t, err)
	require.Equal(t, "workflow-123", result.WorkflowID)
	require.Equal(t, "run-456", result.WorkflowRunID)
}

func TestPipelineWorkflowSuccessWithNoSteps(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(pipelineWf.Workflow, workflow.RegisterOptions{Name: pipelineWf.Name()})

	env.ExecuteWorkflow(pipelineWf.Name(), PipelineWorkflowInput{
		WorkflowDefinition: &WorkflowDefinition{
			Name:  "empty-steps",
			Steps: []StepDefinition{},
		},
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url": "https://example.test",
			},
		},
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, output["workflow-id"])
	require.NotEmpty(t, output["workflow-run-id"])
}
