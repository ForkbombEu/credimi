// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/forkbombeu/credimi/pkg/fcaf/reportgeneration"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// storeFCAFReport enriches the FCAF report JSON, stores it on the run's
// pipeline result as fcaf_report together with its PDF rendering and returns
// the sha256 of the stored JSON. A PDF failure keeps the JSON and is logged.
func storeFCAFReport(
	ctx context.Context,
	app core.App,
	workflowID, runID string,
	reportJSON []byte,
) (string, error) {
	record, err := pipelineresults.FindByWorkflowRun(app, workflowID, runID)
	if err != nil {
		return "", err
	}
	enrichedJSON, _, err := reportgeneration.EnrichReportJSON(app, record, reportJSON)
	if err != nil {
		return "", fmt.Errorf("enrich FCAF report: %w", err)
	}
	file, err := filesystem.NewFileFromBytes(enrichedJSON, "fcaf-assessment.json")
	if err != nil {
		return "", fmt.Errorf("create FCAF report file: %w", err)
	}
	record.Set("fcaf_report", []*filesystem.File{file})
	if err := app.Save(record); err != nil {
		return "", fmt.Errorf("save FCAF report: %w", err)
	}
	sum := sha256.Sum256(enrichedJSON)
	reportSHA256 := hex.EncodeToString(sum[:])

	if err := storeFCAFReportPDF(ctx, app, record, enrichedJSON); err != nil {
		app.Logger().Warn(
			"FCAF report PDF not stored",
			slog.String("workflow_id", workflowID),
			slog.String("run_id", runID),
			slog.String("error", err.Error()),
		)
	}
	return reportSHA256, nil
}

func storeFCAFReportPDF(
	ctx context.Context,
	app core.App,
	record *core.Record,
	enrichedJSON []byte,
) error {
	pdfData, err := reportgeneration.GeneratePipelineFCAFReportPDF(ctx, app, record, enrichedJSON)
	if err != nil {
		return fmt.Errorf("generate FCAF report PDF: %w", err)
	}
	pdfFile, err := filesystem.NewFileFromBytes(pdfData, "fcaf-assessment.pdf")
	if err != nil {
		return fmt.Errorf("create FCAF report PDF file: %w", err)
	}
	record.Set("fcaf_report_pdf", []*filesystem.File{pdfFile})
	if err := app.Save(record); err != nil {
		return fmt.Errorf("save FCAF report PDF: %w", err)
	}
	return nil
}
