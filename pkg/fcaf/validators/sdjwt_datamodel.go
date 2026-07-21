// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

var domesticNamespacePattern = regexp.MustCompile(
	`^eu\.europa\.ec\.eudi\.pid\.([A-Z]{2})(?:-([A-Z0-9]{1,3}))?(?:\.[0-9]+)?$`,
)

type SDJWTClaimNonEmptyUTF8StringValidator struct{}

func (SDJWTClaimNonEmptyUTF8StringValidator) ID() string {
	return "sdjwt.claim_non_empty_utf8_string"
}

func (SDJWTClaimNonEmptyUTF8StringValidator) Validate(_ context.Context, input Input) Result {
	claim, err := decodeClaimParam(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	value, ok := sdjwtClaim(input.Value, claim)
	if !ok {
		return missingClaim(claim)
	}
	text, err := requireUTF8String(value)
	if err != nil {
		return invalidClaim(claim, err)
	}
	if utf8.RuneCountInString(text) == 0 {
		return invalidClaim(claim, fmt.Errorf("value must contain at least one character"))
	}
	return validClaim(claim, "is a non-empty UTF-8 string")
}

type SDJWTClaimInternationalPhoneValidator struct{}

func (SDJWTClaimInternationalPhoneValidator) ID() string {
	return "sdjwt.claim_international_phone"
}

func (SDJWTClaimInternationalPhoneValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Claim     string `json:"claim"`
		MinLength int    `json:"min_length"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Claim == "" {
		return Result{Status: StatusError, Message: "claim param is required"}
	}
	if params.MinLength < 1 {
		return Result{Status: StatusError, Message: "min_length must be greater than zero"}
	}
	value, ok := sdjwtClaim(input.Value, params.Claim)
	if !ok {
		return missingClaim(params.Claim)
	}
	text, err := requireUTF8String(value)
	if err != nil {
		return invalidClaim(params.Claim, err)
	}
	if utf8.RuneCountInString(text) < params.MinLength {
		return invalidClaim(
			params.Claim,
			fmt.Errorf("value must contain at least %d characters", params.MinLength),
		)
	}
	if !internationalPhonePattern.MatchString(text) {
		return invalidClaim(
			params.Claim,
			fmt.Errorf("value must start with + and contain digits only"),
		)
	}
	return validClaim(params.Claim, "is an international phone number")
}

type SDJWTClaimCountryCodeValidator struct{}

func (SDJWTClaimCountryCodeValidator) ID() string { return "sdjwt.claim_country_code" }

func (SDJWTClaimCountryCodeValidator) Validate(_ context.Context, input Input) Result {
	claim, err := decodeClaimParam(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	value, ok := sdjwtClaim(input.Value, claim)
	if !ok {
		return missingClaim(claim)
	}
	text, err := requireUTF8String(value)
	if err != nil {
		return invalidClaim(claim, err)
	}
	if !isPIDCountryCode(text) {
		return invalidClaim(
			claim,
			fmt.Errorf(
				"value %q is not an accepted ISO 3166-1 alpha-2 or user-assigned code",
				text,
			),
		)
	}
	return validClaim(claim, "contains an accepted country code")
}

type SDJWTClaimDateFormatValidator struct{}

func (SDJWTClaimDateFormatValidator) ID() string { return "sdjwt.claim_date_format" }

func (SDJWTClaimDateFormatValidator) Validate(_ context.Context, input Input) Result {
	claim, text, result := sdjwtStringClaim(input)
	if result != nil {
		return *result
	}
	if !fullDatePattern.MatchString(text) {
		return invalidClaim(claim, fmt.Errorf("value must use YYYY-MM-DD format"))
	}
	return validClaim(claim, "uses YYYY-MM-DD format")
}

type SDJWTClaimValidDateValidator struct{}

func (SDJWTClaimValidDateValidator) ID() string { return "sdjwt.claim_valid_date" }

func (SDJWTClaimValidDateValidator) Validate(_ context.Context, input Input) Result {
	claim, text, result := sdjwtStringClaim(input)
	if result != nil {
		return *result
	}
	if err := validateFullDate(text); err != nil {
		return invalidClaim(claim, err)
	}
	return validClaim(claim, "contains a valid date")
}

type SDJWTClaimStringArrayValidator struct{}

func (SDJWTClaimStringArrayValidator) ID() string { return "sdjwt.claim_string_array" }

func (SDJWTClaimStringArrayValidator) Validate(_ context.Context, input Input) Result {
	params, err := decodeClaimAndMinimum(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	values, result := sdjwtArrayClaim(input.Value, params.Claim)
	if result != nil {
		return *result
	}
	if len(values) < params.MinItems {
		return invalidClaim(
			params.Claim,
			fmt.Errorf("array must contain at least %d item(s)", params.MinItems),
		)
	}
	for index, value := range values {
		if _, err := requireUTF8String(value); err != nil {
			return invalidClaim(params.Claim, fmt.Errorf("item %d: %w", index, err))
		}
	}
	return validClaim(params.Claim, "is a non-empty array of UTF-8 strings")
}

type SDJWTClaimCountryCodeArrayValidator struct{}

func (SDJWTClaimCountryCodeArrayValidator) ID() string {
	return "sdjwt.claim_country_code_array"
}

func (SDJWTClaimCountryCodeArrayValidator) Validate(_ context.Context, input Input) Result {
	params, err := decodeClaimAndMinimum(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	values, result := sdjwtArrayClaim(input.Value, params.Claim)
	if result != nil {
		return *result
	}
	if len(values) < params.MinItems {
		return invalidClaim(
			params.Claim,
			fmt.Errorf("array must contain at least %d item(s)", params.MinItems),
		)
	}
	for index, value := range values {
		text, err := requireUTF8String(value)
		if err != nil {
			return invalidClaim(params.Claim, fmt.Errorf("item %d: %w", index, err))
		}
		if !isPIDCountryCode(text) {
			return invalidClaim(
				params.Claim,
				fmt.Errorf("item %d value %q is not an accepted country code", index, text),
			)
		}
	}
	return validClaim(params.Claim, "contains only accepted country codes")
}

type SDJWTClaimObjectValidator struct{}

func (SDJWTClaimObjectValidator) ID() string { return "sdjwt.claim_object" }

func (SDJWTClaimObjectValidator) Validate(_ context.Context, input Input) Result {
	claim, err := decodeClaimParam(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	value, ok := sdjwtClaim(input.Value, claim)
	if !ok {
		return missingClaim(claim)
	}
	if _, ok := value.(map[string]any); !ok {
		return invalidClaim(claim, fmt.Errorf("value is %T, expected object", value))
	}
	return validClaim(claim, "is an object")
}

type SDJWTClaimObjectKeysValidator struct{}

func (SDJWTClaimObjectKeysValidator) ID() string { return "sdjwt.claim_object_keys" }

func (SDJWTClaimObjectKeysValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Claim         string   `json:"claim"`
		Allowed       []string `json:"allowed"`
		MinProperties int      `json:"min_properties"`
		MaxProperties int      `json:"max_properties"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Claim == "" || len(params.Allowed) == 0 {
		return Result{Status: StatusError, Message: "claim and allowed params are required"}
	}
	object, result := sdjwtObjectClaim(input.Value, params.Claim)
	if result != nil {
		return *result
	}
	if len(object) < params.MinProperties ||
		params.MaxProperties > 0 && len(object) > params.MaxProperties {
		return invalidClaim(params.Claim, fmt.Errorf(
			"object has %d properties, expected between %d and %d",
			len(object), params.MinProperties, params.MaxProperties,
		))
	}
	for key := range object {
		if !slices.Contains(params.Allowed, key) {
			return invalidClaim(
				params.Claim,
				fmt.Errorf("object contains unsupported property %q", key),
			)
		}
	}
	return validClaim(params.Claim, "contains only allowed properties")
}

type SDJWTClaimObjectStringValuesValidator struct{}

func (SDJWTClaimObjectStringValuesValidator) ID() string {
	return "sdjwt.claim_object_string_values"
}

func (SDJWTClaimObjectStringValuesValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Claim string   `json:"claim"`
		Keys  []string `json:"keys"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Claim == "" || len(params.Keys) == 0 {
		return Result{Status: StatusError, Message: "claim and keys params are required"}
	}
	object, result := sdjwtObjectClaim(input.Value, params.Claim)
	if result != nil {
		return *result
	}
	for _, key := range params.Keys {
		value, exists := object[key]
		if !exists {
			continue
		}
		if _, err := requireUTF8String(value); err != nil {
			return invalidClaim(params.Claim, fmt.Errorf("property %q: %w", key, err))
		}
	}
	return validClaim(params.Claim, "contains UTF-8 string property values")
}

type SDJWTClaimNestedStringMaxLengthValidator struct{}

func (SDJWTClaimNestedStringMaxLengthValidator) ID() string {
	return "sdjwt.claim_nested_string_max_length"
}

func (SDJWTClaimNestedStringMaxLengthValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Claim     string `json:"claim"`
		Member    string `json:"member"`
		MaxLength int    `json:"max_length"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Claim == "" || params.Member == "" || params.MaxLength < 1 {
		return Result{
			Status:  StatusError,
			Message: "claim, member and positive max_length params are required",
		}
	}
	value, ok := sdjwtClaim(input.Value, params.Claim+"."+params.Member)
	if !ok {
		return missingClaim(params.Claim + "." + params.Member)
	}
	text, err := requireUTF8String(value)
	if err != nil {
		return invalidClaim(params.Claim+"."+params.Member, err)
	}
	if utf8.RuneCountInString(text) > params.MaxLength {
		return invalidClaim(
			params.Claim+"."+params.Member,
			fmt.Errorf("value exceeds %d characters", params.MaxLength),
		)
	}
	return validClaim(
		params.Claim+"."+params.Member,
		fmt.Sprintf("contains at most %d UTF-8 characters", params.MaxLength),
	)
}

type SDJWTClaimIntegerAllowedValidator struct{}

func (SDJWTClaimIntegerAllowedValidator) ID() string { return "sdjwt.claim_integer_allowed" }

func (SDJWTClaimIntegerAllowedValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Claim   string  `json:"claim"`
		Allowed []int64 `json:"allowed"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Claim == "" || len(params.Allowed) == 0 {
		return Result{Status: StatusError, Message: "claim and allowed params are required"}
	}
	value, ok := sdjwtClaim(input.Value, params.Claim)
	if !ok {
		return missingClaim(params.Claim)
	}
	number, ok := integralNumber(value)
	if !ok {
		return invalidClaim(params.Claim, fmt.Errorf("value is %T, expected integer", value))
	}
	if !slices.Contains(params.Allowed, number) {
		return invalidClaim(params.Claim, fmt.Errorf("integer %d is not allowed", number))
	}
	return validClaim(params.Claim, "contains an allowed integer")
}

type SDJWTClaimJPEGDataURLValidator struct{}

func (SDJWTClaimJPEGDataURLValidator) ID() string { return "sdjwt.claim_jpeg_data_url" }

func (SDJWTClaimJPEGDataURLValidator) Validate(_ context.Context, input Input) Result {
	claim, text, result := sdjwtStringClaim(input)
	if result != nil {
		return *result
	}
	if _, err := decodeJPEGDataURL(text); err != nil {
		return invalidClaim(claim, err)
	}
	return validClaim(claim, "contains a base64-encoded JPEG data URL")
}

type SDJWTClaimCountrySubdivisionValidator struct{}

func (SDJWTClaimCountrySubdivisionValidator) ID() string {
	return "sdjwt.claim_country_subdivision"
}

type SDJWTDomesticNamespaceValidator struct{}

func (SDJWTDomesticNamespaceValidator) ID() string { return "sdjwt.domestic_namespace" }

func (SDJWTDomesticNamespaceValidator) Validate(_ context.Context, input Input) Result {
	claims, ok := sdjwtClaims(input.Value)
	if !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("input is %T, expected SD-JWT claims", input.Value),
		}
	}
	matches := 0
	for namespace, value := range claims {
		parts := domesticNamespacePattern.FindStringSubmatch(namespace)
		if parts == nil {
			continue
		}
		if !isPIDCountryCode(parts[1]) {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"domestic namespace %q has an invalid country code",
					namespace,
				),
			}
		}
		object, ok := value.(map[string]any)
		if !ok || len(object) == 0 {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("domestic namespace %q contains no claims", namespace),
			}
		}
		matches++
	}
	if matches == 0 {
		return Result{
			Status:  StatusFail,
			Message: "no valid non-empty PID domestic namespace is present",
		}
	}
	return Result{
		Status:  StatusPass,
		Message: "a valid non-empty PID domestic namespace is present",
	}
}

func (SDJWTClaimCountrySubdivisionValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Claim        string `json:"claim"`
		CountryClaim string `json:"country_claim"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Claim == "" || params.CountryClaim == "" {
		return Result{Status: StatusError, Message: "claim and country_claim params are required"}
	}
	value, ok := sdjwtClaim(input.Value, params.Claim)
	if !ok {
		return missingClaim(params.Claim)
	}
	text, err := requireUTF8String(value)
	if err != nil {
		return invalidClaim(params.Claim, err)
	}
	countryValue, ok := sdjwtClaim(input.Value, params.CountryClaim)
	if !ok {
		return missingClaim(params.CountryClaim)
	}
	country, err := requireUTF8String(countryValue)
	if err != nil {
		return invalidClaim(params.CountryClaim, err)
	}
	if err := validateCountrySubdivision(text, country); err != nil {
		return invalidClaim(params.Claim, err)
	}
	return validClaim(params.Claim, "contains a country subdivision matching the issuing country")
}

type claimAndMinimum struct {
	Claim    string `json:"claim"`
	MinItems int    `json:"min_items"`
}

func decodeClaimAndMinimum(params map[string]any) (claimAndMinimum, error) {
	decoded, err := DecodeParams[claimAndMinimum](params)
	if err != nil {
		return decoded, err
	}
	if decoded.Claim == "" {
		return decoded, fmt.Errorf("claim param is required")
	}
	if decoded.MinItems < 1 {
		return decoded, fmt.Errorf("min_items must be greater than zero")
	}
	return decoded, nil
}

func sdjwtStringClaim(input Input) (string, string, *Result) {
	claim, err := decodeClaimParam(input.Params)
	if err != nil {
		result := Result{Status: StatusError, Message: err.Error()}
		return "", "", &result
	}
	value, ok := sdjwtClaim(input.Value, claim)
	if !ok {
		result := missingClaim(claim)
		return claim, "", &result
	}
	text, err := requireUTF8String(value)
	if err != nil {
		result := invalidClaim(claim, err)
		return claim, "", &result
	}
	return claim, text, nil
}

func sdjwtArrayClaim(value any, claim string) ([]any, *Result) {
	raw, ok := sdjwtClaim(value, claim)
	if !ok {
		result := missingClaim(claim)
		return nil, &result
	}
	values, ok := raw.([]any)
	if !ok {
		result := invalidClaim(claim, fmt.Errorf("value is %T, expected array", raw))
		return nil, &result
	}
	return values, nil
}

func sdjwtObjectClaim(value any, claim string) (map[string]any, *Result) {
	raw, ok := sdjwtClaim(value, claim)
	if !ok {
		result := missingClaim(claim)
		return nil, &result
	}
	object, ok := raw.(map[string]any)
	if !ok {
		result := invalidClaim(claim, fmt.Errorf("value is %T, expected object", raw))
		return nil, &result
	}
	return object, nil
}

func sdjwtClaims(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case *evidence.SDJWTPresentation:
		if typed == nil {
			return nil, false
		}
		return typed.Claims, true
	case evidence.SDJWTPresentation:
		return typed.Claims, true
	case map[string]any:
		return typed, true
	default:
		return nil, false
	}
}

func missingClaim(claim string) Result {
	return Result{Status: StatusFail, Message: fmt.Sprintf("claim %q is missing", claim)}
}

func invalidClaim(claim string, err error) Result {
	return Result{Status: StatusFail, Message: fmt.Sprintf("claim %q is invalid: %v", claim, err)}
}

func validClaim(claim string, description string) Result {
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("claim %q %s", claim, strings.TrimSpace(description)),
	}
}
