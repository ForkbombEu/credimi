// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/recordsecrets"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

const (
	canonifyPrivateYAML   = "DUMMY_PRIVATE_YAML"
	canonifyPrivateSecret = "DUMMY_SECRET_B"
	canonifyPublicSecret  = "DUMMY_PUBLIC_SECRET"
)

type canonifyFixture struct {
	privatePipelinePath   string
	privatePipelineID     string
	publicPipelinePath    string
	privateCredentialPath string
	publicCredentialPath  string
	ownerToken            string
	otherOrgToken         string
}

// setupCanonifyApp seeds userB's organization with an unpublished pipeline, a
// published pipeline, an unpublished credential and a published credential;
// the private ones carry values only members of userB's organization may read.
func setupCanonifyApp(t *testing.T) (*tests.TestApp, canonifyFixture) {
	t.Helper()

	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	canonify.RegisterCanonifyHooks(app)
	recordsecrets.RegisterHooks(app)
	CanonifyRoutes.Add(app)

	org, err := app.FindFirstRecordByFilter("organizations", `name="userB's organization"`)
	require.NoError(t, err)
	orgPath := org.GetString("canonified_name")

	newPipeline := func(name string, published bool) *core.Record {
		pipelineColl, err := app.FindCollectionByNameOrId("pipelines")
		require.NoError(t, err)
		pipeline := core.NewRecord(pipelineColl)
		pipeline.Set("owner", org.Id)
		pipeline.Set("name", name)
		pipeline.Set("description", name)
		pipeline.Set("yaml", canonifyPrivateYAML)
		pipeline.Set("published", published)
		require.NoError(t, app.Save(pipeline))
		return pipeline
	}
	pipeline := newPipeline("privpipe", false)
	publicPipeline := newPipeline("pubpipe", true)

	newCredential := func(issuerName, credentialName, secret string, published bool) string {
		issuerColl, err := app.FindCollectionByNameOrId("credential_issuers")
		require.NoError(t, err)
		issuer := core.NewRecord(issuerColl)
		issuer.Set("owner", org.Id)
		issuer.Set("name", issuerName)
		issuer.Set("url", "https://"+issuerName+".example")
		issuer.Set("published", published)
		require.NoError(t, app.Save(issuer))

		credentialColl, err := app.FindCollectionByNameOrId("credentials")
		require.NoError(t, err)
		credential := core.NewRecord(credentialColl)
		credential.Set("owner", org.Id)
		credential.Set("credential_issuer", issuer.Id)
		credential.Set("name", credentialName)
		credential.Set("secrets", secret)
		credential.Set("published", published)
		require.NoError(t, app.Save(credential))

		return orgPath + "/" + issuer.GetString("canonified_name") + "/" +
			credential.GetString("canonified_name")
	}

	owner, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	ownerToken, err := owner.NewAuthToken()
	require.NoError(t, err)
	other, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	otherOrgToken, err := other.NewAuthToken()
	require.NoError(t, err)

	return app, canonifyFixture{
		privatePipelinePath:   orgPath + "/" + pipeline.GetString("canonified_name"),
		privatePipelineID:     pipeline.Id,
		publicPipelinePath:    orgPath + "/" + publicPipeline.GetString("canonified_name"),
		privateCredentialPath: newCredential("iss", "cred", canonifyPrivateSecret, false),
		publicCredentialPath:  newCredential("pubiss", "pubcred", canonifyPublicSecret, true),
		ownerToken:            ownerToken,
		otherOrgToken:         otherOrgToken,
	}
}

func TestHandleIdentifierValidateEnforcesViewRules(t *testing.T) {
	app, fx := setupCanonifyApp(t)
	defer app.Cleanup()
	mux := buildAppMux(t, app)

	validate := func(path string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/canonify/identifier/validate",
			strings.NewReader(`{"canonified_name":"`+path+`"}`),
		)
		req.Header.Set("Content-Type", "application/json")
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	anonymous := map[string]string{}
	owner := map[string]string{"Authorization": fx.ownerToken}
	otherOrg := map[string]string{"Authorization": fx.otherOrgToken}
	internalAdmin := map[string]string{"Credimi-Api-Key": "internal-test-api-key"}

	cases := []struct {
		name       string
		path       string
		headers    map[string]string
		wantStatus int
		want       []string
		notWant    []string
	}{
		{
			name:       "anonymous cannot read an unpublished pipeline",
			path:       fx.privatePipelinePath,
			headers:    anonymous,
			wantStatus: http.StatusNotFound,
			notWant:    []string{canonifyPrivateYAML},
		},
		{
			name:       "anonymous cannot read an unpublished credential",
			path:       fx.privateCredentialPath,
			headers:    anonymous,
			wantStatus: http.StatusNotFound,
			notWant:    []string{canonifyPrivateSecret},
		},
		{
			name:       "anonymous reads a published credential without its secrets",
			path:       fx.publicCredentialPath,
			headers:    anonymous,
			wantStatus: http.StatusOK,
			want:       []string{`"__canonified_path__":"` + fx.publicCredentialPath + `"`},
			notWant:    []string{canonifyPublicSecret, `"secrets"`},
		},
		{
			name:       "another organization cannot read an unpublished pipeline",
			path:       fx.privatePipelinePath,
			headers:    otherOrg,
			wantStatus: http.StatusNotFound,
			notWant:    []string{canonifyPrivateYAML},
		},
		{
			name:       "another organization reads a published credential without its secrets",
			path:       fx.publicCredentialPath,
			headers:    otherOrg,
			wantStatus: http.StatusOK,
			notWant:    []string{canonifyPublicSecret},
		},
		{
			name:       "owner reads its unpublished pipeline",
			path:       fx.privatePipelinePath,
			headers:    owner,
			wantStatus: http.StatusOK,
			want:       []string{canonifyPrivateYAML},
		},
		{
			name:       "owner reads its credential secrets",
			path:       fx.privateCredentialPath,
			headers:    owner,
			wantStatus: http.StatusOK,
			want:       []string{canonifyPrivateSecret},
		},
		{
			name:       "internal admin key grants nothing on the public route",
			path:       fx.privateCredentialPath,
			headers:    internalAdmin,
			wantStatus: http.StatusNotFound,
			notWant:    []string{canonifyPrivateSecret},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := validate(tc.path, tc.headers)
			require.Equal(t, tc.wantStatus, rec.Code, rec.Body.String())
			for _, want := range tc.want {
				require.Contains(t, rec.Body.String(), want)
			}
			for _, notWant := range tc.notWant {
				require.NotContains(t, rec.Body.String(), notWant)
			}
		})
	}
}

func TestHandleGetIdentifierEnforcesViewRules(t *testing.T) {
	app, fx := setupCanonifyApp(t)
	defer app.Cleanup()
	mux := buildAppMux(t, app)

	get := func(collection, id, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(
			http.MethodGet,
			"/api/canonify/identifier/get?collection="+collection+"&id="+id,
			nil,
		)
		if token != "" {
			req.Header.Set("Authorization", token)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	rec := get("pipelines", fx.privatePipelineID, "")
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
	require.NotContains(t, rec.Body.String(), fx.privatePipelinePath)

	rec = get("pipelines", fx.privatePipelineID, fx.otherOrgToken)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())

	rec = get("pipelines", fx.privatePipelineID, fx.ownerToken)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"identifier":"`+fx.privatePipelinePath)

	superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
	require.NoError(t, err)
	rec = get("_superusers", superuser.Id, "")
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.NotContains(t, rec.Body.String(), "admin@example.org")
}

func TestHandleIdentifierValidateSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	org, err := app.FindFirstRecordByFilter("organizations", `name="userA's organization"`)
	require.NoError(t, err)

	tpl := canonify.CanonifyPaths["organizations"]
	path, err := canonify.BuildPath(app, org, tpl, "")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/canonify/identifier/validate", nil)
	req = req.WithContext(
		context.WithValue(req.Context(), middlewares.ValidatedInputKey, IdentifierValidateRequest{
			CanonifiedName: path,
		}),
	)
	rec := httptest.NewRecorder()

	err = HandleIdentifierValidate()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "valid identifier")
	require.Contains(t, rec.Body.String(), `"__canonified_path__":"`+path+`"`)
}

func TestHandleGetIdentifierMissingParams(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/canonify/identifier/get", nil)
	rec := httptest.NewRecorder()

	err = HandleGetIdentifier()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	requireHandlerErrorHandled(t, rec, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleGetIdentifierSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	org, err := app.FindFirstRecordByFilter("organizations", `name="userA's organization"`)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/canonify/identifier/get?collection=organizations&id="+org.Id,
		nil,
	)
	rec := httptest.NewRecorder()

	err = HandleGetIdentifier()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "\"identifier\"")
}
