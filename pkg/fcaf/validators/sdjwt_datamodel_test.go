// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/stretchr/testify/require"
)

func TestSDJWTNestedClaimResolution(t *testing.T) {
	input := Input{
		Value: &evidence.SDJWTPresentation{Claims: map[string]any{
			"address": map[string]any{"country": "IT"},
		}},
		Params: map[string]any{"claim": "address.country"},
	}

	result := SDJWTClaimCountryCodeValidator{}.Validate(context.Background(), input)

	require.Equal(t, StatusPass, result.Status)
}

func TestSDJWTClaimInternationalPhoneValidator(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		status Status
	}{
		{name: "valid", value: "+3901234", status: StatusPass},
		{name: "too short", value: "+391234", status: StatusFail},
		{name: "missing plus", value: "39012345", status: StatusFail},
		{name: "non digit", value: "+39012A4", status: StatusFail},
		{name: "wrong type", value: 39012345, status: StatusFail},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SDJWTClaimInternationalPhoneValidator{}.Validate(context.Background(), Input{
				Value:  map[string]any{"phone_number": test.value},
				Params: map[string]any{"claim": "phone_number", "min_length": 8},
			})

			require.Equal(t, test.status, result.Status)
		})
	}
}

func TestSDJWTClaimCountryCodeValidator(t *testing.T) {
	tests := []struct {
		value  string
		status Status
	}{
		{value: "IT", status: StatusPass},
		{value: "QM", status: StatusPass},
		{value: "XZ", status: StatusPass},
		{value: "ZZ", status: StatusPass},
		{value: "UK", status: StatusFail},
		{value: "it", status: StatusFail},
		{value: "ITA", status: StatusFail},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			result := SDJWTClaimCountryCodeValidator{}.Validate(context.Background(), Input{
				Value:  map[string]any{"issuing_country": test.value},
				Params: map[string]any{"claim": "issuing_country"},
			})

			require.Equal(t, test.status, result.Status)
		})
	}
}

func TestSDJWTClaimDateValidators(t *testing.T) {
	tests := []struct {
		value        string
		formatStatus Status
		dateStatus   Status
	}{
		{value: "2024-02-29", formatStatus: StatusPass, dateStatus: StatusPass},
		{value: "0000-01-01", formatStatus: StatusPass, dateStatus: StatusPass},
		{value: "2023-02-29", formatStatus: StatusPass, dateStatus: StatusFail},
		{value: "2024-13-01", formatStatus: StatusPass, dateStatus: StatusFail},
		{value: "2024-1-01", formatStatus: StatusFail, dateStatus: StatusFail},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			input := Input{
				Value:  map[string]any{"birthdate": test.value},
				Params: map[string]any{"claim": "birthdate"},
			}

			format := SDJWTClaimDateFormatValidator{}.Validate(context.Background(), input)
			date := SDJWTClaimValidDateValidator{}.Validate(context.Background(), input)

			require.Equal(t, test.formatStatus, format.Status)
			require.Equal(t, test.dateStatus, date.Status)
		})
	}
}

func TestSDJWTClaimCountryCodeArrayValidator(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		status Status
	}{
		{name: "valid", value: []any{"IT", "GR"}, status: StatusPass},
		{name: "empty", value: []any{}, status: StatusFail},
		{name: "mixed types", value: []any{"IT", 10}, status: StatusFail},
		{name: "invalid code", value: []any{"UK"}, status: StatusFail},
		{name: "not array", value: "IT", status: StatusFail},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SDJWTClaimCountryCodeArrayValidator{}.Validate(context.Background(), Input{
				Value:  map[string]any{"nationalities": test.value},
				Params: map[string]any{"claim": "nationalities", "min_items": 1},
			})

			require.Equal(t, test.status, result.Status)
		})
	}
}

func TestSDJWTClaimObjectValidators(t *testing.T) {
	value := map[string]any{
		"place_of_birth": map[string]any{
			"country":  "IT",
			"locality": "Roma",
		},
	}

	shape := SDJWTClaimObjectKeysValidator{}.Validate(context.Background(), Input{
		Value: value,
		Params: map[string]any{
			"claim":          "place_of_birth",
			"allowed":        []string{"country", "region", "locality"},
			"min_properties": 1,
			"max_properties": 3,
		},
	})
	stringsResult := SDJWTClaimObjectStringValuesValidator{}.Validate(context.Background(), Input{
		Value: value,
		Params: map[string]any{
			"claim": "place_of_birth",
			"keys":  []string{"country", "region", "locality"},
		},
	})

	require.Equal(t, StatusPass, shape.Status)
	require.Equal(t, StatusPass, stringsResult.Status)
}

func TestSDJWTClaimObjectKeysRejectsUnknownProperty(t *testing.T) {
	result := SDJWTClaimObjectKeysValidator{}.Validate(context.Background(), Input{
		Value: map[string]any{
			"place_of_birth": map[string]any{"unknown": "value"},
		},
		Params: map[string]any{
			"claim":          "place_of_birth",
			"allowed":        []string{"country", "region", "locality"},
			"min_properties": 1,
			"max_properties": 3,
		},
	})

	require.Equal(t, StatusFail, result.Status)
}

func TestSDJWTClaimIntegerAllowedValidator(t *testing.T) {
	tests := []struct {
		name   string
		value  any
		status Status
	}{
		{name: "integer", value: float64(6), status: StatusPass},
		{name: "fraction", value: 1.5, status: StatusFail},
		{name: "outside set", value: float64(7), status: StatusFail},
		{name: "string", value: "1", status: StatusFail},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SDJWTClaimIntegerAllowedValidator{}.Validate(context.Background(), Input{
				Value: map[string]any{"sex": test.value},
				Params: map[string]any{
					"claim":   "sex",
					"allowed": []int{0, 1, 2, 3, 4, 5, 6, 9},
				},
			})

			require.Equal(t, test.status, result.Status)
		})
	}
}

func TestSDJWTClaimJPEGDataURLValidator(t *testing.T) {
	valid := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString([]byte{0xff, 0xd8, 0xff})

	result := SDJWTClaimJPEGDataURLValidator{}.Validate(context.Background(), Input{
		Value:  map[string]any{"picture": valid},
		Params: map[string]any{"claim": "picture"},
	})
	invalid := SDJWTClaimJPEGDataURLValidator{}.Validate(context.Background(), Input{
		Value:  map[string]any{"picture": "data:image/png;base64,iVBORw0KGgo="},
		Params: map[string]any{"claim": "picture"},
	})

	require.Equal(t, StatusPass, result.Status)
	require.Equal(t, StatusFail, invalid.Status)
}

func TestSDJWTClaimCountrySubdivisionValidator(t *testing.T) {
	result := SDJWTClaimCountrySubdivisionValidator{}.Validate(context.Background(), Input{
		Value: map[string]any{
			"issuing_country":      "GR",
			"issuing_jurisdiction": "GR-I",
		},
		Params: map[string]any{
			"claim":         "issuing_jurisdiction",
			"country_claim": "issuing_country",
		},
	})

	require.Equal(t, StatusPass, result.Status)
}

func TestSDJWTDataModelValidatorsRejectMissingConfiguration(t *testing.T) {
	validators := []Validator{
		SDJWTClaimNonEmptyUTF8StringValidator{},
		SDJWTClaimInternationalPhoneValidator{},
		SDJWTClaimCountryCodeValidator{},
		SDJWTClaimDateFormatValidator{},
		SDJWTClaimValidDateValidator{},
		SDJWTClaimStringArrayValidator{},
		SDJWTClaimCountryCodeArrayValidator{},
		SDJWTClaimObjectValidator{},
		SDJWTClaimObjectKeysValidator{},
		SDJWTClaimObjectStringValuesValidator{},
		SDJWTClaimNestedStringMaxLengthValidator{},
		SDJWTClaimIntegerAllowedValidator{},
		SDJWTClaimJPEGDataURLValidator{},
		SDJWTClaimCountrySubdivisionValidator{},
	}
	for _, validator := range validators {
		t.Run(validator.ID(), func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{})
			require.Equal(t, StatusError, result.Status)
		})
	}
}
