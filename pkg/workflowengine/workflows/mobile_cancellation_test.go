//go:build credimi_extra

// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestMobileAutomationWorkflowPropagatesActivityCancellation(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileAutomationWorkflow()
	env.RegisterWorkflowWithOptions(
		w.Workflow,
		workflow.RegisterOptions{Name: w.Name()},
	)

	mobileActivity := activities.NewRunMobileFlowActivity()
	env.RegisterActivityWithOptions(
		mobileActivity.Execute,
		activity.RegisterOptions{Name: mobileActivity.Name()},
	)

	env.OnActivity(
		mobileActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{}, temporal.NewCanceledError("mobile flow canceled"))

	env.ExecuteWorkflow(
		w.Name(),
		workflowengine.WorkflowInput{
			Payload: MobileAutomationWorkflowPayload{
				Serial:     "emulator-5554",
				ActionCode: "steps: []",
			},
			Config: map[string]any{
				"app_url": "https://example.test",
			},
			ActivityOptions: &workflow.ActivityOptions{
				StartToCloseTimeout: time.Second,
			},
		},
	)

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.True(t, temporal.IsCanceledError(err))
}

func TestMobileActivityOptionsUseSingleAttemptHeartbeat(t *testing.T) {
	options := mobileActivityOptions(
		&workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
			RetryPolicy: &temporal.RetryPolicy{
				MaximumAttempts: 5,
			},
		},
		"runner-1-TaskQueue",
	)

	require.Equal(t, time.Minute, options.StartToCloseTimeout)
	require.Equal(t, 30*time.Second, options.HeartbeatTimeout)
	require.Equal(t, "runner-1-TaskQueue", options.TaskQueue)
	require.NotNil(t, options.RetryPolicy)
	require.Equal(t, int32(1), options.RetryPolicy.MaximumAttempts)
}
