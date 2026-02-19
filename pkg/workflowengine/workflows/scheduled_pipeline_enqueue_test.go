// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"context"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

// TestScheduledPipelineEnqueueWorkflowEnqueuesDeterministicTicket verifies deterministic ticket IDs and enqueue input.
func TestScheduledPipelineEnqueueWorkflowEnqueuesDeterministicTicket(t *testing.T) {
	pipelineYAML := `
name: Scheduled Pipeline
steps:
  - id: step-1
    use: mobile-automation
    with:
      runner_id: runner-b
      action_id: action-1
  - id: step-2
    use: mobile-automation
    with:
      runner_id: runner-a
      action_id: action-2
`

	input := workflowengine.WorkflowInput{
		Payload: ScheduledPipelineEnqueueWorkflowInput{
			PipelineIdentifier:  "pipeline-123",
			OwnerNamespace:      "org-1",
			MaxPipelinesInQueue: 3,
		},
		Config: map[string]any{
			"app_url": "https://example.test",
		},
	}

	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpAct := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(httpAct.Execute, activity.RegisterOptions{
		Name: httpAct.Name(),
	})

	var capturedPayload activities.EnqueuePipelineRunTicketActivityInput
	env.RegisterActivityWithOptions(
		func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			payload, err := workflowengine.DecodePayload[activities.EnqueuePipelineRunTicketActivityInput](
				input.Payload,
			)
			require.NoError(t, err)
			capturedPayload = payload
			return workflowengine.ActivityResult{Output: map[string]any{"status": "queued"}}, nil
		},
		activity.RegisterOptions{Name: activities.EnqueuePipelineRunTicketActivityName},
	)

	env.OnActivity(httpAct.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{
			Output: map[string]any{
				"body": map[string]any{
					"record": map[string]any{
						"yaml": pipelineYAML,
					},
				},
			},
		}, nil)

	w := NewScheduledPipelineEnqueueWorkflow()
	env.ExecuteWorkflow(w.Workflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	require.Equal(
		t,
		"sched/default-test-workflow-id/default-test-run-id",
		capturedPayload.TicketID,
	)
	require.Equal(t, "org-1", capturedPayload.OwnerNamespace)
	require.Equal(t, "pipeline-123", capturedPayload.PipelineIdentifier)
	require.Equal(t, pipelineYAML, capturedPayload.YAML)
	require.Equal(t, 3, capturedPayload.MaxPipelinesInQueue)
	require.ElementsMatch(t, []string{"runner-a", "runner-b"}, capturedPayload.RunnerIDs)

	config := capturedPayload.PipelineConfig
	require.Equal(t, "org-1", config["namespace"])
	require.Equal(t, "https://example.test", config["app_url"])

	require.Equal(t, "pipeline-run", capturedPayload.Memo["test"])

	require.False(t, capturedPayload.EnqueuedAt.IsZero())
	require.Equal(t, time.UTC, capturedPayload.EnqueuedAt.Location())
}

func TestCollectRunnerIDsAndNeedsGlobal(t *testing.T) {
	steps := []scheduledPipelineStep{
		{
			Use: "mobile-automation",
			With: map[string]any{
				"runner_id": "runner-a",
			},
			OnError: []scheduledPipelineStep{
				{
					Use: "mobile-automation",
				},
			},
		},
	}

	runnerIDs := map[string]struct{}{}
	needsGlobal := false

	collectRunnerIDs(steps, runnerIDs, &needsGlobal)

	_, ok := runnerIDs["runner-a"]
	require.True(t, ok)
	require.True(t, needsGlobal)
}

func TestRunnerIDsWithGlobal(t *testing.T) {
	info := scheduledPipelineRunnerInfo{
		RunnerIDs:         []string{"runner-b"},
		NeedsGlobalRunner: true,
	}

	ids := runnerIDsWithGlobal(info, "runner-a")
	require.Equal(t, []string{"runner-a", "runner-b"}, ids)

	ids = runnerIDsWithGlobal(info, "runner-b")
	require.Equal(t, []string{"runner-b"}, ids)
}
