// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/forkbombeu/credimi/pkg/fcaf/reportgeneration"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

const fcafTestPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

func newFCAFReportStoreTestApp(t *testing.T, workflowID, runID string) (core.App, *core.Record) {
	t.Helper()
	app := newPipelineResultsTestApp(t)
	ensureFCAFReportFields(t, app)
	return app, createTestPipelineResult(t, app, workflowID, runID)
}

func attachFCAFTestScreenshot(t *testing.T, app core.App, record *core.Record, name string) string {
	t.Helper()
	imageData, err := base64.StdEncoding.DecodeString(fcafTestPNG)
	require.NoError(t, err)
	imageFile, err := filesystem.NewFileFromBytes(imageData, name)
	require.NoError(t, err)
	record.Set("maestro_screenshots", []*filesystem.File{imageFile})
	require.NoError(t, app.Save(record))
	filenames := record.GetStringSlice("maestro_screenshots")
	require.Len(t, filenames, 1)
	return filenames[0]
}

func TestStoreFCAFReportStoresJSONAndPDF(t *testing.T) {
	app, record := newFCAFReportStoreTestApp(t, "workflow-fcaf", "run-fcaf")
	attachFCAFTestScreenshot(t, app, record, "report-presentation.png")

	report := engine.Report{
		Suite:   "wallet_solution/relying_party",
		Status:  "passed",
		Summary: engine.Summary{Pass: 1},
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
		Presentation: &engine.Presentation{Deeplink: "https://stale.example"},
	}
	reportJSON, err := json.Marshal(report)
	require.NoError(t, err)

	reportSHA256, err := storeFCAFReport(
		t.Context(),
		app,
		"workflow-fcaf",
		"run-fcaf",
		reportJSON,
	)
	require.NoError(t, err)

	reloaded, err := app.FindRecordById("pipeline_results", record.Id)
	require.NoError(t, err)
	require.Len(t, reloaded.GetStringSlice("fcaf_report"), 1)
	require.Len(t, reloaded.GetStringSlice("fcaf_report_pdf"), 1)
	require.Contains(t, reloaded.GetString("fcaf_report"), "fcaf_assessment")
	require.Contains(t, reloaded.GetString("fcaf_report_pdf"), "fcaf_assessment")

	pdf := readPipelineResultFile(t, app, reloaded, "fcaf_report_pdf")
	require.Contains(t, string(pdf), "%PDF-")

	enrichedJSON := readPipelineResultFile(t, app, reloaded, "fcaf_report")
	storedSum := sha256.Sum256(enrichedJSON)
	require.Equal(t, hex.EncodeToString(storedSum[:]), reportSHA256)
	var enrichedReport engine.Report
	require.NoError(t, json.Unmarshal(enrichedJSON, &enrichedReport))
	require.NotNil(t, enrichedReport.Presentation)
	require.Empty(t, enrichedReport.Presentation.Deeplink)
	require.Len(t, enrichedReport.Presentation.Screenshots, 1)
	maestroURLs := pipelineresults.ComputeMaestroScreenshotURLs(app, reloaded)
	require.Equal(t, maestroURLs[0], enrichedReport.Presentation.Screenshots[0].URL)
}

func TestStoreFCAFReportPreservesJSONWhenPDFGenerationFails(t *testing.T) {
	app, record := newFCAFReportStoreTestApp(t, "workflow-fcaf-failure", "run-fcaf-failure")

	reportSHA256, err := storeFCAFReport(
		t.Context(),
		app,
		"workflow-fcaf-failure",
		"run-fcaf-failure",
		[]byte(`{"status":"failed","summary":{}}`),
	)
	require.NoError(t, err)
	require.NotEmpty(t, reportSHA256)

	reloaded, err := app.FindRecordById("pipeline_results", record.Id)
	require.NoError(t, err)
	require.Len(t, reloaded.GetStringSlice("fcaf_report"), 1)
	require.Empty(t, reloaded.GetStringSlice("fcaf_report_pdf"))
}

func TestStoreFCAFReportErrors(t *testing.T) {
	app, _ := newFCAFReportStoreTestApp(t, "workflow-fcaf-errors", "run-fcaf-errors")

	tests := []struct {
		name        string
		workflowID  string
		reportJSON  []byte
		errIs       error
		errContains string
	}{
		{
			name:        "missing pipeline result",
			workflowID:  "workflow-unknown",
			reportJSON:  []byte(`{}`),
			errIs:       pipelineresults.ErrNotFound,
			errContains: "pipeline result not found",
		},
		{
			name:        "invalid report json",
			workflowID:  "workflow-fcaf-errors",
			reportJSON:  []byte(`{`),
			errContains: "enrich FCAF report",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := storeFCAFReport(
				t.Context(),
				app,
				tc.workflowID,
				"run-fcaf-errors",
				tc.reportJSON,
			)
			require.ErrorContains(t, err, tc.errContains)
			if tc.errIs != nil {
				require.ErrorIs(t, err, tc.errIs)
			}
		})
	}
}

func TestFCAFReportStoreError(t *testing.T) {
	cases := []struct {
		name         string
		err          error
		wantCode     string
		nonRetryable bool
	}{
		{
			name:         "missing pipeline result",
			err:          fmt.Errorf("%w: workflow_id wf run_id run", pipelineresults.ErrNotFound),
			wantCode:     errorcodes.RecordNotFound,
			nonRetryable: true,
		},
		{
			name:         "no rows",
			err:          fmt.Errorf("find pipeline result: %w", sql.ErrNoRows),
			wantCode:     errorcodes.RecordNotFound,
			nonRetryable: true,
		},
		{
			name:         "report encode failure",
			err:          fmt.Errorf("%w: unsupported value", errFCAFReportEncode),
			wantCode:     errorcodes.JSONMarshalFailed,
			nonRetryable: true,
		},
		{
			name:         "report enrichment failure",
			err:          fmt.Errorf("%w: decode FCAF report", errFCAFReportEnrich),
			wantCode:     errorcodes.DecodeFailed,
			nonRetryable: true,
		},
		{
			name:     "save failure",
			err:      errors.New("save FCAF report: database is locked"),
			wantCode: errorcodes.DatabaseOperationFailed,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			act := NewFCAFValidationActivity(nil, nil)
			err := fcafReportStoreError(&act.BaseActivity, tc.err)
			requireActivityError(t, err, tc.wantCode, tc.nonRetryable)
			require.ErrorContains(t, err, "store FCAF report")
		})
	}
}

func TestLoadPipelineFCAFReportImages(t *testing.T) {
	app := newPipelineResultsTestApp(t)
	ensureFCAFReportFields(t, app)

	tests := []struct {
		name         string
		screenshot   bool
		wantImages   int
		wantWarnings string
	}{
		{name: "stored screenshot", screenshot: true, wantImages: 1},
		{
			name:         "unstored visual reference",
			wantWarnings: "missing.png was not stored",
		},
	}
	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := createTestPipelineResult(
				t,
				app,
				"workflow-image-"+string(rune('a'+i)),
				"run-image",
			)
			filename := "missing.png"
			if tc.screenshot {
				filename = attachFCAFTestScreenshot(t, app, record, "visual-evidence.png")
			}
			visual := pipelineResultsTestAppURL + "/api/files/pipeline_results/" +
				record.Id + "/" + filename
			report := engine.Report{ExecutedTests: []engine.ExecutedTest{{
				TestID: "test-1",
				Evidence: []engine.ExecutedEvidence{
					{Name: "visual_evidence", Visual: []string{visual}},
				},
			}}}
			images, warnings, err := reportgeneration.LoadPipelineFCAFReportImages(
				app,
				record,
				report,
			)
			require.NoError(t, err)
			require.Len(t, images, tc.wantImages)
			if tc.wantWarnings == "" {
				require.Empty(t, warnings)
				require.Equal(t, filename, images[0].Filename)
				require.NotEmpty(t, images[0].Data)
				return
			}
			require.Len(t, warnings, 1)
			require.Contains(t, warnings[0], tc.wantWarnings)
		})
	}
}
