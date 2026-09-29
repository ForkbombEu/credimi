// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeviceSupportsLiveView(t *testing.T) {
	t.Parallel()
	require.True(t, DeviceSupportsLiveView("android_emulator"))
	require.True(t, DeviceSupportsLiveView("android_phone"))
	require.True(t, DeviceSupportsLiveView(""))
	require.False(t, DeviceSupportsLiveView("ios_simulator"))
	require.False(t, DeviceSupportsLiveView(" ios_simulator "))
}

func TestDeviceDisplayName(t *testing.T) {
	t.Parallel()
	require.Equal(t, "pixel", deviceDisplayName("org/runner/pixel", map[string]any{"name": "pixel"}))
	require.Equal(t, "friendly", deviceDisplayName("org/runner/pixel", map[string]any{"name": "friendly"}))
	require.Equal(t, "pixel", deviceDisplayName("org/runner/pixel", nil))
	require.Equal(t, "solo", deviceDisplayName("solo", nil))
}

func TestResolveAndBuildPipelineExecutionDevicesEmpty(t *testing.T) {
	t.Parallel()
	require.Nil(t, ResolveAndBuildPipelineExecutionDevices(nil, nil, string(WorkflowStatusRunning), nil))
	require.Nil(t, ResolveAndBuildPipelineExecutionDevices(nil, []string{}, string(WorkflowStatusRunning), nil))
}
