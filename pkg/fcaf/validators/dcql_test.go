// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/require"
)

func TestDCQLResponseConstraintsValidator(t *testing.T) {
	tests := []struct {
		name           string
		mode           string
		property       string
		expectedType   string
		expectedValue  any
		expectedFormat string
		valid          bool
		evidence       map[string]any
		status         Status
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
			name: "credential sets options missing is rejected",
			mode: "credential_sets_options_missing",
			evidence: map[string]any{
				"authorization_request": map[string]any{
					"dcql_query": map[string]any{
						"credentials":     []any{validSDJWTCredentialQuery("pid")},
						"credential_sets": []any{map[string]any{"required": true}},
					},
				},
				"observed": map[string]any{
					"wallet_response": map[string]any{
						"value": map[string]any{"error": "invalid_request"},
					},
				},
			},
			status: StatusPass,
		},
		{
			name: "credential sets options empty is rejected",
			mode: "credential_sets_options_empty",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials":     []any{validSDJWTCredentialQuery("pid")},
					"credential_sets": []any{map[string]any{"options": []any{}}},
				},
				"error": "invalid_request",
			},
			status: StatusPass,
		},
		{
			name: "credential sets options non array is rejected",
			mode: "credential_sets_options_non_array",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials":     []any{validSDJWTCredentialQuery("pid")},
					"credential_sets": []any{map[string]any{"options": "pid"}},
				},
				"error": "invalid_request",
			},
			status: StatusPass,
		},
		{
			name: "credential sets options valid references",
			mode: "credential_sets_options_valid_references",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials":     []any{validSDJWTCredentialQuery("pid")},
					"credential_sets": []any{map[string]any{"options": []any{[]any{"pid"}}}},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "credential sets options invalid references are rejected",
			mode: "credential_sets_options_invalid_references",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials":     []any{validSDJWTCredentialQuery("pid")},
					"credential_sets": []any{map[string]any{"options": []any{[]any{"unknown"}}}},
				},
			},
			status: StatusPass,
		},
		{
			name: "required true match",
			mode: "credential_sets_required_true_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credential_sets": []any{map[string]any{"required": true}},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "required true no match",
			mode: "credential_sets_required_true_no_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credential_sets": []any{map[string]any{"required": true}},
				},
			},
			status: StatusPass,
		},
		{
			name: "required omitted",
			mode: "credential_sets_required_omitted",
			evidence: map[string]any{
				"dcql_query": map[string]any{"credential_sets": []any{map[string]any{}}},
				"vp_token":   map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "required false with match",
			mode: "credential_sets_required_false_with_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credential_sets": []any{map[string]any{"required": false}},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "required credential set with unavailable optional set",
			mode: "credential_sets_required_optional",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{
						validSDJWTCredentialQuery("available"),
						validSDJWTCredentialQuery("unavailable"),
					},
					"credential_sets": []any{
						map[string]any{"required": true, "options": []any{[]any{"available"}}},
						map[string]any{"required": false, "options": []any{[]any{"unavailable"}}},
					},
				},
				"vp_token": map[string]any{"available": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "single available credential set option",
			mode: "credential_sets_single_available_option",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{
						validSDJWTCredentialQuery("available"),
						validSDJWTCredentialQuery("unavailable"),
					},
					"credential_sets": []any{map[string]any{"options": []any{[]any{"available"}}}},
				},
				"vp_token": map[string]any{"available": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "combined unavailable credential set option is not presented",
			mode: "credential_sets_combined_option_no_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{
						validSDJWTCredentialQuery("available"),
						validSDJWTCredentialQuery("unavailable"),
					},
					"credential_sets": []any{
						map[string]any{"options": []any{[]any{"available", "unavailable"}}},
					},
				},
			},
			status: StatusPass,
		},
		{
			name: "required combined credential set returns a privacy-preserving error",
			mode: "required_credentials_no_partial_presentation",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{
						validSDJWTCredentialQuery("available"),
						validSDJWTCredentialQuery("unavailable"),
					},
					"credential_sets": []any{
						map[string]any{
							"required": true,
							"options":  []any{[]any{"available", "unavailable"}},
						},
					},
				},
				"error": "access_denied",
			},
			status: StatusPass,
		},
		{
			name: "required combined credential set rejects partial presentation",
			mode: "required_credentials_no_partial_presentation",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{
						validSDJWTCredentialQuery("available"),
						validSDJWTCredentialQuery("unavailable"),
					},
					"credential_sets": []any{
						map[string]any{
							"required": true,
							"options":  []any{[]any{"available", "unavailable"}},
						},
					},
				},
				"vp_token": map[string]any{"available": []any{"presentation"}},
			},
			status: StatusFail,
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
			name: "claims are present and matched",
			mode: "claims_present",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "claims are required",
			mode: "claims_present",
			evidence: map[string]any{
				"dcql_query": map[string]any{"credentials": []any{func() map[string]any {
					credential := validSDJWTCredentialQuery("pid")
					delete(credential, "claims")
					return credential
				}()}},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "unmatched claim path returns no credential",
			mode: "claims_path_no_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{"credentials": []any{func() map[string]any {
					credential := validSDJWTCredentialQuery("pid")
					credential["claims"] = []any{
						map[string]any{"path": []any{"claim_that_does_not_exist"}},
					}
					return credential
				}()}},
			},
			status: StatusPass,
		},
		{
			name: "mismatched claim values return no credential",
			mode: "claims_values_no_match",
			evidence: map[string]any{
				"dcql_query": map[string]any{"credentials": []any{func() map[string]any {
					credential := validSDJWTCredentialQuery("pid")
					credential["claims"] = []any{
						map[string]any{
							"path":   []any{"given_name"},
							"values": []any{"value-that-does-not-match"},
						},
					}
					return credential
				}()}},
			},
			status: StatusPass,
		},
		{
			name: "missing claim id with claim sets is rejected",
			mode: "claim_id_missing_with_claim_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{"credentials": []any{
					map[string]any{
						"claims": []any{
							map[string]any{"path": []any{"given_name"}},
						},
						"claim_sets": []any{[]any{"missing_id"}},
					},
				}},
			},
			status: StatusPass,
		},
		{
			name: "claims without ids and claim sets are accepted",
			mode: "claims_without_id_without_claim_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusPass,
		},
		{
			name: "claim id is not the requested shape",
			mode: "claims_without_id_without_claim_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{"credentials": []any{func() map[string]any {
					credential := validSDJWTCredentialQuery("pid")
					credential["claims"] = []any{
						map[string]any{"id": "given_name", "path": []any{"given_name"}},
					}
					return credential
				}()}},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "claim sets must be absent",
			mode: "claims_without_id_without_claim_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{"credentials": []any{func() map[string]any {
					credential := validSDJWTCredentialQuery("pid")
					credential["claim_sets"] = []any{[]any{"given_name"}}
					return credential
				}()}},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
		},
		{
			name: "accepted request must return a presentation",
			mode: "claims_without_id_without_claim_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
				},
			},
			status: StatusFail,
		},
		{
			name: "duplicate claim ids are rejected",
			mode: "duplicate_claim_ids",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{credentialQueryWithClaimIDs("pid", "name", "name")},
				},
				"error": "invalid_request",
			},
			status: StatusPass,
		},
		{
			name: "unrelated singleton claims array remains valid evidence",
			mode: "duplicate_claim_ids",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{
						credentialQueryWithClaimIDs("pid-a", "name", "name"),
						credentialQueryWithClaimIDs("pid-b", "birth_date"),
					},
				},
				"error": "invalid_request",
			},
			status: StatusPass,
		},
		{
			name: "claim ids are scoped to one claims array",
			mode: "duplicate_claim_ids",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{
						credentialQueryWithClaimIDs("pid-a", "name", "family_name"),
						credentialQueryWithClaimIDs("pid-b", "name", "birth_date"),
					},
				},
				"error": "invalid_request",
			},
			status: StatusFail,
		},
		{
			name: "unique claim ids are not the malformed case",
			mode: "duplicate_claim_ids",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{credentialQueryWithClaimIDs("pid", "name", "family_name")},
				},
				"error": "invalid_request",
			},
			status: StatusFail,
		},
		{
			name: "duplicate claim ids require invalid request",
			mode: "duplicate_claim_ids",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{credentialQueryWithClaimIDs("pid", "name", "name")},
				},
			},
			status: StatusFail,
		},
		{
			name: "empty claim id is rejected",
			mode: "empty_claim_id",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{credentialQueryWithClaimIDs("pid", "")},
				},
				"error": "invalid_request",
			},
			status: StatusPass,
		},
		{
			name: "missing claim id is not the empty id case",
			mode: "empty_claim_id",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{validSDJWTCredentialQuery("pid")},
				},
				"error": "invalid_request",
			},
			status: StatusFail,
		},
		{
			name: "non-empty claim id is not malformed",
			mode: "empty_claim_id",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{credentialQueryWithClaimIDs("pid", "name")},
				},
				"error": "invalid_request",
			},
			status: StatusFail,
		},
		{
			name: "empty claim id requires invalid request",
			mode: "empty_claim_id",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{credentialQueryWithClaimIDs("pid", "")},
				},
			},
			status: StatusFail,
		},
		{
			name:     "claim id containing dot is rejected",
			mode:     "invalid_claim_id_characters",
			evidence: malformedClaimIDEvidence("given.name"),
			status:   StatusPass,
		},
		{
			name:     "claim id containing space is rejected",
			mode:     "invalid_claim_id_characters",
			evidence: malformedClaimIDEvidence("given name"),
			status:   StatusPass,
		},
		{
			name:     "claim id containing colon is rejected",
			mode:     "invalid_claim_id_characters",
			evidence: malformedClaimIDEvidence("given:name"),
			status:   StatusPass,
		},
		{
			name:     "claim id containing slash is rejected",
			mode:     "invalid_claim_id_characters",
			evidence: malformedClaimIDEvidence("given/name"),
			status:   StatusPass,
		},
		{
			name:     "claim id containing non ASCII is rejected",
			mode:     "invalid_claim_id_characters",
			evidence: malformedClaimIDEvidence("given_\u00e9"),
			status:   StatusPass,
		},
		{
			name:     "alphanumeric underscore and hyphen claim id is valid",
			mode:     "invalid_claim_id_characters",
			evidence: malformedClaimIDEvidence("Name_01-test"),
			status:   StatusFail,
		},
		{
			name:     "empty claim id is not the invalid character case",
			mode:     "invalid_claim_id_characters",
			evidence: malformedClaimIDEvidence(""),
			status:   StatusFail,
		},
		{
			name: "malformed claim id requires invalid request",
			mode: "invalid_claim_id_characters",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{credentialQueryWithClaimIDs("pid", "given.name")},
				},
			},
			status: StatusFail,
		},
		{
			name:     "missing claim path is rejected",
			mode:     "claim_path_missing",
			evidence: claimPathEvidence(false, nil, true),
			status:   StatusPass,
		},
		{
			name:     "null claim path is present",
			mode:     "claim_path_missing",
			evidence: claimPathEvidence(true, nil, true),
			status:   StatusFail,
		},
		{
			name:     "empty claim path is present",
			mode:     "claim_path_missing",
			evidence: claimPathEvidence(true, []any{}, true),
			status:   StatusFail,
		},
		{
			name:     "valid claim path is present",
			mode:     "claim_path_missing",
			evidence: claimPathEvidence(true, []any{"given_name"}, true),
			status:   StatusFail,
		},
		{
			name:     "missing claim path requires invalid request",
			mode:     "claim_path_missing",
			evidence: claimPathEvidence(false, nil, false),
			status:   StatusFail,
		},
		{
			name:     "empty claim path is rejected",
			mode:     "claim_path_empty",
			evidence: claimPathEvidence(true, []any{}, true),
			status:   StatusPass,
		},
		{
			name:     "missing claim path is not the empty path case",
			mode:     "claim_path_empty",
			evidence: claimPathEvidence(false, nil, true),
			status:   StatusFail,
		},
		{
			name:     "null claim path is not the empty array case",
			mode:     "claim_path_empty",
			evidence: claimPathEvidence(true, nil, true),
			status:   StatusFail,
		},
		{
			name:     "valid claim path is not empty",
			mode:     "claim_path_empty",
			evidence: claimPathEvidence(true, []any{"given_name"}, true),
			status:   StatusFail,
		},
		{
			name:     "empty claim path requires invalid request",
			mode:     "claim_path_empty",
			evidence: claimPathEvidence(true, []any{}, false),
			status:   StatusFail,
		},
		{
			name:     "null claim path is rejected as non-array",
			mode:     "claim_path_non_array",
			evidence: claimPathEvidence(true, nil, true),
			status:   StatusPass,
		},
		{
			name:     "true claim path is rejected as non-array",
			mode:     "claim_path_non_array",
			evidence: claimPathEvidence(true, true, true),
			status:   StatusPass,
		},
		{
			name:     "false claim path is rejected as non-array",
			mode:     "claim_path_non_array",
			evidence: claimPathEvidence(true, false, true),
			status:   StatusPass,
		},
		{
			name:     "zero claim path is rejected as non-array",
			mode:     "claim_path_non_array",
			evidence: claimPathEvidence(true, float64(0), true),
			status:   StatusPass,
		},
		{
			name:     "number claim path is rejected as non-array",
			mode:     "claim_path_non_array",
			evidence: claimPathEvidence(true, float64(73), true),
			status:   StatusPass,
		},
		{
			name:     "string claim path is rejected as non-array",
			mode:     "claim_path_non_array",
			evidence: claimPathEvidence(true, "given_name", true),
			status:   StatusPass,
		},
		{
			name:     "object claim path is rejected as non-array",
			mode:     "claim_path_non_array",
			evidence: claimPathEvidence(true, map[string]any{"claim": "given_name"}, true),
			status:   StatusPass,
		},
		{
			name:     "missing claim path is not the non-array case",
			mode:     "claim_path_non_array",
			evidence: claimPathEvidence(false, nil, true),
			status:   StatusFail,
		},
		{
			name:     "empty array claim path is still an array",
			mode:     "claim_path_non_array",
			evidence: claimPathEvidence(true, []any{}, true),
			status:   StatusFail,
		},
		{
			name:     "valid claim path is an array",
			mode:     "claim_path_non_array",
			evidence: claimPathEvidence(true, []any{"given_name"}, true),
			status:   StatusFail,
		},
		{
			name:     "non-array claim path requires invalid request",
			mode:     "claim_path_non_array",
			evidence: claimPathEvidence(true, "given_name", false),
			status:   StatusFail,
		},
		{
			name: "allowed claim path components resolve",
			mode: "claim_path_allowed_components",
			evidence: allowedClaimPathComponentsEvidence(
				[]any{"given_name"},
				[]any{"nationality", nil},
				[]any{"nationality", float64(0)},
				true,
			),
			status: StatusPass,
		},
		{
			name: "allowed claim path components require every path to resolve",
			mode: "claim_path_allowed_components",
			evidence: allowedClaimPathComponentsWithClaims(
				[]any{"given_name"},
				[]any{"nationality", nil},
				[]any{"nationality", float64(0)},
				map[string]any{"given_name": "Filippo"},
			),
			status: StatusFail,
		},
		{
			name:     "allowed claim path components require string",
			mode:     "claim_path_allowed_components",
			evidence: allowedClaimPathComponentsEvidence([]any{nil}, []any{float64(0)}, nil, true),
			status:   StatusFail,
		},
		{
			name: "allowed claim path components require null",
			mode: "claim_path_allowed_components",
			evidence: allowedClaimPathComponentsEvidence(
				[]any{"given_name"},
				[]any{"nationality", float64(0)},
				nil,
				true,
			),
			status: StatusFail,
		},
		{
			name: "allowed claim path components require integer",
			mode: "claim_path_allowed_components",
			evidence: allowedClaimPathComponentsEvidence(
				[]any{"given_name"},
				[]any{"nationality", nil},
				nil,
				true,
			),
			status: StatusFail,
		},
		{
			name: "allowed claim path rejects empty path",
			mode: "claim_path_allowed_components",
			evidence: allowedClaimPathComponentsEvidence(
				[]any{},
				[]any{"nationality", nil},
				[]any{"nationality", float64(0)},
				true,
			),
			status: StatusFail,
		},
		{
			name: "allowed claim path rejects boolean",
			mode: "claim_path_allowed_components",
			evidence: allowedClaimPathComponentsEvidence(
				[]any{"given_name"},
				[]any{"nationality", nil},
				[]any{"nationality", true},
				true,
			),
			status: StatusFail,
		},
		{
			name: "allowed claim path rejects negative integer",
			mode: "claim_path_allowed_components",
			evidence: allowedClaimPathComponentsEvidence(
				[]any{"given_name"},
				[]any{"nationality", nil},
				[]any{"nationality", float64(-1)},
				true,
			),
			status: StatusFail,
		},
		{
			name: "allowed claim path rejects fractional number",
			mode: "claim_path_allowed_components",
			evidence: allowedClaimPathComponentsEvidence(
				[]any{"given_name"},
				[]any{"nationality", nil},
				[]any{"nationality", 1.5},
				true,
			),
			status: StatusFail,
		},
		{
			name: "allowed claim path requires presentation",
			mode: "claim_path_allowed_components",
			evidence: allowedClaimPathComponentsEvidence(
				[]any{"given_name"},
				[]any{"nationality", nil},
				[]any{"nationality", float64(0)},
				false,
			),
			status: StatusFail,
		},
		{
			name:     "claim without values is accepted",
			mode:     "claims_without_values",
			evidence: claimWithoutValuesEvidence(false, true),
			status:   StatusPass,
		},
		{
			name:     "claim with values is not the omitted case",
			mode:     "claims_without_values",
			evidence: claimWithoutValuesEvidence(true, true),
			status:   StatusFail,
		},
		{
			name:     "claim without values requires a presentation",
			mode:     "claims_without_values",
			evidence: claimWithoutValuesEvidence(false, false),
			status:   StatusFail,
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
			name: "all credential queries matched without credential sets",
			mode: "without_credential_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{
						validSDJWTCredentialQuery("pid-given-name"),
						validSDJWTCredentialQuery("pid-given-name-copy"),
					},
				},
				"vp_token": map[string]any{
					"pid-given-name":      []any{"given-name-presentation"},
					"pid-given-name-copy": []any{"given-name-copy-presentation"},
				},
			},
			status: StatusPass,
		},
		{
			name: "missing one credential query presentation without credential sets",
			mode: "without_credential_sets",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{
						validSDJWTCredentialQuery("pid-given-name"),
						validSDJWTCredentialQuery("pid-given-name-copy"),
					},
				},
				"vp_token": map[string]any{"pid-given-name": []any{"given-name-presentation"}},
			},
			status: StatusFail,
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
			name:          "holder binding must be true",
			mode:          "property_equals",
			property:      "require_cryptographic_holder_binding",
			expectedValue: true,
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
			name:          "holder binding false does not satisfy true requirement",
			mode:          "property_equals",
			property:      "require_cryptographic_holder_binding",
			expectedValue: true,
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["require_cryptographic_holder_binding"] = false
						return credential
					}()},
				},
				"vp_token": map[string]any{"pid": []any{"presentation"}},
			},
			status: StatusFail,
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
			name:         "non-string trusted authority value item is rejected",
			mode:         "trusted_authority_array_item_type",
			property:     "values",
			expectedType: "array",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"type":   "aki",
							"values": []any{true},
						}}
						return credential
					}()},
				},
			},
			status: StatusPass,
		},
		{
			name:         "all-string trusted authority value items are not malformed",
			mode:         "trusted_authority_array_item_type",
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
			name:         "mixed trusted authority value items expose the invalid item",
			mode:         "trusted_authority_array_item_type",
			property:     "values",
			expectedType: "array",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"type":   "aki",
							"values": []any{"authority-key-id", 1},
						}}
						return credential
					}()},
				},
			},
			status: StatusPass,
		},
		{
			name:     "empty trusted authority value item is rejected",
			mode:     "trusted_authority_empty_string_item",
			property: "values",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"type":   "aki",
							"values": []any{""},
						}}
						return credential
					}()},
				},
			},
			status: StatusPass,
		},
		{
			name:     "non-empty trusted authority value items are not malformed",
			mode:     "trusted_authority_empty_string_item",
			property: "values",
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
			name:     "non-string value item is not the empty-string case",
			mode:     "trusted_authority_empty_string_item",
			property: "values",
			evidence: map[string]any{
				"dcql_query": map[string]any{
					"credentials": []any{func() map[string]any {
						credential := validSDJWTCredentialQuery("pid")
						credential["trusted_authorities"] = []any{map[string]any{
							"type":   "aki",
							"values": []any{true},
						}}
						return credential
					}()},
				},
			},
			status: StatusFail,
		},
		{
			name:           "mso mdoc format presentation",
			mode:           "credential_format_presentation",
			expectedFormat: "mso_mdoc",
			evidence: map[string]any{
				"authorization_request": map[string]any{
					"dcql_query": map[string]any{
						"credentials": []any{map[string]any{
							"id":     "pid_mdoc",
							"format": "mso_mdoc",
							"meta":   map[string]any{"doctype_value": "eu.europa.ec.eudi.pid.1"},
						}},
					},
				},
				"observed": map[string]any{
					"wallet_response": map[string]any{
						"value": map[string]any{
							"vp_token": map[string]any{"pid_mdoc": []any{"device-response"}},
						},
					},
				},
			},
			status: StatusPass,
		},
		{
			name:           "dc sd jwt format presentation",
			mode:           "credential_format_presentation",
			expectedFormat: "dc+sd-jwt",
			evidence: map[string]any{
				"authorization_request": map[string]any{
					"dcql_query": map[string]any{
						"credentials": []any{map[string]any{
							"id":     "pid_sdjwt",
							"format": "dc+sd-jwt",
							"meta":   map[string]any{"vct_values": []any{"urn:eudi:pid:1"}},
						}},
					},
				},
				"observed": map[string]any{
					"wallet_response": map[string]any{
						"value": map[string]any{
							"vp_token": map[string]any{"pid_sdjwt": []any{"sd-jwt-presentation"}},
						},
					},
				},
			},
			status: StatusPass,
		},
		{
			name:           "format presentation rejects opposite format",
			mode:           "credential_format_presentation",
			expectedFormat: "mso_mdoc",
			evidence:       credentialFormatEvidence("dc+sd-jwt", "pid"),
			status:         StatusFail,
		},
		{
			name:           "format presentation rejects missing format",
			mode:           "credential_format_presentation",
			expectedFormat: "mso_mdoc",
			evidence:       credentialFormatEvidence("", "pid"),
			status:         StatusFail,
		},
		{
			name:           "format presentation rejects missing vp token",
			mode:           "credential_format_presentation",
			expectedFormat: "mso_mdoc",
			evidence:       credentialFormatEvidence("mso_mdoc", ""),
			status:         StatusFail,
		},
		{
			name:           "format presentation rejects wrong response query id",
			mode:           "credential_format_presentation",
			expectedFormat: "mso_mdoc",
			evidence:       credentialFormatEvidence("mso_mdoc", "other"),
			status:         StatusFail,
		},
		{
			name:           "format presentation rejects empty presentation array",
			mode:           "credential_format_presentation",
			expectedFormat: "mso_mdoc",
			evidence: credentialFormatEvidenceWithPresentations(
				"pid",
				"mso_mdoc",
				"pid",
				[]any{},
			),
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
			if test.mode == "property_equals" {
				params["expected_value"] = test.expectedValue
			}
			if test.expectedFormat != "" {
				params["expected_format"] = test.expectedFormat
			}
			if test.mode == "trusted_authority_array_item_type" {
				params["expected_type"] = test.expectedType
				params["valid"] = test.valid
				params["item_expected_type"] = "string"
				params["item_valid"] = false
			}
			result := DCQLResponseConstraintsValidator{}.Validate(context.Background(), Input{
				Value:  test.evidence,
				Params: params,
			})
			require.Equal(t, test.status, result.Status, result.Message)
		})
	}
}

func TestDCQLVPTokenResponseModes(t *testing.T) {
	baseEvidence := func() map[string]any {
		return map[string]any{
			"response_type": "vp_token",
			"dcql_query": map[string]any{
				"credentials": []any{validSDJWTCredentialQuery("pid")},
			},
			"vp_token": map[string]any{
				"pid": []any{signedSDJWTPresentation()},
			},
		}
	}

	tests := []struct {
		name   string
		mode   string
		mutate func(map[string]any)
		status Status
	}{
		{
			name:   "signed presentation accepts vp token response type and signed sd jwt",
			mode:   "vp_token_signed_presentation",
			status: StatusPass,
		},
		{
			name: "signed presentation rejects another response type",
			mode: "vp_token_signed_presentation",
			mutate: func(evidence map[string]any) {
				evidence["response_type"] = "code"
			},
			status: StatusFail,
		},
		{
			name: "signed presentation rejects unsigned jwt",
			mode: "vp_token_signed_presentation",
			mutate: func(evidence map[string]any) {
				evidence["vp_token"] = map[string]any{"pid": []any{unsignedSDJWTPresentation()}}
			},
			status: StatusFail,
		},
		{
			name:   "json object accepts vp token object",
			mode:   "vp_token_json_object",
			status: StatusPass,
		},
		{
			name: "json object rejects array",
			mode: "vp_token_json_object",
			mutate: func(evidence map[string]any) {
				evidence["vp_token"] = []any{"presentation"}
			},
			status: StatusFail,
		},
		{
			name:   "query ids accept exact keys",
			mode:   "vp_token_query_ids",
			status: StatusPass,
		},
		{
			name: "query ids reject additional key",
			mode: "vp_token_query_ids",
			mutate: func(evidence map[string]any) {
				evidence["vp_token"] = map[string]any{
					"pid":   []any{"presentation"},
					"other": []any{"presentation"},
				}
			},
			status: StatusFail,
		},
		{
			name:   "presentation arrays accept non empty array",
			mode:   "vp_token_presentation_arrays",
			status: StatusPass,
		},
		{
			name: "presentation arrays reject scalar",
			mode: "vp_token_presentation_arrays",
			mutate: func(evidence map[string]any) {
				evidence["vp_token"] = map[string]any{"pid": "presentation"}
			},
			status: StatusFail,
		},
	}

	validator := DCQLResponseConstraintsValidator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			evidence := baseEvidence()
			if test.mutate != nil {
				test.mutate(evidence)
			}
			result := validator.Validate(context.Background(), Input{
				Value:  evidence,
				Params: map[string]any{"mode": test.mode},
			})
			require.Equal(t, test.status, result.Status, result.Message)
		})
	}
}

func unsignedSDJWTPresentation() string {
	header, _ := json.Marshal(map[string]any{"alg": "none"})
	payload, _ := json.Marshal(map[string]any{"_sd_alg": "sha-256"})
	return base64.RawURLEncoding.EncodeToString(
		header,
	) + "." + base64.RawURLEncoding.EncodeToString(
		payload,
	) + ".~"
}

func signedSDJWTPresentation() string {
	header, _ := json.Marshal(map[string]any{"alg": "ES256"})
	payload, _ := json.Marshal(map[string]any{"_sd_alg": "sha-256"})
	return base64.RawURLEncoding.EncodeToString(
		header,
	) + "." + base64.RawURLEncoding.EncodeToString(
		payload,
	) + ".signature~"
}

func TestDCQLTwoDistinctFormatPresentations(t *testing.T) {
	baseEvidence := func() map[string]any {
		return map[string]any{
			"dcql_query": map[string]any{"credentials": []any{
				validSDJWTCredentialQuery("pid_sdjwt"),
				map[string]any{
					"id":     "pid_mdoc",
					"format": "mso_mdoc",
					"meta":   map[string]any{"doctype_value": "eu.europa.ec.eudi.pid.1"},
				},
			}},
			"vp_token": map[string]any{
				"pid_sdjwt": []any{signedSDJWTPresentation()},
				"pid_mdoc":  []any{validMDocPresentation(t)},
			},
		}
	}
	validator := DCQLResponseConstraintsValidator{}

	for _, test := range []struct {
		name   string
		mutate func(map[string]any)
		status Status
	}{
		{name: "accepts exact signed SD JWT and mdoc mappings", status: StatusPass},
		{
			name: "rejects an extra response key",
			mutate: func(evidence map[string]any) {
				evidence["vp_token"].(map[string]any)["extra"] = []any{"presentation"}
			},
			status: StatusFail,
		},
		{
			name: "rejects an invalid mdoc presentation",
			mutate: func(evidence map[string]any) {
				evidence["vp_token"].(map[string]any)["pid_mdoc"] = []any{"not-an-mdoc"}
			},
			status: StatusFail,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			evidence := baseEvidence()
			if test.mutate != nil {
				test.mutate(evidence)
			}
			result := validator.Validate(
				context.Background(),
				Input{
					Value:  evidence,
					Params: map[string]any{"mode": "two_distinct_format_presentations"},
				},
			)
			require.Equal(t, test.status, result.Status, result.Message)
		})
	}
}

func validMDocPresentation(t *testing.T) string {
	t.Helper()
	itemValue, err := cbor.Marshal("Alice")
	require.NoError(t, err)
	item, err := cbor.Marshal(map[string]any{
		"digestID": uint64(
			1,
		),
		"random":            []byte("salt"),
		"elementIdentifier": "given_name",
		"elementValue":      cbor.RawMessage(itemValue),
	})
	require.NoError(t, err)
	mso, err := cbor.Marshal(map[string]any{"digestAlgorithm": "SHA-256"})
	require.NoError(t, err)
	msoBytes, err := cbor.Marshal(mso)
	require.NoError(t, err)
	raw, err := cbor.Marshal(map[string]any{
		"version": "1.0", "documents": []any{map[string]any{
			"docType": "eu.europa.ec.eudi.pid.1",
			"issuerSigned": map[string]any{
				"nameSpaces": map[string]any{
					"eu.europa.ec.eudi.pid.1": []any{cbor.Tag{Number: 24, Content: item}},
				},
				"issuerAuth": cbor.Tag{
					Number:  18,
					Content: []any{[]byte{}, map[string]any{}, msoBytes, []byte("signature")},
				},
			},
		}}, "status": uint64(0),
	})
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func credentialFormatEvidence(format, responseID string) map[string]any {
	return credentialFormatEvidenceWithPresentations(
		"pid",
		format,
		responseID,
		[]any{"presentation"},
	)
}

func credentialFormatEvidenceWithPresentations(
	queryID string,
	format string,
	responseID string,
	presentations []any,
) map[string]any {
	meta := map[string]any{"doctype_value": "eu.europa.ec.eudi.pid.1"}
	if format == "dc+sd-jwt" {
		meta = map[string]any{"vct_values": []any{"urn:eudi:pid:1"}}
	}
	credential := map[string]any{"id": queryID, "meta": meta}
	if format != "" {
		credential["format"] = format
	}
	evidence := map[string]any{
		"dcql_query": map[string]any{"credentials": []any{credential}},
	}
	if responseID != "" {
		evidence["vp_token"] = map[string]any{responseID: presentations}
	}
	return evidence
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

func credentialQueryWithClaimIDs(id string, claimIDs ...string) map[string]any {
	credential := validSDJWTCredentialQuery(id)
	claims := make([]any, 0, len(claimIDs))
	for index, claimID := range claimIDs {
		claims = append(claims, map[string]any{
			"id":   claimID,
			"path": []any{fmt.Sprintf("claim_%d", index)},
		})
	}
	credential["claims"] = claims
	return credential
}

func malformedClaimIDEvidence(claimID string) map[string]any {
	return map[string]any{
		"dcql_query": map[string]any{
			"credentials": []any{credentialQueryWithClaimIDs("pid", claimID)},
		},
		"error": "invalid_request",
	}
}

func claimPathEvidence(pathPresent bool, path any, withError bool) map[string]any {
	claim := map[string]any{"id": "given_name"}
	if pathPresent {
		claim["path"] = path
	}
	evidence := map[string]any{
		"dcql_query": map[string]any{"credentials": []any{func() map[string]any {
			credential := validSDJWTCredentialQuery("pid")
			credential["claims"] = []any{claim}
			return credential
		}()}},
	}
	if withError {
		evidence["error"] = "invalid_request"
	}
	return evidence
}

func claimWithoutValuesEvidence(valuesPresent bool, withPresentation bool) map[string]any {
	credential := credentialQueryWithClaimIDs("pid", "given_name")
	claim := credential["claims"].([]any)[0].(map[string]any)
	if valuesPresent {
		claim["values"] = []any{"Filippo"}
	}
	evidence := map[string]any{"dcql_query": map[string]any{"credentials": []any{credential}}}
	if withPresentation {
		evidence["vp_token"] = map[string]any{"pid": []any{"presentation"}}
	}
	return evidence
}

func allowedClaimPathComponentsEvidence(
	first []any,
	second []any,
	third []any,
	withPresentation bool,
) map[string]any {
	claims := map[string]any{"given_name": "Filippo", "nationality": []any{"IT"}}
	if !withPresentation {
		claims = nil
	}
	return allowedClaimPathComponentsWithClaims(first, second, third, claims)
}

func allowedClaimPathComponentsWithClaims(
	first []any,
	second []any,
	third []any,
	disclosedClaims map[string]any,
) map[string]any {
	claims := []any{map[string]any{"path": first}, map[string]any{"path": second}}
	if third != nil {
		claims = append(claims, map[string]any{"path": third})
	}
	credential := validSDJWTCredentialQuery("pid")
	credential["claims"] = claims
	evidence := map[string]any{"dcql_query": map[string]any{"credentials": []any{credential}}}
	if disclosedClaims != nil {
		evidence["vp_token"] = map[string]any{"pid": []any{testSDJWTPresentation(disclosedClaims)}}
	}
	return evidence
}

func testSDJWTPresentation(claims map[string]any) string {
	disclosures := make([]string, 0, len(claims))
	digests := make([]any, 0, len(claims))
	for name, value := range claims {
		raw, _ := json.Marshal([]any{"salt-" + name, name, value})
		disclosure := base64.RawURLEncoding.EncodeToString(raw)
		digest := sha256.Sum256([]byte(disclosure))
		disclosures = append(disclosures, disclosure)
		digests = append(digests, base64.RawURLEncoding.EncodeToString(digest[:]))
	}
	header, _ := json.Marshal(map[string]any{"alg": "none"})
	payload, _ := json.Marshal(map[string]any{"_sd_alg": "sha-256", "_sd": digests})
	token := base64.RawURLEncoding.EncodeToString(
		header,
	) + "." + base64.RawURLEncoding.EncodeToString(
		payload,
	) + ".signature"
	if len(disclosures) == 0 {
		return token + "~"
	}
	return token + "~" + strings.Join(disclosures, "~") + "~"
}

func TestDCQLWithoutClaimDisclosures(t *testing.T) {
	validator := DCQLResponseConstraintsValidator{}
	credential := validSDJWTCredentialQuery("pid")
	delete(credential, "claims")

	noDisclosure := validator.Validate(context.Background(), Input{
		Value: map[string]any{
			"dcql_query": map[string]any{"credentials": []any{credential}},
			"vp_token":   map[string]any{"pid": []any{testSDJWTPresentation(map[string]any{})}},
		},
		Params: map[string]any{"mode": "without_claim_disclosures"},
	})
	require.Equal(t, StatusPass, noDisclosure.Status)

	withDisclosure := validator.Validate(context.Background(), Input{
		Value: map[string]any{
			"dcql_query": map[string]any{"credentials": []any{credential}},
			"vp_token": map[string]any{
				"pid": []any{testSDJWTPresentation(map[string]any{"given_name": "Alice"})},
			},
		},
		Params: map[string]any{"mode": "without_claim_disclosures"},
	})
	require.Equal(t, StatusFail, withDisclosure.Status)
}

func TestDCQLClaimSetsSelection(t *testing.T) {
	validator := DCQLResponseConstraintsValidator{}
	credential := validSDJWTCredentialQuery("pid")
	credential["claims"] = []any{
		map[string]any{"id": "first", "path": []any{"given_name"}},
		map[string]any{"id": "second", "path": []any{"family_name"}},
		map[string]any{"id": "third", "path": []any{"birthdate"}},
	}
	credential["claim_sets"] = []any{[]any{"first"}, []any{"second"}, []any{"third"}}
	evidence := map[string]any{
		"dcql_query": map[string]any{"credentials": []any{credential}},
		"vp_token": map[string]any{
			"pid": []any{testSDJWTPresentation(map[string]any{"given_name": "Alice"})},
		},
	}
	result := validator.Validate(context.Background(), Input{
		Value:  evidence,
		Params: map[string]any{"mode": "claim_sets_preferred_option", "expected_value": 0},
	})
	require.Equal(t, StatusPass, result.Status)

	evidence["vp_token"] = map[string]any{
		"pid": []any{testSDJWTPresentation(map[string]any{"family_name": "Smith"})},
	}
	result = validator.Validate(context.Background(), Input{
		Value:  evidence,
		Params: map[string]any{"mode": "claim_sets_preferred_option", "expected_value": 0},
	})
	require.Equal(t, StatusFail, result.Status)

	delete(evidence, "vp_token")
	result = validator.Validate(context.Background(), Input{
		Value: evidence, Params: map[string]any{"mode": "claim_sets_no_match"},
	})
	require.Equal(t, StatusPass, result.Status)
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
