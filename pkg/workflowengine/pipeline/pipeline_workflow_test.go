// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
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
			WorkflowDefinition: &pipeline.WorkflowDefinition{
				Name: "continue-on-error",
				Steps: []pipeline.StepDefinition{
					{
						StepSpec: pipeline.StepSpec{
							ID:  "step-1",
							Use: "json-parse",
							With: pipeline.StepInputs{
								Payload: map[string]any{
									"struct_type": "map",
								},
							},
						},
						ContinueOnError: true,
						OnError: []*pipeline.OnErrorStepDefinition{
							{
								StepSpec: pipeline.StepSpec{
									ID:  "on-error",
									Use: "json-parse",
									With: pipeline.StepInputs{
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
			WorkflowDefinition: &pipeline.WorkflowDefinition{
				Name: "on-success",
				Steps: []pipeline.StepDefinition{
					{
						StepSpec: pipeline.StepSpec{
							ID:  "step-1",
							Use: "json-parse",
							With: pipeline.StepInputs{
								Payload: map[string]any{
									"struct_type": "map",
									"rawJSON":     `{"ok":true}`,
								},
							},
						},
						OnSuccess: []*pipeline.OnSuccessStepDefinition{
							{
								StepSpec: pipeline.StepSpec{
									ID:  "on-success",
									Use: "json-parse",
									With: pipeline.StepInputs{
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
			_ *[]pipeline.StepDefinition,
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
			WorkflowDefinition: &pipeline.WorkflowDefinition{Name: "setup-cancel"},
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
			_ *[]pipeline.StepDefinition,
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
			WorkflowDefinition: &pipeline.WorkflowDefinition{
				Name: "defer-play-store-disable",
				Steps: []pipeline.StepDefinition{
					{
						StepSpec: pipeline.StepSpec{
							ID:  "install-1",
							Use: mobileExternalInstallStepUse,
							With: pipeline.StepInputs{
								Payload: map[string]any{},
								Config: map[string]any{
									"app_url": "https://example.test",
								},
							},
						},
					},
					{
						StepSpec: pipeline.StepSpec{
							ID:  "step-2",
							Use: "order-step",
							With: pipeline.StepInputs{
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

func TestPipelineWorkflowFinallyWithHTTP(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	emailActivity := activities.NewSendMailActivity()
	env.RegisterActivityWithOptions(
		emailActivity.Execute,
		activity.RegisterOptions{Name: emailActivity.Name()},
	)

	var callOrder []string

	env.OnActivity(httpActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{"body": "ok"}}, nil).
		Run(func(args mock.Arguments) {
			input := args.Get(1).(workflowengine.ActivityInput)
			if payload, ok := input.Payload.(*activities.HTTPActivityPayload); ok {
				callOrder = append(callOrder, "http: "+payload.URL)
			} else if payload, ok := input.Payload.(map[string]any); ok {
				if url, ok := payload["url"]; ok {
					callOrder = append(callOrder, "http: "+url.(string))
				}
			}
		})

	env.OnActivity(emailActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{"sent": true}}, nil).
		Run(func(args mock.Arguments) {
			callOrder = append(callOrder, "email")
		})

	wfDef := &pipeline.WorkflowDefinition{
		Name: "test-finally-http",
		Steps: []pipeline.StepDefinition{
			{
				StepSpec: pipeline.StepSpec{
					ID:  "main-step",
					Use: "http-request",
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"url": "https://example.com/main",
						},
					},
				},
			},
		},
		Finally: []pipeline.StepDefinition{
			{
				StepSpec: pipeline.StepSpec{
					ID:  "finally-http",
					Use: "http-request",
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"method": "POST",
							"url":    "https://example.com/finally",
							"body": map[string]any{
								"phase":  "finally",
								"result": "${{result}}",
							},
						},
					},
				},
			},
			{
				StepSpec: pipeline.StepSpec{
					ID:  "finally-email",
					Use: "email",
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"recipient": "test@example.com",
							"subject":   "Pipeline finished",
							"body":      "Done",
						},
					},
				},
			},
		},
	}

	env.ExecuteWorkflow(
		pipelineWf.Name(),
		PipelineWorkflowInput{
			WorkflowDefinition: wfDef,
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

	t.Logf("Call order: %v", callOrder)
	require.Len(t, callOrder, 3, "Should have main step + 2 finally steps")
	require.Contains(t, callOrder[0], "https://example.com/main")
	require.Contains(t, callOrder[1], "https://example.com/finally")
	require.Equal(t, "email", callOrder[2])
}

func TestValidateFinallySteps(t *testing.T) {
	validSteps := []pipeline.StepDefinition{
		{
			StepSpec: pipeline.StepSpec{
				ID:  "valid-http",
				Use: "http-request",
			},
		},
		{
			StepSpec: pipeline.StepSpec{
				ID:  "valid-email",
				Use: "email",
			},
		},
	}
	err := ValidateFinallySteps(validSteps)
	require.NoError(t, err)

	invalidSteps := []pipeline.StepDefinition{
		{
			StepSpec: pipeline.StepSpec{
				ID:  "invalid-json",
				Use: "json-parse",
			},
		},
	}
	err = ValidateFinallySteps(invalidSteps)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
	require.Contains(t, err.Error(), "json-parse")
	require.Contains(t, err.Error(), "invalid-json")
	mixedSteps := []pipeline.StepDefinition{
		{
			StepSpec: pipeline.StepSpec{
				ID:  "invalid-mobile",
				Use: "mobile-automation",
			},
		},
		{
			StepSpec: pipeline.StepSpec{
				ID:  "valid-email",
				Use: "email",
			},
		},
	}
	err = ValidateFinallySteps(mixedSteps)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mobile-automation")
}

func TestPipelineWorkflowFinallyValidationFails(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	wfDef := &pipeline.WorkflowDefinition{
		Name: "test-invalid-finally",
		Steps: []pipeline.StepDefinition{
			{
				StepSpec: pipeline.StepSpec{
					ID:  "main-step",
					Use: "http-request",
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"url": "https://example.com",
						},
					},
				},
			},
		},
		Finally: []pipeline.StepDefinition{
			{
				StepSpec: pipeline.StepSpec{
					ID:  "invalid-finally",
					Use: "json-parse", // NON CONSENTITO!
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"struct_type": "map",
						},
					},
				},
			},
		},
	}

	env.ExecuteWorkflow(
		pipelineWf.Name(),
		PipelineWorkflowInput{
			WorkflowDefinition: wfDef, // ← NON nil, ha finally invalidi
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
	require.Contains(t, err.Error(), "not allowed")
	require.Contains(t, err.Error(), "json-parse")
}

func TestPipelineWorkflowStartWithValidFinally(t *testing.T) {
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

	yamlContent := `
name: test-valid-finally
steps:
  - id: main-step
    use: http-request
    with:
      payload:
        url: "https://example.com/main"
finally:
  - id: valid-http
    use: http-request
    with:
      payload:
        url: "https://example.com/finally"
  - id: valid-email
    use: email
    with:
      payload:
        recipient: "test@example.com"
        subject: "Test"
        body: "Body"
`

	_, err := pipelineWf.Start(
		yamlContent,
		map[string]any{
			"namespace": "default",
			"app_url":   "https://example.test",
		},
		map[string]any{},
		"test/valid-finally",
	)

	require.NoError(t, err)
}

func TestPipelineWorkflowFinallyErrorsDontBlockWorkflow(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	emailActivity := activities.NewSendMailActivity()
	env.RegisterActivityWithOptions(
		emailActivity.Execute,
		activity.RegisterOptions{Name: emailActivity.Name()},
	)

	env.OnActivity(httpActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{"body": "ok"}}, nil)

	env.OnActivity(emailActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, errors.New("email service unavailable"))

	wfDef := &pipeline.WorkflowDefinition{
		Name: "test-finally-errors",
		Steps: []pipeline.StepDefinition{
			{
				StepSpec: pipeline.StepSpec{
					ID:  "main-step",
					Use: "http-request",
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"url": "https://example.com/main",
						},
					},
				},
			},
		},
		Finally: []pipeline.StepDefinition{
			{
				StepSpec: pipeline.StepSpec{
					ID:  "finally-email",
					Use: "email",
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"recipient": "test@example.com",
							"subject":   "Pipeline finished",
							"body":      "Done",
						},
					},
				},
			},
		},
	}

	env.ExecuteWorkflow(
		pipelineWf.Name(),
		PipelineWorkflowInput{
			WorkflowDefinition: wfDef,
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

	finallyErrors, exists := output["finally_errors"]
	require.True(t, exists, "finally_errors should be present in output")

	errorStr := fmt.Sprintf("%v", finallyErrors)
	require.Contains(t, errorStr, "email service unavailable")

	require.Contains(t, output, "main-step")
}
