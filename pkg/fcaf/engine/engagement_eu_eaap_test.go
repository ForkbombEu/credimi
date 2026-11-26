// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/evidence"
	"github.com/forkbombeu/credimi/pkg/fcaf/validators"
	"github.com/stretchr/testify/require"
)

// TestEngagementEUEAAPVerdict runs WS_RP_IA_Engagement__001b from the relying
// party catalog on the evidence a Wallet without an `eu-eaap://` handler
// leaves: Capture's session shows no request retrieval and the mobile step
// stored no screenshot. Android's unresolved intent in Maestro's output makes
// the test fail; without it, the missing screenshot keeps the test blocked.
func TestEngagementEUEAAPVerdict(t *testing.T) {
	const testID = "WS_RP_IA_Engagement__001b"
	cat, err := catalog.Load("../../../config_templates/fcaf/wallet_solution/relying_party")
	require.NoError(t, err)
	fcafEngine, err := New(nil)
	require.NoError(t, err)

	// The session Capture serves when the Wallet never retrieved the request.
	session := map[string]any{
		"session_id":  "s1",
		"status":      "created",
		"deeplink":    "eu-eaap://?client_id=x509_hash%3Aabc&request_uri=https%3A%2F%2Fcapture-wallet.credimi.io%2Fr",
		"request_uri": "https://capture-wallet.credimi.io/openid4vp/sessions/s1/request",
		"raw":         map[string]any{},
	}
	unresolved := "Open eu-eaap://?client_id=x509_hash%3Aabc... FAILED\n" +
		"Activity not started, unable to resolve Intent { act=android.intent.action.VIEW " +
		"dat=eu-eaap://?client_id=x509_hash:abc flg=0x10000000 }\n"
	invokedURL := "https://credimi.io/api/files/pipeline_results/e34qcz24hpcfode/" +
		"engagement_eu_eaap_6cbe4a87_invoke_wallet_fcaf_engagement_eu_eaap_invoked_a1b2c3d4e5.png"

	tests := []struct {
		name          string
		invocation    map[string]any
		wantStatus    validators.Status
		wantAssertion validators.Status
		wantMessage   string
	}{
		{
			name: "no screenshot and an unresolvable eu-eaap intent fails",
			invocation: map[string]any{
				"wallet_screenshots": nil,
				"maestro_output":     unresolved,
				"maestro_stderr":     nil,
			},
			wantStatus:    validators.StatusFail,
			wantAssertion: validators.StatusFail,
			wantMessage:   "the wallet registers no handler for eu-eaap://: Android could not resolve the intent",
		},
		{
			name: "no screenshot and no console output stays blocked",
			invocation: map[string]any{
				"wallet_screenshots": nil,
				"maestro_output":     nil,
				"maestro_stderr":     nil,
			},
			wantStatus:    validators.StatusBlocked,
			wantAssertion: validators.StatusBlocked,
			wantMessage:   validators.ScreenshotEvidenceMissingMessage,
		},
		{
			name: "no screenshot and another failure stays blocked",
			invocation: map[string]any{
				"wallet_screenshots": []any{},
				"maestro_output":     "Assertion is false: \"Home\" is visible\n",
				"maestro_stderr":     "device disconnected",
			},
			wantStatus:    validators.StatusBlocked,
			wantAssertion: validators.StatusBlocked,
			wantMessage:   validators.ScreenshotEvidenceMissingMessage,
		},
		{
			name: "the engagement screenshot passes the screenshot assertion",
			invocation: map[string]any{
				"wallet_screenshots": []any{invokedURL},
				"maestro_output":     "Take screenshot fcaf-engagement-eu-eaap-invoked COMPLETED\n",
				"maestro_stderr":     nil,
			},
			// The session above records no retrieval, so the retrieval
			// assertion still fails the test.
			wantStatus:    validators.StatusFail,
			wantAssertion: validators.StatusPass,
			wantMessage:   `a screenshot named "fcaf-engagement-eu-eaap-invoked" was stored`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report, err := fcafEngine.ExecuteCatalog(
				context.Background(),
				cat,
				[]string{testID},
				"",
				nil,
				evidence.Bundle{PipelineOutputs: map[string]any{
					"pipeline.wallet.engagement.eu-eaap": map[string]any{"output": map[string]any{
						"wallet_engagement": session,
						"wallet_invocation": test.invocation,
					}},
				}},
			)
			require.NoError(t, err)
			require.Len(t, report.Tests, 1)
			result := report.Tests[0]
			require.Equal(t, test.wantStatus, result.Status, "%+v", result.Assertions)
			var screen *AssertionResult
			for index := range result.Assertions {
				if result.Assertions[index].ID == "wallet_shows_the_engagement_screen" {
					screen = &result.Assertions[index]
				}
			}
			require.NotNil(t, screen)
			require.Equal(t, test.wantAssertion, screen.Status, screen.Message)
			require.Contains(t, screen.Message, test.wantMessage)
		})
	}
}
