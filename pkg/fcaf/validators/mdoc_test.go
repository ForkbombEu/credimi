// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/stretchr/testify/require"
)

func TestMDocDataModelValidators(t *testing.T) {
	dateTag := uint64(1004)
	presentation := mdocValidatorPresentation(map[string]evidence.MDocElement{
		"family_name": {
			Identifier: "family_name",
			Value:      "Trotter",
			MajorType:  3,
		},
		"birth_date": {
			Identifier:       "birth_date",
			Value:            "2000-02-29",
			MajorType:        6,
			ContentMajorType: 3,
			Tag:              &dateTag,
		},
		"nationality": {
			Identifier: "nationality",
			Value:      []any{"IT", "GR"},
			MajorType:  4,
		},
		"place_of_birth": {
			Identifier: "place_of_birth",
			Value: map[string]any{
				"country":  "IT",
				"locality": "Roma",
			},
			MajorType: 5,
		},
		"portrait": {
			Identifier: "portrait",
			Value:      []byte{0xff, 0xd8, 0xff},
			MajorType:  2,
		},
		"sex": {
			Identifier: "sex",
			Value:      uint64(6),
			MajorType:  0,
		},
	})

	tests := []struct {
		name      string
		validator Validator
		params    map[string]any
	}{
		{
			name:      "text",
			validator: MDocElementUTF8StringValidator{},
			params:    mdocParams("family_name"),
		},
		{
			name:      "date encoding",
			validator: MDocElementDateEncodingValidator{},
			params: mergeMDocParams("birth_date", map[string]any{
				"allowed_tags": []int{1004},
			}),
		},
		{
			name:      "date format",
			validator: MDocElementDateFormatValidator{},
			params:    mdocParams("birth_date"),
		},
		{
			name:      "valid date",
			validator: MDocElementValidDateValidator{},
			params:    mdocParams("birth_date"),
		},
		{
			name:      "country array",
			validator: MDocElementCountryCodeArrayValidator{},
			params:    mergeMDocParams("nationality", map[string]any{"min_items": 1}),
		},
		{
			name:      "map shape",
			validator: MDocElementMapShapeValidator{},
			params: mergeMDocParams("place_of_birth", map[string]any{
				"allowed_keys":   []string{"country", "region", "locality"},
				"min_properties": 1,
				"max_properties": 3,
			}),
		},
		{
			name:      "map text",
			validator: MDocElementMapTextValuesValidator{},
			params: mergeMDocParams("place_of_birth", map[string]any{
				"keys": []string{"country", "region", "locality"},
			}),
		},
		{
			name:      "map country",
			validator: MDocElementMapMemberCountryCodeValidator{},
			params: mergeMDocParams("place_of_birth", map[string]any{
				"member": "country",
			}),
		},
		{
			name:      "jpeg",
			validator: MDocElementJPEGValidator{},
			params:    mdocParams("portrait"),
		},
		{
			name:      "unsigned integer",
			validator: MDocElementUnsignedIntegerAllowedValidator{},
			params: mergeMDocParams("sex", map[string]any{
				"allowed": []int{0, 1, 2, 3, 4, 5, 6, 9},
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.validator.Validate(context.Background(), Input{
				Value:  presentation,
				Params: test.params,
			})
			require.Equal(t, StatusPass, result.Status, result.Message)
		})
	}
}

func TestMDocElementPresenceRejectsErrorItem(t *testing.T) {
	presentation := mdocValidatorPresentation(map[string]evidence.MDocElement{
		"email_address": {
			Identifier: "email_address",
			Value:      "person@example.test",
			MajorType:  3,
		},
	})
	presentation.Documents[0].Errors = map[string]map[string]int64{
		pidMDocType: {"email_address": 2},
	}

	result := MDocNamespaceElementPresentValidator{}.Validate(context.Background(), Input{
		Value:  presentation,
		Params: mdocParams("email_address"),
	})

	require.Equal(t, StatusFail, result.Status)
	require.Contains(t, result.Message, "ErrorItem")
}

func TestMDocElementValidDateRejectsInvalidCalendarDate(t *testing.T) {
	tag := uint64(1004)
	presentation := mdocValidatorPresentation(map[string]evidence.MDocElement{
		"birth_date": {
			Identifier:       "birth_date",
			Value:            "2023-02-29",
			MajorType:        6,
			ContentMajorType: 3,
			Tag:              &tag,
		},
	})

	result := MDocElementValidDateValidator{}.Validate(context.Background(), Input{
		Value:  presentation,
		Params: mdocParams("birth_date"),
	})

	require.Equal(t, StatusFail, result.Status)
}

func TestMDocValidatorsRejectMissingConfiguration(t *testing.T) {
	validators := []Validator{
		MDocElementCBORTypeValidator{},
		MDocElementUTF8StringValidator{},
		MDocElementDateEncodingValidator{},
		MDocElementDateFormatValidator{},
		MDocElementValidDateValidator{},
		MDocElementCountryCodeValidator{},
		MDocElementStringArrayValidator{},
		MDocElementCountryCodeArrayValidator{},
		MDocElementMapShapeValidator{},
		MDocElementMapTextValuesValidator{},
		MDocElementMapMemberCountryCodeValidator{},
		MDocElementMapMemberUTF8MaxLengthValidator{},
		MDocElementUnsignedIntegerAllowedValidator{},
		MDocElementJPEGValidator{},
		MDocElementCountrySubdivisionValidator{},
	}
	for _, validator := range validators {
		t.Run(validator.ID(), func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{})
			require.Equal(t, StatusError, result.Status)
		})
	}
}

func mdocValidatorPresentation(
	elements map[string]evidence.MDocElement,
) *evidence.MDocPresentation {
	document := evidence.MDocDocument{
		DocType:    pidMDocType,
		Namespaces: map[string]map[string]evidence.MDocElement{pidMDocType: elements},
	}
	return &evidence.MDocPresentation{
		Documents:  []evidence.MDocDocument{document},
		Namespaces: document.Namespaces,
	}
}

func TestMDocDigestAlgorithmValidator(t *testing.T) {
	tests := []struct {
		name       string
		algorithm  string
		selected   int
		expected   string
		wantStatus Status
	}{
		{name: "SHA-256", algorithm: "SHA-256", expected: "SHA-256", wantStatus: StatusPass},
		{name: "SHA-384", algorithm: "SHA-384", expected: "SHA-256", wantStatus: StatusFail},
		{name: "missing", algorithm: "", expected: "SHA-256", wantStatus: StatusFail},
		{
			name:       "selected document out of range",
			algorithm:  "SHA-256",
			selected:   1,
			expected:   "SHA-256",
			wantStatus: StatusFail,
		},
		{
			name:       "negative selected document",
			algorithm:  "SHA-256",
			selected:   -1,
			expected:   "SHA-256",
			wantStatus: StatusFail,
		},
		{name: "algorithm param required", algorithm: "SHA-256", wantStatus: StatusError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presentation := mdocValidatorPresentation(nil)
			presentation.Documents[0].DigestAlgorithm = tt.algorithm
			presentation.SelectedDocument = tt.selected
			got := MDocDigestAlgorithmValidator{}.Validate(context.Background(), Input{
				Value:  presentation,
				Params: map[string]any{"algorithm": tt.expected},
			})

			require.Equal(t, tt.wantStatus, got.Status, got.Message)
		})
	}
}

func mdocParams(element string) map[string]any {
	return map[string]any{
		"namespace": pidMDocType,
		"element":   element,
	}
}

func mergeMDocParams(element string, values map[string]any) map[string]any {
	params := mdocParams(element)
	for key, value := range values {
		params[key] = value
	}
	return params
}

func TestMDocElementValidatorOutcomes(t *testing.T) {
	dateTag := uint64(1004)
	dateTimeTag := uint64(0)
	encodedCBORTag := uint64(24)
	text := func(identifier, value string) evidence.MDocElement {
		return evidence.MDocElement{Identifier: identifier, Value: value, MajorType: 3}
	}
	tagged := func(identifier string, tag *uint64, value any) evidence.MDocElement {
		return evidence.MDocElement{
			Identifier:       identifier,
			Value:            value,
			MajorType:        6,
			ContentMajorType: 3,
			Tag:              tag,
		}
	}
	array := func(identifier string, values []any) evidence.MDocElement {
		return evidence.MDocElement{Identifier: identifier, Value: values, MajorType: 4}
	}
	object := func(identifier string, values map[string]any) evidence.MDocElement {
		return evidence.MDocElement{Identifier: identifier, Value: values, MajorType: 5}
	}
	placeOfBirthParams := func(values map[string]any) map[string]any {
		return mergeMDocParams("place_of_birth", values)
	}

	tests := []struct {
		name      string
		validator Validator
		element   evidence.MDocElement
		extra     map[string]evidence.MDocElement
		params    map[string]any
		status    Status
		message   string
	}{
		{
			name:      "cbor type matches",
			validator: MDocElementCBORTypeValidator{},
			element:   text("family_name", "Trotter"),
			params:    mergeMDocParams("family_name", map[string]any{"major_type": 3}),
			status:    StatusPass,
		},
		{
			name:      "cbor type mismatch reports both types",
			validator: MDocElementCBORTypeValidator{},
			element:   text("family_name", "Trotter"),
			params:    mergeMDocParams("family_name", map[string]any{"major_type": 4}),
			status:    StatusFail,
			message:   "CBOR major type is 3, expected 4",
		},
		{
			name:      "cbor type requires major_type",
			validator: MDocElementCBORTypeValidator{},
			element:   text("family_name", "Trotter"),
			params:    mdocParams("family_name"),
			status:    StatusError,
			message:   "major_type param is required",
		},
		{
			name:      "cbor type rejects major_type above 7",
			validator: MDocElementCBORTypeValidator{},
			element:   text("family_name", "Trotter"),
			params:    mergeMDocParams("family_name", map[string]any{"major_type": 8}),
			status:    StatusError,
			message:   "major_type must be between 0 and 7",
		},
		{
			name:      "cbor type reports missing element",
			validator: MDocElementCBORTypeValidator{},
			element:   text("family_name", "Trotter"),
			params:    mergeMDocParams("given_name", map[string]any{"major_type": 3}),
			status:    StatusFail,
			message:   `mdoc element "given_name" in namespace "eu.europa.ec.eudi.pid.1" is invalid: element is missing`,
		},
		{
			name:      "utf8 string rejects byte string",
			validator: MDocElementUTF8StringValidator{},
			element: evidence.MDocElement{
				Identifier: "family_name",
				Value:      []byte("x"),
				MajorType:  2,
			},
			params:  mdocParams("family_name"),
			status:  StatusFail,
			message: "expected text string type 3",
		},
		{
			name:      "utf8 string rejects invalid utf8 text",
			validator: MDocElementUTF8StringValidator{},
			element:   text("family_name", "\xff\xfe"),
			params:    mdocParams("family_name"),
			status:    StatusFail,
			message:   "not valid UTF-8",
		},
		{
			name:      "date encoding rejects untagged text",
			validator: MDocElementDateEncodingValidator{},
			element:   text("birth_date", "2000-02-29"),
			params:    mergeMDocParams("birth_date", map[string]any{"allowed_tags": []int{1004}}),
			status:    StatusFail,
			message:   "not a tagged CBOR data item",
		},
		{
			name:      "date encoding requires allowed_tags",
			validator: MDocElementDateEncodingValidator{},
			element:   tagged("birth_date", &dateTag, "2000-02-29"),
			params:    mdocParams("birth_date"),
			status:    StatusError,
			message:   "allowed_tags param is required",
		},
		{
			name:      "date encoding rejects tag outside allowed list",
			validator: MDocElementDateEncodingValidator{},
			element:   tagged("birth_date", &dateTimeTag, "2000-02-29T10:00:00Z"),
			params:    mergeMDocParams("birth_date", map[string]any{"allowed_tags": []int{1004}}),
			status:    StatusFail,
			message:   "CBOR tag is 0, expected one of [1004]",
		},
		{
			name:      "date encoding rejects non-text tag content",
			validator: MDocElementDateEncodingValidator{},
			element: evidence.MDocElement{
				Identifier:       "birth_date",
				Value:            uint64(20000229),
				MajorType:        6,
				ContentMajorType: 0,
				Tag:              &dateTag,
			},
			params:  mergeMDocParams("birth_date", map[string]any{"allowed_tags": []int{1004}}),
			status:  StatusFail,
			message: "tag content major type is 0",
		},
		{
			name:      "date encoding rejects decoded non-string content",
			validator: MDocElementDateEncodingValidator{},
			element:   tagged("birth_date", &dateTag, 20000229),
			params:    mergeMDocParams("birth_date", map[string]any{"allowed_tags": []int{1004}}),
			status:    StatusFail,
			message:   "value is int, expected string",
		},
		{
			name:      "date format rejects slash separated full-date",
			validator: MDocElementDateFormatValidator{},
			element:   tagged("birth_date", &dateTag, "2000/02/29"),
			params:    mdocParams("birth_date"),
			status:    StatusFail,
			message:   "tag 1004 value must use YYYY-MM-DD format",
		},
		{
			name:      "date format accepts tag 0 UTC date-time",
			validator: MDocElementDateFormatValidator{},
			element:   tagged("issuance_date", &dateTimeTag, "2024-01-02T10:00:00Z"),
			params:    mdocParams("issuance_date"),
			status:    StatusPass,
		},
		{
			name:      "date format rejects tag 0 with offset",
			validator: MDocElementDateFormatValidator{},
			element:   tagged("issuance_date", &dateTimeTag, "2024-01-02T10:00:00+01:00"),
			params:    mdocParams("issuance_date"),
			status:    StatusFail,
			message:   "tag 0 value must use YYYY-MM-DDThh:mm:ssZ format",
		},
		{
			name:      "date format rejects tags other than 0 and 1004",
			validator: MDocElementDateFormatValidator{},
			element:   tagged("birth_date", &encodedCBORTag, "2000-02-29"),
			params:    mdocParams("birth_date"),
			status:    StatusFail,
			message:   "value must use CBOR tag 0 or 1004",
		},
		{
			name:      "date format rejects non-string tag content",
			validator: MDocElementDateFormatValidator{},
			element:   tagged("birth_date", &dateTag, []byte("2000-02-29")),
			params:    mdocParams("birth_date"),
			status:    StatusFail,
			message:   "expected string",
		},
		{
			name:      "valid date accepts tag 0 UTC date-time",
			validator: MDocElementValidDateValidator{},
			element:   tagged("issuance_date", &dateTimeTag, "2024-02-29T23:59:59Z"),
			params:    mdocParams("issuance_date"),
			status:    StatusPass,
		},
		{
			name:      "valid date rejects impossible tag 0 month",
			validator: MDocElementValidDateValidator{},
			element:   tagged("issuance_date", &dateTimeTag, "2024-13-01T10:00:00Z"),
			params:    mdocParams("issuance_date"),
			status:    StatusFail,
			message:   "not a valid UTC date-time",
		},
		{
			name:      "country code accepts ISO alpha-2",
			validator: MDocElementCountryCodeValidator{},
			element:   text("issuing_country", "IT"),
			params:    mdocParams("issuing_country"),
			status:    StatusPass,
		},
		{
			name:      "country code accepts user-assigned code",
			validator: MDocElementCountryCodeValidator{},
			element:   text("issuing_country", "XK"),
			params:    mdocParams("issuing_country"),
			status:    StatusPass,
		},
		{
			name:      "country code rejects lowercase code",
			validator: MDocElementCountryCodeValidator{},
			element:   text("issuing_country", "it"),
			params:    mdocParams("issuing_country"),
			status:    StatusFail,
			message:   `value "it" is not an accepted country code`,
		},
		{
			name:      "country code rejects non-text element",
			validator: MDocElementCountryCodeValidator{},
			element:   evidence.MDocElement{Identifier: "issuing_country", Value: uint64(380)},
			params:    mdocParams("issuing_country"),
			status:    StatusFail,
			message:   "CBOR major type is 0, expected text string type 3",
		},
		{
			name:      "email accepts bare addr-spec",
			validator: MDocElementRFC5322EmailValidator{},
			element:   text("email_address", "person@example.test"),
			params:    mdocParams("email_address"),
			status:    StatusPass,
		},
		{
			name:      "email rejects empty text",
			validator: MDocElementRFC5322EmailValidator{},
			element:   text("email_address", ""),
			params:    mdocParams("email_address"),
			status:    StatusFail,
			message:   "at least one character",
		},
		{
			name:      "email rejects display name form",
			validator: MDocElementRFC5322EmailValidator{},
			element:   text("email_address", "Alice <person@example.test>"),
			params:    mdocParams("email_address"),
			status:    StatusFail,
			message:   "not a bare RFC 5322 addr-spec",
		},
		{
			name:      "email rejects text without at sign",
			validator: MDocElementRFC5322EmailValidator{},
			element:   text("email_address", "person.example.test"),
			params:    mdocParams("email_address"),
			status:    StatusFail,
			message:   "not a bare RFC 5322 addr-spec",
		},
		{
			name:      "phone accepts international number at minimum length",
			validator: MDocElementInternationalPhoneValidator{},
			element:   text("mobile_phone_number", "+3906"),
			params:    mergeMDocParams("mobile_phone_number", map[string]any{"min_length": 5}),
			status:    StatusPass,
		},
		{
			name:      "phone rejects number below minimum length",
			validator: MDocElementInternationalPhoneValidator{},
			element:   text("mobile_phone_number", "+390"),
			params:    mergeMDocParams("mobile_phone_number", map[string]any{"min_length": 5}),
			status:    StatusFail,
			message:   "at least 5 characters",
		},
		{
			name:      "phone rejects number without plus prefix",
			validator: MDocElementInternationalPhoneValidator{},
			element:   text("mobile_phone_number", "0039061234"),
			params:    mergeMDocParams("mobile_phone_number", map[string]any{"min_length": 5}),
			status:    StatusFail,
			message:   "must start with + and contain digits only",
		},
		{
			name:      "phone rejects separators",
			validator: MDocElementInternationalPhoneValidator{},
			element:   text("mobile_phone_number", "+39 06 1234"),
			params:    mergeMDocParams("mobile_phone_number", map[string]any{"min_length": 5}),
			status:    StatusFail,
			message:   "must start with + and contain digits only",
		},
		{
			name:      "phone requires positive min_length",
			validator: MDocElementInternationalPhoneValidator{},
			element:   text("mobile_phone_number", "+39061234"),
			params:    mdocParams("mobile_phone_number"),
			status:    StatusError,
			message:   "positive min_length param is required",
		},
		{
			name:      "string array accepts text items",
			validator: MDocElementStringArrayValidator{},
			element:   array("given_names", []any{"Ada", "Augusta"}),
			params:    mergeMDocParams("given_names", map[string]any{"min_items": 2}),
			status:    StatusPass,
		},
		{
			name:      "string array rejects fewer items than min_items",
			validator: MDocElementStringArrayValidator{},
			element:   array("given_names", []any{"Ada"}),
			params:    mergeMDocParams("given_names", map[string]any{"min_items": 2}),
			status:    StatusFail,
			message:   "array must contain at least 2 item(s)",
		},
		{
			name:      "string array reports non-text item index",
			validator: MDocElementStringArrayValidator{},
			element:   array("given_names", []any{"Ada", uint64(1)}),
			params:    mergeMDocParams("given_names", map[string]any{"min_items": 1}),
			status:    StatusFail,
			message:   "item 1: value is uint64, expected string",
		},
		{
			name:      "string array rejects non-array major type",
			validator: MDocElementStringArrayValidator{},
			element:   text("given_names", "Ada"),
			params:    mergeMDocParams("given_names", map[string]any{"min_items": 1}),
			status:    StatusFail,
			message:   "CBOR major type is 3, expected array type 4",
		},
		{
			name:      "string array rejects undecoded array value",
			validator: MDocElementStringArrayValidator{},
			element: evidence.MDocElement{
				Identifier: "given_names",
				Value:      []string{"Ada"},
				MajorType:  4,
			},
			params:  mergeMDocParams("given_names", map[string]any{"min_items": 1}),
			status:  StatusFail,
			message: "decoded value is []string, expected array",
		},
		{
			name:      "string array requires positive min_items",
			validator: MDocElementStringArrayValidator{},
			element:   array("given_names", []any{"Ada"}),
			params:    mergeMDocParams("given_names", map[string]any{"min_items": 0}),
			status:    StatusError,
			message:   "positive min_items param is required",
		},
		{
			name:      "country array rejects empty array",
			validator: MDocElementCountryCodeArrayValidator{},
			element:   array("nationality", []any{}),
			params:    mergeMDocParams("nationality", map[string]any{"min_items": 1}),
			status:    StatusFail,
			message:   "array must contain at least 1 item(s)",
		},
		{
			name:      "country array reports non-text item",
			validator: MDocElementCountryCodeArrayValidator{},
			element:   array("nationality", []any{"IT", uint64(380)}),
			params:    mergeMDocParams("nationality", map[string]any{"min_items": 1}),
			status:    StatusFail,
			message:   "item 1: value is uint64, expected string",
		},
		{
			name:      "country array reports alpha-3 item",
			validator: MDocElementCountryCodeArrayValidator{},
			element:   array("nationality", []any{"IT", "ITA"}),
			params:    mergeMDocParams("nationality", map[string]any{"min_items": 1}),
			status:    StatusFail,
			message:   `item 1 value "ITA" is not an accepted country code`,
		},
		{
			name:      "map shape requires consistent property limits",
			validator: MDocElementMapShapeValidator{},
			element:   object("place_of_birth", map[string]any{"country": "IT"}),
			params: placeOfBirthParams(map[string]any{
				"allowed_keys":   []string{"country"},
				"min_properties": 2,
				"max_properties": 1,
			}),
			status:  StatusError,
			message: "valid allowed_keys and property limits are required",
		},
		{
			name:      "map shape rejects non-map major type",
			validator: MDocElementMapShapeValidator{},
			element:   text("place_of_birth", "Roma"),
			params: placeOfBirthParams(map[string]any{
				"allowed_keys":   []string{"country"},
				"max_properties": 1,
			}),
			status:  StatusFail,
			message: "CBOR major type is 3, expected map type 5",
		},
		{
			name:      "map shape rejects non string-keyed map",
			validator: MDocElementMapShapeValidator{},
			element: evidence.MDocElement{
				Identifier: "place_of_birth",
				Value:      map[any]any{uint64(1): "IT"},
				MajorType:  5,
			},
			params: placeOfBirthParams(map[string]any{
				"allowed_keys":   []string{"country"},
				"max_properties": 1,
			}),
			status:  StatusFail,
			message: "expected string-keyed map",
		},
		{
			name:      "map shape rejects too many entries",
			validator: MDocElementMapShapeValidator{},
			element: object("place_of_birth", map[string]any{
				"country":  "IT",
				"locality": "Roma",
			}),
			params: placeOfBirthParams(map[string]any{
				"allowed_keys":   []string{"country", "locality"},
				"min_properties": 1,
				"max_properties": 1,
			}),
			status:  StatusFail,
			message: "map has 2 entries, expected between 1 and 1",
		},
		{
			name:      "map shape rejects empty map below minimum",
			validator: MDocElementMapShapeValidator{},
			element:   object("place_of_birth", map[string]any{}),
			params: placeOfBirthParams(map[string]any{
				"allowed_keys":   []string{"country"},
				"min_properties": 1,
				"max_properties": 3,
			}),
			status:  StatusFail,
			message: "map has 0 entries, expected between 1 and 3",
		},
		{
			name:      "map shape rejects unsupported key",
			validator: MDocElementMapShapeValidator{},
			element:   object("place_of_birth", map[string]any{"planet": "Earth"}),
			params: placeOfBirthParams(map[string]any{
				"allowed_keys":   []string{"country", "region", "locality"},
				"max_properties": 3,
			}),
			status:  StatusFail,
			message: `map contains unsupported key "planet"`,
		},
		{
			name:      "map text values ignore absent optional keys",
			validator: MDocElementMapTextValuesValidator{},
			element:   object("place_of_birth", map[string]any{"locality": "Roma"}),
			params: placeOfBirthParams(map[string]any{
				"keys": []string{"country", "region", "locality"},
			}),
			status: StatusPass,
		},
		{
			name:      "map text values reject non-text listed value",
			validator: MDocElementMapTextValuesValidator{},
			element:   object("place_of_birth", map[string]any{"region": uint64(12)}),
			params: placeOfBirthParams(map[string]any{
				"keys": []string{"country", "region", "locality"},
			}),
			status:  StatusFail,
			message: `map value "region": value is uint64, expected string`,
		},
		{
			name:      "map text values do not inspect unlisted keys",
			validator: MDocElementMapTextValuesValidator{},
			element:   object("place_of_birth", map[string]any{"code": uint64(12)}),
			params:    placeOfBirthParams(map[string]any{"keys": []string{"country"}}),
			status:    StatusPass,
		},
		{
			name:      "map text values reject non-map element",
			validator: MDocElementMapTextValuesValidator{},
			element:   array("place_of_birth", []any{"Roma"}),
			params:    placeOfBirthParams(map[string]any{"keys": []string{"locality"}}),
			status:    StatusFail,
			message:   "value is not a CBOR map",
		},
		{
			name:      "map text values require keys",
			validator: MDocElementMapTextValuesValidator{},
			element:   object("place_of_birth", map[string]any{"locality": "Roma"}),
			params:    mdocParams("place_of_birth"),
			status:    StatusError,
			message:   "keys param is required",
		},
		{
			name:      "map member country rejects missing member",
			validator: MDocElementMapMemberCountryCodeValidator{},
			element:   object("place_of_birth", map[string]any{"locality": "Roma"}),
			params:    placeOfBirthParams(map[string]any{"member": "country"}),
			status:    StatusFail,
			message:   `map member "country" is missing`,
		},
		{
			name:      "map member country rejects country name",
			validator: MDocElementMapMemberCountryCodeValidator{},
			element:   object("place_of_birth", map[string]any{"country": "Italy"}),
			params:    placeOfBirthParams(map[string]any{"member": "country"}),
			status:    StatusFail,
			message:   `map member "country" is not an accepted country code`,
		},
		{
			name:      "map member country rejects numeric member",
			validator: MDocElementMapMemberCountryCodeValidator{},
			element:   object("place_of_birth", map[string]any{"country": uint64(380)}),
			params:    placeOfBirthParams(map[string]any{"member": "country"}),
			status:    StatusFail,
			message:   `map member "country": value is uint64, expected string`,
		},
		{
			name:      "map member country rejects non-map element",
			validator: MDocElementMapMemberCountryCodeValidator{},
			element:   text("place_of_birth", "IT"),
			params:    placeOfBirthParams(map[string]any{"member": "country"}),
			status:    StatusFail,
			message:   "value is not a CBOR map",
		},
		{
			name:      "map member country requires member",
			validator: MDocElementMapMemberCountryCodeValidator{},
			element:   object("place_of_birth", map[string]any{"country": "IT"}),
			params:    mdocParams("place_of_birth"),
			status:    StatusError,
			message:   "member param is required",
		},
		{
			name:      "map member max length accepts value at limit",
			validator: MDocElementMapMemberUTF8MaxLengthValidator{},
			element:   object("place_of_birth", map[string]any{"locality": "Forlì"}),
			params: placeOfBirthParams(map[string]any{
				"member":     "locality",
				"max_length": 5,
			}),
			status: StatusPass,
		},
		{
			name:      "map member max length counts characters not bytes",
			validator: MDocElementMapMemberUTF8MaxLengthValidator{},
			element:   object("place_of_birth", map[string]any{"locality": "Forlì"}),
			params: placeOfBirthParams(map[string]any{
				"member":     "locality",
				"max_length": 4,
			}),
			status:  StatusFail,
			message: `map member "locality" exceeds 4 characters`,
		},
		{
			name:      "map member max length rejects non-text member",
			validator: MDocElementMapMemberUTF8MaxLengthValidator{},
			element:   object("place_of_birth", map[string]any{"locality": []byte("Roma")}),
			params: placeOfBirthParams(map[string]any{
				"member":     "locality",
				"max_length": 10,
			}),
			status:  StatusFail,
			message: `map member "locality": value is []uint8, expected string`,
		},
		{
			name:      "map member max length requires positive max_length",
			validator: MDocElementMapMemberUTF8MaxLengthValidator{},
			element:   object("place_of_birth", map[string]any{"locality": "Roma"}),
			params:    placeOfBirthParams(map[string]any{"member": "locality"}),
			status:    StatusError,
			message:   "positive max_length param is required",
		},
		{
			name:      "unsigned integer rejects value outside allowed list",
			validator: MDocElementUnsignedIntegerAllowedValidator{},
			element:   evidence.MDocElement{Identifier: "sex", Value: uint64(7), MajorType: 0},
			params:    mergeMDocParams("sex", map[string]any{"allowed": []int{0, 1, 2}}),
			status:    StatusFail,
			message:   "value is not an allowed unsigned integer",
		},
		{
			name:      "unsigned integer rejects negative integer major type",
			validator: MDocElementUnsignedIntegerAllowedValidator{},
			element:   evidence.MDocElement{Identifier: "sex", Value: int64(-1), MajorType: 1},
			params:    mergeMDocParams("sex", map[string]any{"allowed": []int{0, 1, 2}}),
			status:    StatusFail,
			message:   "CBOR major type is 1, expected unsigned integer type 0",
		},
		{
			name:      "unsigned integer requires allowed values",
			validator: MDocElementUnsignedIntegerAllowedValidator{},
			element:   evidence.MDocElement{Identifier: "sex", Value: uint64(1), MajorType: 0},
			params:    mdocParams("sex"),
			status:    StatusError,
			message:   "allowed param is required",
		},
		{
			name:      "jpeg rejects text element",
			validator: MDocElementJPEGValidator{},
			element:   text("portrait", "ffd8"),
			params:    mdocParams("portrait"),
			status:    StatusFail,
			message:   "CBOR major type is 3, expected byte string type 2",
		},
		{
			name:      "jpeg rejects png bytes",
			validator: MDocElementJPEGValidator{},
			element: evidence.MDocElement{
				Identifier: "portrait",
				Value:      []byte{0x89, 0x50, 0x4e, 0x47},
				MajorType:  2,
			},
			params:  mdocParams("portrait"),
			status:  StatusFail,
			message: "byte string does not start with JPEG marker FF D8",
		},
		{
			name:      "jpeg rejects single byte",
			validator: MDocElementJPEGValidator{},
			element: evidence.MDocElement{
				Identifier: "portrait",
				Value:      []byte{0xff},
				MajorType:  2,
			},
			params:  mdocParams("portrait"),
			status:  StatusFail,
			message: "JPEG marker",
		},
		{
			name:      "subdivision matches issuing country",
			validator: MDocElementCountrySubdivisionValidator{},
			element:   text("resident_state", "IT-RM"),
			extra: map[string]evidence.MDocElement{
				"issuing_country": text("issuing_country", "IT"),
			},
			params: mergeMDocParams("resident_state", map[string]any{
				"country_element": "issuing_country",
			}),
			status: StatusPass,
		},
		{
			name:      "subdivision rejects prefix of another country",
			validator: MDocElementCountrySubdivisionValidator{},
			element:   text("resident_state", "FR-75"),
			extra: map[string]evidence.MDocElement{
				"issuing_country": text("issuing_country", "IT"),
			},
			params: mergeMDocParams("resident_state", map[string]any{
				"country_element": "issuing_country",
			}),
			status:  StatusFail,
			message: `subdivision country prefix "FR" does not match issuing country "IT"`,
		},
		{
			name:      "subdivision rejects code without separator",
			validator: MDocElementCountrySubdivisionValidator{},
			element:   text("resident_state", "ITRM"),
			extra: map[string]evidence.MDocElement{
				"issuing_country": text("issuing_country", "IT"),
			},
			params: mergeMDocParams("resident_state", map[string]any{
				"country_element": "issuing_country",
			}),
			status:  StatusFail,
			message: "ISO 3166-2 country-subdivision shape",
		},
		{
			name:      "subdivision requires issuing country element",
			validator: MDocElementCountrySubdivisionValidator{},
			element:   text("resident_state", "IT-RM"),
			params: mergeMDocParams("resident_state", map[string]any{
				"country_element": "issuing_country",
			}),
			status:  StatusFail,
			message: `country element "issuing_country" is missing`,
		},
		{
			name:      "subdivision rejects non-text issuing country",
			validator: MDocElementCountrySubdivisionValidator{},
			element:   text("resident_state", "IT-RM"),
			extra: map[string]evidence.MDocElement{"issuing_country": {
				Identifier: "issuing_country",
				Value:      uint64(380),
			}},
			params: mergeMDocParams("resident_state", map[string]any{
				"country_element": "issuing_country",
			}),
			status:  StatusFail,
			message: `country element "issuing_country": value is uint64, expected string`,
		},
		{
			name:      "subdivision requires country_element",
			validator: MDocElementCountrySubdivisionValidator{},
			element:   text("resident_state", "IT-RM"),
			params:    mdocParams("resident_state"),
			status:    StatusError,
			message:   "country_element param is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			elements := map[string]evidence.MDocElement{test.element.Identifier: test.element}
			for identifier, element := range test.extra {
				elements[identifier] = element
			}
			result := test.validator.Validate(context.Background(), Input{
				Value:  mdocValidatorPresentation(elements),
				Params: test.params,
			})
			require.Equal(t, test.status, result.Status, result.Message)
			require.Contains(t, result.Message, test.message)
		})
	}
}

func TestPIDMDocTypeValidatorAcceptsPresentationEncodings(t *testing.T) {
	token := validMDocPresentation(t)
	decoded := mdocValidatorPresentation(nil)

	tests := []struct {
		name    string
		value   any
		status  Status
		message string
	}{
		{name: "decoded presentation pointer", value: decoded, status: StatusPass},
		{name: "decoded presentation value", value: *decoded, status: StatusPass},
		{name: "base64url DeviceResponse", value: token, status: StatusPass},
		{
			name:   "vp_token JSON string",
			value:  `{"pid":["` + token + `"]}`,
			status: StatusPass,
		},
		{
			name:   "vp_token JSON object",
			value:  map[string]any{"pid": []any{token}},
			status: StatusPass,
		},
		{
			name: "other document type",
			value: &evidence.MDocPresentation{
				Documents: []evidence.MDocDocument{{DocType: "org.iso.18013.5.1.mDL"}},
			},
			status:  StatusFail,
			message: `mdoc document type "eu.europa.ec.eudi.pid.1" is missing`,
		},
		{
			name:    "vp_token JSON with ambiguous credential entries",
			value:   map[string]any{"pid": []any{token}, "mdl": []any{token}},
			status:  StatusFail,
			message: "input is map[string]interface {}, expected decoded mdoc presentation",
		},
		{
			name:    "malformed token",
			value:   "not-an-mdoc",
			status:  StatusFail,
			message: "input is string, expected decoded mdoc presentation",
		},
		{
			name:    "nil presentation pointer",
			value:   (*evidence.MDocPresentation)(nil),
			status:  StatusFail,
			message: "input is *evidence.MDocPresentation",
		},
		{
			name:    "unsupported input type",
			value:   42,
			status:  StatusFail,
			message: "input is int, expected decoded mdoc presentation",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := PIDMDocTypeValidator{}.Validate(
				context.Background(),
				Input{Value: test.value},
			)
			require.Equal(t, test.status, result.Status, result.Message)
			require.Contains(t, result.Message, test.message)
		})
	}
}

func TestPIDMDocMandatoryElementsValidator(t *testing.T) {
	presentation := mdocValidatorPresentation(map[string]evidence.MDocElement{
		"family_name": {Identifier: "family_name", Value: "Trotter", MajorType: 3},
		"given_name":  {Identifier: "given_name", Value: "Del", MajorType: 3},
	})

	tests := []struct {
		name    string
		value   any
		params  map[string]any
		status  Status
		message string
	}{
		{
			name:  "all required elements present",
			value: presentation,
			params: map[string]any{
				"namespace":         pidMDocType,
				"required_elements": []string{"family_name", "given_name"},
			},
			status: StatusPass,
		},
		{
			name:  "lists every missing element",
			value: presentation,
			params: map[string]any{
				"namespace":         pidMDocType,
				"required_elements": []string{"family_name", "birth_date", "nationality"},
			},
			status:  StatusFail,
			message: "required mandatory PID mdoc elements are missing: [birth_date nationality]",
		},
		{
			name:  "elements in another namespace do not count",
			value: presentation,
			params: map[string]any{
				"namespace":         "org.iso.18013.5.1",
				"required_elements": []string{"family_name"},
			},
			status:  StatusFail,
			message: "[family_name]",
		},
		{
			name:    "requires required_elements",
			value:   presentation,
			params:  map[string]any{"namespace": pidMDocType},
			status:  StatusError,
			message: "namespace and required_elements params are required",
		},
		{
			name:  "rejects undecodable input",
			value: "not-an-mdoc",
			params: map[string]any{
				"namespace":         pidMDocType,
				"required_elements": []string{"family_name"},
			},
			status:  StatusFail,
			message: "expected decoded mdoc presentation",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := PIDMDocMandatoryElementsValidator{}.Validate(context.Background(), Input{
				Value:  test.value,
				Params: test.params,
			})
			require.Equal(t, test.status, result.Status, result.Message)
			require.Contains(t, result.Message, test.message)
		})
	}
}

func TestMDocDomesticNamespaceValidator(t *testing.T) {
	element := map[string]evidence.MDocElement{
		"tax_id": {Identifier: "tax_id", Value: "RSSMRA", MajorType: 3},
	}

	tests := []struct {
		name       string
		namespaces map[string]map[string]evidence.MDocElement
		status     Status
		message    string
	}{
		{
			name: "accepts versioned domestic namespace",
			namespaces: map[string]map[string]evidence.MDocElement{
				pidMDocType:                  element,
				"eu.europa.ec.eudi.pid.IT.1": element,
			},
			status:  StatusPass,
			message: `"eu.europa.ec.eudi.pid.IT.1"`,
		},
		{
			name: "accepts subdivision domestic namespace",
			namespaces: map[string]map[string]evidence.MDocElement{
				"eu.europa.ec.eudi.pid.DE-BY": element,
			},
			status: StatusPass,
		},
		{
			name: "rejects unassigned country code",
			namespaces: map[string]map[string]evidence.MDocElement{
				"eu.europa.ec.eudi.pid.OO.1": element,
			},
			status:  StatusFail,
			message: "has an invalid country code",
		},
		{
			name: "rejects empty domestic namespace",
			namespaces: map[string]map[string]evidence.MDocElement{
				"eu.europa.ec.eudi.pid.IT.1": {},
			},
			status:  StatusFail,
			message: "contains no elements",
		},
		{
			name: "ignores lowercase country and base namespace",
			namespaces: map[string]map[string]evidence.MDocElement{
				pidMDocType:                  element,
				"eu.europa.ec.eudi.pid.it.1": element,
			},
			status:  StatusFail,
			message: "no valid non-empty PID domestic namespace is present",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := MDocDomesticNamespaceValidator{}.Validate(context.Background(), Input{
				Value: &evidence.MDocPresentation{Namespaces: test.namespaces},
			})
			require.Equal(t, test.status, result.Status, result.Message)
			require.Contains(t, result.Message, test.message)
		})
	}

	result := MDocDomesticNamespaceValidator{}.Validate(context.Background(), Input{Value: 1})
	require.Equal(t, StatusFail, result.Status)
}
