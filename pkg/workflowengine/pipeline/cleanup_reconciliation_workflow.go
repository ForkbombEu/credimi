// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"errors"
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const defaultCleanupReconciliationWorkflowID = "cleanup-reconciliation-manager"

type CleanupReconciliationPayload struct {
	IntervalSeconds int `json:"interval_seconds"`
	MaxRetries      int `json:"max_retries"`
	Limit           int `json:"limit"`
	MaxIterations   int `json:"max_iterations"`
}

// CleanupReconciliationWorkflow periodically replays failed cleanup steps.
type CleanupReconciliationWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

func NewCleanupReconciliationWorkflow() *CleanupReconciliationWorkflow {
	w := &CleanupReconciliationWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (CleanupReconciliationWorkflow) Name() string {
	return "Cleanup reconciliation workflow"
}

func (CleanupReconciliationWorkflow) GetOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{}
}

func (w *CleanupReconciliationWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *CleanupReconciliationWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	payload, err := workflowengine.DecodePayload[CleanupReconciliationPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}

	interval := 5 * time.Minute
	if payload.IntervalSeconds > 0 {
		interval = time.Duration(payload.IntervalSeconds) * time.Second
	}
	maxRetries := payload.MaxRetries
	if maxRetries == 0 {
		maxRetries = 10
	}
	limit := payload.Limit
	if limit == 0 {
		limit = 50
	}

	baseAo := w.GetOptions()
	if input.ActivityOptions != nil {
		baseAo = *input.ActivityOptions
	}
	options := buildCleanupOptions(baseAo)

	fetchActivity := activities.NewCleanupReconciliationActivity()
	updateActivity := activities.NewUpdateFailedCleanupActivity()
	deleteActivity := activities.NewDeleteFailedCleanupActivity()

	iterations := 0
	for {
		var fetchResult workflowengine.ActivityResult
		fetchErr := workflow.ExecuteActivity(
			ctx,
			fetchActivity.Name(),
			workflowengine.ActivityInput{
				Payload: activities.FetchFailedCleanupsPayload{
					Status:     "PENDING",
					MaxRetries: maxRetries,
					Limit:      limit,
				},
			},
		).Get(ctx, &fetchResult)
		if fetchErr != nil {
			logger.Error("cleanup reconciliation fetch failed", "error", fetchErr)
		} else {
			records, err := workflowengine.DecodePayload[[]activities.FailedCleanupRecord](fetchResult.Output)
			if err != nil {
				logger.Error("cleanup reconciliation decode failed", "error", err)
			} else {
				for _, record := range records {
					spec := CleanupStepSpec{
						Name:       record.StepName,
						Type:       cleanupStepType(record.StepName),
						Payload:    record.Payload,
						MaxRetries: 1,
					}
					stepErrors := executeCleanupSpecs(ctx, logger, options, []CleanupStepSpec{spec}, nil, nil)
					if len(stepErrors) == 0 {
						_ = workflow.ExecuteActivity(
							ctx,
							deleteActivity.Name(),
							workflowengine.ActivityInput{
								Payload: activities.DeleteFailedCleanupPayload{RecordID: record.ID},
							},
						).Get(ctx, nil)
						continue
					}

					newRetryCount := record.RetryCount + 1
					status := "PENDING"
					if newRetryCount >= maxRetries {
						status = "ABANDONED"
					}
					_ = workflow.ExecuteActivity(
						ctx,
						updateActivity.Name(),
						workflowengine.ActivityInput{
							Payload: activities.UpdateFailedCleanupPayload{
								RecordID:   record.ID,
								Status:     status,
								RetryCount: newRetryCount,
								Error:      stepErrors[0].Error(),
							},
						},
					).Get(ctx, nil)
					if status == "ABANDONED" {
						logger.Warn("cleanup reconciliation abandoned record", "record_id", record.ID)
					}
				}
			}
		}

		iterations++
		if payload.MaxIterations > 0 && iterations >= payload.MaxIterations {
			break
		}
		workflow.Sleep(ctx, interval)
	}

	return workflowengine.WorkflowResult{Message: "cleanup reconciliation completed"}, nil
}

func StartCleanupReconciliationWorkflow(
	namespace string,
	taskQueue string,
	config CleanupReconciliationPayload,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        defaultCleanupReconciliationWorkflowID,
		TaskQueue: taskQueue,
	}
	w := NewCleanupReconciliationWorkflow()
	input := workflowengine.WorkflowInput{Payload: config, Config: map[string]any{}}
	result, err := workflowengine.StartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
	if err != nil {
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStarted) {
			return result, nil
		}
		return workflowengine.WorkflowResult{}, fmt.Errorf("failed to start cleanup reconciliation workflow: %w", err)
	}
	return result, nil
}
