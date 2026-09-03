// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPWalletNonceMatchValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "matches request URI POST and Request Object nonces",
			value: map[string]any{
				"request_uri_payload": map[string]any{"wallet_nonce": "wallet-nonce-045"},
				"request_object": compactTestJWT(
					t,
					map[string]any{"wallet_nonce": "wallet-nonce-045"},
				),
			},
			wantStatus: StatusPass,
		},
		{
			name: "rejects mismatched nonces",
			value: map[string]any{
				"request_uri_payload": map[string]any{"wallet_nonce": "wallet-nonce-045"},
				"request_object": compactTestJWT(
					t,
					map[string]any{"wallet_nonce": "different-nonce"},
				),
			},
			wantStatus: StatusFail,
		},
		{
			name: "rejects missing POST nonce",
			value: map[string]any{
				"request_uri_payload": map[string]any{},
				"request_object": compactTestJWT(
					t,
					map[string]any{"wallet_nonce": "wallet-nonce-045"},
				),
			},
			wantStatus: StatusFail,
		},
		{
			name: "rejects missing Request Object nonce",
			value: map[string]any{
				"request_uri_payload": map[string]any{"wallet_nonce": "wallet-nonce-045"},
				"request_object":      compactTestJWT(t, map[string]any{}),
			},
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPWalletNonceMatchValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: test.value})

			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
