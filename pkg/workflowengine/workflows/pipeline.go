// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows/pipeline"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const PipelineTaskQueue = "PipelineTaskQueue"

type PipelineWorkflow struct{}

func (PipelineWorkflow) Name() string {
	return "Dynamic Pipeline Workflow"
}

func (PipelineWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

// Workflow executes all steps in the workflow definition sequentially
func (w *PipelineWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {

	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())
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
	var wfDef *pipeline.WorkflowDefinition
	if yamlStr, ok := input.Payload["yaml"].(string); ok {
		var err error
		wfDef, err = pipeline.ParseWorkflow(yamlStr)
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.PipelineParsingError]
			appErr := workflowengine.NewAppError(errCode, err.Error(), yamlStr)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
		}
	} else {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"yaml",
			runMetadata,
		)
	}

	// Initialize result map
	finalOutput := make(map[string]any)
	globalCfg := wfDef.Config

	// Execute each step
	for _, step := range wfDef.Steps {
		_, err := step.Run(ctx, globalCfg, &finalOutput)
		if err != nil {
			logger.Error(step.Name, "step execution error", err)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
		}
	}

	return workflowengine.WorkflowResult{
		Output: finalOutput,
	}, nil
}

// Start launches the pipeline workflow via Temporal client
func (w *PipelineWorkflow) Start(
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        "pipeline-" + uuid.NewString(),
		TaskQueue: PipelineTaskQueue,
	}
	return workflowengine.StartWorkflowWithOptions(workflowOptions, w.Name(), input)
}
