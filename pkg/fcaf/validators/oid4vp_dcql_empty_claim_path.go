// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later
package validators

import "context"

// OID4VPDCQLEmptyClaimPathValidator verifies a DCQL claim whose path is empty.
type OID4VPDCQLEmptyClaimPathValidator struct{}

func (OID4VPDCQLEmptyClaimPathValidator) ID() string { return "oid4vp.dcql_empty_claim_path" }

func (OID4VPDCQLEmptyClaimPathValidator) Validate(_ context.Context, input Input) Result {
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
		claims, ok := credential["claims"].([]any)
		if !ok {
			continue
		}
		for _, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				continue
			}
			if path, ok := claim["path"].([]any); ok && len(path) == 0 {
				return Result{Status: StatusPass, Message: "DCQL claim path is empty"}
			}
		}
	}
	return Result{Status: StatusFail, Message: "dcql_query contains no empty claim path"}
}
