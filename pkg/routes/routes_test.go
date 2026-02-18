// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package routes

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func TestCreateReverseProxy(t *testing.T) {
	var received *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	targetURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	handler := createReverseProxy(server.URL)

	req := httptest.NewRequest(http.MethodGet, "http://example.test/foo", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	req.Header.Set("Origin", "https://origin.example")
	req.Header.Set("Referer", "https://referer.example/path")

	rec := httptest.NewRecorder()
	event := &core.RequestEvent{Event: router.Event{Request: req, Response: rec}}

	err = handler(event)
	require.NoError(t, err)
	require.NotNil(t, received)

	require.Equal(t, "/foo", received.URL.Path)
	require.Equal(t, targetURL.Host, received.Host)
	require.Contains(t, received.Header.Get("X-Forwarded-For"), "1.2.3.4:5678")
	require.Equal(t, "https://origin.example", received.Header.Get("Origin"))
	require.Equal(t, "https://referer.example/path", received.Header.Get("Referer"))
}

func TestCreateReverseProxy_InvalidTarget(t *testing.T) {
	handler := createReverseProxy("://bad-url")

	req := httptest.NewRequest(http.MethodGet, "http://example.test/foo", nil)
	rec := httptest.NewRecorder()
	event := &core.RequestEvent{Event: router.Event{Request: req, Response: rec}}

	err := handler(event)
	require.Error(t, err)
}

func TestBindAppHooks_RegistersCatchAllProxyRoute(t *testing.T) {
	var proxiedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxiedPath = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	t.Setenv("ADDRESS_UI", server.URL)

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	defer app.Cleanup()

	bindAppHooks(app)

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(se *core.ServeEvent) error {
		mux, buildErr := se.Router.BuildMux()
		require.NoError(t, buildErr)

		req := httptest.NewRequest(http.MethodGet, "/any/path", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusAccepted, rec.Code)
		require.Equal(t, "/any/path", proxiedPath)
		return nil
	})
	require.NoError(t, serveErr)
}

func TestSetup_DoesNotPanic(t *testing.T) {
	app := pocketbase.New()
	require.NotPanics(t, func() {
		Setup(app)
	})
}
