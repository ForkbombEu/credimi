// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

const (
	failedCleanupStatusPending   = "PENDING"
	failedCleanupStatusRetrying  = "RETRYING"
	failedCleanupStatusAbandoned = "ABANDONED"
)

type FailedCleanupRecord struct {
	ID         string         `json:"id"`
	WorkflowID string         `json:"workflow_id"`
	StepName   string         `json:"step_name"`
	Payload    map[string]any `json:"payload"`
	RetryCount int            `json:"retry_count"`
	Status     string         `json:"status"`
	Error      string         `json:"error"`
}

type RecordFailedCleanupPayload struct {
	WorkflowID string `json:"workflow_id" yaml:"workflow_id" validate:"required"`
	StepName   string `json:"step_name"   yaml:"step_name"   validate:"required"`
	Error      string `json:"error"       yaml:"error"       validate:"required"`
	RetryCount int    `json:"retry_count" yaml:"retry_count" validate:"required"`
	Payload    any    `json:"payload"     yaml:"payload"`
}

type FetchFailedCleanupsPayload struct {
	Status     string `json:"status"       yaml:"status"`
	MaxRetries int    `json:"max_retries"  yaml:"max_retries"`
	Limit      int    `json:"limit"        yaml:"limit"`
}

type UpdateFailedCleanupPayload struct {
	RecordID   string `json:"record_id"   yaml:"record_id"   validate:"required"`
	Status     string `json:"status"      yaml:"status"      validate:"required"`
	RetryCount int    `json:"retry_count" yaml:"retry_count" validate:"required"`
	Error      string `json:"error"       yaml:"error"`
}

type DeleteFailedCleanupPayload struct {
	RecordID string `json:"record_id" yaml:"record_id" validate:"required"`
}

type RecordFailedCleanupActivity struct {
	workflowengine.BaseActivity
}

type FetchFailedCleanupsActivity struct {
	workflowengine.BaseActivity
}

type UpdateFailedCleanupActivity struct {
	workflowengine.BaseActivity
}

type DeleteFailedCleanupActivity struct {
	workflowengine.BaseActivity
}

func NewRecordFailedCleanupActivity() *RecordFailedCleanupActivity {
	return &RecordFailedCleanupActivity{
		BaseActivity: workflowengine.BaseActivity{Name: "Record failed cleanup"},
	}
}

func NewCleanupReconciliationActivity() *FetchFailedCleanupsActivity {
	return &FetchFailedCleanupsActivity{
		BaseActivity: workflowengine.BaseActivity{Name: "Fetch failed cleanups"},
	}
}

func NewUpdateFailedCleanupActivity() *UpdateFailedCleanupActivity {
	return &UpdateFailedCleanupActivity{
		BaseActivity: workflowengine.BaseActivity{Name: "Update failed cleanup"},
	}
}

func NewDeleteFailedCleanupActivity() *DeleteFailedCleanupActivity {
	return &DeleteFailedCleanupActivity{
		BaseActivity: workflowengine.BaseActivity{Name: "Delete failed cleanup"},
	}
}

func (a *RecordFailedCleanupActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *FetchFailedCleanupsActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *UpdateFailedCleanupActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *DeleteFailedCleanupActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *RecordFailedCleanupActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[RecordFailedCleanupPayload](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}

	app, err := getPocketBaseApp()
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(errCode.Code, err.Error())
	}
	collection, err := app.FindCollectionByNameOrId("failed_cleanups")
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(errCode.Code, err.Error())
	}

	record := core.NewRecord(collection)
	record.Set("workflow_id", payload.WorkflowID)
	record.Set("step_name", payload.StepName)
	record.Set("error", payload.Error)
	record.Set("retry_count", payload.RetryCount)
	record.Set("status", failedCleanupStatusPending)
	record.Set("last_attempt", time.Now())
	if payload.Payload != nil {
		record.Set("payload", normalizePayload(payload.Payload))
	}

	if err := app.Save(record); err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(errCode.Code, err.Error())
	}

	return workflowengine.ActivityResult{Output: record.Id}, nil
}

func (a *FetchFailedCleanupsActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[FetchFailedCleanupsPayload](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}

	app, err := getPocketBaseApp()
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(errCode.Code, err.Error())
	}

	filterParts := []string{}
	status := payload.Status
	if status == "" {
		status = failedCleanupStatusPending
	}
	filterParts = append(filterParts, fmt.Sprintf("status = '%s'", status))
	if payload.MaxRetries > 0 {
		filterParts = append(filterParts, fmt.Sprintf("retry_count < %d", payload.MaxRetries))
	}
	filter := strings.Join(filterParts, " && ")

	limit := payload.Limit
	if limit == 0 {
		limit = 50
	}
	records, err := app.FindRecordsByFilter("failed_cleanups", filter, "", limit, 0)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(errCode.Code, err.Error())
	}

	result := make([]FailedCleanupRecord, 0, len(records))
	for _, record := range records {
		result = append(result, FailedCleanupRecord{
			ID:         record.Id,
			WorkflowID: record.GetString("workflow_id"),
			StepName:   record.GetString("step_name"),
			RetryCount: int(record.GetInt("retry_count")),
			Status:     record.GetString("status"),
			Error:      record.GetString("error"),
			Payload:    workflowengine.AsMap(record.Get("payload")),
		})
	}

	return workflowengine.ActivityResult{Output: result}, nil
}

func (a *UpdateFailedCleanupActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[UpdateFailedCleanupPayload](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}

	app, err := getPocketBaseApp()
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(errCode.Code, err.Error())
	}

	record, err := app.FindRecordById("failed_cleanups", payload.RecordID)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(errCode.Code, err.Error())
	}
	if record == nil {
		return workflowengine.ActivityResult{}, nil
	}

	record.Set("status", payload.Status)
	record.Set("retry_count", payload.RetryCount)
	if payload.Error != "" {
		record.Set("error", payload.Error)
	}
	record.Set("last_attempt", time.Now())

	if err := app.Save(record); err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(errCode.Code, err.Error())
	}
	return workflowengine.ActivityResult{Output: record.Id}, nil
}

func (a *DeleteFailedCleanupActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[DeleteFailedCleanupPayload](input.Payload)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}

	app, err := getPocketBaseApp()
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(errCode.Code, err.Error())
	}

	record, err := app.FindRecordById("failed_cleanups", payload.RecordID)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(errCode.Code, err.Error())
	}
	if record == nil {
		return workflowengine.ActivityResult{}, nil
	}

	if err := app.Delete(record); err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(errCode.Code, err.Error())
	}
	return workflowengine.ActivityResult{Output: payload.RecordID}, nil
}

func normalizePayload(payload any) any {
	if payload == nil {
		return nil
	}
	if mapped := workflowengine.AsMap(payload); mapped != nil {
		return mapped
	}
	decoded, err := workflowengine.DecodePayload[map[string]any](payload)
	if err != nil {
		return payload
	}
	return decoded
}
