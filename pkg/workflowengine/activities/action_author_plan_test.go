// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

// signTestPlan returns a plan token of the test caller's workflow confirmed
// by the session's actor.
func signTestPlan(
	t *testing.T,
	app core.App,
	session actionauthor.BuildRequest,
	plan actionauthor.Spec,
) string {
	t.Helper()
	token, err := actionauthor.SignPlan(app, actionauthor.PlanGrant{
		AuthorWorkflowID:   authorTestAuthorID,
		Plan:               plan,
		ApproverID:         session.ActorID,
		ApproverCollection: session.ActorCollection,
		ConfirmedAt:        time.Now().Unix(),
	})
	require.NoError(t, err)
	return token
}

func TestEffectivePlan(t *testing.T) {
	app, session, _ := authorSaveFixture(t)
	startSpec := actionauthor.Spec{
		Slot:          "get-credential",
		Prerequisites: []actionauthor.PrerequisiteSpec{{Action: "/org/wallet/onboarding"}},
	}
	confirmed := actionauthor.Spec{Slot: "onboarding"}
	normalized := actionauthor.Spec{Slot: "onboarding", Outcome: actionauthor.OutcomeSuccess}
	token := signTestPlan(t, app, session, confirmed)
	ref := func(spec actionauthor.Spec) ActionAuthorSessionRef {
		return ActionAuthorSessionRef{
			SessionWorkflowID: authorTestSessionID,
			Session:           actionauthor.Session{BuildRequest: session, Spec: spec},
		}
	}

	// A start spec is the plan; the session's copy stays as it is.
	fixed := ref(startSpec)
	plan, err := EffectivePlan(app, fixed, authorTestAuthorID, "")
	require.NoError(t, err)
	require.Equal(t, actionauthor.Spec{
		Slot:          "get-credential",
		Outcome:       actionauthor.OutcomeSuccess,
		Prerequisites: []actionauthor.PrerequisiteSpec{{Action: "org/wallet/onboarding"}},
	}, plan)
	require.Equal(t, "/org/wallet/onboarding", fixed.Session.Prerequisites[0].Action)

	_, err = EffectivePlan(app, fixed, authorTestAuthorID, token)
	require.ErrorIs(t, err, actionauthor.ErrInvalid)
	require.ErrorContains(t, err, "this session's plan was fixed when it started")

	_, err = EffectivePlan(app, ref(actionauthor.Spec{Slot: "install-app"}), authorTestAuthorID, "")
	require.ErrorIs(t, err, actionauthor.ErrInvalid)
	require.ErrorContains(t, err, "action author session spec: invalid authoring request: slot")

	// Without a start spec the plan comes from a token the actor confirmed.
	plan, err = EffectivePlan(app, ref(actionauthor.Spec{}), authorTestAuthorID, token)
	require.NoError(t, err)
	require.Equal(t, normalized, plan)

	_, err = EffectivePlan(app, ref(actionauthor.Spec{}), authorTestAuthorID, "")
	require.ErrorIs(t, err, actionauthor.ErrInvalid)
	require.ErrorContains(t, err, "no confirmed plan: propose one with propose_plan")

	other := session
	other.ActorID = "someone-else"
	for name, tc := range map[string]struct {
		token, workflowID, message string
	}{
		"forged":          {token: "forged.x", message: "plan not signed by Credimi"},
		"another session": {workflowID: "ActionAuthorSession-u-author", message: "plan for another session"},
		"another user": {
			token:   signTestPlan(t, app, other, confirmed),
			message: "plan confirmed by another user",
		},
	} {
		if tc.token == "" {
			tc.token = token
		}
		if tc.workflowID == "" {
			tc.workflowID = authorTestAuthorID
		}
		_, err := EffectivePlan(app, ref(actionauthor.Spec{}), tc.workflowID, tc.token)
		require.ErrorIs(t, err, actionauthor.ErrForbidden, name)
		require.ErrorContains(t, err, tc.message, name)
	}

	invalid := signTestPlan(t, app, session, actionauthor.Spec{
		Slot:    "onboarding",
		Outcome: actionauthor.OutcomeRejection,
	})
	_, err = EffectivePlan(app, ref(actionauthor.Spec{}), authorTestAuthorID, invalid)
	require.ErrorIs(t, err, actionauthor.ErrInvalid)
	require.ErrorContains(t, err, "confirmed plan: invalid authoring request: outcome")
}

func TestGenerateActionAuthorParametersUsesThePlanToken(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	caller := authorTestCaller()
	caller.Namespace = namespace
	plan := actionauthor.Spec{
		Slot: "onboarding",
		Parameters: []actionauthor.ParameterSpec{{
			Name:   "PIN",
			Source: actionauthor.SourceSpec{Kind: actionauthor.SourceLiteral, Value: "123456"},
		}},
	}
	generate := func(spec actionauthor.Spec, token string) (GenerateActionAuthorParametersOutput, error) {
		a := NewGenerateActionAuthorParametersActivity(app)
		a.authorSessionLookup = specSessionLookup(t, session, spec).lookup()
		return a.generate(t.Context(), caller, GenerateActionAuthorParametersInput{
			Scope:     ParameterScopeCandidate,
			PlanToken: token,
		})
	}

	output, err := generate(actionauthor.Spec{}, signTestPlan(t, app, session, plan))
	require.NoError(t, err)
	require.Equal(t, map[string]string{"PIN": "123456"}, output.Values)

	_, err = generate(actionauthor.Spec{}, "")
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "no confirmed plan")

	_, err = generate(actionauthor.Spec{}, "forged.x")
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "plan not signed by Credimi")

	_, err = generate(plan, signTestPlan(t, app, session, plan))
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "this session's plan was fixed when it started")
}

func TestCheckActionAuthorProtocolOutcomeUsesThePlanToken(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	caller := authorTestCaller()
	caller.Namespace = namespace
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "done"})
	}))
	t.Cleanup(server.Close)
	plan := actionauthor.Spec{
		Slot: "other",
		Parameters: []actionauthor.ParameterSpec{{
			Name: "deeplink",
			Source: actionauthor.SourceSpec{
				Kind: actionauthor.SourceHTTP,
				Request: &actionauthor.HTTPSourceRequest{
					Method: http.MethodPost,
					URL:    server.URL + "/sessions",
				},
				ValuePath: "deeplink",
				Status: &actionauthor.StatusSpec{
					URL:     server.URL + "/sessions/1",
					Path:    "status",
					Success: []string{"done"},
				},
			},
		}},
	}
	check := func(spec actionauthor.Spec, token string) (CheckActionAuthorProtocolOutcomeOutput, error) {
		a := NewCheckActionAuthorProtocolOutcomeActivity(app)
		a.authorSessionLookup = specSessionLookup(t, session, spec).lookup()
		a.transport = server.Client().Transport
		return a.check(t.Context(), caller, CheckActionAuthorProtocolOutcomeInput{
			Parameter: "deeplink",
			PlanToken: token,
		})
	}

	output, err := check(actionauthor.Spec{}, signTestPlan(t, app, session, plan))
	require.NoError(t, err)
	require.Equal(t, CheckActionAuthorProtocolOutcomeOutput{
		Reached:  true,
		Status:   "done",
		Expected: []string{"done"},
	}, output)

	_, err = check(actionauthor.Spec{}, "")
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "no confirmed plan")

	_, err = check(actionauthor.Spec{}, "forged.x")
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "plan not signed by Credimi")

	_, err = check(plan, signTestPlan(t, app, session, plan))
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "this session's plan was fixed when it started")
}

func TestSaveAuthoredWalletActionUsesThePlanToken(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	build, err := actionauthor.Resolve(app, namespace, session)
	require.NoError(t, err)
	caller := authorTestCaller()
	caller.Namespace = namespace
	plan := actionauthor.Spec{
		Slot: "other",
		Parameters: []actionauthor.ParameterSpec{{
			Name:   "PIN",
			Source: actionauthor.SourceSpec{Kind: actionauthor.SourceLiteral, Value: "123456"},
		}},
	}
	normalized := plan
	normalized.Outcome = actionauthor.OutcomeSuccess
	code := "appId: eu.wallet\n---\n- launchApp\n- inputText: ${PIN}\n" +
		"- assertVisible: Home\n- takeScreenshot: home\n"
	approval := func(planSHA256 string) string {
		token, err := actionauthor.SignApproval(app, actionauthor.Approval{
			AuthorWorkflowID:   caller.WorkflowID,
			CandidateSHA256:    actionauthor.CodeSHA256(code),
			ApproverID:         session.ActorID,
			ApproverCollection: session.ActorCollection,
			ApprovedAt:         time.Now().Unix(),
			PlanSHA256:         planSHA256,
		})
		require.NoError(t, err)
		return token
	}
	request := func(planToken, approvalToken string) actionauthor.SaveRequest {
		return actionauthor.SaveRequest{
			Name:            "Planned action",
			Code:            code,
			CandidateSHA256: actionauthor.CodeSHA256(code),
			InstallerSHA256: build.InstallerSHA256,
			PackageID:       "eu.wallet",
			ReplayCount:     2,
			Approval:        approvalToken,
			PlanToken:       planToken,
		}
	}
	save := func(spec actionauthor.Spec, payload actionauthor.SaveRequest) (actionauthor.Receipt, error) {
		return specSessionLookup(
			t,
			session,
			spec,
		).activity(t, app).
			save(t.Context(), caller, payload)
	}
	token := signTestPlan(t, app, session, plan)
	approved := approval(actionauthor.PlanSHA256(normalized))

	_, err = save(actionauthor.Spec{}, request("", approved))
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "no confirmed plan")

	_, err = save(actionauthor.Spec{}, request("forged.x", approved))
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "plan not signed by Credimi")

	_, err = save(plan, request(token, approved))
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "this session's plan was fixed when it started")

	_, err = save(actionauthor.Spec{}, request(token, approval(actionauthor.PlanSHA256(plan))))
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "approval is for another plan")

	receipt, err := save(actionauthor.Spec{}, request(token, approved))
	require.NoError(t, err)
	require.True(t, receipt.Created)
	require.Equal(t, "other", receipt.Provenance.Slot)
	require.Equal(t, []actionauthor.ProvenanceParameter{{Name: "PIN", Kind: "literal"}},
		receipt.Provenance.Parameters)
}

func TestListActionAuthorWalletActions(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	caller := authorTestCaller()
	caller.Namespace = namespace
	onboarding := seedAuthorAction(
		t,
		app,
		session,
		"Onboarding",
		"appId: eu.wallet\n---\n- launchApp\n",
	)
	a := NewListActionAuthorWalletActionsActivity(app)
	list := func(category string) (ListActionAuthorWalletActionsOutput, error) {
		a.authorSessionLookup = specSessionLookup(t, session, actionauthor.Spec{}).lookup()
		return a.list(t.Context(), caller, ListActionAuthorWalletActionsInput{Category: category})
	}

	output, err := list("")
	require.NoError(t, err)
	require.Equal(t, []actionauthor.WalletAction{{
		Path:       onboarding,
		Name:       "Onboarding",
		Category:   "onboarding",
		Parameters: []string{},
	}}, output.Actions)

	output, err = list("onboarding")
	require.NoError(t, err)
	require.Len(t, output.Actions, 1)

	output, err = list("verify-credential")
	require.NoError(t, err)
	require.Equal(t, []actionauthor.WalletAction{}, output.Actions)
}

func TestCheckActionAuthorPlan(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	caller := authorTestCaller()
	caller.Namespace = namespace
	onboarding := seedAuthorAction(
		t,
		app,
		session,
		"Onboarding",
		"appId: eu.wallet\n---\n- launchApp\n",
	)
	check := func(start, plan actionauthor.Spec) (CheckActionAuthorPlanOutput, error) {
		a := NewCheckActionAuthorPlanActivity(app)
		a.authorSessionLookup = specSessionLookup(t, session, start).lookup()
		return a.check(t.Context(), caller, CheckActionAuthorPlanInput{Plan: &plan})
	}
	problems := func(problems ...string) CheckActionAuthorPlanOutput {
		return CheckActionAuthorPlanOutput{
			Problems:      problems,
			Summary:       []string{},
			Prerequisites: []actionauthor.ResolvedPrerequisite{},
			Parameters:    []actionauthor.ParameterInfo{},
		}
	}
	resolved := []actionauthor.ResolvedPrerequisite{{
		Action:     onboarding,
		Name:       "Onboarding",
		Code:       "appId: eu.wallet\n---\n- launchApp\n",
		Parameters: actionauthor.ActionParameters("appId: eu.wallet\n---\n- launchApp\n"),
	}}
	plan := actionauthor.Spec{
		Slot:          "get-credential",
		Prerequisites: []actionauthor.PrerequisiteSpec{{Action: "/" + onboarding}},
		Parameters: []actionauthor.ParameterSpec{{
			Name:   "PIN",
			Source: actionauthor.SourceSpec{Kind: actionauthor.SourceLiteral, Value: "123456"},
		}},
	}

	output, err := check(actionauthor.Spec{}, plan)
	require.NoError(t, err)
	normalized := plan
	normalized.Outcome = actionauthor.OutcomeSuccess
	normalized.Prerequisites = []actionauthor.PrerequisiteSpec{{Action: onboarding}}
	accepted := CheckActionAuthorPlanOutput{
		Problems: []string{},
		Plan:     &normalized,
		Summary: []string{
			"Category: get-credential-generic",
			"Outcome: success",
			"Starts from: Onboarding (" + onboarding + ")",
			"${PIN}: the fixed value \"123456\"",
		},
		Prerequisites: resolved,
		Parameters:    actionauthor.ParameterInfos(normalized),
	}
	require.Equal(t, accepted, output)
	require.Len(t, output.Parameters, 1)

	output, err = check(actionauthor.Spec{}, actionauthor.Spec{Slot: "get-credential"})
	require.NoError(t, err)
	require.Equal(t, problems(
		"prerequisites: the get-credential slot needs at least one prerequisite",
	), output)
	encoded, err := json.Marshal(output)
	require.NoError(t, err)
	require.JSONEq(t, `{"problems": ["prerequisites: the get-credential slot needs at least `+
		`one prerequisite"], "plan": null, "summary": [], "prerequisites": [], "parameters": []}`,
		string(encoded))

	missing := plan
	missing.Prerequisites = []actionauthor.PrerequisiteSpec{{Action: namespace + "/x/y"}}
	output, err = check(actionauthor.Spec{}, missing)
	require.NoError(t, err)
	require.Equal(t, problems(namespace+"/x/y not found"), output)

	wait := actionauthor.Spec{
		Slot:          "other",
		Outcome:       actionauthor.OutcomeWait,
		WaitSeconds:   240,
		Prerequisites: []actionauthor.PrerequisiteSpec{{Action: onboarding}},
	}
	output, err = check(actionauthor.Spec{}, wait)
	require.NoError(t, err)
	require.Equal(t, CheckActionAuthorPlanOutput{
		Problems: []string{},
		Plan:     &wait,
		Summary: []string{
			"Category: other",
			"Outcome: waits 240 s",
			"Starts from: Onboarding (" + onboarding + ")",
		},
		Prerequisites: resolved,
		Parameters:    []actionauthor.ParameterInfo{},
	}, output)

	wait.WaitSeconds = 0
	output, err = check(actionauthor.Spec{}, wait)
	require.NoError(t, err)
	require.Equal(t, problems("wait_seconds: must be 1-600 for the wait outcome"), output)

	output, err = check(actionauthor.Spec{Slot: "onboarding"}, plan)
	require.NoError(t, err)
	require.Equal(t, problems("this session's plan was fixed when it started"), output)

	// A confirmed plan's token: Credimi verifies it and returns what the agent applies.
	verify := func(start actionauthor.Spec, input CheckActionAuthorPlanInput) (
		CheckActionAuthorPlanOutput, error,
	) {
		a := NewCheckActionAuthorPlanActivity(app)
		a.authorSessionLookup = specSessionLookup(t, session, start).lookup()
		return a.check(t.Context(), caller, input)
	}
	token := signTestPlan(t, app, session, plan)
	output, err = verify(actionauthor.Spec{}, CheckActionAuthorPlanInput{PlanToken: token})
	require.NoError(t, err)
	require.Equal(t, accepted, output)

	other := session
	other.ActorID = "someone-else"
	for name, tc := range map[string]struct {
		start   actionauthor.Spec
		token   string
		code    string
		message string
	}{
		"forged": {
			token:   "forged.x",
			code:    errorcodes.RecordNotAccessible,
			message: "plan not signed by Credimi",
		},
		"another user": {
			token:   signTestPlan(t, app, other, plan),
			code:    errorcodes.RecordNotAccessible,
			message: "plan confirmed by another user",
		},
		"start spec": {
			start:   actionauthor.Spec{Slot: "onboarding"},
			token:   token,
			code:    errorcodes.MissingOrInvalidPayload,
			message: "this session's plan was fixed when it started",
		},
	} {
		_, err := verify(tc.start, CheckActionAuthorPlanInput{PlanToken: tc.token})
		requireActivityError(t, err, tc.code, true)
		require.ErrorContains(t, err, tc.message, name)
	}

	for name, input := range map[string]CheckActionAuthorPlanInput{
		"both":    {Plan: &plan, PlanToken: token},
		"neither": {},
	} {
		_, err := verify(actionauthor.Spec{}, input)
		requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
		require.ErrorContains(t, err, "send either plan or plan_token", name)
	}
}

func TestResolveActionAuthorReference(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	caller := authorTestCaller()
	caller.Namespace = namespace
	credential := seedDynamicCredential(t, app, session)
	offer, err := json.Marshal(map[string]any{
		"credential_issuer":            "https://issuer.example.test",
		"credential_configuration_ids": []string{"Authoring PID"},
	})
	require.NoError(t, err)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(offer)
	}))
	t.Cleanup(server.Close)
	a := NewResolveActionAuthorReferenceActivity(app)
	a.authorSessionLookup = specSessionLookup(t, session, actionauthor.Spec{}).lookup()
	a.transport = server.Client().Transport

	resolution, err := a.resolve(t.Context(), caller, ResolveActionAuthorReferenceInput{
		Text: "openid-credential-offer://?" +
			url.Values{"credential_offer_uri": {server.URL + "/offer"}}.Encode(),
	})
	require.NoError(t, err)
	require.Equal(t, actionauthor.ReferenceResolution{
		Kind: actionauthor.ReferenceCredentialOffer,
		Sources: []actionauthor.ReferenceSource{{
			Label: "Authoring PID",
			Source: actionauthor.SourceSpec{
				Kind:       actionauthor.SourceCredentialOffer,
				Identifier: credential,
			},
		}},
		Decoded: map[string]string{
			"credential_issuer":            "https://issuer.example.test",
			"credential_configuration_ids": "Authoring PID",
		},
	}, resolution)
}

func TestActionAuthorPlanActivitiesRejectCallersOutsideASession(t *testing.T) {
	for name, a := range map[string]workflowengine.ExecutableActivity{
		"list":      NewListActionAuthorWalletActionsActivity(nil),
		"check":     NewCheckActionAuthorPlanActivity(nil),
		"reference": NewResolveActionAuthorReferenceActivity(nil),
	} {
		// The test environment's caller is a workflow of another type.
		_, err := executeActivity(t, a, map[string]any{})
		requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
		require.ErrorContains(t, err, "caller is not an action author session agent", name)

		_, err = executeActivity(t, a, "not a request")
		requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
	}
}
