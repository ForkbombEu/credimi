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
		"request_uri": "https://beta-capture-wallet.credimi.io/openid4vp/sessions/session-id/request",
		"raw": map[string]any{
			"request_uri_http": map[string]any{
				"method":  "GET",
				"headers": map[string]any{"host": "beta-capture-wallet.credimi.io"},
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
			"request_uri": "https://beta-capture-wallet.credimi.io/openid4vp/sessions/session-id/request",
			"raw": map[string]any{"request_uri_http": map[string]any{
				"method":  "POST",
				"headers": map[string]any{"Host": "beta-capture-wallet.credimi.io"},
			}},
		}, method: "POST", status: StatusPass},
		{name: "wrong method", value: validSession, method: "POST", status: StatusFail},
		{name: "wrong host", value: map[string]any{
			"request_uri": "https://beta-capture-wallet.credimi.io/openid4vp/sessions/session-id/request",
			"raw": map[string]any{"request_uri_http": map[string]any{
				"method": "GET", "headers": map[string]any{"host": "example.invalid"},
			}},
		}, method: "GET", status: StatusFail},
		{
			name:   "missing capture",
			value:  map[string]any{"request_uri": "https://beta-capture-wallet.credimi.io/request"},
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
