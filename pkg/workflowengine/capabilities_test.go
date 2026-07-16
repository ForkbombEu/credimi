// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithCredimiCapabilities(t *testing.T) {
	originalMemo := map[string]any{"test": "check"}
	original := WorkflowInput{Config: map[string]any{"memo": originalMemo}}

	updated := WithCredimiCapabilities(original, CredimiCapabilities{Logs: true})

	require.NotContains(t, originalMemo, CredimiCapabilitiesMemoKey)
	memo := updated.Config["memo"].(map[string]any)
	require.Equal(t, "check", memo["test"])
	require.Equal(t, CredimiCapabilities{Logs: true}, memo[CredimiCapabilitiesMemoKey])
}

func TestWithCredimiCapabilitiesInitializesConfigAndMemo(t *testing.T) {
	updated := WithCredimiCapabilities(WorkflowInput{}, CredimiCapabilities{})

	memo := updated.Config["memo"].(map[string]any)
	require.Equal(t, CredimiCapabilities{}, memo[CredimiCapabilitiesMemoKey])
}
