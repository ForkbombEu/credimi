// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
)

// OID4VPErrorResponseRequiredValidator verifies that the Wallet answered a
// request it cannot satisfy with exactly one OAuth2/OID4VP error code and no
// presentation, so credential selection and user authorization never started.
//
// When a definition names a format in not_applicable_when_wallet_supports_format,
// a Wallet whose own evidence says it supports that format is outside the
// source's applicability: the result is not_applicable, not a code mismatch.
type OID4VPErrorResponseRequiredValidator struct{}

func (OID4VPErrorResponseRequiredValidator) ID() string {
	return "oid4vp.error_response_required"
}

func (OID4VPErrorResponseRequiredValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Code                                  string `json:"code"`
		NotApplicableWhenWalletSupportsFormat string `json:"not_applicable_when_wallet_supports_format"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Code == "" {
		return Result{Status: StatusError, Message: "code param is required"}
	}

	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: sessionNotObjectMessage}
	}

	if params.NotApplicableWhenWalletSupportsFormat != "" {
		if reason, supports := walletSupportsCredentialFormat(
			session,
			params.NotApplicableWhenWalletSupportsFormat,
		); supports {
			return Result{Status: StatusNotApplicable, Message: reason}
		}
	}

	responseValue, _ := findObjectKey(session, "vp_token")
	if !isEmptyDCQLValue(responseValue) {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("wallet returned vp_token, expected error %q", params.Code),
		}
	}

	errorValue := findWalletError(session)
	code := normalizeString(errorValue)
	if code == "" {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("wallet returned no error, expected %q", params.Code),
		}
	}
	if code != params.Code {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"wallet returned error %q, expected %q",
				code,
				params.Code,
			),
		}
	}

	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("wallet returned error %q without a presentation", params.Code),
	}
}

// walletSupportsCredentialFormat reports whether the Wallet's own evidence says
// it supports format: either the wallet_metadata it POSTed to the Request URI
// advertises the format, or it presented a credential in that format.
func walletSupportsCredentialFormat(session map[string]any, format string) (string, bool) {
	observed, _ := normalizeJSONObject(session["observed"])
	requestURIPayload, _ := normalizeJSONObject(observed["request_uri_payload"])
	payload, _ := normalizeJSONObject(requestURIPayload["value"])
	if metadata, err := decodeWalletMetadata(payload["wallet_metadata"]); err == nil {
		if formats, ok := normalizeJSONObject(metadata["vp_formats_supported"]); ok {
			if _, exists := formats[format]; exists {
				return fmt.Sprintf(
					"wallet_metadata declares %s support, so the source's precondition (no %s support) does not hold",
					format,
					format,
				), true
			}
		}
	}
	presentations, _ := normalizeJSONObject(session["decoded_presentations"])
	for queryID, raw := range presentations {
		entries, ok := raw.([]any)
		if !ok {
			continue
		}
		for _, entry := range entries {
			presentation, ok := normalizeJSONObject(entry)
			if !ok {
				continue
			}
			if presented, _ := presentation["format"].(string); presented == format {
				return fmt.Sprintf(
					"wallet presented a %s credential for query %q, so it supports %s and the source's precondition does not hold",
					format,
					queryID,
					format,
				), true
			}
		}
	}
	return "", false
}
