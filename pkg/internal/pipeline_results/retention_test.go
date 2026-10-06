// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipelineresults

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func newRetentionTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	canonify.RegisterCanonifyHooks(app)
	return app
}

func createSavedRetentionRecord(t *testing.T, app *tests.TestApp) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	record := createPipelineResultRecord(t, app, coll)
	require.NoError(t, app.Save(record))
	return record
}

func TestDeleteFilesOlderThanDryRun(t *testing.T) {
	app := newRetentionTestApp(t)

	oldRecord := createSavedRetentionRecord(t, app)
	setPipelineResultFiles(
		t,
		app,
		oldRecord.Id,
		[]string{"old-video.mp4"},
		[]string{"old-shot.png"},
		[]string{"old-log.zip"},
	)
	setPipelineResultEvidence(t, app, oldRecord.Id)
	setPipelineResultCreatedAt(t, app, oldRecord.Id, time.Now().UTC().AddDate(0, 0, -40))

	newRecord := createSavedRetentionRecord(t, app)
	setPipelineResultFiles(t, app, newRecord.Id, []string{"new-video.mp4"}, nil, nil)
	setPipelineResultCreatedAt(t, app, newRecord.Id, time.Now().UTC().AddDate(0, 0, -5))

	result, err := DeleteFilesOlderThan(app, DeleteFilesOptions{OlderThanDays: 30, DryRun: true})
	require.NoError(t, err)
	require.True(t, result.DryRun)
	require.Equal(t, defaultRetentionBatchSize, result.BatchSize)
	require.Equal(t, "created", result.CutoffField)
	require.Equal(t, 2, result.TotalRecords)
	require.Equal(t, 1, result.MatchedRecords)
	require.Equal(t, 1, result.RecordsWithFiles)
	require.Equal(t, 0, result.UpdatedRecords)
	require.Equal(t, 3, result.DeletedFiles.Total)

	reloadedOld, err := app.FindRecordById("pipeline_results", oldRecord.Id)
	require.NoError(t, err)
	require.Equal(t, []string{"old-video.mp4"}, reloadedOld.GetStringSlice("video_results"))
	require.Equal(t, []string{"old-shot.png"}, reloadedOld.GetStringSlice("screenshots"))
	require.Equal(t, []string{"old-log.zip"}, reloadedOld.GetStringSlice("logcats"))
	requirePipelineResultEvidence(t, reloadedOld, true)

	reloadedNew, err := app.FindRecordById("pipeline_results", newRecord.Id)
	require.NoError(t, err)
	require.Equal(t, []string{"new-video.mp4"}, reloadedNew.GetStringSlice("video_results"))
}

func TestDeleteFilesOlderThanClearsOldFiles(t *testing.T) {
	app := newRetentionTestApp(t)

	oldRecord := createSavedRetentionRecord(t, app)
	setPipelineResultFiles(
		t,
		app,
		oldRecord.Id,
		[]string{"old-video.mp4"},
		[]string{"old-shot.png"},
		nil,
	)
	setPipelineResultFileField(t, app, oldRecord.Id, "ios_logstreams", []string{"old-ios-log.zip"})
	setPipelineResultEvidence(t, app, oldRecord.Id)
	setPipelineResultCreatedAt(t, app, oldRecord.Id, time.Now().UTC().AddDate(0, 0, -35))

	result, err := DeleteFilesOlderThan(app, DeleteFilesOptions{OlderThanDays: 30, BatchSize: 1})
	require.NoError(t, err)
	require.False(t, result.DryRun)
	require.Equal(t, 1, result.BatchSize)
	require.Equal(t, 1, result.TotalRecords)
	require.Equal(t, 1, result.MatchedRecords)
	require.Equal(t, 1, result.UpdatedRecords)
	require.Equal(t, 3, result.DeletedFiles.Total)

	reloaded, err := app.FindRecordById("pipeline_results", oldRecord.Id)
	require.NoError(t, err)
	require.Empty(t, reloaded.GetStringSlice("video_results"))
	require.Empty(t, reloaded.GetStringSlice("screenshots"))
	require.Empty(t, reloaded.GetStringSlice("logcats"))
	require.Empty(t, reloaded.GetStringSlice("ios_logstreams"))
	requirePipelineResultEvidence(t, reloaded, false)
}

func TestDeleteFilesOlderThanClearsReport(t *testing.T) {
	app := newRetentionTestApp(t)

	oldRecord := createSavedRetentionRecord(t, app)
	setPipelineResultReport(t, app, oldRecord.Id, "workflow-1.md")
	setPipelineResultFileField(
		t,
		app,
		oldRecord.Id,
		"fcaf_report",
		[]string{"fcaf-assessment.json"},
	)
	setPipelineResultFileField(
		t,
		app,
		oldRecord.Id,
		"fcaf_report_pdf",
		[]string{"fcaf-assessment.pdf"},
	)
	setPipelineResultCreatedAt(t, app, oldRecord.Id, time.Now().UTC().AddDate(0, 0, -35))

	result, err := DeleteFilesOlderThan(app, DeleteFilesOptions{OlderThanDays: 30, BatchSize: 10})
	require.NoError(t, err)
	require.Equal(t, 1, result.UpdatedRecords)
	require.Equal(t, 1, result.DeletedFiles.Report)
	require.Equal(t, 1, result.DeletedFiles.FCAFReport)
	require.Equal(t, 1, result.DeletedFiles.FCAFReportPDF)
	require.Equal(t, 3, result.DeletedFiles.Total)

	reloaded, err := app.FindRecordById("pipeline_results", oldRecord.Id)
	require.NoError(t, err)
	require.Empty(t, reloaded.GetStringSlice("report"))
	require.Empty(t, reloaded.GetStringSlice("fcaf_report"))
	require.Empty(t, reloaded.GetStringSlice("fcaf_report_pdf"))
}

func TestRetentionEvidenceHelpers(t *testing.T) {
	app := newRetentionTestApp(t)

	record := createSavedRetentionRecord(t, app)
	setPipelineResultFiles(
		t,
		app,
		record.Id,
		[]string{"video.mp4"},
		[]string{"screenshot.png"},
		[]string{"log.zip"},
	)
	setPipelineResultFileField(t, app, record.Id, "ios_logstreams", []string{"ios-log.zip"})
	setPipelineResultEvidence(t, app, record.Id)

	reloaded, err := app.FindRecordById("pipeline_results", record.Id)
	require.NoError(t, err)
	reloaded.Set("maestro_screenshots", []string{"maestro-1.png", "maestro-2.png"})
	require.True(t, hasEvidence(reloaded))
	require.False(t, hasEvidence(nil))

	clearFiles(reloaded)
	require.Empty(t, reloaded.GetStringSlice("video_results"))
	require.Empty(t, reloaded.GetStringSlice("screenshots"))
	require.Empty(t, reloaded.GetStringSlice("maestro_screenshots"))
	require.Empty(t, reloaded.GetStringSlice("logcats"))
	require.Empty(t, reloaded.GetStringSlice("ios_logstreams"))
	requirePipelineResultEvidence(t, reloaded, false)
}

func TestCountFilesIncludesMaestroScreenshots(t *testing.T) {
	app := newRetentionTestApp(t)
	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := createPipelineResultRecord(t, app, coll)
	record.Set("video_results", []string{"video.mp4"})
	record.Set("screenshots", []string{"final.png"})
	record.Set("maestro_screenshots", []string{"step-1.png", "step-2.png"})

	counts := countFiles(record)
	require.Equal(t, 1, counts.VideoResults)
	require.Equal(t, 1, counts.Screenshots)
	require.Equal(t, 2, counts.MaestroScreenshots)
	require.Equal(t, 4, counts.Total)
	require.Equal(t, FileCounts{}, countFiles(nil))
}

func setPipelineResultCreatedAt(
	t testing.TB,
	app *tests.TestApp,
	recordID string,
	createdAt time.Time,
) {
	t.Helper()

	_, err := app.DB().NewQuery(
		`UPDATE pipeline_results SET created = {:created}, updated = {:updated} WHERE id = {:id}`,
	).Bind(dbx.Params{
		"created": createdAt.UTC(),
		"updated": createdAt.UTC(),
		"id":      recordID,
	}).Execute()
	require.NoError(t, err)
}

// setPipelineResultFileField writes file names straight into the table,
// bypassing the file upload pipeline.
func setPipelineResultFileField(
	t testing.TB,
	app *tests.TestApp,
	recordID string,
	field string,
	values []string,
) {
	t.Helper()

	_, err := app.DB().Update(
		"pipeline_results",
		dbx.Params{field: mustMarshalJSONStringArray(t, values)},
		dbx.HashExp{"id": recordID},
	).Execute()
	require.NoError(t, err)
}

func setPipelineResultEvidence(t testing.TB, app *tests.TestApp, recordID string) {
	t.Helper()

	_, err := app.DB().NewQuery(
		`UPDATE pipeline_results
		SET credential_well_knowns = {:credential_well_knowns},
		    presentation_results = {:presentation_results}
		WHERE id = {:id}`,
	).Bind(dbx.Params{
		"credential_well_knowns": `[{"credential_id":"credential-1"}]`,
		"presentation_results":   `[{"use_case_id":"use-case-1"}]`,
		"id":                     recordID,
	}).Execute()
	require.NoError(t, err)
}

func requirePipelineResultEvidence(t testing.TB, record *core.Record, wantPresent bool) {
	t.Helper()

	var credentialWellKnowns []map[string]any
	var presentationResults []map[string]any
	require.NoError(t, record.UnmarshalJSONField("credential_well_knowns", &credentialWellKnowns))
	require.NoError(t, record.UnmarshalJSONField("presentation_results", &presentationResults))
	if wantPresent {
		require.NotEmpty(t, credentialWellKnowns)
		require.NotEmpty(t, presentationResults)
		return
	}
	require.Empty(t, credentialWellKnowns)
	require.Empty(t, presentationResults)
}
