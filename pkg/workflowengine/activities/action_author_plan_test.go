// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

// seedAuthorAction adds an action to the wallet of the session's build and
// returns its canonified path.
func seedAuthorAction(
	t *testing.T,
	app core.App,
	session actionauthor.BuildRequest,
	name, code string,
) string {
	t.Helper()
	selection, err := actionauthor.Select(app, "", session)
	require.NoError(t, err)
	collection, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("owner", selection.Organization.Id)
	record.Set("wallet", selection.Wallet.Id)
	record.Set("name", name)
	record.Set("code", code)
	record.Set("category", "onboarding")
	require.NoError(t, app.Save(record))
	return selection.Organization.GetString("canonified_name") + "/" +
		selection.Wallet.GetString("canonified_name") + "/" + record.GetString("canonified_name")
}

// recordStartsFrom stores an authoring provenance saying the action at path
// action was saved to start from prerequisites, in order.
func recordStartsFrom(t *testing.T, app core.App, action string, prerequisites ...string) {
	t.Helper()
	record, err := app.FindFirstRecordByData("wallet_actions", "canonified_name", path.Base(action))
	require.NoError(t, err)
	provenance, err := json.Marshal(actionauthor.Provenance{
		WorkflowID:    "authored",
		YAMLSHA256:    actionauthor.CodeSHA256(record.GetString("code")),
		Prerequisites: prerequisites,
	})
	require.NoError(t, err)
	record.Set("authoring_provenance", string(provenance))
	require.NoError(t, app.Save(record))
}

// seedForeignAction adds an unpublished action of userB's organization and
// returns its canonified path, which userA's sessions cannot start from.
func seedForeignAction(t *testing.T, app core.App) string {
	t.Helper()
	org, err := app.FindFirstRecordByData("organizations", "name", "userB's organization")
	require.NoError(t, err)
	save := func(collection string, fields map[string]any) *core.Record {
		c, err := app.FindCollectionByNameOrId(collection)
		require.NoError(t, err)
		record := core.NewRecord(c)
		for key, value := range fields {
			record.Set(key, value)
		}
		require.NoError(t, app.Save(record))
		return record
	}
	wallet := save("wallets", map[string]any{"owner": org.Id, "name": "Foreign wallet"})
	action := save("wallet_actions", map[string]any{
		"owner":    org.Id,
		"wallet":   wallet.Id,
		"name":     "Foreign onboarding",
		"code":     "appId: eu.wallet\n---\n- launchApp\n",
		"category": "onboarding",
	})
	return org.GetString("canonified_name") + "/" + wallet.GetString("canonified_name") + "/" +
		action.GetString("canonified_name")
}

// foreignPlan is plan starting from an action the session's organization
// cannot use.
func foreignPlan(t *testing.T, app core.App, plan actionauthor.Spec) actionauthor.Spec {
	t.Helper()
	foreign := seedForeignAction(t, app)
	plan.Prerequisites = []actionauthor.PrerequisiteSpec{{Action: foreign}}
	return plan
}

func TestGenerateActionAuthorParametersChecksTheSentPlan(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	caller := authorTestCaller()
	caller.Namespace = namespace
	plan := actionauthor.Spec{
		Slot: "other",
		Parameters: []actionauthor.ParameterSpec{{
			Name:   "PIN",
			Source: actionauthor.SourceSpec{Kind: actionauthor.SourceLiteral, Value: "123456"},
		}},
	}
	generate := func(plan actionauthor.Spec) (GenerateActionAuthorParametersOutput, error) {
		a := NewGenerateActionAuthorParametersActivity(app)
		a.authorSessionLookup = buildSessionLookup(t, app, session).lookup()
		return a.generate(t.Context(), caller, GenerateActionAuthorParametersInput{
			Scope: ParameterScopeCandidate,
			Plan:  plan,
		})
	}

	output, err := generate(plan)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"PIN": "123456"}, output.Values)

	_, err = generate(actionauthor.Spec{})
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "no plan: describe the action with describe_action")

	foreign := foreignPlan(t, app, plan)
	_, err = generate(foreign)
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, foreign.Prerequisites[0].Action+" not found")
}

func TestCheckActionAuthorProtocolOutcomeChecksTheSentPlan(t *testing.T) {
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
	check := func(plan actionauthor.Spec) (CheckActionAuthorProtocolOutcomeOutput, error) {
		a := NewCheckActionAuthorProtocolOutcomeActivity(app)
		a.authorSessionLookup = buildSessionLookup(t, app, session).lookup()
		a.transport = server.Client().Transport
		return a.check(t.Context(), caller, CheckActionAuthorProtocolOutcomeInput{
			Parameter: "deeplink",
			Plan:      plan,
		})
	}

	output, err := check(plan)
	require.NoError(t, err)
	require.Equal(t, CheckActionAuthorProtocolOutcomeOutput{
		Reached:  true,
		Status:   "done",
		Expected: []string{"done"},
	}, output)

	_, err = check(actionauthor.Spec{})
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "no plan: describe the action with describe_action")

	foreign := foreignPlan(t, app, plan)
	_, err = check(foreign)
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, foreign.Prerequisites[0].Action+" not found")
}

func TestSaveAuthoredWalletActionChecksTheSentPlan(t *testing.T) {
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
			AuthorWorkflowID: caller.WorkflowID,
			CandidateSHA256:  actionauthor.CodeSHA256(code),
			ApproverID:       session.ActorID,
			ApprovedAt:       time.Now().Unix(),
			PlanSHA256:       planSHA256,
		})
		require.NoError(t, err)
		return token
	}
	request := func(plan actionauthor.Spec, approvalToken string) actionauthor.SaveRequest {
		return actionauthor.SaveRequest{
			Name:            "Planned action",
			Code:            code,
			CandidateSHA256: actionauthor.CodeSHA256(code),
			InstallerSHA256: build.InstallerSHA256,
			PackageID:       "eu.wallet",
			ReplayCount:     1,
			Approval:        approvalToken,
			Spec:            plan,
		}
	}
	save := func(payload actionauthor.SaveRequest) (actionauthor.Receipt, error) {
		return buildSessionLookup(t, app, session).activity(t, app).
			save(t.Context(), caller, payload)
	}
	approved := approval(actionauthor.PlanSHA256(normalized))

	_, err = save(request(actionauthor.Spec{}, approved))
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "no plan: describe the action with describe_action")

	foreign := foreignPlan(t, app, plan)
	_, err = save(request(foreign, approved))
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, foreign.Prerequisites[0].Action+" not found")

	_, err = save(request(plan, approval(actionauthor.PlanSHA256(plan))))
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "approval is for another plan")

	receipt, err := save(request(plan, approved))
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
	getPID := seedAuthorAction(t, app, session, "Get PID", "appId: eu.wallet\n---\n- back\n")
	recordStartsFrom(t, app, getPID, onboarding)
	a := NewListActionAuthorWalletActionsActivity(app)
	list := func(category string) (ListActionAuthorWalletActionsOutput, error) {
		a.authorSessionLookup = buildSessionLookup(t, app, session).lookup()
		return a.list(t.Context(), caller, ListActionAuthorWalletActionsInput{Category: category})
	}

	output, err := list("")
	require.NoError(t, err)
	require.Equal(t, []actionauthor.WalletAction{{
		Path:       getPID,
		Name:       "Get PID",
		Category:   "onboarding",
		Parameters: []string{},
		StartsFrom: []string{onboarding},
	}, {
		Path:       onboarding,
		Name:       "Onboarding",
		Category:   "onboarding",
		Parameters: []string{},
		StartsFrom: []string{},
	}}, output.Actions)
	encoded, err := json.Marshal(output.Actions[1])
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"starts_from":[]`)

	output, err = list("onboarding")
	require.NoError(t, err)
	require.Len(t, output.Actions, 2)

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
	check := func(plan actionauthor.Spec) (CheckActionAuthorPlanOutput, error) {
		a := NewCheckActionAuthorPlanActivity(app)
		a.authorSessionLookup = buildSessionLookup(t, app, session).lookup()
		return a.check(t.Context(), caller, CheckActionAuthorPlanInput{Plan: plan})
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

	output, err := check(plan)
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

	output, err = check(actionauthor.Spec{Slot: "get-credential"})
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
	output, err = check(missing)
	require.NoError(t, err)
	require.Equal(t, problems(namespace+"/x/y not found"), output)

	getPID := seedAuthorAction(t, app, session, "Get PID", "appId: eu.wallet\n---\n- back\n")
	recordStartsFrom(t, app, getPID, onboarding)
	unchained := plan
	unchained.Prerequisites = []actionauthor.PrerequisiteSpec{{Action: getPID}}
	output, err = check(unchained)
	require.NoError(t, err)
	require.Equal(t, problems(
		"prerequisite "+getPID+" starts from "+onboarding+": list it before "+getPID,
	), output)

	wait := actionauthor.Spec{
		Slot:          "other",
		Outcome:       actionauthor.OutcomeWait,
		WaitSeconds:   240,
		Prerequisites: []actionauthor.PrerequisiteSpec{{Action: onboarding}},
	}
	output, err = check(wait)
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
	output, err = check(wait)
	require.NoError(t, err)
	require.Equal(t, problems("wait_seconds: must be 1-600 for the wait outcome"), output)
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
	a.authorSessionLookup = buildSessionLookup(t, app, session).lookup()
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
