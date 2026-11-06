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

// The /action-author-ui proxy, not the UI catch-all, answers its paths.
func TestActionAuthorUIRouteBeatsCatchAll(t *testing.T) {
	var hit string
	record := func(name string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hit = name + ":" + r.URL.Path
		}))
	}
	ui := record("ui")
	defer ui.Close()
	chat := record("chat")
	defer chat.Close()

	t.Setenv("ADDRESS_UI", ui.URL)
	t.Setenv("ADDRESS_ACTION_AUTHOR_UI", chat.URL)

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

		path := "/action-author-ui/ActionAuthorSession-00000000-0000-0000-0000-000000000000/"
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		require.Empty(t, hit, "neither the UI nor the chat service is reached")
		// Unauthenticated callers get 401 from actionauthorui.Handler.
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		return nil
	})
	require.NoError(t, serveErr)
}
