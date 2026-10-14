// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPTransactionDataTypeNonStringValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "accepts a base64url-encoded entry whose type is not a string",
			value: compactTestJWT(
				t,
				transactionDataPayload(encodeTransactionDataEntry(t, map[string]any{
					"type":           1,
					"data":           "dGVzdA==",
					"credential_ids": []any{"pid"},
				})),
			),
			wantStatus: StatusPass,
		},
		{
			name: "accepts an already-decoded object entry whose type is not a string",
			value: compactTestJWT(t, transactionDataPayload(map[string]any{
				"type":           map[string]any{"nested": true},
				"data":           "dGVzdA==",
				"credential_ids": []any{"pid"},
			})),
			wantStatus: StatusPass,
		},
		{
			name:       "rejects an entry that is not base64url-encoded JSON",
			value:      compactTestJWT(t, transactionDataPayload("not base64url json")),
			wantStatus: StatusFail,
		},
		{
			name: "rejects a string type",
			value: compactTestJWT(
				t,
				transactionDataPayload(encodeTransactionDataEntry(t, map[string]any{
					"type":           "payment",
					"data":           "dGVzdA==",
					"credential_ids": []any{"pid"},
				})),
			),
			wantStatus: StatusFail,
		},
		{
			name: "rejects a missing type",
			value: compactTestJWT(
				t,
				transactionDataPayload(encodeTransactionDataEntry(t, map[string]any{
					"data":           "dGVzdA==",
					"credential_ids": []any{"pid"},
				})),
			),
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPTransactionDataTypeNonStringValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: test.value})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
