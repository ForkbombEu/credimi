// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

// SDJWTDisclosureDigestsSHA256Validator verifies that disclosed SD-JWT claims
// use the SHA-256 digest algorithm. Parsing an SD-JWT presentation recomputes
// every disclosure digest and verifies its reference in the issuer payload.
type SDJWTDisclosureDigestsSHA256Validator struct{}

func (SDJWTDisclosureDigestsSHA256Validator) ID() string {
	return "sdjwt.disclosure_digests_sha_256"
}

func (SDJWTDisclosureDigestsSHA256Validator) Validate(_ context.Context, input Input) Result {
	presentations, ok := sdjwtPresentations(input.Value)
	if !ok || len(presentations) == 0 {
		return Result{Status: StatusFail, Message: "SD-JWT presentation evidence is missing or invalid"}
	}

	for index, presentation := range presentations {
		if result := validateSDJWTDisclosureDigestsSHA256(presentation); result != nil {
			result.Message = fmt.Sprintf("presentation[%d]: %s", index, result.Message)
			return *result
		}
	}

	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("all %d SD-JWT presentations use SHA-256 disclosure digests", len(presentations)),
	}
}

func validateSDJWTDisclosureDigestsSHA256(presentation *evidence.SDJWTPresentation) *Result {
	if presentation == nil || presentation.DisclosureCount == 0 {
		return &Result{Status: StatusFail, Message: "SD-JWT presentation contains no disclosed claim digests"}
	}
	if _, ok := presentation.IssuerPayload["_sd"].([]any); !ok {
		return &Result{Status: StatusFail, Message: "SD-JWT issuer payload does not contain disclosure digests"}
	}
	if algorithm, found := presentation.IssuerPayload["_sd_alg"]; found {
		value, ok := algorithm.(string)
		if !ok || value != "sha-256" {
			return &Result{Status: StatusFail, Message: "SD-JWT disclosure digest algorithm is not SHA-256"}
		}
	}
	return nil
}
