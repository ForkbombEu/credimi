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
	stepCIWorkflowActivity := activities.StepCIWorkflowActivity{}
	logger := workflow.GetLogger(ctx)
	subCtx := workflow.WithActivityOptions(ctx, w.GetOptions())

	stepCIInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"yaml": input.Payload["yaml"],
		},
		Config: map[string]string{},
	}
	var stepCIResult workflowengine.ActivityResult

	err := workflow.ExecuteActivity(subCtx, stepCIWorkflowActivity.Name(), stepCIInput).
		Get(subCtx, &stepCIResult)
	if err != nil {
		logger.Error("StepCIExecution failed", "error", err)
		return workflowengine.WorkflowResult{}, err
	}

	result := stepCIResult.Output.(map[string]any)["result"]

	return workflowengine.WorkflowResult{
		Message: fmt.Sprintf("Custom check result: %v", result),
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
