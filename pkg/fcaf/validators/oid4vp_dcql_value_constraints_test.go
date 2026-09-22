// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// constrainedPIDExchange builds a captured exchange for a query restricting
// family_name, age_over_18 and a nationalities array element at once.
func constrainedPIDExchange(multiple bool, tokens ...string) map[string]any {
	credential := map[string]any{
		"id":     "pid",
		"format": "dc+sd-jwt",
		"meta":   map[string]any{"vct_values": []any{testPIDVCT}},
		"claims": []any{
			map[string]any{"path": []any{"family_name"}, "values": []any{"Rossi"}},
			map[string]any{"path": []any{"age_over_18"}, "values": []any{true}},
			map[string]any{"path": []any{"nationalities", 0}, "values": []any{"IT"}},
		},
	}
	if multiple {
		credential["multiple"] = true
	}
	entries := make([]any, 0, len(tokens))
	for _, token := range tokens {
		entries = append(entries, token)
	}
	return map[string]any{
		"authorization_request": map[string]any{
			"dcql_query": map[string]any{"credentials": []any{credential}},
		},
		"observed": map[string]any{"wallet_response": map[string]any{
			"value": map[string]any{"vp_token": map[string]any{"pid": entries}},
		}},
	}
}

func TestOID4VPDCQLValueConstraintsSatisfiedValidator(t *testing.T) {
	matching := testSDJWT(t, testPIDVCT, map[string]any{
		"family_name":   "Rossi",
		"age_over_18":   true,
		"nationalities": []any{"IT"},
	})
	caseTrap := testSDJWT(t, testPIDVCT, map[string]any{
		"family_name":   "ROSSI",
		"age_over_18":   true,
		"nationalities": []any{"IT"},
	})
	ageTrap := testSDJWT(t, testPIDVCT, map[string]any{
		"family_name":   "Rossi",
		"age_over_18":   false,
		"nationalities": []any{"IT"},
	})
	arrayTrap := testSDJWT(t, testPIDVCT, map[string]any{
		"family_name":   "Rossi",
		"age_over_18":   true,
		"nationalities": []any{"FR", "DE"},
	})
	undisclosed := testSDJWT(t, testPIDVCT, map[string]any{"family_name": "Rossi"})

	tests := []struct {
		name       string
		value      any
		params     map[string]any
		wantStatus Status
	}{
		{
			name:       "only the matching credential is released",
			value:      constrainedPIDExchange(true, matching),
			wantStatus: StatusPass,
		},
		{
			name:       "a credential failing the string constraint is released",
			value:      constrainedPIDExchange(true, matching, caseTrap),
			wantStatus: StatusFail,
		},
		{
			name:       "a credential failing the boolean constraint is released",
			value:      constrainedPIDExchange(true, matching, ageTrap),
			wantStatus: StatusFail,
		},
		{
			name:       "a credential failing the array-element constraint is released",
			value:      constrainedPIDExchange(true, matching, arrayTrap),
			wantStatus: StatusFail,
		},
		{
			name:       "a restricted claim is not disclosed at all",
			value:      constrainedPIDExchange(true, undisclosed),
			wantStatus: StatusFail,
		},
		{
			name:       "nothing was released",
			value:      constrainedPIDExchange(true),
			wantStatus: StatusFail,
		},
		{
			name:  "the query restricts fewer claims than required",
			value: constrainedPIDExchange(true, matching),
			params: map[string]any{
				"vct":                 testPIDVCT,
				"minimum_constraints": 5,
			},
			wantStatus: StatusFail,
		},
		{
			name:  "multiple is required but absent",
			value: constrainedPIDExchange(false, matching),
			params: map[string]any{
				"vct":              testPIDVCT,
				"require_multiple": true,
			},
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPDCQLValueConstraintsSatisfiedValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := test.params
			if params == nil {
				params = map[string]any{
					"vct":                 testPIDVCT,
					"minimum_constraints": 3,
					"require_multiple":    true,
				}
			}
			result := validator.Validate(
				context.Background(),
				Input{Value: test.value, Params: params},
			)
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}

func TestOID4VPMalformedStatusCredentialAbsentValidator(t *testing.T) {
	statusToken := func(status any) string {
		claims := map[string]any{"document_number": "CREDIMI-DEMO-CASE"}
		if status != nil {
			claims["status"] = status
		}
		return testSDJWT(t, testPIDVCT, claims)
	}
	valid := statusToken(map[string]any{"status_list": map[string]any{
		"uri": "https://status.example/list/1", "idx": 42,
	}})
	noStatusList := statusToken(map[string]any{})
	negativeIndex := statusToken(map[string]any{"status_list": map[string]any{
		"uri": "https://status.example/list/1", "idx": -1,
	}})
	relativeURI := statusToken(map[string]any{"status_list": map[string]any{
		"uri": "status.example/list/1", "idx": 42,
	}})

	exchange := func(multiple bool, documentNumber string, tokens ...string) map[string]any {
		credential := map[string]any{
			"id":     "pid",
			"format": "dc+sd-jwt",
			"meta":   map[string]any{"vct_values": []any{testPIDVCT}},
			"claims": []any{map[string]any{
				"path":   []any{"document_number"},
				"values": []any{documentNumber},
			}},
		}
		if multiple {
			credential["multiple"] = true
		}
		entries := make([]any, 0, len(tokens))
		for _, token := range tokens {
			entries = append(entries, token)
		}
		return map[string]any{
			"authorization_request": map[string]any{
				"dcql_query": map[string]any{"credentials": []any{credential}},
			},
			"observed": map[string]any{"wallet_response": map[string]any{
				"value": map[string]any{"vp_token": map[string]any{"pid": entries}},
			}},
		}
	}

	tests := []struct {
		name       string
		value      any
		shape      string
		wantStatus Status
	}{
		{
			name:       "wallet holds nothing",
			value:      exchange(true, "CREDIMI-DEMO-CASE"),
			shape:      "missing_status_list",
			wantStatus: StatusPass,
		},
		{
			name:       "wallet holds only a validly issued duplicate",
			value:      exchange(true, "CREDIMI-DEMO-CASE", valid),
			shape:      "missing_status_list",
			wantStatus: StatusPass,
		},
		{
			name:       "wallet retained the token without a status_list",
			value:      exchange(true, "CREDIMI-DEMO-CASE", noStatusList),
			shape:      "missing_status_list",
			wantStatus: StatusFail,
		},
		{
			name:       "wallet retained the malformed one beside the valid duplicate",
			value:      exchange(true, "CREDIMI-DEMO-CASE", valid, noStatusList),
			shape:      "missing_status_list",
			wantStatus: StatusFail,
		},
		{
			name:       "negative index is detected",
			value:      exchange(true, "CREDIMI-DEMO-CASE", negativeIndex),
			shape:      "negative_index",
			wantStatus: StatusFail,
		},
		{
			name:       "relative status list uri is detected",
			value:      exchange(true, "CREDIMI-DEMO-CASE", relativeURI),
			shape:      "malformed_uri",
			wantStatus: StatusFail,
		},
		{
			name:       "a valid reference is not mistaken for a negative index",
			value:      exchange(true, "CREDIMI-DEMO-CASE", valid),
			shape:      "negative_index",
			wantStatus: StatusPass,
		},
		{
			name:       "the probe did not ask for every match",
			value:      exchange(false, "CREDIMI-DEMO-CASE"),
			shape:      "missing_status_list",
			wantStatus: StatusFail,
		},
		{
			name:       "the probe pinned a different credential",
			value:      exchange(true, "CREDIMI-DEMO-OTHER"),
			shape:      "missing_status_list",
			wantStatus: StatusFail,
		},
	}

	validator := OID4VPMalformedStatusCredentialAbsentValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value: test.value,
				Params: map[string]any{
					"vct":   testPIDVCT,
					"claim": "document_number",
					"value": "CREDIMI-DEMO-CASE",
					"shape": test.shape,
				},
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
