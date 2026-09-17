// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestOID4VPDIDSignedRequestValidator(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	const kid = "did:web:verifier.example#key-1"
	tests := []struct {
		name       string
		clientID   string
		wantStatus Status
	}{
		{
			name:       "accepts a DID client identifier signed by its DID key",
			clientID:   "decentralized_identifier:did:web:verifier.example",
			wantStatus: StatusPass,
		},
		{
			name:       "rejects a non-DID client identifier signed by the DID key",
			clientID:   "x509_hash:verifier.example",
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPDIDSignedRequestValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := signedDIDRequestObject(t, privateKey, kid, test.clientID)
			result := validator.Validate(context.Background(), Input{
				Value: didRequestEvidence(kid, privateKey, request),
			})

			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}

func signedDIDRequestObject(
	t *testing.T,
	privateKey *ecdsa.PrivateKey,
	kid string,
	clientID string,
) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"client_id": clientID})
	token.Header["kid"] = kid
	request, err := token.SignedString(privateKey)
	require.NoError(t, err)
	return request
}

func didRequestEvidence(kid string, privateKey *ecdsa.PrivateKey, request string) map[string]any {
	return map[string]any{
		"request_object": request,
		"did_document": map[string]any{
			"verificationMethod": []any{map[string]any{
				"id": kid,
				"publicKeyJwk": map[string]any{
					"kty": "EC",
					"crv": "P-256",
					"x":   base64.RawURLEncoding.EncodeToString(privateKey.X.FillBytes(make([]byte, 32))),
					"y":   base64.RawURLEncoding.EncodeToString(privateKey.Y.FillBytes(make([]byte, 32))),
				},
			}},
		},
	}
}
