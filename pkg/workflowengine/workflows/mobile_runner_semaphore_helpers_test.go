// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func TestDecodeStartQueuedPipelineOutput(t *testing.T) {
	output := activities.StartQueuedPipelineActivityOutput{
		WorkflowID:            "wf-1",
		RunID:                 "run-1",
		WorkflowNamespace:     "ns-1",
		PipelineResultCreated: true,
	}

	decoded, err := decodeStartQueuedPipelineOutput(output)
	require.NoError(t, err)
	require.Equal(t, output, decoded)

	decoded, err = decodeStartQueuedPipelineOutput(map[string]any{
		"workflow_id":             "wf-2",
		"run_id":                  "run-2",
		"workflow_namespace":      "ns-2",
		"pipeline_result_created": false,
	})
	require.NoError(t, err)
	require.Equal(t, "wf-2", decoded.WorkflowID)
	require.Equal(t, "run-2", decoded.RunID)
	require.Equal(t, "ns-2", decoded.WorkflowNamespace)
	require.False(t, decoded.PipelineResultCreated)

	_, err = decodeStartQueuedPipelineOutput(123)
	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, MobileRunnerSemaphoreErrInvalidRequest, appErr.Type())
}

func TestDecodeStartQueuedPipelineOutputMapError(t *testing.T) {
	_, err := decodeStartQueuedPipelineOutput(map[string]any{
		"workflow_id": func() {},
	})
	require.Error(t, err)
}

func TestDecodeCheckWorkflowClosedOutput(t *testing.T) {
	output := activities.CheckWorkflowClosedActivityOutput{Closed: true, Status: "completed"}

	decoded, err := decodeCheckWorkflowClosedOutput(output)
	require.NoError(t, err)
	require.Equal(t, output, decoded)

	decoded, err = decodeCheckWorkflowClosedOutput(map[string]any{
		"closed": true,
		"status": "running",
	})
	require.NoError(t, err)
	require.True(t, decoded.Closed)
	require.Equal(t, "running", decoded.Status)

	_, err = decodeCheckWorkflowClosedOutput(5)
	require.Error(t, err)
}

func TestDecodeRunStatusView(t *testing.T) {
	view := MobileRunnerSemaphoreRunStatusView{
		TicketID: "ticket-1",
		Status:   mobileRunnerSemaphoreRunQueued,
		Position: 1,
		LineLen:  2,
	}

	decoded, err := decodeRunStatusView(view)
	require.NoError(t, err)
	require.Equal(t, view, decoded)

	decoded, err = decodeRunStatusView(map[string]any{
		"ticket_id": "ticket-2",
		"status":    string(mobileRunnerSemaphoreRunRunning),
		"position":  0,
		"line_len":  1,
	})
	require.NoError(t, err)
	require.Equal(t, "ticket-2", decoded.TicketID)
	require.Equal(t, mobileRunnerSemaphoreRunRunning, decoded.Status)

	_, err = decodeRunStatusView(3)
	require.Error(t, err)
}

func TestDecodeRunStatusViewMapErrors(t *testing.T) {
	_, err := decodeRunStatusView(map[string]any{
		"ticket_id": func() {},
	})
	require.Error(t, err)

	_, err = decodeRunStatusView(map[string]any{
		"ticket_id": "ticket-1",
		"position":  "bad",
	})
	require.Error(t, err)
}

func TestDecodeCheckWorkflowClosedOutputMapErrors(t *testing.T) {
	_, err := decodeCheckWorkflowClosedOutputMap(map[string]any{
		"closed": func() {},
	})
	require.Error(t, err)

	_, err = decodeCheckWorkflowClosedOutputMap(map[string]any{
		"closed": "nope",
	})
	require.Error(t, err)
}

func TestBuildRunStatusViewCopiesSlice(t *testing.T) {
	runtime := &mobileRunnerSemaphoreRuntime{}
	state := MobileRunnerSemaphoreRunTicketState{
		Status:            mobileRunnerSemaphoreRunRunning,
		WorkflowID:        "wf-1",
		RunID:             "run-1",
		WorkflowNamespace: "ns-1",
		Request: MobileRunnerSemaphoreEnqueueRunRequest{
			LeaderRunnerID:    "leader-1",
			RequiredRunnerIDs: []string{"r1", "r2"},
		},
		ErrorMessage: "oops",
	}

	view := runtime.buildRunStatusView("ticket-1", state)
	require.Equal(t, "ticket-1", view.TicketID)
	require.Equal(t, mobileRunnerSemaphoreRunRunning, view.Status)
	require.Equal(t, "leader-1", view.LeaderRunnerID)
	require.Equal(t, []string{"r1", "r2"}, view.RequiredRunnerIDs)
	require.Equal(t, "wf-1", view.WorkflowID)
	require.Equal(t, "run-1", view.RunID)
	require.Equal(t, "ns-1", view.WorkflowNamespace)
	require.Equal(t, "oops", view.ErrorMessage)

	state.Request.RequiredRunnerIDs[0] = "changed"
	require.Equal(t, []string{"r1", "r2"}, view.RequiredRunnerIDs)
}

func TestRemoveFromQueue(t *testing.T) {
	queue := []string{"a", "b", "c"}
	updated := removeFromQueue(queue, "b")
	require.Equal(t, []string{"a", "c"}, updated)
	require.Equal(t, queue, removeFromQueue(queue, "missing"))
}

func TestInsertRunQueueSorts(t *testing.T) {
	now := time.Now()
	older := now.Add(-time.Minute)
	tickets := map[string]MobileRunnerSemaphoreRunTicketState{
		"old": {
			Request: MobileRunnerSemaphoreEnqueueRunRequest{TicketID: "old", EnqueuedAt: older},
		},
		"new": {Request: MobileRunnerSemaphoreEnqueueRunRequest{TicketID: "new", EnqueuedAt: now}},
	}
	queue := []string{"new"}
	queue = insertRunQueue(queue, "old", tickets)
	require.Equal(t, []string{"old", "new"}, queue)
}

func TestNextQueuedRunTicketSkipsNonQueued(t *testing.T) {
	runtime := &mobileRunnerSemaphoreRuntime{
		runQueue: []string{"t1", "t2"},
		runTickets: map[string]MobileRunnerSemaphoreRunTicketState{
			"t1": {Status: mobileRunnerSemaphoreRunRunning},
			"t2": {Status: mobileRunnerSemaphoreRunQueued},
		},
	}

	id, _, ok := runtime.nextQueuedRunTicket()
	require.True(t, ok)
	require.Equal(t, "t2", id)
	require.Len(t, runtime.runQueue, 1)
}

func TestAllGrantsReceived(t *testing.T) {
	runtime := &mobileRunnerSemaphoreRuntime{}
	state := MobileRunnerSemaphoreRunTicketState{
		Request: MobileRunnerSemaphoreEnqueueRunRequest{
			RequiredRunnerIDs: []string{"r1", "r2"},
		},
		GrantedRunnerIDs: map[string]bool{"r1": true},
	}
	require.False(t, runtime.allGrantsReceived(state))

	state.GrantedRunnerIDs["r2"] = true
	require.True(t, runtime.allGrantsReceived(state))
}

func TestSortedRunTicketIDs(t *testing.T) {
	now := time.Now()
	runtime := &mobileRunnerSemaphoreRuntime{
		runTickets: map[string]MobileRunnerSemaphoreRunTicketState{
			"b": {Request: MobileRunnerSemaphoreEnqueueRunRequest{TicketID: "b", EnqueuedAt: now}},
			"a": {
				Request: MobileRunnerSemaphoreEnqueueRunRequest{
					TicketID:   "a",
					EnqueuedAt: now.Add(-time.Minute),
				},
			},
		},
	}
	ids := runtime.sortedRunTicketIDs()
	require.Equal(t, []string{"a", "b"}, ids)
}

func TestRunSlotsUsedAndAvailableSlots(t *testing.T) {
	runtime := &mobileRunnerSemaphoreRuntime{
		capacity: 2,
		runTickets: map[string]MobileRunnerSemaphoreRunTicketState{
			"t1": {Status: mobileRunnerSemaphoreRunStarting},
			"t2": {Status: mobileRunnerSemaphoreRunQueued},
			"t3": {Status: mobileRunnerSemaphoreRunRunning},
		},
	}
	require.Equal(t, 2, runtime.runSlotsUsed())
	require.Equal(t, 0, runtime.availableSlots())
}

func TestRunQueuePositionAdditional(t *testing.T) {
	runtime := &mobileRunnerSemaphoreRuntime{runQueue: []string{"a", "b"}}
	pos, lineLen := runtime.runQueuePosition("b")
	require.Equal(t, 1, pos)
	require.Equal(t, 2, lineLen)

	pos, lineLen = runtime.runQueuePosition("missing")
	require.Equal(t, 0, pos)
	require.Equal(t, 2, lineLen)
}

func TestSortRunQueueHandlesMissingTickets(t *testing.T) {
	tickets := map[string]MobileRunnerSemaphoreRunTicketState{
		"present": {
			Request: MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:   "present",
				EnqueuedAt: time.Now(),
			},
		},
	}
	queue := []string{"missing", "present"}
	sorted := sortRunQueue(queue, tickets)
	require.Equal(t, []string{"present", "missing"}, sorted)
}

func TestContainsString(t *testing.T) {
	require.True(t, containsString([]string{"a", "b"}, "b"))
	require.False(t, containsString([]string{"a", "b"}, "c"))
}

func TestRuntimeFlagsAndCounts(t *testing.T) {
	runtime := &mobileRunnerSemaphoreRuntime{
		runnerID: "runner-1",
		capacity: 0,
		runTickets: map[string]MobileRunnerSemaphoreRunTicketState{
			"t1": {Status: mobileRunnerSemaphoreRunRunning},
			"t2": {
				Status: mobileRunnerSemaphoreRunStarting,
				Request: MobileRunnerSemaphoreEnqueueRunRequest{
					LeaderRunnerID: "other",
				},
			},
		},
	}
	require.True(t, runtime.hasRunningTickets())
	require.True(t, runtime.hasFollowerStartingTickets())
	require.Equal(t, 2, runtime.runSlotsUsed())
	require.Equal(t, 0, runtime.availableSlots())
}

func TestInFlightRunCount(t *testing.T) {
	runtime := &mobileRunnerSemaphoreRuntime{
		runTickets: map[string]MobileRunnerSemaphoreRunTicketState{
			"t1": {
				Status: mobileRunnerSemaphoreRunQueued,
				Request: MobileRunnerSemaphoreEnqueueRunRequest{
					OwnerNamespace: "ns-1",
				},
			},
			"t2": {
				Status: mobileRunnerSemaphoreRunRunning,
				Request: MobileRunnerSemaphoreEnqueueRunRequest{
					OwnerNamespace: "ns-1",
				},
			},
			"t3": {
				Status: mobileRunnerSemaphoreRunRunning,
				Request: MobileRunnerSemaphoreEnqueueRunRequest{
					OwnerNamespace: "ns-2",
				},
			},
		},
	}
	require.Equal(t, 2, runtime.inFlightRunCount("ns-1"))
	require.Equal(t, 1, runtime.inFlightRunCount("ns-2"))
}

func TestMaybeScheduleContinue(t *testing.T) {
	runtime := &mobileRunnerSemaphoreRuntime{
		runnerID:    "runner-1",
		capacity:    1,
		updateCount: mobileRunnerSemaphoreMaxUpdateBatches,
		runTickets:  map[string]MobileRunnerSemaphoreRunTicketState{},
		runQueue:    []string{},
	}

	runtime.maybeScheduleContinue()
	require.True(t, runtime.shouldContinue)
	require.NotNil(t, runtime.continueInput.Payload)
}

func TestRunQueuePosition(t *testing.T) {
	runtime := &mobileRunnerSemaphoreRuntime{runQueue: []string{"t1", "t2"}}
	pos, lineLen := runtime.runQueuePosition("t2")
	require.Equal(t, 1, pos)
	require.Equal(t, 2, lineLen)
	pos, lineLen = runtime.runQueuePosition("missing")
	require.Equal(t, 0, pos)
	require.Equal(t, 2, lineLen)
}

func TestSortRunQueue(t *testing.T) {
	t0 := time.Now()
	t1 := t0.Add(time.Second)
	tickets := map[string]MobileRunnerSemaphoreRunTicketState{
		"a": {Request: MobileRunnerSemaphoreEnqueueRunRequest{TicketID: "a", EnqueuedAt: t1}},
		"b": {Request: MobileRunnerSemaphoreEnqueueRunRequest{TicketID: "b", EnqueuedAt: t0}},
		"c": {Request: MobileRunnerSemaphoreEnqueueRunRequest{TicketID: "c", EnqueuedAt: t0}},
	}
	queue := sortRunQueue([]string{"a", "b", "c"}, tickets)
	require.Equal(t, []string{"b", "c", "a"}, queue)
}

func TestCopyRunTicketsDeepCopy(t *testing.T) {
	now := time.Now()
	original := map[string]MobileRunnerSemaphoreRunTicketState{
		"ticket": {
			Request: MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:          "ticket",
				RequiredRunnerIDs: []string{"r1"},
				PipelineConfig:    map[string]any{"k": "v"},
				Memo:              map[string]any{"m": "x"},
			},
			GrantedRunnerIDs: map[string]bool{"r1": true},
			StartedAt:        &now,
		},
	}

	copied := copyRunTickets(original)
	require.Equal(t, original, copied)

	entry := copied["ticket"]
	entry.Request.RequiredRunnerIDs[0] = "r2"
	entry.Request.PipelineConfig["k"] = "changed"
	entry.GrantedRunnerIDs["r1"] = false
	copied["ticket"] = entry

	require.Equal(t, "r1", original["ticket"].Request.RequiredRunnerIDs[0])
	require.Equal(t, "v", original["ticket"].Request.PipelineConfig["k"])
	require.True(t, original["ticket"].GrantedRunnerIDs["r1"])
}
