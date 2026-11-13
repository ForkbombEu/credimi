// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

// newFakeGiteaServer serves the token and user endpoints of a gitea-style
// OAuth2 provider that signs in fresh@example.org.
func newFakeGiteaServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"fake","token_type":"bearer"}`))
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(
			`{"id":4242,"login":"fresh","full_name":"Fresh","email":"fresh@example.org","active":true}`,
		))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// apiKeyTokenFor exchanges a new API key of the user with email for a token
// through /api/apikey/authenticate.
func apiKeyTokenFor(t testing.TB, app core.App, email string) string {
	t.Helper()

	user, err := app.FindAuthRecordByEmail("users", email)
	require.NoError(t, err)
	apiKey, err := handlers.NewApiKeyService(handlers.NewAppAdapter(app)).
		GenerateApiKey(user.Id, "oauth2")
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/apikey/authenticate", nil)
	req.Header.Set("Credimi-Api-Key", apiKey)
	event := &core.RequestEvent{App: app}
	event.Request = req
	event.Response = rec
	require.NoError(t, handlers.AuthenticateApiKey()(event))

	var body struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotEmpty(t, body.Token)
	return body.Token
}

func TestHookRefuseAPIKeyOAuth2Linking(t *testing.T) {
	provider := newFakeGiteaServer(t)
	const body = `{"provider":"gitea","code":"c0de","codeVerifier":"v3rifier",` +
		`"redirectURL":"http://localhost/redirect"}`

	newApp := func(t testing.TB) *tests.TestApp {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)

		users, err := app.FindCollectionByNameOrId("users")
		require.NoError(t, err)
		users.OAuth2.Enabled = true
		users.OAuth2.Providers = []core.OAuth2ProviderConfig{{
			Name:         "gitea",
			ClientId:     "client",
			ClientSecret: "secret",
			AuthURL:      provider.URL + "/authorize",
			TokenURL:     provider.URL + "/token",
			UserInfoURL:  provider.URL + "/user",
		}}
		require.NoError(t, app.Save(users))

		HookRefuseAPIKeyOAuth2Linking(app)
		return app
	}

	t.Run("a token exchanged for an API key cannot link an OAuth2 account", func(t *testing.T) {
		headers := map[string]string{}
		(&tests.ApiScenario{
			Name:            "auth-with-oauth2 with an API key token",
			Method:          http.MethodPost,
			URL:             "/api/collections/users/auth-with-oauth2",
			Body:            strings.NewReader(body),
			Headers:         headers,
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{"exchanged for an API key"},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := newApp(t)
				headers["Authorization"] = apiKeyTokenFor(t, app, "userA@example.org")
				return app
			},
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
				_, err := app.FindFirstExternalAuthByExpr(dbx.HashExp{
					"provider":   "gitea",
					"providerId": "4242",
				})
				require.Error(t, err, "no OAuth2 account is linked")
			},
		}).Test(t)
	})

	t.Run("an OAuth2 sign-in without Authorization still works", func(t *testing.T) {
		(&tests.ApiScenario{
			Name:            "auth-with-oauth2 without Authorization",
			Method:          http.MethodPost,
			URL:             "/api/collections/users/auth-with-oauth2",
			Body:            strings.NewReader(body),
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"token":`, `"fresh@example.org"`},
			TestAppFactory:  newApp,
		}).Test(t)
	})
}
