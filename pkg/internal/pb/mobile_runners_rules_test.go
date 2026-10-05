// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
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

const (
	rulesTestUserB     = "userB@example.org"
	rulesTestSuperuser = "admin@example.org"
	rulesTestHeartbeat = "2026-10-05 10:00:00.000Z"
)

// rulesTestActor is who sends a collection API request in the rules tests.
type rulesTestActor int

const (
	actorOrgOwner rulesTestActor = iota
	actorNonMember
	actorSuperuser
)

func TestMobileRunnersUpdateRuleGuardsPrivilegedFields(t *testing.T) {
	cases := []struct {
		name     string
		actor    rulesTestActor
		body     string
		wantCode int
		// want lists the stored fields that differ from the seeded runner.
		want map[string]any
	}{
		{
			name:     "org owner cannot self-grant admin_managed",
			body:     `{"admin_managed":true,"published":true}`,
			wantCode: http.StatusNotFound,
		},
		{
			name:     "org owner cannot move the runner to another org",
			body:     `{"owner":"` + jsHooksUserBOrg + `"}`,
			wantCode: http.StatusNotFound,
		},
		{
			name:     "org owner cannot re-enable a disabled runner",
			body:     `{"disabled":false}`,
			wantCode: http.StatusNotFound,
		},
		{
			name:     "org owner cannot fake liveness",
			body:     `{"online":true,"last_heartbeat_at":"` + rulesTestHeartbeat + `"}`,
			wantCode: http.StatusNotFound,
		},
		{
			name:     "non-member cannot self-grant admin_managed",
			actor:    actorNonMember,
			body:     `{"admin_managed":true,"published":true}`,
			wantCode: http.StatusNotFound,
		},
		{
			name:     "org owner can still edit ordinary fields",
			body:     `{"name":"renamed-runner","published":true}`,
			wantCode: http.StatusOK,
			want:     map[string]any{"name": "renamed-runner", "published": true},
		},
		{
			name:     "superuser can change privileged fields",
			actor:    actorSuperuser,
			body:     `{"admin_managed":true,"disabled":false}`,
			wantCode: http.StatusOK,
			want:     map[string]any{"admin_managed": true, "disabled": false},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newRulesTestApp(t)
			runner := createRulesTestRunner(t, app, jsHooksUserAOrg, "rules-runner")
			// seed the server-owned fields so clearing them is observable
			runner.Set("disabled", true)
			require.NoError(t, app.Save(runner))

			rec := serveRulesRequest(
				buildRulesTestMux(t, app),
				http.MethodPatch,
				"/api/collections/mobile_runners/records/"+runner.Id,
				tc.body,
				rulesTestToken(t, app, tc.actor),
			)
			require.Equal(t, tc.wantCode, rec.Code, rec.Body.String())

			want := map[string]any{
				"owner":             jsHooksUserAOrg,
				"name":              "rules-runner",
				"published":         false,
				"admin_managed":     false,
				"disabled":          true,
				"online":            false,
				"last_heartbeat_at": "",
			}
			for field, value := range tc.want {
				want[field] = value
			}
			assertStoredFields(t, app, "mobile_runners", runner.Id, want)
		})
	}
}

func TestMobileRunnersCreateRuleGuardsPrivilegedFields(t *testing.T) {
	cases := []struct {
		name     string
		extra    string
		wantCode int
	}{
		{name: "org owner cannot create an admin_managed runner", extra: `"admin_managed":true`},
		{name: "org owner cannot create a runner as online", extra: `"online":true`},
		{
			name:  "org owner cannot create a runner with a heartbeat",
			extra: `"last_heartbeat_at":"` + rulesTestHeartbeat + `"`,
		},
		{name: "org owner cannot create a disabled runner", extra: `"disabled":true`},
		{name: "org owner can create an ordinary runner", wantCode: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantCode == 0 {
				tc.wantCode = http.StatusBadRequest
			}
			body := `{"owner":"` + jsHooksUserAOrg + `","name":"created-runner","ip":"127.0.0.1"`
			if tc.extra != "" {
				body += "," + tc.extra
			}
			body += "}"

			app := newRulesTestApp(t)
			rec := serveRulesRequest(
				buildRulesTestMux(t, app),
				http.MethodPost,
				"/api/collections/mobile_runners/records",
				body,
				rulesTestToken(t, app, actorOrgOwner),
			)
			require.Equal(t, tc.wantCode, rec.Code, rec.Body.String())
			assertCreatedCount(t, app, "mobile_runners", "created-runner", tc.wantCode)
		})
	}
}

func TestMobileDevicesUpdateRuleGuardsBindingAndLiveness(t *testing.T) {
	cases := []struct {
		name     string
		actor    rulesTestActor
		body     func(otherRunnerID string) string
		wantCode int
		want     map[string]any
		// rebound expects the device stored under org B's runner.
		rebound bool
	}{
		{
			name:     "org owner cannot move the device to another org",
			body:     func(string) string { return `{"owner":"` + jsHooksUserBOrg + `"}` },
			wantCode: http.StatusNotFound,
		},
		{
			name:     "org owner cannot rebind the device to another runner",
			body:     func(other string) string { return `{"runner":"` + other + `"}` },
			wantCode: http.StatusNotFound,
		},
		{
			name:     "org owner cannot fake liveness",
			body:     func(string) string { return `{"online":true}` },
			wantCode: http.StatusNotFound,
		},
		{
			name:     "non-member cannot edit the device",
			actor:    actorNonMember,
			body:     func(string) string { return `{"description":"hijacked"}` },
			wantCode: http.StatusNotFound,
		},
		{
			name:     "org owner can still edit ordinary fields",
			body:     func(string) string { return `{"description":"lab phone","live_stream":true}` },
			wantCode: http.StatusOK,
			want:     map[string]any{"description": "lab phone", "live_stream": true},
		},
		{
			name: "superuser can rebind the device",
			body: func(other string) string {
				return `{"owner":"` + jsHooksUserBOrg + `","runner":"` + other + `"}`
			},
			actor:    actorSuperuser,
			wantCode: http.StatusOK,
			rebound:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newRulesTestApp(t)
			runner := createRulesTestRunner(t, app, jsHooksUserAOrg, "rules-runner")
			otherRunner := createRulesTestRunner(t, app, jsHooksUserBOrg, "other-runner")
			device := createRulesTestDevice(t, app, runner)

			rec := serveRulesRequest(
				buildRulesTestMux(t, app),
				http.MethodPatch,
				"/api/collections/mobile_devices/records/"+device.Id,
				tc.body(otherRunner.Id),
				rulesTestToken(t, app, tc.actor),
			)
			require.Equal(t, tc.wantCode, rec.Code, rec.Body.String())

			want := map[string]any{
				"owner":       jsHooksUserAOrg,
				"runner":      runner.Id,
				"online":      false,
				"description": "",
				"live_stream": false,
			}
			for field, value := range tc.want {
				want[field] = value
			}
			if tc.rebound {
				want["owner"] = jsHooksUserBOrg
				want["runner"] = otherRunner.Id
			}
			assertStoredFields(t, app, "mobile_devices", device.Id, want)
		})
	}
}

func TestMobileDevicesCreateRuleGuardsBindingAndLiveness(t *testing.T) {
	cases := []struct {
		name      string
		otherOrg  bool
		extra     string
		wantCode  int
		wantCount int
	}{
		{name: "org owner cannot bind a device to another org's runner", otherOrg: true},
		{name: "org owner cannot create a device as online", extra: `"online":true`},
		{name: "org owner can create a device on its own runner", wantCode: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantCode == 0 {
				tc.wantCode = http.StatusBadRequest
			}
			app := newRulesTestApp(t)
			runner := createRulesTestRunner(t, app, jsHooksUserAOrg, "rules-runner")
			if tc.otherOrg {
				runner = createRulesTestRunner(t, app, jsHooksUserBOrg, "other-runner")
			}
			body := `{"owner":"` + jsHooksUserAOrg + `","runner":"` + runner.Id +
				`","name":"created-device","canonified_name":"created-device","type":"android_emulator"`
			if tc.extra != "" {
				body += "," + tc.extra
			}
			body += "}"

			rec := serveRulesRequest(
				buildRulesTestMux(t, app),
				http.MethodPost,
				"/api/collections/mobile_devices/records",
				body,
				rulesTestToken(t, app, actorOrgOwner),
			)
			require.Equal(t, tc.wantCode, rec.Code, rec.Body.String())
			assertCreatedCount(t, app, "mobile_devices", "created-device", tc.wantCode)
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

func createRulesTestRunner(t *testing.T, app core.App, ownerID, name string) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	record.Set("owner", ownerID)
	record.Set("name", name)
	record.Set("canonified_name", name)
	record.Set("ip", "127.0.0.1")
	require.NoError(t, app.Save(record))
	return record
}

func createRulesTestDevice(t *testing.T, app core.App, runner *core.Record) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	record.Set("owner", runner.GetString("owner"))
	record.Set("runner", runner.Id)
	record.Set("name", "rules-device")
	record.Set("canonified_name", "rules-device")
	record.Set("type", "android_emulator")
	require.NoError(t, app.Save(record))
	return record
}

func assertStoredFields(
	t *testing.T,
	app core.App,
	collection, id string,
	want map[string]any,
) {
	t.Helper()
	stored, err := app.FindRecordById(collection, id)
	require.NoError(t, err)
	for field, value := range want {
		if b, ok := value.(bool); ok {
			assert.Equal(t, b, stored.GetBool(field), field)
			continue
		}
		assert.Equal(t, value, stored.GetString(field), field)
	}
}

func assertCreatedCount(t *testing.T, app core.App, collection, name string, code int) {
	t.Helper()
	stored, err := app.FindAllRecords(collection, dbx.HashExp{"name": name})
	require.NoError(t, err)
	if code == http.StatusOK {
		assert.Len(t, stored, 1)
		return
	}
	assert.Empty(t, stored)
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

func rulesTestToken(t *testing.T, app core.App, actor rulesTestActor) string {
	t.Helper()
	collection, email := "users", jsHooksUserA
	switch actor {
	case actorNonMember:
		email = rulesTestUserB
	case actorSuperuser:
		collection, email = core.CollectionNameSuperusers, rulesTestSuperuser
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
