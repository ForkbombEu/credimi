// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
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
	MobileRunnersPublicRoutes.Add(app)
	MobileRunnerRegistrationRoutes.Add(app)
	MobileRunnerLifecycleRoutes.Add(app)
	seedInternalAdminKey(t, app)
	t.Setenv(mobilerunner.CredentialSecretEnvVar, "test-runner-credential-secret")

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

func mobileRunnerProbeRecord(runnerURL string, adminManaged bool) *core.Record {
	record := core.NewRecord(core.NewBaseCollection("mobile_runners"))
	record.Set("ip", runnerURL)
	record.Set("admin_managed", adminManaged)
	return record
}

func TestCheckMobileRunnerHealthHTTP(t *testing.T) {
	t.Run("empty url is classified as malformed", func(t *testing.T) {
		online, devices, err := checkMobileRunnerHealthHTTP(
			t.Context(),
			mobileRunnerProbeRecord(" ", false),
		)
		require.ErrorIs(t, err, errMalformedMobileRunnerURL)
		require.False(t, online)
		require.Nil(t, devices)
	})

	t.Run("url without a scheme is classified as malformed", func(t *testing.T) {
		online, devices, err := checkMobileRunnerHealthHTTP(
			t.Context(),
			mobileRunnerProbeRecord("192.168.1.10:8050", false),
		)
		require.ErrorIs(t, err, errMalformedMobileRunnerURL)
		require.False(t, online)
		require.Nil(t, devices)
	})

	t.Run("healthy runner returns devices without credentials", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/health", r.URL.Path)
			require.Empty(t, r.Header.Get(APIKeyHeaderName))
			_, _ = io.WriteString(
				w,
				`{"devices":[{"serial":"ABC123","state":"device","model":"Pixel 8"}]}`,
			)
		}))
		t.Cleanup(server.Close)

		online, devices, err := checkMobileRunnerHealthHTTP(
			t.Context(),
			mobileRunnerProbeRecord(server.URL, true),
		)
		require.NoError(t, err)
		require.True(t, online)
		require.Len(t, devices, 1)
		require.Equal(t, "ABC123", devices[0].Serial)
		require.Equal(t, "device", devices[0].State)
		require.Equal(t, "Pixel 8", devices[0].Model)
	})
}

// A tenant chooses its runner's address, so probes must never reach a
// loopback or private destination on its behalf; admin-managed runners may.
func TestRunnerHealthProbesBlockTenantLoopbackRunners(t *testing.T) {
	probes := []struct {
		name  string
		probe func(context.Context, *core.Record) (bool, error)
	}{
		{
			name: "runner list probe",
			probe: func(ctx context.Context, runner *core.Record) (bool, error) {
				online, _, err := checkMobileRunnerHealthHTTP(ctx, runner)
				return online, err
			},
		},
		{name: "run path probe", probe: checkRunnerReachableHTTP},
	}
	for _, tc := range probes {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					_, _ = io.WriteString(w, `{"devices":[]}`)
				}),
			)
			t.Cleanup(server.Close)

			online, err := tc.probe(t.Context(), mobileRunnerProbeRecord(server.URL, false))
			require.NoError(t, err)
			require.False(t, online)
			require.Zero(t, requests.Load())

			online, err = tc.probe(t.Context(), mobileRunnerProbeRecord(server.URL, true))
			require.NoError(t, err)
			require.True(t, online)
			require.Equal(t, int32(1), requests.Load())
		})
	}
}

func TestListMobileRunners(t *testing.T) {
	t.Run("excludes disabled runners", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)
		orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
		require.NoError(t, err)

		createMobileRunnerRecord(t, app, orgID, "enabled-runner", "http://127.0.0.1:1", false)
		createMobileRunnerRecord(t, app, orgID, "disabled-runner", "http://127.0.0.1:1", false)
		disabled, err := app.FindFirstRecordByFilter(
			"mobile_runners",
			"name = {:name}",
			map[string]any{"name": "disabled-runner"},
		)
		require.NoError(t, err)
		disabled.Set("disabled", true)
		require.NoError(t, app.Save(disabled))

		req := httptest.NewRequest(http.MethodGet, "/api/mobile-runners?view=selector", nil)
		rec := httptest.NewRecorder()
		err = HandleListMobileRunners()(&core.RequestEvent{
			App: app, Auth: user, Event: router.Event{Request: req, Response: rec},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)

		var response ListMobileRunnersPublicResponseSchema
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		require.Len(t, response.Runners, 1)
		require.Equal(t, "usera-s-organization/enabled-runner", response.Runners[0].Path)
	})

	t.Run("user sees owned and public runners with health and queue details", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)
		userOrgID, err := pbutils.GetUserOrganizationID(app, user.Id)
		require.NoError(t, err)
		setOrganizationPublished(t, app, userOrgID, true)
		otherOrg := createOtherWalletAPKOrganization(t, app)

		createMobileRunnerRecord(t, app, userOrgID, "owned-offline", "offline-runner", false)
		createMobileRunnerRecord(t, app, userOrgID, "owned-online", "online-owned", false)
		createMobileRunnerRecord(t, app, otherOrg.Id, "other-private", "online-private", false)
		createMobileRunnerRecord(t, app, otherOrg.Id, "other-public", "online-public", true)

		originalHealth := checkMobileRunnerHealth
		checkMobileRunnerHealth = func(_ context.Context, runner *core.Record) (bool, []MobileRunnerHealthDevice, error) {
			if mobilerunner.RunnerURL(runner) == "offline-runner" {
				return false, nil, nil
			}
			return true, []MobileRunnerHealthDevice{
				{Serial: "ABC123", State: "device", Model: "Pixel_8"},
			}, nil
		}
		t.Cleanup(func() {
			checkMobileRunnerHealth = originalHealth
		})

		originalQuery := queryMobileDeviceSemaphoreState
		queryMobileDeviceSemaphoreState = func(_ context.Context, runnerID string) (workflows.MobileDeviceSemaphoreStateView, error) {
			if runnerID == "usera-s-organization/owned-online" {
				return workflows.MobileDeviceSemaphoreStateView{
					DeviceID: runnerID,
					QueueLen: 3,
				}, nil
			}
			return workflows.MobileDeviceSemaphoreStateView{DeviceID: runnerID, QueueLen: 1}, nil
		}
		t.Cleanup(func() {
			queryMobileDeviceSemaphoreState = originalQuery
		})

		req := httptest.NewRequest(http.MethodGet, "/api/mobile-runners", nil)
		rec := httptest.NewRecorder()
		event := &core.RequestEvent{
			App:  app,
			Auth: user,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		}

		err = HandleListMobileRunners()(event)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)

		var response ListMobileRunnersPublicResponseSchema
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		require.Len(t, response.Runners, 3)

		var raw map[string][]map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))
		require.NotContains(t, raw["runners"][0], "id")
		require.NotContains(t, raw["runners"][0], "organization")
		require.NotContains(t, raw["runners"][0], "canonified_name")
		require.NotContains(t, raw["runners"][0], "serial")

		require.Equal(t, "usera-s-organization/owned-online", response.Runners[0].Path)
		require.Equal(t, "owned-online", response.Runners[0].Name)
		require.True(t, response.Runners[0].IsOwned)
		require.Equal(t, "online", response.Runners[0].HealthStatus)
		require.NotNil(t, response.Runners[0].QueueLength)
		require.Equal(t, 3, *response.Runners[0].QueueLength)
		require.Equal(t, []MobileRunnerHealthDevice{
			{Serial: "ABC123", State: "device", Model: "Pixel_8"},
		}, response.Runners[0].Devices)

		require.Equal(t, "usera-s-organization/owned-offline", response.Runners[1].Path)
		require.True(t, response.Runners[1].IsOwned)
		require.Equal(t, "offline", response.Runners[1].HealthStatus)
		require.Nil(t, response.Runners[1].QueueLength)

		require.Equal(t, "other-org/other-public", response.Runners[2].Path)
		require.False(t, response.Runners[2].IsOwned)
		require.Equal(t, "online", response.Runners[2].HealthStatus)
		require.NotNil(t, response.Runners[2].QueueLength)
		require.Equal(t, 1, *response.Runners[2].QueueLength)
	})

	t.Run("admin key sees every runner", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		userOrgID, err := getOrgIDfromName("userA's organization")
		require.NoError(t, err)
		otherOrg := createOtherWalletAPKOrganization(t, app)

		createMobileRunnerRecord(t, app, userOrgID, "owned-runner", "http://127.0.0.1:1", false)
		createMobileRunnerRecord(t, app, otherOrg.Id, "other-private", "http://127.0.0.1:1", false)

		originalHealth := checkMobileRunnerHealth
		checkMobileRunnerHealth = func(context.Context, *core.Record) (bool, []MobileRunnerHealthDevice, error) {
			return false, nil, nil
		}
		t.Cleanup(func() {
			checkMobileRunnerHealth = originalHealth
		})

		superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/api/mobile-runners", nil)
		rec := httptest.NewRecorder()
		event := &core.RequestEvent{
			App:  app,
			Auth: superuser,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		}

		err = HandleListMobileRunners()(event)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)

		var response ListMobileRunnersPublicResponseSchema
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		require.Len(t, response.Runners, 2)
		require.ElementsMatch(t, []string{
			"usera-s-organization/owned-runner",
			"other-org/other-private",
		}, []string{
			response.Runners[0].Path,
			response.Runners[1].Path,
		})
	})

	t.Run(
		"normal user visibility depends only on ownership and published state",
		func(t *testing.T) {
			app := setupMobileRunnerApp(t)
			defer app.Cleanup()

			user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)
			userOrgID, err := pbutils.GetUserOrganizationID(app, user.Id)
			require.NoError(t, err)
			setOrganizationPublished(t, app, userOrgID, true)
			otherOrg := createOtherWalletAPKOrganization(t, app)

			createMobileRunnerRecord(
				t,
				app,
				userOrgID,
				"owned-private",
				"http://127.0.0.1:1",
				false,
			)
			createMobileRunnerRecord(
				t,
				app,
				otherOrg.Id,
				"other-published",
				"http://127.0.0.1:1",
				true,
			)
			createMobileRunnerRecord(
				t,
				app,
				otherOrg.Id,
				"other-private",
				"http://127.0.0.1:1",
				false,
			)
			createMobileRunnerRecord(
				t,
				app,
				otherOrg.Id,
				"admin-published",
				"http://127.0.0.1:1",
				true,
			)
			createMobileRunnerRecord(
				t,
				app,
				otherOrg.Id,
				"admin-private",
				"http://127.0.0.1:1",
				false,
			)

			setMobileRunnerAdminManaged(t, app, "other-org/admin-published", true)
			setMobileRunnerAdminManaged(t, app, "other-org/admin-private", true)

			originalHealth := checkMobileRunnerHealth
			checkMobileRunnerHealth = func(context.Context, *core.Record) (bool, []MobileRunnerHealthDevice, error) {
				return false, nil, nil
			}
			t.Cleanup(func() {
				checkMobileRunnerHealth = originalHealth
			})

			req := httptest.NewRequest(http.MethodGet, "/api/mobile-runners", nil)
			rec := httptest.NewRecorder()
			event := &core.RequestEvent{
				App:  app,
				Auth: user,
				Event: router.Event{
					Request:  req,
					Response: rec,
				},
			}

			err = HandleListMobileRunners()(event)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, rec.Code)

			var response ListMobileRunnersPublicResponseSchema
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			require.ElementsMatch(t, []string{
				"usera-s-organization/owned-private",
				"other-org/other-published",
				"other-org/admin-published",
			}, []string{
				response.Runners[0].Path,
				response.Runners[1].Path,
				response.Runners[2].Path,
			})
		},
	)

	t.Run(
		"unpublished normal user organization sees owned and published admin-managed runners",
		func(t *testing.T) {
			app := setupMobileRunnerApp(t)
			defer app.Cleanup()

			user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)
			userOrgID, err := pbutils.GetUserOrganizationID(app, user.Id)
			require.NoError(t, err)
			setOrganizationPublished(t, app, userOrgID, false)
			otherOrg := createOtherWalletAPKOrganization(t, app)

			createMobileRunnerRecord(
				t,
				app,
				userOrgID,
				"owned-private",
				"http://127.0.0.1:1",
				false,
			)
			createMobileRunnerRecord(
				t,
				app,
				otherOrg.Id,
				"other-published",
				"http://127.0.0.1:1",
				true,
			)
			createMobileRunnerRecord(
				t,
				app,
				otherOrg.Id,
				"admin-published",
				"http://127.0.0.1:1",
				true,
			)
			setMobileRunnerAdminManaged(t, app, "other-org/admin-published", true)

			originalHealth := checkMobileRunnerHealth
			checkMobileRunnerHealth = func(context.Context, *core.Record) (bool, []MobileRunnerHealthDevice, error) {
				return false, nil, nil
			}
			t.Cleanup(func() {
				checkMobileRunnerHealth = originalHealth
			})

			req := httptest.NewRequest(http.MethodGet, "/api/mobile-runners", nil)
			rec := httptest.NewRecorder()
			event := &core.RequestEvent{
				App:  app,
				Auth: user,
				Event: router.Event{
					Request:  req,
					Response: rec,
				},
			}

			err = HandleListMobileRunners()(event)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, rec.Code)

			var response ListMobileRunnersPublicResponseSchema
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			require.Len(t, response.Runners, 2)
			require.ElementsMatch(t, []string{
				"usera-s-organization/owned-private",
				"other-org/admin-published",
			}, []string{
				response.Runners[0].Path,
				response.Runners[1].Path,
			})
		},
	)

	t.Run("selector view skips queue details and sensitive runner fields", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)
		userOrgID, err := pbutils.GetUserOrganizationID(app, user.Id)
		require.NoError(t, err)

		createMobileRunnerRecord(t, app, userOrgID, "owned-online", "online-owned", false)

		originalHealth := checkMobileRunnerHealth
		checkMobileRunnerHealth = func(context.Context, *core.Record) (bool, []MobileRunnerHealthDevice, error) {
			return true, []MobileRunnerHealthDevice{
				{Serial: "ABC123", State: "device", Model: "Pixel_8"},
			}, nil
		}
		t.Cleanup(func() {
			checkMobileRunnerHealth = originalHealth
		})

		queryCalled := false
		originalQuery := queryMobileDeviceSemaphoreState
		queryMobileDeviceSemaphoreState = func(_ context.Context, runnerID string) (workflows.MobileDeviceSemaphoreStateView, error) {
			queryCalled = true
			return workflows.MobileDeviceSemaphoreStateView{DeviceID: runnerID, QueueLen: 3}, nil
		}
		t.Cleanup(func() {
			queryMobileDeviceSemaphoreState = originalQuery
		})

		req := httptest.NewRequest(http.MethodGet, "/api/mobile-runners?view=selector", nil)
		rec := httptest.NewRecorder()
		event := &core.RequestEvent{
			App:  app,
			Auth: user,
			Event: router.Event{
				Request:  req,
				Response: rec,
			},
		}

		err = HandleListMobileRunners()(event)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)
		require.False(t, queryCalled)

		var response ListMobileRunnersPublicResponseSchema
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		require.Len(t, response.Runners, 1)
		require.Equal(t, "usera-s-organization/owned-online", response.Runners[0].Path)
		require.Equal(t, "online", response.Runners[0].HealthStatus)
		require.Nil(t, response.Runners[0].QueueLength)
		require.Empty(t, response.Runners[0].Devices)
		require.Empty(t, response.Runners[0].URL)
		require.Empty(t, response.Runners[0].Type)

		var raw map[string][]map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))
		require.NotContains(t, raw["runners"][0], "queue_len")
		require.NotContains(t, raw["runners"][0], "runner_url")
		require.NotContains(t, raw["runners"][0], "runner_id")
		require.Contains(t, raw["runners"][0], "path")
		require.Contains(t, raw["runners"][0], "health_status")
		require.NotContains(t, raw["runners"][0], "devices")
		require.NotContains(t, raw["runners"][0], "type")
	})
}

func TestListMobileDevices(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	ownerID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, ownerID, "device-host", "https://runner.example", false)

	runner, err := canonify.Resolve(app, "/usera-s-organization/device-host")
	require.NoError(t, err)
	devices, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(devices)
	device.Set("owner", ownerID)
	device.Set("runner", runner.Id)
	device.Set("name", "pixel-8")
	device.Set("type", "android_phone")
	device.Set("serial", "ABC123")
	device.Set("online", true)
	require.NoError(t, app.Save(device))
	markRunnerHeartbeat(t, app, runner, time.Now())
	originalHealth := checkMobileRunnerHealth
	checkMobileRunnerHealth = func(context.Context, *core.Record) (bool, []MobileRunnerHealthDevice, error) {
		t.Fatal("the device selector must not probe runners")
		return false, nil, nil
	}
	t.Cleanup(func() { checkMobileRunnerHealth = originalHealth })

	originalQuery := queryMobileDeviceSemaphoreState
	queryMobileDeviceSemaphoreState = func(_ context.Context, deviceID string) (workflows.MobileDeviceSemaphoreStateView, error) {
		require.Equal(t, "usera-s-organization/device-host/pixel-8", deviceID)
		return workflows.MobileDeviceSemaphoreStateView{DeviceID: deviceID, QueueLen: 2}, nil
	}
	t.Cleanup(func() { queryMobileDeviceSemaphoreState = originalQuery })

	req := httptest.NewRequest(http.MethodGet, "/api/mobile-devices", nil)
	rec := httptest.NewRecorder()
	event := &core.RequestEvent{
		App:   app,
		Auth:  user,
		Event: router.Event{Request: req, Response: rec},
	}
	require.NoError(t, HandleListMobileDevices()(event))
	require.Equal(t, http.StatusOK, rec.Code)

	var response ListMobileDevicesPublicResponseSchema
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Devices, 1)
	item := response.Devices[0]
	require.Equal(t, "usera-s-organization/device-host/pixel-8", item.Path)
	require.Equal(t, "usera-s-organization/device-host", item.RunnerID)
	require.Equal(t, "device-host", item.RunnerName)
	require.Equal(t, "android_phone", item.Type)
	require.Equal(t, "ABC123", item.Serial)
	require.True(t, item.IsOwned)
	require.True(t, item.IsOnline)
	require.NotNil(t, item.QueueLength)
	require.Equal(t, 2, *item.QueueLength)
}

func TestListMobileDevicesMarksDeviceOfflineWhenHeartbeatIsStale(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	ownerID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, ownerID, "offline-host", "https://offline.example", false)
	runner, err := canonify.Resolve(app, "/usera-s-organization/offline-host")
	require.NoError(t, err)
	devices, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(devices)
	device.Set("owner", ownerID)
	device.Set("runner", runner.Id)
	device.Set("name", "pixel")
	device.Set("type", "android_phone")
	device.Set("online", true)
	require.NoError(t, app.Save(device))
	markRunnerHeartbeat(
		t,
		app,
		runner,
		time.Now().Add(-2*mobilerunner.DefaultSelectorHeartbeatTTL),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/mobile-devices", nil)
	rec := httptest.NewRecorder()
	event := &core.RequestEvent{
		App:   app,
		Auth:  user,
		Event: router.Event{Request: req, Response: rec},
	}
	require.NoError(t, HandleListMobileDevices()(event))

	var response ListMobileDevicesPublicResponseSchema
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Devices, 1)
	require.False(t, response.Devices[0].IsOnline)
	require.Nil(t, response.Devices[0].QueueLength)
}

func TestListMobileDevicesSkipsDisabledRunnersWithoutProbing(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	ownerID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, ownerID, "disabled-host", "https://disabled.example", false)
	runner, err := canonify.Resolve(app, "/usera-s-organization/disabled-host")
	require.NoError(t, err)
	runner.Set("disabled", true)
	require.NoError(t, app.Save(runner))

	devices, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(devices)
	device.Set("owner", ownerID)
	device.Set("runner", runner.Id)
	device.Set("name", "pixel")
	device.Set("type", "android_phone")
	device.Set("online", true)
	require.NoError(t, app.Save(device))

	originalHealth := checkMobileRunnerHealth
	checkMobileRunnerHealth = func(_ context.Context, runner *core.Record) (bool, []MobileRunnerHealthDevice, error) {
		t.Fatalf("unexpected health probe for disabled runner %q", mobilerunner.RunnerURL(runner))

		return false, nil, nil
	}
	t.Cleanup(func() { checkMobileRunnerHealth = originalHealth })

	req := httptest.NewRequest(http.MethodGet, "/api/mobile-devices", nil)
	rec := httptest.NewRecorder()
	event := &core.RequestEvent{
		App:   app,
		Auth:  user,
		Event: router.Event{Request: req, Response: rec},
	}
	require.NoError(t, HandleListMobileDevices()(event))
	require.Equal(t, http.StatusOK, rec.Code)

	var response ListMobileDevicesPublicResponseSchema
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Empty(t, response.Devices)
}

func TestProbeMobileRunnerHealthsRunsConcurrentlyWithShortDeadline(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	const runnerCount = 4
	records := make([]*core.Record, 0, runnerCount)
	for i := range runnerCount {
		record := core.NewRecord(collection)
		record.Set("owner", orgID)
		record.Set("name", fmt.Sprintf("slow-runner-%d", i))
		record.Set("type", "android_emulator")
		record.Set("ip", fmt.Sprintf("https://slow-%d.example", i))
		records = append(records, record)
	}

	started := make(chan struct{}, runnerCount)
	release := make(chan struct{})
	deadlines := make(chan time.Duration, runnerCount)

	originalHealth := checkMobileRunnerHealth
	checkMobileRunnerHealth = func(ctx context.Context, _ *core.Record) (bool, []MobileRunnerHealthDevice, error) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok, "list probes must carry a deadline")
		deadlines <- time.Until(deadline)
		started <- struct{}{}
		<-release

		return true, nil, nil
	}
	t.Cleanup(func() { checkMobileRunnerHealth = originalHealth })

	done := make(chan []mobileRunnerHealth, 1)
	go func() { done <- probeMobileRunnerHealths(context.Background(), records) }()

	// All probes must be in flight before any of them returns: a sequential
	// implementation would block on the first one forever.
	for range runnerCount {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("probes did not run concurrently")
		}
	}
	close(release)

	healths := <-done
	require.Len(t, healths, runnerCount)
	for _, health := range healths {
		require.True(t, health.online)
		require.NoError(t, health.err)
	}
	for range runnerCount {
		remaining := <-deadlines
		require.Positive(t, remaining)
		require.LessOrEqual(t, remaining, mobileRunnerListHealthTimeout)
	}
}

func TestListMobileDevicesReportsMisconfiguredRunnerAsOffline(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	ownerID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, ownerID, "malformed-host", "192.168.1.10:8050", false)
	runner, err := canonify.Resolve(app, "/usera-s-organization/malformed-host")
	require.NoError(t, err)

	devices, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(devices)
	device.Set("owner", ownerID)
	device.Set("runner", runner.Id)
	device.Set("name", "pixel")
	device.Set("type", "android_phone")
	device.Set("online", true)
	require.NoError(t, app.Save(device))
	// A fresh heartbeat must not make an uncallable runner selectable.
	markRunnerHeartbeat(t, app, runner, time.Now())

	req := httptest.NewRequest(http.MethodGet, "/api/mobile-devices", nil)
	rec := httptest.NewRecorder()
	event := &core.RequestEvent{
		App:   app,
		Auth:  user,
		Event: router.Event{Request: req, Response: rec},
	}
	require.NoError(t, HandleListMobileDevices()(event))
	require.Equal(t, http.StatusOK, rec.Code)

	var response ListMobileDevicesPublicResponseSchema
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Devices, 1)
	require.False(t, response.Devices[0].IsOnline)
	require.Nil(t, response.Devices[0].QueueLength)
}

func markRunnerHeartbeat(
	t testing.TB,
	app *tests.TestApp,
	runner *core.Record,
	at time.Time,
) {
	t.Helper()

	record, err := app.FindRecordById("mobile_runners", runner.Id)
	require.NoError(t, err)
	record.Set("online", true)
	record.Set("last_heartbeat_at", at.UTC().Format("2006-01-02 15:04:05.000Z"))
	require.NoError(t, app.Save(record))
}

// The catalog must not be able to offer a device whose runner stopped
// reporting, and must not pay a probe timeout per runner to say so.
func TestRecentlyAliveTracksHeartbeatFreshness(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	ownerID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, ownerID, "beating-host", "https://runner.example", false)
	runner, err := canonify.Resolve(app, "/usera-s-organization/beating-host")
	require.NoError(t, err)

	now := time.Now()
	markRunnerHeartbeat(t, app, runner, now.Add(-30*time.Second))
	fresh, err := app.FindRecordById("mobile_runners", runner.Id)
	require.NoError(t, err)
	require.True(t, mobilerunner.RecentlyAlive(fresh, now))

	markRunnerHeartbeat(t, app, runner, now.Add(-90*time.Second))
	stale, err := app.FindRecordById("mobile_runners", runner.Id)
	require.NoError(t, err)
	require.False(t, mobilerunner.RecentlyAlive(stale, now))
}

func TestListMobileRunnersWithMalformedURL(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, orgID, "malformed-runner", "192.168.1.10:8050", false)

	req := httptest.NewRequest(http.MethodGet, "/api/mobile-runners?view=selector", nil)
	rec := httptest.NewRecorder()
	event := &core.RequestEvent{
		App:   app,
		Auth:  user,
		Event: router.Event{Request: req, Response: rec},
	}

	require.NoError(t, HandleListMobileRunners()(event))
	require.Equal(t, http.StatusOK, rec.Code)

	var response ListMobileRunnersPublicResponseSchema
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Len(t, response.Runners, 1)
	require.Equal(t, "misconfigured", response.Runners[0].HealthStatus)
}

func createMobileRunnerRecord(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	name string,
	runnerURL string,
	published bool,
) {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("name", name)
	record.Set("ip", runnerURL)
	record.Set("type", "android_emulator")
	record.Set("published", published)
	require.NoError(t, app.Save(record))
}

func setOrganizationPublished(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	published bool,
) {
	t.Helper()

	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	org.Set("published", published)
	require.NoError(t, app.Save(org))
}

func setMobileRunnerAdminManaged(
	t testing.TB,
	app *tests.TestApp,
	runnerPath string,
	adminManaged bool,
) {
	t.Helper()

	record, err := canonify.Resolve(app, "/"+runnerPath)
	require.NoError(t, err)
	record.Set("admin_managed", adminManaged)
	require.NoError(t, app.Save(record))
}

func TestPreviewMobileDeviceID(t *testing.T) {
	t.Run(
		"user preview derives a first-run child ID before the runner is saved",
		func(t *testing.T) {
			app := setupMobileRunnerApp(t)
			defer app.Cleanup()

			user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)

			event := performMobileRunnerRequest(
				t,
				app,
				user,
				"/api/mobile-device/preview-id",
				PreviewMobileDeviceIDRequest{
					RunnerID: "usera-s-organization/new-runner",
					Name:     " Pixel-7a ",
				},
			)

			require.NoError(t, HandlePreviewMobileDeviceID()(event))
			recorder := responseRecorder(t, event)
			require.Equal(t, http.StatusOK, recorder.Code)

			body := decodeJSONBody(t, recorder)
			require.Equal(t, "usera-s-organization/new-runner", body["runner_id"])
			require.Equal(t, "pixel-7a", body["canonified_name"])
			require.Equal(t, "usera-s-organization/new-runner/pixel-7a", body["device_id"])
		},
	)

	t.Run(
		"user preview rejects a pending runner outside the user's organization",
		func(t *testing.T) {
			app := setupMobileRunnerApp(t)
			defer app.Cleanup()

			user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)

			event := performMobileRunnerRequest(
				t,
				app,
				user,
				"/api/mobile-device/preview-id",
				PreviewMobileDeviceIDRequest{
					RunnerID: "userb-s-organization/new-runner",
					Name:     "Device",
				},
			)

			err = HandlePreviewMobileDeviceID()(event)
			recorder := responseRecorder(t, event)
			requireHandlerErrorHandled(t, recorder, err)
			require.Equal(t, http.StatusForbidden, recorder.Code)
		},
	)

	t.Run("user preview uses user organization and increments canonified name", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)
		orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
		require.NoError(t, err)

		coll, err := app.FindCollectionByNameOrId("mobile_runners")
		require.NoError(t, err)

		record := core.NewRecord(coll)
		record.Set("owner", orgID)
		record.Set("name", "Test Runner")
		record.Set("ip", "https://existing.example")
		record.Set("type", "android_emulator")
		require.NoError(t, app.Save(record))

		event := performMobileRunnerRequest(
			t,
			app,
			user,
			"/api/mobile-runner/preview-id",
			PreviewMobileDeviceIDRequest{
				RunnerID: "usera-s-organization/test-runner",
				Name:     "Test Device",
			},
		)

		err = HandlePreviewMobileDeviceID()(event)
		require.NoError(t, err)

		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusOK, recorder.Code)

		body := decodeJSONBody(t, recorder)
		require.Equal(t, "usera-s-organization/test-runner", body["runner_id"])
		require.Equal(t, "test-device", body["canonified_name"])
		require.Equal(t, "usera-s-organization/test-runner/test-device", body["device_id"])
	})

	t.Run("admin preview rejects an unknown runner without an organization", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
		require.NoError(t, err)

		event := performMobileRunnerRequest(
			t,
			app,
			superuser,
			"/api/mobile-runner/preview-id",
			PreviewMobileDeviceIDRequest{RunnerID: "runner-one", Name: "Runner One"},
		)

		err = HandlePreviewMobileDeviceID()(event)

		recorder := responseRecorder(t, event)
		requireHandlerErrorHandled(t, recorder, err)
		require.Equal(t, http.StatusNotFound, recorder.Code)
		require.Contains(t, recorder.Body.String(), "runner_id does not reference a mobile runner")
	})

	t.Run("admin preview can target another organization", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
		require.NoError(t, err)
		createRunnerEvent := performMobileRunnerRequest(
			t,
			app,
			superuser,
			"/api/mobile-runner",
			UpsertMobileRunnerRequest{
				Organization: "userb-s-organization",
				Name:         "Runner B",
				IP:           "https://runner-b.example",
				Type:         "android_emulator",
			},
		)
		require.NoError(t, HandleUpsertMobileRunner()(createRunnerEvent))

		event := performMobileRunnerRequest(
			t,
			app,
			superuser,
			"/api/mobile-runner/preview-id",
			PreviewMobileDeviceIDRequest{
				Organization: "userb-s-organization",
				RunnerID:     "userb-s-organization/runner-b",
				Name:         "Runner B",
			},
		)

		err = HandlePreviewMobileDeviceID()(event)
		require.NoError(t, err)

		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusOK, recorder.Code)

		body := decodeJSONBody(t, recorder)
		require.Equal(t, "userb-s-organization/runner-b", body["runner_id"])
		require.Equal(t, "userb-s-organization/runner-b/runner-b", body["device_id"])

		derivedOwnerEvent := performMobileRunnerRequest(
			t,
			app,
			superuser,
			"/api/mobile-device/preview-id",
			PreviewMobileDeviceIDRequest{
				RunnerID: "userb-s-organization/runner-b",
				Name:     "Derived Owner Device",
			},
		)
		require.NoError(t, HandlePreviewMobileDeviceID()(derivedOwnerEvent))
		require.Equal(t, http.StatusOK, responseRecorder(t, derivedOwnerEvent).Code)
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
				Type:        "android_emulator",
				Published:   &published,
			},
		)

		err = HandleUpsertMobileRunner()(event)
		require.NoError(t, err)

		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusOK, recorder.Code)

		body := decodeJSONBody(t, recorder)
		require.Equal(t, "usera-s-organization/my-phone", body["runner_id"])
		require.Equal(t, false, body["admin_managed"])

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
		orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
		require.NoError(t, err)

		coll, err := app.FindCollectionByNameOrId("mobile_runners")
		require.NoError(t, err)

		record := core.NewRecord(coll)
		record.Set("owner", orgID)
		record.Set("name", "My Phone")
		record.Set("ip", "https://old.example")
		record.Set("type", "android_emulator")
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
				Type:         "android_emulator",
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
				Type:         "android_emulator",
			},
		)

		err = HandleUpsertMobileRunner()(event)

		recorder := responseRecorder(t, event)
		requireHandlerErrorHandled(t, recorder, err)
		require.Equal(t, http.StatusConflict, recorder.Code)
		require.Contains(t, recorder.Body.String(), "does not match the next available id")
	})

	t.Run("admin create sets admin_managed", func(t *testing.T) {
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
				Organization: "userb-s-organization",
				Name:         "Runner B",
				IP:           "https://runner-b.example",
				Type:         "android_emulator",
			},
		)

		err = HandleUpsertMobileRunner()(event)
		require.NoError(t, err)

		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusOK, recorder.Code)
		body := decodeJSONBody(t, recorder)
		require.Equal(t, true, body["admin_managed"])

		runner, err := canonify.Resolve(app, "/userb-s-organization/runner-b")
		require.NoError(t, err)
		require.True(t, runner.GetBool("admin_managed"))
	})

	t.Run("admin update existing runner does not flip admin_managed", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)
		superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
		require.NoError(t, err)

		createEvent := performMobileRunnerRequest(
			t,
			app,
			user,
			"/api/mobile-runner",
			UpsertMobileRunnerRequest{
				Name: "Mutable Phone",
				IP:   "https://runner-mutable.example",
				Type: "android_emulator",
			},
		)
		require.NoError(t, HandleUpsertMobileRunner()(createEvent))

		updateEvent := performMobileRunnerRequest(
			t,
			app,
			superuser,
			"/api/mobile-runner",
			UpsertMobileRunnerRequest{
				Organization: "usera-s-organization",
				RunnerID:     "/usera-s-organization/mutable-phone",
				Name:         "Mutable Phone",
				IP:           "https://runner-mutable-updated.example",
				Type:         "android_emulator",
			},
		)
		require.NoError(t, HandleUpsertMobileRunner()(updateEvent))

		runner, err := canonify.Resolve(app, "/usera-s-organization/mutable-phone")
		require.NoError(t, err)
		require.False(t, runner.GetBool("admin_managed"))
		require.Equal(t, "https://runner-mutable-updated.example", runner.GetString("ip"))
	})

	t.Run("runner key switch sets admin_managed from the key", func(t *testing.T) {
		cases := []struct {
			name      string
			firstKey  string
			secondKey string
			wantAdmin bool
		}{
			{
				name:      "user key to admin key",
				firstKey:  "users",
				secondKey: "_superusers",
				wantAdmin: true,
			},
			{
				name:      "admin key to user key",
				firstKey:  "_superusers",
				secondKey: "users",
				wantAdmin: false,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				app := setupMobileRunnerApp(t)
				defer app.Cleanup()

				principals := map[string]*core.Record{}
				var err error
				principals["users"], err = app.FindAuthRecordByEmail("users", "userA@example.org")
				require.NoError(t, err)
				principals["_superusers"], err = app.FindAuthRecordByEmail(
					"_superusers",
					"admin@example.org",
				)
				require.NoError(t, err)

				register := func(principal, runnerID string) {
					event := performMobileRunnerRequest(
						t,
						app,
						principals[principal],
						"/api/mobile-runner",
						UpsertMobileRunnerRequest{
							Organization: "usera-s-organization",
							RunnerID:     runnerID,
							Name:         "Switching Phone",
							IP:           "https://runner-switching.example",
							Type:         "android_emulator",
						},
					)
					// runners authenticate with Credimi-Api-Key alone
					event.Request.Header.Set(APIKeyHeaderName, "runner-key")
					require.NoError(t, HandleUpsertMobileRunner()(event))
				}

				register(tc.firstKey, "")
				runner, err := canonify.Resolve(app, "/usera-s-organization/switching-phone")
				require.NoError(t, err)
				require.Equal(t, !tc.wantAdmin, runner.GetBool("admin_managed"))

				register(tc.secondKey, "/usera-s-organization/switching-phone")
				switched, err := canonify.Resolve(app, "/usera-s-organization/switching-phone")
				require.NoError(t, err)
				require.Equal(t, runner.Id, switched.Id)
				require.Equal(t, tc.wantAdmin, switched.GetBool("admin_managed"))
			})
		}
	})
}

func upsertRotatingPhone(
	t *testing.T,
	app *tests.TestApp,
	user *core.Record,
	runnerID string,
) (*core.RequestEvent, error) {
	t.Helper()

	published := true
	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-runner",
		UpsertMobileRunnerRequest{
			RunnerID:  runnerID,
			Name:      "Rotating Phone",
			IP:        "https://runner-rotating.example",
			Type:      "android_emulator",
			Published: &published,
		},
	)
	return event, HandleUpsertMobileRunner()(event)
}

func TestUpsertMobileRunnerIssuesRunnerCredential(t *testing.T) {
	t.Run("every upsert rotates the credential the lifecycle then returns", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)

		credentials := make([]any, 0, 2)
		for generation, runnerID := range []string{"", "/usera-s-organization/rotating-phone"} {
			event, err := upsertRotatingPhone(t, app, user, runnerID)
			require.NoError(t, err)
			body := decodeJSONBody(t, responseRecorder(t, event))

			record, err := canonify.Resolve(app, "/usera-s-organization/rotating-phone")
			require.NoError(t, err)
			require.Equal(t, generation+1, record.GetInt("credential_generation"))
			credential, err := mobilerunner.Credential(record)
			require.NoError(t, err)
			require.Equal(t, credential, body["runner_credential"])
			credentials = append(credentials, body["runner_credential"])
		}
		require.NotEqual(t, credentials[0], credentials[1])

		heartbeat := performMobileRunnerRequest(
			t,
			app,
			user,
			"/api/mobile-runner/lifecycle/heartbeat",
			MobileRunnerLifecycleRequest{RunnerID: "/usera-s-organization/rotating-phone"},
		)
		require.NoError(t, HandleMobileRunnerLifecycleHeartbeat()(heartbeat))
		require.Equal(
			t,
			credentials[1],
			decodeJSONBody(t, responseRecorder(t, heartbeat))["runner_credential"],
		)
	})

	t.Run("an organization member publishes its runner", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()

		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)
		memberRole, err := app.FindFirstRecordByData("orgRoles", "name", "member")
		require.NoError(t, err)
		authorization, err := app.FindFirstRecordByData("orgAuthorizations", "user", user.Id)
		require.NoError(t, err)
		authorization.Set("role", memberRole.Id)
		require.NoError(t, app.Save(authorization))

		event, err := upsertRotatingPhone(t, app, user, "")
		require.NoError(t, err)
		require.Equal(t, true, decodeJSONBody(t, responseRecorder(t, event))["published"])

		record, err := canonify.Resolve(app, "/usera-s-organization/rotating-phone")
		require.NoError(t, err)
		require.True(t, record.GetBool("published"))
	})

	t.Run("missing secret saves nothing", func(t *testing.T) {
		app := setupMobileRunnerApp(t)
		defer app.Cleanup()
		t.Setenv(mobilerunner.CredentialSecretEnvVar, "")

		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)

		event, err := upsertRotatingPhone(t, app, user, "")
		recorder := responseRecorder(t, event)
		requireHandlerErrorHandled(t, recorder, err)
		require.Equal(t, http.StatusInternalServerError, recorder.Code)

		_, err = canonify.Resolve(app, "/usera-s-organization/rotating-phone")
		require.Error(t, err)
	})
}
