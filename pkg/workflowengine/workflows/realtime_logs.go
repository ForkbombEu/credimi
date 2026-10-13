// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"encoding/json"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

// realtimeLogsTracker remembers the logs last published by a poller so
// unchanged polls do not resend the same list.
type realtimeLogsTracker struct {
	lastSent string
}

// changed reports whether logs differ from the last sent value and records
// them. Logs that cannot be marshalled always count as changed.
func (t *realtimeLogsTracker) changed(logs []map[string]any) bool {
	encoded, err := json.Marshal(logs)
	if err != nil {
		return true
	}
	if string(encoded) == t.lastSent {
		return false
	}
	t.lastSent = string(encoded)
	return true
}

// sendRealtimeLogsUpdate publishes workflow logs to the realtime subscription
// topic through the SendRealtimeLogs activity.
func sendRealtimeLogsUpdate(
	ctx workflow.Context,
	subscription string,
	logs []map[string]any,
) error {
	input := workflowengine.ActivityInput{
		Payload: activities.SendRealtimeLogsInput{
			Subscription: subscription,
			Logs:         logs,
		},
	}
	return workflow.ExecuteActivity(ctx, activities.SendRealtimeLogsActivityName, input).
		Get(ctx, nil)
}
