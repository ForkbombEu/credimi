// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/forkbombeu/credimi/pkg/fcaf/engine"
	"github.com/forkbombeu/credimi/pkg/fcaf/reportgeneration"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/spf13/cobra"
)

type fcafPresentationBackfillOptions struct {
	skipPDF bool
	dryRun  bool
}

type fcafPresentationBackfillSummary struct {
	scanned             int
	updated             int
	skipped             int
	scoreboardRefreshed int
	errors              int
}

func newFCAFBackfillPresentationCommand(app core.App) *cobra.Command {
	options := fcafPresentationBackfillOptions{}
	cmd := &cobra.Command{
		Use:   "backfill-presentation",
		Short: "Backfill presentation data in historical FCAF reports",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			summary, err := backfillFCAFPresentations(
				cmd.Context(),
				app,
				options,
				cmd.OutOrStdout(),
				cmd.ErrOrStderr(),
			)
			suffix := ""
			if options.dryRun {
				suffix = " (dry run)"
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"summary%s: scanned=%d updated=%d skipped=%d scoreboard_refreshed=%d errors=%d\n",
				suffix,
				summary.scanned,
				summary.updated,
				summary.skipped,
				summary.scoreboardRefreshed,
				summary.errors,
			)
			return err
		},
	}
	cmd.Flags().BoolVar(&options.skipPDF, "skip-pdf", false, "skip FCAF PDF regeneration")
	cmd.Flags().BoolVar(&options.dryRun, "dry-run", false, "report changes without writing")
	return cmd
}

func backfillFCAFPresentations(
	ctx context.Context,
	app core.App,
	options fcafPresentationBackfillOptions,
	out io.Writer,
	errOut io.Writer,
) (fcafPresentationBackfillSummary, error) {
	summary := fcafPresentationBackfillSummary{}
	if app == nil {
		summary.errors++
		return summary, fmt.Errorf("backfill FCAF presentations: app is required")
	}

	records, err := app.FindRecordsByFilter(
		"pipeline_results",
		"fcaf_report != ''",
		"created,id",
		-1,
		0,
	)
	if err != nil {
		summary.errors++
		return summary, fmt.Errorf("find pipeline results with FCAF reports: %w", err)
	}

	fileSystem, err := app.NewFilesystem()
	if err != nil {
		summary.errors++
		return summary, fmt.Errorf("open PocketBase filesystem: %w", err)
	}
	defer fileSystem.Close()

	for _, record := range records {
		summary.scanned++
		filename := record.GetString("fcaf_report")
		rawJSON, err := readFCAFReport(fileSystem, record, filename)
		if err != nil {
			recordBackfillError(&summary, errOut, record, err)
			continue
		}

		var report engine.Report
		if err := json.Unmarshal(rawJSON, &report); err != nil {
			recordBackfillError(
				&summary,
				errOut,
				record,
				fmt.Errorf("decode FCAF report: %w", err),
			)
			continue
		}
		needsPresentation := fcafReportNeedsPresentationBackfill(report)
		if !needsPresentation {
			summary.skipped++
			if !options.dryRun {
				_ = refreshScoreboardArtifactsAfterBackfill(&summary, errOut, app, record)
			}
			continue
		}
		if options.dryRun {
			summary.updated++
			fmt.Fprintf(out, "would update pipeline_results/%s\n", record.Id)
			continue
		}

		enrichedJSON, _, err := reportgeneration.EnrichReportJSON(app, record, rawJSON)
		if err != nil {
			recordBackfillError(
				&summary,
				errOut,
				record,
				fmt.Errorf("enrich FCAF report: %w", err),
			)
			continue
		}

		reportFile, err := filesystem.NewFileFromBytes(enrichedJSON, "fcaf-assessment.json")
		if err != nil {
			recordBackfillError(
				&summary,
				errOut,
				record,
				fmt.Errorf("prepare FCAF report file: %w", err),
			)
			continue
		}

		var pdfFile *filesystem.File
		if !options.skipPDF {
			pdfData, err := reportgeneration.GeneratePipelineFCAFReportPDF(
				ctx,
				app,
				record,
				enrichedJSON,
			)
			if err != nil {
				recordBackfillError(
					&summary,
					errOut,
					record,
					fmt.Errorf("generate FCAF report PDF: %w", err),
				)
				continue
			}
			pdfFile, err = filesystem.NewFileFromBytes(pdfData, "fcaf-assessment.pdf")
			if err != nil {
				recordBackfillError(
					&summary,
					errOut,
					record,
					fmt.Errorf("prepare FCAF report PDF file: %w", err),
				)
				continue
			}
		}

		record.Set("fcaf_report", []*filesystem.File{reportFile})
		if pdfFile != nil {
			record.Set("fcaf_report_pdf", []*filesystem.File{pdfFile})
		}
		if err := app.Save(record); err != nil {
			recordBackfillError(
				&summary,
				errOut,
				record,
				fmt.Errorf("save pipeline result: %w", err),
			)
			continue
		}
		summary.updated++
		fmt.Fprintf(out, "updated pipeline_results/%s\n", record.Id)
		if err := refreshScoreboardArtifactsAfterBackfill(&summary, errOut, app, record); err != nil {
			continue
		}
	}

	if summary.errors > 0 {
		return summary, fmt.Errorf("backfill completed with %d error(s)", summary.errors)
	}
	return summary, nil
}

func readFCAFReport(
	fileSystem *filesystem.System,
	record *core.Record,
	filename string,
) ([]byte, error) {
	reader, err := fileSystem.GetReader(record.BaseFilesPath() + "/" + filename)
	if err != nil {
		return nil, fmt.Errorf("open FCAF report %q: %w", filename, err)
	}
	data, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read FCAF report %q: %w", filename, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close FCAF report %q: %w", filename, closeErr)
	}
	return data, nil
}

func fcafReportNeedsPresentationBackfill(report engine.Report) bool {
	return report.Presentation == nil
}

func refreshScoreboardArtifactsAfterBackfill(
	summary *fcafPresentationBackfillSummary,
	errOut io.Writer,
	app core.App,
	record *core.Record,
) error {
	n, err := reportgeneration.RefreshScoreboardLatestExecutionArtifacts(app, record.Id)
	if err != nil {
		recordBackfillError(
			summary,
			errOut,
			record,
			fmt.Errorf("refresh scoreboard artifacts: %w", err),
		)
		return err
	}
	summary.scoreboardRefreshed += n
	return nil
}

func recordBackfillError(
	summary *fcafPresentationBackfillSummary,
	errOut io.Writer,
	record *core.Record,
	err error,
) {
	summary.errors++
	fmt.Fprintf(errOut, "pipeline_results/%s: %v\n", record.Id, err)
}
