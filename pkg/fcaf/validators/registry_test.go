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
	"encoding/base64"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestRegistryRejectsDuplicateIDs(t *testing.T) {
	_, err := NewRegistry(EvidencePresentValidator{}, EvidencePresentValidator{})

	require.ErrorContains(t, err, "duplicate validator id")
}

func TestEvidencePresentValidator(t *testing.T) {
	got := EvidencePresentValidator{}.Validate(context.Background(), Input{Value: "value"})

	require.Equal(t, StatusPass, got.Status)
}

func TestFCAFBlockedValidator(t *testing.T) {
	got := FCAFBlockedValidator{}.Validate(context.Background(), Input{
		Params: map[string]any{"reason": "no OpenID Federation entity"},
	})
	require.Equal(t, StatusBlocked, got.Status)
	require.Equal(t, "no OpenID Federation entity", got.Message)

	got = FCAFBlockedValidator{}.Validate(context.Background(), Input{})
	require.Equal(t, StatusError, got.Status)
}

func TestJSONFieldStringPrefixValidator(t *testing.T) {
	v := JSONFieldStringPrefixValidator{}
	input := Input{
		Value:  map[string]any{"deeplink": "haip-vp://example"},
		Params: map[string]any{"field": "deeplink", "prefix": "haip-vp://"},
	}
	require.Equal(t, StatusPass, v.Validate(context.Background(), input).Status)

	input.Value = map[string]any{"deeplink": "openid4vp://example"}
	result := v.Validate(context.Background(), input)
	require.Equal(t, StatusFail, result.Status)
	require.Contains(t, result.Message, "does not start")

	for _, tt := range []struct {
		name    string
		value   any
		message string
	}{
		{name: "missing field", value: map[string]any{}, message: `required field "deeplink" is missing`},
		{
			name:    "non-string field",
			value:   map[string]any{"deeplink": []any{"haip-vp://example"}},
			message: `field "deeplink" is []interface {}, expected string`,
		},
		{name: "non-object evidence", value: "haip-vp://example", message: "input is string, expected object"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input.Value = tt.value
			got := v.Validate(context.Background(), input)
			require.Equal(t, StatusFail, got.Status)
			require.Equal(t, tt.message, got.Message)
		})
	}
}

func TestSDJWTClaimUTF8StringValidator(t *testing.T) {
	got := SDJWTClaimUTF8StringValidator{}.Validate(context.Background(), Input{
		Value: &evidence.SDJWTPresentation{Claims: map[string]any{"email": "person@example.test"}},
		Params: map[string]any{
			"claim": "email",
			"vectors": map[string]any{
				"positive": []string{"fixtures/fcaf/validators/sdjwt/email_utf8_positive.yaml"},
				"negative": []string{"fixtures/fcaf/validators/sdjwt/email_utf8_negative.yaml"},
			},
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestSDJWTClaimRFC5322EmailValidator(t *testing.T) {
	got := SDJWTClaimRFC5322EmailValidator{}.Validate(context.Background(), Input{
		Value: map[string]any{"email": "person@example.test"},
		Params: map[string]any{
			"claim": "email",
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestSDJWTClaimStringPrefixValidator(t *testing.T) {
	got := SDJWTClaimStringPrefixValidator{}.Validate(context.Background(), Input{
		Value: map[string]any{"vct": "urn:eudi:pid:1"},
		Params: map[string]any{
			"claim":  "vct",
			"prefix": "urn:eudi:pid:",
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestPIDSDJWTVCTValidator(t *testing.T) {
	got := PIDSDJWTVCTValidator{}.Validate(context.Background(), Input{
		Value: &evidence.SDJWTPresentation{Claims: map[string]any{
			"vct": "urn:eudi:pid:1",
		}},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestSDJWTIssuerX509HeaderValidator(t *testing.T) {
	certificate := testX509Certificate(t)
	tests := []struct {
		name        string
		headers     map[string]any
		wantStatus  Status
		wantMessage string
	}{
		{
			name:       "valid x5c chain",
			headers:    map[string]any{"x5c": []any{certificate}},
			wantStatus: StatusPass,
		},
		{
			name:       "missing x5c chain",
			headers:    map[string]any{},
			wantStatus: StatusFail,
		},
		{
			name:       "invalid certificate",
			headers:    map[string]any{"x5c": []any{"not-base64"}},
			wantStatus: StatusFail,
		},
		{
			name:        "reports index of non-string certificate",
			headers:     map[string]any{"x5c": []any{certificate, 42}},
			wantStatus:  StatusFail,
			wantMessage: "SD-JWT issuer x5c certificate 1 is not a string",
		},
		{
			name:        "rejects empty certificate entry",
			headers:     map[string]any{"x5c": []any{""}},
			wantStatus:  StatusFail,
			wantMessage: "SD-JWT issuer x5c certificate 0 is not a string",
		},
		{
			name: "rejects base64 that is not DER",
			headers: map[string]any{
				"x5c": []any{base64.StdEncoding.EncodeToString([]byte("not a certificate"))},
			},
			wantStatus:  StatusFail,
			wantMessage: "SD-JWT issuer x5c certificate 0 is not valid DER",
		},
		{
			name:        "rejects base64url encoded certificate",
			headers:     map[string]any{"x5c": []any{"-_-_"}},
			wantStatus:  StatusFail,
			wantMessage: "SD-JWT issuer x5c certificate 0 is not base64 encoded",
		},
		{
			name:        "rejects empty x5c array",
			headers:     map[string]any{"x5c": []any{}},
			wantStatus:  StatusFail,
			wantMessage: "SD-JWT issuer x5c chain is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SDJWTIssuerX509HeaderValidator{}.Validate(context.Background(), Input{
				Value: &evidence.SDJWTPresentation{ProtectedHeaders: tt.headers},
			})

			require.Equal(t, tt.wantStatus, got.Status, got.Message)
			require.Contains(t, got.Message, tt.wantMessage)
		})
	}

	got := SDJWTIssuerX509HeaderValidator{}.Validate(context.Background(), Input{Value: 42})
	require.Equal(t, StatusFail, got.Status)
	require.Equal(t, "SD-JWT issuer protected headers are missing", got.Message)
}

func TestSDJWTCNFConformsValidator(t *testing.T) {
	p256 := elliptic.P256().Params()
	p256X := base64.RawURLEncoding.EncodeToString(p256.Gx.FillBytes(make([]byte, 32)))
	p256Y := base64.RawURLEncoding.EncodeToString(p256.Gy.FillBytes(make([]byte, 32)))
	tests := []struct {
		name        string
		issuerCNF   any
		omitCNF     bool
		wantStatus  Status
		wantMessage string
	}{
		{
			name: "valid EC public JWK",
			issuerCNF: map[string]any{
				"jwk": map[string]any{
					"kty": "EC",
					"crv": "P-256",
					"x":   p256X,
					"y":   p256Y,
				},
			},
			wantStatus: StatusPass,
		},
		{
			name:       "valid key identifier",
			issuerCNF:  map[string]any{"kid": "holder-key-1"},
			wantStatus: StatusPass,
		},
		{
			name:       "valid HTTPS JWK set reference",
			issuerCNF:  map[string]any{"jku": "https://holder.example/keys.json"},
			wantStatus: StatusPass,
		},
		{
			name: "valid HTTPS JWK set reference with key identifier",
			issuerCNF: map[string]any{
				"jku": "https://holder.example/keys.json",
				"kid": "holder-key-1",
			},
			wantStatus: StatusPass,
		},
		{
			name:       "valid compact JWE",
			issuerCNF:  map[string]any{"jwe": "header.key.iv.ciphertext.tag"},
			wantStatus: StatusPass,
		},
		{
			name:       "missing cnf",
			omitCNF:    true,
			wantStatus: StatusFail,
		},
		{
			name:       "cnf is not an object",
			issuerCNF:  "holder-key-1",
			wantStatus: StatusFail,
		},
		{
			name:       "empty cnf",
			issuerCNF:  map[string]any{},
			wantStatus: StatusFail,
		},
		{
			name: "JWK contains private key material",
			issuerCNF: map[string]any{
				"jwk": map[string]any{
					"kty": "EC",
					"crv": "P-256",
					"x":   p256X,
					"y":   p256Y,
					"d":   "private",
				},
			},
			wantStatus: StatusFail,
		},
		{
			name: "EC JWK is missing a coordinate",
			issuerCNF: map[string]any{
				"jwk": map[string]any{
					"kty": "EC",
					"crv": "P-256",
					"x":   p256X,
				},
			},
			wantStatus: StatusFail,
		},
		{
			name: "EC JWK coordinate is not base64url",
			issuerCNF: map[string]any{
				"jwk": map[string]any{
					"kty": "EC",
					"crv": "P-256",
					"x":   "not+base64url",
					"y":   p256Y,
				},
			},
			wantStatus: StatusFail,
		},
		{
			name: "EC JWK coordinate has wrong length",
			issuerCNF: map[string]any{
				"jwk": map[string]any{
					"kty": "EC",
					"crv": "P-256",
					"x":   base64.RawURLEncoding.EncodeToString(make([]byte, 31)),
					"y":   p256Y,
				},
			},
			wantStatus: StatusFail,
		},
		{
			name:       "JWK set URL is not HTTPS",
			issuerCNF:  map[string]any{"jku": "http://holder.example/keys.json", "kid": "key-1"},
			wantStatus: StatusFail,
		},
		{
			name: "EC JWK coordinates are not on the declared curve",
			issuerCNF: map[string]any{
				"jwk": map[string]any{
					"kty": "EC",
					"crv": "P-256",
					"x":   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
					"y":   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
				},
			},
			wantStatus: StatusFail,
		},
		{
			name: "multiple proof keys",
			issuerCNF: map[string]any{
				"jwk": map[string]any{
					"kty": "OKP",
					"crv": "Ed25519",
					"x":   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
				},
				"jwe": "header.key.iv.ciphertext.tag",
			},
			wantStatus: StatusFail,
		},
		{
			name:       "unknown confirmation method only",
			issuerCNF:  map[string]any{"custom": "key"},
			wantStatus: StatusFail,
		},
		{
			name:        "JWK member is not an object",
			issuerCNF:   map[string]any{"jwk": "holder-key"},
			wantStatus:  StatusFail,
			wantMessage: `cnf member "jwk" must be an object`,
		},
		{
			name:        "JWK is empty",
			issuerCNF:   map[string]any{"jwk": map[string]any{}},
			wantStatus:  StatusFail,
			wantMessage: "cnf JWK is invalid: JWK is empty",
		},
		{
			name:        "JWK kty is missing",
			issuerCNF:   map[string]any{"jwk": map[string]any{"crv": "P-256"}},
			wantStatus:  StatusFail,
			wantMessage: "JWK kty must be a non-empty string",
		},
		{
			name:        "symmetric JWK kty is not a public key",
			issuerCNF:   map[string]any{"jwk": map[string]any{"kty": "oct"}},
			wantStatus:  StatusFail,
			wantMessage: `unsupported public JWK kty "oct"`,
		},
		{
			name:        "EC JWK crv is missing",
			issuerCNF:   map[string]any{"jwk": map[string]any{"kty": "EC", "x": p256X, "y": p256Y}},
			wantStatus:  StatusFail,
			wantMessage: "JWK crv must be a non-empty string",
		},
		{
			name: "EC JWK uses unsupported curve",
			issuerCNF: map[string]any{"jwk": map[string]any{
				"kty": "EC",
				"crv": "secp256k1",
				"x":   p256X,
				"y":   p256Y,
			}},
			wantStatus:  StatusFail,
			wantMessage: `unsupported EC curve "secp256k1"`,
		},
		{
			name: "RSA public JWK",
			issuerCNF: map[string]any{"jwk": map[string]any{
				"kty": "RSA",
				"n":   base64.RawURLEncoding.EncodeToString([]byte{0xc3, 0x01, 0x02}),
				"e":   "AQAB",
			}},
			wantStatus: StatusPass,
		},
		{
			name:        "RSA JWK is missing modulus",
			issuerCNF:   map[string]any{"jwk": map[string]any{"kty": "RSA", "e": "AQAB"}},
			wantStatus:  StatusFail,
			wantMessage: "JWK n must be a non-empty string",
		},
		{
			name: "RSA JWK exponent is padded base64",
			issuerCNF: map[string]any{"jwk": map[string]any{
				"kty": "RSA",
				"n":   base64.RawURLEncoding.EncodeToString([]byte{0xc3, 0x01, 0x02}),
				"e":   "AQAB==",
			}},
			wantStatus:  StatusFail,
			wantMessage: "JWK e must be unpadded base64url",
		},
		{
			name:        "OKP JWK crv is missing",
			issuerCNF:   map[string]any{"jwk": map[string]any{"kty": "OKP", "x": p256X}},
			wantStatus:  StatusFail,
			wantMessage: "JWK crv must be a non-empty string",
		},
		{
			name: "OKP JWK uses unsupported curve",
			issuerCNF: map[string]any{"jwk": map[string]any{
				"kty": "OKP",
				"crv": "Ed25519ph",
				"x":   p256X,
			}},
			wantStatus:  StatusFail,
			wantMessage: `unsupported OKP curve "Ed25519ph"`,
		},
		{
			name: "Ed448 JWK has Ed25519 key length",
			issuerCNF: map[string]any{"jwk": map[string]any{
				"kty": "OKP",
				"crv": "Ed448",
				"x":   base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
			}},
			wantStatus:  StatusFail,
			wantMessage: "JWK x must decode to 57 bytes",
		},
		{
			name:        "JWE has too few segments",
			issuerCNF:   map[string]any{"jwe": "header.payload.signature"},
			wantStatus:  StatusFail,
			wantMessage: `cnf member "jwe" must be a compact JWE`,
		},
		{
			name:        "JWK set URL is not a string",
			issuerCNF:   map[string]any{"jku": []any{"https://holder.example/keys.json"}},
			wantStatus:  StatusFail,
			wantMessage: `cnf member "jku" must be a string`,
		},
		{
			name:        "JWK set URL has no host",
			issuerCNF:   map[string]any{"jku": "https:///keys.json"},
			wantStatus:  StatusFail,
			wantMessage: `cnf member "jku" must be an HTTPS URI`,
		},
		{
			name:        "key identifier is empty",
			issuerCNF:   map[string]any{"kid": ""},
			wantStatus:  StatusFail,
			wantMessage: `cnf member "kid" must be a non-empty string`,
		},
		{
			name: "JWK and key identifier identify two keys",
			issuerCNF: map[string]any{
				"jwk": map[string]any{"kty": "EC", "crv": "P-256", "x": p256X, "y": p256Y},
				"kid": "holder-key-1",
			},
			wantStatus:  StatusFail,
			wantMessage: "cnf identifies more than one proof-of-possession key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := map[string]any{}
			if !tt.omitCNF {
				payload["cnf"] = tt.issuerCNF
			}
			got := SDJWTCNFConformsValidator{}.Validate(context.Background(), Input{
				Value: &evidence.SDJWTPresentation{IssuerPayload: payload},
			})

			require.Equal(t, tt.wantStatus, got.Status, got.Message)
			require.Contains(t, got.Message, tt.wantMessage)
		})
	}

	got := SDJWTCNFConformsValidator{}.Validate(context.Background(), Input{Value: []any{}})
	require.Equal(t, StatusFail, got.Status)
	require.Equal(t, "SD-JWT issuer payload is missing", got.Message)
}

func testX509Certificate(t *testing.T) string {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(3600, 0),
	}
	der, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&privateKey.PublicKey,
		privateKey,
	)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(der)
}

func TestPIDSDJWTMandatoryClaimsValidator(t *testing.T) {
	got := PIDSDJWTMandatoryClaimsValidator{}.Validate(context.Background(), Input{
		Value: &evidence.SDJWTPresentation{Claims: map[string]any{
			"family_name":       "Trotter",
			"given_name":        "Filippo",
			"birthdate":         "1999-11-01",
			"place_of_birth":    map[string]any{"country": "IT"},
			"nationalities":     []any{"IT"},
			"date_of_expiry":    "2026-10-11",
			"issuing_authority": "GR Administrative authority",
			"issuing_country":   "GR",
			"email":             "person@example.test",
		}},
		Params: map[string]any{
			"required_elements": []string{
				"family_name",
				"given_name",
				"birthdate",
				"place_of_birth",
				"nationalities",
				"date_of_expiry",
				"issuing_authority",
				"issuing_country",
			},
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestMDocNamespaceElementPresentValidator(t *testing.T) {
	got := MDocNamespaceElementPresentValidator{}.Validate(context.Background(), Input{
		Value: &evidence.MDocPresentation{
			Documents: []evidence.MDocDocument{{
				DocType: "eu.europa.ec.eudi.pid.1",
			}},
			Namespaces: map[string]map[string]evidence.MDocElement{
				"eu.europa.ec.eudi.pid.1": {
					"email_address": {
						Identifier: "email_address",
						Value:      "person@example.test",
						MajorType:  3,
					},
				},
			},
		},
		Params: map[string]any{
			"namespace": "eu.europa.ec.eudi.pid.1",
			"element":   "email_address",
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestJOSEJWEEncryptedResponseValidator(t *testing.T) {
	tests := []struct {
		name        string
		value       any
		wantStatus  Status
		wantMessage string
	}{
		{name: "compact JWE", value: "a.b.c.d.e", wantStatus: StatusPass},
		{
			name:        "compact JWS is not encrypted",
			value:       "a.b.c",
			wantStatus:  StatusFail,
			wantMessage: "response is not a compact JWE",
		},
		{
			name:       "JSON serialized JWE",
			value:      map[string]any{"protected": "e30", "ciphertext": "abc"},
			wantStatus: StatusPass,
		},
		{
			name:        "plain JSON response",
			value:       map[string]any{"vp_token": map[string]any{}},
			wantStatus:  StatusFail,
			wantMessage: "response does not contain JWE ciphertext",
		},
		{
			name:        "unsupported value",
			value:       []any{"a.b.c.d.e"},
			wantStatus:  StatusFail,
			wantMessage: "input is []interface {}, expected string or object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JOSEJWEEncryptedResponseValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value},
			)

			require.Equal(t, tt.wantStatus, got.Status)
			require.Contains(t, got.Message, tt.wantMessage)
		})
	}
}

func TestJOSEJWSSignedRequestValidator(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	now := time.Now()
	certificateDER, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{
			SerialNumber: big.NewInt(1),
			NotBefore:    now.Add(-time.Hour),
			NotAfter:     now.Add(time.Hour),
		},
		&x509.Certificate{
			SerialNumber: big.NewInt(1),
			NotBefore:    now.Add(-time.Hour),
			NotAfter:     now.Add(time.Hour),
		},
		&privateKey.PublicKey,
		privateKey,
	)
	require.NoError(t, err)

	signedRequest := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"nonce": "nonce-1"})
	signedRequest.Header["x5c"] = []string{base64.StdEncoding.EncodeToString(certificateDER)}
	compactRequest, err := signedRequest.SignedString(privateKey)
	require.NoError(t, err)

	validator := JOSEJWSSignedRequestValidator{}
	require.Equal(
		t,
		StatusPass,
		validator.Validate(context.Background(), Input{Value: compactRequest}).Status,
	)
	require.Equal(
		t,
		StatusFail,
		validator.Validate(context.Background(), Input{Value: "header.payload.signature"}).Status,
	)

	expiredRequest := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"nonce": "nonce-1",
		"iat":   now.Add(-time.Hour).Unix(),
		"exp":   now.Add(-55 * time.Minute).Unix(),
	})
	expiredRequest.Header["x5c"] = []string{base64.StdEncoding.EncodeToString(certificateDER)}
	compactExpiredRequest, err := expiredRequest.SignedString(privateKey)
	require.NoError(t, err)
	require.Equal(
		t,
		StatusPass,
		validator.Validate(context.Background(), Input{Value: compactExpiredRequest}).Status,
		"captured Request Objects are validated after their exp",
	)
}

func TestJOSEJWSInvalidSignatureValidator(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	now := time.Now()
	certificateDER, err := x509.CreateCertificate(
		rand.Reader,
		&x509.Certificate{
			SerialNumber: big.NewInt(1),
			NotBefore:    now.Add(-time.Hour),
			NotAfter:     now.Add(time.Hour),
		},
		&x509.Certificate{
			SerialNumber: big.NewInt(1),
			NotBefore:    now.Add(-time.Hour),
			NotAfter:     now.Add(time.Hour),
		},
		&privateKey.PublicKey,
		privateKey,
	)
	require.NoError(t, err)

	request := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"nonce": "nonce-1"})
	request.Header["x5c"] = []string{base64.StdEncoding.EncodeToString(certificateDER)}
	compactRequest, err := request.SignedString(privateKey)
	require.NoError(t, err)
	parts := strings.Split(compactRequest, ".")
	require.Len(t, parts, 3)
	if parts[2][0] == 'A' {
		parts[2] = "B" + parts[2][1:]
	} else {
		parts[2] = "A" + parts[2][1:]
	}

	got := JOSEJWSInvalidSignatureValidator{}.Validate(
		context.Background(),
		Input{Value: strings.Join(parts, ".")},
	)
	require.Equal(t, StatusPass, got.Status)
}

func TestJSONFieldRequiredValidator(t *testing.T) {
	got := JSONFieldRequiredValidator{}.Validate(context.Background(), Input{
		Value: map[string]any{"email": "person@example.test"},
		Params: map[string]any{
			"field": "email",
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestJSONFieldEqualsValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      map[string]any
		expected   any
		wantStatus Status
	}{
		{
			name:       "matching value",
			value:      map[string]any{"request_uri_method": "post"},
			expected:   "post",
			wantStatus: StatusPass,
		},
		{
			name:       "different value",
			value:      map[string]any{"request_uri_method": "get"},
			expected:   "post",
			wantStatus: StatusFail,
		},
		{
			name:       "missing field",
			value:      map[string]any{},
			expected:   "post",
			wantStatus: StatusFail,
		},
		{
			name:       "different type",
			value:      map[string]any{"request_uri_method": 1},
			expected:   "1",
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JSONFieldEqualsValidator{}.Validate(context.Background(), Input{
				Value: tt.value,
				Params: map[string]any{
					"field": "request_uri_method",
					"value": tt.expected,
				},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestEvidenceNonEmptyValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name:       "screenshot URLs",
			value:      []any{"https://example.test/selection.png"},
			wantStatus: StatusPass,
		},
		{name: "empty screenshot URLs", value: []any{}, wantStatus: StatusFail},
		{name: "missing evidence", value: nil, wantStatus: StatusFail},
		{name: "text", value: "https://example.test", wantStatus: StatusPass},
		{name: "empty text", value: "", wantStatus: StatusFail},
		{name: "string list", value: []string{"a"}, wantStatus: StatusPass},
		{name: "empty string list", value: []string{}, wantStatus: StatusFail},
		{name: "object", value: map[string]any{"vp_token": "x"}, wantStatus: StatusPass},
		{name: "empty object", value: map[string]any{}, wantStatus: StatusFail},
		{name: "unsupported type is not evidence", value: 42, wantStatus: StatusFail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvidenceNonEmptyValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value},
			)
			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestEvidenceMinimumItemsValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		minItems   int
		wantStatus Status
	}{
		{name: "enough items", value: []any{"one", "two"}, minItems: 2, wantStatus: StatusPass},
		{name: "too few items", value: []any{"one"}, minItems: 2, wantStatus: StatusFail},
		{name: "not an array", value: "one", minItems: 1, wantStatus: StatusFail},
		{name: "string array", value: []string{"one", "two"}, minItems: 2, wantStatus: StatusPass},
		{name: "short string array", value: []string{"one"}, minItems: 2, wantStatus: StatusFail},
		{name: "invalid minimum", value: []any{"one"}, minItems: 0, wantStatus: StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvidenceMinimumItemsValidator{}.Validate(context.Background(), Input{
				Value:  tt.value,
				Params: map[string]any{"min_items": tt.minItems},
			})
			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestJSONFieldPresenceValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		field      string
		present    bool
		wantStatus Status
	}{
		{
			name:       "required field present",
			value:      map[string]any{"vp_token": map[string]any{}},
			field:      "vp_token",
			present:    true,
			wantStatus: StatusPass,
		},
		{
			name:       "forbidden field absent",
			value:      map[string]any{"vp_token": map[string]any{}},
			field:      "access_token",
			present:    false,
			wantStatus: StatusPass,
		},
		{
			name:       "required field absent",
			value:      map[string]any{},
			field:      "vp_token",
			present:    true,
			wantStatus: StatusFail,
		},
		{
			name:       "forbidden field present",
			value:      map[string]any{"access_token": "token"},
			field:      "access_token",
			present:    false,
			wantStatus: StatusFail,
		},
		{
			name:       "non-object evidence",
			value:      "vp_token=value",
			field:      "vp_token",
			present:    true,
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JSONFieldPresenceValidator{}.Validate(context.Background(), Input{
				Value: tt.value,
				Params: map[string]any{
					"field":   tt.field,
					"present": tt.present,
				},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestJWTHeaderFieldEqualsValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name:       "matching typ",
			value:      "eyJ0eXAiOiJvYXV0aC1hdXRoei1yZXErand0IiwiYWxnIjoiRVMyNTYifQ.e30.signature",
			wantStatus: StatusPass,
		},
		{
			name:       "different typ",
			value:      "eyJ0eXAiOiJKV1QiLCJhbGciOiJFUzI1NiJ9.e30.signature",
			wantStatus: StatusFail,
		},
		{
			name:       "missing typ",
			value:      "eyJhbGciOiJFUzI1NiJ9.e30.signature",
			wantStatus: StatusFail,
		},
		{
			name:       "header not base64url",
			value:      "eyJ0eXAi=.e30.signature",
			wantStatus: StatusFail,
		},
		{
			name:       "not compact JWT",
			value:      "invalid",
			wantStatus: StatusFail,
		},
		{
			name:       "not string",
			value:      map[string]any{},
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JWTHeaderFieldEqualsValidator{}.Validate(context.Background(), Input{
				Value: tt.value,
				Params: map[string]any{
					"field": "typ",
					"value": "oauth-authz-req+jwt",
				},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestJWTPayloadFieldEqualsValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name:       "matching response type",
			value:      "e30.eyJyZXNwb25zZV90eXBlIjoidnBfdG9rZW4ifQ.signature",
			wantStatus: StatusPass,
		},
		{
			name:       "different response type",
			value:      "e30.eyJyZXNwb25zZV90eXBlIjoiY29kZSJ9.signature",
			wantStatus: StatusFail,
		},
		{
			name:       "missing response type",
			value:      "e30.e30.signature",
			wantStatus: StatusFail,
		},
		{
			name:       "not compact JWT",
			value:      "invalid",
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JWTPayloadFieldEqualsValidator{}.Validate(context.Background(), Input{
				Value: tt.value,
				Params: map[string]any{
					"field": "response_type",
					"value": "vp_token",
				},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestJWTPayloadObjectKeysAllowedValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantStatus Status
	}{
		{
			name: "defined metadata keys",
			value: "e30." +
				"eyJjbGllbnRfbWV0YWRhdGEiOnsiandrcyI6e30sInZwX2Zvcm1hdHNfc3VwcG9ydGVkIjp7fX19." +
				"signature",
			wantStatus: StatusPass,
		},
		{
			name: "undefined metadata key",
			value: "e30." +
				"eyJjbGllbnRfbWV0YWRhdGEiOnsiandrcyI6e30sInVua25vd24iOnRydWV9fQ." +
				"signature",
			wantStatus: StatusFail,
		},
		{
			name:       "missing metadata",
			value:      "e30.e30.signature",
			wantStatus: StatusFail,
		},
		{
			name: "metadata is not an object",
			value: "e30." + base64.RawURLEncoding.EncodeToString(
				[]byte(`{"client_metadata":"jwks"}`),
			) + ".signature",
			wantStatus: StatusFail,
		},
		{
			name:       "malformed JWT",
			value:      "e30.signature",
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JWTPayloadObjectKeysAllowedValidator{}.Validate(context.Background(), Input{
				Value: tt.value,
				Params: map[string]any{
					"field": "client_metadata",
					"allowed_keys": []string{
						"jwks",
						"vp_formats_supported",
						"encrypted_response_enc_values_supported",
					},
				},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestJWTPayloadFieldPresenceValidator(t *testing.T) {
	token := "e30.eyJjbGllbnRfaWQiOiJjbGllbnQtMSJ9.signature"
	tests := []struct {
		name       string
		field      string
		present    bool
		wantStatus Status
	}{
		{name: "required field present", field: "client_id", present: true, wantStatus: StatusPass},
		{name: "forbidden field absent", field: "iss", present: false, wantStatus: StatusPass},
		{name: "required field absent", field: "iss", present: true, wantStatus: StatusFail},
		{
			name:       "forbidden field present",
			field:      "client_id",
			present:    false,
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JWTPayloadFieldPresenceValidator{}.Validate(context.Background(), Input{
				Value: token,
				Params: map[string]any{
					"field":   tt.field,
					"present": tt.present,
				},
			})

			require.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestJWTHeaderFieldPresenceValidator(t *testing.T) {
	validator := JWTHeaderFieldPresenceValidator{}
	require.Equal(
		t,
		StatusPass,
		validator.Validate(
			context.Background(),
			Input{
				Value:  "eyJhbGciOiJFUzI1NiJ9.e30.signature",
				Params: map[string]any{"field": "typ", "present": false},
			},
		).Status,
	)
	require.Equal(
		t,
		StatusFail,
		validator.Validate(
			context.Background(),
			Input{
				Value:  "eyJ0eXAiOiJKV1QifQ.e30.signature",
				Params: map[string]any{"field": "typ", "present": false},
			},
		).Status,
	)
}

func TestJWTPayloadFieldsDifferValidator(t *testing.T) {
	validator := JWTPayloadFieldsDifferValidator{}
	require.Equal(
		t,
		StatusPass,
		validator.Validate(
			context.Background(),
			Input{
				Value:  "e30.eyJjbGllbnRfaWQiOiJjbGllbnQtMSIsImlzcyI6ImNsaWVudC0yIn0.signature",
				Params: map[string]any{"first": "client_id", "second": "iss"},
			},
		).Status,
	)
	require.Equal(
		t,
		StatusFail,
		validator.Validate(
			context.Background(),
			Input{
				Value:  "e30.eyJjbGllbnRfaWQiOiJjbGllbnQtMSIsImlzcyI6ImNsaWVudC0xIn0.signature",
				Params: map[string]any{"first": "client_id", "second": "iss"},
			},
		).Status,
	)

	missing := validator.Validate(context.Background(), Input{
		Value:  "e30.eyJjbGllbnRfaWQiOiJjbGllbnQtMSJ9.signature",
		Params: map[string]any{"first": "client_id", "second": "iss"},
	})
	require.Equal(t, StatusFail, missing.Status)
	require.Equal(
		t,
		`JWT payload fields "client_id" and "iss" must both be present`,
		missing.Message,
	)

	notJWT := validator.Validate(context.Background(), Input{
		Value:  "e30.signature",
		Params: map[string]any{"first": "client_id", "second": "iss"},
	})
	require.Equal(t, StatusFail, notJWT.Status)
	require.Equal(t, "input is not a compact JWT", notJWT.Message)

	unconfigured := validator.Validate(context.Background(), Input{
		Value:  "e30.e30.signature",
		Params: map[string]any{"first": "client_id"},
	})
	require.Equal(t, StatusError, unconfigured.Status)
}

func TestCompactJWTPartRejectsMalformedSegments(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		message string
	}{
		{
			name:    "not a string",
			value:   []byte("e30.e30.sig"),
			message: "input is []uint8, expected compact JWT",
		},
		{name: "four segments", value: "e30.e30.sig.extra", message: "input is not a compact JWT"},
		{
			name:    "payload not base64url",
			value:   "e30.e30=.sig",
			message: "JWT part 1 is not base64url encoded",
		},
		{
			name: "payload is a JSON array",
			value: "e30." + base64.RawURLEncoding.EncodeToString(
				[]byte(`["client_id"]`),
			) + ".sig",
			message: "JWT part 1 is not a JSON object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JWTPayloadFieldPresenceValidator{}.Validate(context.Background(), Input{
				Value:  tt.value,
				Params: map[string]any{"field": "client_id", "present": true},
			})

			require.Equal(t, StatusFail, got.Status)
			require.Equal(t, tt.message, got.Message)
		})
	}
}

func TestGenericValidatorsRejectIncompleteConfiguration(t *testing.T) {
	token := "e30.e30.signature"
	object := map[string]any{"field": "value"}
	tests := []struct {
		name      string
		validator Validator
		value     any
		params    map[string]any
		message   string
	}{
		{
			name:      "string prefix requires field",
			validator: JSONFieldStringPrefixValidator{},
			value:     object,
			params:    map[string]any{"prefix": "x"},
			message:   fieldParamRequired,
		},
		{
			name:      "string prefix requires prefix",
			validator: JSONFieldStringPrefixValidator{},
			value:     object,
			params:    map[string]any{"field": "field"},
			message:   "prefix param is required",
		},
		{
			name:      "json presence requires field",
			validator: JSONFieldPresenceValidator{},
			value:     object,
			params:    map[string]any{"present": true},
			message:   fieldParamRequired,
		},
		{
			name:      "json presence requires present",
			validator: JSONFieldPresenceValidator{},
			value:     object,
			params:    map[string]any{"field": "field"},
			message:   "present param is required",
		},
		{
			name:      "json equals requires field",
			validator: JSONFieldEqualsValidator{},
			value:     object,
			params:    map[string]any{"value": "value"},
			message:   fieldParamRequired,
		},
		{
			name:      "json required requires field",
			validator: JSONFieldRequiredValidator{},
			value:     object,
			params:    map[string]any{},
			message:   fieldParamRequired,
		},
		{
			name:      "jwt equals requires field",
			validator: JWTPayloadFieldEqualsValidator{},
			value:     token,
			params:    map[string]any{"value": "value"},
			message:   fieldParamRequired,
		},
		{
			name:      "jwt presence requires field",
			validator: JWTHeaderFieldPresenceValidator{},
			value:     token,
			params:    map[string]any{"present": true},
			message:   fieldParamRequired,
		},
		{
			name:      "jwt presence requires present",
			validator: JWTHeaderFieldPresenceValidator{},
			value:     token,
			params:    map[string]any{"field": "typ"},
			message:   "present param is required",
		},
		{
			name:      "object keys requires field",
			validator: JWTPayloadObjectKeysAllowedValidator{},
			value:     token,
			params:    map[string]any{"allowed_keys": []string{"jwks"}},
			message:   fieldParamRequired,
		},
		{
			name:      "object keys requires allowed_keys",
			validator: JWTPayloadObjectKeysAllowedValidator{},
			value:     token,
			params:    map[string]any{"field": "client_metadata"},
			message:   "allowed_keys param is required",
		},
		{
			name:      "minimum items rejects non-numeric min_items",
			validator: EvidenceMinimumItemsValidator{},
			value:     []any{"one"},
			params:    map[string]any{"min_items": "one"},
			message:   "cannot unmarshal string into Go struct field .min_items of type int",
		},
		{
			name:      "digest algorithm rejects non-string param",
			validator: SDJWTPresentationDigestAlgorithmValidator{},
			value:     map[string]any{"status": "done"},
			params:    map[string]any{"digest_algorithm": 256},
			message:   "cannot unmarshal number into Go struct field .digest_algorithm of type string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.validator.Validate(context.Background(), Input{
				Value:  tt.value,
				Params: tt.params,
			})

			require.Equal(t, StatusError, got.Status, got.Message)
			require.Contains(t, got.Message, tt.message)
		})
	}
}

func TestSDJWTClaimPresentValidator(t *testing.T) {
	got := SDJWTClaimPresentValidator{}.Validate(context.Background(), Input{
		Value: map[string]any{"email": "person@example.test"},
		Params: map[string]any{
			"claim": "email",
		},
	})

	require.Equal(t, StatusPass, got.Status)
}

func TestSDJWTKBJWTPresentValidator(t *testing.T) {
	validHeader := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"alg":"ES256","typ":"kb+jwt"}`),
	)
	validPayload := base64.RawURLEncoding.EncodeToString([]byte(`{"iat":1}`))
	setKBJWT := func(kbJWT string) func(*evidence.SDJWTPresentation) {
		return func(p *evidence.SDJWTPresentation) {
			p.KeyBindingJWT = kbJWT
		}
	}

	tests := []struct {
		name        string
		options     keyBindingFixtureOptions
		mutate      func(*evidence.SDJWTPresentation)
		wantStatus  Status
		wantMessage string
	}{
		{
			name:        "valid P-256 KB-JWT",
			wantStatus:  StatusPass,
			wantMessage: "all 1 presentations contain a structurally valid KB-JWT",
		},
		{
			name:        "missing KB-JWT",
			mutate:      setKBJWT(""),
			wantStatus:  StatusFail,
			wantMessage: "SD-JWT presentation does not contain a KB-JWT",
		},
		{
			name:        "not a compact JWT",
			mutate:      setKBJWT("not-a-jwt"),
			wantStatus:  StatusFail,
			wantMessage: "KB-JWT is not a compact JWT",
		},
		{
			name:        "header not base64url",
			mutate:      setKBJWT("*." + validPayload + ".c2ln"),
			wantStatus:  StatusFail,
			wantMessage: "KB-JWT header is not valid base64url",
		},
		{
			name: "header not JSON",
			mutate: setKBJWT(base64.RawURLEncoding.EncodeToString(
				[]byte("kb+jwt"),
			) + "." + validPayload + ".c2ln"),
			wantStatus:  StatusFail,
			wantMessage: "KB-JWT header is not valid JSON",
		},
		{
			name:        "wrong typ header",
			options:     keyBindingFixtureOptions{typ: "JWT"},
			wantStatus:  StatusFail,
			wantMessage: `KB-JWT protected header "typ" must equal "kb+jwt"`,
		},
		{
			name: "alg missing",
			mutate: setKBJWT(base64.RawURLEncoding.EncodeToString(
				[]byte(`{"typ":"kb+jwt"}`),
			) + "." + validPayload + ".c2ln"),
			wantStatus:  StatusFail,
			wantMessage: `KB-JWT protected header "alg" must be a non-empty string`,
		},
		{
			name:        "payload not base64url",
			mutate:      setKBJWT(validHeader + ".*.c2ln"),
			wantStatus:  StatusFail,
			wantMessage: "KB-JWT payload is not valid base64url",
		},
		{
			name: "payload not JSON object",
			mutate: setKBJWT(validHeader + "." + base64.RawURLEncoding.EncodeToString(
				[]byte(`["iat"]`),
			) + ".c2ln"),
			wantStatus:  StatusFail,
			wantMessage: "KB-JWT payload is not valid JSON",
		},
		{
			name: "missing iat claim",
			options: keyBindingFixtureOptions{mutateClaims: func(claims jwt.MapClaims) {
				delete(claims, "iat")
			}},
			wantStatus:  StatusFail,
			wantMessage: `KB-JWT claim "iat" must be a valid NumericDate`,
		},
		{
			name: "aud claim is not a string",
			options: keyBindingFixtureOptions{mutateClaims: func(claims jwt.MapClaims) {
				claims["aud"] = []string{"verifier.example"}
			}},
			wantStatus:  StatusFail,
			wantMessage: `KB-JWT claim "aud" must be a non-empty string`,
		},
		{
			name: "missing nonce claim",
			options: keyBindingFixtureOptions{mutateClaims: func(claims jwt.MapClaims) {
				delete(claims, "nonce")
			}},
			wantStatus:  StatusFail,
			wantMessage: `KB-JWT claim "nonce" must be a non-empty string`,
		},
		{
			name: "sd_hash is not base64url",
			options: keyBindingFixtureOptions{mutateClaims: func(claims jwt.MapClaims) {
				claims["sd_hash"] = "not+base64url"
			}},
			wantStatus:  StatusFail,
			wantMessage: `KB-JWT claim "sd_hash" must be an unpadded base64url SHA-256 digest`,
		},
		{
			name: "sd_hash is empty string",
			options: keyBindingFixtureOptions{mutateClaims: func(claims jwt.MapClaims) {
				claims["sd_hash"] = ""
			}},
			wantStatus:  StatusFail,
			wantMessage: `KB-JWT claim "sd_hash" must be a non-empty string`,
		},
		{
			name: "missing sd_hash claim",
			options: keyBindingFixtureOptions{mutateClaims: func(claims jwt.MapClaims) {
				delete(claims, "sd_hash")
			}},
			wantStatus:  StatusFail,
			wantMessage: `KB-JWT claim "sd_hash" must be a non-empty string`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presentation := newKeyBindingPresentation(t, tt.options)
			if tt.mutate != nil {
				tt.mutate(presentation)
			}

			result := SDJWTKBJWTPresentValidator{}.Validate(
				context.Background(),
				Input{Value: presentation},
			)

			require.Equal(t, tt.wantStatus, result.Status, result.Message)
			require.Contains(t, result.Message, tt.wantMessage)
		})
	}
}

func TestSDJWTKBJWTPresentValidatorChecksEveryPresentation(t *testing.T) {
	first := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	second := newKeyBindingPresentation(t, keyBindingFixtureOptions{})

	result := SDJWTKBJWTPresentValidator{}.Validate(context.Background(), Input{
		Value: []*evidence.SDJWTPresentation{first, second},
	})

	require.Equal(t, StatusPass, result.Status, result.Message)
	require.Equal(t, 2, result.Details["presentation_count"])
}

func TestSDJWTKBJWTPresentValidatorRejectsInvalidSecondPresentation(t *testing.T) {
	first := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	second := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	second.KeyBindingJWT = ""

	result := SDJWTKBJWTPresentValidator{}.Validate(context.Background(), Input{
		Value: []*evidence.SDJWTPresentation{first, second},
	})

	require.Equal(t, StatusFail, result.Status)
	require.Contains(t, result.Message, "presentation[1]")
}

func TestSDJWTClaimValidatorOutcomes(t *testing.T) {
	claims := map[string]any{
		"vct":            "urn:eudi:pid:1",
		"email":          "person@example.test",
		"given_name":     "Ada",
		"age_over_18":    true,
		"nationalities":  []any{"IT"},
		"place_of_birth": map[string]any{"locality": "Roma"},
		"age_in_years":   float64(36),
		"birth_year":     nil,
	}
	presentation := &evidence.SDJWTPresentation{Claims: claims}

	tests := []struct {
		name      string
		validator Validator
		value     any
		params    map[string]any
		status    Status
		message   string
	}{
		{
			name:      "present resolves nested claim path",
			validator: SDJWTClaimPresentValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "place_of_birth.locality"},
			status:    StatusPass,
		},
		{
			name:      "present reports missing claim",
			validator: SDJWTClaimPresentValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "family_name"},
			status:    StatusFail,
			message:   `claim "family_name" is missing`,
		},
		{
			name:      "present requires claim param",
			validator: SDJWTClaimPresentValidator{},
			value:     presentation,
			params:    map[string]any{},
			status:    StatusError,
			message:   claimParamRequired,
		},
		{
			name:      "present reads disclosed claim from compact presentation",
			validator: SDJWTClaimPresentValidator{},
			value:     testSDJWTPresentation(map[string]any{"given_name": "Ada"}),
			params:    map[string]any{"claim": "given_name"},
			status:    StatusPass,
		},
		{
			name:      "present reads claim from vp_token object",
			validator: SDJWTClaimPresentValidator{},
			value: map[string]any{
				"pid": []any{testSDJWTPresentation(map[string]any{"given_name": "Ada"})},
			},
			params: map[string]any{"claim": "given_name"},
			status: StatusPass,
		},
		{
			name:      "present rejects undisclosed claim in compact presentation",
			validator: SDJWTClaimPresentValidator{},
			value:     testSDJWTPresentation(map[string]any{"given_name": "Ada"}),
			params:    map[string]any{"claim": "family_name"},
			status:    StatusFail,
			message:   `claim "family_name" is missing`,
		},
		{
			name:      "presence requires present param",
			validator: SDJWTClaimPresenceValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "given_name"},
			status:    StatusError,
			message:   "present param is required",
		},
		{
			name:      "presence requires claim param",
			validator: SDJWTClaimPresenceValidator{},
			value:     presentation,
			params:    map[string]any{"present": false},
			status:    StatusError,
			message:   claimParamRequired,
		},
		{
			name:      "type string",
			validator: SDJWTClaimTypeValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "given_name", "type": "string"},
			status:    StatusPass,
		},
		{
			name:      "type number",
			validator: SDJWTClaimTypeValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "age_in_years", "type": "number"},
			status:    StatusPass,
		},
		{
			name:      "type boolean",
			validator: SDJWTClaimTypeValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "age_over_18", "type": "boolean"},
			status:    StatusPass,
		},
		{
			name:      "type array",
			validator: SDJWTClaimTypeValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "nationalities", "type": "array"},
			status:    StatusPass,
		},
		{
			name:      "type object",
			validator: SDJWTClaimTypeValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "place_of_birth", "type": "object"},
			status:    StatusPass,
		},
		{
			name:      "type null",
			validator: SDJWTClaimTypeValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "birth_year", "type": "null"},
			status:    StatusPass,
		},
		{
			name:      "type mismatch names actual type",
			validator: SDJWTClaimTypeValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "nationalities", "type": "string"},
			status:    StatusFail,
			message:   `claim "nationalities" is array, expected string`,
		},
		{
			name:      "type names Go type for non-JSON values",
			validator: SDJWTClaimTypeValidator{},
			value:     map[string]any{"portrait": []byte{0xff}},
			params:    map[string]any{"claim": "portrait", "type": "string"},
			status:    StatusFail,
			message:   `claim "portrait" is []uint8, expected string`,
		},
		{
			name:      "type reports missing claim",
			validator: SDJWTClaimTypeValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "family_name", "type": "string"},
			status:    StatusFail,
			message:   `claim "family_name" is missing`,
		},
		{
			name:      "prefix rejects other prefix",
			validator: SDJWTClaimStringPrefixValidator{},
			value:     map[string]any{"vct": "urn:eu.europa.ec.eudi:pid:1"},
			params:    map[string]any{"claim": "vct", "prefix": "urn:eudi:pid:"},
			status:    StatusFail,
			message:   `claim "vct" does not start with "urn:eudi:pid:"`,
		},
		{
			name:      "prefix rejects non-string claim",
			validator: SDJWTClaimStringPrefixValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "nationalities", "prefix": "IT"},
			status:    StatusFail,
			message:   `claim "nationalities" is []interface {}, expected string`,
		},
		{
			name:      "prefix reports missing claim",
			validator: SDJWTClaimStringPrefixValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "family_name", "prefix": "T"},
			status:    StatusFail,
			message:   `claim "family_name" is missing`,
		},
		{
			name:      "prefix requires claim",
			validator: SDJWTClaimStringPrefixValidator{},
			value:     presentation,
			params:    map[string]any{"prefix": "urn:"},
			status:    StatusError,
			message:   claimParamRequired,
		},
		{
			name:      "prefix requires prefix",
			validator: SDJWTClaimStringPrefixValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "vct"},
			status:    StatusError,
			message:   "prefix param is required",
		},
		{
			name:      "utf8 rejects invalid bytes",
			validator: SDJWTClaimUTF8StringValidator{},
			value:     map[string]any{"given_name": "Ad\xffa"},
			params:    map[string]any{"claim": "given_name"},
			status:    StatusFail,
			message:   `claim "given_name" is not valid UTF-8`,
		},
		{
			name:      "utf8 rejects non-string claim",
			validator: SDJWTClaimUTF8StringValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "age_over_18"},
			status:    StatusFail,
			message:   `claim "age_over_18" is bool, expected string`,
		},
		{
			name:      "utf8 reports missing claim",
			validator: SDJWTClaimUTF8StringValidator{},
			value:     presentation,
			params:    map[string]any{"claim": "family_name"},
			status:    StatusFail,
			message:   `claim "family_name" is missing`,
		},
		{
			name:      "utf8 requires claim",
			validator: SDJWTClaimUTF8StringValidator{},
			value:     presentation,
			params:    map[string]any{},
			status:    StatusError,
			message:   claimParamRequired,
		},
		{
			name:      "email rejects display-name form",
			validator: SDJWTClaimRFC5322EmailValidator{},
			value:     map[string]any{"email": "Ada <person@example.test>"},
			params:    map[string]any{"claim": "email"},
			status:    StatusFail,
			message:   `claim "email" must be a bare RFC 5322 addr-spec`,
		},
		{
			name:      "email rejects unparsable address",
			validator: SDJWTClaimRFC5322EmailValidator{},
			value:     map[string]any{"email": "person.example.test"},
			params:    map[string]any{"claim": "email"},
			status:    StatusFail,
			message:   `claim "email" is not a valid RFC 5322 address`,
		},
		{
			name:      "email rejects non-string claim",
			validator: SDJWTClaimRFC5322EmailValidator{},
			value:     map[string]any{"email": []any{"person@example.test"}},
			params:    map[string]any{"claim": "email"},
			status:    StatusFail,
			message:   "expected string",
		},
		{
			name:      "email reports missing claim",
			validator: SDJWTClaimRFC5322EmailValidator{},
			value:     map[string]any{},
			params:    map[string]any{"claim": "email"},
			status:    StatusFail,
			message:   `claim "email" is missing`,
		},
		{
			name:      "email requires claim",
			validator: SDJWTClaimRFC5322EmailValidator{},
			value:     presentation,
			params:    map[string]any{},
			status:    StatusError,
			message:   claimParamRequired,
		},
		{
			name:      "pid vct rejects legacy vct",
			validator: PIDSDJWTVCTValidator{},
			value:     map[string]any{"vct": "urn:eu.europa.ec.eudi:pid:1"},
			status:    StatusFail,
			message:   `claim "vct" does not start with "urn:eudi:pid:"`,
		},
		{
			name:      "pid vct rejects non-string vct",
			validator: PIDSDJWTVCTValidator{},
			value:     map[string]any{"vct": []any{"urn:eudi:pid:1"}},
			status:    StatusFail,
			message:   `claim "vct" is []interface {}, expected string`,
		},
		{
			name:      "pid vct reports missing vct",
			validator: PIDSDJWTVCTValidator{},
			value:     map[string]any{},
			status:    StatusFail,
			message:   `claim "vct" is missing`,
		},
		{
			name:      "mandatory claims lists every missing claim",
			validator: PIDSDJWTMandatoryClaimsValidator{},
			value:     presentation,
			params: map[string]any{
				"required_elements": []string{"given_name", "family_name", "birthdate"},
			},
			status:  StatusFail,
			message: "required mandatory PID SD-JWT claims are missing: [family_name birthdate]",
		},
		{
			name:      "mandatory claims fall back to decoded namespaces",
			validator: PIDSDJWTMandatoryClaimsValidator{},
			value: map[string]any{
				"namespaces": map[string]any{
					"ignored":   "not-an-object",
					pidMDocType: map[string]any{"family_name": "Trotter"},
				},
			},
			params: map[string]any{"required_elements": []string{"family_name"}},
			status: StatusPass,
		},
		{
			name:      "mandatory claims reject undecodable evidence",
			validator: PIDSDJWTMandatoryClaimsValidator{},
			value:     42,
			params:    map[string]any{"required_elements": []string{"family_name"}},
			status:    StatusFail,
			message:   "[family_name]",
		},
		{
			name:      "mandatory claims require required_elements",
			validator: PIDSDJWTMandatoryClaimsValidator{},
			value:     presentation,
			params:    map[string]any{},
			status:    StatusError,
			message:   "required_elements param is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: test.params,
			})
			require.Equal(t, test.status, got.Status, got.Message)
			require.Contains(t, got.Message, test.message)
		})
	}
}

func TestSDJWTClaimUTF8StringValidatorVectors(t *testing.T) {
	const (
		positive = "fixtures/fcaf/validators/sdjwt/email_utf8_positive.yaml"
		negative = "fixtures/fcaf/validators/sdjwt/email_utf8_negative.yaml"
	)
	dir := t.TempDir()
	writeVectors := func(name, content string) string {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		return path
	}
	textCase := writeVectors("text.yaml", "cases:\n  - id: plain\n    text: hello\n")
	emptyCase := writeVectors("empty.yaml", "cases:\n  - id: empty\n")
	badBase64 := writeVectors("bad.yaml", "cases:\n  - id: bad\n    bytes_base64: \"!!\"\n")
	malformed := writeVectors("malformed.yaml", "cases: [\n")

	tests := []struct {
		name    string
		vectors map[string]any
		status  Status
		message string
	}{
		{
			name:    "invalid bytes in positive vectors fail",
			vectors: map[string]any{"positive": []string{negative}},
			status:  StatusFail,
			message: `positive UTF-8 vector "invalid-two-byte-sequence" is invalid`,
		},
		{
			name:    "valid text in negative vectors fails",
			vectors: map[string]any{"positive": []string{positive}, "negative": []string{textCase}},
			status:  StatusFail,
			message: `negative UTF-8 vector "plain" is valid`,
		},
		{
			name:    "case without text or bytes is a definition error",
			vectors: map[string]any{"positive": []string{emptyCase}},
			status:  StatusError,
			message: `vector case "empty" must define text or bytes_base64`,
		},
		{
			name:    "malformed base64 negative case is a definition error",
			vectors: map[string]any{"negative": []string{badBase64}},
			status:  StatusError,
			message: `decode bytes_base64 for case "bad"`,
		},
		{
			name:    "malformed base64 positive case is a definition error",
			vectors: map[string]any{"positive": []string{badBase64}},
			status:  StatusError,
			message: `decode bytes_base64 for case "bad"`,
		},
		{
			name:    "unparsable vector file is a definition error",
			vectors: map[string]any{"negative": []string{malformed}},
			status:  StatusError,
			message: "parse vector file",
		},
		{
			name: "missing vector file is a definition error",
			vectors: map[string]any{
				"positive": []string{"fixtures/fcaf/validators/sdjwt/missing.yaml"},
			},
			status:  StatusError,
			message: "file not found",
		},
		{
			name:    "vectors must be an object",
			vectors: nil,
			status:  StatusError,
			message: "cannot unmarshal string into Go struct field .vectors",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := map[string]any{"claim": "email", "vectors": test.vectors}
			if test.vectors == nil {
				params["vectors"] = "fixtures"
			}
			got := SDJWTClaimUTF8StringValidator{}.Validate(context.Background(), Input{
				Value:  map[string]any{"email": "person@example.test"},
				Params: params,
			})
			require.Equal(t, test.status, got.Status, got.Message)
			require.Contains(t, got.Message, test.message)
		})
	}
}

func TestJOSEJWSValidatorsReportX5CFailures(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	certificate := testX509Certificate(t)
	sign := func(x5c any) string {
		t.Helper()
		request := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"nonce": "nonce-1"})
		if x5c != nil {
			request.Header["x5c"] = x5c
		}
		compact, err := request.SignedString(privateKey)
		require.NoError(t, err)
		return compact
	}

	// The invalid-signature validator passes only when the signature itself is
	// wrong; every other verification failure must not be mistaken for it.
	tests := []struct {
		name          string
		value         any
		signedMessage string
		invalidStatus Status
	}{
		{
			name:          "x5c missing",
			value:         sign(nil),
			signedMessage: "JWS header x5c certificate chain is missing",
			invalidStatus: StatusFail,
		},
		{
			name:          "x5c leaf is not a string",
			value:         sign([]any{42}),
			signedMessage: "JWS header x5c leaf certificate is not a string",
			invalidStatus: StatusFail,
		},
		{
			name:          "x5c leaf is not base64",
			value:         sign([]string{"not base64!"}),
			signedMessage: "decode JWS x5c leaf certificate",
			invalidStatus: StatusFail,
		},
		{
			name: "x5c leaf is not a certificate",
			value: sign(
				[]string{base64.StdEncoding.EncodeToString([]byte("not a certificate"))},
			),
			signedMessage: "parse JWS x5c leaf certificate",
			invalidStatus: StatusFail,
		},
		{
			name:          "x5c leaf belongs to another key",
			value:         sign([]string{certificate}),
			signedMessage: "signature is invalid",
			invalidStatus: StatusPass,
		},
		{
			name:          "input is not a string",
			value:         map[string]any{"request": "jws"},
			signedMessage: "input is map[string]interface {}, expected compact JWS",
			invalidStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signed := JOSEJWSSignedRequestValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value},
			)
			require.Equal(t, StatusFail, signed.Status)
			require.Contains(t, signed.Message, tt.signedMessage)

			invalid := JOSEJWSInvalidSignatureValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value},
			)
			require.Equal(t, tt.invalidStatus, invalid.Status, invalid.Message)
			if tt.invalidStatus == StatusFail {
				require.Contains(t, invalid.Message, tt.signedMessage)
			}
		})
	}
}

func TestJOSEJWSInvalidSignatureValidatorRejectsValidSignature(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(3600, 0),
	}
	der, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&privateKey.PublicKey,
		privateKey,
	)
	require.NoError(t, err)
	request := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"nonce": "nonce-1"})
	request.Header["x5c"] = []string{base64.StdEncoding.EncodeToString(der)}
	compact, err := request.SignedString(privateKey)
	require.NoError(t, err)

	got := JOSEJWSInvalidSignatureValidator{}.Validate(
		context.Background(),
		Input{Value: compact},
	)

	require.Equal(t, StatusFail, got.Status)
	require.Equal(t, "signed request JWS signature is valid", got.Message)
}
