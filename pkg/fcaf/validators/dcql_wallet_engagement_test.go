// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// guardedModeCase is one mode of dcql.response_satisfies_constraints whose pass
// can be reached without the Wallet answering the request at all.
//
// evidence is a query and response the mode accepts today. Each case is then
// replayed three times: on a session record that shows the Wallet never fetched
// the request (the pass has no evidence behind it and must be reported blocked),
// on a session where the Wallet fetched it and answered nothing (the mode's own
// verdict stands), and on a session where the Wallet posted a presentation (the
// mode's own verdict stands, which the case states because a few of these modes
// only assert the delivered query).
type guardedModeCase struct {
	mode               string
	params             map[string]any
	evidence           map[string]any
	credentialID       string
	presentationStatus Status
}

func guardedModeParams(mode string, extra map[string]any) map[string]any {
	params := map[string]any{"mode": mode}
	for key, value := range extra {
		params[key] = value
	}
	return params
}

func acmeAkiAuthority() map[string]any {
	// "AQID" is base64url for three bytes, which is all the aki authority
	// shape check requires.
	return map[string]any{"type": "aki", "values": []any{"AQID"}}
}

func guardedDCQLModeCases() []guardedModeCase {
	plainQuery := map[string]any{
		"dcql_query": map[string]any{"credentials": []any{validSDJWTCredentialQuery("pid")}},
	}
	akiQuery := func() map[string]any {
		credential := validSDJWTCredentialQuery("pid")
		credential["trusted_authorities"] = []any{acmeAkiAuthority()}
		return map[string]any{
			"dcql_query": map[string]any{"credentials": []any{credential}},
		}
	}
	authorityWith := func(authority map[string]any) map[string]any {
		credential := validSDJWTCredentialQuery("pid")
		credential["trusted_authorities"] = []any{authority}
		return map[string]any{
			"dcql_query": map[string]any{"credentials": []any{credential}},
		}
	}
	credentialWith := func(change func(map[string]any)) map[string]any {
		credential := validSDJWTCredentialQuery("pid")
		change(credential)
		return map[string]any{
			"dcql_query": map[string]any{"credentials": []any{credential}},
		}
	}
	claimlessQuery := map[string]any{
		"dcql_query": map[string]any{"credentials": []any{map[string]any{
			"id":     "pid",
			"format": "mso_mdoc",
			"meta":   map[string]any{"doctype_value": "eu.europa.ec.eudi.pid.1"},
			"claims": []any{map[string]any{
				"path": []any{"eu.europa.ec.eudi.pid.1", "birth_place"},
			}},
		}}},
	}

	return []guardedModeCase{
		{
			mode:         "no_match",
			evidence:     plainQuery,
			credentialID: "pid",
		},
		{
			mode:         "request_rejected",
			evidence:     plainQuery,
			credentialID: "pid",
		},
		{
			mode:         "trusted_authorities_no_match",
			evidence:     akiQuery(),
			credentialID: "pid",
		},
		{
			mode: "credential_sets_required_true_no_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
					"credential_sets": []any{
						map[string]any{"required": true, "options": []any{[]any{"pid"}}},
					},
				},
			},
			credentialID: "pid",
		},
		{
			mode: "credential_sets_optional_no_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
					"credential_sets": []any{
						map[string]any{"required": false, "options": []any{[]any{"pid"}}},
					},
				},
				"vp_token": map[string]any{},
			},
			credentialID: "pid",
		},
		{
			mode: "credential_sets_combined_option_no_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{
						validSDJWTCredentialQuery("pid"),
						validSDJWTCredentialQuery("pid_mdoc"),
					},
					"credential_sets": []any{map[string]any{
						"required": true,
						"options":  []any{[]any{"pid", "pid_mdoc"}},
					}},
				},
			},
			credentialID: "pid",
		},
		{
			mode:               "claims_path_no_match",
			params:             map[string]any{"expected_claim_path": []any{"given_name"}},
			evidence:           plainQuery,
			credentialID:       "pid",
			presentationStatus: StatusFail,
		},
		{
			mode: "claims_values_no_match",
			evidence: credentialWith(func(credential map[string]any) {
				credential["claims"] = []any{map[string]any{
					"path":   []any{"given_name"},
					"values": []any{"Alice"},
				}}
			}),
			credentialID: "pid",
		},
		{
			mode: "mdoc_claim_path_no_match",
			params: map[string]any{
				"expected_claim_path": []any{"eu.europa.ec.eudi.pid.1", "birth_place"},
			},
			evidence:           claimlessQuery,
			credentialID:       "pid",
			presentationStatus: StatusPass,
		},
		{
			mode: "empty_claims",
			evidence: credentialWith(func(credential map[string]any) {
				credential["claims"] = []any{}
			}),
			credentialID: "pid",
		},
		{
			mode: "empty_array",
			params: map[string]any{
				"property": "claims",
			},
			evidence: credentialWith(func(credential map[string]any) {
				credential["claims"] = []any{}
			}),
			credentialID: "pid",
		},
		{
			mode: "claim_sets_no_match",
			evidence: credentialWith(func(credential map[string]any) {
				credential["claims"] = []any{
					map[string]any{"id": "c1", "path": []any{"given_name"}},
				}
				credential["claim_sets"] = []any{[]any{"c1"}}
			}),
			credentialID: "pid",
		},
		{
			mode: "claim_sets_without_claims",
			evidence: credentialWith(func(credential map[string]any) {
				delete(credential, "claims")
				credential["claim_sets"] = []any{[]any{"c1"}}
			}),
			credentialID: "pid",
		},
		{
			mode:         "trusted_authority_property_absent",
			params:       map[string]any{"property": "values"},
			evidence:     authorityWith(map[string]any{"type": "aki"}),
			credentialID: "pid",
		},
		{
			mode: "trusted_authority_property_type",
			params: map[string]any{
				"property": "values", "expected_type": "array", "valid": false,
			},
			evidence:     authorityWith(map[string]any{"type": "aki", "values": "not-an-array"}),
			credentialID: "pid",
		},
		{
			mode: "trusted_authority_array_item_type",
			params: map[string]any{
				"property": "values", "expected_type": "array", "valid": false,
				"item_expected_type": "string", "item_valid": false,
			},
			evidence: authorityWith(map[string]any{
				"type": "aki", "values": []any{"AQID", float64(7)},
			}),
			credentialID: "pid",
		},
		{
			mode:   "trusted_authority_empty_string_item",
			params: map[string]any{"property": "values"},
			evidence: authorityWith(map[string]any{
				"type": "aki", "values": []any{""},
			}),
			credentialID: "pid",
		},
		{
			mode: "property_type",
			params: map[string]any{
				"property":      "require_cryptographic_holder_binding",
				"expected_type": "boolean",
				"valid":         false,
			},
			evidence: credentialWith(func(credential map[string]any) {
				credential["require_cryptographic_holder_binding"] = "false"
			}),
			credentialID: "pid",
		},
		// The explicit error-code modes reach their pass through
		// validateErrorCode's absent-response branch, so a Wallet that never
		// received the request passes them unless the guard speaks.
		{
			mode:         "invalid_request_generic",
			evidence:     plainQuery,
			credentialID: "pid",
		},
		{
			mode:         "invalid_scope",
			evidence:     plainQuery,
			credentialID: "pid",
		},
		{
			mode:         "access_denied",
			evidence:     plainQuery,
			credentialID: "pid",
		},
		{
			mode:         "invalid_client",
			evidence:     plainQuery,
			credentialID: "pid",
		},
		{
			mode:         "transaction_data_error",
			evidence:     plainQuery,
			credentialID: "pid",
		},
		// claim_path_member_type_error, wallet_error_expected and
		// vp_formats_not_supported belong to the guarded set too, but all three
		// are missing from the allow-list at the top of Validate, so no
		// definition can reach their pass today and there is nothing to
		// exercise here.
	}
}

// TestDCQLGuardedModesRequireAnEngagedWallet is the guard's contract for the
// dcql.response_satisfies_constraints family: every mode whose pass states what
// the Wallet did with a request must not report that pass when the Capture
// session shows the Wallet never got the request, while the same mode keeps its
// verdict on a session where the Wallet did engage.
func TestDCQLGuardedModesRequireAnEngagedWallet(t *testing.T) {
	validator := DCQLResponseConstraintsValidator{}
	validate := func(value any, params map[string]any) Result {
		return validator.Validate(context.Background(), Input{Value: value, Params: params})
	}

	cases := guardedDCQLModeCases()
	for _, test := range cases {
		t.Run(test.mode, func(t *testing.T) {
			params := guardedModeParams(test.mode, test.params)

			baseline := validate(retrievedSessionWithoutResponse(test.evidence), params)
			require.Equal(t, StatusPass, baseline.Status, baseline.Message)

			never := validate(neverRetrievedSession(test.evidence), params)
			require.Equal(t, StatusBlocked, never.Status, never.Message)
			require.Equal(t, walletNeverRetrievedMessage, never.Message)

			presented := validate(
				retrievedSessionWithPresentation(test.evidence, test.credentialID),
				params,
			)
			want := test.presentationStatus
			if want == "" {
				want = StatusFail
			}
			require.Equal(t, want, presented.Status, presented.Message)
		})
	}
}

// TestDCQLModesThatNeedAResponseAreUnguarded pins the other half of the
// inventory: a mode whose own precondition already requires a Wallet response
// reports its failure unchanged, so the guard cannot hide it.
func TestDCQLModesThatNeedAResponseAreUnguarded(t *testing.T) {
	validator := DCQLResponseConstraintsValidator{}
	evidence := map[string]any{
		"dcql_query": map[string]any{
			"credentials": []any{validSDJWTCredentialQuery("pid")},
		},
	}
	for _, mode := range []string{
		"credentials_match",
		"claims_present",
		"claims_without_values",
		"claim_path_allowed_components",
		"claim_sets",
		"mdoc_without_claim_disclosures",
		"wallet_error_required",
		"claim_path_missing",
	} {
		t.Run(mode, func(t *testing.T) {
			require.False(t, dcqlModeAssertsWalletResponse(mode))
			result := validator.Validate(context.Background(), Input{
				Value:  neverRetrievedSession(evidence),
				Params: guardedModeParams(mode, nil),
			})
			require.Equal(t, StatusFail, result.Status, result.Message)
		})
	}
}
