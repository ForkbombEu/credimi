// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateEmulatorStatus(t *testing.T) {
	startedEmulators := map[string]any{}

	entry := updateEmulatorStatus(startedEmulators, "v1", map[string]any{
		"serial":      "emulator-5554",
		"boot_status": bootStatusStarted,
	})

	require.Equal(t, "v1", entry["version_id"])
	require.Equal(t, "emulator-5554", entry["serial"])
	require.Equal(t, bootStatusStarted, entry["boot_status"])

	updateEmulatorStatus(startedEmulators, "v1", map[string]any{
		"boot_status": bootStatusRecording,
		"recording":   true,
	})

	entry, ok := startedEmulators["v1"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, bootStatusRecording, entry["boot_status"])
	require.Equal(t, true, entry["recording"])
	require.Equal(t, "emulator-5554", entry["serial"])
}

func TestCountActiveEmulators(t *testing.T) {
	startedEmulators := map[string]any{
		"v1": map[string]any{
			"boot_status": bootStatusRecording,
		},
		"v2": map[string]any{
			"boot_status": bootStatusStopped,
		},
	}

	require.Equal(t, 1, countActiveEmulators(startedEmulators))
}

func TestGetEmulatorMetricsInit(t *testing.T) {
	runData := map[string]any{}
	metrics := getEmulatorMetrics(&runData)

	require.Equal(t, 0, metrics["active_emulators"])
	require.Equal(t, 0, metrics["pending_starts"])
	require.Equal(t, 0, metrics["failed_starts"])
	require.Equal(t, 0, metrics["cleanup_errors"])

	stored, ok := runData["emulator_metrics"].(map[string]int)
	require.True(t, ok)
	require.Equal(t, metrics, stored)
}

func TestCountActiveEmulatorsWithEmptyBootStatus(t *testing.T) {
	startedEmulators := map[string]any{
		"v1": map[string]any{
			"boot_status": "",
		},
		"v2": map[string]any{
			"serial": "emulator-5556",
			// boot_status missing entirely
		},
		"v3": map[string]any{
			"boot_status": bootStatusRecording,
		},
	}

	// Empty or missing boot_status should be counted as active
	require.Equal(t, 3, countActiveEmulators(startedEmulators))
}

func TestCountActiveEmulatorsWithMixedStatuses(t *testing.T) {
	startedEmulators := map[string]any{
		"v1": map[string]any{
			"boot_status": bootStatusStarting,
		},
		"v2": map[string]any{
			"boot_status": bootStatusStarted,
		},
		"v3": map[string]any{
			"boot_status": bootStatusBooted,
		},
		"v4": map[string]any{
			"boot_status": bootStatusRecording,
		},
		"v5": map[string]any{
			"boot_status": bootStatusStopped,
		},
	}

	// Only stopped emulators should not be counted as active
	require.Equal(t, 4, countActiveEmulators(startedEmulators))
}

func TestUpdateEmulatorStatusWithNilMap(t *testing.T) {
	result := updateEmulatorStatus(nil, "v1", map[string]any{
		"serial":      "emulator-5554",
		"boot_status": bootStatusStarted,
	})

	require.Nil(t, result)
}

func TestGetEmulatorMetricsReturnsConsistentInstance(t *testing.T) {
	runData := map[string]any{}

	// First call initializes metrics
	metrics1 := getEmulatorMetrics(&runData)
	require.Equal(t, 0, metrics1["active_emulators"])

	// Modify the metrics
	metrics1["active_emulators"] = 5
	metrics1["pending_starts"] = 2

	// Second call should return the same instance
	metrics2 := getEmulatorMetrics(&runData)
	require.Equal(t, 5, metrics2["active_emulators"])
	require.Equal(t, 2, metrics2["pending_starts"])

	// Verify they're the same instance by checking they share the same underlying map
	require.Equal(t, metrics1, metrics2)
}

func TestUpsertEmulatorSearchAttributesWithEmptyParameters(t *testing.T) {
	// Note: This test verifies the function doesn't panic with various empty parameter combinations
	// We can't easily verify the actual Temporal workflow behavior without mocking,
	// but we ensure the function handles edge cases gracefully

	// Test with all empty parameters - should return early
	upsertEmulatorSearchAttributes(nil, "", "", "")

	// Test with only versionID
	upsertEmulatorSearchAttributes(nil, "v1", "", "")

	// Test with only serial
	upsertEmulatorSearchAttributes(nil, "", "emulator-5554", "")

	// Test with only bootStatus
	upsertEmulatorSearchAttributes(nil, "", "", bootStatusStarted)

	// Test with versionID and serial
	upsertEmulatorSearchAttributes(nil, "v1", "emulator-5554", "")

	// Test with all parameters
	upsertEmulatorSearchAttributes(nil, "v1", "emulator-5554", bootStatusStarted)

	// If we reach here without panicking, the test passes
}
