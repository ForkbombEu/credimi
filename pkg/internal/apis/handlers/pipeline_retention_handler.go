// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
)

const (
	pipelineRetentionDefaultBatchSize = 100
	pipelineRetentionMaxBatchSize     = 500
)

var pipelineRetentionFileFields = []string{
	"video_results",
	"screenshots",
	"logcats",
	"ios_logstreams",
}

type DeletePipelineResultFilesRequest struct {
	OlderThanDays int  `json:"older_than_days" validate:"required,min=1"`
	DryRun        bool `json:"dry_run"`
	BatchSize     int  `json:"batch_size"      validate:"omitempty,min=1,max=500"`
}

type PipelineResultFileCounts struct {
	VideoResults  int `json:"video_results"`
	Screenshots   int `json:"screenshots"`
	Logcats       int `json:"logcats"`
	IOSLogstreams int `json:"ios_logstreams"`
	Total         int `json:"total"`
}

type DeletePipelineResultFilesResponse struct {
	OlderThanDays    int                      `json:"older_than_days"`
	DryRun           bool                     `json:"dry_run"`
	BatchSize        int                      `json:"batch_size"`
	CutoffField      string                   `json:"cutoff_field"`
	Cutoff           string                   `json:"cutoff"`
	TotalRecords     int                      `json:"total_records"`
	ScannedRecords   int                      `json:"scanned_records"`
	MatchedRecords   int                      `json:"matched_records"`
	RecordsWithFiles int                      `json:"records_with_files"`
	UpdatedRecords   int                      `json:"updated_records"`
	DeletedFiles     PipelineResultFileCounts `json:"deleted_files"`
}

func HandleDeletePipelineResultFiles() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[DeletePipelineResultFilesRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"request.validation",
				"invalid_request",
				err.Error(),
			).JSON(e)
		}

		batchSize := input.BatchSize
		if batchSize == 0 {
			batchSize = pipelineRetentionDefaultBatchSize
		}

		cutoff := time.Now().UTC().AddDate(0, 0, -input.OlderThanDays)
		response, err := deletePipelineResultFilesOlderThan(
			e.App,
			cutoff,
			input.OlderThanDays,
			input.DryRun,
			batchSize,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"pipeline_results",
				"failed to delete retained files",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, response)
	}
}

func deletePipelineResultFilesOlderThan(
	app core.App,
	cutoff time.Time,
	olderThanDays int,
	dryRun bool,
	batchSize int,
) (DeletePipelineResultFilesResponse, error) {
	response := DeletePipelineResultFilesResponse{
		OlderThanDays: olderThanDays,
		DryRun:        dryRun,
		BatchSize:     batchSize,
		CutoffField:   "created",
		Cutoff:        cutoff.Format(time.RFC3339),
	}

	totalRecords, err := countPipelineResultRecords(app)
	if err != nil {
		return response, fmt.Errorf("count pipeline_results: %w", err)
	}
	response.TotalRecords = totalRecords

	offset := 0

	for {
		records, err := app.FindRecordsByFilter("pipeline_results", "", "created", batchSize, offset)
		if err != nil {
			return response, fmt.Errorf("list pipeline_results: %w", err)
		}
		if len(records) == 0 {
			return response, nil
		}

		stop := false

		for _, record := range records {
			response.ScannedRecords++

			created := record.GetDateTime("created").Time().UTC()
			if created.After(cutoff) {
				stop = true
				break
			}

			response.MatchedRecords++
			counts := countPipelineResultFiles(record)
			if counts.Total == 0 {
				continue
			}

			response.RecordsWithFiles++
			response.DeletedFiles = addPipelineResultFileCounts(response.DeletedFiles, counts)

			if dryRun {
				continue
			}

			clearPipelineResultFiles(record)
			if err := app.Save(record); err != nil {
				return response, fmt.Errorf("save pipeline_result %s: %w", record.Id, err)
			}
			response.UpdatedRecords++
		}

		if stop {
			return response, nil
		}

		offset += len(records)
	}
}

func countPipelineResultRecords(app core.App) (int, error) {
	var total int

	if err := app.RecordQuery("pipeline_results").Select("count(*)").Limit(1).Row(&total); err != nil {
		return 0, err
	}

	return total, nil
}

func countPipelineResultFiles(record *core.Record) PipelineResultFileCounts {
	if record == nil {
		return PipelineResultFileCounts{}
	}

	counts := PipelineResultFileCounts{
		VideoResults:  len(record.GetStringSlice("video_results")),
		Screenshots:   len(record.GetStringSlice("screenshots")),
		Logcats:       len(record.GetStringSlice("logcats")),
		IOSLogstreams: len(record.GetStringSlice("ios_logstreams")),
	}
	counts.Total = counts.VideoResults + counts.Screenshots + counts.Logcats + counts.IOSLogstreams

	return counts
}

func addPipelineResultFileCounts(
	left PipelineResultFileCounts,
	right PipelineResultFileCounts,
) PipelineResultFileCounts {
	left.VideoResults += right.VideoResults
	left.Screenshots += right.Screenshots
	left.Logcats += right.Logcats
	left.IOSLogstreams += right.IOSLogstreams
	left.Total += right.Total

	return left
}

func clearPipelineResultFiles(record *core.Record) {
	for _, field := range pipelineRetentionFileFields {
		record.Set(field, []string{})
	}
}
