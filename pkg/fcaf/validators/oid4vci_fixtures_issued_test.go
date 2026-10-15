// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VCIFixturesIssuedValidator(t *testing.T) {
	session := func(fixture, status string) map[string]any {
		return map[string]any{"fixture_id": fixture, "status": status}
	}
	fixtures := []any{"pid_default", "pid_person_b"}

	tests := []struct {
		name       string
		value      any
		params     map[string]any
		wantStatus Status
	}{
		{
			name: "every fixture issued",
			value: []any{
				session("pid_default", "credential_issued"),
				session("pid_person_b", "credential_issued"),
			},
			params:     map[string]any{"fixtures": fixtures},
			wantStatus: StatusPass,
		},
		{
			name: "one fixture offered but never fetched",
			value: []any{
				session("pid_default", "credential_issued"),
				session("pid_person_b", "created"),
			},
			params:     map[string]any{"fixtures": fixtures},
			wantStatus: StatusFail,
		},
		{
			name:       "one fixture never offered",
			value:      []any{session("pid_default", "credential_issued")},
			params:     map[string]any{"fixtures": fixtures},
			wantStatus: StatusFail,
		},
		{
			name: "the same fixture issued twice is still one fixture",
			value: []any{
				session("pid_default", "credential_issued"),
				session("pid_default", "credential_issued"),
			},
			params:     map[string]any{"fixtures": fixtures},
			wantStatus: StatusFail,
		},
		{
			name:       "a single issuer session record",
			value:      session("pid_default", "credential_issued"),
			params:     map[string]any{"fixtures": []any{"pid_default"}},
			wantStatus: StatusPass,
		},
		{
			name:       "an issuer session that is not an object",
			value:      []any{"credential_issued"},
			params:     map[string]any{"fixtures": []any{"pid_default"}},
			wantStatus: StatusFail,
		},
		{
			name:       "no fixtures requested",
			value:      []any{session("pid_default", "credential_issued")},
			params:     map[string]any{},
			wantStatus: StatusError,
		},
	}

	validator := OID4VCIFixturesIssuedValidator{}
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
