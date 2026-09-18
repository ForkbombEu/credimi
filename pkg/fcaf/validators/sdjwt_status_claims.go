// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"net/url"
)

// SDJWTClaimNonNegativeIntegerValidator verifies that a resolved SD-JWT claim
// carries an integral number that is not negative, the shape the status-list
// `idx` member uses.
type SDJWTClaimNonNegativeIntegerValidator struct{}

// ID returns the validator identifier.
func (SDJWTClaimNonNegativeIntegerValidator) ID() string {
	return "sdjwt.claim_non_negative_integer"
}

// Validate requires the claim to resolve to an integer greater than or equal to
// zero, accepting the float64 JSON representation alongside Go integer values.
func (SDJWTClaimNonNegativeIntegerValidator) Validate(_ context.Context, input Input) Result {
	claim, err := decodeClaimParam(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	value, ok := sdjwtClaim(input.Value, claim)
	if !ok {
		return missingClaim(claim)
	}
	if !isNonNegativeInteger(value) {
		return invalidClaim(
			claim,
			fmt.Errorf("value is %T, expected a non-negative integer", value),
		)
	}
	return validClaim(claim, "contains a non-negative integer")
}

// SDJWTClaimURIValidator verifies that a resolved SD-JWT claim carries an
// absolute RFC 3986 URI, the shape the status-list `uri` member uses.
type SDJWTClaimURIValidator struct{}

// ID returns the validator identifier.
func (SDJWTClaimURIValidator) ID() string {
	return "sdjwt.claim_uri"
}

// Validate requires the claim to resolve to a string that parses as an absolute
// URI with a scheme, without constraining which scheme it uses.
func (SDJWTClaimURIValidator) Validate(_ context.Context, input Input) Result {
	claim, err := decodeClaimParam(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	value, ok := sdjwtClaim(input.Value, claim)
	if !ok {
		return missingClaim(claim)
	}
	text, err := requireUTF8String(value)
	if err != nil {
		return invalidClaim(claim, err)
	}
	parsed, err := url.Parse(text)
	if err != nil {
		return invalidClaim(claim, fmt.Errorf("value is not a valid URI: %w", err))
	}
	if parsed.Scheme == "" || !parsed.IsAbs() {
		return invalidClaim(claim, fmt.Errorf("value %q is not an absolute URI", text))
	}
	return validClaim(claim, "contains an absolute URI")
}
