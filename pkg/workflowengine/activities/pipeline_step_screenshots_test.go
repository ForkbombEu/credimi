// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"encoding/base64"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

func TestDeletePipelineStepScreenshotsKeepsOtherFiles(t *testing.T) {
	app := newPipelineResultsTestApp(t)
	record := createTestPipelineResult(t, app, "workflow-shots", "run-shots")
	imageData, err := base64.StdEncoding.DecodeString(fcafTestPNG)
	require.NoError(t, err)
	files := make([]*filesystem.File, 0, 2)
	for _, name := range []string{"attempt-1.png", "attempt-2.png"} {
		file, err := filesystem.NewFileFromBytes(imageData, name)
		require.NoError(t, err)
		files = append(files, file)
	}
	record.Set("maestro_screenshots", files)
	require.NoError(t, app.Save(record))
	stored := record.GetStringSlice("maestro_screenshots")
	require.Len(t, stored, 2)
	runIdentifier, err := canonify.BuildPath(
		app,
		record,
		canonify.CanonifyPaths["pipeline_results"],
		"",
	)
	require.NoError(t, err)

	result, err := executeActivity(t, NewDeletePipelineStepScreenshotsActivity(app),
		DeletePipelineStepScreenshotsInput{
			RunIdentifier: canonify.NormalizePath(runIdentifier),
			FileNames:     []string{stored[0], "not-stored.png"},
		},
	)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"deleted": []any{stored[0]}}, result.Output)

	reloaded, err := app.FindRecordById("pipeline_results", record.Id)
	require.NoError(t, err)
	require.Equal(t, []string{stored[1]}, reloaded.GetStringSlice("maestro_screenshots"))
}

func TestDeletePipelineStepScreenshotsRequiresFiles(t *testing.T) {
	app := newPipelineResultsTestApp(t)
	_, err := executeActivity(t, NewDeletePipelineStepScreenshotsActivity(app),
		DeletePipelineStepScreenshotsInput{RunIdentifier: "org/run"},
	)
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
}
