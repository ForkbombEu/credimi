// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const signingWalletMetadata = `{"request_object_signing_alg_values_supported":["ES256"],` +
	`"vp_formats_supported":{"dc+sd-jwt":{}}}`

func TestOID4VPWalletMetadataValidator(t *testing.T) {
	tests := []struct {
		name     string
		metadata any
		params   map[string]any
		want     Status
	}{
		{
			name:     "posted signing algorithms are present",
			metadata: signingWalletMetadata,
			params:   map[string]any{"present": []any{"request_object_signing_alg_values_supported"}},
			want:     StatusPass,
		},
		{
			name:     "an object wallet_metadata is accepted like the form-encoded string",
			metadata: map[string]any{"vp_formats_supported": map[string]any{"mso_mdoc": map[string]any{}}},
			params:   map[string]any{"present": []any{"vp_formats_supported"}},
			want:     StatusPass,
		},
		{
			name:     "signing algorithms omitted for a prefix that precludes signing",
			metadata: `{"vp_formats_supported":{"dc+sd-jwt":{}}}`,
			params:   map[string]any{"absent": []any{"request_object_signing_alg_values_supported"}},
			want:     StatusPass,
		},
		{
			name:     "signing algorithms posted where they must be omitted",
			metadata: signingWalletMetadata,
			params:   map[string]any{"absent": []any{"request_object_signing_alg_values_supported"}},
			want:     StatusFail,
		},
		{
			name:     "an empty algorithm list does not count as present",
			metadata: `{"request_object_signing_alg_values_supported":[]}`,
			params:   map[string]any{"present": []any{"request_object_signing_alg_values_supported"}},
			want:     StatusFail,
		},
		{
			name:     "a null wallet_metadata cannot satisfy an absence check",
			metadata: "null",
			params:   map[string]any{"absent": []any{"request_object_signing_alg_values_supported"}},
			want:     StatusFail,
		},
		{
			name:     "a Wallet that posted no wallet_metadata fails",
			metadata: nil,
			params:   map[string]any{"absent": []any{"request_object_signing_alg_values_supported"}},
			want:     StatusFail,
		},
		{
			name:     "at least one expectation is required",
			metadata: signingWalletMetadata,
			params:   map[string]any{},
			want:     StatusError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := map[string]any{"wallet_nonce": "nonce-1"}
			if test.metadata != nil {
				payload["wallet_metadata"] = test.metadata
			}

			got := OID4VPWalletMetadataValidator{}.Validate(
				context.Background(),
				Input{Value: payload, Params: test.params},
			)

			require.Equal(t, test.want, got.Status, got.Message)
		})
	}
}

func TestOID4VPUnencryptedRequestObjectRejectedValidator(t *testing.T) {
	const encryptingMetadata = `{"jwks":{"keys":[{"kty":"EC","crv":"P-256","x":"x","y":"y"}]}}`
	tests := []struct {
		name          string
		metadata      any
		requestObject string
		response      map[string]any
		want          Status
	}{
		{
			name:          "a Wallet that publishes no jwks does not require encryption",
			metadata:      `{"authorization_encryption_alg_values_supported":["ECDH-ES"]}`,
			requestObject: "header.payload.signature",
			response:      map[string]any{"vp_token": map[string]any{"pid": []any{"token"}}},
			want:          StatusNotApplicable,
		},
		{
			name:          "an encryption-requiring Wallet rejects the unencrypted Request Object",
			metadata:      encryptingMetadata,
			requestObject: "header.payload.signature",
			response:      map[string]any{"error": "invalid_request"},
			want:          StatusPass,
		},
		{
			name:          "an encryption-requiring Wallet that presents fails",
			metadata:      encryptingMetadata,
			requestObject: "header.payload.signature",
			response:      map[string]any{"vp_token": map[string]any{"pid": []any{"token"}}},
			want:          StatusFail,
		},
		{
			name:          "an encrypted Request Object is not the source precondition",
			metadata:      encryptingMetadata,
			requestObject: "header.key.iv.ciphertext.tag",
			response:      map[string]any{"error": "invalid_request"},
			want:          StatusFail,
		},
		{
			name:          "no Request URI POST means there is no wallet_metadata to judge",
			metadata:      nil,
			requestObject: "header.payload.signature",
			response:      map[string]any{"error": "invalid_request"},
			want:          StatusFail,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := map[string]any{
				"raw":      map[string]any{"authorization_request_jwt": test.requestObject},
				"observed": map[string]any{"wallet_response": map[string]any{"value": test.response}},
			}
			if test.metadata != nil {
				observed, _ := session["observed"].(map[string]any)
				observed["request_uri_payload"] = map[string]any{
					"value": map[string]any{"wallet_metadata": test.metadata},
				}
			}

			got := OID4VPUnencryptedRequestObjectRejectedValidator{}.Validate(
				context.Background(),
				Input{Value: session},
			)

			require.Equal(t, test.want, got.Status, got.Message)
		})
	}
}
