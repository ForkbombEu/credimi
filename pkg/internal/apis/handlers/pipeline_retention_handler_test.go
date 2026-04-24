// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func setupPipelineRetentionApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	PipelineTemporalInternalRoutes.Add(app)
	seedInternalAdminKey(t, app)

	return app
}

func TestDeletePipelineResultFilesDryRun(t *testing.T) {
	app := setupPipelineRetentionApp(t)
	defer app.Cleanup()

	oldRecord := createPipelineRetentionRecord(t, app)
	require.NoError(t, app.Save(oldRecord))
	setPipelineResultFiles(
		t,
		app,
		oldRecord.Id,
		[]string{"old-video.mp4"},
		[]string{"old-shot.png"},
		[]string{"old-log.zip"},
		nil,
	)
	setPipelineResultCreatedAt(t, app, oldRecord.Id, time.Now().UTC().AddDate(0, 0, -40))

	newRecord := createPipelineRetentionRecord(t, app)
	require.NoError(t, app.Save(newRecord))
	setPipelineResultFiles(t, app, newRecord.Id, []string{"new-video.mp4"}, nil, nil, nil)
	setPipelineResultCreatedAt(t, app, newRecord.Id, time.Now().UTC().AddDate(0, 0, -5))

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/retention/delete-files",
			strings.NewReader(`{"older_than_days":30,"dry_run":true}`),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var response DeletePipelineResultFilesResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		require.True(t, response.DryRun)
		require.Equal(t, 2, response.TotalRecords)
		require.Equal(t, 1, response.MatchedRecords)
		require.Equal(t, 1, response.RecordsWithFiles)
		require.Equal(t, 0, response.UpdatedRecords)
		require.Equal(t, 3, response.DeletedFiles.Total)

		reloadedOld, err := app.FindRecordById("pipeline_results", oldRecord.Id)
		require.NoError(t, err)
		require.Equal(t, []string{"old-video.mp4"}, reloadedOld.GetStringSlice("video_results"))
		require.Equal(t, []string{"old-shot.png"}, reloadedOld.GetStringSlice("screenshots"))
		require.Equal(t, []string{"old-log.zip"}, reloadedOld.GetStringSlice("logcats"))

		reloadedNew, err := app.FindRecordById("pipeline_results", newRecord.Id)
		require.NoError(t, err)
		require.Equal(t, []string{"new-video.mp4"}, reloadedNew.GetStringSlice("video_results"))

		return nil
	})
	require.NoError(t, serveErr)
}

func TestDeletePipelineResultFilesClearsOldFiles(t *testing.T) {
	app := setupPipelineRetentionApp(t)
	defer app.Cleanup()

	oldRecord := createPipelineRetentionRecord(t, app)
	require.NoError(t, app.Save(oldRecord))
	setPipelineResultFiles(
		t,
		app,
		oldRecord.Id,
		[]string{"old-video.mp4"},
		[]string{"old-shot.png"},
		nil,
		[]string{"old-ios-log.zip"},
	)
	setPipelineResultCreatedAt(t, app, oldRecord.Id, time.Now().UTC().AddDate(0, 0, -35))

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/retention/delete-files",
			strings.NewReader(`{"older_than_days":30,"dry_run":false,"batch_size":1}`),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var response DeletePipelineResultFilesResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		require.False(t, response.DryRun)
		require.Equal(t, 1, response.BatchSize)
		require.Equal(t, 1, response.TotalRecords)
		require.Equal(t, 1, response.MatchedRecords)
		require.Equal(t, 1, response.UpdatedRecords)
		require.Equal(t, 3, response.DeletedFiles.Total)

		reloaded, err := app.FindRecordById("pipeline_results", oldRecord.Id)
		require.NoError(t, err)
		require.Empty(t, reloaded.GetStringSlice("video_results"))
		require.Empty(t, reloaded.GetStringSlice("screenshots"))
		require.Empty(t, reloaded.GetStringSlice("logcats"))
		require.Empty(t, reloaded.GetStringSlice("ios_logstreams"))

		return nil
	})
	require.NoError(t, serveErr)
}

func TestDeletePipelineResultFilesValidatesRequest(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "older_than_days must be positive",
		Method:         http.MethodPost,
		URL:            "/api/pipeline/retention/delete-files",
		Body:           strings.NewReader(`{"older_than_days":0}`),
		ExpectedStatus: http.StatusBadRequest,
		ExpectedContent: []string{
			`"Validation failed"`,
		},
		Headers: map[string]string{
			"Content-Type":    "application/json",
			"Credimi-Api-Key": "internal-test-api-key",
		},
		TestAppFactory: setupPipelineRetentionApp,
	}

	scenario.Test(t)
}

func createPipelineRetentionRecord(t testing.TB, app *tests.TestApp) *core.Record {
	t.Helper()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	pipelineColl, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipelineRecord := core.NewRecord(pipelineColl)
	pipelineRecord.Set("owner", orgID)
	pipelineRecord.Set("name", "pipeline-retention-"+testRandString())
	pipelineRecord.Set("description", "retention test")
	pipelineRecord.Set("steps", map[string]any{"rest-chain": map[string]any{"yaml": "name: t\nsteps: []"}})
	pipelineRecord.Set("yaml", "name: t\nsteps: []")
	require.NoError(t, app.Save(pipelineRecord))

	resultColl, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	resultRecord := core.NewRecord(resultColl)
	resultRecord.Set("owner", orgID)
	resultRecord.Set("pipeline", pipelineRecord.Id)
	resultRecord.Set("workflow_id", "wf-"+testRandString())
	resultRecord.Set("run_id", "run-"+testRandString())

	return resultRecord
}

func setPipelineResultCreatedAt(t testing.TB, app *tests.TestApp, recordID string, createdAt time.Time) {
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

func setPipelineResultFiles(
	t testing.TB,
	app *tests.TestApp,
	recordID string,
	videoResults []string,
	screenshots []string,
	logcats []string,
	iosLogstreams []string,
) {
	t.Helper()

	_, err := app.DB().NewQuery(
		`UPDATE pipeline_results
		SET video_results = {:video_results},
		    screenshots = {:screenshots},
		    logcats = {:logcats},
		    ios_logstreams = {:ios_logstreams}
		WHERE id = {:id}`,
	).Bind(dbx.Params{
		"video_results":  mustMarshalJSONStringArray(t, videoResults),
		"screenshots":    mustMarshalJSONStringArray(t, screenshots),
		"logcats":        mustMarshalJSONStringArray(t, logcats),
		"ios_logstreams": mustMarshalJSONStringArray(t, iosLogstreams),
		"id":             recordID,
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
