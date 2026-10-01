// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

func ExecuteStep(
	id string,
	use string,
	with pipeline.StepInputs,
	activityOptions *pipeline.ActivityOptionsConfig,
	ctx workflow.Context,
	globalCfg map[string]any,
	dataCtx map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	errCode := errorcodes.Codes[errorcodes.PipelineInputError]
	s := &pipeline.StepDefinition{
		StepSpec: pipeline.StepSpec{
			ID:              id,
			Use:             use,
			With:            with,
			ActivityOptions: activityOptions,
			Metadata:        nil,
		},
		ContinueOnError: false,
	}

	step := registry.Registry[s.Use]
	stepGlobalCfg := globalCfg
	if step.Kind == registry.TaskActivity {
		stepGlobalCfg = selectConfigKeys(globalCfg, step.InheritedConfigKeys)
	}
	err := pipeline.ResolveInputs(s, stepGlobalCfg, dataCtx)
	if err != nil {
		appErr := workflowengine.NewAppError(
			workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: fmt.Sprintf("error resolving inputs for step %s: %s", s.ID, err.Error()),
			},
		)

		return nil, appErr
	}
	switch step.Kind {
	case registry.TaskActivity:
		payload, err := DecodePayload(s)
		if err != nil {
			appErr := workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf(
						"error decoding payload for step %s: %s",
						s.ID,
						err.Error(),
					),
				},
			)

			return nil, appErr
		}
		ctx = workflow.WithActivityOptions(ctx, ao)
		act := step.NewFunc().(workflowengine.Activity)
		input := workflowengine.ActivityInput{
			Payload: payload,
			Config:  workflowengine.StringifyConfig(s.With.Config),
		}
		var result workflowengine.ActivityResult

		if s.Use == "email" {
			cfgAct := act.(workflowengine.ConfigurableActivity)

			if err := cfgAct.Configure(&input); err != nil {
				appErr := workflowengine.NewAppError(
					workflowengine.WorkflowError{
						Code:    errCode.Code,
						Summary: errCode.Description,
						Message: fmt.Sprintf(
							"error configuring activity %s: %s",
							s.ID,
							err.Error(),
						),
					},
				)

				return result, appErr
			}
		}
		execAct, ok := act.(workflowengine.ExecutableActivity)
		if !ok {
			appErr := workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf("activity %s is not executable", s.ID),
				},
			)

			return result, appErr
		}

		if err := ensureStepInputSize(s.ID, input); err != nil {
			return nil, err
		}
		err = workflow.ExecuteActivity(ctx, execAct.Name(), input).Get(ctx, &result)
		if err != nil {
			return result, err
		}

		return workflowengine.StepOutputFromActivityResult(step.OutputKind, result), nil
	case registry.TaskWorkflow:
		payload, err := DecodePayload(s)
		if err != nil {
			appErr := workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf(
						"error decoding payload for step %s: %s",
						s.ID,
						err.Error(),
					),
				},
			)

			return nil, appErr
		}
		taskqueue := PipelineTaskQueue
		if step.CustomTaskQueue {
			configuredTaskQueue, ok := s.With.Config["taskqueue"].(string)
			if !ok || strings.TrimSpace(configuredTaskQueue) == "" {
				errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
				return nil, workflowengine.NewAppError(
					workflowengine.WorkflowError{
						Code:    errCode.Code,
						Summary: errCode.Description,
						Message: fmt.Sprintf("missing or invalid taskqueue for step %s", s.ID),
					},
				)
			}
			taskqueue = configuredTaskQueue
		}
		w := step.NewFunc().(workflowengine.Workflow)
		appURL, ok := s.With.Config["app_url"].(string)
		if ok && appURL == "" {
			errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
			appErr := workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: fmt.Sprintf("missing or invalid app_url for step %s", s.ID),
				},
			)

			return nil, appErr
		}
		input := workflowengine.WorkflowInput{
			Payload:         payload,
			Config:          workflowengine.MergeTelemetryConfig(ctx, s.With.Config),
			ActivityOptions: &ao,
		}

		var memo map[string]any
		memo, _ = input.Config["memo"].(map[string]any)

		var result workflowengine.WorkflowResult
		opts := workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf(
				"%s-%s",
				workflow.GetInfo(ctx).WorkflowExecution.ID,
				canonify.CanonifyPlain(s.ID),
			),
			TaskQueue:         taskqueue,
			ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
			Memo:              memo,
		}
		ctxChild := workflow.WithChildOptions(ctx, opts)
		if err := ensureStepInputSize(s.ID, input); err != nil {
			return nil, err
		}
		err = workflow.ExecuteChildWorkflow(
			ctxChild,
			w.Name(),
			input,
		).Get(ctxChild, &result)
		if err != nil {
			return result, err
		}
		return result.Output, nil
	}

	return nil, nil
}

func Execute(
	s *pipeline.StepDefinition,
	ctx workflow.Context,
	globalCfg map[string]any,
	dataCtx map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	with := s.With
	with.Config = maps.Clone(s.With.Config)
	if with.Config == nil {
		with.Config = map[string]any{}
	}
	with.Config[workflowengine.StepIDConfigKey] = s.ID
	return ExecuteStep(s.ID, s.Use, with, s.ActivityOptions, ctx, globalCfg, dataCtx, ao)
}

// maxStepInputBytes keeps a step's scheduling command below the 4 MiB gRPC message limit
// the Temporal frontend enforces on workflow task completions.
const maxStepInputBytes = 3 << 20

// ensureStepInputSize rejects a step input that would make the workflow task completion
// exceed the Temporal gRPC message limit, which would otherwise wedge the run.
func ensureStepInputSize(stepID string, input any) error {
	// A marshal error is left to the SDK, which reports it when it encodes the input.
	data, marshalErr := json.Marshal(input)
	if marshalErr != nil || len(data) <= maxStepInputBytes {
		return nil //nolint:nilerr // the SDK reports encoding errors itself
	}
	errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
	return workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: fmt.Sprintf(
			"step %s input is %d bytes; the limit is %d bytes because Temporal rejects workflow task messages above 4 MiB",
			stepID,
			len(data),
			maxStepInputBytes,
		),
	})
}

// selectConfigKeys copies the listed keys that are present in cfg.
func selectConfigKeys(cfg map[string]any, keys []string) map[string]any {
	selected := make(map[string]any, len(keys))
	for _, key := range keys {
		if value, ok := cfg[key]; ok {
			selected[key] = value
		}
	}
	return selected
}

func ExecuteOnError(
	s *pipeline.OnErrorStepDefinition,
	ctx workflow.Context,
	globalCfg map[string]any,
	dataCtx map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	return ExecuteStep(s.ID, s.Use, s.With, s.ActivityOptions, ctx, globalCfg, dataCtx, ao)
}

func ExecuteOnSuccess(
	s *pipeline.OnSuccessStepDefinition,
	ctx workflow.Context,
	globalCfg map[string]any,
	dataCtx map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	return ExecuteStep(s.ID, s.Use, s.With, s.ActivityOptions, ctx, globalCfg, dataCtx, ao)
}

// runChildPipeline executes a nested child pipeline and returns its outputs. returnOutputs
// lists the child's step outputs the parent references; the child returns only those.
func runChildPipeline(
	ctx workflow.Context,
	step pipeline.StepDefinition,
	input PipelineWorkflowInput,
	workflowName string,
	dataCtx map[string]any,
	runMetadata *workflowengine.WorkflowRunMetadata,
	returnOutputs []string,
) (any, error) {
	// Fetch child pipeline YAML
	yaml, err := fetchChildPipelineYAML(ctx, step, input, runMetadata)
	if err != nil {
		return nil, err
	}

	// Parse workflow definition
	wfDef, err := pipeline.ParseWorkflow(yaml)
	if err != nil {
		return nil, workflowengine.NewWorkflowError(
			workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errorcodes.Codes[errorcodes.PipelineParsingError].Code,
					Summary: errorcodes.Codes[errorcodes.PipelineParsingError].Description,
					Message: err.Error(),
				},
			),
			runMetadata,
		)
	}

	memo := map[string]any{"test": wfDef.Name}
	options := PrepareWorkflowOptions(wfDef.Runtime)
	options.Options.Memo = memo

	childOpts := workflow.ChildWorkflowOptions{
		WorkflowID: fmt.Sprintf(
			"%s-%s-%s",
			workflow.GetInfo(ctx).WorkflowExecution.ID,
			canonify.CanonifyPlain(step.ID),
			wfDef.Name,
		),
		TaskQueue:         PipelineTaskQueue,
		ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
	}

	ao := PrepareActivityOptions(options.ActivityOptions, step.ActivityOptions)
	childOpts.RetryPolicy = ao.RetryPolicy
	ctxChild := workflow.WithChildOptions(ctx, childOpts)
	err = pipeline.ResolveInputs(&step, input.WorkflowInput.Config, dataCtx)
	if err != nil {
		return nil, err
	}
	childInput := PipelineWorkflowInput{
		WorkflowDefinition: wfDef,
		WorkflowInput: workflowengine.WorkflowInput{
			Config:          workflowengine.MergeTelemetryConfig(ctx, step.With.Config),
			Payload:         step.With.Payload,
			ActivityOptions: &ao,
		},
		Debug:         wfDef.Runtime.Debug,
		ReturnOutputs: returnOutputs,
	}
	childInput.WorkflowInput.Config[workflowengine.StepIDConfigKey] = step.ID

	if err := ensureStepInputSize(step.ID, childInput); err != nil {
		return nil, err
	}
	var childResult workflowengine.WorkflowResult
	err = workflow.ExecuteChildWorkflow(
		ctxChild,
		workflowName,
		childInput,
	).Get(ctxChild, &childResult)

	if err != nil {
		return nil, err
	}

	return childResult.Output, nil
}

// fetchChildPipelineYAML fetches the pipeline YAML from an internal API route.
func fetchChildPipelineYAML(
	ctx workflow.Context,
	step pipeline.StepDefinition,
	input PipelineWorkflowInput,
	meta *workflowengine.WorkflowRunMetadata,
) (string, error) {
	pipelineID, ok := step.With.Payload["pipeline_id"].(string)
	if !ok || pipelineID == "" {
		return "", workflowengine.NewMissingOrInvalidPayloadError(
			fmt.Errorf("missing pipeline_id"),
			meta,
		)
	}

	appURL, ok := input.WorkflowInput.Config["app_url"].(string)
	if !ok || appURL == "" {
		return "", workflowengine.NewWorkflowError(
			workflowengine.NewMissingConfigError("app_url", meta),
			meta,
		)
	}

	act := activities.NewInternalHTTPActivity()
	var response workflowengine.ActivityResult
	req := workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method: http.MethodGet,
			URL: utils.JoinURL(
				workflowengine.InternalAppURLFromConfig(input.WorkflowInput.Config),
				"api", "pipeline", "get-yaml",
			),
			QueryParams: map[string]string{
				"pipeline_identifier": pipelineID,
			},
			ExpectedStatus: 200,
		},
	}

	if err := workflow.ExecuteActivity(ctx, act.Name(), req).Get(ctx, &response); err != nil {
		return "", workflowengine.NewWorkflowError(err, meta)
	}

	body, ok := response.Output.(map[string]any)["body"].(string)
	if !ok {
		return "", workflowengine.NewWorkflowError(
			workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errorcodes.Codes[errorcodes.UnexpectedActivityOutput].Code,
					Summary: errorcodes.Codes[errorcodes.UnexpectedActivityOutput].Description,
					Message: "invalid HTTP output",
					Details: map[string]any{"payload": response.Output},
				},
			),
			meta,
		)
	}

	return body, nil
}

func ExtractPipelineOutput(dataCtx map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range dataCtx {
		if key == "inputs" {
			continue
		}

		if stepOutput, ok := value.(map[string]any); ok {
			if outputs, exists := stepOutput["outputs"]; exists {
				result[key] = map[string]any{"outputs": outputs}
			} else {
				result[key] = map[string]any{"outputs": value}
			}
		} else {
			result[key] = map[string]any{"outputs": value}
		}
	}

	return result
}

func enrichDataContext(
	dataCtx map[string]any,
	pipelineName string,
	pipelineURL string,
	hasErrors bool,
	currentTime string,
) map[string]any {
	enriched := make(map[string]any)

	for k, v := range dataCtx {
		enriched[k] = v
	}

	enriched["pipeline_output"] = ExtractPipelineOutput(dataCtx)

	enriched["pipeline_name"] = pipelineName
	enriched["pipeline_url"] = pipelineURL

	if hasErrors {
		enriched["result"] = resultFailed
	} else {
		enriched["result"] = resultSuccess
	}

	enriched["date"] = currentTime

	return enriched
}
