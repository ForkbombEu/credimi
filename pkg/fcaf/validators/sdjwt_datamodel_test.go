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

func TestSDJWTDataModelValidatorOutcomes(t *testing.T) {
	claims := map[string]any{
		"family_name":          "Trotter",
		"empty_name":           "",
		"age":                  float64(36),
		"birthdate":            float64(20000229),
		"given_names":          []any{"Ada", "Augusta"},
		"mixed_names":          []any{"Ada", float64(1)},
		"place_of_birth":       map[string]any{"country": "IT", "locality": "Forlì"},
		"address":              map[string]any{"street": float64(12), "region": "Lazio"},
		"issuing_country":      "IT",
		"numeric_country":      float64(380),
		"issuing_jurisdiction": "IT-RM",
		"foreign_jurisdiction": "FR-75",
		"bad_jurisdiction":     "ITRM",
		"numeric_jurisdiction": float64(1),
	}

	tests := []struct {
		name      string
		validator Validator
		params    map[string]any
		status    Status
		message   string
	}{
		{
			name:      "non-empty string accepts text",
			validator: SDJWTClaimNonEmptyUTF8StringValidator{},
			params:    map[string]any{"claim": "family_name"},
			status:    StatusPass,
		},
		{
			name:      "non-empty string rejects empty text",
			validator: SDJWTClaimNonEmptyUTF8StringValidator{},
			params:    map[string]any{"claim": "empty_name"},
			status:    StatusFail,
			message:   `claim "empty_name" is invalid: value must contain at least one character`,
		},
		{
			name:      "non-empty string rejects number",
			validator: SDJWTClaimNonEmptyUTF8StringValidator{},
			params:    map[string]any{"claim": "age"},
			status:    StatusFail,
			message:   `claim "age" is invalid: value is float64, expected string`,
		},
		{
			name:      "non-empty string reports missing claim",
			validator: SDJWTClaimNonEmptyUTF8StringValidator{},
			params:    map[string]any{"claim": "given_name"},
			status:    StatusFail,
			message:   `claim "given_name" is missing`,
		},
		{
			name:      "phone requires positive min_length",
			validator: SDJWTClaimInternationalPhoneValidator{},
			params:    map[string]any{"claim": "phone_number"},
			status:    StatusError,
			message:   "min_length must be greater than zero",
		},
		{
			name:      "phone reports missing claim",
			validator: SDJWTClaimInternationalPhoneValidator{},
			params:    map[string]any{"claim": "phone_number", "min_length": 8},
			status:    StatusFail,
			message:   `claim "phone_number" is missing`,
		},
		{
			name:      "country code reports missing claim",
			validator: SDJWTClaimCountryCodeValidator{},
			params:    map[string]any{"claim": "nationality"},
			status:    StatusFail,
			message:   `claim "nationality" is missing`,
		},
		{
			name:      "country code rejects numeric code",
			validator: SDJWTClaimCountryCodeValidator{},
			params:    map[string]any{"claim": "numeric_country"},
			status:    StatusFail,
			message:   "value is float64, expected string",
		},
		{
			name:      "date format rejects numeric date",
			validator: SDJWTClaimDateFormatValidator{},
			params:    map[string]any{"claim": "birthdate"},
			status:    StatusFail,
			message:   `claim "birthdate" is invalid: value is float64, expected string`,
		},
		{
			name:      "valid date reports missing claim",
			validator: SDJWTClaimValidDateValidator{},
			params:    map[string]any{"claim": "expiry_date"},
			status:    StatusFail,
			message:   `claim "expiry_date" is missing`,
		},
		{
			name:      "string array accepts text items at min_items",
			validator: SDJWTClaimStringArrayValidator{},
			params:    map[string]any{"claim": "given_names", "min_items": 2},
			status:    StatusPass,
		},
		{
			name:      "string array rejects fewer items than min_items",
			validator: SDJWTClaimStringArrayValidator{},
			params:    map[string]any{"claim": "given_names", "min_items": 3},
			status:    StatusFail,
			message:   "array must contain at least 3 item(s)",
		},
		{
			name:      "string array reports non-text item index",
			validator: SDJWTClaimStringArrayValidator{},
			params:    map[string]any{"claim": "mixed_names", "min_items": 1},
			status:    StatusFail,
			message:   "item 1: value is float64, expected string",
		},
		{
			name:      "string array rejects scalar claim",
			validator: SDJWTClaimStringArrayValidator{},
			params:    map[string]any{"claim": "family_name", "min_items": 1},
			status:    StatusFail,
			message:   `claim "family_name" is invalid: value is string, expected array`,
		},
		{
			name:      "string array reports missing claim",
			validator: SDJWTClaimStringArrayValidator{},
			params:    map[string]any{"claim": "nicknames", "min_items": 1},
			status:    StatusFail,
			message:   `claim "nicknames" is missing`,
		},
		{
			name:      "string array requires positive min_items",
			validator: SDJWTClaimStringArrayValidator{},
			params:    map[string]any{"claim": "given_names", "min_items": 0},
			status:    StatusError,
			message:   "min_items must be greater than zero",
		},
		{
			name:      "country array reports missing claim",
			validator: SDJWTClaimCountryCodeArrayValidator{},
			params:    map[string]any{"claim": "nationalities", "min_items": 1},
			status:    StatusFail,
			message:   `claim "nationalities" is missing`,
		},
		{
			name:      "object accepts object claim",
			validator: SDJWTClaimObjectValidator{},
			params:    map[string]any{"claim": "place_of_birth"},
			status:    StatusPass,
		},
		{
			name:      "object rejects array claim",
			validator: SDJWTClaimObjectValidator{},
			params:    map[string]any{"claim": "given_names"},
			status:    StatusFail,
			message:   "value is []interface {}, expected object",
		},
		{
			name:      "object reports missing claim",
			validator: SDJWTClaimObjectValidator{},
			params:    map[string]any{"claim": "residence"},
			status:    StatusFail,
			message:   `claim "residence" is missing`,
		},
		{
			name:      "object keys rejects too many properties",
			validator: SDJWTClaimObjectKeysValidator{},
			params: map[string]any{
				"claim":          "place_of_birth",
				"allowed":        []string{"country", "locality"},
				"max_properties": 1,
			},
			status:  StatusFail,
			message: "object has 2 properties, expected between 0 and 1",
		},
		{
			name:      "object keys rejects too few properties",
			validator: SDJWTClaimObjectKeysValidator{},
			params: map[string]any{
				"claim":          "place_of_birth",
				"allowed":        []string{"country", "locality"},
				"min_properties": 3,
			},
			status:  StatusFail,
			message: "object has 2 properties",
		},
		{
			name:      "object keys treats zero max_properties as unbounded",
			validator: SDJWTClaimObjectKeysValidator{},
			params: map[string]any{
				"claim":   "place_of_birth",
				"allowed": []string{"country", "locality"},
			},
			status: StatusPass,
		},
		{
			name:      "object keys rejects scalar claim",
			validator: SDJWTClaimObjectKeysValidator{},
			params: map[string]any{
				"claim":   "family_name",
				"allowed": []string{"country"},
			},
			status:  StatusFail,
			message: "value is string, expected object",
		},
		{
			name:      "object string values reject numeric listed property",
			validator: SDJWTClaimObjectStringValuesValidator{},
			params: map[string]any{
				"claim": "address",
				"keys":  []string{"street", "region"},
			},
			status:  StatusFail,
			message: `claim "address" is invalid: property "street": value is float64, expected string`,
		},
		{
			name:      "object string values ignore unlisted properties",
			validator: SDJWTClaimObjectStringValuesValidator{},
			params:    map[string]any{"claim": "address", "keys": []string{"region", "locality"}},
			status:    StatusPass,
		},
		{
			name:      "object string values report missing claim",
			validator: SDJWTClaimObjectStringValuesValidator{},
			params:    map[string]any{"claim": "residence", "keys": []string{"region"}},
			status:    StatusFail,
			message:   `claim "residence" is missing`,
		},
		{
			name:      "nested max length counts characters at boundary",
			validator: SDJWTClaimNestedStringMaxLengthValidator{},
			params: map[string]any{
				"claim":      "place_of_birth",
				"member":     "locality",
				"max_length": 5,
			},
			status: StatusPass,
		},
		{
			name:      "nested max length rejects longer value",
			validator: SDJWTClaimNestedStringMaxLengthValidator{},
			params: map[string]any{
				"claim":      "place_of_birth",
				"member":     "locality",
				"max_length": 4,
			},
			status:  StatusFail,
			message: `claim "place_of_birth.locality" is invalid: value exceeds 4 characters`,
		},
		{
			name:      "nested max length rejects numeric member",
			validator: SDJWTClaimNestedStringMaxLengthValidator{},
			params: map[string]any{
				"claim":      "address",
				"member":     "street",
				"max_length": 10,
			},
			status:  StatusFail,
			message: `claim "address.street" is invalid: value is float64, expected string`,
		},
		{
			name:      "nested max length reports missing member",
			validator: SDJWTClaimNestedStringMaxLengthValidator{},
			params: map[string]any{
				"claim":      "place_of_birth",
				"member":     "region",
				"max_length": 10,
			},
			status:  StatusFail,
			message: `claim "place_of_birth.region" is missing`,
		},
		{
			name:      "integer allowed reports missing claim",
			validator: SDJWTClaimIntegerAllowedValidator{},
			params:    map[string]any{"claim": "sex", "allowed": []int{0, 1}},
			status:    StatusFail,
			message:   `claim "sex" is missing`,
		},
		{
			name:      "jpeg data URL reports missing claim",
			validator: SDJWTClaimJPEGDataURLValidator{},
			params:    map[string]any{"claim": "picture"},
			status:    StatusFail,
			message:   `claim "picture" is missing`,
		},
		{
			name:      "subdivision rejects other country prefix",
			validator: SDJWTClaimCountrySubdivisionValidator{},
			params: map[string]any{
				"claim":         "foreign_jurisdiction",
				"country_claim": "issuing_country",
			},
			status:  StatusFail,
			message: `subdivision country prefix "FR" does not match issuing country "IT"`,
		},
		{
			name:      "subdivision rejects malformed code",
			validator: SDJWTClaimCountrySubdivisionValidator{},
			params: map[string]any{
				"claim":         "bad_jurisdiction",
				"country_claim": "issuing_country",
			},
			status:  StatusFail,
			message: "ISO 3166-2 country-subdivision shape",
		},
		{
			name:      "subdivision rejects numeric code",
			validator: SDJWTClaimCountrySubdivisionValidator{},
			params: map[string]any{
				"claim":         "numeric_jurisdiction",
				"country_claim": "issuing_country",
			},
			status:  StatusFail,
			message: `claim "numeric_jurisdiction" is invalid`,
		},
		{
			name:      "subdivision reports missing claim",
			validator: SDJWTClaimCountrySubdivisionValidator{},
			params: map[string]any{
				"claim":         "resident_state",
				"country_claim": "issuing_country",
			},
			status:  StatusFail,
			message: `claim "resident_state" is missing`,
		},
		{
			name:      "subdivision reports missing country claim",
			validator: SDJWTClaimCountrySubdivisionValidator{},
			params: map[string]any{
				"claim":         "issuing_jurisdiction",
				"country_claim": "resident_country",
			},
			status:  StatusFail,
			message: `claim "resident_country" is missing`,
		},
		{
			name:      "subdivision rejects numeric country claim",
			validator: SDJWTClaimCountrySubdivisionValidator{},
			params: map[string]any{
				"claim":         "issuing_jurisdiction",
				"country_claim": "numeric_country",
			},
			status:  StatusFail,
			message: `claim "numeric_country" is invalid: value is float64, expected string`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.validator.Validate(context.Background(), Input{
				Value:  claims,
				Params: test.params,
			})

			require.Equal(t, test.status, result.Status, result.Message)
			require.Contains(t, result.Message, test.message)
		})
	}
}

func TestSDJWTDomesticNamespaceValidator(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		status  Status
		message string
	}{
		{
			name:    "accepts domestic type with lowercase country",
			value:   map[string]any{"vct": "urn:eudi:pid:de:1", "tax_id": "123"},
			status:  StatusPass,
			message: `"urn:eudi:pid:de:1" is a valid domestic PID type`,
		},
		{
			name: "accepts user-assigned subdivision type in compact presentation",
			value: testSDJWTPresentation(map[string]any{
				"vct":                    "urn:eudi:pid:XX-AB:2",
				"credimi_domestic_claim": "credimi-domestic-value",
			}),
			status: StatusPass,
		},
		{
			name:    "rejects EU-wide base type",
			value:   map[string]any{"vct": "urn:eudi:pid:1"},
			status:  StatusFail,
			message: `"urn:eudi:pid:1" is not a domestic PID type`,
		},
		{
			name:    "rejects domestic type without version",
			value:   map[string]any{"vct": "urn:eudi:pid:de"},
			status:  StatusFail,
			message: "is not a domestic PID type",
		},
		{
			name:    "rejects unassigned country code",
			value:   map[string]any{"vct": "urn:eudi:pid:oo:1"},
			status:  StatusFail,
			message: `domestic PID type "urn:eudi:pid:oo:1" has an invalid country code`,
		},
		{
			name:    "rejects missing vct",
			value:   map[string]any{"given_name": "Ada"},
			status:  StatusFail,
			message: `claim "vct" is missing`,
		},
		{
			name:    "rejects non-string vct",
			value:   map[string]any{"vct": 1},
			status:  StatusFail,
			message: `claim "vct" is int, expected string`,
		},
		{
			name:    "rejects non SD-JWT input",
			value:   42,
			status:  StatusFail,
			message: "input is int, expected SD-JWT claims",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SDJWTDomesticNamespaceValidator{}.Validate(
				context.Background(),
				Input{Value: test.value},
			)

			require.Equal(t, test.status, result.Status, result.Message)
			require.Contains(t, result.Message, test.message)
		})
	}
}
