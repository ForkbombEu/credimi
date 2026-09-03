// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
)

// OID4VPTransactionDataCredentialIDsValidator verifies that every transaction
// data credential ID is a DCQL Credential Query ID.
type OID4VPTransactionDataCredentialIDsValidator struct{}

func (OID4VPTransactionDataCredentialIDsValidator) ID() string {
	return "oid4vp.transaction_data_credential_ids_match_dcql"
}

func (OID4VPTransactionDataCredentialIDsValidator) Validate(_ context.Context, input Input) Result {
	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}

	dcqlQuery, ok := normalizeJSONObject(payload["dcql_query"])
	if !ok {
		return Result{Status: StatusFail, Message: requestObjectDCQLQueryMissingMessage}
	}
	credentials, ok := dcqlQuery["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{
			Status:  StatusFail,
			Message: "Request Object dcql_query.credentials is missing or empty",
		}
	}
	credentialIDs := make(map[string]struct{}, len(credentials))
	for index, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("dcql_query.credentials[%d] is not an object", index),
			}
		}
		id, _ := credential["id"].(string)
		if id == "" {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("dcql_query.credentials[%d].id is missing or empty", index),
			}
		}
		credentialIDs[id] = struct{}{}
	}

	transactionData, ok := payload["transaction_data"].([]any)
	if !ok || len(transactionData) == 0 {
		return Result{
			Status:  StatusFail,
			Message: "Request Object transaction_data is missing or empty",
		}
	}
	for index, rawTransactionData := range transactionData {
		entry, ok := normalizeJSONObject(rawTransactionData)
		if !ok {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("transaction_data[%d] is not an object", index),
			}
		}
		if transactionType, _ := entry["type"].(string); transactionType == "" {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("transaction_data[%d].type is missing or empty", index),
			}
		}
		if data, _ := entry["data"].(string); data == "" {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("transaction_data[%d].data is missing or empty", index),
			}
		}
		ids, ok := entry["credential_ids"].([]any)
		if !ok || len(ids) == 0 {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"transaction_data[%d].credential_ids is missing or empty",
					index,
				),
			}
		}
		for credentialIndex, rawID := range ids {
			id, ok := rawID.(string)
			if !ok || id == "" {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"transaction_data[%d].credential_ids[%d] is not a non-empty string",
						index,
						credentialIndex,
					),
				}
			}
			if _, found := credentialIDs[id]; !found {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"transaction_data[%d].credential_ids[%d] %q does not match a DCQL credential ID",
						index,
						credentialIndex,
						id,
					),
				}
			}
		}
	}

	return Result{
		Status:  StatusPass,
		Message: "all transaction_data credential_ids match requested DCQL credentials",
	}
}
