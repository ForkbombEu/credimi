// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later
package validators

import "context"

// OID4VPDCQLTrustedAuthoritiesUnsupportedTypeValidator verifies a trusted authority type
// that is not defined by OID4VP.
type OID4VPDCQLTrustedAuthoritiesUnsupportedTypeValidator struct{}

func (OID4VPDCQLTrustedAuthoritiesUnsupportedTypeValidator) ID() string {
	return "oid4vp.dcql_trusted_authorities_unsupported_type"
}

func (OID4VPDCQLTrustedAuthoritiesUnsupportedTypeValidator) Validate(_ context.Context, input Input) Result {
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
			if authority["type"] == "unsupported" {
				return Result{Status: StatusPass, Message: "trusted_authorities contains unsupported type"}
			}
		}
	}
	return Result{Status: StatusFail, Message: "dcql_query contains no unsupported trusted_authorities type"}
}
