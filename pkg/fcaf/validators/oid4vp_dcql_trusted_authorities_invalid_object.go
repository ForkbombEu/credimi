// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later
package validators

import "context"

type OID4VPDCQLTrustedAuthoritiesInvalidObjectValidator struct{}

func (OID4VPDCQLTrustedAuthoritiesInvalidObjectValidator) ID() string {
	return "oid4vp.dcql_trusted_authorities_invalid_object"
}

func (OID4VPDCQLTrustedAuthoritiesInvalidObjectValidator) Validate(
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
			continue
		}
		authorities, ok := credential["trusted_authorities"].([]any)
		if !ok || len(authorities) == 0 {
			continue
		}
		for _, rawAuthority := range authorities {
			authority, ok := normalizeJSONObject(rawAuthority)
			if ok {
				_, hasType := authority["type"]
				_, hasValues := authority["values"]
				if !hasType && !hasValues {
					return Result{
						Status:  StatusPass,
						Message: "trusted_authorities contains an undefined object",
					}
				}
			}
		}
	}
	return Result{
		Status:  StatusFail,
		Message: "dcql_query contains no undefined trusted_authorities object",
	}
}
