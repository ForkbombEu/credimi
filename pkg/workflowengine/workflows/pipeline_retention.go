// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"fmt"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const PipelineRetentionTaskQueue = "PipelineRetentionTaskQueue"
const PipelineRetentionDefaultBatchSize = 100

var pipelineRetentionStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

type PipelineRetentionWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

type PipelineRetentionWorkflowInput struct {
	OlderThanDays int  `json:"older_than_days"`
	DryRun        bool `json:"dry_run"`
	BatchSize     int  `json:"batch_size,omitempty"`
}

func NewPipelineRetentionWorkflow() *PipelineRetentionWorkflow {
	w := &PipelineRetentionWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (w *PipelineRetentionWorkflow) Name() string {
	return "PipelineRetentionWorkflow"
}

func (w *PipelineRetentionWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *PipelineRetentionWorkflow) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "pipeline-retention-" + uuid.NewString(),
		TaskQueue:                PipelineRetentionTaskQueue,
		WorkflowExecutionTimeout: time.Hour,
	}

	return pipelineRetentionStartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}

func (w *PipelineRetentionWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *PipelineRetentionWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	payload, err := workflowengine.DecodePayload[PipelineRetentionWorkflowInput](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}

	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			input.RunMetadata,
		)
	}

	httpActivity := activities.NewInternalHTTPActivity()
	batchSize := payload.BatchSize
	if batchSize == 0 {
		batchSize = PipelineRetentionDefaultBatchSize
	}
	request := workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method: http.MethodPost,
			URL: utils.JoinURL(
				appURL,
				"api", "pipeline", "retention", "delete-files",
			),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: map[string]any{
				"older_than_days": payload.OlderThanDays,
				"dry_run":         payload.DryRun,
				"batch_size":      batchSize,
			},
			ExpectedStatus: http.StatusOK,
		},
	}

	var httpResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(ctx, httpActivity.Name(), request).Get(ctx, &httpResult); err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, input.RunMetadata)
	}

	output, ok := httpResult.Output.(map[string]any)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: invalid output format", errCode.Description),
			httpResult.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, input.RunMetadata)
	}

	body, ok := output["body"].(map[string]any)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: missing body in output", errCode.Description),
			output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, input.RunMetadata)
	}

	message := "Pipeline retention completed"
	if payload.DryRun {
		message = "Pipeline retention dry run completed"
	}

	return workflowengine.WorkflowResult{
		Message: message,
		Output:  body,
	}, nil
}
