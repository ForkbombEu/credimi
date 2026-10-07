// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/pocketbase/dbx"
	"github.com/stretchr/testify/require"
)

func TestDeletePipelineResultFilesActivityDryRun(t *testing.T) {
	app := newCredimiTestApp(t)

	result := createTestPipelineResult(t, app, "wf-retention", "run-retention")

	_, err := app.DB().Update(
		"pipeline_results",
		dbx.Params{
			"video_results": `["old-video.mp4"]`,
			"created":       time.Now().UTC().AddDate(0, 0, -40),
		},
		dbx.HashExp{"id": result.Id},
	).Execute()
	require.NoError(t, err)

	output, err := executeActivity(
		t,
		NewDeletePipelineResultFilesActivity(app),
		pipelineresults.DeleteFilesOptions{OlderThanDays: 30, DryRun: true},
	)
	require.NoError(t, err)
	counts, ok := output.Output.(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, counts["dry_run"])
	require.InDelta(t, 1, counts["matched_records"], 0)
	require.InDelta(t, 0, counts["updated_records"], 0)
	deleted, ok := counts["deleted_files"].(map[string]any)
	require.True(t, ok)
	require.InDelta(t, 1, deleted["video_results"], 0)

	reloaded, err := app.FindRecordById("pipeline_results", result.Id)
	require.NoError(t, err)
	require.Equal(t, []string{"old-video.mp4"}, reloaded.GetStringSlice("video_results"))
}

func TestDeletePipelineResultFilesActivityRequiresAge(t *testing.T) {
	app := newCredimiTestApp(t)
	_, err := executeActivity(
		t,
		NewDeletePipelineResultFilesActivity(app),
		pipelineresults.DeleteFilesOptions{DryRun: true},
	)
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
}

func TestDeletePipelineResultFilesActivityDatabaseFailure(t *testing.T) {
	app := newCredimiTestApp(t)
	_, err := app.DB().NewQuery("DROP TABLE pipeline_results").Execute()
	require.NoError(t, err)

	_, err = executeActivity(
		t,
		NewDeletePipelineResultFilesActivity(app),
		pipelineresults.DeleteFilesOptions{OlderThanDays: 30, DryRun: true},
	)
	requireActivityError(t, err, errorcodes.DatabaseOperationFailed, false)
}
