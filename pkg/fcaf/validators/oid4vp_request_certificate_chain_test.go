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
