// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package runqueue

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/stretchr/testify/require"
)

func TestAggregateDeviceStatuses_Empty(t *testing.T) {
	got := AggregateDeviceStatuses(nil)

	require.Equal(t, mobiledevicesemaphore.MobileDeviceSemaphoreRunNotFound, got.Status)
	require.Equal(t, 0, got.Position)
	require.Equal(t, 0, got.LineLen)
	require.Equal(t, "", got.WorkflowID)
	require.Equal(t, "", got.RunID)
	require.Equal(t, "", got.WorkflowNamespace)
	require.Equal(t, "", got.ErrorMessage)
}

func TestAggregateDeviceStatuses_PriorityAndMetadata(t *testing.T) {
	statuses := []DeviceStatus{
		{
			DeviceID:          "runner-a",
			Status:            mobiledevicesemaphore.MobileDeviceSemaphoreRunQueued,
			Position:          1,
			LineLen:           2,
			WorkflowID:        "wf-queued",
			RunID:             "run-queued",
			WorkflowNamespace: "org-a",
		},
		{
			DeviceID:          "runner-b",
			Status:            mobiledevicesemaphore.MobileDeviceSemaphoreRunRunning,
			Position:          3,
			LineLen:           5,
			WorkflowID:        "wf-running",
			RunID:             "run-running",
			WorkflowNamespace: "org-b",
		},
		{
			DeviceID:     "runner-c",
			Status:       mobiledevicesemaphore.MobileDeviceSemaphoreRunFailed,
			Position:     2,
			LineLen:      4,
			ErrorMessage: "runner failed",
		},
	}

	got := AggregateDeviceStatuses(statuses)

	require.Equal(t, mobiledevicesemaphore.MobileDeviceSemaphoreRunFailed, got.Status)
	require.Equal(t, 3, got.Position)
	require.Equal(t, 5, got.LineLen)
	require.Equal(t, "wf-running", got.WorkflowID)
	require.Equal(t, "run-running", got.RunID)
	require.Equal(t, "org-b", got.WorkflowNamespace)
	require.Equal(t, "runner failed", got.ErrorMessage)
}

func TestAggregateDeviceStatuses_UsesFirstRunningAndFirstFailureMessage(t *testing.T) {
	statuses := []DeviceStatus{
		{
			DeviceID:          "runner-a",
			Status:            mobiledevicesemaphore.MobileDeviceSemaphoreRunRunning,
			WorkflowID:        "wf-first",
			RunID:             "run-first",
			WorkflowNamespace: "ns-first",
		},
		{
			DeviceID:          "runner-b",
			Status:            mobiledevicesemaphore.MobileDeviceSemaphoreRunRunning,
			WorkflowID:        "wf-second",
			RunID:             "run-second",
			WorkflowNamespace: "ns-second",
		},
		{
			DeviceID:     "runner-c",
			Status:       mobiledevicesemaphore.MobileDeviceSemaphoreRunFailed,
			ErrorMessage: "first error",
		},
		{
			DeviceID:     "runner-d",
			Status:       mobiledevicesemaphore.MobileDeviceSemaphoreRunFailed,
			ErrorMessage: "second error",
		},
	}

	got := AggregateDeviceStatuses(statuses)

	require.Equal(t, "wf-first", got.WorkflowID)
	require.Equal(t, "run-first", got.RunID)
	require.Equal(t, "ns-first", got.WorkflowNamespace)
	require.Equal(t, "first error", got.ErrorMessage)
}

func TestRunStatusPriority(t *testing.T) {
	tests := []struct {
		status mobiledevicesemaphore.MobileDeviceSemaphoreRunStatus
		want   int
	}{
		{status: mobiledevicesemaphore.MobileDeviceSemaphoreRunFailed, want: 4},
		{status: mobiledevicesemaphore.MobileDeviceSemaphoreRunCanceled, want: 4},
		{status: mobiledevicesemaphore.MobileDeviceSemaphoreRunRunning, want: 3},
		{status: mobiledevicesemaphore.MobileDeviceSemaphoreRunStarting, want: 2},
		{status: mobiledevicesemaphore.MobileDeviceSemaphoreRunQueued, want: 1},
		{status: mobiledevicesemaphore.MobileDeviceSemaphoreRunNotFound, want: 0},
		{status: mobiledevicesemaphore.MobileDeviceSemaphoreRunStatus("unknown"), want: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.status), func(t *testing.T) {
			require.Equal(t, tt.want, runStatusPriority(tt.status))
		})
	}
}
