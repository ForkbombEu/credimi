// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPRequestEncryptionJWKValidator(t *testing.T) {
	validator := OID4VPRequestEncryptionJWKValidator{}
	withAlg := testSignedRequest(t, map[string]any{"jwks": map[string]any{"keys": []any{
		map[string]any{"kty": "EC", "crv": "P-256", "use": "enc", "alg": "ECDH-ES"},
	}}})
	withoutAlg := testSignedRequest(t, map[string]any{"jwks": map[string]any{"keys": []any{
		map[string]any{"kty": "EC", "crv": "P-256", "use": "enc"},
	}}})
	mismatchedAlg := testSignedRequest(t, map[string]any{"jwks": map[string]any{"keys": []any{
		map[string]any{"kty": "EC", "crv": "P-256", "use": "enc", "alg": "ECDH-ES+A256KW"},
	}}})
	signingKeyFirst := testSignedRequest(t, map[string]any{"jwks": map[string]any{"keys": []any{
		map[string]any{"kty": "EC", "crv": "P-256", "use": "sig", "alg": "ES256"},
		map[string]any{"kty": "EC", "crv": "P-256", "use": "enc"},
	}}})
	noKeys := testSignedRequest(t, map[string]any{"jwks": map[string]any{"keys": []any{}}})
	noJWKS := testSignedRequest(t, map[string]any{})

	present := true
	absent := false
	for _, tt := range []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{
			"alg absent as required",
			withoutAlg,
			map[string]any{"field": "alg", "present": absent},
			StatusPass,
		},
		{
			"alg present when absence required",
			withAlg,
			map[string]any{"field": "alg", "present": absent},
			StatusFail,
		},
		{
			"mismatched alg value",
			mismatchedAlg,
			map[string]any{"field": "alg", "value": "ECDH-ES+A256KW"},
			StatusPass,
		},
		{
			"alg value differs",
			withAlg,
			map[string]any{"field": "alg", "value": "ECDH-ES+A256KW"},
			StatusFail,
		},
		{
			"encryption key selected over signing key",
			signingKeyFirst,
			map[string]any{"field": "alg", "present": absent},
			StatusPass,
		},
		{
			"required field missing",
			withoutAlg,
			map[string]any{"field": "alg", "value": "ECDH-ES"},
			StatusFail,
		},
		{
			"presence and value both checked",
			withAlg,
			map[string]any{"field": "alg", "present": present, "value": "ECDH-ES"},
			StatusPass,
		},
		{"no keys published", noKeys, map[string]any{"field": "alg", "present": absent}, StatusFail},
		{"no jwks published", noJWKS, map[string]any{"field": "alg", "present": absent}, StatusFail},
		{
			"request object missing",
			nil,
			map[string]any{"field": "alg", "present": absent},
			StatusFail,
		},
		{"field param missing", withAlg, map[string]any{"present": absent}, StatusError},
		{"no check requested", withAlg, map[string]any{"field": "alg"}, StatusError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(
				t,
				tt.status,
				validator.Validate(
					context.Background(),
					Input{Value: tt.value, Params: tt.params},
				).Status,
			)
		})
	}
}
