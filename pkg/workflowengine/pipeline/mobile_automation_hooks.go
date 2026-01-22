// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/internal/telemetry"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/avdpool"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/workflow"
)

func MobileAutomationSetupHook(
	ctx workflow.Context,
	steps *[]StepDefinition,
	ao *workflow.ActivityOptions,
	config map[string]any,
	runData *map[string]any,
) (err error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, *ao)

	info := workflow.GetInfo(ctx)
	traceCtx := telemetry.ContextFromWorkflow(ctx)
	traceCtx, span := otel.Tracer("credimi/pipeline").Start(
		traceCtx,
		"pipeline.MobileAutomationSetupHook",
		trace.WithAttributes(
			attribute.String("namespace", info.Namespace),
			attribute.String("workflow_id", info.WorkflowExecution.ID),
		),
	)
	defer span.End()
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
	}()

	httpActivity := activities.NewHTTPActivity()
	startEmuActivity := activities.NewStartEmulatorActivity()
	installActivity := activities.NewApkInstallActivity()
	recordActivity := activities.NewStartRecordingActivity()
	// unlockActivity := activities.NewUnlockEmulatorActivity()
	mobileServerURL := utils.GetEnvironmentVariable("MAESTRO_WORKER", "http://localhost:8050")

	startedEmulators := make(map[string]any)
	if alreadyStartedEmu, ok := (*runData)["started_emulators"].(map[string]any); ok {
		startedEmulators = alreadyStartedEmu
	}
	poolWorkflowID := avdpool.DefaultPoolWorkflowID
	poolSlotAcquired, _ := (*runData)["pool_slot_acquired"].(bool)
	var heartbeatStop workflow.Channel
	if stopChannel, ok := (*runData)["pool_heartbeat_stop"].(workflow.Channel); ok {
		heartbeatStop = stopChannel
	}
	defer func() {
		if err == nil || !poolSlotAcquired {
			return
		}
		stopPoolHeartbeat(ctx, heartbeatStop, *runData)
		if releaseErr := avdpool.ReleaseSlot(ctx, poolWorkflowID); releaseErr != nil {
			logger.Warn("failed releasing pool slot after setup error", "error", releaseErr)
		}
	}()

	for i := range *steps {
		step := &(*steps)[i]

		if step.Use != "mobile-automation" {
			continue
		}

		SetConfigValue(&step.With.Config, "app_url", config["app_url"])

		logger.Info("MobileAutomationSetupHook: processing step", "id", step.ID)

		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		payload, err := workflowengine.DecodePayload[workflows.MobileAutomationWorkflowPipelinePayload](
			step.With.Payload,
		)
		if err != nil {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("error decoding payload for step %s: %s", step.ID, err.Error()),
			)
		}
		// If action_code is present, version_id is REQUIRED
		if payload.ActionCode != "" {
			if payload.VersionID == "" {
				return workflowengine.NewAppError(
					errCode,
					fmt.Sprintf("missing or invalid version_id for step %s", step.ID))
			}
		}
		// If action_code is NOT present -> action_id is REQUIRED
		if payload.ActionCode == "" {
			if payload.ActionID == "" {
				return workflowengine.NewAppError(
					errCode,
					fmt.Sprintf("missing or invalid action_id for step %s", step.ID),
				)
			}
		}
		appURL, ok := config["app_url"].(string)
		if !ok {
			errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("missing or invalid app_url for step %s", step.ID),
			)
		}

		req := workflowengine.ActivityInput{
			Payload: map[string]any{
				"method": "POST",
				"url":    utils.JoinURL(mobileServerURL, "fetch-apk-and-action"),
				"headers": map[string]any{
					"Content-Type": "application/json",
				},
				"body": map[string]any{
					"instance_url":       appURL,
					"version_identifier": payload.VersionID,
					"action_identifier":  payload.ActionID,
				},
				"expected_status": 200,
			},
		}

		var res workflowengine.ActivityResult
		if err := workflow.ExecuteActivity(ctx, httpActivity.Name(), req).
			Get(ctx, &res); err != nil {
			return err
		}
		errCode = errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		body, ok := res.Output.(map[string]any)["body"].(map[string]any)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("invalid HTTP response format for step %s", step.ID),
				res.Output,
			)
		}

		apkPath, ok := body["apk_path"].(string)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing apk_path in response for step %s",
					errCode.Description,
					step.ID,
				),
				body,
			)
		}
		versionIdentifier, ok := body["version_id"].(string)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing version_id in response for step %s",
					errCode.Description,
					step.ID,
				),
				body,
			)
		}
		versionIdentifier = strings.TrimPrefix(versionIdentifier, "/")
		actionCode := payload.ActionCode
		if actionCode == "" {
			actionCode, ok = body["code"].(string)
			if !ok || actionCode == "" {
				return workflowengine.NewAppError(
					errCode,
					fmt.Sprintf(
						"%s: missing action_code in response for step %s",
						errCode.Description,
						step.ID,
					),
					body,
				)
			}
			SetPayloadValue(&step.With.Payload, "action_code", actionCode)
			SetPayloadValue(&step.With.Payload, "stored_action_code", true)
		}
		SetPayloadValue(&step.With.Payload, "version_id", versionIdentifier)

		if emuInfo, ok := startedEmulators[versionIdentifier]; ok {
			emuInfoMap, ok := emuInfo.(map[string]any)
			if !ok {
				return workflowengine.NewAppError(
					errCode,
					fmt.Sprintf(
						"%s: invalid emulator info for step %s",
						errCode.Description,
						step.ID,
					),
					emuInfo,
				)
			}
			serial, ok := emuInfoMap["serial"].(string)
			if !ok || serial == "" {
				return workflowengine.NewAppError(
					errCode,
					fmt.Sprintf(
						"%s: missing serial in emulator info for step %s",
						errCode.Description,
						step.ID,
					),
					emuInfo,
				)
			}
			logger.Info(
				"Emulator already started, skipping start",
				"version",
				versionIdentifier,
				"serial",
				serial,
			)
			span.SetAttributes(
				attribute.String("emulator_serial", serial),
				attribute.String("version_id", versionIdentifier),
			)
			(*runData)["latest_emulator_serial"] = serial
			(*runData)["latest_version_id"] = versionIdentifier
			(*runData)["status"] = "running"
			_ = workflowengine.UpsertSearchAttributes(ctx, map[string]any{
				"emulator_serial": serial,
				"version_id":      versionIdentifier,
				"boot_status":     "already_running",
				"status":          "running",
			})
			SetPayloadValue(&step.With.Payload, "emulator_serial", serial)
			continue
		}

		if !poolSlotAcquired {
			poolWaitStart := workflow.Now(ctx)
			(*runData)["status"] = "queued"
			_ = workflowengine.UpsertSearchAttributes(ctx, map[string]any{
				"status": "queued",
			})
			if err := avdpool.AcquireSlot(ctx, poolWorkflowID, time.Minute); err != nil {
				return err
			}
			poolWait := workflow.Now(ctx).Sub(poolWaitStart)
			poolWaitMs := int64(poolWait.Milliseconds())
			(*runData)["pool_wait_ms"] = poolWaitMs
			_ = workflowengine.UpsertSearchAttributes(ctx, map[string]any{
				"pool_wait_ms": poolWaitMs,
			})
			poolSlotAcquired = true
			(*runData)["pool_slot_acquired"] = true
			if heartbeatStop == nil {
				heartbeatStop = workflow.NewChannel(ctx)
				(*runData)["pool_heartbeat_stop"] = heartbeatStop
				startPoolHeartbeat(ctx, poolWorkflowID, heartbeatStop, avdpool.DefaultHeartbeatInterval)
			}
		}

		mobileAo := *ao
		mobileAo.TaskQueue = workflows.MobileAutomationTaskQueue
		mobileCtx := workflow.WithActivityOptions(ctx, mobileAo)

		startEmulatorAo := mobileAo
		startEmulatorAo.HeartbeatTimeout = time.Minute
		startEmulatorAo.StartToCloseTimeout = 10 * time.Minute
		startEmulatorAo.ScheduleToCloseTimeout = 10 * time.Minute
		startEmulatorCtx := workflow.WithActivityOptions(ctx, startEmulatorAo)

		recordAo := mobileAo
		recordAo.HeartbeatTimeout = time.Minute
		recordAo.StartToCloseTimeout = 35 * time.Minute
		recordAo.ScheduleToCloseTimeout = 35 * time.Minute
		recordCtx := workflow.WithActivityOptions(ctx, recordAo)

		bootStart := workflow.Now(ctx)
		(*runData)["status"] = "booting"
		_ = workflowengine.UpsertSearchAttributes(ctx, map[string]any{
			"status": "booting",
		})
		startResult := workflowengine.ActivityResult{}
		startEmuInput := workflowengine.ActivityInput{
			Payload: map[string]any{"version_id": versionIdentifier},
		}
		err = workflow.ExecuteActivity(startEmulatorCtx, startEmuActivity.Name(), startEmuInput).
			Get(startEmulatorCtx, &startResult)
		if err != nil {
			return err
		}
		serial, ok := startResult.Output.(map[string]any)["serial"].(string)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing serial in response for step %s",
					errCode.Description,
					step.ID,
				),
				startResult.Output,
			)
		}
		cloneName, ok := startResult.Output.(map[string]any)["clone_name"].(string)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing clone_name in response for step %s",
					errCode.Description,
					step.ID,
				),
				startResult.Output,
			)
		}
		span.SetAttributes(
			attribute.String("emulator_serial", serial),
			attribute.String("clone_name", cloneName),
			attribute.String("version_id", versionIdentifier),
		)
		bootDuration := workflow.Now(ctx).Sub(bootStart)
		bootTimeMs := int64(bootDuration.Milliseconds())
		(*runData)["boot_time_ms"] = bootTimeMs
		(*runData)["boot_status"] = "ready"
		(*runData)["latest_emulator_serial"] = serial
		(*runData)["latest_clone_name"] = cloneName
		(*runData)["latest_version_id"] = versionIdentifier
		_ = workflowengine.UpsertSearchAttributes(ctx, map[string]any{
			"emulator_serial": serial,
			"version_id":      versionIdentifier,
			"clone_name":      cloneName,
			"boot_status":     "ready",
			"boot_time_ms":    bootTimeMs,
			"status":          "running",
		})
		SetPayloadValue(&step.With.Payload, "clone_name", cloneName)
		SetPayloadValue(&step.With.Payload, "emulator_serial", serial)
		appendCleanupStepSpec(runData, cleanupSpecForEmulator(serial, cloneName))

		installInput := workflowengine.ActivityInput{
			Payload: map[string]any{"apk": apkPath, "emulator_serial": serial},
		}
		if err := workflow.ExecuteActivity(mobileCtx, installActivity.Name(), installInput).
			Get(mobileCtx, nil); err != nil {
			return err
		}
		startRecordInput := workflowengine.ActivityInput{
			Payload: map[string]any{
				"emulator_serial": serial,
				"workflow_id":     workflow.GetInfo(mobileCtx).WorkflowExecution.ID,
			},
		}
		var recordResult workflowengine.ActivityResult
		if err := workflow.ExecuteActivity(
			recordCtx,
			recordActivity.Name(),
			startRecordInput,
		).Get(recordCtx, &recordResult); err != nil {
			return err
		}
		adbPID, ok := recordResult.Output.(map[string]any)["adb_process_pid"].(float64)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing adb_process in start record video response for step %s",
					errCode.Description,
					step.ID,
				),
				recordResult.Output,
			)
		}
		ffmpegPID, ok := recordResult.Output.(map[string]any)["ffmpeg_process_pid"].(float64)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing ffmpeg_process in start record video response for step %s",
					errCode.Description,
					step.ID,
				),
				recordResult.Output,
			)
		}
		logcatPID, ok := recordResult.Output.(map[string]any)["logcat_process_pid"].(float64)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing logcat_process in start record video response for step %s",
					errCode.Description,
					step.ID,
				),
				recordResult.Output,
			)
		}
		videoPath, ok := recordResult.Output.(map[string]any)["video_path"].(string)
		if !ok {
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing video_path in start record video response for step %s",
					errCode.Description,
					step.ID,
				),
				recordResult.Output,
			)
		}
		SetPayloadValue(&step.With.Payload, "recording_adb_pid", int(adbPID))
		SetPayloadValue(&step.With.Payload, "recording_ffmpeg_pid", int(ffmpegPID))
		SetPayloadValue(&step.With.Payload, "recording_logcat_pid", int(logcatPID))
		(*runData)["recording_active"] = true
		(*runData)["status"] = "recording"
		runIdentifier, ok := (*runData)["run_identifier"].(string)
		if !ok || runIdentifier == "" {
			return workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
				fmt.Sprintf("missing run_identifier for step %s", step.ID),
			)
		}
		appendCleanupStepSpec(runData, cleanupSpecForRecording(StopRecordingCleanupPayload{
			EmulatorSerial:   serial,
			AdbProcessPid:    int(adbPID),
			FfmpegProcessPid: int(ffmpegPID),
			LogcatProcessPid: int(logcatPID),
			VideoPath:        videoPath,
			RunIdentifier:    runIdentifier,
			VersionID:        versionIdentifier,
			AppURL:           appURL,
		}))
		_ = workflowengine.UpsertSearchAttributes(ctx, map[string]any{
			"status": "recording",
		})
		startedEmulators[versionIdentifier] = map[string]any{
			"serial":     serial,
			"recording":  true,
			"video_path": videoPath,
		}
		SetRunDataValue(runData, "started_emulators", startedEmulators)
		// unlockInput := workflowengine.ActivityInput{Payload: map[string]any{"emulator_serial": serial}}
		// if err := workflow.ExecuteActivity(mobileCtx, unlockActivity.Name(), unlockInput).Get(ctx, nil); err != nil {
		// 	return err
		// }
	}

	return nil
}

func MobileAutomationCleanupHook(
	ctx workflow.Context,
	steps []StepDefinition,
	ao *workflow.ActivityOptions,
	config map[string]any,
	runData map[string]any,
	output *map[string]any,
) error {
	ctx, _ = workflow.NewDisconnectedContext(ctx)
	logger := workflow.GetLogger(ctx)
	appURL, ok := config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
			"missing or invalid app_url in workflow input config",
		)
	}
	poolWorkflowID := avdpool.DefaultPoolWorkflowID
	poolSlotAcquired, _ := runData["pool_slot_acquired"].(bool)
	if stopChannel, ok := runData["pool_heartbeat_stop"].(workflow.Channel); ok {
		stopPoolHeartbeat(ctx, stopChannel, runData)
	}
	if poolSlotAcquired {
		defer func() {
			if err := avdpool.ReleaseSlot(ctx, poolWorkflowID); err != nil {
				logger.Warn("failed releasing pool slot during cleanup", "error", err)
			}
		}()
	}

	specs := cleanupStepSpecs(runData)
	if err := validateCleanupSpecs(specs); err != nil {
		return err
	}
	if len(specs) == 0 {
		return nil
	}

	baseAo := workflow.ActivityOptions{}
	if ao != nil {
		baseAo = *ao
	}
	options := buildCleanupOptions(baseAo)
	recordFailure := func(ctx workflow.Context, spec CleanupStepSpec, stepErr error, attempts int) error {
		info := workflow.GetInfo(ctx)
		return recordFailedCleanup(ctx, options, spec, stepErr, attempts, info.WorkflowExecution.ID)
	}

	cleanupErrors := executeCleanupSpecs(ctx, logger, options, specs, output, recordFailure)
	if len(cleanupErrors) > 0 {
		logger.Warn("cleanup saga recorded failures", "count", len(cleanupErrors))
	}

	if err := startCleanupVerificationWorkflow(ctx, specs, appURL, runData); err != nil {
		logger.Warn("failed to start cleanup verification workflow", "error", err)
	}

	return nil
}

func cleanupRecording(
	ctx workflow.Context,
	recordAo workflow.ActivityOptions,
	payload workflows.MobileAutomationWorkflowPayload,
	runData map[string]any,
	output *map[string]any,
	cleanupErrs *[]error,
	appURL string,
) {
	logger := workflow.GetLogger(ctx)
	recordCtx := workflow.WithActivityOptions(ctx, recordAo)

	startedEmulators, ok := runData["started_emulators"].(map[string]any)
	if !ok {
		return
	}

	emuInfo, ok := startedEmulators[payload.VersionID].(map[string]any)
	if !ok {
		return
	}

	recording, ok := emuInfo["recording"].(bool)
	if !ok || !recording {
		return
	}

	videoPath, ok := emuInfo["video_path"].(string)
	if !ok || videoPath == "" {
		*cleanupErrs = append(*cleanupErrs,
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
				"missing video_path for emulator "+payload.EmulatorSerial,
			),
		)
		return
	}

	stopRecordingActivity := activities.NewStopRecordingActivity()
	var stopResult workflowengine.ActivityResult

	if err := workflow.ExecuteActivity(
		recordCtx,
		stopRecordingActivity.Name(),
		workflowengine.ActivityInput{
			Payload: map[string]any{
				"emulator_serial":    payload.EmulatorSerial,
				"video_path":         videoPath,
				"adb_process_pid":    payload.RecordingAdbPid,
				"ffmpeg_process_pid": payload.RecordingFfmpegPid,
				"logcat_process_pid": payload.RecordingLogcatPid,
			},
		},
	).Get(recordCtx, &stopResult); err != nil {
		logger.Error("cleanup: stop recording failed", "error", err)
		*cleanupErrs = append(*cleanupErrs, err)
		return
	}

	lastFramePath, ok := stopResult.Output.(map[string]any)["last_frame_path"].(string)
	if !ok || lastFramePath == "" {
		*cleanupErrs = append(*cleanupErrs,
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
				"missing last_frame_path in stop recording result",
				stopResult.Output,
			),
		)
		return
	}

	runIdentifier, ok := runData["run_identifier"].(string)
	if !ok || runIdentifier == "" {
		*cleanupErrs = append(*cleanupErrs,
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
				"missing run_identifier in run data",
			),
		)
		return
	}

	httpActivity := activities.NewHTTPActivity()
	var storeResult workflowengine.ActivityResult

	if err := workflow.ExecuteActivity(
		ctx,
		httpActivity.Name(),
		workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method: http.MethodPost,
				URL: utils.JoinURL(
					utils.GetEnvironmentVariable("MAESTRO_WORKER", ""),
					"store-pipeline-result",
				),
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]any{
					"video_path":         videoPath,
					"last_frame_path":    lastFramePath,
					"run_identifier":     runIdentifier,
					"version_identifier": payload.VersionID,
					"instance_url":       appURL,
				},
				ExpectedStatus: 200,
			},
		},
	).Get(ctx, &storeResult); err != nil {
		logger.Error("cleanup: store result failed", "error", err)
		*cleanupErrs = append(*cleanupErrs, err)
		return
	}

	body, ok := storeResult.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		*cleanupErrs = append(*cleanupErrs,
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
				"missing body in store result",
				storeResult.Output,
			),
		)
		return
	}
	resultURLs := workflowengine.AsSliceOfStrings(body["result_urls"])
	frameURLs := workflowengine.AsSliceOfStrings(body["screenshot_urls"])

	if len(resultURLs) == 0 || len(frameURLs) == 0 {
		*cleanupErrs = append(*cleanupErrs,
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
				"missing result or screenshot URLs",
				storeResult.Output,
			),
		)
		return
	}

	if *output == nil {
		*output = make(map[string]any)
	}

	(*output)["result_video_urls"] =
		append((*output)["result_video_urls"].([]string), resultURLs...)
	(*output)["screenshot_urls"] =
		append((*output)["screenshot_urls"].([]string), frameURLs...)
}

func startPoolHeartbeat(
	ctx workflow.Context,
	poolWorkflowID string,
	stopCh workflow.Channel,
	interval time.Duration,
) {
	logger := workflow.GetLogger(ctx)
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			var stopRequested bool
			selector := workflow.NewSelector(ctx)
			timer := workflow.NewTimer(ctx, interval)

			selector.AddFuture(timer, func(f workflow.Future) {
				if err := f.Get(ctx, nil); err != nil {
					return
				}
				if err := avdpool.SendHeartbeat(ctx, poolWorkflowID); err != nil {
					logger.Warn("failed sending pool heartbeat", "error", err)
				}
			})
			selector.AddReceive(stopCh, func(c workflow.ReceiveChannel, _ bool) {
				var signal struct{}
				c.Receive(ctx, &signal)
				stopRequested = true
			})

			selector.Select(ctx)
			if stopRequested {
				return
			}
		}
	})
}

func stopPoolHeartbeat(ctx workflow.Context, stopCh workflow.Channel, runData map[string]any) {
	if stopCh == nil {
		return
	}
	if runData != nil {
		if stopped, _ := runData["pool_heartbeat_stopped"].(bool); stopped {
			return
		}
	}
	// Avoid deadlock from double-send on the unbuffered stop channel.
	stopCh.Send(ctx, struct{}{})
	if runData != nil {
		runData["pool_heartbeat_stopped"] = true
	}
}
