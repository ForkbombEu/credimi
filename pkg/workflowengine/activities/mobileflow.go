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

	res, err := mobile.ApkInstall(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{
		Output: res,
	}, nil
}

type ApkUninstallActivity struct {
	workflowengine.BaseActivity
}

func NewApkUninstallActivity() *ApkUninstallActivity {
	return &ApkUninstallActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Uninstall APK from device",
		},
	}
}

func (a *ApkUninstallActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *ApkUninstallActivity) Execute(
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

	res, err := mobile.ApkUninstall(ctx, runInput)
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
