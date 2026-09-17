// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJOSEJWEProtectedHeaderValidator(t *testing.T) {
	validator := JOSEJWEProtectedHeaderValidator{}
	compactJWE := testCompactJWE(t, map[string]any{
		"alg": "ECDH-ES",
		"enc": "A128CBC-HS256",
		"epk": map[string]any{"crv": "P-256"},
	})

	for _, tt := range []struct {
		name   string
		params map[string]any
		status Status
	}{
		{name: "matching top-level header", params: map[string]any{"field": "alg", "value": "ECDH-ES"}, status: StatusPass},
		{name: "matching nested header", params: map[string]any{"field": "epk.crv", "value": "P-256"}, status: StatusPass},
		{name: "mismatching header", params: map[string]any{"field": "enc", "value": "A256GCM"}, status: StatusFail},
		{name: "missing header", params: map[string]any{"field": "kid", "present": false}, status: StatusPass},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(
				context.Background(),
				Input{Value: compactJWE, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status)
		})
	}
}

func testCompactJWE(t *testing.T, header map[string]any) string {
	t.Helper()
	encoded, err := json.Marshal(header)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(encoded) + ".encrypted.iv.ciphertext.tag"
}
