// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"fmt"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/githubapp"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"go.temporal.io/sdk/client"
)

const (
	GitHubPRCommentWorkflowName = "GitHub PR comment workflow"
	GitHubPRCommentUpdateSignal = "github-pr-comment-update"
)

type UpdateGitHubPRCommentInput struct {
	Repository        string `json:"repository"`
	PullRequestNumber int    `json:"pull_request_number"`
	CommitSHA         string `json:"commit_sha,omitempty"`
	TicketID          string `json:"ticket_id"`
	Status            string `json:"status"`
	Position          *int   `json:"position,omitempty"`
	PipelineURL       string `json:"pipeline_url,omitempty"`
	AppURL            string `json:"app_url,omitempty"`
	WorkflowID        string `json:"workflow_id,omitempty"`
	RunID             string `json:"run_id,omitempty"`
	WorkflowStatus    string `json:"workflow_status,omitempty"`
	ErrorMessage      string `json:"error_message,omitempty"`
	CommentID         int64  `json:"comment_id,omitempty"`
}

type PatchGitHubPRCommentInput struct {
	Repository        string `json:"repository"`
	PullRequestNumber int    `json:"pull_request_number"`
	CommentID         int64  `json:"comment_id,omitempty"`
	Body              string `json:"body"`
}

type PatchGitHubPRCommentOutput struct {
	CommentID int64 `json:"comment_id"`
}

type UpdateGitHubPRCommentActivity struct {
	workflowengine.BaseActivity
}

func NewUpdateGitHubPRCommentActivity() *UpdateGitHubPRCommentActivity {
	return &UpdateGitHubPRCommentActivity{
		BaseActivity: workflowengine.BaseActivity{Name: "Update GitHub PR comment"},
	}
}

func (a *UpdateGitHubPRCommentActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *UpdateGitHubPRCommentActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[UpdateGitHubPRCommentInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	if strings.TrimSpace(payload.Repository) == "" ||
		payload.PullRequestNumber <= 0 ||
		strings.TrimSpace(payload.TicketID) == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, "repository, pull_request_number, and ticket_id are required")
	}
	if err := SignalGitHubPRCommentUpdate(ctx, payload); err != nil {
		return result, err
	}
	return result, nil
}

type PatchGitHubPRCommentActivity struct {
	workflowengine.BaseActivity
}

func NewPatchGitHubPRCommentActivity() *PatchGitHubPRCommentActivity {
	return &PatchGitHubPRCommentActivity{
		BaseActivity: workflowengine.BaseActivity{Name: "Patch GitHub PR comment"},
	}
}

func (a *PatchGitHubPRCommentActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *PatchGitHubPRCommentActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[PatchGitHubPRCommentInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	client, err := githubapp.NewFromEnv()
	if err != nil {
		return result, err
	}
	comment, err := client.CreateOrUpdatePRComment(ctx, githubapp.PRComment{
		Repository:        payload.Repository,
		PullRequestNumber: payload.PullRequestNumber,
		CommentID:         payload.CommentID,
		Marker:            githubapp.Marker(),
		Body:              payload.Body,
	})
	if err != nil {
		return result, err
	}
	result.Output = PatchGitHubPRCommentOutput{CommentID: comment.CommentID}
	return result, nil
}

func SignalGitHubPRCommentUpdate(ctx context.Context, input UpdateGitHubPRCommentInput) error {
	temporalClient, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return err
	}
	workflowID := GitHubPRCommentWorkflowID(input.Repository, input.PullRequestNumber)
	_, err = temporalClient.SignalWithStartWorkflow(
		ctx,
		workflowID,
		GitHubPRCommentUpdateSignal,
		input,
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: mobilerunnersemaphore.TaskQueue,
		},
		GitHubPRCommentWorkflowName,
		workflowengine.WorkflowInput{},
	)
	return err
}

func GitHubPRCommentWorkflowID(repository string, prNumber int) string {
	repository = strings.Trim(strings.TrimSpace(repository), "/")
	return fmt.Sprintf("github-pr-comment/%s/%d", repository, prNumber)
}

func buildGitHubPRCommentBody(input UpdateGitHubPRCommentInput) string {
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "queued"
	}
	lines := []string{fmt.Sprintf("Credimi wallet APK pipeline is %s.", status)}
	if input.Position != nil && status == "queued" {
		lines = append(lines, fmt.Sprintf("Queue position: %d.", *input.Position))
	}
	if strings.TrimSpace(input.PipelineURL) != "" {
		lines = append(lines, fmt.Sprintf("Pipeline: %s", input.PipelineURL))
	}
	if runURL := buildRunURL(input.AppURL, input.WorkflowID, input.RunID); runURL != "" {
		lines = append(lines, fmt.Sprintf("Run logs: %s", runURL))
	}
	if strings.TrimSpace(input.WorkflowStatus) != "" {
		lines = append(lines, fmt.Sprintf("Result: %s", formatWorkflowResult(input.WorkflowStatus)))
	}
	if strings.TrimSpace(input.ErrorMessage) != "" {
		lines = append(lines, fmt.Sprintf("Error: %s", input.ErrorMessage))
	}
	return strings.Join(lines, "\n")
}

func BuildGitHubPRCommentBodyForWorkflow(input UpdateGitHubPRCommentInput) string {
	return buildGitHubPRCommentBody(input)
}

func formatWorkflowResult(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success", "completed", "workflow_execution_status_completed":
		return "success"
	case "failed", "failure", "workflow_execution_status_failed":
		return "failed"
	case "canceled", "cancelled", "workflow_execution_status_canceled":
		return "canceled"
	case "terminated", "workflow_execution_status_terminated":
		return "terminated"
	case "timed_out", "timeout", "workflow_execution_status_timed_out":
		return "timed out"
	default:
		return strings.TrimSpace(status)
	}
}

func buildRunURL(appURL string, workflowID string, runID string) string {
	if strings.TrimSpace(appURL) == "" ||
		strings.TrimSpace(workflowID) == "" ||
		strings.TrimSpace(runID) == "" {
		return ""
	}
	return utils.JoinURL(appURL, "my", "tests", "runs", workflowID, runID)
}
