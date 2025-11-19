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

type StartEmulatorActivity struct {
	workflowengine.BaseActivity
}

func NewStartEmulatorActivity() *StartEmulatorActivity {
	return &StartEmulatorActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Start emulator",
		},
	}
}

func (a *StartEmulatorActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *StartEmulatorActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := mobile.MobileActivityInput{
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
		},
	}
	res, err := mobile.StartEmulator(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{
		Output: res,
	}, nil
}

type ApkInstallActivity struct {
	workflowengine.BaseActivity
}

func NewApkInstallActivity() *ApkInstallActivity {
	return &ApkInstallActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Install APK on device",
		},
	}
}

func (a *ApkInstallActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *ApkInstallActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := mobile.MobileActivityInput{
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
		},
		CommandContext: exec.CommandContext,
	}

	res, err := mobile.ApkInstall(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{
		Output: res,
	}, nil
}

type UnlockEmulatorActivity struct {
	workflowengine.BaseActivity
}

func NewUnlockEmulatorActivity() *UnlockEmulatorActivity {
	return &UnlockEmulatorActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Unlock emulator",
		},
	}
}

func (a *UnlockEmulatorActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *UnlockEmulatorActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := mobile.MobileActivityInput{
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
		},
		CommandContext: exec.CommandContext,
	}

	res, err := mobile.UnlockEmulator(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{
		Output: res,
	}, nil
}

type StopEmulatorActivity struct {
	workflowengine.BaseActivity
}

func NewStopEmulatorActivity() *StopEmulatorActivity {
	return &StopEmulatorActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Stop emulator",
		},
	}
}

func (a *StopEmulatorActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *StopEmulatorActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := mobile.MobileActivityInput{
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
		},
	}

	res, err := mobile.StopEmulator(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{
		Output: res,
	}, nil
}

type RunMobileFlowActivity struct {
	workflowengine.BaseActivity
}

func NewRunMobileFlowActivity() *RunMobileFlowActivity {
	return &RunMobileFlowActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Run a mobile test flow",
		},
	}
}

func (a *RunMobileFlowActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *RunMobileFlowActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := mobile.MobileActivityInput{
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
