//go:build credimi_extra

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
			Name: "Setup mobile device",
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
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		nil,
		false,
	)

	res, err := mobile.StartEmulator(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
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
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		nil,
		true,
	)

	res, err := mobile.ApkInstall(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
}

type ApkPostInstallChecksActivity struct {
	workflowengine.BaseActivity
}

func NewApkPostInstallChecksActivity() *ApkPostInstallChecksActivity {
	return &ApkPostInstallChecksActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Run APK post-install checks",
		},
	}
}

func (a *ApkPostInstallChecksActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *ApkPostInstallChecksActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		map[string]mobile.ErrorCode{
			"TempFileCreationFailed": {
				Code:        errorcodes.Codes[errorcodes.TempFileCreationFailed].Code,
				Description: errorcodes.Codes[errorcodes.TempFileCreationFailed].Description,
			},
		},
		true,
	)

	res, err := mobile.ApkPostInstallChecks(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
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
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		nil,
		true,
	)

	res, err := mobile.UnlockEmulator(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
}

type StartIOSSimulatorActivity struct {
	workflowengine.BaseActivity
}

func NewStartIOSSimulatorActivity() *StartIOSSimulatorActivity {
	return &StartIOSSimulatorActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Setup iOS simulator",
		},
	}
}

func (a *StartIOSSimulatorActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *StartIOSSimulatorActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		nil,
		false,
	)

	res, err := mobile.StartIOSSimulator(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
}

type InstallIOSAppActivity struct {
	workflowengine.BaseActivity
}

func NewInstallIOSAppActivity() *InstallIOSAppActivity {
	return &InstallIOSAppActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Install iOS app on device",
		},
	}
}

func (a *InstallIOSAppActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *InstallIOSAppActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		nil,
		true,
	)

	res, err := mobile.InstallIOSApp(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
}

type IOSPostInstallChecksActivity struct {
	workflowengine.BaseActivity
}

func NewIOSPostInstallChecksActivity() *IOSPostInstallChecksActivity {
	return &IOSPostInstallChecksActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Run iOS post-install checks",
		},
	}
}

func (a *IOSPostInstallChecksActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *IOSPostInstallChecksActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		map[string]mobile.ErrorCode{
			"TempFileCreationFailed": {
				Code:        errorcodes.Codes[errorcodes.TempFileCreationFailed].Code,
				Description: errorcodes.Codes[errorcodes.TempFileCreationFailed].Description,
			},
		},
		true,
	)

	res, err := mobile.IOSAppPostInstallChecks(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
}

type CleanupDeviceActivity struct {
	workflowengine.BaseActivity
}

func NewCleanupDeviceActivity() *CleanupDeviceActivity {
	return &CleanupDeviceActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Cleanup mobile device",
		},
	}
}

func (a *CleanupDeviceActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *CleanupDeviceActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		nil,
		true,
	)

	res, err := mobile.CleanupDevice(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
}

type ListInstalledAppsActivity struct {
	workflowengine.BaseActivity
}

func NewListInstalledAppsActivity() *ListInstalledAppsActivity {
	return &ListInstalledAppsActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "List installed mobile apps",
		},
	}
}

func (a *ListInstalledAppsActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *ListInstalledAppsActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		nil,
		true,
	)

	res, err := mobile.ListInstalledApps(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
}

type DisableAndroidPlayStoreActivity struct {
	workflowengine.BaseActivity
}

func NewDisableAndroidPlayStoreActivity() *DisableAndroidPlayStoreActivity {
	return &DisableAndroidPlayStoreActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Disable Android Play Store",
		},
	}
}

func (a *DisableAndroidPlayStoreActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *DisableAndroidPlayStoreActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		nil,
		true,
	)

	res, err := mobile.DisableAndroidPlayStore(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
}

type StartRecordingActivity struct {
	workflowengine.BaseActivity
}

func NewStartRecordingActivity() *StartRecordingActivity {
	return &StartRecordingActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Start recording device screen",
		},
	}
}

func (a *StartRecordingActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *StartRecordingActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		map[string]mobile.ErrorCode{
			"TempFileCreationFailed": {
				Code:        errorcodes.Codes[errorcodes.TempFileCreationFailed].Code,
				Description: errorcodes.Codes[errorcodes.TempFileCreationFailed].Description,
			},
		},
		true,
	)

	res, err := mobile.StartVideoRecording(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
}

type StartIOSRecordingActivity struct {
	workflowengine.BaseActivity
}

func NewStartIOSRecordingActivity() *StartIOSRecordingActivity {
	return &StartIOSRecordingActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Start recording iOS device screen",
		},
	}
}

func (a *StartIOSRecordingActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *StartIOSRecordingActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		map[string]mobile.ErrorCode{
			"TempFileCreationFailed": {
				Code:        errorcodes.Codes[errorcodes.TempFileCreationFailed].Code,
				Description: errorcodes.Codes[errorcodes.TempFileCreationFailed].Description,
			},
		},
		true,
	)

	res, err := mobile.StartIOSVideoRecording(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
}

type StopRecordingActivity struct {
	workflowengine.BaseActivity
}

func NewStopRecordingActivity() *StopRecordingActivity {
	return &StopRecordingActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Stop recording device screen",
		},
	}
}

func (a *StopRecordingActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *StopRecordingActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		nil,
		true,
	)

	res, err := mobile.StopVideoRecording(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
}

type StopIOSRecordingActivity struct {
	workflowengine.BaseActivity
}

func NewStopIOSRecordingActivity() *StopIOSRecordingActivity {
	return &StopIOSRecordingActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Stop recording iOS device screen",
		},
	}
}

func (a *StopIOSRecordingActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *StopIOSRecordingActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		nil,
		true,
	)

	res, err := mobile.StopIOSVideoRecording(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
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
	runInput := buildMobileInput(
		input.Payload,
		a.NewActivityError,
		map[string]mobile.ErrorCode{
			"TempFileCreationFailed": {
				Code:        errorcodes.Codes[errorcodes.TempFileCreationFailed].Code,
				Description: errorcodes.Codes[errorcodes.TempFileCreationFailed].Description,
			},
		},
		true,
	)

	res, err := mobile.RunMobileFlow(ctx, runInput)
	if err != nil {
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{
		Output: res["output"],
	}, nil
}

func buildMobileInput(
	payload any,
	newErr func(code string, msg string, details ...any) error,
	extraErrorCodes map[string]mobile.ErrorCode,
	withCommand bool,
) mobile.MobileActivityInput {
	baseCodes := map[string]mobile.ErrorCode{
		"MissingOrInvalidPayload": {
			Code:        errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
			Description: errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Description,
		},
		"CommandExecutionFailed": {
			Code:        errorcodes.Codes[errorcodes.CommandExecutionFailed].Code,
			Description: errorcodes.Codes[errorcodes.CommandExecutionFailed].Description,
		},
	}

	for k, v := range extraErrorCodes {
		baseCodes[k] = v
	}

	in := mobile.MobileActivityInput{
		Payload:          payload,
		GetEnv:           utils.GetEnvironmentVariable,
		NewActivityError: newErr,
		ErrorCodes:       baseCodes,
	}

	if withCommand {
		in.CommandContext = exec.CommandContext
	}

	return in
}
