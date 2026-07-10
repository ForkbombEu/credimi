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
