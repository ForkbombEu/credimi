// SPDX-FileCopyrightText: 2025 Forkbomb BV
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
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestMobileWorkflowQueriesAndSignals(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	wf := NewMobileAutomationWorkflow()
	env.RegisterWorkflow(wf.Workflow)

	runActivity := activities.NewRunMobileFlowActivity()
	env.RegisterActivityWithOptions(runActivity.Execute, activity.RegisterOptions{
		Name: runActivity.Name(),
	})
	env.OnActivity(runActivity.Name(), mock.Anything, mock.Anything).
		After(2*time.Hour).
		Return(workflowengine.ActivityResult{Output: map[string]any{"ok": true}}, nil)

	env.RegisterDelayedCallback(func() {
		encoded, err := env.QueryWorkflow(workflowengine.PipelineStateQuery)
		require.NoError(t, err)
		var state workflowengine.PipelineState
		require.NoError(t, encoded.Get(&state))
		require.Equal(t, "emu-1", state.EmulatorSerial)
		require.Equal(t, "version-1", state.VersionID)
		require.False(t, state.ForceCleanup)

		env.SignalWorkflow(workflowengine.ForceCleanupSignal, struct{}{})
	}, time.Hour)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(workflowengine.PauseRecordingSignal, struct{}{})
	}, time.Hour+30*time.Minute)

	env.RegisterDelayedCallback(func() {
		encoded, err := env.QueryWorkflow(workflowengine.PipelineStateQuery)
		require.NoError(t, err)
		var state workflowengine.PipelineState
		require.NoError(t, encoded.Get(&state))
		require.True(t, state.ForceCleanup)
		require.True(t, state.RecordingPaused)
	}, time.Hour+31*time.Minute)

	env.RegisterDelayedCallback(func() {
		encoded, err := env.QueryWorkflow(workflowengine.ResourceUsageQuery)
		require.NoError(t, err)
		var usage workflowengine.ResourceUsage
		require.NoError(t, encoded.Get(&usage))
	}, time.Hour+45*time.Minute)

	input := workflowengine.WorkflowInput{
		Payload: MobileAutomationWorkflowPayload{
			EmulatorSerial: "emu-1",
			VersionID:      "version-1",
			CloneName:      "clone-1",
			ActionCode:     "- flow",
		},
		Config: map[string]any{
			"app_url": "https://example.test",
		},
		ActivityOptions: &workflow.ActivityOptions{StartToCloseTimeout: time.Minute},
	}

	env.ExecuteWorkflow(wf.Workflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}
