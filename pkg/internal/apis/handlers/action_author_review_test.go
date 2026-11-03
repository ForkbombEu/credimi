// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

const reviewAuthorWorkflow = "ActionAuthorSession-ticket-1-author"

var reviewCandidate = strings.Repeat("ab", 32)

// reviewStub records the decision delivered to the agent workflow; session
// is the running session the lookup answers.
type reviewStub struct {
	namespaces []string
	workflowID string
	toolID     string
	result     map[string]any
	session    actionauthor.Session
}

// stubActionAuthorReview answers the session lookup with an Android session
// of the handler wallet started by actorID with an onboarding spec, and the
// update with updateErr.
func stubActionAuthorReview(t *testing.T, actorID string, lookupErr, updateErr error) *reviewStub {
	t.Helper()
	stub := &reviewStub{session: actionauthor.Session{
		BuildRequest: actionauthor.BuildRequest{
			Actor:                   actionauthor.Actor{ActorID: actorID, ActorCollection: "users"},
			WalletVersionIdentifier: authoringAndroidVersion,
			Platform:                actionauthor.PlatformAndroid,
		},
		Spec: actionauthor.Spec{Slot: "onboarding"},
	}}
	origClient := queueTemporalClient
	origLookup := lookupActionAuthorSession
	origUpdate := updateActionAuthorWorkflow
	t.Cleanup(func() {
		queueTemporalClient = origClient
		lookupActionAuthorSession = origLookup
		updateActionAuthorWorkflow = origUpdate
	})
	queueTemporalClient = func(namespace string) (client.Client, error) {
		stub.namespaces = append(stub.namespaces, namespace)
		return nil, nil
	}
	lookupActionAuthorSession = func(
		_ context.Context,
		_ client.Client,
		authorWorkflowID, authorRunID string,
	) (activities.ActionAuthorSessionRef, error) {
		require.Equal(t, reviewAuthorWorkflow, authorWorkflowID)
		require.Empty(t, authorRunID)
		return activities.ActionAuthorSessionRef{Session: stub.session}, lookupErr
	}
	updateActionAuthorWorkflow = func(
		_ context.Context,
		_ client.Client,
		workflowID, toolID string,
		result map[string]any,
	) error {
		stub.workflowID, stub.toolID, stub.result = workflowID, toolID, result
		return updateErr
	}
	return stub
}

func reviewBody(namespace string, accepted bool) string {
	return fmt.Sprintf(
		`{"namespace":%q,"author_workflow_id":%q,"tool_id":"tool-1","candidate_sha256":%q,"accepted":%t}`,
		namespace,
		reviewAuthorWorkflow,
		reviewCandidate,
		accepted,
	)
}

func TestActionAuthorReviewSignsAcceptedCandidates(t *testing.T) {
	user, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	stub := stubActionAuthorReview(t, user.Id, nil, nil)
	(&tests.ApiScenario{
		Name:            "accepted",
		Method:          http.MethodPost,
		URL:             "/api/action-author/reviews",
		Body:            strings.NewReader(reviewBody(authoringNamespace, true)),
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"accepted":true`, `"tool_id":"tool-1"`},
		TestAppFactory:  setupActionAuthorApp,
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			require.Equal(t, []string{authoringNamespace}, stub.namespaces)
			require.Equal(t, reviewAuthorWorkflow, stub.workflowID)
			require.Equal(t, "tool-1", stub.toolID)
			require.Equal(t, true, stub.result["accepted"])
			token, _ := stub.result["approval"].(string)
			approval, err := actionauthor.VerifyApproval(
				app, token, reviewAuthorWorkflow, reviewCandidate, time.Now(),
			)
			require.NoError(t, err)
			require.Equal(t, user.Id, approval.ApproverID)
			require.Equal(t, "users", approval.ApproverCollection)
			require.Equal(t, actionauthor.PlanSHA256(actionauthor.Spec{
				Slot:    "onboarding",
				Outcome: actionauthor.OutcomeSuccess,
			}), approval.PlanSHA256)
		},
	}).Test(t)

	stub = stubActionAuthorReview(t, user.Id, nil, nil)
	(&tests.ApiScenario{
		Name:            "refused",
		Method:          http.MethodPost,
		URL:             "/api/action-author/reviews",
		Body:            strings.NewReader(reviewBody(authoringNamespace, false)),
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"accepted":false`},
		TestAppFactory:  setupActionAuthorApp,
	}).Test(t)
	require.Equal(t, map[string]any{"accepted": false, "approval": ""}, stub.result)
}

func TestActionAuthorReviewRejectsOtherUsersAndSettledReviews(t *testing.T) {
	user, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	for _, tc := range []struct {
		name      string
		user      string
		namespace string
		actorID   string
		lookupErr error
		updateErr error
		status    int
		expected  []string
		delivered bool
	}{
		{
			name:      "organization the caller does not own",
			user:      "userB",
			namespace: authoringNamespace,
			status:    http.StatusNotFound,
			expected:  []string{`"session_not_found"`},
		},
		{
			name:      "unknown namespace",
			user:      "userA",
			namespace: "missing-organization",
			status:    http.StatusNotFound,
			expected:  []string{`"session_not_found"`},
		},
		{
			name:      "not a running session agent",
			user:      "userA",
			namespace: authoringNamespace,
			lookupErr: fmt.Errorf("%w: action author session is not running", actionauthor.ErrForbidden),
			status:    http.StatusNotFound,
			expected:  []string{`"session_not_found"`},
		},
		{
			name:      "Temporal unavailable",
			user:      "userA",
			namespace: authoringNamespace,
			lookupErr: errors.New("dial failed"),
			status:    http.StatusServiceUnavailable,
			expected:  []string{`"temporal_unavailable"`},
		},
		{
			name:      "another user's session",
			user:      "userA",
			namespace: authoringNamespace,
			actorID:   "another-user",
			status:    http.StatusForbidden,
			expected: []string{
				`"forbidden"`,
				"only the user who started this session can review its actions",
			},
		},
		{
			name:      "settled review",
			user:      "userA",
			namespace: authoringNamespace,
			updateErr: temporal.NewApplicationError("unknown tool id", "ValueError"),
			status:    http.StatusConflict,
			expected: []string{
				`"review_not_pending"`,
				"this save review is no longer pending; reload the chat",
			},
			delivered: true,
		},
	} {
		actorID := tc.actorID
		if actorID == "" {
			actorID = user.Id
		}
		stub := stubActionAuthorReview(t, actorID, tc.lookupErr, tc.updateErr)
		(&tests.ApiScenario{
			Name:            tc.name,
			Method:          http.MethodPost,
			URL:             "/api/action-author/reviews",
			Body:            strings.NewReader(reviewBody(tc.namespace, true)),
			Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, tc.user)},
			ExpectedStatus:  tc.status,
			ExpectedContent: tc.expected,
			TestAppFactory:  setupActionAuthorApp,
		}).Test(t)
		require.Equal(t, tc.delivered, stub.workflowID != "", tc.name)
	}

	(&tests.ApiScenario{
		Name:   "invalid candidate digest",
		Method: http.MethodPost,
		URL:    "/api/action-author/reviews",
		Body: strings.NewReader(
			`{"namespace":"x","author_workflow_id":"y","tool_id":"z","candidate_sha256":"short","accepted":true}`,
		),
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusBadRequest,
		ExpectedContent: []string{"CandidateSHA256"},
		TestAppFactory:  setupActionAuthorApp,
	}).Test(t)
}

func reviewBodyWithPlan(token string) string {
	return fmt.Sprintf(
		`{"namespace":%q,"author_workflow_id":%q,"tool_id":"tool-1","candidate_sha256":%q,"accepted":true,"plan_token":%q}`,
		authoringNamespace,
		reviewAuthorWorkflow,
		reviewCandidate,
		token,
	)
}

// signHandlerPlan signs a plan of the review workflow confirmed by userID; the
// test apps share test_pb_data's signing secret.
func signHandlerPlan(t *testing.T, userID string, plan actionauthor.Spec) string {
	t.Helper()
	app := setupActionAuthorApp(t)
	defer app.Cleanup()
	token, err := actionauthor.SignPlan(app, actionauthor.PlanGrant{
		AuthorWorkflowID:   reviewAuthorWorkflow,
		Plan:               plan,
		ApproverID:         userID,
		ApproverCollection: "users",
		ConfirmedAt:        time.Now().Unix(),
	})
	require.NoError(t, err)
	return token
}

func TestActionAuthorReviewBindsThePlan(t *testing.T) {
	user, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	plan := actionauthor.Spec{
		Slot: "other",
		Parameters: []actionauthor.ParameterSpec{{
			Name:   "PIN",
			Source: actionauthor.SourceSpec{Kind: actionauthor.SourceLiteral, Value: "123456"},
		}},
	}
	token := signHandlerPlan(t, user.Id, plan)

	stub := stubActionAuthorReview(t, user.Id, nil, nil)
	stub.session.Spec = actionauthor.Spec{}
	(&tests.ApiScenario{
		Name:            "accepted with the confirmed plan",
		Method:          http.MethodPost,
		URL:             "/api/action-author/reviews",
		Body:            strings.NewReader(reviewBodyWithPlan(token)),
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"accepted":true`},
		TestAppFactory:  setupActionAuthorApp,
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			approvalToken, _ := stub.result["approval"].(string)
			approval, err := actionauthor.VerifyApproval(
				app, approvalToken, reviewAuthorWorkflow, reviewCandidate, time.Now(),
			)
			require.NoError(t, err)
			normalized := plan
			normalized.Outcome = actionauthor.OutcomeSuccess
			require.Equal(t, actionauthor.PlanSHA256(normalized), approval.PlanSHA256)
		},
	}).Test(t)

	for _, tc := range []struct {
		name     string
		fixed    bool
		token    string
		status   int
		expected []string
	}{
		{
			name:     "no confirmed plan",
			status:   http.StatusBadRequest,
			expected: []string{`"invalid_request"`, "no confirmed plan"},
		},
		{
			name:     "forged plan",
			token:    "forged.x",
			status:   http.StatusForbidden,
			expected: []string{`"forbidden"`},
		},
		{
			name:     "plan token for a fixed plan",
			fixed:    true,
			token:    token,
			status:   http.StatusBadRequest,
			expected: []string{`"invalid_request"`, "this session's plan was fixed when it started"},
		},
	} {
		stub := stubActionAuthorReview(t, user.Id, nil, nil)
		if !tc.fixed {
			stub.session.Spec = actionauthor.Spec{}
		}
		(&tests.ApiScenario{
			Name:            tc.name,
			Method:          http.MethodPost,
			URL:             "/api/action-author/reviews",
			Body:            strings.NewReader(reviewBodyWithPlan(tc.token)),
			Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
			ExpectedStatus:  tc.status,
			ExpectedContent: tc.expected,
			TestAppFactory:  setupActionAuthorApp,
		}).Test(t)
		require.Empty(t, stub.workflowID, tc.name)
	}

	// A refusal signs nothing: an unreadable token still delivers it.
	stub = stubActionAuthorReview(t, user.Id, nil, nil)
	stub.session.Spec = actionauthor.Spec{}
	(&tests.ApiScenario{
		Name:   "refused with a bad plan token",
		Method: http.MethodPost,
		URL:    "/api/action-author/reviews",
		Body: strings.NewReader(strings.Replace(
			reviewBodyWithPlan("forged.x"), `"accepted":true`, `"accepted":false`, 1,
		)),
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"accepted":false`, `"tool_id":"tool-1"`},
		TestAppFactory:  setupActionAuthorApp,
	}).Test(t)
	require.Equal(t, reviewAuthorWorkflow, stub.workflowID)
	require.Equal(t, map[string]any{"accepted": false, "approval": ""}, stub.result)
}
