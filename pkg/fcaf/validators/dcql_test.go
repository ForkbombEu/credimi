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
		name     string
		mode     string
		evidence map[string]any
		status   Status
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
				"dcql_query": map[string]any{"credentials": []any{map[string]any{"id": "unknown"}}},
				"error":      "invalid_request",
			},
			status: StatusPass,
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
			result := DCQLResponseConstraintsValidator{}.Validate(context.Background(), Input{
				Value:  test.evidence,
				Params: map[string]any{"mode": test.mode},
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
