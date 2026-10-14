// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"testing"
	"time"

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

func TestStepDefinition_UnmarshalYAML_WithPayloadBlock(t *testing.T) {
	yamlData := `
id: "step1"
use: "mobile_automation"
with:
  config:
    foo: "bar"
  payload:
    action_id: "onboarding-0001"
    version_id: "v1"
  version_id: "override"
`

	var step StepDefinition
	err := yaml.Unmarshal([]byte(yamlData), &step)
	require.NoError(t, err)

	require.Equal(t, map[string]any{"foo": "bar"}, step.With.Config)
	require.Equal(t, "onboarding-0001", step.With.Payload["action_id"])
	require.Equal(t, "override", step.With.Payload["version_id"])
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

func TestFinallyDefinition_UnmarshalYAML(t *testing.T) {
	t.Run("legacy list becomes always", func(t *testing.T) {
		yml := `
finally:
  - id: notify
    use: email
    with:
      payload:
        subject: done
`

		var wf WorkflowDefinition
		err := yaml.Unmarshal([]byte(yml), &wf)
		require.NoError(t, err)
		require.Len(t, wf.Finally.Always, 1)
		require.Equal(t, "notify", wf.Finally.Always[0].ID)
		require.Empty(t, wf.Finally.OnSuccess)
		require.Empty(t, wf.Finally.OnFailure)
	})

	t.Run("grouped handlers", func(t *testing.T) {
		yml := `
finally:
  always:
    - id: notify-any
      use: http-request
      with:
        payload:
          url: https://example.com/any
  on_success:
    - id: notify-success
      use: email
      with:
        payload:
          subject: success
  on_failure:
    - id: notify-failure
      use: email
      with:
        payload:
          subject: failure
`

		var wf WorkflowDefinition
		err := yaml.Unmarshal([]byte(yml), &wf)
		require.NoError(t, err)
		require.Len(t, wf.Finally.Always, 1)
		require.Len(t, wf.Finally.OnSuccess, 1)
		require.Len(t, wf.Finally.OnFailure, 1)
		require.Equal(t, "notify-any", wf.Finally.Always[0].ID)
		require.Equal(t, "notify-success", wf.Finally.OnSuccess[0].ID)
		require.Equal(t, "notify-failure", wf.Finally.OnFailure[0].ID)
	})
}

// A date-like scalar must reach the http-request body as the exact text the
// author wrote. yaml.v3 otherwise resolves an unquoted `2012-03-04` to
// time.Time, and the http-request activity marshals the body back to JSON as
// `2012-03-04T00:00:00Z`. The webapp pipeline serializer (`yaml` core schema)
// emits such values unquoted, so the delivered request would no longer match
// the credential claim (FCAF WS_RP_IA_MainInteraction__032 / __033).
func TestParseWorkflow_KeepsDateLikeBodyValuesVerbatim(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		want  string
	}{
		{"unquoted", "2012-03-04", "2012-03-04"},
		{"double quoted", `"2012-03-04"`, "2012-03-04"},
		{"single quoted", "'2012-03-04'", "2012-03-04"},
		{"unquoted expiry", "2031-01-01", "2031-01-01"},
		{"unquoted with time", "2031-01-01T00:00:00Z", "2031-01-01T00:00:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			yamlStr := "name: t\nsteps:\n- id: s\n  use: http-request\n  with:\n" +
				"    body:\n      claims:\n      - path:\n        - birthdate\n" +
				"        values:\n        - " + tc.value + "\n"
			wf, err := ParseWorkflow(yamlStr)
			require.NoError(t, err)

			body := wf.Steps[0].With.Payload["body"]
			// The http-request activity marshals the body map to JSON.
			encoded, err := json.Marshal(body)
			require.NoError(t, err)
			require.JSONEq(
				t,
				`{"claims":[{"path":["birthdate"],"values":["`+tc.want+`"]}]}`,
				string(encoded),
			)
		})
	}
}

// A date-like scalar referenced through a YAML alias must also keep its
// literal text, since aliases are resolved while decoding step inputs.
func TestParseWorkflow_KeepsAnchoredDateLikeBodyValuesVerbatim(t *testing.T) {
	yamlStr := "name: t\nanchor: &dob 2012-03-04\nsteps:\n- id: s\n  use: http-request\n" +
		"  with:\n    body:\n      values:\n      - *dob\n"
	wf, err := ParseWorkflow(yamlStr)
	require.NoError(t, err)

	encoded, err := json.Marshal(wf.Steps[0].With.Payload["body"])
	require.NoError(t, err)
	require.JSONEq(t, `{"values":["2012-03-04"]}`, string(encoded))
}

// Retagging date-like scalars in step inputs must stay inside the `with`
// subtree: an alias reaching such a scalar from a field outside `with` has to
// keep yaml.v3's own resolution (time.Time) there. Mutating a shared anchored
// node in place would silently change every other reference to it.
func TestParseWorkflow_TimestampRetagStaysInsideStepInputs(t *testing.T) {
	t.Run("anchor declared inside with", func(t *testing.T) {
		yamlStr := "name: t\nsteps:\n- id: s\n  use: http-request\n" +
			"  with:\n    body: &dob\n      date: 2012-03-04\n" +
			"  metadata:\n    shared: *dob\n"
		wf, err := ParseWorkflow(yamlStr)
		require.NoError(t, err)

		encoded, err := json.Marshal(wf.Steps[0].With.Payload["body"])
		require.NoError(t, err)
		require.JSONEq(t, `{"date":"2012-03-04"}`, string(encoded))

		shared, ok := wf.Steps[0].Metadata["shared"].(map[string]any)
		require.True(t, ok, "aliased metadata should decode as a map")
		require.IsType(t, time.Time{}, shared["date"])
	})

	t.Run("anchor declared outside with", func(t *testing.T) {
		yamlStr := "name: t\nconfig:\n  dob: &dob 2012-03-04\nsteps:\n- id: s\n" +
			"  use: http-request\n  with:\n    body:\n      values:\n      - *dob\n" +
			"  metadata:\n    shared: *dob\n"
		wf, err := ParseWorkflow(yamlStr)
		require.NoError(t, err)

		encoded, err := json.Marshal(wf.Steps[0].With.Payload["body"])
		require.NoError(t, err)
		require.JSONEq(t, `{"values":["2012-03-04"]}`, string(encoded))

		require.IsType(t, time.Time{}, wf.Config["dob"])
		require.IsType(t, time.Time{}, wf.Steps[0].Metadata["shared"])
	})
}
