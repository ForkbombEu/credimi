// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func requireRegistrationAPIError(
	t testing.TB,
	err error,
	wantCode int,
	wantDomain string,
	wantReason string,
) *apierror.APIError {
	t.Helper()

	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, wantCode, apiErr.Code)
	require.Equal(t, wantDomain, apiErr.Domain)
	require.Equal(t, wantReason, apiErr.Reason)
	return apiErr
}

type registrationFixture struct {
	app       *tests.TestApp
	user      *core.Record
	otherUser *core.Record
	superuser *core.Record
	orgID     string
	otherOrg  string
}

func setupRegistrationFixture(t *testing.T) registrationFixture {
	t.Helper()

	app := setupMobileRunnerApp(t)
	t.Cleanup(app.Cleanup)

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	otherUser, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	otherOrg, err := pbutils.GetUserOrganizationID(app, otherUser.Id)
	require.NoError(t, err)

	createMobileRunnerRecord(t, app, orgID, "Lab Runner", "https://lab.example", false)
	createMobileRunnerRecord(t, app, otherOrg, "Other Runner", "https://other.example", false)

	return registrationFixture{
		app:       app,
		user:      user,
		otherUser: otherUser,
		superuser: superuser,
		orgID:     orgID,
		otherOrg:  otherOrg,
	}
}

func (f registrationFixture) upsertDevice(
	t *testing.T,
	auth *core.Record,
	input UpsertMobileDeviceRequest,
) (*core.RequestEvent, error) {
	t.Helper()

	event := performMobileRunnerRequest(t, f.app, auth, "/api/mobile-device", input)
	return event, HandleUpsertMobileDevice()(event)
}

func (f registrationFixture) mustUpsertDevice(
	t *testing.T,
	input UpsertMobileDeviceRequest,
) map[string]any {
	t.Helper()

	event, err := f.upsertDevice(t, f.user, input)
	require.NoError(t, err)
	recorder := responseRecorder(t, event)
	require.Equal(t, http.StatusOK, recorder.Code)
	return decodeJSONBody(t, recorder)
}

func TestUpsertMobileDevice(t *testing.T) {
	const runnerID = "usera-s-organization/lab-runner"

	t.Run("user create persists a trimmed device under the runner", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		liveStream := true

		body := f.mustUpsertDevice(t, UpsertMobileDeviceRequest{
			RunnerID:    runnerID,
			Name:        " Pixel 7a ",
			Description: " bench phone ",
			Type:        " android_phone ",
			Serial:      " SER-1 ",
			LiveStream:  &liveStream,
		})
		require.Equal(t, runnerID+"/pixel-7a", body["device_id"])
		require.Equal(t, "Pixel 7a", body["name"])
		require.Equal(t, "pixel-7a", body["canonified_name"])
		require.Equal(t, true, body["live_stream"])

		record, err := canonify.Resolve(f.app, "/"+runnerID+"/pixel-7a")
		require.NoError(t, err)
		require.Equal(t, body["id"], record.Id)
		require.Equal(t, f.orgID, record.GetString("owner"))
		require.Equal(t, "bench phone", record.GetString("description"))
		require.Equal(t, "android_phone", record.GetString("type"))
		require.Equal(t, "SER-1", record.GetString("serial"))
		require.True(t, record.GetBool("live_stream"))
	})

	t.Run("update by device_id keeps the record and an absent live_stream", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		liveStream := true
		created := f.mustUpsertDevice(t, UpsertMobileDeviceRequest{
			RunnerID:   runnerID,
			Name:       "Pixel",
			Type:       "android_phone",
			Serial:     "SER-1",
			LiveStream: &liveStream,
		})

		// Re-registering with the device's own serial must not conflict with itself.
		updated := f.mustUpsertDevice(t, UpsertMobileDeviceRequest{
			RunnerID:    runnerID,
			DeviceID:    "/" + runnerID + "/pixel",
			Name:        "Pixel",
			Description: "re-registered",
			Type:        "android_phone",
			Serial:      "SER-1",
		})
		require.Equal(t, created["id"], updated["id"])
		require.Equal(t, true, updated["live_stream"])

		record, err := f.app.FindRecordById("mobile_devices", created["id"].(string))
		require.NoError(t, err)
		require.Equal(t, "re-registered", record.GetString("description"))
	})

	t.Run("existing device cannot be renamed", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		f.mustUpsertDevice(t, UpsertMobileDeviceRequest{
			RunnerID: runnerID,
			Name:     "Pixel",
			Type:     "android_emulator",
		})

		_, err := f.upsertDevice(t, f.user, UpsertMobileDeviceRequest{
			RunnerID: runnerID,
			DeviceID: runnerID + "/pixel",
			Name:     "Renamed Pixel",
			Type:     "android_emulator",
		})
		requireRegistrationAPIError(
			t, err, http.StatusConflict, "device_id", "device_identity_immutable",
		)
	})

	t.Run("create rejects a device_id that differs from the next available id", func(t *testing.T) {
		f := setupRegistrationFixture(t)

		_, err := f.upsertDevice(t, f.user, UpsertMobileDeviceRequest{
			RunnerID: runnerID,
			DeviceID: runnerID + "/something-else",
			Name:     "Pixel",
			Type:     "android_emulator",
		})
		requireRegistrationAPIError(t, err, http.StatusConflict, "device_id", "device_id_conflict")

		records, findErr := f.app.FindAllRecords("mobile_devices")
		require.NoError(t, findErr)
		for _, record := range records {
			require.NotEqual(t, "Pixel", record.GetString("name"))
		}
	})

	t.Run("constraint violations", func(t *testing.T) {
		cases := []struct {
			name        string
			existing    UpsertMobileDeviceRequest
			input       UpsertMobileDeviceRequest
			wantCode    int
			wantDomain  string
			wantReason  string
			wantMessage string
		}{
			{
				name:       "unknown device type",
				input:      UpsertMobileDeviceRequest{Name: "Box", Type: "desktop"},
				wantCode:   http.StatusBadRequest,
				wantDomain: "type",
				wantReason: "invalid_device_type",
			},
			{
				name:        "second android emulator on the same runner",
				existing:    UpsertMobileDeviceRequest{Name: "Emu One", Type: "android_emulator"},
				input:       UpsertMobileDeviceRequest{Name: "Emu Two", Type: "android_emulator"},
				wantCode:    http.StatusConflict,
				wantDomain:  "type",
				wantReason:  "device_type_limit",
				wantMessage: `runner already has android_emulator device "Emu One"; only one is allowed`,
			},
			{
				name:     "second ios simulator on the same runner",
				existing: UpsertMobileDeviceRequest{Name: "Sim One", Type: "ios_simulator"},
				input:    UpsertMobileDeviceRequest{Name: "Sim Two", Type: "ios_simulator"},
				wantCode: http.StatusConflict, wantDomain: "type", wantReason: "device_type_limit",
			},
			{
				name: "redroid reusing an android phone serial",
				existing: UpsertMobileDeviceRequest{
					Name: "Phone", Type: "android_phone", Serial: "SER-1",
				},
				input: UpsertMobileDeviceRequest{
					Name: "Container", Type: "redroid", Serial: " SER-1 ",
				},
				wantCode:    http.StatusConflict,
				wantDomain:  "serial",
				wantReason:  "device_serial_conflict",
				wantMessage: `serial "SER-1" is already registered for device "Phone"; choose a different serial for device "Container"`,
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				f := setupRegistrationFixture(t)
				if tc.existing.Name != "" {
					tc.existing.RunnerID = runnerID
					f.mustUpsertDevice(t, tc.existing)
				}
				tc.input.RunnerID = runnerID

				_, err := f.upsertDevice(t, f.user, tc.input)
				apiErr := requireRegistrationAPIError(
					t, err, tc.wantCode, tc.wantDomain, tc.wantReason,
				)
				if tc.wantMessage != "" {
					require.Equal(t, tc.wantMessage, apiErr.Message)
				}
			})
		}
	})

	t.Run("devices of different kinds may coexist on one runner", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		f.mustUpsertDevice(t, UpsertMobileDeviceRequest{
			RunnerID: runnerID, Name: "Emu", Type: "android_emulator", Serial: "SER-1",
		})
		f.mustUpsertDevice(t, UpsertMobileDeviceRequest{
			RunnerID: runnerID, Name: "Sim", Type: "ios_simulator",
		})
		// Serial uniqueness only applies among physical/redroid devices.
		f.mustUpsertDevice(t, UpsertMobileDeviceRequest{
			RunnerID: runnerID, Name: "Phone", Type: "android_phone", Serial: "SER-1",
		})
		f.mustUpsertDevice(t, UpsertMobileDeviceRequest{
			RunnerID: runnerID, Name: "Phone Two", Type: "android_phone",
		})
		f.mustUpsertDevice(t, UpsertMobileDeviceRequest{
			RunnerID: runnerID, Name: "Phone Three", Type: "android_phone",
		})
	})

	t.Run("ownership and resolution failures", func(t *testing.T) {
		cases := []struct {
			name       string
			auth       func(registrationFixture) *core.Record
			input      UpsertMobileDeviceRequest
			wantCode   int
			wantDomain string
			wantReason string
		}{
			{
				name:       "unauthenticated caller",
				auth:       func(registrationFixture) *core.Record { return nil },
				input:      UpsertMobileDeviceRequest{RunnerID: runnerID},
				wantCode:   http.StatusUnauthorized,
				wantDomain: "auth",
				wantReason: "authentication_required",
			},
			{
				name: "user targets another organization's runner",
				auth: func(f registrationFixture) *core.Record { return f.user },
				input: UpsertMobileDeviceRequest{
					RunnerID: "userb-s-organization/other-runner",
				},
				wantCode:   http.StatusForbidden,
				wantDomain: "runner_id",
				wantReason: "runner_id_owner_mismatch",
			},
			{
				name:       "user targets an unknown runner",
				auth:       func(f registrationFixture) *core.Record { return f.user },
				input:      UpsertMobileDeviceRequest{RunnerID: "usera-s-organization/missing"},
				wantCode:   http.StatusNotFound,
				wantDomain: "runner_id",
				wantReason: "runner_not_found",
			},
			{
				name:       "runner_id references an organization",
				auth:       func(f registrationFixture) *core.Record { return f.user },
				input:      UpsertMobileDeviceRequest{RunnerID: "usera-s-organization"},
				wantCode:   http.StatusBadRequest,
				wantDomain: "runner_id",
				wantReason: "invalid_runner_id",
			},
			{
				name:       "superuser without organization targets an unknown runner",
				auth:       func(f registrationFixture) *core.Record { return f.superuser },
				input:      UpsertMobileDeviceRequest{RunnerID: "usera-s-organization/missing"},
				wantCode:   http.StatusNotFound,
				wantDomain: "runner_id",
				wantReason: "runner_not_found",
			},
			{
				name: "superuser organization conflicts with the runner's owner",
				auth: func(f registrationFixture) *core.Record { return f.superuser },
				input: UpsertMobileDeviceRequest{
					Organization: "userb-s-organization",
					RunnerID:     runnerID,
				},
				wantCode:   http.StatusForbidden,
				wantDomain: "runner_id",
				wantReason: "runner_id_owner_mismatch",
			},
			{
				name: "device_id references the runner instead of a device",
				auth: func(f registrationFixture) *core.Record { return f.user },
				input: UpsertMobileDeviceRequest{
					RunnerID: runnerID,
					DeviceID: runnerID,
				},
				wantCode:   http.StatusBadRequest,
				wantDomain: "device_id",
				wantReason: "invalid_device_id",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				f := setupRegistrationFixture(t)
				tc.input.Name = "Pixel"
				tc.input.Type = "android_emulator"

				_, err := f.upsertDevice(t, tc.auth(f), tc.input)
				requireRegistrationAPIError(t, err, tc.wantCode, tc.wantDomain, tc.wantReason)
			})
		}
	})

	t.Run("device_id of another organization is rejected", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		otherRunner, err := canonify.Resolve(f.app, "/userb-s-organization/other-runner")
		require.NoError(t, err)
		otherDeviceID := createMobileDeviceForLifecycleTest(t, f.app, otherRunner, "foreign")

		_, err = f.upsertDevice(t, f.user, UpsertMobileDeviceRequest{
			RunnerID: runnerID,
			DeviceID: otherDeviceID,
			Name:     "foreign",
			Type:     "android_emulator",
		})
		requireRegistrationAPIError(
			t, err, http.StatusForbidden, "device_id", "device_id_owner_mismatch",
		)
	})

	t.Run("superuser without organization registers under the runner's owner", func(t *testing.T) {
		f := setupRegistrationFixture(t)

		event, err := f.upsertDevice(t, f.superuser, UpsertMobileDeviceRequest{
			RunnerID: "/userb-s-organization/other-runner",
			Name:     "Admin Device",
			Type:     "ios_simulator",
		})
		require.NoError(t, err)
		body := decodeJSONBody(t, responseRecorder(t, event))
		require.Equal(t, "userb-s-organization/other-runner/admin-device", body["device_id"])

		record, err := f.app.FindRecordById("mobile_devices", body["id"].(string))
		require.NoError(t, err)
		require.Equal(t, f.otherOrg, record.GetString("owner"))
	})
}

func TestReconcileMobileDevices(t *testing.T) {
	const runnerID = "usera-s-organization/lab-runner"

	t.Run("removes only devices absent from the declared inventory", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		runner, err := canonify.Resolve(f.app, "/"+runnerID)
		require.NoError(t, err)
		keptID := createMobileDeviceForLifecycleTest(t, f.app, runner, "kept")
		staleID := createMobileDeviceForLifecycleTest(t, f.app, runner, "stale")
		otherRunner, err := canonify.Resolve(f.app, "/userb-s-organization/other-runner")
		require.NoError(t, err)
		foreignID := createMobileDeviceForLifecycleTest(t, f.app, otherRunner, "foreign")

		event := performMobileRunnerRequest(
			t,
			f.app,
			f.user,
			"/api/mobile-device/reconcile",
			ReconcileMobileDevicesRequest{
				RunnerID:  runnerID,
				DeviceIDs: []string{" /" + keptID},
			},
		)
		require.NoError(t, HandleReconcileMobileDevices()(event))
		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(
			t,
			map[string]any{"removed_device_ids": []any{staleID}},
			decodeJSONBody(t, recorder),
		)

		_, err = canonify.Resolve(f.app, "/"+keptID)
		require.NoError(t, err)
		_, err = canonify.Resolve(f.app, "/"+staleID)
		require.Error(t, err)
		_, err = canonify.Resolve(f.app, "/"+foreignID)
		require.NoError(t, err, "devices of other runners must never be reconciled away")
	})

	t.Run("empty inventory reports an empty removal list", func(t *testing.T) {
		f := setupRegistrationFixture(t)

		event := performMobileRunnerRequest(
			t,
			f.app,
			f.user,
			"/api/mobile-device/reconcile",
			ReconcileMobileDevicesRequest{RunnerID: runnerID},
		)
		require.NoError(t, HandleReconcileMobileDevices()(event))
		require.Equal(
			t,
			map[string]any{"removed_device_ids": []any{}},
			decodeJSONBody(t, responseRecorder(t, event)),
		)
	})

	t.Run("unknown runner is not found", func(t *testing.T) {
		f := setupRegistrationFixture(t)

		event := performMobileRunnerRequest(
			t,
			f.app,
			f.user,
			"/api/mobile-device/reconcile",
			ReconcileMobileDevicesRequest{RunnerID: "usera-s-organization/missing"},
		)
		requireRegistrationAPIError(
			t,
			HandleReconcileMobileDevices()(event),
			http.StatusNotFound,
			"runner_id",
			"runner_not_found",
		)
	})

	t.Run("another organization's runner is forbidden and untouched", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		otherRunner, err := canonify.Resolve(f.app, "/userb-s-organization/other-runner")
		require.NoError(t, err)
		foreignID := createMobileDeviceForLifecycleTest(t, f.app, otherRunner, "foreign")

		event := performMobileRunnerRequest(
			t,
			f.app,
			f.user,
			"/api/mobile-device/reconcile",
			ReconcileMobileDevicesRequest{RunnerID: "userb-s-organization/other-runner"},
		)
		requireRegistrationAPIError(
			t,
			HandleReconcileMobileDevices()(event),
			http.StatusForbidden,
			"runner_id",
			"runner_id_owner_mismatch",
		)
		_, err = canonify.Resolve(f.app, "/"+foreignID)
		require.NoError(t, err)
	})
}

func TestDeleteMobileDevice(t *testing.T) {
	const runnerID = "usera-s-organization/lab-runner"

	t.Run("owner deletes a device and gets its normalized id back", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		runner, err := canonify.Resolve(f.app, "/"+runnerID)
		require.NoError(t, err)
		deviceID := createMobileDeviceForLifecycleTest(t, f.app, runner, "doomed")

		event := performMobileRunnerRequest(
			t,
			f.app,
			f.user,
			"/api/mobile-device",
			DeleteMobileDeviceRequest{RunnerID: runnerID, DeviceID: "/" + deviceID},
		)
		require.NoError(t, HandleDeleteMobileDevice()(event))
		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Equal(t, map[string]any{"device_id": deviceID}, decodeJSONBody(t, recorder))

		_, err = canonify.Resolve(f.app, "/"+deviceID)
		require.Error(t, err)
	})

	t.Run("missing device is not found", func(t *testing.T) {
		f := setupRegistrationFixture(t)

		event := performMobileRunnerRequest(
			t,
			f.app,
			f.user,
			"/api/mobile-device",
			DeleteMobileDeviceRequest{RunnerID: runnerID, DeviceID: runnerID + "/ghost"},
		)
		requireRegistrationAPIError(
			t,
			HandleDeleteMobileDevice()(event),
			http.StatusNotFound,
			"device_id",
			"device_not_found",
		)
	})

	t.Run("user cannot delete another organization's device", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		otherRunner, err := canonify.Resolve(f.app, "/userb-s-organization/other-runner")
		require.NoError(t, err)
		foreignID := createMobileDeviceForLifecycleTest(t, f.app, otherRunner, "foreign")

		event := performMobileRunnerRequest(
			t,
			f.app,
			f.user,
			"/api/mobile-device",
			DeleteMobileDeviceRequest{
				RunnerID: "userb-s-organization/other-runner",
				DeviceID: foreignID,
			},
		)
		requireRegistrationAPIError(
			t,
			HandleDeleteMobileDevice()(event),
			http.StatusForbidden,
			"device_id",
			"device_id_owner_mismatch",
		)
		_, err = canonify.Resolve(f.app, "/"+foreignID)
		require.NoError(t, err)
	})

	t.Run("superuser deletes using the runner's owner", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		otherRunner, err := canonify.Resolve(f.app, "/userb-s-organization/other-runner")
		require.NoError(t, err)
		foreignID := createMobileDeviceForLifecycleTest(t, f.app, otherRunner, "foreign")

		event := performMobileRunnerRequest(
			t,
			f.app,
			f.superuser,
			"/api/mobile-device",
			DeleteMobileDeviceRequest{
				RunnerID: "userb-s-organization/other-runner",
				DeviceID: foreignID,
			},
		)
		require.NoError(t, HandleDeleteMobileDevice()(event))
		_, err = canonify.Resolve(f.app, "/"+foreignID)
		require.Error(t, err)
	})
}

func TestPreviewMobileRunnerID(t *testing.T) {
	t.Run("user preview reports a conflict with the existing runner", func(t *testing.T) {
		f := setupRegistrationFixture(t)

		event := performMobileRunnerRequest(
			t,
			f.app,
			f.user,
			"/api/mobile-runner/preview-id",
			PreviewMobileRunnerIDRequest{Name: " Lab Runner "},
		)
		require.NoError(t, HandlePreviewMobileRunnerID()(event))
		body := decodeJSONBody(t, responseRecorder(t, event))
		require.Equal(t, "usera-s-organization", body["organization"])
		require.Equal(t, true, body["conflict"])
		require.Equal(t, "usera-s-organization/lab-runner", body["existing_runner_id"])
		require.Equal(t, "lab-runner-1", body["canonified_name"])
		require.Equal(t, "usera-s-organization/lab-runner-1", body["runner_id"])
	})

	t.Run("user preview of a free name has no conflict", func(t *testing.T) {
		f := setupRegistrationFixture(t)

		event := performMobileRunnerRequest(
			t,
			f.app,
			f.user,
			"/api/mobile-runner/preview-id",
			// organization is ignored for non-admin callers
			PreviewMobileRunnerIDRequest{Organization: "userb-s-organization", Name: "Fresh"},
		)
		require.NoError(t, HandlePreviewMobileRunnerID()(event))
		body := decodeJSONBody(t, responseRecorder(t, event))
		require.Equal(t, "usera-s-organization/fresh", body["runner_id"])
		require.Equal(t, false, body["conflict"])
		require.NotContains(t, body, "existing_runner_id")
	})

	t.Run("owner resolution failures", func(t *testing.T) {
		cases := []struct {
			name       string
			admin      bool
			org        string
			wantCode   int
			wantReason string
		}{
			{
				name:       "unauthenticated",
				wantCode:   http.StatusUnauthorized,
				wantReason: "authentication_required",
			},
			{
				name:       "superuser without organization",
				admin:      true,
				org:        "  ",
				wantCode:   http.StatusBadRequest,
				wantReason: "organization_required",
			},
			{
				name:       "superuser with unknown organization",
				admin:      true,
				org:        "no-such-org",
				wantCode:   http.StatusNotFound,
				wantReason: "organization_not_found",
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				f := setupRegistrationFixture(t)
				var auth *core.Record
				wantDomain := "auth"
				if tc.admin {
					auth = f.superuser
					wantDomain = "organization"
				}

				event := performMobileRunnerRequest(
					t,
					f.app,
					auth,
					"/api/mobile-runner/preview-id",
					PreviewMobileRunnerIDRequest{Organization: tc.org, Name: "Runner"},
				)
				requireRegistrationAPIError(
					t,
					HandlePreviewMobileRunnerID()(event),
					tc.wantCode,
					wantDomain,
					tc.wantReason,
				)
			})
		}
	})
}

func TestUpsertMobileRunnerRejectsIdentityChanges(t *testing.T) {
	cases := []struct {
		name       string
		input      UpsertMobileRunnerRequest
		wantCode   int
		wantDomain string
		wantReason string
	}{
		{
			name: "renaming an existing runner",
			input: UpsertMobileRunnerRequest{
				RunnerID: "usera-s-organization/lab-runner",
				Name:     "Renamed Runner",
			},
			wantCode:   http.StatusConflict,
			wantDomain: "name",
			wantReason: "runner_name_conflict",
		},
		{
			name: "taking over another organization's runner",
			input: UpsertMobileRunnerRequest{
				RunnerID: "userb-s-organization/other-runner",
				Name:     "Other Runner",
			},
			wantCode:   http.StatusForbidden,
			wantDomain: "runner_id",
			wantReason: "runner_id_owner_mismatch",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := setupRegistrationFixture(t)
			tc.input.IP = "https://hijack.example"

			event := performMobileRunnerRequest(t, f.app, f.user, "/api/mobile-runner", tc.input)
			requireRegistrationAPIError(
				t,
				HandleUpsertMobileRunner()(event),
				tc.wantCode,
				tc.wantDomain,
				tc.wantReason,
			)

			runner, err := canonify.Resolve(f.app, "/"+canonify.NormalizePath(tc.input.RunnerID))
			require.NoError(t, err)
			require.NotEqual(t, "https://hijack.example", runner.GetString("ip"))
		})
	}
}

func TestCanonifyPreviewError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "exhausted name suffixes",
			err:        canonify.ErrExhaustedAttempts,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "wrapped exhausted name suffixes",
			err:        fmt.Errorf("canonify: %w", canonify.ErrExhaustedAttempts),
			wantStatus: http.StatusConflict,
		},
		{
			name:       "database failure",
			err:        errors.New("canonify: resolve path: no such table"),
			wantStatus: http.StatusInternalServerError,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := canonifyPreviewError("failed_to_canonify_device_name", tc.err)
			require.Equal(t, tc.wantStatus, apiErr.Code)
			require.Equal(t, "failed_to_canonify_device_name", apiErr.Reason)
			require.Equal(t, tc.err.Error(), apiErr.Message)
		})
	}
}

func TestPreviewMobileDeviceIDDatabaseFailure(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, orgID, "Existing Runner", "http://runner.test", false)
	// The device name lookup now fails with a database error, not a conflict.
	_, err = app.DB().NewQuery("DROP TABLE mobile_devices").Execute()
	require.NoError(t, err)

	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-device/preview-id",
		PreviewMobileDeviceIDRequest{
			RunnerID: "usera-s-organization/existing-runner",
			Name:     "Pixel-7a",
		},
	)

	err = HandlePreviewMobileDeviceID()(event)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusInternalServerError, apiErr.Code)
	require.Equal(t, "failed_to_canonify_device_name", apiErr.Reason)
}
