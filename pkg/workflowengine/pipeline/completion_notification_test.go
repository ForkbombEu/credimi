// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestPipelineReportsCompletionNotification(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	notificationActivity := activities.NewSendPipelineCompletionNotificationActivity()
	env.RegisterActivityWithOptions(
		notificationActivity.Execute,
		activity.RegisterOptions{Name: notificationActivity.Name()},
	)

	originalSetupHooks := setupHooks
	originalCleanupHooks := cleanupHooks
	setupHooks = []SetupFunc{}
	cleanupHooks = []CleanupFunc{}
	t.Cleanup(func() {
		setupHooks = originalSetupHooks
		cleanupHooks = originalCleanupHooks
	})

	var notified bool
	env.OnActivity(
		notificationActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(activities.SendPipelineCompletionNotificationInput)
			if !ok {
				decoded, err := workflowengine.
					DecodePayload[activities.SendPipelineCompletionNotificationInput](
					input.Payload,
				)
				if err != nil {
					return false
				}
				payload = decoded
			}
			return payload.AppURL == "https://example.test" &&
				payload.WorkflowID == "default-test-workflow-id" &&
				payload.RunID == "default-test-run-id" &&
				payload.Result == resultSuccess
		}),
	).
		Run(func(_ mock.Arguments) {
			notified = true
		}).
		Return(workflowengine.ActivityResult{}, nil).
		Once()

	env.ExecuteWorkflow(pipelineWf.Name(), PipelineWorkflowInput{
		WorkflowDefinition: &pipeline.WorkflowDefinition{
			Name:  "test-pipeline",
			Steps: []pipeline.StepDefinition{},
		},
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url":                          "https://example.test",
				"pipeline_completion_notification": true,
			},
			ActivityOptions: &workflow.ActivityOptions{StartToCloseTimeout: time.Second},
		},
	})

	require.NoError(t, env.GetWorkflowError())
	require.True(t, notified, "completion notification activity should have been invoked")
	env.AssertExpectations(t)
}

func TestPipelineSkipsCompletionNotificationWithoutConfigKey(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	notificationActivity := activities.NewSendPipelineCompletionNotificationActivity()
	env.RegisterActivityWithOptions(
		notificationActivity.Execute,
		activity.RegisterOptions{Name: notificationActivity.Name()},
	)

	originalSetupHooks := setupHooks
	originalCleanupHooks := cleanupHooks
	setupHooks = []SetupFunc{}
	cleanupHooks = []CleanupFunc{}
	t.Cleanup(func() {
		setupHooks = originalSetupHooks
		cleanupHooks = originalCleanupHooks
	})

	// No OnActivity mock: any invocation of the notification activity would
	// panic the test environment, which is exactly the failure we want caught.
	env.ExecuteWorkflow(pipelineWf.Name(), PipelineWorkflowInput{
		WorkflowDefinition: &pipeline.WorkflowDefinition{
			Name:  "test-pipeline",
			Steps: []pipeline.StepDefinition{},
		},
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url": "https://example.test",
			},
			ActivityOptions: &workflow.ActivityOptions{StartToCloseTimeout: time.Second},
		},
	})

	require.NoError(t, env.GetWorkflowError())
}

func TestTruncateWorkflowErrorKeepsRunesIntact(t *testing.T) {
	t.Parallel()

	// 100 "€" runes = 300 bytes, plus a suffix that pushes past the limit.
	long := strings.Repeat("€", 100) + "trailing text beyond the limit"
	got := truncateWorkflowError(long)

	require.LessOrEqual(t, len([]rune(got)), 141)
	require.True(t, utf8.ValidString(got), "truncated message must stay valid UTF-8")
	require.Contains(t, got, "€")
}
