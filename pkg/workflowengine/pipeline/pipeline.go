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
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	temporalclient "github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	PipelineTaskQueue = "PipelineTaskQueue"
)

type PipelineWorkflow struct{}

var pipelineTemporalClient = temporalclient.GetTemporalClientWithNamespace

type PipelineWorkflowInput struct {
	WorkflowDefinition *pipeline.WorkflowDefinition `yaml:"workflow_definition" json:"workflow_definition"`
	WorkflowInput      workflowengine.WorkflowInput `yaml:"workflow_input"      json:"workflow_input"`

	Debug         bool           `yaml:"debug,omitempty"           json:"debug,omitempty"`
	ParentRunData map[string]any `yaml:"parent_run_data,omitempty" json:"parent_run_data,omitempty"`
}

type pipelineExecutionState struct {
	errorsList     []string
	finalOutput    map[string]any
	previousStepID string
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
	if config == nil {
		config = map[string]any{}
	}
	debug := input.Debug

	cleanupErrors := []error{}

	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	runID := workflow.GetInfo(ctx).WorkflowExecution.RunID
	appURL, _ := config["app_url"].(string)
	runMetadata := &workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflowID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI: utils.JoinURL(
			appURL,
			"my", "tests", "runs",
			workflowID,
			runID,
		),
	}

	reportDone := func() {
		reportMobileRunnerSemaphoreDone(ctx, logger, config, workflowID, runID)
	}
	defer reportDone()

	if wfDef == nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			fmt.Errorf("workflow_definition is required"),
			runMetadata,
		)
	}

	if err := ValidateFinallySteps(wfDef.Finally); err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}

	result := workflowengine.WorkflowResult{}

	state := newPipelineExecutionState(workflowID, runID)

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
		finalResult := "success"

		if ctx.Err() != nil && temporal.IsCanceledError(ctx.Err()) {
			finalResult = "canceled"
		} else if len(state.errorsList) > 0 {
			finalResult = "failed"
		}
		runCleanupHooks(
			ctx,
			wfDef.Steps,
			&ao,
			config,
			runData,
			&state.finalOutput,
			logger,
			&cleanupErrors,
		)
		finalCtx, _ := workflow.NewDisconnectedContext(ctx)
		var finallyErrors []error
		runFinallySteps(
			finalCtx,
			wfDef.Finally,
			ao,
			config,
			wfDef.Name,
			runMetadata.TemporalUI,
			finalResult,
			state.finalOutput,
			logger,
			&finallyErrors,
		)
		if len(finallyErrors) > 0 {
			finallyErrorStrs := make([]string, 0, len(finallyErrors))
			for _, err := range finallyErrors {
				finallyErrorStrs = append(finallyErrorStrs, err.Error())
			}
			if state.finalOutput == nil {
				state.finalOutput = make(map[string]any)
			}
			state.finalOutput["finally_errors"] = finallyErrorStrs
			logger.Warn("Finally steps failed", "errors", finallyErrorStrs)
		}
	}()

	if err := runSetupHooks(ctx, &wfDef.Steps, &ao, config, &runData); err != nil {
		return workflowengine.WorkflowResult{}, wrapWorkflowCancellationError(err, runMetadata)
	}

	var err error
	ao, err = w.executeSteps(
		ctx,
		input,
		wfDef.Steps,
		ao,
		config,
		&runData,
		runMetadata,
		state,
		debug,
		logger,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runPendingPlayStoreDisableAfterSteps(ctx, &ao, config, &runData); err != nil {
		return workflowengine.WorkflowResult{}, wrapWorkflowCancellationError(err, runMetadata)
	}

	if len(state.errorsList) > 0 {
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("workflow completed with %d step errors", len(state.errorsList)),
		)
		return result, workflowengine.NewWorkflowError(
			appErr,
			runMetadata,
			state.errorsList,
			state.finalOutput,
		)
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
			state.finalOutput,
		)
	}

	return workflowengine.WorkflowResult{
		Output: state.finalOutput,
	}, nil
}

func newPipelineExecutionState(workflowID string, runID string) *pipelineExecutionState {
	return &pipelineExecutionState{
		errorsList: []string{},
		finalOutput: map[string]any{
			"workflow-id":     workflowID,
			"workflow-run-id": runID,
			"result_video_warning": "Video recordings are limited to 30 minutes. " +
				"Tests exceeding this duration may result in an incomplete video.",
		},
	}
}

func wrapWorkflowCancellationError(
	err error,
	runMetadata *workflowengine.WorkflowErrorMetadata,
) error {
	if temporal.IsCanceledError(err) {
		return workflowengine.NewWorkflowCancellationError(runMetadata)
	}

	return err
}

func (w *PipelineWorkflow) executeSteps(
	ctx workflow.Context,
	input PipelineWorkflowInput,
	steps []pipeline.StepDefinition,
	ao workflow.ActivityOptions,
	config map[string]any,
	runData *map[string]any,
	runMetadata *workflowengine.WorkflowErrorMetadata,
	state *pipelineExecutionState,
	debug bool,
	logger log.Logger,
) (workflow.ActivityOptions, error) {
	for _, step := range steps {
		nextAO, err := w.executeStep(
			ctx,
			input,
			step,
			ao,
			config,
			runData,
			runMetadata,
			state,
			debug,
			logger,
		)
		if err != nil {
			return nextAO, err
		}
		ao = nextAO
	}

	return ao, nil
}

func (w *PipelineWorkflow) executeStep(
	ctx workflow.Context,
	input PipelineWorkflowInput,
	step pipeline.StepDefinition,
	ao workflow.ActivityOptions,
	config map[string]any,
	runData *map[string]any,
	runMetadata *workflowengine.WorkflowErrorMetadata,
	state *pipelineExecutionState,
	debug bool,
	logger log.Logger,
) (workflow.ActivityOptions, error) {
	if err := runPendingPlayStoreDisableIfNeeded(ctx, step, &ao, config, runData); err != nil {
		return ao, wrapWorkflowCancellationError(err, runMetadata)
	}

	stepInputs := buildPipelineStepInputs(
		state.finalOutput,
		workflowengine.AsMap(input.WorkflowInput.Payload),
	)

	switch step.Use {
	case "debug":
		runDebugActivity(
			ctx,
			logger,
			state.previousStepID,
			state.finalOutput,
			input.WorkflowInput.Payload,
		)
		return ao, nil
	case "child-pipeline":
		return ao, w.executeChildPipelineStep(
			ctx,
			input,
			step,
			stepInputs,
			ao,
			config,
			runMetadata,
			state,
			logger,
		)
	default:
		return w.executeRegularStep(
			ctx,
			input,
			step,
			stepInputs,
			ao,
			config,
			runMetadata,
			state,
			debug,
			logger,
		)
	}
}

func buildPipelineStepInputs(finalOutput map[string]any, payload map[string]any) map[string]any {
	stepInputs := map[string]any{
		"inputs": payload,
	}
	for k, v := range finalOutput {
		stepInputs[k] = v
	}

	return stepInputs
}

func buildEnrichedStepInputs(
	ctx workflow.Context,
	payload map[string]any,
	finalOutput map[string]any,
	pipelineName string,
	pipelineURL string,
	hasErrors bool,
) map[string]any {
	return enrichDataContext(
		buildPipelineStepInputs(
			finalOutput,
			payload,
		),
		pipelineName,
		pipelineURL,
		hasErrors,
		workflow.Now(ctx).Format(time.RFC3339),
	)
}

func (w *PipelineWorkflow) executeChildPipelineStep(
	ctx workflow.Context,
	input PipelineWorkflowInput,
	step pipeline.StepDefinition,
	stepInputs map[string]any,
	ao workflow.ActivityOptions,
	config map[string]any,
	runMetadata *workflowengine.WorkflowErrorMetadata,
	state *pipelineExecutionState,
	logger log.Logger,
) error {
	logger.Info("Running step", "id", step.ID, "use", step.Use)

	pipelineName := input.WorkflowDefinition.Name
	pipelineURL := runMetadata.TemporalUI
	payload := workflowengine.AsMap(input.WorkflowInput.Payload)
	stepInputs = buildEnrichedStepInputs(
		ctx,
		payload,
		state.finalOutput,
		pipelineName,
		pipelineURL,
		len(state.errorsList) > 0,
	)
	childOut, err := runChildPipeline(ctx, step, input, w.Name(), stepInputs, runMetadata)
	if err != nil {
		return handleChildPipelineStepError(
			ctx,
			step,
			payload,
			childOut,
			err,
			ao,
			config,
			runMetadata,
			state,
			logger,
			pipelineName,
			pipelineURL,
		)
	}

	state.finalOutput[step.ID] = map[string]any{
		"outputs": childOut,
	}
	successInputs := buildEnrichedStepInputs(
		ctx,
		payload,
		state.finalOutput,
		pipelineName,
		pipelineURL,
		len(state.errorsList) > 0,
	)
	runStepSuccessHooks(ctx, step, successInputs, state.errorsList, ao, config, logger)

	return nil
}

func handleChildPipelineStepError(
	ctx workflow.Context,
	step pipeline.StepDefinition,
	payload map[string]any,
	childOut any,
	err error,
	ao workflow.ActivityOptions,
	config map[string]any,
	runMetadata *workflowengine.WorkflowErrorMetadata,
	state *pipelineExecutionState,
	logger log.Logger,
	pipelineName string,
	pipelineURL string,
) error {
	if temporal.IsTimeoutError(err) {
		return err
	}
	if temporal.IsCanceledError(err) {
		return workflowengine.NewWorkflowCancellationError(runMetadata)
	}

	logger.Error(step.ID, "step execution error", err)
	if childOut != nil {
		state.finalOutput[step.ID] = map[string]any{"outputs": childOut}
	}
	errorInputs := buildEnrichedStepInputs(
		ctx,
		payload,
		state.finalOutput,
		pipelineName,
		pipelineURL,
		true,
	)
	runStepErrorHooks(ctx, step, errorInputs, state.errorsList, ao, config, logger)
	if step.ContinueOnError {
		if out := workflowengine.ExtractOutputFromError(err); out != nil {
			childOut = out
		}
		state.finalOutput[step.ID] = map[string]any{
			"outputs": childOut,
		}
		state.errorsList = append(state.errorsList, err.Error())
		return nil
	}

	return workflowengine.NewWorkflowError(err, runMetadata)
}

func (w *PipelineWorkflow) executeRegularStep(
	ctx workflow.Context,
	input PipelineWorkflowInput,
	step pipeline.StepDefinition,
	stepInputs map[string]any,
	ao workflow.ActivityOptions,
	config map[string]any,
	runMetadata *workflowengine.WorkflowErrorMetadata,
	state *pipelineExecutionState,
	debug bool,
	logger log.Logger,
) (workflow.ActivityOptions, error) {
	logger.Info("Running step", "id", step.ID, "use", step.Use)

	ao = PrepareActivityOptions(ao, step.ActivityOptions)

	pipelineName := input.WorkflowDefinition.Name
	pipelineURL := runMetadata.TemporalUI
	payload := workflowengine.AsMap(input.WorkflowInput.Payload)
	enrichedStepInputs := buildEnrichedStepInputs(
		ctx,
		payload,
		state.finalOutput,
		pipelineName,
		pipelineURL,
		len(state.errorsList) > 0,
	)

	stepOutput, err := Execute(&step, ctx, config, enrichedStepInputs, ao)
	if err != nil {
		if stepOutput != nil {
			state.finalOutput[step.ID] = map[string]any{"outputs": stepOutput}
		}
		return ao, handleRegularStepError(
			ctx,
			step,
			payload,
			stepOutput,
			err,
			ao,
			config,
			runMetadata,
			state,
			logger,
			pipelineName,
			pipelineURL,
		)
	}

	state.finalOutput[step.ID] = map[string]any{"outputs": stepOutput}
	successInputs := buildEnrichedStepInputs(
		ctx,
		payload,
		state.finalOutput,
		pipelineName,
		pipelineURL,
		len(state.errorsList) > 0,
	)

	runStepSuccessHooks(ctx, step, successInputs, state.errorsList, ao, config, logger)
	if debug {
		runDebugActivity(ctx, logger, step.ID, state.finalOutput, input.WorkflowInput.Payload)
	}
	state.previousStepID = step.ID

	return ao, nil
}

func handleRegularStepError(
	ctx workflow.Context,
	step pipeline.StepDefinition,
	payload map[string]any,
	stepOutput any,
	err error,
	ao workflow.ActivityOptions,
	config map[string]any,
	runMetadata *workflowengine.WorkflowErrorMetadata,
	state *pipelineExecutionState,
	logger log.Logger,
	pipelineName string,
	pipelineURL string,
) error {
	if temporal.IsCanceledError(err) {
		return workflowengine.NewWorkflowCancellationError(runMetadata)
	}

	logger.Error(step.ID, "step execution error", err)
	if stepOutput != nil {
		state.finalOutput[step.ID] = map[string]any{"outputs": stepOutput}
	}
	errorInputs := buildEnrichedStepInputs(
		ctx,
		payload,
		state.finalOutput,
		pipelineName,
		pipelineURL,
		true,
	)
	runStepErrorHooks(ctx, step, errorInputs, state.errorsList, ao, config, logger)

	if step.ContinueOnError {
		state.errorsList = append(state.errorsList, err.Error())
		return nil
	}
	errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
	appErr := workflowengine.NewAppError(
		errCode,
		fmt.Sprintf("error executing step %s: %s", step.ID, err.Error()),
		step.ID,
		state.finalOutput,
	)
	return workflowengine.NewWorkflowError(appErr, runMetadata)
}

func runStepErrorHooks(
	ctx workflow.Context,
	step pipeline.StepDefinition,
	stepInputs map[string]any,
	errorsList []string,
	ao workflow.ActivityOptions,
	config map[string]any,
	logger log.Logger,
) {
	if len(step.OnError) == 0 {
		return
	}

	logger.Info(
		"Executing onError steps for step",
		"step_id",
		step.ID,
		"count",
		len(step.OnError),
		"continue_on_error",
		step.ContinueOnError,
	)
	ExecuteEventStepsOnError(ctx, step.OnError, stepInputs, errorsList, ao, config)
}

func runStepSuccessHooks(
	ctx workflow.Context,
	step pipeline.StepDefinition,
	stepInputs map[string]any,
	errorsList []string,
	ao workflow.ActivityOptions,
	config map[string]any,
	logger log.Logger,
) {
	if len(step.OnSuccess) == 0 {
		return
	}

	logger.Info(
		"Executing onSuccess steps for step",
		"step_id",
		step.ID,
		"count",
		len(step.OnSuccess),
	)
	ExecuteEventStepsOnSuccess(ctx, step.OnSuccess, stepInputs, errorsList, ao, config)
}

// Start launches the pipeline workflow via Temporal client
func (w *PipelineWorkflow) Start(
	inputYaml string,
	config map[string]any,
	memo map[string]any,
	pipelineIdentifier string,
) (workflowengine.WorkflowResult, error) {
	var result workflowengine.WorkflowResult

	var wfDef *pipeline.WorkflowDefinition
	wfDef, err := pipeline.ParseWorkflow(inputYaml)
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

	c, err := pipelineTemporalClient(
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

	runnerInfo, _ := ParsePipelineRunnerInfo(inputYaml)
	// Add global_runner_id to config if specified
	if wfDef.Runtime.GlobalRunnerID != "" {
		config["global_runner_id"] = wfDef.Runtime.GlobalRunnerID
	}
	globalRunnerID := GlobalRunnerIDFromConfig(config)
	runnerIDs := RunnerIDsWithGlobal(runnerInfo, globalRunnerID)
	config["disable_android_play_store"] = wfDef.Runtime.DisableAndroidPlayStore
	entityIDs, err := pipeline.ParseEntityIDs(inputYaml)
	if err != nil {
		return result, fmt.Errorf("failed to parse entity IDs: %w", err)
	}

	workflowengine.ApplyPipelineSearchAttributes(&options.Options, pipelineIdentifier, runnerIDs, entityIDs)

	input := PipelineWorkflowInput{
		WorkflowDefinition: wfDef,
		WorkflowInput: workflowengine.WorkflowInput{
			Config:          config,
			ActivityOptions: &options.ActivityOptions,
		},
		Debug: wfDef.Runtime.Debug,
	}

	if wfDef.Runtime.Schedule.Interval != nil {
		searchAttributes := workflowengine.PipelineTypedSearchAttributes(pipelineIdentifier, runnerIDs, entityIDs)
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
				ID:                    fmt.Sprintf("scheduled_%s", options.Options.ID),
				Workflow:              w.Name(),
				TaskQueue:             options.Options.TaskQueue,
				Args:                  []any{input},
				Memo:                  memo,
				TypedSearchAttributes: searchAttributes,
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
	eventSteps []*pipeline.OnErrorStepDefinition,
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

		_, execErr := ExecuteOnError(eventStep, ctx, config, stepInputs, aO)
		if execErr != nil {
			errorsList = append(errorsList, execErr.Error())
		}
	}
	return errorsList
}

func ExecuteEventStepsOnSuccess(
	ctx workflow.Context,
	eventSteps []*pipeline.OnSuccessStepDefinition,
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

		_, execErr := ExecuteOnSuccess(eventStep, ctx, config, stepInputs, aO)
		if execErr != nil {
			errorsList = append(errorsList, execErr.Error())
		}
	}
	return errorsList
}

func runFinallySteps(
	ctx workflow.Context,
	finallySteps []pipeline.StepDefinition,
	ao workflow.ActivityOptions,
	config map[string]any,
	pipelineName string,
	pipelineURL string,
	finalResult string,
	finalOutput map[string]any,
	logger log.Logger,
	errorList *[]error,
) {
	if len(finallySteps) == 0 {
		return
	}
	logger.Info("Executing finally steps", "count", len(finallySteps))

	stepInputs := buildPipelineStepInputs(finalOutput, workflowengine.AsMap(config))
	for _, step := range finallySteps {
		logger.Info("Running finally step", "id", step.ID, "use", step.Use)

		enrichedStepInputs := enrichDataContext(
			stepInputs,
			pipelineName,
			pipelineURL,
			false,
			workflow.Now(ctx).Format(time.RFC3339),
		)
		enrichedStepInputs["result"] = finalResult

		_, err := Execute(&step, ctx, config, enrichedStepInputs, ao)
		if err != nil {
			logger.Error("Finally step filed", "step_id", step.ID, "error", err)
			if errorList != nil {
				*errorList = append(*errorList, err)
			}
		}
	}
}

func ValidateFinallySteps(finallySteps []pipeline.StepDefinition) error {
	allowedTypes := map[string]bool{
		"email":        true,
		"http-request": true,
	}

	for _, step := range finallySteps {
		if !allowedTypes[step.Use] {
			return fmt.Errorf("finally step '%s' uses '%s' which is not allowed. Only email and http-request are allowed",
				step.ID, step.Use)
		}
	}
	return nil
}
