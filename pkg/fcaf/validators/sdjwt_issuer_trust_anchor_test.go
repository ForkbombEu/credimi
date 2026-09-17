// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testCertificate struct {
	certificate *x509.Certificate
	der         []byte
	key         *ecdsa.PrivateKey
}

func newTestCertificate(
	t *testing.T,
	commonName string,
	isCA bool,
	parent *testCertificate,
) testCertificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}
	if isCA {
		template.KeyUsage = x509.KeyUsageCertSign
	} else {
		template.KeyUsage = x509.KeyUsageDigitalSignature
	}
	signer, signerKey := template, key
	if parent != nil {
		signer, signerKey = parent.certificate, parent.key
	}
	der, err := x509.CreateCertificate(rand.Reader, template, signer, &key.PublicKey, signerKey)
	require.NoError(t, err)
	certificate, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return testCertificate{certificate: certificate, der: der, key: key}
}

func trustAnchorEvidence(chain ...testCertificate) string {
	encoded := make([]string, 0, len(chain))
	for _, certificate := range chain {
		encoded = append(encoded, base64.StdEncoding.EncodeToString(certificate.der))
	}
	header := map[string]any{"alg": "none", "typ": "dc+sd-jwt"}
	if len(encoded) > 0 {
		header["x5c"] = encoded
	}
	return sdjwtTokenWithHeader(header)
}

func sdjwtTokenWithHeader(header map[string]any) string {
	encodedHeader, _ := json.Marshal(header)
	encodedPayload, _ := json.Marshal(map[string]any{
		"vct":     "urn:eudi:pid:1",
		"_sd_alg": "sha-256",
	})
	return base64.RawURLEncoding.EncodeToString(encodedHeader) + "." +
		base64.RawURLEncoding.EncodeToString(encodedPayload) + ".signature~"
}

func TestIssuerTrustAnchorExcludedAcceptsChainWithoutARoot(t *testing.T) {
	validator := SDJWTIssuerTrustAnchorExcludedValidator{}
	root := newTestCertificate(t, "anchor", true, nil)
	intermediate := newTestCertificate(t, "intermediate", true, &root)
	leaf := newTestCertificate(t, "issuer", false, &intermediate)

	leafOnly := validator.Validate(context.Background(), Input{
		Value: trustAnchorEvidence(leaf),
	})
	require.Equal(t, StatusPass, leafOnly.Status, leafOnly.Message)

	withIntermediate := validator.Validate(context.Background(), Input{
		Value: trustAnchorEvidence(leaf, intermediate),
	})
	require.Equal(t, StatusPass, withIntermediate.Status, withIntermediate.Message)
}

func TestIssuerTrustAnchorExcludedRejectsEmbeddedAnchor(t *testing.T) {
	validator := SDJWTIssuerTrustAnchorExcludedValidator{}
	root := newTestCertificate(t, "anchor", true, nil)
	intermediate := newTestCertificate(t, "intermediate", true, &root)
	leaf := newTestCertificate(t, "issuer", false, &intermediate)

	withRoot := validator.Validate(context.Background(), Input{
		Value: trustAnchorEvidence(leaf, intermediate, root),
	})
	require.Equal(t, StatusFail, withRoot.Status, withRoot.Message)

	// The beta Capture issuer currently signs a self-signed end-entity
	// certificate, which is simultaneously the leaf and the trust anchor.
	selfSignedLeaf := newTestCertificate(t, "issuer", false, nil)
	selfSigned := validator.Validate(context.Background(), Input{
		Value: trustAnchorEvidence(selfSignedLeaf),
	})
	require.Equal(t, StatusFail, selfSigned.Status, selfSigned.Message)
}

func TestIssuerTrustAnchorExcludedRejectsBrokenChain(t *testing.T) {
	validator := SDJWTIssuerTrustAnchorExcludedValidator{}
	root := newTestCertificate(t, "anchor", true, nil)
	intermediate := newTestCertificate(t, "intermediate", true, &root)
	otherRoot := newTestCertificate(t, "other-anchor", true, nil)
	unrelated := newTestCertificate(t, "unrelated-intermediate", true, &otherRoot)
	leaf := newTestCertificate(t, "issuer", false, &intermediate)

	unlinked := validator.Validate(context.Background(), Input{
		Value: trustAnchorEvidence(leaf, unrelated),
	})
	require.Equal(t, StatusFail, unlinked.Status, unlinked.Message)

	missingChain := validator.Validate(context.Background(), Input{
		Value: trustAnchorEvidence(),
	})
	require.Equal(t, StatusFail, missingChain.Status, missingChain.Message)
}
