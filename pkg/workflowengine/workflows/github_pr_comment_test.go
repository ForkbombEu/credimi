// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
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

func TestBuildGitHubPRCommentDocumentUsesMarkdownTitle(t *testing.T) {
	document := buildGitHubPRCommentDocument(githubPRCommentWorkflowState{
		Sections: map[string]activities.UpdateGitHubPRCommentInput{
			"abc1234": {Status: "running"},
		},
	})

	require.Contains(t, document, "## Credimi wallet APK pipeline runs")
	require.Contains(t, document, "### `abc1234`")
	require.NotContains(t, document, "### `abc1234` · Credimi wallet APK")
}
