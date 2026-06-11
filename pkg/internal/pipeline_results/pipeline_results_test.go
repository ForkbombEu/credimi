// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipelineresults

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data/"

func TestHandlePipelineResultsEnrichSetsArtifacts(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	RegisterPipelineResultsHooks(app)
	app.Settings().Meta.AppURL = "https://app.test"
	ensurePipelineResultReportField(t, app)

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := createPipelineResultRecord(t, app, coll)
	require.NoError(t, app.Save(record))
	setPipelineResultFiles(
		t,
		app,
		record.Id,
		[]string{"abc_result_video_main.mp4"},
		[]string{"abc_screenshot_main.png"},
		[]string{"abc_logfile_main.zip"},
	)
	setPipelineResultReport(t, app, record.Id, "run_report.md")

	record, err = app.FindRecordById(coll.Id, record.Id)
	require.NoError(t, err)

	event := &core.RecordEnrichEvent{App: app}
	event.Record = record
	require.NoError(t, app.OnRecordEnrich().Trigger(event, func(e *core.RecordEnrichEvent) error {
		return e.Next()
	}))

	var artifacts map[string]any
	switch got := record.Get("artifacts").(type) {
	case map[string]any:
		artifacts = got
	case PipelineExecutionArtifacts:
		raw, err := json.Marshal(got)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &artifacts))
	default:
		require.Failf(t, "artifacts field missing", "got type %T", record.Get("artifacts"))
	}

	results, ok := artifacts["results"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, results)
	require.Contains(t, artifacts["report"], "run_report.md")
}

func ensurePipelineResultReportField(t testing.TB, app *tests.TestApp) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	if collection.Fields.GetByName("report") == nil {
		collection.Fields.Add(&core.FileField{Name: "report", MaxSelect: 1})
	}
	require.NoError(t, app.Save(collection))
}

func createPipelineResultRecord(
	t testing.TB,
	app *tests.TestApp,
	coll *core.Collection,
) *core.Record {
	t.Helper()

	orgRecord, err := app.FindFirstRecordByFilter(
		"organizations",
		"name = {:name}",
		dbx.Params{"name": "userA's organization"},
	)
	require.NoError(t, err)

	pipelineColl, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelineColl)
	pipelineRecord.Set("owner", orgRecord.Id)
	pipelineName := "pipeline-artifacts-" + testRandString()
	pipelineRecord.Set("name", pipelineName)
	pipelineRecord.Set("canonified_name", pipelineName)
	pipelineRecord.Set("description", "artifacts enrich test")
	pipelineRecord.Set("yaml", "name: t\nsteps: []")
	require.NoError(t, app.Save(pipelineRecord))

	record := core.NewRecord(coll)
	record.Set("owner", orgRecord.Id)
	record.Set("pipeline", pipelineRecord.Id)
	record.Set("workflow_id", "wf-"+testRandString())
	record.Set("run_id", "run-"+testRandString())

	return record
}

func setPipelineResultFiles(
	t testing.TB,
	app *tests.TestApp,
	recordID string,
	videoResults []string,
	screenshots []string,
	logcats []string,
) {
	t.Helper()

	_, err := app.DB().NewQuery(
		`UPDATE pipeline_results
		SET video_results = {:video_results},
		    screenshots = {:screenshots},
		    logcats = {:logcats}
		WHERE id = {:id}`,
	).Bind(dbx.Params{
		"video_results": mustMarshalJSONStringArray(t, videoResults),
		"screenshots":   mustMarshalJSONStringArray(t, screenshots),
		"logcats":       mustMarshalJSONStringArray(t, logcats),
		"id":            recordID,
	}).Execute()
	require.NoError(t, err)
}

func setPipelineResultReport(t testing.TB, app *tests.TestApp, recordID string, report string) {
	t.Helper()

	_, err := app.DB().NewQuery(
		`UPDATE pipeline_results
		SET report = {:report}
		WHERE id = {:id}`,
	).Bind(dbx.Params{
		"report": mustMarshalJSONStringArray(t, []string{report}),
		"id":     recordID,
	}).Execute()
	require.NoError(t, err)
}

func mustMarshalJSONStringArray(t testing.TB, values []string) string {
	t.Helper()

	if values == nil {
		values = []string{}
	}

	data, err := json.Marshal(values)
	require.NoError(t, err)

	return string(data)
}

func testRandString() string {
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}
