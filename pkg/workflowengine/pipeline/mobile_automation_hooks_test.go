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
