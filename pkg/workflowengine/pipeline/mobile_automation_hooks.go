// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"go.temporal.io/sdk/workflow"
)

const (
	searchAttrEmulatorSerial = "emulator_serial"
	searchAttrVersionID      = "version_id"
	searchAttrBootStatus     = "boot_status"
	bootStatusStarting       = "starting"
	bootStatusStarted        = "started"
	bootStatusBooted         = "booted"
	bootStatusRecording      = "recording"
	bootStatusStopped        = "stopped"
)

func MobileAutomationSetupHook(
	ctx workflow.Context,
	steps *[]StepDefinition,
	input workflowengine.WorkflowInput,
	runData *map[string]any,
) error {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)

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
	metrics := getEmulatorMetrics(runData)

	for i := range *steps {
		step := &(*steps)[i]

		if step.Use != "mobile-automation" {
			continue
		}

		SetConfigValue(&step.With.Config, "app_url", input.Config["app_url"])

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
		appURL, ok := input.Config["app_url"].(string)
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
		if err := workflow.ExecuteActivity(ctx, httpActivity.Name(), req).Get(ctx, &res); err != nil {
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
			bootStatus, ok := emuInfoMap["boot_status"].(string)
			if !ok || bootStatus == "" {
				bootStatus = bootStatusStarted
				updateEmulatorStatus(startedEmulators, versionIdentifier, map[string]any{
					"boot_status": bootStatus,
					"serial":      serial,
				})
			}
			logger.Info(
				"Emulator already started, skipping start",
				"version",
				versionIdentifier,
				"serial",
				serial,
				"boot_status",
				bootStatus,
			)
			upsertEmulatorSearchAttributes(ctx, versionIdentifier, serial, bootStatus)
			metrics["active_emulators"] = countActiveEmulators(startedEmulators)
			updateEmulatorGauges(ctx, metrics)
			SetPayloadValue(&step.With.Payload, "emulator_serial", serial)
			continue
		}

		mobileAo := *input.ActivityOptions
		mobileAo.TaskQueue = workflows.MobileAutomationTaskQueue
		mobileCtx := workflow.WithActivityOptions(ctx, mobileAo)
		startResult := workflowengine.ActivityResult{}
		metrics["pending_starts"]++
		updateEmulatorGauges(ctx, metrics)
		updateEmulatorStatus(startedEmulators, versionIdentifier, map[string]any{
			"boot_status": bootStatusStarting,
		})
		upsertEmulatorSearchAttributes(ctx, versionIdentifier, "", bootStatusStarting)
		startEmuInput := workflowengine.ActivityInput{
			Payload: map[string]any{"version_id": versionIdentifier},
		}
		err = workflow.ExecuteActivity(mobileCtx, startEmuActivity.Name(), startEmuInput).
			Get(ctx, &startResult)
		if err != nil {
			metrics["pending_starts"]--
			updateEmulatorGauges(ctx, metrics)
			incrementEmulatorCounter(ctx, metrics, "failed_starts")
			return err
		}
		metrics["pending_starts"]--
		updateEmulatorGauges(ctx, metrics)
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
		SetPayloadValue(&step.With.Payload, "emulator_serial", serial)
		updateEmulatorStatus(startedEmulators, versionIdentifier, map[string]any{
			"serial":      serial,
			"boot_status": bootStatusStarted,
		})
		upsertEmulatorSearchAttributes(ctx, versionIdentifier, serial, bootStatusStarted)
		SetRunDataValue(runData, "started_emulators", startedEmulators)
		metrics["active_emulators"] = countActiveEmulators(startedEmulators)
		updateEmulatorGauges(ctx, metrics)

		installInput := workflowengine.ActivityInput{
			Payload: map[string]any{"apk": apkPath, "emulator_serial": serial},
		}
		if err := workflow.ExecuteActivity(mobileCtx, installActivity.Name(), installInput).Get(mobileCtx, nil); err != nil {
			return err
		}
		updateEmulatorStatus(startedEmulators, versionIdentifier, map[string]any{
			"boot_status": bootStatusBooted,
		})
		upsertEmulatorSearchAttributes(ctx, versionIdentifier, serial, bootStatusBooted)
		startRecordInput := workflowengine.ActivityInput{
			Payload: map[string]any{
				"emulator_serial": serial,
				"workflow_id":     workflow.GetInfo(mobileCtx).WorkflowExecution.ID,
			},
		}
		var recordResult workflowengine.ActivityResult
		if err := workflow.ExecuteActivity(
			mobileCtx,
			recordActivity.Name(),
			startRecordInput,
		).Get(mobileCtx, &recordResult); err != nil {
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
		updateEmulatorStatus(startedEmulators, versionIdentifier, map[string]any{
			"serial":      serial,
			"recording":   true,
			"video_path":  videoPath,
			"boot_status": bootStatusRecording,
		})
		upsertEmulatorSearchAttributes(ctx, versionIdentifier, serial, bootStatusRecording)
		metrics["active_emulators"] = countActiveEmulators(startedEmulators)
		updateEmulatorGauges(ctx, metrics)
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
	input workflowengine.WorkflowInput,
	runData map[string]any,
	output *map[string]any,
) error {
	ctx, _ = workflow.NewDisconnectedContext(ctx)
	logger := workflow.GetLogger(ctx)
	mobileAo := *input.ActivityOptions

	mobileAo.TaskQueue = workflows.MobileAutomationTaskQueue
	mobileCtx := workflow.WithActivityOptions(ctx, mobileAo)
	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
			"missing or invalid app_url in workflow input config",
		)
	}
	metrics := getEmulatorMetrics(&runData)
	startedEmulators, _ := runData["started_emulators"].(map[string]any)

	stoppedEmulators := make(map[string]struct{})
	var cleanupErrs []error

	for _, step := range steps {
		if step.Use != "mobile-automation" {
			continue
		}

		payload, err := workflowengine.DecodePayload[workflows.MobileAutomationWorkflowPayload](
			step.With.Payload,
		)
		if err != nil {
			return workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
				"error decoding payload for step "+step.ID,
			)
		}

		if payload.EmulatorSerial == "" {
			return workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
				"missing emulator serial for step "+step.ID,
			)
		}

		if _, alreadyStopped := stoppedEmulators[payload.EmulatorSerial]; alreadyStopped {
			continue
		}

		cleanupRecording(
			mobileCtx,
			payload,
			runData,
			input,
			output,
			&cleanupErrs,
			metrics,
		)

		// Always stop emulator
		if err := workflow.ExecuteActivity(
			mobileCtx,
			activities.NewStopEmulatorActivity().Name(),
			workflowengine.ActivityInput{
				Payload: map[string]any{"emulator_serial": payload.EmulatorSerial},
			},
		).Get(ctx, nil); err != nil {
			logger.Error(
				"failed stopping emulator",
				"emulator",
				payload.EmulatorSerial,
				"error",
				err,
			)
			incrementEmulatorCounter(ctx, metrics, "cleanup_errors")
			return err // stopping emulator is fatal
		}
		updateEmulatorStatus(startedEmulators, payload.VersionID, map[string]any{
			"boot_status": bootStatusStopped,
			"recording":   false,
		})
		upsertEmulatorSearchAttributes(ctx, payload.VersionID, payload.EmulatorSerial, bootStatusStopped)
		metrics["active_emulators"] = countActiveEmulators(startedEmulators)
		updateEmulatorGauges(ctx, metrics)

		stoppedEmulators[payload.EmulatorSerial] = struct{}{}
	}

	if len(cleanupErrs) > 0 {
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		return workflowengine.NewAppError(
			errCode,
			"one or more errors occurred during mobile automation cleanup",
			cleanupErrs,
		)
	}

	return nil
}

func cleanupRecording(
	ctx workflow.Context,
	payload workflows.MobileAutomationWorkflowPayload,
	runData map[string]any,
	input workflowengine.WorkflowInput,
	output *map[string]any,
	cleanupErrs *[]error,
	metrics map[string]int,
) {
	logger := workflow.GetLogger(ctx)

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
		incrementEmulatorCounter(ctx, metrics, "cleanup_errors")
		return
	}

	stopRecordingActivity := activities.NewStopRecordingActivity()
	var stopResult workflowengine.ActivityResult

	if err := workflow.ExecuteActivity(
		ctx,
		stopRecordingActivity.Name(),
		workflowengine.ActivityInput{
			Payload: map[string]any{
				"video_path":         videoPath,
				"adb_process_pid":    payload.RecordingAdbPid,
				"ffmpeg_process_pid": payload.RecordingFfmpegPid,
				"logcat_process_pid": payload.RecordingLogcatPid,
			},
		},
	).Get(ctx, &stopResult); err != nil {
		logger.Error("cleanup: stop recording failed", "error", err)
		*cleanupErrs = append(*cleanupErrs, err)
		incrementEmulatorCounter(ctx, metrics, "cleanup_errors")
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
		incrementEmulatorCounter(ctx, metrics, "cleanup_errors")
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
		incrementEmulatorCounter(ctx, metrics, "cleanup_errors")
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
				URL:    utils.JoinURL(utils.GetEnvironmentVariable("MAESTRO_WORKER", ""), "store-pipeline-result"),
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]any{
					"video_path":         videoPath,
					"last_frame_path":    lastFramePath,
					"run_identifier":     runIdentifier,
					"version_identifier": payload.VersionID,
					"instance_url":       input.Config["app_url"],
				},
				ExpectedStatus: 200,
			},
		},
	).Get(ctx, &storeResult); err != nil {
		logger.Error("cleanup: store result failed", "error", err)
		*cleanupErrs = append(*cleanupErrs, err)
		incrementEmulatorCounter(ctx, metrics, "cleanup_errors")
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
		incrementEmulatorCounter(ctx, metrics, "cleanup_errors")
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
		incrementEmulatorCounter(ctx, metrics, "cleanup_errors")
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

func upsertEmulatorSearchAttributes(
	ctx workflow.Context,
	versionID string,
	serial string,
	bootStatus string,
) {
	if versionID == "" && serial == "" && bootStatus == "" {
		return
	}
	attrs := map[string]any{}
	if versionID != "" {
		attrs[searchAttrVersionID] = versionID
	}
	if serial != "" {
		attrs[searchAttrEmulatorSerial] = serial
	}
	if bootStatus != "" {
		attrs[searchAttrBootStatus] = bootStatus
	}
	if len(attrs) == 0 {
		return
	}
	if err := workflow.UpsertSearchAttributes(ctx, attrs); err != nil {
		workflow.GetLogger(ctx).Error(
			"failed to upsert search attributes",
			"error",
			err,
			"attributes",
			attrs,
		)
	}
}

func getEmulatorMetrics(runData *map[string]any) map[string]int {
	metrics, ok := (*runData)["emulator_metrics"].(map[string]int)
	if ok {
		return metrics
	}
	metrics = map[string]int{
		"active_emulators": 0,
		"pending_starts":   0,
		"failed_starts":    0,
		"cleanup_errors":   0,
	}
	(*runData)["emulator_metrics"] = metrics
	return metrics
}

func updateEmulatorGauges(ctx workflow.Context, metrics map[string]int) {
	metricsHandler := workflow.GetMetricsHandler(ctx)
	metricsHandler.Gauge("active_emulators").Update(float64(metrics["active_emulators"]))
	metricsHandler.Gauge("pending_starts").Update(float64(metrics["pending_starts"]))
}

func incrementEmulatorCounter(
	ctx workflow.Context,
	metrics map[string]int,
	metricName string,
) {
	metrics[metricName]++
	workflow.GetMetricsHandler(ctx).Counter(metricName).Inc(1)
}

func updateEmulatorStatus(
	startedEmulators map[string]any,
	versionID string,
	fields map[string]any,
) map[string]any {
	if startedEmulators == nil {
		return nil
	}
	entry, ok := startedEmulators[versionID].(map[string]any)
	if !ok {
		entry = map[string]any{
			"version_id": versionID,
		}
	}
	for key, value := range fields {
		entry[key] = value
	}
	startedEmulators[versionID] = entry
	return entry
}

func countActiveEmulators(startedEmulators map[string]any) int {
	activeCount := 0
	for _, entry := range startedEmulators {
		status, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		bootStatus, _ := status["boot_status"].(string)
		if bootStatus == bootStatusStopped {
			continue
		}
		activeCount++
	}
	return activeCount
}
