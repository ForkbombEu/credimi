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
			name: "request refused before consent",
			session: map[string]any{
				"status": "request_retrieved",
				"checks": map[string]any{"presentation_valid": nil},
			},
			want: StatusPass,
		},
		{
			// A bare status map is not a Capture session record, so the guard
			// cannot see engagement and the validator's own verdict stands.
			name:    "evidence without a session record",
			session: map[string]any{"status": "created"},
			want:    StatusPass,
		},
		{
			name: "wallet never retrieved the request",
			session: neverRetrievedSession(
				map[string]any{"checks": map[string]any{"presentation_valid": nil}},
			),
			want: StatusBlocked,
		},
		{
			name: "wallet retrieved the request and answered nothing",
			session: retrievedSessionWithoutResponse(map[string]any{
				"checks": map[string]any{"presentation_valid": nil},
			}),
			want: StatusPass,
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
			name: "wallet answered with an error response",
			session: map[string]any{
				"status": "presentation_invalid",
				"checks": map[string]any{"presentation_valid": false},
				"raw": map[string]any{
					"presentation_response_decrypted": map[string]any{"error": "invalid_request"},
					"presentation_response_verifier_http": map[string]any{
						"body": `{"error":"invalid_presentation"}`,
					},
				},
			},
			want: StatusPass,
		},
		{
			name: "wallet sent a vp_token alongside an error",
			session: map[string]any{
				"status": "presentation_invalid",
				"checks": map[string]any{"presentation_valid": false},
				"raw": map[string]any{"presentation_response_decrypted": map[string]any{
					"error":    "invalid_request",
					"vp_token": map[string]any{"pid": []any{"presentation"}},
				}},
			},
			want: StatusFail,
		},
		{
			name: "only the verifier's own reply carries an error",
			session: map[string]any{
				"status": "presentation_invalid",
				"checks": map[string]any{"presentation_valid": false},
				"raw": map[string]any{"presentation_response_verifier_http": map[string]any{
					"body": `{"error":"invalid_presentation"}`,
				}},
			},
			want: StatusFail,
		},
		{
			name: "presentation decoded without a terminal status",
			session: map[string]any{
				"status":                "request_retrieved",
				"decoded_presentations": map[string]any{},
			},
			want: StatusFail,
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
