// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPTransactionDataCredentialIDsValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "accepts transaction data matching the DCQL query",
			value: compactTestJWT(t, map[string]any{
				"dcql_query":       map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
				"transaction_data": []any{map[string]any{"type": "payment", "data": "dGVzdA==", "credential_ids": []any{"pid"}}},
			}),
			wantStatus: StatusPass,
		},
		{
			name: "rejects unknown credential ID",
			value: compactTestJWT(t, map[string]any{
				"dcql_query":       map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
				"transaction_data": []any{map[string]any{"type": "payment", "data": "dGVzdA==", "credential_ids": []any{"other"}}},
			}),
			wantStatus: StatusFail,
		},
		{
			name: "rejects non-string credential ID",
			value: compactTestJWT(t, map[string]any{
				"dcql_query":       map[string]any{"credentials": []any{map[string]any{"id": "pid"}}},
				"transaction_data": []any{map[string]any{"type": "payment", "data": "dGVzdA==", "credential_ids": []any{1}}},
			}),
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPTransactionDataCredentialIDsValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: test.value})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
