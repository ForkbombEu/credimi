// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"fmt"

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
	subCtx := workflow.WithActivityOptions(ctx, w.GetOptions())
	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(subCtx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(subCtx).Namespace,
		TemporalUI:   fmt.Sprintf("%s/my/tests/runs/%s/%s", input.Config["app_url"], workflow.GetInfo(subCtx).WorkflowExecution.ID, workflow.GetInfo(subCtx).WorkflowExecution.RunID),
	}
	stepCIInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"yaml": input.Payload["yaml"],
		},
	}
	var stepCIResult workflowengine.ActivityResult

	err := workflow.ExecuteActivity(subCtx, stepCIWorkflowActivity.Name(), stepCIInput).
		Get(subCtx, &stepCIResult)

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
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        "custom" + "-" + uuid.NewString(),
		TaskQueue: CustomCheckTaskQueque,
	}
	return workflowengine.StartWorkflowWithOptions(workflowOptions, w.Name(), input)
}
