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
			logger.Info(
				"Emulator already started, skipping start",
				"version",
				versionIdentifier,
				"serial",
				serial,
			)
			SetPayloadValue(&step.With.Payload, "emulator_serial", serial)
			continue
		}

		mobileAo := *input.ActivityOptions
		mobileAo.TaskQueue = workflows.MobileAutomationTaskQueue
		mobileCtx := workflow.WithActivityOptions(ctx, mobileAo)
		startResult := workflowengine.ActivityResult{}
		startEmuInput := workflowengine.ActivityInput{
			Payload: map[string]any{"version_id": versionIdentifier},
		}
		err = workflow.ExecuteActivity(mobileCtx, startEmuActivity.Name(), startEmuInput).
			Get(ctx, &startResult)
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
		SetPayloadValue(&step.With.Payload, "emulator_serial", serial)

		installInput := workflowengine.ActivityInput{
			Payload: map[string]any{"apk": apkPath, "emulator_serial": serial},
		}
		if err := workflow.ExecuteActivity(mobileCtx, installActivity.Name(), installInput).Get(ctx, nil); err != nil {
			return err
		}
		startRecordInput := workflowengine.ActivityInput{
			Payload: map[string]any{
				"emulator_serial": serial,
				"workflow_id":     workflow.GetInfo(ctx).WorkflowExecution.ID,
			},
		}
		var recordResult workflowengine.ActivityResult
		if err := workflow.ExecuteActivity(
			mobileCtx,
			recordActivity.Name(),
			startRecordInput,
		).Get(ctx, &recordResult); err != nil {
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
		startedEmulators[versionIdentifier] = map[string]any{
			"serial":     serial,
			"recording":  true,
			"video_path": videoPath,
		}
		// unlockInput := workflowengine.ActivityInput{Payload: map[string]any{"emulator_serial": serial}}
		// if err := workflow.ExecuteActivity(mobileCtx, unlockActivity.Name(), unlockInput).Get(ctx, nil); err != nil {
		// 	return err
		// }
	}
	SetRunDataValue(runData, "started_emulators", startedEmulators)

	return nil
}

func MobileAutomationCleanupHook(
	ctx workflow.Context,
	steps []StepDefinition,
	input workflowengine.WorkflowInput,
	runData map[string]any,
	output *map[string]any,
) error {
	logger := workflow.GetLogger(ctx)
	mobileAo := *input.ActivityOptions

	mobileAo.TaskQueue = workflows.MobileAutomationTaskQueue
	mobileCtx := workflow.WithActivityOptions(ctx, mobileAo)
	mobileServerURL := utils.GetEnvironmentVariable("MAESTRO_WORKER", "http://localhost:8050")
	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
			"missing or invalid app_url in workflow input config",
		)
	}

	stopActivity := activities.NewStopEmulatorActivity()
	stopRecordingActivity := activities.NewStopRecordingActivity()
	httpActivity := activities.NewHTTPActivity()

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
			errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("error decoding payload for step %s: %s", step.ID, err.Error()),
			)
		}

		if payload.EmulatorSerial == "" {
			logger.Error("MobileAutomationCleanupHook: no emulator serial found.", "step", step.ID)
			return workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
				fmt.Sprintf("missing or invalid emulator serial	 for step %s", step.ID),
			)
		}

		logger.Info(
			"MobileAutomationCleanupHook: stopping emulator",
			"emulator",
			payload.EmulatorSerial,
			"step",
			step.ID,
		)
		if _, ok := stoppedEmulators[payload.EmulatorSerial]; ok {
			logger.Info("Emulator already stopped", "emulator", payload.EmulatorSerial)
			continue
		}

		if startedEmulators, ok := runData["started_emulators"].(map[string]any); ok {
			if emuInfo, ok := startedEmulators[payload.VersionID].(map[string]any); ok {
				if recording, ok := emuInfo["recording"].(bool); ok && recording {
					stopRecordInput := workflowengine.ActivityInput{
						Payload: map[string]any{
							"adb_process_pid":    payload.RecordingAdbPid,
							"ffmpeg_process_pid": payload.RecordingFfmpegPid,
						},
					}
					if err := workflow.ExecuteActivity(
						mobileCtx, stopRecordingActivity.Name(), stopRecordInput,
					).Get(ctx, nil); err != nil {
						logger.Error("cleanup: stop recording failed", "error", err)
						cleanupErrs = append(cleanupErrs, err)
					}
					videoPath, ok := emuInfo["video_path"].(string)
					if !ok || videoPath == "" {
						errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
						appErr := workflowengine.NewAppError(
							errCode,
							fmt.Sprintf(
								"%s: missing or invalid video path for emulator %s",
								errCode.Description,
								payload.EmulatorSerial,
							),
						)
						cleanupErrs = append(cleanupErrs, appErr)
					} else {
						runIdentifier, ok := runData["run_identifier"].(string)
						if !ok || runIdentifier == "" {
							errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
							appErr := workflowengine.NewAppError(
								errCode,
								fmt.Sprintf(
									"%s: missing or invalid run identifier in run data",
									errCode.Description,
								),
							)
							cleanupErrs = append(cleanupErrs, appErr)
						} else {
							storeResultInput := workflowengine.ActivityInput{
								Payload: activities.HTTPActivityPayload{
									Method: http.MethodPost,
									URL:    utils.JoinURL(mobileServerURL, "store-pipeline-result"),
									Headers: map[string]string{
										"Content-Type": "application/json",
									},
									Body: map[string]any{
										"video_path":     videoPath,
										"run_identifier": runIdentifier,
										"instance_url":   appURL,
									},
									ExpectedStatus: 200,
								},
							}
							var storeResultResponse workflowengine.ActivityResult
							if err := workflow.ExecuteActivity(
								ctx, httpActivity.Name(), storeResultInput,
							).Get(ctx, &storeResultResponse); err != nil {
								logger.Error("cleanup: store result failed", "error", err)
								cleanupErrs = append(cleanupErrs, err)
							} else {
								resultURLSRaw, ok := storeResultResponse.Output.(map[string]any)["body"].(map[string]any)["result_urls"]
								resultURLS := workflowengine.AsSliceOfStrings(resultURLSRaw)
								if !ok || resultURLS == nil || len(resultURLS) == 0 {
									errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
									appErr := workflowengine.NewAppError(
										errCode,
										fmt.Sprintf(
											"%s: 'result_urls'", errCode.Description),
										storeResultResponse.Output,
									)
									cleanupErrs = append(cleanupErrs, appErr)
								}
								if *output == nil {
									*output = make(map[string]any)
								}

								existing, _ := (*output)["result_video_urls"].([]string)
								(*output)["result_video_urls"] = append(existing, resultURLS...)
							}
						}
					}
				}
			}
		}

		stopInput := workflowengine.ActivityInput{
			Payload: map[string]any{"emulator_serial": payload.EmulatorSerial},
		}
		if err := workflow.ExecuteActivity(mobileCtx, stopActivity.Name(), stopInput).Get(ctx, nil); err != nil {
			logger.Error(
				"MobileAutomationCleanupHook: error stopping emulator",
				payload.EmulatorSerial,
				"error",
				err,
			)
			return err
		}

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
