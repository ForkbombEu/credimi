// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later
package validators

import "context"

// OID4VPDCQLTrustedAuthoritiesMissingTypeValidator verifies a trusted authority
// whose required type property is absent.
type OID4VPDCQLTrustedAuthoritiesMissingTypeValidator struct{}

func (OID4VPDCQLTrustedAuthoritiesMissingTypeValidator) ID() string {
	return "oid4vp.dcql_trusted_authorities_missing_type"
}

func (OID4VPDCQLTrustedAuthoritiesMissingTypeValidator) Validate(
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
	for _, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			continue
		}
		authorities, ok := credential["trusted_authorities"].([]any)
		if !ok {
			continue
		}
		for _, rawAuthority := range authorities {
			authority, ok := normalizeJSONObject(rawAuthority)
			if !ok {
				continue
			}
			if _, hasType := authority["type"]; !hasType {
				return Result{
					Status:  StatusPass,
					Message: "trusted_authorities contains an authority without type",
				}
			}
		}
	}
	return Result{
		Status:  StatusFail,
		Message: "dcql_query contains no trusted authority without type",
	}
}
