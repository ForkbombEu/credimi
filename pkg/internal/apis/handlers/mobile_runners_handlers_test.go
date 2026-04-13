// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func setupMobileRunnerApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	MobileRunnerRegistrationRoutes.Add(app)
	MobileRunnersTemporalInternalRoutes.Add(app)
	seedInternalAdminKey(t, app)

	return app
}

func performMobileRunnerRequest(
	t testing.TB,
	app *tests.TestApp,
	auth *core.Record,
	url string,
	validatedInput any,
) *core.RequestEvent {
	t.Helper()

	var requestBody *bytes.Reader
	if validatedInput != nil {
		payload, err := json.Marshal(validatedInput)
		require.NoError(t, err)
		requestBody = bytes.NewReader(payload)
	} else {
		requestBody = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(http.MethodPost, url, requestBody)
	rec := httptest.NewRecorder()
	if validatedInput != nil {
		req = req.WithContext(
			context.WithValue(req.Context(), middlewares.ValidatedInputKey, validatedInput),
		)
	}

	return &core.RequestEvent{
		App:  app,
		Auth: auth,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}
}

func decodeJSONBody(t testing.TB, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &decoded))
	return decoded
}

func responseRecorder(t testing.TB, event *core.RequestEvent) *httptest.ResponseRecorder {
	t.Helper()

	recorder, ok := event.Response.(*httptest.ResponseRecorder)
	require.True(t, ok)
	return recorder
}

func TestGetMobileRunner(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing runner_identifier parameter",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner",
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"runner_identifier"`,
				`"runner_identifier is required"`,
			},
			TestAppFactory: setupMobileRunnerApp,
		},
		{
			Name:           "nonexistent runner identifier",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner?runner_identifier=does-not-exist",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"mobile runner not found"`,
			},
			TestAppFactory: setupMobileRunnerApp,
		},
		{
			Name:           "valid runner identifier",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner?runner_identifier=usera-s-organization/test-runner",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"type"`,
				`"runner_url"`,
				`"serial"`,
				`android_phone`,
				`https://192.168.1.10:8050`,
				`SERIAL123`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupMobileRunnerApp(t)

				coll, err := app.FindCollectionByNameOrId("mobile_runners")
				require.NoError(t, err)

				record := core.NewRecord(coll)
				record.Set("owner", orgID)
				record.Set("type", "android_phone")
				record.Set("serial", "SERIAL123")
				record.Set("ip", "https://192.168.1.10")
				record.Set("port", "8050")
				record.Set("name", "test-runner")

				require.NoError(t, app.Save(record))

				return app
			},
		},
		{
			Name:           "valid runner identifier with no port",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner?runner_identifier=usera-s-organization/no-port-runner",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"type"`,
				`"runner_url"`,
				`"serial"`,
				`android_emulator`,
				`http://192.168.1.20`,
				`SERIAL999`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupMobileRunnerApp(t)

				coll, err := app.FindCollectionByNameOrId("mobile_runners")
				require.NoError(t, err)

				record := core.NewRecord(coll)
				record.Set("owner", orgID)
				record.Set("type", "android_emulator")
				record.Set("serial", "SERIAL999")
				record.Set("ip", "http://192.168.1.20")
				record.Set("name", "no-port-runner")

				require.NoError(t, app.Save(record))

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		if scenario.Headers == nil {
			scenario.Headers = map[string]string{}
		}
		scenario.Headers["Credimi-Api-Key"] = "internal-test-api-key"
		scenario.Test(t)
	}
}

func TestListMobileRunnerURLs(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "empty runners list",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner/list-urls",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"runners":[]`,
			},
			TestAppFactory: setupMobileRunnerApp,
		},
		{
			Name:           "multiple runners",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner/list-urls",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"runners"`,
				`http://192.168.1.10`,
				`https://192.168.1.11:9000`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupMobileRunnerApp(t)

				coll, err := app.FindCollectionByNameOrId("mobile_runners")
				require.NoError(t, err)

				// Runner 1
				r1 := core.NewRecord(coll)
				r1.Set("owner", orgID)
				r1.Set("serial", "SERIAL1")
				r1.Set("ip", "http://192.168.1.10")
				r1.Set("name", "runner-1")

				// Runner 2
				r2 := core.NewRecord(coll)
				r2.Set("owner", orgID)
				r2.Set("serial", "SERIAL2")
				r2.Set("ip", "https://192.168.1.11")
				r2.Set("port", "9000")
				r2.Set("name", "runner-2")

				require.NoError(t, app.Save(r1))
				require.NoError(t, app.Save(r2))

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		if scenario.Headers == nil {
			scenario.Headers = map[string]string{}
		}
		scenario.Headers["Credimi-Api-Key"] = "internal-test-api-key"
		scenario.Test(t)
	}
}

func TestGetMobileRunnerSemaphore(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing runner_identifier parameter",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner/semaphore",
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"runner_identifier"`,
				`"runner_identifier is required"`,
			},
			TestAppFactory: setupMobileRunnerApp,
		},
		{
			Name:           "nonexistent runner identifier",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner/semaphore?runner_identifier=does-not-exist",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"mobile runner not found"`,
			},
			TestAppFactory: setupMobileRunnerApp,
		},
		{
			Name:           "semaphore not found for runner",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner/semaphore?runner_identifier=usera-s-organization/runner-without-semaphore",
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"runner semaphore not found"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupMobileRunnerApp(t)

				coll, err := app.FindCollectionByNameOrId("mobile_runners")
				require.NoError(t, err)

				record := core.NewRecord(coll)
				record.Set("owner", orgID)
				record.Set("serial", "SERIAL123")
				record.Set("ip", "https://192.168.1.10")
				record.Set("port", "8050")
				record.Set("name", "runner-without-semaphore")
				require.NoError(t, app.Save(record))

				originalQuery := queryMobileRunnerSemaphoreState
				queryMobileRunnerSemaphoreState = func(_ context.Context, _ string) (workflows.MobileRunnerSemaphoreStateView, error) {
					return workflows.MobileRunnerSemaphoreStateView{}, errSemaphoreNotFound
				}
				t.Cleanup(func() {
					queryMobileRunnerSemaphoreState = originalQuery
				})

				return app
			},
		},
		{
			Name:           "semaphore state returned",
			Method:         http.MethodGet,
			URL:            "/api/mobile-runner/semaphore?runner_identifier=usera-s-organization/test-semaphore-runner",
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"runner_id":"test-semaphore-runner"`,
				`"capacity":1`,
				`"slots_used":1`,
				`"queue_len":2`,
				`"in_use":true`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupMobileRunnerApp(t)

				coll, err := app.FindCollectionByNameOrId("mobile_runners")
				require.NoError(t, err)

				record := core.NewRecord(coll)
				record.Set("owner", orgID)
				record.Set("serial", "SERIAL321")
				record.Set("ip", "https://192.168.1.99")
				record.Set("port", "9000")
				record.Set("name", "test-semaphore-runner")
				require.NoError(t, app.Save(record))

				originalQuery := queryMobileRunnerSemaphoreState
				queryMobileRunnerSemaphoreState = func(_ context.Context, _ string) (workflows.MobileRunnerSemaphoreStateView, error) {
					return workflows.MobileRunnerSemaphoreStateView{
						RunnerID:  "test-semaphore-runner",
						Capacity:  1,
						SlotsUsed: 1,
						QueueLen:  2,
					}, nil
				}
				t.Cleanup(func() {
					queryMobileRunnerSemaphoreState = originalQuery
				})

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		if scenario.Headers == nil {
			scenario.Headers = map[string]string{}
		}
		scenario.Headers["Credimi-Api-Key"] = "internal-test-api-key"
		scenario.Test(t)
	}
}

func TestPreviewMobileRunnerID(t *testing.T) {
	t.Run("user preview uses user organization and increments canonified name", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)
		orgID, err := GetUserOrganizationID(app, user.Id)
		require.NoError(t, err)

		coll, err := app.FindCollectionByNameOrId("mobile_runners")
		require.NoError(t, err)

		record := core.NewRecord(coll)
		record.Set("owner", orgID)
		record.Set("name", "Test Runner")
		record.Set("ip", "https://existing.example")
		require.NoError(t, app.Save(record))

		event := performMobileRunnerRequest(
			t,
			app,
			user,
			"/api/mobile-runner/preview-id",
			PreviewMobileRunnerIDRequest{Name: "Test Runner"},
		)

		err = HandlePreviewMobileRunnerID()(event)
		require.NoError(t, err)

		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusOK, recorder.Code)

		body := decodeJSONBody(t, recorder)
		require.Equal(t, "usera-s-organization", body["organization"])
		require.Equal(t, "test-runner-1", body["canonified_name"])
		require.Equal(t, "/usera-s-organization/test-runner-1", body["runner_id"])
	})

	t.Run("admin preview requires an explicit organization", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
		require.NoError(t, err)

		event := performMobileRunnerRequest(
			t,
			app,
			superuser,
			"/api/mobile-runner/preview-id",
			PreviewMobileRunnerIDRequest{Name: "Runner One"},
		)

		err = HandlePreviewMobileRunnerID()(event)
		require.NoError(t, err)

		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Contains(t, recorder.Body.String(), "organization is required")
	})

	t.Run("admin preview can target another organization", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
		require.NoError(t, err)

		event := performMobileRunnerRequest(
			t,
			app,
			superuser,
			"/api/mobile-runner/preview-id",
			PreviewMobileRunnerIDRequest{
				Organization: "userb-s-organization",
				Name:         "Runner B",
			},
		)

		err = HandlePreviewMobileRunnerID()(event)
		require.NoError(t, err)

		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusOK, recorder.Code)

		body := decodeJSONBody(t, recorder)
		require.Equal(t, "/userb-s-organization/runner-b", body["runner_id"])
	})
}

func TestUpsertMobileRunner(t *testing.T) {
	t.Run("user create stores a new runner", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)

		published := true
		event := performMobileRunnerRequest(
			t,
			app,
			user,
			"/api/mobile-runner",
			UpsertMobileRunnerRequest{
				Name:        "My Phone",
				IP:          "https://runner.example.trycloudflare.com",
				Description: "lab device",
				Published:   &published,
			},
		)

		err = HandleUpsertMobileRunner()(event)
		require.NoError(t, err)

		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusOK, recorder.Code)

		body := decodeJSONBody(t, recorder)
		require.Equal(t, "/usera-s-organization/my-phone", body["runner_id"])

		record, err := canonify.Resolve(app, "/usera-s-organization/my-phone")
		require.NoError(t, err)
		require.Equal(t, "lab device", record.GetString("description"))
		require.Equal(t, "https://runner.example.trycloudflare.com", record.GetString("ip"))
		require.True(t, record.GetBool("published"))
	})

	t.Run("runner_id update keeps the same record", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)
		orgID, err := GetUserOrganizationID(app, user.Id)
		require.NoError(t, err)

		coll, err := app.FindCollectionByNameOrId("mobile_runners")
		require.NoError(t, err)

		record := core.NewRecord(coll)
		record.Set("owner", orgID)
		record.Set("name", "My Phone")
		record.Set("ip", "https://old.example")
		require.NoError(t, app.Save(record))
		recordID := record.Id

		event := performMobileRunnerRequest(
			t,
			app,
			user,
			"/api/mobile-runner",
			UpsertMobileRunnerRequest{
				RunnerID:     "/usera-s-organization/my-phone",
				Name:         "My Phone",
				IP:           "https://new.example",
				Description:  "updated",
				Organization: "ignored-for-user",
			},
		)

		err = HandleUpsertMobileRunner()(event)
		require.NoError(t, err)

		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusOK, recorder.Code)

		updated, err := app.FindRecordById("mobile_runners", recordID)
		require.NoError(t, err)
		require.Equal(t, "https://new.example", updated.GetString("ip"))
		require.Equal(t, "updated", updated.GetString("description"))
	})

	t.Run("admin create requires matching preview runner_id", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
		require.NoError(t, err)

		event := performMobileRunnerRequest(
			t,
			app,
			superuser,
			"/api/mobile-runner",
			UpsertMobileRunnerRequest{
				RunnerID:     "/userb-s-organization/conflicting-id",
				Organization: "userb-s-organization",
				Name:         "Runner B",
				IP:           "https://runner-b.example",
			},
		)

		err = HandleUpsertMobileRunner()(event)
		require.NoError(t, err)

		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusConflict, recorder.Code)
		require.Contains(t, recorder.Body.String(), "does not match the next available id")
	})
}
