// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	"github.com/stretchr/testify/require"
)

// Evidence that was never captured, and screenshot evidence that stored no
// screenshot, must report blocked with a message that names the harness gap.
// The Wallet's own response members are the exception: nil there is the
// absence the assertion exists to test, so the validator still judges it.
func TestMissingInputEvidence(t *testing.T) {
	tests := []struct {
		name      string
		assertion dsl.AssertionDefinition
		value     any
		want      *validators.Result
	}{
		{
			name: "screenshot list was never captured",
			assertion: dsl.AssertionDefinition{
				Validator: "evidence.non_empty",
				Input:     "evidence.visual_evidence",
			},
			value: nil,
			want: &validators.Result{
				Status:  validators.StatusBlocked,
				Message: validators.ScreenshotEvidenceMissingMessage,
			},
		},
		{
			name: "screenshot list is empty",
			assertion: dsl.AssertionDefinition{
				Validator: "evidence.non_empty",
				Input:     "evidence.visual_evidence",
			},
			value: []any{},
			want: &validators.Result{
				Status:  validators.StatusBlocked,
				Message: validators.ScreenshotEvidenceMissingMessage,
			},
		},
		{
			name: "named screenshot list is empty",
			assertion: dsl.AssertionDefinition{
				Validator: "evidence.screenshot_named",
				Input:     "evidence.numeric_issuance_screenshots",
			},
			value: []string{},
			want: &validators.Result{
				Status:  validators.StatusBlocked,
				Message: validators.ScreenshotEvidenceMissingMessage,
			},
		},
		{
			name: "request object was never captured",
			assertion: dsl.AssertionDefinition{
				Validator: "jwt.payload_field_presence",
				Input:     "evidence.request_object",
			},
			value: nil,
			want: &validators.Result{
				Status:  validators.StatusBlocked,
				Message: "evidence request_object was not captured",
			},
		},
		{
			name: "session was never captured",
			assertion: dsl.AssertionDefinition{
				Validator: "dcql.response_satisfies_constraints",
				Input:     "evidence.capture_session",
			},
			value: nil,
			want: &validators.Result{
				Status:  validators.StatusBlocked,
				Message: "evidence capture_session was not captured",
			},
		},
		{
			// A nil Wallet response is the Wallet having sent nothing, which
			// the assertion must report as a real failure.
			name: "wallet response is left to the validator",
			assertion: dsl.AssertionDefinition{
				Validator: "json.field_required",
				Input:     "evidence.wallet_response",
			},
			value: nil,
			want:  nil,
		},
		{
			name: "referenced wallet response is left to the validator",
			assertion: dsl.AssertionDefinition{
				Validator: "json.field_required",
				Input:     "evidence.referenced_wallet_response",
			},
			value: nil,
			want:  nil,
		},
		{
			name: "non-empty screenshot list is left to the validator",
			assertion: dsl.AssertionDefinition{
				Validator: "evidence.non_empty",
				Input:     "evidence.visual_evidence",
			},
			value: []any{"https://example.test/screenshot.png"},
			want:  nil,
		},
		{
			name: "empty non-screenshot list is left to the validator",
			assertion: dsl.AssertionDefinition{
				Validator: "evidence.non_empty",
				Input:     "evidence.pid_sdjwt_presentations",
			},
			value: []any{},
			want:  nil,
		},
		{
			name: "runtime input is not evidence",
			assertion: dsl.AssertionDefinition{
				Validator: "evidence.present",
				Input:     "runtime.value",
			},
			value: nil,
			want:  nil,
		},
		{
			// A skip records a setup limitation and never reads the value.
			name: "skip statement is never blocked by a nil input",
			assertion: dsl.AssertionDefinition{
				Validator: "fcaf.skipped",
				Input:     "evidence.request_object",
			},
			value: nil,
			want:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, missingInputEvidence(test.assertion, test.value))
		})
	}
}

// TestEngineBlocksHarnessGaps covers the three shapes the run review found
// scored as Wallet failures: a nil screenshot list, an empty screenshot list,
// and a producing step that never captured its output. A nil wallet_response
// keeps its failure, and the blocked verdict outranks a fail recorded beside it.
func TestEngineBlocksHarnessGaps(t *testing.T) {
	tests := []struct {
		name        string
		evidence    map[string]dsl.EvidenceBinding
		output      map[string]any
		assertions  []dsl.AssertionDefinition
		wantStatus  validators.Status
		wantMessage string
	}{
		{
			name: "nil screenshot evidence blocks",
			evidence: map[string]dsl.EvidenceBinding{
				"visual_evidence": {From: "pipeline.step.outputs.visual_evidence"},
			},
			output: map[string]any{"visual_evidence": nil},
			assertions: []dsl.AssertionDefinition{{
				ID:        "screen-is-shown",
				Validator: "evidence.non_empty",
				Input:     "evidence.visual_evidence",
			}},
			wantStatus:  validators.StatusBlocked,
			wantMessage: validators.ScreenshotEvidenceMissingMessage,
		},
		{
			name: "empty screenshot evidence blocks",
			evidence: map[string]dsl.EvidenceBinding{
				"visual_evidence": {From: "pipeline.step.outputs.visual_evidence"},
			},
			output: map[string]any{"visual_evidence": []any{}},
			assertions: []dsl.AssertionDefinition{{
				ID:        "screen-is-shown",
				Validator: "evidence.non_empty",
				Input:     "evidence.visual_evidence",
			}},
			wantStatus:  validators.StatusBlocked,
			wantMessage: validators.ScreenshotEvidenceMissingMessage,
		},
		{
			name: "uncaptured request object blocks",
			evidence: map[string]dsl.EvidenceBinding{
				"request_object": {From: "pipeline.step.outputs.request_object"},
			},
			output: map[string]any{"request_object": nil},
			assertions: []dsl.AssertionDefinition{{
				ID:        "client-metadata-is-delivered",
				Validator: "jwt.payload_field_presence",
				Input:     "evidence.request_object",
				Params:    map[string]any{"field": "client_metadata", "present": true},
			}},
			wantStatus:  validators.StatusBlocked,
			wantMessage: "evidence request_object was not captured",
		},
		{
			name: "nil wallet response stays a failure",
			evidence: map[string]dsl.EvidenceBinding{
				"wallet_response": {From: "pipeline.step.outputs.wallet_response"},
			},
			output: map[string]any{"wallet_response": nil},
			assertions: []dsl.AssertionDefinition{{
				ID:        "authorization-response-posted",
				Validator: "json.field_required",
				Input:     "evidence.wallet_response",
				Params:    map[string]any{"field": "vp_token"},
			}},
			wantStatus: validators.StatusFail,
		},
		{
			name: "blocked evidence outranks a wallet failure",
			evidence: map[string]dsl.EvidenceBinding{
				"visual_evidence": {From: "pipeline.step.outputs.visual_evidence"},
				"wallet_response": {From: "pipeline.step.outputs.wallet_response"},
			},
			output: map[string]any{
				"visual_evidence": []any{},
				"wallet_response": nil,
			},
			assertions: []dsl.AssertionDefinition{
				{
					ID:        "authorization-response-posted",
					Validator: "json.field_required",
					Input:     "evidence.wallet_response",
					Params:    map[string]any{"field": "vp_token"},
				},
				{
					ID:        "screen-is-shown",
					Validator: "evidence.non_empty",
					Input:     "evidence.visual_evidence",
				},
			},
			wantStatus:  validators.StatusBlocked,
			wantMessage: validators.ScreenshotEvidenceMissingMessage,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cat := &catalog.Catalog{Tests: map[string]dsl.TestDefinition{
				"test-1": {
					ID:         "test-1",
					Suite:      dsl.Suite{SUT: "wallet_solution", Role: "relying_party"},
					Evidence:   test.evidence,
					Assertions: test.assertions,
					Verdict:    dsl.VerdictPolicy{PassWhen: "all_assertions_pass"},
				},
			}}
			fcafEngine, err := New(nil)
			require.NoError(t, err)

			report, err := fcafEngine.ExecuteCatalog(
				context.Background(),
				cat,
				[]string{"test-1"},
				"",
				nil,
				evidence.Bundle{PipelineOutputs: map[string]any{
					"pipeline.step": map[string]any{"output": test.output},
				}},
			)

			require.NoError(t, err)
			require.Equal(t, test.wantStatus, report.Tests[0].Status)
			if test.wantMessage != "" {
				found := false
				for _, assertion := range report.Tests[0].Assertions {
					if assertion.Message == test.wantMessage {
						found = true
					}
				}
				require.Truef(
					t,
					found,
					"no assertion reported %q: %+v",
					test.wantMessage,
					report.Tests[0].Assertions,
				)
			}
		})
	}
}
