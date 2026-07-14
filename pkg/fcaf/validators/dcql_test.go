// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDCQLResponseConstraintsValidator(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		property     string
		expectedType string
		valid        bool
		evidence     map[string]any
		status       Status
	}{
		{
			name: "credential sets",
			mode: "credential_sets",
			evidence: map[string]any{
				"request": map[string]any{
					"dcql_query": map[string]any{
						"credential_sets": []any{map[string]any{"options": []any{[]any{"pid"}}}},
					},
				},
				"wallet_response": map[string]any{
					"vp_token": map[string]any{"pid": []any{"presentation"}},
				},
			},
			status: StatusPass,
		},
		{
			name: "credentials matched",
			mode: "credentials_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "credentials matched without credential sets",
			mode: "without_credential_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "credentials matched without trusted authorities",
			mode: "without_trusted_authorities",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "trusted authorities unexpectedly present",
			mode: "without_trusted_authorities",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"type":   "aki",
							"values": []any{"authority-key-id"},
						}}
						return credential
					}()},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "credentials matched without claims",
			mode: "without_claims",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						delete(credential, "claims")
						return credential
					}()},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "claims unexpectedly present",
			mode: "without_claims",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["claims"] = []any{map[string]any{"path": []any{"given_name"}}}
						return credential
					}()},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "empty claims rejected",
			mode: "empty_claims",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{map[string]any{
						"id":     "pid",
						"format": "dc+sd-jwt",
						"meta":   map[string]any{"vct_values": []any{"urn:eudi:pid:1"}},
						"claims": []any{},
					}},
				},
			},
			status: StatusPass,
		},
		{
			name: "empty claims incorrectly returns credential",
			mode: "empty_claims",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{map[string]any{
						"id":     "pid",
						"format": "dc+sd-jwt",
						"meta":   map[string]any{"vct_values": []any{"urn:eudi:pid:1"}},
						"claims": []any{},
					}},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "non-empty claims do not satisfy empty claims case",
			mode: "empty_claims",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
				},
			},
			status: StatusFail,
		},
		{
			name:     "empty claim sets rejected",
			mode:     "empty_array",
			property: "claim_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{map[string]any{"claim_sets": []any{}}},
				},
			},
			status: StatusPass,
		},
		{
			name:     "non-empty claim sets do not satisfy empty array case",
			mode:     "empty_array",
			property: "claim_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{map[string]any{"claim_sets": []any{[]any{"given_name"}}}},
				},
			},
			status: StatusFail,
		},
		{
			name:     "empty claim sets incorrectly returns credential",
			mode:     "empty_array",
			property: "claim_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{map[string]any{"claim_sets": []any{}}},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "omitted multiple returns one credential",
			mode: "multiple_default_false",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "omitted multiple returns more than one credential",
			mode: "multiple_default_false",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
				},
				"vp_token": map[string]any{"pid": []any{"presentation-1", "presentation-2"}},
			},
			status: StatusFail,
		},
		{
			name: "multiple is present instead of omitted",
			mode: "multiple_default_false",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["multiple"] = false
						return credential
					}()},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "multiple true returns multiple credentials",
			mode: "multiple_true",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{credentialQueryWithMultiple("pid", true)},
				},
				"vp_token": map[string]any{"pid": []any{"presentation-1", "presentation-2"}},
			},
			status: StatusPass,
		},
		{
			name: "multiple true returns one credential",
			mode: "multiple_true",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{credentialQueryWithMultiple("pid", true)},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "multiple false cannot satisfy multiple true mode",
			mode: "multiple_true",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{credentialQueryWithMultiple("pid", false)},
				},
				"vp_token": map[string]any{"pid": []any{"presentation-1", "presentation-2"}},
			},
			status: StatusFail,
		},
		{
			name: "credential sets unexpectedly present",
			mode: "without_credential_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials":     []any{validSDJWTCredentialQuery("pid")},
					"credential_sets": []any{map[string]any{"options": []any{[]any{"pid"}}}},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "credential entry missing format",
			mode: "credentials_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{map[string]any{"id": "pid", "meta": map[string]any{}}},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "credential entry has unsupported format",
			mode: "credentials_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{map[string]any{
						"id":     "pid",
						"format": "jwt_vc_json",
						"meta":   map[string]any{},
					}},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "credential entries have duplicate ids",
			mode: "credentials_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{
						validSDJWTCredentialQuery("pid"),
						validSDJWTCredentialQuery("pid"),
					},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "credential missing",
			mode: "credentials_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
				},
				"vp_token": map[string]any{},
			},
			status: StatusFail,
		},
		{
			name: "no match error",
			mode: "no_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("unknown")},
				},
				"error": "invalid_request",
			},
			status: StatusPass,
		},
		{
			name:         "non-boolean property is rejected",
			mode:         "property_type",
			property:     "require_cryptographic_holder_binding",
			expectedType: "boolean",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["require_cryptographic_holder_binding"] = "true"
						return credential
					}()},
				},
				"error": "invalid_request",
			},
			status: StatusPass,
		},
		{
			name:         "boolean holder binding returns a credential",
			mode:         "property_type",
			property:     "require_cryptographic_holder_binding",
			expectedType: "boolean",
			valid:        true,
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["require_cryptographic_holder_binding"] = true
						return credential
					}()},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name:         "malformed holder binding returning a credential fails",
			mode:         "property_type",
			property:     "require_cryptographic_holder_binding",
			expectedType: "boolean",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["require_cryptographic_holder_binding"] = "true"
						return credential
					}()},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name:         "non-string trusted authority type is rejected",
			mode:         "trusted_authority_property_type",
			property:     "type",
			expectedType: "string",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"type":   true,
							"values": []any{"authority-key-id"},
						}}
						return credential
					}()},
				},
			},
			status: StatusPass,
		},
		{
			name:         "string trusted authority type is not malformed",
			mode:         "trusted_authority_property_type",
			property:     "type",
			expectedType: "string",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"type":   "aki",
							"values": []any{"authority-key-id"},
						}}
						return credential
					}()},
				},
			},
			status: StatusFail,
		},
		{
			name:         "missing trusted authority type is not a format test",
			mode:         "trusted_authority_property_type",
			property:     "type",
			expectedType: "string",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"values": []any{"authority-key-id"},
						}}
						return credential
					}()},
				},
			},
			status: StatusFail,
		},
		{
			name:         "malformed trusted authority returning a credential fails",
			mode:         "trusted_authority_property_type",
			property:     "type",
			expectedType: "string",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"type":   true,
							"values": []any{"authority-key-id"},
						}}
						return credential
					}()},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name:         "non-array trusted authority values are rejected",
			mode:         "trusted_authority_property_type",
			property:     "values",
			expectedType: "array",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"type":   "aki",
							"values": "authority-key-id",
						}}
						return credential
					}()},
				},
			},
			status: StatusPass,
		},
		{
			name:         "array trusted authority values are not malformed",
			mode:         "trusted_authority_property_type",
			property:     "values",
			expectedType: "array",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"type":   "aki",
							"values": []any{"authority-key-id"},
						}}
						return credential
					}()},
				},
			},
			status: StatusFail,
		},
		{
			name:         "missing trusted authority values is not a format test",
			mode:         "trusted_authority_property_type",
			property:     "values",
			expectedType: "array",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"type": "aki",
						}}
						return credential
					}()},
				},
			},
			status: StatusFail,
		},
		{
			name:         "non-array values with malformed type do not isolate values",
			mode:         "trusted_authority_property_type",
			property:     "values",
			expectedType: "array",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"type":   true,
							"values": "authority-key-id",
						}}
						return credential
					}()},
				},
			},
			status: StatusFail,
		},
		{
			name: "claim sets",
			mode: "claim_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{map[string]any{
						"id":         "pid",
						"claim_sets": []any{[]any{"given_name"}},
					}},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := map[string]any{"mode": test.mode}
			if test.property != "" {
				params["property"] = test.property
			}
			if test.mode == "property_type" || test.mode == "trusted_authority_property_type" {
				params["expected_type"] = test.expectedType
				params["valid"] = test.valid
			}
			result := DCQLResponseConstraintsValidator{}.Validate(context.Background(), Input{
				Value:  test.evidence,
				Params: params,
			})
			require.Equal(t, test.status, result.Status, result.Message)
		})
	}
}

func validSDJWTCredentialQuery(id string) map[string]any {
	return map[string]any{
		"id":     id,
		"format": "dc+sd-jwt",
		"meta": map[string]any{
			"vct_values": []any{"urn:eu.europa.ec.eudi:pid:1"},
		},
		"claims": []any{
			map[string]any{"path": []any{"given_name"}},
		},
	}
}

func credentialQueryWithMultiple(id string, multiple bool) map[string]any {
	credential := validSDJWTCredentialQuery(id)
	credential["multiple"] = multiple
	return credential
}

func TestMatchesJSONType(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
		matches  bool
	}{
		{name: "boolean", value: true, expected: "boolean", matches: true},
		{name: "boolean string", value: "true", expected: "boolean", matches: false},
		{name: "string", value: "value", expected: "string", matches: true},
		{name: "number integer", value: float64(1), expected: "number", matches: true},
		{name: "number decimal", value: 1.5, expected: "number", matches: true},
		{name: "integer", value: float64(1), expected: "integer", matches: true},
		{name: "decimal is not integer", value: 1.5, expected: "integer", matches: false},
		{name: "array", value: []any{"value"}, expected: "array", matches: true},
		{name: "object", value: map[string]any{"key": "value"}, expected: "object", matches: true},
		{name: "null", value: nil, expected: "null", matches: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.matches, matchesJSONType(test.value, test.expected))
		})
	}
}
