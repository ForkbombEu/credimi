// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/sdk/workflow"
)

// CleanupVerificationPayload defines the input for the verification workflow.
type CleanupVerificationPayload struct {
	WorkflowID    string            `json:"workflow_id"`
	RunID         string            `json:"run_id"`
	RunIdentifier string            `json:"run_identifier"`
	DelaySeconds  int               `json:"delay_seconds"`
	StepSpecs     []CleanupStepSpec `json:"step_specs"`
}

// CleanupVerificationWorkflow re-runs cleanup steps after a delay.
type CleanupVerificationWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

func NewCleanupVerificationWorkflow() *CleanupVerificationWorkflow {
	w := &CleanupVerificationWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (CleanupVerificationWorkflow) Name() string {
	return "Cleanup verification workflow"
}

func (CleanupVerificationWorkflow) GetOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{}
}

func (w *CleanupVerificationWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *CleanupVerificationWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	payload, err := workflowengine.DecodePayload[CleanupVerificationPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}

	delay := time.Minute
	if payload.DelaySeconds > 0 {
		delay = time.Duration(payload.DelaySeconds) * time.Second
	}
	workflow.Sleep(ctx, delay)

	baseAo := w.GetOptions()
	if input.ActivityOptions != nil {
		baseAo = *input.ActivityOptions
	}
	options := buildCleanupOptions(baseAo)

	recordFailure := func(ctx workflow.Context, spec CleanupStepSpec, stepErr error, attempts int) error {
		workflowID := payload.WorkflowID
		if workflowID == "" {
			workflowID = workflow.GetInfo(ctx).WorkflowExecution.ID
		}
		return recordFailedCleanup(ctx, options, spec, stepErr, attempts, workflowID)
	}

	cleanupErrors := executeCleanupSpecs(ctx, logger, options, payload.StepSpecs, nil, recordFailure)
	if len(cleanupErrors) > 0 {
		logger.Warn("cleanup verification recorded failures", "count", len(cleanupErrors))
	}

	return workflowengine.WorkflowResult{Message: "cleanup verification completed"}, nil
}
