// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

// covRunnerWrongInput simulates a validated input of the wrong type.
type covRunnerWrongInput struct{}

type covRunnerFixture struct {
	app       *tests.TestApp
	user      *core.Record
	superuser *core.Record
	orgID     string
	runner    *core.Record
	otherOrg  string
}

// covRunnerSetup builds an app with a saved runner "lab-runner" owned by
// userA's organization.
func covRunnerSetup(t *testing.T) covRunnerFixture {
	t.Helper()
	app := setupMobileRunnerApp(t)
	t.Cleanup(app.Cleanup)

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	userB, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	otherOrg, err := pbutils.GetUserOrganizationID(app, userB.Id)
	require.NoError(t, err)

	createMobileRunnerRecord(t, app, orgID, "lab-runner", "https://runner.example", false)
	runner, err := canonify.Resolve(app, "/usera-s-organization/lab-runner")
	require.NoError(t, err)

	return covRunnerFixture{
		app:       app,
		user:      user,
		superuser: superuser,
		orgID:     orgID,
		runner:    runner,
		otherOrg:  otherOrg,
	}
}

func covRunnerCreateDevice(
	t *testing.T,
	app core.App,
	runner *core.Record,
	name, deviceType, serial string,
) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(collection)
	device.Set("owner", runner.GetString("owner"))
	device.Set("runner", runner.Id)
	device.Set("name", name)
	device.Set("type", deviceType)
	device.Set("serial", serial)
	require.NoError(t, app.Save(device))
	return device
}

func covRunnerCall(
	t *testing.T,
	handler func(*core.RequestEvent) error,
	event *core.RequestEvent,
) (int, map[string]any, string) {
	t.Helper()
	err := handler(event)
	recorder := responseRecorder(t, event)
	if err != nil {
		requireHandlerErrorHandled(t, recorder, err)
	}
	body := recorder.Body.String()
	if recorder.Code == http.StatusOK {
		return recorder.Code, decodeJSONBody(t, recorder), body
	}
	return recorder.Code, nil, body
}

func TestUpsertMobileDeviceCreatesAndUpdates(t *testing.T) {
	fx := covRunnerSetup(t)
	liveStream := true

	status, body, raw := covRunnerCall(t, HandleUpsertMobileDevice(), performMobileRunnerRequest(
		t, fx.app, fx.user, "/api/mobile-device",
		UpsertMobileDeviceRequest{
			RunnerID:    "usera-s-organization/lab-runner",
			Name:        " Pixel 7 ",
			Description: " bench ",
			Type:        "android_phone",
			Serial:      " SER1 ",
			LiveStream:  &liveStream,
		},
	))
	require.Equal(t, http.StatusOK, status, raw)
	assert.Equal(t, "usera-s-organization/lab-runner/pixel-7", body["device_id"])
	assert.Equal(t, "Pixel 7", body["name"])
	assert.Equal(t, "bench", body["description"])
	assert.Equal(t, "SER1", body["serial"])
	assert.Equal(t, true, body["live_stream"])

	device, err := canonify.Resolve(fx.app, "/usera-s-organization/lab-runner/pixel-7")
	require.NoError(t, err)
	assert.Equal(t, fx.orgID, device.GetString("owner"))
	assert.Equal(t, fx.runner.Id, device.GetString("runner"))

	// Update the same device without live_stream: it keeps the stored value
	// and changing its own serial does not conflict with itself.
	status, body, raw = covRunnerCall(t, HandleUpsertMobileDevice(), performMobileRunnerRequest(
		t, fx.app, fx.user, "/api/mobile-device",
		UpsertMobileDeviceRequest{
			DeviceID: "usera-s-organization/lab-runner/pixel-7",
			RunnerID: "usera-s-organization/lab-runner",
			Name:     "Pixel 7",
			Type:     "android_phone",
			Serial:   "SER1",
		},
	))
	require.Equal(t, http.StatusOK, status, raw)
	assert.Equal(t, device.Id, body["id"])
	assert.Equal(t, true, body["live_stream"])
	assert.Empty(t, body["description"])

	records, err := fx.app.FindRecordsByFilter(
		"mobile_devices", "runner = {:r}", "", 0, 0, map[string]any{"r": fx.runner.Id},
	)
	require.NoError(t, err)
	assert.Len(t, records, 1)
}

func TestUpsertMobileDeviceRejections(t *testing.T) {
	testCases := []struct {
		name       string
		seed       func(t *testing.T, fx covRunnerFixture)
		auth       func(fx covRunnerFixture) *core.Record
		request    UpsertMobileDeviceRequest
		noInput    bool
		wantStatus int
		wantReason string
	}{
		{
			name:       "missing validated input",
			noInput:    true,
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_request",
		},
		{
			name: "unauthenticated caller",
			auth: func(covRunnerFixture) *core.Record { return nil },
			request: UpsertMobileDeviceRequest{
				RunnerID: "usera-s-organization/lab-runner", Name: "D", Type: "redroid",
			},
			wantStatus: http.StatusUnauthorized,
			wantReason: "authentication_required",
		},
		{
			name: "unknown runner",
			request: UpsertMobileDeviceRequest{
				RunnerID: "usera-s-organization/ghost", Name: "D", Type: "redroid",
			},
			wantStatus: http.StatusNotFound,
			wantReason: "runner_not_found",
		},
		{
			name: "runner of another organization",
			seed: func(t *testing.T, fx covRunnerFixture) {
				createMobileRunnerRecord(
					t,
					fx.app,
					fx.otherOrg,
					"b-runner",
					"https://b.example",
					false,
				)
			},
			request: UpsertMobileDeviceRequest{
				RunnerID: "userb-s-organization/b-runner", Name: "D", Type: "redroid",
			},
			wantStatus: http.StatusForbidden,
			wantReason: "runner_id_owner_mismatch",
		},
		{
			name: "runner_id pointing at a device",
			seed: func(t *testing.T, fx covRunnerFixture) {
				covRunnerCreateDevice(t, fx.app, fx.runner, "dev", "redroid", "")
			},
			request: UpsertMobileDeviceRequest{
				RunnerID: "usera-s-organization/lab-runner/dev", Name: "D", Type: "redroid",
			},
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_runner_id",
		},
		{
			name: "device_id not matching next available id",
			request: UpsertMobileDeviceRequest{
				DeviceID: "usera-s-organization/lab-runner/other-name",
				RunnerID: "usera-s-organization/lab-runner",
				Name:     "Pixel",
				Type:     "redroid",
			},
			wantStatus: http.StatusConflict,
			wantReason: "device_id_conflict",
		},
		{
			name: "device_id pointing at a runner",
			request: UpsertMobileDeviceRequest{
				DeviceID: "usera-s-organization/lab-runner",
				RunnerID: "usera-s-organization/lab-runner",
				Name:     "Pixel",
				Type:     "redroid",
			},
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_device_id",
		},
		{
			name: "device_id owned by another organization",
			seed: func(t *testing.T, fx covRunnerFixture) {
				createMobileRunnerRecord(
					t,
					fx.app,
					fx.otherOrg,
					"b-runner",
					"https://b.example",
					false,
				)
				bRunner, err := canonify.Resolve(fx.app, "/userb-s-organization/b-runner")
				require.NoError(t, err)
				covRunnerCreateDevice(t, fx.app, bRunner, "bdev", "redroid", "")
			},
			request: UpsertMobileDeviceRequest{
				DeviceID: "userb-s-organization/b-runner/bdev",
				RunnerID: "usera-s-organization/lab-runner",
				Name:     "bdev",
				Type:     "redroid",
			},
			wantStatus: http.StatusForbidden,
			wantReason: "device_id_owner_mismatch",
		},
		{
			name: "renaming an existing device",
			seed: func(t *testing.T, fx covRunnerFixture) {
				covRunnerCreateDevice(t, fx.app, fx.runner, "dev", "redroid", "")
			},
			request: UpsertMobileDeviceRequest{
				DeviceID: "usera-s-organization/lab-runner/dev",
				RunnerID: "usera-s-organization/lab-runner",
				Name:     "renamed",
				Type:     "redroid",
			},
			wantStatus: http.StatusConflict,
			wantReason: "device_identity_immutable",
		},
		{
			name: "moving an existing device to another runner",
			seed: func(t *testing.T, fx covRunnerFixture) {
				covRunnerCreateDevice(t, fx.app, fx.runner, "dev", "redroid", "")
				createMobileRunnerRecord(
					t,
					fx.app,
					fx.orgID,
					"second",
					"https://second.example",
					false,
				)
			},
			request: UpsertMobileDeviceRequest{
				DeviceID: "usera-s-organization/lab-runner/dev",
				RunnerID: "usera-s-organization/second",
				Name:     "dev",
				Type:     "redroid",
			},
			wantStatus: http.StatusConflict,
			wantReason: "device_identity_immutable",
		},
		{
			name: "unsupported device type",
			request: UpsertMobileDeviceRequest{
				RunnerID: "usera-s-organization/lab-runner", Name: "D", Type: "ios_phone",
			},
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_device_type",
		},
		{
			name: "second emulator on the same runner",
			seed: func(t *testing.T, fx covRunnerFixture) {
				covRunnerCreateDevice(t, fx.app, fx.runner, "emu-1", "android_emulator", "")
			},
			request: UpsertMobileDeviceRequest{
				RunnerID: "usera-s-organization/lab-runner",
				Name:     "emu-2",
				Type:     "android_emulator",
			},
			wantStatus: http.StatusConflict,
			wantReason: "device_type_limit",
		},
		{
			name: "second simulator on the same runner",
			seed: func(t *testing.T, fx covRunnerFixture) {
				covRunnerCreateDevice(t, fx.app, fx.runner, "sim-1", "ios_simulator", "")
			},
			request: UpsertMobileDeviceRequest{
				RunnerID: "usera-s-organization/lab-runner", Name: "sim-2", Type: "ios_simulator",
			},
			wantStatus: http.StatusConflict,
			wantReason: "device_type_limit",
		},
		{
			name: "phone serial already used by a redroid",
			seed: func(t *testing.T, fx covRunnerFixture) {
				covRunnerCreateDevice(t, fx.app, fx.runner, "red", "redroid", "SER9")
			},
			request: UpsertMobileDeviceRequest{
				RunnerID: "usera-s-organization/lab-runner",
				Name:     "phone",
				Type:     "android_phone",
				Serial:   "SER9",
			},
			wantStatus: http.StatusConflict,
			wantReason: "device_serial_conflict",
		},
		{
			name: "save failure",
			seed: func(t *testing.T, fx covRunnerFixture) {
				fx.app.OnRecordCreate("mobile_devices").BindFunc(func(*core.RecordEvent) error {
					return errors.New("boom")
				})
			},
			request: UpsertMobileDeviceRequest{
				RunnerID: "usera-s-organization/lab-runner", Name: "D", Type: "redroid",
			},
			wantStatus: http.StatusInternalServerError,
			wantReason: "failed_to_save_mobile_device",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fx := covRunnerSetup(t)
			if tc.seed != nil {
				tc.seed(t, fx)
			}
			auth := fx.user
			if tc.auth != nil {
				auth = tc.auth(fx)
			}
			var input any = tc.request
			if tc.noInput {
				input = covRunnerWrongInput{}
			}
			status, _, raw := covRunnerCall(
				t,
				HandleUpsertMobileDevice(),
				performMobileRunnerRequest(
					t, fx.app, auth, "/api/mobile-device", input,
				),
			)
			require.Equal(t, tc.wantStatus, status, raw)
			assert.Contains(t, raw, tc.wantReason)
		})
	}
}

func TestUpsertMobileDeviceAllowsDistinctPhoneSerials(t *testing.T) {
	fx := covRunnerSetup(t)
	covRunnerCreateDevice(t, fx.app, fx.runner, "phone-a", "android_phone", "SER1")
	covRunnerCreateDevice(t, fx.app, fx.runner, "emu", "android_emulator", "SER2")

	status, body, raw := covRunnerCall(t, HandleUpsertMobileDevice(), performMobileRunnerRequest(
		t, fx.app, fx.superuser, "/api/mobile-device",
		UpsertMobileDeviceRequest{
			RunnerID: "usera-s-organization/lab-runner",
			Name:     "phone-b",
			Type:     "redroid",
			Serial:   "SER2",
		},
	))
	require.Equal(t, http.StatusOK, status, raw)
	assert.Equal(t, "usera-s-organization/lab-runner/phone-b", body["device_id"])
}

func TestDeleteMobileDevice(t *testing.T) {
	testCases := []struct {
		name       string
		seed       func(t *testing.T, fx covRunnerFixture)
		admin      bool
		request    DeleteMobileDeviceRequest
		noInput    bool
		wantStatus int
		wantReason string
		wantGone   string
	}{
		{
			name: "owner deletes its device",
			seed: func(t *testing.T, fx covRunnerFixture) {
				covRunnerCreateDevice(t, fx.app, fx.runner, "dev", "redroid", "")
			},
			request: DeleteMobileDeviceRequest{
				RunnerID: "usera-s-organization/lab-runner",
				DeviceID: "/usera-s-organization/lab-runner/dev",
			},
			wantStatus: http.StatusOK,
			wantGone:   "/usera-s-organization/lab-runner/dev",
		},
		{
			name: "superuser deletes using the runner owner",
			seed: func(t *testing.T, fx covRunnerFixture) {
				covRunnerCreateDevice(t, fx.app, fx.runner, "dev", "redroid", "")
			},
			admin: true,
			request: DeleteMobileDeviceRequest{
				RunnerID: "usera-s-organization/lab-runner",
				DeviceID: "usera-s-organization/lab-runner/dev",
			},
			wantStatus: http.StatusOK,
			wantGone:   "/usera-s-organization/lab-runner/dev",
		},
		{
			name:       "missing validated input",
			noInput:    true,
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_request",
		},
		{
			name: "unknown device",
			request: DeleteMobileDeviceRequest{
				RunnerID: "usera-s-organization/lab-runner",
				DeviceID: "usera-s-organization/lab-runner/ghost",
			},
			wantStatus: http.StatusNotFound,
			wantReason: "device_not_found",
		},
		{
			name:  "superuser with unknown runner",
			admin: true,
			request: DeleteMobileDeviceRequest{
				RunnerID: "usera-s-organization/ghost",
				DeviceID: "usera-s-organization/ghost/dev",
			},
			wantStatus: http.StatusNotFound,
			wantReason: "runner_not_found",
		},
		{
			name: "device of another organization",
			seed: func(t *testing.T, fx covRunnerFixture) {
				createMobileRunnerRecord(
					t,
					fx.app,
					fx.otherOrg,
					"b-runner",
					"https://b.example",
					false,
				)
				bRunner, err := canonify.Resolve(fx.app, "/userb-s-organization/b-runner")
				require.NoError(t, err)
				covRunnerCreateDevice(t, fx.app, bRunner, "bdev", "redroid", "")
			},
			request: DeleteMobileDeviceRequest{
				RunnerID: "userb-s-organization/b-runner",
				DeviceID: "userb-s-organization/b-runner/bdev",
			},
			wantStatus: http.StatusForbidden,
			wantReason: "device_id_owner_mismatch",
		},
		{
			name: "delete failure",
			seed: func(t *testing.T, fx covRunnerFixture) {
				covRunnerCreateDevice(t, fx.app, fx.runner, "dev", "redroid", "")
				fx.app.OnRecordDelete("mobile_devices").BindFunc(func(*core.RecordEvent) error {
					return errors.New("boom")
				})
			},
			request: DeleteMobileDeviceRequest{
				RunnerID: "usera-s-organization/lab-runner",
				DeviceID: "usera-s-organization/lab-runner/dev",
			},
			wantStatus: http.StatusInternalServerError,
			wantReason: "failed_to_delete_mobile_device",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fx := covRunnerSetup(t)
			if tc.seed != nil {
				tc.seed(t, fx)
			}
			auth := fx.user
			if tc.admin {
				auth = fx.superuser
			}
			var input any = tc.request
			if tc.noInput {
				input = covRunnerWrongInput{}
			}
			status, body, raw := covRunnerCall(
				t,
				HandleDeleteMobileDevice(),
				performMobileRunnerRequest(
					t, fx.app, auth, "/api/mobile-device", input,
				),
			)
			require.Equal(t, tc.wantStatus, status, raw)
			if tc.wantReason != "" {
				assert.Contains(t, raw, tc.wantReason)
			}
			if tc.wantGone != "" {
				assert.Equal(t, "usera-s-organization/lab-runner/dev", body["device_id"])
				_, err := canonify.Resolve(fx.app, tc.wantGone)
				assert.Error(t, err)
			}
		})
	}
}

func TestReconcileMobileDevicesRemovesOnlyUnlistedDevices(t *testing.T) {
	fx := covRunnerSetup(t)
	covRunnerCreateDevice(t, fx.app, fx.runner, "keep", "redroid", "")
	covRunnerCreateDevice(t, fx.app, fx.runner, "stale", "android_phone", "")
	createMobileRunnerRecord(t, fx.app, fx.orgID, "other-runner", "https://o.example", false)
	otherRunner, err := canonify.Resolve(fx.app, "/usera-s-organization/other-runner")
	require.NoError(t, err)
	covRunnerCreateDevice(t, fx.app, otherRunner, "untouched", "redroid", "")

	status, body, raw := covRunnerCall(
		t,
		HandleReconcileMobileDevices(),
		performMobileRunnerRequest(
			t, fx.app, fx.user, "/api/mobile-device/reconcile",
			ReconcileMobileDevicesRequest{
				RunnerID:  "usera-s-organization/lab-runner",
				DeviceIDs: []string{" /usera-s-organization/lab-runner/keep"},
			},
		),
	)
	require.Equal(t, http.StatusOK, status, raw)
	assert.Equal(t, []any{"usera-s-organization/lab-runner/stale"}, body["removed_device_ids"])

	_, err = canonify.Resolve(fx.app, "/usera-s-organization/lab-runner/keep")
	assert.NoError(t, err)
	_, err = canonify.Resolve(fx.app, "/usera-s-organization/lab-runner/stale")
	assert.Error(t, err)
	_, err = canonify.Resolve(fx.app, "/usera-s-organization/other-runner/untouched")
	assert.NoError(t, err)

	// A second reconcile with the same inventory is a no-op.
	status, body, raw = covRunnerCall(t, HandleReconcileMobileDevices(), performMobileRunnerRequest(
		t, fx.app, fx.user, "/api/mobile-device/reconcile",
		ReconcileMobileDevicesRequest{
			RunnerID:  "usera-s-organization/lab-runner",
			DeviceIDs: []string{"usera-s-organization/lab-runner/keep"},
		},
	))
	require.Equal(t, http.StatusOK, status, raw)
	assert.Equal(t, []any{}, body["removed_device_ids"])
}

func TestReconcileMobileDevicesRejections(t *testing.T) {
	testCases := []struct {
		name       string
		seed       func(t *testing.T, fx covRunnerFixture)
		request    ReconcileMobileDevicesRequest
		noInput    bool
		wantStatus int
		wantReason string
	}{
		{
			name:       "missing validated input",
			noInput:    true,
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_request",
		},
		{
			name:       "unknown runner",
			request:    ReconcileMobileDevicesRequest{RunnerID: "usera-s-organization/ghost"},
			wantStatus: http.StatusNotFound,
			wantReason: "runner_not_found",
		},
		{
			name: "runner of another organization",
			seed: func(t *testing.T, fx covRunnerFixture) {
				createMobileRunnerRecord(
					t,
					fx.app,
					fx.otherOrg,
					"b-runner",
					"https://b.example",
					false,
				)
			},
			request:    ReconcileMobileDevicesRequest{RunnerID: "userb-s-organization/b-runner"},
			wantStatus: http.StatusForbidden,
			wantReason: "runner_id_owner_mismatch",
		},
		{
			name: "delete failure",
			seed: func(t *testing.T, fx covRunnerFixture) {
				covRunnerCreateDevice(t, fx.app, fx.runner, "stale", "redroid", "")
				fx.app.OnRecordDelete("mobile_devices").BindFunc(func(*core.RecordEvent) error {
					return errors.New("boom")
				})
			},
			request:    ReconcileMobileDevicesRequest{RunnerID: "usera-s-organization/lab-runner"},
			wantStatus: http.StatusInternalServerError,
			wantReason: "failed_to_delete_mobile_device",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fx := covRunnerSetup(t)
			if tc.seed != nil {
				tc.seed(t, fx)
			}
			var input any = tc.request
			if tc.noInput {
				input = covRunnerWrongInput{}
			}
			status, _, raw := covRunnerCall(
				t,
				HandleReconcileMobileDevices(),
				performMobileRunnerRequest(
					t, fx.app, fx.user, "/api/mobile-device/reconcile", input,
				),
			)
			require.Equal(t, tc.wantStatus, status, raw)
			assert.Contains(t, raw, tc.wantReason)
		})
	}
}

func TestPreviewMobileRunnerID(t *testing.T) {
	testCases := []struct {
		name         string
		admin        bool
		anonymous    bool
		request      PreviewMobileRunnerIDRequest
		noInput      bool
		wantStatus   int
		wantReason   string
		wantRunnerID string
		wantConflict bool
		wantExisting string
	}{
		{
			name:         "fresh name for the user organization",
			request:      PreviewMobileRunnerIDRequest{Name: " New Runner "},
			wantStatus:   http.StatusOK,
			wantRunnerID: "usera-s-organization/new-runner",
		},
		{
			name:         "taken name is suffixed and reports the conflict",
			request:      PreviewMobileRunnerIDRequest{Name: "lab-runner"},
			wantStatus:   http.StatusOK,
			wantRunnerID: "usera-s-organization/lab-runner-1",
			wantConflict: true,
			wantExisting: "usera-s-organization/lab-runner",
		},
		{
			name:  "superuser targets another organization",
			admin: true,
			request: PreviewMobileRunnerIDRequest{
				Organization: "userb-s-organization",
				Name:         "lab-runner",
			},
			wantStatus:   http.StatusOK,
			wantRunnerID: "userb-s-organization/lab-runner",
		},
		{
			name:       "superuser without organization",
			admin:      true,
			request:    PreviewMobileRunnerIDRequest{Name: "x"},
			wantStatus: http.StatusBadRequest,
			wantReason: "organization_required",
		},
		{
			name:       "superuser with unknown organization",
			admin:      true,
			request:    PreviewMobileRunnerIDRequest{Organization: "nope", Name: "x"},
			wantStatus: http.StatusNotFound,
			wantReason: "organization_not_found",
		},
		{
			name:       "anonymous caller",
			anonymous:  true,
			request:    PreviewMobileRunnerIDRequest{Name: "x"},
			wantStatus: http.StatusUnauthorized,
			wantReason: "authentication_required",
		},
		{
			name:       "missing validated input",
			noInput:    true,
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_request",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fx := covRunnerSetup(t)
			auth := fx.user
			if tc.admin {
				auth = fx.superuser
			}
			if tc.anonymous {
				auth = nil
			}
			var input any = tc.request
			if tc.noInput {
				input = covRunnerWrongInput{}
			}
			status, body, raw := covRunnerCall(
				t,
				HandlePreviewMobileRunnerID(),
				performMobileRunnerRequest(
					t, fx.app, auth, "/api/mobile-runner/preview-id", input,
				),
			)
			require.Equal(t, tc.wantStatus, status, raw)
			if tc.wantReason != "" {
				assert.Contains(t, raw, tc.wantReason)
				return
			}
			assert.Equal(t, tc.wantRunnerID, body["runner_id"])
			assert.Equal(t, tc.wantConflict, body["conflict"])
			if tc.wantExisting != "" {
				assert.Equal(t, tc.wantExisting, body["existing_runner_id"])
			} else {
				assert.NotContains(t, body, "existing_runner_id")
			}
		})
	}
}

func TestUpsertMobileRunnerConflicts(t *testing.T) {
	testCases := []struct {
		name       string
		request    UpsertMobileRunnerRequest
		wantStatus int
		wantReason string
	}{
		{
			name: "requested runner_id differs from next available id",
			request: UpsertMobileRunnerRequest{
				RunnerID: "usera-s-organization/other-id",
				Name:     "Brand New",
				IP:       "https://x.example",
			},
			wantStatus: http.StatusConflict,
			wantReason: "runner_id_conflict",
		},
		{
			name: "existing runner_id with a different name",
			request: UpsertMobileRunnerRequest{
				RunnerID: "usera-s-organization/lab-runner",
				Name:     "Renamed",
				IP:       "https://x.example",
			},
			wantStatus: http.StatusConflict,
			wantReason: "runner_name_conflict",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fx := covRunnerSetup(t)
			status, _, raw := covRunnerCall(
				t,
				HandleUpsertMobileRunner(),
				performMobileRunnerRequest(
					t, fx.app, fx.user, "/api/mobile-runner", tc.request,
				),
			)
			require.Equal(t, tc.wantStatus, status, raw)
			assert.Contains(t, raw, tc.wantReason)
		})
	}
}

func TestUpsertMobileRunnerSaveFailure(t *testing.T) {
	fx := covRunnerSetup(t)
	fx.app.OnRecordCreate("mobile_runners").BindFunc(func(*core.RecordEvent) error {
		return errors.New("boom")
	})
	status, _, raw := covRunnerCall(t, HandleUpsertMobileRunner(), performMobileRunnerRequest(
		t, fx.app, fx.user, "/api/mobile-runner",
		UpsertMobileRunnerRequest{Name: "fresh", IP: "https://x.example"},
	))
	require.Equal(t, http.StatusInternalServerError, status, raw)
	assert.Contains(t, raw, "failed_to_save_mobile_runner")
}

func TestMobileRunnerRegistrationRejectsBadIdentity(t *testing.T) {
	fx := covRunnerSetup(t)
	covRunnerCreateDevice(t, fx.app, fx.runner, "dev", "redroid", "")
	createMobileRunnerRecord(t, fx.app, fx.otherOrg, "b-runner", "https://b.example", false)

	testCases := []struct {
		name       string
		handler    func() func(*core.RequestEvent) error
		auth       *core.Record
		input      any
		wantStatus int
		wantReason string
	}{
		{
			name:       "anonymous runner upsert",
			handler:    HandleUpsertMobileRunner,
			input:      UpsertMobileRunnerRequest{Name: "x", IP: "https://x.example"},
			wantStatus: http.StatusUnauthorized,
			wantReason: "authentication_required",
		},
		{
			name:    "runner upsert with a device identifier",
			handler: HandleUpsertMobileRunner,
			auth:    fx.user,
			input: UpsertMobileRunnerRequest{
				RunnerID: "usera-s-organization/lab-runner/dev",
				Name:     "dev",
				IP:       "https://x.example",
			},
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_runner_id",
		},
		{
			name:       "runner upsert with wrong input type",
			handler:    HandleUpsertMobileRunner,
			auth:       fx.user,
			input:      covRunnerWrongInput{},
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_request",
		},
		{
			name:    "device preview on a foreign runner",
			handler: HandlePreviewMobileDeviceID,
			auth:    fx.user,
			input: PreviewMobileDeviceIDRequest{
				RunnerID: "userb-s-organization/b-runner",
				Name:     "d",
			},
			wantStatus: http.StatusForbidden,
			wantReason: "runner_id_owner_mismatch",
		},
		{
			name:       "device preview with wrong input type",
			handler:    HandlePreviewMobileDeviceID,
			auth:       fx.user,
			input:      covRunnerWrongInput{},
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_request",
		},
		{
			name:       "superuser reconcile of an unknown runner",
			handler:    HandleReconcileMobileDevices,
			auth:       fx.superuser,
			input:      ReconcileMobileDevicesRequest{RunnerID: "usera-s-organization/ghost"},
			wantStatus: http.StatusNotFound,
			wantReason: "runner_not_found",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status, _, raw := covRunnerCall(t, tc.handler(), performMobileRunnerRequest(
				t, fx.app, tc.auth, "/api/mobile-runner", tc.input,
			))
			require.Equal(t, tc.wantStatus, status, raw)
			assert.Contains(t, raw, tc.wantReason)
		})
	}
}

func covRunnerStubTemporal(
	t *testing.T,
	queueClient func(string) (tclient.Client, error),
	lifecycleClient func(string) (tclient.Client, error),
	queryState func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error),
) {
	t.Helper()
	origQueue := queueTemporalClient
	origLifecycle := mobileRunnerLifecycleTemporalClient
	origQuery := queryMobileDeviceSemaphoreState
	t.Cleanup(func() {
		queueTemporalClient = origQueue
		mobileRunnerLifecycleTemporalClient = origLifecycle
		queryMobileDeviceSemaphoreState = origQuery
	})
	queueTemporalClient = queueClient
	mobileRunnerLifecycleTemporalClient = lifecycleClient
	queryMobileDeviceSemaphoreState = queryState
}

func covRunnerFailingClient(string) (tclient.Client, error) {
	return nil, errors.New("temporal unavailable")
}

func covRunnerStartedClient(t *testing.T) func(string) (tclient.Client, error) {
	mockClient := temporalmocks.NewClient(t)
	mockClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, workflows.MobileDeviceSemaphoreWorkflowName, mock.Anything).
		Return(nil, &serviceerror.WorkflowExecutionAlreadyStarted{}).
		Maybe()
	mockClient.On("UpdateWorkflow", mock.Anything, mock.Anything).
		Return(nil, serviceerror.NewNotFound("no semaphore")).Maybe()
	return func(string) (tclient.Client, error) { return mockClient, nil }
}

func TestMobileRunnerLifecycleRejectsUnresolvableRunners(t *testing.T) {
	handlers := map[string]func() func(*core.RequestEvent) error{
		"resume":    HandleMobileRunnerLifecycleResume,
		"heartbeat": HandleMobileRunnerLifecycleHeartbeat,
		"pause":     HandleMobileRunnerLifecyclePause,
	}
	testCases := []struct {
		name       string
		runnerID   string
		wrongInput bool
		wantStatus int
		wantReason string
	}{
		{
			name:       "wrong input type",
			wrongInput: true,
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_request",
		},
		{
			name:       "blank runner id",
			runnerID:   " / ",
			wantStatus: http.StatusBadRequest,
			wantReason: "runner_id_required",
		},
		{
			name:       "unknown runner",
			runnerID:   "usera-s-organization/ghost",
			wantStatus: http.StatusNotFound,
			wantReason: "mobile_runner_not_found",
		},
		{
			name:       "device instead of runner",
			runnerID:   "usera-s-organization/lab-runner/dev",
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_runner_id",
		},
		{
			name:       "runner of another organization",
			runnerID:   "userb-s-organization/b-runner",
			wantStatus: http.StatusForbidden,
			wantReason: "runner_owner_mismatch",
		},
	}

	fx := covRunnerSetup(t)
	covRunnerCreateDevice(t, fx.app, fx.runner, "dev", "redroid", "")
	createMobileRunnerRecord(t, fx.app, fx.otherOrg, "b-runner", "https://b.example", false)

	for handlerName, handler := range handlers {
		for _, tc := range testCases {
			t.Run(handlerName+"/"+tc.name, func(t *testing.T) {
				var input any = MobileRunnerLifecycleRequest{RunnerID: tc.runnerID}
				if tc.wrongInput {
					input = covRunnerWrongInput{}
				}
				status, _, raw := covRunnerCall(t, handler(), performMobileRunnerRequest(
					t, fx.app, fx.user, "/api/mobile-runner/lifecycle/"+handlerName, input,
				))
				require.Equal(t, tc.wantStatus, status, raw)
				assert.Contains(t, raw, tc.wantReason)
			})
		}
	}
}

func TestMobileRunnerLifecycleRejectsForeignDeviceReportsAtomically(t *testing.T) {
	for name, handler := range map[string]func() func(*core.RequestEvent) error{
		"failed_to_apply_device_resume":    HandleMobileRunnerLifecycleResume,
		"failed_to_apply_device_heartbeat": HandleMobileRunnerLifecycleHeartbeat,
	} {
		t.Run(name, func(t *testing.T) {
			fx := covRunnerSetup(t)
			own := covRunnerCreateDevice(t, fx.app, fx.runner, "own", "redroid", "")
			own.Set("online", true)
			require.NoError(t, fx.app.Save(own))
			createMobileRunnerRecord(t, fx.app, fx.orgID, "second", "https://s.example", false)
			second, err := canonify.Resolve(fx.app, "/usera-s-organization/second")
			require.NoError(t, err)
			covRunnerCreateDevice(t, fx.app, second, "foreign", "redroid", "")

			status, _, raw := covRunnerCall(t, handler(), performMobileRunnerRequest(
				t, fx.app, fx.superuser, "/api/mobile-runner/lifecycle",
				MobileRunnerLifecycleRequest{
					RunnerID: "usera-s-organization/lab-runner",
					Devices: []MobileDeviceLifecycleState{
						{DeviceID: "usera-s-organization/second/foreign", Online: true},
					},
				},
			))
			require.Equal(t, http.StatusBadRequest, status, raw)
			assert.Contains(t, raw, name)

			runner, err := fx.app.FindRecordById("mobile_runners", fx.runner.Id)
			require.NoError(t, err)
			assert.False(t, runner.GetBool("online"), "runner heartbeat must roll back")
			stored, err := fx.app.FindRecordById("mobile_devices", own.Id)
			require.NoError(t, err)
			assert.True(t, stored.GetBool("online"), "unreported device must not be marked offline")
		})
	}
}

func TestMobileRunnerLifecycleSemaphoreFailures(t *testing.T) {
	notFound := func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error) {
		return workflows.MobileDeviceSemaphoreStateView{}, errSemaphoreNotFound
	}
	pausedState := func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error) {
		return workflows.MobileDeviceSemaphoreStateView{Paused: true}, nil
	}
	queryFails := func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error) {
		return workflows.MobileDeviceSemaphoreStateView{}, errors.New("query failed")
	}

	testCases := []struct {
		name         string
		handler      func() func(*core.RequestEvent) error
		online       bool
		queueClient  func(t *testing.T) func(string) (tclient.Client, error)
		lifecycle    func(t *testing.T) func(string) (tclient.Client, error)
		query        func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error)
		wantStatus   int
		wantReason   string
		wantRunnerUp bool
		wantDeviceOn bool
	}{
		{
			name:        "resume cannot ensure the device semaphore",
			handler:     HandleMobileRunnerLifecycleResume,
			online:      true,
			queueClient: func(*testing.T) func(string) (tclient.Client, error) { return covRunnerFailingClient },
			lifecycle:   func(*testing.T) func(string) (tclient.Client, error) { return covRunnerFailingClient },
			query:       notFound,
			wantStatus:  http.StatusInternalServerError,
			wantReason:  "failed_to_ensure_device_semaphore",
		},
		{
			name:        "resume update fails",
			handler:     HandleMobileRunnerLifecycleResume,
			online:      true,
			queueClient: covRunnerStartedClient,
			lifecycle:   func(*testing.T) func(string) (tclient.Client, error) { return covRunnerFailingClient },
			query:       notFound,
			wantStatus:  http.StatusInternalServerError,
			wantReason:  "failed_to_resume_device_semaphore",
		},
		{
			name:         "resume tolerates a missing semaphore",
			handler:      HandleMobileRunnerLifecycleResume,
			online:       true,
			queueClient:  covRunnerStartedClient,
			lifecycle:    covRunnerStartedClient,
			query:        notFound,
			wantStatus:   http.StatusOK,
			wantRunnerUp: true,
			wantDeviceOn: true,
		},
		{
			name:        "heartbeat cannot ensure the device semaphore",
			handler:     HandleMobileRunnerLifecycleHeartbeat,
			online:      true,
			queueClient: func(*testing.T) func(string) (tclient.Client, error) { return covRunnerFailingClient },
			lifecycle:   func(*testing.T) func(string) (tclient.Client, error) { return covRunnerFailingClient },
			query:       notFound,
			wantStatus:  http.StatusInternalServerError,
			wantReason:  "failed_to_ensure_device_semaphore",
		},
		{
			name:        "heartbeat resume of a paused semaphore fails",
			handler:     HandleMobileRunnerLifecycleHeartbeat,
			online:      true,
			queueClient: covRunnerStartedClient,
			lifecycle:   func(*testing.T) func(string) (tclient.Client, error) { return covRunnerFailingClient },
			query:       pausedState,
			wantStatus:  http.StatusInternalServerError,
			wantReason:  "failed_to_resume_device_semaphore",
		},
		{
			name:         "heartbeat resume tolerates a semaphore that vanished",
			handler:      HandleMobileRunnerLifecycleHeartbeat,
			online:       true,
			queueClient:  covRunnerStartedClient,
			lifecycle:    covRunnerStartedClient,
			query:        pausedState,
			wantStatus:   http.StatusOK,
			wantRunnerUp: true,
			wantDeviceOn: true,
		},
		{
			name:        "heartbeat pause after failed state query fails",
			handler:     HandleMobileRunnerLifecycleHeartbeat,
			queueClient: func(*testing.T) func(string) (tclient.Client, error) { return covRunnerFailingClient },
			lifecycle:   func(*testing.T) func(string) (tclient.Client, error) { return covRunnerFailingClient },
			query:       queryFails,
			wantStatus:  http.StatusInternalServerError,
			wantReason:  "failed_to_pause_device_semaphore",
		},
		{
			name:         "heartbeat for offline device without semaphore skips the update",
			handler:      HandleMobileRunnerLifecycleHeartbeat,
			queueClient:  func(*testing.T) func(string) (tclient.Client, error) { return covRunnerFailingClient },
			lifecycle:    func(*testing.T) func(string) (tclient.Client, error) { return covRunnerFailingClient },
			query:        notFound,
			wantStatus:   http.StatusOK,
			wantRunnerUp: true,
		},
		{
			name:        "pause update fails after devices go offline",
			handler:     HandleMobileRunnerLifecyclePause,
			queueClient: func(*testing.T) func(string) (tclient.Client, error) { return covRunnerFailingClient },
			lifecycle:   func(*testing.T) func(string) (tclient.Client, error) { return covRunnerFailingClient },
			query:       notFound,
			wantStatus:  http.StatusInternalServerError,
			wantReason:  "failed_to_pause_device_semaphore",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fx := covRunnerSetup(t)
			device := covRunnerCreateDevice(t, fx.app, fx.runner, "dev", "redroid", "")
			device.Set("online", true)
			require.NoError(t, fx.app.Save(device))
			covRunnerStubTemporal(t, tc.queueClient(t), tc.lifecycle(t), tc.query)

			status, body, raw := covRunnerCall(t, tc.handler(), performMobileRunnerRequest(
				t, fx.app, fx.user, "/api/mobile-runner/lifecycle",
				MobileRunnerLifecycleRequest{
					RunnerID:  "usera-s-organization/lab-runner",
					RequestID: "req-1",
					Devices: []MobileDeviceLifecycleState{
						{DeviceID: "usera-s-organization/lab-runner/dev", Online: tc.online},
					},
				},
			))
			require.Equal(t, tc.wantStatus, status, raw)
			if tc.wantReason != "" {
				assert.Contains(t, raw, tc.wantReason)
			} else {
				assert.Equal(t, "usera-s-organization/lab-runner", body["runner_id"])
			}
			runner, err := fx.app.FindRecordById("mobile_runners", fx.runner.Id)
			require.NoError(t, err)
			stored, err := fx.app.FindRecordById("mobile_devices", device.Id)
			require.NoError(t, err)
			if tc.wantStatus == http.StatusOK {
				assert.Equal(t, tc.wantRunnerUp, runner.GetBool("online"))
				assert.Equal(t, tc.wantDeviceOn, stored.GetBool("online"))
			}
			if tc.name == "pause update fails after devices go offline" {
				assert.False(t, runner.GetBool("online"))
				assert.False(t, stored.GetBool("online"))
			}
		})
	}
}

func TestListMobileDevicesOrdersOwnedOnlineFirst(t *testing.T) {
	fx := covRunnerSetup(t)
	markRunnerHeartbeat(t, fx.app, fx.runner, time.Now())
	online := covRunnerCreateDevice(t, fx.app, fx.runner, "zz-online", "redroid", "")
	online.Set("online", true)
	require.NoError(t, fx.app.Save(online))
	covRunnerCreateDevice(t, fx.app, fx.runner, "aa-offline", "android_phone", "")
	setOrganizationPublished(t, fx.app, fx.orgID, true)
	createMobileRunnerRecord(t, fx.app, fx.otherOrg, "b-runner", "https://b.example", true)
	bRunner, err := canonify.Resolve(fx.app, "/userb-s-organization/b-runner")
	require.NoError(t, err)
	covRunnerCreateDevice(t, fx.app, bRunner, "shared", "redroid", "")

	list := func(t *testing.T, auth *core.Record) (int, ListMobileDevicesPublicResponseSchema, string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/mobile-devices", nil)
		rec := httptest.NewRecorder()
		event := &core.RequestEvent{
			App:   fx.app,
			Auth:  auth,
			Event: router.Event{Request: req, Response: rec},
		}
		err := HandleListMobileDevices()(event)
		if err != nil {
			requireHandlerErrorHandled(t, rec, err)
		}
		var response ListMobileDevicesPublicResponseSchema
		if rec.Code == http.StatusOK {
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		}
		return rec.Code, response, rec.Body.String()
	}
	paths := func(response ListMobileDevicesPublicResponseSchema) []string {
		out := make([]string, 0, len(response.Devices))
		for _, device := range response.Devices {
			out = append(out, device.Path)
		}
		return out
	}

	t.Run("user sees owned online, owned offline, then shared", func(t *testing.T) {
		covRunnerStubTemporal(t, covRunnerFailingClient, covRunnerFailingClient,
			func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error) {
				return workflows.MobileDeviceSemaphoreStateView{}, errSemaphoreNotFound
			})
		status, response, raw := list(t, fx.user)
		require.Equal(t, http.StatusOK, status, raw)
		assert.Equal(t, []string{
			"usera-s-organization/lab-runner/zz-online",
			"usera-s-organization/lab-runner/aa-offline",
			"userb-s-organization/b-runner/shared",
		}, paths(response))
		require.NotNil(t, response.Devices[0].QueueLength)
		assert.Equal(t, 0, *response.Devices[0].QueueLength)
		assert.False(t, response.Devices[2].IsOwned)
		assert.True(t, response.Devices[2].IsPublished)
	})

	t.Run("superuser owns nothing and sorts by online then path", func(t *testing.T) {
		covRunnerStubTemporal(t, covRunnerFailingClient, covRunnerFailingClient,
			func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error) {
				return workflows.MobileDeviceSemaphoreStateView{QueueLen: 4}, nil
			})
		status, response, raw := list(t, fx.superuser)
		require.Equal(t, http.StatusOK, status, raw)
		got := paths(response)
		require.GreaterOrEqual(t, len(got), 3)
		assert.Equal(t, "usera-s-organization/lab-runner/zz-online", got[0])
		require.NotNil(t, response.Devices[0].QueueLength)
		assert.Equal(t, 4, *response.Devices[0].QueueLength)
		for _, device := range response.Devices {
			assert.False(t, device.IsOwned)
		}
		assert.Less(t,
			slices.Index(got, "usera-s-organization/lab-runner/aa-offline"),
			slices.Index(got, "userb-s-organization/b-runner/shared"),
		)
	})

	t.Run("queue query failure is a server error", func(t *testing.T) {
		covRunnerStubTemporal(t, covRunnerFailingClient, covRunnerFailingClient,
			func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error) {
				return workflows.MobileDeviceSemaphoreStateView{}, errors.New("temporal down")
			})
		status, _, raw := list(t, fx.user)
		require.Equal(t, http.StatusInternalServerError, status, raw)
		assert.Contains(t, raw, "failed_to_query_device_queue")
	})

	t.Run("anonymous caller", func(t *testing.T) {
		status, _, raw := list(t, nil)
		require.Equal(t, http.StatusUnauthorized, status, raw)
		assert.Contains(t, raw, "authentication_required")
	})
}
