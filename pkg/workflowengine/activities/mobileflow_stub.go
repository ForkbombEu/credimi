//go:build !credimi_extra

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

const mobileAutomationDisabledMessage = "mobile automation is disabled; build with -tags=credimi_extra"

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
	return workflowengine.ActivityResult{}, a.NewActivityError(
		errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
		mobileAutomationDisabledMessage,
	)
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
	return workflowengine.ActivityResult{}, a.NewActivityError(
		errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
		mobileAutomationDisabledMessage,
	)
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
	return workflowengine.ActivityResult{}, a.NewActivityError(
		errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
		mobileAutomationDisabledMessage,
	)
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
	return workflowengine.ActivityResult{}, a.NewActivityError(
		errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
		mobileAutomationDisabledMessage,
	)
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
	return workflowengine.ActivityResult{}, a.NewActivityError(
		errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
		mobileAutomationDisabledMessage,
	)
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
	return workflowengine.ActivityResult{}, a.NewActivityError(
		errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
		mobileAutomationDisabledMessage,
	)
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
	return workflowengine.ActivityResult{}, a.NewActivityError(
		errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
		mobileAutomationDisabledMessage,
	)
}
