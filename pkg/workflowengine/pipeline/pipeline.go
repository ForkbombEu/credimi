// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"context"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	temporalclient "github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const PipelineTaskQueue = "PipelineTaskQueue"

type PipelineWorkflow struct{}

type PipelineWorkflowInput struct {
	WorkflowDefinition *WorkflowDefinition          `yaml:"workflow_definition" json:"workflow_definition"`
	WorkflowInput      workflowengine.WorkflowInput `yaml:"workflow_input"      json:"workflow_input"`
	Debug              bool                         `yaml:"debug,omitempty"     json:"debug,omitempty"`
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

	errorsList := []string{}
	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	runID := workflow.GetInfo(ctx).WorkflowExecution.RunID
	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflowID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI: utils.JoinURL(
			input.WorkflowInput.Config["app_url"].(string),
			"my", "tests", "runs",
			workflowID,
			runID,
		),
	}

	result := workflowengine.WorkflowResult{}

	// Final workflow output returned
	finalOutput := map[string]any{
		"workflow-id":     workflowID,
		"workflow-run-id": runID,
	}

	defer func() {
		cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
		for _, hook := range cleanupHooks {
			if err := hook(cleanupCtx, input.WorkflowDefinition.Steps, input.WorkflowInput, &finalOutput); err != nil {
				logger.Error("cleanup hook error", "error", err)
			}
		}
	}()
	for _, hook := range setupHooks {
		if err := hook(ctx, &input.WorkflowDefinition.Steps, input.WorkflowInput); err != nil {
			return result, workflowengine.NewWorkflowError(err, runMetadata)
		}
	}
	var previousStepID string
	for _, step := range input.WorkflowDefinition.Steps {
		stepInputs := map[string]any{
			"inputs": input.WorkflowInput.Payload,
		}
		for k, v := range finalOutput {
			stepInputs[k] = v
		}
		switch step.Use {
		case "debug":
			runDebugActivity(ctx, logger, previousStepID, finalOutput, input.WorkflowInput.Payload)
			continue
		case "child-pipeline":
			logger.Info("Running step", "id", step.ID, "use", step.Use)

			childOut, err := runChildPipeline(ctx, step, input, w.Name(), stepInputs, runMetadata)
			if err != nil {
				logger.Error(step.ID, "step execution error", err)
				if step.ContinueOnError {
					if out := workflowengine.ExtractOutputFromError(err); out != nil {
						childOut = out
					}
					finalOutput[step.ID] = map[string]any{
						"outputs": childOut,
					}

					errorsList = append(errorsList, err.Error())
					continue
				}
				return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
					err,
					runMetadata,
				)
			}

			finalOutput[step.ID] = map[string]any{
				"outputs": childOut,
			}
		default:
			logger.Info("Running step", "id", step.ID, "use", step.Use)

			ao := PrepareActivityOptions(
				*input.WorkflowInput.ActivityOptions,
				step.ActivityOptions,
			)

			if step.Use == "mobile_automation" {
				step.With.Payload["run_identifier"] = getPipelineRunIdentifier(
					workflow.GetInfo(ctx).Namespace,
					workflowID,
					runID,
				)
			}
			stepOutput, err := step.Execute(ctx, input.WorkflowInput.Config, stepInputs, ao)
			if err != nil {
				logger.Error(step.ID, "step execution error", err)
				errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
				appErr := workflowengine.NewAppError(
					errCode,
					fmt.Sprintf("error executing step %s: %s", step.ID, err.Error()),
					step.ID,
				)

				if step.ContinueOnError {
					if out := workflowengine.ExtractOutputFromError(err); out != nil {
						stepOutput = out
					}
					finalOutput[step.ID] = map[string]any{"outputs": stepOutput}
					errorsList = append(errorsList, err.Error())
					continue
				}
				return result, workflowengine.NewWorkflowError(appErr, runMetadata)
			}

			finalOutput[step.ID] = map[string]any{"outputs": stepOutput}
			if input.Debug {
				runDebugActivity(ctx, logger, step.ID, finalOutput, input.WorkflowInput.Payload)
			}
			previousStepID = step.ID
		}
	}

	if len(errorsList) > 0 {
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("workflow completed with %d step errors", len(errorsList)),
		)
		return result, workflowengine.NewWorkflowError(appErr, runMetadata, errorsList)
	}

	return workflowengine.WorkflowResult{
		Output: finalOutput,
	}, nil
}

// Start launches the pipeline workflow via Temporal client
func (w *PipelineWorkflow) Start(
	inputYaml string,
	config map[string]any,
	memo map[string]any,
) (workflowengine.WorkflowResult, error) {
	var result workflowengine.WorkflowResult

	var wfDef *WorkflowDefinition
	wfDef, err := ParseWorkflow(inputYaml)
	if err != nil {
		return result, err
	}

	memo["test"] = wfDef.Name
	options := PrepareWorkflowOptions(wfDef.Runtime)
	options.Options.Memo = memo
	options.Options.ID = fmt.Sprintf(
		"Pipeline-%s-%s",
		canonify.CanonifyPlain(wfDef.Name),
		uuid.NewString(),
	)
	namespace, ok := config["namespace"].(string)
	if !ok || namespace == "" {
		return result, fmt.Errorf("namespace is required")
	}

	c, err := temporalclient.GetTemporalClientWithNamespace(
		namespace,
	)
	if err != nil {
		return result, fmt.Errorf("unable to create client: %w", err)
	}
	for k, v := range wfDef.Config {
		if _, exists := config[k]; !exists {
			config[k] = v
		}
	}

	input := PipelineWorkflowInput{
		WorkflowDefinition: wfDef,
		WorkflowInput: workflowengine.WorkflowInput{
			Config:          config,
			ActivityOptions: &options.ActivityOptions,
		},
		Debug: wfDef.Runtime.Debug,
	}

	if wfDef.Runtime.Schedule.Interval != nil {
		ctx := context.Background()
		scheduleID := fmt.Sprintf("schedule_id_%s", options.Options.ID)
		scheduleHandle, err := c.ScheduleClient().Create(ctx, client.ScheduleOptions{
			ID: scheduleID,
			Spec: client.ScheduleSpec{
				Intervals: []client.ScheduleIntervalSpec{
					{
						Every: *wfDef.Runtime.Schedule.Interval,
					},
				},
			},
			Action: &client.ScheduleWorkflowAction{
				ID:        fmt.Sprintf("scheduled_%s", options.Options.ID),
				Workflow:  w.Name(),
				TaskQueue: options.Options.TaskQueue,
				Args:      []any{input},
				Memo:      memo,
			},
		})
		if err != nil {
			return result, fmt.Errorf(
				"failed to start scheduled workflow from pipeline %s: %w",
				wfDef.Name,
				err,
			)
		}

		_, err = scheduleHandle.Describe(ctx)
		if err != nil {
			return result, fmt.Errorf(
				"failed to describe scheduled workflow from pipeline %s: %w",
				wfDef.Name,
				err,
			)
		}
		result = workflowengine.WorkflowResult{
			WorkflowID: scheduleHandle.GetID(),
			Message: fmt.Sprintf(
				"Workflow %s scheduled successfully with ID %s",
				w.Name(),
				scheduleHandle.GetID(),
			),
		}
		return result, nil
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
