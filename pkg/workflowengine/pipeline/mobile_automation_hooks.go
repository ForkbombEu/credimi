// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

func MobileAutomationSetupHook(
	ctx workflow.Context,
	steps *[]StepDefinition,
	ao *workflow.ActivityOptions,
	config map[string]any,
	runData *map[string]any,
) error {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, *ao)

	httpActivity := activities.NewHTTPActivity()
	startEmuActivity := activities.NewStartEmulatorActivity()
	installActivity := activities.NewApkInstallActivity()
	recordActivity := activities.NewStartRecordingActivity()

	settedDevices := getOrCreateSettedDevices(runData)

	// Get global_runner_id from config if present
	globalRunnerID, _ := config["global_runner_id"].(string)

	// First pass: validate runner configuration
	hasMobileSteps := false
	allStepsHaveRunner := true
	anyStepHasRunner := false

	for i := range *steps {
		step := &(*steps)[i]
		if step.Use != "mobile-automation" {
			continue
		}
		hasMobileSteps = true

		// Try to decode the payload to check if runner_id is set
		runnerID, _ := step.With.Payload["runner_id"].(string)
		if runnerID == "" {
			allStepsHaveRunner = false
		} else {
			anyStepHasRunner = true
		}
	}

	// Validate: either all steps have runner_id OR global_runner_id is set
	if hasMobileSteps {
		if !allStepsHaveRunner && globalRunnerID == "" {
			errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
			return workflowengine.NewAppError(
				errCode,
				"mobile-automation steps require either a global_runner_id or runner_id for each step",
			)
		}

		// If all steps have runner_id and global_runner_id is also set, warn about unused global runner
		if globalRunnerID != "" && allStepsHaveRunner {
			logger.Warn(
				"global_runner_id is set but all mobile-automation steps have their own runner_id - global_runner_id will be ignored",
			)
		}

		// If mixing global and specific runners, that's not allowed
		if globalRunnerID != "" && anyStepHasRunner && !allStepsHaveRunner {
			errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
			return workflowengine.NewAppError(
				errCode,
				"cannot mix global_runner_id with step-specific runner_id: use one or the other",
			)
		}
	}

	// Second pass: process each step
	for i := range *steps {
		step := &(*steps)[i]

		if step.Use != "mobile-automation" {
			continue
		}

		// Use global_runner_id if step doesn't have runner_id
		if runnerID, _ := step.With.Payload["runner_id"].(string); runnerID == "" && globalRunnerID != "" {
			if step.With.Payload == nil {
				step.With.Payload = make(map[string]any)
			}
			step.With.Payload["runner_id"] = globalRunnerID
		}

		if err := processStep(
			ctx,
			step,
			config,
			ao,
			settedDevices,
			httpActivity,
			startEmuActivity,
			installActivity,
			logger,
		); err != nil {
			return err
		}
	}

	if err := startRecordingForDevices(
		ctx,
		settedDevices,
		ao,
		recordActivity,
	); err != nil {
		return err
	}

	SetRunDataValue(runData, "setted_devices", settedDevices)

	return nil
}

func getOrCreateSettedDevices(runData *map[string]any) map[string]any {
	settedDevices := make(map[string]any)
	if alreadyStartedDevices, ok := (*runData)["setted_devices"].(map[string]any); ok {
		settedDevices = alreadyStartedDevices
	}
	return settedDevices
}

func processStep(
	ctx workflow.Context,
	step *StepDefinition,
	config map[string]any,
	ao *workflow.ActivityOptions,
	settedDevices map[string]any,
	httpActivity *activities.HTTPActivity,
	startEmuActivity *activities.StartEmulatorActivity,
	installActivity *activities.ApkInstallActivity,
	logger log.Logger,
) error {
	SetConfigValue(&step.With.Config, "app_url", config["app_url"])
	logger.Info("MobileAutomationSetupHook: processing step", "id", step.ID)

	payload, err := decodeAndValidatePayload(step)
	if err != nil {
		return err
	}

	taskqueue := fmt.Sprintf("%s-%s", payload.RunnerID, "TaskQueue")
	SetConfigValue(&step.With.Config, "taskqueue", taskqueue)
	mobileAo := *ao
	mobileAo.TaskQueue = taskqueue
	mobileCtx := workflow.WithActivityOptions(ctx, mobileAo)

	appURL, ok := config["app_url"].(string)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("missing or invalid app_url for step %s", step.ID),
		)
	}

	deviceMap, err := getOrCreateDeviceMap(
		ctx,
		mobileCtx,
		payload,
		settedDevices,
		appURL,
		step.ID,
		httpActivity,
		startEmuActivity,
	)
	if err != nil {
		return err
	}

	serial, ok := deviceMap["serial"].(string)
	if !ok {
		serial = ""
	}
	SetPayloadValue(&step.With.Payload, "serial", serial)

	runnerURL, ok := deviceMap["runner_url"].(string)
	if !ok || runnerURL == "" {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("missing or invalid runner_url for step %s", step.ID),
			deviceMap,
		)
	}

	if err := fetchAndInstallAPK(
		ctx,
		mobileCtx,
		step,
		payload,
		deviceMap,
		appURL,
		runnerURL,
		serial,
		httpActivity,
		installActivity,
	); err != nil {
		return err
	}

	return nil
}

func decodeAndValidatePayload(
	step *StepDefinition,
) (*workflows.MobileAutomationWorkflowPipelinePayload, error) {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	payload, err := workflowengine.DecodePayload[workflows.MobileAutomationWorkflowPipelinePayload](
		step.With.Payload,
	)
	if err != nil {
		return nil, workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("error decoding payload for step %s: %s", step.ID, err.Error()),
		)
	}

	// If action_code is present, version_id is REQUIRED
	if payload.ActionCode != "" {
		if payload.VersionID == "" {
			return nil, workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("missing or invalid version_id for step %s", step.ID))
		}
	}
	// If action_code is NOT present -> action_id is REQUIRED
	if payload.ActionCode == "" {
		if payload.ActionID == "" {
			return nil, workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("missing or invalid action_id for step %s", step.ID),
			)
		}
	}

	return &payload, nil
}

func getOrCreateDeviceMap(
	ctx workflow.Context,
	mobileCtx workflow.Context,
	payload *workflows.MobileAutomationWorkflowPipelinePayload,
	settedDevices map[string]any,
	appURL string,
	stepID string,
	httpActivity *activities.HTTPActivity,
	startEmuActivity *activities.StartEmulatorActivity,
) (map[string]any, error) {
	deviceInfo, exists := settedDevices[payload.RunnerID]
	var deviceMap map[string]any
	if exists {
		deviceMap = deviceInfo.(map[string]any)
		return deviceMap, nil
	}

	deviceMap = map[string]any{
		"installed": make(map[string]string),
		"recording": false,
	}
	settedDevices[payload.RunnerID] = deviceMap

	if err := setupNewDevice(
		ctx,
		mobileCtx,
		payload,
		deviceMap,
		appURL,
		stepID,
		httpActivity,
		startEmuActivity,
	); err != nil {
		return nil, err
	}

	return deviceMap, nil
}

func setupNewDevice(
	ctx workflow.Context,
	mobileCtx workflow.Context,
	payload *workflows.MobileAutomationWorkflowPipelinePayload,
	deviceMap map[string]any,
	appURL string,
	stepID string,
	httpActivity *activities.HTTPActivity,
	startEmuActivity *activities.StartEmulatorActivity,
) error {
	runnerURL, serial, err := fetchRunnerInfo(
		ctx,
		payload,
		appURL,
		stepID,
		httpActivity,
	)
	if err != nil {
		return err
	}

	if serial == "" {
		cloneName, newSerial, err := startEmulator(
			ctx,
			mobileCtx,
			payload,
			stepID,
			startEmuActivity,
		)
		if err != nil {
			return err
		}
		serial = newSerial
		deviceMap["clone_name"] = cloneName
	}

	deviceMap["runner_url"] = runnerURL
	deviceMap["serial"] = serial

	return nil
}

func fetchRunnerInfo(
	ctx workflow.Context,
	payload *workflows.MobileAutomationWorkflowPipelinePayload,
	appURL string,
	stepID string,
	httpActivity *activities.HTTPActivity,
) (string, string, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]

	runnerReq := workflowengine.ActivityInput{
		Payload: map[string]any{
			"method":          http.MethodGet,
			"url":             utils.JoinURL(appURL, "api", "mobile-runner"),
			"expected_status": 200,
			"query_params": map[string]string{
				"runner_identifier": payload.RunnerID,
			},
		},
	}

	var runnerRes workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(ctx, httpActivity.Name(), runnerReq).
		Get(ctx, &runnerRes); err != nil {
		return "", "", err
	}

	body, ok := runnerRes.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		return "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("invalid HTTP response format for step %s", stepID),
			runnerRes.Output,
		)
	}

	runnerURL, ok := body["runner_url"].(string)
	if !ok || runnerURL == "" {
		return "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("missing or invalid runner_url for step %s", stepID),
			body,
		)
	}

	serial, ok := body["serial"].(string)
	if !ok {
		return "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("invalid device serial for step %s", stepID),
			body,
		)
	}

	return runnerURL, serial, nil
}

func startEmulator(
	ctx workflow.Context,
	mobileCtx workflow.Context,
	payload *workflows.MobileAutomationWorkflowPipelinePayload,
	stepID string,
	startEmuActivity *activities.StartEmulatorActivity,
) (string, string, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]

	startResult := workflowengine.ActivityResult{}
	startInput := workflowengine.ActivityInput{
		Payload: map[string]any{"device_name": payload.RunnerID},
	}
	err := workflow.ExecuteActivity(mobileCtx, startEmuActivity.Name(), startInput).
		Get(ctx, &startResult)
	if err != nil {
		return "", "", err
	}

	serial, ok := startResult.Output.(map[string]any)["serial"].(string)
	if !ok {
		return "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing serial in response for step %s",
				errCode.Description,
				stepID,
			),
			startResult.Output,
		)
	}

	cloneName, ok := startResult.Output.(map[string]any)["clone_name"].(string)
	if !ok {
		return "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing clone_name in response for step %s",
				errCode.Description,
				stepID,
			),
			startResult.Output,
		)
	}

	return cloneName, serial, nil
}

func fetchAndInstallAPK(
	ctx workflow.Context,
	mobileCtx workflow.Context,
	step *StepDefinition,
	payload *workflows.MobileAutomationWorkflowPipelinePayload,
	deviceMap map[string]any,
	appURL string,
	runnerURL string,
	serial string,
	httpActivity *activities.HTTPActivity,
	installActivity *activities.ApkInstallActivity,
) error {
	req := workflowengine.ActivityInput{
		Payload: map[string]any{
			"method": http.MethodPost,
			"url":    utils.JoinURL(runnerURL, "fetch-apk-and-action"),
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

	apkPath, versionIdentifier, actionCode, err := parseAPKResponse(
		res,
		payload,
		step,
	)
	if err != nil {
		return err
	}

	if err := installAPKIfNeeded(
		mobileCtx,
		deviceMap,
		apkPath,
		versionIdentifier,
		serial,
		step.ID,
		installActivity,
	); err != nil {
		return err
	}

	if payload.ActionCode == "" {
		SetPayloadValue(&step.With.Payload, "action_code", actionCode)
		SetPayloadValue(&step.With.Payload, "stored_action_code", true)
	}

	return nil
}

func parseAPKResponse(
	res workflowengine.ActivityResult,
	payload *workflows.MobileAutomationWorkflowPipelinePayload,
	step *StepDefinition,
) (string, string, string, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]

	body, ok := res.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		return "", "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("invalid HTTP response format for step %s", step.ID),
			res.Output,
		)
	}

	apkPath, ok := body["apk_path"].(string)
	if !ok {
		return "", "", "", workflowengine.NewAppError(
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
		return "", "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing version_id in response for step %s",
				errCode.Description,
				step.ID,
			),
			body,
		)
	}

	actionCode := payload.ActionCode
	if actionCode == "" {
		actionCode, ok = body["code"].(string)
		if !ok || actionCode == "" {
			return "", "", "", workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing action_code in response for step %s",
					errCode.Description,
					step.ID,
				),
				body,
			)
		}
	}

	return apkPath, versionIdentifier, actionCode, nil
}

func installAPKIfNeeded(
	mobileCtx workflow.Context,
	deviceMap map[string]any,
	apkPath string,
	versionIdentifier string,
	serial string,
	stepID string,
	installActivity *activities.ApkInstallActivity,
) error {
	installed, ok := deviceMap["installed"].(map[string]string)
	if !ok {
		installed = make(map[string]string)
	}

	if _, ok := installed[versionIdentifier]; !ok {
		installInput := workflowengine.ActivityInput{
			Payload: map[string]any{"apk": apkPath, "serial": serial},
		}
		installOutput := workflowengine.ActivityResult{}
		if err := workflow.ExecuteActivity(mobileCtx, installActivity.Name(), installInput).
			Get(mobileCtx, &installOutput); err != nil {
			return err
		}

		packageID, ok := installOutput.Output.(map[string]any)["package_id"].(string)
		if !ok {
			errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing package_id in response for step %s",
					errCode.Description,
					stepID,
				),
				installOutput.Output,
			)
		}
		installed[versionIdentifier] = packageID
		deviceMap["installed"] = installed
	}

	return nil
}

func startRecordingForDevices(
	ctx workflow.Context,
	settedDevices map[string]any,
	ao *workflow.ActivityOptions,
	recordActivity *activities.StartRecordingActivity,
) error {
	for runnerID, dev := range settedDevices {
		deviceMap := dev.(map[string]any)
		recording := deviceMap["recording"].(bool)
		if recording {
			continue
		}

		if err := startRecordingForDevice(
			ctx,
			runnerID,
			deviceMap,
			ao,
			recordActivity,
		); err != nil {
			return err
		}
	}
	return nil
}

func startRecordingForDevice(
	ctx workflow.Context,
	runnerID string,
	deviceMap map[string]any,
	ao *workflow.ActivityOptions,
	recordActivity *activities.StartRecordingActivity,
) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]

	serial, ok := deviceMap["serial"].(string)
	if !ok || serial == "" {
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("missing serial for device %s", runnerID),
		)
	}

	mobileAO := *ao
	mobileAO.TaskQueue = fmt.Sprintf("%s-TaskQueue", runnerID)
	mobileCtx := workflow.WithActivityOptions(ctx, mobileAO)

	startRecordInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"serial":      serial,
			"workflow_id": workflow.GetInfo(mobileCtx).WorkflowExecution.ID,
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

	if err := extractAndStoreRecordingInfo(
		recordResult,
		deviceMap,
		runnerID,
	); err != nil {
		return err
	}

	return nil
}

func extractAndStoreRecordingInfo(
	recordResult workflowengine.ActivityResult,
	deviceMap map[string]any,
	runnerID string,
) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]

	adbPID, ok := recordResult.Output.(map[string]any)["adb_process_pid"].(float64)
	if !ok {
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing adb_process in start record video response for device %s",
				errCode.Description,
				runnerID,
			),
			recordResult.Output,
		)
	}

	ffmpegPID, ok := recordResult.Output.(map[string]any)["ffmpeg_process_pid"].(float64)
	if !ok {
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing ffmpeg_process in start record video response for device %s",
				errCode.Description,
				runnerID,
			),
			recordResult.Output,
		)
	}

	logcatPID, ok := recordResult.Output.(map[string]any)["logcat_process_pid"].(float64)
	if !ok {
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing logcat_process in start record video response for device %s",
				errCode.Description,
				runnerID,
			),
			recordResult.Output,
		)
	}

	videoPath, ok := recordResult.Output.(map[string]any)["video_path"].(string)
	if !ok {
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing video_path in start record video response for device %s",
				errCode.Description,
				runnerID,
			),
			recordResult.Output,
		)
	}

	deviceMap["recording_adb_pid"] = int(adbPID)
	deviceMap["recording_ffmpeg_pid"] = int(ffmpegPID)
	deviceMap["recording_logcat_pid"] = int(logcatPID)
	deviceMap["recording"] = true
	deviceMap["video_path"] = videoPath

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
	mobileAo := *ao

	appURL, ok := config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
			"missing or invalid app_url in workflow input config",
		)
	}

	var cleanupErrs []error

	devices, ok := runData["setted_devices"].(map[string]any)
	if !ok {
		return nil
	}

	runIdentifier, ok := runData["run_identifier"].(string)
	if !ok || runIdentifier == "" {
		cleanupErrs = append(cleanupErrs,
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
				"missing run_identifier in run data",
			),
		)
	}

	for runnerID, raw := range devices {
		if err := cleanupDevice(
			ctx,
			runnerID,
			raw,
			&mobileAo,
			runIdentifier,
			appURL,
			output,
			&cleanupErrs,
			logger,
		); err != nil {
			return err
		}
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

func cleanupDevice(
	ctx workflow.Context,
	runnerID string,
	raw any,
	mobileAo *workflow.ActivityOptions,
	runIdentifier string,
	appURL string,
	output *map[string]any,
	cleanupErrs *[]error,
	logger log.Logger,
) error {
	deviceMap, err := parseDeviceMap(runnerID, raw)
	if err != nil {
		*cleanupErrs = append(*cleanupErrs, err)
	}

	serial, cloneName, packages, err := extractDeviceInfo(runnerID, deviceMap)
	if err != nil {
		*cleanupErrs = append(*cleanupErrs, err)
	}

	mobileAo.TaskQueue = fmt.Sprintf("%s-%s", runnerID, "TaskQueue")
	mobileCtx := workflow.WithActivityOptions(ctx, *mobileAo)

	cleanupRecording(
		mobileCtx,
		runnerID,
		deviceMap,
		runIdentifier,
		output,
		cleanupErrs,
		appURL,
	)

	cleanupPayload := map[string]any{
		"serial":       serial,
		"clone_name":   cloneName,
		"apk_packages": packages,
	}

	if err := workflow.ExecuteActivity(
		mobileCtx,
		activities.NewCleanupDeviceActivity().Name(),
		workflowengine.ActivityInput{
			Payload: cleanupPayload,
		},
	).Get(ctx, nil); err != nil {
		logger.Error(
			"failed ",
			"mobile device cleanup",
			runnerID,
			"error",
			err,
		)
		return err
	}

	deviceMap["cleaned"] = true

	return nil
}

func parseDeviceMap(
	runnerID string,
	raw any,
) (map[string]any, error) {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	deviceMap, ok := raw.(map[string]any)
	if !ok {
		return nil, workflowengine.NewAppError(
			errCode,
			"error decoding payload for device "+runnerID,
			raw,
		)
	}
	return deviceMap, nil
}

func extractDeviceInfo(
	runnerID string,
	deviceMap map[string]any,
) (string, string, []string, error) {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]

	serial, ok := deviceMap["serial"].(string)
	if !ok || serial == "" {
		return "", "", nil, workflowengine.NewAppError(
			errCode,
			"error decoding payload for device "+runnerID,
			deviceMap,
		)
	}

	cloneName, _ := deviceMap["clone_name"].(string)

	var packages []string

	if installed, ok := deviceMap["installed"].(map[string]string); ok {
		for _, pkg := range installed {
			if pkg != "" {
				packages = append(packages, pkg)
			}
		}
	} else {
		return "", "", nil, workflowengine.NewAppError(
			errCode,
			"error decoding payload for device "+runnerID,
			deviceMap,
		)
	}

	return serial, cloneName, packages, nil
}

func cleanupRecording(
	ctx workflow.Context,
	runnerID string,
	deviceInfo map[string]any,
	runID string,
	output *map[string]any,
	cleanupErrs *[]error,
	appURL string,
) {
	logger := workflow.GetLogger(ctx)

	runner_url, ok := deviceInfo["runner_url"].(string)
	if !ok || runner_url == "" {
		*cleanupErrs = append(*cleanupErrs,
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
				"missing runner_url for device "+runnerID,
			),
		)
		return
	}

	recording, ok := deviceInfo["recording"].(bool)
	if !ok || !recording {
		return
	}

	recordingInfo, err := extractRecordingInfo(runnerID, deviceInfo, cleanupErrs)
	if err != nil {
		return // Error already added to cleanupErrs
	}

	lastFramePath, err := stopRecording(
		ctx,
		recordingInfo,
		cleanupErrs,
		logger,
	)
	if err != nil {
		return // Error already added to cleanupErrs
	}

	if err := storeRecordingResults(
		ctx,
		runner_url,
		recordingInfo.videoPath,
		lastFramePath,
		runID,
		runnerID,
		appURL,
		output,
		cleanupErrs,
		logger,
	); err != nil {
		return // Error already added to cleanupErrs
	}
}

type recordingInfo struct {
	videoPath string
	adbPid    int
	ffmpegPid int
	logcatPid int
}

func extractRecordingInfo(
	runnerID string,
	deviceInfo map[string]any,
	cleanupErrs *[]error,
) (*recordingInfo, error) {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]

	videoPath, ok := deviceInfo["video_path"].(string)
	if !ok || videoPath == "" {
		*cleanupErrs = append(*cleanupErrs,
			workflowengine.NewAppError(
				errCode,
				"missing video_path for device "+runnerID,
			),
		)
		return nil, workflowengine.NewAppError(
			errCode,
			"missing video_path for device "+runnerID,
		)
	}

	recordingAdbPid, ok := deviceInfo["recording_adb_pid"].(int)
	if !ok || recordingAdbPid == 0 {
		*cleanupErrs = append(*cleanupErrs,
			workflowengine.NewAppError(
				errCode,
				"missing recording_adb_pid for device "+runnerID,
			),
		)
		return nil, workflowengine.NewAppError(
			errCode,
			"missing recording_adb_pid for device "+runnerID,
		)
	}

	recordingFfmpegPid, ok := deviceInfo["recording_ffmpeg_pid"].(int)
	if !ok || recordingFfmpegPid == 0 {
		*cleanupErrs = append(*cleanupErrs,
			workflowengine.NewAppError(
				errCode,
				"missing recording_ffmpeg_pid for device "+runnerID,
			),
		)
		return nil, workflowengine.NewAppError(
			errCode,
			"missing recording_ffmpeg_pid for device "+runnerID,
		)
	}

	recordingLogcatPid, ok := deviceInfo["recording_logcat_pid"].(int)
	if !ok || recordingLogcatPid == 0 {
		*cleanupErrs = append(*cleanupErrs,
			workflowengine.NewAppError(
				errCode,
				"missing recording_logcat_pid for device "+runnerID,
			),
		)
		return nil, workflowengine.NewAppError(
			errCode,
			"missing recording_logcat_pid for device "+runnerID,
		)
	}

	return &recordingInfo{
		videoPath: videoPath,
		adbPid:    recordingAdbPid,
		ffmpegPid: recordingFfmpegPid,
		logcatPid: recordingLogcatPid,
	}, nil
}

func stopRecording(
	ctx workflow.Context,
	info *recordingInfo,
	cleanupErrs *[]error,
	logger log.Logger,
) (string, error) {
	stopRecordingActivity := activities.NewStopRecordingActivity()
	var stopResult workflowengine.ActivityResult

	if err := workflow.ExecuteActivity(
		ctx,
		stopRecordingActivity.Name(),
		workflowengine.ActivityInput{
			Payload: map[string]any{
				"video_path":         info.videoPath,
				"adb_process_pid":    info.adbPid,
				"ffmpeg_process_pid": info.ffmpegPid,
				"logcat_process_pid": info.logcatPid,
			},
		},
	).Get(ctx, &stopResult); err != nil {
		logger.Error("cleanup: stop recording failed", "error", err)
		*cleanupErrs = append(*cleanupErrs, err)
		return "", err
	}

	lastFramePath, ok := stopResult.Output.(map[string]any)["last_frame_path"].(string)
	if !ok || lastFramePath == "" {
		err := workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
			"missing last_frame_path in stop recording result",
			stopResult.Output,
		)
		*cleanupErrs = append(*cleanupErrs, err)
		return "", err
	}

	return lastFramePath, nil
}

func storeRecordingResults(
	ctx workflow.Context,
	runnerURL string,
	videoPath string,
	lastFramePath string,
	runID string,
	runnerID string,
	appURL string,
	output *map[string]any,
	cleanupErrs *[]error,
	logger log.Logger,
) error {
	httpActivity := activities.NewHTTPActivity()
	var storeResult workflowengine.ActivityResult

	if err := workflow.ExecuteActivity(
		ctx,
		httpActivity.Name(),
		workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method: http.MethodPost,
				URL: utils.JoinURL(
					runnerURL,
					"store-pipeline-result",
				),
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]any{
					"video_path":        videoPath,
					"last_frame_path":   lastFramePath,
					"run_identifier":    runID,
					"runner_identifier": runnerID,
					"instance_url":      appURL,
				},
				ExpectedStatus: 200,
			},
		},
	).Get(ctx, &storeResult); err != nil {
		logger.Error("cleanup: store result failed", "error", err)
		*cleanupErrs = append(*cleanupErrs, err)
		return err
	}

	if err := extractAndStoreURLs(storeResult, output, cleanupErrs); err != nil {
		return err
	}

	return nil
}

func extractAndStoreURLs(
	storeResult workflowengine.ActivityResult,
	output *map[string]any,
	cleanupErrs *[]error,
) error {
	body, ok := storeResult.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		err := workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
			"missing body in store result",
			storeResult.Output,
		)
		*cleanupErrs = append(*cleanupErrs, err)
		return err
	}

	resultURLs := workflowengine.AsSliceOfStrings(body["result_urls"])
	frameURLs := workflowengine.AsSliceOfStrings(body["screenshot_urls"])

	if len(resultURLs) == 0 || len(frameURLs) == 0 {
		err := workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
			"missing result or screenshot URLs",
			storeResult.Output,
		)
		*cleanupErrs = append(*cleanupErrs, err)
		return err
	}

	if *output == nil {
		*output = make(map[string]any)
	}

	(*output)["result_video_urls"] =
		append((*output)["result_video_urls"].([]string), resultURLs...)
	(*output)["screenshot_urls"] =
		append((*output)["screenshot_urls"].([]string), frameURLs...)

	return nil
}
