// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/workflow"
)

func TestPrepareWorkflowOptionsDefaultsAndOverrides(t *testing.T) {
	defaultOpts := PrepareWorkflowOptions(pipeline.RuntimeConfig{})
	require.Equal(t, PipelineTaskQueue, defaultOpts.Options.TaskQueue)
	require.Equal(t, 24*time.Hour, defaultOpts.Options.WorkflowExecutionTimeout)
	require.Equal(t, 10*time.Minute, defaultOpts.ActivityOptions.ScheduleToCloseTimeout)
	require.Equal(t, 5*time.Minute, defaultOpts.ActivityOptions.StartToCloseTimeout)
	require.Equal(t, 30*time.Second, defaultOpts.ActivityOptions.HeartbeatTimeout)
	require.NotNil(t, defaultOpts.ActivityOptions.RetryPolicy)
	require.Equal(
		t,
		DefaultRetryMaxAttempts,
		defaultOpts.ActivityOptions.RetryPolicy.MaximumAttempts,
	)

	rc := pipeline.RuntimeConfig{}
	rc.Temporal.ExecutionTimeout = "1h"
	rc.Temporal.ActivityOptions.StartToCloseTimeout = "30s"
	rc.Temporal.ActivityOptions.ScheduleToCloseTimeout = "2m"
	rc.Temporal.ActivityOptions.RetryPolicy.MaximumAttempts = 3
	rc.Temporal.ActivityOptions.RetryPolicy.InitialInterval = "1s"
	rc.Temporal.ActivityOptions.RetryPolicy.MaximumInterval = "10s"
	rc.Temporal.ActivityOptions.RetryPolicy.BackoffCoefficient = 1.5

	overrides := PrepareWorkflowOptions(rc)
	require.Equal(t, time.Hour, overrides.Options.WorkflowExecutionTimeout)
	require.Equal(t, 2*time.Minute, overrides.ActivityOptions.ScheduleToCloseTimeout)
	require.Equal(t, 30*time.Second, overrides.ActivityOptions.StartToCloseTimeout)
	require.Equal(t, int32(3), overrides.ActivityOptions.RetryPolicy.MaximumAttempts)
	require.Equal(t, time.Second, overrides.ActivityOptions.RetryPolicy.InitialInterval)
	require.Equal(t, 10*time.Second, overrides.ActivityOptions.RetryPolicy.MaximumInterval)
	require.Equal(t, 1.5, overrides.ActivityOptions.RetryPolicy.BackoffCoefficient)
}

func TestPrepareActivityOptionsOverrides(t *testing.T) {
	global := PrepareWorkflowOptions(pipeline.RuntimeConfig{}).ActivityOptions

	stepAO := &pipeline.ActivityOptionsConfig{}
	stepAO.RetryPolicy.MaximumAttempts = 2
	stepAO.RetryPolicy.InitialInterval = "2s"
	stepAO.StartToCloseTimeout = "45s"

	out := PrepareActivityOptions(global, stepAO)
	require.Equal(t, int32(2), out.RetryPolicy.MaximumAttempts)
	require.Equal(t, 2*time.Second, out.RetryPolicy.InitialInterval)
	require.Equal(t, 45*time.Second, out.StartToCloseTimeout)
	require.Equal(t, global.HeartbeatTimeout, out.HeartbeatTimeout)
	require.Equal(t, global.ScheduleToCloseTimeout, out.ScheduleToCloseTimeout)
}

func TestPrepareActivityOptionsAddsDefaultHeartbeatTimeout(t *testing.T) {
	out := PrepareActivityOptions(
		workflow.ActivityOptions{
			ScheduleToCloseTimeout: 10 * time.Minute,
			StartToCloseTimeout:    5 * time.Minute,
		},
		nil,
	)

	require.Equal(t, 30*time.Second, out.HeartbeatTimeout)
}

func TestPrepareActivityOptionsDoesNotMutateGlobalRetryPolicy(t *testing.T) {
	global := PrepareWorkflowOptions(pipeline.RuntimeConfig{}).ActivityOptions
	before := *global.RetryPolicy

	stepAO := &pipeline.ActivityOptionsConfig{}
	stepAO.RetryPolicy.MaximumAttempts = 7
	stepAO.RetryPolicy.InitialInterval = "9s"
	stepAO.RetryPolicy.MaximumInterval = "90s"
	stepAO.RetryPolicy.BackoffCoefficient = 3

	out := PrepareActivityOptions(global, stepAO)
	require.Equal(t, int32(7), out.RetryPolicy.MaximumAttempts)
	require.NotSame(t, global.RetryPolicy, out.RetryPolicy)
	require.Equal(t, before, *global.RetryPolicy)
}

func TestPrepareActivityOptionsNilGlobalRetryPolicy(t *testing.T) {
	stepAO := &pipeline.ActivityOptionsConfig{}
	stepAO.RetryPolicy.MaximumAttempts = 4
	stepAO.RetryPolicy.InitialInterval = "1s"
	stepAO.RetryPolicy.MaximumInterval = "5s"
	stepAO.RetryPolicy.BackoffCoefficient = 1.5

	var out workflow.ActivityOptions
	require.NotPanics(t, func() {
		out = PrepareActivityOptions(workflow.ActivityOptions{}, stepAO)
	})
	require.NotNil(t, out.RetryPolicy)
	require.Equal(t, int32(4), out.RetryPolicy.MaximumAttempts)
	require.Equal(t, time.Second, out.RetryPolicy.InitialInterval)
	require.Equal(t, 5*time.Second, out.RetryPolicy.MaximumInterval)
	require.Equal(t, 1.5, out.RetryPolicy.BackoffCoefficient)
}

func TestPrepareActivityOptionsPreservesUnmergedFields(t *testing.T) {
	global := PrepareWorkflowOptions(pipeline.RuntimeConfig{}).ActivityOptions
	global.TaskQueue = "custom-queue"
	global.ScheduleToStartTimeout = 3 * time.Minute

	stepAO := &pipeline.ActivityOptionsConfig{StartToCloseTimeout: "45s"}

	out := PrepareActivityOptions(global, stepAO)
	require.Equal(t, "custom-queue", out.TaskQueue)
	require.Equal(t, 3*time.Minute, out.ScheduleToStartTimeout)
	require.Equal(t, 45*time.Second, out.StartToCloseTimeout)
}

func TestSetPayloadValueAndMergePayload(t *testing.T) {
	var payload map[string]any
	require.NoError(t, SetPayloadValue(&payload, "key", "value"))
	require.Equal(t, "value", payload["key"])

	err := SetPayloadValue(nil, "key", "value")
	require.Error(t, err)

	dst := map[string]any{
		"nested": map[string]any{"a": 1},
	}
	srcNested := map[string]any{"b": 2}
	src := map[string]any{
		"nested": srcNested,
		"list":   []any{map[string]any{"x": "y"}},
	}
	require.NoError(t, MergePayload(&dst, &src))

	srcNested["b"] = 3
	srcList := src["list"].([]any)[0].(map[string]any)
	srcList["x"] = "z"

	require.Equal(t, float64(2), dst["nested"].(map[string]any)["b"])
	require.Equal(t, "y", dst["list"].([]any)[0].(map[string]any)["x"])
}

func TestDeepCopyFallback(t *testing.T) {
	ch := make(chan int)
	copied := deepCopy(ch)
	require.Equal(t, ch, copied)
}

func TestRunIdentifier(t *testing.T) {
	got := RunIdentifier("acme", "workflow 1", "run 2")
	require.Contains(t, got, "acme/")
	require.NotContains(t, got, " ")
}
