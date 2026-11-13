// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// A failure must say what the Wallet actually did: sent nothing, sent an
// error, or sent a presentation. The 12/11/2026 run reported a missing
// Authorization Response as a malformed vp_token, and a shared credential as a
// missing error.
func TestFailuresDescribeTheWalletAnswer(t *testing.T) {
	query := map[string]any{"dcql_query": map[string]any{
		"credentials": []any{map[string]any{
			"id":     "pid",
			"format": "dc+sd-jwt",
			"meta":   map[string]any{"vct_values": []any{"urn:eudi:pid:1"}},
			"claims": []any{map[string]any{"path": []any{"given_name"}}},
		}},
	}}
	unanswered := retrievedSessionWithoutResponse(query)
	presented := retrievedSessionWithPresentation(query, "pid")

	cases := []struct {
		name      string
		validator Validator
		params    map[string]any
		value     any
		contains  []string
		excludes  []string
	}{
		{
			name:      "presentation required but the wallet sent nothing",
			validator: DCQLResponseConstraintsValidator{},
			params:    map[string]any{"mode": "credentials_match"},
			value:     unanswered,
			contains: []string{
				"wallet returned no vp_token",
				"verifier session request_retrieved: the wallet retrieved the request and sent no Authorization Response",
			},
			excludes: []string{"not an object keyed"},
		},
		{
			name:      "error required but the wallet shared a credential",
			validator: DCQLResponseConstraintsValidator{},
			params:    map[string]any{"mode": "invalid_request_required"},
			value:     presented,
			contains:  []string{"wallet returned a vp_token, expected error invalid_request"},
			excludes:  []string{"verifier session"},
		},
		{
			name:      "error required but the wallet sent nothing",
			validator: OID4VPErrorResponseRequiredValidator{},
			params:    map[string]any{"code": "invalid_request"},
			value:     unanswered,
			contains: []string{
				`wallet returned no error, expected "invalid_request"`,
				"the wallet retrieved the request and sent no Authorization Response",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.validator.Validate(
				context.Background(),
				Input{Value: tc.value, Params: tc.params},
			)
			require.Equal(t, StatusFail, result.Status, result.Message)
			for _, text := range tc.contains {
				require.Contains(t, result.Message, text)
			}
			for _, text := range tc.excludes {
				require.NotContains(t, result.Message, text)
			}
		})
	}
}
