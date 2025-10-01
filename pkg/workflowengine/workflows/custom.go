// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const CustomCheckTaskQueque = "custom-check-task-queue"

type CustomCheckWorkflow struct{}

func (CustomCheckWorkflow) Name() string {
	return "Custom Check Workflow"
}

func (CustomCheckWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *CustomCheckWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	stepCIWorkflowActivity := activities.NewStepCIWorkflowActivity()
	logger := workflow.GetLogger(ctx)

	opts := w.GetOptions()
	if input.ActivityOptions != nil {
		opts = *input.ActivityOptions
	}
	ctx = workflow.WithActivityOptions(ctx, opts)
	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI: fmt.Sprintf(
			"%s/my/tests/runs/%s/%s",
			input.Config["app_url"],
			workflow.GetInfo(ctx).WorkflowExecution.ID,
			workflow.GetInfo(ctx).WorkflowExecution.RunID,
		),
	}
	yaml, ok := input.Payload["yaml"].(string)
	if !ok || yaml == "" {
		id, ok := input.Payload["id"].(string)
		if !ok || id == "" {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
				"id or yaml",
				runMetadata,
			)
		}
		var HTTPActivity = activities.NewHTTPActivity()
		var HTTPResponse workflowengine.ActivityResult
		err := workflow.ExecuteActivity(ctx, HTTPActivity.Name(), workflowengine.ActivityInput{
			Payload: map[string]any{
				"method": "POST",
				"url": fmt.Sprintf(
					"%s/%s",
					input.Config["app_url"].(string),
					"api/canonify/identifier/validate",
				),
				"body": map[string]any{
					"canonified_name": id,
				},
				"expected_status": 200,
			},
		}).Get(ctx, &HTTPResponse)
		if err != nil {
			logger.Error(HTTPActivity.Name(), "error", err)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
		}
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		output, ok := HTTPResponse.Output.(map[string]any)
		if !ok {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: invalid output format", errCode.Description),
				HTTPResponse.Output,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
		}

		body, ok := output["body"].(map[string]any)
		if !ok {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: missing body in output", errCode.Description),
				output,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
		}

		record, ok := body["record"].(map[string]any)
		if !ok {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: missing record in body", errCode.Description),
				body,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
		}

		yaml, ok = record["yaml"].(string)
		if !ok || yaml == "" {
			appErr := workflowengine.NewAppError(
				errCode,
				"missing yaml in custom check record",
				record,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
		}
		input.Payload["yaml"] = yaml
	}

	stepCIInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"yaml": yaml,
			"env":  input.Config["env"],
		},
	}
	var stepCIResult workflowengine.ActivityResult

	err := workflow.ExecuteActivity(ctx, stepCIWorkflowActivity.Name(), stepCIInput).
		Get(ctx, &stepCIResult)

	if err != nil {
		logger.Error(stepCIWorkflowActivity.Name(), "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}
	result := stepCIResult.Output.(map[string]any)
	return workflowengine.WorkflowResult{
		Output: result["tests"].([]any),
	}, nil
}

func (w *CustomCheckWorkflow) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        "custom" + "-" + uuid.NewString(),
		TaskQueue: CustomCheckTaskQueque,
	}
	return workflowengine.StartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}
