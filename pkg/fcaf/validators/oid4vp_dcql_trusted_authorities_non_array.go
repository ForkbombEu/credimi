// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import "context"

type OID4VPDCQLTrustedAuthoritiesNonArrayValidator struct{}

func (OID4VPDCQLTrustedAuthoritiesNonArrayValidator) ID() string {
	return "oid4vp.dcql_trusted_authorities_non_array"
}

func (OID4VPDCQLTrustedAuthoritiesNonArrayValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	query, ok := normalizeJSONObject(payload["dcql_query"])
	if !ok {
		return Result{Status: StatusFail, Message: requestObjectDCQLQueryMissingMessage}
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
		if value, exists := credential["trusted_authorities"]; exists {
			if _, array := value.([]any); !array {
				return Result{
					Status:  StatusPass,
					Message: "DCQL credential trusted_authorities is not an array",
				}
			}
		}
	}
	return Result{
		Status:  StatusFail,
		Message: "dcql_query contains no non-array trusted_authorities",
	}
}
