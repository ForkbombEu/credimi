// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestStepDefinition_UnmarshalYAML(t *testing.T) {
	yamlData := `
id: "step1"
use: "echo"
with:
  config:
    foo: "bar"
  scalar_input: "hello"
  map_input:
    key1: "value1"
    key2: 42
  list_input:
    - 1
    - 2
    - 3
activity_options:
  schedule_to_close_timeout: "15m"
  start_to_close_timeout: "10m"
  retry_policy:
    maximum_attempts: 3
    initial_interval: "1s"
    maximum_interval: "10s"
    backoff_coefficient: 2.0
metadata:
  note: "example step"
`

	var step StepDefinition
	err := yaml.Unmarshal([]byte(yamlData), &step)
	require.NoError(t, err)

	// Top-level fields
	require.Equal(t, "step1", step.ID)
	require.Equal(t, "echo", step.Use)
	require.NotNil(t, step.ActivityOptions)
	require.Equal(t, "15m", step.ActivityOptions.ScheduleToCloseTimeout)
	require.Equal(t, "10m", step.ActivityOptions.StartToCloseTimeout)
	require.Equal(t, int32(3), step.ActivityOptions.RetryPolicy.MaximumAttempts)
	require.Equal(t, "1s", step.ActivityOptions.RetryPolicy.InitialInterval)
	require.Equal(t, "10s", step.ActivityOptions.RetryPolicy.MaximumInterval)
	require.Equal(t, 2.0, step.ActivityOptions.RetryPolicy.BackoffCoefficient)
	require.Equal(t, map[string]any{"note": "example step"}, step.Metadata)

	// StepInputs.Config
	require.Equal(t, map[string]any{"foo": "bar"}, step.With.Config)

	// scalar input
	require.Equal(t, "hello", step.With.Payload["scalar_input"])

	// map input
	expectedMap := map[string]any{"key1": "value1", "key2": 42}
	require.Equal(t, expectedMap, step.With.Payload["map_input"])

	// list input
	require.Equal(t, []any{1, 2, 3}, step.With.Payload["list_input"])
}

func TestWorkflowDefinition_UnmarshalYAML(t *testing.T) {
	yml := `
config:
  globalKey: globalVal
steps:
  - use: step1
    with:
      config:
        apiKey: "abc123"
      url: "http://example.com"
    activity_options:
      schedule_to_close_timeout: "20m"
      start_to_close_timeout: "15m"
      retry_policy:
        maximum_attempts: 5
  - use: step2
    with:
      config:
        image: "alpine:latest"
      args:
        - run
        - echo
`

	var wf WorkflowDefinition
	err := yaml.Unmarshal([]byte(yml), &wf)
	require.NoError(t, err)

	// Global config
	require.Equal(t, "globalVal", wf.Config["globalKey"])
	require.Len(t, wf.Steps, 2)

	// Step 1
	s1 := wf.Steps[0]
	require.Equal(t, "step1", s1.Use)
	require.Equal(t, "abc123", s1.With.Config["apiKey"])

	require.Equal(t, "http://example.com", s1.With.Payload["url"])

	require.NotNil(t, s1.ActivityOptions)
	require.Equal(t, "20m", s1.ActivityOptions.ScheduleToCloseTimeout)
	require.Equal(t, "15m", s1.ActivityOptions.StartToCloseTimeout)
	require.Equal(t, int32(5), s1.ActivityOptions.RetryPolicy.MaximumAttempts)

	// Step 2
	s2 := wf.Steps[1]
	require.Equal(t, "step2", s2.Use)
	require.Equal(t, "alpine:latest", s2.With.Config["image"])

	require.Equal(t, []any{"run", "echo"}, s2.With.Payload["args"])

	require.Nil(t, s2.ActivityOptions)
}
