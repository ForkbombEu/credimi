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
	require.Contains(t, body, "![status: queued](https://img.shields.io/badge/status-queued-yellow)")
	require.Contains(t, body, "| Field | Value |")
	require.Contains(t, body, "| Queue position | `2` |")
	require.Contains(t, body, "| Pipeline | [Open pipeline](https://credimi.test/my/pipelines/org/pipeline) |")
	require.NotContains(t, body, "Credimi wallet APK pipeline is")
	require.NotContains(t, body, "Credimi wallet APK pipeline finished")

	body = buildGitHubPRCommentBody(UpdateGitHubPRCommentInput{
		Status:         "running",
		AppURL:         "https://credimi.test",
		WorkflowID:     "Pipeline-test",
		RunID:          "run-1",
		WorkflowStatus: "WORKFLOW_EXECUTION_STATUS_COMPLETED",
	})
	require.Contains(t, body, "![status: success](https://img.shields.io/badge/status-success-brightgreen)")
	require.Contains(t, body, "| Run logs | [Open run logs](https://credimi.test/my/tests/runs/Pipeline-test/run-1) |")
	require.NotContains(t, body, "Credimi wallet APK pipeline is")
	require.NotContains(t, body, "Credimi wallet APK pipeline finished")
	require.NotContains(t, body, "Result:")

	body = buildGitHubPRCommentBody(UpdateGitHubPRCommentInput{
		Status:         "terminated",
		WorkflowStatus: "failed",
		ErrorMessage:   "emulator | boot timed out",
	})
	require.Contains(t, body, "![status: failed](https://img.shields.io/badge/status-failed-red)")
	require.Contains(t, body, "| Error | emulator \\| boot timed out |")
	require.NotContains(t, body, "Credimi wallet APK pipeline is")
	require.NotContains(t, body, "Credimi wallet APK pipeline finished")
	require.NotContains(t, body, "Result:")
}
