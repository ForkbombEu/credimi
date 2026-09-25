// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOID4VPNoPresentationValidator pins the normative outcome FCAF negative
// tests require: a wallet handed a malformed Authorization Request must reject
// it, so any Authorization Response it submits fails the test regardless of
// whether the verifier could validate that response.
func TestOID4VPNoPresentationValidator(t *testing.T) {
	cases := []struct {
		name    string
		session any
		want    Status
	}{
		{
			name:    "request refused before consent",
			session: map[string]any{"status": "request_retrieved", "checks": map[string]any{"presentation_valid": nil}},
			want:    StatusPass,
		},
		{
			name:    "deeplink never opened",
			session: map[string]any{"status": "created"},
			want:    StatusPass,
		},
		{
			name: "wallet shared a valid presentation",
			session: map[string]any{
				"status":                "presentation_validated",
				"decoded_presentations": map[string]any{"pid": []any{"presentation"}},
				"checks":                map[string]any{"presentation_valid": true},
			},
			want: StatusFail,
		},
		{
			name: "wallet answered with an invalid presentation",
			session: map[string]any{
				"status": "presentation_invalid",
				"checks": map[string]any{"presentation_valid": false},
			},
			want: StatusFail,
		},
		{
			name:    "presentation decoded without a terminal status",
			session: map[string]any{"status": "request_retrieved", "decoded_presentations": map[string]any{}},
			want:    StatusFail,
		},
		{
			name:    "session evidence is missing",
			session: nil,
			want:    StatusFail,
		},
		{
			name:    "session evidence carries no status",
			session: map[string]any{"checks": map[string]any{}},
			want:    StatusFail,
		},
	}

	validator := OID4VPNoPresentationValidator{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: tc.session})
			require.Equal(t, tc.want, result.Status, result.Message)
		})
	}
}
