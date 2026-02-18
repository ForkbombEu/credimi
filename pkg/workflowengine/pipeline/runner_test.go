// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestValidateRunnerIDYAML(t *testing.T) {
	t.Run("no mobile-automation steps", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
steps:
  - id: step1
    use: rest
`
		require.NoError(t, ValidateRunnerIDYAML(yamlContent))
	})

	t.Run("global runner conflicts with step runner", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
runtime:
  global_runner_id: global-runner
steps:
  - id: step1
    use: mobile-automation
    with:
      runner_id: step-runner
`
		err := ValidateRunnerIDYAML(yamlContent)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "step1"`)
		require.Contains(t, err.Error(), "global_runner_id is set")
	})

	t.Run("missing step runner without global", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
steps:
  - id: step1
    use: mobile-automation
    with:
      runner_id: step-runner
  - id: step2
    use: mobile-automation
`
		err := ValidateRunnerIDYAML(yamlContent)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "step2"`)
		require.Contains(t, err.Error(), "missing runner_id")
	})

	t.Run("first conflict step is deterministic", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
runtime:
  global_runner_id: global-runner
steps:
  - id: stepA
    use: mobile-automation
    with:
      runner_id: step-runner-a
  - id: stepB
    use: mobile-automation
    with:
      runner_id: step-runner-b
`
		err := ValidateRunnerIDYAML(yamlContent)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "stepA"`)
	})
}

func TestExecuteStepActivity(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	workflowName := "execute-step-activity"
	executeStepActivityWorkflow := func(ctx workflow.Context) (map[string]any, error) {
		ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
		ctx = workflow.WithActivityOptions(ctx, ao)

		step := StepDefinition{
			StepSpec: StepSpec{
				ID:  "step-1",
				Use: "http-request",
				With: StepInputs{
					Payload: map[string]any{
						"url": "https://example.com",
					},
				},
			},
		}

		output, err := ExecuteStep(
			step.ID,
			step.Use,
			step.With,
			step.ActivityOptions,
			ctx,
			map[string]any{},
			map[string]any{},
			ao,
		)
		if err != nil {
			return nil, err
		}

		return output.(map[string]any), nil
	}
	env.RegisterWorkflowWithOptions(
		executeStepActivityWorkflow,
		workflow.RegisterOptions{Name: workflowName},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: map[string]any{"body": "ok"}}, nil)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "ok", result["body"])
}

func TestExecuteStepWorkflow(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	childName := workflows.NewMobileAutomationWorkflow().Name()
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
			output := map[string]any{
				"app_url":   input.Config["app_url"],
				"taskqueue": input.Config["taskqueue"],
			}
			return workflowengine.WorkflowResult{Output: output}, nil
		},
		workflow.RegisterOptions{Name: childName},
	)

	workflowName := "execute-step-workflow"
	executeStepWorkflow := func(ctx workflow.Context) (map[string]any, error) {
		ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
		ctx = workflow.WithActivityOptions(ctx, ao)

		step := StepDefinition{
			StepSpec: StepSpec{
				ID:  "step-1",
				Use: "mobile-automation",
				With: StepInputs{
					Config: map[string]any{
						"taskqueue": "custom-queue",
						"app_url":   "",
					},
					Payload: map[string]any{
						"runner_id": "runner-1",
					},
				},
			},
		}

		output, err := ExecuteStep(
			step.ID,
			step.Use,
			step.With,
			step.ActivityOptions,
			ctx,
			map[string]any{},
			map[string]any{},
			ao,
		)
		if err != nil {
			return nil, err
		}

		return output.(map[string]any), nil
	}
	env.RegisterWorkflowWithOptions(
		executeStepWorkflow,
		workflow.RegisterOptions{Name: workflowName},
	)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "custom-queue", result["taskqueue"])
	require.Equal(t, "http://localhost:8090", result["app_url"])
}

func TestFetchChildPipelineYAML(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	workflowName := "fetch-child-yaml"
	fetchChildPipelineYAMLWorkflow := func(
		ctx workflow.Context,
		step StepDefinition,
		input PipelineWorkflowInput,
	) (string, error) {
		ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
		ctx = workflow.WithActivityOptions(ctx, ao)
		meta := &workflowengine.WorkflowErrorMetadata{}
		return fetchChildPipelineYAML(ctx, step, input, meta)
	}
	env.RegisterWorkflowWithOptions(
		fetchChildPipelineYAMLWorkflow,
		workflow.RegisterOptions{Name: workflowName},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: map[string]any{"body": "yaml-body"}}, nil)

	step := StepDefinition{
		StepSpec: StepSpec{
			ID:  "step-1",
			Use: "child-pipeline",
			With: StepInputs{
				Payload: map[string]any{
					"pipeline_id": "pipeline-1",
				},
			},
		},
	}
	input := PipelineWorkflowInput{
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{"app_url": "http://localhost:8090"},
		},
	}

	env.ExecuteWorkflow(workflowName, step, input)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result string
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "yaml-body", result)
}

func TestFetchChildPipelineYAMLErrors(t *testing.T) {
	tests := []struct {
		name          string
		stepPayload   map[string]any
		config        map[string]any
		activityBody  any
		expectMessage string
	}{
		{
			name:          "missing pipeline id",
			stepPayload:   map[string]any{},
			config:        map[string]any{"app_url": "http://localhost:8090"},
			expectMessage: "missing pipeline_id",
		},
		{
			name:          "missing app url",
			stepPayload:   map[string]any{"pipeline_id": "pipeline-1"},
			config:        map[string]any{},
			expectMessage: "app_url",
		},
		{
			name:          "invalid http output",
			stepPayload:   map[string]any{"pipeline_id": "pipeline-1"},
			config:        map[string]any{"app_url": "http://localhost:8090"},
			activityBody:  123,
			expectMessage: "invalid HTTP output",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()

			var httpActivity *activities.HTTPActivity
			if tc.activityBody != nil {
				httpActivity = activities.NewHTTPActivity()
				env.RegisterActivityWithOptions(
					httpActivity.Execute,
					activity.RegisterOptions{Name: httpActivity.Name()},
				)
			}

			workflowName := "fetch-child-yaml-errors"
			fetchChildPipelineYAMLWorkflow := func(
				ctx workflow.Context,
				step StepDefinition,
				input PipelineWorkflowInput,
			) (string, error) {
				ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
				ctx = workflow.WithActivityOptions(ctx, ao)
				meta := &workflowengine.WorkflowErrorMetadata{}
				return fetchChildPipelineYAML(ctx, step, input, meta)
			}
			env.RegisterWorkflowWithOptions(
				fetchChildPipelineYAMLWorkflow,
				workflow.RegisterOptions{Name: workflowName},
			)

			if tc.activityBody != nil {
				env.OnActivity(
					httpActivity.Name(),
					mock.Anything,
					mock.Anything,
				).Return(
					workflowengine.ActivityResult{Output: map[string]any{"body": tc.activityBody}},
					nil,
				)
			}

			step := StepDefinition{
				StepSpec: StepSpec{
					ID:  "step-1",
					Use: "child-pipeline",
					With: StepInputs{
						Payload: tc.stepPayload,
					},
				},
			}
			input := PipelineWorkflowInput{
				WorkflowInput: workflowengine.WorkflowInput{
					Config: tc.config,
				},
			}

			env.ExecuteWorkflow(workflowName, step, input)
			require.True(t, env.IsWorkflowCompleted())
			err := env.GetWorkflowError()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectMessage)
		})
	}
}
