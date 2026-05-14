// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"strconv"
	"strings"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

const (
	GitHubPRCommentConfigKey                  = "github_pr_comment"
	GitHubPRCommentConfigRepositoryKey        = "repository"
	GitHubPRCommentConfigPullRequestNumberKey = "pull_request_number"
	GitHubPRCommentConfigCommitSHAKey         = "commit_sha"
	GitHubPRCommentConfigPipelineIDKey        = "pipeline_id"
	GitHubPRCommentConfigPipelineURLKey       = "pipeline_url"
	GitHubPRCommentConfigAppURLKey            = "app_url"
	GitHubPRCommentConfigSectionTitleKey      = "section_title"
)

func reportGitHubPRCommentDone(
	ctx workflow.Context,
	logger log.Logger,
	config map[string]any,
	workflowID string,
	runID string,
	workflowResult string,
) {
	commentConfig := workflowengine.AsMap(config[GitHubPRCommentConfigKey])
	if len(commentConfig) == 0 {
		return
	}

	repository := workflowengine.AsString(commentConfig[GitHubPRCommentConfigRepositoryKey])
	prNumber := intFromWorkflowConfig(commentConfig[GitHubPRCommentConfigPullRequestNumberKey])
	if strings.TrimSpace(repository) == "" || prNumber <= 0 {
		return
	}

	updateActivity := activities.NewUpdateGitHubPRCommentActivity()
	payload := activities.UpdateGitHubPRCommentInput{
		Repository:        repository,
		PullRequestNumber: prNumber,
		CommitSHA:         workflowengine.AsString(commentConfig[GitHubPRCommentConfigCommitSHAKey]),
		Status:            "running",
		PipelineID:        workflowengine.AsString(commentConfig[GitHubPRCommentConfigPipelineIDKey]),
		PipelineURL:       workflowengine.AsString(commentConfig[GitHubPRCommentConfigPipelineURLKey]),
		AppURL:            workflowengine.AsString(commentConfig[GitHubPRCommentConfigAppURLKey]),
		WorkflowID:        workflowID,
		RunID:             runID,
		WorkflowStatus:    workflowResult,
		SectionTitle:      workflowengine.AsString(commentConfig[GitHubPRCommentConfigSectionTitleKey]),
	}

	finalCtx, _ := workflow.NewDisconnectedContext(ctx)
	if err := workflow.ExecuteActivity(
		finalCtx,
		updateActivity.Name(),
		workflowengine.ActivityInput{Payload: payload},
	).Get(finalCtx, nil); err != nil {
		logger.Error(
			"failed to update github pr comment",
			"workflow_id",
			workflowID,
			"run_id",
			runID,
			"error",
			err,
		)
	}
}

func intFromWorkflowConfig(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return parsed
		}
	}
	return 0
}
