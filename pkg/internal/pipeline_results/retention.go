// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipelineresults

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const defaultRetentionBatchSize = 100

var retentionFileFields = []string{
	"video_results",
	"screenshots",
	"maestro_screenshots",
	"logcats",
	"ios_logstreams",
	"report",
	"fcaf_report",
	"fcaf_report_pdf",
}

var retentionEvidenceFields = []string{
	"credential_well_knowns",
	"presentation_results",
}

// DeleteFilesOptions selects the pipeline results whose files are deleted.
type DeleteFilesOptions struct {
	OlderThanDays int  `json:"older_than_days" validate:"required,min=1"`
	DryRun        bool `json:"dry_run"`
	BatchSize     int  `json:"batch_size"      validate:"omitempty,min=1,max=500"`
}

// FileCounts counts pipeline result files per field.
type FileCounts struct {
	VideoResults       int `json:"video_results"`
	Screenshots        int `json:"screenshots"`
	MaestroScreenshots int `json:"maestro_screenshots"`
	Logcats            int `json:"logcats"`
	IOSLogstreams      int `json:"ios_logstreams"`
	Report             int `json:"report"`
	FCAFReport         int `json:"fcaf_report"`
	FCAFReportPDF      int `json:"fcaf_report_pdf"`
	Total              int `json:"total"`
}

// DeleteFilesResult reports what a retention run matched and deleted.
type DeleteFilesResult struct {
	OlderThanDays    int        `json:"older_than_days"`
	DryRun           bool       `json:"dry_run"`
	BatchSize        int        `json:"batch_size"`
	CutoffField      string     `json:"cutoff_field"`
	Cutoff           string     `json:"cutoff"`
	TotalRecords     int        `json:"total_records"`
	ScannedRecords   int        `json:"scanned_records"`
	MatchedRecords   int        `json:"matched_records"`
	RecordsWithFiles int        `json:"records_with_files"`
	UpdatedRecords   int        `json:"updated_records"`
	DeletedFiles     FileCounts `json:"deleted_files"`
}

// DeleteFilesOlderThan clears the files and evidence of pipeline results
// created more than opts.OlderThanDays ago.
func DeleteFilesOlderThan(app core.App, opts DeleteFilesOptions) (DeleteFilesResult, error) {
	batchSize := opts.BatchSize
	if batchSize == 0 {
		batchSize = defaultRetentionBatchSize
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -opts.OlderThanDays)
	return deleteFilesBefore(app, cutoff, opts.OlderThanDays, opts.DryRun, batchSize)
}

func deleteFilesBefore(
	app core.App,
	cutoff time.Time,
	olderThanDays int,
	dryRun bool,
	batchSize int,
) (DeleteFilesResult, error) {
	result := DeleteFilesResult{
		OlderThanDays: olderThanDays,
		DryRun:        dryRun,
		BatchSize:     batchSize,
		CutoffField:   "created",
		Cutoff:        cutoff.Format(time.RFC3339),
	}

	totalRecords, err := countRecords(app)
	if err != nil {
		return result, fmt.Errorf("count pipeline_results: %w", err)
	}
	result.TotalRecords = totalRecords

	offset := 0

	for {
		records, err := app.FindRecordsByFilter(
			collectionName,
			"",
			"created",
			batchSize,
			offset,
		)
		if err != nil {
			return result, fmt.Errorf("list pipeline_results: %w", err)
		}
		if len(records) == 0 {
			return result, nil
		}

		stop := false

		for _, record := range records {
			result.ScannedRecords++

			created := record.GetDateTime("created").Time().UTC()
			if created.After(cutoff) {
				stop = true
				break
			}

			result.MatchedRecords++
			counts := countFiles(record)
			hasFiles := counts.Total > 0
			if !hasFiles && !hasEvidence(record) {
				continue
			}

			if hasFiles {
				result.RecordsWithFiles++
			}
			result.DeletedFiles = addFileCounts(result.DeletedFiles, counts)

			if dryRun {
				continue
			}

			clearFiles(record)
			if err := app.Save(record); err != nil {
				return result, fmt.Errorf("save pipeline_result %s: %w", record.Id, err)
			}
			result.UpdatedRecords++
		}

		if stop {
			return result, nil
		}

		offset += len(records)
	}
}

func countRecords(app core.App) (int, error) {
	var total int

	if err := app.RecordQuery(collectionName).
		Select("count(*)").
		Limit(1).
		Row(&total); err != nil {
		return 0, err
	}

	return total, nil
}

func countFiles(record *core.Record) FileCounts {
	if record == nil {
		return FileCounts{}
	}

	counts := FileCounts{
		VideoResults:       len(record.GetStringSlice("video_results")),
		Screenshots:        len(record.GetStringSlice("screenshots")),
		MaestroScreenshots: len(record.GetStringSlice("maestro_screenshots")),
		Logcats:            len(record.GetStringSlice("logcats")),
		IOSLogstreams:      len(record.GetStringSlice("ios_logstreams")),
		Report:             len(record.GetStringSlice("report")),
		FCAFReport:         len(record.GetStringSlice("fcaf_report")),
		FCAFReportPDF:      len(record.GetStringSlice("fcaf_report_pdf")),
	}
	counts.Total = counts.VideoResults + counts.Screenshots + counts.MaestroScreenshots +
		counts.Logcats + counts.IOSLogstreams + counts.Report + counts.FCAFReport +
		counts.FCAFReportPDF

	return counts
}

func addFileCounts(left FileCounts, right FileCounts) FileCounts {
	left.VideoResults += right.VideoResults
	left.Screenshots += right.Screenshots
	left.MaestroScreenshots += right.MaestroScreenshots
	left.Logcats += right.Logcats
	left.IOSLogstreams += right.IOSLogstreams
	left.Report += right.Report
	left.FCAFReport += right.FCAFReport
	left.FCAFReportPDF += right.FCAFReportPDF
	left.Total += right.Total

	return left
}

func clearFiles(record *core.Record) {
	for _, field := range retentionFileFields {
		record.Set(field, []string{})
	}
	for _, field := range retentionEvidenceFields {
		record.Set(field, []any{})
	}
}

func hasEvidence(record *core.Record) bool {
	if record == nil {
		return false
	}
	for _, field := range retentionEvidenceFields {
		if len(strings.TrimSpace(record.GetString(field))) > 0 {
			return true
		}
	}
	return false
}
