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

const GitHubPRCommentConfigKey = "github_pr_comment"

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

	repository := workflowengine.AsString(commentConfig["repository"])
	prNumber := intFromWorkflowConfig(commentConfig["pull_request_number"])
	if strings.TrimSpace(repository) == "" || prNumber <= 0 {
		return
	}

	updateActivity := activities.NewUpdateGitHubPRCommentActivity()
	payload := activities.UpdateGitHubPRCommentInput{
		Repository:        repository,
		PullRequestNumber: prNumber,
		CommitSHA:         workflowengine.AsString(commentConfig["commit_sha"]),
		Status:            "running",
		PipelineID:        workflowengine.AsString(commentConfig["pipeline_id"]),
		PipelineURL:       workflowengine.AsString(commentConfig["pipeline_url"]),
		AppURL:            workflowengine.AsString(commentConfig["app_url"]),
		WorkflowID:        workflowID,
		RunID:             runID,
		WorkflowStatus:    workflowResult,
		SectionTitle:      workflowengine.AsString(commentConfig["section_title"]),
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
