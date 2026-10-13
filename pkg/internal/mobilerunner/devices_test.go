// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data"

const (
	orgAID = "co35481b68u3zj3"
	orgBID = "3u4982xn6ah0433"
)

func TestRunnerURL(t *testing.T) {
	coll := core.NewBaseCollection("mobile_runners")
	for _, tc := range []struct {
		name string
		ip   string
		port string
		want string
	}{
		{name: "empty ip", ip: " ", port: "8050", want: ""},
		{name: "no port", ip: "http://runner.test", want: "http://runner.test"},
		{name: "with port", ip: " http://runner.test ", port: " 8050 ", want: "http://runner.test:8050"},
		{name: "trailing slash with port", ip: "http://runner.test/", port: "8050", want: "http://runner.test:8050"},
		{name: "trailing slash without port", ip: "http://runner.test/", want: "http://runner.test/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := core.NewRecord(coll)
			record.Set("ip", tc.ip)
			record.Set("port", tc.port)
			require.Equal(t, tc.want, RunnerURL(record))
		})
	}
}

func TestNormalizeDeviceIDs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []string
		want   []string
	}{
		{name: "nil", values: nil, want: []string{}},
		{
			name:   "comma lists and duplicates",
			values: []string{" runner-2 , runner-1", "runner-2", "  ", "runner-3"},
			want:   []string{"runner-1", "runner-2", "runner-3"},
		},
		{
			name:   "leading slash",
			values: []string{" /tenant/runner-2 , /tenant/runner-1", "tenant/runner-2"},
			want:   []string{"tenant/runner-1", "tenant/runner-2"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, NormalizeDeviceIDs(tc.values))
		})
	}
}

func TestRunnerIdentifier(t *testing.T) {
	app := newDevicesTestApp(t)
	runner := createRunner(t, app, orgAID, "Runner One", false, false)

	got, err := RunnerIdentifier(app, runner)
	require.NoError(t, err)
	require.Equal(t, "usera-s-organization/runner-one", got)
}

func TestValidateDeviceAccess(t *testing.T) {
	for _, tc := range []struct {
		name          string
		published     bool
		adminManaged  bool
		orgPublished  bool
		deviceOwner   string
		deviceID      func(t *testing.T, app *tests.TestApp, device *core.Record) string
		wantErr       error
		wantErrSubstr string
	}{
		{
			name:        "owned device",
			deviceOwner: orgAID,
		},
		{
			name:        "unknown device",
			deviceOwner: orgAID,
			deviceID: func(*testing.T, *tests.TestApp, *core.Record) string {
				return "usera-s-organization/missing/device"
			},
			wantErr: ErrDeviceNotFound,
		},
		{
			name:        "record outside mobile_devices",
			deviceOwner: orgAID,
			deviceID: func(t *testing.T, app *tests.TestApp, device *core.Record) string {
				runner, err := app.FindRecordById("mobile_runners", device.GetString("runner"))
				require.NoError(t, err)
				id, err := RunnerIdentifier(app, runner)
				require.NoError(t, err)
				return id
			},
			wantErr: ErrDeviceNotFound,
		},
		{
			name:          "private foreign device",
			deviceOwner:   orgBID,
			wantErr:       ErrDeviceNotAccessible,
			wantErrSubstr: "is private and does not belong to the caller organization",
		},
		{
			name:         "published runner and published org",
			deviceOwner:  orgBID,
			published:    true,
			orgPublished: true,
		},
		{
			name:         "published unmanaged runner and unpublished org",
			deviceOwner:  orgBID,
			published:    true,
			orgPublished: false,
			wantErr:      ErrDeviceNotAccessible,
		},
		{
			name:         "admin managed runner and unpublished org",
			deviceOwner:  orgBID,
			published:    true,
			adminManaged: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newDevicesTestApp(t)
			if tc.orgPublished {
				org, err := app.FindRecordById("organizations", orgAID)
				require.NoError(t, err)
				org.Set("published", true)
				require.NoError(t, app.Save(org))
			}
			runner := createRunner(
				t,
				app,
				tc.deviceOwner,
				"Runner One",
				tc.published,
				tc.adminManaged,
			)
			device := createDevice(t, app, tc.deviceOwner, runner.Id, "Device One")

			deviceID := devicePath(t, app, device)
			if tc.deviceID != nil {
				deviceID = tc.deviceID(t, app, device)
			}

			err := ValidateDeviceAccess(app, orgAID, []string{deviceID})
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
			require.Contains(t, err.Error(), tc.wantErrSubstr)
		})
	}
}

func TestResolveDevice(t *testing.T) {
	for _, tc := range []struct {
		name      string
		breakDB   string
		wantErr   error
		wantOther bool
	}{
		{name: "device and runner found"},
		{
			name:    "device record deleted",
			breakDB: "DELETE FROM mobile_devices",
			wantErr: ErrDeviceNotFound,
		},
		{
			name:      "device lookup fails",
			breakDB:   "DROP TABLE mobile_devices",
			wantOther: true,
		},
		{
			name:      "runner lookup fails",
			breakDB:   "DROP TABLE mobile_runners",
			wantOther: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newDevicesTestApp(t)
			runner := createRunner(t, app, orgAID, "Runner One", false, false)
			device := createDevice(t, app, orgAID, runner.Id, "Device One")
			deviceID := devicePath(t, app, device)
			if tc.breakDB != "" {
				_, err := app.DB().NewQuery(tc.breakDB).Execute()
				require.NoError(t, err)
			}

			gotDevice, gotRunner, err := ResolveDevice(app, deviceID)
			switch {
			case tc.wantErr != nil:
				require.ErrorIs(t, err, tc.wantErr)
			case tc.wantOther:
				require.Error(t, err)
				require.NotErrorIs(t, err, ErrDeviceNotFound)
				require.NotErrorIs(t, err, ErrDeviceRunnerNotFound)
				// ValidateDeviceAccess must not report a lookup failure as not found.
				accessErr := ValidateDeviceAccess(app, orgAID, []string{deviceID})
				require.Error(t, accessErr)
				require.NotErrorIs(t, accessErr, ErrDeviceRunnerNotFound)
			default:
				require.NoError(t, err)
				require.Equal(t, device.Id, gotDevice.Id)
				require.Equal(t, runner.Id, gotRunner.Id)
			}
		})
	}
}

func TestResolveRunner(t *testing.T) {
	app := newDevicesTestApp(t)
	runner := createRunner(t, app, orgAID, "Resolved Runner", false, false)
	device := createDevice(t, app, orgAID, runner.Id, "Device One")
	runnerID, err := RunnerIdentifier(app, runner)
	require.NoError(t, err)

	for _, tc := range []struct {
		name    string
		id      string
		wantErr error
	}{
		{name: "runner", id: runnerID},
		{name: "leading slash", id: "/" + runnerID},
		{name: "unknown path", id: runnerID + "-missing", wantErr: ErrRunnerNotFound},
		{name: "device path", id: devicePath(t, app, device), wantErr: ErrRunnerNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveRunner(app, tc.id)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, runner.Id, got.Id)
		})
	}
}

func newDevicesTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	canonify.RegisterCanonifyHooks(app)
	return app
}

func createRunner(
	t *testing.T,
	app *tests.TestApp,
	ownerID string,
	name string,
	published bool,
	adminManaged bool,
) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(coll)
	runner.Set("owner", ownerID)
	runner.Set("name", name)
	runner.Set("ip", "http://runner.test")
	runner.Set("type", "android_emulator")
	runner.Set("published", published)
	runner.Set("admin_managed", adminManaged)
	require.NoError(t, app.Save(runner))
	return runner
}

func createDevice(t *testing.T, app *tests.TestApp, ownerID, runnerID, name string) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(coll)
	device.Set("owner", ownerID)
	device.Set("runner", runnerID)
	device.Set("name", name)
	device.Set("type", "android_emulator")
	device.Set("serial", name+"-serial")
	require.NoError(t, app.Save(device))
	return device
}

func devicePath(t *testing.T, app *tests.TestApp, device *core.Record) string {
	t.Helper()
	path, err := canonify.BuildPath(app, device, canonify.CanonifyPaths["mobile_devices"], "")
	require.NoError(t, err)
	return canonify.NormalizePath(path)
}
