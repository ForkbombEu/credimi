// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

// CompletionNotificationConfigKey marks queued runs that should send a web
// push notification when they reach a terminal state. The queue handler
// injects it for every run it starts, directly or through the semaphore.
const CompletionNotificationConfigKey = "pipeline_completion_notification"

func reportPipelineCompletionNotification(
	ctx workflow.Context,
	logger log.Logger,
	config map[string]any,
	workflowID string,
	runID string,
	workflowResult string,
	workflowErr error,
) {
	if config == nil {
		return
	}

	// The queue start path injects this key when it also creates the
	// pipeline_results record the notification handler resolves.
	if enabled, _ := config[CompletionNotificationConfigKey].(bool); !enabled {
		return
	}
	appURL, _ := config["app_url"].(string)
	if appURL == "" {
		return
	}

	notificationActivity := activities.NewSendPipelineCompletionNotificationActivity()
	payload := activities.SendPipelineCompletionNotificationInput{
		AppURL:       appURL,
		WorkflowID:   workflowID,
		RunID:        runID,
		Result:       workflowResult,
		ErrorMessage: workflowErrorMessage(workflowErr),
	}

	if err := workflow.ExecuteActivity(
		ctx,
		notificationActivity.Name(),
		workflowengine.ActivityInput{Payload: payload},
	).Get(ctx, nil); err != nil {
		logger.Error(
			"failed to send pipeline completion notification",
			"workflow_id",
			workflowID,
			"run_id",
			runID,
			"error",
			err,
		)
	}
}

// truncateWorkflowError caps a notification body by rune count so multi-byte
// characters are never split mid-rune.
func truncateWorkflowError(message string) string {
	const maxLength = 140
	runes := []rune(message)
	if len(runes) <= maxLength {
		return message
	}
	return string(runes[:maxLength]) + "…"
}

// workflowErrorMessage derives a short, notification-sized cause from the
// workflow error. Empty for canceled runs and plain Temporal errors.
func workflowErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	wfErr := workflowengine.ParseWorkflowError(err)
	if wfErr.Code == "" && wfErr.Summary == "" {
		return ""
	}
	if wfErr.Code == "" {
		return truncateWorkflowError(wfErr.Summary)
	}
	if wfErr.Summary == "" {
		return truncateWorkflowError(wfErr.Code)
	}
	return truncateWorkflowError(wfErr.Code + ": " + wfErr.Summary)
}
