// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPStateEchoValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "returns the request state unchanged",
			value: map[string]any{
				"request_object": compactTestJWT(
					t,
					map[string]any{"state": "lIj5WZg5_X5Y7g-kMW7-KRGOo"},
				),
				"response": map[string]any{
					"state":    "lIj5WZg5_X5Y7g-kMW7-KRGOo",
					"vp_token": map[string]any{},
				},
			},
			wantStatus: StatusPass,
		},
		{
			name: "rejects a changed state",
			value: map[string]any{
				"request_object": compactTestJWT(
					t,
					map[string]any{"state": "lIj5WZg5_X5Y7g-kMW7-KRGOo"},
				),
				"response": map[string]any{
					"state":    "other-state",
					"vp_token": map[string]any{},
				},
			},
			wantStatus: StatusFail,
		},
		{
			name: "rejects a response without state",
			value: map[string]any{
				"request_object": compactTestJWT(
					t,
					map[string]any{"state": "lIj5WZg5_X5Y7g-kMW7-KRGOo"},
				),
				"response": map[string]any{"vp_token": map[string]any{}},
			},
			wantStatus: StatusFail,
		},
		{
			name: "rejects a null state in the response",
			value: map[string]any{
				"request_object": compactTestJWT(
					t,
					map[string]any{"state": "lIj5WZg5_X5Y7g-kMW7-KRGOo"},
				),
				"response": map[string]any{"state": nil},
			},
			wantStatus: StatusFail,
		},
		{
			name: "errors on a request without state",
			value: map[string]any{
				"request_object": compactTestJWT(t, map[string]any{"nonce": "fcaf"}),
				"response":       map[string]any{"state": "lIj5WZg5_X5Y7g-kMW7-KRGOo"},
			},
			wantStatus: StatusError,
		},
		{
			name: "errors on a served request object that is not a compact JWS",
			value: map[string]any{
				"request_object": "not-a-jwt",
				"response":       map[string]any{"state": "lIj5WZg5_X5Y7g-kMW7-KRGOo"},
			},
			wantStatus: StatusError,
		},
		{
			name: "errors on a missing request object",
			value: map[string]any{
				"response": map[string]any{"state": "lIj5WZg5_X5Y7g-kMW7-KRGOo"},
			},
			wantStatus: StatusError,
		},
		{
			name: "rejects a missing decrypted response",
			value: map[string]any{
				"request_object": compactTestJWT(
					t,
					map[string]any{"state": "lIj5WZg5_X5Y7g-kMW7-KRGOo"},
				),
			},
			wantStatus: StatusFail,
		},
		{
			name:       "errors on input that is not an object",
			value:      "lIj5WZg5_X5Y7g-kMW7-KRGOo",
			wantStatus: StatusError,
		},
	}

	validator := OID4VPStateEchoValidator{}
	require.Equal(t, "oid4vp.state_echoed", validator.ID())
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: test.value})

			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
