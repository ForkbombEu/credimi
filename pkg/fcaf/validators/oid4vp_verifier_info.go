// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
)

// OID4VPVerifierInfoAllCredentialsValidator verifies that every verifier_info
// attestation applies to all requested DCQL credentials by omitting credential_ids.
type OID4VPVerifierInfoAllCredentialsValidator struct{}

func (OID4VPVerifierInfoAllCredentialsValidator) ID() string {
	return "oid4vp.verifier_info_applies_all_dcql_credentials"
}

func (OID4VPVerifierInfoAllCredentialsValidator) Validate(_ context.Context, input Input) Result {
	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}

	dcqlQuery, ok := normalizeJSONObject(payload["dcql_query"])
	if !ok {
		return Result{Status: StatusFail, Message: "Request Object dcql_query is missing or not an object"}
	}
	credentials, ok := dcqlQuery["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "Request Object dcql_query.credentials is missing or empty"}
	}
	for index, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("dcql_query.credentials[%d] is not an object", index)}
		}
		if id, _ := credential["id"].(string); id == "" {
			return Result{Status: StatusFail, Message: fmt.Sprintf("dcql_query.credentials[%d].id is missing or empty", index)}
		}
	}

	attestations, ok := payload["verifier_info"].([]any)
	if !ok || len(attestations) == 0 {
		return Result{Status: StatusFail, Message: "Request Object verifier_info is missing or empty"}
	}
	for index, rawAttestation := range attestations {
		attestation, ok := normalizeJSONObject(rawAttestation)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("verifier_info[%d] is not an object", index)}
		}
		if format, _ := attestation["format"].(string); format == "" {
			return Result{Status: StatusFail, Message: fmt.Sprintf("verifier_info[%d].format is missing or empty", index)}
		}
		if _, exists := attestation["data"]; !exists {
			return Result{Status: StatusFail, Message: fmt.Sprintf("verifier_info[%d].data is missing", index)}
		}
		if _, exists := attestation["credential_ids"]; exists {
			return Result{Status: StatusFail, Message: fmt.Sprintf("verifier_info[%d] contains credential_ids", index)}
		}
	}

	return Result{Status: StatusPass, Message: "all verifier_info attestations omit credential_ids for the requested DCQL credentials"}
}
