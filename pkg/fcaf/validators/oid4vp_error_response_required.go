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
type OID4VPErrorResponseRequiredValidator struct{}

func (OID4VPErrorResponseRequiredValidator) ID() string {
	return "oid4vp.error_response_required"
}

func (OID4VPErrorResponseRequiredValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Code string `json:"code"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Code == "" {
		return Result{Status: StatusError, Message: "code param is required"}
	}

	session, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "captured presentation session is not an object"}
	}

	responseValue, _ := findObjectKey(session, "vp_token")
	if !isEmptyDCQLValue(responseValue) {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("wallet returned vp_token, expected error %q", params.Code),
		}
	}

	errorValue, _ := findObjectKey(session, "error")
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
