// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import "context"

// OID4VPTransactionDataTypeNonStringValidator verifies a malformed type field.
type OID4VPTransactionDataTypeNonStringValidator struct{}

func (OID4VPTransactionDataTypeNonStringValidator) ID() string {
	return "oid4vp.transaction_data_type_non_string"
}

func (OID4VPTransactionDataTypeNonStringValidator) Validate(_ context.Context, input Input) Result {
	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	entries, ok := payload["transaction_data"].([]any)
	if !ok || len(entries) == 0 {
		return Result{Status: StatusFail, Message: "Request Object transaction_data is missing or empty"}
	}
	for _, raw := range entries {
		entry, ok := normalizeJSONObject(raw)
		if !ok {
			return Result{Status: StatusFail, Message: "transaction_data entry is not an object"}
		}
		if value, exists := entry["type"]; exists {
			if _, stringType := value.(string); !stringType {
				return Result{Status: StatusPass, Message: "transaction_data type is not a string"}
			}
		}
	}
	return Result{Status: StatusFail, Message: "transaction_data type is a string or missing"}
}
