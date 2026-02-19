// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package runqueue

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"github.com/stretchr/testify/require"
)

func TestAggregateRunnerStatuses_Empty(t *testing.T) {
	got := AggregateRunnerStatuses(nil)

	require.Equal(t, mobilerunnersemaphore.MobileRunnerSemaphoreRunNotFound, got.Status)
	require.Equal(t, 0, got.Position)
	require.Equal(t, 0, got.LineLen)
	require.Equal(t, "", got.WorkflowID)
	require.Equal(t, "", got.RunID)
	require.Equal(t, "", got.WorkflowNamespace)
	require.Equal(t, "", got.ErrorMessage)
}

func TestAggregateRunnerStatuses_PriorityAndMetadata(t *testing.T) {
	statuses := []RunnerStatus{
		{
			RunnerID:          "runner-a",
			Status:            mobilerunnersemaphore.MobileRunnerSemaphoreRunQueued,
			Position:          1,
			LineLen:           2,
			WorkflowID:        "wf-queued",
			RunID:             "run-queued",
			WorkflowNamespace: "org-a",
		},
		{
			RunnerID:          "runner-b",
			Status:            mobilerunnersemaphore.MobileRunnerSemaphoreRunRunning,
			Position:          3,
			LineLen:           5,
			WorkflowID:        "wf-running",
			RunID:             "run-running",
			WorkflowNamespace: "org-b",
		},
		{
			RunnerID:     "runner-c",
			Status:       mobilerunnersemaphore.MobileRunnerSemaphoreRunFailed,
			Position:     2,
			LineLen:      4,
			ErrorMessage: "runner failed",
		},
	}

	got := AggregateRunnerStatuses(statuses)

	require.Equal(t, mobilerunnersemaphore.MobileRunnerSemaphoreRunFailed, got.Status)
	require.Equal(t, 3, got.Position)
	require.Equal(t, 5, got.LineLen)
	require.Equal(t, "wf-running", got.WorkflowID)
	require.Equal(t, "run-running", got.RunID)
	require.Equal(t, "org-b", got.WorkflowNamespace)
	require.Equal(t, "runner failed", got.ErrorMessage)
}

func TestAggregateRunnerStatuses_UsesFirstRunningAndFirstFailureMessage(t *testing.T) {
	statuses := []RunnerStatus{
		{
			RunnerID:          "runner-a",
			Status:            mobilerunnersemaphore.MobileRunnerSemaphoreRunRunning,
			WorkflowID:        "wf-first",
			RunID:             "run-first",
			WorkflowNamespace: "ns-first",
		},
		{
			RunnerID:          "runner-b",
			Status:            mobilerunnersemaphore.MobileRunnerSemaphoreRunRunning,
			WorkflowID:        "wf-second",
			RunID:             "run-second",
			WorkflowNamespace: "ns-second",
		},
		{
			RunnerID:     "runner-c",
			Status:       mobilerunnersemaphore.MobileRunnerSemaphoreRunFailed,
			ErrorMessage: "first error",
		},
		{
			RunnerID:     "runner-d",
			Status:       mobilerunnersemaphore.MobileRunnerSemaphoreRunFailed,
			ErrorMessage: "second error",
		},
	}

	got := AggregateRunnerStatuses(statuses)

	require.Equal(t, "wf-first", got.WorkflowID)
	require.Equal(t, "run-first", got.RunID)
	require.Equal(t, "ns-first", got.WorkflowNamespace)
	require.Equal(t, "first error", got.ErrorMessage)
}

func TestRunStatusPriority(t *testing.T) {
	tests := []struct {
		status mobilerunnersemaphore.MobileRunnerSemaphoreRunStatus
		want   int
	}{
		{status: mobilerunnersemaphore.MobileRunnerSemaphoreRunFailed, want: 4},
		{status: mobilerunnersemaphore.MobileRunnerSemaphoreRunCanceled, want: 4},
		{status: mobilerunnersemaphore.MobileRunnerSemaphoreRunRunning, want: 3},
		{status: mobilerunnersemaphore.MobileRunnerSemaphoreRunStarting, want: 2},
		{status: mobilerunnersemaphore.MobileRunnerSemaphoreRunQueued, want: 1},
		{status: mobilerunnersemaphore.MobileRunnerSemaphoreRunNotFound, want: 0},
		{status: mobilerunnersemaphore.MobileRunnerSemaphoreRunStatus("unknown"), want: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.status), func(t *testing.T) {
			require.Equal(t, tt.want, runStatusPriority(tt.status))
		})
	}
}
