// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
)

// OID4VPWalletNonceMatchValidator verifies that the wallet_nonce supplied in
// the request_uri POST exactly matches the signed Request Object claim.
type OID4VPWalletNonceMatchValidator struct{}

func (OID4VPWalletNonceMatchValidator) ID() string {
	return "oid4vp.wallet_nonce_matches_request_object"
}

func (OID4VPWalletNonceMatchValidator) Validate(_ context.Context, input Input) Result {
	postedWalletNonce, payload, err := decodeWalletNonceEvidence(input.Value)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}

	requestObjectWalletNonce, ok := payload["wallet_nonce"].(string)
	if !ok || requestObjectWalletNonce == "" {
		return Result{
			Status:  StatusFail,
			Message: "Request Object wallet_nonce is missing or not a string",
		}
	}
	if postedWalletNonce != requestObjectWalletNonce {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"request_uri POST wallet_nonce %q does not match Request Object wallet_nonce %q",
				postedWalletNonce,
				requestObjectWalletNonce,
			),
		}
	}

	return Result{
		Status:  StatusPass,
		Message: "request_uri POST and Request Object wallet_nonce values match exactly",
	}
}

func decodeWalletNonceEvidence(value any) (string, map[string]any, error) {
	evidence, ok := normalizeJSONObject(value)
	if !ok {
		return "", nil, fmt.Errorf("input is not an object")
	}

	requestURIPayload, ok := normalizeJSONObject(evidence["request_uri_payload"])
	if !ok {
		return "", nil, fmt.Errorf("request_uri_payload is missing or not an object")
	}
	postedWalletNonce, ok := requestURIPayload["wallet_nonce"].(string)
	if !ok || postedWalletNonce == "" {
		return "", nil, fmt.Errorf("request_uri POST wallet_nonce is missing or not a string")
	}

	requestObject, ok := evidence["request_object"].(string)
	if !ok || requestObject == "" {
		return "", nil, fmt.Errorf("request_object is missing or not a string")
	}
	payload, err := compactJWTPart(requestObject, 1)
	if err != nil {
		return "", nil, err
	}
	return postedWalletNonce, payload, nil
}
