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

// TestPipelineWorkflowReplaysMainHistory replays the history of a one-step
// mobile-automation pipeline recorded with origin/main (testdata/replay), so a
// change of the pipeline's command sequence fails here instead of breaking
// pipelines that are running during a deploy.
func TestPipelineWorkflowReplaysMainHistory(t *testing.T) {
	file, err := os.Open("testdata/replay/mobile_pipeline_main.json")
	require.NoError(t, err)
	defer file.Close()
	history, err := client.HistoryFromJSON(file, client.HistoryJSONOptions{})
	require.NoError(t, err)

	replayer, err := worker.NewWorkflowReplayerWithOptions(worker.WorkflowReplayerOptions{
		// The workers' converter; the recorded payloads carry no encrypted secrets.
		DataConverter: temporalcrypto.NewDataConverter(make([]byte, 32)),
	})
	require.NoError(t, err)
	pipelineWf := NewPipelineWorkflow()
	replayer.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	// Step child workflow IDs derive from the pipeline's own ID, so the
	// replay runs under the recorded execution.
	require.NoError(t, replayer.ReplayWorkflowHistoryWithOptions(
		nil,
		history,
		worker.ReplayWorkflowHistoryOptions{
			OriginalExecution: workflow.Execution{
				ID:    "Pipeline-b5-replay-settings-5e02e573-8165-4a50-ad6b-47156f99d143",
				RunID: "01a19bf3-8e59-7960-a4c1-ed64d18554a5",
			},
		},
	))
}
