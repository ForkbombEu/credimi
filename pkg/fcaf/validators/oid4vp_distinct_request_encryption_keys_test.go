// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPDistinctRequestEncryptionKeysValidator(t *testing.T) {
	validator := OID4VPDistinctRequestEncryptionKeysValidator{}
	request := func(t *testing.T, x string, kid string) string {
		t.Helper()
		return testSignedRequest(t, map[string]any{"jwks": map[string]any{"keys": []any{
			map[string]any{"kty": "EC", "crv": "P-256", "x": x, "y": "y-value", "use": "enc", "kid": kid},
		}}})
	}
	first := request(t, "key-one", "kid-one")
	second := request(t, "key-two", "kid-two")
	relabelled := request(t, "key-one", "kid-two")

	for _, tt := range []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{"distinct keys", []any{first, second}, nil, StatusPass},
		{"reused key", []any{first, first}, nil, StatusFail},
		{"reused key behind a new kid", []any{first, relabelled}, nil, StatusFail},
		{"single request", []any{first}, nil, StatusFail},
		{
			"three distinct keys required",
			[]any{first, second},
			map[string]any{"minimum_keys": 3},
			StatusFail,
		},
		{"missing request object", []any{first, ""}, nil, StatusFail},
		{"not an array", first, nil, StatusFail},
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
