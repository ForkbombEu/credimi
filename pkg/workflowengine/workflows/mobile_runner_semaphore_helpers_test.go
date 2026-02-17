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

func TestBuildHoldersViewSorted(t *testing.T) {
	view := buildHoldersView(map[string]MobileRunnerSemaphoreHolder{
		"lease-b": {LeaseID: "lease-b"},
		"lease-a": {LeaseID: "lease-a"},
	})
	require.Len(t, view, 2)
	require.Equal(t, "lease-a", view[0].LeaseID)
	require.Equal(t, "lease-b", view[1].LeaseID)
}

func TestBuildQueuePreviewSkipsMissing(t *testing.T) {
	now := time.Now()
	requests := map[string]MobileRunnerSemaphoreRequestState{
		"req-1": {
			Request: MobileRunnerSemaphoreAcquireRequest{
				RequestID: "req-1",
				LeaseID:   "lease-1",
			},
			RequestedAt: now,
		},
	}
	preview := buildQueuePreview([]string{"req-1", "missing"}, requests)
	require.Len(t, preview, 1)
	require.Equal(t, "req-1", preview[0].RequestID)
	require.Equal(t, "lease-1", preview[0].LeaseID)
	require.Equal(t, now, preview[0].RequestedAt)
}

func TestRemoveFromQueue(t *testing.T) {
	queue := []string{"a", "b", "c"}
	updated := removeFromQueue(queue, "b")
	require.Equal(t, []string{"a", "c"}, updated)
	require.Equal(t, queue, removeFromQueue(queue, "missing"))
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

	copy := copyRunTickets(original)
	require.Equal(t, original, copy)

	entry := copy["ticket"]
	entry.Request.RequiredRunnerIDs[0] = "r2"
	entry.Request.PipelineConfig["k"] = "changed"
	entry.GrantedRunnerIDs["r1"] = false
	copy["ticket"] = entry

	require.Equal(t, "r1", original["ticket"].Request.RequiredRunnerIDs[0])
	require.Equal(t, "v", original["ticket"].Request.PipelineConfig["k"])
	require.True(t, original["ticket"].GrantedRunnerIDs["r1"])
}
