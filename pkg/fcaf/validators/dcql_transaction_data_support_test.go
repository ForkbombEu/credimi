// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// transactionDataSession builds the evidence shape the transaction_data modes
// read: a Capture verifier session record with the delivered query, Capture's
// checks object, and the captured Wallet response.
func transactionDataSession(checks any, response map[string]any) map[string]any {
	session := map[string]any{
		"session_id":       "11111111-2222-3333-4444-555555555555",
		"status":           "created",
		"request_delivery": "by_reference",
		"checks":           checks,
		"dcql_query": map[string]any{
			"credentials": []any{map[string]any{"id": "pid", "format": "dc+sd-jwt"}},
		},
	}
	if response != nil {
		session["raw"] = map[string]any{"presentation_response_decrypted": response}
	}
	return session
}

func TestTransactionDataSupportNotApplicableMode(t *testing.T) {
	validator := DCQLResponseConstraintsValidator{}
	supportParam := map[string]any{
		"mode": "transaction_data_error_required",
		"not_applicable_when_wallet_supports_transaction_data": true,
	}
	verifiedChecks := map[string]any{"transaction_data_verified": true}
	unverifiedChecks := map[string]any{"transaction_data_verified": false}
	hashesResponse := map[string]any{
		"state":                   "state-1",
		"vp_token":                map[string]any{"pid": []any{"presentation"}},
		"transaction_data_hashes": []any{"hash-1"},
	}
	rejectionResponse := map[string]any{"error": "invalid_transaction_data"}

	tests := []struct {
		name   string
		value  any
		params map[string]any
		status Status
	}{
		{
			// The property the source's applicability turns on: a Wallet that
			// bound transaction_data is not a Wallet the source is about.
			name:   "verified transaction_data support is not applicable",
			value:  transactionDataSession(verifiedChecks, hashesResponse),
			params: supportParam,
			status: StatusNotApplicable,
		},
		{
			// Same evidence without the param: the definition did not claim the
			// source's applicability, so the rejection requirement decides.
			name:   "verified support without the param still requires the error",
			value:  transactionDataSession(verifiedChecks, hashesResponse),
			params: map[string]any{"mode": "transaction_data_error_required"},
			status: StatusFail,
		},
		{
			// transaction_data_hashes alone proves the support even when
			// Capture recorded no judgement (checks object absent).
			name:   "returned transaction_data_hashes are not applicable",
			value:  transactionDataSession(nil, hashesResponse),
			params: supportParam,
			status: StatusNotApplicable,
		},
		{
			name:   "returned transaction_data_hashes without the param still require the error",
			value:  transactionDataSession(nil, hashesResponse),
			params: map[string]any{"mode": "transaction_data_error_required"},
			status: StatusFail,
		},
		{
			// The reference Wallet path: no support, rejection delivered.
			name:   "unverified support keeps the rejection requirement",
			value:  transactionDataSession(unverifiedChecks, rejectionResponse),
			params: supportParam,
			status: StatusPass,
		},
		{
			name:   "unjudged session keeps the rejection requirement",
			value:  transactionDataSession(nil, rejectionResponse),
			params: supportParam,
			status: StatusPass,
		},
		{
			name:   "no response keeps the rejection requirement",
			value:  transactionDataSession(unverifiedChecks, nil),
			params: supportParam,
			status: StatusFail,
		},
		{
			// A non-boolean judgement says nothing about the support, so the
			// rejection the source prescribes is still what decides.
			name: "string judgement is not support",
			value: transactionDataSession(
				map[string]any{"transaction_data_verified": "true"},
				rejectionResponse,
			),
			params: supportParam,
			status: StatusPass,
		},
		{
			name: "transaction_data_error mode accepts the param",
			value: transactionDataSession(
				verifiedChecks,
				hashesResponse,
			),
			params: map[string]any{
				"mode": "transaction_data_error",
				"not_applicable_when_wallet_supports_transaction_data": true,
			},
			status: StatusNotApplicable,
		},
		{
			// Only the transaction_data error modes consult the param; on any
			// other mode it would be silently dropped.
			name:  "other modes reject the param",
			value: transactionDataSession(verifiedChecks, hashesResponse),
			params: map[string]any{
				"mode": "request_rejected",
				"not_applicable_when_wallet_supports_transaction_data": true,
			},
			status: StatusError,
		},
		{
			// A wrong error with no proven support is still a failure: the
			// param only removes the requirement a supporting Wallet refutes.
			name:   "unproven support keeps a wrong error failing",
			value:  transactionDataSession(nil, map[string]any{"error": "access_denied"}),
			params: supportParam,
			status: StatusFail,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(
				t,
				test.status,
				validator.Validate(
					context.Background(),
					Input{Value: test.value, Params: test.params},
				).Status,
			)
		})
	}
}
