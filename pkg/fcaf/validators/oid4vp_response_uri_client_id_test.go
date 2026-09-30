// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPResponseURIClientIDMismatchValidator(t *testing.T) {
	tests := []struct {
		name       string
		payload    map[string]any
		wantStatus Status
	}{
		{
			name: "response_uri host differs from the client identifier",
			payload: map[string]any{
				"client_id":    "x509_san_dns:verifier.example.org",
				"response_uri": "https://other.example.com/openid4vp/response",
			},
			wantStatus: StatusPass,
		},
		{
			name: "response_uri host matches the client identifier",
			payload: map[string]any{
				"client_id":    "x509_san_dns:verifier.example.org",
				"response_uri": "https://verifier.example.org/openid4vp/response",
			},
			wantStatus: StatusFail,
		},
		{
			name: "matching host with a port is still the same FQDN",
			payload: map[string]any{
				"client_id":    "x509_san_dns:verifier.example.org",
				"response_uri": "https://verifier.example.org:8443/openid4vp/response",
			},
			wantStatus: StatusFail,
		},
		{
			name: "request does not use the required client identifier prefix",
			payload: map[string]any{
				"client_id":    "x509_hash:VGVzdA",
				"response_uri": "https://other.example.com/openid4vp/response",
			},
			wantStatus: StatusFail,
		},
		{
			name:       "request carries no response_uri",
			payload:    map[string]any{"client_id": "x509_san_dns:verifier.example.org"},
			wantStatus: StatusFail,
		},
		{
			name: "response_uri is not an absolute URI",
			payload: map[string]any{
				"client_id":    "x509_san_dns:verifier.example.org",
				"response_uri": "/openid4vp/response",
			},
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPResponseURIClientIDMismatchValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  compactTestJWT(t, test.payload),
				Params: map[string]any{"prefix": "x509_san_dns"},
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
