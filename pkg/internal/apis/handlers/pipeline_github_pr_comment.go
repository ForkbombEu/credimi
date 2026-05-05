// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/githubapp"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
)

var signalGitHubPRCommentUpdate = activities.SignalGitHubPRCommentUpdate

func buildWalletAPKGitHubPRNotification(
	metadata map[string]any,
	appURL string,
	pipelineIdentifier string,
) *workflows.MobileRunnerSemaphoreNotification {
	repository := metadataString(metadata, "repository")
	prNumber := pullRequestNumberFromMetadata(metadata)
	if repository == "" || prNumber <= 0 {
		return nil
	}

	return &workflows.MobileRunnerSemaphoreNotification{
		GitHubPR: &workflows.MobileRunnerSemaphoreGitHubPRNotification{
			Repository:         repository,
			PullRequestNumber:  prNumber,
			CommitSHA:          metadataString(metadata, "event.pull_request.head.sha"),
			PipelineIdentifier: pipelineIdentifier,
			PipelineURL:        buildPipelinePageURL(appURL, pipelineIdentifier),
			AppURL:             appURL,
		},
	}
}

func maybeCreateWalletAPKQueuedPRComment(
	ctx context.Context,
	notification *workflows.MobileRunnerSemaphoreNotification,
	response PipelineRunWalletAPKResponse,
) (int64, error) {
	if notification == nil || notification.GitHubPR == nil {
		return 0, nil
	}
	err := signalGitHubPRCommentUpdate(ctx, activities.UpdateGitHubPRCommentInput{
		Repository:        notification.GitHubPR.Repository,
		PullRequestNumber: notification.GitHubPR.PullRequestNumber,
		CommentID:         notification.GitHubPR.CommentID,
		CommitSHA:         notification.GitHubPR.CommitSHA,
		Status:            string(response.Status),
		Position:          response.Position,
		PipelineURL:       notification.GitHubPR.PipelineURL,
		AppURL:            notification.GitHubPR.AppURL,
		WorkflowID:        response.WorkflowID,
		RunID:             response.RunID,
		TicketID:          response.TicketID,
		ErrorMessage:      response.ErrorMessage,
	})
	if err != nil {
		return 0, err
	}
	return 0, nil
}

type walletAPKPRCommentBodyInput struct {
	Status         string
	Position       *int
	PipelineURL    string
	RunURL         string
	WorkflowStatus string
	ErrorMessage   string
}

func buildWalletAPKPRCommentBody(input walletAPKPRCommentBodyInput) string {
	var lines []string
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "queued"
	}
	lines = append(lines, fmt.Sprintf("Credimi wallet APK pipeline is %s.", status))
	if input.Position != nil && status == "queued" {
		lines = append(lines, fmt.Sprintf("Queue position: %d.", *input.Position))
	}
	if input.PipelineURL != "" {
		lines = append(lines, fmt.Sprintf("Pipeline: %s", input.PipelineURL))
	}
	if input.RunURL != "" {
		lines = append(lines, fmt.Sprintf("Run logs: %s", input.RunURL))
	}
	if input.WorkflowStatus != "" {
		lines = append(lines, fmt.Sprintf("Result: %s", formatWalletAPKWorkflowResult(input.WorkflowStatus)))
	}
	if input.ErrorMessage != "" {
		lines = append(lines, fmt.Sprintf("Error: %s", input.ErrorMessage))
	}
	return strings.Join(lines, "\n")
}

func formatWalletAPKWorkflowResult(status string) string {
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

func pullRequestNumberFromMetadata(metadata map[string]any) int {
	return githubapp.IntFromAny(metadataValue(metadata, "event.number"))
}

func metadataString(metadata map[string]any, key string) string {
	value, _ := metadataValue(metadata, key).(string)
	return strings.TrimSpace(value)
}

func metadataValue(metadata map[string]any, key string) any {
	if metadata == nil {
		return nil
	}
	if value, ok := metadata[key]; ok {
		return value
	}
	parts := strings.Split(key, ".")
	var current any = metadata
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[part]
	}
	return current
}
