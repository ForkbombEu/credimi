// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	temporalclient "github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	PipelineTaskQueue = "PipelineTaskQueue"
)

type PipelineWorkflow struct{}

type PipelineWorkflowInput struct {
	WorkflowDefinition *WorkflowDefinition          `yaml:"workflow_definition" json:"workflow_definition"`
	WorkflowInput      workflowengine.WorkflowInput `yaml:"workflow_input"      json:"workflow_input"`

	Debug         bool           `yaml:"debug,omitempty"           json:"debug,omitempty"`
	Scheduled     bool           `yaml:"scheduled,omitempty"       json:"scheduled,omitempty"`
	ParentRunData map[string]any `yaml:"parent_run_data,omitempty" json:"parent_run_data,omitempty"`
}

func NewPipelineWorkflow() *PipelineWorkflow {
	return &PipelineWorkflow{}
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

	var ao workflow.ActivityOptions

	if input.WorkflowInput.ActivityOptions != nil {
		ao = *input.WorkflowInput.ActivityOptions
		ctx = workflow.WithActivityOptions(ctx, ao)
	}

	wfDef := input.WorkflowDefinition
	config := input.WorkflowInput.Config
	debug := input.Debug

	errorsList := []string{}
	cleanupErrors := []error{}

	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	runID := workflow.GetInfo(ctx).WorkflowExecution.RunID
	runMetadata := &workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflowID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI: utils.JoinURL(
			config["app_url"].(string),
			"my", "tests", "runs",
			workflowID,
			runID,
		),
	}

	if input.Scheduled {
		var err error
		ctx, wfDef, ao, debug, err = w.handleScheduledRun(
			ctx,
			input,
			config,
			runMetadata,
		)
		if err != nil {
			return workflowengine.WorkflowResult{}, err
		}
	}

	result := workflowengine.WorkflowResult{}

	// Final workflow output returned
	finalOutput := map[string]any{
		"workflow-id":     workflowID,
		"workflow-run-id": runID,
		"result_video_warning": "Video recordings are limited to 30 minutes. " +
			"Tests exceeding this duration may result in an incomplete video.",
	}

	runData := map[string]any{
		"run_identifier": getPipelineRunIdentifier(
			workflow.GetInfo(ctx).Namespace,
			workflowID,
			runID,
		),
	}

	if input.ParentRunData != nil {
		// For child pipelines, inherit parent run data
		runData = input.ParentRunData
	}
	defer func() {
		runCleanupHooks(
			ctx,
			wfDef.Steps,
			&ao,
			config,
			runData,
			&finalOutput,
			logger,
			&cleanupErrors,
		)
	}()

	if err := runSetupHooks(ctx, &wfDef.Steps, &ao, config, &runData); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	var previousStepID string
	for _, step := range wfDef.Steps {
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
				if temporal.IsTimeoutError(err) {
					return workflowengine.WorkflowResult{}, err
				}

				if temporal.IsCanceledError(err) {
					return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowCancellationError(
						runMetadata,
					)
				}
				logger.Error(step.ID, "step execution error", err)
				if len(step.OnError) > 0 {
					logger.Info("Executing onError steps for step",
						"step_id", step.ID,
						"count", len(step.OnError),
						"continue_on_error", step.ContinueOnError)

					ExecuteEventStepsOnError(ctx, step.OnError, stepInputs, errorsList, ao, config)
				}
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
			if len(step.OnSuccess) > 0 {
				logger.Info(
					"Executing onSuccess steps for step",
					"step_id",
					step.ID,
					"count",
					len(step.OnSuccess),
				)
				ExecuteEventStepsOnSuccess(ctx, step.OnSuccess, stepInputs, errorsList, ao, config)
			}

			finalOutput[step.ID] = map[string]any{
				"outputs": childOut,
			}
		default:
			logger.Info("Running step", "id", step.ID, "use", step.Use)

			ao = PrepareActivityOptions(
				ao,
				step.ActivityOptions,
			)

			stepOutput, err := step.Execute(ctx, config, stepInputs, ao)
			if err != nil {
				if temporal.IsCanceledError(err) {
					return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowCancellationError(
						runMetadata,
					)
				}
				logger.Error(step.ID, "step execution error", err)
				errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
				appErr := workflowengine.NewAppError(
					errCode,
					fmt.Sprintf("error executing step %s: %s", step.ID, err.Error()),
					step.ID,
					finalOutput,
				)
				if len(step.OnError) > 0 {
					logger.Info("Executing onError steps for step",
						"step_id", step.ID,
						"count", len(step.OnError),
						"continue_on_error", step.ContinueOnError)

					ExecuteEventStepsOnError(ctx, step.OnError, stepInputs, errorsList, ao, config)
				}
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

			if len(step.OnSuccess) > 0 {
				logger.Info(
					"Executing onSuccess steps for step",
					"step_id",
					step.ID,
					"count",
					len(step.OnSuccess),
				)
				ExecuteEventStepsOnSuccess(ctx, step.OnSuccess, stepInputs, errorsList, ao, config)
			}

			finalOutput[step.ID] = map[string]any{"outputs": stepOutput}
			if debug {
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
		return result, workflowengine.NewWorkflowError(appErr, runMetadata, errorsList, finalOutput)
	}

	if len(cleanupErrors) > 0 {
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("workflow completed with %d cleanup errors", len(cleanupErrors)),
		)
		return result, workflowengine.NewWorkflowError(
			appErr,
			runMetadata,
			cleanupErrors,
			finalOutput,
		)
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

func ExecuteEventStepsOnError(
	ctx workflow.Context,
	eventSteps []*OnErrorStepDefinition,
	stepInputs map[string]any,
	existingErrors []string,
	ao workflow.ActivityOptions,
	config map[string]any,
) []string {
	errorsList := existingErrors
	if errorsList == nil {
		errorsList = []string{}
	}
	for _, eventStep := range eventSteps {
		aO := PrepareActivityOptions(
			ao,
			eventStep.ActivityOptions,
		)

		_, execErr := eventStep.ExecuteOnError(ctx, config, stepInputs, aO)
		if execErr != nil {
			errorsList = append(errorsList, execErr.Error())
		}
	}
	return errorsList
}

func ExecuteEventStepsOnSuccess(
	ctx workflow.Context,
	eventSteps []*OnSuccessStepDefinition,
	stepInputs map[string]any,
	existingErrors []string,
	ao workflow.ActivityOptions,
	config map[string]any,
) []string {
	errorsList := existingErrors
	if errorsList == nil {
		errorsList = []string{}
	}
	for _, eventStep := range eventSteps {
		aO := PrepareActivityOptions(
			ao,
			eventStep.ActivityOptions,
		)

		_, execErr := eventStep.ExecuteOnSuccess(ctx, config, stepInputs, aO)
		if execErr != nil {
			errorsList = append(errorsList, execErr.Error())
		}
	}
	return errorsList
}

func (w *PipelineWorkflow) handleScheduledRun(
	ctx workflow.Context,
	input PipelineWorkflowInput,
	config map[string]any,
	runMetadata *workflowengine.WorkflowErrorMetadata,
) (
	workflow.Context,
	*WorkflowDefinition,
	workflow.ActivityOptions,
	bool,
	error,
) {
	pipelineID, ok := input.WorkflowInput.Payload.(map[string]any)["pipeline_id"].(string)
	if !ok || pipelineID == "" {
		return ctx, nil, workflow.ActivityOptions{}, false,
			workflowengine.NewMissingOrInvalidPayloadError(
				fmt.Errorf("missing pipeline_id"),
				runMetadata,
			)
	}

	httpCtx := workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{
			ScheduleToCloseTimeout: time.Minute,
			StartToCloseTimeout:    30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    time.Second,
				BackoffCoefficient: 1.0,
				MaximumInterval:    time.Minute,
				MaximumAttempts:    1,
			},
		},
	)

	httpActivity := activities.NewHTTPActivity()

	// --- Validate pipeline identifier
	recRequest := workflowengine.ActivityInput{
		Payload: map[string]any{
			"method": "POST",
			"url": utils.JoinURL(
				config["app_url"].(string),
				"api", "canonify", "identifier", "validate",
			),
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
			"body": map[string]any{
				"canonified_name": pipelineID,
			},
			"expected_status": 200,
		},
	}

	var recResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(httpCtx, httpActivity.Name(), recRequest).
		Get(httpCtx, &recResult); err != nil {
		return ctx, nil, workflow.ActivityOptions{}, false,
			workflowengine.NewWorkflowError(err, runMetadata)
	}

	rec, ok := recResult.Output.(map[string]any)["body"].(map[string]any)["record"].(map[string]any)
	if !ok {
		return ctx, nil, workflow.ActivityOptions{}, false,
			workflowengine.NewWorkflowError(
				workflowengine.NewAppError(
					errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
					"missing 'record' in response",
				),
				runMetadata,
			)
	}

	pipelineYaml, ok := rec["yaml"].(string)
	if !ok {
		return ctx, nil, workflow.ActivityOptions{}, false,
			workflowengine.NewWorkflowError(
				workflowengine.NewAppError(
					errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
					"missing 'yaml' in response",
				),
				runMetadata,
			)
	}

	wfDef, err := ParseWorkflow(pipelineYaml)
	if err != nil {
		return ctx, nil, workflow.ActivityOptions{}, false,
			workflowengine.NewWorkflowError(
				workflowengine.NewAppError(
					errorcodes.Codes[errorcodes.PipelineParsingError],
					err.Error(),
				),
				runMetadata,
			)
	}

	debug := wfDef.Runtime.Debug

	for k, v := range wfDef.Config {
		if _, exists := config[k]; !exists {
			config[k] = v
		}
	}

	options := PrepareWorkflowOptions(wfDef.Runtime)
	ctx = workflow.WithActivityOptions(ctx, options.ActivityOptions)

	createRequest := workflowengine.ActivityInput{
		Payload: map[string]any{
			"method": "POST",
			"url": utils.JoinURL(
				config["app_url"].(string),
				"api", "pipeline", "pipeline-execution-results",
			),
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
			"body": map[string]any{
				"owner":       config["namespace"].(string),
				"pipeline_id": pipelineID,
				"workflow_id": workflow.GetInfo(ctx).WorkflowExecution.ID,
				"run_id":      workflow.GetInfo(ctx).WorkflowExecution.RunID,
			},
			"expected_status": 200,
		},
	}

	if err := workflow.ExecuteActivity(httpCtx, httpActivity.Name(), createRequest).
		Get(httpCtx, nil); err != nil {
		return ctx, nil, workflow.ActivityOptions{}, false,
			workflowengine.NewWorkflowError(err, runMetadata)
	}

	return ctx, wfDef, options.ActivityOptions, debug, nil
}
