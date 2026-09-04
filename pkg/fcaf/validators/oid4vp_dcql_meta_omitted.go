// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import "context"

// OID4VPDCQLMetaOmittedValidator verifies a DCQL credential that omits meta.
type OID4VPDCQLMetaOmittedValidator struct{}

func (OID4VPDCQLMetaOmittedValidator) ID() string {
	return "oid4vp.dcql_meta_omitted"
}

func (OID4VPDCQLMetaOmittedValidator) Validate(_ context.Context, input Input) Result {
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
		if _, exists := credential["meta"]; !exists {
			return Result{Status: StatusPass, Message: "DCQL credential omits meta"}
		}
	}
	return Result{Status: StatusFail, Message: "dcql_query contains no credential without meta"}
}
