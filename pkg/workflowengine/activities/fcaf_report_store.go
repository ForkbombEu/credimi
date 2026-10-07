// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"

	"github.com/forkbombeu/credimi/pkg/fcaf/reportgeneration"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

var (
	// errFCAFReportEncode reports a failure to encode the FCAF report as JSON.
	errFCAFReportEncode = errors.New("encode report")
	// errFCAFReportEnrich reports a failure to decode the FCAF report JSON and
	// enrich it with its presentation.
	errFCAFReportEnrich = errors.New("enrich FCAF report")
)

// fcafReportStoreError maps a missing pipeline result to a non-retryable
// CRE233, a report encode or enrichment failure to a non-retryable CRE203 or
// CRE225 and any other (database) failure to a retryable CRE235.
func fcafReportStoreError(a *workflowengine.BaseActivity, err error) error {
	err = fmt.Errorf("store FCAF report: %w", err)
	switch {
	case errors.Is(err, pipelineresults.ErrNotFound), errors.Is(err, sql.ErrNoRows):
		return a.NewCodedError(errorcodes.RecordNotFound, false, err)
	case errors.Is(err, errFCAFReportEncode):
		return a.NewCodedError(errorcodes.JSONMarshalFailed, false, err)
	case errors.Is(err, errFCAFReportEnrich):
		return a.NewCodedError(errorcodes.DecodeFailed, false, err)
	default:
		return a.NewCodedError(errorcodes.DatabaseOperationFailed, true, err)
	}
}

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
		return "", fmt.Errorf("%w: %w", errFCAFReportEnrich, err)
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
