// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func createTestRunner(
	t *testing.T,
	app *tests.TestApp,
	ownerID string,
	name string,
	port string,
) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(coll)
	runner.Set("owner", ownerID)
	runner.Set("name", name)
	runner.Set("ip", "http://runner.test/")
	runner.Set("port", port)
	runner.Set("type", "android_emulator")
	require.NoError(t, app.Save(runner))
	return runner
}

func createTestDevice(t *testing.T, app *tests.TestApp, ownerID, runnerID, name string) string {
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
	path, err := canonify.BuildPath(app, device, canonify.CanonifyPaths["mobile_devices"], "")
	require.NoError(t, err)
	return canonify.NormalizePath(path)
}

func TestGetMobileDeviceActivity(t *testing.T) {
	app := newCredimiTestApp(t)
	runner := createTestRunner(t, app, testOrgAID, "Runner One", "8050")
	deviceID := createTestDevice(t, app, testOrgAID, runner.Id, "Device One")
	runnerPath, err := canonify.BuildPath(app, runner, canonify.CanonifyPaths["mobile_runners"], "")
	require.NoError(t, err)

	cases := []struct {
		name       string
		identifier string
		wantErr    string
		want       map[string]any
	}{
		{
			name:       "device with a runner port",
			identifier: deviceID,
			want: map[string]any{
				"device_id":  deviceID,
				"runner_id":  canonify.NormalizePath(runnerPath),
				"type":       "android_emulator",
				"serial":     "Device One-serial",
				"runner_url": "http://runner.test:8050",
			},
		},
		{
			name:       "unknown identifier",
			identifier: testOrgANamespace + "/missing/device",
			wantErr:    errorcodes.RecordNotFound,
		},
		{
			name:       "identifier of a runner",
			identifier: canonify.NormalizePath(runnerPath),
			wantErr:    errorcodes.RecordNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := executeActivity(t, NewGetMobileDeviceActivity(app), GetMobileDeviceInput{
				DeviceIdentifier: tc.identifier,
			})
			if tc.wantErr != "" {
				requireActivityError(t, err, tc.wantErr, true)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, result.Output)
		})
	}
}

func TestValidateDeviceAccessActivity(t *testing.T) {
	cases := []struct {
		name      string
		owner     string
		namespace string
		breakDB   string
		wantErr   string
		retryable bool
	}{
		{
			name:      "owned device",
			owner:     testOrgAID,
			namespace: testOrgANamespace,
		},
		{
			name:      "private foreign device",
			owner:     testOrgBID,
			namespace: testOrgANamespace,
			wantErr:   errorcodes.RecordNotAccessible,
		},
		{
			name:      "unknown organization",
			owner:     testOrgAID,
			namespace: "missing-org",
			wantErr:   errorcodes.RecordNotFound,
		},
		{
			name:      "device runner lookup fails",
			owner:     testOrgAID,
			namespace: testOrgANamespace,
			breakDB:   "DROP TABLE mobile_runners",
			wantErr:   errorcodes.DatabaseOperationFailed,
			retryable: true,
		},
		{
			name:      "organization lookup fails",
			owner:     testOrgAID,
			namespace: testOrgANamespace,
			breakDB:   "DROP TABLE organizations",
			wantErr:   errorcodes.DatabaseOperationFailed,
			retryable: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newCredimiTestApp(t)
			runner := createTestRunner(t, app, tc.owner, "Runner One", "")
			deviceID := createTestDevice(t, app, tc.owner, runner.Id, "Device One")
			if tc.breakDB != "" {
				_, err := app.DB().NewQuery(tc.breakDB).Execute()
				require.NoError(t, err)
			}

			_, err := executeActivity(
				t,
				NewValidateDeviceAccessActivity(app),
				ValidateDeviceAccessInput{
					OwnerNamespace: tc.namespace,
					DeviceIDs:      []string{deviceID},
				},
			)
			if tc.wantErr != "" {
				requireActivityError(t, err, tc.wantErr, !tc.retryable)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateDeviceAccessActivityUnknownDevice(t *testing.T) {
	app := newCredimiTestApp(t)
	_, err := executeActivity(t, NewValidateDeviceAccessActivity(app), ValidateDeviceAccessInput{
		OwnerNamespace: testOrgANamespace,
		DeviceIDs:      []string{testOrgANamespace + "/missing/device"},
	})
	requireActivityError(t, err, errorcodes.RecordNotFound, true)
}
