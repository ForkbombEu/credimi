// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
)

// OID4VPUnsupportedResponseTypeValidator verifies rejection or discontinuation
// after a Wallet receives a response type that HAIP does not permit.
type OID4VPUnsupportedResponseTypeValidator struct{}

func (OID4VPUnsupportedResponseTypeValidator) ID() string {
	return "oid4vp.unsupported_response_type_handled"
}

func (OID4VPUnsupportedResponseTypeValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		ExpectedResponseType string `json:"expected_response_type"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.ExpectedResponseType == "" {
		return Result{Status: StatusError, Message: "expected_response_type is required"}
	}

	responseTypes := collectObjectFieldValues(input.Value, "response_type")
	if len(responseTypes) == 0 {
		return Result{
			Status:  StatusFail,
			Message: "captured request does not contain response_type",
		}
	}
	for _, rawResponseType := range responseTypes {
		responseType, ok := rawResponseType.(string)
		if !ok || responseType != params.ExpectedResponseType {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"captured response_type is %v, expected %q",
					rawResponseType,
					params.ExpectedResponseType,
				),
			}
		}
	}

	walletResponse, found := findObjectKey(input.Value, "wallet_response")
	if !found || walletResponse == nil {
		return Result{
			Status:  StatusPass,
			Message: "wallet discontinued the interaction without submitting a response",
			Details: map[string]any{"outcome": "discontinued"},
		}
	}
	walletResponseObject, ok := normalizeJSONObject(walletResponse)
	if !ok {
		return Result{Status: StatusFail, Message: "captured wallet_response is not an object"}
	}
	value, found := walletResponseObject["value"]
	if !found || value == nil {
		return Result{
			Status:  StatusPass,
			Message: "wallet discontinued the interaction without submitting a response value",
			Details: map[string]any{"outcome": "discontinued"},
		}
	}
	response, ok := normalizeJSONObject(value)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: "captured wallet response value is not an object",
		}
	}
	if vpToken, found := response["vp_token"]; found && !isEmptyDCQLValue(vpToken) {
		return Result{
			Status:  StatusFail,
			Message: "wallet returned vp_token for an unsupported response_type",
		}
	}
	errorValue, found := response["error"]
	if !found {
		return Result{
			Status:  StatusFail,
			Message: "wallet submitted a response without an error",
		}
	}
	errorText, ok := errorValue.(string)
	if !ok || errorText == "" {
		return Result{Status: StatusFail, Message: "wallet error must be a non-empty string"}
	}

	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("wallet returned error %q for unsupported response_type", errorText),
		Details: map[string]any{"error": errorText, "outcome": "error"},
	}
}

func collectObjectFieldValues(value any, field string) []any {
	values := make([]any, 0)
	var visit func(any)
	visit = func(current any) {
		switch typed := current.(type) {
		case map[string]any:
			for key, child := range typed {
				if key == field {
					values = append(values, child)
				}
				visit(child)
			}
		case []any:
			for _, child := range typed {
				visit(child)
			}
		}
	}
	visit(value)
	return values
}
