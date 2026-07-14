// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
)

var dcqlIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type DCQLResponseConstraintsValidator struct{}

func (DCQLResponseConstraintsValidator) ID() string {
	return "dcql.response_satisfies_constraints"
}

func (DCQLResponseConstraintsValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Mode string `json:"mode"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	switch params.Mode {
	case "credential_sets", "credentials_match", "no_match", "claim_sets":
	default:
		return Result{
			Status:  StatusError,
			Message: "mode must be credential_sets, credentials_match, no_match, or claim_sets",
		}
	}

	root, ok := normalizeJSONObject(input.Value)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("DCQL evidence is %T, expected object", input.Value),
		}
	}
	queryValue, found := findObjectKey(root, "dcql_query")
	if !found {
		return Result{Status: StatusFail, Message: "captured evidence does not contain dcql_query"}
	}
	query, ok := normalizeJSONObject(queryValue)
	if !ok {
		return Result{Status: StatusFail, Message: "captured dcql_query is not an object"}
	}

	responseValue, _ := findObjectKey(root, "vp_token")
	errorValue, _ := findObjectKey(root, "error")
	switch params.Mode {
	case "credential_sets":
		sets, ok := query["credential_sets"].([]any)
		if !ok || len(sets) == 0 {
			return Result{
				Status:  StatusFail,
				Message: "dcql_query does not contain non-empty credential_sets",
			}
		}
		if isEmptyDCQLValue(responseValue) {
			return Result{
				Status:  StatusFail,
				Message: "wallet response contains no vp_token for credential_sets",
			}
		}
	case "credentials_match":
		credentials, ok := query["credentials"].([]any)
		if !ok || len(credentials) == 0 {
			return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
		}
		if err := validateDCQLCredentialQueries(credentials); err != nil {
			return Result{Status: StatusFail, Message: err.Error()}
		}
		response, ok := normalizeJSONObject(responseValue)
		if !ok {
			return Result{
				Status:  StatusFail,
				Message: "wallet vp_token is not an object keyed by credential query ID",
			}
		}
		for index, rawCredential := range credentials {
			credential, ok := normalizeJSONObject(rawCredential)
			if !ok {
				return Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("credentials[%d] is not an object", index),
				}
			}
			id, _ := credential["id"].(string)
			if id == "" {
				return Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("credentials[%d] has no id", index),
				}
			}
			if isEmptyDCQLValue(response[id]) {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"vp_token has no presentation for credential query %q",
						id,
					),
				}
			}
		}
	case "no_match":
		if errorText, _ := errorValue.(string); errorText == "invalid_request" {
			break
		}
		if !isEmptyDCQLValue(responseValue) {
			return Result{
				Status:  StatusFail,
				Message: "wallet returned a credential for a no-match DCQL query",
			}
		}
	case "claim_sets":
		credentials, ok := query["credentials"].([]any)
		if !ok || !containsClaimSets(credentials) {
			return Result{
				Status:  StatusFail,
				Message: "dcql_query credentials contain no claim_sets",
			}
		}
		if isEmptyDCQLValue(responseValue) {
			return Result{
				Status:  StatusFail,
				Message: "wallet response contains no vp_token for claim_sets",
			}
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("wallet response satisfies DCQL %s constraints", params.Mode),
	}
}

func validateDCQLCredentialQueries(credentials []any) error {
	ids := make(map[string]struct{}, len(credentials))
	for index, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return fmt.Errorf("credentials[%d] is not an object", index)
		}
		id, _ := credential["id"].(string)
		if !dcqlIDPattern.MatchString(id) {
			return fmt.Errorf("credentials[%d].id is not a valid DCQL identifier", index)
		}
		if _, duplicate := ids[id]; duplicate {
			return fmt.Errorf("credentials[%d].id %q is duplicated", index, id)
		}
		ids[id] = struct{}{}

		format, _ := credential["format"].(string)
		if format == "" {
			return fmt.Errorf("credentials[%d].format is missing", index)
		}
		meta, ok := normalizeJSONObject(credential["meta"])
		if !ok {
			return fmt.Errorf("credentials[%d].meta is not an object", index)
		}
		switch format {
		case "dc+sd-jwt":
			if !nonEmptyStringArray(meta["vct_values"]) {
				return fmt.Errorf("credentials[%d].meta.vct_values is not a non-empty string array", index)
			}
		case "mso_mdoc":
			docType, _ := meta["doctype_value"].(string)
			if docType == "" {
				return fmt.Errorf("credentials[%d].meta.doctype_value is missing", index)
			}
		}
		if claims, exists := credential["claims"]; exists {
			items, ok := claims.([]any)
			if !ok || len(items) == 0 {
				return fmt.Errorf("credentials[%d].claims is not a non-empty array", index)
			}
			for claimIndex, rawClaim := range items {
				claim, ok := normalizeJSONObject(rawClaim)
				if !ok || !nonEmptyStringArray(claim["path"]) {
					return fmt.Errorf("credentials[%d].claims[%d].path is invalid", index, claimIndex)
				}
			}
		}
	}
	return nil
}

func nonEmptyStringArray(value any) bool {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return false
	}
	for _, item := range items {
		text, ok := item.(string)
		if !ok || text == "" {
			return false
		}
	}
	return true
}

func normalizeJSONObject(value any) (map[string]any, bool) {
	if object, ok := value.(map[string]any); ok {
		return object, true
	}
	text, ok := value.(string)
	if !ok {
		return nil, false
	}
	var object map[string]any
	if err := json.Unmarshal([]byte(text), &object); err != nil {
		return nil, false
	}
	return object, true
}

func findObjectKey(value any, key string) (any, bool) {
	object, ok := normalizeJSONObject(value)
	if !ok {
		return nil, false
	}
	if found, exists := object[key]; exists {
		return found, true
	}
	for _, child := range object {
		if found, exists := findObjectKey(child, key); exists {
			return found, true
		}
		if array, ok := child.([]any); ok {
			for _, item := range array {
				if found, exists := findObjectKey(item, key); exists {
					return found, true
				}
			}
		}
	}
	return nil, false
}

func containsClaimSets(credentials []any) bool {
	for _, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			continue
		}
		claimSets, ok := credential["claim_sets"].([]any)
		if ok && len(claimSets) > 0 {
			return true
		}
	}
	return false
}

func isEmptyDCQLValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return typed == ""
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}
