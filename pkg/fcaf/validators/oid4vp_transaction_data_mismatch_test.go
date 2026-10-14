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
			name: "accepts base64url-encoded string credential ID absent from the DCQL query",
			value: compactTestJWT(
				t,
				transactionDataPayload(encodeTransactionDataEntry(t, map[string]any{
					"type":           "payment",
					"data":           "dGVzdA==",
					"credential_ids": []any{"other"},
				})),
			),
			wantStatus: StatusPass,
		},
		{
			name: "accepts an already-decoded object entry with a mismatched credential ID",
			value: compactTestJWT(t, transactionDataPayload(map[string]any{
				"type":           "payment",
				"data":           "dGVzdA==",
				"credential_ids": []any{"other"},
			})),
			wantStatus: StatusPass,
		},
		{
			name: "rejects an entry that is not base64url-encoded JSON",
			value: compactTestJWT(
				t,
				transactionDataPayload("not base64url json"),
			),
			wantStatus: StatusFail,
		},
		{
			name: "rejects matching credential ID",
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
			name: "rejects non-string mismatched credential ID",
			value: compactTestJWT(
				t,
				transactionDataPayload(encodeTransactionDataEntry(t, map[string]any{
					"type":           "payment",
					"data":           "dGVzdA==",
					"credential_ids": []any{1},
				})),
			),
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
