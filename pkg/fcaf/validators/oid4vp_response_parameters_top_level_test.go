// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPResponseParametersTopLevelValidator(t *testing.T) {
	vpToken := map[string]any{"pid": []any{"eyJhbGciOiJFUzI1NiJ9.e30.sig~"}}

	tests := []struct {
		name       string
		value      any
		required   []string
		wantStatus Status
	}{
		{
			name:       "response parameters at the top level",
			value:      map[string]any{"vp_token": vpToken, "state": "abc"},
			wantStatus: StatusPass,
		},
		{
			name:       "whole response wrapped in a sub-object",
			value:      map[string]any{"response": map[string]any{"vp_token": vpToken}},
			wantStatus: StatusFail,
		},
		{
			name: "state nested below a non-response member",
			value: map[string]any{
				"vp_token": vpToken,
				"envelope": map[string]any{"state": "abc"},
			},
			wantStatus: StatusFail,
		},
		{
			name:       "credential query id shadowing a response parameter name is not nesting",
			value:      map[string]any{"vp_token": map[string]any{"state": []any{"ey.e30.sig~"}}},
			wantStatus: StatusPass,
		},
		{
			name:       "explicitly required member is missing",
			value:      map[string]any{"vp_token": vpToken},
			required:   []string{"vp_token", "state"},
			wantStatus: StatusFail,
		},
		{
			name:       "decrypted response is not an object",
			value:      "vp_token=abc",
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPResponseParametersTopLevelValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := map[string]any{}
			if test.required != nil {
				params["required"] = test.required
			}
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: params,
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
