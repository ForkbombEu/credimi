// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import "context"

// OID4VPTransactionDataCredentialIDsNonArrayValidator verifies malformed credential_ids.
type OID4VPTransactionDataCredentialIDsNonArrayValidator struct{}

func (OID4VPTransactionDataCredentialIDsNonArrayValidator) ID() string {
	return "oid4vp.transaction_data_credential_ids_non_array"
}

func (OID4VPTransactionDataCredentialIDsNonArrayValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	entries, ok := payload["transaction_data"].([]any)
	if !ok || len(entries) == 0 {
		return Result{
			Status:  StatusFail,
			Message: "Request Object transaction_data is missing or empty",
		}
	}
	for _, raw := range entries {
		entry, ok := normalizeJSONObject(raw)
		if !ok {
			return Result{Status: StatusFail, Message: "transaction_data entry is not an object"}
		}
		if _, exists := entry["credential_ids"]; !exists {
			return Result{Status: StatusFail, Message: "transaction_data credential_ids is missing"}
		}
		if _, isArray := entry["credential_ids"].([]any); !isArray {
			return Result{
				Status:  StatusPass,
				Message: "transaction_data credential_ids is not an array",
			}
		}
	}
	return Result{Status: StatusFail, Message: "transaction_data credential_ids is an array"}
}
