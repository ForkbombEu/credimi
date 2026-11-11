// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// LinkedStepsWorkflow runs a block of linked steps once each. Temporal retries the whole
// workflow, so every attempt starts again from the block's first step: a deeplink a step
// produces is single use, and a retry of its consumer alone would reuse it.
type LinkedStepsWorkflow struct{}

// LinkedStepsWorkflowInput carries the block and the pipeline context its steps resolve
// their inputs against.
type LinkedStepsWorkflowInput struct {
	Steps []pipeline.StepDefinition `json:"steps"`
	// PipelineConfig is never sent as "config": pipelinehistory maps a child's
	// config.step_id to a main step.
	PipelineConfig  map[string]any           `json:"pipeline_config"`
	Payload         map[string]any           `json:"payload"`
	PriorOutputs    map[string]any           `json:"prior_outputs"`
	FailedSteps     []string                 `json:"failed_steps"`
	HasFailures     bool                     `json:"has_failures"`
	PipelineName    string                   `json:"pipeline_name"`
	PipelineURL     string                   `json:"pipeline_url"`
	ActivityOptions workflow.ActivityOptions `json:"activity_options"`
	MaximumAttempts int32                    `json:"maximum_attempts"`
	Debug           bool                     `json:"debug"`
}

// NewLinkedStepsWorkflow returns the workflow that runs a block of linked steps.
func NewLinkedStepsWorkflow() *LinkedStepsWorkflow {
	return &LinkedStepsWorkflow{}
}

// Name returns the Temporal workflow type of a linked steps block.
func (LinkedStepsWorkflow) Name() string {
	return pipeline.LinkedStepsWorkflowName
}

// Workflow runs the block once, as the current Temporal attempt.
func (w *LinkedStepsWorkflow) Workflow(
	ctx workflow.Context,
	input LinkedStepsWorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.run(ctx, input, workflow.GetInfo(ctx).Attempt)
}

func (w *LinkedStepsWorkflow) run(
	ctx workflow.Context,
	input LinkedStepsWorkflowInput,
	attempt int32,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	info := workflow.GetInfo(ctx)

	finalOutput := maps.Clone(input.PriorOutputs)
	if finalOutput == nil {
		finalOutput = map[string]any{}
	}
	failedSteps := make(map[string]struct{}, len(input.FailedSteps))
	for _, id := range input.FailedSteps {
		failedSteps[id] = struct{}{}
	}
	state := &pipelineExecutionState{
		finalOutput:   finalOutput,
		resultOutput:  map[string]any{},
		failedSteps:   failedSteps,
		linkedBlock:   true,
		priorFailures: input.HasFailures,
	}
	runMetadata := &workflowengine.WorkflowRunMetadata{
		WorkflowName: pipeline.LinkedStepsWorkflowName,
		WorkflowID:   info.WorkflowExecution.ID,
		RunID:        info.WorkflowExecution.RunID,
		Namespace:    info.Namespace,
		TemporalUI:   input.PipelineURL,
	}
	synthetic := PipelineWorkflowInput{
		WorkflowDefinition: &pipeline.WorkflowDefinition{Name: input.PipelineName},
		WorkflowInput: workflowengine.WorkflowInput{
			Config:  input.PipelineConfig,
			Payload: input.Payload,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, input.ActivityOptions)

	for _, step := range input.Steps {
		step.ContinueOnError = false
		err := (&PipelineWorkflow{}).executeRegularStep(
			ctx,
			synthetic,
			step,
			input.ActivityOptions,
			input.PipelineConfig,
			runMetadata,
			state,
			input.Debug,
			logger,
		)
		if err == nil {
			continue
		}
		if temporal.IsCanceledError(err) || ctx.Err() != nil {
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowCancellationError(
				runMetadata,
			)
		}
		// executeRegularStep reports every step failure through handleRegularStepError,
		// which keeps the step's own error in state.stepErr.
		stepErr := state.stepErr
		output := linkedStepsOutput(input.Steps, state.finalOutput)
		failure := newPipelineStepFailure(step.ID, stepErr)
		failures := []pipelineStepFailure{failure}
		appErr := workflowengine.NewAppError(workflowengine.WorkflowError{
			Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
			Summary: buildPipelineFailureSummary(failures),
			Details: map[string]any{
				"errors": buildPipelineFailureErrors(failures),
				"output": output,
			},
		})
		if !linkedBlockRetryable(stepErr) {
			return workflowengine.WorkflowResult{}, workflowengine.NewNonRetryableWorkflowError(
				appErr,
				runMetadata,
			)
		}
		if attempt < input.MaximumAttempts {
			deleteAttemptScreenshots(ctx, logger, state, input.Steps)
		}
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	return workflowengine.WorkflowResult{
		WorkflowID:    info.WorkflowExecution.ID,
		WorkflowRunID: info.WorkflowExecution.RunID,
		Output:        linkedStepsOutput(input.Steps, state.finalOutput),
	}, nil
}

// linkedStepsOutput returns the full {"outputs": ...} entries of the block steps that
// produced one: reports read every step output, so nothing is trimmed.
func linkedStepsOutput(
	steps []pipeline.StepDefinition,
	finalOutput map[string]any,
) map[string]any {
	output := make(map[string]any, len(steps))
	for _, step := range steps {
		if entry, ok := finalOutput[step.ID]; ok {
			output[step.ID] = entry
		}
	}
	return output
}

// linkedBlockRetryable decides whether a failed block attempt is retried. A failure inside
// an activity or child workflow is retried with a fresh deeplink, even when the inner error
// is non-retryable for the same input. Engine errors raised before a step is scheduled
// (input resolution, failed dependencies, oversized input) are non-retryable.
func linkedBlockRetryable(err error) bool {
	var activityErr *temporal.ActivityError
	if errors.As(err, &activityErr) {
		return true
	}
	var childErr *temporal.ChildWorkflowExecutionError
	if errors.As(err, &childErr) {
		return true
	}
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.NonRetryable() {
		return false
	}
	return true
}

const (
	deleteScreenshotsTimeout     = 30 * time.Second
	deleteScreenshotsMaxAttempts = 3
)

// deleteAttemptScreenshots removes from the pipeline result the screenshots the mobile
// steps of a failed attempt stored, before Temporal retries the block, so only the final
// attempt's screenshots are kept. Failures are logged: the attempt's failure is what the
// block reports.
func deleteAttemptScreenshots(
	ctx workflow.Context,
	logger log.Logger,
	state *pipelineExecutionState,
	steps []pipeline.StepDefinition,
) {
	byRun := map[string][]string{}
	var runs []string
	for _, step := range steps {
		if step.Use != mobileAutomationStepUse {
			continue
		}
		entry, ok := state.finalOutput[step.ID]
		if !ok {
			continue
		}
		runIdentifier, _ := step.With.Config["run_identifier"].(string)
		if runIdentifier == "" {
			continue
		}
		outputs := workflowengine.AsMap(workflowengine.AsMap(entry)["outputs"])
		flowOutput := workflowengine.AsMap(outputs["flow_output"])
		for _, raw := range workflowengine.AsSliceOfStrings(flowOutput["maestro_screenshot_urls"]) {
			parsed, err := url.Parse(raw)
			if err != nil {
				continue
			}
			name := path.Base(parsed.Path)
			if name == "" || name == "." || name == "/" {
				continue
			}
			if _, seen := byRun[runIdentifier]; !seen {
				runs = append(runs, runIdentifier)
			}
			byRun[runIdentifier] = append(byRun[runIdentifier], name)
		}
	}
	if len(runs) == 0 {
		return
	}
	actCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: deleteScreenshotsTimeout,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: deleteScreenshotsMaxAttempts},
	})
	for _, runIdentifier := range runs {
		err := workflow.ExecuteActivity(
			actCtx,
			activities.DeletePipelineStepScreenshotsActivityName,
			workflowengine.ActivityInput{Payload: activities.DeletePipelineStepScreenshotsInput{
				RunIdentifier: runIdentifier,
				FileNames:     byRun[runIdentifier],
			}},
		).Get(actCtx, nil)
		if err != nil {
			logger.Warn("delete screenshots of retried linked steps attempt", "error", err)
		}
	}
}

// executeLinkedSteps runs a block of linked steps as one Linked Steps Workflow child,
// retried by Temporal with the retry policy of the block's last step.
func (w *PipelineWorkflow) executeLinkedSteps(
	ctx workflow.Context,
	input PipelineWorkflowInput,
	block []pipeline.StepDefinition,
	ao workflow.ActivityOptions,
	config map[string]any,
	runMetadata *workflowengine.WorkflowRunMetadata,
	state *pipelineExecutionState,
	debug bool,
	logger log.Logger,
) error {
	first := block[0]
	ids := make([]string, 0, len(block))
	for _, step := range block {
		ids = append(ids, step.ID)
	}

	for _, step := range block {
		if workflowengine.AsBool(step.With.Config[mobileExternalInstallConfigKey]) {
			errCode := errorcodes.Codes[errorcodes.PipelineInputError]
			err := workflowengine.NewNonRetryableAppError(workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: fmt.Sprintf(
					"step %s installs the wallet from an external source and cannot be linked",
					step.ID,
				),
			})
			return handleLinkedStepsFailure(
				block,
				newPipelineStepFailure(step.ID, err),
				state,
				runMetadata,
			)
		}
	}

	blockAO := PrepareActivityOptions(ao, block[len(block)-1].ActivityOptions)
	rp := blockAO.RetryPolicy
	if rp.MaximumAttempts == 0 {
		rp.MaximumAttempts = DefaultRetryMaxAttempts
	}

	cfg := workflowengine.MergeTelemetryConfig(ctx, config)
	for _, key := range []string{
		workflowengine.TelemetryParentWorkflowIDKey,
		workflowengine.TelemetryParentRunIDKey,
		workflowengine.TelemetryCurrentWorkflowIDKey,
		workflowengine.TelemetryCurrentRunIDKey,
		workflowengine.StepIDConfigKey,
	} {
		delete(cfg, key)
	}

	failedSteps := slices.Sorted(maps.Keys(state.failedSteps))
	blockInput := LinkedStepsWorkflowInput{
		Steps:           slices.Clone(block),
		PipelineConfig:  cfg,
		Payload:         workflowengine.AsMap(input.WorkflowInput.Payload),
		PriorOutputs:    linkedStepsPriorOutputs(block, state.finalOutput),
		FailedSteps:     failedSteps,
		HasFailures:     state.hasFailures(),
		PipelineName:    input.WorkflowDefinition.Name,
		PipelineURL:     runMetadata.TemporalUI,
		ActivityOptions: ao,
		MaximumAttempts: rp.MaximumAttempts,
		Debug:           debug,
	}
	if err := ensureStepInputSize(first.ID, blockInput); err != nil {
		return handleLinkedStepsFailure(
			block,
			newPipelineStepFailure(first.ID, err),
			state,
			runMetadata,
		)
	}

	info := workflow.GetInfo(ctx)
	childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:        pipeline.LinkedStepsBlockID(info.WorkflowExecution.ID, first.ID),
		TaskQueue:         PipelineTaskQueue,
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
		RetryPolicy:       rp,
		Memo: map[string]any{
			workflowengine.LinkedStepsMemoKey: workflowengine.LinkedStepsMemo{
				Steps:           ids,
				MaximumAttempts: rp.MaximumAttempts,
			},
		},
		StaticSummary: summaryLine(strings.Join(ids, " → ")),
	})
	logger.Info("Running linked steps", "ids", ids, "maximum_attempts", rp.MaximumAttempts)

	var result workflowengine.WorkflowResult
	err := workflow.ExecuteChildWorkflow(childCtx, pipeline.LinkedStepsWorkflowName, blockInput).
		Get(ctx, &result)
	if err != nil {
		if temporal.IsCanceledError(err) {
			return workflowengine.NewWorkflowCancellationError(runMetadata)
		}
		partial := workflowengine.ExtractOutputFromError(err)
		for _, id := range ids {
			if entry, ok := partial[id]; ok {
				state.finalOutput[id] = entry
			}
		}
		return handleLinkedStepsFailure(
			block,
			linkedStepsFailure(first.ID, err),
			state,
			runMetadata,
		)
	}

	// Each step's debug activity ran inside the block, which receives the debug flag.
	output := workflowengine.AsMap(result.Output)
	for _, id := range ids {
		if entry, ok := output[id]; ok {
			state.finalOutput[id] = entry
		}
	}
	state.previousStepID = ids[len(ids)-1]
	return nil
}

// linkedStepsPriorOutputs returns the outputs of earlier steps the block references, or
// every output when a step references the whole pipeline_output.
func linkedStepsPriorOutputs(
	block []pipeline.StepDefinition,
	finalOutput map[string]any,
) map[string]any {
	referenced := map[string]struct{}{}
	for _, step := range block {
		ids, all := pipeline.ReferencedStepIDs(step.With)
		if all {
			return maps.Clone(finalOutput)
		}
		for _, id := range ids {
			referenced[id] = struct{}{}
		}
	}
	prior := make(map[string]any, len(referenced))
	for id := range referenced {
		if entry, ok := finalOutput[id]; ok {
			prior[id] = entry
		}
	}
	return prior
}

// linkedStepsFailure returns the failure of the block step that stopped the last attempt,
// read from the block's failure details, or a failure of the first step when the block
// reports none, as after a timeout or termination.
func linkedStepsFailure(firstStepID string, err error) pipelineStepFailure {
	if temporal.IsTimeoutError(err) || temporal.IsTerminatedError(err) {
		return newPipelineStepFailure(firstStepID, err)
	}
	errs, ok := workflowengine.ParseWorkflowError(err).Details["errors"].([]any)
	if !ok || len(errs) == 0 {
		return newPipelineStepFailure(firstStepID, err)
	}
	raw, marshalErr := json.Marshal(errs[0])
	if marshalErr != nil {
		return newPipelineStepFailure(firstStepID, err)
	}
	var failure workflowengine.WorkflowError
	if json.Unmarshal(raw, &failure) != nil {
		return newPipelineStepFailure(firstStepID, err)
	}
	stepID, _ := failure.Details["step_id"].(string)
	if stepID == "" {
		return newPipelineStepFailure(firstStepID, err)
	}
	return pipelineStepFailure{StepID: stepID, Failure: failure}
}

// handleLinkedStepsFailure records a failed block. With continue_on_error the failing
// step, the block steps after it and those that never ran count as failed and the pipeline
// continues; earlier block steps that ran keep their last attempt's outputs.
func handleLinkedStepsFailure(
	block []pipeline.StepDefinition,
	failure pipelineStepFailure,
	state *pipelineExecutionState,
	runMetadata *workflowengine.WorkflowRunMetadata,
) error {
	if !block[0].ContinueOnError {
		return newPipelineExecutionError(
			prependPipelineStepFailure(failure, state.failures),
			state.resultOutput,
			runMetadata,
		)
	}
	state.failures = append(state.failures, failure)
	failed := false
	for _, step := range block {
		if step.ID == failure.StepID {
			failed = true
		}
		if _, ran := state.finalOutput[step.ID]; failed || !ran {
			state.failedSteps[step.ID] = struct{}{}
		}
	}
	return nil
}
