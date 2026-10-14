// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// The URLs below are copied from the reference run's stored evidence
// (fcaf_assessment_4t9hxzgxlf.json), so the file names are the ones Credimi
// actually produces: Snakecase(<step id>-<screenshot name>)_<random>.png.
const (
	noMatchTerminalURL = "https://credimi.io/api/files/pipeline_results/e34qcz24hpcfode/" +
		"dcql_claims_path_no_match_00a99e11_wallet_expect_no_matching_document_he550dgvgz.png"
	noMatchTerminalQueryURL = "https://credimi.io/api/files/pipeline_results/e34qcz24hpcfode/" +
		"dcql_claims_values_no_match_9e05fa90_wallet_expect_no_matching_document_p57gj2sum3.png" +
		"?thumb=200x0"
	// A notice screenshot, named after the step that takes it. The step id is
	// `dcql-trust-mechanisms-157d0f39-expect-no-matching-credential`.
	noMatchNoticeURL = "https://credimi.io/api/files/pipeline_results/e34qcz24hpcfode/" +
		"dcql_trust_mechanisms_157d0f39_expect_no_matching_credential_no_match_notice_9w2vqk4d1a.png"
	// The longest step id that uses fcaf-expect-no-matching-document, with a
	// notice name that does not fit the room left before PocketBase's
	// 100-character cap, so the name is truncated away and cannot be proven.
	truncatedNoticeURL = "https://credimi.io/api/files/pipeline_results/e34qcz24hpcfode/" +
		"mdoc_status_reference_rejection_46c13b7f_probe_mdoc_status_without_status_list_" +
		"no_matching_document__a1b2c3d4e5.png"
)

func TestEvidenceScreenshotNamedValidator(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]any
		value       any
		wantStatus  Status
		wantMessage string
	}{
		{
			name:       "matches the notice screenshot",
			params:     map[string]any{"name": "no-match-notice"},
			value:      []any{noMatchTerminalURL, noMatchNoticeURL},
			wantStatus: StatusPass,
		},
		{
			name:        "the terminal screenshot alone does not prove the notice",
			params:      map[string]any{"name": "no-match-notice"},
			value:       []any{noMatchTerminalURL, noMatchTerminalQueryURL},
			wantStatus:  StatusFail,
			wantMessage: `no screenshot named "no-match-notice" was stored`,
		},
		{
			name:       "matches the terminal screenshot name",
			params:     map[string]any{"name": "expect-no-matching-document"},
			value:      []any{noMatchTerminalURL},
			wantStatus: StatusPass,
		},
		{
			name:       "ignores a query string",
			params:     map[string]any{"name": "expect-no-matching-document"},
			value:      []any{noMatchTerminalQueryURL},
			wantStatus: StatusPass,
		},
		{
			name:        "blocks a name the 100-character cap truncates away",
			params:      map[string]any{"name": "no-matching-document-notice"},
			value:       []any{truncatedNoticeURL},
			wantStatus:  StatusBlocked,
			wantMessage: "cut off at PocketBase's 100-character cap",
		},
		{
			// The same cap applies to a name whose stem is under it: the stored
			// name is complete, so a missing screenshot is the Wallet's.
			name:        "a stem under the cap still fails",
			params:      map[string]any{"name": "no-matching-document-notice"},
			value:       []any{noMatchTerminalURL},
			wantStatus:  StatusFail,
			wantMessage: `no screenshot named "no-matching-document-notice" was stored`,
		},
		{
			name:       "accepts a single URL string",
			params:     map[string]any{"name": "no-match-notice"},
			value:      noMatchNoticeURL,
			wantStatus: StatusPass,
		},
		{
			name:        "fails when the step stored no screenshot",
			params:      map[string]any{"name": "no-match-notice"},
			value:       nil,
			wantStatus:  StatusFail,
			wantMessage: "expected a list of screenshot URLs",
		},
		{
			name:        "fails when the evidence is not a URL list",
			params:      map[string]any{"name": "no-match-notice"},
			value:       map[string]any{"screenshots": []any{noMatchNoticeURL}},
			wantStatus:  StatusFail,
			wantMessage: "expected a list of screenshot URLs",
		},
		{
			name:        "fails when a list item is not a URL",
			params:      map[string]any{"name": "no-match-notice"},
			value:       []any{noMatchNoticeURL, 7},
			wantStatus:  StatusFail,
			wantMessage: "evidence[1] is int, expected a screenshot URL",
		},
		{
			name:        "fails when the list is empty",
			params:      map[string]any{"name": "no-match-notice"},
			value:       []any{},
			wantStatus:  StatusFail,
			wantMessage: "evidence contains no screenshot",
		},
		{
			name:        "requires the name param",
			params:      map[string]any{},
			value:       []any{noMatchNoticeURL},
			wantStatus:  StatusError,
			wantMessage: "name param is required",
		},
	}

	validator := EvidenceScreenshotNamedValidator{}
	require.Equal(t, "evidence.screenshot_named", validator.ID())
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := validator.Validate(context.Background(), Input{
				Value:  test.value,
				Params: test.params,
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
			if test.wantMessage != "" {
				require.Contains(t, result.Message, test.wantMessage)
			}
		})
	}
}
