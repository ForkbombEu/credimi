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
run: "echo"
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
retry:
  count: 3
timeout: "5m"
metadata:
  note: "example step"
`

	var step StepDefinition
	err := yaml.Unmarshal([]byte(yamlData), &step)
	require.NoError(t, err)
	// Top-level fields
	require.Equal(t, "step1", step.ID)
	require.Equal(t, "echo", step.Run)
	require.Equal(t, map[string]any{"count": 3}, step.Retry)
	require.Equal(t, "5m", step.Timeout)
	require.Equal(t, map[string]interface{}{"note": "example step"}, step.Metadata)

	// StepInputs
	require.Equal(t, map[string]string{"foo": "bar"}, step.With.Config)

	payload := step.With.Payload

	// scalar input
	require.Contains(t, payload, "scalar_input")
	require.Equal(t, "hello", payload["scalar_input"].Value)

	// map input
	require.Contains(t, payload, "map_input")
	expectedMap := map[string]any{"key1": "value1", "key2": 42}
	require.Equal(t, expectedMap, payload["map_input"].Value)

	// list input
	require.Contains(t, payload, "list_input")
	require.Equal(t, []any{1, 2, 3}, payload["list_input"].Value)
}

func TestWorkflowDefinition_UnmarshalYAML(t *testing.T) {
	yml := `
config:
  globalKey: globalVal
steps:
  - run: step1
    with:
      config:
        apiKey: "abc123"
      url: "http://example.com"
  - run: step2
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
	require.Equal(t, "step1", s1.Run)
	require.Equal(t, "abc123", s1.With.Config["apiKey"])
	require.Equal(t, "http://example.com", s1.With.Payload["url"].Value)

	// Step 2
	s2 := wf.Steps[1]
	require.Equal(t, "step2", s2.Run)
	require.Equal(t, "alpine:latest", s2.With.Config["image"])
	require.Equal(t, []any{"run", "echo"}, s2.With.Payload["args"].Value)
}
