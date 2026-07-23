// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobiledevicesemaphore

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWorkflowID(t *testing.T) {
	workflowID := WorkflowID("runner-1")

	require.Equal(t, "mobile-device-semaphore/runner-1", workflowID)
}

func TestWorkflowIDTrimsLeadingSlash(t *testing.T) {
	workflowID := WorkflowID("/tenant-a/runner-1")

	require.Equal(t, "mobile-device-semaphore/tenant-a/runner-1", workflowID)
}

func TestWorkflowStateJSONRoundTrip(t *testing.T) {
	requestedAt := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
	startedAt := requestedAt.Add(5 * time.Minute)
	doneAt := requestedAt.Add(10 * time.Minute)

	state := MobileDeviceSemaphoreWorkflowState{
		Capacity:    2,
		UpdateCount: 3,
		RunQueue:    []string{"ticket-1"},
		RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
			"ticket-1": {
				Request: MobileDeviceSemaphoreEnqueueRunRequest{
					TicketID:           "ticket-1",
					OwnerNamespace:     "tenant-1",
					EnqueuedAt:         requestedAt,
					DeviceID:           "runner-1",
					RequiredDeviceIDs:  []string{"runner-1"},
					LeaderDeviceID:     "runner-1",
					PipelineIdentifier: "pipeline-1",
					YAML:               "steps: []",
					PipelineConfig:     map[string]any{"key": "value"},
					Memo:               map[string]any{"trace_id": "trace-1"},
				},
				Status:            MobileDeviceSemaphoreRunRunning,
				WorkflowID:        "workflow-1",
				RunID:             "run-1",
				WorkflowNamespace: "tenant-1",
				GrantedDeviceIDs:  map[string]bool{"runner-1": true},
				StartedAt:         &startedAt,
				DoneAt:            &doneAt,
			},
		},
	}

	data, err := json.Marshal(state)
	require.NoError(t, err)

	var decoded MobileDeviceSemaphoreWorkflowState
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.Equal(t, state, decoded)
}
