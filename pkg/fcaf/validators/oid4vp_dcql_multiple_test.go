// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPDCQLMultipleNonBooleanValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "accepts a non-boolean multiple property",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id": "pid", "multiple": "not-a-boolean",
				}}},
			}),
			wantStatus: StatusPass,
		},
		{
			name: "rejects a boolean multiple property",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id": "pid", "multiple": false,
				}}},
			}),
			wantStatus: StatusFail,
		},
		{
			name: "rejects a missing multiple property",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
			}),
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPDCQLMultipleNonBooleanValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: test.value})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
