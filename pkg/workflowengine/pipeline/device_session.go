// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesession"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// deviceSessionCheckInterval is how often a held device session checks
	// that its pipeline run still runs, so a terminated pipeline frees it.
	deviceSessionCheckInterval = 2 * time.Minute

	deviceSessionCleanupStartToClose    = 10 * time.Minute
	deviceSessionCleanupScheduleToClose = 20 * time.Minute
	deviceSessionCleanupHeartbeat       = 30 * time.Second
	deviceSessionCleanupMaximumAttempts = 3

	// deviceSessionActivityTimeout bounds one attempt of the activities that
	// start, wait for and release a device session: preparing or cleaning a
	// device runs many runner activities.
	deviceSessionActivityTimeout = time.Hour

	// deviceSessionsRunDataKey lists the device sessions a pipeline started.
	deviceSessionsRunDataKey = "device_sessions"
	// runnerNamespaceConfigKey is the runner namespace a mobile step runs in.
	runnerNamespaceConfigKey = "runner_namespace"
	// pipelineRunConfigKey names the pipeline run of a mobile step, also
	// inside a linked steps block.
	pipelineRunConfigKey = "pipeline_run"
)

// MobileDeviceSessionsCleanupHook releases every device session the pipeline
// started. Each session cleans its device, unless the pipeline's cancellation
// policy skips it, and returns the evidence it stored.
func MobileDeviceSessionsCleanupHook(
	ctx workflow.Context,
	_ *pipeline.WorkflowDefinition,
	_ *workflow.ActivityOptions,
	_ map[string]any,
	runData map[string]any,
	output *map[string]any,
) error {
	ctx, _ = workflow.NewDisconnectedContext(ctx)
	sessions, _ := runData[deviceSessionsRunDataKey].([]map[string]any)
	var cleanupErrs []string
	for _, session := range sessions {
		deviceID := workflowengine.AsString(session["device_id"])
		skip := shouldSkipDeviceCleanup(runData, deviceID)
		if skip {
			appendCleanupWarning(
				output,
				fmt.Sprintf(
					"device cleanup skipped for %s because pipeline cancellation policy requested it",
					deviceID,
				),
			)
		}
		ref := mobiledevicesession.RunnerWorkflowRef{
			Namespace:  workflowengine.AsString(session["namespace"]),
			WorkflowID: workflowengine.AsString(session["workflow_id"]),
			RunID:      workflowengine.AsString(session["run_id"]),
		}
		var res workflowengine.ActivityResult
		err := workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, deviceSessionActivityOptions(ref.WorkflowID)),
			activities.ReleaseDeviceSessionActivityName,
			workflowengine.ActivityInput{Payload: activities.ReleaseDeviceSessionPayload{
				Ref:         ref,
				SkipCleanup: skip,
			}},
		).Get(ctx, &res)
		if err != nil {
			cleanupErrs = append(cleanupErrs, err.Error())
			continue
		}
		cleanup, err := workflowengine.DecodePayload[mobiledevicesession.Cleanup](res.Output)
		if err != nil {
			cleanupErrs = append(cleanupErrs, err.Error())
			continue
		}
		if *output == nil {
			*output = map[string]any{}
		}
		if len(cleanup.ResultVideoURLs) > 0 {
			(*output)["result_video_urls"] = appendUniqueStrings(
				workflowengine.AsSliceOfStrings((*output)["result_video_urls"]),
				cleanup.ResultVideoURLs...,
			)
		}
		if len(cleanup.ScreenshotURLs) > 0 {
			(*output)["screenshot_urls"] = appendUniqueStrings(
				workflowengine.AsSliceOfStrings((*output)["screenshot_urls"]),
				cleanup.ScreenshotURLs...,
			)
		}
		for _, warning := range cleanup.Warnings {
			appendCleanupWarning(output, warning)
		}
		cleanupErrs = append(cleanupErrs, cleanup.Errors...)
	}
	if len(cleanupErrs) == 0 {
		return nil
	}
	errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
	return workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: "one or more errors occurred during mobile automation cleanup",
		Details: map[string]any{"payload": cleanupErrs},
	})
}

// DeviceSessionPayload is the payload of the device session of one device of
// a pipeline run.
type DeviceSessionPayload struct {
	DeviceID string `json:"device_id"`
	// Requests prepares the device for every mobile step that uses it.
	Requests         []MobileDeviceRequest              `json:"requests"`
	RunIdentifier    string                             `json:"run_identifier"`
	DisablePlayStore bool                               `json:"disable_play_store"`
	NoVideo          bool                               `json:"no_video"`
	Pipeline         mobiledevicesession.PipelineRunRef `json:"pipeline"`
}

// DeviceSessionReady is what the ready update answers: the prepared device of
// each step, by step ID, and the device's state, which the pipeline exposes
// to the live view.
type DeviceSessionReady struct {
	Prepared map[string]PreparedMobileDevice `json:"prepared"`
	Device   map[string]any                  `json:"device"`
}

// DeviceSessionWorkflow prepares one device for a pipeline run in the
// device's runner namespace, holds it while the pipeline runs its steps and
// cleans it when the pipeline releases it.
type DeviceSessionWorkflow struct{}

// NewDeviceSessionWorkflow returns the device session workflow.
func NewDeviceSessionWorkflow() *DeviceSessionWorkflow {
	return &DeviceSessionWorkflow{}
}

// Name returns the Temporal workflow type of a device session.
func (DeviceSessionWorkflow) Name() string {
	return mobiledevicesession.WorkflowName
}

// Workflow runs the device session.
func (w *DeviceSessionWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	payload, err := workflowengine.DecodePayload[DeviceSessionPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewNonRetryableAppError(
			invalidDeviceSessionPayload(err),
		)
	}
	var ao workflow.ActivityOptions
	if input.ActivityOptions != nil {
		ao = *input.ActivityOptions
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	config := input.Config
	if config == nil {
		config = map[string]any{}
	}

	settedDevices := map[string]any{}
	runData := map[string]any{
		"run_identifier": payload.RunIdentifier,
		"setted_devices": settedDevices,
	}
	// The live view and debugging read the device with the pipeline's query.
	if err := workflow.SetQueryHandler(
		ctx,
		PipelineMobileDevicesQuery,
		func() (map[string]any, error) {
			return settedDevices, nil
		},
	); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	var (
		ready       = DeviceSessionReady{Prepared: map[string]PreparedMobileDevice{}}
		setupErr    error
		setupDone   bool
		released    bool
		skipCleanup bool
		cleanup     mobiledevicesession.Cleanup
		cleanupDone bool
	)
	// Handlers are registered before any blocking call, so updates sent
	// right after the start are delivered.
	if err := workflow.SetUpdateHandler(
		ctx,
		mobiledevicesession.ReadyUpdate,
		func(ctx workflow.Context) (DeviceSessionReady, error) {
			if err := workflow.Await(ctx, func() bool { return setupDone }); err != nil {
				return DeviceSessionReady{}, err
			}
			return ready, setupErr
		},
	); err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	if err := workflow.SetUpdateHandler(
		ctx,
		mobiledevicesession.ReleaseUpdate,
		func(ctx workflow.Context, release mobiledevicesession.Release) (
			mobiledevicesession.Cleanup,
			error,
		) {
			if !released {
				released = true
				skipCleanup = release.SkipCleanup
			}
			if err := workflow.Await(ctx, func() bool { return cleanupDone }); err != nil {
				return mobiledevicesession.Cleanup{}, err
			}
			return cleanup, nil
		},
	); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	setupErr = setupDeviceSession(ctx, ao, payload, settedDevices, ready.Prepared)
	if device, ok := settedDevices[payload.DeviceID].(map[string]any); ok {
		ready.Device = device
	}
	setupDone = true

	holdDeviceSession(ctx, payload, &released)

	if !skipCleanup {
		cleanup = cleanDeviceSession(ctx, config, runData)
	} else {
		logger.Info("device session cleanup skipped", "device_id", payload.DeviceID)
	}
	cleanupDone = true

	// The release update returns the cleanup, also when the session was canceled.
	finishCtx, _ := workflow.NewDisconnectedContext(ctx)
	if err := workflow.Await(finishCtx, func() bool {
		return workflow.AllHandlersFinished(finishCtx)
	}); err != nil {
		logger.Warn("device session handlers did not finish", "error", err)
	}
	return workflowengine.WorkflowResult{Output: cleanup}, nil
}

func invalidDeviceSessionPayload(err error) workflowengine.WorkflowError {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	return workflowengine.WorkflowError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: fmt.Sprintf("invalid device session payload: %v", err),
	}
}

// setupDeviceSession prepares the device for every request, then disables
// the Play Store when asked and starts the device log and the recording. It
// stops at the first error.
func setupDeviceSession(
	ctx workflow.Context,
	ao workflow.ActivityOptions,
	payload DeviceSessionPayload,
	settedDevices map[string]any,
	prepared map[string]PreparedMobileDevice,
) error {
	for _, req := range payload.Requests {
		device, err := prepareMobileDevice(ctx, &ao, settedDevices, req)
		if err != nil {
			return err
		}
		prepared[req.StepID] = device
	}
	if payload.DisablePlayStore {
		if err := disablePlayStoreForDevices(disablePlayStoreForDevicesInput{
			ctx:           ctx,
			settedDevices: settedDevices,
			ao:            &ao,
		}); err != nil {
			return err
		}
	}
	captureInput := deviceCaptureInput{
		ctx:           ctx,
		settedDevices: settedDevices,
		ao:            &ao,
		runIdentifier: payload.RunIdentifier,
	}
	// The device log is always captured; only the screen recording is optional.
	if err := startDeviceLogForDevices(captureInput); err != nil {
		return err
	}
	// no_video pipelines run for hours: a recording capped at 30 minutes is useless.
	if payload.NoVideo {
		return nil
	}
	return startRecordingForDevices(captureInput)
}

// holdDeviceSession waits until the pipeline releases the device. It also
// releases it when the pipeline run no longer runs, or when the session is
// canceled.
func holdDeviceSession(
	ctx workflow.Context,
	payload DeviceSessionPayload,
	released *bool,
) {
	logger := workflow.GetLogger(ctx)
	checkCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
	})
	for !*released {
		ok, err := workflow.AwaitWithTimeout(
			ctx,
			deviceSessionCheckInterval,
			func() bool { return *released },
		)
		if err != nil {
			// The session was canceled: clean the device.
			*released = true
			return
		}
		if ok {
			return
		}
		var result workflowengine.ActivityResult
		err = workflow.ExecuteActivity(
			checkCtx,
			activities.CheckPipelineRunActivityName,
			workflowengine.ActivityInput{Payload: activities.CheckPipelineRunPayload{
				Pipeline:        payload.Pipeline,
				RunnerNamespace: workflow.GetInfo(ctx).Namespace,
			}},
		).Get(checkCtx, &result)
		if err != nil {
			logger.Warn("device session could not check its pipeline run", "error", err)
			continue
		}
		check, err := workflowengine.DecodePayload[activities.CheckPipelineRunResult](
			result.Output,
		)
		if err == nil && !check.Running {
			logger.Info("pipeline run ended without releasing its device")
			*released = true
		}
	}
}

// cleanDeviceSession runs the pipeline's device cleanup and returns the
// evidence it stored and what went wrong.
func cleanDeviceSession(
	ctx workflow.Context,
	config map[string]any,
	runData map[string]any,
) mobiledevicesession.Cleanup {
	cleanupAO := workflow.ActivityOptions{
		StartToCloseTimeout:    deviceSessionCleanupStartToClose,
		ScheduleToCloseTimeout: deviceSessionCleanupScheduleToClose,
		HeartbeatTimeout:       deviceSessionCleanupHeartbeat,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: deviceSessionCleanupMaximumAttempts,
		},
	}
	output := map[string]any{}
	err := MobileAutomationCleanupHook(ctx, nil, &cleanupAO, config, runData, &output)
	cleanup := mobiledevicesession.Cleanup{
		ResultVideoURLs: workflowengine.AsSliceOfStrings(output["result_video_urls"]),
		ScreenshotURLs:  workflowengine.AsSliceOfStrings(output["screenshot_urls"]),
		Warnings:        workflowengine.AsSliceOfStrings(output[cleanupWarningsOutputKey]),
	}
	if err != nil {
		cleanup.Errors = append(cleanup.Errors, err.Error())
	}
	return cleanup
}
