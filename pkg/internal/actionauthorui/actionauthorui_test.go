// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthorui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	testDataDir = "../../../test_pb_data"
	session     = "ActionAuthorSession-testrunner00001-0f8fad5b-d9cb-469f-a165-70867728950e"
	chat        = session + "-author"
	otherChat   = "ActionAuthorSession-testrunner00001-00000000-0000-0000-0000-000000000000-author"
)

func TestClassify(t *testing.T) {
	attach := url.Values{"session_id": {chat}}
	for _, tc := range []struct {
		method string
		path   string
		query  url.Values
		want   route
	}{
		{http.MethodGet, "", nil, routeProxy},
		{http.MethodHead, "", nil, routeProxy},
		{http.MethodPost, "", nil, routeForbidden},
		{http.MethodGet, "assets/index-1.js", nil, routeProxy},
		{http.MethodGet, "assets", nil, routeForbidden},
		{http.MethodGet, "assets/", nil, routeForbidden},
		{http.MethodGet, "assets/a/b.js", nil, routeForbidden},
		{http.MethodGet, "favicon.svg", nil, routeProxy},
		{http.MethodGet, "action-embed.css", nil, routeProxy},
		{http.MethodGet, "action-embed.js", nil, routeProxy},
		{http.MethodPost, "action-embed.js", nil, routeForbidden},
		{http.MethodGet, "index.js", nil, routeForbidden},
		{http.MethodGet, "api/agents", nil, routeProxy},
		{http.MethodGet, "api/sessions", nil, routeProxy},
		{http.MethodPost, "api/sessions", nil, routeForbidden},
		{http.MethodGet, "api/status/" + chat, nil, routeProxy},
		{http.MethodGet, "api/workflow-status/" + chat, nil, routeProxy},
		{http.MethodGet, "api/agent-interface/" + chat, nil, routeProxy},
		{http.MethodGet, "api/status/" + otherChat, nil, routeForbidden},
		{http.MethodGet, "api/attach", attach, routeProxy},
		{http.MethodGet, "api/attach", url.Values{"session_id": {otherChat}}, routeForbidden},
		{http.MethodGet, "api/attach", url.Values{"session_id": {chat, otherChat}}, routeForbidden},
		{http.MethodGet, "api/attach", nil, routeForbidden},
		{http.MethodPost, "api/sessions/" + chat + "/close", nil, routeProxy},
		{http.MethodPost, "api/sessions/" + otherChat + "/close", nil, routeForbidden},
		{http.MethodPost, "api/messages", nil, routeBody},
		{http.MethodPost, "api/chat", nil, routeBody},
		{http.MethodPost, "api/approve", nil, routeForbidden},
		{http.MethodGet, "api/messages", nil, routeForbidden},
		{http.MethodPost, "api/callback-result", nil, routeForbidden},
		{http.MethodGet, "api/files/x", nil, routeForbidden},
		{http.MethodDelete, "api/sessions/" + chat + "/close", nil, routeForbidden},
	} {
		t.Run(tc.method+" /"+tc.path, func(t *testing.T) {
			got := classify(tc.method, strings.Split(tc.path, "/"), tc.query, chat)
			require.Equal(t, tc.want, got)
		})
	}
}

type upstreamCall struct {
	path, query, namespace, chat, token, cookie, auth, body string
}

// stubNamespaceToken signs "token-<namespace>", or fails with err, and
// records the namespaces asked for.
func stubNamespaceToken(t *testing.T, err error) *[]string {
	t.Helper()
	namespaces := []string{}
	orig := namespaceTokenFn
	t.Cleanup(func() { namespaceTokenFn = orig })
	namespaceTokenFn = func(namespace string) (string, error) {
		namespaces = append(namespaces, namespace)
		if err != nil {
			return "", err
		}
		return "token-" + namespace, nil
	}
	return &namespaces
}

// stubSessionRuns makes c list runs (oldest first) as the runs of sessionID,
// newest first as Temporal's visibility lists them.
func stubSessionRuns(t *testing.T, c *temporalmocks.Client, sessionID string, runs ...string) {
	t.Helper()
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	executions := make([]*workflowpb.WorkflowExecutionInfo, 0, len(runs))
	for i := len(runs) - 1; i >= 0; i-- {
		executions = append(executions, &workflowpb.WorkflowExecutionInfo{
			Execution: &commonpb.WorkflowExecution{WorkflowId: sessionID, RunId: runs[i]},
			Type:      &commonpb.WorkflowType{Name: activities.ActionAuthorSessionWorkflowName},
			StartTime: timestamppb.New(start.Add(time.Duration(i) * time.Minute)),
		})
	}
	c.On("ListWorkflow", mock.Anything, mock.MatchedBy(
		func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
			return req.GetQuery() == fmt.Sprintf("WorkflowId = %q", sessionID)
		},
	)).Return(&workflowservice.ListWorkflowExecutionsResponse{Executions: executions}, nil).Maybe()
}

// historyIterator iterates over a session's history.
type historyIterator struct{ events []*historypb.HistoryEvent }

func (h *historyIterator) HasNext() bool { return len(h.events) > 0 }

func (h *historyIterator) Next() (*historypb.HistoryEvent, error) {
	if len(h.events) == 0 {
		return nil, errors.New("no more events")
	}
	event := h.events[0]
	h.events = h.events[1:]
	return event, nil
}

// startSession fakes in c the session sessionID that actorID started for
// organization, as the start activity leaves it: its only run "run" with
// Credimi's grant in its start input and Credimi's run grant signal.
func startSession(
	t *testing.T,
	app core.App,
	c *temporalmocks.Client,
	sessionID, organization, actorID string,
) {
	t.Helper()
	namespace, ok := actionauthor.SessionNamespace(sessionID)
	require.True(t, ok)
	const (
		device  = "tenant/runner/device"
		version = "tenant/wallet/v1"
	)
	grant, err := actionauthor.SignSession(app, actionauthor.SessionGrant{
		SessionWorkflowID:       sessionID,
		Namespace:               namespace,
		Organization:            organization,
		DeviceID:                device,
		Platform:                actionauthor.PlatformAndroid,
		WalletVersionIdentifier: version,
		ActorID:                 actorID,
	})
	require.NoError(t, err)
	input, err := temporalcrypto.DataConverter().ToPayloads(workflowengine.WorkflowInput{
		Payload: map[string]any{
			"device_id":                 device,
			"platform":                  actionauthor.PlatformAndroid,
			"wallet_version_identifier": version,
			"actor_id":                  actorID,
			"namespace":                 namespace,
			"organization":              organization,
			"grant":                     grant,
		},
	})
	require.NoError(t, err)
	token, err := actionauthor.SignRun(app, actionauthor.RunGrant{
		SessionWorkflowID: sessionID,
		RunID:             "run",
	})
	require.NoError(t, err)
	runGrant, err := temporalcrypto.DataConverter().ToPayloads(token)
	require.NoError(t, err)
	stubSessionRuns(t, c, sessionID, "run")
	c.On(
		"GetWorkflowHistory",
		mock.Anything,
		sessionID,
		"run",
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(func(
		context.Context,
		string,
		string,
		bool,
		enumspb.HistoryEventFilterType,
	) client.HistoryEventIterator {
		return &historyIterator{events: []*historypb.HistoryEvent{{
			EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
				WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
					Input: input,
				},
			},
		}, {
			// Credimi's run grant signal for the run.
			EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED,
			Attributes: &historypb.HistoryEvent_WorkflowExecutionSignaledEventAttributes{
				WorkflowExecutionSignaledEventAttributes: &historypb.WorkflowExecutionSignaledEventAttributes{
					SignalName: activities.ActionAuthorRunGrantSignal,
					Input:      runGrant,
				},
			},
		}}}
	}).Maybe()
}

// stubSessionTemporal answers every namespace with c, but brokenNamespace
// with an error, empties the session cache, and counts the clients asked for.
func stubSessionTemporal(t *testing.T, c client.Client, brokenNamespace string) *int {
	t.Helper()
	t.Setenv(
		temporalcrypto.SecretsEncryptionKeyEnv,
		"MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA=",
	)
	resetSessionRefs := func() {
		sessionRefs.Clear()
		sessionRefCount.Store(0)
	}
	resetSessionRefs()
	clients := 0
	orig := clientFor
	t.Cleanup(func() {
		clientFor = orig
		resetSessionRefs()
	})
	clientFor = func(namespace string) (client.Client, error) {
		clients++
		if namespace == brokenNamespace {
			return nil, errors.New("dial failed")
		}
		return c, nil
	}
	return &clients
}

func TestHandler(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	organization, err := pbutils.GetUserOrganizationCanonifiedName(app, user.Id)
	require.NoError(t, err)
	require.NotEmpty(t, organization)
	// Sessions run in their runner's namespace, for the caller's organization.
	const namespace = "runner-testrunner00001"
	temporal := &temporalmocks.Client{}
	const broken = "ActionAuthorSession-brokenrunner001-11111111-2222-3333-4444-555555555555"
	clients := stubSessionTemporal(t, temporal, "runner-brokenrunner001")
	startSession(t, app, temporal, session, organization, user.Id)
	foreign := "ActionAuthorSession-testrunner00001-11111111-2222-3333-4444-555555555555"
	startSession(t, app, temporal, foreign, "other-org", user.Id)
	unknown := "ActionAuthorSession-testrunner00001-00000000-0000-0000-0000-000000000000"
	stubSessionRuns(t, temporal, unknown)
	tokenNamespaces := stubNamespaceToken(t, nil)
	cookieOf := func(user *core.Record) *http.Cookie {
		token, err := user.NewAuthToken()
		require.NoError(t, err)
		return &http.Cookie{
			Name:  authCookieName,
			Value: url.PathEscape(`{"token":"` + token + `","record":{"id":"` + user.Id + `"}}`),
		}
	}
	authCookie := cookieOf(user)
	// userB joins userA's organization as a plain member.
	member, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	membership, err := app.FindFirstRecordByData("orgAuthorizations", "user", member.Id)
	require.NoError(t, err)
	memberRole, err := app.FindFirstRecordByData("orgRoles", "name", "member")
	require.NoError(t, err)
	membership.Set("organization", orgID)
	membership.Set("role", memberRole.Id)
	require.NoError(t, app.Save(membership))
	memberCookie := cookieOf(member)

	var calls []upstreamCall
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		calls = append(calls, upstreamCall{
			path:      r.URL.Path,
			query:     r.URL.RawQuery,
			namespace: r.Header.Get(NamespaceHeader),
			chat:      r.Header.Get(ChatHeader),
			token:     r.Header.Get(TokenHeader),
			cookie:    r.Header.Get("Cookie"),
			auth:      r.Header.Get("Authorization"),
			body:      string(body),
		})
		_, _ = io.WriteString(w, `{}`)
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	require.NoError(t, err)
	handler := Handler(target)

	base := PathPrefix + "/" + session
	serve := func(req *http.Request, cookies ...*http.Cookie) (*httptest.ResponseRecorder, error) {
		calls = nil
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		err := handler(&core.RequestEvent{
			App:   app,
			Event: router.Event{Request: req, Response: rec},
		})
		return rec, err
	}
	post := func(path, origin, body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		return req
	}
	requireStatus := func(t *testing.T, err error, status int) {
		t.Helper()
		var apiErr *router.ApiError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, status, apiErr.Status)
		require.Empty(t, calls)
	}

	t.Run("rejects requests without a session", func(t *testing.T) {
		_, err := serve(httptest.NewRequest(http.MethodGet, base+"/", nil))
		requireStatus(t, err, http.StatusUnauthorized)
	})

	t.Run("hides unknown and foreign sessions", func(t *testing.T) {
		_, err := serve(
			httptest.NewRequest(http.MethodGet, PathPrefix+"/"+unknown+"/", nil),
			authCookie,
		)
		requireStatus(t, err, http.StatusNotFound)
		_, err = serve(
			httptest.NewRequest(http.MethodGet, PathPrefix+"/"+foreign+"/", nil),
			authCookie,
		)
		requireStatus(t, err, http.StatusNotFound)
		_, err = serve(
			httptest.NewRequest(http.MethodGet, PathPrefix+"/not-a-session/", nil),
			authCookie,
		)
		requireStatus(t, err, http.StatusNotFound)
	})

	t.Run("answers 500 when Temporal fails", func(t *testing.T) {
		_, err := serve(
			httptest.NewRequest(http.MethodGet, PathPrefix+"/"+broken+"/", nil),
			authCookie,
		)
		requireStatus(t, err, http.StatusInternalServerError)
	})

	t.Run("proxies chat reads with the scope headers only", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, base+"/api/status/"+chat, nil)
		req.Header.Set(NamespaceHeader, "evil")
		req.Header.Set(ChatHeader, otherChat)
		req.Header.Set(TokenHeader, "forged")
		req.Header.Set("Authorization", "Bearer x")
		*tokenNamespaces = nil
		_, err := serve(req, authCookie, &http.Cookie{Name: "keep", Value: "1"})
		require.NoError(t, err)
		require.Equal(t, []upstreamCall{{
			path:      "/api/status/" + chat,
			namespace: namespace,
			chat:      chat,
			token:     "token-" + namespace,
			cookie:    "keep=1",
		}}, calls)
		require.Equal(t, []string{namespace}, *tokenNamespaces)
	})

	t.Run("answers 503 without a Temporal token", func(t *testing.T) {
		stubNamespaceToken(t, errors.New("no signing key"))
		rec, err := serve(
			httptest.NewRequest(http.MethodGet, base+"/api/status/"+chat, nil),
			authCookie,
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusServiceUnavailable, rec.Code)
		require.Contains(t, rec.Body.String(), `"reason":"temporal_unavailable"`)
		require.Empty(t, calls)
	})

	t.Run("keeps the query of attach", func(t *testing.T) {
		_, err := serve(
			httptest.NewRequest(http.MethodGet, base+"/api/attach?session_id="+chat, nil),
			authCookie,
		)
		require.NoError(t, err)
		require.Len(t, calls, 1)
		require.Equal(t, "/api/attach", calls[0].path)
		require.Equal(t, "session_id="+chat, calls[0].query)
	})

	t.Run("refuses other chats", func(t *testing.T) {
		_, err := serve(
			httptest.NewRequest(http.MethodGet, base+"/api/status/"+otherChat, nil),
			authCookie,
		)
		requireStatus(t, err, http.StatusForbidden)
		_, err = serve(
			httptest.NewRequest(http.MethodGet, base+"/api/attach?session_id="+otherChat, nil),
			authCookie,
		)
		requireStatus(t, err, http.StatusForbidden)
	})

	t.Run("refuses escaped and dot segments", func(t *testing.T) {
		for _, path := range []string{
			base + "/api/status%2F" + chat,
			base + "/assets/%2e%2e",
		} {
			_, err := serve(httptest.NewRequest(http.MethodGet, path, nil), authCookie)
			requireStatus(t, err, http.StatusForbidden)
		}
	})

	t.Run("checks the origin and session of writes", func(t *testing.T) {
		host := "http://example.com"
		body := `{"session_id":"` + chat + `","content":"hi"}`
		_, err := serve(post(base+"/api/messages", "", body), authCookie)
		requireStatus(t, err, http.StatusForbidden)
		_, err = serve(post(base+"/api/messages", "https://evil.example", body), authCookie)
		requireStatus(t, err, http.StatusForbidden)
		_, err = serve(
			post(base+"/api/messages", host, `{"session_id":"`+otherChat+`"}`),
			authCookie,
		)
		requireStatus(t, err, http.StatusForbidden)
		_, err = serve(post(base+"/api/messages", host, "not json"), authCookie)
		requireStatus(t, err, http.StatusForbidden)
		_, err = serve(
			post(base+"/api/messages", host, `{"session_id":"`+strings.Repeat("a", maxBody)+`"}`),
			authCookie,
		)
		requireStatus(t, err, http.StatusRequestEntityTooLarge)

		_, err = serve(post(base+"/api/messages", host, body), authCookie)
		require.NoError(t, err)
		require.Len(t, calls, 1)
		require.Equal(t, "/api/messages", calls[0].path)
		require.Equal(t, body, calls[0].body)
	})

	t.Run("refuses ambiguous session ids", func(t *testing.T) {
		host := "http://example.com"
		for _, body := range []string{
			`{"session_id":"` + otherChat + `","Session_Id":"` + chat + `"}`,
			`{"SESSION_ID":"` + chat + `"}`,
			`{"ſession_id":"` + chat + `"}`,
			`{"session_id":"` + otherChat + `","session_id":"` + chat + `"}`,
			`{"session_id":1}`,
			`{"session_id":null}`,
			`[]`,
			`{"session_id":"` + chat + `"} {}`,
		} {
			_, err := serve(post(base+"/api/messages", host, body), authCookie)
			requireStatus(t, err, http.StatusForbidden)
		}
		_, err := serve(
			post(base+"/api/approve", host, `{"session_id":"`+chat+`"}`),
			authCookie,
		)
		requireStatus(t, err, http.StatusForbidden)
	})

	t.Run("hides the chat from other users", func(t *testing.T) {
		body := `{"session_id":"` + chat + `","content":"hi"}`
		_, err := serve(post(base+"/api/messages", "http://example.com", body), memberCookie)
		requireStatus(t, err, http.StatusNotFound)

		_, err = serve(
			httptest.NewRequest(http.MethodGet, base+"/api/status/"+chat, nil),
			memberCookie,
		)
		requireStatus(t, err, http.StatusNotFound)
		require.Len(t, calls, 0)
	})

	t.Run("refuses callback results", func(t *testing.T) {
		_, err := serve(
			post(base+"/api/callback-result", "http://example.com", `{"session_id":"`+chat+`"}`),
			authCookie,
		)
		requireStatus(t, err, http.StatusForbidden)
	})

	t.Run("adds the trailing slash", func(t *testing.T) {
		rec, err := serve(httptest.NewRequest(http.MethodGet, base+"?s="+chat, nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusPermanentRedirect, rec.Code)
		require.Equal(t, base+"/?s="+chat, rec.Header().Get("Location"))
	})

	t.Run("looks a session up in Temporal once", func(t *testing.T) {
		sessionRefs.Clear()
		sessionRefCount.Store(0)
		lists := func() int {
			n := 0
			for _, call := range temporal.Calls {
				if call.Method == "ListWorkflow" {
					n++
				}
			}
			return n
		}
		clientsBefore, listsBefore := *clients, lists()
		for range 2 {
			_, err := serve(
				httptest.NewRequest(http.MethodGet, base+"/api/status/"+chat, nil),
				authCookie,
			)
			require.NoError(t, err)
			require.Len(t, calls, 1)
		}
		require.Equal(t, clientsBefore+1, *clients)
		require.Equal(t, listsBefore+1, lists())
	})

	t.Run("looks a session up again after a minute", func(t *testing.T) {
		sessionRefs.Clear()
		sessionRefCount.Store(0)
		start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		at := start
		origNow, origClientFor := now, clientFor
		t.Cleanup(func() { now, clientFor = origNow, origClientFor })
		now = func() time.Time { return at }
		_, err := serve(
			httptest.NewRequest(http.MethodGet, base+"/api/status/"+chat, nil),
			authCookie,
		)
		require.NoError(t, err)
		require.Len(t, calls, 1)

		// The session's runs are deleted in Temporal.
		gone := &temporalmocks.Client{}
		stubSessionRuns(t, gone, session)
		clientFor = func(string) (client.Client, error) { return gone, nil }

		at = start.Add(59 * time.Second)
		_, err = serve(
			httptest.NewRequest(http.MethodGet, base+"/api/status/"+chat, nil),
			authCookie,
		)
		require.NoError(t, err)
		require.Len(t, calls, 1)

		at = start.Add(61 * time.Second)
		_, err = serve(
			httptest.NewRequest(http.MethodGet, base+"/api/status/"+chat, nil),
			authCookie,
		)
		requireStatus(t, err, http.StatusNotFound)
	})
}
