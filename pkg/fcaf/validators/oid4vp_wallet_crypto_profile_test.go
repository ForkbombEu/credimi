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

func cryptoProfileRequestObject(t *testing.T, clientMetadata map[string]any) string {
	t.Helper()
	encode := func(value any) string {
		raw, err := json.Marshal(value)
		require.NoError(t, err)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	return encode(map[string]any{"alg": "ES256"}) + "." +
		encode(map[string]any{"client_metadata": clientMetadata}) + ".sig"
}

func cryptoProfileSession(
	t *testing.T,
	walletMetadata map[string]any,
	digestAlgorithms []string,
	clientMetadata map[string]any,
) map[string]any {
	t.Helper()
	encodedMetadata, err := json.Marshal(walletMetadata)
	require.NoError(t, err)
	session := map[string]any{
		"status": "presentation_validated",
		"observed": map[string]any{"request_uri_payload": map[string]any{
			"value": map[string]any{"wallet_metadata": string(encodedMetadata)},
		}},
		"raw": map[string]any{
			"authorization_request_jwt": cryptoProfileRequestObject(t, clientMetadata),
		},
	}
	if digestAlgorithms != nil {
		entries := make([]any, 0, len(digestAlgorithms))
		for _, algorithm := range digestAlgorithms {
			entries = append(entries, map[string]any{"digest_algorithm": algorithm})
		}
		session["decoded_presentations"] = map[string]any{"pid": entries}
	}
	return session
}

func TestOID4VPWalletMetadataHashAlgorithms(t *testing.T) {
	signingOnly := map[string]any{"vp_formats_supported": map[string]any{
		"dc+sd-jwt": map[string]any{"sd-jwt_alg_values": []any{"ES256", "ES384"}},
	}}
	declaring := map[string]any{"fcaf_hash_algorithms": []any{"sha-256", "SHA-384"}}

	for _, test := range []struct {
		name      string
		metadata  map[string]any
		presented []string
		want      Status
	}{
		{"declares the presented hash algorithm", declaring, []string{"sha-384"}, StatusPass},
		{"ES384 is a signature algorithm, not a hash", signingOnly, []string{"sha-384"}, StatusFail},
		{"only SHA-256 presented", signingOnly, []string{"sha-256"}, StatusNotApplicable},
		{"nothing presented", signingOnly, nil, StatusNotApplicable},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := OID4VPWalletMetadataHashAlgorithmsValidator{}.Validate(
				context.Background(),
				Input{
					Value: cryptoProfileSession(t, test.metadata, test.presented, map[string]any{}),
				},
			)
			require.Equal(t, test.want, result.Status, result.Message)
		})
	}
}

func TestOID4VPPresentedDigestAlgorithms(t *testing.T) {
	params := map[string]any{"allowed": []any{"sha-256"}}
	for _, test := range []struct {
		name      string
		presented []string
		want      Status
	}{
		{"all SHA-256", []string{"sha-256", "SHA-256"}, StatusPass},
		{"one SHA-384 presentation", []string{"sha-256", "sha-384"}, StatusFail},
		{"no presentation", nil, StatusFail},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := OID4VPPresentedDigestAlgorithmsValidator{}.Validate(
				context.Background(),
				Input{
					Value: cryptoProfileSession(
						t,
						map[string]any{},
						test.presented,
						map[string]any{},
					),
					Params: params,
				},
			)
			require.Equal(t, test.want, result.Status, result.Message)
		})
	}
}

func TestOID4VPClientMetadataHashAlgorithms(t *testing.T) {
	params := map[string]any{"allowed": []any{"sha-256"}}
	for _, test := range []struct {
		name     string
		metadata map[string]any
		want     Status
	}{
		{
			"signature algorithms only",
			map[string]any{"vp_formats_supported": map[string]any{
				"dc+sd-jwt": map[string]any{"sd-jwt_alg_values": []any{"ES384", "HS512"}},
			}},
			StatusPass,
		},
		{"SHA-256 named", map[string]any{"hash_algorithms": []any{"sha-256"}}, StatusPass},
		{"SHA-512 named", map[string]any{"hash_algorithms": []any{"sha-512"}}, StatusFail},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := OID4VPClientMetadataHashAlgorithmsValidator{}.Validate(
				context.Background(),
				Input{
					Value:  cryptoProfileRequestObject(t, test.metadata),
					Params: params,
				},
			)
			require.Equal(t, test.want, result.Status, result.Message)
		})
	}
}

func TestOID4VPResponseEncUnsupportedErrorRequired(t *testing.T) {
	params := map[string]any{"wallet_only_enc": "A256GCM", "code": "access_denied"}
	narrowed := map[string]any{"encrypted_response_enc_values_supported": []any{"A128GCM"}}
	onlyA256 := map[string]any{"authorization_encryption_enc_values_supported": []any{"A256GCM"}}
	withError := func(session map[string]any, code string) map[string]any {
		delete(session, "decoded_presentations")
		session["status"] = "presentation_error"
		session["observed"].(map[string]any)["wallet_response"] = map[string]any{
			"value": map[string]any{"error": code},
		}
		return session
	}

	for _, test := range []struct {
		name  string
		value map[string]any
		want  Status
	}{
		{
			"wallet supports more than A256GCM",
			cryptoProfileSession(t, map[string]any{
				"authorization_encryption_enc_values_supported": []any{"A128GCM", "A256GCM"},
			}, []string{"sha-256"}, narrowed),
			StatusNotApplicable,
		},
		{
			"A256GCM-only wallet answers access_denied",
			withError(cryptoProfileSession(t, onlyA256, nil, narrowed), "access_denied"),
			StatusPass,
		},
		{
			"A256GCM-only wallet answers another error",
			withError(cryptoProfileSession(t, onlyA256, nil, narrowed), "invalid_request"),
			StatusFail,
		},
		{
			"verifier offered A256GCM",
			cryptoProfileSession(t, onlyA256, nil, map[string]any{
				"encrypted_response_enc_values_supported": []any{"A128GCM", "A256GCM"},
			}),
			StatusFail,
		},
		{
			"wallet states no enc values",
			cryptoProfileSession(t, map[string]any{}, nil, narrowed),
			StatusBlocked,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := OID4VPResponseEncUnsupportedErrorRequiredValidator{}.Validate(
				context.Background(),
				Input{Value: test.value, Params: params},
			)
			require.Equal(t, test.want, result.Status, result.Message)
		})
	}
}
