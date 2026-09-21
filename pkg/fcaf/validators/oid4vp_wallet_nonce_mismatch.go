// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
)

// OID4VPWalletNonceMismatchValidator verifies that the Request Object returned
// from a POST request_uri flow has the expected non-matching nonce state.
type OID4VPWalletNonceMismatchValidator struct{}

func (OID4VPWalletNonceMismatchValidator) ID() string {
	return "oid4vp.wallet_nonce_mismatches_request_object"
}

func (OID4VPWalletNonceMismatchValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Expected string `json:"expected"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Expected != "different" && params.Expected != "missing" {
		return Result{Status: StatusError, Message: "expected param must be different or missing"}
	}

	postedWalletNonce, payload, err := decodeWalletNonceEvidence(input.Value)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	requestObjectWalletNonce, present := payload["wallet_nonce"]

	switch params.Expected {
	case "missing":
		if present {
			return Result{Status: StatusFail, Message: "Request Object wallet_nonce is present"}
		}
		return Result{Status: StatusPass, Message: "Request Object wallet_nonce is absent"}
	case "different":
		nonce, ok := requestObjectWalletNonce.(string)
		if !ok || nonce == "" {
			return Result{Status: StatusFail, Message: "Request Object wallet_nonce is missing or not a string"}
		}
		if nonce == postedWalletNonce {
			return Result{Status: StatusFail, Message: "Request Object wallet_nonce matches the request_uri POST"}
		}
		return Result{Status: StatusPass, Message: fmt.Sprintf("Request Object wallet_nonce %q differs from request_uri POST wallet_nonce %q", nonce, postedWalletNonce)}
	}

	return Result{Status: StatusError, Message: "unreachable wallet_nonce expectation"}
}
