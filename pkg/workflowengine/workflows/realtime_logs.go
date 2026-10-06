// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"net/http"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

// sendRealtimeLogsUpdate posts workflow logs to a Credimi log-update route
// through the internal HTTP activity, which authenticates with
// CREDIMI_INTERNAL_ADMIN_KEY.
func sendRealtimeLogsUpdate(
	ctx workflow.Context,
	url string,
	workflowID string,
	logs []map[string]any,
	timeout string,
) error {
	internalHTTPActivity := activities.NewInternalHTTPActivity()
	input := workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method: http.MethodPost,
			URL:    url,
			Headers: map[string]string{
				workflowengine.HTTPHeaderContentType: workflowengine.MIMEApplicationJSON,
			},
			Body: map[string]any{
				"workflow_id": workflowID,
				"logs":        logs,
			},
			ExpectedStatus: http.StatusOK,
			Timeout:        timeout,
		},
	}
	return workflow.ExecuteActivity(ctx, internalHTTPActivity.Name(), input).Get(ctx, nil)
}
