// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"os/exec"

	"github.com/forkbombeu/credimi-extra/maestro"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

type MaestroFlowActivity struct {
	workflowengine.BaseActivity
}

func NewMaestroFlowActivity() *MaestroFlowActivity {
	return &MaestroFlowActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Run a Maestro flow",
		},
	}
}
func (a *MaestroFlowActivity) Name() string {
	return a.BaseActivity.Name
}
func (a *MaestroFlowActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {

	// Build the input struct for the library function
	runInput := maestro.RunMaestroFlowInput{
		Payload:          input.Payload,
		GetEnv:           utils.GetEnvironmentVariable,
		NewActivityError: a.NewActivityError,
		ErrorCodes: map[string]maestro.ErrorCode{
			"MissingOrInvalidPayload": {
				Code:        errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
				Description: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
			},
			"CommandExecutionFailed": {
				Code:        errorcodes.Codes[errorcodes.CommandExecutionFailed].Code,
				Description: errorcodes.Codes[errorcodes.CommandExecutionFailed].Description,
			},
			"TempFileCreationFailed": {
				Code:        errorcodes.Codes[errorcodes.TempFileCreationFailed].Code,
				Description: errorcodes.Codes[errorcodes.TempFileCreationFailed].Description,
			},
		},
		CommandContext: exec.CommandContext,
	}

	// Call the portable library function
	res, err := maestro.RunMaestroFlow(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	// Return result
	return workflowengine.ActivityResult{
		Output: res["output"],
	}, nil
}
