// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"os/exec"

	"github.com/forkbombeu/credimi-extra/mobile"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

type MobileFlowActivity struct {
	workflowengine.BaseActivity
}

func NewMobileFlowActivity() *MobileFlowActivity {
	return &MobileFlowActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Run a Mobile flow",
		},
	}
}
func (a *MobileFlowActivity) Name() string {
	return a.BaseActivity.Name
}
func (a *MobileFlowActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {

	runInput := mobile.RunMobileFlowInput{
		Payload:          input.Payload,
		GetEnv:           utils.GetEnvironmentVariable,
		NewActivityError: a.NewActivityError,
		ErrorCodes: map[string]mobile.ErrorCode{
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

	res, err := mobile.RunMobileFlow(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{
		Output: res["output"],
	}, nil
}
