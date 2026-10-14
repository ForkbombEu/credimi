// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOID4VPTransactionDataCredentialIDsNonArrayValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "accepts a base64url-encoded entry whose credential_ids is not an array",
			value: compactTestJWT(
				t,
				transactionDataPayload(encodeTransactionDataEntry(t, map[string]any{
					"type":           "payment",
					"data":           "dGVzdA==",
					"credential_ids": "pid",
				})),
			),
			wantStatus: StatusPass,
		},
		{
			name: "accepts an already-decoded object entry whose credential_ids is not an array",
			value: compactTestJWT(t, transactionDataPayload(map[string]any{
				"type":           "payment",
				"data":           "dGVzdA==",
				"credential_ids": 1,
			})),
			wantStatus: StatusPass,
		},
		{
			name:       "rejects an entry that is not base64url-encoded JSON",
			value:      compactTestJWT(t, transactionDataPayload("not base64url json")),
			wantStatus: StatusFail,
		},
		{
			name: "rejects an array credential_ids",
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
			name: "rejects a missing credential_ids",
			value: compactTestJWT(
				t,
				transactionDataPayload(encodeTransactionDataEntry(t, map[string]any{
					"type": "payment",
					"data": "dGVzdA==",
				})),
			),
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPTransactionDataCredentialIDsNonArrayValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{Value: test.value})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
