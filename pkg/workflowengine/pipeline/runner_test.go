// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateRunnerIDYAML(t *testing.T) {
	t.Run("no mobile-automation steps", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
steps:
  - id: step1
    use: rest
`
		require.NoError(t, ValidateRunnerIDYAML(yamlContent))
	})

	t.Run("global runner conflicts with step runner", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
runtime:
  global_runner_id: global-runner
steps:
  - id: step1
    use: mobile-automation
    with:
      runner_id: step-runner
`
		err := ValidateRunnerIDYAML(yamlContent)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "step1"`)
		require.Contains(t, err.Error(), "global_runner_id is set")
	})

	t.Run("missing step runner without global", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
steps:
  - id: step1
    use: mobile-automation
    with:
      runner_id: step-runner
  - id: step2
    use: mobile-automation
`
		err := ValidateRunnerIDYAML(yamlContent)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "step2"`)
		require.Contains(t, err.Error(), "missing runner_id")
	})

	t.Run("first conflict step is deterministic", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
runtime:
  global_runner_id: global-runner
steps:
  - id: stepA
    use: mobile-automation
    with:
      runner_id: step-runner-a
  - id: stepB
    use: mobile-automation
    with:
      runner_id: step-runner-b
`
		err := ValidateRunnerIDYAML(yamlContent)
		require.Error(t, err)
		require.Contains(t, err.Error(), `step "stepA"`)
	})
}
