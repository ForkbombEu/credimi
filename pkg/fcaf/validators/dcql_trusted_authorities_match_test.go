// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTrustedAuthoritiesMatchMode(t *testing.T) {
	der, aki := testX509CertWithAKI(t)
	issuerJWT := buildSDJWTPresentation(t, der)

	dcqlQuery := map[string]any{
		"dcql_query": map[string]any{
			"credentials": []any{
				map[string]any{
					"id":     "pid-query",
					"format": "dc+sd-jwt",
					"meta": map[string]any{
						"vct_values": []any{"urn:eudi:pid:1"},
					},
					"trusted_authorities": []any{
						map[string]any{
							"type":   "aki",
							"values": []any{aki},
						},
					},
				},
			},
		},
		"vp_token": map[string]any{
			"pid-query": []any{issuerJWT},
		},
	}

	result := DCQLResponseConstraintsValidator{}.Validate(context.Background(), Input{
		Value: dcqlQuery,
		Params: map[string]any{
			"mode": "trusted_authorities_match",
		},
	})

	require.Equal(t, StatusPass, result.Status, result.Message)
}

func TestTrustedAuthoritiesMatchRejectsWrongIssuer(t *testing.T) {
	der, _ := testX509CertWithAKI(t)
	_, wrongAKI := testX509CertWithAKI(t)

	issuerJWT := buildSDJWTPresentation(t, der)

	dcqlQuery := map[string]any{
		"dcql_query": map[string]any{
			"credentials": []any{
				map[string]any{
					"id":     "pid-query",
					"format": "dc+sd-jwt",
					"meta": map[string]any{
						"vct_values": []any{"urn:eudi:pid:1"},
					},
					"trusted_authorities": []any{
						map[string]any{
							"type":   "aki",
							"values": []any{wrongAKI},
						},
					},
				},
			},
		},
		"vp_token": map[string]any{
			"pid-query": []any{issuerJWT},
		},
	}

	result := DCQLResponseConstraintsValidator{}.Validate(context.Background(), Input{
		Value: dcqlQuery,
		Params: map[string]any{
			"mode": "trusted_authorities_match",
		},
	})

	require.Equal(t, StatusFail, result.Status, result.Message)
	require.Contains(t, result.Message, "issuer does not match")
}

func TestTrustedAuthoritiesMatchRejectsMissingTA(t *testing.T) {
	der, _ := testX509CertWithAKI(t)
	issuerJWT := buildSDJWTPresentation(t, der)

	dcqlQuery := map[string]any{
		"dcql_query": map[string]any{
			"credentials": []any{
				map[string]any{
					"id":     "pid-query",
					"format": "dc+sd-jwt",
					"meta": map[string]any{
						"vct_values": []any{"urn:eudi:pid:1"},
					},
				},
			},
		},
		"vp_token": map[string]any{
			"pid-query": []any{issuerJWT},
		},
	}

	result := DCQLResponseConstraintsValidator{}.Validate(context.Background(), Input{
		Value: dcqlQuery,
		Params: map[string]any{
			"mode": "trusted_authorities_match",
		},
	})

	require.Equal(t, StatusFail, result.Status, result.Message)
	require.Contains(t, result.Message, "does not contain trusted_authorities")
}

func testX509CertWithAKI(t *testing.T) (der []byte, aki string) {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	keyID := sha1.Sum(publicKeyBytes)
	template := &x509.Certificate{
		SerialNumber:   big.NewInt(time.Now().UnixNano()),
		NotBefore:      time.Unix(0, 0),
		NotAfter:       time.Unix(9999999999, 0),
		AuthorityKeyId: keyID[:],
		SubjectKeyId:   keyID[:],
	}
	raw, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(raw)
	require.NoError(t, err)
	require.NotEmpty(t, cert.AuthorityKeyId, "self-signed cert must have AuthorityKeyId")
	return raw, base64.RawURLEncoding.EncodeToString(cert.AuthorityKeyId)
}

func buildSDJWTPresentation(t *testing.T, issuerCertDER []byte) string {
	t.Helper()
	certB64 := base64.StdEncoding.EncodeToString(issuerCertDER)
	header := map[string]any{
		"alg": "ES256",
		"typ": "dc+sd-jwt",
		"x5c": []any{certB64},
	}
	payload := map[string]any{
		"_sd_alg": "sha-256",
		"vct":     "urn:eudi:pid:1",
		"iss":     "https://issuer.example",
	}
	encode := func(v map[string]any) string {
		raw, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	return encode(header) + "." + encode(payload) + ".c2lnbmF0dXJl~"
}
