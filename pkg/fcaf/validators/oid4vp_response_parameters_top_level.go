// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"sort"
)

// oid4vpResponseParameterNames are the Authorization Response members defined
// by OpenID4VP 1.0 Section 8.1 and the OAuth 2.0 responses it reuses. Finding
// one of them below the root of the decrypted payload is the observable form
// of the sub-object nesting that Section 8.3 forbids.
var oid4vpResponseParameterNames = map[string]struct{}{
	"vp_token":          {},
	"state":             {},
	"error":             {},
	"error_description": {},
	"id_token":          {},
	"code":              {},
	"iss":               {},
}

// OID4VPResponseParametersTopLevelValidator verifies that the decrypted
// Authorization Response JWT carries the response contents as top-level JSON
// members, as required by OpenID4VP 1.0 Section 8.3. A Wallet that wraps the
// response in a sub-object fails either because a required member is missing
// from the root or because a response parameter is reachable one level down.
type OID4VPResponseParametersTopLevelValidator struct{}

func (OID4VPResponseParametersTopLevelValidator) ID() string {
	return "oid4vp.response_parameters_top_level"
}

func (OID4VPResponseParametersTopLevelValidator) Validate(
	_ context.Context,
	input Input,
) Result {
	params, err := DecodeParams[struct {
		Required []string `json:"required"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	required := params.Required
	if len(required) == 0 {
		required = []string{"vp_token"}
	}

	payload, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: "decrypted authorization response is not a JSON object",
		}
	}

	for _, name := range required {
		if _, exists := payload[name]; !exists {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"decrypted authorization response has no top-level %q member",
					name,
				),
			}
		}
	}

	for _, name := range sortedKeys(payload) {
		if _, isResponseParameter := oid4vpResponseParameterNames[name]; isResponseParameter {
			continue
		}
		nested, ok := normalizeJSONObject(payload[name])
		if !ok {
			continue
		}
		for _, nestedName := range sortedKeys(nested) {
			if _, isResponseParameter := oid4vpResponseParameterNames[nestedName]; isResponseParameter {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"decrypted authorization response nests %q inside the %q sub-object",
						nestedName,
						name,
					),
				}
			}
		}
	}

	return Result{
		Status:  StatusPass,
		Message: "decrypted authorization response carries the response parameters at the top level",
	}
}

func sortedKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
