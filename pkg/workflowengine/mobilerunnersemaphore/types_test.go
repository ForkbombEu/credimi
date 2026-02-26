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

func TestWorkflowStateJSONRoundTrip(t *testing.T) {
	requestedAt := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
	startedAt := requestedAt.Add(5 * time.Minute)
	doneAt := requestedAt.Add(10 * time.Minute)

	state := MobileRunnerSemaphoreWorkflowState{
		Capacity:    2,
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
