// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPResponseMediaTypeValidator(t *testing.T) {
	validator := HTTPResponseMediaTypeValidator{}
	for _, tt := range []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{"case insensitive string header", map[string]any{"headers": map[string]any{"content-type": "application/oauth-authz-req+jwt; charset=utf-8"}}, map[string]any{"media_type": "application/oauth-authz-req+jwt"}, StatusPass},
		{"string array header", map[string]any{"headers": map[string]any{"Content-Type": []any{"application/oauth-authz-req+jwt"}}}, map[string]any{"media_type": "application/oauth-authz-req+jwt"}, StatusPass},
		{"wrong media type", map[string]any{"headers": map[string]any{"Content-Type": "application/json"}}, map[string]any{"media_type": "application/oauth-authz-req+jwt"}, StatusFail},
		{"missing header", map[string]any{"headers": map[string]any{}}, map[string]any{"media_type": "application/oauth-authz-req+jwt"}, StatusFail},
		{"missing parameter", map[string]any{"headers": map[string]any{}}, nil, StatusError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(
				t,
				tt.status,
				validator.Validate(
					context.Background(),
					Input{Value: tt.value, Params: tt.params},
				).Status,
			)
		})
	}
}

func TestOID4VPPresentationResponseHTTPValidator(t *testing.T) {
	validator := OID4VPPresentationResponseHTTPValidator{}
	valid := testPresentationSession(
		"application/x-www-form-urlencoded; charset=utf-8",
		"response=compact",
	)
	invalidUTF8 := testPresentationSession("application/x-www-form-urlencoded", "response=%FF")
	for _, tt := range []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{"method", valid, map[string]any{"method": "POST"}, StatusPass},
		{"media type and response", valid, map[string]any{"media_type": "application/x-www-form-urlencoded", "require_response_parameter": true}, StatusPass},
		{"response only", valid, map[string]any{"response_only": true, "require_response_parameter": true}, StatusPass},
		{"extra form key", testPresentationSession("application/x-www-form-urlencoded", "response=compact&state=x"), map[string]any{"response_only": true}, StatusFail},
		{"duplicate response", testPresentationSession("application/x-www-form-urlencoded", "response=a&response=b"), map[string]any{"require_response_parameter": true}, StatusFail},
		{"wrong method", valid, map[string]any{"method": "GET"}, StatusFail},
		{"invalid UTF8", invalidUTF8, map[string]any{"form_utf8": true}, StatusFail},
		{"missing raw capture", map[string]any{"raw": map[string]any{}}, map[string]any{"method": "POST"}, StatusFail},
		{"no check", valid, nil, StatusError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(
				t,
				tt.status,
				validator.Validate(
					context.Background(),
					Input{Value: tt.value, Params: tt.params},
				).Status,
			)
		})
	}
}

func TestOID4VPResponseEncryptionValidator(t *testing.T) {
	validator := OID4VPResponseEncryptionValidator{}
	jwks := map[string]any{"keys": []any{map[string]any{"kid": "generated-kid", "kty": "EC"}}}
	request := testSignedRequest(t, map[string]any{"jwks": jwks})
	jwe := testCompactJWE(t, map[string]any{"kid": "generated-kid", "enc": "A128GCM"})
	valid := map[string]any{
		"request_object":             request,
		"presentation_response_http": map[string]any{"body": "response=" + jwe},
	}
	replacementRequest := testSignedRequest(
		t,
		map[string]any{"jwks": jwks, "authorization_encrypted_response_enc": "A256GCM"},
	)
	replacement := map[string]any{
		"request_object": replacementRequest,
		"presentation_response_http": map[string]any{
			"body": "response=" + testCompactJWE(
				t,
				map[string]any{"kid": "generated-kid", "enc": "A256GCM"},
			),
		},
		"generated_jwks": jwks,
	}
	for _, tt := range []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{"metadata kid", valid, map[string]any{"match_metadata_kid": true}, StatusPass},
		{"default encryption", valid, map[string]any{"expected_enc": "A128GCM", "metadata_enc_absent": true}, StatusPass},
		{"replacement metadata", replacement, map[string]any{"expected_enc": "A256GCM", "metadata_enc": "A256GCM", "preserve_generated_jwks": true}, StatusPass},
		{"unknown kid", map[string]any{"request_object": request, "presentation_response_http": map[string]any{"body": "response=" + testCompactJWE(t, map[string]any{"kid": "unknown", "enc": "A128GCM"})}}, map[string]any{"match_metadata_kid": true}, StatusFail},
		{"wrong encryption", valid, map[string]any{"expected_enc": "A256GCM"}, StatusFail},
		{"explicit encryption when absent", valid, map[string]any{"metadata_enc": "A256GCM"}, StatusFail},
		{"mismatched generated JWK", map[string]any{"request_object": replacementRequest, "presentation_response_http": replacement["presentation_response_http"], "generated_jwks": map[string]any{"keys": []any{}}}, map[string]any{"preserve_generated_jwks": true}, StatusFail},
		{"missing raw evidence", map[string]any{"request_object": request}, map[string]any{"expected_enc": "A128GCM"}, StatusFail},
		{"non compact response", map[string]any{"request_object": request, "presentation_response_http": map[string]any{"body": "response=not-a-jwe"}}, map[string]any{"expected_enc": "A128GCM"}, StatusFail},
		{"malformed protected header", map[string]any{"request_object": request, "presentation_response_http": map[string]any{"body": "response=%%%.a.b.c.d"}}, map[string]any{"expected_enc": "A128GCM"}, StatusFail},
		{"contradictory metadata checks", valid, map[string]any{"metadata_enc": "A128GCM", "metadata_enc_absent": true}, StatusError},
		{"no check", valid, nil, StatusError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(
				t,
				tt.status,
				validator.Validate(
					context.Background(),
					Input{Value: tt.value, Params: tt.params},
				).Status,
			)
		})
	}
}

func testPresentationSession(contentType, body string) map[string]any {
	return map[string]any{
		"raw": map[string]any{
			"presentation_response_http": map[string]any{
				"method":  "POST",
				"headers": map[string]any{"Content-Type": contentType},
				"body":    body,
			},
		},
	}
}

func testSignedRequest(t *testing.T, metadata map[string]any) string {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"client_metadata": metadata})
	require.NoError(t, err)
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}
