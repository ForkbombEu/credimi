// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseEntityIDs(t *testing.T) {
	t.Run("empty yaml returns zero value", func(t *testing.T) {
		got, err := ParseEntityIDs("   ")
		require.NoError(t, err)
		require.Empty(t, got.Actions)
		require.Empty(t, got.Versions)
		require.Empty(t, got.Credentials)
		require.Empty(t, got.UseCases)
		require.Empty(t, got.ConformanceChecks)
		require.Empty(t, got.CustomChecks)
	})

	t.Run("invalid yaml returns error", func(t *testing.T) {
		_, err := ParseEntityIDs("[")
		require.Error(t, err)
	})

	t.Run("collects action_ids and version_ids from multiple steps", func(t *testing.T) {
    yamlStr := `
name: test
steps:
  - id: step-1
    use: mobile-automation
    with:
      action_id: /org/action-onboarding
      version_id: /org/wallet/v1-0-0
  - id: step-2
    use: mobile-automation-2
    with:
      action_id: /org/action-offer
      version_id: /org/wallet/v2-0-0
  - id: step-3
    use: mobile-automation-3
    with:
      action_id: /org/action-check
      version_id: /org/wallet/v3-0-0
`

    	got, err := ParseEntityIDs(yamlStr)
    	require.NoError(t, err)
    	require.Equal(t, []string{"org/action-check", "org/action-offer", "org/action-onboarding"}, got.Actions)
    	require.Equal(t, []string{"org/wallet/v1-0-0", "org/wallet/v2-0-0", "org/wallet/v3-0-0"}, got.Versions)
	})

	t.Run("collects credential_ids from multiple steps", func(t *testing.T) {
		yamlStr := `
name: test
steps:
  - id: step-1
    use: credential-offer
    with:
      credential_id: /issuer/credential-1
  - id: step-2
    use: credential-offer
    with:
      credential_id: /issuer/credential-2
`

		got, err := ParseEntityIDs(yamlStr)
		require.NoError(t, err)
		require.Equal(t, []string{"issuer/credential-1", "issuer/credential-2"}, got.Credentials)
	})

	t.Run("collects use_case_ids from multiple steps", func(t *testing.T) {
		yamlStr := `
name: test
steps:
  - id: step-1
    use: use-check-1
    with:
      use_case_id: /uc/presentation
  - id: step-2
    use: use-check-2
    with:
      use_case_id: /uc/issuance
`

		got, err := ParseEntityIDs(yamlStr)
		require.NoError(t, err)
		require.Equal(t, []string{"uc/issuance", "uc/presentation"}, got.UseCases)
	})

	t.Run("collects check_ids and distinguishes conformance vs custom", func(t *testing.T) {
		yamlStr := `
name: test
steps:
  - id: step-1
    use: conformance-check
    with:
      check_id: /conformance/check-1
  - id: step-2
    use: custom-check
    with:
      check_id: /custom/check-2
  - id: step-3
    use: conformance-check
    with:
      check_id: /conformance/check-3
`

		got, err := ParseEntityIDs(yamlStr)
		require.NoError(t, err)
		require.Equal(t, []string{"conformance/check-1", "conformance/check-3"}, got.ConformanceChecks)
		require.Equal(t, []string{"custom/check-2"}, got.CustomChecks)
	})

	t.Run("collects all entity types together", func(t *testing.T) {
		yamlStr := `
name: test
steps:
  - id: step-1
    use: mobile-automation
    with:
      action_id: /org/action-1
      version_id: /org/version-1
  - id: step-2
    use: credential-offer
    with:
      credential_id: /issuer/credential-1
  - id: step-3
    use: conformance-check
    with:
      check_id: /conformance/check-1
  - id: step-4
    use: custom-check
    with:
      check_id: /custom/check-1
`

		got, err := ParseEntityIDs(yamlStr)
		require.NoError(t, err)
		require.Equal(t, []string{"org/action-1"}, got.Actions)
		require.Equal(t, []string{"org/version-1"}, got.Versions)
		require.Equal(t, []string{"issuer/credential-1"}, got.Credentials)
		require.Equal(t, []string{"conformance/check-1"}, got.ConformanceChecks)
		require.Equal(t, []string{"custom/check-1"}, got.CustomChecks)
	})

	t.Run("handles steps with on_error and on_success branches", func(t *testing.T) {
		yamlStr := `
name: test
steps:
  - id: step-1
    use: mobile-automation
    with:
      action_id: /org/action-main
    on_error:
      - id: err-step
        use: conformance-check
        with:
          check_id: /conformance/error-check
    on_success:
      - id: success-step
        use: custom-check
        with:
          check_id: /custom/success-check
`

		got, err := ParseEntityIDs(yamlStr)
		require.NoError(t, err)
		require.Equal(t, []string{"org/action-main"}, got.Actions)
		require.Equal(t, []string{"conformance/error-check"}, got.ConformanceChecks)
		require.Equal(t, []string{"custom/success-check"}, got.CustomChecks)
	})

	t.Run("deduplicates duplicate IDs", func(t *testing.T) {
		yamlStr := `
name: test
steps:
  - id: step-1
    use: mobile-automation
    with:
      action_id: /org/action-duplicate
  - id: step-2
    use: mobile-automation
    with:
      action_id: /org/action-duplicate
  - id: step-3
    use: conformance-check
    with:
      check_id: /conformance/check-duplicate
  - id: step-4
    use: conformance-check
    with:
      check_id: /conformance/check-duplicate
`

		got, err := ParseEntityIDs(yamlStr)
		require.NoError(t, err)
		require.Len(t, got.Actions, 1)
		require.Equal(t, []string{"org/action-duplicate"}, got.Actions)
		require.Len(t, got.ConformanceChecks, 1)
		require.Equal(t, []string{"conformance/check-duplicate"}, got.ConformanceChecks)
	})

	t.Run("ignores empty strings", func(t *testing.T) {
		yamlStr := `
name: test
steps:
  - id: step-1
    use: mobile-automation
    with:
      action_id: ""
      version_id: ""
  - id: step-2
    use: conformance-check
    with:
      check_id: ""
`

		got, err := ParseEntityIDs(yamlStr)
		require.NoError(t, err)
		require.Empty(t, got.Actions)
		require.Empty(t, got.Versions)
		require.Empty(t, got.ConformanceChecks)
	})

	t.Run("handles missing payload gracefully", func(t *testing.T) {
		yamlStr := `
name: test
steps:
  - id: step-1
    use: mobile-automation
    with:
      action_id: /org/action-1
  - id: step-2
    use: echo
  - id: step-3
    use: conformance-check
`

		got, err := ParseEntityIDs(yamlStr)
		require.NoError(t, err)
		require.Equal(t, []string{"org/action-1"}, got.Actions)
	})
}
