// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testIssuerBase  = "https://capture.test/issuers/eu-pid-device-bound"
	testSessionID   = "a0eb3914-e4bb-477c-b5e2-3f47cfcbd5ea"
	testConfigID    = "eu-pid-device-bound"
	testSDJWTConfig = "urn:eu.europa.ec.eudi:pid:1.sd-jwt.key-attestation-required"
	testMdocConfig  = "urn:eu.europa.ec.eudi:pid:1.mdoc.key-attestation-required"
	testPARState    = "8xbgacOyB9hKdgeuUyoKBte6_1WAgcWC_u3MvhQT54c"
)

func redacted(present bool) map[string]any {
	return map[string]any{"redacted": true, "present": present, "length": 43}
}

func testCapturedRequest(
	at, method, path string,
	query map[string]any,
	body any,
	headers map[string]any,
) map[string]any {
	allHeaders := map[string]any{
		"host":              "capture.test",
		"x-forwarded-proto": "https",
	}
	for key, value := range headers {
		allHeaders[key] = value
	}
	if query == nil {
		query = map[string]any{}
	}
	return map[string]any{
		"at":                      at,
		"method":                  method,
		"path":                    path,
		"session_id":              testSessionID,
		"issuer_configuration_id": testConfigID,
		"query":                   query,
		"body":                    body,
		"headers":                 allHeaders,
	}
}

func testPARBody() map[string]any {
	return map[string]any{
		"client_id":             "eudiw-abca",
		"response_type":         "code",
		"redirect_uri":          "eu.europa.ec.euidi://authorization",
		"scope":                 testSDJWTConfig,
		"state":                 testPARState,
		"code_challenge":        "wnETeejFsJbAERJLMKe-vJFmttAX9FChhbITAocbq4c",
		"code_challenge_method": "S256",
		"issuer_state":          "1-tJ7Wco5Wm-SF40rmhHkUhl8B8XqUL5aqXnfuI1zS8",
	}
}

func testPAR() map[string]any {
	return testCapturedRequest(
		"2026-10-22T07:37:47.148Z", "POST", "/issuers/eu-pid-device-bound/par", nil, testPARBody(),
		map[string]any{"content-type": "application/x-www-form-urlencoded; charset=UTF-8"},
	)
}

func testCredentialRequestBody() map[string]any {
	return map[string]any{
		"credential_configuration_id": testSDJWTConfig,
		"proofs":                      map[string]any{"attestation": redacted(true)},
	}
}

// testAuthorizeRequest is the Wallet's browser Authorization Request, which
// Capture records in the global ledger only.
func testAuthorizeRequest(at string, state string) map[string]any {
	request := testCapturedRequest(
		at,
		"GET",
		"/issuers/eu-pid-device-bound/authorize",
		map[string]any{
			"client_id":   "eudiw-abca",
			"state":       state,
			"request_uri": redacted(true),
		},
		map[string]any{},
		nil,
	)
	request["session_id"] = nil
	return request
}

func testExchange(sessionRequests []any, ledger []any) map[string]any {
	return map[string]any{
		"capture_session": map[string]any{
			"session_id":              testSessionID,
			"issuer_configuration_id": testConfigID,
			"raw": map[string]any{
				"oid4vci_requests":   sessionRequests,
				"credential_request": testCredentialRequestBody(),
			},
		},
		"request_ledger": ledger,
		"issuer_metadata": map[string]any{
			"credential_endpoint": testIssuerBase + "/credential",
			"credential_configurations_supported": map[string]any{
				testSDJWTConfig: map[string]any{"scope": testSDJWTConfig},
				testMdocConfig:  map[string]any{"scope": testMdocConfig},
			},
		},
		"authorization_server_metadata": map[string]any{
			"pushed_authorization_request_endpoint": testIssuerBase + "/par",
			"authorization_endpoint":                testIssuerBase + "/authorize",
		},
	}
}

func testSessionRequests() []any {
	return []any{
		testPAR(),
		testCapturedRequest(
			"2026-10-22T07:38:24.819Z",
			"GET",
			"/issuers/eu-pid-device-bound/redirect",
			map[string]any{"code": redacted(true), "state": "upstream"},
			map[string]any{},
			nil,
		),
		testCapturedRequest(
			"2026-10-22T07:38:29.297Z",
			"POST",
			"/issuers/eu-pid-device-bound/credential",
			nil,
			testCredentialRequestBody(),
			map[string]any{"content-type": "application/jwt"},
		),
	}
}

// testLedger is the global ledger: the session's PAR, which Capture records
// there too, then the given entries.
func testLedger(entries ...any) []any {
	return append([]any{testPAR()}, entries...)
}

// testAuthorizeLedger adds the Wallet's Authorization Request and Capture's
// own redirect to its chained OAuth server after the PAR.
func testAuthorizeLedger(extra ...any) []any {
	return testLedger(append([]any{
		testAuthorizeRequest("2026-10-22T07:38:24.600Z", testPARState),
		testCapturedRequest(
			"2026-10-22T07:38:24.730Z",
			"GET",
			"/authorization-servers/eu-pid-device-bound/authorize",
			map[string]any{"client_id": "credimi-capture-wallet"},
			map[string]any{},
			nil,
		),
	}, extra...)...)
}

func validateWalletRequest(value any, params map[string]any) Result {
	return OID4VCIWalletRequestValidator{}.Validate(
		context.Background(),
		Input{Value: value, Params: params},
	)
}

func TestOID4VCIWalletRequestPushedAuthorizationRequest(t *testing.T) {
	parParams := map[string]any{
		"path_suffix": "/par",
		"methods":     []any{"POST"},
		"endpoint": map[string]any{
			"metadata": "authorization_server_metadata",
			"field":    "pushed_authorization_request_endpoint",
		},
		"require_tls":  true,
		"forbid_query": true,
		"media_type":   "application/x-www-form-urlencoded",
		"parameters": map[string]any{
			"in": "body",
			"required": []any{
				"scope",
				"response_type",
				"client_id",
				"redirect_uri",
				"code_challenge",
				"code_challenge_method",
			},
		},
	}
	withPAR := func(mutate func(map[string]any)) map[string]any {
		par := testPAR()
		mutate(par)
		return testExchange([]any{par}, nil)
	}
	for _, tt := range []struct {
		name   string
		value  any
		status Status
	}{
		{"conforming PAR", testExchange(testSessionRequests(), nil), StatusPass},
		{"GET method", withPAR(func(par map[string]any) { par["method"] = "GET" }), StatusFail},
		{"query component", withPAR(func(par map[string]any) { par["query"] = map[string]any{"x": "1"} }), StatusFail},
		{"other host", withPAR(func(par map[string]any) {
			par["headers"].(map[string]any)["host"] = "attacker.test"
		}), StatusFail},
		{"explicit default port", withPAR(func(par map[string]any) {
			par["headers"].(map[string]any)["host"] = "capture.test:443"
		}), StatusPass},
		{"other port", withPAR(func(par map[string]any) {
			par["headers"].(map[string]any)["host"] = "capture.test:8443"
		}), StatusFail},
		{"other path", withPAR(func(par map[string]any) { par["path"] = "/issuers/other/par" }), StatusFail},
		{"plain http", withPAR(func(par map[string]any) {
			par["headers"].(map[string]any)["x-forwarded-proto"] = "http"
		}), StatusFail},
		{"unrecorded scheme", withPAR(func(par map[string]any) {
			delete(par["headers"].(map[string]any), "x-forwarded-proto")
		}), StatusFail},
		{"JSON body", withPAR(func(par map[string]any) {
			par["headers"].(map[string]any)["content-type"] = "application/json"
		}), StatusFail},
		{"missing code_challenge", withPAR(func(par map[string]any) {
			delete(par["body"].(map[string]any), "code_challenge")
		}), StatusFail},
		{"null scope", withPAR(func(par map[string]any) { par["body"].(map[string]any)["scope"] = nil }), StatusFail},
		{"no PAR recorded", testExchange(nil, nil), StatusFail},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := validateWalletRequest(tt.value, parParams)
			require.Equal(t, tt.status, result.Status, result.Message)
		})
	}
}

func TestOID4VCIWalletRequestOptionalParameters(t *testing.T) {
	params := map[string]any{
		"path_suffix": "/par",
		"parameters": map[string]any{
			"in":       "body",
			"required": []any{"scope"},
			"absent":   []any{"request"},
		},
	}
	require.Equal(
		t,
		StatusPass,
		validateWalletRequest(testExchange(testSessionRequests(), nil), params).Status,
	)

	par := testPAR()
	par["body"].(map[string]any)["request"] = "eyJ..."
	require.Equal(
		t,
		StatusFail,
		validateWalletRequest(testExchange([]any{par}, nil), params).Status,
	)
}

func TestOID4VCIWalletRequestAuthorizationRequestFromLedger(t *testing.T) {
	params := map[string]any{
		"path_suffix":     "/authorize",
		"ledger_fallback": true,
		"methods":         []any{"GET", "POST"},
		"endpoint": map[string]any{
			"metadata": "authorization_server_metadata",
			"field":    "authorization_endpoint",
		},
		"parameters": map[string]any{
			"in":       "query_or_body",
			"required": []any{"request_uri", "client_id"},
		},
	}
	withoutState := testAuthorizeRequest("2026-10-22T07:38:20.000Z", "")
	delete(withoutState["query"].(map[string]any), "state")
	otherSession := testAuthorizeRequest("2026-10-22T07:38:21.000Z", "other")
	otherSession["session_id"] = "another-session"
	otherIssuer := testAuthorizeRequest("2026-10-22T07:38:21.000Z", "other")
	otherIssuer["issuer_configuration_id"] = "eu-pid-jwt-proof-only"
	afterCallback := testAuthorizeRequest("2026-10-22T07:40:00.000Z", "late")
	beforePAR := testAuthorizeRequest("2026-10-22T07:30:00.000Z", "early")
	redactedAbsent := testAuthorizeRequest("2026-10-22T07:38:24.600Z", testPARState)
	redactedAbsent["query"].(map[string]any)["request_uri"] = redacted(false)

	for _, tt := range []struct {
		name   string
		ledger []any
		status Status
	}{
		{"single request in the PAR window", testAuthorizeLedger(), StatusPass},
		{"requests outside the window, issuer or session are ignored",
			testAuthorizeLedger(otherSession, otherIssuer, afterCallback, beforePAR), StatusPass},
		{"PAR state singles out the Wallet request", testAuthorizeLedger(withoutState), StatusPass},
		{"two requests without the PAR state are ambiguous", testLedger(
			testAuthorizeRequest("2026-10-22T07:38:20.000Z", "a"),
			testAuthorizeRequest("2026-10-22T07:38:21.000Z", "b"),
		), StatusInconclusive},
		{"no request in the window", testLedger(afterCallback), StatusFail},
		{"request_uri recorded as absent", testLedger(redactedAbsent), StatusFail},
		{"ledger evicted past the PAR", []any{
			testAuthorizeRequest("2026-10-22T07:38:24.600Z", testPARState),
		}, StatusInconclusive},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := validateWalletRequest(testExchange(testSessionRequests(), tt.ledger), params)
			require.Equal(t, tt.status, result.Status, result.Message)
		})
	}

	noLedger := testExchange(testSessionRequests(), nil)
	delete(noLedger, "request_ledger")
	result := validateWalletRequest(noLedger, params)
	require.Equal(
		t,
		StatusInconclusive,
		result.Status,
		"an exchange without the ledger cannot attribute it",
	)
}

func TestOID4VCIWalletRequestCredentialRequest(t *testing.T) {
	params := map[string]any{
		"path_suffix": "/credential",
		"methods":     []any{"POST"},
		"endpoint": map[string]any{
			"metadata": "issuer_metadata",
			"field":    "credential_endpoint",
		},
	}
	require.Equal(
		t,
		StatusPass,
		validateWalletRequest(testExchange(testSessionRequests(), nil), params).Status,
	)

	exchange := testExchange(testSessionRequests(), nil)
	exchange["issuer_metadata"].(map[string]any)["credential_endpoint"] = testIssuerBase + "/credential?tenant=a"
	require.Equal(t, StatusFail, validateWalletRequest(exchange, params).Status,
		"a query component of the endpoint must be requested")

	delete(exchange["issuer_metadata"].(map[string]any), "credential_endpoint")
	require.Equal(t, StatusInconclusive, validateWalletRequest(exchange, params).Status)
}

func TestOID4VCIPARScopeSupportedValidator(t *testing.T) {
	withScope := func(scope any) map[string]any {
		par := testPAR()
		par["body"].(map[string]any)["scope"] = scope
		return testExchange([]any{par}, nil)
	}
	for _, tt := range []struct {
		name   string
		value  any
		status Status
	}{
		{"advertised scope", withScope(testSDJWTConfig), StatusPass},
		{"two advertised scopes", withScope(testSDJWTConfig + "  " + testMdocConfig), StatusPass},
		{"one unknown scope value", withScope(testSDJWTConfig + " openid"), StatusFail},
		{"empty scope", withScope(" "), StatusFail},
		{"non-string scope", withScope([]any{testSDJWTConfig}), StatusFail},
		{"no PAR", testExchange(nil, nil), StatusFail},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := OID4VCIPARScopeSupportedValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
		})
	}
}

func TestOID4VCICredentialConfigurationScopeRequestedValidator(t *testing.T) {
	withConfiguration := func(id any) map[string]any {
		exchange := testExchange(testSessionRequests(), nil)
		raw := exchange["capture_session"].(map[string]any)["raw"].(map[string]any)
		raw["credential_request"].(map[string]any)["credential_configuration_id"] = id
		return exchange
	}
	for _, tt := range []struct {
		name   string
		value  any
		status Status
	}{
		{"configuration of the requested scope", withConfiguration(testSDJWTConfig), StatusPass},
		{"advertised configuration whose scope was not requested", withConfiguration(testMdocConfig), StatusFail},
		{"unknown configuration", withConfiguration("urn:unknown"), StatusFail},
		{"non-string identifier", withConfiguration(7.0), StatusFail},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := OID4VCICredentialConfigurationScopeRequestedValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
		})
	}
}

func TestOID4VCIProofJWTHeaderValidator(t *testing.T) {
	jwtHeader := func(mutate func(map[string]any)) []any {
		header := map[string]any{
			"typ":                     "openid4vci-proof+jwt",
			"alg":                     "ES256",
			"jwk":                     map[string]any{"kty": "EC"},
			"proof_type":              "jwt",
			"key_attestation_present": true,
			"source":                  "credential_request.proofs.jwt[0].header",
		}
		mutate(header)
		return []any{header}
	}
	unchanged := func(map[string]any) {}
	attestationOnly := []any{
		map[string]any{
			"typ":                     "key-attestation+jwt",
			"alg":                     "ES256",
			"proof_type":              "attestation",
			"key_attestation_present": false,
		},
	}
	presence := map[string]any{"required_members": []any{"alg", "typ", "key_attestation"}}
	typString := map[string]any{"string_members": []any{"typ"}}
	typValue := map[string]any{"member_equals": map[string]any{"typ": "openid4vci-proof+jwt"}}
	algValue := map[string]any{
		"member_one_of": map[string]any{"alg": []any{"ES256", "ES384", "ES512"}},
	}

	for _, tt := range []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{"members present", jwtHeader(unchanged), presence, StatusPass},
		{"key attestation absent", jwtHeader(func(h map[string]any) { h["key_attestation_present"] = false }), presence, StatusFail},
		{"typ null", jwtHeader(func(h map[string]any) { h["typ"] = nil }), presence, StatusFail},
		{"typ string", jwtHeader(unchanged), typString, StatusPass},
		{"typ number", jwtHeader(func(h map[string]any) { h["typ"] = 1.0 }), typString, StatusFail},
		{"typ value", jwtHeader(unchanged), typValue, StatusPass},
		{"typ of a key attestation", jwtHeader(func(h map[string]any) { h["typ"] = "key-attestation+jwt" }), typValue, StatusFail},
		{"ES512", jwtHeader(func(h map[string]any) { h["alg"] = "ES512" }), algValue, StatusPass},
		{"ES256K", jwtHeader(func(h map[string]any) { h["alg"] = "ES256K" }), algValue, StatusFail},
		{"RS256", jwtHeader(func(h map[string]any) { h["alg"] = "RS256" }), algValue, StatusFail},
		{"attestation proofs only", attestationOnly, presence, StatusNotApplicable},
		{"no proof header recorded", nil, presence, StatusInconclusive},
		{"no check", jwtHeader(unchanged), map[string]any{}, StatusError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := OID4VCIProofJWTHeaderValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
		})
	}

	mixed := append(jwtHeader(func(h map[string]any) { h["alg"] = "EdDSA" }), attestationOnly...)
	result := OID4VCIProofJWTHeaderValidator{}.Validate(
		context.Background(),
		Input{Value: mixed, Params: algValue},
	)
	require.Equal(
		t,
		StatusFail,
		result.Status,
		"a jwt proof among other proof types is still checked",
	)
}

func TestOID4VCIIssuedCredentialPresentedValidator(t *testing.T) {
	holderKey := map[string]any{"kty": "EC", "crv": "P-256", "x": "xPF", "y": "nKe"}
	evidence := func(mutate func(issuance, presentation map[string]any)) map[string]any {
		issuance := map[string]any{
			"status":            "credential_issued",
			"issuer_identifier": testIssuerBase,
			"observed": map[string]any{"wallet_jwks": map[string]any{
				"jwks": map[string]any{"keys": []any{holderKey}},
			}},
		}
		presentation := map[string]any{
			"status": "presentation_validated",
			"checks": map[string]any{"presentation_valid": true},
			"raw": map[string]any{"decoded_presentations": map[string]any{
				"issued_pid": []any{map[string]any{
					"format": "dc+sd-jwt",
					"claims": map[string]any{
						"iss": testIssuerBase,
						"cnf": map[string]any{
							"jwk": map[string]any{
								"kty": "EC",
								"crv": "P-256",
								"x":   "xPF",
								"y":   "nKe",
							},
						},
					},
				}},
			}},
		}
		mutate(issuance, presentation)
		return map[string]any{"issuance_session": issuance, "presentation_session": presentation}
	}
	presented := func(presentation map[string]any) map[string]any {
		byQuery := presentation["raw"].(map[string]any)["decoded_presentations"].(map[string]any)
		return byQuery["issued_pid"].([]any)[0].(map[string]any)["claims"].(map[string]any)
	}
	for _, tt := range []struct {
		name   string
		value  any
		status Status
	}{
		{"issued credential presented", evidence(func(_, _ map[string]any) {}), StatusPass},
		{"issuance did not complete", evidence(func(i, _ map[string]any) { i["status"] = "authorization_requested" }), StatusFail},
		{"no holder key observed", evidence(func(i, _ map[string]any) { delete(i, "observed") }), StatusInconclusive},
		{"presentation rejected", evidence(func(_, p map[string]any) { p["status"] = "presentation_invalid" }), StatusFail},
		{"presentation not valid", evidence(func(_, p map[string]any) {
			p["checks"].(map[string]any)["presentation_valid"] = false
		}), StatusFail},
		{"another credential of the same issuer", evidence(func(_, p map[string]any) {
			presented(p)["cnf"] = map[string]any{"jwk": map[string]any{"kty": "EC", "crv": "P-256", "x": "other", "y": "nKe"}}
		}), StatusFail},
		{"another issuer's credential", evidence(func(_, p map[string]any) {
			presented(p)["iss"] = "https://capture.test/issuers/eu-pid-jwt-proof-only"
		}), StatusFail},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := OID4VCIIssuedCredentialPresentedValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
		})
	}
}
