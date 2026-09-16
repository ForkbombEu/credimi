// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func testCallbackEvidence(status any, contentType, cacheControl, body string) map[string]any {
	headers := map[string]any{}
	if contentType != "" {
		headers["Content-Type"] = contentType
	}
	if cacheControl != "" {
		headers["Cache-Control"] = cacheControl
	}
	return map[string]any{
		"verifier_http": map[string]any{
			"status":  status,
			"headers": headers,
			"body":    body,
		},
		"configured_redirect_uri": "https://verifier.eudiw.dev/?response_code=configured",
	}
}

func TestOID4VPResponseEndpointCallbackValidator(t *testing.T) {
	validator := OID4VPResponseEndpointCallbackValidator{}
	redirectBody := `{"redirect_uri":"https://verifier.eudiw.dev/?response_code=refreshed"}`

	for _, tt := range []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{
			name:   "status matches",
			value:  testCallbackEvidence(200, "application/json", "no-store", redirectBody),
			params: map[string]any{"status": 200},
			status: StatusPass,
		},
		{
			name:   "status mismatch",
			value:  testCallbackEvidence(400, "application/json", "no-store", redirectBody),
			params: map[string]any{"status": 200},
			status: StatusFail,
		},
		{
			name:   "status missing",
			value:  testCallbackEvidence(nil, "application/json", "no-store", redirectBody),
			params: map[string]any{"status": 200},
			status: StatusFail,
		},
		{
			name:   "media type matches with charset",
			value:  testCallbackEvidence(200, "application/json; charset=utf-8", "no-store", redirectBody),
			params: map[string]any{"media_type": "application/json"},
			status: StatusPass,
		},
		{
			name:   "media type mismatch",
			value:  testCallbackEvidence(200, "text/html", "no-store", redirectBody),
			params: map[string]any{"media_type": "application/json"},
			status: StatusFail,
		},
		{
			name:   "no-store present",
			value:  testCallbackEvidence(200, "application/json", "no-store", redirectBody),
			params: map[string]any{"require_no_store": true},
			status: StatusPass,
		},
		{
			name:   "no-store within directive list",
			value:  testCallbackEvidence(200, "application/json", "no-cache, no-store", redirectBody),
			params: map[string]any{"require_no_store": true},
			status: StatusPass,
		},
		{
			name:   "no-store absent",
			value:  testCallbackEvidence(200, "application/json", "public, max-age=60", redirectBody),
			params: map[string]any{"require_no_store": true},
			status: StatusFail,
		},
		{
			name:   "no-store header missing",
			value:  testCallbackEvidence(200, "application/json", "", redirectBody),
			params: map[string]any{"require_no_store": true},
			status: StatusFail,
		},
		{
			name:   "redirect uri targets configured location",
			value:  testCallbackEvidence(200, "application/json", "no-store", redirectBody),
			params: map[string]any{"match_configured_redirect_uri": true},
			status: StatusPass,
		},
		{
			name: "redirect uri with different path",
			value: testCallbackEvidence(
				200,
				"application/json",
				"no-store",
				`{"redirect_uri":"https://verifier.eudiw.dev/other?response_code=refreshed"}`,
			),
			params: map[string]any{"match_configured_redirect_uri": true},
			status: StatusFail,
		},
		{
			name: "redirect uri with different host",
			value: testCallbackEvidence(
				200,
				"application/json",
				"no-store",
				`{"redirect_uri":"https://verifier.example.org/?response_code=refreshed"}`,
			),
			params: map[string]any{"match_configured_redirect_uri": true},
			status: StatusFail,
		},
		{
			name: "redirect uri without https",
			value: testCallbackEvidence(
				200,
				"application/json",
				"no-store",
				`{"redirect_uri":"http://verifier.eudiw.dev/?response_code=refreshed"}`,
			),
			params: map[string]any{"match_configured_redirect_uri": true},
			status: StatusFail,
		},
		{
			name:   "callback body is not JSON",
			value:  testCallbackEvidence(200, "application/json", "no-store", "<html></html>"),
			params: map[string]any{"match_configured_redirect_uri": true},
			status: StatusFail,
		},
		{
			name:   "callback body without redirect_uri",
			value:  testCallbackEvidence(200, "application/json", "no-store", `{"status":"ok"}`),
			params: map[string]any{"match_configured_redirect_uri": true},
			status: StatusFail,
		},
		{
			name: "configured redirect uri missing",
			value: map[string]any{
				"verifier_http": map[string]any{"status": 200, "headers": map[string]any{}, "body": redirectBody},
			},
			params: map[string]any{"match_configured_redirect_uri": true},
			status: StatusFail,
		},
		{
			name:   "verifier http capture missing",
			value:  map[string]any{"configured_redirect_uri": "https://verifier.eudiw.dev/"},
			params: map[string]any{"status": 200},
			status: StatusFail,
		},
		{
			name:   "callback evidence is not an object",
			value:  "not-an-object",
			params: map[string]any{"status": 200},
			status: StatusFail,
		},
		{
			name:   "no check requested",
			value:  testCallbackEvidence(200, "application/json", "no-store", redirectBody),
			params: nil,
			status: StatusError,
		},
		{
			name: "all checks together",
			value: testCallbackEvidence(
				200,
				"application/json; charset=utf-8",
				"no-store",
				redirectBody,
			),
			params: map[string]any{
				"status":                        200,
				"media_type":                    "application/json",
				"require_no_store":              true,
				"match_configured_redirect_uri": true,
			},
			status: StatusPass,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.Validate(context.Background(), Input{Value: tt.value, Params: tt.params})
			require.Equal(t, tt.status, got.Status, got.Message)
		})
	}
}
