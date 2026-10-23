// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// encodeTransactionDataEntry renders one transaction_data entry the way an
// OpenID4VP 1.0 Section 5.1 Request Object carries it: the base64url string of
// a JSON object.
func encodeTransactionDataEntry(t *testing.T, entry map[string]any) string {
	t.Helper()
	encoded, err := json.Marshal(entry)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func transactionDataPayload(entries ...any) map[string]any {
	return map[string]any{
		"dcql_query": map[string]any{
			"credentials": []any{map[string]any{"id": "pid"}},
		},
		"transaction_data": entries,
	}
}

func TestOID4VPTransactionDataCredentialIDsValidator(t *testing.T) {
	tests := []struct {
		name       string
		value      any
		wantStatus Status
	}{
		{
			name: "accepts base64url-encoded entry matching the DCQL query",
			value: compactTestJWT(
				t,
				transactionDataPayload(encodeTransactionDataEntry(t, map[string]any{
					"type":           "payment",
					"data":           "dGVzdA==",
					"credential_ids": []any{"pid"},
				})),
			),
			wantStatus: StatusPass,
		},
		{
			name: "accepts an already-decoded object entry",
			value: compactTestJWT(t, transactionDataPayload(map[string]any{
				"type":           "payment",
				"data":           "dGVzdA==",
				"credential_ids": []any{"pid"},
			})),
			wantStatus: StatusPass,
		},
		{
			name: "rejects unknown credential ID",
			value: compactTestJWT(
				t,
				transactionDataPayload(encodeTransactionDataEntry(t, map[string]any{
					"type":           "payment",
					"data":           "dGVzdA==",
					"credential_ids": []any{"other"},
				})),
			),
			wantStatus: StatusFail,
		},
		{
			name: "rejects non-string credential ID",
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
		{
			name:       "rejects an entry that is not base64url-encoded JSON",
			value:      compactTestJWT(t, transactionDataPayload("not base64url json")),
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
