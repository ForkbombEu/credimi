// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

var dcqlIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type DCQLResponseConstraintsValidator struct{}

func (DCQLResponseConstraintsValidator) ID() string {
	return "dcql.response_satisfies_constraints"
}

func (DCQLResponseConstraintsValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Mode          string `json:"mode"`
		Property      string `json:"property"`
		ExpectedType  string `json:"expected_type"`
		Valid         bool   `json:"valid"`
		ExpectedValue any    `json:"expected_value"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	switch params.Mode {
	case "credential_sets",
		"claims_present",
		"claims_path_no_match",
		"claims_values_no_match",
		"claim_id_missing_with_claim_sets",
		"claims_without_id_without_claim_sets",
		"duplicate_claim_ids",
		"empty_claim_id",
		"invalid_claim_id_characters",
		"claim_path_missing",
		"claim_path_empty",
		"claim_path_non_array",
		"claim_path_allowed_components",
		"claims_without_values",
		"credential_sets_options_missing",
		"credential_sets_options_empty",
		"credential_sets_options_non_array",
		"credential_sets_options_valid_references",
		"credential_sets_options_invalid_references",
		"credential_sets_required_true_match",
		"credential_sets_required_true_no_match",
		"credential_sets_required_omitted",
		"credential_sets_required_false_with_match",
		"credentials_match",
		"without_credential_sets",
		"without_trusted_authorities",
		"without_claims",
		"empty_claims",
		"empty_array",
		"property_type",
		"property_equals",
		"trusted_authority_property_type",
		"trusted_authority_array_item_type",
		"trusted_authority_empty_string_item",
		"multiple_default_false",
		"multiple_true",
		"no_match",
		"request_rejected",
		"trusted_authorities_match",
		"trusted_authorities_no_match",
		"claim_sets":
	default:
		return Result{
			Status:  StatusError,
			Message: "mode must be credential_sets, credentials_match, without_credential_sets, without_trusted_authorities, without_claims, empty_claims, empty_array, property_type, property_equals, trusted_authority_property_type, trusted_authority_array_item_type, trusted_authority_empty_string_item, multiple_default_false, multiple_true, no_match, request_rejected, trusted_authorities_match, trusted_authorities_no_match, or claim_sets",
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
	case "claims_present":
		return validateClaimsPresent(query, responseValue)
	case "claims_path_no_match":
		return validateClaimsPathNoMatch(query, responseValue)
	case "claims_values_no_match":
		return validateClaimsValuesNoMatch(query, responseValue)
	case "claim_id_missing_with_claim_sets":
		return validateMissingClaimIDWithClaimSets(query, responseValue)
	case "claims_without_id_without_claim_sets":
		return validateClaimsWithoutIDWithoutClaimSets(query, responseValue)
	case "duplicate_claim_ids":
		return validateDuplicateClaimIDs(query, responseValue, errorValue)
	case "empty_claim_id":
		return validateEmptyClaimID(query, responseValue, errorValue)
	case "invalid_claim_id_characters":
		return validateInvalidClaimIDCharacters(query, responseValue, errorValue)
	case "claim_path_missing":
		return validateMissingClaimPath(query, responseValue, errorValue)
	case "claim_path_empty":
		return validateEmptyClaimPath(query, responseValue, errorValue)
	case "claim_path_non_array":
		return validateNonArrayClaimPath(query, responseValue, errorValue)
	case "claim_path_allowed_components":
		return validateAllowedClaimPathComponents(query, responseValue)
	case "claims_without_values":
		return validateClaimsWithoutValues(query, responseValue)
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
		if errorValue != "invalid_request" {
			return Result{Status: StatusFail, Message: "wallet did not return invalid_request for a query missing credential_sets.options"}
		}
		return Result{Status: StatusPass, Message: "wallet returned invalid_request for credential_sets without options"}
	case "credential_sets_options_empty", "credential_sets_options_non_array", "credential_sets_options_valid_references", "credential_sets_options_invalid_references":
		return validateCredentialSetsOptions(query, responseValue, errorValue, params.Mode)
	case "credential_sets_required_true_match", "credential_sets_required_true_no_match", "credential_sets_required_omitted", "credential_sets_required_false_with_match":
		return validateCredentialSetsRequired(query, responseValue, params.Mode)
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
	case "trusted_authorities_match":
		return validateTrustedAuthoritiesMatch(query, responseValue)
	case "trusted_authorities_no_match":
		return validateTrustedAuthoritiesNoMatch(query, responseValue)
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
	case "property_equals":
		if params.Property == "" {
			return Result{Status: StatusError, Message: "property is required for property_equals mode"}
		}
		if _, exists := input.Params["expected_value"]; !exists {
			return Result{Status: StatusError, Message: "expected_value is required for property_equals mode"}
		}
		credentials, ok := query["credentials"].([]any)
		if !ok || len(credentials) == 0 {
			return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
		}
		for index, rawCredential := range credentials {
			credential, ok := normalizeJSONObject(rawCredential)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", index)}
			}
			value, exists := credential[params.Property]
			if !exists || !reflect.DeepEqual(value, params.ExpectedValue) {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].%s does not equal the expected value", index, params.Property)}
			}
		}
		if isEmptyDCQLValue(responseValue) {
			return Result{Status: StatusFail, Message: fmt.Sprintf("wallet returned no credential for %s", params.Property)}
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

func validateCredentialSetsOptions(query map[string]any, responseValue, errorValue any, mode string) Result {
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
		if (mode == "credential_sets_options_empty" || mode == "credential_sets_options_non_array") && errorValue != "invalid_request" {
			return Result{Status: StatusFail, Message: "wallet did not return invalid_request for an invalid credential_sets.options query"}
		}
		return Result{Status: StatusPass, Message: "wallet rejected invalid credential_sets.options"}
	}
	if isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned no vp_token for valid credential_sets.options references"}
	}
	return Result{Status: StatusPass, Message: "wallet processed valid credential_sets.options references"}
}

func validateCredentialSetsRequired(query map[string]any, responseValue any, mode string) Result {
	sets, ok := query["credential_sets"].([]any)
	if !ok || len(sets) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credential_sets"}
	}
	response, responseOK := normalizeJSONObject(responseValue)
	for index, rawSet := range sets {
		set, ok := normalizeJSONObject(rawSet)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credential_sets[%d] is not an object", index)}
		}
		required, exists := set["required"]
		if mode == "credential_sets_required_true_match" && (!exists || required != true) {
			return Result{Status: StatusFail, Message: "required is not true"}
		}
		if mode == "credential_sets_required_true_no_match" && (!exists || required != true) {
			return Result{Status: StatusFail, Message: "required is not true"}
		}
		if mode == "credential_sets_required_omitted" && exists {
			return Result{Status: StatusFail, Message: "required is present"}
		}
		if mode == "credential_sets_required_false_with_match" && required != false {
			return Result{Status: StatusFail, Message: "required is not false"}
		}
	}
	if mode == "credential_sets_required_true_match" || mode == "credential_sets_required_omitted" || mode == "credential_sets_required_false_with_match" {
		if !responseOK || isEmptyDCQLValue(response) {
			return Result{Status: StatusFail, Message: "wallet returned no vp_token for a satisfiable credential set"}
		}
		return Result{Status: StatusPass, Message: "wallet presented the credential set"}
	}
	if !isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned a presentation for a missing required credential set"}
	}
	return Result{Status: StatusPass, Message: "wallet stopped without presenting a missing required credential set"}
}

func validateClaimsPresent(query map[string]any, responseValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	response, ok := normalizeJSONObject(responseValue)
	if !ok {
		return Result{Status: StatusFail, Message: "wallet vp_token is not an object keyed by credential query ID"}
	}
	for index, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", index)}
		}
		id, ok := credential["id"].(string)
		if !ok || id == "" {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].id is not a non-empty string", index)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", index)}
		}
		for claimIndex, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] is not an object", index, claimIndex)}
			}
			path, ok := claim["path"].([]any)
			if !ok || len(path) == 0 {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path is not a non-empty array", index, claimIndex)}
			}
			for pathIndex, segment := range path {
				if _, ok := segment.(string); !ok {
					return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path[%d] is not a string", index, claimIndex, pathIndex)}
				}
			}
		}
		if isEmptyDCQLValue(response[id]) {
			return Result{Status: StatusFail, Message: fmt.Sprintf("vp_token has no presentation for credential query %q", id)}
		}
	}
	return Result{Status: StatusPass, Message: "wallet processed credential queries with claims"}
}

func validateClaimsPathNoMatch(query map[string]any, responseValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	for index, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", index)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", index)}
		}
		for claimIndex, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] is not an object", index, claimIndex)}
			}
			path, ok := claim["path"].([]any)
			if !ok || len(path) == 0 {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path is not a non-empty array", index, claimIndex)}
			}
		}
	}
	if !isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned a credential for an unmatched claim path"}
	}
	return Result{Status: StatusPass, Message: "wallet returned no credential for the unmatched claim path"}
}

func validateClaimsValuesNoMatch(query map[string]any, responseValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	for credentialIndex, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", credentialIndex)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", credentialIndex)}
		}
		for claimIndex, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] is not an object", credentialIndex, claimIndex)}
			}
			path, pathOK := claim["path"].([]any)
			values, valuesOK := claim["values"].([]any)
			if !pathOK || len(path) == 0 {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path is not a non-empty array", credentialIndex, claimIndex)}
			}
			if !valuesOK || len(values) == 0 {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].values is not a non-empty array", credentialIndex, claimIndex)}
			}
		}
	}
	if !isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned a credential for mismatched claim values"}
	}
	return Result{Status: StatusPass, Message: "wallet returned no credential for mismatched claim values"}
}

func validateMissingClaimIDWithClaimSets(query map[string]any, responseValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	foundMissingID := false
	for credentialIndex, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", credentialIndex)}
		}
		claimSets, ok := credential["claim_sets"].([]any)
		if !ok || len(claimSets) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claim_sets is not a non-empty array", credentialIndex)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", credentialIndex)}
		}
		for _, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				continue
			}
			if _, exists := claim["id"]; !exists {
				foundMissingID = true
			}
		}
	}
	if !foundMissingID {
		return Result{Status: StatusFail, Message: "claims contain no missing id"}
	}
	if !isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned a credential for claims missing id with claim_sets"}
	}
	return Result{Status: StatusPass, Message: "wallet rejected claims missing id with claim_sets"}
}

func validateClaimsWithoutIDWithoutClaimSets(query map[string]any, responseValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	response, ok := normalizeJSONObject(responseValue)
	if !ok {
		return Result{Status: StatusFail, Message: "wallet vp_token is not an object keyed by credential query ID"}
	}
	for credentialIndex, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", credentialIndex)}
		}
		if _, exists := credential["claim_sets"]; exists {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] contains claim_sets", credentialIndex)}
		}
		id, ok := credential["id"].(string)
		if !ok || id == "" {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].id is not a non-empty string", credentialIndex)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", credentialIndex)}
		}
		for claimIndex, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] is not an object", credentialIndex, claimIndex)}
			}
			if _, exists := claim["id"]; exists {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] contains id", credentialIndex, claimIndex)}
			}
			path, ok := claim["path"].([]any)
			if !ok || len(path) == 0 {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path is not a non-empty array", credentialIndex, claimIndex)}
			}
			for pathIndex, segment := range path {
				if value, ok := segment.(string); !ok || value == "" {
					return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path[%d] is not a non-empty string", credentialIndex, claimIndex, pathIndex)}
				}
			}
		}
		if isEmptyDCQLValue(response[id]) {
			return Result{Status: StatusFail, Message: fmt.Sprintf("vp_token has no presentation for credential query %q", id)}
		}
	}
	return Result{Status: StatusPass, Message: "wallet matched claims without ids when claim_sets was absent"}
}

func validateDuplicateClaimIDs(query map[string]any, responseValue any, errorValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	foundDuplicate := false
	for credentialIndex, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", credentialIndex)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", credentialIndex)}
		}
		seen := make(map[string]struct{}, len(claims))
		for claimIndex, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] is not an object", credentialIndex, claimIndex)}
			}
			id, ok := claim["id"].(string)
			if !ok || id == "" {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].id is not a non-empty string", credentialIndex, claimIndex)}
			}
			if _, exists := seen[id]; exists {
				foundDuplicate = true
			}
			seen[id] = struct{}{}
		}
	}
	if !foundDuplicate {
		return Result{Status: StatusFail, Message: "no credential claims array contains a duplicate id"}
	}
	if !isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned a credential for duplicate claim ids"}
	}
	if errorText, _ := errorValue.(string); errorText != "invalid_request" {
		return Result{Status: StatusFail, Message: "wallet did not return invalid_request for duplicate claim ids"}
	}
	return Result{Status: StatusPass, Message: "wallet rejected duplicate claim ids with invalid_request"}
}

func validateEmptyClaimID(query map[string]any, responseValue any, errorValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	foundEmpty := false
	for credentialIndex, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", credentialIndex)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", credentialIndex)}
		}
		for claimIndex, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] is not an object", credentialIndex, claimIndex)}
			}
			idValue, exists := claim["id"]
			if !exists {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].id is missing", credentialIndex, claimIndex)}
			}
			id, ok := idValue.(string)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].id is not a string", credentialIndex, claimIndex)}
			}
			if id == "" {
				foundEmpty = true
			}
		}
	}
	if !foundEmpty {
		return Result{Status: StatusFail, Message: "no claim id is empty"}
	}
	if !isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned a credential for an empty claim id"}
	}
	if errorText, _ := errorValue.(string); errorText != "invalid_request" {
		return Result{Status: StatusFail, Message: "wallet did not return invalid_request for an empty claim id"}
	}
	return Result{Status: StatusPass, Message: "wallet rejected an empty claim id with invalid_request"}
}

func validateInvalidClaimIDCharacters(query map[string]any, responseValue any, errorValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	foundInvalid := false
	for credentialIndex, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", credentialIndex)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", credentialIndex)}
		}
		for claimIndex, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] is not an object", credentialIndex, claimIndex)}
			}
			idValue, exists := claim["id"]
			if !exists {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].id is missing", credentialIndex, claimIndex)}
			}
			id, ok := idValue.(string)
			if !ok || id == "" {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].id is not a non-empty string", credentialIndex, claimIndex)}
			}
			if !dcqlIDPattern.MatchString(id) {
				foundInvalid = true
			}
		}
	}
	if !foundInvalid {
		return Result{Status: StatusFail, Message: "no claim id contains a forbidden character"}
	}
	if !isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned a credential for a malformed claim id"}
	}
	if errorText, _ := errorValue.(string); errorText != "invalid_request" {
		return Result{Status: StatusFail, Message: "wallet did not return invalid_request for a malformed claim id"}
	}
	return Result{Status: StatusPass, Message: "wallet rejected a malformed claim id with invalid_request"}
}

func validateMissingClaimPath(query map[string]any, responseValue any, errorValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	foundMissing := false
	for credentialIndex, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", credentialIndex)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", credentialIndex)}
		}
		for claimIndex, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] is not an object", credentialIndex, claimIndex)}
			}
			if _, exists := claim["path"]; !exists {
				foundMissing = true
			}
		}
	}
	if !foundMissing {
		return Result{Status: StatusFail, Message: "no claim is missing path"}
	}
	if !isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned a credential for a claim missing path"}
	}
	if errorText, _ := errorValue.(string); errorText != "invalid_request" {
		return Result{Status: StatusFail, Message: "wallet did not return invalid_request for a claim missing path"}
	}
	return Result{Status: StatusPass, Message: "wallet rejected a claim missing path with invalid_request"}
}

func validateEmptyClaimPath(query map[string]any, responseValue any, errorValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	foundEmpty := false
	for credentialIndex, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", credentialIndex)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", credentialIndex)}
		}
		for claimIndex, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] is not an object", credentialIndex, claimIndex)}
			}
			pathValue, exists := claim["path"]
			if !exists {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path is missing", credentialIndex, claimIndex)}
			}
			path, ok := pathValue.([]any)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path is not an array", credentialIndex, claimIndex)}
			}
			if len(path) == 0 {
				foundEmpty = true
			}
		}
	}
	if !foundEmpty {
		return Result{Status: StatusFail, Message: "no claim path is empty"}
	}
	if !isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned a credential for an empty claim path"}
	}
	if errorText, _ := errorValue.(string); errorText != "invalid_request" {
		return Result{Status: StatusFail, Message: "wallet did not return invalid_request for an empty claim path"}
	}
	return Result{Status: StatusPass, Message: "wallet rejected an empty claim path with invalid_request"}
}

func validateNonArrayClaimPath(query map[string]any, responseValue any, errorValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	foundNonArray := false
	for credentialIndex, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", credentialIndex)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", credentialIndex)}
		}
		for claimIndex, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] is not an object", credentialIndex, claimIndex)}
			}
			pathValue, exists := claim["path"]
			if !exists {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path is missing", credentialIndex, claimIndex)}
			}
			if _, ok := pathValue.([]any); !ok {
				foundNonArray = true
			}
		}
	}
	if !foundNonArray {
		return Result{Status: StatusFail, Message: "no claim path has a non-array value"}
	}
	if !isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned a credential for a non-array claim path"}
	}
	if errorText, _ := errorValue.(string); errorText != "invalid_request" {
		return Result{Status: StatusFail, Message: "wallet did not return invalid_request for a non-array claim path"}
	}
	return Result{Status: StatusPass, Message: "wallet rejected a non-array claim path with invalid_request"}
}

func validateAllowedClaimPathComponents(query map[string]any, responseValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	response, ok := normalizeJSONObject(responseValue)
	if !ok {
		return Result{Status: StatusFail, Message: "wallet vp_token is not an object keyed by credential query ID"}
	}
	seenString := false
	seenNull := false
	seenNonNegativeInteger := false
	for credentialIndex, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", credentialIndex)}
		}
		id, ok := credential["id"].(string)
		if !ok || id == "" {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].id is not a non-empty string", credentialIndex)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", credentialIndex)}
		}
		paths := make([][]any, 0, len(claims))
		for claimIndex, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] is not an object", credentialIndex, claimIndex)}
			}
			path, ok := claim["path"].([]any)
			if !ok || len(path) == 0 {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path is not a non-empty array", credentialIndex, claimIndex)}
			}
			paths = append(paths, path)
			for componentIndex, component := range path {
				switch typed := component.(type) {
				case string:
					if typed == "" {
						return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path[%d] is an empty string", credentialIndex, claimIndex, componentIndex)}
					}
					seenString = true
				case nil:
					seenNull = true
				default:
					if !isNonNegativeInteger(component) {
						return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path[%d] is not a string, null, or non-negative integer", credentialIndex, claimIndex, componentIndex)}
					}
					seenNonNegativeInteger = true
				}
			}
		}
		presentations, ok := response[id].([]any)
		if !ok || len(presentations) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("vp_token has no presentation for credential query %q", id)}
		}
		for presentationIndex, rawPresentation := range presentations {
			token, ok := rawPresentation.(string)
			if !ok || token == "" {
				return Result{Status: StatusFail, Message: fmt.Sprintf("vp_token[%q][%d] is not an SD-JWT presentation", id, presentationIndex)}
			}
			presentation, err := evidence.ParseSDJWTPresentation(token)
			if err != nil {
				return Result{Status: StatusFail, Message: fmt.Sprintf("vp_token[%q][%d] is not a valid SD-JWT presentation: %v", id, presentationIndex, err)}
			}
			for pathIndex, path := range paths {
				if !claimPathResolves(presentation.Claims, path) {
					return Result{Status: StatusFail, Message: fmt.Sprintf("vp_token[%q][%d] does not disclose a value resolved by claims[%d].path", id, presentationIndex, pathIndex)}
				}
			}
		}
	}
	if !seenString || !seenNull || !seenNonNegativeInteger {
		return Result{Status: StatusFail, Message: "claim paths do not cover string, null, and non-negative integer components"}
	}
	return Result{Status: StatusPass, Message: "wallet resolved claim paths with all allowed component types"}
}

func claimPathResolves(root any, path []any) bool {
	values := []any{root}
	for _, component := range path {
		next := make([]any, 0)
		for _, value := range values {
			switch typed := component.(type) {
			case string:
				object, ok := value.(map[string]any)
				if !ok {
					continue
				}
				if resolved, exists := object[typed]; exists {
					next = append(next, resolved)
				}
			case nil:
				array, ok := value.([]any)
				if ok {
					next = append(next, array...)
				}
			default:
				array, ok := value.([]any)
				if !ok {
					continue
				}
				index, ok := claimPathArrayIndex(typed, len(array))
				if ok {
					next = append(next, array[index])
				}
			}
		}
		if len(next) == 0 {
			return false
		}
		values = next
	}
	return len(values) > 0
}

func claimPathArrayIndex(value any, length int) (int, bool) {
	if !isNonNegativeInteger(value) {
		return 0, false
	}
	var index uint64
	switch typed := value.(type) {
	case int:
		index = uint64(typed)
	case int8:
		index = uint64(typed)
	case int16:
		index = uint64(typed)
	case int32:
		index = uint64(typed)
	case int64:
		index = uint64(typed)
	case uint:
		index = uint64(typed)
	case uint8:
		index = uint64(typed)
	case uint16:
		index = uint64(typed)
	case uint32:
		index = uint64(typed)
	case uint64:
		index = typed
	case float32:
		index = uint64(typed)
	case float64:
		index = uint64(typed)
	default:
		return 0, false
	}
	if index >= uint64(length) {
		return 0, false
	}
	return int(index), true
}

func isNonNegativeInteger(value any) bool {
	switch typed := value.(type) {
	case int:
		return typed >= 0
	case int8:
		return typed >= 0
	case int16:
		return typed >= 0
	case int32:
		return typed >= 0
	case int64:
		return typed >= 0
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		return typed >= 0 && typed == float32(int64(typed))
	case float64:
		return typed >= 0 && typed == float64(int64(typed))
	default:
		return false
	}
}

func validateClaimsWithoutValues(query map[string]any, responseValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	response, ok := normalizeJSONObject(responseValue)
	if !ok {
		return Result{Status: StatusFail, Message: "wallet vp_token is not an object keyed by credential query ID"}
	}
	for credentialIndex, rawCredential := range credentials {
		credential, ok := normalizeJSONObject(rawCredential)
		if !ok {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] is not an object", credentialIndex)}
		}
		id, ok := credential["id"].(string)
		if !ok || id == "" {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].id is not a non-empty string", credentialIndex)}
		}
		claims, ok := credential["claims"].([]any)
		if !ok || len(claims) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims is not a non-empty array", credentialIndex)}
		}
		for claimIndex, rawClaim := range claims {
			claim, ok := normalizeJSONObject(rawClaim)
			if !ok {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] is not an object", credentialIndex, claimIndex)}
			}
			if _, exists := claim["values"]; exists {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d] contains values", credentialIndex, claimIndex)}
			}
			if !nonEmptyStringArray(claim["path"]) {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].claims[%d].path is invalid", credentialIndex, claimIndex)}
			}
		}
		if isEmptyDCQLValue(response[id]) {
			return Result{Status: StatusFail, Message: fmt.Sprintf("vp_token has no presentation for credential query %q", id)}
		}
	}
	return Result{Status: StatusPass, Message: "wallet matched claims without values"}
}

func validateTrustedAuthoritiesMatch(query map[string]any, responseValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	if err := validateDCQLCredentialQueries(credentials); err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	response, ok := normalizeJSONObject(responseValue)
	if !ok {
		return Result{Status: StatusFail, Message: "wallet vp_token is not an object keyed by credential query ID"}
	}
	for credentialIndex, rawCredential := range credentials {
		credential, _ := normalizeJSONObject(rawCredential)
		id, _ := credential["id"].(string)
		presentations, ok := response[id].([]any)
		if !ok || len(presentations) == 0 {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("vp_token has no presentation for credential query %q", id),
			}
		}
		authorities, hasTA := credential["trusted_authorities"].([]any)
		if !hasTA || len(authorities) == 0 {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("credentials[%d] does not contain trusted_authorities", credentialIndex),
			}
		}
		for presentationIndex, rawPresentation := range presentations {
			token, ok := rawPresentation.(string)
			if !ok || token == "" {
				return Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("vp_token[%q][%d] is not an SD-JWT presentation", id, presentationIndex),
				}
			}
			presentation, err := evidence.ParseSDJWTPresentation(token)
			if err != nil {
				return Result{
					Status:  StatusFail,
					Message: fmt.Sprintf("vp_token[%q][%d] is not a valid SD-JWT: %v", id, presentationIndex, err),
				}
			}
			if !credentialMatchesTrustedAuthorities(presentation, authorities) {
				return Result{
					Status: StatusFail,
					Message: fmt.Sprintf(
						"vp_token[%q][%d] issuer does not match any trusted_authority",
						id,
						presentationIndex,
					),
				}
			}
		}
	}
	return Result{
		Status:  StatusPass,
		Message: "every returned credential issuer matches at least one trusted_authority",
	}
}

func validateTrustedAuthoritiesNoMatch(query map[string]any, responseValue any) Result {
	credentials, ok := query["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return Result{Status: StatusFail, Message: "dcql_query does not contain credentials"}
	}
	if err := validateDCQLCredentialQueries(credentials); err != nil {
		return Result{Status: StatusFail, Message: err.Error()}
	}
	for index, rawCredential := range credentials {
		credential, _ := normalizeJSONObject(rawCredential)
		authorities, ok := credential["trusted_authorities"].([]any)
		if !ok || len(authorities) == 0 {
			return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d] does not contain trusted_authorities", index)}
		}
		for authorityIndex, rawAuthority := range authorities {
			authority, ok := normalizeJSONObject(rawAuthority)
			if !ok || authority["type"] != "aki" {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d] is not a valid aki authority", index, authorityIndex)}
			}
			values, ok := authority["values"].([]any)
			if !ok || len(values) == 0 {
				return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d].values is empty", index, authorityIndex)}
			}
			for valueIndex, rawValue := range values {
				value, ok := rawValue.(string)
				if !ok || value == "" {
					return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d].values[%d] is not a string", index, authorityIndex, valueIndex)}
				}
				decoded, err := base64.RawURLEncoding.DecodeString(value)
				if err != nil || len(decoded) == 0 {
					return Result{Status: StatusFail, Message: fmt.Sprintf("credentials[%d].trusted_authorities[%d].values[%d] is not base64url", index, authorityIndex, valueIndex)}
				}
			}
		}
	}
	if !isEmptyDCQLValue(responseValue) {
		return Result{Status: StatusFail, Message: "wallet returned a credential for an unmatched trusted_authorities query"}
	}
	return Result{Status: StatusPass, Message: "wallet returned no credential for valid unmatched trusted_authorities"}
}

func credentialMatchesTrustedAuthorities(presentation *evidence.SDJWTPresentation, authorities []any) bool {
	for _, rawAuthority := range authorities {
		authority, ok := normalizeJSONObject(rawAuthority)
		if !ok {
			continue
		}
		authType, _ := authority["type"].(string)
		if authType == "" {
			continue
		}
		values, _ := authority["values"].([]any)
		if len(values) == 0 {
			continue
		}
		switch authType {
		case "aki":
			if sdjwtMatchesAKI(presentation, values) {
				return true
			}
		default:
			if sdjwtMatchesIssuerClaim(presentation, values) {
				return true
			}
		}
	}
	return false
}

func sdjwtMatchesAKI(presentation *evidence.SDJWTPresentation, values []any) bool {
	rawChain, ok := presentation.ProtectedHeaders["x5c"].([]any)
	if !ok || len(rawChain) == 0 {
		return false
	}
	encoded, ok := rawChain[0].(string)
	if !ok || encoded == "" {
		return false
	}
	der, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return false
	}
	if len(cert.AuthorityKeyId) == 0 {
		return false
	}
	encodedAKI := base64.RawURLEncoding.EncodeToString(cert.AuthorityKeyId)
	for _, rawValue := range values {
		value, ok := rawValue.(string)
		if ok && value == encodedAKI {
			return true
		}
	}
	return false
}

func sdjwtMatchesIssuerClaim(presentation *evidence.SDJWTPresentation, values []any) bool {
	iss, _ := presentation.IssuerPayload["iss"].(string)
	if iss == "" {
		return false
	}
	for _, rawValue := range values {
		value, ok := rawValue.(string)
		if ok && value == iss {
			return true
		}
	}
	return false
}
