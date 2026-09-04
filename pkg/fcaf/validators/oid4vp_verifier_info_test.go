// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPVerifierInfoAllCredentialsValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "accepts attestation without credential IDs",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
				"verifier_info": []any{
					map[string]any{"format": "example", "data": map[string]any{}},
				},
			}),
			wantStatus: StatusPass,
		},
		{
			name: "rejects attestation with credential IDs",
			value: compactTestJWT(t, map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
				"verifier_info": []any{
					map[string]any{
						"format":         "example",
						"data":           map[string]any{},
						"credential_ids": []any{"pid"},
					},
				},
			}),
			wantStatus: StatusFail,
		},
		{
			name: "rejects malformed attestation",
			value: compactTestJWT(t, map[string]any{
				"dcql_query":    map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
				"verifier_info": []any{map[string]any{"format": "example"}},
			}),
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPVerifierInfoAllCredentialsValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: test.value})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
