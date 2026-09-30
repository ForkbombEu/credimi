// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestOID4VPX509ClientIDLeafMismatchValidator(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		DNSNames:     []string{"verifier.example"},
	}
	leaf, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&privateKey.PublicKey,
		privateKey,
	)
	require.NoError(t, err)
	digest := sha256.Sum256(leaf)
	leafHash := base64.RawURLEncoding.EncodeToString(digest[:])

	signed := func(clientID string, withX5C bool) string {
		token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"client_id": clientID})
		if withX5C {
			token.Header["x5c"] = []string{base64.StdEncoding.EncodeToString(leaf)}
		}
		compact, err := token.SignedString(privateKey)
		require.NoError(t, err)
		return compact
	}

	tests := []struct {
		name    string
		request string
		prefix  string
		want    Status
	}{
		{
			"x509_hash of another certificate",
			signed("x509_hash:"+base64.RawURLEncoding.EncodeToString(make([]byte, 32)), true),
			"x509_hash",
			StatusPass,
		},
		{
			"x509_hash that binds the leaf",
			signed("x509_hash:"+leafHash, true),
			"x509_hash",
			StatusFail,
		},
		{
			"x509_san_dns name missing from the leaf",
			signed("x509_san_dns:mismatch.example", true),
			"x509_san_dns",
			StatusPass,
		},
		{
			"x509_san_dns name listed by the leaf",
			signed("x509_san_dns:verifier.example", true),
			"x509_san_dns",
			StatusFail,
		},
		{
			"client_id under a different prefix",
			signed("x509_hash:"+leafHash, true),
			"x509_san_dns",
			StatusFail,
		},
		{
			"no x5c leaf to compare against",
			signed("x509_san_dns:mismatch.example", false),
			"x509_san_dns",
			StatusFail,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := OID4VPX509ClientIDLeafMismatchValidator{}.Validate(context.Background(), Input{
				Value:  test.request,
				Params: map[string]any{"prefix": test.prefix},
			})
			require.Equal(t, test.want, got.Status, got.Message)
		})
	}
}

func TestOID4VPDIDSigningKeyUnlistedValidator(t *testing.T) {
	listedKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	unlistedKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	const didClientID = "decentralized_identifier:did:web:verifier.example"

	tests := []struct {
		name     string
		signer   *ecdsa.PrivateKey
		clientID string
		want     Status
	}{
		{"signed by a key the DID document does not list", unlistedKey, didClientID, StatusPass},
		{"signed by the listed DID key", listedKey, didClientID, StatusFail},
		{"not a DID client identifier", unlistedKey, "x509_hash:verifier.example", StatusFail},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := signedDIDRequestObject(t, test.signer, test.clientID)
			got := OID4VPDIDSigningKeyUnlistedValidator{}.Validate(context.Background(), Input{
				Value: didRequestEvidence(listedKey, request),
			})
			require.Equal(t, test.want, got.Status, got.Message)
		})
	}

	t.Run("a DID document without verification methods proves nothing", func(t *testing.T) {
		request := signedDIDRequestObject(t, unlistedKey, didClientID)
		got := OID4VPDIDSigningKeyUnlistedValidator{}.Validate(context.Background(), Input{
			Value: map[string]any{"request_object": request, "did_document": map[string]any{}},
		})
		require.Equal(t, StatusFail, got.Status, got.Message)
	})
}
