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

func (s *StepDefinition) Execute(
	ctx workflow.Context,
	globalCfg map[string]string,
	dataCtx *map[string]any,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult

	errCode := errorcodes.Codes[errorcodes.PipelineInputError]

	input, err := ResolveInputs(*s, globalCfg, *dataCtx)
	if err != nil {
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("error resolving inputs for step %s: %s", s.ID, err.Error()),
		)
		return result, appErr
	}
	act := activities.Registry[s.Run].NewFunc()

	if s.Run == "email" {
		cfgAct := act.(workflowengine.ConfigurableActivity)
		if err := cfgAct.Configure(input); err != nil {
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

	err = workflow.ExecuteActivity(ctx, execAct.Name(), *input).Get(ctx, &result)
	if err != nil {
		return result, err
	}
	var output any
	switch activities.Registry[s.Run].OutputKind {
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
		(*dataCtx)[s.ID] = map[string]any{
			"outputs": output,
		}
	}

	return result, nil
}
