// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestPlanSignAndVerify(t *testing.T) {
	app, _, _ := authoringFixture(t)
	grant := PlanGrant{
		AuthorWorkflowID:   "ActionAuthorSession-t-author",
		Plan:               testPlan(),
		ApproverID:         "user-1",
		ApproverCollection: "users",
		ConfirmedAt:        1_800_000_000,
	}
	token, err := SignPlan(app, grant)
	require.NoError(t, err)

	verified, err := VerifyPlan(app, token, grant.AuthorWorkflowID)
	require.NoError(t, err)
	assert.Equal(t, grant, verified)

	// The payload segment is the grant, as the dev UI decodes it.
	payload, signature, _ := strings.Cut(token, ".")
	decoded, err := base64.RawURLEncoding.DecodeString(payload)
	require.NoError(t, err)
	var shown struct {
		Plan Spec `json:"plan"`
	}
	require.NoError(t, json.Unmarshal(decoded, &shown))
	assert.Equal(t, grant.Plan, shown.Plan)

	tamperedGrant := grant
	tamperedGrant.Plan.Slot = "other"
	tamperedJSON, err := json.Marshal(tamperedGrant)
	require.NoError(t, err)
	tampered := base64.RawURLEncoding.EncodeToString(tamperedJSON) + "." + signature

	approval, err := SignApproval(app, Approval{AuthorWorkflowID: grant.AuthorWorkflowID})
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		token, workflowID, message string
	}{
		"forged":          {token: "forged.x", message: "plan not signed by Credimi"},
		"unsigned":        {token: payload, message: "plan not signed by Credimi"},
		"tampered":        {token: tampered, message: "plan not signed by Credimi"},
		"approval token":  {token: approval, message: "plan not signed by Credimi"},
		"another session": {workflowID: "ActionAuthorSession-u-author", message: "plan for another session"},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.token == "" {
				tc.token = token
			}
			if tc.workflowID == "" {
				tc.workflowID = grant.AuthorWorkflowID
			}
			_, err := VerifyPlan(app, tc.token, tc.workflowID)
			require.ErrorIs(t, err, ErrForbidden)
			assert.ErrorContains(t, err, tc.message)
		})
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

func TestSpecIsEmpty(t *testing.T) {
	assert.True(t, Spec{}.IsEmpty())
	assert.False(t, Spec{Outcome: OutcomeSuccess}.IsEmpty())
	assert.False(t, Spec{Slot: "other"}.IsEmpty())
	assert.False(t, Spec{WaitSeconds: 20}.IsEmpty())
	assert.False(t, Spec{Parameters: []ParameterSpec{{Name: "x"}}}.IsEmpty())
	assert.False(t, testPlan().IsEmpty())
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
		"${request}: POST https://verifier.example/sessions → deeplink",
		"${pin}: fixed value",
	}, PlanSummary(plan, []ResolvedPrerequisite{
		{Action: "tenant/wallet/onboarding", Name: "Onboarding"},
	}))

	assert.Equal(t, []string{
		"Category: get-credential-generic",
		"Outcome: the wallet rejects the flow",
		"Starts from: a freshly installed app",
		"${offer}: credential offer deeplink of tenant/issuer/pid",
	}, PlanSummary(Spec{
		Slot:    "get-credential",
		Outcome: OutcomeRejection,
		Parameters: []ParameterSpec{{
			Name:   "offer",
			Source: SourceSpec{Kind: SourceCredentialOffer, Identifier: "tenant/issuer/pid"},
		}},
	}, nil))

	assert.Equal(t, []string{
		"Category: other",
		"Outcome: waits 240 s",
		"Starts from: Onboarding (tenant/wallet/onboarding)",
	}, PlanSummary(Spec{Slot: "other", Outcome: OutcomeWait, WaitSeconds: 240},
		[]ResolvedPrerequisite{{Action: "tenant/wallet/onboarding", Name: "Onboarding"}}))
}
