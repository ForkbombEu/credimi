// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthorui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

const (
	testDataDir = "../../../test_pb_data"
	session     = "ActionAuthorSession-0f8fad5b-d9cb-469f-a165-70867728950e"
	chat        = session + "-author"
	otherChat   = "ActionAuthorSession-00000000-0000-0000-0000-000000000000-author"
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
		{http.MethodPost, "api/approve", nil, routeBody},
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
	path, query, namespace, chat, cookie, auth, body string
}

func TestHandler(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	namespace, err := pbutils.GetUserOrganizationCanonifiedName(app, user.Id)
	require.NoError(t, err)
	require.NotEmpty(t, namespace)
	require.NoError(t, actionauthor.RecordSessionRun(
		app, namespace, session, "run", actionauthor.SessionDetails{},
	))
	foreign := "ActionAuthorSession-11111111-2222-3333-4444-555555555555"
	require.NoError(t, actionauthor.RecordSessionRun(
		app, "other-org", foreign, "run", actionauthor.SessionDetails{},
	))
	token, err := user.NewAuthToken()
	require.NoError(t, err)
	authCookie := &http.Cookie{
		Name:  authCookieName,
		Value: url.PathEscape(`{"token":"` + token + `","record":{"id":"` + user.Id + `"}}`),
	}

	var calls []upstreamCall
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		calls = append(calls, upstreamCall{
			path:      r.URL.Path,
			query:     r.URL.RawQuery,
			namespace: r.Header.Get(NamespaceHeader),
			chat:      r.Header.Get(ChatHeader),
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

	t.Run("hides unrecorded and foreign sessions", func(t *testing.T) {
		unrecorded := PathPrefix + "/ActionAuthorSession-00000000-0000-0000-0000-000000000000/"
		_, err := serve(httptest.NewRequest(http.MethodGet, unrecorded, nil), authCookie)
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

	t.Run("proxies chat reads with the scope headers only", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, base+"/api/status/"+chat, nil)
		req.Header.Set(NamespaceHeader, "evil")
		req.Header.Set(ChatHeader, otherChat)
		req.Header.Set("Authorization", "Bearer x")
		_, err := serve(req, authCookie, &http.Cookie{Name: "keep", Value: "1"})
		require.NoError(t, err)
		require.Equal(t, []upstreamCall{{
			path:      "/api/status/" + chat,
			namespace: namespace,
			chat:      chat,
			cookie:    "keep=1",
		}}, calls)
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
}
