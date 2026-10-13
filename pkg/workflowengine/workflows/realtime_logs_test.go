// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// testWorkflowID is the workflow ID the Temporal test environment assigns by default.
const testWorkflowID = "default-test-workflow-id"

// registerRealtimeLogsActivity registers the SendRealtimeLogs activity; the
// Temporal test environment requires registrations before any mock.
func registerRealtimeLogsActivity(env *testsuite.TestWorkflowEnvironment) {
	env.RegisterActivityWithOptions(
		activities.NewSendRealtimeLogsActivity(nil).Execute,
		activity.RegisterOptions{Name: activities.SendRealtimeLogsActivityName},
	)
}

// onRealtimeLogsActivity mocks SendRealtimeLogs calls targeting subscription,
// invoking onPush for each call.
func onRealtimeLogsActivity(
	env *testsuite.TestWorkflowEnvironment,
	subscription string,
	onPush func(),
) *testsuite.MockCallWrapper {
	return env.OnActivity(
		activities.SendRealtimeLogsActivityName,
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			return realtimeLogsSubscription(input) == subscription
		}),
	).
		Run(func(_ mock.Arguments) {
			onPush()
		}).
		Return(workflowengine.ActivityResult{}, nil)
}

func realtimeLogsSubscription(input workflowengine.ActivityInput) string {
	payload, err := workflowengine.DecodePayload[activities.SendRealtimeLogsInput](input.Payload)
	if err != nil {
		return ""
	}
	return payload.Subscription
}

func TestSendRealtimeLogsUpdateExecutesActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	logs := []map[string]any{{"message": "ok"}}
	registerRealtimeLogsActivity(env)
	env.OnActivity(
		activities.SendRealtimeLogsActivityName,
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, err := workflowengine.DecodePayload[activities.SendRealtimeLogsInput](
				input.Payload,
			)
			return err == nil &&
				payload.Subscription == "wf-1"+EWCSubscription &&
				len(payload.Logs) == 1 &&
				payload.Logs[0]["message"] == "ok"
		}),
	).
		Return(workflowengine.ActivityResult{}, nil).
		Once()

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		ctx = workflow.WithActivityOptions(ctx, DefaultActivityOptions)
		return sendRealtimeLogsUpdate(ctx, "wf-1"+EWCSubscription, logs)
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}
