// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import "context"

type OID4VPDCQLCredentialsNonArrayValidator struct{}

func (OID4VPDCQLCredentialsNonArrayValidator) ID() string { return "oid4vp.dcql_credentials_non_array" }

func (OID4VPDCQLCredentialsNonArrayValidator) Validate(_ context.Context, input Input) Result {
	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	query, ok := normalizeJSONObject(payload["dcql_query"])
	if !ok {
		return Result{Status: StatusFail, Message: "Request Object dcql_query is missing or not an object"}
	}
	credentials, exists := query["credentials"]
	if !exists {
		return Result{Status: StatusFail, Message: "dcql_query.credentials is missing"}
	}
	if _, isArray := credentials.([]any); isArray {
		return Result{Status: StatusFail, Message: "dcql_query.credentials is an array"}
	}
	return Result{Status: StatusPass, Message: "dcql_query.credentials is not an array"}
}
