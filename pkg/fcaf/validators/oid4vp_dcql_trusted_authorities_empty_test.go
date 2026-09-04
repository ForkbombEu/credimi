// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPDCQLTrustedAuthoritiesEmptyValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "accepts an empty trusted authorities array",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id": "pid", "trusted_authorities": []any{},
				}}},
			}),
			wantStatus: StatusPass,
		},
		{
			name: "rejects omitted trusted authorities",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
			}),
			wantStatus: StatusFail,
		},
		{
			name: "rejects a non-empty trusted authorities array",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id": "pid", "trusted_authorities": []any{map[string]any{"type": "aki"}},
				}}},
			}),
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPDCQLTrustedAuthoritiesEmptyValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: test.value})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
