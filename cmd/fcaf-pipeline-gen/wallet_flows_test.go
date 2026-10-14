// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWalletActionShareConsumption pins the generator's rule that a Share tap
// spends a credential instance, whatever outcome the action expects. The
// negative actions keep a conditional Share: a tolerant Wallet that shows the
// consent screen answers it, and the presentation it then records is what makes
// the source's negative assertion fail on evidence. Counting that Share as zero
// models the instance as unspent, so a single tolerant share would starve every
// later presentation of the same scenario, whose no-match or rejection evidence
// would then only prove an empty Wallet.
func TestWalletActionShareConsumption(t *testing.T) {
	actions, err := loadWalletActions(
		filepath.Join("..", "..", "config_templates", "fcaf", "imports"),
	)
	require.NoError(t, err)

	for _, testCase := range []struct {
		name   string
		action string
		want   int
	}{
		{
			name:   "conditional consent share of a refused request",
			action: walletActionNamespace + "fcaf-expect-request-rejected",
			want:   1,
		},
		{
			name:   "conditional consent share of a no-match request",
			action: walletActionNamespace + "fcaf-expect-no-matching-document",
			want:   1,
		},
		{
			name:   "consent share of an accepted request",
			action: walletActionNamespace + "fcaf-exercise-wallet-generic",
			want:   1,
		},
		{
			name:   "consent share of an accepted DC API request",
			action: walletActionNamespace + "fcaf-dc-api-present",
			want:   1,
		},
		{
			name:   "DC API refusal never shares",
			action: walletActionNamespace + "fcaf-dc-api-expect-rejected",
			want:   0,
		},
		{
			name:   "credential refusal never presents",
			action: walletActionNamespace + "fcaf-expect-credential-rejected",
			want:   0,
		},
		{
			name:   "issuance never presents",
			action: issuanceActionID,
			want:   0,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			consumed, err := credentialInstancesConsumed(
				map[string]any{
					"use":  mobileAutomationTask,
					"with": map[string]any{"action_id": testCase.action},
				},
				actions,
			)
			require.NoError(t, err)
			require.Equal(t, testCase.want, consumed)
		})
	}
}
