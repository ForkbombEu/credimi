// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPUnsupportedResponseTypeValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		expected   string
		wantStatus Status
	}{
		{
			name: "detailed unsupported response type error",
			value: map[string]any{
				"presentation_request": map[string]any{"response_type": "vp_token id_token"},
				"observed": map[string]any{"wallet_response": map[string]any{
					"value": map[string]any{"error": "unsupported_response_type"},
				}},
			},
			expected:   "vp_token id_token",
			wantStatus: StatusPass,
		},
		{
			name: "unspecified wallet error",
			value: map[string]any{
				"presentation_request": map[string]any{"response_type": "code"},
				"observed": map[string]any{"wallet_response": map[string]any{
					"value": map[string]any{"error": "invalid_request"},
				}},
			},
			expected:   "code",
			wantStatus: StatusPass,
		},
		{
			name: "interaction discontinued without wallet response",
			value: map[string]any{
				"presentation_request": map[string]any{"response_type": "code"},
				"observed":             map[string]any{},
			},
			expected:   "code",
			wantStatus: StatusPass,
		},
		{
			name: "wrong response type",
			value: map[string]any{
				"presentation_request": map[string]any{"response_type": "vp_token"},
				"observed":             map[string]any{},
			},
			expected:   "code",
			wantStatus: StatusFail,
		},
		{
			name: "missing response type",
			value: map[string]any{
				"presentation_request": map[string]any{},
				"observed":             map[string]any{},
			},
			expected:   "code",
			wantStatus: StatusFail,
		},
		{
			name: "credential returned for unsupported response type",
			value: map[string]any{
				"presentation_request": map[string]any{"response_type": "code"},
				"observed": map[string]any{"wallet_response": map[string]any{
					"value": map[string]any{"vp_token": map[string]any{"pid": []any{"credential"}}},
				}},
			},
			expected:   "code",
			wantStatus: StatusFail,
		},
		{
			name: "submitted response without an error",
			value: map[string]any{
				"presentation_request": map[string]any{"response_type": "code"},
				"observed": map[string]any{"wallet_response": map[string]any{
					"value": map[string]any{"state": "returned"},
				}},
			},
			expected:   "code",
			wantStatus: StatusFail,
		},
		{
			name: "non string error",
			value: map[string]any{
				"presentation_request": map[string]any{"response_type": "code"},
				"observed": map[string]any{"wallet_response": map[string]any{
					"value": map[string]any{"error": true},
				}},
			},
			expected:   "code",
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPUnsupportedResponseTypeValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: map[string]any{"expected_response_type": test.expected},
			})

			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
