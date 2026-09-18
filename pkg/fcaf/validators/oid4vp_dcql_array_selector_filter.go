// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

// OID4VPDCQLArraySelectorFilterValidator verifies that a DCQL array selector
// drops the selected elements that cannot satisfy the trailing path component,
// while the satisfiable elements stay in the presented selection.
type OID4VPDCQLArraySelectorFilterValidator struct{}

// ID returns the validator identifier.
func (OID4VPDCQLArraySelectorFilterValidator) ID() string {
	return "oid4vp.dcql_array_selector_filter"
}

// Validate requires the session-bound query to carry the exact claims path and
// the presented credential to disclose the retained values without the values
// that belong only to the removed elements.
func (OID4VPDCQLArraySelectorFilterValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	params, err := DecodeParams[struct {
		VCT             string   `json:"vct"`
		Path            []any    `json:"path"`
		RequiredValues  []string `json:"required_values"`
		ForbiddenValues []string `json:"forbidden_values"`
		ForbiddenClaims []string `json:"forbidden_claims"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.VCT == "" {
		return Result{Status: StatusError, Message: "vct param is required"}
	}
	if len(params.Path) < 2 {
		return Result{Status: StatusError, Message: "path param needs at least two components"}
	}
	claim, ok := params.Path[0].(string)
	if !ok || claim == "" {
		return Result{Status: StatusError, Message: "path param must start with a claim name"}
	}
	if len(params.RequiredValues) == 0 {
		return Result{Status: StatusError, Message: "required_values param is required"}
	}
	if len(params.ForbiddenValues) == 0 && len(params.ForbiddenClaims) == 0 {
		return Result{
			Status:  StatusError,
			Message: "forbidden_values or forbidden_claims param is required",
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
			Status: StatusFail,
			Message: fmt.Sprintf(
				"vp_token[%q] is not a single presentation",
				credentialID,
			),
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
	selection, found := presentation.Claims[claim]
	if !found {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("presented credential does not disclose claim %q", claim),
		}
	}
	disclosed := collectStringLeaves(selection, nil)
	for _, value := range params.RequiredValues {
		if !containsString(disclosed, value) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"claim %q does not disclose retained value %q",
					claim,
					value,
				),
			}
		}
	}
	for _, value := range params.ForbiddenValues {
		if containsString(disclosed, value) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"claim %q still discloses %q, so the unsatisfiable element was not removed",
					claim,
					value,
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
		Status: StatusPass,
		Message: fmt.Sprintf(
			"claim %q discloses only the elements that satisfy the claims path",
			claim,
		),
	}
}

func arraySelectorCredentialID(
	query map[string]any,
	vct string,
	path []any,
) (string, *Result) {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) != 1 {
		return "", &Result{
			Status:  StatusFail,
			Message: "dcql_query must contain exactly one credential query",
		}
	}
	credential, ok := normalizeJSONObject(credentials[0])
	if !ok || credential["format"] != "dc+sd-jwt" {
		return "", &Result{
			Status:  StatusFail,
			Message: "credential query format must be dc+sd-jwt",
		}
	}
	id, ok := credential["id"].(string)
	if !ok || id == "" {
		return "", &Result{
			Status:  StatusFail,
			Message: "credential query id is not a non-empty string",
		}
	}
	meta, ok := normalizeJSONObject(credential["meta"])
	if !ok || !reflect.DeepEqual(meta["vct_values"], []any{vct}) {
		return "", &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("credential query does not request vct %q", vct),
		}
	}
	claims, ok := credential["claims"].([]any)
	if !ok || len(claims) != 1 {
		return "", &Result{
			Status:  StatusFail,
			Message: "credential query must contain exactly one claim",
		}
	}
	claim, ok := normalizeJSONObject(claims[0])
	if !ok || !equalClaimPath(claim["path"], path) {
		return "", &Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("credential query claim path is not %v", path),
		}
	}
	return id, nil
}

// equalClaimPath compares claims paths across representations: YAML decodes an
// index as an int while the captured JSON query carries it as a float64.
func equalClaimPath(captured any, expected []any) bool {
	components, ok := captured.([]any)
	if !ok || len(components) != len(expected) {
		return false
	}
	for index, component := range components {
		if !equalClaimPathComponent(component, expected[index]) {
			return false
		}
	}
	return true
}

func equalClaimPathComponent(captured any, expected any) bool {
	if captured == nil || expected == nil {
		return captured == nil && expected == nil
	}
	capturedIndex, capturedNumeric := claimPathIndex(captured)
	expectedIndex, expectedNumeric := claimPathIndex(expected)
	if capturedNumeric || expectedNumeric {
		return capturedNumeric && expectedNumeric && capturedIndex == expectedIndex
	}
	return captured == expected
}

func claimPathIndex(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float64:
		return typed, true
	case json.Number:
		index, err := typed.Float64()
		return index, err == nil
	}
	return 0, false
}

func collectStringLeaves(value any, collected []string) []string {
	switch typed := value.(type) {
	case string:
		return append(collected, typed)
	case []any:
		for _, item := range typed {
			collected = collectStringLeaves(item, collected)
		}
	case map[string]any:
		for _, item := range typed {
			collected = collectStringLeaves(item, collected)
		}
	}
	return collected
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
