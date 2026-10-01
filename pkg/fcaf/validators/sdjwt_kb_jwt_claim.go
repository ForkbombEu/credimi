// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"strings"
)

// SDJWTKBJWTClaimStringPrefixValidator reads a claim of the Key Binding JWT
// rather than of the credential.
//
// The KB-JWT audience is where a presentation records who it was produced for,
// and over the Digital Credentials API that is the browser origin prefixed by
// `origin:` instead of a Client Identifier. The credential claims say nothing
// about it, so the existing sdjwt.claim_* family cannot reach this value: the
// parser keeps the Key Binding payload beside the claims, not inside them.
type SDJWTKBJWTClaimStringPrefixValidator struct{}

// ID returns the validator identifier.
func (SDJWTKBJWTClaimStringPrefixValidator) ID() string {
	return "sdjwt.kb_jwt_claim_string_prefix"
}

// Validate requires every presented KB-JWT to carry the claim with the prefix.
func (SDJWTKBJWTClaimStringPrefixValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Claim  string `json:"claim"`
		Prefix string `json:"prefix"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Claim == "" {
		return Result{Status: StatusError, Message: claimParamRequired}
	}
	if params.Prefix == "" {
		return Result{Status: StatusError, Message: "prefix param is required"}
	}

	presentations, ok := sdjwtPresentations(input.Value)
	if !ok || len(presentations) == 0 {
		return Result{
			Status:  StatusFail,
			Message: "evidence does not contain an SD-JWT presentation",
		}
	}
	for index, presentation := range presentations {
		if presentation.KeyBinding == nil {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"presentation %d carries no Key Binding JWT",
					index,
				),
			}
		}
		value, found := resolveObjectPath(presentation.KeyBinding, params.Claim)
		if !found {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"presentation %d Key Binding JWT has no %q claim",
					index,
					params.Claim,
				),
			}
		}
		text, ok := value.(string)
		if !ok {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"presentation %d Key Binding JWT claim %q is %T, expected a string",
					index,
					params.Claim,
					value,
				),
			}
		}
		if !strings.HasPrefix(text, params.Prefix) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"presentation %d Key Binding JWT claim %q is %q, expected the prefix %q",
					index,
					params.Claim,
					text,
					params.Prefix,
				),
			}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"every Key Binding JWT claim %q carries the prefix %q",
			params.Claim,
			params.Prefix,
		),
	}
}
