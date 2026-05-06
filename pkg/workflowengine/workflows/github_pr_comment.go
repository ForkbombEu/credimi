// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/githubapp"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

type GitHubPRCommentWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

type githubPRCommentWorkflowState struct {
	Repository        string
	PullRequestNumber int
	CommentID         int64
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
		signalCh.Receive(ctx, &update)
		applyGitHubPRCommentUpdate(&state, update)
		if err := patchGitHubPRComment(ctx, &state); err != nil {
			workflow.GetLogger(ctx).Error("failed to patch github pr comment", "error", err)
		}
	}
}

func applyGitHubPRCommentUpdate(
	state *githubPRCommentWorkflowState,
	update activities.UpdateGitHubPRCommentInput,
) {
	if state.Repository == "" {
		state.Repository = update.Repository
	}
	if state.PullRequestNumber == 0 {
		state.PullRequestNumber = update.PullRequestNumber
	}
	key := githubPRCommentSectionKey(update)
	state.Sections[key] = update
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
		"Credimi wallet APK pipeline runs",
		"",
		githubapp.Marker(),
	)

	for _, key := range keys {
		update := state.Sections[key]
		title := githubPRCommentSectionTitle(update, key)
		lines = append(
			lines,
			"",
			fmt.Sprintf("<!-- credimi-wallet-apk-run:%s:start -->", key),
			fmt.Sprintf("### `%s`", title),
			"",
			activities.BuildGitHubPRCommentBodyForWorkflow(update),
			fmt.Sprintf("<!-- credimi-wallet-apk-run:%s:end -->", key),
		)
	}
	return strings.Join(lines, "\n")
}

func githubPRCommentSectionKey(update activities.UpdateGitHubPRCommentInput) string {
	sha := strings.TrimSpace(update.CommitSHA)
	if sha != "" {
		if len(sha) > 7 {
			return sha[:7]
		}
		return sha
	}
	if update.TicketID != "" {
		return update.TicketID
	}
	return "run"
}

func githubPRCommentSectionTitle(update activities.UpdateGitHubPRCommentInput, key string) string {
	sha := strings.TrimSpace(update.CommitSHA)
	if sha == "" {
		return key
	}
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
