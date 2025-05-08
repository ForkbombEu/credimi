// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"errors"

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
	stepCIWorkflowActivity := activities.StepCIWorkflowActivity{}
	logger := workflow.GetLogger(ctx)

	stepCIInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"yaml": input.Payload["yaml"],
		},
		Config: map[string]string{},
	}
	var stepCIResult workflowengine.ActivityResult

	err := workflow.ExecuteActivity(ctx, stepCIWorkflowActivity.Name(), stepCIInput).
		Get(ctx, &stepCIResult)
	if err != nil {
		logger.Error("StepCIExecution failed", "error", err)
		return workflowengine.WorkflowResult{}, err
	}

	result, ok := stepCIResult.Output.(map[string]any)["result"].(string)
	if !ok {
		result = ""
	}
	if result == "" {
		logger.Error("StepCIExecution result is empty")
		return workflowengine.WorkflowResult{}, errors.New("StepCIExecution result is empty")
	}

	return workflowengine.WorkflowResult{
		Message: "Workflow completed successfully",
	}, nil
}

func (w *CustomCheckWorkflow) Start(
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID: "custom" + "-" + uuid.NewString(),
		TaskQueue: CustomCheckTaskQueque,
	}
	return workflowengine.StartWorkflowWithOptions(workflowOptions, w.Name(), input)

}
