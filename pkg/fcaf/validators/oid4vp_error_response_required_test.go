// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func capturedWalletResponse(value map[string]any) map[string]any {
	return map[string]any{
		"observed": map[string]any{
			"wallet_response": map[string]any{
				"value":  value,
				"source": "presentation_response",
			},
		},
	}
}

func TestOID4VPErrorResponseRequiredValidator(t *testing.T) {
	validator := OID4VPErrorResponseRequiredValidator{}

	for _, tt := range []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{
			name:   "required error without presentation",
			value:  capturedWalletResponse(map[string]any{"error": "invalid_scope", "error_description": "unknown"}),
			params: map[string]any{"code": "invalid_scope"},
			status: StatusPass,
		},
		{
			name:   "different error code",
			value:  capturedWalletResponse(map[string]any{"error": "invalid_request"}),
			params: map[string]any{"code": "invalid_scope"},
			status: StatusFail,
		},
		{
			name:   "no error at all",
			value:  capturedWalletResponse(map[string]any{}),
			params: map[string]any{"code": "invalid_scope"},
			status: StatusFail,
		},
		{
			name:   "error code plus presentation",
			value:  capturedWalletResponse(map[string]any{"error": "invalid_scope", "vp_token": map[string]any{"pid": []any{"a.b.c"}}}),
			params: map[string]any{"code": "invalid_scope"},
			status: StatusFail,
		},
		{
			name:   "presentation without error",
			value:  capturedWalletResponse(map[string]any{"vp_token": map[string]any{"pid": []any{"a.b.c"}}}),
			params: map[string]any{"code": "invalid_scope"},
			status: StatusFail,
		},
		{
			name:   "unsupported format error",
			value:  capturedWalletResponse(map[string]any{"error": "vp_formats_not_supported"}),
			params: map[string]any{"code": "vp_formats_not_supported"},
			status: StatusPass,
		},
		{
			name: "error recorded only in the raw response",
			value: map[string]any{
				"raw": map[string]any{
					"presentation_response": map[string]any{"error": "invalid_request"},
				},
			},
			params: map[string]any{"code": "invalid_request"},
			status: StatusPass,
		},
		{
			name:   "missing code param",
			value:  capturedWalletResponse(map[string]any{"error": "invalid_scope"}),
			params: map[string]any{},
			status: StatusError,
		},
		{
			name:   "evidence is not an object",
			value:  "not-an-object",
			params: map[string]any{"code": "invalid_scope"},
			status: StatusFail,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.Validate(
				context.Background(),
				Input{Value: tt.value, Params: tt.params},
			)
			require.Equal(t, tt.status, got.Status, got.Message)
		})
	}
}
