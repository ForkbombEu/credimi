// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
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
	runnerID string,
	runnerType string,
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
			RunnerID:           runnerID,
			RunnerType:         runnerType,
			PipelineURL:        buildPipelinePageURL(appURL, pipelineIdentifier),
			AppURL:             appURL,
		},
	}
}

func maybeCreateWalletAPKQueuedPRComment(
	ctx context.Context,
	notification *workflows.MobileRunnerSemaphoreNotification,
	response PipelineRunWalletAPKResponse,
) error {
	if notification == nil || notification.GitHubPR == nil {
		return nil
	}
	return signalGitHubPRCommentUpdate(ctx, activities.UpdateGitHubPRCommentInput{
		Repository:        notification.GitHubPR.Repository,
		PullRequestNumber: notification.GitHubPR.PullRequestNumber,
		CommitSHA:         notification.GitHubPR.CommitSHA,
		Status:            string(response.Status),
		Position:          response.Position,
		PipelineID:        notification.GitHubPR.PipelineIdentifier,
		RunnerID:          githubPRCommentRunnerID(notification.GitHubPR.RunnerID, response.RunnerIDs),
		RunnerType:        notification.GitHubPR.RunnerType,
		PipelineURL:       notification.GitHubPR.PipelineURL,
		AppURL:            notification.GitHubPR.AppURL,
		WorkflowID:        response.WorkflowID,
		RunID:             response.RunID,
		TicketID:          response.TicketID,
		ErrorMessage:      response.ErrorMessage,
	})
}

func githubPRCommentRunnerID(runnerID string, runnerIDs []string) string {
	if strings.TrimSpace(runnerID) != "" {
		return runnerID
	}
	if len(runnerIDs) == 0 {
		return ""
	}
	return runnerIDs[0]
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
