// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	euEAAPInvokedURL = "https://credimi.io/api/files/pipeline_results/e34qcz24hpcfode/" +
		"engagement_eu_eaap_6cbe4a87_invoke_wallet_fcaf_engagement_eu_eaap_invoked_a1b2c3d4e5.png"
	euEAAPOnboardURL = "https://credimi.io/api/files/pipeline_results/e34qcz24hpcfode/" +
		"engagement_eu_eaap_6cbe4a87_invoke_wallet_pid_issued_a1b2c3d4e5.png"
	// What `am start` answers, and Maestro's openLink reports, for a link no
	// installed app handles.
	euEAAPUnresolved = "Open eu-eaap://?client_id=x509_hash%3Aabc... FAILED\n" +
		"Activity not started, unable to resolve Intent { act=android.intent.action.VIEW " +
		"dat=eu-eaap://?client_id=x509_hash:abc&request_uri=https://capture-wallet.credimi.io/r " +
		"flg=0x10000000 }\n"
)

func TestEvidenceCustomSchemeInvokedValidator(t *testing.T) {
	params := map[string]any{"name": "fcaf-engagement-eu-eaap-invoked", "scheme": "eu-eaap://"}
	tests := []struct {
		name        string
		params      map[string]any
		value       any
		wantStatus  Status
		wantMessage string
	}{
		{
			name:       "the engagement screenshot proves the invocation",
			params:     params,
			value:      map[string]any{"wallet_screenshots": []any{euEAAPInvokedURL}},
			wantStatus: StatusPass,
		},
		{
			name:   "an unresolvable intent in stdout fails without screenshots",
			params: params,
			value: map[string]any{
				"wallet_screenshots": nil,
				"maestro_output":     euEAAPUnresolved,
			},
			wantStatus: StatusFail,
			wantMessage: "the wallet registers no handler for eu-eaap://: " +
				"Android could not resolve the intent (Activity not started, unable to resolve Intent",
		},
		{
			name:   "an unresolvable intent in stderr fails beside other screenshots",
			params: params,
			value: map[string]any{
				"wallet_screenshots": []any{euEAAPOnboardURL},
				"maestro_stderr": "maestro.android.AndroidOperationFailedException: " +
					"Activity not started, unable to resolve Intent { dat=EU-EAAP://?x }",
			},
			wantStatus:  StatusFail,
			wantMessage: "the wallet registers no handler for eu-eaap://",
		},
		{
			name:   "no screenshot and no console output stays blocked",
			params: params,
			value: map[string]any{
				"wallet_screenshots": nil,
				"maestro_output":     nil,
				"maestro_stderr":     nil,
			},
			wantStatus:  StatusBlocked,
			wantMessage: ScreenshotEvidenceMissingMessage,
		},
		{
			name:   "another failure stays blocked",
			params: params,
			value: map[string]any{
				"wallet_screenshots": []any{},
				"maestro_output":     "Assertion is false: \"Home\" is visible\n",
			},
			wantStatus:  StatusBlocked,
			wantMessage: ScreenshotEvidenceMissingMessage,
		},
		{
			name:   "an unresolvable intent for another scheme stays blocked",
			params: params,
			value: map[string]any{
				"maestro_output": "Activity not started, unable to resolve Intent " +
					"{ dat=openid-credential-offer://?credential_offer_uri=x }\n",
			},
			wantStatus:  StatusBlocked,
			wantMessage: ScreenshotEvidenceMissingMessage,
		},
		{
			name:        "other screenshots without the engagement one fail as screenshot_named",
			params:      params,
			value:       map[string]any{"wallet_screenshots": []any{euEAAPOnboardURL}},
			wantStatus:  StatusFail,
			wantMessage: `no screenshot named "fcaf-engagement-eu-eaap-invoked" was stored`,
		},
		{
			name:        "scheme is required",
			params:      map[string]any{"name": "fcaf-engagement-eu-eaap-invoked"},
			value:       map[string]any{},
			wantStatus:  StatusError,
			wantMessage: "scheme param is required",
		},
		{
			name:        "name is required",
			params:      map[string]any{"scheme": "eu-eaap://"},
			value:       map[string]any{"maestro_output": euEAAPUnresolved},
			wantStatus:  StatusError,
			wantMessage: "name param is required",
		},
		{
			name:        "a non-object value fails",
			params:      params,
			value:       []any{euEAAPInvokedURL},
			wantStatus:  StatusFail,
			wantMessage: "expected an object",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := EvidenceCustomSchemeInvokedValidator{}.Validate(context.Background(), Input{
				Value:  test.value,
				Params: test.params,
			})
			require.Equal(t, test.wantStatus, result.Status, result.Message)
			require.Contains(t, result.Message, test.wantMessage)
		})
	}
}
