// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"fmt"
	"strconv"
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
	return utils.JoinURL("github-pr-comment", repository, strconv.Itoa(prNumber))
}

func buildGitHubPRCommentBody(input UpdateGitHubPRCommentInput) string {
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "queued"
	}
	result := ""
	if strings.TrimSpace(input.WorkflowStatus) != "" {
		result = formatWorkflowResult(input.WorkflowStatus)
	}
	badgeStatus := status
	if result != "" {
		badgeStatus = result
	}
	tableRows := make([][2]string, 0, 4)
	if input.Position != nil && status == "queued" {
		tableRows = append(tableRows, [2]string{"Queue position", fmt.Sprintf("`%d`", *input.Position)})
	}
	if strings.TrimSpace(input.PipelineURL) != "" {
		tableRows = append(tableRows, [2]string{"Pipeline", markdownLink("Open pipeline", input.PipelineURL)})
	}
	if runURL := buildRunURL(input.AppURL, input.WorkflowID, input.RunID); runURL != "" {
		tableRows = append(tableRows, [2]string{"Run logs", markdownLink("Open run logs", runURL)})
	}
	if strings.TrimSpace(input.ErrorMessage) != "" {
		tableRows = append(tableRows, [2]string{"Error", markdownTableCell(input.ErrorMessage)})
	}

	lines := make([]string, 0, 4+len(tableRows))
	lines = append(lines, formatPRCommentStatusBadge(badgeStatus))
	if len(tableRows) > 0 {
		lines = append(lines, "", "| Field | Value |", "| --- | --- |")
		for _, row := range tableRows {
			lines = append(lines, fmt.Sprintf("| %s | %s |", row[0], row[1]))
		}
	}
	return strings.Join(lines, "\n")
}

func BuildGitHubPRCommentBodyForWorkflow(input UpdateGitHubPRCommentInput) string {
	return buildGitHubPRCommentBody(input)
}

func formatPRCommentStatusBadge(status string) string {
	message := normalizePRCommentBadgeMessage(status)
	color := prCommentBadgeColor(message)
	badgeURL := utils.JoinURL("https://img.shields.io", "badge", fmt.Sprintf("status-%s-%s", message, color))
	return fmt.Sprintf("![status: %s](%s)", message, badgeURL)
}

func markdownLink(label string, linkURL string) string {
	return fmt.Sprintf("[%s](%s)", label, strings.TrimSpace(linkURL))
}

func markdownTableCell(value string) string {
	cell := strings.TrimSpace(value)
	cell = strings.ReplaceAll(cell, "\r\n", " ")
	cell = strings.ReplaceAll(cell, "\n", " ")
	cell = strings.ReplaceAll(cell, "|", "\\|")
	return cell
}

func normalizePRCommentBadgeMessage(status string) string {
	message := strings.ToLower(strings.TrimSpace(status))
	message = strings.ReplaceAll(message, " ", "_")
	if message == "" {
		return "queued"
	}
	return message
}

func prCommentBadgeColor(status string) string {
	switch status {
	case "queued":
		return "yellow"
	case "starting", "running":
		return "blue"
	case "success", "successful", "completed":
		return "brightgreen"
	case "failed", "failure":
		return "red"
	case "canceled", "cancelled":
		return "lightgrey"
	case "terminated", "timed_out", "timeout":
		return "orange"
	default:
		return "informational"
	}
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
