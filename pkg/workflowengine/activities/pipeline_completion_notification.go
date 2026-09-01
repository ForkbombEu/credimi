// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

type SendPipelineCompletionNotificationInput struct {
	AppURL       string `json:"app_url"`
	WorkflowID   string `json:"workflow_id"`
	RunID        string `json:"run_id"`
	Result       string `json:"result"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type SendPipelineCompletionNotificationActivity struct {
	workflowengine.BaseActivity
}

func NewSendPipelineCompletionNotificationActivity() *SendPipelineCompletionNotificationActivity {
	return &SendPipelineCompletionNotificationActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Send pipeline completion notification",
		},
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

	appURL := strings.TrimSpace(payload.AppURL)
	workflowID := strings.TrimSpace(payload.WorkflowID)
	runID := strings.TrimSpace(payload.RunID)
	if appURL == "" || workflowID == "" || runID == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(workflowengine.ActivityError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: "app_url, workflow_id, and run_id are required",
		})
	}

	apiKey := strings.TrimSpace(os.Getenv("CREDIMI_INTERNAL_ADMIN_KEY"))
	if apiKey == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
		return result, a.NewActivityError(workflowengine.ActivityError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: "CREDIMI_INTERNAL_ADMIN_KEY is required",
		})
	}

	body, err := json.Marshal(SendPipelineCompletionNotificationInput{
		AppURL:       appURL,
		WorkflowID:   workflowID,
		RunID:        runID,
		Result:       strings.TrimSpace(payload.Result),
		ErrorMessage: strings.TrimSpace(payload.ErrorMessage),
	})
	if err != nil {
		return result, fmt.Errorf("marshal pipeline completion notification payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		utils.JoinURL(appURL, "api", "web-push", "pipeline-completed"),
		bytes.NewReader(body),
	)
	if err != nil {
		return result, fmt.Errorf("build pipeline completion notification request: %w", err)
	}
	req.Header.Set(workflowengine.HTTPHeaderContentType, workflowengine.MIMEApplicationJSON)
	req.Header.Set("Credimi-Api-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, fmt.Errorf("post pipeline completion notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf(
			"pipeline completion notification status: %s",
			resp.Status,
		)
	}
	return result, nil
}
