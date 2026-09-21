// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPWalletNonceMismatchValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		expected   string
		wantStatus Status
	}{
		{
			name: "different Request Object nonce",
			value: map[string]any{
				"request_uri_payload": map[string]any{"wallet_nonce": "wallet-nonce"},
				"request_object":      compactTestJWT(t, map[string]any{"wallet_nonce": "different-nonce"}),
			},
			expected:   "different",
			wantStatus: StatusPass,
		},
		{
			name: "missing Request Object nonce",
			value: map[string]any{
				"request_uri_payload": map[string]any{"wallet_nonce": "wallet-nonce"},
				"request_object":      compactTestJWT(t, map[string]any{}),
			},
			expected:   "missing",
			wantStatus: StatusPass,
		},
		{
			name: "matching Request Object nonce",
			value: map[string]any{
				"request_uri_payload": map[string]any{"wallet_nonce": "wallet-nonce"},
				"request_object":      compactTestJWT(t, map[string]any{"wallet_nonce": "wallet-nonce"}),
			},
			expected:   "different",
			wantStatus: StatusFail,
		},
		{
			name: "present nonce when absent expected",
			value: map[string]any{
				"request_uri_payload": map[string]any{"wallet_nonce": "wallet-nonce"},
				"request_object":      compactTestJWT(t, map[string]any{"wallet_nonce": "different-nonce"}),
			},
			expected:   "missing",
			wantStatus: StatusFail,
		},
		{
			name: "missing POST nonce",
			value: map[string]any{
				"request_uri_payload": map[string]any{},
				"request_object":      compactTestJWT(t, map[string]any{}),
			},
			expected:   "missing",
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPWalletNonceMismatchValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: map[string]any{"expected": test.expected},
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
