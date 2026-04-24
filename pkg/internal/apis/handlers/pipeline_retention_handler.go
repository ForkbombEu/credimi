// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

const (
	pipelineRetentionDefaultBatchSize = 100
	pipelineRetentionDefaultDays      = 30
	pipelineRetentionDefaultInterval  = 1
	pipelineRetentionScheduleID       = "pipeline-retention-schedule"
)

var pipelineRetentionFileFields = []string{
	"video_results",
	"screenshots",
	"logcats",
	"ios_logstreams",
}

var pipelineRetentionImmediateTriggerOptions = client.ScheduleTriggerOptions{
	Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
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

type SchedulePipelineRetentionRequest struct {
	OlderThanDays int `json:"older_than_days" validate:"omitempty,min=1"`
	IntervalDays  int `json:"interval_days"   validate:"omitempty,min=1"`
}

type SchedulePipelineRetentionResponse struct {
	Message           string `json:"message"`
	ScheduleID        string `json:"schedule_id"`
	WorkflowNamespace string `json:"workflowNamespace"`
}

type DeletePipelineRetentionScheduleResponse struct {
	Success           bool   `json:"success"`
	Message           string `json:"message"`
	ScheduleID        string `json:"schedule_id"`
	WorkflowNamespace string `json:"workflowNamespace"`
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

func HandleSchedulePipelineRetentionWorkflow() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[SchedulePipelineRetentionRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"request.validation",
				"invalid_request",
				err.Error(),
			).JSON(e)
		}

		olderThanDays := input.OlderThanDays
		if olderThanDays == 0 {
			olderThanDays = pipelineRetentionDefaultDays
		}

		intervalDays := input.IntervalDays
		if intervalDays == 0 {
			intervalDays = pipelineRetentionDefaultInterval
		}

		namespace := workflows.DefaultNamespace
		appURL := e.App.Settings().Meta.AppURL

		c, err := scheduleTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"failed to create temporal client",
				err.Error(),
			).JSON(e)
		}

		ctx := context.Background()
		scheduleID := pipelineRetentionScheduleID
		options := buildPipelineRetentionScheduleOptions(
			scheduleID,
			appURL,
			olderThanDays,
			intervalDays,
		)

		_, err = c.ScheduleClient().Create(ctx, options)
		if err != nil {
			if isScheduleAlreadyExistsError(err) {
				handle := c.ScheduleClient().GetHandle(ctx, scheduleID)
				err = handle.Update(ctx, client.ScheduleUpdateOptions{
					DoUpdate: func(client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
						return &client.ScheduleUpdate{Schedule: buildPipelineRetentionSchedule(
							appURL,
							olderThanDays,
							intervalDays,
						)}, nil
					},
				})
				if err == nil {
					err = handle.Trigger(ctx, pipelineRetentionImmediateTriggerOptions)
				}
			}
		} else {
			handle := c.ScheduleClient().GetHandle(ctx, scheduleID)
			err = handle.Trigger(ctx, pipelineRetentionImmediateTriggerOptions)
		}
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"schedule",
				"failed to upsert retention schedule",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, SchedulePipelineRetentionResponse{
			Message: fmt.Sprintf(
				"Pipeline retention triggered now and scheduled every %d day(s) with older_than_days=%d",
				intervalDays,
				olderThanDays,
			),
			ScheduleID:        scheduleID,
			WorkflowNamespace: workflows.DefaultNamespace,
		})
	}
}

func HandleDeletePipelineRetentionSchedule() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		namespace := workflows.DefaultNamespace

		c, err := scheduleTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"failed to create temporal client",
				err.Error(),
			).JSON(e)
		}

		ctx := context.Background()
		handle := c.ScheduleClient().GetHandle(ctx, pipelineRetentionScheduleID)

		if err := handle.Delete(ctx); err != nil {
			var notFound *serviceerror.NotFound
			if errors.As(err, &notFound) {
				return apierror.New(
					http.StatusNotFound,
					"schedule",
					"retention schedule not found",
					err.Error(),
				).JSON(e)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"schedule",
				"failed to delete retention schedule",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, DeletePipelineRetentionScheduleResponse{
			Success:           true,
			Message:           "Pipeline retention schedule deleted successfully",
			ScheduleID:        pipelineRetentionScheduleID,
			WorkflowNamespace: namespace,
		})
	}
}

func buildPipelineRetentionScheduleOptions(
	scheduleID string,
	appURL string,
	olderThanDays int,
	intervalDays int,
) client.ScheduleOptions {
	return client.ScheduleOptions{
		ID:      scheduleID,
		Spec:    buildPipelineRetentionScheduleSpec(intervalDays),
		Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
		Action:  buildPipelineRetentionScheduleAction(appURL, olderThanDays),
	}
}

func buildPipelineRetentionSchedule(
	appURL string,
	olderThanDays int,
	intervalDays int,
) *client.Schedule {
	return &client.Schedule{
		Spec: &client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{{
				Every: time.Duration(intervalDays) * 24 * time.Hour,
			}},
		},
		Policy: buildPipelineRetentionSchedulePolicy(),
		State:  buildPipelineRetentionScheduleState(),
		Action: buildPipelineRetentionScheduleAction(appURL, olderThanDays),
	}
}

func buildPipelineRetentionScheduleSpec(intervalDays int) client.ScheduleSpec {
	return client.ScheduleSpec{
		Intervals: []client.ScheduleIntervalSpec{{
			Every: time.Duration(intervalDays) * 24 * time.Hour,
		}},
	}
}

func buildPipelineRetentionSchedulePolicy() *client.SchedulePolicies {
	return &client.SchedulePolicies{
		Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
	}
}

func buildPipelineRetentionScheduleState() *client.ScheduleState {
	return &client.ScheduleState{}
}

func buildPipelineRetentionScheduleAction(
	appURL string,
	olderThanDays int,
) *client.ScheduleWorkflowAction {
	workflowName := workflows.NewPipelineRetentionWorkflow().Name()

	return &client.ScheduleWorkflowAction{
		ID:        pipelineRetentionScheduleID,
		Workflow:  workflowName,
		TaskQueue: workflows.PipelineRetentionTaskQueue,
		Args: []interface{}{
			workflowengine.WorkflowInput{
				Payload: workflows.PipelineRetentionWorkflowInput{
					OlderThanDays: olderThanDays,
					DryRun:        false,
				},
				Config: map[string]any{
					"app_url": appURL,
				},
			},
		},
	}
}

func isScheduleAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}

	var alreadyExists *serviceerror.AlreadyExists
	if errors.As(err, &alreadyExists) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already registered") ||
		strings.Contains(msg, "already exists")
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
		records, err := app.FindRecordsByFilter(
			"pipeline_results",
			"",
			"created",
			batchSize,
			offset,
		)
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
