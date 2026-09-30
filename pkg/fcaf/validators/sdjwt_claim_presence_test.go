// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSDJWTClaimPresenceValidator(t *testing.T) {
	withStatus := map[string]any{"pid": []any{testSDJWT(t, testPIDVCT, map[string]any{
		"status": map[string]any{"status_list": map[string]any{"idx": 1}},
	})}}
	withoutStatus := map[string]any{"pid": []any{testSDJWT(t, testPIDVCT, map[string]any{
		"given_name": "Mario",
	})}}

	tests := []struct {
		name       string
		value      any
		claim      string
		present    bool
		wantStatus Status
	}{
		{
			name:       "required claim is present",
			value:      withStatus,
			claim:      "status",
			present:    true,
			wantStatus: StatusPass,
		},
		{
			name:       "forbidden claim is absent",
			value:      withoutStatus,
			claim:      "status",
			present:    false,
			wantStatus: StatusPass,
		},
		{
			name:       "required claim is missing",
			value:      withoutStatus,
			claim:      "status",
			present:    true,
			wantStatus: StatusFail,
		},
		{
			name:       "forbidden claim is disclosed",
			value:      withStatus,
			claim:      "status",
			present:    false,
			wantStatus: StatusFail,
		},
		{
			name:       "nested path is resolved",
			value:      withStatus,
			claim:      "status.status_list.idx",
			present:    true,
			wantStatus: StatusPass,
		},
		{
			name:       "evidence carries no presentation",
			value:      "not-a-presentation",
			claim:      "status",
			present:    false,
			wantStatus: StatusFail,
		},
	}

	validator := SDJWTClaimPresenceValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: map[string]any{"claim": test.claim, "present": test.present},
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}

// TestClaimsValuesNoMatchPinsTheConstrainedValue covers the optional value
// pinning that binds a no-match outcome to one named credential fixture
// instead of to any unsatisfiable query.
func TestClaimsValuesNoMatchPinsTheConstrainedValue(t *testing.T) {
	exchange := func(documentNumber string, presented bool) map[string]any {
		record := map[string]any{
			"authorization_request": map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id":     "pid",
					"format": "dc+sd-jwt",
					"meta":   map[string]any{"vct_values": []any{testPIDVCT}},
					"claims": []any{map[string]any{
						"path":   []any{"document_number"},
						"values": []any{documentNumber},
					}},
				}}},
			},
		}
		if presented {
			record["observed"] = map[string]any{"wallet_response": map[string]any{
				"value": map[string]any{"vp_token": map[string]any{
					"pid": []any{testSDJWT(t, testPIDVCT, map[string]any{
						"document_number": documentNumber,
					})},
				}},
			}}
		}
		return record
	}

	tests := []struct {
		name       string
		value      any
		params     map[string]any
		wantStatus Status
	}{
		{
			name:  "nothing returned for the pinned value",
			value: exchange("CREDIMI-DEMO-CASE", false),
			params: map[string]any{
				"mode":                "claims_values_no_match",
				"expected_claim_path": []any{"document_number"},
				"expected_value":      "CREDIMI-DEMO-CASE",
			},
			wantStatus: StatusPass,
		},
		{
			name:  "a credential was returned for the pinned value",
			value: exchange("CREDIMI-DEMO-CASE", true),
			params: map[string]any{
				"mode":                "claims_values_no_match",
				"expected_claim_path": []any{"document_number"},
				"expected_value":      "CREDIMI-DEMO-CASE",
			},
			wantStatus: StatusFail,
		},
		{
			name:  "the query constrained a different value",
			value: exchange("CREDIMI-DEMO-OTHER", false),
			params: map[string]any{
				"mode":                "claims_values_no_match",
				"expected_claim_path": []any{"document_number"},
				"expected_value":      "CREDIMI-DEMO-CASE",
			},
			wantStatus: StatusFail,
		},
		{
			name:  "the query never constrained the pinned path",
			value: exchange("CREDIMI-DEMO-CASE", false),
			params: map[string]any{
				"mode":                "claims_values_no_match",
				"expected_claim_path": []any{"family_name"},
				"expected_value":      "Rossi",
			},
			wantStatus: StatusFail,
		},
		{
			name:       "pinning is optional",
			value:      exchange("CREDIMI-DEMO-CASE", false),
			params:     map[string]any{"mode": "claims_values_no_match"},
			wantStatus: StatusPass,
		},
	}

	validator := DCQLResponseConstraintsValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: test.params,
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
		})
	}
}
