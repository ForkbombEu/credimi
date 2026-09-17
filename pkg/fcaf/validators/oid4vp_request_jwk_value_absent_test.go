// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPRequestJWKValueAbsentValidator(t *testing.T) {
	validator := OID4VPRequestJWKValueAbsentValidator{}
	withoutBareECDHES := testSignedRequest(t, map[string]any{"jwks": map[string]any{"keys": []any{
		map[string]any{"kty": "EC", "crv": "P-256", "use": "enc", "alg": "ECDH-ES+A256KW"},
	}}})
	withBareECDHES := testSignedRequest(t, map[string]any{"jwks": map[string]any{"keys": []any{
		map[string]any{"kty": "EC", "crv": "P-256", "use": "enc", "alg": "ECDH-ES+A256KW"},
		map[string]any{"kty": "EC", "crv": "P-256", "use": "sig", "alg": "ECDH-ES"},
	}}})
	withoutJWKS := testSignedRequest(t, map[string]any{})

	for _, tt := range []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{"non-bare ECDH-ES JWK", withoutBareECDHES, map[string]any{"field": "alg", "value": "ECDH-ES"}, StatusPass},
		{"bare ECDH-ES JWK anywhere in set", withBareECDHES, map[string]any{"field": "alg", "value": "ECDH-ES"}, StatusFail},
		{"JWKs omitted", withoutJWKS, map[string]any{"field": "alg", "value": "ECDH-ES"}, StatusPass},
		{"field missing", withoutBareECDHES, map[string]any{"value": "ECDH-ES"}, StatusError},
		{"value missing", withoutBareECDHES, map[string]any{"field": "alg"}, StatusError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: tt.value, Params: tt.params})
			require.Equal(t, tt.status, result.Status)
		})
	}
}
