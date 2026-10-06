// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

// covWERequireSemaphoreInvalidRequest asserts err carries the invalid-request semaphore code.
func covWERequireSemaphoreInvalidRequest(t *testing.T, err error, wantText string) {
	t.Helper()
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, MobileDeviceSemaphoreErrInvalidRequest, appErr.Type())
	assert.Contains(t, err.Error(), wantText)
}

func TestCovWESemaphoreActivityOutputDecoders(t *testing.T) {
	unmarshalable := map[string]any{"bad": make(chan int)}
	tests := []struct {
		name     string
		decode   func() (any, error)
		want     any
		wantText string
	}{
		{
			name: "start output with wrong field type",
			decode: func() (any, error) {
				return decodeStartQueuedPipelineOutput(map[string]any{"workflow_id": 5})
			},
			wantText: "failed to decode activity output",
		},
		{
			name: "start output that cannot be encoded",
			decode: func() (any, error) {
				return decodeStartQueuedPipelineOutput(unmarshalable)
			},
			wantText: "failed to encode activity output",
		},
		{
			name: "cancel output that cannot be encoded",
			decode: func() (any, error) {
				return decodeCancelWorkflowOutput(unmarshalable)
			},
			wantText: "failed to encode cancel workflow output",
		},
		{
			name: "cancel output with wrong field type",
			decode: func() (any, error) {
				return decodeCancelWorkflowOutput(map[string]any{"canceled": "yes"})
			},
			wantText: "failed to decode cancel workflow output",
		},
		{
			name: "cancel output of unknown type",
			decode: func() (any, error) {
				return decodeCancelWorkflowOutput(42)
			},
			wantText: "unexpected cancel workflow output",
		},
		{
			name: "typed signal output passes through",
			decode: func() (any, error) {
				return decodeSignalWorkflowOutput(activities.SignalWorkflowActivityOutput{
					Signaled: true,
					Status:   "running",
				})
			},
			want: activities.SignalWorkflowActivityOutput{Signaled: true, Status: "running"},
		},
		{
			name: "signal output map is decoded",
			decode: func() (any, error) {
				return decodeSignalWorkflowOutput(
					map[string]any{"signaled": false, "status": "closed"},
				)
			},
			want: activities.SignalWorkflowActivityOutput{Signaled: false, Status: "closed"},
		},
		{
			name: "signal output that cannot be encoded",
			decode: func() (any, error) {
				return decodeSignalWorkflowOutput(unmarshalable)
			},
			wantText: "failed to marshal signal workflow output",
		},
		{
			name: "signal output with wrong field type",
			decode: func() (any, error) {
				return decodeSignalWorkflowOutput(map[string]any{"signaled": "yes"})
			},
			wantText: "failed to decode signal workflow output",
		},
		{
			name: "signal output of unknown type",
			decode: func() (any, error) {
				return decodeSignalWorkflowOutput([]string{"x"})
			},
			wantText: "unsupported signal workflow output type []string",
		},
		{
			name: "cleanup output that cannot be encoded",
			decode: func() (any, error) {
				return decodeCleanupMobileDeviceSemaphoreResourcesOutput(unmarshalable)
			},
			wantText: "failed to encode cleanup output",
		},
		{
			name: "cleanup output with wrong field type",
			decode: func() (any, error) {
				return decodeCleanupMobileDeviceSemaphoreResourcesOutput(
					map[string]any{"cleanup_failures": "one"},
				)
			},
			wantText: "failed to decode cleanup output",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.decode()
			if tc.wantText != "" {
				covWERequireSemaphoreInvalidRequest(t, err, tc.wantText)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCovWEPruneTerminalRunTickets(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	old := now.Add(-terminalRunRetention - time.Second)
	recent := now.Add(-terminalRunRetention + time.Second)
	r := &mobileDeviceSemaphoreRuntime{
		runQueue: []string{"queued", "old-failed"},
		runTickets: map[string]MobileDeviceSemaphoreRunTicketState{
			"old-failed":   {Status: mobileDeviceSemaphoreRunFailed, DoneAt: &old},
			"old-canceled": {Status: mobileDeviceSemaphoreRunCanceled, DoneAt: &old},
			"recent":       {Status: mobileDeviceSemaphoreRunFailed, DoneAt: &recent},
			"no-done-at":   {Status: mobileDeviceSemaphoreRunNotFound},
			"queued":       {Status: mobileDeviceSemaphoreRunQueued, DoneAt: &old},
		},
	}

	r.pruneTerminalRunTickets(now)

	assert.ElementsMatch(t, []string{"recent", "no-done-at", "queued"}, covWEKeysOf(r.runTickets))
	assert.Equal(t, []string{"queued"}, r.runQueue)
	assert.Equal(t, 2, r.updateCount)
	assert.True(t, r.queuePositionsDirty)
}

func covWEKeysOf(m map[string]MobileDeviceSemaphoreRunTicketState) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestCovWESemaphoreCopyHelpersPreserveNil(t *testing.T) {
	assert.Nil(t, copyQueue(nil))
	assert.Nil(t, copyRunTickets(nil))
	assert.Nil(t, copyStringAnyMap(nil))
	assert.Nil(t, copyStringBoolMap(nil))

	original := map[string]MobileDeviceSemaphoreRunTicketState{
		"t": {
			Request: MobileDeviceSemaphoreEnqueueRunRequest{
				RequiredDeviceIDs: []string{"d"},
				PipelineConfig:    map[string]any{"k": "v"},
			},
			GrantedDeviceIDs: map[string]bool{"d": true},
		},
	}
	copied := copyRunTickets(original)
	copied["t"].Request.RequiredDeviceIDs[0] = "changed"
	copied["t"].Request.PipelineConfig["k"] = "changed"
	copied["t"].GrantedDeviceIDs["d"] = false
	assert.Equal(t, "d", original["t"].Request.RequiredDeviceIDs[0])
	assert.Equal(t, "v", original["t"].Request.PipelineConfig["k"])
	assert.True(t, original["t"].GrantedDeviceIDs["d"])
}

func TestCovWESortRunQueueOrdersUnknownTicketsLast(t *testing.T) {
	early := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tickets := map[string]MobileDeviceSemaphoreRunTicketState{
		"known-late": {
			Request: MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:   "known-late",
				EnqueuedAt: early.Add(time.Hour),
			},
		},
		"known-early": {
			Request: MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:   "known-early",
				EnqueuedAt: early,
			},
		},
	}

	got := sortRunQueue([]string{"zz-unknown", "known-late", "aa-unknown", "known-early"}, tickets)

	assert.Equal(t, []string{"known-early", "known-late", "aa-unknown", "zz-unknown"}, got)
}
