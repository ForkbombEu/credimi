// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// scriptedActivity is a step activity whose result a test computes per call.
type scriptedActivity struct {
	*runtimeCaptureActivity
	fn func(runtimeCapturePayload) (map[string]any, error)
}

func (a *scriptedActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[runtimeCapturePayload](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}
	output, err := a.fn(payload)
	return workflowengine.ActivityResult{Output: output}, err
}

func registerScriptedActivity(
	t *testing.T,
	env *testsuite.TestWorkflowEnvironment,
	name string,
	fn func(runtimeCapturePayload) (map[string]any, error),
) {
	t.Helper()
	act := &scriptedActivity{runtimeCaptureActivity: &runtimeCaptureActivity{name: name}, fn: fn}
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: name})
	orig, hadOrig := registry.Registry[name]
	t.Cleanup(func() {
		if hadOrig {
			registry.Registry[name] = orig
			return
		}
		delete(registry.Registry, name)
	})
	registry.Registry[name] = registry.TaskFactory{
		Kind:        registry.TaskActivity,
		NewFunc:     func(core.App) any { return act },
		PayloadType: reflect.TypeOf(runtimeCapturePayload{}),
		OutputKind:  workflowengine.OutputMap,
	}
}

func newLinkedStepsTestEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)
	linkedWf := NewLinkedStepsWorkflow()
	env.RegisterWorkflowWithOptions(
		linkedWf.Workflow,
		workflow.RegisterOptions{Name: linkedWf.Name()},
	)
	origSetupHooks := setupHooks
	origCleanupHooks := cleanupHooks
	setupHooks = []SetupFunc{}
	cleanupHooks = []CleanupFunc{}
	t.Cleanup(func() {
		setupHooks = origSetupHooks
		cleanupHooks = origCleanupHooks
	})
	return env
}

func linkedTestStep(id, use, text, linkedTo string) pipeline.StepDefinition {
	return pipeline.StepDefinition{
		StepSpec: pipeline.StepSpec{
			ID:   id,
			Use:  use,
			With: pipeline.StepInputs{Payload: map[string]any{"text": text}},
		},
		LinkedTo: linkedTo,
	}
}

func runLinkedTestPipeline(
	env *testsuite.TestWorkflowEnvironment,
	steps []pipeline.StepDefinition,
	maximumAttempts int32,
) {
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: "pipeline-wf"})
	env.ExecuteWorkflow(
		NewPipelineWorkflow().Name(),
		PipelineWorkflowInput{
			WorkflowDefinition: &pipeline.WorkflowDefinition{Name: "linked", Steps: steps},
			WorkflowInput: workflowengine.WorkflowInput{
				Config: map[string]any{"app_url": "https://example.test"},
				ActivityOptions: &workflow.ActivityOptions{
					StartToCloseTimeout: time.Second,
					// The server defaults a zero initial interval to 1s; the test
					// environment does not, and a zero backoff never retries.
					RetryPolicy: &temporal.RetryPolicy{
						InitialInterval: time.Second,
						MaximumAttempts: maximumAttempts,
					},
				},
			},
		},
	)
}

func countLinkedStepsChildren(env *testsuite.TestWorkflowEnvironment) *int {
	count := 0
	env.SetOnChildWorkflowStartedListener(
		func(info *workflow.Info, _ workflow.Context, _ converter.EncodedValues) {
			if info.WorkflowType.Name == pipeline.LinkedStepsWorkflowName {
				count++
			}
		},
	)
	return &count
}

func TestLinkedStepsRunAsOneBlock(t *testing.T) {
	env := newLinkedStepsTestEnv(t)
	captured := []string{}
	registerRuntimeCaptureActivity(t, env, "capture-runtime", &captured)
	children := countLinkedStepsChildren(env)

	runLinkedTestPipeline(env, []pipeline.StepDefinition{
		linkedTestStep("offer", "capture-runtime", "offer", ""),
		linkedTestStep("use", "capture-runtime", "${{ offer.outputs.text }}", "offer"),
		linkedTestStep("after", "capture-runtime", "${{ use.outputs.text }}", ""),
	}, 3)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, []string{"offer", "offer", "offer"}, captured)
	require.Equal(t, 1, *children)
}

func TestLinkedStepsRetryFromTheProducer(t *testing.T) {
	env := newLinkedStepsTestEnv(t)
	offers := 0
	registerScriptedActivity(
		t,
		env,
		"linked-offer",
		func(runtimeCapturePayload) (map[string]any, error) {
			offers++
			return map[string]any{"text": fmt.Sprintf("offer-%d", offers)}, nil
		},
	)
	consumed := []string{}
	registerScriptedActivity(
		t,
		env,
		"linked-consume",
		func(p runtimeCapturePayload) (map[string]any, error) {
			consumed = append(consumed, p.Text)
			if len(consumed) == 1 {
				return nil, errors.New("deeplink rejected")
			}
			return map[string]any{"text": p.Text}, nil
		},
	)
	captured := []string{}
	registerRuntimeCaptureActivity(t, env, "capture-runtime", &captured)
	starts := map[string]int{}
	env.SetOnActivityStartedListener(
		func(info *activity.Info, _ context.Context, _ converter.EncodedValues) {
			starts[info.ActivityType.Name]++
		},
	)
	children := countLinkedStepsChildren(env)

	runLinkedTestPipeline(env, []pipeline.StepDefinition{
		linkedTestStep("offer", "linked-offer", "x", ""),
		linkedTestStep("use", "linked-consume", "${{ offer.outputs.text }}", "offer"),
		linkedTestStep("after", "capture-runtime", "${{ use.outputs.text }}", ""),
	}, 2)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 2, offers)
	require.Equal(t, []string{"offer-1", "offer-2"}, consumed)
	require.Equal(t, []string{"offer-2"}, captured)
	// Each attempt runs every inner activity once: the block, not the activity, retries.
	require.Equal(t, 2, starts["linked-offer"])
	require.Equal(t, 2, starts["linked-consume"])
	require.Equal(t, 2, *children)
}

func TestLinkedStepsBlockRetryPolicy(t *testing.T) {
	for _, tc := range []struct {
		name      string
		lastRetry int32
		want      int32
	}{
		{name: "runtime policy", want: 2},
		{name: "last step policy", lastRetry: 4, want: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newLinkedStepsTestEnv(t)
			var info *workflow.Info
			var received LinkedStepsWorkflowInput
			env.OnWorkflow(pipeline.LinkedStepsWorkflowName, mock.Anything, mock.Anything).Return(
				func(ctx workflow.Context, input LinkedStepsWorkflowInput) (
					workflowengine.WorkflowResult,
					error,
				) {
					info = workflow.GetInfo(ctx)
					received = input
					return workflowengine.WorkflowResult{Output: map[string]any{
						"offer": map[string]any{"outputs": "o"},
						"use":   map[string]any{"outputs": "u"},
					}}, nil
				},
			)
			use := linkedTestStep("use", "capture-runtime", "${{ offer.outputs }}", "offer")
			if tc.lastRetry > 0 {
				use.ActivityOptions = &pipeline.ActivityOptionsConfig{
					RetryPolicy: pipeline.RetryPolicy{MaximumAttempts: tc.lastRetry},
				}
			}
			captured := []string{}
			registerRuntimeCaptureActivity(t, env, "capture-runtime", &captured)

			runLinkedTestPipeline(env, []pipeline.StepDefinition{
				linkedTestStep("before", "capture-runtime", "before", ""),
				linkedTestStep("offer", "capture-runtime", "${{ before.outputs.text }}", ""),
				use,
				linkedTestStep("after", "capture-runtime", "${{ use.outputs }}", ""),
			}, 2)

			require.NoError(t, env.GetWorkflowError())
			require.Equal(t, []string{"before", "u"}, captured)
			require.NotNil(t, info)
			require.Equal(t, "pipeline-wf-offer", info.WorkflowExecution.ID)
			require.Equal(t, tc.want, received.MaximumAttempts)
			require.Equal(t, []string{"before"}, slices.Sorted(maps.Keys(received.PriorOutputs)))
			require.NotContains(
				t,
				received.PipelineConfig,
				workflowengine.TelemetryParentWorkflowIDKey,
			)
		})
	}
}

func TestLinkedStepsInputErrorIsNotRetried(t *testing.T) {
	env := newLinkedStepsTestEnv(t)
	captured := []string{}
	registerRuntimeCaptureActivity(t, env, "capture-runtime", &captured)
	children := countLinkedStepsChildren(env)

	runLinkedTestPipeline(env, []pipeline.StepDefinition{
		linkedTestStep("offer", "capture-runtime", "offer", ""),
		linkedTestStep("use", "capture-runtime", "${{ missing.outputs }}", "offer"),
	}, 3)

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Equal(t, 1, *children)
	require.Equal(t, []string{"offer"}, captured)
	errorsList := requireFailureErrors(t, err)
	require.Len(t, errorsList, 1)
	require.Equal(t, "CRE228", errorsList[0]["code"])
	require.Equal(t, "use", errorsList[0]["details"].(map[string]any)["step_id"])
}

func TestLinkedStepsContinueOnError(t *testing.T) {
	env := newLinkedStepsTestEnv(t)
	captured := []string{}
	registerRuntimeCaptureActivity(t, env, "capture-runtime", &captured)

	offer := linkedTestStep("offer", "capture-runtime", "offer", "")
	offer.ContinueOnError = true
	use := linkedTestStep("use", "capture-runtime", "${{ offer.outputs.text }}", "offer")
	use.ContinueOnError = true
	use.With.Payload["fail"] = true
	after := linkedTestStep("after", "capture-runtime", "after-${{ offer.outputs.text }}", "")
	needsUse := linkedTestStep("needs-use", "capture-runtime", "${{ use.outputs.text }}", "")
	needsUse.ContinueOnError = true

	runLinkedTestPipeline(env, []pipeline.StepDefinition{offer, use, after, needsUse}, 1)

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Equal(t, []string{"offer", "offer", "after-offer"}, captured)
	errorsList := requireFailureErrors(t, err)
	stepIDs := make([]string, 0, len(errorsList))
	for _, entry := range errorsList {
		stepIDs = append(stepIDs, entry["details"].(map[string]any)["step_id"].(string))
	}
	require.ElementsMatch(t, []string{"use", "needs-use"}, stepIDs)
}

func TestLinkedStepsSeeThePipelineRun(t *testing.T) {
	env := newLinkedStepsTestEnv(t)
	captured := []string{}
	registerRuntimeCaptureActivity(t, env, "capture-runtime", &captured)

	runLinkedTestPipeline(env, []pipeline.StepDefinition{
		linkedTestStep("offer", "capture-runtime", "offer", ""),
		linkedTestStep("use", "capture-runtime", "${{ workflow_id }}", "offer"),
	}, 1)

	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, []string{"offer", "pipeline-wf"}, captured)
}

func TestLinkedStepsDeleteScreenshotsOfRetriedAttempts(t *testing.T) {
	for _, tc := range []struct {
		name    string
		attempt int32
		deleted []activities.DeletePipelineStepScreenshotsInput
	}{
		{
			name:    "attempt before the last",
			attempt: 1,
			deleted: []activities.DeletePipelineStepScreenshotsInput{
				{RunIdentifier: "ns/run", FileNames: []string{"s_abc.png"}},
			},
		},
		{name: "last attempt", attempt: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			registerFailingWorkflow(
				t,
				env,
				mobileAutomationStepUse,
				"fake-mobile-automation",
				workflowengine.WorkflowError{
					Code:    "CRE201",
					Summary: "Maestro flow failed",
					Details: map[string]any{"output": map[string]any{
						"flow_output": map[string]any{
							"maestro_screenshot_urls": []any{
								"https://x/api/files/pipeline_results/r/s_abc.png",
							},
						},
					}},
				},
			)
			var deleted []activities.DeletePipelineStepScreenshotsInput
			env.RegisterActivityWithOptions(
				func(
					_ context.Context,
					input workflowengine.ActivityInput,
				) (workflowengine.ActivityResult, error) {
					payload, err := workflowengine.
						DecodePayload[activities.DeletePipelineStepScreenshotsInput](input.Payload)
					require.NoError(t, err)
					deleted = append(deleted, payload)
					return workflowengine.ActivityResult{}, nil
				},
				activity.RegisterOptions{
					Name: activities.DeletePipelineStepScreenshotsActivityName,
				},
			)
			env.RegisterWorkflowWithOptions(
				func(ctx workflow.Context, input LinkedStepsWorkflowInput, attempt int32) (
					workflowengine.WorkflowResult,
					error,
				) {
					return NewLinkedStepsWorkflow().run(ctx, input, attempt)
				},
				workflow.RegisterOptions{Name: "linked-steps-run"},
			)

			env.ExecuteWorkflow("linked-steps-run", LinkedStepsWorkflowInput{
				Steps: []pipeline.StepDefinition{{StepSpec: pipeline.StepSpec{
					ID:   "wallet",
					Use:  mobileAutomationStepUse,
					With: pipeline.StepInputs{Config: map[string]any{"run_identifier": "ns/run"}},
				}}},
				PipelineConfig: map[string]any{},
				ActivityOptions: workflow.ActivityOptions{
					StartToCloseTimeout: time.Second,
				},
				MaximumAttempts: 2,
			}, tc.attempt)

			require.Error(t, env.GetWorkflowError())
			require.Equal(t, tc.deleted, deleted)
		})
	}
}

func TestLinkedBlockRetryable(t *testing.T) {
	require.True(t, linkedBlockRetryable(errors.New("plain")))
	require.True(t, linkedBlockRetryable(temporal.NewApplicationError("retry", "CRE1")))
	require.False(t, linkedBlockRetryable(
		temporal.NewNonRetryableApplicationError("input", "CRE228", nil),
	))
}
