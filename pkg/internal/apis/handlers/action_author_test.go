// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

const (
	authoringAndroidVersion = "usera-s-organization/authoring-handler-wallet/android-only"
	authoringBothVersion    = "usera-s-organization/authoring-handler-wallet/both-platforms"
	authoringAdminRunner    = "usera-s-organization/admin-runner"
	authoringTenantRunner   = "usera-s-organization/tenant-runner"
	authoringForeignRunner  = "userb-s-organization/foreign-admin-runner"
	authoringDisabledRunner = "usera-s-organization/disabled-runner"
	authoringNamespace      = "usera-s-organization"
	authoringForeignNS      = "userb-s-organization"
)

// setupActionAuthorApp seeds userA's organization with an Android-only build,
// a build with both installers, an admin-managed runner with one device of
// each type, a tenant runner, a disabled admin-managed runner, and an
// admin-managed runner of userB's organization.
func setupActionAuthorApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	canonify.RegisterCanonifyHooks(app)
	ActionAuthorRoutes.Add(app)
	org, err := app.FindFirstRecordByData("organizations", "name", "userA's organization")
	require.NoError(t, err)
	foreignOrg, err := app.FindFirstRecordByData("organizations", "name", "userB's organization")
	require.NoError(t, err)

	wallets, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(wallets)
	wallet.Set("owner", org.Id)
	wallet.Set("name", "Authoring handler wallet")
	require.NoError(t, app.Save(wallet))
	versions, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)
	for tag, installers := range map[string][]string{
		"android-only":   {"android_installer"},
		"both-platforms": {"android_installer", "ios_installer"},
	} {
		version := core.NewRecord(versions)
		version.Set("owner", org.Id)
		version.Set("wallet", wallet.Id)
		version.Set("tag", tag)
		for _, field := range installers {
			file, err := filesystem.NewFileFromBytes([]byte(field), field+".zip")
			require.NoError(t, err)
			version.Set(field, []*filesystem.File{file})
		}
		require.NoError(t, app.Save(version))
	}

	adminRunner := createAuthoringRunner(t, app, org.Id, "admin-runner", true)
	for _, deviceType := range []string{
		"android_phone",
		"android_emulator",
		"redroid",
		"ios_simulator",
		"ios_phone",
	} {
		createAuthoringDevice(t, app, adminRunner, deviceType)
	}
	createAuthoringDevice(
		t,
		app,
		createAuthoringRunner(t, app, org.Id, "tenant-runner", false),
		"android_phone",
	)
	disabledRunner := createAuthoringRunner(t, app, org.Id, "disabled-runner", true)
	disabledRunner.Set("disabled", true)
	require.NoError(t, app.Save(disabledRunner))
	createAuthoringDevice(t, app, disabledRunner, "android_emulator")
	createAuthoringDevice(
		t,
		app,
		createAuthoringRunner(t, app, foreignOrg.Id, "foreign-admin-runner", true),
		"android_emulator",
	)
	return app
}

func createAuthoringRunner(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	name string,
	adminManaged bool,
) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(collection)
	runner.Set("owner", orgID)
	runner.Set("name", name)
	runner.Set("ip", "https://"+name+".example.test")
	runner.Set("admin_managed", adminManaged)
	runner.Set("online", true)
	runner.Set("last_heartbeat_at", types.NowDateTime())
	require.NoError(t, app.Save(runner))
	return runner
}

func createAuthoringDevice(
	t testing.TB,
	app *tests.TestApp,
	runner *core.Record,
	deviceType string,
) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(collection)
	device.Set("owner", runner.GetString("owner"))
	device.Set("runner", runner.Id)
	name := strings.ReplaceAll(deviceType, "_", "-")
	device.Set("name", name)
	device.Set("canonified_name", name)
	device.Set("type", deviceType)
	// Only the emulator reports itself online.
	device.Set("online", deviceType == "android_emulator")
	if deviceType == "ios_phone" {
		// Runners may report it, but the collection does not list it yet.
		require.NoError(t, app.SaveNoValidate(device))
		return
	}
	require.NoError(t, app.Save(device))
}

func authoringToken(t *testing.T, user string) string {
	t.Helper()
	record, err := getUserRecordFromName(user)
	require.NoError(t, err)
	token, err := record.NewAuthToken()
	require.NoError(t, err)
	return token
}

// stubActionAuthorQueue makes runners reachable as given and records the
// enqueued ticket.
func stubActionAuthorQueue(
	t *testing.T,
	reachable bool,
) *mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest {
	t.Helper()
	origReachable := checkRunnerReachable
	origClient := queueTemporalClient
	origEnqueue := enqueueActionAuthorTicket
	t.Cleanup(func() {
		checkRunnerReachable = origReachable
		queueTemporalClient = origClient
		enqueueActionAuthorTicket = origEnqueue
	})
	checkRunnerReachable = func(context.Context, *core.Record) (bool, error) { return reachable, nil }
	queueTemporalClient = func(string) (client.Client, error) { return nil, nil }
	enqueued := &mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest{}
	enqueueActionAuthorTicket = func(
		_ context.Context,
		_ client.Client,
		_ string,
		req mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest,
	) (mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse, error) {
		*enqueued = req
		return mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse{
			TicketID: req.TicketID,
			Status:   mobiledevicesemaphore.MobileDeviceSemaphoreRunQueued,
			Position: 1,
		}, nil
	}
	return enqueued
}

func startSessionBody(deviceID, version string) string {
	return `{"device_id":"` + deviceID + `","wallet_version_identifier":"` + version + `","slot":"onboarding"}`
}

func TestActionAuthorStartRejectsInvalidDevicesAndRunners(t *testing.T) {
	token := authoringToken(t, "userA")
	scenarios := []struct {
		name       string
		reachable  bool
		body       string
		status     int
		expected   []string
		unexpected []string
	}{
		{
			name:     "missing device",
			body:     `{"wallet_version_identifier":"` + authoringAndroidVersion + `","slot":"onboarding"}`,
			status:   http.StatusBadRequest,
			expected: []string{"ActionAuthorStartRequest.DeviceID", "'required' tag"},
		},
		{
			name:     "unknown device",
			body:     startSessionBody(authoringAdminRunner+"/missing", authoringAndroidVersion),
			status:   http.StatusNotFound,
			expected: []string{`"device_not_found"`},
		},
		{
			name:   "iOS phone",
			body:   startSessionBody(authoringAdminRunner+"/ios-phone", authoringBothVersion),
			status: http.StatusUnprocessableEntity,
			expected: []string{
				`"device_type_not_supported"`,
				"the action author supports Android devices and iOS simulators",
			},
		},
		{
			name: "iOS simulator with an Android-only version",
			body: startSessionBody(
				authoringAdminRunner+"/ios-simulator",
				authoringAndroidVersion,
			),
			status:   http.StatusBadRequest,
			expected: []string{"selected version has no iOS app"},
		},
		{
			name: "tenant runner",
			body: startSessionBody(
				authoringTenantRunner+"/android-phone",
				authoringAndroidVersion,
			),
			status: http.StatusForbidden,
			expected: []string{
				`"runner_not_supported"`,
				"the action author runs only on admin-managed runners",
			},
		},
		{
			name: "device of another organization",
			body: startSessionBody(
				authoringForeignRunner+"/android-emulator",
				authoringAndroidVersion,
			),
			status:     http.StatusNotFound,
			expected:   []string{`"device_not_found"`},
			unexpected: []string{"runner_not_owned", "android_emulator"},
		},
		{
			name:      "disabled runner",
			reachable: true,
			body: startSessionBody(
				authoringDisabledRunner+"/android-emulator",
				authoringAndroidVersion,
			),
			status:   http.StatusServiceUnavailable,
			expected: []string{"device runner is offline"},
		},
		{
			name:      "offline runner",
			reachable: false,
			body: startSessionBody(
				authoringAdminRunner+"/android-phone",
				authoringAndroidVersion,
			),
			status:   http.StatusServiceUnavailable,
			expected: []string{"device runner is offline"},
		},
	}
	for _, scenario := range scenarios {
		enqueued := stubActionAuthorQueue(t, scenario.reachable)
		(&tests.ApiScenario{
			Name:               scenario.name,
			Method:             http.MethodPost,
			URL:                "/api/action-author/sessions",
			Body:               strings.NewReader(scenario.body),
			Headers:            map[string]string{"Authorization": "Bearer " + token},
			ExpectedStatus:     scenario.status,
			ExpectedContent:    scenario.expected,
			NotExpectedContent: scenario.unexpected,
			TestAppFactory:     setupActionAuthorApp,
		}).Test(t)
		require.Empty(t, enqueued.TicketID, scenario.name)
	}
}

func TestActionAuthorStartRejectsSuperuserOnRunnerOfAnotherOrganization(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	superuser, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, "admin@example.org")
	require.NoError(t, err)
	token, err := superuser.NewAuthToken()
	require.NoError(t, err)
	app.Cleanup()
	enqueued := stubActionAuthorQueue(t, true)

	(&tests.ApiScenario{
		Name:   "superuser on a runner of another organization",
		Method: http.MethodPost,
		URL:    "/api/action-author/sessions",
		Body: strings.NewReader(startSessionBody(
			authoringForeignRunner+"/android-emulator",
			authoringAndroidVersion,
		)),
		Headers:        map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus: http.StatusForbidden,
		ExpectedContent: []string{
			`"runner_not_owned"`,
			"the device's runner belongs to another organization",
		},
		TestAppFactory: setupActionAuthorApp,
	}).Test(t)
	require.Empty(t, enqueued.TicketID)
}

func TestActionAuthorStartQueuesSessionWithDevicePlatform(t *testing.T) {
	token := authoringToken(t, "userA")
	scenarios := []struct {
		device   string
		version  string
		platform string
	}{
		{device: "android-phone", version: authoringAndroidVersion, platform: "android"},
		{device: "android-emulator", version: authoringAndroidVersion, platform: "android"},
		{device: "redroid", version: authoringAndroidVersion, platform: "android"},
		{device: "ios-simulator", version: authoringBothVersion, platform: "ios"},
	}
	for _, scenario := range scenarios {
		enqueued := stubActionAuthorQueue(t, true)
		deviceID := authoringAdminRunner + "/" + scenario.device
		var response ActionAuthorStartResponse
		(&tests.ApiScenario{
			Name:   scenario.device,
			Method: http.MethodPost,
			URL:    "/api/action-author/sessions",
			Body: strings.NewReader(
				`{"device_id":"` + deviceID + `","wallet_version_identifier":"` + scenario.version +
					`","slot":"onboarding","app_id":"eu.wallet"}`,
			),
			Headers:         map[string]string{"Authorization": "Bearer " + token},
			ExpectedStatus:  http.StatusAccepted,
			ExpectedContent: []string{`"ticket_id"`},
			TestAppFactory:  setupActionAuthorApp,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(body, &response))
			},
		}).Test(t)

		require.NotEmpty(t, response.TicketID)
		require.Equal(t, ActionAuthorStartResponse{
			TicketID:                response.TicketID,
			DeviceID:                deviceID,
			Status:                  "queued",
			Position:                1,
			WorkflowNamespace:       "usera-s-organization",
			WalletVersionIdentifier: scenario.version,
		}, response)
		require.Equal(t, response.TicketID, enqueued.TicketID)
		require.Equal(t, "usera-s-organization", enqueued.OwnerNamespace)
		require.Equal(t, deviceID, enqueued.DeviceID)
		require.Equal(t, []string{deviceID}, enqueued.RequiredDeviceIDs)
		require.Equal(t, deviceID, enqueued.LeaderDeviceID)
		require.False(t, enqueued.EnqueuedAt.IsZero())
		require.Equal(t, mobiledevicesemaphore.RunKindActionAuthor, enqueued.RunKind)
		require.Empty(t, enqueued.YAML)
		user, err := getUserRecordFromName("userA")
		require.NoError(t, err)
		require.Equal(t, map[string]any{
			"device_id":                 deviceID,
			"platform":                  scenario.platform,
			"wallet_version_identifier": scenario.version,
			"actor_id":                  user.Id,
			"actor_collection":          "users",
			"slot":                      "onboarding",
			"app_id":                    "eu.wallet",
		}, enqueued.ActionAuthor)
	}
}

func TestActionAuthorStartMapsQueueFailures(t *testing.T) {
	token := authoringToken(t, "userA")
	for name, tc := range map[string]struct {
		clientErr  error
		enqueueErr error
		status     int
		expected   string
	}{
		"temporal unavailable": {
			clientErr: errors.New("dial failed"),
			status:    http.StatusServiceUnavailable,
			expected:  `"temporal_unavailable"`,
		},
		"queue limit": {
			enqueueErr: temporal.NewApplicationError(
				"queue limit exceeded for runner admin-runner: 1 of 1",
				mobiledevicesemaphore.ErrQueueLimitExceeded,
			),
			status:   http.StatusConflict,
			expected: "queue limit exceeded",
		},
		"enqueue failure": {
			enqueueErr: errors.New("update failed"),
			status:     http.StatusServiceUnavailable,
			expected:   `"start_failed"`,
		},
	} {
		stubActionAuthorQueue(t, true)
		origCancel := cancelRunTicket
		var canceled []string
		cancelRunTicket = func(
			_ context.Context,
			deviceID string,
			req mobiledevicesemaphore.MobileDeviceSemaphoreRunCancelRequest,
		) (mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView, error) {
			canceled = append(canceled, deviceID+"/"+req.TicketID)
			return mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView{}, nil
		}
		t.Cleanup(func() { cancelRunTicket = origCancel })
		if tc.clientErr != nil {
			queueTemporalClient = func(string) (client.Client, error) { return nil, tc.clientErr }
		}
		if tc.enqueueErr != nil {
			enqueueActionAuthorTicket = func(
				context.Context,
				client.Client,
				string,
				mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest,
			) (mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse, error) {
				return mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse{}, tc.enqueueErr
			}
		}
		(&tests.ApiScenario{
			Name:            name,
			Method:          http.MethodPost,
			URL:             "/api/action-author/sessions",
			Body:            strings.NewReader(startSessionBody(authoringAdminRunner+"/android-phone", authoringAndroidVersion)),
			Headers:         map[string]string{"Authorization": "Bearer " + token},
			ExpectedStatus:  tc.status,
			ExpectedContent: []string{tc.expected},
			TestAppFactory:  setupActionAuthorApp,
		}).Test(t)
		if name == "enqueue failure" {
			require.Len(t, canceled, 1, name)
		} else {
			require.Empty(t, canceled, name)
		}
	}
}

func TestActionAuthorDevicesListsOwnedAdminRunnerDevices(t *testing.T) {
	for _, tc := range []struct {
		user     string
		expected []ActionAuthorDevice
	}{
		{
			user: "userA",
			expected: []ActionAuthorDevice{
				authoringDevice("android-emulator", "android_emulator", "android", true),
				authoringDevice("android-phone", "android_phone", "android", false),
				authoringDevice("ios-simulator", "ios_simulator", "ios", false),
				authoringDevice("redroid", "redroid", "android", false),
			},
		},
		{user: "userC", expected: []ActionAuthorDevice{}},
	} {
		var response ActionAuthorDevicesResponse
		(&tests.ApiScenario{
			Name:           tc.user,
			Method:         http.MethodGet,
			URL:            "/api/action-author/devices",
			Headers:        map[string]string{"Authorization": "Bearer " + authoringToken(t, tc.user)},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"devices"`,
			},
			TestAppFactory: setupActionAuthorApp,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				body, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(body, &response))
			},
		}).Test(t)
		require.Equal(t, tc.expected, response.Devices, tc.user)
	}
}

func TestActionAuthorDevicesListsEveryAdminRunnerForSuperusers(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	superuser, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, "admin@example.org")
	require.NoError(t, err)
	token, err := superuser.NewAuthToken()
	require.NoError(t, err)
	app.Cleanup()

	var response ActionAuthorDevicesResponse
	(&tests.ApiScenario{
		Name:            "superuser",
		Method:          http.MethodGet,
		URL:             "/api/action-author/devices",
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"devices"`},
		TestAppFactory:  setupActionAuthorApp,
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(body, &response))
		},
	}).Test(t)

	runners := map[string]int{}
	for _, device := range response.Devices {
		runners[device.RunnerName]++
	}
	require.Equal(t, map[string]int{"admin-runner": 4, "foreign-admin-runner": 1}, runners)
	require.Equal(t, "admin-runner", response.Devices[0].RunnerName)
	for _, device := range response.Devices {
		if device.RunnerName == "foreign-admin-runner" {
			require.Equal(t, authoringForeignNS, device.Namespace)
			require.NotEqual(t, response.Devices[0].Namespace, device.Namespace)
		} else {
			require.Equal(t, authoringNamespace, device.Namespace)
		}
	}
}

// authoringDevice is a device of userA's admin-managed runner.
func authoringDevice(name, deviceType, platform string, online bool) ActionAuthorDevice {
	return ActionAuthorDevice{
		Path:       authoringAdminRunner + "/" + name,
		Namespace:  authoringNamespace,
		Name:       name,
		RunnerName: "admin-runner",
		Type:       deviceType,
		Platform:   platform,
		IsOnline:   online,
	}
}
