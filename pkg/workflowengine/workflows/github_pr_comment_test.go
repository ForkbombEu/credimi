// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestGitHubPRCommentWorkflowIdleTimeout(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	w := NewGitHubPRCommentWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func TestBuildGitHubPRCommentDocumentGroupsRunsUnderCommitTitle(t *testing.T) {
	document := buildGitHubPRCommentDocument(githubPRCommentWorkflowState{
		LatestCommitSHA: "abc123456789",
		Sections: map[string]activities.UpdateGitHubPRCommentInput{
			"abc1234::org/pipeline::org/runner-1": {
				CommitSHA:  "abc123456789",
				PipelineID: "org/pipeline",
				RunnerID:   "org/runner-1",
				RunnerType: "android_phone",
				Status:     "running",
			},
			"abc1234::org/pipeline-2::org/runner-2": {
				CommitSHA:  "abc123456789",
				PipelineID: "org/pipeline-2",
				RunnerID:   "org/runner-2",
				RunnerType: "android_emulator",
				Status:     "queued",
			},
		},
	})

	require.Contains(t, document, "## Credimi wallet APK pipeline runs")
	require.Contains(t, document, "### `abc1234`")
	require.Equal(t, 1, strings.Count(document, "### `"))
	require.Contains(t, document, "| Pipeline ID | `org/pipeline` |")
	require.Contains(t, document, "| Runner | `org/runner-1(android_phone)` |")
	require.Contains(t, document, "| Pipeline ID | `org/pipeline-2` |")
	require.Contains(t, document, "| Runner | `org/runner-2(android_emulator)` |")
	require.NotContains(t, document, "### `abc1234 /")
}

func TestApplyGitHubPRCommentUpdateKeepsLatestCommitOnly(t *testing.T) {
	state := githubPRCommentWorkflowState{
		Sections: map[string]activities.UpdateGitHubPRCommentInput{},
	}

	applyGitHubPRCommentUpdate(&state, activities.UpdateGitHubPRCommentInput{
		CommitSHA:  "oldcommit",
		PipelineID: "pipeline-a",
		RunnerID:   "runner-1",
		Status:     "queued",
	})
	applyGitHubPRCommentUpdate(&state, activities.UpdateGitHubPRCommentInput{
		CommitSHA:  "newcommit",
		PipelineID: "pipeline-a",
		RunnerID:   "runner-1",
		Status:     "queued",
	})
	applyGitHubPRCommentUpdate(&state, activities.UpdateGitHubPRCommentInput{
		CommitSHA:  "oldcommit",
		PipelineID: "pipeline-a",
		RunnerID:   "runner-1",
		Status:     "running",
	})
	applyGitHubPRCommentUpdate(&state, activities.UpdateGitHubPRCommentInput{
		CommitSHA:  "newcommit",
		PipelineID: "pipeline-b",
		RunnerID:   "runner-2",
		Status:     "queued",
	})

	require.Equal(t, "newcommit", state.LatestCommitSHA)
	require.Len(t, state.Sections, 2)
	require.Contains(t, state.Sections, "newcomm::pipeline-a::runner-1")
	require.Contains(t, state.Sections, "newcomm::pipeline-b::runner-2")
	require.NotContains(t, state.Sections, "oldcomm::pipeline-a::runner-1")
}

func TestApplyGitHubPRCommentUpdateAcceptsRunningNewCommitAfterTerminalCommit(t *testing.T) {
	state := githubPRCommentWorkflowState{
		Sections: map[string]activities.UpdateGitHubPRCommentInput{},
	}

	applyGitHubPRCommentUpdate(&state, activities.UpdateGitHubPRCommentInput{
		CommitSHA:      "oldcommit",
		PipelineID:     "pipeline-a",
		RunnerID:       "runner-1",
		Status:         "running",
		WorkflowStatus: "WORKFLOW_EXECUTION_STATUS_COMPLETED",
	})
	applyGitHubPRCommentUpdate(&state, activities.UpdateGitHubPRCommentInput{
		CommitSHA:  "newcommit",
		PipelineID: "pipeline-a",
		RunnerID:   "runner-1",
		Status:     "running",
	})

	require.Equal(t, "newcommit", state.LatestCommitSHA)
	require.Len(t, state.Sections, 1)
	require.Contains(t, state.Sections, "newcomm::pipeline-a::runner-1")
	require.NotContains(t, state.Sections, "oldcomm::pipeline-a::runner-1")
}

func TestApplyGitHubPRCommentUpdateIgnoresDifferentRunningCommitWhileCurrentCommitActive(t *testing.T) {
	state := githubPRCommentWorkflowState{
		Sections: map[string]activities.UpdateGitHubPRCommentInput{},
	}

	changed := applyGitHubPRCommentUpdate(&state, activities.UpdateGitHubPRCommentInput{
		CommitSHA:  "current",
		PipelineID: "pipeline-a",
		RunnerID:   "runner-1",
		Status:     "running",
	})
	require.True(t, changed)

	changed = applyGitHubPRCommentUpdate(&state, activities.UpdateGitHubPRCommentInput{
		CommitSHA:  "different",
		PipelineID: "pipeline-a",
		RunnerID:   "runner-1",
		Status:     "running",
	})
	require.False(t, changed)

	require.Equal(t, "current", state.LatestCommitSHA)
	require.Len(t, state.Sections, 1)
	require.Contains(t, state.Sections, "current::pipeline-a::runner-1")
	require.NotContains(t, state.Sections, "differe::pipeline-a::runner-1")
}

func TestApplyGitHubPRCommentUpdateUsesCurrentHeadSHA(t *testing.T) {
	state := githubPRCommentWorkflowState{
		Sections: map[string]activities.UpdateGitHubPRCommentInput{},
	}

	changed := applyGitHubPRCommentUpdate(&state, activities.UpdateGitHubPRCommentInput{
		CommitSHA:      "oldcommit",
		CurrentHeadSHA: "newcommit",
		PipelineID:     "pipeline-a",
		RunnerID:       "runner-1",
		Status:         "queued",
	})
	require.False(t, changed)
	require.Empty(t, state.Sections)

	changed = applyGitHubPRCommentUpdate(&state, activities.UpdateGitHubPRCommentInput{
		CommitSHA:      "newcommit",
		CurrentHeadSHA: "newcommit",
		PipelineID:     "pipeline-a",
		RunnerID:       "runner-1",
		Status:         "running",
	})
	require.True(t, changed)
	require.Equal(t, "newcommit", state.LatestCommitSHA)
	require.Contains(t, state.Sections, "newcomm::pipeline-a::runner-1")
}
