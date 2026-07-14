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
		Mode         string `json:"mode"`
		Property     string `json:"property"`
		ExpectedType string `json:"expected_type"`
		Valid        bool   `json:"valid"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	switch params.Mode {
	case "credential_sets",
		"credential_sets_options_missing",
		"credential_sets_options_empty",
		"credential_sets_options_non_array",
		"credential_sets_options_valid_references",
		"credential_sets_options_invalid_references",
		"credentials_match",
		"without_credential_sets",
		"without_trusted_authorities",
		"without_claims",
		"empty_claims",
		"empty_array",
		"property_type",
		"trusted_authority_property_type",
		"trusted_authority_array_item_type",
		"trusted_authority_empty_string_item",
		"multiple_default_false",
		"multiple_true",
		"no_match",
		"request_rejected",
		"claim_sets":
	default:
		return Result{
			Status:  StatusError,
			Message: "mode must be credential_sets, credentials_match, without_credential_sets, without_trusted_authorities, without_claims, empty_claims, empty_array, property_type, trusted_authority_property_type, trusted_authority_array_item_type, trusted_authority_empty_string_item, multiple_default_false, multiple_true, no_match, request_rejected, or claim_sets",
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
	case "credential_sets_options_missing":
		credentials, ok := query["credentials"].([]any)
		if !ok || len(credentials) == 0 {
			return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
		}
		sets, ok := query["credential_sets"].([]any)
		if !ok || len(sets) == 0 {
			return Result{Status: StatusFail, Message: "dcql_query does not contain non-empty credential_sets"}
		}
		for index, rawSet := range sets {
			set, ok := normalizeJSONObject(rawSet)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credential_sets[%d] is not an object", index)}
			}
			if _, exists := set["options"]; exists {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credential_sets[%d].options is present", index)}
			}
		}
		if !isEmptyDCQLValue(responseValue) {
			return Result{Status: StatusFail, Message: "wallet returned a vp_token for a query missing credential_sets.options"}
		}
		return Result{Status: StatusPass, Message: "wallet rejected credential_sets without options"}
	case "credential_sets_options_empty", "credential_sets_options_non_array", "credential_sets_options_valid_references", "credential_sets_options_invalid_references":
		return validateCredentialSetsOptions(query, responseValue, params.Mode)
	case "credentials_match",
		"without_credential_sets",
		"without_trusted_authorities",
		"without_claims",
		"multiple_default_false",
		"multiple_true":
		if params.Mode == "without_credential_sets" {
			if _, exists := query["credential_sets"]; exists {
				return Result{
					Status:  StatusFail,
					Message: "dcql_query contains credential_sets",
				}
			}
		}
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
			if params.Mode == "without_trusted_authorities" {
				if _, exists := credential["trusted_authorities"]; exists {
					return Result{
						Status:  StatusFail,
						Message: fmt.Sprintf("credentials[%d] contains trusted_authorities", index),
					}
				}
			}
			if params.Mode == "without_claims" {
				if _, exists := credential["claims"]; exists {
					return Result{
						Status:  StatusFail,
						Message: fmt.Sprintf("credentials[%d] contains claims", index),
					}
				}
			}
			id, _ := credential["id"].(string)
			if id == "" {
				return Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("credentials[%d] has no id", index),
				}
			}
			presentation := response[id]
			if isEmptyDCQLValue(presentation) {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"vp_token has no presentation for credential query %q",
						id,
					),
				}
			}
			if params.Mode == "multiple_default_false" {
				if _, exists := credential["multiple"]; exists {
					return Result{
						Status:  StatusFail,
						Message: fmt.Sprintf("credentials[%d] contains multiple", index),
					}
				}
				presentations, ok := presentation.([]any)
				if !ok || len(presentations) != 1 {
					return Result{
						Status: StatusFail,
						Message: fmt.Sprintf(
							"vp_token must contain exactly one presentation for credential query %q",
							id,
						),
					}
				}
			}
			if params.Mode == "multiple_true" {
				multiple, ok := credential["multiple"].(bool)
				if !ok || !multiple {
					return Result{
						Status:  StatusFail,
						Message: fmt.Sprintf("credentials[%d].multiple is not true", index),
					}
				}
				presentations, ok := presentation.([]any)
				if !ok || len(presentations) < 2 {
					return Result{
						Status: StatusFail,
						Message: fmt.Sprintf(
							"vp_token must contain multiple presentations for credential query %q",
							id,
						),
					}
				}
			}
		}
	case "empty_claims", "empty_array":
		property := "claims"
		if params.Mode == "empty_array" {
			property = params.Property
			if property == "" {
				return Result{
					Status:  StatusError,
					Message: "property is required for empty_array mode",
				}
			}
		}
		credentials, ok := query["credentials"].([]any)
		if !ok || len(credentials) == 0 {
			return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
		}
		for index, rawCredential := range credentials {
			credential, ok := normalizeJSONObject(rawCredential)
			if !ok {
				return Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("credentials[%d] is not an object", index),
				}
			}
			value, exists := credential[property]
			if !exists {
				return Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("credentials[%d] does not contain %s", index, property),
				}
			}
			items, ok := value.([]any)
			if !ok || len(items) != 0 {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"credentials[%d].%s is not an empty array",
						index,
						property,
					),
				}
			}
		}
		if !isEmptyDCQLValue(responseValue) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"wallet returned a credential for a query with empty %s",
					property,
				),
			}
		}
	case "no_match", "request_rejected":
		credentials, ok := query["credentials"].([]any)
		if !ok || len(credentials) == 0 {
			return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
		}
		if err := validateDCQLCredentialQueries(credentials); err != nil {
			return Result{Status: StatusFail, Message: err.Error()}
		}
		if errorText, _ := errorValue.(string); errorText == "invalid_request" {
			break
		}
		if !isEmptyDCQLValue(responseValue) {
			return Result{
				Status:  StatusFail,
				Message: "wallet returned a credential for a no-match DCQL query",
			}
		}
	case "property_type":
		if params.Property == "" {
			return Result{
				Status:  StatusError,
				Message: "property is required for property_type mode",
			}
		}
		if _, exists := input.Params["valid"]; !exists {
			return Result{Status: StatusError, Message: "valid is required for property_type mode"}
		}
		if !supportedJSONType(params.ExpectedType) {
			return Result{
				Status:  StatusError,
				Message: "expected_type must be boolean, string, number, integer, array, object, or null",
			}
		}
		credentials, ok := query["credentials"].([]any)
		if !ok || len(credentials) == 0 {
			return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
		}
		for index, rawCredential := range credentials {
			credential, ok := normalizeJSONObject(rawCredential)
			if !ok {
				return Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("credentials[%d] is not an object", index),
				}
			}
			value, exists := credential[params.Property]
			if !exists {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"credentials[%d] does not contain %s",
						index,
						params.Property,
					),
				}
			}
			matches := matchesJSONType(value, params.ExpectedType)
			if matches != params.Valid {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"credentials[%d].%s type validity is %t, expected %t",
						index,
						params.Property,
						matches,
						params.Valid,
					),
				}
			}
		}
		if params.Valid && isEmptyDCQLValue(responseValue) {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("wallet returned no credential for valid %s", params.Property),
			}
		}
		if !params.Valid && !isEmptyDCQLValue(responseValue) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"wallet returned a credential for invalid %s type",
					params.Property,
				),
			}
		}
	case "trusted_authority_array_item_type":
		if params.Property == "" {
			return Result{Status: StatusError, Message: "property is required for trusted_authority_array_item_type mode"}
		}
		if _, exists := input.Params["valid"]; !exists {
			return Result{Status: StatusError, Message: "valid is required for trusted_authority_array_item_type mode"}
		}
		if !supportedJSONType(params.ExpectedType) || params.ExpectedType != "array" {
			return Result{Status: StatusError, Message: "expected_type must be array for trusted_authority_array_item_type mode"}
		}
		itemExpectedType, ok := input.Params["item_expected_type"].(string)
		if !ok || !supportedJSONType(itemExpectedType) {
			return Result{Status: StatusError, Message: "item_expected_type must be boolean, string, number, integer, array, object, or null"}
		}
		itemValid, ok := input.Params["item_valid"].(bool)
		if !ok {
			return Result{Status: StatusError, Message: "item_valid must be a boolean"}
		}
		credentials, ok := query["credentials"].([]any)
		if !ok || len(credentials) == 0 {
			return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
		}
		if err := validateDCQLCredentialQueries(credentials); err != nil {
			return Result{Status: StatusFail, Message: err.Error()}
		}
		for credentialIndex, rawCredential := range credentials {
			credential, _ := normalizeJSONObject(rawCredential)
			authorities, ok := credential["trusted_authorities"].([]any)
			if !ok || len(authorities) == 0 {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities is not a non-empty array", credentialIndex)}
			}
			for authorityIndex, rawAuthority := range authorities {
				authority, ok := normalizeJSONObject(rawAuthority)
				if !ok {
					return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d] is not an object", credentialIndex, authorityIndex)}
				}
				if authorityType, ok := authority["type"].(string); !ok || authorityType == "" {
					return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d].type is not a non-empty string", credentialIndex, authorityIndex)}
				}
				values, ok := authority[params.Property].([]any)
				if !ok || len(values) == 0 {
					return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d].%s is not a non-empty array", credentialIndex, authorityIndex, params.Property)}
				}
				foundInvalidItem := false
				for itemIndex, item := range values {
					matches := matchesJSONType(item, itemExpectedType)
					if !itemValid && !matches {
						foundInvalidItem = true
					}
					if itemValid && !matches {
						return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d].%s[%d] type validity is %t, expected %t", credentialIndex, authorityIndex, params.Property, itemIndex, matches, itemValid)}
					}
				}
				if !itemValid && !foundInvalidItem {
					return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d].%s contains no invalid item", credentialIndex, authorityIndex, params.Property)}
				}
			}
		}
		if !params.Valid && !isEmptyDCQLValue(responseValue) {
			return Result{Status: StatusFail, Message: fmt.Sprintf("wallet returned a credential for invalid trusted authority %s item type", params.Property)}
		}
	case "trusted_authority_property_type":
		if params.Property == "" {
			return Result{
				Status:  StatusError,
				Message: "property is required for trusted_authority_property_type mode",
			}
		}
		if _, exists := input.Params["valid"]; !exists {
			return Result{
				Status:  StatusError,
				Message: "valid is required for trusted_authority_property_type mode",
			}
		}
		if !supportedJSONType(params.ExpectedType) {
			return Result{
				Status:  StatusError,
				Message: "expected_type must be boolean, string, number, integer, array, object, or null",
			}
		}
		credentials, ok := query["credentials"].([]any)
		if !ok || len(credentials) == 0 {
			return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
		}
		if err := validateDCQLCredentialQueries(credentials); err != nil {
			return Result{Status: StatusFail, Message: err.Error()}
		}
		for credentialIndex, rawCredential := range credentials {
			credential, _ := normalizeJSONObject(rawCredential)
			authorities, ok := credential["trusted_authorities"].([]any)
			if !ok || len(authorities) == 0 {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"credentials[%d].trusted_authorities is not a non-empty array",
						credentialIndex,
					),
				}
			}
			for authorityIndex, rawAuthority := range authorities {
				authority, ok := normalizeJSONObject(rawAuthority)
				if !ok {
					return Result{
						Status: StatusFail,
						Message: fmt.Sprintf(
							"credentials[%d].trusted_authorities[%d] is not an object",
							credentialIndex,
							authorityIndex,
						),
					}
				}
				value, exists := authority[params.Property]
				if !exists {
					return Result{
						Status: StatusFail,
						Message: fmt.Sprintf(
							"credentials[%d].trusted_authorities[%d] does not contain %s",
							credentialIndex,
							authorityIndex,
							params.Property,
						),
					}
				}
				if params.Property == "type" && !nonEmptyStringArray(authority["values"]) {
					return Result{
						Status: StatusFail,
						Message: fmt.Sprintf(
							"credentials[%d].trusted_authorities[%d].values is not a non-empty string array",
							credentialIndex,
							authorityIndex,
						),
					}
				}
				if params.Property == "values" {
					authorityType, ok := authority["type"].(string)
					if !ok || authorityType == "" {
						return Result{
							Status: StatusFail,
							Message: fmt.Sprintf(
								"credentials[%d].trusted_authorities[%d].type is not a non-empty string",
								credentialIndex,
								authorityIndex,
							),
						}
					}
				}
				matches := matchesJSONType(value, params.ExpectedType)
				if matches != params.Valid {
					return Result{
						Status: StatusFail,
						Message: fmt.Sprintf(
							"credentials[%d].trusted_authorities[%d].%s type validity is %t, expected %t",
							credentialIndex,
							authorityIndex,
							params.Property,
							matches,
							params.Valid,
						),
					}
				}
			}
		}
		if params.Valid && isEmptyDCQLValue(responseValue) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"wallet returned no credential for valid trusted authority %s",
					params.Property,
				),
			}
		}
		if !params.Valid && !isEmptyDCQLValue(responseValue) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"wallet returned a credential for invalid trusted authority %s type",
					params.Property,
				),
			}
		}
	case "trusted_authority_empty_string_item":
		if params.Property == "" {
			return Result{Status: StatusError, Message: "property is required for trusted_authority_empty_string_item mode"}
		}
		credentials, ok := query["credentials"].([]any)
		if !ok || len(credentials) == 0 {
			return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
		}
		if err := validateDCQLCredentialQueries(credentials); err != nil {
			return Result{Status: StatusFail, Message: err.Error()}
		}
		for credentialIndex, rawCredential := range credentials {
			credential, _ := normalizeJSONObject(rawCredential)
			authorities, ok := credential["trusted_authorities"].([]any)
			if !ok || len(authorities) == 0 {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities is not a non-empty array", credentialIndex)}
			}
			for authorityIndex, rawAuthority := range authorities {
				authority, ok := normalizeJSONObject(rawAuthority)
				if !ok {
					return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d] is not an object", credentialIndex, authorityIndex)}
				}
				if authorityType, ok := authority["type"].(string); !ok || authorityType == "" {
					return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d].type is not a non-empty string", credentialIndex, authorityIndex)}
				}
				values, ok := authority[params.Property].([]any)
				if !ok || len(values) == 0 {
					return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d].%s is not a non-empty array", credentialIndex, authorityIndex, params.Property)}
				}
				foundEmpty := false
				for itemIndex, item := range values {
					itemString, ok := item.(string)
					if !ok {
						return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d].%s[%d] is not a string", credentialIndex, authorityIndex, params.Property, itemIndex)}
					}
					if itemString == "" {
						foundEmpty = true
					}
				}
				if !foundEmpty {
					return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d].%s contains no empty string item", credentialIndex, authorityIndex, params.Property)}
				}
			}
		}
		if !isEmptyDCQLValue(responseValue) {
			return Result{Status: StatusFail, Message: fmt.Sprintf("wallet returned a credential for trusted authority %s containing an empty string", params.Property)}
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

func supportedJSONType(expected string) bool {
	switch expected {
	case "boolean", "string", "number", "integer", "array", "object", "null":
		return true
	default:
		return false
	}
}

func matchesJSONType(value any, expected string) bool {
	switch expected {
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return true
		default:
			return false
		}
	case "integer":
		switch typed := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		case float32:
			return typed == float32(int64(typed))
		case float64:
			return typed == float64(int64(typed))
		default:
			return false
		}
	case "array":
		_, ok := value.([]any)
		return ok
	case "object":
		_, ok := normalizeJSONObject(value)
		return ok
	case "null":
		return value == nil
	default:
		return false
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
				return fmt.Errorf(
					"credentials[%d].meta.vct_values is not a non-empty string array",
					index,
				)
			}
		case "mso_mdoc":
			docType, _ := meta["doctype_value"].(string)
			if docType == "" {
				return fmt.Errorf("credentials[%d].meta.doctype_value is missing", index)
			}
		default:
			return fmt.Errorf("credentials[%d].format %q is not supported", index, format)
		}
		if claims, exists := credential["claims"]; exists {
			items, ok := claims.([]any)
			if !ok || len(items) == 0 {
				return fmt.Errorf("credentials[%d].claims is not a non-empty array", index)
			}
			for claimIndex, rawClaim := range items {
				claim, ok := normalizeJSONObject(rawClaim)
				if !ok || !nonEmptyStringArray(claim["path"]) {
					return fmt.Errorf(
						"credentials[%d].claims[%d].path is invalid",
						index,
						claimIndex,
					)
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

func validateCredentialSetsOptions(query map[string]any, responseValue any, mode string) Result {
	credentials, ok := query["credentials"].([]any)
	sets, setsOK := query["credential_sets"].([]any)
	if !ok || len(credentials) == 0 || !setsOK || len(sets) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query must contain credentials and credential_sets"}
	}
	ids := make(map[string]struct{}, len(credentials))
	for _, raw := range credentials {
		credential, ok := normalizeJSONObject(raw)
		if !ok {
			return Result{Status: StatusFail, Message: "dcql credential is not an object"}
		}
		id, ok := credential["id"].(string)
		if !ok || id == "" {
			return Result{Status: StatusFail, Message: "dcql credential id is invalid"}
		}
		ids[id] = struct{}{}
	}
	invalid := false
	for _, raw := range sets {
		set, ok := normalizeJSONObject(raw)
		if !ok {
			invalid = true
			continue
		}
		options, exists := set["options"]
		if !exists {
			invalid = true
			continue
		}
		groups, ok := options.([]any)
		if mode == "credential_sets_options_non_array" {
			if ok {
				return Result{Status: StatusFail, Message: "credential_sets.options is an array"}
			}
			invalid = true
			continue
		}
		if !ok || len(groups) == 0 {
			invalid = true
			continue
		}
		for _, rawGroup := range groups {
			group, ok := rawGroup.([]any)
			if !ok || len(group) == 0 {
				invalid = true
				continue
			}
			for _, rawID := range group {
				id, ok := rawID.(string)
				if !ok {
					invalid = true
					continue
				}
				if _, found := ids[id]; !found {
					invalid = true
				}
			}
		}
	}
	if mode == "credential_sets_options_valid_references" && invalid {
		return Result{Status: StatusFail, Message: "credential_sets.options contains invalid references"}
	}
	if mode == "credential_sets_options_invalid_references" && !invalid {
		return Result{Status: StatusFail, Message: "credential_sets.options contains no invalid references"}
	}
	if mode == "credential_sets_options_empty" && !invalid {
		return Result{Status: StatusFail, Message: "credential_sets.options is non-empty"}
	}
	if mode == "credential_sets_options_non_array" || mode == "credential_sets_options_empty" || mode == "credential_sets_options_invalid_references" {
		if !isEmptyDCQLValue(responseValue) {
			return Result{Status: StatusFail, Message: "wallet returned a vp_token for an invalid credential_sets.options query"}
		}
		return Result{Status: StatusPass, Message: "wallet rejected invalid credential_sets.options"}
	}
	if isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned no vp_token for valid credential_sets.options references"}
	}
	return Result{Status: StatusPass, Message: "wallet processed valid credential_sets.options references"}
}
