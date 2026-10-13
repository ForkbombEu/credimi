// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPRequestAudienceValidator(t *testing.T) {
	const walletIssuer = "https://wallet.example"
	session := func(aud any, walletMetadata any, response map[string]any) map[string]any {
		value := map[string]any{
			"raw": map[string]any{
				"authorization_request_jwt":       compactTestJWT(t, map[string]any{"aud": aud}),
				"presentation_response_decrypted": response,
			},
		}
		if walletMetadata != nil {
			value["observed"] = map[string]any{"request_uri_payload": map[string]any{
				"value": map[string]any{"wallet_metadata": walletMetadata},
			}}
		}
		return value
	}
	presented := map[string]any{"vp_token": map[string]any{"pid": []any{"token"}}}
	rejected := map[string]any{"error": "invalid_request"}
	dynamicMetadata := `{"issuer":"` + walletIssuer + `"}`

	tests := []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{
			name:    "static discovery presents for self-issued aud",
			value:   session(staticDiscoveryAudience, `{"vp_formats_supported":{}}`, presented),
			params:  map[string]any{"discovery": "static", "outcome": "accepted"},
			status:  StatusPass,
			message: "under static discovery",
		},
		{
			name:   "static discovery without a POST retrieval",
			value:  session([]any{staticDiscoveryAudience}, nil, presented),
			params: map[string]any{"discovery": "static", "outcome": "accepted"},
			status: StatusPass,
		},
		{
			name:    "self-issued aud without presentation",
			value:   session(staticDiscoveryAudience, nil, rejected),
			params:  map[string]any{"discovery": "static", "outcome": "accepted"},
			status:  StatusFail,
			message: "returned no vp_token",
		},
		{
			name:  "static discovery rejects another aud with invalid_request",
			value: session("https://verifier.example", nil, rejected),
			params: map[string]any{
				"discovery": "static", "outcome": "rejected", "code": "invalid_request",
			},
			status: StatusPass,
		},
		{
			name:    "static discovery presents for another aud",
			value:   session("https://verifier.example", nil, presented),
			params:  map[string]any{"discovery": "static", "outcome": "rejected"},
			status:  StatusFail,
			message: "returned vp_token",
		},
		{
			name:    "rejection case served the expected aud",
			value:   session(staticDiscoveryAudience, nil, rejected),
			params:  map[string]any{"discovery": "static", "outcome": "rejected"},
			status:  StatusInconclusive,
			message: "does not set up the rejected case",
		},
		{
			name:    "dynamic test does not apply to a static-discovery Wallet",
			value:   session("https://verifier.example", `{"vp_formats_supported":{}}`, rejected),
			params:  map[string]any{"discovery": "dynamic", "outcome": "rejected"},
			status:  StatusNotApplicable,
			message: "Wallet uses static discovery",
		},
		{
			name: "dynamic discovery answers an aud other than issuer with any error",
			value: session(
				staticDiscoveryAudience,
				dynamicMetadata,
				map[string]any{"error": "access_denied"},
			),
			params:  map[string]any{"discovery": "dynamic", "outcome": "rejected"},
			status:  StatusPass,
			message: `error "access_denied"`,
		},
		{
			name:    "dynamic discovery rejection without an error",
			value:   session(staticDiscoveryAudience, dynamicMetadata, map[string]any{}),
			params:  map[string]any{"discovery": "dynamic", "outcome": "rejected"},
			status:  StatusFail,
			message: "returned no error",
		},
		{
			name:    "served Request Object is missing",
			value:   map[string]any{"raw": map[string]any{}},
			params:  map[string]any{"discovery": "static", "outcome": "accepted"},
			status:  StatusFail,
			message: "served Request Object",
		},
		{
			name:   "unknown discovery mode",
			value:  session(staticDiscoveryAudience, nil, presented),
			params: map[string]any{"discovery": "federation", "outcome": "accepted"},
			status: StatusError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := OID4VPRequestAudienceValidator{}.Validate(
				context.Background(),
				Input{Value: test.value, Params: test.params},
			)
			require.Equal(t, test.status, result.Status, result.Message)
			require.Contains(t, result.Message, test.message)
		})
	}
}
