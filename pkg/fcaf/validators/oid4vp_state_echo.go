// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
)

// OID4VPStateEchoValidator verifies that the Wallet returned the state
// parameter of the Authorization Request unchanged in the Authorization
// Response, as OpenID4VP 1.0 Section 5.2 requires. Its input carries the
// Request Object served to the Wallet as `request_object` and the decrypted
// Authorization Response as `response`.
type OID4VPStateEchoValidator struct{}

// ID returns the validator identifier.
func (OID4VPStateEchoValidator) ID() string {
	return "oid4vp.state_echoed"
}

// Validate compares the state claim the Wallet received with the state member
// of the response it posted.
func (OID4VPStateEchoValidator) Validate(_ context.Context, input Input) Result {
	evidence, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "input is not an object"}
	}

	requestObject, ok := evidence["request_object"].(string)
	if !ok || requestObject == "" {
		return Result{Status: StatusFail, Message: "request_object is missing or not a string"}
	}
	payload, err := compactJWTPart(requestObject, 1)
	if err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	requestState, ok := payload["state"].(string)
	if !ok || requestState == "" {
		return Result{
			Status:  StatusFail,
			Message: "Authorization Request state is missing or not a string",
		}
	}

	response, ok := normalizeJSONObject(evidence["response"])
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: "decrypted Authorization Response is missing or not an object",
		}
	}
	responseState, ok := response["state"].(string)
	if !ok || responseState == "" {
		return Result{
			Status:  StatusFail,
			Message: "Authorization Response state is missing or not a string",
		}
	}

	if responseState != requestState {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"Authorization Response state %q does not match the Authorization Request state %q",
				responseState,
				requestState,
			),
		}
	}

	return Result{
		Status:  StatusPass,
		Message: "Authorization Response returned the Authorization Request state unchanged",
	}
}
