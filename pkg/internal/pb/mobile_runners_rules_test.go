// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rulesTestUserB = "userB@example.org"

func TestMobileRunnersUpdateRuleGuardsPrivilegedFields(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		superuser     bool
		adminManaged  bool
		body          string
		wantCode      int
		wantOwner     string
		wantAdmin     bool
		wantName      string
		wantPublished bool
	}{
		{
			name:      "org owner cannot self-grant admin_managed",
			email:     jsHooksUserA,
			body:      `{"admin_managed":true,"published":true}`,
			wantCode:  http.StatusNotFound,
			wantOwner: jsHooksUserAOrg,
			wantName:  "rules-runner",
		},
		{
			name:      "org owner cannot move the runner to another org",
			email:     jsHooksUserA,
			body:      `{"owner":"` + jsHooksUserBOrg + `"}`,
			wantCode:  http.StatusNotFound,
			wantOwner: jsHooksUserAOrg,
			wantName:  "rules-runner",
		},
		{
			name:         "org owner cannot clear admin_managed",
			email:        jsHooksUserA,
			adminManaged: true,
			body:         `{"admin_managed":false}`,
			wantCode:     http.StatusNotFound,
			wantOwner:    jsHooksUserAOrg,
			wantAdmin:    true,
			wantName:     "rules-runner",
		},
		{
			name:      "non-member cannot self-grant admin_managed",
			email:     rulesTestUserB,
			body:      `{"admin_managed":true,"published":true}`,
			wantCode:  http.StatusNotFound,
			wantOwner: jsHooksUserAOrg,
			wantName:  "rules-runner",
		},
		{
			name:          "org owner can still edit ordinary fields",
			email:         jsHooksUserA,
			body:          `{"name":"renamed-runner","published":true}`,
			wantCode:      http.StatusOK,
			wantOwner:     jsHooksUserAOrg,
			wantName:      "renamed-runner",
			wantPublished: true,
		},
		{
			name:      "superuser can set admin_managed",
			email:     "admin@example.org",
			superuser: true,
			body:      `{"admin_managed":true}`,
			wantCode:  http.StatusOK,
			wantOwner: jsHooksUserAOrg,
			wantAdmin: true,
			wantName:  "rules-runner",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := newRulesTestApp(t)
			runner := createRulesTestRunner(t, app, tc.adminManaged)
			mux := buildRulesTestMux(t, app)

			rec := serveRulesRequest(
				mux,
				http.MethodPatch,
				"/api/collections/mobile_runners/records/"+runner.Id,
				tc.body,
				rulesTestToken(t, app, tc.email, tc.superuser),
			)
			require.Equal(t, tc.wantCode, rec.Code, rec.Body.String())

			stored, err := app.FindRecordById("mobile_runners", runner.Id)
			require.NoError(t, err)
			assert.Equal(t, tc.wantOwner, stored.GetString("owner"))
			assert.Equal(t, tc.wantAdmin, stored.GetBool("admin_managed"))
			assert.Equal(t, tc.wantName, stored.GetString("name"))
			assert.Equal(t, tc.wantPublished, stored.GetBool("published"))
		})
	}
}

func TestMobileRunnersCreateRuleGuardsAdminManaged(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCode  int
		wantAdmin bool
	}{
		{
			name: "org owner cannot create an admin_managed runner",
			body: `{"owner":"` + jsHooksUserAOrg +
				`","name":"created-runner","ip":"127.0.0.1","admin_managed":true}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "org owner can create an ordinary runner",
			body:     `{"owner":"` + jsHooksUserAOrg + `","name":"created-runner","ip":"127.0.0.1"}`,
			wantCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app := newRulesTestApp(t)
			mux := buildRulesTestMux(t, app)

			rec := serveRulesRequest(
				mux,
				http.MethodPost,
				"/api/collections/mobile_runners/records",
				tc.body,
				rulesTestToken(t, app, jsHooksUserA, false),
			)
			require.Equal(t, tc.wantCode, rec.Code, rec.Body.String())

			stored, err := app.FindAllRecords(
				"mobile_runners",
				dbx.HashExp{"name": "created-runner"},
			)
			require.NoError(t, err)
			if tc.wantCode != http.StatusOK {
				assert.Empty(t, stored)
				return
			}
			var created struct {
				ID string `json:"id"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
			require.Len(t, stored, 1)
			assert.Equal(t, created.ID, stored[0].Id)
			assert.Equal(t, tc.wantAdmin, stored[0].GetBool("admin_managed"))
		})
	}
}

func newRulesTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	return app
}

func createRulesTestRunner(t *testing.T, app core.App, adminManaged bool) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	record.Set("owner", jsHooksUserAOrg)
	record.Set("name", "rules-runner")
	record.Set("canonified_name", "rules-runner")
	record.Set("ip", "127.0.0.1")
	record.Set("admin_managed", adminManaged)
	require.NoError(t, app.Save(record))
	return record
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

func rulesTestToken(t *testing.T, app core.App, email string, superuser bool) string {
	t.Helper()
	collection := "users"
	if superuser {
		collection = core.CollectionNameSuperusers
	}
	user, err := app.FindAuthRecordByEmail(collection, email)
	require.NoError(t, err)
	token, err := user.NewAuthToken()
	require.NoError(t, err)
	return token
}

func serveRulesRequest(
	mux http.Handler,
	method, path, body, token string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}
