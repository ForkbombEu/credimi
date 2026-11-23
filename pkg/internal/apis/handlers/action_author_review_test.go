// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

const (
	reviewSessionWorkflow = authoringSessionID
	reviewAuthorWorkflow  = reviewSessionWorkflow + "-author"
)

var (
	reviewSHA1 = strings.Repeat("ab", 32)
	reviewSHA2 = strings.Repeat("cd", 32)
)

// reviewStub records the decision delivered to the agent workflow; session
// is the running session the lookup answers, started for organization.
type reviewStub struct {
	namespaces   []string
	workflowID   string
	toolID       string
	result       map[string]any
	session      actionauthor.BuildRequest
	organization string
	// mode is the session's mode; fcaf authors the relying-party suite.
	mode string
}

// stubActionAuthorReview answers the session lookup with an Android session
// of the handler wallet started by actorID for userA's organization (or
// stub.organization), and the update with updateErr.
func stubActionAuthorReview(t *testing.T, actorID string, lookupErr, updateErr error) *reviewStub {
	t.Helper()
	stub := &reviewStub{
		session: actionauthor.BuildRequest{
			Actor:                   actionauthor.Actor{ActorID: actorID},
			WalletVersionIdentifier: authoringAndroidVersion,
			Platform:                actionauthor.PlatformAndroid,
		},
		organization: authoringNamespace,
	}
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
		app core.App,
		_ client.Client,
		namespace, authorWorkflowID, authorRunID string,
	) (activities.ActionAuthorSessionRef, error) {
		require.NotNil(t, app)
		require.Equal(t, stub.namespaces[len(stub.namespaces)-1], namespace)
		require.Equal(t, reviewAuthorWorkflow, authorWorkflowID)
		require.Empty(t, authorRunID)
		ref := activities.ActionAuthorSessionRef{
			Namespace:    namespace,
			Organization: stub.organization,
			Session:      stub.session,
			Mode:         stub.mode,
		}
		if stub.mode == actionauthor.SessionModeFCAF {
			ref.FCAFSuite = actionauthor.FCAFRelyingPartySuite
		}
		return ref, lookupErr
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

// reviewBody sends decisions on the pending review tool-1.
func reviewBody(t testing.TB, decisions ...ActionAuthorReviewDecision) string {
	t.Helper()
	encoded, err := json.Marshal(ActionAuthorReviewRequest{
		AuthorWorkflowID: reviewAuthorWorkflow,
		ToolID:           "tool-1",
		Decisions:        decisions,
	})
	require.NoError(t, err)
	return string(encoded)
}

// reviewDecision is a decision on key, whose code hashes to sha.
func reviewDecision(
	key, sha, decision string,
	plan actionauthor.Spec,
	comment string,
) ActionAuthorReviewDecision {
	return ActionAuthorReviewDecision{
		Key:             key,
		Name:            "Action " + key,
		CandidateSHA256: sha,
		Decision:        decision,
		Comment:         comment,
		Plan:            plan,
	}
}

// tooManyReviewDecisions returns n refusals of distinct keys.
func tooManyReviewDecisions(n int, plan actionauthor.Spec) []ActionAuthorReviewDecision {
	decisions := make([]ActionAuthorReviewDecision, 0, n)
	for i := range n {
		decisions = append(
			decisions,
			reviewDecision(fmt.Sprintf("a%d", i+1), reviewSHA1, "refuse", plan, ""),
		)
	}
	return decisions
}

// deliveredDecisions returns the decisions the stub delivered to the agent.
func deliveredDecisions(t testing.TB, stub *reviewStub) []map[string]any {
	t.Helper()
	decisions, ok := stub.result["decisions"].([]map[string]any)
	require.True(t, ok, "delivered result: %v", stub.result)
	return decisions
}

// requireApproval checks that decision carries the caller's approval of sha,
// of plan once normalized and of the decision's key.
func requireApproval(
	t testing.TB,
	app core.App,
	decision map[string]any,
	sha string,
	plan actionauthor.Spec,
	approverID string,
) {
	t.Helper()
	token, _ := decision["approval"].(string)
	approval, err := actionauthor.VerifyApproval(app, token, reviewAuthorWorkflow, sha, time.Now())
	require.NoError(t, err)
	require.Equal(t, approverID, approval.ApproverID)
	require.NoError(t, plan.Normalize())
	require.Equal(t, actionauthor.PlanSHA256(plan), approval.PlanSHA256)
	require.Equal(t, decision["key"], approval.Key)
}

func TestActionAuthorReviewDecisions(t *testing.T) {
	user, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	onboarding := actionauthor.Spec{Slot: "onboarding"}
	pinPlan := func() actionauthor.Spec {
		return actionauthor.Spec{
			Slot:          "other",
			Prerequisites: []actionauthor.PrerequisiteSpec{{Candidate: "a1"}},
			Parameters: []actionauthor.ParameterSpec{{
				Name: "PIN",
				Source: actionauthor.SourceSpec{
					Kind:  actionauthor.SourceLiteral,
					Value: "123456",
				},
			}},
		}
	}
	for _, tc := range []struct {
		name      string
		user      string
		actorID   string
		updateErr error
		app       func(testing.TB) *tests.TestApp
		body      string
		status    int
		expected  []string
		delivered bool
		check     func(t testing.TB, app *tests.TestApp, stub *reviewStub)
	}{
		{
			name: "accept two actions",
			body: reviewBody(t,
				reviewDecision("a1", reviewSHA1, "accept", onboarding, ""),
				reviewDecision("a2", reviewSHA2, "accept", pinPlan(), ""),
			),
			status: http.StatusOK,
			expected: []string{
				`"tool_id":"tool-1"`,
				`"decisions":[{"key":"a1","decision":"accept"},{"key":"a2","decision":"accept"}]`,
			},
			delivered: true,
			check: func(t testing.TB, app *tests.TestApp, stub *reviewStub) {
				// The session is looked up in the runner namespace it runs in.
				require.Equal(t, []string{authoringSessionNamespace}, stub.namespaces)
				require.Equal(t, "tool-1", stub.toolID)
				decisions := deliveredDecisions(t, stub)
				require.Len(t, decisions, 2)
				require.Equal(t, "a1", decisions[0]["key"])
				require.Equal(t, "Action a1", decisions[0]["name"])
				require.Equal(t, "accept", decisions[0]["decision"])
				require.Empty(t, decisions[0]["comment"])
				requireApproval(t, app, decisions[0], reviewSHA1, onboarding, user.Id)
				require.Equal(t, "a2", decisions[1]["key"])
				requireApproval(t, app, decisions[1], reviewSHA2, pinPlan(), user.Id)
			},
		},
		{
			name: "refuse, then change with a comment",
			body: reviewBody(t,
				reviewDecision("a1", reviewSHA1, "refuse", actionauthor.Spec{}, "ignored"),
				reviewDecision("a2", reviewSHA2, "change", actionauthor.Spec{}, " name it E2E "),
			),
			status:    http.StatusOK,
			expected:  []string{`{"key":"a1","decision":"refuse"},{"key":"a2","decision":"change"}`},
			delivered: true,
			check: func(t testing.TB, _ *tests.TestApp, stub *reviewStub) {
				require.Equal(t, []map[string]any{
					{
						"key": "a1", "name": "Action a1", "decision": "refuse",
						"approval": "", "comment": "",
					},
					{
						"key": "a2", "name": "Action a2", "decision": "change",
						"approval": "", "comment": "name it E2E",
					},
				}, deliveredDecisions(t, stub))
			},
		},
		{
			name: "accepted by a member who is not an owner",
			body: reviewBody(t, reviewDecision("a1", reviewSHA1, "accept", onboarding, "")),
			app: func(t testing.TB) *tests.TestApp {
				app := setupActionAuthorApp(t)
				demoteToMember(t, app, "userA@example.org")
				return app
			},
			status:    http.StatusOK,
			expected:  []string{`"decision":"accept"`},
			delivered: true,
		},
		{
			name:      "accept from a candidate saved by an earlier review",
			body:      reviewBody(t, reviewDecision("a2", reviewSHA2, "accept", pinPlan(), "")),
			status:    http.StatusOK,
			expected:  []string{`"decision":"accept"`},
			delivered: true,
		},
		{
			// A body that still names a namespace is accepted; the field is ignored.
			name: "body with a namespace",
			body: strings.Replace(
				reviewBody(t, reviewDecision("a1", reviewSHA1, "accept", onboarding, "")),
				"{", `{"namespace":"userb-s-organization",`, 1,
			),
			status:    http.StatusOK,
			expected:  []string{`"decision":"accept"`},
			delivered: true,
			check: func(t testing.TB, _ *tests.TestApp, stub *reviewStub) {
				require.Equal(t, []string{authoringSessionNamespace}, stub.namespaces)
			},
		},
		{
			name: "change without a comment",
			body: reviewBody(t,
				reviewDecision("a1", reviewSHA1, "accept", onboarding, ""),
				reviewDecision("a2", reviewSHA2, "change", actionauthor.Spec{}, "  "),
			),
			status:   http.StatusBadRequest,
			expected: []string{`"invalid_request"`, "request changes for a2: say what to change"},
		},
		{
			name: "accept starting from a refused action",
			body: reviewBody(t,
				reviewDecision("a1", reviewSHA1, "refuse", actionauthor.Spec{}, ""),
				reviewDecision("a2", reviewSHA2, "accept", pinPlan(), ""),
			),
			status:   http.StatusBadRequest,
			expected: []string{`"invalid_request"`, "a2 starts from a1, which you did not accept"},
		},
		{
			name: "duplicate key",
			body: reviewBody(t,
				reviewDecision("a1", reviewSHA1, "refuse", actionauthor.Spec{}, ""),
				reviewDecision("a1", reviewSHA2, "refuse", actionauthor.Spec{}, ""),
			),
			status:   http.StatusBadRequest,
			expected: []string{`"invalid_request"`, "decision a1 is given twice"},
		},
		{
			name: "key not a candidate key",
			body: reviewBody(t,
				reviewDecision("a0", reviewSHA1, "refuse", actionauthor.Spec{}, ""),
			),
			status:   http.StatusBadRequest,
			expected: []string{`"invalid_request"`, "decision key a0: must match a<number>"},
		},
		{
			name: "accept without a plan",
			body: reviewBody(t,
				reviewDecision("a1", reviewSHA1, "accept", actionauthor.Spec{}, ""),
			),
			status: http.StatusBadRequest,
			expected: []string{
				`"invalid_request"`,
				"no plan: describe the action with describe_action",
			},
		},
		{
			name: "accept with an invalid plan",
			body: reviewBody(t, reviewDecision("a1", reviewSHA1, "accept", actionauthor.Spec{
				Slot: "onboarding", Outcome: actionauthor.OutcomeRejection,
			}, "")),
			status:   http.StatusBadRequest,
			expected: []string{`"invalid_request"`},
		},
		{
			name:     "unknown decision",
			body:     reviewBody(t, reviewDecision("a1", reviewSHA1, "maybe", onboarding, "")),
			status:   http.StatusBadRequest,
			expected: []string{"Validation failed", "Decisions[0].Decision"},
		},
		{
			name:     "no decision",
			body:     reviewBody(t),
			status:   http.StatusBadRequest,
			expected: []string{"Validation failed", "ActionAuthorReviewRequest.Decisions"},
		},
		{
			name:     "more decisions than a suite",
			body:     reviewBody(t, tooManyReviewDecisions(41, onboarding)...),
			status:   http.StatusBadRequest,
			expected: []string{"Validation failed", "ActionAuthorReviewRequest.Decisions"},
		},
		{
			name:     "invalid candidate digest",
			body:     reviewBody(t, reviewDecision("a1", "short", "refuse", onboarding, "")),
			status:   http.StatusBadRequest,
			expected: []string{"CandidateSHA256"},
		},
		{
			name:     "another user's session",
			actorID:  "another-user",
			body:     reviewBody(t, reviewDecision("a1", reviewSHA1, "accept", onboarding, "")),
			status:   http.StatusNotFound,
			expected: []string{`"session_not_found"`},
		},
		{
			name:     "member of another organization",
			user:     "userB",
			body:     reviewBody(t, reviewDecision("a1", reviewSHA1, "accept", onboarding, "")),
			status:   http.StatusNotFound,
			expected: []string{`"session_not_found"`},
		},
		{
			name: "profile on a chat action",
			body: func() string {
				decision := reviewDecision("a1", reviewSHA1, "accept", onboarding, "")
				decision.Profile = map[string]any{"pin": "123456"}
				return reviewBody(t, decision)
			}(),
			status: http.StatusBadRequest,
			expected: []string{
				`"invalid_request"`,
				"profile: only FCAF actions carry a profile",
			},
		},
		{
			name:      "update rejected",
			updateErr: temporal.NewApplicationError("unknown tool id", "ValueError"),
			body:      reviewBody(t, reviewDecision("a1", reviewSHA1, "refuse", onboarding, "")),
			status:    http.StatusConflict,
			expected: []string{
				`"review_not_pending"`,
				"this review is no longer pending; reload the chat",
			},
			delivered: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			userName, actorID, appFactory := tc.user, tc.actorID, tc.app
			if userName == "" {
				userName = "userA"
			}
			if actorID == "" {
				actorID = user.Id
			}
			if appFactory == nil {
				appFactory = setupActionAuthorApp
			}
			stub := stubActionAuthorReview(t, actorID, nil, tc.updateErr)
			(&tests.ApiScenario{
				Name:   tc.name,
				Method: http.MethodPost,
				URL:    "/api/action-author/reviews",
				Body:   strings.NewReader(tc.body),
				Headers: map[string]string{
					"Authorization": "Bearer " + authoringToken(t, userName),
				},
				ExpectedStatus:     tc.status,
				ExpectedContent:    tc.expected,
				NotExpectedContent: []string{"not authorized to author actions for this wallet"},
				TestAppFactory:     appFactory,
				AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
					require.Equal(t, tc.delivered, stub.workflowID != "")
					if tc.delivered {
						require.Equal(t, reviewAuthorWorkflow, stub.workflowID)
					}
					if tc.check != nil {
						tc.check(t, app, stub)
					}
				},
			}).Test(t)
		})
	}
}

func TestActionAuthorReviewRejectsSessionsTheCallerCannotReview(t *testing.T) {
	user, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	for _, tc := range []struct {
		name string
		// organization the session was started for, empty for userA's.
		organization string
		lookupErr    error
		status       int
		expected     string
	}{
		{
			name:      "session not visible in Temporal",
			lookupErr: actionauthor.ErrSessionNotVisible,
			status:    http.StatusNotFound,
			expected:  `"session_not_found"`,
		},
		{
			name:         "session of another organization",
			organization: authoringForeignNS,
			status:       http.StatusNotFound,
			expected:     `"session_not_found"`,
		},
		{
			name:      "not a running session agent",
			lookupErr: fmt.Errorf("%w: action author session is not running", actionauthor.ErrForbidden),
			status:    http.StatusNotFound,
			expected:  `"session_not_found"`,
		},
		{
			name:      "Temporal unavailable",
			lookupErr: errors.New("dial failed"),
			status:    http.StatusServiceUnavailable,
			expected:  `"temporal_unavailable"`,
		},
	} {
		stub := stubActionAuthorReview(t, user.Id, tc.lookupErr, nil)
		if tc.organization != "" {
			stub.organization = tc.organization
		}
		(&tests.ApiScenario{
			Name:   tc.name,
			Method: http.MethodPost,
			URL:    "/api/action-author/reviews",
			Body: strings.NewReader(reviewBody(t, reviewDecision(
				"a1", reviewSHA1, "accept", actionauthor.Spec{Slot: "onboarding"}, "",
			))),
			Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
			ExpectedStatus:  tc.status,
			ExpectedContent: []string{tc.expected},
			TestAppFactory:  setupActionAuthorApp,
		}).Test(t)
		require.Empty(t, stub.workflowID, tc.name)
	}
}

// An accept of an FCAF action binds the profile the review showed.
func TestActionAuthorReviewBindsTheFCAFProfile(t *testing.T) {
	user, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	plan := actionauthor.Spec{Slot: "onboarding", FCAFRole: "fcaf-reset-and-onboard"}
	profile := map[string]any{"pin": "123456", "notes": "fresh test wallet"}
	oversized := map[string]any{"notes": strings.Repeat("x", 8192)}
	for _, tc := range []struct {
		name     string
		profile  map[string]any
		status   int
		expected string
	}{
		{name: "with a profile", profile: profile, status: http.StatusOK},
		{name: "without a profile", status: http.StatusOK},
		{
			name:     "profile over 8 KiB",
			profile:  oversized,
			status:   http.StatusBadRequest,
			expected: "fcaf_profile is only for FCAF actions, at most 8 KiB",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := stubActionAuthorReview(t, user.Id, nil, nil)
			stub.mode = actionauthor.SessionModeFCAF
			decision := reviewDecision("a1", reviewSHA1, "accept", plan, "")
			decision.Profile = tc.profile
			expected := []string{`"decision":"accept"`}
			if tc.expected != "" {
				expected = []string{`"invalid_request"`, tc.expected}
			}
			(&tests.ApiScenario{
				Name:   tc.name,
				Method: http.MethodPost,
				URL:    "/api/action-author/reviews",
				Body:   strings.NewReader(reviewBody(t, decision)),
				Headers: map[string]string{
					"Authorization": "Bearer " + authoringToken(t, "userA"),
				},
				ExpectedStatus:  tc.status,
				ExpectedContent: expected,
				TestAppFactory:  setupActionAuthorApp,
				AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
					if tc.status != http.StatusOK {
						require.Empty(t, stub.workflowID)
						return
					}
					decisions := deliveredDecisions(t, stub)
					require.Len(t, decisions, 1)
					requireApproval(t, app, decisions[0], reviewSHA1, plan, user.Id)
					token, _ := decisions[0]["approval"].(string)
					approval, err := actionauthor.VerifyApproval(
						app, token, reviewAuthorWorkflow, reviewSHA1, time.Now(),
					)
					require.NoError(t, err)
					require.Equal(t, actionauthor.ProfileSHA256(tc.profile), approval.ProfileSHA256)
				},
			}).Test(t)
		})
	}
}

func TestActionAuthorReviewBindsTheReplacedCode(t *testing.T) {
	user, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	const code = "appId: eu.wallet\n---\n- launchApp\n"
	target := path.Dir(authoringAndroidVersion) + "/replaced-onboarding"
	appWithTarget := func(t testing.TB) *tests.TestApp {
		app := setupActionAuthorApp(t)
		wallet, err := app.FindFirstRecordByData(
			"wallets", "canonified_name", path.Base(path.Dir(authoringAndroidVersion)),
		)
		require.NoError(t, err)
		actions, err := app.FindCollectionByNameOrId("wallet_actions")
		require.NoError(t, err)
		action := core.NewRecord(actions)
		action.Set("owner", wallet.GetString("owner"))
		action.Set("wallet", wallet.Id)
		action.Set("name", "Replaced onboarding")
		action.Set("code", code)
		action.Set("category", "onboarding")
		require.NoError(t, app.Save(action))
		return app
	}
	plan := actionauthor.Spec{
		Slot:     "onboarding",
		Replaces: &actionauthor.ReplaceSpec{Action: target},
	}

	stub := stubActionAuthorReview(t, user.Id, nil, nil)
	(&tests.ApiScenario{
		Name:   "accepted replacing an action",
		Method: http.MethodPost,
		URL:    "/api/action-author/reviews",
		Body: strings.NewReader(
			reviewBody(t, reviewDecision("a1", reviewSHA1, "accept", plan, "")),
		),
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"decision":"accept"`},
		TestAppFactory:  appWithTarget,
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			normalized := actionauthor.Spec{
				Slot: "onboarding",
				Replaces: &actionauthor.ReplaceSpec{
					Action:     target,
					CodeSHA256: actionauthor.CodeSHA256(code),
				},
			}
			decisions := deliveredDecisions(t, stub)
			requireApproval(t, app, decisions[0], reviewSHA1, normalized, user.Id)
			token, _ := decisions[0]["approval"].(string)
			approval, err := actionauthor.VerifyApproval(
				app, token, reviewAuthorWorkflow, reviewSHA1, time.Now(),
			)
			require.NoError(t, err)
			normalized.Replaces.CodeSHA256 = actionauthor.CodeSHA256("other code")
			require.NotEqual(t, actionauthor.PlanSHA256(normalized), approval.PlanSHA256)
		},
	}).Test(t)

	stale := actionauthor.Spec{
		Slot: "onboarding",
		Replaces: &actionauthor.ReplaceSpec{
			Action:     target,
			CodeSHA256: actionauthor.CodeSHA256("other code"),
		},
	}
	stub = stubActionAuthorReview(t, user.Id, nil, nil)
	(&tests.ApiScenario{
		Name:   "stale replaced code",
		Method: http.MethodPost,
		URL:    "/api/action-author/reviews",
		Body: strings.NewReader(
			reviewBody(t, reviewDecision("a1", reviewSHA1, "accept", stale, "")),
		),
		Headers:        map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedContent: []string{
			`"invalid_request"`,
			"changed since the plan was checked; describe the action again",
		},
		TestAppFactory: appWithTarget,
	}).Test(t)
	require.Empty(t, stub.workflowID)
}
