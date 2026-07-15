// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

type keyBindingFixtureOptions struct {
	cnfKey       *ecdsa.PrivateKey
	signingKey   *ecdsa.PrivateKey
	jwkAlgorithm string
	typ          string
	mutateClaims func(jwt.MapClaims)
	tamper       bool
}

func TestSDJWTKeyBindingMatchesCNFValidator(t *testing.T) {
	tests := []struct {
		name       string
		options    keyBindingFixtureOptions
		mutate     func(*evidence.SDJWTPresentation)
		wantStatus Status
	}{
		{name: "valid P-256 key binding", wantStatus: StatusPass},
		{
			name: "KB-JWT signed by a different holder key",
			options: keyBindingFixtureOptions{
				signingKey: generateP256Key(t),
			},
			wantStatus: StatusFail,
		},
		{
			name:       "tampered KB-JWT signature",
			options:    keyBindingFixtureOptions{tamper: true},
			wantStatus: StatusFail,
		},
		{
			name:       "wrong KB-JWT typ",
			options:    keyBindingFixtureOptions{typ: "JWT"},
			wantStatus: StatusFail,
		},
		{
			name: "missing iat",
			options: keyBindingFixtureOptions{mutateClaims: func(claims jwt.MapClaims) {
				delete(claims, "iat")
			}},
			wantStatus: StatusFail,
		},
		{
			name: "aud is not a string",
			options: keyBindingFixtureOptions{mutateClaims: func(claims jwt.MapClaims) {
				claims["aud"] = []string{"verifier.example"}
			}},
			wantStatus: StatusFail,
		},
		{
			name: "missing nonce",
			options: keyBindingFixtureOptions{mutateClaims: func(claims jwt.MapClaims) {
				delete(claims, "nonce")
			}},
			wantStatus: StatusFail,
		},
		{
			name: "sd_hash is not base64url",
			options: keyBindingFixtureOptions{mutateClaims: func(claims jwt.MapClaims) {
				claims["sd_hash"] = "not+base64url"
			}},
			wantStatus: StatusFail,
		},
		{
			name: "sd_hash binds a different presentation",
			options: keyBindingFixtureOptions{mutateClaims: func(claims jwt.MapClaims) {
				digest := sha256.Sum256([]byte("different SD-JWT~"))
				claims["sd_hash"] = base64.RawURLEncoding.EncodeToString(digest[:])
			}},
			wantStatus: StatusFail,
		},
		{
			name:       "JWK alg conflicts with KB-JWT alg",
			options:    keyBindingFixtureOptions{jwkAlgorithm: "ES384"},
			wantStatus: StatusFail,
		},
		{
			name: "presentation has no KB-JWT",
			mutate: func(presentation *evidence.SDJWTPresentation) {
				presentation.KeyBindingJWT = ""
			},
			wantStatus: StatusFail,
		},
		{
			name: "presentation bytes changed after KB-JWT creation",
			mutate: func(presentation *evidence.SDJWTPresentation) {
				presentation.SDJWT += "changed~"
			},
			wantStatus: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presentation := newKeyBindingPresentation(t, tt.options)
			if tt.mutate != nil {
				tt.mutate(presentation)
			}

			result := SDJWTKeyBindingMatchesCNFValidator{}.Validate(
				context.Background(),
				Input{Value: presentation},
			)

			require.Equal(t, tt.wantStatus, result.Status, result.Message)
		})
	}
}

func TestSDJWTKeyBindingMatchesCNFValidatorBlocksUnresolvedConfirmationMethods(t *testing.T) {
	result := SDJWTKeyBindingMatchesCNFValidator{}.Validate(context.Background(), Input{
		Value: &evidence.SDJWTPresentation{
			IssuerPayload: map[string]any{"cnf": map[string]any{"kid": "holder-key"}},
		},
	})

	require.Equal(t, StatusBlocked, result.Status)
	require.Contains(t, result.Message, "resolver evidence")
}

func TestSDJWTDeviceBindingValidatorsCheckEveryPresentation(t *testing.T) {
	first := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	second := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	values := []*evidence.SDJWTPresentation{first, second}

	structural := SDJWTCNFConformsValidator{}.Validate(context.Background(), Input{Value: values})
	cryptographic := SDJWTKeyBindingMatchesCNFValidator{}.Validate(
		context.Background(),
		Input{Value: values},
	)

	require.Equal(t, StatusPass, structural.Status, structural.Message)
	require.Equal(t, 2, structural.Details["presentation_count"])
	require.Equal(t, StatusPass, cryptographic.Status, cryptographic.Message)
	require.Equal(t, 2, cryptographic.Details["presentation_count"])
}

func TestSDJWTDeviceBindingValidatorsRejectInvalidSecondPresentation(t *testing.T) {
	first := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	second := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	second.IssuerPayload["cnf"] = map[string]any{}

	structural := SDJWTCNFConformsValidator{}.Validate(context.Background(), Input{
		Value: []*evidence.SDJWTPresentation{first, second},
	})

	require.Equal(t, StatusFail, structural.Status)
	require.Contains(t, structural.Message, "presentation[1]")

	second = newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	second.KeyBindingJWT = tamperCompactJWT(second.KeyBindingJWT)
	cryptographic := SDJWTKeyBindingMatchesCNFValidator{}.Validate(context.Background(), Input{
		Value: []*evidence.SDJWTPresentation{first, second},
	})

	require.Equal(t, StatusFail, cryptographic.Status)
	require.Contains(t, cryptographic.Message, "presentation[1]")
}

func TestSDJWTKeyBindingCollectionBlocksUnresolvedSecondPresentation(t *testing.T) {
	first := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	second := &evidence.SDJWTPresentation{
		IssuerPayload: map[string]any{"cnf": map[string]any{"kid": "holder-key"}},
	}

	result := SDJWTKeyBindingMatchesCNFValidator{}.Validate(context.Background(), Input{
		Value: []*evidence.SDJWTPresentation{first, second},
	})

	require.Equal(t, StatusBlocked, result.Status)
	require.Contains(t, result.Message, "presentation[1]")
}

func TestSDJWTKeyBindingCollectionDoesNotLetBlockedPresentationHideFailure(t *testing.T) {
	blocked := &evidence.SDJWTPresentation{
		IssuerPayload: map[string]any{"cnf": map[string]any{"kid": "holder-key"}},
	}
	invalid := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	invalid.KeyBindingJWT = tamperCompactJWT(invalid.KeyBindingJWT)

	result := SDJWTKeyBindingMatchesCNFValidator{}.Validate(context.Background(), Input{
		Value: []*evidence.SDJWTPresentation{blocked, invalid},
	})

	require.Equal(t, StatusFail, result.Status)
	require.Contains(t, result.Message, "presentation[1]")
}

func TestSDJWTKeyBindingMatchesCNFValidatorSupportsDirectPublicJWKFamilies(t *testing.T) {
	t.Run("RSA", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		presentation := newKeyBindingPresentationWithSigner(t, map[string]any{
			"kty": "RSA",
			"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}, jwt.SigningMethodRS256, key)

		result := SDJWTKeyBindingMatchesCNFValidator{}.Validate(
			context.Background(),
			Input{Value: presentation},
		)

		require.Equal(t, StatusPass, result.Status, result.Message)
	})

	t.Run("Ed25519", func(t *testing.T) {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)
		presentation := newKeyBindingPresentationWithSigner(t, map[string]any{
			"kty": "OKP",
			"crv": "Ed25519",
			"x":   base64.RawURLEncoding.EncodeToString(publicKey),
		}, jwt.SigningMethodEdDSA, privateKey)

		result := SDJWTKeyBindingMatchesCNFValidator{}.Validate(
			context.Background(),
			Input{Value: presentation},
		)

		require.Equal(t, StatusPass, result.Status, result.Message)
	})
}

func TestSDJWTKeyBindingMatchesCNFValidatorRejectsNoneAlgorithm(t *testing.T) {
	presentation := newKeyBindingPresentation(t, keyBindingFixtureOptions{})
	digest := sha256.Sum256([]byte(presentation.SDJWT))
	presentation.KeyBindingJWT = unsignedJWT(t,
		map[string]any{"alg": "none", "typ": "kb+jwt"},
		map[string]any{
			"iat":     time.Now().Unix(),
			"aud":     "x509_hash:verifier.example",
			"nonce":   "fcaf-device-binding-007",
			"sd_hash": base64.RawURLEncoding.EncodeToString(digest[:]),
		},
	)

	result := SDJWTKeyBindingMatchesCNFValidator{}.Validate(
		context.Background(),
		Input{Value: presentation},
	)

	require.Equal(t, StatusFail, result.Status)
}

func TestParseSDJWTPresentationPreservesKeyBindingInput(t *testing.T) {
	presentation := newKeyBindingPresentation(t, keyBindingFixtureOptions{})

	require.NotEmpty(t, presentation.Raw)
	require.NotEmpty(t, presentation.SDJWT)
	require.True(
		t,
		len(presentation.SDJWT) > 0 && presentation.SDJWT[len(presentation.SDJWT)-1] == '~',
	)
	require.NotEmpty(t, presentation.KeyBindingJWT)
	require.Equal(t, presentation.Raw, presentation.SDJWT+presentation.KeyBindingJWT)
	require.Equal(t, "kb+jwt", presentation.KeyBinding["_protected_header"].(map[string]any)["typ"])
}

func newKeyBindingPresentation(
	t *testing.T,
	options keyBindingFixtureOptions,
) *evidence.SDJWTPresentation {
	t.Helper()
	cnfKey := options.cnfKey
	if cnfKey == nil {
		cnfKey = generateP256Key(t)
	}
	signingKey := options.signingKey
	if signingKey == nil {
		signingKey = cnfKey
	}

	jwk := p256PublicJWK(cnfKey)
	if options.jwkAlgorithm != "" {
		jwk["alg"] = options.jwkAlgorithm
	}
	presentation := newKeyBindingPresentationWithSigner(t, jwk, jwt.SigningMethodES256, signingKey)
	if options.typ != "" || options.mutateClaims != nil || options.tamper {
		presentation = customizedKeyBindingPresentation(t, jwk, signingKey, options)
	}
	return presentation
}

func newKeyBindingPresentationWithSigner(
	t *testing.T,
	jwk map[string]any,
	method jwt.SigningMethod,
	signingKey any,
) *evidence.SDJWTPresentation {
	t.Helper()
	return signedKeyBindingPresentation(t, jwk, method, signingKey, "kb+jwt", nil, false)
}

func customizedKeyBindingPresentation(
	t *testing.T,
	jwk map[string]any,
	signingKey *ecdsa.PrivateKey,
	options keyBindingFixtureOptions,
) *evidence.SDJWTPresentation {
	t.Helper()
	typ := options.typ
	if typ == "" {
		typ = "kb+jwt"
	}
	return signedKeyBindingPresentation(
		t,
		jwk,
		jwt.SigningMethodES256,
		signingKey,
		typ,
		options.mutateClaims,
		options.tamper,
	)
}

func signedKeyBindingPresentation(
	t *testing.T,
	jwk map[string]any,
	method jwt.SigningMethod,
	signingKey any,
	typ string,
	mutateClaims func(jwt.MapClaims),
	tamper bool,
) *evidence.SDJWTPresentation {
	t.Helper()
	issuerPayload := map[string]any{
		"_sd_alg": "sha-256",
		"cnf":     map[string]any{"jwk": jwk},
		"vct":     "urn:eudi:pid:1",
	}
	issuerJWT := unsignedJWT(t, map[string]any{"alg": "ES256", "typ": "dc+sd-jwt"}, issuerPayload)
	sdjwt := issuerJWT + "~"
	digest := sha256.Sum256([]byte(sdjwt))
	claims := jwt.MapClaims{
		"iat":     time.Now().Unix(),
		"aud":     "x509_hash:verifier.example",
		"nonce":   "fcaf-device-binding-007",
		"sd_hash": base64.RawURLEncoding.EncodeToString(digest[:]),
	}
	if mutateClaims != nil {
		mutateClaims(claims)
	}
	kbJWT := jwt.NewWithClaims(method, claims)
	kbJWT.Header["typ"] = typ
	signedKBJWT, err := kbJWT.SignedString(signingKey)
	require.NoError(t, err)
	if tamper {
		signedKBJWT = tamperCompactJWT(signedKBJWT)
	}

	presentation, err := evidence.ParseSDJWTPresentation(sdjwt + signedKBJWT)
	require.NoError(t, err)
	return presentation
}

func generateP256Key(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return key
}

func p256PublicJWK(key *ecdsa.PrivateKey) map[string]any {
	return map[string]any{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(key.X.FillBytes(make([]byte, 32))),
		"y":   base64.RawURLEncoding.EncodeToString(key.Y.FillBytes(make([]byte, 32))),
	}
}

func unsignedJWT(t *testing.T, header, payload map[string]any) string {
	t.Helper()
	encoded := func(value map[string]any) string {
		raw, err := json.Marshal(value)
		require.NoError(t, err)
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	return encoded(header) + "." + encoded(payload) + ".c2lnbmF0dXJl"
}

func tamperCompactJWT(token string) string {
	signatureStart := strings.LastIndex(token, ".") + 1
	last := token[signatureStart]
	replacement := byte('A')
	if last == replacement {
		replacement = 'B'
	}
	return token[:signatureStart] + string(replacement) + token[signatureStart+1:]
}
