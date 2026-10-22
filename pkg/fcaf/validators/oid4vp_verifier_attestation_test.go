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
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	verifierAttestationTestClientID    = "verifier_attestation:capture-wallet.credimi.io"
	verifierAttestationTestVerifier    = "capture-wallet.credimi.io"
	verifierAttestationTestIssuer      = "https://capture-wallet.credimi.io/openid4vp/verifier-attestation-issuer"
	verifierAttestationTestKeyID       = "credimi-fake-verifier-attestation-issuer-key"
	verifierAttestationTestResponseURI = "https://capture-wallet.credimi.io/openid4vp/sessions/session-1/response?session=6f1e"
	verifierAttestationTestOtherURI    = "https://capture-wallet.credimi.io/openid4vp/sessions/session-2/response?session=9a2b"
)

func TestOID4VPVerifierAttestationRequestValidator(t *testing.T) {
	keys := newVerifierAttestationTestKeys(t)

	tests := []struct {
		name           string
		params         map[string]any
		mutate         func(*verifierAttestationOptions)
		mutateEvidence func(map[string]any)
		build          func(*testing.T, *verifierAttestationTestKeys) any
		wantStatus     Status
		wantMessage    string
	}{
		{
			name:        "accepts the production shape",
			params:      defaultVerifierAttestationParams(),
			wantStatus:  StatusPass,
			wantMessage: "attestation in jwt header, sub matches client_id capture-wallet.credimi.io",
		},
		{
			name:   "accepts a sub that differs when other is expected",
			params: verifierAttestationParamsWith(map[string]any{"subject": "other"}),
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.sub = "other-verifier.example"
			},
			wantStatus:  StatusPass,
			wantMessage: "sub other-verifier.example differs from client_id capture-wallet.credimi.io",
		},
		{
			name:        "rejects a sub that matches when other is expected",
			params:      verifierAttestationParamsWith(map[string]any{"subject": "other"}),
			wantStatus:  StatusFail,
			wantMessage: `sub "capture-wallet.credimi.io" matches client_id`,
		},
		{
			name: "rejects a sub that differs when client_id is expected",
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.sub = "other-verifier.example"
			},
			wantStatus:  StatusFail,
			wantMessage: "does not match client_id",
		},
		{
			name: "rejects an issuer mismatch",
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.iss = "https://other.example/verifier-attestation-issuer"
			},
			wantStatus:  StatusFail,
			wantMessage: "does not match expected issuer",
		},
		{
			name:   "accepts a Request Object signed by an unrelated key when invalid is expected",
			params: verifierAttestationParamsWith(map[string]any{"request_signature": "invalid"}),
			mutate: func(options *verifierAttestationOptions) {
				options.requestSigner = keys.unrelated
			},
			wantStatus:  StatusPass,
			wantMessage: "Request Object signature does not verify against cnf.jwk",
		},
		{
			name: "rejects a Request Object signed by an unrelated key when valid is expected",
			mutate: func(options *verifierAttestationOptions) {
				options.requestSigner = keys.unrelated
			},
			wantStatus:  StatusFail,
			wantMessage: "Request Object signature does not verify against the attestation cnf.jwk",
		},
		{
			name: "accepts an attestation signed by an unrelated key when invalid is expected",
			params: verifierAttestationParamsWith(
				map[string]any{"attestation_signature": "invalid"},
			),
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.signer = keys.unrelated
			},
			wantStatus:  StatusPass,
			wantMessage: "attestation signature does not verify against issuer key",
		},
		{
			name: "rejects an attestation signed by an unrelated key when valid is expected",
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.signer = keys.unrelated
			},
			wantStatus:  StatusFail,
			wantMessage: "attestation signature does not verify against",
		},
		{
			name: "rejects a kid the issuer JWKS does not publish",
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.kid = "unpublished-kid"
			},
			wantStatus:  StatusFail,
			wantMessage: `kid "unpublished-kid" is not published by attestation_issuer_jwks`,
		},
		{
			name: "rejects a kid the issuer JWKS does not publish even when invalid is expected",
			params: verifierAttestationParamsWith(
				map[string]any{"attestation_signature": "invalid"},
			),
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.kid = "unpublished-kid"
			},
			wantStatus:  StatusFail,
			wantMessage: "is not published by attestation_issuer_jwks",
		},
		{
			name: "accepts an attestation without a kid when the JWKS publishes one key",
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.kid = nil
			},
			wantStatus:  StatusPass,
			wantMessage: "attestation signature verifies against issuer key credimi-fake-verifier-attestation-issuer-key",
		},
		{
			name: "rejects an attestation without a kid when the JWKS publishes several keys",
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.kid = nil
			},
			mutateEvidence: func(evidence map[string]any) {
				jwks, _ := evidence["attestation_issuer_jwks"].(map[string]any)
				publishedKeys, _ := jwks["keys"].([]any)
				jwks["keys"] = append(
					publishedKeys,
					verifierAttestationTestJWK(&keys.unrelated.PublicKey, "second-key"),
				)
			},
			wantStatus:  StatusFail,
			wantMessage: "has no kid and attestation_issuer_jwks publishes 2 keys",
		},
		{
			name:        "accepts a jwt header absent",
			params:      map[string]any{"jwt_header": "absent"},
			mutate:      func(options *verifierAttestationOptions) { options.attestation = nil },
			wantStatus:  StatusPass,
			wantMessage: "no jwt header member, client_id verifier_attestation:capture-wallet.credimi.io",
		},
		{
			name:        "rejects a missing jwt header when present is expected",
			mutate:      func(options *verifierAttestationOptions) { options.attestation = nil },
			wantStatus:  StatusFail,
			wantMessage: "jwt header member is missing",
		},
		{
			name:        "rejects a jwt header when absent is expected",
			params:      map[string]any{"jwt_header": "absent"},
			wantStatus:  StatusFail,
			wantMessage: "jwt header member carries a verifier attestation",
		},
		{
			name: "rejects redirect_uris when absent is expected",
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.redirectURIs = []any{verifierAttestationTestOtherURI}
			},
			wantStatus:  StatusFail,
			wantMessage: "carries redirect_uris",
		},
		{
			name: "accepts redirect_uris containing the response uri",
			params: verifierAttestationParamsWith(
				map[string]any{"redirect_uris": "matches_response_uri"},
			),
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.redirectURIs = []any{
					verifierAttestationTestOtherURI,
					verifierAttestationTestResponseURI,
				}
			},
			wantStatus:  StatusPass,
			wantMessage: "redirect_uris contains response_uri",
		},
		{
			name: "rejects redirect_uris without the response uri when matches is expected",
			params: verifierAttestationParamsWith(
				map[string]any{"redirect_uris": "matches_response_uri"},
			),
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.redirectURIs = []any{verifierAttestationTestOtherURI}
			},
			wantStatus:  StatusFail,
			wantMessage: "does not contain response_uri",
		},
		{
			name: "rejects a missing response_uri when matches is expected",
			params: verifierAttestationParamsWith(
				map[string]any{"redirect_uris": "matches_response_uri"},
			),
			mutate: func(options *verifierAttestationOptions) {
				options.responseURI = ""
				options.attestation.redirectURIs = []any{verifierAttestationTestOtherURI}
			},
			wantStatus:  StatusFail,
			wantMessage: "response_uri is missing",
		},
		{
			name: "rejects redirect_uris that is not an array of strings",
			params: verifierAttestationParamsWith(
				map[string]any{"redirect_uris": "matches_response_uri"},
			),
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.redirectURIs = []any{verifierAttestationTestOtherURI, 42}
			},
			wantStatus:  StatusFail,
			wantMessage: "not an array of strings",
		},
		{
			name: "accepts redirect_uris excluding the response uri",
			params: verifierAttestationParamsWith(
				map[string]any{"redirect_uris": "excludes_response_uri"},
			),
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.redirectURIs = []any{verifierAttestationTestOtherURI}
			},
			wantStatus:  StatusPass,
			wantMessage: "redirect_uris excludes response_uri",
		},
		{
			name: "rejects redirect_uris containing the response uri when excludes is expected",
			params: verifierAttestationParamsWith(
				map[string]any{"redirect_uris": "excludes_response_uri"},
			),
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.redirectURIs = []any{verifierAttestationTestResponseURI}
			},
			wantStatus:  StatusFail,
			wantMessage: "contains response_uri",
		},
		{
			name: "rejects empty redirect_uris when excludes is expected",
			params: verifierAttestationParamsWith(
				map[string]any{"redirect_uris": "excludes_response_uri"},
			),
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.redirectURIs = []any{}
			},
			wantStatus:  StatusFail,
			wantMessage: "not a non-empty array of strings",
		},
		{
			name: "rejects a wrong attestation typ",
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.typ = "JWT"
			},
			wantStatus:  StatusFail,
			wantMessage: `typ is "JWT", expected verifier-attestation+jwt`,
		},
		{
			name: "rejects a client_id without the verifier_attestation prefix",
			mutate: func(options *verifierAttestationOptions) {
				options.clientID = verifierAttestationTestVerifier
			},
			wantStatus:  StatusFail,
			wantMessage: "is not verifier_attestation:<verifier identifier>",
		},
		{
			name: "rejects an empty verifier identifier after the prefix",
			mutate: func(options *verifierAttestationOptions) {
				options.clientID = "verifier_attestation:"
			},
			wantStatus:  StatusFail,
			wantMessage: "is not verifier_attestation:<verifier identifier>",
		},
		{
			name: "rejects a cnf.jwk that is not a P-256 EC key",
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.cnfJWK = map[string]any{
					"kty": "EC",
					"crv": "P-256",
					"x":   base64.RawURLEncoding.EncodeToString(make([]byte, 16)),
					"y":   verifierAttestationTestJWK(&keys.request.PublicKey, "")["y"],
				}
			},
			wantStatus:  StatusFail,
			wantMessage: "cnf.jwk: JWK x coordinate is 16 bytes, expected 32",
		},
		{
			name: "rejects a kid that is not a string",
			mutate: func(options *verifierAttestationOptions) {
				options.attestation.kid = 42
			},
			wantStatus:  StatusFail,
			wantMessage: "kid is not a string",
		},
		{
			name: "rejects missing attestation_issuer_jwks",
			mutateEvidence: func(evidence map[string]any) {
				delete(evidence, "attestation_issuer_jwks")
			},
			wantStatus:  StatusFail,
			wantMessage: "attestation_issuer_jwks is missing",
		},
		{
			name:       "accepts a capture whose claims have expired",
			mutate:     func(options *verifierAttestationOptions) { options.expired = true },
			wantStatus: StatusPass,
		},
		{
			name: "rejects a non-object input",
			build: func(*testing.T, *verifierAttestationTestKeys) any {
				return []any{"not an object"}
			},
			wantStatus:  StatusFail,
			wantMessage: "evidence is not an object",
		},
		{
			name: "rejects a request_object that is not a compact JWS",
			build: func(*testing.T, *verifierAttestationTestKeys) any {
				return map[string]any{"request_object": "not-a-jws"}
			},
			wantStatus:  StatusFail,
			wantMessage: "parse verifier_attestation Request Object",
		},
		{
			name: "rejects a request_object that is not a string",
			build: func(*testing.T, *verifierAttestationTestKeys) any {
				return map[string]any{"request_object": 42}
			},
			wantStatus:  StatusFail,
			wantMessage: "Request Object is missing",
		},
		{
			name:        "rejects a missing jwt_header param",
			params:      map[string]any{},
			wantStatus:  StatusError,
			wantMessage: "jwt_header param must be present or absent",
		},
		{
			name:        "rejects an unknown jwt_header param value",
			params:      map[string]any{"jwt_header": "sometimes"},
			wantStatus:  StatusError,
			wantMessage: "jwt_header param must be present or absent",
		},
		{
			name:        "rejects a present-mode param that is missing",
			params:      verifierAttestationParamsWithout("subject"),
			wantStatus:  StatusError,
			wantMessage: "subject param must be client_id or other",
		},
		{
			name:        "rejects an unknown subject param value",
			params:      verifierAttestationParamsWith(map[string]any{"subject": "wallet"}),
			wantStatus:  StatusError,
			wantMessage: "subject param must be client_id or other",
		},
		{
			name:        "rejects a missing issuer param",
			params:      verifierAttestationParamsWithout("issuer"),
			wantStatus:  StatusError,
			wantMessage: "issuer param is required",
		},
		{
			name: "rejects present-mode params alongside jwt_header absent",
			params: map[string]any{
				"jwt_header": "absent",
				"subject":    "client_id",
			},
			wantStatus:  StatusError,
			wantMessage: "jwt_header absent takes no other params",
		},
	}

	validator := OID4VPVerifierAttestationRequestValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var value any
			if test.build != nil {
				value = test.build(t, keys)
			} else {
				options := defaultVerifierAttestationOptions(keys)
				if test.mutate != nil {
					test.mutate(&options)
				}
				evidence := buildVerifierAttestationEvidence(t, keys, options)
				if test.mutateEvidence != nil {
					test.mutateEvidence(evidence)
				}
				value = evidence
			}

			params := test.params
			if params == nil {
				params = defaultVerifierAttestationParams()
			}
			result := validator.Validate(context.Background(), Input{
				Value:  value,
				Params: params,
			})

			require.Equal(t, test.wantStatus, result.Status, result.Message)
			if test.wantMessage != "" {
				require.Contains(t, result.Message, test.wantMessage)
			}
		})
	}
}

type verifierAttestationTestKeys struct {
	request   *ecdsa.PrivateKey
	issuer    *ecdsa.PrivateKey
	unrelated *ecdsa.PrivateKey
}

func newVerifierAttestationTestKeys(t *testing.T) *verifierAttestationTestKeys {
	t.Helper()
	generate := func() *ecdsa.PrivateKey {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		return key
	}
	return &verifierAttestationTestKeys{
		request:   generate(),
		issuer:    generate(),
		unrelated: generate(),
	}
}

type verifierAttestationOptions struct {
	clientID      string
	responseURI   string
	requestSigner *ecdsa.PrivateKey
	expired       bool
	attestation   *verifierAttestationTokenOptions
}

type verifierAttestationTokenOptions struct {
	sub          string
	iss          string
	typ          string
	kid          any
	cnfKey       *ecdsa.PrivateKey
	cnfJWK       map[string]any
	redirectURIs any
	signer       *ecdsa.PrivateKey
	expired      bool
}

func defaultVerifierAttestationOptions(
	keys *verifierAttestationTestKeys,
) verifierAttestationOptions {
	return verifierAttestationOptions{
		clientID:      verifierAttestationTestClientID,
		responseURI:   verifierAttestationTestResponseURI,
		requestSigner: keys.request,
		attestation: &verifierAttestationTokenOptions{
			sub:    verifierAttestationTestVerifier,
			iss:    verifierAttestationTestIssuer,
			typ:    "verifier-attestation+jwt",
			kid:    verifierAttestationTestKeyID,
			cnfKey: keys.request,
			signer: keys.issuer,
		},
	}
}

func defaultVerifierAttestationParams() map[string]any {
	return map[string]any{
		"jwt_header":            "present",
		"subject":               "client_id",
		"issuer":                verifierAttestationTestIssuer,
		"request_signature":     "valid",
		"attestation_signature": "valid",
		"redirect_uris":         "absent",
	}
}

func verifierAttestationParamsWith(overrides map[string]any) map[string]any {
	params := defaultVerifierAttestationParams()
	for name, value := range overrides {
		params[name] = value
	}
	return params
}

func verifierAttestationParamsWithout(name string) map[string]any {
	params := defaultVerifierAttestationParams()
	delete(params, name)
	return params
}

func buildVerifierAttestationEvidence(
	t *testing.T,
	keys *verifierAttestationTestKeys,
	options verifierAttestationOptions,
) map[string]any {
	t.Helper()

	requestClaims := jwt.MapClaims{"client_id": options.clientID}
	if options.responseURI != "" {
		requestClaims["response_uri"] = options.responseURI
	}
	addVerifierAttestationTimes(requestClaims, options.expired)
	requestHeader := map[string]any{"typ": "oauth-authz-req+jwt"}
	if options.attestation != nil {
		requestHeader["jwt"] = buildVerifierAttestationToken(t, options.attestation)
	}
	request := signVerifierAttestationTestToken(
		t,
		options.requestSigner,
		requestHeader,
		requestClaims,
	)

	return map[string]any{
		"request_object": request,
		"attestation_issuer_jwks": map[string]any{
			"keys": []any{
				verifierAttestationTestJWK(
					&keys.issuer.PublicKey,
					verifierAttestationTestKeyID,
				),
			},
		},
	}
}

func buildVerifierAttestationToken(
	t *testing.T,
	options *verifierAttestationTokenOptions,
) string {
	t.Helper()

	cnfJWK := options.cnfJWK
	if cnfJWK == nil {
		cnfJWK = verifierAttestationTestJWK(&options.cnfKey.PublicKey, "")
	}
	claims := jwt.MapClaims{
		"iss": options.iss,
		"sub": options.sub,
		"cnf": map[string]any{"jwk": cnfJWK},
	}
	addVerifierAttestationTimes(claims, options.expired)
	if options.redirectURIs != nil {
		claims["redirect_uris"] = options.redirectURIs
	}
	header := map[string]any{"typ": options.typ}
	if options.kid != nil {
		header["kid"] = options.kid
	}
	return signVerifierAttestationTestToken(t, options.signer, header, claims)
}

func addVerifierAttestationTimes(claims jwt.MapClaims, expired bool) {
	now := time.Now()
	if expired {
		claims["iat"] = now.Add(-2 * time.Hour).Unix()
		claims["exp"] = now.Add(-time.Hour).Unix()
		return
	}
	claims["iat"] = now.Add(-time.Minute).Unix()
	claims["exp"] = now.Add(time.Hour).Unix()
}

func signVerifierAttestationTestToken(
	t *testing.T,
	key *ecdsa.PrivateKey,
	header map[string]any,
	claims jwt.MapClaims,
) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	for name, value := range header {
		token.Header[name] = value
	}
	signed, err := token.SignedString(key)
	require.NoError(t, err)
	return signed
}

func verifierAttestationTestJWK(key *ecdsa.PublicKey, kid string) map[string]any {
	jwk := map[string]any{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(key.X.FillBytes(make([]byte, 32))),
		"y":   base64.RawURLEncoding.EncodeToString(key.Y.FillBytes(make([]byte, 32))),
		"use": "sig",
		"alg": "ES256",
	}
	if kid != "" {
		jwk["kid"] = kid
	}
	return jwk
}
