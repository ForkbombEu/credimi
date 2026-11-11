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
		{
			name:   "injected issuance never presents",
			action: injectedIssuanceActionID,
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

// TestMultipleTruePresentationSpendsEveryHeldInstance pins that a Share of a
// multiple: true query spends every instance of its format: the wallet returns
// all of them, so a later presentation in the same scenario holds none unless
// one is issued for it. Counting such a Share as one instance once let an
// inventory query empty the wallet before the query it was meant to prepare.
func TestMultipleTruePresentationSpendsEveryHeldInstance(t *testing.T) {
	session := func(multiple bool) map[string]any {
		credential := map[string]any{"id": "pid", "format": "dc+sd-jwt"}
		if multiple {
			credential["multiple"] = true
		}
		return map[string]any{"with": map[string]any{"body": map[string]any{
			"presentation_request": map[string]any{
				"dcql_query": map[string]any{"credentials": []any{credential}},
			},
		}}}
	}
	present := map[string]any{"id": "present", "use": mobileAutomationTask}

	for _, testCase := range []struct {
		name          string
		multiple      bool
		wantRemaining int
	}{
		{name: "single presentation spends one of three", multiple: false, wantRemaining: 2},
		{name: "multiple: true spends all three", multiple: true, wantRemaining: 0},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			available := map[credentialFormat]int{formatSDJWT: 3}
			sessions, issuances, err := pidIssuancesBefore(
				present,
				session(testCase.multiple),
				1,
				available,
				nil,
			)
			require.NoError(t, err)
			require.Empty(t, sessions)
			require.Empty(t, issuances)
			require.Equal(t, testCase.wantRemaining, available[formatSDJWT])

			_, next, err := pidIssuancesBefore(present, session(false), 1, available, nil)
			require.NoError(t, err)
			if testCase.wantRemaining == 0 {
				require.NotEmpty(t, next, "a later Share must get a freshly issued PID")
			} else {
				require.Empty(t, next)
			}
		})
	}
}

// TestPresentationsSpendTheirOwnCredentialType pins that a Share of a
// non-PID SD-JWT VC the scenario issued spends that type, never a PID, and
// that running out of it stops generation instead of injecting a PID no query
// for that type can match. A type the scenario never issued is a near-miss
// request, answered by the PID the wallet holds.
func TestPresentationsSpendTheirOwnCredentialType(t *testing.T) {
	const degree = "urn:credimi:degree:1"
	issueDegree := map[string]any{
		"id":   "obtain-degree",
		"use":  mobileAutomationTask,
		"with": map[string]any{"action_id": issuanceActionID},
	}
	degreeOffer := map[string]any{"with": map[string]any{"body": map[string]any{
		"credential_configuration_id": degree + ".sd-jwt.key-attestation-required",
	}}}
	request := func(vct string) map[string]any {
		return map[string]any{"with": map[string]any{"body": map[string]any{
			"presentation_request": map[string]any{"dcql_query": map[string]any{
				"credentials": []any{map[string]any{
					"id":     "credential",
					"format": "dc+sd-jwt",
					"meta":   map[string]any{"vct_values": []any{vct}},
				}},
			}},
		}}}
	}
	present := map[string]any{"id": "present", "use": mobileAutomationTask}

	available := map[credentialFormat]int{}
	_, _, err := pidIssuancesBefore(issueDegree, degreeOffer, 0, available, nil)
	require.NoError(t, err)

	sessions, _, err := pidIssuancesBefore(present, request(degree), 1, available, nil)
	require.NoError(t, err)
	require.Empty(t, sessions, "the issued degree answers the first degree presentation")

	_, _, err = pidIssuancesBefore(present, request(degree), 1, available, nil)
	require.ErrorContains(t, err, degree, "a spent degree must not be replaced by a PID")

	sessions, _, err = pidIssuancesBefore(
		present,
		request("urn:credimi:unissued:1"),
		1,
		available,
		nil,
	)
	require.NoError(t, err)
	require.Len(t, sessions, 1, "a type the scenario never issued falls back to a PID")
}
