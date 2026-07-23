// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data"

func TestParsePipelineDeviceInfo(t *testing.T) {
	t.Run("empty yaml returns zero value", func(t *testing.T) {
		got, err := ParsePipelineDeviceInfo("   ")
		require.NoError(t, err)
		require.False(t, got.NeedsGlobalDevice)
		require.Empty(t, got.DeviceIDs)
	})

	t.Run("invalid yaml returns error", func(t *testing.T) {
		_, err := ParsePipelineDeviceInfo("[")
		require.Error(t, err)
	})

	t.Run("collects and deduplicates device ids from steps and branches", func(t *testing.T) {
		yamlStr := `
name: test
steps:
  - id: step-1
    use: mobile-automation
    with:
      payload:
        device_id: tenant-a/runner-b/device-b
  - id: step-2
    use: mobile-automation
    with:
      payload:
        device_id: tenant-a/runner-a/device-a
  - id: step-3
    use: echo
    with:
      message: ok
    on_error:
      - id: err-step
        use: mobile-automation
        with:
          payload:
            device_id: tenant-a/runner-c/device-c
    on_success:
      - id: success-step
        use: mobile-automation
        with:
          payload:
            device_id: tenant-a/runner-a/device-a
  - id: step-4
    use: mobile-automation
    with:
      action_id: missing-runner-id
`

		got, err := ParsePipelineDeviceInfo(yamlStr)
		require.NoError(t, err)
		require.True(t, got.NeedsGlobalDevice)
		require.Equal(t, []string{"tenant-a/runner-a/device-a", "tenant-a/runner-b/device-b", "tenant-a/runner-c/device-c"}, got.DeviceIDs)
	})

	t.Run("normalizes leading slash device ids", func(t *testing.T) {
		yamlStr := `
name: test
steps:
  - id: step-1
    use: mobile-automation
    with:
      payload:
        device_id: /tenant-a/runner-a/device-a
`

		got, err := ParsePipelineDeviceInfo(yamlStr)
		require.NoError(t, err)
		require.Equal(t, []string{"tenant-a/runner-a/device-a"}, got.DeviceIDs)
	})
}

func TestDeviceIDsWithGlobal(t *testing.T) {
	t.Run("adds global device if needed and missing", func(t *testing.T) {
		info := PipelineDeviceInfo{
			DeviceIDs:         []string{"device-b"},
			NeedsGlobalDevice: true,
		}
		got := DeviceIDsWithGlobal(info, " device-a ")
		require.Equal(t, []string{"device-a", "device-b"}, got)
	})

	t.Run("normalizes global runner leading slash", func(t *testing.T) {
		info := PipelineDeviceInfo{
			DeviceIDs:         []string{"tenant-a/runner-b/device-b"},
			NeedsGlobalDevice: true,
		}
		got := DeviceIDsWithGlobal(info, " /tenant-a/runner-a/device-a ")
		require.Equal(t, []string{"tenant-a/runner-a/device-a", "tenant-a/runner-b/device-b"}, got)
	})

	t.Run("does not duplicate global runner", func(t *testing.T) {
		info := PipelineDeviceInfo{
			DeviceIDs:         []string{"device-a", "device-b"},
			NeedsGlobalDevice: true,
		}
		got := DeviceIDsWithGlobal(info, "device-a")
		require.Equal(t, []string{"device-a", "device-b"}, got)
	})

	t.Run("ignores global runner when not needed", func(t *testing.T) {
		info := PipelineDeviceInfo{
			DeviceIDs:         []string{"device-a"},
			NeedsGlobalDevice: false,
		}
		got := DeviceIDsWithGlobal(info, "device-b")
		require.Equal(t, []string{"device-a"}, got)
	})
}

func TestGlobalDeviceIDFromConfig(t *testing.T) {
	require.Equal(t, "", GlobalDeviceIDFromConfig(nil))
	require.Equal(t, "", GlobalDeviceIDFromConfig(map[string]any{"global_device_id": 12}))
	require.Equal(
		t,
		"runner-a",
		GlobalDeviceIDFromConfig(map[string]any{"global_device_id": " runner-a "}),
	)
	require.Equal(
		t,
		"tenant-a/runner-a",
		GlobalDeviceIDFromConfig(map[string]any{"global_device_id": " /tenant-a/runner-a "}),
	)
}

func TestResolveDeviceRecord(t *testing.T) {
	t.Run("empty runner id returns nil", func(t *testing.T) {
		got := ResolveDeviceRecord(nil, " ", nil)
		require.Nil(t, got)
	})

	t.Run("returns cached record", func(t *testing.T) {
		cache := map[string]map[string]any{
			"tenant/runner-a": {"id": "cached-id"},
		}
		got := ResolveDeviceRecord(nil, "/tenant/runner-a", cache)
		require.Equal(t, map[string]any{"id": "cached-id"}, got)
	})

	t.Run("not found runner is cached as nil", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		defer app.Cleanup()

		cache := map[string]map[string]any{}
		runnerID := "missing-org/missing-runner"

		got := ResolveDeviceRecord(app, runnerID, cache)
		require.Nil(t, got)
		_, ok := cache[runnerID]
		require.True(t, ok)
		require.Nil(t, cache[runnerID])
	})

	t.Run("rejects a runner identifier", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		defer app.Cleanup()

		org, err := app.FindFirstRecordByFilter("organizations", "1=1")
		require.NoError(t, err)
		orgCanon := org.GetString("canonified_name")
		require.NotEmpty(t, orgCanon)

		runnersCollection, err := app.FindCollectionByNameOrId("mobile_runners")
		require.NoError(t, err)
		newRunner := core.NewRecord(runnersCollection)
		newRunner.Set("owner", org.Id)
		newRunner.Set("name", "test-runner")
		newRunner.Set("canonified_name", "test-runner")
		newRunner.Set("ip", "127.0.0.1")
		newRunner.Set("type", "android_emulator")
		require.NoError(t, app.Save(newRunner))

		cache := map[string]map[string]any{}
		runnerID := "/" + orgCanon + "/test-runner"
		got := ResolveDeviceRecord(app, runnerID, cache)

		require.Nil(t, got)
		require.Contains(t, cache, orgCanon+"/test-runner")
		require.Nil(t, cache[orgCanon+"/test-runner"])
	})
}

func TestResolveDeviceRecords(t *testing.T) {
	t.Run("empty input returns empty slice", func(t *testing.T) {
		got := ResolveDeviceRecords(nil, nil, nil)
		require.Empty(t, got)
	})

	t.Run("rejects non-device records", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		defer app.Cleanup()

		org, err := app.FindFirstRecordByFilter("organizations", "1=1")
		require.NoError(t, err)
		orgCanon := org.GetString("canonified_name")
		require.NotEmpty(t, orgCanon)

		runnersCollection, err := app.FindCollectionByNameOrId("mobile_runners")
		require.NoError(t, err)
		newRunner := core.NewRecord(runnersCollection)
		newRunner.Set("owner", org.Id)
		newRunner.Set("name", "queue-runner")
		newRunner.Set("canonified_name", "queue-runner")
		newRunner.Set("ip", "127.0.0.1")
		newRunner.Set("type", "android_emulator")
		require.NoError(t, app.Save(newRunner))

		cache := map[string]map[string]any{}
		resolvableDeviceID := orgCanon + "/queue-runner"
		got := ResolveDeviceRecords(
			app,
			[]string{
				resolvableDeviceID,
				"missing-org/missing-runner",
			},
			cache,
		)

		require.Empty(t, got)
	})
}
