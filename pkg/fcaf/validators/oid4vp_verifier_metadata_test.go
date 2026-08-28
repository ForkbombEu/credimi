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

func TestOID4VPVerifierMetadataExclusiveValidator(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		status  Status
	}{
		{
			name: "metadata appears only inside client_metadata",
			payload: map[string]any{
				"response_type": "vp_token",
				"client_metadata": map[string]any{
					"vp_formats_supported":     map[string]any{"mso_mdoc": map[string]any{}},
					"response_types_supported": []any{"vp_token"},
				},
			},
			status: StatusPass,
		},
		{
			name:    "client_metadata is missing",
			payload: map[string]any{"response_type": "vp_token"},
			status:  StatusFail,
		},
		{
			name:    "client_metadata is not an object",
			payload: map[string]any{"client_metadata": "not-an-object"},
			status:  StatusFail,
		},
		{
			name:    "client_metadata is empty",
			payload: map[string]any{"client_metadata": map[string]any{}},
			status:  StatusFail,
		},
		{
			name: "vp_formats_supported is missing",
			payload: map[string]any{
				"client_metadata": map[string]any{"response_types_supported": []any{"vp_token"}},
			},
			status: StatusFail,
		},
		{
			name: "vp_formats_supported is empty",
			payload: map[string]any{
				"client_metadata": map[string]any{"vp_formats_supported": map[string]any{}},
			},
			status: StatusFail,
		},
		{
			name: "metadata is duplicated outside client_metadata",
			payload: map[string]any{
				"vp_formats_supported": map[string]any{"dc+sd-jwt": map[string]any{}},
				"client_metadata": map[string]any{
					"vp_formats_supported": map[string]any{"mso_mdoc": map[string]any{}},
				},
			},
			status: StatusFail,
		},
	}

	validator := OID4VPVerifierMetadataExclusiveValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(
				context.Background(),
				Input{Value: compactTestJWT(t, test.payload)},
			)
			require.Equal(t, test.status, result.Status, result.Message)
		})
	}
}

func compactTestJWT(t *testing.T, payload map[string]any) string {
	t.Helper()
	headerJSON, err := json.Marshal(map[string]any{"alg": "ES256"})
	require.NoError(t, err)
	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(payloadJSON) + ".signature"
}
