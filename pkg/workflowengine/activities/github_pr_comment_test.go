// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build unit

package activities

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildGitHubPRCommentBody(t *testing.T) {
	position := 2

	body := buildGitHubPRCommentBody(UpdateGitHubPRCommentInput{
		Status:      "queued",
		Position:    &position,
		PipelineURL: "https://credimi.test/my/pipelines/org/pipeline",
	})
	require.Contains(t, body, "Credimi wallet APK pipeline is queued.")
	require.Contains(t, body, "Queue position: 2.")
	require.Contains(t, body, "Pipeline: https://credimi.test/my/pipelines/org/pipeline")

	body = buildGitHubPRCommentBody(UpdateGitHubPRCommentInput{
		Status:         "running",
		AppURL:         "https://credimi.test",
		WorkflowID:     "Pipeline-test",
		RunID:          "run-1",
		WorkflowStatus: "WORKFLOW_EXECUTION_STATUS_COMPLETED",
	})
	require.Contains(t, body, "Credimi wallet APK pipeline is running.")
	require.Contains(t, body, "Run logs: https://credimi.test/my/tests/runs/Pipeline-test/run-1")
	require.Contains(t, body, "Result: success")

	body = buildGitHubPRCommentBody(UpdateGitHubPRCommentInput{
		Status:         "terminated",
		WorkflowStatus: "failed",
	})
	require.Contains(t, body, "Result: failed")
}
