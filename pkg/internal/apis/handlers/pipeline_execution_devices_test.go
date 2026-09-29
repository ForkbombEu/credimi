// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
)

func TestPipelineExecutionDeviceLiveView(t *testing.T) {
	t.Parallel()
	live := map[string]any{"name": "Pixel", "live_stream": true}
	notLive := map[string]any{"name": "Pixel", "live_stream": false}

	require.True(t, pipelineExecutionDevice("org/runner/pixel", live, true).LiveView)
	require.False(t, pipelineExecutionDevice("org/runner/pixel", live, false).LiveView,
		"a finished execution has nothing to watch")
	require.False(t, pipelineExecutionDevice("org/runner/pixel", notLive, true).LiveView,
		"the runner did not report live stream for this device")
	require.False(
		t,
		pipelineExecutionDevice("org/runner/pixel", map[string]any{"name": "Pixel"}, true).LiveView,
		"records registered by runners without live stream support",
	)
	require.False(t, pipelineExecutionDevice("org/runner/pixel", nil, true).LiveView,
		"an unresolved device record")
}

func TestDeviceDisplayName(t *testing.T) {
	t.Parallel()
	require.Equal(
		t,
		"pixel",
		deviceDisplayName("org/runner/pixel", map[string]any{"name": "pixel"}),
	)
	require.Equal(
		t,
		"friendly",
		deviceDisplayName("org/runner/pixel", map[string]any{"name": "friendly"}),
	)
	require.Equal(t, "pixel", deviceDisplayName("org/runner/pixel", nil))
	require.Equal(t, "solo", deviceDisplayName("solo", nil))
}

func TestResolveAndBuildPipelineExecutionDevicesEmpty(t *testing.T) {
	t.Parallel()
	require.Nil(
		t,
		ResolveAndBuildPipelineExecutionDevices(nil, nil, string(WorkflowStatusRunning), nil),
	)
	require.Nil(
		t,
		ResolveAndBuildPipelineExecutionDevices(
			nil,
			[]string{},
			string(WorkflowStatusRunning),
			nil,
		),
	)
}

func TestDeviceIDsFromSearchAttributeField(t *testing.T) {
	t.Parallel()
	require.Nil(t, deviceIDsFromSearchAttributeField(nil))

	payload, err := temporalcrypto.DataConverter().ToPayload([]string{"org/runner/a", "org/runner/b"})
	require.NoError(t, err)
	attrs := &commonpb.SearchAttributes{
		IndexedFields: map[string]*commonpb.Payload{
			workflowengine.DeviceIdentifiersSearchAttribute: payload,
		},
	}
	require.Equal(t, []string{"org/runner/a", "org/runner/b"}, deviceIDsFromSearchAttributeField(attrs))
}
