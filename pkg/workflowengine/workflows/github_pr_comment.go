// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/githubapp"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

const githubPRCommentWorkflowIdleTimeout = 24 * time.Hour

type GitHubPRCommentWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

type githubPRCommentWorkflowState struct {
	Repository        string
	PullRequestNumber int
	CommentID         int64
	LatestCommitSHA   string
	Sections          map[string]activities.UpdateGitHubPRCommentInput
}

func NewGitHubPRCommentWorkflow() *GitHubPRCommentWorkflow {
	w := &GitHubPRCommentWorkflow{}
	w.WorkflowFunc = w.ExecuteWorkflow
	return w
}

func (GitHubPRCommentWorkflow) Name() string {
	return activities.GitHubPRCommentWorkflowName
}

func (GitHubPRCommentWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *GitHubPRCommentWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *GitHubPRCommentWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	state := githubPRCommentWorkflowState{
		Sections: map[string]activities.UpdateGitHubPRCommentInput{},
	}
	signalCh := workflow.GetSignalChannel(ctx, activities.GitHubPRCommentUpdateSignal)

	for {
		var update activities.UpdateGitHubPRCommentInput
		selector := workflow.NewSelector(ctx)
		receivedSignal := false
		timerCtx, cancelTimer := workflow.WithCancel(ctx)
		selector.AddReceive(signalCh, func(ch workflow.ReceiveChannel, more bool) {
			ch.Receive(ctx, &update)
			receivedSignal = true
			cancelTimer()
		})
		selector.AddFuture(workflow.NewTimer(timerCtx, githubPRCommentWorkflowIdleTimeout), func(workflow.Future) {
			receivedSignal = false
		})
		selector.Select(ctx)
		if !receivedSignal {
			return workflowengine.WorkflowResult{}, nil
		}
		if !applyGitHubPRCommentUpdate(&state, update) {
			continue
		}
		if err := patchGitHubPRComment(ctx, &state); err != nil {
			workflow.GetLogger(ctx).Error("failed to patch github pr comment", "error", err)
		}
	}
}

func applyGitHubPRCommentUpdate(
	state *githubPRCommentWorkflowState,
	update activities.UpdateGitHubPRCommentInput,
) bool {
	changed := false
	if state.Repository == "" && strings.TrimSpace(update.Repository) != "" {
		state.Repository = update.Repository
		changed = true
	}
	if state.PullRequestNumber == 0 && update.PullRequestNumber > 0 {
		state.PullRequestNumber = update.PullRequestNumber
		changed = true
	}
	if !applyGitHubPRCommentCommitScope(state, update) {
		return changed
	}
	key := githubPRCommentSectionKey(update)
	state.Sections[key] = update
	return true
}

func applyGitHubPRCommentCommitScope(
	state *githubPRCommentWorkflowState,
	update activities.UpdateGitHubPRCommentInput,
) bool {
	commitSHA := strings.TrimSpace(update.CommitSHA)
	if commitSHA == "" {
		return true
	}
	currentHeadSHA := strings.TrimSpace(update.CurrentHeadSHA)
	if currentHeadSHA != "" {
		if commitSHA != currentHeadSHA {
			return false
		}
		if state.LatestCommitSHA != commitSHA {
			state.LatestCommitSHA = commitSHA
			state.Sections = map[string]activities.UpdateGitHubPRCommentInput{}
		}
		return true
	}
	if state.LatestCommitSHA == "" {
		state.LatestCommitSHA = commitSHA
		return true
	}
	if commitSHA == state.LatestCommitSHA {
		return true
	}
	if !isGitHubPRCommentNewCommitUpdate(update) &&
		!isGitHubPRCommentDisplayedCommitTerminal(*state) {
		return false
	}
	state.LatestCommitSHA = commitSHA
	state.Sections = map[string]activities.UpdateGitHubPRCommentInput{}
	return true
}

func isGitHubPRCommentNewCommitUpdate(update activities.UpdateGitHubPRCommentInput) bool {
	switch strings.ToLower(strings.TrimSpace(update.Status)) {
	case "", "queued", "starting":
		return true
	default:
		return false
	}
}

func isGitHubPRCommentDisplayedCommitTerminal(state githubPRCommentWorkflowState) bool {
	if len(state.Sections) == 0 {
		return false
	}
	for _, update := range state.Sections {
		if !isGitHubPRCommentTerminalUpdate(update) {
			return false
		}
	}
	return true
}

func isGitHubPRCommentTerminalUpdate(update activities.UpdateGitHubPRCommentInput) bool {
	if isTerminalGitHubPRCommentStatus(update.WorkflowStatus) {
		return true
	}
	return isTerminalGitHubPRCommentStatus(update.Status)
}

func isTerminalGitHubPRCommentStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "success",
		"successful",
		"completed",
		"failed",
		"failure",
		"canceled",
		"cancelled",
		"terminated",
		"timed out",
		"timed_out",
		"timeout",
		"workflow_execution_status_completed",
		"workflow_execution_status_failed",
		"workflow_execution_status_canceled",
		"workflow_execution_status_terminated",
		"workflow_execution_status_timed_out":
		return true
	default:
		return false
	}
}

func patchGitHubPRComment(ctx workflow.Context, state *githubPRCommentWorkflowState) error {
	patchActivity := activities.NewPatchGitHubPRCommentActivity()
	activityCtx := workflow.WithActivityOptions(ctx, DefaultActivityOptions)
	var result workflowengine.ActivityResult
	err := workflow.ExecuteActivity(
		activityCtx,
		patchActivity.Name(),
		workflowengine.ActivityInput{
			Payload: activities.PatchGitHubPRCommentInput{
				Repository:        state.Repository,
				PullRequestNumber: state.PullRequestNumber,
				CommentID:         state.CommentID,
				Body:              buildGitHubPRCommentDocument(*state),
			},
		},
	).Get(activityCtx, &result)
	if err != nil {
		return err
	}
	output, err := workflowengine.DecodePayload[activities.PatchGitHubPRCommentOutput](result.Output)
	if err == nil && output.CommentID > 0 {
		state.CommentID = output.CommentID
	}
	return nil
}

func buildGitHubPRCommentDocument(state githubPRCommentWorkflowState) string {
	keys := make([]string, 0, len(state.Sections))
	for key := range state.Sections {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, 3+6*len(keys))
	lines = append(
		lines,
		"## Credimi wallet APK pipeline runs",
		"",
		githubapp.Marker(),
	)

	if title := githubPRCommentDocumentTitle(state, keys); title != "" {
		lines = append(lines, "", fmt.Sprintf("### `%s`", title))
	}

	for _, key := range keys {
		update := state.Sections[key]
		lines = append(
			lines,
			"",
			fmt.Sprintf("<!-- credimi-wallet-apk-run:%s:start -->", key),
			activities.BuildGitHubPRCommentBodyForWorkflow(update),
			fmt.Sprintf("<!-- credimi-wallet-apk-run:%s:end -->", key),
		)
	}
	return strings.Join(lines, "\n")
}

func githubPRCommentSectionKey(update activities.UpdateGitHubPRCommentInput) string {
	parts := make([]string, 0, 3)
	if sha := shortGitHubPRCommentSHA(update.CommitSHA); sha != "" {
		parts = append(parts, sha)
	}
	if pipelineID := strings.TrimSpace(update.PipelineID); pipelineID != "" {
		parts = append(parts, pipelineID)
	}
	if runnerID := strings.TrimSpace(update.RunnerID); runnerID != "" {
		parts = append(parts, runnerID)
	}
	if len(parts) > 0 {
		return strings.Join(parts, "::")
	}
	if update.TicketID != "" {
		return update.TicketID
	}
	return "run"
}

func githubPRCommentDocumentTitle(state githubPRCommentWorkflowState, keys []string) string {
	if sha := shortGitHubPRCommentSHA(state.LatestCommitSHA); sha != "" {
		return sha
	}
	for _, key := range keys {
		if sha := shortGitHubPRCommentSHA(state.Sections[key].CommitSHA); sha != "" {
			return sha
		}
	}
	return ""
}

func shortGitHubPRCommentSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return ""
	}
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
