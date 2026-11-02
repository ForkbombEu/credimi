// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func planBody(namespace string, plan actionauthor.Spec, accepted bool) string {
	encoded, err := json.Marshal(plan)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(
		`{"namespace":%q,"author_workflow_id":%q,"tool_id":"plan-review-1","plan":%s,"accepted":%t}`,
		namespace,
		reviewAuthorWorkflow,
		encoded,
		accepted,
	)
}

// handlerPlan starts from the handler wallet's onboarding.
func handlerPlan() actionauthor.Spec {
	return actionauthor.Spec{
		Slot:          "get-credential",
		Prerequisites: []actionauthor.PrerequisiteSpec{{Action: "/" + authoringOnboardingAction}},
		Parameters: []actionauthor.ParameterSpec{{
			Name: "deeplink",
			Source: actionauthor.SourceSpec{
				Kind: actionauthor.SourceHTTP,
				Request: &actionauthor.HTTPSourceRequest{
					Method: http.MethodPost,
					URL:    "https://issuer.example.test/sessions",
				},
				ValuePath: "deeplink",
			},
		}},
	}
}

func TestActionAuthorPlanSignsConfirmedPlans(t *testing.T) {
	user, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	stub := stubActionAuthorReview(t, user.Id, nil, nil)
	stub.session.Spec = actionauthor.Spec{}
	(&tests.ApiScenario{
		Name:            "accepted",
		Method:          http.MethodPost,
		URL:             "/api/action-author/plans",
		Body:            strings.NewReader(planBody(authoringNamespace, handlerPlan(), true)),
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"accepted":true`, `"tool_id":"plan-review-1"`},
		TestAppFactory:  setupActionAuthorSpecApp,
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			require.Equal(t, []string{authoringNamespace}, stub.namespaces)
			require.Equal(t, reviewAuthorWorkflow, stub.workflowID)
			require.Equal(t, "plan-review-1", stub.toolID)
			require.Equal(t, true, stub.result["accepted"])
			normalized := handlerPlan()
			normalized.Outcome = actionauthor.OutcomeSuccess
			normalized.Prerequisites[0].Action = authoringOnboardingAction
			require.Equal(t, normalized, stub.result["plan"])
			require.Equal(t, []actionauthor.ResolvedPrerequisite{{
				Action:     authoringOnboardingAction,
				Name:       "Onboarding",
				Code:       "appId: eu.wallet\n---\n- launchApp\n",
				Parameters: []string{},
			}}, stub.result["prerequisites"])
			require.Equal(t, []actionauthor.ParameterInfo{{
				Name:        "deeplink",
				Kind:        actionauthor.SourceHTTP,
				Description: "POST https://issuer.example.test/sessions → deeplink",
			}}, stub.result["parameters"])
			token, _ := stub.result["plan_token"].(string)
			grant, err := actionauthor.VerifyPlan(app, token, reviewAuthorWorkflow)
			require.NoError(t, err)
			require.Equal(t, normalized, grant.Plan)
			require.Equal(t, user.Id, grant.ApproverID)
			require.Equal(t, "users", grant.ApproverCollection)
		},
	}).Test(t)

	stub = stubActionAuthorReview(t, user.Id, nil, nil)
	stub.session.Spec = actionauthor.Spec{}
	(&tests.ApiScenario{
		Name:            "refused",
		Method:          http.MethodPost,
		URL:             "/api/action-author/plans",
		Body:            strings.NewReader(planBody(authoringNamespace, actionauthor.Spec{}, false)),
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"accepted":false`},
		TestAppFactory:  setupActionAuthorSpecApp,
	}).Test(t)
	require.Equal(t, map[string]any{
		"accepted":      false,
		"plan_token":    "",
		"plan":          nil,
		"prerequisites": []actionauthor.ResolvedPrerequisite{},
		"parameters":    []actionauthor.ParameterInfo{},
	}, stub.result)
}

func TestActionAuthorPlanRejectsOtherUsersFixedAndInvalidPlans(t *testing.T) {
	user, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	missingPrerequisite := handlerPlan()
	missingPrerequisite.Prerequisites = nil
	for _, tc := range []struct {
		name      string
		user      string
		actorID   string
		fixed     bool
		plan      actionauthor.Spec
		updateErr error
		status    int
		expected  []string
		delivered bool
	}{
		{
			name:     "organization the caller does not own",
			user:     "userB",
			status:   http.StatusNotFound,
			expected: []string{`"session_not_found"`},
		},
		{
			name:    "another user's session",
			user:    "userA",
			actorID: "another-user",
			status:  http.StatusForbidden,
			expected: []string{
				`"forbidden"`,
				"only the user who started this session can confirm its plans",
			},
		},
		{
			name:   "plan fixed at start",
			user:   "userA",
			fixed:  true,
			plan:   handlerPlan(),
			status: http.StatusConflict,
			expected: []string{
				`"plan_fixed"`,
				"this session's plan was fixed when it started",
			},
		},
		{
			name:   "invalid plan",
			user:   "userA",
			plan:   missingPrerequisite,
			status: http.StatusBadRequest,
			expected: []string{
				`"invalid_request"`,
				"the get-credential slot needs at least one prerequisite",
			},
		},
		{
			name:      "settled plan card",
			user:      "userA",
			plan:      handlerPlan(),
			updateErr: temporal.NewApplicationError("unknown tool id", "ValueError"),
			status:    http.StatusConflict,
			expected: []string{
				`"plan_not_pending"`,
				"this plan is no longer pending; reload the chat",
			},
			delivered: true,
		},
	} {
		actorID := tc.actorID
		if actorID == "" {
			actorID = user.Id
		}
		stub := stubActionAuthorReview(t, actorID, nil, tc.updateErr)
		if !tc.fixed {
			stub.session.Spec = actionauthor.Spec{}
		}
		(&tests.ApiScenario{
			Name:            tc.name,
			Method:          http.MethodPost,
			URL:             "/api/action-author/plans",
			Body:            strings.NewReader(planBody(authoringNamespace, tc.plan, true)),
			Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, tc.user)},
			ExpectedStatus:  tc.status,
			ExpectedContent: tc.expected,
			TestAppFactory:  setupActionAuthorSpecApp,
		}).Test(t)
		require.Equal(t, tc.delivered, stub.workflowID != "", tc.name)
	}
}
