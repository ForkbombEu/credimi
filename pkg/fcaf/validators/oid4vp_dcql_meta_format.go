// SPDX-FileCopyrightText: 2026 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import "context"

// OID4VPDCQLMetaFormatMismatchValidator verifies DCQL metadata from another credential format.
type OID4VPDCQLMetaFormatMismatchValidator struct{}

func (OID4VPDCQLMetaFormatMismatchValidator) ID() string {
	return "oid4vp.dcql_meta_format_mismatch"
}

func (OID4VPDCQLMetaFormatMismatchValidator) Validate(_ context.Context, input Input) Result {
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
		format, _ := credential["format"].(string)
		meta, ok := normalizeJSONObject(credential["meta"])
		if !ok {
			continue
		}
		switch format {
		case "dc+sd-jwt":
			if _, exists := meta["doctype_value"]; exists {
				return Result{Status: StatusPass, Message: "SD-JWT credential uses mdoc doctype_value metadata"}
			}
		case "mso_mdoc":
			if _, exists := meta["vct_values"]; exists {
				return Result{Status: StatusPass, Message: "mdoc credential uses SD-JWT vct_values metadata"}
			}
		}
	}
	return Result{Status: StatusFail, Message: "dcql_query contains no credential metadata from another format"}
}
