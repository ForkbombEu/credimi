// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testPlan() Spec {
	return Spec{
		Slot:    "verify-credential",
		Outcome: OutcomeSuccess,
		Prerequisites: []PrerequisiteSpec{
			{Action: "tenant/wallet/onboarding"},
		},
		Parameters: []ParameterSpec{
			{
				Name: "request",
				Source: SourceSpec{
					Kind: SourceHTTP,
					Request: &HTTPSourceRequest{
						Method: "POST",
						URL:    "https://verifier.example/sessions",
					},
					ValuePath: "deeplink",
					Status: &StatusSpec{
						URL:     "https://verifier.example/sessions/${id}",
						Path:    "status",
						Success: []string{"done"},
					},
				},
			},
			{Name: "pin", Source: SourceSpec{Kind: SourceLiteral, Value: "123456"}},
		},
	}
}

func TestPlanSHA256(t *testing.T) {
	plan := testPlan()
	sha := PlanSHA256(plan)
	assert.Len(t, sha, 64)
	assert.Equal(t, sha, PlanSHA256(testPlan()))

	other := testPlan()
	other.Parameters[1].Source.Value = "654321"
	assert.NotEqual(t, sha, PlanSHA256(other))
	other = testPlan()
	other.Outcome = OutcomeRejection
	assert.NotEqual(t, sha, PlanSHA256(other))
	wait := Spec{Slot: "other", Outcome: OutcomeWait, WaitSeconds: 20}
	longer := wait
	longer.WaitSeconds = 240
	assert.NotEqual(t, PlanSHA256(wait), PlanSHA256(longer))
}

func TestParameterInfosAndPlanSummary(t *testing.T) {
	plan := testPlan()
	assert.Equal(t, []ParameterInfo{
		{
			Name:        "request",
			Kind:        SourceHTTP,
			Description: "POST https://verifier.example/sessions → deeplink",
			HasStatus:   true,
		},
		{Name: "pin", Kind: SourceLiteral, Description: "fixed value"},
	}, ParameterInfos(plan))
	assert.NotNil(t, ParameterInfos(Spec{}))
	assert.Empty(t, ParameterInfos(Spec{}))

	assert.Equal(t, []string{
		"Category: verify-credential",
		"Outcome: success",
		"Starts from: Onboarding (tenant/wallet/onboarding)",
		"${request}: POST https://verifier.example/sessions, value from response field deeplink",
		"  status: GET https://verifier.example/sessions/${id}, field status, expected done",
		"${pin}: the fixed value \"123456\"",
	}, PlanSummary(plan, []ResolvedPrerequisite{
		{Action: "tenant/wallet/onboarding", Name: "Onboarding"},
	}, nil))

	assert.Equal(t, []string{
		"Category: verify-credential",
		"Replaces: Verify PID (tenant/wallet/verify-pid)",
		"Outcome: success",
		"Starts from: Onboarding (tenant/wallet/onboarding)",
	}, PlanSummary(Spec{Slot: "verify-credential", Outcome: OutcomeSuccess},
		[]ResolvedPrerequisite{{Action: "tenant/wallet/onboarding", Name: "Onboarding"}},
		&ReplaceTarget{Action: "tenant/wallet/verify-pid", Name: "Verify PID"}))

	assert.Equal(t, []string{
		"Category: get-credential-generic",
		"Outcome: the wallet rejects the flow",
		"Starts from: a freshly installed app",
		"${offer}: a new credential offer from tenant/issuer/pid on every run",
	}, PlanSummary(Spec{
		Slot:    "get-credential",
		Outcome: OutcomeRejection,
		Parameters: []ParameterSpec{{
			Name:   "offer",
			Source: SourceSpec{Kind: SourceCredentialOffer, Identifier: "tenant/issuer/pid"},
		}},
	}, nil, nil))

	assert.Equal(t, []string{
		"Category: other",
		"Outcome: waits 240 s",
		"Starts from: Onboarding (tenant/wallet/onboarding)",
	}, PlanSummary(Spec{Slot: "other", Outcome: OutcomeWait, WaitSeconds: 240},
		[]ResolvedPrerequisite{{Action: "tenant/wallet/onboarding", Name: "Onboarding"}}, nil))

	assert.Equal(t, []string{
		"Category: install-app",
		"Outcome: success",
		"Starts from: the wallet is not installed",
	}, PlanSummary(Spec{Slot: "install-app", Outcome: OutcomeSuccess}, nil, nil))
}

func TestPlanSummaryDescribesSourcesInFull(t *testing.T) {
	longBody := map[string]any{"text": strings.Repeat("é", 600)}
	plan := Spec{
		Slot:    "verify-credential",
		Outcome: OutcomeRejection,
		Prerequisites: []PrerequisiteSpec{
			{Action: "tenant/wallet/onboarding", Parameters: []ParameterSpec{
				{Name: "pin", Source: SourceSpec{Kind: SourceLiteral, Value: "12\"34\n"}},
			}},
			{Action: "tenant/wallet/get-pid", Parameters: []ParameterSpec{
				{Name: "offer", Source: SourceSpec{
					Kind:       SourceCredentialOffer,
					Identifier: "tenant/issuer/pid",
				}},
				{Name: "call", Source: SourceSpec{
					Kind: SourceHTTP,
					Request: &HTTPSourceRequest{
						Method: "GET",
						URL:    "https://issuer.example/offer",
					},
					ValuePath: "uri",
				}},
			}},
		},
		Parameters: []ParameterSpec{
			{Name: "request", Source: SourceSpec{
				Kind: SourceHTTP,
				Request: &HTTPSourceRequest{
					Method: "POST",
					URL:    "https://verifier.example/sessions",
					Headers: map[string]string{
						"Content-Type": "application/json",
						"Accept":       "application/json",
					},
					Body: map[string]any{"dcql": "<pid>", "n": 1},
				},
				ValuePath: "deeplink",
				Captures:  map[string]string{"sid": "id", "nonce": "state.nonce"},
				Status: &StatusSpec{
					URL:       "https://verifier.example/sessions/${sid}",
					Path:      "status",
					Success:   []string{"done"},
					Rejection: []string{"denied", "failed"},
				},
			}},
			{Name: "long", Source: SourceSpec{
				Kind: SourceHTTP,
				Request: &HTTPSourceRequest{
					Method: "POST",
					URL:    "https://x.example",
					Body:   longBody,
				},
				ValuePath: "v",
			}},
			{Name: "presentation", Source: SourceSpec{
				Kind:       SourceUseCaseVerification,
				Identifier: "tenant/verifier/age",
			}},
		},
	}
	lines := PlanSummary(plan, []ResolvedPrerequisite{
		{Action: "tenant/wallet/onboarding", Name: "Onboarding"},
		{Action: "tenant/wallet/get-pid", Name: "Get PID"},
	}, nil)
	longLine := "  body: " + string([]rune(`{"text":"` + strings.Repeat("é", 600))[:500]) + "…"
	assert.Equal(t, []string{
		"Category: verify-credential",
		"Outcome: the wallet rejects the flow",
		"Starts from: Onboarding (tenant/wallet/onboarding)",
		`  ${pin}: the fixed value "12\"34\n"`,
		"Starts from: Get PID (tenant/wallet/get-pid)",
		"  ${offer}: a new credential offer from tenant/issuer/pid on every run",
		"  ${call}: GET https://issuer.example/offer, value from response field uri",
		"${request}: POST https://verifier.example/sessions, value from response field deeplink",
		"  headers: Accept: application/json",
		"  headers: Content-Type: application/json",
		`  body: {"dcql":"<pid>","n":1}`,
		"  captures: nonce from state.nonce",
		"  captures: sid from id",
		"  status: GET https://verifier.example/sessions/${sid}, field status, " +
			"expected denied, failed",
		"${long}: POST https://x.example, value from response field v",
		longLine,
		"${presentation}: a new presentation request from tenant/verifier/age on every run",
	}, lines)
}
