// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"context"
	"fmt"
	"strings"
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

	childPipelineStepUse = "child-pipeline"
	httpRequestStepUse   = "http-request"
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
	failures       []pipelineStepFailure
	finalOutput    map[string]any
	previousStepID string
}

type pipelineStepFailure struct {
	StepID  string
	Message string
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

const (
	resultSuccess  = "success"
	resultFailed   = "failed"
	resultCanceled = "canceled"
)

// Workflow executes all steps in the workflow definition sequentially
func (w *PipelineWorkflow) Workflow(
	ctx workflow.Context,
	input PipelineWorkflowInput,
) (result workflowengine.WorkflowResult, finalErr error) {
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
	runMetadata := &workflowengine.WorkflowRunMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflowID,
		RunID:        runID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI: utils.JoinURL(
			appURL,
			"my", "tests", "runs",
			workflowID,
			runID,
		),
	}

	defer func() {
		finalResult := pipelineFinalResult(ctx, finalErr)
		reportGitHubPRCommentDone(
			ctx,
			logger,
			config,
			workflowID,
			runID,
			finalResult,
		)
		reportMobileRunnerSemaphoreDone(
			ctx,
			logger,
			config,
			workflowID,
			runID,
			finalResult,
		)
	}()

	if wfDef == nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			fmt.Errorf("workflow_definition is required"),
			runMetadata,
		)
	}

	if err := ValidateFinallySteps(wfDef.Finally); err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}

	state := newPipelineExecutionState(workflowID, runID)
	if hasMobileAutomationStep(wfDef.Steps) {
		state.finalOutput["result_video_warning"] = "Video recordings are limited to 30 minutes. " +
			"Tests exceeding this duration may result in an incomplete video."
	}

	runData := map[string]any{
		"run_identifier": getPipelineRunIdentifier(
			workflow.GetInfo(ctx).Namespace,
			workflowID,
			runID,
		),
	}

	if input.ParentRunData != nil {
		runData = input.ParentRunData
	}

	defer func() {
		finalResult := pipelineFinalResult(ctx, finalErr)

		runCleanupHooks(
			ctx,
			wfDef,
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
			workflowengine.AsMap(input.WorkflowInput.Payload),
			wfDef.Name,
			runMetadata.TemporalUI,
			finalResult,
			buildFinallyPipelineOutput(state.finalOutput, runMetadata.TemporalUI, finalErr),
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

	if err := runSetupHooks(ctx, wfDef, config, &runData, &state.finalOutput, logger); err != nil {
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
		finalErr = err
		return workflowengine.WorkflowResult{}, finalErr
	}

	if err := runPendingPlayStoreDisableAfterSteps(ctx, &ao, config, &runData); err != nil {
		finalErr = wrapWorkflowCancellationError(err, runMetadata)
		return workflowengine.WorkflowResult{}, finalErr
	}

	if len(state.failures) > 0 {
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		appErr := workflowengine.NewAppError(workflowengine.WorkflowError{
			Code:    errCode.Code,
			Summary: buildPipelineFailureSummary(state.failures),
			Details: map[string]any{
				"errors": buildPipelineFailureErrors(state.failures),
				"output": state.finalOutput,
			},
		})

		finalErr = workflowengine.NewWorkflowError(
			appErr,
			runMetadata,
		)
		result = workflowengine.WorkflowResult{}
		return result, finalErr
	}

	if len(cleanupErrors) > 0 {
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		appErr := workflowengine.NewAppError(workflowengine.WorkflowError{
			Code:    errCode.Code,
			Summary: fmt.Sprintf("workflow completed with %d cleanup errors", len(cleanupErrors)),
			Details: map[string]any{
				"errors": buildPipelineCleanupFailureErrors(cleanupErrors),
				"output": state.finalOutput,
			},
		})

		finalErr = workflowengine.NewWorkflowError(
			appErr,
			runMetadata,
		)
		result = workflowengine.WorkflowResult{}
		return result, finalErr
	}

	result = workflowengine.WorkflowResult{
		WorkflowID:    workflowID,
		WorkflowRunID: runID,
		Output:        state.finalOutput,
	}
	return result, finalErr
}

func pipelineFinalResult(ctx workflow.Context, finalErr error) string {
	if ctx.Err() != nil && temporal.IsCanceledError(ctx.Err()) {
		return resultCanceled
	}
	if finalErr != nil {
		return resultFailed
	}
	return resultSuccess
}

func buildPipelineFailureSummary(failures []pipelineStepFailure) string {
	var base string
	if len(failures) == 1 {
		base = "Pipeline failed: 1 step failed"
	} else {
		base = fmt.Sprintf("Pipeline failed: %d steps failed", len(failures))
	}

	stepIDs := uniqueOrderedStepIDs(failures)
	if len(stepIDs) == 0 {
		return base
	}

	display := stepIDs
	if len(display) > 3 {
		display = display[:3]
	}

	summary := fmt.Sprintf("%s (%s", base, strings.Join(display, ", "))
	if len(stepIDs) > len(display) {
		summary = fmt.Sprintf("%s, +%d more", summary, len(stepIDs)-len(display))
	}
	return summary + ")"
}

func buildPipelineFailureErrors(stepFailures []pipelineStepFailure) []workflowengine.WorkflowError {
	failures := make([]workflowengine.WorkflowError, 0, len(stepFailures))
	for _, failure := range stepFailures {
		failures = append(failures, workflowengine.WorkflowError{
			Message: failure.Message,
			Details: map[string]any{
				"step_id": failure.StepID,
			},
		})
	}
	return failures
}

func uniqueOrderedStepIDs(failures []pipelineStepFailure) []string {
	stepIDs := make([]string, 0, len(failures))
	seen := make(map[string]struct{}, len(failures))
	for _, failure := range failures {
		if failure.StepID == "" {
			continue
		}
		if _, ok := seen[failure.StepID]; ok {
			continue
		}
		seen[failure.StepID] = struct{}{}
		stepIDs = append(stepIDs, failure.StepID)
	}
	return stepIDs
}

func buildPipelineCleanupFailureErrors(errorsList []error) []workflowengine.WorkflowError {
	failures := make([]workflowengine.WorkflowError, 0, len(errorsList))
	for _, err := range errorsList {
		failures = append(failures, workflowengine.WorkflowError{
			Message: err.Error(),
		})
	}
	return failures
}

func newPipelineExecutionState(workflowID string, runID string) *pipelineExecutionState {
	return &pipelineExecutionState{
		failures: []pipelineStepFailure{},
		finalOutput: map[string]any{
			"workflow-id":     workflowID,
			"workflow-run-id": runID,
		},
	}
}

func hasMobileAutomationStep(steps []pipeline.StepDefinition) bool {
	for _, step := range steps {
		if step.Use == mobileAutomationStepUse {
			return true
		}
	}
	return false
}

func wrapWorkflowCancellationError(
	err error,
	runMetadata *workflowengine.WorkflowRunMetadata,
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
	runMetadata *workflowengine.WorkflowRunMetadata,
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
	runMetadata *workflowengine.WorkflowRunMetadata,
	state *pipelineExecutionState,
	debug bool,
	logger log.Logger,
) (workflow.ActivityOptions, error) {
	if err := runPendingPlayStoreDisableIfNeeded(ctx, step, &ao, config, runData); err != nil {
		return ao, wrapWorkflowCancellationError(err, runMetadata)
	}

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
	case childPipelineStepUse:
		return ao, w.executeChildPipelineStep(
			ctx,
			input,
			step,
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

func buildFinallyPipelineOutput(
	finalOutput map[string]any,
	pipelineURL string,
	finalErr error,
) map[string]any {
	pipelineOutput := map[string]any{
		"outputs": collectFinallyOutputs(finalOutput),
		"metadata": map[string]any{
			"pipeline_url":    pipelineURL,
			"workflow_id":     finalOutput["workflow-id"],
			"workflow_run_id": finalOutput["workflow-run-id"],
		},
	}

	if warning, ok := finalOutput["result_video_warning"]; ok {
		pipelineOutput["metadata"].(map[string]any)["result_video_warning"] = warning
	}
	if warnings, ok := finalOutput["cleanup_warnings"]; ok {
		pipelineOutput["cleanup_warnings"] = warnings
	}
	if warnings, ok := finalOutput[setupWarningsOutputKey]; ok {
		pipelineOutput[setupWarningsOutputKey] = warnings
	}
	if finalErr != nil {
		pipelineOutput["error"] = finalErr.Error()
	}

	return pipelineOutput
}

func collectFinallyOutputs(finalOutput map[string]any) map[string]any {
	outputs := make(map[string]any)

	for key, value := range finalOutput {
		switch key {
		case "workflow-id",
			"workflow-run-id",
			"result_video_warning",
			setupWarningsOutputKey,
			"cleanup_warnings",
			"finally_errors":
			continue
		}

		outputs[key] = value
	}

	return outputs
}

func (w *PipelineWorkflow) executeChildPipelineStep(
	ctx workflow.Context,
	input PipelineWorkflowInput,
	step pipeline.StepDefinition,
	ao workflow.ActivityOptions,
	config map[string]any,
	runMetadata *workflowengine.WorkflowRunMetadata,
	state *pipelineExecutionState,
	logger log.Logger,
) error {
	logger.Info("Running step", "id", step.ID, "use", step.Use)

	pipelineName := input.WorkflowDefinition.Name
	pipelineURL := runMetadata.TemporalUI
	payload := workflowengine.AsMap(input.WorkflowInput.Payload)
	stepInputs := buildEnrichedStepInputs(
		ctx,
		payload,
		state.finalOutput,
		pipelineName,
		pipelineURL,
		len(state.failures) > 0,
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
		len(state.failures) > 0,
	)
	state.failures = runStepSuccessHooks(
		ctx,
		step,
		successInputs,
		state.failures,
		ao,
		config,
		logger,
	)

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
	runMetadata *workflowengine.WorkflowRunMetadata,
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
	if step.ContinueOnError {
		state.failures = append(state.failures, pipelineStepFailure{
			StepID:  step.ID,
			Message: err.Error(),
		})
		if out := workflowengine.ExtractOutputFromError(err); out != nil {
			childOut = out
		}
		state.finalOutput[step.ID] = map[string]any{
			"outputs": childOut,
		}
		state.failures = runStepErrorHooks(
			ctx,
			step,
			errorInputs,
			state.failures,
			ao,
			config,
			logger,
		)
		return nil
	}

	state.failures = runStepErrorHooks(
		ctx,
		step,
		errorInputs,
		state.failures,
		ao,
		config,
		logger,
	)

	return workflowengine.NewWorkflowError(err, runMetadata)
}

func (w *PipelineWorkflow) executeRegularStep(
	ctx workflow.Context,
	input PipelineWorkflowInput,
	step pipeline.StepDefinition,
	ao workflow.ActivityOptions,
	config map[string]any,
	runMetadata *workflowengine.WorkflowRunMetadata,
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
		len(state.failures) > 0,
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
		len(state.failures) > 0,
	)

	state.failures = runStepSuccessHooks(
		ctx,
		step,
		successInputs,
		state.failures,
		ao,
		config,
		logger,
	)
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
	runMetadata *workflowengine.WorkflowRunMetadata,
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
	if step.ContinueOnError {
		state.failures = append(state.failures, pipelineStepFailure{
			StepID:  step.ID,
			Message: err.Error(),
		})
		state.failures = runStepErrorHooks(
			ctx,
			step,
			errorInputs,
			state.failures,
			ao,
			config,
			logger,
		)
		return nil
	}
	state.failures = runStepErrorHooks(
		ctx,
		step,
		errorInputs,
		state.failures,
		ao,
		config,
		logger,
	)
	errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
	appErr := workflowengine.NewAppError(
		workflowengine.WorkflowError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: fmt.Sprintf("error executing step %s: %s", step.ID, err.Error()),
			Details: map[string]any{
				"step_id": step.ID,
				"output":  state.finalOutput,
			},
		},
	)
	return workflowengine.NewWorkflowError(appErr, runMetadata)
}

func runStepErrorHooks(
	ctx workflow.Context,
	step pipeline.StepDefinition,
	stepInputs map[string]any,
	failures []pipelineStepFailure,
	ao workflow.ActivityOptions,
	config map[string]any,
	logger log.Logger,
) []pipelineStepFailure {
	if len(step.OnError) == 0 {
		return failures
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
	return ExecuteEventStepsOnError(ctx, step.OnError, stepInputs, failures, ao, config)
}

func runStepSuccessHooks(
	ctx workflow.Context,
	step pipeline.StepDefinition,
	stepInputs map[string]any,
	failures []pipelineStepFailure,
	ao workflow.ActivityOptions,
	config map[string]any,
	logger log.Logger,
) []pipelineStepFailure {
	if len(step.OnSuccess) == 0 {
		return failures
	}

	logger.Info(
		"Executing onSuccess steps for step",
		"step_id",
		step.ID,
		"count",
		len(step.OnSuccess),
	)
	return ExecuteEventStepsOnSuccess(ctx, step.OnSuccess, stepInputs, failures, ao, config)
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
		if isReservedWorkflowInputConfigKey(k) {
			continue
		}
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

	workflowengine.ApplyPipelineSearchAttributes(
		&options.Options,
		pipelineIdentifier,
		runnerIDs,
		entityIDs,
	)

	input := PipelineWorkflowInput{
		WorkflowDefinition: wfDef,
		WorkflowInput: workflowengine.WorkflowInput{
			Config:          config,
			ActivityOptions: &options.ActivityOptions,
		},
		Debug: wfDef.Runtime.Debug,
	}

	if wfDef.Runtime.Schedule.Interval != nil {
		searchAttributes := workflowengine.PipelineTypedSearchAttributes(
			pipelineIdentifier,
			runnerIDs,
			entityIDs,
		)
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

func isReservedWorkflowInputConfigKey(key string) bool {
	return key == tempWalletVersionConfigKey ||
		key == tempCredentialsConfigKey ||
		key == tempUseCaseVerificationsConfigKey ||
		key == GitHubPRCommentConfigKey
}

func ExecuteEventStepsOnError(
	ctx workflow.Context,
	eventSteps []*pipeline.OnErrorStepDefinition,
	stepInputs map[string]any,
	existingFailures []pipelineStepFailure,
	ao workflow.ActivityOptions,
	config map[string]any,
) []pipelineStepFailure {
	return executeEventSteps(
		ctx,
		eventSteps,
		stepInputs,
		existingFailures,
		ao,
		config,
		func(eventStep *pipeline.OnErrorStepDefinition) string { return eventStep.ID },
		func(eventStep *pipeline.OnErrorStepDefinition) *pipeline.ActivityOptionsConfig {
			return eventStep.ActivityOptions
		},
		func(
			eventStep *pipeline.OnErrorStepDefinition,
			ctx workflow.Context,
			config map[string]any,
			stepInputs map[string]any,
			ao workflow.ActivityOptions,
		) (any, error) {
			return ExecuteOnError(eventStep, ctx, config, stepInputs, ao)
		},
	)
}

func ExecuteEventStepsOnSuccess(
	ctx workflow.Context,
	eventSteps []*pipeline.OnSuccessStepDefinition,
	stepInputs map[string]any,
	existingFailures []pipelineStepFailure,
	ao workflow.ActivityOptions,
	config map[string]any,
) []pipelineStepFailure {
	return executeEventSteps(
		ctx,
		eventSteps,
		stepInputs,
		existingFailures,
		ao,
		config,
		func(eventStep *pipeline.OnSuccessStepDefinition) string { return eventStep.ID },
		func(eventStep *pipeline.OnSuccessStepDefinition) *pipeline.ActivityOptionsConfig {
			return eventStep.ActivityOptions
		},
		func(
			eventStep *pipeline.OnSuccessStepDefinition,
			ctx workflow.Context,
			config map[string]any,
			stepInputs map[string]any,
			ao workflow.ActivityOptions,
		) (any, error) {
			return ExecuteOnSuccess(eventStep, ctx, config, stepInputs, ao)
		},
	)
}

func executeEventSteps[T any](
	ctx workflow.Context,
	eventSteps []T,
	stepInputs map[string]any,
	existingFailures []pipelineStepFailure,
	ao workflow.ActivityOptions,
	config map[string]any,
	stepID func(T) string,
	stepActivityOptions func(T) *pipeline.ActivityOptionsConfig,
	execute func(T, workflow.Context, map[string]any, map[string]any, workflow.ActivityOptions) (any, error),
) []pipelineStepFailure {
	failures := existingFailures
	if failures == nil {
		failures = []pipelineStepFailure{}
	}
	for _, eventStep := range eventSteps {
		aO := PrepareActivityOptions(
			ao,
			stepActivityOptions(eventStep),
		)

		_, execErr := execute(eventStep, ctx, config, stepInputs, aO)
		if execErr != nil {
			failures = append(failures, pipelineStepFailure{
				StepID:  stepID(eventStep),
				Message: execErr.Error(),
			})
		}
	}
	return failures
}

func runFinallySteps(
	ctx workflow.Context,
	finallyDef pipeline.FinallyDefinition,
	ao workflow.ActivityOptions,
	config map[string]any,
	payload map[string]any,
	pipelineName string,
	pipelineURL string,
	finalResult string,
	pipelineOutput map[string]any,
	finalOutput map[string]any,
	logger log.Logger,
	errorList *[]error,
) {
	finallySteps := finallyStepsForResult(finallyDef, finalResult)
	if len(finallySteps) == 0 {
		return
	}
	logger.Info("Executing finally steps", "count", len(finallySteps))

	stepInputs := buildPipelineStepInputs(finalOutput, payload)
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
		enrichedStepInputs["pipeline_output"] = pipelineOutput

		_, err := ExecuteStep(
			step.ID,
			step.Use,
			step.With,
			step.ActivityOptions,
			ctx,
			config,
			enrichedStepInputs,
			ao,
		)
		if err != nil {
			logger.Error("Finally step failed", "step_id", step.ID, "error", err)
			if errorList != nil {
				*errorList = append(*errorList, err)
			}
		}
	}
}

func finallyStepsForResult(
	finallyDef pipeline.FinallyDefinition,
	finalResult string,
) []pipeline.FinallyStepDefinition {
	steps := make([]pipeline.FinallyStepDefinition, 0, len(finallyDef.Always))
	steps = append(steps, finallyDef.Always...)
	if finalResult == resultSuccess {
		steps = append(steps, finallyDef.OnSuccess...)
	}
	if finalResult == resultFailed {
		steps = append(steps, finallyDef.OnFailure...)
	}
	return steps
}

func ValidateFinallySteps(finallyDef pipeline.FinallyDefinition) error {
	allowedTypes := map[string]bool{
		"email":            true,
		httpRequestStepUse: true,
	}

	for _, step := range finallyDef.AllSteps() {
		if !allowedTypes[step.Use] {
			return fmt.Errorf(
				"finally step '%s' uses '%s' which is not allowed. Only email and http-request are allowed",
				step.ID,
				step.Use,
			)
		}
	}
	return nil
}
