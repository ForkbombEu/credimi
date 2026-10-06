// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

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
