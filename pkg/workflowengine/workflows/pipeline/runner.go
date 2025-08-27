// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

func (s *StepDefinition) Run(ctx workflow.Context, globalCfg map[string]string, dataCtx *map[string]any) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult

	errCode := errorcodes.Codes[errorcodes.PipelineInputError]

	input, err := ResolveInputs(*s, globalCfg, *dataCtx)
	if err != nil {
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("error resolving inputs for step %s: %s", s.Name, err.Error()),
		)
		return result, appErr
	}
	act := activities.Registry[s.Activity].NewFunc()

	//Configure if the activity supports it
	if cfgAct, ok := act.(workflowengine.ConfigurableActivity); ok {
		if err := cfgAct.Configure(input); err != nil {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("error configuring activity %s: %s", s.Name, err.Error()),
			)
			return result, appErr
		}
	}

	execAct, ok := act.(workflowengine.ExecutableActivity)
	if !ok {
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("activity %s is not executable", s.Name),
		)
		return result, appErr
	}

	err = workflow.ExecuteActivity(ctx, execAct.Name(), *input).Get(ctx, &result)
	if err != nil {
		return result, err
	}
	var output any
	switch activities.Registry[s.Activity].OutputKind {
	case workflowengine.OutputMap:
		output = workflowengine.AsMap(result.Output)

	case workflowengine.OutputString:
		output = workflowengine.AsString(result.Output)

	case workflowengine.OutputArrayOfString:
		output = workflowengine.AsSliceOfStrings(result.Output)
	case workflowengine.OutputArrayOfMap:
		output = workflowengine.AsSliceOfMaps(result.Output)
	case workflowengine.OutputAny:
		output = result
	}

	if output != nil {
		(*dataCtx)[s.Name] = map[string]any{
			"output": output,
		}
	}

	return result, nil
}
