// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPTransactionDataCredentialIDsMismatchValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "accepts string credential ID absent from DCQL query",
			value: compactTestJWT(t, map[string]any{
				"dcql_query":       map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
				"transaction_data": []any{map[string]any{"type": "payment", "data": "dGVzdA==", "credential_ids": []any{"other"}}},
			}),
			wantStatus: StatusPass,
		},
		{
			name: "rejects matching credential ID",
			value: compactTestJWT(t, map[string]any{
				"dcql_query":       map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
				"transaction_data": []any{map[string]any{"type": "payment", "data": "dGVzdA==", "credential_ids": []any{"pid"}}},
			}),
			wantStatus: StatusFail,
		},
		{
			name: "rejects non-string mismatched credential ID",
			value: compactTestJWT(t, map[string]any{
				"dcql_query":       map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
				"transaction_data": []any{map[string]any{"type": "payment", "data": "dGVzdA==", "credential_ids": []any{1}}},
			}),
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPTransactionDataCredentialIDsMismatchValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: test.value})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
