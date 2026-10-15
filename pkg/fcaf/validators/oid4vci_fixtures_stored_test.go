// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VCIFixturesStoredValidator(t *testing.T) {
	const added = "https://credimi.test/api/files/r/x/scenario_obtain_pid_credential_added_abcdefghij.png"
	const other = "https://credimi.test/api/files/r/x/scenario_obtain_pid_offer_screen_abcdefghij.png"
	entry := func(fixture, status string, screenshots ...any) map[string]any {
		return map[string]any{
			"issuer_session":     map[string]any{"fixture_id": fixture, "status": status},
			"wallet_screenshots": screenshots,
		}
	}
	params := map[string]any{
		"fixtures":          []any{"pid_default", "pid_person_b"},
		"stored_screenshot": "credential-added",
	}

	tests := []struct {
		name       string
		value      any
		params     map[string]any
		wantStatus Status
	}{
		{
			name: "every fixture fetched and stored",
			value: []any{
				entry("pid_default", "credential_issued", added),
				entry("pid_person_b", "credential_issued", added),
			},
			params:     params,
			wantStatus: StatusPass,
		},
		{
			name: "a fixture fetched but not confirmed as stored by the Wallet",
			value: []any{
				entry("pid_default", "credential_issued", added),
				entry("pid_person_b", "credential_issued", other),
			},
			params:     params,
			wantStatus: StatusFail,
		},
		{
			name: "a failed issuance step stores no screenshot",
			value: []any{
				entry("pid_default", "credential_issued", added),
				entry("pid_person_b", "credential_issued"),
			},
			params:     params,
			wantStatus: StatusFail,
		},
		{
			name: "a fixture offered but never fetched",
			value: []any{
				entry("pid_default", "credential_issued", added),
				entry("pid_person_b", "created", added),
			},
			params:     params,
			wantStatus: StatusFail,
		},
		{
			name:       "a fixture never offered",
			value:      []any{entry("pid_default", "credential_issued", added)},
			params:     params,
			wantStatus: StatusFail,
		},
		{
			name: "a later successful issuance of the same fixture counts",
			value: []any{
				entry("pid_default", "credential_issued", added),
				entry("pid_person_b", "created"),
				entry("pid_person_b", "credential_issued", added),
			},
			params:     params,
			wantStatus: StatusPass,
		},
		{
			name:       "an entry without an issuer session",
			value:      []any{map[string]any{"wallet_screenshots": []any{added}}},
			params:     params,
			wantStatus: StatusFail,
		},
		{
			name:       "no stored screenshot name requested",
			value:      []any{entry("pid_default", "credential_issued", added)},
			params:     map[string]any{"fixtures": []any{"pid_default"}},
			wantStatus: StatusError,
		},
		{
			name:       "no fixtures requested",
			value:      []any{entry("pid_default", "credential_issued", added)},
			params:     map[string]any{"stored_screenshot": "credential-added"},
			wantStatus: StatusError,
		},
	}

	validator := OID4VCIFixturesStoredValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: test.params,
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
