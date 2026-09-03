// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import "context"

// OID4VPDCQLTrustedAuthoritiesEmptyValidator verifies an empty trusted_authorities array.
type OID4VPDCQLTrustedAuthoritiesEmptyValidator struct{}

func (OID4VPDCQLTrustedAuthoritiesEmptyValidator) ID() string {
	return "oid4vp.dcql_trusted_authorities_empty"
}

func (OID4VPDCQLTrustedAuthoritiesEmptyValidator) Validate(_ context.Context, input Input) Result {
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
		authorities, exists := credential["trusted_authorities"].([]any)
		if exists && len(authorities) == 0 {
			return Result{
				Status:  StatusPass,
				Message: "DCQL credential contains an empty trusted_authorities array",
			}
		}
	}
	return Result{
		Status:  StatusFail,
		Message: "dcql_query contains no credential with an empty trusted_authorities array",
	}
}
