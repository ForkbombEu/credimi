// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"os"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	replayPipelineWorkflowID = "Pipeline-fast-only-onboarding-6fa7e966-fe70-4d88-8b45-d7b74c64be5f"
	replayPipelineRunID      = "01a1f32a-1fd7-7da7-9f20-757de20a5f5e"
	replaySessionRunID       = "01a1f32a-2004-7442-be79-0179074b4fe7"
)

// replayHistory replays a recorded history (testdata/replay) of a one-step
// mobile-automation pipeline run, so a change of the workflow's command
// sequence fails here instead of breaking runs that are open during a deploy.
// Workflow IDs of the run's activities derive from its own ID, so the replay
// runs under the recorded execution.
func replayHistory(
	t *testing.T,
	path string,
	register func(worker.WorkflowReplayer),
	execution workflow.Execution,
) {
	t.Helper()
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()
	history, err := client.HistoryFromJSON(file, client.HistoryJSONOptions{})
	require.NoError(t, err)

	replayer, err := worker.NewWorkflowReplayerWithOptions(worker.WorkflowReplayerOptions{
		// The workers' converter; the recorded payloads carry no encrypted secrets.
		DataConverter: temporalcrypto.NewDataConverter(make([]byte, 32)),
	})
	require.NoError(t, err)
	register(replayer)
	require.NoError(t, replayer.ReplayWorkflowHistoryWithOptions(
		nil,
		history,
		worker.ReplayWorkflowHistoryOptions{OriginalExecution: execution},
	))
}

func TestPipelineWorkflowReplaysMainHistory(t *testing.T) {
	replayHistory(
		t,
		"testdata/replay/mobile_pipeline_main.json",
		func(replayer worker.WorkflowReplayer) {
			pipelineWf := NewPipelineWorkflow()
			replayer.RegisterWorkflowWithOptions(
				pipelineWf.Workflow,
				workflow.RegisterOptions{Name: pipelineWf.Name()},
			)
		},
		workflow.Execution{ID: replayPipelineWorkflowID, RunID: replayPipelineRunID},
	)
}

func TestDeviceSessionWorkflowReplaysHistory(t *testing.T) {
	replayHistory(
		t,
		"testdata/replay/mobile_device_session.json",
		func(replayer worker.WorkflowReplayer) {
			sessionWf := NewDeviceSessionWorkflow()
			replayer.RegisterWorkflowWithOptions(
				sessionWf.Workflow,
				workflow.RegisterOptions{Name: sessionWf.Name()},
			)
		},
		workflow.Execution{
			ID:    replayPipelineWorkflowID + "-device-forkbomb-bv-andrea-local-runner-pixel-7a",
			RunID: replaySessionRunID,
		},
	)
}
