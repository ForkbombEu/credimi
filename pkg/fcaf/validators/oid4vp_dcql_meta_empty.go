// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import "context"

// OID4VPDCQLMetaEmptyValidator verifies a DCQL credential with an empty meta object.
type OID4VPDCQLMetaEmptyValidator struct{}

func (OID4VPDCQLMetaEmptyValidator) ID() string {
	return "oid4vp.dcql_meta_empty"
}

func (OID4VPDCQLMetaEmptyValidator) Validate(_ context.Context, input Input) Result {
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
		meta, exists := normalizeJSONObject(credential["meta"])
		if exists && len(meta) == 0 {
			return Result{Status: StatusPass, Message: "DCQL credential contains an empty meta object"}
		}
	}
	return Result{Status: StatusFail, Message: "dcql_query contains no credential with an empty meta object"}
}
