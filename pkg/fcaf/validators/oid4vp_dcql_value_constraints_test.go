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
	matching := testSDJWT(t, map[string]any{
		"family_name":   "Rossi",
		"age_over_18":   true,
		"nationalities": []any{"IT"},
	})
	caseTrap := testSDJWT(t, map[string]any{
		"family_name":   "ROSSI",
		"age_over_18":   true,
		"nationalities": []any{"IT"},
	})
	ageTrap := testSDJWT(t, map[string]any{
		"family_name":   "Rossi",
		"age_over_18":   false,
		"nationalities": []any{"IT"},
	})
	arrayTrap := testSDJWT(t, map[string]any{
		"family_name":   "Rossi",
		"age_over_18":   true,
		"nationalities": []any{"FR", "DE"},
	})
	undisclosed := testSDJWT(t, map[string]any{"family_name": "Rossi"})

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
