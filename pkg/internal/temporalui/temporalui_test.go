// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalui

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data"

func TestClassify(t *testing.T) {
	const ns = "acme"
	cases := []struct {
		name string
		path string
		want routeKind
	}{
		{name: "prefix only redirects home", path: "/temporal-ui", want: routeHome},
		{name: "prefix slash redirects home", path: "/temporal-ui/", want: routeHome},
		{name: "static asset", path: "/temporal-ui/_app/immutable/x.js", want: routeProxy},
		{name: "favicon", path: "/temporal-ui/favicon.png", want: routePage},
		{
			name: "own namespace page",
			path: "/temporal-ui/namespaces/acme/workflows",
			want: routePage,
		},
		{
			name: "other namespace page redirects home",
			path: "/temporal-ui/namespaces/other/workflows",
			want: routeHome,
		},
		{name: "settings", path: "/temporal-ui/api/v1/settings", want: routeProxy},
		{name: "cluster info", path: "/temporal-ui/api/v1/cluster-info", want: routeProxy},
		{name: "system info", path: "/temporal-ui/api/v1/system-info", want: routeProxy},
		{name: "namespace list", path: "/temporal-ui/api/v1/namespaces", want: routeNamespaceList},
		{name: "own namespace", path: "/temporal-ui/api/v1/namespaces/acme", want: routeProxy},
		{
			name: "own namespace history",
			path: "/temporal-ui/api/v1/namespaces/acme/workflows/wf%2Fid/history",
			want: routeProxy,
		},
		{
			name: "other namespace",
			path: "/temporal-ui/api/v1/namespaces/other/workflows",
			want: routeForbidden,
		},
		{name: "empty namespace", path: "/temporal-ui/api/v1/namespaces/", want: routeForbidden},
		{
			name: "encoded slash in namespace",
			path: "/temporal-ui/api/v1/namespaces/acme%2F..%2Fother",
			want: routeForbidden,
		},
		{
			name: "dot-dot segment",
			path: "/temporal-ui/api/v1/namespaces/acme/../other",
			want: routeForbidden,
		},
		{
			name: "encoded dot-dot segment",
			path: "/temporal-ui/api/v1/namespaces/acme/%2e%2e/other",
			want: routeForbidden,
		},
		{name: "unscoped api", path: "/temporal-ui/api/v1/nexus/endpoints", want: routeForbidden},
		{name: "settings subpath", path: "/temporal-ui/api/v1/settings/x", want: routeForbidden},
		{
			name: "unknown api version",
			path: "/temporal-ui/api/v2/namespaces/acme",
			want: routeForbidden,
		},
		{name: "sso", path: "/temporal-ui/auth/sso", want: routeForbidden},
		{name: "prefix lookalike", path: "/temporal-uix/api/v1/namespaces", want: routeForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, classify(tc.path, ns))
		})
	}
}

type upstreamCall struct {
	path   string
	cookie string
	auth   string
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
	token, err := user.NewAuthToken()
	require.NoError(t, err)
	authCookie := &http.Cookie{
		Name:  authCookieName,
		Value: url.PathEscape(`{"token":"` + token + `","record":{"id":"` + user.Id + `"}}`),
	}

	var calls []upstreamCall
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, upstreamCall{
			path:   r.URL.Path,
			cookie: r.Header.Get("Cookie"),
			auth:   r.Header.Get("Authorization"),
		})
		if strings.Contains(r.URL.Path, "/api/") {
			_, _ = io.WriteString(w, `{"namespaceInfo":{"name":"`+namespace+`"}}`)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<html><head><title>t</title></head><body></body></html>`)
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	require.NoError(t, err)
	handler := Handler(target)

	serve := func(method, path string, cookies ...*http.Cookie) (*httptest.ResponseRecorder, error) {
		req := httptest.NewRequest(method, path, nil)
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
	requireStatus := func(t *testing.T, err error, status int) {
		t.Helper()
		var apiErr *router.ApiError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, status, apiErr.Status)
	}

	t.Run("rejects requests without a session", func(t *testing.T) {
		calls = nil
		_, err := serve(http.MethodGet, "/temporal-ui/api/v1/namespaces/"+namespace)
		requireStatus(t, err, http.StatusUnauthorized)
		require.Empty(t, calls)
	})

	t.Run("rejects an invalid session token", func(t *testing.T) {
		calls = nil
		bad := &http.Cookie{Name: authCookieName, Value: url.PathEscape(`{"token":"nope"}`)}
		_, err := serve(http.MethodGet, "/temporal-ui/api/v1/namespaces/"+namespace, bad)
		requireStatus(t, err, http.StatusUnauthorized)
		require.Empty(t, calls)
	})

	t.Run("rejects writes", func(t *testing.T) {
		calls = nil
		_, err := serve(
			http.MethodPost,
			"/temporal-ui/api/v1/namespaces/"+namespace+"/workflows/wf/terminate",
			authCookie,
		)
		requireStatus(t, err, http.StatusMethodNotAllowed)
		require.Empty(t, calls)
	})

	t.Run("rejects another organization namespace", func(t *testing.T) {
		calls = nil
		_, err := serve(
			http.MethodGet,
			"/temporal-ui/api/v1/namespaces/other-org/workflows",
			authCookie,
		)
		requireStatus(t, err, http.StatusForbidden)
		require.Empty(t, calls)
	})

	t.Run("redirects the root to the caller namespace", func(t *testing.T) {
		rec, err := serve(http.MethodGet, "/temporal-ui/", authCookie)
		require.NoError(t, err)
		require.Equal(t, http.StatusFound, rec.Code)
		require.Equal(
			t,
			"/temporal-ui/namespaces/"+namespace+"/workflows",
			rec.Header().Get("Location"),
		)
	})

	t.Run("proxies own namespace without Credimi credentials", func(t *testing.T) {
		calls = nil
		other := &http.Cookie{Name: "_csrf", Value: "keep"}
		rec, err := serve(
			http.MethodGet,
			"/temporal-ui/api/v1/namespaces/"+namespace+"/workflows/wf/history",
			authCookie,
			other,
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, calls, 1)
		require.Equal(
			t,
			"/temporal-ui/api/v1/namespaces/"+namespace+"/workflows/wf/history",
			calls[0].path,
		)
		require.Equal(t, "_csrf=keep", calls[0].cookie)
		require.Empty(t, calls[0].auth)
	})

	t.Run("lists only the caller namespace", func(t *testing.T) {
		calls = nil
		rec, err := serve(http.MethodGet, "/temporal-ui/api/v1/namespaces?pageSize=100", authCookie)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Len(t, calls, 1)
		require.Equal(t, "/temporal-ui/api/v1/namespaces/"+namespace, calls[0].path)

		var list struct {
			Namespaces []struct {
				NamespaceInfo struct {
					Name string `json:"name"`
				} `json:"namespaceInfo"`
			} `json:"namespaces"`
			NextPageToken string `json:"nextPageToken"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		require.Len(t, list.Namespaces, 1)
		require.Equal(t, namespace, list.Namespaces[0].NamespaceInfo.Name)
		require.Empty(t, list.NextPageToken)
	})

	t.Run("hides the Temporal UI shell on pages", func(t *testing.T) {
		rec, err := serve(
			http.MethodGet,
			"/temporal-ui/namespaces/"+namespace+"/workflows/wf/run/history",
			authCookie,
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), embedStyle+"</head>")
	})
}
