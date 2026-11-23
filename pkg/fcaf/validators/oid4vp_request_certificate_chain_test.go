// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// requestObjectWithChain signs a Request Object with the chain's leaf key and
// delivers the chain in the x5c header, as the capture verifier does.
func requestObjectWithChain(t *testing.T, chain ...testCertificate) string {
	t.Helper()
	encoded := make([]any, 0, len(chain))
	for _, entry := range chain {
		encoded = append(encoded, base64.StdEncoding.EncodeToString(entry.der))
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"client_id": "x509_hash:x"})
	token.Header["x5c"] = encoded
	signed, err := token.SignedString(chain[0].key)
	require.NoError(t, err)
	return signed
}

func TestOID4VPRequestCertificateChainValidator(t *testing.T) {
	root := newTestCertificate(t, "root", true, nil)
	intermediate := newTestCertificate(t, "intermediate", true, &root)
	leafUnderIntermediate := newTestCertificate(t, "leaf", false, &intermediate)
	leafUnderRoot := newTestCertificate(t, "leaf", false, &root)
	selfSignedLeaf := newTestCertificate(t, "leaf", false, nil)
	strayLeaf := newTestCertificate(t, "stray", false, nil)

	incomplete := requestObjectWithChain(t, leafUnderIntermediate, intermediate)
	untrusted := requestObjectWithChain(t, leafUnderRoot, root)
	single := requestObjectWithChain(t, selfSignedLeaf)
	unlinked := requestObjectWithChain(t, leafUnderIntermediate, root)
	strayParent := requestObjectWithChain(t, leafUnderIntermediate, strayLeaf)
	fullChain := requestObjectWithChain(t, leafUnderIntermediate, intermediate, root)

	tests := []struct {
		name       string
		value      any
		shape      string
		wantStatus Status
	}{
		{"incomplete chain", incomplete, chainShapeIncomplete, StatusPass},
		{"complete chain is not incomplete", untrusted, chainShapeIncomplete, StatusFail},
		{"self-signed leaf is not incomplete", single, chainShapeIncomplete, StatusFail},
		{"unrelated parent does not form a chain", strayParent, chainShapeIncomplete, StatusFail},

		{"untrusted root", untrusted, chainShapeUntrustedRoot, StatusPass},
		{"incomplete chain carries no root", incomplete, chainShapeUntrustedRoot, StatusFail},
		{"single certificate carries no root", single, chainShapeUntrustedRoot, StatusFail},
		{"certificates that do not link", unlinked, chainShapeUntrustedRoot, StatusFail},

		{"self-signed leaf", single, chainShapeSelfSignedLeaf, StatusPass},
		{"two certificates are not a lone leaf", untrusted, chainShapeSelfSignedLeaf, StatusFail},

		{"leaf followed by its trust anchor", untrusted, chainShapeTrustAnchorIncluded, StatusPass},
		{
			"intermediate is not a trust anchor",
			incomplete,
			chainShapeTrustAnchorIncluded,
			StatusFail,
		},
		{
			"anchor at the end of a longer chain",
			fullChain,
			chainShapeTrustAnchorIncluded,
			StatusFail,
		},
		{"leaf alone carries no anchor", single, chainShapeTrustAnchorIncluded, StatusFail},
		{"root that did not issue the leaf", unlinked, chainShapeTrustAnchorIncluded, StatusFail},

		{"evidence is not a compact Request Object", 42, chainShapeIncomplete, StatusFail},
	}

	validator := OID4VPRequestCertificateChainValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: map[string]any{"shape": test.shape},
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}

	t.Run("unknown shape is a definition error", func(t *testing.T) {
		result := validator.Validate(context.Background(), Input{
			Value:  incomplete,
			Params: map[string]any{"shape": "expired"},
		})
		require.Equal(t, StatusError, result.Status, result.Message)
	})
}

// The deployed verifier leaf is issued directly by the trusted PID Issuer CA,
// so x5c=[leaf] is the complete chain: the shape must prove the anchor is
// absent and that the top-most certificate names the out-of-chain issuer
// (WS_RP_SM_RpIntegrity__016/018/020).
func TestOID4VPRequestCertificateChainTrustAnchorExcluded(t *testing.T) {
	root := newTestCertificate(t, "PID Issuer CA 02", true, nil)
	intermediate := newTestCertificate(t, "Fake Verifier Intermediate", true, &root)
	leafUnderRoot := newTestCertificate(t, "Fake Verifier", false, &root)
	leafUnderIntermediate := newTestCertificate(t, "Fake Verifier", false, &intermediate)
	selfSignedLeaf := newTestCertificate(t, "Fake Verifier", false, nil)
	// A parent that is neither self-signed nor the leaf's issuer proves the DN
	// link check, which requireLinkedChain alone does not perform.
	strayParent := newTestCertificate(t, "stray", true, &root)

	tests := []struct {
		name       string
		value      string
		issuer     string
		wantStatus Status
	}{
		{
			// x5c=[leaf]: complete because the issuer is a held trust anchor.
			name:       "leaf alone names its out-of-chain issuer",
			value:      requestObjectWithChain(t, leafUnderRoot),
			issuer:     "PID Issuer CA 02",
			wantStatus: StatusPass,
		},
		{
			name:       "multi-certificate chain stops before its anchor",
			value:      requestObjectWithChain(t, leafUnderIntermediate, intermediate),
			issuer:     "PID Issuer CA 02",
			wantStatus: StatusPass,
		},
		{
			name:       "leaf plus self-signed anchor fails",
			value:      requestObjectWithChain(t, leafUnderRoot, root),
			issuer:     "PID Issuer CA 02",
			wantStatus: StatusFail,
		},
		{
			name:       "self-signed certificate anywhere in x5c fails",
			value:      requestObjectWithChain(t, selfSignedLeaf),
			issuer:     "PID Issuer CA 02",
			wantStatus: StatusFail,
		},
		{
			name:       "leaf issued by another anchor fails",
			value:      requestObjectWithChain(t, leafUnderRoot),
			issuer:     "Other Issuer CA",
			wantStatus: StatusFail,
		},
		{
			name:       "certificates that do not link fail",
			value:      requestObjectWithChain(t, leafUnderRoot, strayParent),
			issuer:     "PID Issuer CA 02",
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPRequestCertificateChainValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value: test.value,
				Params: map[string]any{
					"shape":              chainShapeTrustAnchorExcluded,
					"issuer_common_name": test.issuer,
				},
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}

	t.Run("without issuer_common_name the shape still excludes the anchor", func(t *testing.T) {
		result := validator.Validate(context.Background(), Input{
			Value:  requestObjectWithChain(t, leafUnderRoot),
			Params: map[string]any{"shape": chainShapeTrustAnchorExcluded},
		})
		require.Equal(t, StatusPass, result.Status, result.Message)
		require.Contains(t, result.Message, "PID Issuer CA 02")
	})
}
