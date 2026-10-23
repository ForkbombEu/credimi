// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPRequestURIRetrievalValidator(t *testing.T) {
	validator := OID4VPRequestURIRetrievalValidator{}
	validSession := map[string]any{
		"request_uri": "https://capture-wallet.credimi.io/openid4vp/sessions/session-id/request",
		"raw": map[string]any{
			"request_uri_http": map[string]any{
				"method":  "GET",
				"headers": map[string]any{"host": "capture-wallet.credimi.io"},
			},
		},
	}

	tests := []struct {
		name   string
		value  any
		method string
		status Status
	}{
		{name: "matching get retrieval", value: validSession, method: "GET", status: StatusPass},
		{name: "matching post retrieval", value: map[string]any{
			"request_uri": "https://capture-wallet.credimi.io/openid4vp/sessions/session-id/request",
			"raw": map[string]any{"request_uri_http": map[string]any{
				"method":  "POST",
				"headers": map[string]any{"Host": "capture-wallet.credimi.io"},
			}},
		}, method: "POST", status: StatusPass},
		{name: "wrong method", value: validSession, method: "POST", status: StatusFail},
		{name: "wrong host", value: map[string]any{
			"request_uri": "https://capture-wallet.credimi.io/openid4vp/sessions/session-id/request",
			"raw": map[string]any{"request_uri_http": map[string]any{
				"method": "GET", "headers": map[string]any{"host": "example.invalid"},
			}},
		}, method: "GET", status: StatusFail},
		{
			name:   "missing capture",
			value:  map[string]any{"request_uri": "https://capture-wallet.credimi.io/request"},
			method: "GET",
			status: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  tt.value,
				Params: map[string]any{"method": tt.method},
			})
			require.Equal(t, tt.status, result.Status)
		})
	}
}

// TestOID4VPRequestURIRetrievalValidatorPathTarget covers ER3.3 of
// WS_RP_IA_ProtocolFlow__002b/002d: the Wallet requests the path of the
// delivered request_uri, without a query or fragment part.
func TestOID4VPRequestURIRetrievalValidatorPathTarget(t *testing.T) {
	const path = "/openid4vp/sessions/92273070-d264-46da-935f-a2a7d611caec/request"
	session := func(requestURI string, target any) map[string]any {
		retrieval := map[string]any{
			"method":  "POST",
			"headers": map[string]any{"host": "capture-wallet.credimi.io"},
		}
		if target != nil {
			retrieval["target"] = target
		}
		return map[string]any{
			"request_uri": requestURI,
			"raw":         map[string]any{"request_uri_http": retrieval},
		}
	}
	requestURI := "https://capture-wallet.credimi.io" + path

	tests := []struct {
		name    string
		value   map[string]any
		require bool
		status  Status
	}{
		{"exact path", session(requestURI, path), true, StatusPass},
		{
			"query appended by the Wallet",
			session(requestURI, path+"?client_id=x"),
			true,
			StatusFail,
		},
		{"empty query marker", session(requestURI, path+"?"), true, StatusFail},
		{"fragment character sent", session(requestURI, path+"#frag"), true, StatusFail},
		{
			"another path",
			session(requestURI, "/openid4vp/sessions/other/request"),
			true,
			StatusFail,
		},
		{"target not recorded", session(requestURI, nil), true, StatusFail},
		{"target not a string", session(requestURI, 42), true, StatusFail},
		{
			"query in the delivered request_uri is not part of the path",
			session(requestURI+"?session=1", path+"?session=1"),
			true,
			StatusFail,
		},
		{
			"path of a request_uri with a query",
			session(requestURI+"?session=1", path),
			true,
			StatusPass,
		},
		{"target ignored unless required", session(requestURI, path+"?x=1"), false, StatusPass},
	}

	validator := OID4VPRequestURIRetrievalValidator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value: tt.value,
				Params: map[string]any{
					"method":              "POST",
					"require_path_target": tt.require,
				},
			})
			require.Equal(t, tt.status, result.Status, result.Message)
		})
	}
}

func TestOID4VPRequestURIRetrievalValidatorHTTPProperties(t *testing.T) {
	validator := OID4VPRequestURIRetrievalValidator{}
	params := map[string]any{
		"method":     "POST",
		"media_type": "application/x-www-form-urlencoded",
		"accept":     "application/oauth-authz-req+jwt",
		"form_utf8":  true,
	}
	validSession := map[string]any{
		"request_uri": "https://capture-wallet.credimi.io/openid4vp/sessions/session-id/request",
		"raw": map[string]any{"request_uri_http": map[string]any{
			"method": "POST",
			"headers": map[string]any{
				"Host":         "capture-wallet.credimi.io",
				"Content-Type": "application/x-www-form-urlencoded; charset=utf-8",
				"Accept":       "application/oauth-authz-req+jwt",
			},
			"body": "w%C3%A4llet_nonce=%E2%82%AC",
		}},
	}

	tests := []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{name: "valid POST evidence", value: validSession, params: params, status: StatusPass},
		{
			name: "wrong accept header",
			value: map[string]any{
				"request_uri": "https://capture-wallet.credimi.io/openid4vp/sessions/session-id/request",
				"raw": map[string]any{"request_uri_http": map[string]any{
					"method": "POST",
					"headers": map[string]any{
						"Host":         "capture-wallet.credimi.io",
						"Content-Type": "application/x-www-form-urlencoded",
						"Accept":       "application/json",
					},
					"body": "wallet_nonce=value",
				}},
			},
			params: params,
			status: StatusFail,
		},
		{
			name: "invalid UTF-8 form value",
			value: map[string]any{
				"request_uri": "https://capture-wallet.credimi.io/openid4vp/sessions/session-id/request",
				"raw": map[string]any{"request_uri_http": map[string]any{
					"method": "POST",
					"headers": map[string]any{
						"Host":         "capture-wallet.credimi.io",
						"Content-Type": "application/x-www-form-urlencoded",
						"Accept":       "application/oauth-authz-req+jwt",
					},
					"body": "wallet_nonce=%FF",
				}},
			},
			params: params,
			status: StatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, result.Status)
		})
	}
}

func TestOID4VPRequestURINotRetrievedValidator(t *testing.T) {
	validator := OID4VPRequestURINotRetrievedValidator{}

	tests := []struct {
		name   string
		value  any
		status Status
	}{
		{name: "no raw capture or retrieval event", value: map[string]any{
			"events": []any{map[string]any{"type": "presentation_request_created"}},
		}, status: StatusPass},
		{name: "raw capture present", value: map[string]any{
			"raw": map[string]any{"request_uri_http": map[string]any{"method": "GET"}},
		}, status: StatusFail},
		{name: "retrieval event present", value: map[string]any{
			"events": []any{map[string]any{"type": "vp_request_retrieved"}},
		}, status: StatusFail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: tt.value})
			require.Equal(t, tt.status, result.Status)
		})
	}
}
