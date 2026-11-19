// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevice"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/temporalnexus"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/proto"
)

type fakeActivity struct {
	name string
}

// registerStubActivity registers a no-op activity under name so tests can mock
// it with env.OnActivity.
func registerStubActivity(env *testsuite.TestWorkflowEnvironment, name string) {
	env.RegisterActivityWithOptions(
		func(context.Context, workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{}, nil
		},
		activity.RegisterOptions{Name: name},
	)
}

// resolveRecordInputMatcher matches a ResolveRecord activity call with want.
func resolveRecordInputMatcher(want activities.ResolveRecordInput) any {
	return mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
		payload, err := workflowengine.DecodePayload[activities.ResolveRecordInput](input.Payload)
		return err == nil && payload == want
	})
}

// Runner-directed HTTP is its own activity, so a test that drives a step which
// talks to a runner has to register it.
func registerMobileRunnerHTTPActivity(
	env *testsuite.TestWorkflowEnvironment,
) *activities.MobileRunnerHTTPActivity {
	runnerHTTPActivity := activities.NewMobileRunnerHTTPActivity(nil)
	env.RegisterActivityWithOptions(
		runnerHTTPActivity.Execute,
		activity.RegisterOptions{Name: runnerHTTPActivity.Name()},
	)

	return runnerHTTPActivity
}

func (f *fakeActivity) Name() string {
	return f.name
}

func (f *fakeActivity) NewActivityError(workflowengine.ActivityError) error {
	return errors.New("activity error")
}

func (f *fakeActivity) NewNonRetryableActivityError(workflowengine.ActivityError) error {
	return errors.New("activity error")
}

func (f *fakeActivity) NewMissingOrInvalidPayloadError(err error) error {
	return err
}

type fakeConfigActivity struct {
	fakeActivity
	configErr error
}

func (f *fakeConfigActivity) Configure(*workflowengine.ActivityInput) error {
	return f.configErr
}

func TestValidateDeviceIDYAML(t *testing.T) {
	t.Run("no mobile-automation steps", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
steps:
  - id: step1
    use: rest
`
		require.NoError(t, ValidateDeviceIDYAML(yamlContent))
	})

	t.Run("global runner conflicts with step runner", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
runtime:
  global_device_id: global-device
steps:
  - id: step1
    use: mobile-automation
    with:
      payload:
        device_id: step-device
`
		err := ValidateDeviceIDYAML(yamlContent)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "step1"`)
		require.Contains(t, err.Error(), "global_device_id is set")
	})

	t.Run("missing step runner without global", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
steps:
  - id: step1
    use: mobile-automation
    with:
      payload:
        device_id: step-device
  - id: step2
    use: mobile-automation
`
		err := ValidateDeviceIDYAML(yamlContent)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "step2"`)
		require.Contains(t, err.Error(), "missing device_id")
	})

	t.Run("first conflict step is deterministic", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
runtime:
  global_device_id: global-device
steps:
  - id: stepA
    use: mobile-automation
    with:
      payload:
        device_id: step-device-a
  - id: stepB
    use: mobile-automation
    with:
      payload:
        device_id: step-device-b
`
		err := ValidateDeviceIDYAML(yamlContent)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "stepA"`)
	})

	t.Run("missing nested on_error device_id", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
steps:
  - id: step1
    use: rest
    on_error:
      - id: error-step
        use: mobile-automation
        with:
          payload:
            device_id: " / "
`
		err := ValidateDeviceIDYAML(yamlContent)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "error-step"`)
		require.Contains(t, err.Error(), "missing device_id")
	})

	t.Run("valid nested on_success device_id", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
steps:
  - id: step1
    use: rest
    on_success:
      - id: success-step
        use: mobile-automation
        with:
          payload:
            device_id: /tenant/runner/device
`
		require.NoError(t, ValidateDeviceIDYAML(yamlContent))
	})

	t.Run("nested device_id conflicts with global_device_id", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
runtime:
  global_device_id: tenant/global/device
steps:
  - id: step1
    use: rest
    on_error:
      - id: error-step
        use: mobile-automation
        with:
          payload:
            device_id: tenant/runner/device
`
		err := ValidateDeviceIDYAML(yamlContent)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "error-step"`)
		require.Contains(t, err.Error(), "global_device_id is set")
	})

	t.Run("nested mobile step inherits global_device_id", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
runtime:
  global_device_id: tenant/global/device
steps:
  - id: step1
    use: rest
    on_success:
      - id: success-step
        use: mobile-automation
`
		require.NoError(t, ValidateDeviceIDYAML(yamlContent))
	})

	t.Run("nested non-mobile step does not require device_id", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
steps:
  - id: step1
    use: rest
    on_success:
      - id: success-step
        use: echo
`
		require.NoError(t, ValidateDeviceIDYAML(yamlContent))
	})
}

func TestExecuteStepActivity(t *testing.T) {
	globalCfg := map[string]any{
		"app_url":   "https://app.example",
		"namespace": "tenant",
		"app_logo":  "https://app.example/logo.png",
		workflowengine.TelemetryRootWorkflowIDKey: "root-wf",
		workflowengine.TelemetryRootRunIDKey:      "root-run",
	}
	tests := []struct {
		name       string
		activity   workflowengine.ExecutableActivity
		step       pipeline.StepDefinition
		wantConfig map[string]string
	}{
		{
			name:     "http-request receives only its own config and step_id",
			activity: activities.NewHTTPActivity(),
			step: pipeline.StepDefinition{StepSpec: pipeline.StepSpec{
				ID:  "step-1",
				Use: "http-request",
				With: pipeline.StepInputs{
					Config:  map[string]any{"timeout": "5s"},
					Payload: map[string]any{"url": "https://example.com"},
				},
			}},
			wantConfig: map[string]string{"timeout": "5s", "step_id": "step-1"},
		},
		{
			name:     "fcaf-validation inherits root run IDs",
			activity: activities.NewFCAFValidationActivity(nil, registry.StepActivityOutputKind),
			step: pipeline.StepDefinition{StepSpec: pipeline.StepSpec{
				ID:  "validate",
				Use: "fcaf-validation",
				With: pipeline.StepInputs{
					Payload: map[string]any{
						"test_id":          "test-a",
						"pipeline_outputs": map[string]any{"a": "b"},
					},
				},
			}},
			wantConfig: map[string]string{
				workflowengine.TelemetryRootWorkflowIDKey: "root-wf",
				workflowengine.TelemetryRootRunIDKey:      "root-run",
				"step_id":                                 "validate",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			env.RegisterActivityWithOptions(
				tc.activity.Execute,
				activity.RegisterOptions{Name: tc.activity.Name()},
			)

			workflowName := "execute-step-activity"
			env.RegisterWorkflowWithOptions(
				func(ctx workflow.Context) (map[string]any, error) {
					ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
					ctx = workflow.WithActivityOptions(ctx, ao)
					step := tc.step
					output, err := Execute(&step, ctx, globalCfg, map[string]any{}, ao)
					if err != nil {
						return nil, err
					}
					return output.(map[string]any), nil
				},
				workflow.RegisterOptions{Name: workflowName},
			)

			var gotConfig map[string]string
			env.OnActivity(tc.activity.Name(), mock.Anything, mock.Anything).Return(
				func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
					gotConfig = input.Config
					return workflowengine.ActivityResult{Output: map[string]any{"body": "ok"}}, nil
				},
			)

			env.ExecuteWorkflow(workflowName)
			require.True(t, env.IsWorkflowCompleted())
			require.NoError(t, env.GetWorkflowError())

			var result map[string]any
			require.NoError(t, env.GetWorkflowResult(&result))
			require.Equal(t, "ok", result["body"])
			require.Equal(t, tc.wantConfig, gotConfig)
			require.NotContains(t, tc.step.With.Config, "step_id")
		})
	}
}

func TestExecuteStepRejectsOversizedInput(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	// The 4 MiB input is decoded and measured in workflow code. Under -race on a
	// contended CI runner that takes more than the SDK's default 1 s deadlock
	// detection timeout, which then fails the workflow with TMPRL1101 before the
	// size guard's CRE229 error can surface. The guard, not that timing, is what
	// this test checks.
	env.SetWorkerOptions(worker.Options{DeadlockDetectionTimeout: 30 * time.Second})

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	scheduled := 0
	env.SetOnActivityStartedListener(
		func(*activity.Info, context.Context, converter.EncodedValues) { scheduled++ },
	)

	workflowName := "execute-step-oversized"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			return ExecuteStep(
				"big",
				"http-request",
				pipeline.StepInputs{Payload: map[string]any{
					"url":    "https://example.com",
					"method": "POST",
					"body":   strings.Repeat("a", maxStepInputBytes+1),
				}},
				nil,
				ctx,
				map[string]any{},
				map[string]any{},
				ao,
			)
		},
		workflow.RegisterOptions{Name: workflowName},
	)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.PipelineExecutionError].Code)
	require.Contains(t, err.Error(), "step big input is")
	require.Zero(t, scheduled)
}

func TestRunChildPipelineSuccess(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	childYAML := `
name: child-pipeline
steps: []
`

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
			})
			step := pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID:  "child",
					Use: "custom",
					With: pipeline.StepInputs{
						Payload: map[string]any{"pipeline_id": "tenant/child"},
						Config:  map[string]any{},
					},
				},
			}
			input := PipelineWorkflowInput{
				WorkflowInput: workflowengine.WorkflowInput{
					Config: map[string]any{"namespace": "tenant"},
				},
			}

			output, err := runChildPipeline(
				ctx,
				step,
				input,
				"child-workflow",
				map[string]any{},
				&workflowengine.WorkflowRunMetadata{},
				[]string{"inner"},
			)
			if err != nil {
				return nil, err
			}
			result, _ := output.(map[string]any)
			return result, nil
		},
		workflow.RegisterOptions{Name: "parent-workflow"},
	)

	var childInput PipelineWorkflowInput
	env.RegisterWorkflowWithOptions(
		func(_ workflow.Context, input PipelineWorkflowInput) (workflowengine.WorkflowResult, error) {
			childInput = input
			return workflowengine.WorkflowResult{
				Output: map[string]any{"child": true},
			}, nil
		},
		workflow.RegisterOptions{Name: "child-workflow"},
	)

	registerStubActivity(env, activities.ResolveRecordActivityName)
	env.OnActivity(
		activities.ResolveRecordActivityName,
		mock.Anything,
		resolveRecordInputMatcher(activities.ResolveRecordInput{
			CanonifiedName: "tenant/child",
			Collection:     "pipelines",
			OwnerNamespace: "tenant",
		}),
	).
		Return(workflowengine.ActivityResult{Output: map[string]any{"yaml": childYAML}}, nil).
		Once()

	env.ExecuteWorkflow("parent-workflow")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, map[string]any{"child": true}, result)
	require.Equal(t, []string{"inner"}, childInput.ReturnOutputs)
	require.Equal(t, "child", childInput.WorkflowInput.Config[workflowengine.StepIDConfigKey])
}

func TestFetchChildPipelineYAMLValidationErrors(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (string, error) {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
			})
			step := pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID:  "child",
					Use: "custom",
					With: pipeline.StepInputs{
						Payload: map[string]any{},
					},
				},
			}
			input := PipelineWorkflowInput{
				WorkflowInput: workflowengine.WorkflowInput{
					Config: map[string]any{"namespace": "tenant"},
				},
			}
			_, err := fetchChildPipelineYAML(
				ctx,
				step,
				input,
				&workflowengine.WorkflowRunMetadata{},
			)
			if err == nil {
				return "", errors.New("expected error")
			}
			return workflowengine.ParseWorkflowError(err).Code, nil
		},
		workflow.RegisterOptions{Name: "fetch-missing-id"},
	)

	env.ExecuteWorkflow("fetch-missing-id")
	require.NoError(t, env.GetWorkflowError())
	var errCode string
	require.NoError(t, env.GetWorkflowResult(&errCode))
	require.Equal(t, errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code, errCode)
}

func TestFetchChildPipelineYAMLInvalidOutput(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (string, error) {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
			})
			step := pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID:  "child",
					Use: "custom",
					With: pipeline.StepInputs{
						Payload: map[string]any{"pipeline_id": "tenant/child"},
					},
				},
			}
			input := PipelineWorkflowInput{
				WorkflowInput: workflowengine.WorkflowInput{
					Config: map[string]any{"namespace": "tenant"},
				},
			}
			_, err := fetchChildPipelineYAML(
				ctx,
				step,
				input,
				&workflowengine.WorkflowRunMetadata{},
			)
			if err == nil {
				return "", errors.New("expected error")
			}
			return err.Error(), nil
		},
		workflow.RegisterOptions{Name: "fetch-invalid-output"},
	)

	registerStubActivity(env, activities.ResolveRecordActivityName)
	env.OnActivity(activities.ResolveRecordActivityName, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{"yaml": 123}}, nil).
		Once()

	env.ExecuteWorkflow("fetch-invalid-output")
	require.NoError(t, env.GetWorkflowError())
	var errMsg string
	require.NoError(t, env.GetWorkflowResult(&errMsg))
	require.Contains(t, errMsg, "invalid resolved pipeline output")
}

func TestFetchChildPipelineYAML(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	registerStubActivity(env, activities.ResolveRecordActivityName)

	workflowName := "fetch-child-yaml"
	fetchChildPipelineYAMLWorkflow := func(
		ctx workflow.Context,
		step pipeline.StepDefinition,
		input PipelineWorkflowInput,
	) (string, error) {
		ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
		ctx = workflow.WithActivityOptions(ctx, ao)
		meta := &workflowengine.WorkflowRunMetadata{}
		return fetchChildPipelineYAML(ctx, step, input, meta)
	}
	env.RegisterWorkflowWithOptions(
		fetchChildPipelineYAMLWorkflow,
		workflow.RegisterOptions{Name: workflowName},
	)

	env.OnActivity(
		activities.ResolveRecordActivityName,
		mock.Anything,
		resolveRecordInputMatcher(activities.ResolveRecordInput{
			CanonifiedName: "pipeline-1",
			Collection:     "pipelines",
			OwnerNamespace: "tenant",
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{"yaml": "yaml-body"}}, nil)

	step := pipeline.StepDefinition{
		StepSpec: pipeline.StepSpec{
			ID:  "step-1",
			Use: "child-pipeline",
			With: pipeline.StepInputs{
				Payload: map[string]any{
					"pipeline_id": "pipeline-1",
				},
			},
		},
	}
	input := PipelineWorkflowInput{
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{"namespace": "tenant"},
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
			config:        map[string]any{"namespace": "tenant"},
			expectMessage: "missing pipeline_id",
		},
		{
			name:          "missing namespace",
			stepPayload:   map[string]any{"pipeline_id": "pipeline-1"},
			config:        map[string]any{},
			expectMessage: "namespace",
		},
		{
			name:          "invalid resolved output",
			stepPayload:   map[string]any{"pipeline_id": "pipeline-1"},
			config:        map[string]any{"namespace": "tenant"},
			activityBody:  123,
			expectMessage: "invalid resolved pipeline output",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()

			if tc.activityBody != nil {
				registerStubActivity(env, activities.ResolveRecordActivityName)
			}

			workflowName := "fetch-child-yaml-errors"
			fetchChildPipelineYAMLWorkflow := func(
				ctx workflow.Context,
				step pipeline.StepDefinition,
				input PipelineWorkflowInput,
			) (string, error) {
				ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
				ctx = workflow.WithActivityOptions(ctx, ao)
				meta := &workflowengine.WorkflowRunMetadata{}
				return fetchChildPipelineYAML(ctx, step, input, meta)
			}
			env.RegisterWorkflowWithOptions(
				fetchChildPipelineYAMLWorkflow,
				workflow.RegisterOptions{Name: workflowName},
			)

			if tc.activityBody != nil {
				env.OnActivity(
					activities.ResolveRecordActivityName,
					mock.Anything,
					mock.Anything,
				).Return(
					workflowengine.ActivityResult{Output: map[string]any{"yaml": tc.activityBody}},
					nil,
				)
			}

			step := pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID:  "step-1",
					Use: "child-pipeline",
					With: pipeline.StepInputs{
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

func TestExecuteStepResolveInputsError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	workflowName := "execute-step-resolve-error"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			step := pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID:  "step-1",
					Use: "http-request",
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"url": "${{inputs.missing}}",
						},
					},
				},
			}
			_, err := ExecuteStep(
				step.ID,
				step.Use,
				step.With,
				step.ActivityOptions,
				ctx,
				map[string]any{},
				map[string]any{},
				ao,
			)
			return err
		},
		workflow.RegisterOptions{Name: workflowName},
	)

	env.ExecuteWorkflow(workflowName)
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "error resolving inputs")
}

func TestExecuteStepNonExecutableActivity(t *testing.T) {
	orig, ok := registry.Registry["non-exec"]
	if ok {
		t.Cleanup(func() { registry.Registry["non-exec"] = orig })
	} else {
		t.Cleanup(func() { delete(registry.Registry, "non-exec") })
	}

	registry.Registry["non-exec"] = registry.TaskFactory{
		Kind:        registry.TaskActivity,
		NewFunc:     func(core.App) any { return &fakeActivity{name: "non-exec"} },
		PayloadType: reflect.TypeOf(map[string]any{}),
		OutputKind:  workflowengine.OutputMap,
	}

	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	workflowName := "execute-step-non-exec"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			step := pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID:  "step-1",
					Use: "non-exec",
					With: pipeline.StepInputs{
						Payload: map[string]any{"foo": "bar"},
					},
				},
			}
			_, err := ExecuteStep(
				step.ID,
				step.Use,
				step.With,
				step.ActivityOptions,
				ctx,
				map[string]any{},
				map[string]any{},
				ao,
			)
			return err
		},
		workflow.RegisterOptions{Name: workflowName},
	)

	env.ExecuteWorkflow(workflowName)
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "not executable")
}

func TestExecuteStepEmailConfigureError(t *testing.T) {
	orig := registry.Registry["email"]
	t.Cleanup(func() { registry.Registry["email"] = orig })

	registry.Registry["email"] = registry.TaskFactory{
		Kind: registry.TaskActivity,
		NewFunc: func(core.App) any {
			return &fakeConfigActivity{
				fakeActivity: fakeActivity{name: "email"},
				configErr:    errors.New("bad config"),
			}
		},
		PayloadType: reflect.TypeOf(map[string]any{}),
		OutputKind:  workflowengine.OutputMap,
	}

	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	workflowName := "execute-step-email-config-error"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			step := pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID:  "step-1",
					Use: "email",
					With: pipeline.StepInputs{
						Payload: map[string]any{"foo": "bar"},
					},
				},
			}
			_, err := ExecuteStep(
				step.ID,
				step.Use,
				step.With,
				step.ActivityOptions,
				ctx,
				map[string]any{},
				map[string]any{},
				ao,
			)
			return err
		},
		workflow.RegisterOptions{Name: workflowName},
	)

	env.ExecuteWorkflow(workflowName)
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "error configuring activity")
}

func TestEnsureStepInputSizeCountsEncryptedSecrets(t *testing.T) {
	const secretsBytes = 200 << 10
	input := workflowengine.ActivityInput{
		Payload: map[string]any{
			"data": strings.Repeat("a", maxStepInputBytes-secretsBytes-(16<<10)),
		},
		Secrets: map[string]any{"token": strings.Repeat("s", secretsBytes)},
	}

	plain, err := json.Marshal(input)
	require.NoError(t, err)
	require.LessOrEqual(t, len(plain), maxStepInputBytes)

	err = ensureStepInputSize("big-step", input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "step big-step input is")
}

func TestEnsureStepInputSizeAcceptsSmallInput(t *testing.T) {
	input := workflowengine.ActivityInput{
		Payload: map[string]any{"data": "small"},
		Secrets: map[string]any{"token": "secret"},
	}

	require.NoError(t, ensureStepInputSize("small-step", input))
}

func TestEncodedStepInputSizeDecidesLikeTheEncodedPayload(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		secret  string
	}{
		{
			name:    "plain input under the limit",
			payload: strings.Repeat("a", maxStepInputBytes-(4<<10)),
		},
		{name: "plain input over the limit", payload: strings.Repeat("a", maxStepInputBytes+1)},
		{
			name:    "large secrets within the bound",
			payload: strings.Repeat("a", 700<<10),
			secret:  strings.Repeat("s", 1500<<10),
		},
		{
			name:    "escaped secrets within the bound",
			payload: strings.Repeat("a", 1200<<10),
			secret:  strings.Repeat("<", 100<<10),
		},
		{
			name:    "secrets that only encryption pushes over the limit",
			payload: strings.Repeat("a", maxStepInputBytes-(216<<10)),
			secret:  strings.Repeat("s", 200<<10),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := workflowengine.ActivityInput{Payload: map[string]any{"data": tc.payload}}
			if tc.secret != "" {
				input.Secrets = map[string]any{"token": tc.secret}
			}
			payloads, err := temporalcrypto.DataConverter().ToPayloads(input)
			require.NoError(t, err)
			exact := proto.Size(payloads)

			got, err := encodedStepInputSize(input)
			require.NoError(t, err)
			require.Equal(t, exact > maxStepInputBytes, got > maxStepInputBytes,
				"exact %d bytes, measured %d bytes", exact, got)
		})
	}
}

func TestSummaryLineStripsLineBreaks(t *testing.T) {
	require.Equal(t, "step-1 (http-request)", summaryLine("step-1 (http-request)"))
	require.Equal(t, "step-1 (http-request)", summaryLine("step-1\r\n (http-\nrequest)\r"))
}

// executeStepOutcome is what a workflow running ExecuteStep reports, so a
// test also sees the output a failed step keeps.
type executeStepOutcome struct {
	Output       any    `json:"output"`
	Error        string `json:"error"`
	Code         string `json:"code"`
	NonRetryable bool   `json:"non_retryable"`
}

// registerRunStepOperation serves the Run step operation in env like a runner
// namespace does, starting step as the step workflow, and returns the inputs
// the operation received.
func registerRunStepOperation(
	env *testsuite.TestWorkflowEnvironment,
	step func(workflow.Context, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error),
) *[]mobiledevice.RunStepInput {
	inputs := &[]mobiledevice.RunStepInput{}
	env.RegisterWorkflowWithOptions(
		step,
		workflow.RegisterOptions{Name: mobiledevice.StepWorkflowName},
	)
	service := nexus.NewService(mobiledevice.ServiceName)
	if err := service.Register(temporalnexus.MustNewWorkflowRunOperationWithOptions(
		temporalnexus.WorkflowRunOperationOptions[
			mobiledevice.RunStepInput,
			workflowengine.WorkflowResult,
		]{
			Name: mobiledevice.RunStepOperationName,
			Handler: func(
				ctx context.Context,
				in mobiledevice.RunStepInput,
				options nexus.StartOperationOptions,
			) (temporalnexus.WorkflowHandle[workflowengine.WorkflowResult], error) {
				*inputs = append(*inputs, in)
				return temporalnexus.ExecuteUntypedWorkflow[workflowengine.WorkflowResult](
					ctx,
					options,
					client.StartWorkflowOptions{ID: in.WorkflowID},
					mobiledevice.StepWorkflowName,
					in.Input,
				)
			},
		},
	)); err != nil {
		panic(err)
	}
	env.RegisterNexusService(service)
	return inputs
}

func TestExecuteStepRunsMobileStepInRunnerNamespace(t *testing.T) {
	pipelineRun := map[string]any{
		"namespace":   "tenant",
		"workflow_id": "pipeline-1",
		"run_id":      "run-1",
	}
	stepConfig := func(without string) map[string]any {
		config := map[string]any{
			"taskqueue":                       "tenant/runner-1-TaskQueue",
			workflowengine.StepIDConfigKey:    "step-1",
			runnerNamespaceConfigKey:          "runner-abc",
			pipelineRunConfigKey:              pipelineRun,
			pipelineExecutionTimeoutConfigKey: "1h0m0s",
		}
		delete(config, without)
		return config
	}
	missingCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code
	failureCode := errorcodes.Codes[errorcodes.PipelineExecutionError].Code

	tests := []struct {
		name             string
		config           map[string]any
		result           workflowengine.WorkflowResult
		err              error
		wantCalls        int
		wantOutput       any
		wantCode         string
		wantError        string
		wantNonRetryable bool
	}{
		{
			name:   "returns the workflow output",
			config: stepConfig(""),
			result: workflowengine.WorkflowResult{
				Output: map[string]any{"result_video_urls": []any{"https://video"}},
			},
			wantCalls:  1,
			wantOutput: map[string]any{"result_video_urls": []any{"https://video"}},
		},
		{
			name:   "keeps the output of a failed workflow",
			config: stepConfig(""),
			err: workflowengine.NewNonRetryableAppError(workflowengine.WorkflowError{
				Code:    failureCode,
				Message: "flow failed",
				Details: map[string]any{
					"output": map[string]any{"screenshot_urls": []any{"https://shot"}},
				},
			}),
			wantCalls:        1,
			wantOutput:       map[string]any{"screenshot_urls": []any{"https://shot"}},
			wantCode:         failureCode,
			wantError:        "flow failed",
			wantNonRetryable: true,
		},
		{
			name:             "rejects a step without runner namespace",
			config:           stepConfig(runnerNamespaceConfigKey),
			wantCode:         missingCode,
			wantError:        "missing runner_namespace for step step-1",
			wantNonRetryable: true,
		},
		{
			name:             "rejects a step without pipeline run",
			config:           stepConfig(pipelineRunConfigKey),
			wantCode:         missingCode,
			wantError:        "missing pipeline_run for step step-1",
			wantNonRetryable: true,
		},
		{
			name:             "rejects a step without execution timeout",
			config:           stepConfig(pipelineExecutionTimeoutConfigKey),
			wantCode:         missingCode,
			wantError:        "missing pipeline_execution_timeout for step step-1",
			wantNonRetryable: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: "pipeline-1"})

			nexusOptions := recordNexusOptions(env)
			inputs := registerRunStepOperation(
				env,
				func(workflow.Context, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
					return tt.result, tt.err
				},
			)
			env.RegisterWorkflowWithOptions(
				func(ctx workflow.Context) (executeStepOutcome, error) {
					output, err := ExecuteStep(
						"step-1",
						mobileAutomationStepUse,
						pipeline.StepInputs{
							Config: tt.config,
							Payload: map[string]any{
								"action_id": "action-1",
								"device_id": "runner-1/device-1",
							},
						},
						nil,
						ctx,
						map[string]any{"app_url": "https://example.test"},
						map[string]any{},
						workflow.ActivityOptions{StartToCloseTimeout: time.Second},
					)
					outcome := executeStepOutcome{Output: output}
					if err != nil {
						outcome.Error = err.Error()
						outcome.Code = workflowengine.ParseWorkflowError(err).Code
						var appErr *temporal.ApplicationError
						outcome.NonRetryable = errors.As(err, &appErr) && appErr.NonRetryable()
					}
					return outcome, nil
				},
				workflow.RegisterOptions{Name: "execute-mobile-step"},
			)

			env.ExecuteWorkflow("execute-mobile-step")
			require.NoError(t, env.GetWorkflowError())
			var outcome executeStepOutcome
			require.NoError(t, env.GetWorkflowResult(&outcome))

			assert.Equal(t, tt.wantCode, outcome.Code)
			assert.Equal(t, tt.wantNonRetryable, outcome.NonRetryable)
			assert.Contains(t, outcome.Error, tt.wantError)
			require.Len(t, *inputs, tt.wantCalls)
			if tt.wantCalls == 0 {
				return
			}
			require.Len(t, *nexusOptions, 1)
			assert.Equal(
				t,
				deviceOperationScheduleToStart,
				(*nexusOptions)[0].ScheduleToStartTimeout,
			)
			assert.Equal(t, time.Hour, (*nexusOptions)[0].ScheduleToCloseTimeout)
			assert.Equal(t, tt.wantOutput, outcome.Output)

			in := (*inputs)[0]
			assert.Equal(t, "pipeline-1-step-1", in.WorkflowID)
			assert.Equal(t, time.Hour, in.ExecutionTimeout)
			assert.Equal(t, mobiledevice.PipelineRun{
				Namespace:  "tenant",
				WorkflowID: "pipeline-1",
				RunID:      "run-1",
			}, in.Pipeline)
			assert.Equal(t, "tenant/runner-1-TaskQueue", in.Input.Config["taskqueue"])
			assert.Equal(t, "https://example.test", in.Input.Config["app_url"])
			assert.Equal(t, "step-1", in.Input.Config[workflowengine.StepIDConfigKey])
		})
	}
}
