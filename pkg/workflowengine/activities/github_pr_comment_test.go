// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build unit

package activities

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestGitHubPRCommentActivitiesRejectInvalidPayload(t *testing.T) {
	updateActivity := NewUpdateGitHubPRCommentActivity()
	require.Equal(t, "Update GitHub PR comment", updateActivity.Name())

	_, err := updateActivity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: UpdateGitHubPRCommentInput{},
	})
	require.Error(t, err)

	patchActivity := NewPatchGitHubPRCommentActivity()
	require.Equal(t, "Patch GitHub PR comment", patchActivity.Name())

	_, err = patchActivity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: "not a patch payload",
	})
	require.Error(t, err)
}

func TestGitHubPRCommentWorkflowHelpers(t *testing.T) {
	require.Equal(
		t,
		"github-pr-comment/acme/wallet/7",
		GitHubPRCommentWorkflowID(" /acme/wallet/ ", 7),
	)

	body := BuildGitHubPRCommentBodyForWorkflow(UpdateGitHubPRCommentInput{})
	require.Contains(t, body, "status-queued-yellow")
}

func TestBuildGitHubPRCommentBody(t *testing.T) {
	position := 2

	body := buildGitHubPRCommentBody(UpdateGitHubPRCommentInput{
		Status:     "queued",
		Position:   &position,
		PipelineID: "org/pipeline",
		RunnerID:   "org/runner-1",
		RunnerType: "android_phone",
	})
	require.Contains(t, body, "![status: queued](https://img.shields.io/badge/status-queued-yellow)")
	require.Contains(t, body, "| Field | Value |")
	require.Contains(t, body, "| Queue position | `2` |")
	require.Contains(t, body, "| Pipeline ID | `org/pipeline` |")
	require.Contains(t, body, "| Runner | `org/runner-1(android_phone)` |")
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
