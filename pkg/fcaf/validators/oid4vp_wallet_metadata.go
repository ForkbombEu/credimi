// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// OID4VPWalletMetadataValidator checks the wallet_metadata the Wallet POSTed to
// the Request URI (OpenID4VP Section 5.10.1). Its input is the captured
// request_uri payload.
type OID4VPWalletMetadataValidator struct{}

func (OID4VPWalletMetadataValidator) ID() string { return "oid4vp.wallet_metadata" }

func (OID4VPWalletMetadataValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Present []string `json:"present"`
		Absent  []string `json:"absent"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if len(params.Present) == 0 && len(params.Absent) == 0 {
		return Result{Status: StatusError, Message: "present or absent param is required"}
	}

	payload, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: "request_uri payload is missing or not an object",
		}
	}
	metadata, err := decodeWalletMetadata(payload["wallet_metadata"])
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}

	for _, member := range params.Present {
		if isEmptyDCQLValue(metadata[member]) {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("wallet_metadata %q is missing or empty", member),
			}
		}
	}
	for _, member := range params.Absent {
		if _, exists := metadata[member]; exists {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("wallet_metadata %q is present", member),
			}
		}
	}

	parts := make([]string, 0, 2)
	if len(params.Present) > 0 {
		parts = append(parts, "carries "+strings.Join(params.Present, ", "))
	}
	if len(params.Absent) > 0 {
		parts = append(parts, "omits "+strings.Join(params.Absent, ", "))
	}
	return Result{Status: StatusPass, Message: "wallet_metadata " + strings.Join(parts, " and ")}
}

// OID4VPUnencryptedRequestObjectRejectedValidator applies only to a Wallet
// whose POSTed wallet_metadata publishes jwks, which OpenID4VP Section 5.10.1
// uses to require an encrypted Request Object. Such a Wallet must answer the
// unencrypted Request Object it received with invalid_request and no
// presentation. Its input is the captured presentation session.
type OID4VPUnencryptedRequestObjectRejectedValidator struct{}

func (OID4VPUnencryptedRequestObjectRejectedValidator) ID() string {
	return "oid4vp.unencrypted_request_object_rejected"
}

func (OID4VPUnencryptedRequestObjectRejectedValidator) Validate(
	ctx context.Context,
	input Input,
) Result {
	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "captured presentation session is not an object"}
	}
	observed, _ := normalizeJSONObject(session["observed"])
	requestURIPayload, _ := normalizeJSONObject(observed["request_uri_payload"])
	payload, ok := normalizeJSONObject(requestURIPayload["value"])
	if !ok {
		return Result{Status: StatusFail, Message: "Wallet did not POST to the Request URI"}
	}
	metadata, err := decodeWalletMetadata(payload["wallet_metadata"])
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	jwks, _ := normalizeJSONObject(metadata["jwks"])
	if keys, _ := jwks["keys"].([]any); len(keys) == 0 {
		return Result{
			Status:  StatusNotApplicable,
			Message: "wallet_metadata publishes no jwks, so the Wallet does not require an encrypted Request Object",
		}
	}

	raw, _ := normalizeJSONObject(session["raw"])
	requestObject, _ := raw["authorization_request_jwt"].(string)
	if strings.Count(requestObject, ".") != 2 {
		return Result{
			Status:  StatusFail,
			Message: "served Request Object is not an unencrypted compact JWS",
		}
	}

	return OID4VPErrorResponseRequiredValidator{}.Validate(ctx, Input{
		Value:  session,
		Params: map[string]any{"code": invalidRequestError},
	})
}

func decodeWalletMetadata(value any) (map[string]any, error) {
	if encoded, ok := value.(string); ok {
		var metadata map[string]any
		if err := json.Unmarshal([]byte(encoded), &metadata); err != nil || metadata == nil {
			return nil, fmt.Errorf("wallet_metadata is not a JSON object")
		}
		return metadata, nil
	}
	metadata, ok := normalizeJSONObject(value)
	if !ok {
		return nil, fmt.Errorf("wallet_metadata is missing or not a JSON object")
	}
	return metadata, nil
}
