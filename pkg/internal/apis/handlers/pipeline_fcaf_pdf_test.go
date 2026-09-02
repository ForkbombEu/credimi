// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

func TestUpdatePipelineExecutionFCAFReportStoresJSONAndPDF(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	record := createFCAFReportPipelineResult(t, app, "workflow-fcaf", "run-fcaf")
	report := engine.Report{
		Suite:  "wallet_solution/relying_party",
		Status: "passed",
		Summary: engine.Summary{
			Pass: 1,
		},
		ExecutedTests: []engine.ExecutedTest{
			{
				TestID: "WS_RP_IA_MainInteraction__003",
				Title:  "Match credentials when DCQL query includes valid trusted authorities.",
				Status: "passed",
				Assertions: []engine.ExecutedCheck{
					{ID: "assertion", Kind: "assertion", Status: "passed"},
				},
				Outcome: engine.TestOutcome{Status: "passed"},
			},
		},
	}
	reportJSON, err := json.Marshal(report)
	require.NoError(t, err)

	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)
	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	require.NoError(t, app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/pipeline-execution-results/fcaf-report",
			jsonBody(map[string]any{
				"workflow_id": "workflow-fcaf",
				"run_id":      "run-fcaf",
				"json":        string(reportJSON),
			}),
		)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		mux.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
		return nil
	}))

	reloaded, err := app.FindRecordById("pipeline_results", record.Id)
	require.NoError(t, err)
	require.Len(t, reloaded.GetStringSlice("fcaf_report"), 1)
	require.Len(t, reloaded.GetStringSlice("fcaf_report_pdf"), 1)
	require.Contains(t, reloaded.GetString("fcaf_report"), "fcaf_assessment")
	require.Contains(t, reloaded.GetString("fcaf_report_pdf"), "fcaf_assessment")

	fileSystem, err := app.NewFilesystem()
	require.NoError(t, err)
	defer fileSystem.Close()
	reader, err := fileSystem.GetFile(
		reloaded.BaseFilesPath() + "/" + reloaded.GetString("fcaf_report_pdf"),
	)
	require.NoError(t, err)
	defer reader.Close()
	pdf, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Contains(t, string(pdf), "%PDF-")
}

func TestLoadPipelineFCAFReportImagesLoadsStoredScreenshots(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	ensureStepScreenshotField(t, app)

	record := createFCAFReportPipelineResult(t, app, "workflow-image", "run-image")
	imageData, err := base64.StdEncoding.DecodeString(
		"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=",
	)
	require.NoError(t, err)
	imageFile, err := filesystem.NewFileFromBytes(imageData, "visual-evidence.png")
	require.NoError(t, err)
	record.Set("maestro_screenshots", []*filesystem.File{imageFile})
	require.NoError(t, app.Save(record))

	filenames := record.GetStringSlice("maestro_screenshots")
	require.Len(t, filenames, 1)
	filename := filenames[0]
	report := engine.Report{ExecutedTests: []engine.ExecutedTest{{
		TestID: "test-1",
		Evidence: []engine.ExecutedEvidence{
			{
				Name: "visual_evidence",
				Visual: []string{
					"https://app.test/api/files/pipeline_results/" + record.Id + "/" + filename,
				},
			},
		},
	}}}
	images, warnings, err := loadPipelineFCAFReportImages(app, record, report)
	require.NoError(t, err)
	require.Empty(t, warnings)
	require.Len(t, images, 1)
	require.Equal(t, filename, images[0].Filename)
	require.NotEmpty(t, images[0].Data)
}

func TestLoadPipelineFCAFReportImagesWarnsForUnstoredVisualReference(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()
	ensureStepScreenshotField(t, app)

	record := createFCAFReportPipelineResult(t, app, "workflow-missing", "run-missing")
	report := engine.Report{ExecutedTests: []engine.ExecutedTest{{
		TestID: "test-1",
		Evidence: []engine.ExecutedEvidence{
			{
				Name: "visual_evidence",
				Visual: []string{
					"https://app.test/api/files/pipeline_results/" + record.Id + "/missing.png",
				},
			},
		},
	}}}
	images, warnings, err := loadPipelineFCAFReportImages(app, record, report)
	require.NoError(t, err)
	require.Empty(t, images)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "missing.png was not stored")
}

func TestUpdatePipelineExecutionFCAFReportPreservesJSONWhenPDFGenerationFails(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	record := createFCAFReportPipelineResult(t, app, "workflow-fcaf-failure", "run-fcaf-failure")
	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)
	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	require.NoError(t, app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		require.NoError(t, err)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/pipeline/pipeline-execution-results/fcaf-report",
			jsonBody(map[string]any{
				"workflow_id": "workflow-fcaf-failure",
				"run_id":      "run-fcaf-failure",
				"json":        `{"status":"failed","summary":{}}`,
			}),
		)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		mux.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusInternalServerError, recorder.Code)
		return nil
	}))

	reloaded, err := app.FindRecordById("pipeline_results", record.Id)
	require.NoError(t, err)
	require.Len(t, reloaded.GetStringSlice("fcaf_report"), 1)
	require.Empty(t, reloaded.GetStringSlice("fcaf_report_pdf"))
}

func createFCAFReportPipelineResult(
	t testing.TB,
	app *tests.TestApp,
	workflowID string,
	runID string,
) *core.Record {
	t.Helper()

	organization, err := app.FindFirstRecordByFilter(
		"organizations",
		"name = {:name}",
		map[string]any{"name": "userA's organization"},
	)
	require.NoError(t, err)
	pipelineRecord := createPipelineRecord(t, app, organization.Id, "FCAF complete validation")
	collection, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	if collection.Fields.GetByName("fcaf_report") == nil {
		collection.Fields.Add(&core.FileField{Name: "fcaf_report", MaxSelect: 1})
	}
	if collection.Fields.GetByName("fcaf_report_pdf") == nil {
		collection.Fields.Add(&core.FileField{Name: "fcaf_report_pdf", MaxSelect: 1})
	}
	require.NoError(t, app.Save(collection))

	record := core.NewRecord(collection)
	record.Set("owner", organization.Id)
	record.Set("pipeline", pipelineRecord.Id)
	record.Set("workflow_id", workflowID)
	record.Set("run_id", runID)
	require.NoError(t, app.Save(record))
	return record
}
