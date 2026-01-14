// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/forkbombeu/credimi-extra/mobile"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/activity"
)

var mobileTracer = otel.Tracer("credimi/workflowengine/mobile")

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
	ctx, span := mobileTracer.Start(ctx, "StartEmulatorActivity")
	defer span.End()
	annotateActivitySpan(ctx, span)

	runInput := buildMobileInput(
		ctx,
		input.Payload,
		a.NewActivityError,
		nil,
		false,
	)

	res, err := mobile.StartEmulator(ctx, runInput)
	if err != nil {
		span.RecordError(err)
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
	ctx, span := mobileTracer.Start(ctx, "ApkInstallActivity")
	defer span.End()
	annotateActivitySpan(ctx, span)

	runInput := buildMobileInput(
		ctx,
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

	res, err := mobile.ApkInstall(ctx, runInput)
	if err != nil {
		span.RecordError(err)
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
	ctx, span := mobileTracer.Start(ctx, "UnlockEmulatorActivity")
	defer span.End()
	annotateActivitySpan(ctx, span)

	runInput := buildMobileInput(
		ctx,
		input.Payload,
		a.NewActivityError,
		nil,
		true,
	)

	res, err := mobile.UnlockEmulator(ctx, runInput)
	if err != nil {
		span.RecordError(err)
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{Output: res}, nil
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
	ctx, span := mobileTracer.Start(ctx, "StopEmulatorActivity")
	defer span.End()
	annotateActivitySpan(ctx, span)

	runInput := buildMobileInput(
		ctx,
		input.Payload,
		a.NewActivityError,
		nil,
		false,
	)

	res, err := mobile.StopEmulator(ctx, runInput)
	if err != nil {
		span.RecordError(err)
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
			Name: "Start recording emulator screen",
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
	ctx, span := mobileTracer.Start(ctx, "StartRecordingActivity")
	defer span.End()
	annotateActivitySpan(ctx, span)

	runInput := buildMobileInput(
		ctx,
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
		span.RecordError(err)
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
			Name: "Stop recording emulator screen",
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
	ctx, span := mobileTracer.Start(ctx, "StopRecordingActivity")
	defer span.End()
	annotateActivitySpan(ctx, span)

	runInput := buildMobileInput(
		ctx,
		input.Payload,
		a.NewActivityError,
		nil,
		true,
	)

	res, err := mobile.StopVideoRecording(ctx, runInput)
	if err != nil {
		span.RecordError(err)
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
	ctx, span := mobileTracer.Start(ctx, "RunMobileFlowActivity")
	defer span.End()
	annotateActivitySpan(ctx, span)

	runInput := buildMobileInput(
		ctx,
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
		span.RecordError(err)
		return workflowengine.ActivityResult{}, err
	}

	return workflowengine.ActivityResult{
		Output: res["output"],
	}, nil
}

func buildMobileInput(
	ctx context.Context,
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

	activityInfo := activity.GetInfo(ctx)
	correlationID := fmt.Sprintf("%s/%s", activityInfo.WorkflowExecution.ID, activityInfo.ActivityID)
	logger := activity.GetLogger(ctx)
	baseFields := []any{
		"correlation_id", correlationID,
		"workflow_id", activityInfo.WorkflowExecution.ID,
		"activity_id", activityInfo.ActivityID,
	}

	logInfo := func(message string, keyValues ...any) {
		fields := append([]any{}, baseFields...)
		fields = append(fields, keyValues...)
		logger.Info(message, fields...)
	}

	logError := func(message string, keyValues ...any) {
		fields := append([]any{}, baseFields...)
		fields = append(fields, keyValues...)
		logger.Error(message, fields...)
	}

	in := mobile.MobileActivityInput{
		Payload:          payload,
		GetEnv:           utils.GetEnvironmentVariable,
		NewActivityError: newErr,
		ErrorCodes:       baseCodes,
		CorrelationID:    correlationID,
		LogInfo:          logInfo,
		LogError:         logError,
	}

	if withCommand {
		in.CommandContext = exec.CommandContext
	}

	return in
}

func annotateActivitySpan(ctx context.Context, span trace.Span) {
	activityInfo := activity.GetInfo(ctx)
	span.SetAttributes(
		attribute.String("workflow.id", activityInfo.WorkflowExecution.ID),
		attribute.String("activity.id", activityInfo.ActivityID),
	)
}
