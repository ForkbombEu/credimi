// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/forkbombeu/credimi/pkg/internal/webpush"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

const SendPipelineCompletionNotificationActivityName = "Send pipeline completion notification"

type SendPipelineCompletionNotificationInput struct {
	WorkflowID   string `json:"workflow_id"             validate:"required"`
	RunID        string `json:"run_id"                  validate:"required"`
	Result       string `json:"result"                  validate:"required"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type SendPipelineCompletionNotificationOutput struct {
	Sent int `json:"sent"`
}

type SendPipelineCompletionNotificationActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewSendPipelineCompletionNotificationActivity(
	app core.App,
) *SendPipelineCompletionNotificationActivity {
	return &SendPipelineCompletionNotificationActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: SendPipelineCompletionNotificationActivityName,
		},
		app: app,
	}
}

func (a *SendPipelineCompletionNotificationActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *SendPipelineCompletionNotificationActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[SendPipelineCompletionNotificationInput](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	sent, err := sendPipelineCompletionNotification(ctx, a.app, payload)
	if err != nil {
		if errors.Is(err, pipelineresults.ErrNotFound) {
			errCode := errorcodes.Codes[errorcodes.RecordNotFound]
			return result, a.NewNonRetryableActivityError(workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
			})
		}
		errCode := errorcodes.Codes[errorcodes.ExecuteHTTPRequestFailed]
		return result, a.NewActivityError(workflowengine.ActivityError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: err.Error(),
		})
	}
	result.Output = SendPipelineCompletionNotificationOutput{Sent: sent}
	return result, nil
}

// sendPipelineCompletionNotification sends the web push notification of a
// finished pipeline run to the subscriptions of the run's organization and
// returns how many were sent.
func sendPipelineCompletionNotification(
	ctx context.Context,
	app core.App,
	in SendPipelineCompletionNotificationInput,
) (int, error) {
	record, err := pipelineresults.FindByWorkflowRun(app, in.WorkflowID, in.RunID)
	if err != nil {
		return 0, err
	}
	pipeline, err := app.FindRecordById("pipelines", record.GetString("pipeline"))
	if err != nil {
		return 0, fmt.Errorf("lookup pipeline: %w", err)
	}
	organization, err := app.FindRecordById("organizations", record.GetString("owner"))
	if err != nil {
		return 0, fmt.Errorf("lookup organization: %w", err)
	}
	duration := ""
	if startedAt := record.GetDateTime("created"); !startedAt.IsZero() {
		duration = time.Since(startedAt.Time()).Round(time.Second).String()
	}

	sent, err := webpush.NotifyPipelineRunCompletion(ctx, app, webpush.CompletionRequest{
		OrgID:        record.GetString("owner"),
		PipelineName: pipeline.GetString("name"),
		Organization: organization.GetString("name"),
		WorkflowID:   in.WorkflowID,
		RunID:        in.RunID,
		Result:       in.Result,
		Duration:     duration,
		Error:        in.ErrorMessage,
		AppURL:       workflowengine.AppURL(app),
	})
	if err != nil {
		return 0, fmt.Errorf("send pipeline completion notifications: %w", err)
	}
	return sent, nil
}
