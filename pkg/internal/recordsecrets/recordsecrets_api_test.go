// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package recordsecrets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	publishedCredentialID = "credsecrets0001"
	dummySecret           = "DUMMYSECRET"
)

func TestSecretsAreNotQueryableByNonOwners(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	RegisterHooks(app)
	createPublishedCredential(t, app)
	mux := buildTestMux(t, app)

	ownerToken := authToken(t, app, "userA@example.org")
	otherToken := authToken(t, app, "userB@example.org")

	probes := []struct {
		name       string
		collection string
		query      url.Values
	}{
		{
			name:       "credentials secrets prefix filter",
			collection: "credentials",
			query: url.Values{
				"filter": {"id='" + publishedCredentialID + "' && secrets~'DUMMY%'"},
			},
		},
		{
			name:       "credentials secrets sort",
			collection: "credentials",
			query:      url.Values{"sort": {"secrets"}},
		},
		{
			name:       "credentials secrets through issuer back relation",
			collection: "credential_issuers",
			query: url.Values{
				"filter": {"credentials_via_credential_issuer.secrets~'DUMMY%'"},
			},
		},
		{
			name:       "use case verifications secrets filter",
			collection: "use_cases_verifications",
			query:      url.Values{"filter": {"secrets~'%'"}},
		},
	}
	for _, auth := range []struct {
		name  string
		token string
	}{
		{name: "anonymous"},
		{name: "other organization", token: otherToken},
		{name: "owner organization", token: ownerToken},
	} {
		for _, probe := range probes {
			t.Run(auth.name+" cannot query "+probe.name, func(t *testing.T) {
				path := "/api/collections/" + probe.collection + "/records?" + probe.query.Encode()
				rec := serve(mux, http.MethodGet, path, auth.token, "")
				require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			})
		}
	}

	t.Run("anonymous lists the published credential without secrets", func(t *testing.T) {
		query := url.Values{"filter": {"id='" + publishedCredentialID + "'"}}
		rec := serve(
			mux,
			http.MethodGet,
			"/api/collections/credentials/records?"+query.Encode(),
			"",
			"",
		)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var list struct {
			TotalItems int `json:"totalItems"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		assert.Equal(t, 1, list.TotalItems)
		assert.NotContains(t, rec.Body.String(), `"secrets"`)
		assert.NotContains(t, rec.Body.String(), dummySecret)
	})

	t.Run("owner reads secrets", func(t *testing.T) {
		rec := serve(
			mux,
			http.MethodGet,
			"/api/collections/credentials/records/"+publishedCredentialID,
			ownerToken,
			"",
		)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Body.String(), `"secrets":"`+dummySecret+`"`)
	})

	t.Run("owner updates secrets", func(t *testing.T) {
		rec := serve(
			mux,
			http.MethodPatch,
			"/api/collections/credentials/records/"+publishedCredentialID,
			ownerToken,
			`{"secrets":"token: rotated\n"}`,
		)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		stored, err := app.FindRecordById("credentials", publishedCredentialID)
		require.NoError(t, err)
		assert.Equal(t, "token: rotated\n", stored.GetString("secrets"))
	})
}

func createPublishedCredential(t testing.TB, app *tests.TestApp) {
	t.Helper()

	ownerOrg := findOrganizationByName(t, app, "userA's organization")
	issuer := createCredentialIssuer(t, app, ownerOrg.Id)
	issuer.Set("published", true)
	require.NoError(t, app.Save(issuer))

	coll, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	record.Set("id", publishedCredentialID)
	record.Set("owner", ownerOrg.Id)
	record.Set("name", "published secret credential")
	record.Set("credential_issuer", issuer.Id)
	record.Set("published", true)
	record.Set("secrets", dummySecret)
	require.NoError(t, app.Save(record))
}

func buildTestMux(t testing.TB, app *tests.TestApp) http.Handler {
	t.Helper()

	router, err := apis.NewRouter(app)
	require.NoError(t, err)

	var mux http.Handler
	serveEvent := &core.ServeEvent{App: app, Router: router}
	require.NoError(t, app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		var err error
		mux, err = e.Router.BuildMux()
		return err
	}))
	return mux
}

func authToken(t testing.TB, app *tests.TestApp, email string) string {
	t.Helper()

	user, err := app.FindAuthRecordByEmail("users", email)
	require.NoError(t, err)
	token, err := user.NewAuthToken()
	require.NoError(t, err)
	return token
}

func serve(mux http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}
