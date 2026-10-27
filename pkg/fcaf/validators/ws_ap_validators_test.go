// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	wsAPAnchor         = "2026-10-22T07:38:29.297Z"
	wsAPRequestURI     = "urn:ietf:params:oauth:request_uri:5620ecbd"
	wsAPDeeplink       = "eudi-openid4ci://?credential_offer=%7B%22credential_issuer%22%3A%22https%3A%2F%2Fcapture.test%2Fissuers%2Feu-pid-device-bound%22%2C%22grants%22%3A%7B%7D%7D"
	wsAPInvokedShot    = "https://capture.test/files/engage_offer_wallet_invoked_ab12cd34ef.png"
	wsAPNotInvokedShot = "https://capture.test/files/engage_offer_wallet_not_invoked_ab12cd34ef.png"
)

// wsBase64 is unpadded base64url, the encoding of every compact JOSE part.
func wsBase64(text string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(text))
}

// wsCompactJWS builds a compact JWS from raw header and payload text.
func wsCompactJWS(header, payload, signature string) string {
	return wsBase64(header) + "." + wsBase64(payload) + "." + signature
}

// wsCompactJWE builds a five-part compact JWE with an empty encrypted key,
// the shape a "dir" key management mode produces.
func wsCompactJWE(header, encryptedKey string) string {
	return wsBase64(header) + "." + encryptedKey + ".aXY.Y3Q.dGFn"
}

// wsSignedJWT signs claims with an ES256 (or other) key and adds header members.
func wsSignedJWT(
	t *testing.T,
	method jwt.SigningMethod,
	key any,
	header, claims map[string]any,
) string {
	t.Helper()
	if claims == nil {
		claims = map[string]any{}
	}
	token := jwt.NewWithClaims(method, jwt.MapClaims(claims))
	for name, value := range header {
		token.Header[name] = value
	}
	compact, err := token.SignedString(key)
	require.NoError(t, err)
	return compact
}

// wsTamperPayload replaces a compact JWS payload without re-signing it.
func wsTamperPayload(t *testing.T, compact, payload string) string {
	t.Helper()
	parts := strings.Split(compact, ".")
	require.Len(t, parts, 3)
	parts[1] = wsBase64(payload)
	return strings.Join(parts, ".")
}

func wsECJWK(key *ecdsa.PrivateKey) map[string]any {
	return map[string]any{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(key.X.FillBytes(make([]byte, 32))),
		"y":   base64.RawURLEncoding.EncodeToString(key.Y.FillBytes(make([]byte, 32))),
	}
}

func wsSelfSignedCertificate(t *testing.T, key *ecdsa.PrivateKey) []byte {
	t.Helper()
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	return der
}

// wsPARWithResponse adds the issuer's PAR response, whose request_uri the
// Authorization Request must repeat.
func wsPARWithResponse(requestURI any) map[string]any {
	par := testPAR()
	body := map[string]any{}
	if requestURI != nil {
		body["request_uri"] = requestURI
	}
	par["response"] = map[string]any{
		"status":       float64(201),
		"content_type": "application/json; charset=utf-8",
		"body":         body,
	}
	return par
}

// wsJWTExchange puts one JWT in the PAR's raw_headers, where Capture records
// the OAuth-Client-Attestation.
func wsJWTExchange(jwtValue string) map[string]any {
	par := testPAR()
	par["raw_headers"] = []any{
		map[string]any{"name": "Oauth-Client-Attestation", "value": jwtValue},
	}
	exchange := testExchange([]any{par})
	exchange["issuer_metadata"].(map[string]any)["credential_issuer"] = testIssuerBase
	return exchange
}

func wsNonceRequest(at, nonce string) map[string]any {
	request := testCapturedRequest(
		at, "POST", "/issuers/eu-pid-device-bound/nonce", nil, map[string]any{}, nil,
	)
	request["response"] = map[string]any{
		"status":       float64(200),
		"content_type": "application/json",
		"body":         map[string]any{"c_nonce": nonce},
	}
	return request
}

func wsNonceExchange(requests []any, proofs map[string]any) map[string]any {
	exchange := testExchange(requests)
	if proofs != nil {
		exchange["capture_session"].(map[string]any)["raw"].(map[string]any)["credential_request_proofs"] = proofs
	}
	return exchange
}

func wsUnattributedRequest(at, path, host string) map[string]any {
	return testCapturedRequest(at, "GET", path, nil, nil, map[string]any{"host": host})
}

// wsObservedExchange is an exchange with a session_requests window, and with
// the issuer identifier an unattributed request's Host is matched against.
func wsObservedExchange(requests []any, windowTo string) map[string]any {
	exchange := testExchange(requests)
	exchange["capture_session"].(map[string]any)["issuer_identifier"] = testIssuerBase
	if windowTo != "" {
		exchange["session_requests"] = map[string]any{
			"window": map[string]any{
				"from": "2026-10-22T07:37:00Z",
				"to":   windowTo,
			},
			"unattributed_requests": []any{},
		}
	}
	return exchange
}

func wsOfferParams() map[string]any {
	return map[string]any{
		"scheme":           "eudi-openid4ci",
		"offer_present":    []any{"credential_issuer"},
		"when_not_invoked": "fail",
	}
}

func wsOfferExchange() map[string]any {
	exchange := testExchange(testSessionRequests())
	session := exchange["capture_session"].(map[string]any)
	session["issuer_identifier"] = testIssuerBase
	session["events"] = []any{map[string]any{
		"at":   "2026-10-22T07:37:40.000Z",
		"type": defaultEngagementPageEvent,
	}}
	// The deeplink carries the delivered offer by value: credential_issuer and
	// grants, no credential_configuration_ids.
	exchange["session_creation"] = map[string]any{"deeplink": wsAPDeeplink}
	exchange["screenshots"] = []any{wsAPInvokedShot}
	return exchange
}

func wsCredentialBodyExchange(rawBody map[string]any, advertiseEncryption bool) map[string]any {
	request := testCapturedRequest(
		wsAPAnchor, "POST", "/issuers/eu-pid-device-bound/credential", nil, nil,
		map[string]any{"content-type": "application/json"},
	)
	if rawBody != nil {
		request["raw_body"] = rawBody
	}
	exchange := testExchange([]any{request})
	if advertiseEncryption {
		exchange["issuer_metadata"].(map[string]any)["credential_request_encryption"] = map[string]any{
			"alg_values_supported": []any{"ECDH-ES"},
		}
	}
	return exchange
}

func TestSharedJSONValidator(t *testing.T) {
	jsonBody := func(data string) map[string]any {
		return map[string]any{
			"content_type": "application/json",
			"base64":       base64.StdEncoding.EncodeToString([]byte(data)),
		}
	}
	for _, tt := range []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{"valid object", `{"a":1}`, nil, StatusPass, ""},
		{"valid array", `[1,"two",{"three":true}]`, nil, StatusPass, ""},
		{"invalid syntax", `{"a":`, nil, StatusFail, "Shared_JSON_01"},
		{"two JSON texts", `{"a":1}{"b":2}`, nil, StatusFail, "Shared_JSON_01"},
		{"duplicate member in a nested object", `{"a":{"b":1,"b":2}}`, nil, StatusFail, "Shared_JSON_02"},
		{"duplicate member in an array element", `{"a":[{"b":1,"b":2}]}`, nil, StatusFail, "Shared_JSON_02"},
		{"duplicate member under an escaped name", `{"a":1,"\u0061":2}`, nil, StatusFail, "Shared_JSON_02"},
		{"same member name in sibling objects", `[{"a":1},{"a":2}]`, nil, StatusPass, ""},
		{"same member name in nested objects", `{"a":{"a":1}}`, nil, StatusPass, ""},
		{"byte order mark", "\ufeff{}", map[string]any{"cases": []any{"03"}}, StatusFail, "byte order mark"},
		{"invalid UTF-8", "{\"a\":\"\xff\"}", map[string]any{"cases": []any{"03"}}, StatusFail, "valid UTF-8"},
		{"unpaired high surrogate escape", `{"a":"\ud800"}`, nil, StatusFail, "unpaired surrogate"},
		{"unpaired low surrogate escape", `{"a":"\ude00"}`, nil, StatusFail, "unpaired surrogate"},
		{"paired surrogate escape", `{"a":"\ud83d\ude00"}`, nil, StatusPass, ""},
		{"raw body base64", jsonBody(`{"a":1}`), nil, StatusPass, ""},
		{"raw body base64 duplicate member", jsonBody(`{"a":1,"a":2}`), nil, StatusFail, "Shared_JSON_02"},
		{"raw body text", map[string]any{"content_type": "application/json", "text": `{"a":1}`}, nil, StatusPass, ""},
		{"cases restrict to one passing case", `{"a":1,"a":2}`, map[string]any{"cases": []any{"01"}}, StatusPass, ""},
		{"unknown case", `{"a":1}`, map[string]any{"cases": []any{"09"}}, StatusError, "has no case"},
		{"redacted raw body", redacted(true), nil, StatusInconclusive, "redacted"},
		{"neither string nor raw body", float64(7), nil, StatusFail, "expected a string or a raw body"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := SharedJSONValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestSharedJWTJWSValidator(t *testing.T) {
	valid := wsCompactJWS(`{"alg":"ES256"}`, `{"iss":"https://capture.test"}`, "c2ln")
	for _, tt := range []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{"valid compact JWS", valid, nil, StatusPass, ""},
		{"two parts", "a.b", nil, StatusFail, "Shared_JWT_JWS_01"},
		{"four parts", "a.b.c.d", nil, StatusFail, "Shared_JWT_JWS_01"},
		{"padded signature", wsCompactJWS(`{"alg":"ES256"}`, `{}`, "c2ln="), nil, StatusFail, "not an unpadded base64url character"},
		{"space in a part", wsCompactJWS(`{"alg":"ES256"}`, `{}`, "c2 ln"), nil, StatusFail, "base64url character"},
		{"impossible part length", wsCompactJWS(`{"alg":"ES256"}`, `{}`, "abcde"), nil, StatusFail, "not canonical unpadded base64url"},
		{"stray bits in the signature", wsCompactJWS(`{"alg":"ES256"}`, `{}`, "AB"), nil, StatusFail, "not canonical unpadded base64url"},
		{"header is not JSON", wsCompactJWS(`not json`, `{}`, "c2ln"), nil, StatusFail, "Shared_JSON_01"},
		{"duplicate header member", wsCompactJWS(`{"alg":"ES256","alg":"ES256"}`, `{}`, "c2ln"), nil, StatusFail, "Shared_JSON_02"},
		{"header without alg", wsCompactJWS(`{"typ":"JWT"}`, `{}`, "c2ln"), nil, StatusFail, "no alg member"},
		{"header without alg, case 01 only", wsCompactJWS(`{"typ":"JWT"}`, `{}`, "c2ln"), map[string]any{"cases": []any{"01"}}, StatusPass, ""},
		{"empty header", wsCompactJWS(`{}`, `{}`, "c2ln"), nil, StatusFail, "has no member"},
		{"payload is an array", wsCompactJWS(`{"alg":"ES256"}`, `[1,2]`, "c2ln"), nil, StatusFail, "not a JSON object"},
		{"payload duplicate member", wsCompactJWS(`{"alg":"ES256"}`, `{"a":1,"a":2}`, "c2ln"), nil, StatusFail, "Shared_JSON_02"},
		{"redacted value", redacted(true), nil, StatusInconclusive, "redacted"},
		{"not a string", float64(1), nil, StatusFail, "expected a string or a raw body"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := SharedJWTJWSValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestSharedJWTJWEValidator(t *testing.T) {
	for _, tt := range []struct {
		name    string
		value   any
		status  Status
		message string
	}{
		{"valid five-part JWE with empty encrypted key", wsCompactJWE(`{"alg":"dir","enc":"A128GCM"}`, ""), StatusPass, ""},
		{"non-empty encrypted key", wsCompactJWE(`{"alg":"ECDH-ES","enc":"A128GCM"}`, "a2V5"), StatusPass, ""},
		{"four parts", wsBase64(`{"alg":"dir","enc":"A128GCM"}`) + ".aXY.Y3Q.dGFn", StatusFail, "Shared_JWT_JWE_01"},
		{"header without enc", wsCompactJWE(`{"alg":"dir","zip":"DEF"}`, ""), StatusFail, "no enc member"},
		{"header without alg", wsCompactJWE(`{"enc":"A128GCM","zip":"DEF"}`, ""), StatusFail, "no alg member"},
		{"single-member header", wsCompactJWE(`{}`, ""), StatusFail, "expected at least two"},
		{"header is not JSON", wsCompactJWE(`not json`, ""), StatusFail, "Shared_JSON_01"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := SharedJWTJWEValidator{}.Validate(context.Background(), Input{Value: tt.value})
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestCaptureSelector(t *testing.T) {
	par := testPAR()
	par["raw_headers"] = []any{
		map[string]any{"name": "Oauth-Client-Attestation", "value": "ca-jwt"},
		map[string]any{"name": "OAuth-Client-Attestation-PoP", "value": "pop-jwt"},
	}
	exchange := testExchange([]any{par})
	exchange["capture_session"].(map[string]any)["raw"].(map[string]any)["credential_request_proofs"] =
		map[string]any{
			"at": wsAPAnchor,
			"jwt": []any{
				map[string]any{"payload": map[string]any{"nonce": "n-1"}},
				map[string]any{"payload": map[string]any{"nonce": "n-2"}},
			},
		}
	redactedHeaders := testPAR()
	redactedHeaders["raw_headers"] = []any{
		map[string]any{"name": "Oauth-Client-Attestation", "value": redacted(true)},
	}
	redactedExchange := testExchange([]any{redactedHeaders})

	for _, tt := range []struct {
		name     string
		value    any
		selector captureSelector
		status   Status
		want     []any
		message  string
	}{
		{
			name:     "header name in any case",
			value:    exchange,
			selector: captureSelector{Request: "/par", Path: "raw_headers[oauth-client-attestation]"},
			want:     []any{"ca-jwt"},
		},
		{
			name:     "dashed header name",
			value:    exchange,
			selector: captureSelector{Request: "/par", Path: "raw_headers[oauth-client-attestation-pop]"},
			want:     []any{"pop-jwt"},
		},
		{
			name:     "every array element",
			value:    exchange,
			selector: captureSelector{Path: "raw.credential_request_proofs.jwt[].payload.nonce"},
			want:     []any{"n-1", "n-2"},
		},
		{
			name:     "numeric index",
			value:    exchange,
			selector: captureSelector{Path: "raw.credential_request_proofs.jwt.1.payload.nonce"},
			want:     []any{"n-2"},
		},
		{
			name:     "index out of range selects nothing",
			value:    exchange,
			selector: captureSelector{Path: "raw.credential_request_proofs.jwt.2.payload.nonce", Absent: StatusNotApplicable},
			status:   StatusNotApplicable,
		},
		{
			name:     "nothing selected is not applicable",
			value:    exchange,
			selector: captureSelector{Request: "/par", Path: "raw_headers[Missing]", Absent: StatusNotApplicable},
			status:   StatusNotApplicable,
		},
		{
			name:     "nothing selected defaults to fail",
			value:    exchange,
			selector: captureSelector{Request: "/par", Path: "raw_headers[Missing]"},
			status:   StatusFail,
		},
		{
			name:     "unknown request",
			value:    exchange,
			selector: captureSelector{Request: "/token", Path: "raw_headers[oauth-client-attestation]"},
			status:   StatusFail,
		},
		{
			name:     "required path missing",
			value:    exchange,
			selector: captureSelector{Request: "/par", Path: "raw_headers[oauth-client-attestation]", Require: "raw_body"},
			status:   StatusFail,
		},
		{
			name:     "required path present, selected path absent",
			value:    exchange,
			selector: captureSelector{Request: "/par", Path: "raw_headers[Missing]", Require: "raw_headers", Absent: StatusNotApplicable},
			status:   StatusNotApplicable,
		},
		{
			name:     "redacted value is inconclusive",
			value:    redactedExchange,
			selector: captureSelector{Request: "/par", Path: "raw_headers[oauth-client-attestation]"},
			status:   StatusInconclusive,
			message:  "redacted",
		},
		{
			name:     "path starts at the input without capture_session",
			value:    map[string]any{"a": "b"},
			selector: captureSelector{Path: "a"},
			want:     []any{"b"},
		},
		{
			name:     "non-object input",
			value:    "text",
			selector: captureSelector{Path: "a"},
			status:   StatusFail,
		},
		{
			name:     "empty path",
			value:    exchange,
			selector: captureSelector{Request: "/par"},
			status:   StatusError,
		},
		{
			name:     "invalid absent status",
			value:    exchange,
			selector: captureSelector{Request: "/par", Path: "raw_headers[Missing]", Absent: StatusPass},
			status:   StatusError,
		},
		{
			name:     "reason is reported",
			value:    exchange,
			selector: captureSelector{Request: "/par", Path: "raw_headers[Missing]", Absent: StatusNotApplicable, Reason: "the source does not require it"},
			status:   StatusNotApplicable,
			message:  "the source does not require it",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			selected, result := tt.selector.resolve(tt.value)
			if tt.status != "" {
				require.NotNil(t, result)
				require.Equal(t, tt.status, result.Status, result.Message)
				require.Contains(t, result.Message, tt.message)
				return
			}
			require.Nil(t, result)
			values := make([]any, 0, len(selected))
			for _, item := range selected {
				values = append(values, item.value)
			}
			require.Equal(t, tt.want, values)
		})
	}

	_, result := resolveSelection(Input{Params: map[string]any{"select": "not an object"}})
	require.NotNil(t, result)
	require.Equal(t, StatusError, result.Status)
}

func TestJOSEMemberValidator(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	claims := map[string]any{
		"iss": testIssuerBase,
		"aud": testIssuerBase,
		"sub": float64(7),
	}
	token := wsSignedJWT(t, jwt.SigningMethodES256, key, nil, claims)
	otherAudience := wsSignedJWT(
		t,
		jwt.SigningMethodES256,
		key,
		nil,
		map[string]any{"aud": "https://other.test"},
	)
	selectParams := map[string]any{
		"request": "/par",
		"path":    "raw_headers[oauth-client-attestation]",
	}
	exchange := wsJWTExchange(token)
	otherAudienceExchange := wsJWTExchange(otherAudience)
	withoutIssuer := wsJWTExchange(token)
	delete(withoutIssuer["issuer_metadata"].(map[string]any), "credential_issuer")

	for _, tt := range []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{
			name:   "required and equal payload members",
			value:  token,
			params: map[string]any{"part": "payload", "required": []any{"iss"}, "equals": map[string]any{"iss": testIssuerBase}},
			status: StatusPass,
		},
		{
			name:    "absent required member",
			value:   token,
			params:  map[string]any{"part": "payload", "required": []any{"scope"}},
			status:  StatusFail,
			message: "member scope is absent",
		},
		{
			name:    "member is not a string",
			value:   token,
			params:  map[string]any{"part": "payload", "string": []any{"sub"}},
			status:  StatusFail,
			message: "expected string",
		},
		{
			name:   "string member",
			value:  token,
			params: map[string]any{"part": "payload", "string": []any{"iss"}},
			status: StatusPass,
		},
		{
			name:   "header alg is one of",
			value:  token,
			params: map[string]any{"part": "header", "one_of": map[string]any{"alg": []any{"ES256", "ES384"}}},
			status: StatusPass,
		},
		{
			name:    "header alg is not one of",
			value:   token,
			params:  map[string]any{"part": "header", "one_of": map[string]any{"alg": []any{"HS256"}}},
			status:  StatusFail,
			message: "expected one of",
		},
		{
			name:  "aud equals the exchange's credential issuer",
			value: exchange,
			params: map[string]any{
				"select":          selectParams,
				"part":            "payload",
				"required":        []any{"aud"},
				"equals_exchange": map[string]any{"aud": "issuer_metadata.credential_issuer"},
			},
			status: StatusPass,
		},
		{
			name:  "aud differs from the exchange's credential issuer",
			value: otherAudienceExchange,
			params: map[string]any{
				"select":          selectParams,
				"part":            "payload",
				"equals_exchange": map[string]any{"aud": "issuer_metadata.credential_issuer"},
			},
			status:  StatusFail,
			message: "expected",
		},
		{
			name:  "exchange records no credential issuer",
			value: withoutIssuer,
			params: map[string]any{
				"select":          selectParams,
				"part":            "payload",
				"equals_exchange": map[string]any{"aud": "issuer_metadata.credential_issuer"},
			},
			status:  StatusInconclusive,
			message: "credential_issuer",
		},
		{
			name:    "unknown part",
			value:   token,
			params:  map[string]any{"part": "body", "required": []any{"iss"}},
			status:  StatusError,
			message: "part must be header or payload",
		},
		{
			name:    "no member check",
			value:   token,
			params:  map[string]any{"part": "payload"},
			status:  StatusError,
			message: "at least one member check is required",
		},
		{
			name:    "value is not a compact JWS",
			value:   "not-a-jwt",
			params:  map[string]any{"part": "payload", "required": []any{"iss"}},
			status:  StatusFail,
			message: "not a compact JWS",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := JOSEMemberValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestJOSESignatureAlgorithmValidator(t *testing.T) {
	key256, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	key384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	key521, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	require.NoError(t, err)
	es256 := wsSignedJWT(
		t,
		jwt.SigningMethodES256,
		key256,
		nil,
		map[string]any{"iss": testIssuerBase},
	)
	es384 := wsSignedJWT(
		t,
		jwt.SigningMethodES384,
		key384,
		nil,
		map[string]any{"iss": testIssuerBase},
	)
	es512 := wsSignedJWT(
		t,
		jwt.SigningMethodES512,
		key521,
		nil,
		map[string]any{"iss": testIssuerBase},
	)
	hs256 := wsSignedJWT(
		t,
		jwt.SigningMethodHS256,
		[]byte("secret"),
		nil,
		map[string]any{"iss": testIssuerBase},
	)
	none := wsCompactJWS(`{"alg":"none"}`, `{}`, "")
	withoutAlg := wsCompactJWS(`{"typ":"JWT"}`, `{}`, "c2ln")

	for _, tt := range []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{"default accepts ES256", es256, nil, StatusPass, "ES256"},
		{"default accepts ES384", es384, nil, StatusPass, "ES384"},
		{"default accepts ES512", es512, nil, StatusPass, "ES512"},
		{"HS256 is not a default algorithm", hs256, nil, StatusFail, "expected one of"},
		{"none is not a signature algorithm", none, nil, StatusFail, "expected one of"},
		{"explicit allowed list rejects ES384", es384, map[string]any{"allowed": []any{"ES256"}}, StatusFail, "expected one of ES256"},
		{"explicit allowed list accepts ES384", es384, map[string]any{"allowed": []any{"ES256", "ES384"}}, StatusPass, ""},
		{"header without alg", withoutAlg, nil, StatusFail, "expected one of"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := JOSESignatureAlgorithmValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestJOSESignatureValidator(t *testing.T) {
	signingKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	popKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	attestationKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	claims := map[string]any{"iss": testIssuerBase}

	leaf := wsSelfSignedCertificate(t, signingKey)
	x5cToken := wsSignedJWT(t, jwt.SigningMethodES256, signingKey, map[string]any{
		"x5c": []string{base64.StdEncoding.EncodeToString(leaf)},
	}, claims)
	tamperedToken := wsTamperPayload(t, x5cToken, `{"iss":"https://attacker.test"}`)
	withoutX5C := wsSignedJWT(t, jwt.SigningMethodES256, signingKey, nil, claims)
	// An ES384 label over a P-256 signature whose R and S are zero-padded to
	// the 48 bytes golang-jwt checks: the signature math verifies.
	es384OverP256 := func() string {
		signingInput := wsBase64(`{"alg":"ES384","x5c":["`+
			base64.StdEncoding.EncodeToString(leaf)+`"]}`) + "." + wsBase64(`{}`)
		digest := sha512.Sum384([]byte(signingInput))
		r, s, err := ecdsa.Sign(rand.Reader, signingKey, digest[:])
		require.NoError(t, err)
		signature := append(r.FillBytes(make([]byte, 48)), s.FillBytes(make([]byte, 48))...)
		return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
	}()

	clientAttestation := wsSignedJWT(t, jwt.SigningMethodES256, attestationKey, nil, map[string]any{
		"cnf": map[string]any{"jwk": wsECJWK(popKey)},
	})
	popToken := wsSignedJWT(t, jwt.SigningMethodES256, popKey, nil, claims)
	popSignedByAnotherKey := wsSignedJWT(t, jwt.SigningMethodES256, signingKey, nil, claims)

	headers := func(pop string) map[string]any {
		par := testPAR()
		par["raw_headers"] = []any{
			map[string]any{"name": "Oauth-Client-Attestation", "value": clientAttestation},
			map[string]any{"name": "OAuth-Client-Attestation-PoP", "value": pop},
		}
		return testExchange([]any{par})
	}
	popSelect := map[string]any{
		"select": map[string]any{
			"request": "/par",
			"path":    "raw_headers[oauth-client-attestation-pop]",
		},
	}
	withKey := func(key map[string]any) map[string]any {
		params := map[string]any{}
		for name, value := range popSelect {
			params[name] = value
		}
		params["key"] = key
		return params
	}

	siblingKey := map[string]any{
		"source": "sibling",
		"path":   "raw_headers[Oauth-Client-Attestation]",
		"claim":  "cnf.jwk",
	}
	attestationNames := func(key *ecdsa.PrivateKey) string {
		return wsSignedJWT(t, jwt.SigningMethodES256, attestationKey, nil, map[string]any{
			"attested_keys": []any{wsECJWK(key)},
		})
	}
	headerJWTPop := func(attestation string) string {
		return wsSignedJWT(t, jwt.SigningMethodES256, popKey, map[string]any{
			"key_attestation": attestation,
		}, claims)
	}

	for _, tt := range []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{"x5c signature verifies", x5cToken, map[string]any{"key": map[string]any{"source": "x5c"}}, StatusPass, ""},
		{"x5c tampered payload", tamperedToken, map[string]any{"key": map[string]any{"source": "x5c"}}, StatusFail, "does not verify"},
		{"x5c header without certificate", withoutX5C, map[string]any{"key": map[string]any{"source": "x5c"}}, StatusFail, "no x5c certificate"},
		{"ES384 label over a P-256 key", es384OverP256, map[string]any{"key": map[string]any{"source": "x5c"}}, StatusFail, "requires P-384"},
		{
			name:   "sibling cnf.jwk verifies the PoP",
			value:  headers(popToken),
			params: withKey(siblingKey),
			status: StatusPass,
		},
		{
			name:    "sibling cnf.jwk is another key",
			value:   headers(popSignedByAnotherKey),
			params:  withKey(siblingKey),
			status:  StatusFail,
			message: "does not verify",
		},
		{
			name:    "sibling path not recorded",
			value:   headers(popToken),
			params:  withKey(map[string]any{"source": "sibling", "path": "raw_headers[Missing]", "claim": "cnf.jwk"}),
			status:  StatusFail,
			message: "records no",
		},
		{
			name:    "sibling claim missing",
			value:   headers(popToken),
			params:  withKey(map[string]any{"source": "sibling", "path": "raw_headers[Oauth-Client-Attestation]", "claim": "cnf.other"}),
			status:  StatusFail,
			message: "no cnf.other claim",
		},
		{
			name:   "header_jwt attested key verifies the PoP",
			value:  headerJWTPop(attestationNames(popKey)),
			params: map[string]any{"key": map[string]any{"source": "header_jwt", "member": "key_attestation", "claim": "attested_keys.0"}},
			status: StatusPass,
		},
		{
			name:    "header_jwt attests another key",
			value:   headerJWTPop(attestationNames(signingKey)),
			params:  map[string]any{"key": map[string]any{"source": "header_jwt", "member": "key_attestation", "claim": "attested_keys.0"}},
			status:  StatusFail,
			message: "does not verify",
		},
		{
			name:    "header_jwt member missing",
			value:   popToken,
			params:  map[string]any{"key": map[string]any{"source": "header_jwt", "member": "key_attestation", "claim": "attested_keys.0"}},
			status:  StatusFail,
			message: "has no key_attestation JWT",
		},
		{
			name:    "header_jwt without member and claim",
			value:   popToken,
			params:  map[string]any{"key": map[string]any{"source": "header_jwt"}},
			status:  StatusError,
			message: "key.member and key.claim are required",
		},
		{
			name:    "unknown key source",
			value:   popToken,
			params:  map[string]any{"key": map[string]any{"source": "jwks"}},
			status:  StatusError,
			message: "key.source must be x5c, header_jwt or sibling",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := JOSESignatureValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestHTTPHeaderCountValidator(t *testing.T) {
	params := func(count int) map[string]any {
		return map[string]any{
			"request": "/par",
			"header":  "oauth-client-attestation",
			"count":   count,
		}
	}
	withCounts := func(counts any) map[string]any {
		par := testPAR()
		if counts != nil {
			par["header_counts"] = counts
		}
		return testExchange([]any{par})
	}
	for _, tt := range []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{"one occurrence", withCounts(map[string]any{"oauth-client-attestation": float64(1)}), params(1), StatusPass, ""},
		{"two occurrences for one", withCounts(map[string]any{"oauth-client-attestation": float64(2)}), params(1), StatusFail, "carries 2"},
		{"absent header counts as zero", withCounts(map[string]any{"other": float64(1)}), params(0), StatusPass, ""},
		{"absent header cannot satisfy one", withCounts(map[string]any{"other": float64(1)}), params(1), StatusFail, "carries 0"},
		{
			name:   "header param name is case-insensitive",
			value:  withCounts(map[string]any{"oauth-client-attestation": float64(1)}),
			params: map[string]any{"request": "/par", "header": "OAuth-Client-Attestation", "count": 1},
			status: StatusPass,
		},
		{"no header_counts is inconclusive", withCounts(nil), params(1), StatusInconclusive, "header_counts"},
		{"no request", testExchange(nil), params(1), StatusFail, "no Wallet request"},
		{
			name:    "missing params",
			value:   withCounts(map[string]any{"oauth-client-attestation": float64(1)}),
			params:  map[string]any{"request": "/par"},
			status:  StatusError,
			message: "request, header and count params are required",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := HTTPHeaderCountValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestJSONValueShapeValidator(t *testing.T) {
	exchange := testExchange(testSessionRequests())
	for _, tt := range []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{"string", "abc", map[string]any{"type": "string"}, StatusPass, ""},
		{"number", float64(1), map[string]any{"type": "number"}, StatusPass, ""},
		{"wrong type", "abc", map[string]any{"type": "number"}, StatusFail, "expected number"},
		{
			name:   "array length and item type",
			value:  []any{"a", "b"},
			params: map[string]any{"type": "array", "length": 2, "items_type": "string"},
			status: StatusPass,
		},
		{"array length mismatch", []any{"a"}, map[string]any{"length": 2}, StatusFail, "has 1 element(s), expected 2"},
		{"item type mismatch", []any{"a", float64(1)}, map[string]any{"items_type": "string"}, StatusFail, "expected string"},
		{"length of a non-array", "abc", map[string]any{"length": 1}, StatusFail, "expected array"},
		{"unknown type name", "abc", map[string]any{"type": "text"}, StatusError, "is unknown"},
		{
			name:   "presence through a selector",
			value:  exchange,
			params: map[string]any{"select": map[string]any{"request": "/par", "path": "body.scope"}},
			status: StatusPass,
		},
		{
			name:    "selected member absent",
			value:   exchange,
			params:  map[string]any{"select": map[string]any{"request": "/par", "path": "body.missing"}},
			status:  StatusFail,
			message: "no body.missing",
		},
		{
			name:  "selected member absent, not applicable",
			value: exchange,
			params: map[string]any{"select": map[string]any{
				"request": "/par",
				"path":    "body.missing",
				"absent":  "not_applicable",
			}},
			status: StatusNotApplicable,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := JSONValueShapeValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestURISyntaxValidator(t *testing.T) {
	authorize := testAuthorizeRequest("2026-10-22T07:38:24.600Z")
	authorized := testExchange([]any{authorize})
	redactedAuthorize := testAuthorizeRequest("2026-10-22T07:38:24.600Z")
	redactedAuthorize["query"].(map[string]any)["request_uri"] = redacted(true)
	redactedExchange := testExchange([]any{redactedAuthorize})
	selectURI := map[string]any{
		"select": map[string]any{"request": "/authorize", "path": "query.request_uri"},
	}

	for _, tt := range []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{"request_uri URN", wsAPRequestURI, nil, StatusPass, ""},
		{"https URL", "https://capture.test/issuers/eu-pid-device-bound/par", nil, StatusPass, ""},
		{"space", "https://capture.test/a b", nil, StatusFail, "outside the RFC 3986 set"},
		{"no scheme", "capture.test/par", nil, StatusFail, "no RFC 3986 scheme"},
		{"bad percent-encoding", "https://capture.test/%zz", nil, StatusFail, "invalid percent-encoding"},
		{"empty string", "", nil, StatusFail, "is empty"},
		{"not a string", map[string]any{"a": 1}, nil, StatusFail, "expected string"},
		{"selected request_uri", authorized, selectURI, StatusPass, ""},
		{"selected redacted request_uri", redactedExchange, selectURI, StatusInconclusive, "redacted"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := URISyntaxValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestOID4VCIRequestValuesEqualValidator(t *testing.T) {
	params := map[string]any{
		"source": map[string]any{"request": "/par", "path": "response.body.request_uri"},
		"target": map[string]any{"request": "/authorize", "path": "query.request_uri"},
	}
	withTargetAt := func(at string, target any) map[string]any {
		authorize := testAuthorizeRequest(at)
		authorize["query"].(map[string]any)["request_uri"] = target
		return testExchange([]any{wsPARWithResponse(wsAPRequestURI), authorize})
	}
	equal := testExchange([]any{
		wsPARWithResponse(wsAPRequestURI),
		testAuthorizeRequest("2026-10-22T07:38:24.600Z"),
	})

	for _, tt := range []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{"equal request_uri", equal, params, StatusPass, ""},
		{"differing request_uri", withTargetAt("2026-10-22T07:38:24.600Z", "urn:ietf:params:oauth:request_uri:other"), params, StatusFail, "but response.body.request_uri of the preceding"},
		{"target before source", withTargetAt("2026-10-22T07:37:00.000Z", wsAPRequestURI), params, StatusFail, "follows no request"},
		{"redacted source", testExchange([]any{
			wsPARWithResponse(redacted(true)),
			testAuthorizeRequest("2026-10-22T07:38:24.600Z"),
		}), params, StatusInconclusive, "redacted"},
		{"redacted target", withTargetAt("2026-10-22T07:38:24.600Z", redacted(true)), params, StatusInconclusive, "redacted"},
		{"no Authorization Request", testExchange([]any{wsPARWithResponse(wsAPRequestURI)}), params, StatusFail, "0 ending in /authorize"},
		{"no PAR response member", testExchange([]any{
			wsPARWithResponse(nil),
			testAuthorizeRequest("2026-10-22T07:38:24.600Z"),
		}), params, StatusFail, "records no response.body.request_uri"},
		{"missing source params", equal, map[string]any{"target": params["target"]}, StatusError, "source and target need request and path"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := OID4VCIRequestValuesEqualValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestOID4VCIIssuerResponseValidator(t *testing.T) {
	responded := func(mutate func(par map[string]any)) map[string]any {
		par := wsPARWithResponse(wsAPRequestURI)
		if mutate != nil {
			mutate(par)
		}
		return testExchange([]any{par})
	}
	params := map[string]any{
		"request":          "/par",
		"status":           201,
		"media_type":       "application/json",
		"required_members": []any{"request_uri"},
		"members_equal":    map[string]any{"request_uri": wsAPRequestURI},
	}
	withParams := func(extra map[string]any) map[string]any {
		out := map[string]any{}
		for name, value := range params {
			out[name] = value
		}
		for name, value := range extra {
			out[name] = value
		}
		return out
	}

	for _, tt := range []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{"expected response", responded(nil), params, StatusPass, ""},
		{"other status", responded(func(par map[string]any) {
			par["response"].(map[string]any)["status"] = float64(200)
		}), params, StatusBlocked, "has status 200"},
		{"other media type", responded(func(par map[string]any) {
			par["response"].(map[string]any)["content_type"] = "text/plain"
		}), params, StatusBlocked, "Content-Type"},
		{"missing required member", responded(func(par map[string]any) {
			par["response"].(map[string]any)["body"] = map[string]any{}
		}), params, StatusBlocked, "has no request_uri"},
		{"forbidden member present", responded(nil), withParams(map[string]any{
			"absent_members": []any{"request_uri"},
		}), StatusBlocked, "carries request_uri"},
		{"member with another value", responded(nil), withParams(map[string]any{
			"members_equal": map[string]any{"request_uri": "urn:other"},
		}), StatusBlocked, "expected \"urn:other\""},
		{"no response recorded", testExchange([]any{testPAR()}), params, StatusBlocked, "was not recorded"},
		{"no request", testExchange(nil), params, StatusBlocked, "sent no request"},
		{"missing request param", responded(nil), map[string]any{}, StatusError, "request param is required"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := OID4VCIIssuerResponseValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestOID4VCINonceFromNonceEndpointValidator(t *testing.T) {
	proofs := func(at string, members map[string]any) map[string]any {
		out := map[string]any{"request_id": "req-1", "proofs": float64(1)}
		if at != "" {
			out["at"] = at
		}
		for name, value := range members {
			out[name] = value
		}
		return out
	}
	proofWithNonce := func(nonce string) map[string]any {
		return map[string]any{
			"compact": "a.b.c",
			"payload": map[string]any{"nonce": nonce},
		}
	}
	before := wsNonceRequest("2026-10-22T07:38:20.000Z", "n-1")
	after := wsNonceRequest("2026-10-22T07:38:35.000Z", "n-1")

	for _, tt := range []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{
			name: "attestation proof repeats the nonce",
			value: wsNonceExchange([]any{before}, proofs(wsAPAnchor, map[string]any{
				"attestation": []any{proofWithNonce("n-1")},
			})),
			params: map[string]any{"proof_type": "attestation"},
			status: StatusPass,
		},
		{
			name: "jwt proof repeats the nonce",
			value: wsNonceExchange([]any{before}, proofs(wsAPAnchor, map[string]any{
				"jwt": []any{proofWithNonce("n-1")},
			})),
			params: map[string]any{"proof_type": "jwt"},
			status: StatusPass,
		},
		{
			name: "nonce response at the Credential Request time counts",
			value: wsNonceExchange([]any{wsNonceRequest(wsAPAnchor, "n-1")}, proofs(wsAPAnchor, map[string]any{
				"jwt": []any{proofWithNonce("n-1")},
			})),
			params: map[string]any{"proof_type": "jwt"},
			status: StatusPass,
		},
		{
			name: "jwt proof carries another nonce",
			value: wsNonceExchange([]any{before}, proofs(wsAPAnchor, map[string]any{
				"jwt": []any{proofWithNonce("n-2")},
			})),
			params:  map[string]any{"proof_type": "jwt"},
			status:  StatusFail,
			message: "is not the c_nonce",
		},
		{
			name: "no jwt proof at all",
			value: wsNonceExchange([]any{before}, proofs(wsAPAnchor, map[string]any{
				"attestation": []any{proofWithNonce("n-1")},
			})),
			params:  map[string]any{"proof_type": "jwt"},
			status:  StatusNotApplicable,
			message: "sent no jwt proof",
		},
		{
			name: "empty jwt proof list",
			value: wsNonceExchange([]any{before}, proofs(wsAPAnchor, map[string]any{
				"jwt": []any{},
			})),
			params: map[string]any{"proof_type": "jwt"},
			status: StatusNotApplicable,
		},
		{
			name:    "no credential_request_proofs",
			value:   wsNonceExchange([]any{before}, nil),
			params:  map[string]any{"proof_type": "jwt"},
			status:  StatusFail,
			message: "records no raw.credential_request_proofs",
		},
		{
			name: "nonce response after the Credential Request does not count",
			value: wsNonceExchange([]any{after}, proofs(wsAPAnchor, map[string]any{
				"jwt": []any{proofWithNonce("n-1")},
			})),
			params:  map[string]any{"proof_type": "jwt"},
			status:  StatusFail,
			message: "no Nonce Endpoint response preceding the Credential Request",
		},
		{
			name: "proofs without a time",
			value: wsNonceExchange([]any{before}, proofs("", map[string]any{
				"jwt": []any{proofWithNonce("n-1")},
			})),
			params:  map[string]any{"proof_type": "jwt"},
			status:  StatusFail,
			message: "record no time",
		},
		{
			name: "no Nonce Endpoint response",
			value: wsNonceExchange(nil, proofs(wsAPAnchor, map[string]any{
				"jwt": []any{proofWithNonce("n-1")},
			})),
			params:  map[string]any{"proof_type": "jwt"},
			status:  StatusFail,
			message: "no Nonce Endpoint response preceding the Credential Request",
		},
		{
			name:    "unknown proof type",
			value:   wsNonceExchange([]any{before}, nil),
			params:  map[string]any{"proof_type": "x509"},
			status:  StatusError,
			message: "proof_type must be jwt or attestation",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := OID4VCINonceFromNonceEndpointValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestOID4VCINoRequestAfterValidator(t *testing.T) {
	later := testCapturedRequest(
		"2026-10-22T07:38:35.000Z",
		"POST",
		"/issuers/eu-pid-device-bound/token",
		nil,
		map[string]any{},
		nil,
	)
	withUnattributed := func(windowTo string, requests ...map[string]any) map[string]any {
		exchange := wsObservedExchange(testSessionRequests(), windowTo)
		listing := exchange["session_requests"].(map[string]any)
		entries := make([]any, 0, len(requests))
		for _, request := range requests {
			entries = append(entries, request)
		}
		listing["unattributed_requests"] = entries
		return exchange
	}

	for _, tt := range []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{
			name:   "no later request",
			value:  wsObservedExchange(testSessionRequests(), "2026-10-22T07:38:40Z"),
			params: map[string]any{"after": "/credential"},
			status: StatusPass,
		},
		{
			name:    "later attributed request",
			value:   wsObservedExchange(append(testSessionRequests(), later), "2026-10-22T07:38:40Z"),
			params:  map[string]any{"after": "/credential"},
			status:  StatusFail,
			message: "/issuers/eu-pid-device-bound/token",
		},
		{
			name:   "later request outside the suffixes",
			value:  wsObservedExchange(append(testSessionRequests(), later), "2026-10-22T07:38:40Z"),
			params: map[string]any{"after": "/credential", "path_suffixes": []any{"/authorize"}},
			status: StatusPass,
		},
		{
			name:    "later request matching a suffix",
			value:   wsObservedExchange(append(testSessionRequests(), later), "2026-10-22T07:38:40Z"),
			params:  map[string]any{"after": "/credential", "path_suffixes": []any{"/token"}},
			status:  StatusFail,
			message: "/issuers/eu-pid-device-bound/token",
		},
		{
			name: "retried anchor request",
			value: wsObservedExchange(
				append(testSessionRequests(), testCapturedRequest(
					"2026-10-22T07:38:35.000Z", "POST", "/issuers/eu-pid-device-bound/credential", nil, map[string]any{}, nil,
				)),
				"2026-10-22T07:38:40Z",
			),
			params:  map[string]any{"after": "/credential"},
			status:  StatusFail,
			message: "the Wallet sent POST /issuers/eu-pid-device-bound/credential",
		},
		{
			name: "unattributed authorize matched by path prefix",
			value: withUnattributed(
				"2026-10-22T07:38:40Z",
				wsUnattributedRequest("2026-10-22T07:38:30Z", "/issuers/eu-pid-device-bound/authorize", "capture.test"),
			),
			params: map[string]any{"after": "/credential", "unattributed": map[string]any{
				"path_prefixes": []any{"/issuers/eu-pid-device-bound/"},
			}},
			status:  StatusInconclusive,
			message: "/issuers/eu-pid-device-bound/authorize",
		},
		{
			name: "later attributed request in a short window",
			value: wsObservedExchange(
				append(testSessionRequests(), testCapturedRequest(
					"2026-10-22T07:38:30.000Z", "POST", "/issuers/eu-pid-device-bound/token", nil, map[string]any{}, nil,
				)),
				"2026-10-22T07:38:32Z",
			),
			params:  map[string]any{"after": "/credential"},
			status:  StatusFail,
			message: "/issuers/eu-pid-device-bound/token",
		},
		{
			name: "unattributed authorize on the issuer host",
			value: withUnattributed(
				"2026-10-22T07:38:40Z",
				wsUnattributedRequest("2026-10-22T07:38:30Z", "/issuers/eu-pid-device-bound/authorize", "capture.test"),
			),
			params: map[string]any{"after": "/credential", "unattributed": map[string]any{
				"path_suffixes": []any{"/authorize"},
			}},
			status:  StatusInconclusive,
			message: "/issuers/eu-pid-device-bound/authorize",
		},
		{
			name: "unattributed authorize on another host",
			value: withUnattributed(
				"2026-10-22T07:38:40Z",
				wsUnattributedRequest("2026-10-22T07:38:30Z", "/issuers/eu-pid-device-bound/authorize", "other.test"),
			),
			params: map[string]any{"after": "/credential", "unattributed": map[string]any{
				"path_suffixes": []any{"/authorize"},
			}},
			status: StatusPass,
		},
		{
			name: "unattributed authorize before the anchor",
			value: withUnattributed(
				"2026-10-22T07:38:40Z",
				wsUnattributedRequest("2026-10-22T07:38:20Z", "/issuers/eu-pid-device-bound/authorize", "capture.test"),
			),
			params: map[string]any{"after": "/credential", "unattributed": map[string]any{
				"path_suffixes": []any{"/authorize"},
			}},
			status: StatusPass,
		},
		{
			name: "unattributed authorize of another issuer",
			value: withUnattributed(
				"2026-10-22T07:38:40Z",
				wsUnattributedRequest("2026-10-22T07:38:30Z", "/issuers/other/authorize", "capture.test"),
			),
			params: map[string]any{"after": "/credential", "unattributed": map[string]any{
				"path_suffixes": []any{"/authorize"},
			}},
			status: StatusPass,
		},
		{
			name: "unattributed chained authorization-server request",
			value: withUnattributed(
				"2026-10-22T07:38:40Z",
				wsUnattributedRequest("2026-10-22T07:38:30Z", "/authorization-servers/eu-pid-device-bound/authorize", "capture.test"),
			),
			params: map[string]any{"after": "/credential", "unattributed": map[string]any{
				"path_suffixes": []any{"/authorize"},
			}},
			status: StatusPass,
		},
		{
			name:    "window shorter than five seconds",
			value:   wsObservedExchange(testSessionRequests(), "2026-10-22T07:38:33Z"),
			params:  map[string]any{"after": "/credential"},
			status:  StatusInconclusive,
			message: "less than the 5 s",
		},
		{
			name:    "no session_requests window",
			value:   wsObservedExchange(testSessionRequests(), ""),
			params:  map[string]any{"after": "/credential"},
			status:  StatusInconclusive,
			message: "no session_requests window",
		},
		{
			name:    "observe_seconds longer than the window",
			value:   wsObservedExchange(testSessionRequests(), "2026-10-22T07:38:40Z"),
			params:  map[string]any{"after": "/credential", "observe_seconds": 15},
			status:  StatusInconclusive,
			message: "less than the 15 s",
		},
		{
			name:   "observe_seconds shorter than the window",
			value:  wsObservedExchange(testSessionRequests(), "2026-10-22T07:38:40Z"),
			params: map[string]any{"after": "/credential", "observe_seconds": 1},
			status: StatusPass,
		},
		{
			name:    "no anchor request",
			value:   wsObservedExchange(testSessionRequests(), "2026-10-22T07:38:40Z"),
			params:  map[string]any{"after": "/token"},
			status:  StatusBlocked,
			message: "sent no request ending in /token",
		},
		{
			name:    "missing after param",
			value:   wsObservedExchange(testSessionRequests(), "2026-10-22T07:38:40Z"),
			params:  map[string]any{},
			status:  StatusError,
			message: "after param is required",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := OID4VCINoRequestAfterValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestOID4VCIOfferEngagementValidator(t *testing.T) {
	engagementWindow := func(exchange map[string]any) {
		exchange["session_requests"] = map[string]any{
			"window": map[string]any{
				"from": "2026-10-22T07:37:00Z",
				"to":   "2026-10-22T07:38:40Z",
			},
			"unattributed_requests": []any{},
		}
	}
	notInvokedAt := func(at string) func(map[string]any) {
		return func(exchange map[string]any) {
			exchange["screenshots"] = []any{wsAPNotInvokedShot}
			exchange["capture_session"].(map[string]any)["events"] = []any{
				map[string]any{"at": at, "type": defaultEngagementPageEvent},
			}
		}
	}
	withParams := func(mutate func(map[string]any)) map[string]any {
		params := wsOfferParams()
		if mutate != nil {
			mutate(params)
		}
		return params
	}

	for _, tt := range []struct {
		name    string
		mutate  func(map[string]any)
		params  map[string]any
		status  Status
		message string
	}{
		{
			name:   "invoked and sent the PAR",
			params: withParams(func(params map[string]any) { params["expect_requests"] = []any{"/par"} }),
			status: StatusPass,
		},
		{
			name:    "invoked but sent no expected request",
			params:  withParams(func(params map[string]any) { params["expect_requests"] = []any{"/token"} }),
			status:  StatusFail,
			message: "no request ending in /token",
		},
		{
			name:   "forbid_session_requests sees the PAR",
			mutate: engagementWindow,
			params: withParams(func(params map[string]any) {
				params["forbid_session_requests"] = true
			}),
			status:  StatusFail,
			message: "/par",
		},
		{
			name: "forbid_session_requests sees the PAR in a short window",
			mutate: func(exchange map[string]any) {
				raw := exchange["capture_session"].(map[string]any)["raw"].(map[string]any)
				raw["oid4vci_requests"] = append(raw["oid4vci_requests"].([]any), testCapturedRequest(
					"2026-10-22T07:37:41.000Z", "POST", "/issuers/eu-pid-device-bound/token", nil, map[string]any{}, nil,
				))
				exchange["session_requests"] = map[string]any{
					"window": map[string]any{
						"from": "2026-10-22T07:37:00Z",
						"to":   "2026-10-22T07:37:43Z",
					},
					"unattributed_requests": []any{},
				}
			},
			params: withParams(func(params map[string]any) {
				params["forbid_session_requests"] = true
			}),
			status:  StatusFail,
			message: "/issuers/eu-pid-device-bound/token",
		},
		{
			name: "forbid_unattributed sees the metadata fetch",
			mutate: func(exchange map[string]any) {
				engagementWindow(exchange)
				exchange["session_requests"].(map[string]any)["unattributed_requests"] = []any{
					wsUnattributedRequest(
						"2026-10-22T07:37:50Z",
						"/issuers/eu-pid-device-bound/.well-known/openid-credential-issuer",
						"capture.test",
					),
				}
			},
			params: withParams(func(params map[string]any) {
				params["forbid_unattributed"] = map[string]any{
					"path_suffixes": []any{"openid-credential-issuer"},
				}
			}),
			status:  StatusInconclusive,
			message: "openid-credential-issuer",
		},
		{
			name: "another host's fetch is ignored",
			mutate: func(exchange map[string]any) {
				engagementWindow(exchange)
				exchange["session_requests"].(map[string]any)["unattributed_requests"] = []any{
					wsUnattributedRequest(
						"2026-10-22T07:37:50Z",
						"/issuers/eu-pid-device-bound/.well-known/openid-credential-issuer",
						"other.test",
					),
				}
			},
			params: withParams(func(params map[string]any) {
				params["forbid_unattributed"] = map[string]any{
					"path_suffixes": []any{"openid-credential-issuer"},
				}
			}),
			status: StatusPass,
		},
		{
			name:    "not invoked fails",
			mutate:  notInvokedAt("2026-10-22T07:40:00.000Z"),
			params:  withParams(nil),
			status:  StatusFail,
			message: "not invoked",
		},
		{
			name:   "not invoked is not applicable",
			mutate: notInvokedAt("2026-10-22T07:40:00.000Z"),
			params: withParams(func(params map[string]any) {
				params["when_not_invoked"] = "not_applicable"
			}),
			status: StatusNotApplicable,
		},
		{
			name:   "not invoked blocks the test",
			mutate: notInvokedAt("2026-10-22T07:40:00.000Z"),
			params: withParams(func(params map[string]any) {
				params["when_not_invoked"] = "blocked"
			}),
			status: StatusBlocked,
		},
		{
			name:    "page served yet not invoked while the Wallet sent a request",
			mutate:  notInvokedAt("2026-10-22T07:37:40.000Z"),
			params:  withParams(nil),
			status:  StatusInconclusive,
			message: "yet the Wallet sent",
		},
		{
			name: "both invocation screenshots",
			mutate: func(exchange map[string]any) {
				exchange["screenshots"] = []any{wsAPInvokedShot, wsAPNotInvokedShot}
			},
			params: withParams(nil),
			status: StatusInconclusive,
		},
		{
			name: "neither invocation screenshot",
			mutate: func(exchange map[string]any) {
				exchange["screenshots"] = []any{
					"https://capture.test/files/engage_offer_offer_link_presented_ab12cd34ef.png",
				}
			},
			params: withParams(nil),
			status: StatusInconclusive,
		},
		{
			name: "deeplink of another scheme",
			mutate: func(exchange map[string]any) {
				exchange["session_creation"] = map[string]any{"deeplink": "openid4vp://?request_uri=x"}
			},
			params:  withParams(nil),
			status:  StatusBlocked,
			message: "does not use eudi-openid4ci",
		},
		{
			name: "delivered offer lacks a required member",
			params: withParams(func(params map[string]any) {
				params["offer_present"] = []any{"credential_issuer", "credential_configuration_ids"}
			}),
			status:  StatusBlocked,
			message: "no credential_configuration_ids",
		},
		{
			name: "delivered offer carries a forbidden member",
			params: withParams(func(params map[string]any) {
				params["offer_absent"] = []any{"grants"}
			}),
			status:  StatusBlocked,
			message: "carries grants",
		},
		{
			name: "offer delivered by reference",
			mutate: func(exchange map[string]any) {
				exchange["session_creation"] = map[string]any{
					"deeplink": "eudi-openid4ci://?credential_offer_uri=https%3A%2F%2Fcapture.test%2Foffer",
				}
			},
			params:  withParams(nil),
			status:  StatusBlocked,
			message: "no Credential Offer by value",
		},
		{
			name: "no engagement page event",
			mutate: func(exchange map[string]any) {
				delete(exchange["capture_session"].(map[string]any), "events")
			},
			params:  withParams(nil),
			status:  StatusBlocked,
			message: "engagement page was never served",
		},
		{
			name:   "invalid when_not_invoked",
			params: withParams(func(params map[string]any) { params["when_not_invoked"] = "pass" }),
			status: StatusError,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			exchange := wsOfferExchange()
			if tt.mutate != nil {
				tt.mutate(exchange)
			}
			result := OID4VCIOfferEngagementValidator{}.Validate(
				context.Background(),
				Input{Value: exchange, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}

func TestOID4VCICredentialRequestBodyValidator(t *testing.T) {
	jsonBody := func(data string) map[string]any {
		return map[string]any{
			"content_type": "application/json",
			"base64":       base64.StdEncoding.EncodeToString([]byte(data)),
		}
	}
	for _, tt := range []struct {
		name    string
		value   any
		status  Status
		message string
	}{
		{"application/json passes every Shared_JSON case", wsCredentialBodyExchange(jsonBody(`{"credential_configuration_id":"urn:x"}`), false), StatusPass, ""},
		{"duplicate member", wsCredentialBodyExchange(jsonBody(`{"a":1,"a":2}`), false), StatusFail, "Shared_JSON_02"},
		{"invalid JSON", wsCredentialBodyExchange(jsonBody(`{"a":`), false), StatusFail, "Shared_JSON_01"},
		{
			name: "text body with a media type parameter",
			value: wsCredentialBodyExchange(map[string]any{
				"content_type": "application/json; charset=utf-8",
				"text":         `{"a":1}`,
			}, false),
			status: StatusPass,
		},
		{
			name: "jwt with advertised encryption",
			value: wsCredentialBodyExchange(map[string]any{
				"content_type": "application/jwt",
				"text":         "a.b.c.d.e",
			}, true),
			status:  StatusNotApplicable,
			message: "encrypted the Credential Request",
		},
		{
			name: "jwt without advertised encryption",
			value: wsCredentialBodyExchange(map[string]any{
				"content_type": "application/jwt",
				"text":         "a.b.c.d.e",
			}, false),
			status:  StatusFail,
			message: "advertises no credential_request_encryption",
		},
		{"no raw_body", wsCredentialBodyExchange(nil, false), StatusInconclusive, "raw_body"},
		{
			name: "other media type",
			value: wsCredentialBodyExchange(map[string]any{
				"content_type": "text/plain",
				"text":         "{}",
			}, false),
			status:  StatusFail,
			message: "expected application/json",
		},
		{
			name: "unparsable Content-Type",
			value: wsCredentialBodyExchange(map[string]any{
				"content_type": "application/json; charset",
				"text":         "{}",
			}, false),
			status:  StatusFail,
			message: "does not parse",
		},
		{
			name: "base64 does not decode",
			value: wsCredentialBodyExchange(map[string]any{
				"content_type": "application/json",
				"base64":       "!!!",
			}, false),
			status:  StatusError,
			message: "base64 does not decode",
		},
		{"no Credential Request", testExchange(nil), StatusFail, "no Credential Request"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := OID4VCICredentialRequestBodyValidator{}.Validate(
				context.Background(),
				Input{Value: tt.value},
			)
			require.Equal(t, tt.status, result.Status, result.Message)
			require.Contains(t, result.Message, tt.message)
		})
	}
}
