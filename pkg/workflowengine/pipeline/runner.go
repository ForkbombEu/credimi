// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"fmt"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
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
	with StepInputs,
	activityOptions *ActivityOptionsConfig,
	ctx workflow.Context,
	globalCfg map[string]any,
	dataCtx map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	errCode := errorcodes.Codes[errorcodes.PipelineInputError]
	s := &StepDefinition{
		StepSpec: StepSpec{
			ID:              id,
			Use:             use,
			With:            with,
			ActivityOptions: activityOptions,
			Metadata:        nil,
		},
		ContinueOnError: false,
	}

	err := ResolveInputs(s, globalCfg, dataCtx)
	if err != nil {
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("error resolving inputs for step %s: %s", s.ID, err.Error()),
		)
		return nil, appErr
	}
	step := registry.Registry[s.Use]
	switch step.Kind {
	case registry.TaskActivity:
		payload, err := s.DecodePayload()
		if err != nil {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("error decoding payload for step %s: %s", s.ID, err.Error()),
			)
			return nil, appErr
		}
		ctx = workflow.WithActivityOptions(ctx, ao)
		act := step.NewFunc().(workflowengine.Activity)
		input := workflowengine.ActivityInput{
			Payload: payload,
			Config:  convertMapAnyToString(s.With.Config),
		}
		var result workflowengine.ActivityResult

		if s.Use == "email" {
			cfgAct := act.(workflowengine.ConfigurableActivity)

			if err := cfgAct.Configure(&input); err != nil {
				appErr := workflowengine.NewAppError(
					errCode,
					fmt.Sprintf("error configuring activity %s: %s", s.ID, err.Error()),
				)
				return result, appErr
			}
		}
		execAct, ok := act.(workflowengine.ExecutableActivity)
		if !ok {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("activity %s is not executable", s.ID),
			)
			return result, appErr
		}

		err = workflow.ExecuteActivity(ctx, execAct.Name(), input).Get(ctx, &result)
		if err != nil {
			return result, err
		}
		var output any
		switch registry.Registry[s.Use].OutputKind {
		case workflowengine.OutputMap:
			output = workflowengine.AsMap(result.Output)

		case workflowengine.OutputString:
			output = workflowengine.AsString(result.Output)

		case workflowengine.OutputArrayOfString:
			output = workflowengine.AsSliceOfStrings(result.Output)

		case workflowengine.OutputArrayOfMap:
			output = workflowengine.AsSliceOfMaps(result.Output)

		case workflowengine.OutputBool:
			output = workflowengine.AsBool(result.Output)

		case workflowengine.OutputAny:
			output = result
		}

		return output, nil
	case registry.TaskWorkflow:
		payload, err := s.DecodePayload()
		if err != nil {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("error decoding payload for step %s: %s", s.ID, err.Error()),
			)
			return nil, appErr
		}
		taskqueue := PipelineTaskQueue
		if step.CustomTaskQueue {
			taskqueue = s.With.Config["taskqueue"].(string)
		}
		w := step.NewFunc().(workflowengine.Workflow)
		appURL, ok := s.With.Config["app_url"].(string)
		if ok && appURL == "" {
			s.With.Config["app_url"] = "http://localhost:8090"
		}
		input := workflowengine.WorkflowInput{
			Payload:         payload,
			Config:          s.With.Config,
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

func (s *StepDefinition) Execute(
	ctx workflow.Context,
	globalCfg map[string]any,
	dataCtx map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	return ExecuteStep(s.ID, s.Use, s.With, s.ActivityOptions, ctx, globalCfg, dataCtx, ao)
}

func (s *OnErrorStepDefinition) ExecuteOnError(
	ctx workflow.Context,
	globalCfg map[string]any,
	dataCtx map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	return ExecuteStep(s.ID, s.Use, s.With, s.ActivityOptions, ctx, globalCfg, dataCtx, ao)
}

func (s *OnSuccessStepDefinition) ExecuteOnSuccess(
	ctx workflow.Context,
	globalCfg map[string]any,
	dataCtx map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	return ExecuteStep(s.ID, s.Use, s.With, s.ActivityOptions, ctx, globalCfg, dataCtx, ao)
}

// runChildPipeline executes a nested child pipeline and returns its outputs
func runChildPipeline(
	ctx workflow.Context,
	step StepDefinition,
	input PipelineWorkflowInput,
	workflowName string,
	dataCtx map[string]any,
	runMetadata *workflowengine.WorkflowErrorMetadata,
) (any, error) {
	// Fetch child pipeline YAML
	yaml, err := fetchChildPipelineYAML(ctx, step, input, runMetadata)
	if err != nil {
		return nil, err
	}

	// Parse workflow definition
	wfDef, err := ParseWorkflow(yaml)
	if err != nil {
		return nil, workflowengine.NewWorkflowError(
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.PipelineParsingError],
				err.Error(),
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

	ctxChild := workflow.WithChildOptions(ctx, childOpts)
	ao := PrepareActivityOptions(options.ActivityOptions, step.ActivityOptions)
	err = ResolveInputs(&step, input.WorkflowInput.Config, dataCtx)
	if err != nil {
		return nil, err
	}
	childInput := PipelineWorkflowInput{
		WorkflowDefinition: wfDef,
		WorkflowInput: workflowengine.WorkflowInput{
			Config:          step.With.Config,
			Payload:         step.With.Payload,
			ActivityOptions: &ao,
		},
		Debug: wfDef.Runtime.Debug,
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

// fetchChildPipelineYAML fetches the pipeline YAML from HTTPActivity
func fetchChildPipelineYAML(
	ctx workflow.Context,
	step StepDefinition,
	input PipelineWorkflowInput,
	meta *workflowengine.WorkflowErrorMetadata,
) (string, error) {
	pipelineID, ok := step.With.Payload["pipeline_id"].(string)
	if !ok || pipelineID == "" {
		return "", workflowengine.NewWorkflowError(fmt.Errorf("missing pipeline_id"), meta)
	}

	appURL, ok := input.WorkflowInput.Config["app_url"].(string)
	if !ok || appURL == "" {
		return "", workflowengine.NewWorkflowError(
			workflowengine.NewMissingConfigError("app_url", meta),
			meta,
		)
	}

	act := activities.NewHTTPActivity()
	var response workflowengine.ActivityResult
	req := workflowengine.ActivityInput{
		Payload: activities.HTTPActivityPayload{
			Method: http.MethodGet,
			URL:    utils.JoinURL(appURL, "api", "pipeline", "get-yaml"),
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
				errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
				"invalid HTTP output",
				response.Output,
			),
			meta,
		)
	}

	return body, nil
}
