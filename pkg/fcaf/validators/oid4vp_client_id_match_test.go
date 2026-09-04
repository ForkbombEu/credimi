// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPClientIDMatchValidator(t *testing.T) {
	const clientID = "x509_hash:matching-client"

	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "matches identical prefixed client IDs",
			value: map[string]any{
				"deeplink":       "haip-vp://?client_id=x509_hash%3Amatching-client&request_uri=https%3A%2F%2Fverifier.example%2Frequest",
				"request_object": compactTestJWT(t, map[string]any{"client_id": clientID}),
			},
			wantStatus: StatusPass,
		},
		{
			name: "rejects client IDs with different prefixes",
			value: map[string]any{
				"deeplink": "haip-vp://?client_id=x509_hash%3Amatching-client",
				"request_object": compactTestJWT(
					t,
					map[string]any{"client_id": "did:example:matching-client"},
				),
			},
			wantStatus: StatusFail,
		},
		{
			name: "rejects missing outer client ID",
			value: map[string]any{
				"deeplink":       "haip-vp://?request_uri=https%3A%2F%2Fverifier.example%2Frequest",
				"request_object": compactTestJWT(t, map[string]any{"client_id": clientID}),
			},
			wantStatus: StatusFail,
		},
		{
			name: "rejects missing request object client ID",
			value: map[string]any{
				"deeplink":       "haip-vp://?client_id=x509_hash%3Amatching-client",
				"request_object": compactTestJWT(t, map[string]any{}),
			},
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPClientIDMatchValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: test.value})

			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
