// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
)

// OID4VPVerifierMetadataExclusiveValidator verifies that verifier metadata is
// non-empty, contains supported presentation formats, and appears only inside
// the client_metadata Authorization Request parameter.
type OID4VPVerifierMetadataExclusiveValidator struct{}

func (OID4VPVerifierMetadataExclusiveValidator) ID() string {
	return "oid4vp.verifier_metadata_exclusive"
}

func (OID4VPVerifierMetadataExclusiveValidator) Validate(_ context.Context, input Input) Result {
	payload, err := compactJWTPart(input.Value, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}

	clientMetadataValue, exists := payload["client_metadata"]
	if !exists {
		return Result{
			Status:  StatusFail,
			Message: "JWT payload field \"client_metadata\" is missing",
		}
	}
	clientMetadata, ok := clientMetadataValue.(map[string]any)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: "JWT payload field \"client_metadata\" is not an object",
		}
	}
	if len(clientMetadata) == 0 {
		return Result{Status: StatusFail, Message: "JWT payload field \"client_metadata\" is empty"}
	}

	vpFormatsValue, exists := clientMetadata["vp_formats_supported"]
	if !exists {
		return Result{
			Status:  StatusFail,
			Message: "client_metadata field \"vp_formats_supported\" is missing",
		}
	}
	vpFormats, ok := vpFormatsValue.(map[string]any)
	if !ok || len(vpFormats) == 0 {
		return Result{
			Status:  StatusFail,
			Message: "client_metadata field \"vp_formats_supported\" must be a non-empty object",
		}
	}

	for field := range clientMetadata {
		if _, duplicated := payload[field]; duplicated {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"verifier metadata field %q also appears outside client_metadata",
					field,
				),
			}
		}
	}

	return Result{
		Status:  StatusPass,
		Message: "verifier metadata is provided exclusively through client_metadata",
	}
}
