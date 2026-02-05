// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunnersemaphore

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWorkflowID(t *testing.T) {
	workflowID := WorkflowID("runner-1")

	require.Equal(t, "mobile-runner-semaphore/runner-1", workflowID)
}

func TestPermitLeaseID(t *testing.T) {
	workflowID := "mobile-runner-semaphore/runner-1"
	leaseID := PermitLeaseID(workflowID, "run-1", "runner-1")

	require.Equal(t, "mobile-runner-semaphore/runner-1/run-1/runner-1", leaseID)

	otherLeaseID := PermitLeaseID(workflowID, "run-2", "runner-1")
	require.NotEqual(t, leaseID, otherLeaseID)
}

func TestWorkflowStateJSONRoundTrip(t *testing.T) {
	requestedAt := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
	grantedAt := requestedAt.Add(2 * time.Minute)
	startedAt := requestedAt.Add(5 * time.Minute)
	doneAt := requestedAt.Add(10 * time.Minute)

	state := MobileRunnerSemaphoreWorkflowState{
		Capacity: 2,
		Holders: map[string]MobileRunnerSemaphoreHolder{
			"lease-1": {
				LeaseID:        "lease-1",
				RequestID:      "req-1",
				OwnerNamespace: "tenant-1",
				OwnerWorkflowID: "wf-1",
				OwnerRunID:     "run-1",
				GrantedAt:      grantedAt,
				QueueWaitMs:    1200,
			},
		},
		Queue: []string{"req-2"},
		Requests: map[string]MobileRunnerSemaphoreRequestState{
			"req-2": {
				Request: MobileRunnerSemaphoreAcquireRequest{
					RequestID:      "req-2",
					LeaseID:        "lease-2",
					OwnerNamespace: "tenant-2",
					WaitTimeout:    30 * time.Second,
				},
				Status:      MobileRunnerSemaphoreRequestQueued,
				RequestedAt: requestedAt,
			},
		},
		LastGrantAt: &grantedAt,
		UpdateCount: 3,
		RunQueue:    []string{"ticket-1"},
		RunTickets: map[string]MobileRunnerSemaphoreRunTicketState{
			"ticket-1": {
				Request: MobileRunnerSemaphoreEnqueueRunRequest{
					TicketID:           "ticket-1",
					OwnerNamespace:     "tenant-1",
					EnqueuedAt:         requestedAt,
					RunnerID:           "runner-1",
					RequiredRunnerIDs:  []string{"runner-1"},
					LeaderRunnerID:     "runner-1",
					PipelineIdentifier: "pipeline-1",
					YAML:               "steps: []",
					PipelineConfig:     map[string]any{"key": "value"},
					Memo:               map[string]any{"trace_id": "trace-1"},
				},
				Status:            MobileRunnerSemaphoreRunRunning,
				WorkflowID:        "workflow-1",
				RunID:             "run-1",
				WorkflowNamespace: "tenant-1",
				GrantedRunnerIDs:  map[string]bool{"runner-1": true},
				StartedAt:         &startedAt,
				DoneAt:            &doneAt,
			},
		},
	}

	data, err := json.Marshal(state)
	require.NoError(t, err)

	var decoded MobileRunnerSemaphoreWorkflowState
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	require.Equal(t, state, decoded)
}

func TestWorkflowStateJSONLegacyCompatibility(t *testing.T) {
	legacyJSON := []byte(`{
		"capacity": 1,
		"holders": {
			"lease-1": {
				"lease_id": "lease-1",
				"request_id": "req-1",
				"granted_at": "2026-02-03T12:00:00Z"
			}
		},
		"queue": ["req-1"],
		"requests": {
			"req-1": {
				"request": {
					"request_id": "req-1",
					"lease_id": "lease-1"
				},
				"status": "queued",
				"requested_at": "2026-02-03T12:00:00Z"
			}
		}
	}`)

	var decoded MobileRunnerSemaphoreWorkflowState
	err := json.Unmarshal(legacyJSON, &decoded)
	require.NoError(t, err)

	require.Equal(t, 1, decoded.Capacity)
	require.Nil(t, decoded.RunQueue)
	require.Nil(t, decoded.RunTickets)
	require.Len(t, decoded.Queue, 1)
	require.Len(t, decoded.Requests, 1)
}
