// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
)

const pidMDocType = "eu.europa.ec.eudi.pid.1"

var mdocDomesticNamespacePattern = regexp.MustCompile(
	`^eu\.europa\.ec\.eudi\.pid\.([A-Z]{2})(?:-([A-Z0-9]{1,3}))?(?:\.[0-9]+)?$`,
)

type PIDMDocTypeValidator struct{}

func (PIDMDocTypeValidator) ID() string { return "pid.mdoc_doc_type" }

func (PIDMDocTypeValidator) Validate(_ context.Context, input Input) Result {
	presentation, ok := mdocPresentation(input.Value)
	if !ok {
		return wrongMDocInput(input.Value)
	}
	if _, ok := presentation.Document(pidMDocType); !ok {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("mdoc document type %q is missing", pidMDocType),
		}
	}
	return Result{Status: StatusPass, Message: fmt.Sprintf("mdoc document type is %q", pidMDocType)}
}

type PIDMDocMandatoryElementsValidator struct{}

func (PIDMDocMandatoryElementsValidator) ID() string {
	return "pid.mdoc_required_mandatory_elements_present"
}

type MDocDigestAlgorithmValidator struct{}

func (MDocDigestAlgorithmValidator) ID() string { return "mdoc.digest_algorithm" }

func (MDocDigestAlgorithmValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Algorithm string `json:"algorithm"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Algorithm == "" {
		return Result{Status: StatusError, Message: "algorithm param is required"}
	}
	presentation, ok := mdocPresentation(input.Value)
	if !ok {
		return wrongMDocInput(input.Value)
	}
	if presentation.SelectedDocument < 0 ||
		presentation.SelectedDocument >= len(presentation.Documents) {
		return Result{Status: StatusFail, Message: "selected mdoc document is missing"}
	}
	actual := presentation.Documents[presentation.SelectedDocument].DigestAlgorithm
	if actual != params.Algorithm {
		return Result{
			Status: StatusFail,
			Message: fmt.Sprintf(
				"mdoc digest algorithm is %q, expected %q",
				actual,
				params.Algorithm,
			),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: fmt.Sprintf("mdoc digest algorithm is %q", actual),
	}
}

func (PIDMDocMandatoryElementsValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Namespace        string   `json:"namespace"`
		RequiredElements []string `json:"required_elements"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if params.Namespace == "" || len(params.RequiredElements) == 0 {
		return Result{
			Status:  StatusError,
			Message: "namespace and required_elements params are required",
		}
	}
	presentation, ok := mdocPresentation(input.Value)
	if !ok {
		return wrongMDocInput(input.Value)
	}
	missing := make([]string, 0)
	for _, identifier := range params.RequiredElements {
		if _, exists := presentation.Element(params.Namespace, identifier); !exists {
			missing = append(missing, identifier)
		}
	}
	if len(missing) > 0 {
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("required mandatory PID mdoc elements are missing: %v", missing),
		}
	}
	return Result{
		Status:  StatusPass,
		Message: "all required mandatory PID mdoc elements are present",
	}
}

type MDocElementCBORTypeValidator struct{}

func (MDocElementCBORTypeValidator) ID() string { return "mdoc.element_cbor_type" }

func (MDocElementCBORTypeValidator) Validate(_ context.Context, input Input) Result {
	params, err := decodeMDocElementParams(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	expected, err := DecodeParams[struct {
		MajorType uint8 `json:"major_type"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if _, exists := input.Params["major_type"]; !exists {
		return Result{Status: StatusError, Message: "major_type param is required"}
	}
	if expected.MajorType > 7 {
		return Result{Status: StatusError, Message: "major_type must be between 0 and 7"}
	}
	element, result := requireMDocElement(input.Value, params)
	if result != nil {
		return *result
	}
	if element.MajorType != expected.MajorType {
		return invalidMDocElement(params, fmt.Errorf(
			"CBOR major type is %d, expected %d",
			element.MajorType,
			expected.MajorType,
		))
	}
	return validMDocElement(params, fmt.Sprintf("has CBOR major type %d", expected.MajorType))
}

type MDocElementUTF8StringValidator struct{}

func (MDocElementUTF8StringValidator) ID() string { return "mdoc.element_utf8_string" }

func (MDocElementUTF8StringValidator) Validate(_ context.Context, input Input) Result {
	params, err := decodeMDocElementParams(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	element, result := requireMDocElement(input.Value, params)
	if result != nil {
		return *result
	}
	if element.MajorType != 3 {
		return invalidMDocElement(
			params,
			fmt.Errorf("CBOR major type is %d, expected text string type 3", element.MajorType),
		)
	}
	if _, err := requireUTF8String(element.Value); err != nil {
		return invalidMDocElement(params, err)
	}
	return validMDocElement(params, "is a valid CBOR UTF-8 text string")
}

type MDocElementDateEncodingValidator struct{}

func (MDocElementDateEncodingValidator) ID() string { return "mdoc.element_date_encoding" }

func (MDocElementDateEncodingValidator) Validate(_ context.Context, input Input) Result {
	params, err := decodeMDocElementParams(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	element, result := requireMDocElement(input.Value, params)
	if result != nil {
		return *result
	}
	if element.MajorType != 6 || element.Tag == nil {
		return invalidMDocElement(params, fmt.Errorf("value is not a tagged CBOR data item"))
	}
	allowed, err := DecodeParams[struct {
		AllowedTags []uint64 `json:"allowed_tags"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if len(allowed.AllowedTags) == 0 {
		return Result{Status: StatusError, Message: "allowed_tags param is required"}
	}
	if !slices.Contains(allowed.AllowedTags, *element.Tag) {
		return invalidMDocElement(
			params,
			fmt.Errorf("CBOR tag is %d, expected one of %v", *element.Tag, allowed.AllowedTags),
		)
	}
	if element.ContentMajorType != 3 {
		return invalidMDocElement(
			params,
			fmt.Errorf(
				"tag content major type is %d, expected text string type 3",
				element.ContentMajorType,
			),
		)
	}
	if _, err := requireUTF8String(element.Value); err != nil {
		return invalidMDocElement(params, err)
	}
	return validMDocElement(params, fmt.Sprintf("uses CBOR tag %d with UTF-8 text", *element.Tag))
}

type MDocElementDateFormatValidator struct{}

func (MDocElementDateFormatValidator) ID() string { return "mdoc.element_date_format" }

func (MDocElementDateFormatValidator) Validate(_ context.Context, input Input) Result {
	params, element, text, result := mdocTaggedDate(input)
	if result != nil {
		return *result
	}
	if *element.Tag == 1004 && !fullDatePattern.MatchString(text) {
		return invalidMDocElement(params, fmt.Errorf("tag 1004 value must use YYYY-MM-DD format"))
	}
	if *element.Tag == 0 && (len(text) != 20 || !strings.HasSuffix(text, "Z")) {
		return invalidMDocElement(
			params,
			fmt.Errorf("tag 0 value must use YYYY-MM-DDThh:mm:ssZ format"),
		)
	}
	return validMDocElement(
		params,
		fmt.Sprintf("has the required tag %d date format", *element.Tag),
	)
}

type MDocElementValidDateValidator struct{}

func (MDocElementValidDateValidator) ID() string { return "mdoc.element_valid_date" }

func (MDocElementValidDateValidator) Validate(_ context.Context, input Input) Result {
	params, element, text, result := mdocTaggedDate(input)
	if result != nil {
		return *result
	}
	var err error
	if *element.Tag == 1004 {
		err = validateFullDate(text)
	} else {
		err = validateRFC3339UTCDateTime(text)
	}
	if err != nil {
		return invalidMDocElement(params, err)
	}
	return validMDocElement(params, "contains a valid date")
}

type MDocElementCountryCodeValidator struct{}

func (MDocElementCountryCodeValidator) ID() string { return "mdoc.element_country_code" }

func (MDocElementCountryCodeValidator) Validate(_ context.Context, input Input) Result {
	params, text, result := mdocTextElement(input)
	if result != nil {
		return *result
	}
	if !isPIDCountryCode(text) {
		return invalidMDocElement(
			params,
			fmt.Errorf("value %q is not an accepted country code", text),
		)
	}
	return validMDocElement(params, "contains an accepted country code")
}

type MDocElementRFC5322EmailValidator struct{}

func (MDocElementRFC5322EmailValidator) ID() string {
	return "mdoc.element_rfc5322_email"
}

func (MDocElementRFC5322EmailValidator) Validate(_ context.Context, input Input) Result {
	params, text, result := mdocTextElement(input)
	if result != nil {
		return *result
	}
	if utf8.RuneCountInString(text) == 0 {
		return invalidMDocElement(params, fmt.Errorf("value must contain at least one character"))
	}
	address, err := mail.ParseAddress(text)
	if err != nil || address.Address != text || address.Name != "" {
		return invalidMDocElement(params, fmt.Errorf("value is not a bare RFC 5322 addr-spec"))
	}
	return validMDocElement(params, "contains a valid RFC 5322 email address")
}

type MDocElementInternationalPhoneValidator struct{}

func (MDocElementInternationalPhoneValidator) ID() string {
	return "mdoc.element_international_phone"
}

func (MDocElementInternationalPhoneValidator) Validate(_ context.Context, input Input) Result {
	params, text, result := mdocTextElement(input)
	if result != nil {
		return *result
	}
	limits, err := DecodeParams[struct {
		MinLength int `json:"min_length"`
	}](input.Params)
	if err != nil || limits.MinLength < 1 {
		return Result{Status: StatusError, Message: "positive min_length param is required"}
	}
	if utf8.RuneCountInString(text) < limits.MinLength {
		return invalidMDocElement(
			params,
			fmt.Errorf("value must contain at least %d characters", limits.MinLength),
		)
	}
	if !internationalPhonePattern.MatchString(text) {
		return invalidMDocElement(
			params,
			fmt.Errorf("value must start with + and contain digits only"),
		)
	}
	return validMDocElement(params, "contains an international phone number")
}

type MDocElementStringArrayValidator struct{}

func (MDocElementStringArrayValidator) ID() string { return "mdoc.element_string_array" }

func (MDocElementStringArrayValidator) Validate(_ context.Context, input Input) Result {
	params, minimum, values, result := mdocArray(input)
	if result != nil {
		return *result
	}
	if len(values) < minimum {
		return invalidMDocElement(
			params,
			fmt.Errorf("array must contain at least %d item(s)", minimum),
		)
	}
	for index, value := range values {
		if _, err := requireUTF8String(value); err != nil {
			return invalidMDocElement(params, fmt.Errorf("item %d: %w", index, err))
		}
	}
	return validMDocElement(params, "is a non-empty array of UTF-8 strings")
}

type MDocElementCountryCodeArrayValidator struct{}

func (MDocElementCountryCodeArrayValidator) ID() string {
	return "mdoc.element_country_code_array"
}

func (MDocElementCountryCodeArrayValidator) Validate(_ context.Context, input Input) Result {
	params, minimum, values, result := mdocArray(input)
	if result != nil {
		return *result
	}
	if len(values) < minimum {
		return invalidMDocElement(
			params,
			fmt.Errorf("array must contain at least %d item(s)", minimum),
		)
	}
	for index, value := range values {
		text, err := requireUTF8String(value)
		if err != nil {
			return invalidMDocElement(params, fmt.Errorf("item %d: %w", index, err))
		}
		if !isPIDCountryCode(text) {
			return invalidMDocElement(
				params,
				fmt.Errorf("item %d value %q is not an accepted country code", index, text),
			)
		}
	}
	return validMDocElement(params, "contains only accepted country codes")
}

type MDocElementMapShapeValidator struct{}

func (MDocElementMapShapeValidator) ID() string { return "mdoc.element_map_shape" }

func (MDocElementMapShapeValidator) Validate(_ context.Context, input Input) Result {
	params, err := decodeMDocElementParams(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	limits, err := DecodeParams[struct {
		Allowed       []string `json:"allowed_keys"`
		MinProperties int      `json:"min_properties"`
		MaxProperties int      `json:"max_properties"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if len(limits.Allowed) == 0 || limits.MinProperties < 0 ||
		limits.MaxProperties < limits.MinProperties {
		return Result{
			Status:  StatusError,
			Message: "valid allowed_keys and property limits are required",
		}
	}
	element, result := requireMDocElement(input.Value, params)
	if result != nil {
		return *result
	}
	if element.MajorType != 5 {
		return invalidMDocElement(
			params,
			fmt.Errorf("CBOR major type is %d, expected map type 5", element.MajorType),
		)
	}
	object, ok := element.Value.(map[string]any)
	if !ok {
		return invalidMDocElement(
			params,
			fmt.Errorf("decoded value is %T, expected string-keyed map", element.Value),
		)
	}
	if len(object) < limits.MinProperties || len(object) > limits.MaxProperties {
		return invalidMDocElement(
			params,
			fmt.Errorf(
				"map has %d entries, expected between %d and %d",
				len(object),
				limits.MinProperties,
				limits.MaxProperties,
			),
		)
	}
	for key := range object {
		if !slices.Contains(limits.Allowed, key) {
			return invalidMDocElement(params, fmt.Errorf("map contains unsupported key %q", key))
		}
	}
	return validMDocElement(params, "contains only allowed map entries")
}

type MDocElementMapTextValuesValidator struct{}

func (MDocElementMapTextValuesValidator) ID() string {
	return "mdoc.element_map_text_values"
}

func (MDocElementMapTextValuesValidator) Validate(_ context.Context, input Input) Result {
	params, object, keys, result := mdocMapWithKeys(input)
	if result != nil {
		return *result
	}
	for _, key := range keys {
		value, exists := object[key]
		if !exists {
			continue
		}
		if _, err := requireUTF8String(value); err != nil {
			return invalidMDocElement(params, fmt.Errorf("map value %q: %w", key, err))
		}
	}
	return validMDocElement(params, "contains UTF-8 text map values")
}

type MDocElementMapMemberCountryCodeValidator struct{}

func (MDocElementMapMemberCountryCodeValidator) ID() string {
	return "mdoc.element_map_member_country_code"
}

func (MDocElementMapMemberCountryCodeValidator) Validate(_ context.Context, input Input) Result {
	params, member, value, result := mdocMapMember(input)
	if result != nil {
		return *result
	}
	text, err := requireUTF8String(value)
	if err != nil {
		return invalidMDocElement(params, fmt.Errorf("map member %q: %w", member, err))
	}
	if !isPIDCountryCode(text) {
		return invalidMDocElement(
			params,
			fmt.Errorf("map member %q is not an accepted country code", member),
		)
	}
	return validMDocElement(
		params,
		fmt.Sprintf("map member %q contains an accepted country code", member),
	)
}

type MDocElementMapMemberUTF8MaxLengthValidator struct{}

func (MDocElementMapMemberUTF8MaxLengthValidator) ID() string {
	return "mdoc.element_map_member_utf8_max_length"
}

func (MDocElementMapMemberUTF8MaxLengthValidator) Validate(_ context.Context, input Input) Result {
	params, member, value, result := mdocMapMember(input)
	if result != nil {
		return *result
	}
	limit, err := DecodeParams[struct {
		MaxLength int `json:"max_length"`
	}](input.Params)
	if err != nil || limit.MaxLength < 1 {
		return Result{Status: StatusError, Message: "positive max_length param is required"}
	}
	text, err := requireUTF8String(value)
	if err != nil {
		return invalidMDocElement(params, fmt.Errorf("map member %q: %w", member, err))
	}
	if utf8.RuneCountInString(text) > limit.MaxLength {
		return invalidMDocElement(
			params,
			fmt.Errorf("map member %q exceeds %d characters", member, limit.MaxLength),
		)
	}
	return validMDocElement(
		params,
		fmt.Sprintf("map member %q is within %d characters", member, limit.MaxLength),
	)
}

type MDocElementUnsignedIntegerAllowedValidator struct{}

func (MDocElementUnsignedIntegerAllowedValidator) ID() string {
	return "mdoc.element_unsigned_integer_allowed"
}

func (MDocElementUnsignedIntegerAllowedValidator) Validate(_ context.Context, input Input) Result {
	params, err := decodeMDocElementParams(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	allowed, err := DecodeParams[struct {
		Allowed []int64 `json:"allowed"`
	}](input.Params)
	if err != nil || len(allowed.Allowed) == 0 {
		return Result{Status: StatusError, Message: "allowed param is required"}
	}
	element, result := requireMDocElement(input.Value, params)
	if result != nil {
		return *result
	}
	if element.MajorType != 0 {
		return invalidMDocElement(
			params,
			fmt.Errorf(
				"CBOR major type is %d, expected unsigned integer type 0",
				element.MajorType,
			),
		)
	}
	number, ok := integralNumber(element.Value)
	if !ok || !slices.Contains(allowed.Allowed, number) {
		return invalidMDocElement(params, fmt.Errorf("value is not an allowed unsigned integer"))
	}
	return validMDocElement(params, "contains an allowed unsigned integer")
}

type MDocElementJPEGValidator struct{}

func (MDocElementJPEGValidator) ID() string { return "mdoc.element_jpeg" }

func (MDocElementJPEGValidator) Validate(_ context.Context, input Input) Result {
	params, err := decodeMDocElementParams(input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	element, result := requireMDocElement(input.Value, params)
	if result != nil {
		return *result
	}
	if element.MajorType != 2 {
		return invalidMDocElement(
			params,
			fmt.Errorf("CBOR major type is %d, expected byte string type 2", element.MajorType),
		)
	}
	value, ok := element.Value.([]byte)
	if !ok || !hasJPEGStart(value) {
		return invalidMDocElement(
			params,
			fmt.Errorf("byte string does not start with JPEG marker FF D8"),
		)
	}
	return validMDocElement(params, "contains a JPEG byte string")
}

type MDocDomesticNamespaceValidator struct{}

func (MDocDomesticNamespaceValidator) ID() string { return "mdoc.domestic_namespace" }

func (MDocDomesticNamespaceValidator) Validate(_ context.Context, input Input) Result {
	presentation, ok := mdocPresentation(input.Value)
	if !ok {
		return wrongMDocInput(input.Value)
	}
	for namespace, elements := range presentation.Namespaces {
		parts := mdocDomesticNamespacePattern.FindStringSubmatch(namespace)
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
		if len(elements) == 0 {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("domestic namespace %q contains no elements", namespace),
			}
		}
		return Result{
			Status:  StatusPass,
			Message: fmt.Sprintf("valid non-empty domestic namespace %q is present", namespace),
		}
	}
	return Result{
		Status:  StatusFail,
		Message: "no valid non-empty PID domestic namespace is present",
	}
}

type MDocElementCountrySubdivisionValidator struct{}

func (MDocElementCountrySubdivisionValidator) ID() string {
	return "mdoc.element_country_subdivision"
}

func (MDocElementCountrySubdivisionValidator) Validate(_ context.Context, input Input) Result {
	params, text, result := mdocTextElement(input)
	if result != nil {
		return *result
	}
	countryParams, err := DecodeParams[struct {
		CountryElement string `json:"country_element"`
	}](input.Params)
	if err != nil || countryParams.CountryElement == "" {
		return Result{Status: StatusError, Message: "country_element param is required"}
	}
	presentation, _ := mdocPresentation(input.Value)
	countryElement, ok := presentation.Element(params.Namespace, countryParams.CountryElement)
	if !ok {
		return invalidMDocElement(
			params,
			fmt.Errorf("country element %q is missing", countryParams.CountryElement),
		)
	}
	country, err := requireUTF8String(countryElement.Value)
	if err != nil {
		return invalidMDocElement(
			params,
			fmt.Errorf("country element %q: %w", countryParams.CountryElement, err),
		)
	}
	if err := validateCountrySubdivision(text, country); err != nil {
		return invalidMDocElement(params, err)
	}
	return validMDocElement(params, "contains a country subdivision matching the issuing country")
}

type mdocElementParams struct {
	Namespace string `json:"namespace"`
	Element   string `json:"element"`
}

func decodeMDocElementParams(params map[string]any) (mdocElementParams, error) {
	decoded, err := DecodeParams[mdocElementParams](params)
	if err != nil {
		return decoded, err
	}
	if decoded.Namespace == "" || decoded.Element == "" {
		return decoded, fmt.Errorf("namespace and element params are required")
	}
	return decoded, nil
}

func mdocPresentation(value any) (*evidence.MDocPresentation, bool) {
	switch typed := value.(type) {
	case *evidence.MDocPresentation:
		return typed, typed != nil
	case evidence.MDocPresentation:
		return &typed, true
	default:
		return nil, false
	}
}

func requireMDocElement(value any, params mdocElementParams) (evidence.MDocElement, *Result) {
	presentation, ok := mdocPresentation(value)
	if !ok {
		result := wrongMDocInput(value)
		return evidence.MDocElement{}, &result
	}
	if document, ok := presentation.Document(pidMDocType); ok {
		if code, hasError := document.Errors[params.Namespace][params.Element]; hasError {
			result := invalidMDocElement(
				params,
				fmt.Errorf("document contains ErrorItem code %d", code),
			)
			return evidence.MDocElement{}, &result
		}
	}
	element, ok := presentation.Element(params.Namespace, params.Element)
	if !ok {
		result := invalidMDocElement(params, fmt.Errorf("element is missing"))
		return evidence.MDocElement{}, &result
	}
	return element, nil
}

func mdocTextElement(input Input) (mdocElementParams, string, *Result) {
	params, err := decodeMDocElementParams(input.Params)
	if err != nil {
		result := Result{Status: StatusError, Message: err.Error()}
		return params, "", &result
	}
	element, result := requireMDocElement(input.Value, params)
	if result != nil {
		return params, "", result
	}
	if element.MajorType != 3 {
		invalid := invalidMDocElement(
			params,
			fmt.Errorf("CBOR major type is %d, expected text string type 3", element.MajorType),
		)
		return params, "", &invalid
	}
	text, err := requireUTF8String(element.Value)
	if err != nil {
		invalid := invalidMDocElement(params, err)
		return params, "", &invalid
	}
	return params, text, nil
}

func mdocTaggedDate(input Input) (mdocElementParams, evidence.MDocElement, string, *Result) {
	params, err := decodeMDocElementParams(input.Params)
	if err != nil {
		result := Result{Status: StatusError, Message: err.Error()}
		return params, evidence.MDocElement{}, "", &result
	}
	element, result := requireMDocElement(input.Value, params)
	if result != nil {
		return params, element, "", result
	}
	if element.MajorType != 6 || element.Tag == nil || (*element.Tag != 0 && *element.Tag != 1004) {
		invalid := invalidMDocElement(params, fmt.Errorf("value must use CBOR tag 0 or 1004"))
		return params, element, "", &invalid
	}
	text, err := requireUTF8String(element.Value)
	if err != nil {
		invalid := invalidMDocElement(params, err)
		return params, element, "", &invalid
	}
	return params, element, text, nil
}

func mdocArray(input Input) (mdocElementParams, int, []any, *Result) {
	params, err := decodeMDocElementParams(input.Params)
	if err != nil {
		result := Result{Status: StatusError, Message: err.Error()}
		return params, 0, nil, &result
	}
	minimum, err := DecodeParams[struct {
		MinItems int `json:"min_items"`
	}](input.Params)
	if err != nil || minimum.MinItems < 1 {
		result := Result{Status: StatusError, Message: "positive min_items param is required"}
		return params, 0, nil, &result
	}
	element, result := requireMDocElement(input.Value, params)
	if result != nil {
		return params, minimum.MinItems, nil, result
	}
	if element.MajorType != 4 {
		invalid := invalidMDocElement(
			params,
			fmt.Errorf("CBOR major type is %d, expected array type 4", element.MajorType),
		)
		return params, minimum.MinItems, nil, &invalid
	}
	values, ok := element.Value.([]any)
	if !ok {
		invalid := invalidMDocElement(
			params,
			fmt.Errorf("decoded value is %T, expected array", element.Value),
		)
		return params, minimum.MinItems, nil, &invalid
	}
	return params, minimum.MinItems, values, nil
}

func mdocMapWithKeys(input Input) (mdocElementParams, map[string]any, []string, *Result) {
	params, err := decodeMDocElementParams(input.Params)
	if err != nil {
		result := Result{Status: StatusError, Message: err.Error()}
		return params, nil, nil, &result
	}
	keyParams, err := DecodeParams[struct {
		Keys []string `json:"keys"`
	}](input.Params)
	if err != nil || len(keyParams.Keys) == 0 {
		result := Result{Status: StatusError, Message: "keys param is required"}
		return params, nil, nil, &result
	}
	element, result := requireMDocElement(input.Value, params)
	if result != nil {
		return params, nil, keyParams.Keys, result
	}
	object, ok := element.Value.(map[string]any)
	if element.MajorType != 5 || !ok {
		invalid := invalidMDocElement(params, fmt.Errorf("value is not a CBOR map"))
		return params, nil, keyParams.Keys, &invalid
	}
	return params, object, keyParams.Keys, nil
}

func mdocMapMember(input Input) (mdocElementParams, string, any, *Result) {
	params, err := decodeMDocElementParams(input.Params)
	if err != nil {
		result := Result{Status: StatusError, Message: err.Error()}
		return params, "", nil, &result
	}
	memberParams, err := DecodeParams[struct {
		Member string `json:"member"`
	}](input.Params)
	if err != nil || memberParams.Member == "" {
		result := Result{Status: StatusError, Message: "member param is required"}
		return params, "", nil, &result
	}
	element, result := requireMDocElement(input.Value, params)
	if result != nil {
		return params, memberParams.Member, nil, result
	}
	object, ok := element.Value.(map[string]any)
	if element.MajorType != 5 || !ok {
		invalid := invalidMDocElement(params, fmt.Errorf("value is not a CBOR map"))
		return params, memberParams.Member, nil, &invalid
	}
	value, exists := object[memberParams.Member]
	if !exists {
		invalid := invalidMDocElement(
			params,
			fmt.Errorf("map member %q is missing", memberParams.Member),
		)
		return params, memberParams.Member, nil, &invalid
	}
	return params, memberParams.Member, value, nil
}

func wrongMDocInput(value any) Result {
	return Result{
		Status:  StatusFail,
		Message: fmt.Sprintf("input is %T, expected decoded mdoc presentation", value),
	}
}

func invalidMDocElement(params mdocElementParams, err error) Result {
	return Result{Status: StatusFail, Message: fmt.Sprintf(
		"mdoc element %q in namespace %q is invalid: %v",
		params.Element,
		params.Namespace,
		err,
	)}
}

func validMDocElement(params mdocElementParams, description string) Result {
	return Result{Status: StatusPass, Message: fmt.Sprintf(
		"mdoc element %q in namespace %q %s",
		params.Element,
		params.Namespace,
		description,
	)}
}
