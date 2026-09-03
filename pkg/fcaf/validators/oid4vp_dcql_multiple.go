// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import "context"

// OID4VPDCQLMultipleNonBooleanValidator verifies a malformed DCQL multiple property.
type OID4VPDCQLMultipleNonBooleanValidator struct{}

func (OID4VPDCQLMultipleNonBooleanValidator) ID() string {
	return "oid4vp.dcql_multiple_non_boolean"
}

func (OID4VPDCQLMultipleNonBooleanValidator) Validate(_ context.Context, input Input) Result {
	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	query, ok := normalizeJSONObject(payload["dcql_query"])
	if !ok {
		return Result{Status: StatusFail, Message: "Request Object dcql_query is missing or not an object"}
	}
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query.credentials is missing or empty"}
	}
	for _, raw := range credentials {
		credential, ok := normalizeJSONObject(raw)
		if !ok {
			return Result{Status: StatusFail, Message: "dcql_query credential is not an object"}
		}
		if value, exists := credential["multiple"]; exists {
			if _, isBoolean := value.(bool); !isBoolean {
				return Result{Status: StatusPass, Message: "dcql_query credential multiple is not a boolean"}
			}
		}
	}
	return Result{Status: StatusFail, Message: "dcql_query credential multiple is boolean or missing"}
}
