// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

// OID4VPDCQLObjectPropertyFilterValidator verifies that a DCQL object-property
// path discloses only the requested member of the selected object.
type OID4VPDCQLObjectPropertyFilterValidator struct{}

// ID returns the validator identifier.
func (OID4VPDCQLObjectPropertyFilterValidator) ID() string {
	return "oid4vp.dcql_object_property_filter"
}

// Validate requires the session-bound query to carry the exact claims path and
// the presented SD-JWT to retain the requested object member only.
func (OID4VPDCQLObjectPropertyFilterValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	params, err := DecodeParams[struct {
		VCT                 string   `json:"vct"`
		Path                []any    `json:"path"`
		RequiredProperties  []string `json:"required_properties"`
		ForbiddenProperties []string `json:"forbidden_properties"`
		ForbiddenClaims     []string `json:"forbidden_claims"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.VCT == "" {
		return Result{Status: StatusError, Message: "vct param is required"}
	}
	if len(params.Path) != 2 {
		return Result{
			Status:  StatusError,
			Message: "path param must contain an object claim and property",
		}
	}
	claim, ok := params.Path[0].(string)
	if !ok || claim == "" {
		return Result{
			Status:  StatusError,
			Message: "path param must start with an object claim name",
		}
	}
	if _, ok := params.Path[1].(string); !ok {
		return Result{
			Status:  StatusError,
			Message: "path param must end with an object property name",
		}
	}
	if len(params.RequiredProperties) == 0 {
		return Result{Status: StatusError, Message: "required_properties param is required"}
	}
	if len(params.ForbiddenProperties) == 0 && len(params.ForbiddenClaims) == 0 {
		return Result{
			Status:  StatusError,
			Message: "forbidden_properties or forbidden_claims param is required",
		}
	}

	root, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "DCQL evidence is not an object"}
	}
	queryValue, found := findObjectKey(root, "dcql_query")
	if !found {
		return Result{Status: StatusFail, Message: "captured evidence does not contain dcql_query"}
	}
	query, ok := normalizeJSONObject(queryValue)
	if !ok {
		return Result{Status: StatusFail, Message: "captured dcql_query is not an object"}
	}
	credentialID, result := arraySelectorCredentialID(query, params.VCT, params.Path)
	if result != nil {
		return *result
	}

	responseValue, found := findObjectKey(root, "vp_token")
	if !found {
		return Result{Status: StatusFail, Message: "captured evidence does not contain vp_token"}
	}
	response, ok := normalizeJSONObject(responseValue)
	if !ok {
		return Result{Status: StatusFail, Message: "wallet vp_token is not an object"}
	}
	presentations, ok := response[credentialID].([]any)
	if !ok || len(presentations) != 1 {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("vp_token[%q] is not a single presentation", credentialID),
		}
	}
	token, ok := presentations[0].(string)
	if !ok || token == "" {
		return Result{Status: StatusFail, Message: "presented credential is not an SD-JWT"}
	}
	presentation, err := evidence.ParseSDJWTPresentation(token)
	if err != nil {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("presented credential is invalid: %v", err),
		}
	}
	if presentation.Claims["vct"] != params.VCT {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"presented credential vct is %v, expected %q",
				presentation.Claims["vct"],
				params.VCT,
			),
		}
	}
	object, ok := normalizeJSONObject(presentation.Claims[claim])
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("presented credential does not disclose object claim %q", claim),
		}
	}
	for _, property := range params.RequiredProperties {
		if _, exists := object[property]; !exists {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"object claim %q does not disclose requested property %q",
					claim,
					property,
				),
			}
		}
	}
	for _, property := range params.ForbiddenProperties {
		if _, exists := object[property]; exists {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"object claim %q discloses unrequested property %q",
					claim,
					property,
				),
			}
		}
	}
	for _, forbidden := range params.ForbiddenClaims {
		if _, disclosed := presentation.Claims[forbidden]; disclosed {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"presented credential discloses unrequested claim %q",
					forbidden,
				),
			}
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("object claim %q discloses only requested properties", claim),
	}
}
