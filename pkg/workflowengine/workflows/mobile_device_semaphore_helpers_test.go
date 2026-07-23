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
	require.Equal(t, MobileDeviceSemaphoreErrInvalidRequest, appErr.Type())
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
	view := MobileDeviceSemaphoreRunStatusView{
		TicketID: "ticket-1",
		Status:   mobileDeviceSemaphoreRunQueued,
		Position: 1,
		LineLen:  2,
	}

	decoded, err := decodeRunStatusView(view)
	require.NoError(t, err)
	require.Equal(t, view, decoded)

	decoded, err = decodeRunStatusView(map[string]any{
		"ticket_id": "ticket-2",
		"status":    string(mobileDeviceSemaphoreRunRunning),
		"position":  0,
		"line_len":  1,
	})
	require.NoError(t, err)
	require.Equal(t, "ticket-2", decoded.TicketID)
	require.Equal(t, mobileDeviceSemaphoreRunRunning, decoded.Status)

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

func TestDecodeCancelWorkflowOutput(t *testing.T) {
	output := activities.CancelWorkflowActivityOutput{Canceled: true, Status: "CANCELED"}

	decoded, err := decodeCancelWorkflowOutput(output)
	require.NoError(t, err)
	require.Equal(t, output, decoded)

	decoded, err = decodeCancelWorkflowOutput(map[string]any{
		"canceled": false,
		"status":   "NOT_FOUND",
	})
	require.NoError(t, err)
	require.False(t, decoded.Canceled)
	require.Equal(t, "NOT_FOUND", decoded.Status)

	_, err = decodeCancelWorkflowOutput(1)
	require.Error(t, err)
}

func TestDecodeCleanupMobileDeviceSemaphoreResourcesOutput(t *testing.T) {
	output := activities.CleanupMobileDeviceSemaphoreResourcesActivityOutput{
		CleanupFailures: []string{"boom"},
	}

	decoded, err := decodeCleanupMobileDeviceSemaphoreResourcesOutput(output)
	require.NoError(t, err)
	require.Equal(t, output, decoded)

	decoded, err = decodeCleanupMobileDeviceSemaphoreResourcesOutput(map[string]any{
		"cleanup_failures": []any{"boom", "bang"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"boom", "bang"}, decoded.CleanupFailures)

	_, err = decodeCleanupMobileDeviceSemaphoreResourcesOutput(1)
	require.Error(t, err)
}

func TestBuildRunStatusViewCopiesSlice(t *testing.T) {
	runtime := &mobileDeviceSemaphoreRuntime{}
	state := MobileDeviceSemaphoreRunTicketState{
		Status:            mobileDeviceSemaphoreRunRunning,
		WorkflowID:        "wf-1",
		RunID:             "run-1",
		WorkflowNamespace: "ns-1",
		Request: MobileDeviceSemaphoreEnqueueRunRequest{
			LeaderDeviceID:    "leader-1",
			RequiredDeviceIDs: []string{"r1", "r2"},
		},
		ErrorMessage: "oops",
	}

	view := runtime.buildRunStatusView("ticket-1", state)
	require.Equal(t, "ticket-1", view.TicketID)
	require.Equal(t, mobileDeviceSemaphoreRunRunning, view.Status)
	require.Equal(t, "leader-1", view.LeaderDeviceID)
	require.Equal(t, []string{"r1", "r2"}, view.RequiredDeviceIDs)
	require.Equal(t, "wf-1", view.WorkflowID)
	require.Equal(t, "run-1", view.RunID)
	require.Equal(t, "ns-1", view.WorkflowNamespace)
	require.Equal(t, "oops", view.ErrorMessage)

	state.Request.RequiredDeviceIDs[0] = "changed"
	require.Equal(t, []string{"r1", "r2"}, view.RequiredDeviceIDs)
}

func TestRunTicketHelpers(t *testing.T) {
	state := MobileDeviceSemaphoreRunTicketState{
		WorkflowID:        "wf-1",
		WorkflowNamespace: "ns-1",
		Request: MobileDeviceSemaphoreEnqueueRunRequest{
			PipelineConfig: map[string]any{"app_url": "https://example.test"},
		},
	}
	require.True(t, ticketHasStartedWorkflow(state))
	require.Equal(t, "https://example.test", runTicketAppURL(state))

	state.WorkflowNamespace = ""
	state.Request.PipelineConfig = nil
	state.Request.Notification = &MobileDeviceSemaphoreNotification{
		GitHubPR: &MobileDeviceSemaphoreGitHubPRNotification{AppURL: "https://fallback.test"},
	}
	require.False(t, ticketHasStartedWorkflow(state))
	require.Equal(t, "https://fallback.test", runTicketAppURL(state))

	require.Equal(t, []string{"a", "b"}, sortedDeviceIDs([]string{"b", "a"}))
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
	tickets := map[string]MobileDeviceSemaphoreRunTicketState{
		"old": {
			Request: MobileDeviceSemaphoreEnqueueRunRequest{TicketID: "old", EnqueuedAt: older},
		},
		"new": {Request: MobileDeviceSemaphoreEnqueueRunRequest{TicketID: "new", EnqueuedAt: now}},
	}
	queue := []string{"new"}
	queue = insertRunQueue(queue, "old", tickets)
	require.Equal(t, []string{"old", "new"}, queue)
}

func TestNextQueuedRunTicketSkipsNonQueued(t *testing.T) {
	runtime := &mobileDeviceSemaphoreRuntime{
		runQueue: []string{"t1", "t2"},
		runTickets: map[string]MobileDeviceSemaphoreRunTicketState{
			"t1": {Status: mobileDeviceSemaphoreRunRunning},
			"t2": {Status: mobileDeviceSemaphoreRunQueued},
		},
	}

	id, _, ok := runtime.nextQueuedRunTicket()
	require.True(t, ok)
	require.Equal(t, "t2", id)
	require.Len(t, runtime.runQueue, 1)
}

func TestAllGrantsReceived(t *testing.T) {
	runtime := &mobileDeviceSemaphoreRuntime{}
	state := MobileDeviceSemaphoreRunTicketState{
		Request: MobileDeviceSemaphoreEnqueueRunRequest{
			RequiredDeviceIDs: []string{"r1", "r2"},
		},
		GrantedDeviceIDs: map[string]bool{"r1": true},
	}
	require.False(t, runtime.allGrantsReceived(state))

	state.GrantedDeviceIDs["r2"] = true
	require.True(t, runtime.allGrantsReceived(state))
}

func TestSortedRunTicketIDs(t *testing.T) {
	now := time.Now()
	runtime := &mobileDeviceSemaphoreRuntime{
		runTickets: map[string]MobileDeviceSemaphoreRunTicketState{
			"b": {Request: MobileDeviceSemaphoreEnqueueRunRequest{TicketID: "b", EnqueuedAt: now}},
			"a": {
				Request: MobileDeviceSemaphoreEnqueueRunRequest{
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
	runtime := &mobileDeviceSemaphoreRuntime{
		capacity: 2,
		runTickets: map[string]MobileDeviceSemaphoreRunTicketState{
			"t1": {Status: mobileDeviceSemaphoreRunStarting},
			"t2": {Status: mobileDeviceSemaphoreRunQueued},
			"t3": {Status: mobileDeviceSemaphoreRunRunning},
		},
	}
	require.Equal(t, 2, runtime.runSlotsUsed())
	require.Equal(t, 0, runtime.availableSlots())
}

func TestRunQueuePositionAdditional(t *testing.T) {
	runtime := &mobileDeviceSemaphoreRuntime{runQueue: []string{"a", "b"}}
	pos, lineLen := runtime.runQueuePosition("b")
	require.Equal(t, 1, pos)
	require.Equal(t, 2, lineLen)

	pos, lineLen = runtime.runQueuePosition("missing")
	require.Equal(t, 0, pos)
	require.Equal(t, 2, lineLen)
}

func TestSortRunQueueHandlesMissingTickets(t *testing.T) {
	tickets := map[string]MobileDeviceSemaphoreRunTicketState{
		"present": {
			Request: MobileDeviceSemaphoreEnqueueRunRequest{
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
	runtime := &mobileDeviceSemaphoreRuntime{
		runnerID: "runner-1",
		capacity: 0,
		runTickets: map[string]MobileDeviceSemaphoreRunTicketState{
			"t1": {Status: mobileDeviceSemaphoreRunRunning},
			"t2": {
				Status: mobileDeviceSemaphoreRunStarting,
				Request: MobileDeviceSemaphoreEnqueueRunRequest{
					LeaderDeviceID: "other",
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
	runtime := &mobileDeviceSemaphoreRuntime{
		runTickets: map[string]MobileDeviceSemaphoreRunTicketState{
			"t1": {
				Status: mobileDeviceSemaphoreRunQueued,
				Request: MobileDeviceSemaphoreEnqueueRunRequest{
					OwnerNamespace: "ns-1",
				},
			},
			"t2": {
				Status: mobileDeviceSemaphoreRunRunning,
				Request: MobileDeviceSemaphoreEnqueueRunRequest{
					OwnerNamespace: "ns-1",
				},
			},
			"t3": {
				Status: mobileDeviceSemaphoreRunRunning,
				Request: MobileDeviceSemaphoreEnqueueRunRequest{
					OwnerNamespace: "ns-2",
				},
			},
		},
	}
	require.Equal(t, 2, runtime.inFlightRunCount("ns-1"))
	require.Equal(t, 1, runtime.inFlightRunCount("ns-2"))
}

func TestMaybeScheduleContinue(t *testing.T) {
	runtime := &mobileDeviceSemaphoreRuntime{
		runnerID:    "runner-1",
		capacity:    1,
		updateCount: mobileDeviceSemaphoreMaxUpdateBatches,
		runTickets:  map[string]MobileDeviceSemaphoreRunTicketState{},
		runQueue:    []string{},
	}

	runtime.maybeScheduleContinue()
	require.True(t, runtime.shouldContinue)
	require.NotNil(t, runtime.continueInput.Payload)
}

func TestRunQueuePosition(t *testing.T) {
	runtime := &mobileDeviceSemaphoreRuntime{runQueue: []string{"t1", "t2"}}
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
	tickets := map[string]MobileDeviceSemaphoreRunTicketState{
		"a": {Request: MobileDeviceSemaphoreEnqueueRunRequest{TicketID: "a", EnqueuedAt: t1}},
		"b": {Request: MobileDeviceSemaphoreEnqueueRunRequest{TicketID: "b", EnqueuedAt: t0}},
		"c": {Request: MobileDeviceSemaphoreEnqueueRunRequest{TicketID: "c", EnqueuedAt: t0}},
	}
	queue := sortRunQueue([]string{"a", "b", "c"}, tickets)
	require.Equal(t, []string{"b", "c", "a"}, queue)
}

func TestCopyRunTicketsDeepCopy(t *testing.T) {
	now := time.Now()
	original := map[string]MobileDeviceSemaphoreRunTicketState{
		"ticket": {
			Request: MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:          "ticket",
				RequiredDeviceIDs: []string{"r1"},
				PipelineConfig:    map[string]any{"k": "v"},
				Memo:              map[string]any{"m": "x"},
			},
			GrantedDeviceIDs: map[string]bool{"r1": true},
			StartedAt:        &now,
		},
	}

	copied := copyRunTickets(original)
	require.Equal(t, original, copied)

	entry := copied["ticket"]
	entry.Request.RequiredDeviceIDs[0] = "r2"
	entry.Request.PipelineConfig["k"] = "changed"
	entry.GrantedDeviceIDs["r1"] = false
	copied["ticket"] = entry

	require.Equal(t, "r1", original["ticket"].Request.RequiredDeviceIDs[0])
	require.Equal(t, "v", original["ticket"].Request.PipelineConfig["k"])
	require.True(t, original["ticket"].GrantedDeviceIDs["r1"])
}
