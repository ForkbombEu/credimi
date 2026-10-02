// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

// Reproduce production routing: catch-all UI proxy vs /temporal-ui handler.
func TestTemporalUIRouteBeatsCatchAll(t *testing.T) {
	var hit string

	ui := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = "ui:" + r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ui"))
	}))
	defer ui.Close()

	temporal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = "temporal:" + r.URL.Path
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><head></head><body>temporal</body></html>"))
	}))
	defer temporal.Close()

	t.Setenv("ADDRESS_UI", ui.URL)
	t.Setenv("ADDRESS_TEMPORAL_UI", temporal.URL)

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

		path := "/temporal-ui/namespaces/fcaf-1/workflows/Pipeline-x/run-y/timeline"
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		t.Logf("status=%d hit=%q", rec.Code, hit)
		require.NotContains(t, hit, "ui:", "catch-all must not steal /temporal-ui")
		// Unauthenticated callers get 401 from temporalui.Handler before proxying.
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		return nil
	})
	require.NoError(t, serveErr)
}
