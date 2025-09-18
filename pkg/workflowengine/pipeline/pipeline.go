// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"context"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	temporalclient "github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

const PipelineTaskQueue = "PipelineTaskQueue"

type PipelineWorkflow struct{}

type PipelineWorkflowInput struct {
	WorkflowDefinition *WorkflowDefinition
	WorkflowBlock      *WorkflowBlock
	WorkflowInput      workflowengine.WorkflowInput
}

func (PipelineWorkflow) Name() string {
	return "Dynamic Pipeline Workflow"
}

func (PipelineWorkflow) GetOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{}
}

// Workflow executes all steps in the workflow definition sequentially
func (w *PipelineWorkflow) Workflow(
	ctx workflow.Context,
	input PipelineWorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, *input.WorkflowInput.ActivityOptions)
	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI: fmt.Sprintf(
			"%s/my/tests/runs/%s/%s",
			input.WorkflowInput.Config["app_url"],
			workflow.GetInfo(ctx).WorkflowExecution.ID,
			workflow.GetInfo(ctx).WorkflowExecution.RunID,
		),
	}
	global := map[string]any{}
	if g, ok := input.WorkflowInput.Config["global"].(map[string]any); ok {
		global = g
	}

	// Normalize to string map
	globalCfg := make(map[string]string)
	for k, v := range global {
		if str, ok := v.(string); ok {
			globalCfg[k] = str
		} else {
			return workflowengine.WorkflowResult{}, fmt.Errorf("global config key %q has non-string value of type %T", k, v)
		}
	}

	result := workflowengine.WorkflowResult{}
	finalOutput := map[string]any{}
	var steps []StepDefinition
	var checks map[string]WorkflowBlock
	switch {
	case input.WorkflowBlock != nil:
		steps = input.WorkflowBlock.Steps

	case input.WorkflowDefinition != nil:
		steps = input.WorkflowDefinition.Steps
		checks = input.WorkflowDefinition.Checks

	default:
		errCode := errorcodes.Codes[errorcodes.PipelineParsingError]
		appErr := workflowengine.NewAppError(errCode, "no workflow definition or block provided")
		return result, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	for _, step := range steps {
		logger.Info("Running step", "id", step.ID, "use", step.Use)
		if subBlock, ok := checks[step.Use]; ok {
			childOpts := workflow.ChildWorkflowOptions{
				WorkflowID: fmt.Sprintf(
					"%s-%s",
					workflow.GetInfo(ctx).WorkflowExecution.ID,
					step.ID,
				),
				TaskQueue:         PipelineTaskQueue,
				ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
			}
			ctxChild := workflow.WithChildOptions(ctx, childOpts)
			ao := PrepareActivityOptions(
				input.WorkflowInput.ActivityOptions.RetryPolicy,
				step.Retry,
				step.Timeout,
			)

			localCfg := MergeConfigs(globalCfg, step.With.Config)
			inputs, err := ResolveSubworkflowInputs(step, subBlock, globalCfg, finalOutput)
			if err != nil {
				return result, workflowengine.NewWorkflowError(err, runMetadata)
			}
			childWorkflowInput := workflowengine.WorkflowInput{
				Config:          map[string]any{"global": localCfg},
				Payload:         inputs,
				ActivityOptions: &ao,
			}

			childInput := PipelineWorkflowInput{
				WorkflowBlock: &subBlock,
				WorkflowInput: childWorkflowInput,
			}
			var childResult workflowengine.WorkflowResult
			err = workflow.ExecuteChildWorkflow(
				ctxChild,
				w.Name(),
				childInput,
			).Get(ctxChild, &childResult)
			if err != nil {
				logger.Error(step.ID, "child workflow execution error", err)
				return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
					err,
					runMetadata,
				)
			}

			finalOutput[step.ID] = make(map[string]any)
			for k, v := range subBlock.Outputs {
				res, err := ResolveExpressions(v, childResult.Output.(map[string]any))
				if err != nil {
					errCode := errorcodes.Codes[errorcodes.PipelineParsingError]
					appErr := workflowengine.NewAppError(
						errCode,
						fmt.Sprintf(
							"error resolving expressions for step %s: %s",
							step.ID,
							err.Error(),
						),
					)
					return result, workflowengine.NewWorkflowError(appErr, runMetadata)
				}
				finalOutput[step.ID].(map[string]any)["outputs"] = make(map[string]any)
				finalOutput[step.ID].(map[string]any)["outputs"].(map[string]any)[k] = res
			}
			continue
		}

		finalOutput["inputs"] = input.WorkflowInput.Payload
		ao := PrepareActivityOptions(
			input.WorkflowInput.ActivityOptions.RetryPolicy,
			step.Retry,
			step.Timeout,
		)

		_, err := step.Execute(ctx, globalCfg, &finalOutput, ao)
		if err != nil {
			logger.Error(step.ID, "step execution error", err)
			errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("error executing step %s: %s", step.ID, err.Error()),
				step.ID,
			)
			return result, workflowengine.NewWorkflowError(appErr, runMetadata)
		}
	}
	delete(finalOutput, "inputs")
	return workflowengine.WorkflowResult{
		Output: finalOutput,
	}, nil
}

// Start launches the pipeline workflow via Temporal client
func (w *PipelineWorkflow) Start(
	inputYaml string,
	namespace string,
	app_url string,
	memo map[string]any,
) (workflowengine.WorkflowResult, error) {
	var result workflowengine.WorkflowResult

	var wfDef *WorkflowDefinition
	wfDef, err := ParseWorkflow(inputYaml)
	if err != nil {
		return result, err
	}

	options := PrepareWorkflowOptions(wfDef.Runtime)
	options.Options.Memo = memo
	options.Options.ID = fmt.Sprintf("Pipeline-%s-%s", wfDef.Name, uuid.NewString())

	if namespace != "" {
		options.Namespace = namespace
	}

	c, err := temporalclient.GetTemporalClientWithNamespace(
		options.Namespace,
	)
	if err != nil {
		return result, fmt.Errorf("unable to create client: %w", err)
	}

	input := PipelineWorkflowInput{
		WorkflowDefinition: wfDef,
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url": app_url,
				"global":  wfDef.Config,
			},
			ActivityOptions: &options.ActivityOptions,
		},
	}

	// Start the workflow execution.
	wf, err := c.ExecuteWorkflow(context.Background(), options.Options, w.Name(), input)
	if err != nil {
		return result, fmt.Errorf("failed to start workflow: %w", err)
	}
	result = workflowengine.WorkflowResult{
		WorkflowID:    wf.GetID(),
		WorkflowRunID: wf.GetRunID(),
		Message: fmt.Sprintf(
			"Workflow %s started successfully with ID %s",
			w.Name(),
			wf.GetID(),
		),
	}
	return result, nil
}
