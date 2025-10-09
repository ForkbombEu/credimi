// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

func (s *StepDefinition) Execute(
	ctx workflow.Context,
	globalCfg map[string]string,
	dataCtx *map[string]any,
	ao workflow.ActivityOptions,
) (any, error) {
	errCode := errorcodes.Codes[errorcodes.PipelineInputError]

	payload, cfg, err := ResolveInputs(*s, globalCfg, *dataCtx)
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
		ctx = workflow.WithActivityOptions(ctx, ao)
		act := step.NewFunc().(workflowengine.Activity)
		input := workflowengine.ActivityInput{
			Payload: payload,
			Config:  cfg,
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

		if output != nil {
			(*dataCtx)[s.ID] = map[string]any{
				"outputs": output,
			}
		}
		return result, nil
	case registry.TaskWorkflow:
		w := step.NewFunc().(workflowengine.Workflow)
		if cfg["app_url"] == "" {
			cfg["app_url"] = "http://localhost:8090"
		}
		input := workflowengine.WorkflowInput{
			Payload:         payload,
			Config:          convertStringMap(cfg),
			ActivityOptions: &ao,
		}

		var result workflowengine.WorkflowResult
		opts := workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf(
				"%s-%s",
				workflow.GetInfo(ctx).WorkflowExecution.ID,
				s.ID,
			),
			TaskQueue:         PipelineTaskQueue,
			ParentClosePolicy: enums.PARENT_CLOSE_POLICY_TERMINATE,
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
		(*dataCtx)[s.ID] = map[string]any{
			"outputs": result.Output,
		}
		return result, nil
	}

	return nil, nil
}
