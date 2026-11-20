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
	replayPipelineWorkflowID = "Pipeline-nexus-one-step-9ec80428-2eeb-4cec-a9b4-8bf6aba3add2"
	replayPipelineRunID      = "01a1f41c-51c1-7dbf-8360-2a3fc34c0ede"
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
