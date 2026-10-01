// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipelineresults

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineResultsCollectionIsSuperuserOnly(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := createPipelineResultRecord(t, app, coll)
	screenshot, err := filesystem.NewFileFromBytes([]byte("png-bytes"), "shot.png")
	require.NoError(t, err)
	record.Set("screenshots", []*filesystem.File{screenshot})
	require.NoError(t, app.Save(record))

	screenshots := record.GetStringSlice("screenshots")
	require.Len(t, screenshots, 1)

	mux := buildRulesTestMux(t, app)
	tokenA := authTokenForEmail(t, app, "users", "userA@example.org")
	tokenB := authTokenForEmail(t, app, "users", "userB@example.org")
	tokenAdmin := authTokenForEmail(t, app, core.CollectionNameSuperusers, "admin@example.org")

	listPath := "/api/collections/pipeline_results/records?perPage=500"
	viewPath := "/api/collections/pipeline_results/records/" + record.Id

	tests := []struct {
		name         string
		token        string
		wantListCode int
		wantViewCode int
	}{
		{
			name:         "anonymous",
			wantListCode: http.StatusForbidden,
			wantViewCode: http.StatusForbidden,
		},
		{
			name:         "other org member",
			token:        tokenB,
			wantListCode: http.StatusForbidden,
			wantViewCode: http.StatusForbidden,
		},
		{
			name:         "owner org member",
			token:        tokenA,
			wantListCode: http.StatusForbidden,
			wantViewCode: http.StatusForbidden,
		},
		{
			name:         "superuser",
			token:        tokenAdmin,
			wantListCode: http.StatusOK,
			wantViewCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			listRec := serveRulesRequest(mux, listPath, tc.token)
			require.Equal(t, tc.wantListCode, listRec.Code, listRec.Body.String())
			if tc.wantListCode == http.StatusOK {
				var list struct {
					Items []struct {
						ID string `json:"id"`
					} `json:"items"`
				}
				require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &list))
				ids := make([]string, 0, len(list.Items))
				for _, item := range list.Items {
					ids = append(ids, item.ID)
				}
				assert.Contains(t, ids, record.Id)
			}

			viewRec := serveRulesRequest(mux, viewPath, tc.token)
			assert.Equal(t, tc.wantViewCode, viewRec.Code, viewRec.Body.String())
		})
	}

	t.Run("anonymous file download stays public", func(t *testing.T) {
		fileRec := serveRulesRequest(
			mux,
			"/api/files/pipeline_results/"+record.Id+"/"+screenshots[0],
			"",
		)
		require.Equal(t, http.StatusOK, fileRec.Code, fileRec.Body.String())
		assert.Equal(t, "png-bytes", fileRec.Body.String())
	})
}

func buildRulesTestMux(t *testing.T, app *tests.TestApp) http.Handler {
	t.Helper()
	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	var mux http.Handler
	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	require.NoError(t, app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		var err error
		mux, err = e.Router.BuildMux()
		return err
	}))
	require.NotNil(t, mux)
	return mux
}

func authTokenForEmail(t *testing.T, app *tests.TestApp, collection, email string) string {
	t.Helper()
	user, err := app.FindAuthRecordByEmail(collection, email)
	require.NoError(t, err)
	token, err := user.NewAuthToken()
	require.NoError(t, err)
	return token
}

func serveRulesRequest(mux http.Handler, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}
