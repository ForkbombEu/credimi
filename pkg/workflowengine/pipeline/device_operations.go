// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"
	"sort"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevice"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	deviceCleanupStartToClose    = 10 * time.Minute
	deviceCleanupScheduleToClose = 20 * time.Minute
	deviceCleanupHeartbeat       = 30 * time.Second
	deviceCleanupMaximumAttempts = 3

	// deviceOperationScheduleToStart bounds how long a mobile-device Nexus
	// operation waits for the runner namespace's worker to start it, so a
	// disabled or deleted runner fails the pipeline instead of stalling it.
	// It covers a Credimi restart without failing running pipelines.
	deviceOperationScheduleToStart = 5 * time.Minute

	// runnerNamespaceConfigKey is the runner namespace a mobile step runs
	// in, and the Nexus endpoint the pipeline reaches it through.
	runnerNamespaceConfigKey = "runner_namespace"
	// pipelineRunConfigKey names the pipeline run of a mobile step, also
	// inside a linked steps block.
	pipelineRunConfigKey = "pipeline_run"
	// pipelineExecutionTimeoutConfigKey is the pipeline's execution timeout,
	// which bounds a mobile step's workflow.
	pipelineExecutionTimeoutConfigKey = "pipeline_execution_timeout"
)

// MobileDevicesCleanupHook cleans every device the pipeline prepared, through
// the Clean up operation of the device's runner namespace, unless the
// pipeline's cancellation policy skips it, and merges the evidence it stored.
func MobileDevicesCleanupHook(
	ctx workflow.Context,
	wfDef *pipeline.WorkflowDefinition,
	ao *workflow.ActivityOptions,
	config map[string]any,
	runData map[string]any,
	output *map[string]any,
) error {
	ctx, _ = workflow.NewDisconnectedContext(ctx)
	devices, _ := runData["setted_devices"].(map[string]any)
	deviceIDs := make([]string, 0, len(devices))
	for deviceID := range devices {
		deviceIDs = append(deviceIDs, deviceID)
	}
	sort.Strings(deviceIDs)

	pipelineRun := currentPipelineRun(ctx)
	var runtime pipeline.RuntimeConfig
	if wfDef != nil {
		runtime = wfDef.Runtime
	}
	executionTimeout := PrepareWorkflowOptions(runtime).Options.WorkflowExecutionTimeout
	var activityOptions workflow.ActivityOptions
	if ao != nil {
		activityOptions = *ao
	}
	var cleanupErrs []string
	for _, deviceID := range deviceIDs {
		device, _ := devices[deviceID].(map[string]any)
		runnerNamespace, _ := device[runnerNamespaceConfigKey].(string)
		if runnerNamespace == "" {
			continue
		}
		if shouldSkipDeviceCleanup(runData, deviceID) {
			appendCleanupWarning(
				output,
				fmt.Sprintf(
					"device cleanup skipped for %s because pipeline cancellation policy requested it",
					deviceID,
				),
			)
			continue
		}
		var cleanup mobiledevice.CleanUpResult
		if err := workflow.NewNexusClient(runnerNamespace, mobiledevice.ServiceName).
			ExecuteOperation(
				ctx,
				mobiledevice.CleanUpOperationName,
				mobiledevice.CleanUpInput{
					WorkflowID: mobiledevice.CleanUpWorkflowID(
						pipelineRun.WorkflowID,
						deviceID,
					),
					DeviceID:         deviceID,
					Device:           device,
					RunIdentifier:    workflowengine.AsString(runData["run_identifier"]),
					ExecutionTimeout: executionTimeout,
					Pipeline:         pipelineRun,
					Workflow: workflowengine.WorkflowInput{
						Config:          config,
						ActivityOptions: &activityOptions,
					},
				},
				workflow.NexusOperationOptions{
					ScheduleToCloseTimeout: executionTimeout,
					ScheduleToStartTimeout: deviceOperationScheduleToStart,
					Summary:                "device " + deviceID,
				},
			).Get(ctx, &cleanup); err != nil {
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

// deviceCleanupActivityOptions bound the cleanup of one device.
func deviceCleanupActivityOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout:    deviceCleanupStartToClose,
		ScheduleToCloseTimeout: deviceCleanupScheduleToClose,
		HeartbeatTimeout:       deviceCleanupHeartbeat,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: deviceCleanupMaximumAttempts,
		},
	}
}

// deviceWorkflowContext applies the pipeline's activity options to ctx and
// returns them with the pipeline's config.
func deviceWorkflowContext(
	ctx workflow.Context,
	in workflowengine.WorkflowInput,
) (workflow.Context, workflow.ActivityOptions, map[string]any) {
	var ao workflow.ActivityOptions
	if in.ActivityOptions != nil {
		ao = *in.ActivityOptions
	}
	config := in.Config
	if config == nil {
		config = map[string]any{}
	}
	return workflow.WithActivityOptions(ctx, ao), ao, config
}

// PrepareDeviceWorkflow prepares one device of a pipeline run in its runner's
// namespace: it sets the device up for every mobile step that uses it, then
// disables the Play Store when asked and starts the device log and the
// recording. When the setup fails it cleans the device before failing.
func PrepareDeviceWorkflow(
	ctx workflow.Context,
	in mobiledevice.PrepareInput,
) (mobiledevice.PrepareResult, error) {
	ctx, ao, config := deviceWorkflowContext(ctx, in.Workflow)
	settedDevices := map[string]any{}
	runData := map[string]any{
		"run_identifier": in.RunIdentifier,
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
		return mobiledevice.PrepareResult{}, err
	}

	steps := map[string]mobiledevice.PreparedMobileDevice{}
	if err := setUpDevice(ctx, ao, in, settedDevices, steps); err != nil {
		cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
		cleanupAO := deviceCleanupActivityOptions()
		output := map[string]any{}
		cleanupErr := MobileAutomationCleanupHook(
			cleanupCtx,
			nil,
			&cleanupAO,
			config,
			runData,
			&output,
		)
		if cleanupErr == nil {
			return mobiledevice.PrepareResult{}, err
		}
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		return mobiledevice.PrepareResult{}, workflowengine.NewAppError(
			workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: fmt.Sprintf("%v; cleanup: %v", err, cleanupErr),
			},
		)
	}
	device, _ := settedDevices[in.DeviceID].(map[string]any)
	return mobiledevice.PrepareResult{Device: device, Steps: steps}, nil
}

// setUpDevice prepares the device for every request, then disables the Play
// Store when asked and starts the device log and the recording. It stops at
// the first error.
func setUpDevice(
	ctx workflow.Context,
	ao workflow.ActivityOptions,
	in mobiledevice.PrepareInput,
	settedDevices map[string]any,
	steps map[string]mobiledevice.PreparedMobileDevice,
) error {
	for _, req := range in.Requests {
		device, err := prepareMobileDevice(ctx, &ao, settedDevices, req)
		if err != nil {
			return err
		}
		steps[req.StepID] = device
	}
	if in.DisablePlayStore {
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
		runIdentifier: in.RunIdentifier,
	}
	// The device log is always captured; only the screen recording is optional.
	if err := startDeviceLogForDevices(captureInput); err != nil {
		return err
	}
	// no_video pipelines run for hours: a recording capped at 30 minutes is useless.
	if in.NoVideo {
		return nil
	}
	return startRecordingForDevices(captureInput)
}

// CleanUpDeviceWorkflow runs the pipeline's device cleanup on one device in
// its runner's namespace and returns the evidence it stored and what went
// wrong. It never fails: the pipeline reports the errors it returns.
func CleanUpDeviceWorkflow(
	ctx workflow.Context,
	in mobiledevice.CleanUpInput,
) (mobiledevice.CleanUpResult, error) {
	ctx, _, config := deviceWorkflowContext(ctx, in.Workflow)
	runData := map[string]any{
		"run_identifier": in.RunIdentifier,
		"setted_devices": map[string]any{in.DeviceID: deviceEntryFromJSON(in.Device)},
	}
	cleanupAO := deviceCleanupActivityOptions()
	output := map[string]any{}
	err := MobileAutomationCleanupHook(ctx, nil, &cleanupAO, config, runData, &output)
	result := mobiledevice.CleanUpResult{
		ResultVideoURLs: workflowengine.AsSliceOfStrings(output["result_video_urls"]),
		ScreenshotURLs:  workflowengine.AsSliceOfStrings(output["screenshot_urls"]),
		Warnings:        workflowengine.AsSliceOfStrings(output[cleanupWarningsOutputKey]),
	}
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	}
	return result, nil
}

// deviceEntryFromJSON restores the Go types the cleanup reads from a
// setted_devices entry that crossed the Nexus operations as JSON: process
// IDs as int and the installed packages as map[string]string.
func deviceEntryFromJSON(device map[string]any) map[string]any {
	restored := make(map[string]any, len(device))
	for key, value := range device {
		restored[key] = value
	}
	for _, key := range []string{"device_log_pid", "recording_process_pid", "recording_ffmpeg_pid"} {
		if pid, ok := restored[key].(float64); ok {
			restored[key] = int(pid)
		}
	}
	if installed, ok := restored["installed"].(map[string]any); ok {
		packages := make(map[string]string, len(installed))
		for version, packageID := range installed {
			packages[version], _ = packageID.(string)
		}
		restored["installed"] = packages
	}
	return restored
}
