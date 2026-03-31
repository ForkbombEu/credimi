// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type orderedActivity struct {
	name  string
	order *[]string
}

func (a *orderedActivity) Name() string {
	return a.name
}

func (a *orderedActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, _ := input.Payload.(map[string]any)
	*a.order = append(*a.order, workflowengine.AsString(payload["value"]))
	return workflowengine.ActivityResult{Output: map[string]any{"ok": true}}, nil
}

func (a *orderedActivity) NewActivityError(string, string, ...any) error {
	return errors.New("activity error")
}

func (a *orderedActivity) NewNonRetryableActivityError(string, string, ...any) error {
	return errors.New("activity error")
}

func (a *orderedActivity) NewMissingOrInvalidPayloadError(err error) error {
	return err
}

type orderedWorkflow struct {
	name string
}

func (w *orderedWorkflow) Name() string {
	return w.name
}

func (w *orderedWorkflow) GetOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{}
}

func (w *orderedWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.ExecuteWorkflow(ctx, input)
}

func (w *orderedWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)

	var activityResult workflowengine.ActivityResult
	err := workflow.ExecuteActivity(
		ctx,
		"order-activity",
		workflowengine.ActivityInput{
			Payload: map[string]any{
				"value": "external-step",
			},
		},
	).Get(ctx, &activityResult)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	return workflowengine.WorkflowResult{
		Output: map[string]any{"ok": true},
	}, nil
}

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

func TestPipelineWorkflowWrapsSetupHookCancellation(t *testing.T) {
	origSetupHooks := setupHooks
	t.Cleanup(func() {
		setupHooks = origSetupHooks
	})

	setupHooks = []SetupFunc{
		func(
			_ workflow.Context,
			_ *[]StepDefinition,
			_ *workflow.ActivityOptions,
			_ map[string]any,
			_ *map[string]any,
		) error {
			return temporal.NewCanceledError("setup canceled")
		},
	}

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
			WorkflowDefinition: &WorkflowDefinition{Name: "setup-cancel"},
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
	require.True(t, temporal.IsCanceledError(err))
}

func TestPipelineWorkflowDefersPlayStoreDisableUntilAfterExternalInstallSteps(t *testing.T) {
	origSetupHooks := setupHooks
	origCleanupHooks := cleanupHooks
	origExternalInstall := registry.Registry[mobileExternalInstallStepUse]
	origOrderStep, hadOrderStep := registry.Registry["order-step"]
	t.Cleanup(func() {
		setupHooks = origSetupHooks
		cleanupHooks = origCleanupHooks
		registry.Registry[mobileExternalInstallStepUse] = origExternalInstall
		if hadOrderStep {
			registry.Registry["order-step"] = origOrderStep
		} else {
			delete(registry.Registry, "order-step")
		}
	})

	setupHooks = []SetupFunc{
		func(
			_ workflow.Context,
			_ *[]StepDefinition,
			_ *workflow.ActivityOptions,
			_ map[string]any,
			runData *map[string]any,
		) error {
			(*runData)["setted_devices"] = map[string]any{
				"tenant/runner-1": map[string]any{
					"serial": "serial-1",
					"type":   deviceTypeAndroidPhone.String(),
				},
			}
			(*runData)[mobilePendingPlayStoreDisableRunDataKey] = true
			return nil
		},
	}
	cleanupHooks = nil

	registry.Registry[mobileExternalInstallStepUse] = registry.TaskFactory{
		Kind:        registry.TaskWorkflow,
		NewFunc:     func() any { return &orderedWorkflow{name: "ordered-external-install"} },
		PayloadType: reflect.TypeOf(map[string]any{}),
	}

	var order []string
	registry.Registry["order-step"] = registry.TaskFactory{
		Kind: registry.TaskActivity,
		NewFunc: func() any {
			return &orderedActivity{name: "order-activity", order: &order}
		},
		PayloadType: reflect.TypeOf(map[string]any{}),
		OutputKind:  workflowengine.OutputMap,
	}

	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	externalWorkflow := &orderedWorkflow{name: "ordered-external-install"}
	env.RegisterWorkflowWithOptions(
		externalWorkflow.Workflow,
		workflow.RegisterOptions{Name: externalWorkflow.Name()},
	)

	orderAct := &orderedActivity{name: "order-activity", order: &order}
	env.RegisterActivityWithOptions(
		orderAct.Execute,
		activity.RegisterOptions{Name: orderAct.Name()},
	)

	disableAct := activities.NewDisableAndroidPlayStoreActivity()
	env.RegisterActivityWithOptions(
		disableAct.Execute,
		activity.RegisterOptions{Name: disableAct.Name()},
	)
	env.OnActivity(
		disableAct.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			return ok && workflowengine.AsString(payload["serial"]) == "serial-1"
		}),
	).Return(func(
		_ context.Context,
		_ workflowengine.ActivityInput,
	) (workflowengine.ActivityResult, error) {
		order = append(order, "disable-play-store")
		return workflowengine.ActivityResult{Output: map[string]any{"message": "disabled"}}, nil
	}).Once()

	env.ExecuteWorkflow(
		pipelineWf.Name(),
		PipelineWorkflowInput{
			WorkflowDefinition: &WorkflowDefinition{
				Name: "defer-play-store-disable",
				Steps: []StepDefinition{
					{
						StepSpec: StepSpec{
							ID:  "install-1",
							Use: mobileExternalInstallStepUse,
							With: StepInputs{
								Payload: map[string]any{},
								Config: map[string]any{
									"app_url": "https://example.test",
								},
							},
						},
					},
					{
						StepSpec: StepSpec{
							ID:  "step-2",
							Use: "order-step",
							With: StepInputs{
								Payload: map[string]any{
									"value": "after-step",
								},
							},
						},
					},
				},
			},
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url":                              "https://example.test",
					mobileDisableAndroidPlayStoreConfigKey: true,
				},
				ActivityOptions: &workflow.ActivityOptions{
					StartToCloseTimeout: time.Second,
				},
			},
		},
	)

	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, []string{"external-step", "disable-play-store", "after-step"}, order)
}
