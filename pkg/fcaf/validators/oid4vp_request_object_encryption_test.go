// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPEncryptedRequestObjectProcessedValidator(t *testing.T) {
	const advertisingMetadata = `{"jwks":{"keys":[{"kty":"EC","crv":"P-256",` +
		`"x":"x","y":"y","kid":"probe-wallet-enc"}]},` +
		`"authorization_encryption_alg_values_supported":["ECDH-ES"],` +
		`"authorization_encryption_enc_values_supported":["A256GCM"]}`

	encryptedDetail := func(alg, enc, algSource, encSource string) map[string]any {
		return map[string]any{
			"alg":        alg,
			"enc":        enc,
			"kid":        "probe-wallet-enc",
			"alg_source": algSource,
			"enc_source": encSource,
		}
	}
	encryptedEvent := func(detail map[string]any) map[string]any {
		return map[string]any{
			"at":     "2026-10-22T10:00:00Z",
			"type":   "vp_request_object_encrypted",
			"detail": detail,
		}
	}
	presentation := map[string]any{"vp_token": map[string]any{"pid": []any{"token"}}}

	tests := []struct {
		name        string
		input       any
		metadata    any
		post        bool
		events      []any
		jwe         string
		response    map[string]any
		want        Status
		wantMessage []string
	}{
		{
			name:     "non-object input is not a captured session",
			input:    "captured",
			want:     StatusFail,
			post:     true,
			metadata: advertisingMetadata,
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			jwe:      "a.b.c.d.e",
			response: presentation,
		},
		{
			name:     "a Wallet that never POSTed to the Request URI does not require encryption",
			metadata: advertisingMetadata,
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			jwe:      "a.b.c.d.e",
			response: presentation,
			want:     StatusNotApplicable,
		},
		{
			name:     "wallet_metadata without jwks does not require encryption",
			post:     true,
			metadata: `{"authorization_encryption_alg_values_supported":["ECDH-ES"]}`,
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			jwe:      "a.b.c.d.e",
			response: presentation,
			want:     StatusNotApplicable,
		},
		{
			name:     "an object wallet_metadata without jwks does not require encryption",
			post:     true,
			metadata: map[string]any{"vp_formats_supported": map[string]any{}},
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			jwe:      "a.b.c.d.e",
			response: presentation,
			want:     StatusNotApplicable,
		},
		{
			name: "an empty jwks key set does not require encryption",
			post: true,
			metadata: map[string]any{
				"jwks": map[string]any{"keys": []any{}},
				"authorization_encryption_alg_values_supported": []any{"ECDH-ES"},
				"authorization_encryption_enc_values_supported": []any{"A256GCM"},
			},
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			jwe:      "a.b.c.d.e",
			response: presentation,
			want:     StatusNotApplicable,
		},
		{
			name:     "malformed wallet_metadata is not an object",
			post:     true,
			metadata: `{"jwks":`,
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			jwe:         "a.b.c.d.e",
			response:    presentation,
			want:        StatusFail,
			wantMessage: []string{"wallet_metadata"},
		},
		{
			name: "an encryption key without authorization_encryption algorithms fails",
			post: true,
			metadata: map[string]any{
				"jwks": map[string]any{"keys": []any{map[string]any{"kty": "EC"}}},
				"authorization_encryption_enc_values_supported": []any{"A256GCM"},
			},
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			jwe:         "a.b.c.d.e",
			response:    presentation,
			want:        StatusFail,
			wantMessage: []string{"authorization_encryption_alg_values_supported"},
		},
		{
			name: "an encryption key without authorization_encryption encodings fails",
			post: true,
			metadata: map[string]any{
				"jwks": map[string]any{"keys": []any{map[string]any{"kty": "EC"}}},
				"authorization_encryption_alg_values_supported": []any{"ECDH-ES"},
			},
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			jwe:         "a.b.c.d.e",
			response:    presentation,
			want:        StatusFail,
			wantMessage: []string{"authorization_encryption_enc_values_supported"},
		},
		{
			name:        "no encrypted Request Object event fails",
			post:        true,
			metadata:    advertisingMetadata,
			events:      []any{map[string]any{"type": "vp_request_retrieved"}},
			jwe:         "a.b.c.d.e",
			response:    presentation,
			want:        StatusFail,
			wantMessage: []string{"vp_request_object_encrypted"},
		},
		{
			name:     "an unsupported encryption report is included when no encrypted event exists",
			post:     true,
			metadata: advertisingMetadata,
			events: []any{map[string]any{
				"type":   "vp_request_object_encryption_unsupported",
				"detail": map[string]any{"alg": "ECDH-ES", "enc": "A128GCM"},
			}},
			jwe:         "a.b.c.d.e",
			response:    presentation,
			want:        StatusFail,
			wantMessage: []string{"vp_request_object_encryption_unsupported", "A128GCM"},
		},
		{
			name:     "an algorithm the Wallet did not advertise fails",
			post:     true,
			metadata: advertisingMetadata,
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES+A256KW",
						"A256GCM",
						"request_object",
						"authorization_encryption",
					),
				),
			},
			jwe:         "a.b.c.d.e",
			response:    presentation,
			want:        StatusFail,
			wantMessage: []string{"alg", "ECDH-ES+A256KW"},
		},
		{
			name:     "an encoding the Wallet did not advertise fails",
			post:     true,
			metadata: advertisingMetadata,
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A128GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			jwe:         "a.b.c.d.e",
			response:    presentation,
			want:        StatusFail,
			wantMessage: []string{"enc", "A128GCM"},
		},
		{
			name:     "a missing request JWE fails",
			post:     true,
			metadata: advertisingMetadata,
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			response:    presentation,
			want:        StatusFail,
			wantMessage: []string{"authorization_request_jwe"},
		},
		{
			name:     "a JWE with four parts is not a compact JWE",
			post:     true,
			metadata: advertisingMetadata,
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			jwe:         "a.b.c.d",
			response:    presentation,
			want:        StatusFail,
			wantMessage: []string{"4 parts, expected 5"},
		},
		{
			name:     "no presentation after the encrypted Request Object fails",
			post:     true,
			metadata: advertisingMetadata,
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			jwe:         "a.b.c.d.e",
			want:        StatusFail,
			wantMessage: []string{"did not present"},
		},
		{
			name:     "an empty vp_token is no presentation",
			post:     true,
			metadata: advertisingMetadata,
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
			},
			jwe:         "a.b.c.d.e",
			response:    map[string]any{"vp_token": map[string]any{}},
			want:        StatusFail,
			wantMessage: []string{"did not present"},
		},
		{
			name:     "a Wallet that decrypts the Request Object with its advertised algorithms passes",
			post:     true,
			metadata: advertisingMetadata,
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES",
						"A256GCM",
						"authorization_encryption",
						"authorization_encryption",
					),
				),
				map[string]any{"type": "vp_presentation_response_received"},
			},
			jwe:      "a.b.c.d.e",
			response: presentation,
			want:     StatusPass,
			wantMessage: []string{
				"alg ECDH-ES",
				"enc A256GCM",
				"alg_source authorization_encryption",
				"enc_source authorization_encryption",
				"kid probe-wallet-enc",
			},
		},
		{
			name: "a Wallet that advertises request_object_encryption members also passes",
			post: true,
			metadata: map[string]any{
				"jwks": map[string]any{
					"keys": []any{map[string]any{"kty": "EC", "kid": "probe-wallet-enc"}},
				},
				"authorization_encryption_alg_values_supported":  []any{"ECDH-ES"},
				"authorization_encryption_enc_values_supported":  []any{"A256GCM"},
				"request_object_encryption_alg_values_supported": []any{"ECDH-ES+A256KW"},
				"request_object_encryption_enc_values_supported": []any{"A128CBC-HS256"},
			},
			events: []any{
				encryptedEvent(
					encryptedDetail(
						"ECDH-ES+A256KW",
						"A128CBC-HS256",
						"request_object",
						"request_object",
					),
				),
			},
			jwe:         "a.b.c.d.e",
			response:    presentation,
			want:        StatusPass,
			wantMessage: []string{"alg_source request_object", "enc_source request_object"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := test.input
			if value == nil {
				session := map[string]any{"events": test.events}
				observed := map[string]any{}
				if test.post {
					payload := map[string]any{}
					if test.metadata != nil {
						payload["wallet_metadata"] = test.metadata
					}
					observed["request_uri_payload"] = map[string]any{"value": payload}
				}
				session["observed"] = observed
				raw := map[string]any{}
				if test.jwe != "" {
					raw["authorization_request_jwe"] = test.jwe
				}
				if test.response != nil {
					raw["presentation_response_decrypted"] = test.response
				}
				if len(raw) > 0 {
					session["raw"] = raw
				}
				value = session
			}

			got := OID4VPEncryptedRequestObjectProcessedValidator{}.Validate(
				context.Background(),
				Input{Value: value},
			)

			require.Equal(t, test.want, got.Status, got.Message)
			for _, fragment := range test.wantMessage {
				require.Contains(t, got.Message, fragment)
			}
		})
	}
}
