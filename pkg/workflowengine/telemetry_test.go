// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type telemetryStringer string

func (s telemetryStringer) String() string { return string(s) }

func telemetryConfigWorkflow(ctx workflow.Context, cfg map[string]any) (map[string]string, error) {
	return ActivityTelemetryConfig(ctx, cfg), nil
}

func TestActivityTelemetryConfig(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: "telemetry-workflow"})

	input := map[string]any{
		TelemetryRootWorkflowIDKey: "original-root",
		"attempt":                  3,
		"empty":                    nil,
	}
	env.ExecuteWorkflow(telemetryConfigWorkflow, input)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]string
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "original-root", result[TelemetryRootWorkflowIDKey])
	require.NotEmpty(t, result[TelemetryRootRunIDKey])
	require.Equal(t, "telemetry-workflow", result[TelemetryParentWorkflowIDKey])
	require.Equal(t, result[TelemetryRootRunIDKey], result[TelemetryParentRunIDKey])
	require.Equal(t, "telemetry-workflow", result[TelemetryCurrentWorkflowIDKey])
	require.Equal(t, result[TelemetryRootRunIDKey], result[TelemetryCurrentRunIDKey])
	require.Equal(t, "3", result["attempt"])
	require.NotContains(t, result, "empty")
	require.Equal(t, "original-root", input[TelemetryRootWorkflowIDKey])
	require.NotContains(t, input, TelemetryCurrentWorkflowIDKey)
}

func TestStringConfigValue(t *testing.T) {
	tests := []struct {
		name string
		cfg  map[string]any
		want string
	}{
		{name: "nil config", want: "fallback"},
		{name: "missing value", cfg: map[string]any{}, want: "fallback"},
		{name: "nil value", cfg: map[string]any{"key": nil}, want: "fallback"},
		{name: "string", cfg: map[string]any{"key": "value"}, want: "value"},
		{name: "empty string", cfg: map[string]any{"key": ""}, want: "fallback"},
		{
			name: "stringer",
			cfg:  map[string]any{"key": telemetryStringer("stringer")},
			want: "stringer",
		},
		{
			name: "empty stringer",
			cfg:  map[string]any{"key": telemetryStringer("")},
			want: "fallback",
		},
		{name: "number", cfg: map[string]any{"key": 42}, want: "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, stringConfigValue(tt.cfg, "key", "fallback"))
		})
	}
}

func TestCopyAnyMap(t *testing.T) {
	require.Empty(t, copyAnyMap(nil))

	source := map[string]any{"key": fmt.Stringer(telemetryStringer("value"))}
	copied := copyAnyMap(source)
	require.Equal(t, source, copied)
	copied["key"] = "changed"
	require.NotEqual(t, source, copied)
}
