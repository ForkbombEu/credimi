// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// decodeTransactionDataEntry decodes one entry of the Request Object's
// transaction_data array. OpenID4VP 1.0 Section 5.1 encodes every entry as a
// base64url string carrying a JSON object, so a conformant entry is decoded
// here before it is inspected. A value that already is a JSON object is
// returned as is, which also covers captures and callers that hold the decoded
// form. An entry that is neither is a different defect than the one the
// transaction_data validators look for, so every caller fails on it instead of
// reporting the property it was asked to prove.
func decodeTransactionDataEntry(value any) (map[string]any, bool) {
	if object, ok := value.(map[string]any); ok {
		return object, true
	}
	encoded, ok := value.(string)
	if !ok {
		return nil, false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, false
		}
	}
	var object map[string]any
	if err := json.Unmarshal(decoded, &object); err != nil {
		return nil, false
	}
	return object, true
}

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
		entry, ok := decodeTransactionDataEntry(rawTransactionData)
		if !ok {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"transaction_data[%d] is not a base64url-encoded JSON object",
					index,
				),
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
