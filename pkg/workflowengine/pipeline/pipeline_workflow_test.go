// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// TestPipelineWorkflowFailsWithoutDefinition asserts a clear error when workflow_definition is missing.
func TestPipelineWorkflowFailsWithoutDefinition(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	env.ExecuteWorkflow(
		pipelineWf.Name(),
		PipelineWorkflowInput{
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url": "https://example.test",
				},
				ActivityOptions: &workflow.ActivityOptions{
					StartToCloseTimeout: time.Second,
				},
			},
		},
	)

	err := env.GetWorkflowError()
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, errorcodes.MissingOrInvalidPayload, appErr.Type())
	require.Contains(t, appErr.Error(), "workflow_definition")
}

func TestPipelineWorkflowContinueOnError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	jsonAct := activities.NewJSONActivity(map[string]reflect.Type{
		"map": reflect.TypeOf(map[string]any{}),
	})
	env.RegisterActivityWithOptions(
		jsonAct.Execute,
		activity.RegisterOptions{Name: jsonAct.Name()},
	)

	env.ExecuteWorkflow(
		pipelineWf.Name(),
		PipelineWorkflowInput{
			WorkflowDefinition: &WorkflowDefinition{
				Name: "continue-on-error",
				Steps: []StepDefinition{
					{
						StepSpec: StepSpec{
							ID:  "step-1",
							Use: "json-parse",
							With: StepInputs{
								Payload: map[string]any{
									"struct_type": "map",
								},
							},
						},
						ContinueOnError: true,
						OnError: []*OnErrorStepDefinition{
							{
								StepSpec: StepSpec{
									ID:  "on-error",
									Use: "json-parse",
									With: StepInputs{
										Payload: map[string]any{
											"struct_type": "map",
											"rawJSON":     `{"ok":true}`,
										},
									},
								},
							},
						},
					},
				},
			},
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url": "https://example.test",
				},
				ActivityOptions: &workflow.ActivityOptions{
					StartToCloseTimeout: time.Second,
				},
			},
		},
	)

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "workflow completed with 1 step errors")
}

func TestPipelineWorkflowOnSuccessWithDebug(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	jsonAct := activities.NewJSONActivity(map[string]reflect.Type{
		"map": reflect.TypeOf(map[string]any{}),
	})
	env.RegisterActivityWithOptions(
		jsonAct.Execute,
		activity.RegisterOptions{Name: jsonAct.Name()},
	)

	debugAct := NewDebugActivity()
	env.RegisterActivityWithOptions(
		debugAct.Execute,
		activity.RegisterOptions{Name: debugAct.Name()},
	)

	env.ExecuteWorkflow(
		pipelineWf.Name(),
		PipelineWorkflowInput{
			WorkflowDefinition: &WorkflowDefinition{
				Name: "on-success",
				Steps: []StepDefinition{
					{
						StepSpec: StepSpec{
							ID:  "step-1",
							Use: "json-parse",
							With: StepInputs{
								Payload: map[string]any{
									"struct_type": "map",
									"rawJSON":     `{"ok":true}`,
								},
							},
						},
						OnSuccess: []*OnSuccessStepDefinition{
							{
								StepSpec: StepSpec{
									ID:  "on-success",
									Use: "json-parse",
									With: StepInputs{
										Payload: map[string]any{
											"struct_type": "map",
											"rawJSON":     `{"ok":true}`,
										},
									},
								},
							},
						},
					},
				},
			},
			Debug: true,
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url": "https://example.test",
				},
				ActivityOptions: &workflow.ActivityOptions{
					StartToCloseTimeout: time.Second,
				},
			},
		},
	)

	require.NoError(t, env.GetWorkflowError())

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	require.Contains(t, output, "step-1")
}
