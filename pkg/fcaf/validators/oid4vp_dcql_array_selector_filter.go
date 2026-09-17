// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"reflect"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

// OID4VPDCQLArraySelectorFilterValidator verifies that a DCQL array selector
// removes objects that lack the selected key.
type OID4VPDCQLArraySelectorFilterValidator struct{}

// ID returns the validator identifier.
func (OID4VPDCQLArraySelectorFilterValidator) ID() string {
	return "oid4vp.dcql_array_selector_filter"
}

// Validate checks the request path and the degree values retained in the
// resulting SD-JWT presentation.
func (OID4VPDCQLArraySelectorFilterValidator) Validate(_ context.Context, input Input) Result {
	root, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{Status: StatusFail, Message: "DCQL evidence is not an object"}
	}
	queryValue, found := findObjectKey(root, "dcql_query")
	if !found {
		return Result{Status: StatusFail, Message: "captured evidence does not contain dcql_query"}
	}
	query, ok := normalizeJSONObject(queryValue)
	if !ok || !isDegreeArraySelectorQuery(query) {
		return Result{
			Status:  StatusFail,
			Message: "captured DCQL query is not the degree type array selector",
		}
	}

	responseValue, found := findObjectKey(root, "vp_token")
	if !found {
		return Result{Status: StatusFail, Message: "captured evidence does not contain vp_token"}
	}
	response, ok := normalizeJSONObject(responseValue)
	if !ok {
		return Result{Status: StatusFail, Message: "wallet vp_token is not an object"}
	}
	presentations, ok := response["degree"].([]any)
	if !ok || len(presentations) != 1 {
		return Result{
			Status:  StatusFail,
			Message: "wallet vp_token must contain one degree presentation",
		}
	}
	token, ok := presentations[0].(string)
	if !ok || token == "" {
		return Result{Status: StatusFail, Message: "degree presentation is not an SD-JWT"}
	}
	presentation, err := evidence.ParseSDJWTPresentation(token)
	if err != nil {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("degree presentation is invalid: %v", err),
		}
	}
	if presentation.Claims["vct"] != "urn:credimi:degree:1" {
		return Result{Status: StatusFail, Message: "degree presentation has an unexpected vct"}
	}
	if !hasRetainedDegreeTypes(presentation.Claims["degrees"]) {
		return Result{
			Status:  StatusFail,
			Message: "degree presentation does not retain exactly the entries with a type",
		}
	}
	return Result{
		Status:  StatusPass,
		Message: "degree array selector retained only entries with a type",
	}
}

func isDegreeArraySelectorQuery(query map[string]any) bool {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) != 1 {
		return false
	}
	credential, ok := normalizeJSONObject(credentials[0])
	if !ok || credential["id"] != "degree" || credential["format"] != "dc+sd-jwt" {
		return false
	}
	meta, ok := normalizeJSONObject(credential["meta"])
	if !ok || !reflect.DeepEqual(meta["vct_values"], []any{"urn:credimi:degree:1"}) {
		return false
	}
	claims, ok := credential["claims"].([]any)
	if !ok || len(claims) != 1 {
		return false
	}
	claim, ok := normalizeJSONObject(claims[0])
	return ok && reflect.DeepEqual(claim["path"], []any{"degrees", nil, "type"})
}

func hasRetainedDegreeTypes(value any) bool {
	degrees, ok := value.([]any)
	if !ok || len(degrees) != 2 {
		return false
	}
	expected := []string{"Bachelor of Science", "Master of Science"}
	for index, expectedType := range expected {
		degree, ok := normalizeJSONObject(degrees[index])
		if !ok || len(degree) != 1 || degree["type"] != expectedType {
			return false
		}
	}
	return true
}
