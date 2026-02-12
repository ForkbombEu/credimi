// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

const (
	mobileAutomationStepUse                = "mobile-automation"
	mobileRunnerSemaphoreTicketIDConfigKey = "mobile_runner_semaphore_ticket_id"
)

type processStepInput struct {
	ctx              workflow.Context
	step             *StepDefinition
	config           map[string]any
	ao               *workflow.ActivityOptions
	settedDevices    map[string]any
	runData          *map[string]any
	httpActivity     *activities.HTTPActivity
	startEmuActivity *activities.StartEmulatorActivity
	installActivity  *activities.ApkInstallActivity
	logger           log.Logger
	globalRunnerID   string
}

type fetchAndInstallAPKInput struct {
	ctx             workflow.Context
	mobileCtx       workflow.Context
	step            *StepDefinition
	payload         *workflows.MobileAutomationWorkflowPipelinePayload
	deviceMap       map[string]any
	appURL          string
	runnerURL       string
	serial          string
	httpActivity    *activities.HTTPActivity
	installActivity *activities.ApkInstallActivity
}

type getOrCreateDeviceMapInput struct {
	ctx              workflow.Context
	mobileCtx        workflow.Context
	payload          *workflows.MobileAutomationWorkflowPipelinePayload
	settedDevices    map[string]any
	appURL           string
	stepID           string
	httpActivity     *activities.HTTPActivity
	startEmuActivity *activities.StartEmulatorActivity
}

type setupNewDeviceInput struct {
	ctx              workflow.Context
	mobileCtx        workflow.Context
	payload          *workflows.MobileAutomationWorkflowPipelinePayload
	deviceMap        map[string]any
	appURL           string
	stepID           string
	httpActivity     *activities.HTTPActivity
	startEmuActivity *activities.StartEmulatorActivity
}

type fetchRunnerInfoInput struct {
	ctx          workflow.Context
	payload      *workflows.MobileAutomationWorkflowPipelinePayload
	appURL       string
	stepID       string
	httpActivity *activities.HTTPActivity
}

type startEmulatorInput struct {
	ctx              workflow.Context
	mobileCtx        workflow.Context
	payload          *workflows.MobileAutomationWorkflowPipelinePayload
	stepID           string
	startEmuActivity *activities.StartEmulatorActivity
}

type installAPKIfNeededInput struct {
	mobileCtx       workflow.Context
	deviceMap       map[string]any
	apkPath         string
	versionID       string
	serial          string
	stepID          string
	installActivity *activities.ApkInstallActivity
}

type startRecordingForDevicesInput struct {
	ctx            workflow.Context
	settedDevices  map[string]any
	ao             *workflow.ActivityOptions
	recordActivity *activities.StartRecordingActivity
}

type startRecordingForDeviceInput struct {
	ctx            workflow.Context
	runnerID       string
	deviceMap      map[string]any
	ao             *workflow.ActivityOptions
	recordActivity *activities.StartRecordingActivity
}

type cleanupDeviceInput struct {
	ctx           workflow.Context
	runnerID      string
	raw           any
	mobileAo      *workflow.ActivityOptions
	runIdentifier string
	appURL        string
	output        *map[string]any
	cleanupErrs   *[]error
	logger        log.Logger
}

type cleanupRecordingInput struct {
	ctx         workflow.Context
	runnerID    string
	deviceInfo  map[string]any
	runID       string
	output      *map[string]any
	cleanupErrs *[]error
	appURL      string
}

type storeRecordingResultsInput struct {
	ctx        workflow.Context
	runnerURL  string
	videoPath  string
	lastFrame  string
	logcatPath string
	runID      string
	runnerID   string
	appURL     string
	output     *map[string]any
	logger     log.Logger
}

func MobileAutomationSetupHook(
	ctx workflow.Context,
	steps *[]StepDefinition,
	ao *workflow.ActivityOptions,
	config map[string]any,
	runData *map[string]any,
) error {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, *ao)

	// Validate runner_id configuration
	globalRunnerID, _ := config["global_runner_id"].(string)
	if err := validateRunnerIDConfiguration(steps, globalRunnerID); err != nil {
		return err
	}
	semaphoreManaged := isSemaphoreManagedRun(config)

	httpActivity := activities.NewHTTPActivity()
	startEmuActivity := activities.NewStartEmulatorActivity()
	installActivity := activities.NewApkInstallActivity()
	recordActivity := activities.NewStartRecordingActivity()

	runnerIDs, err := collectMobileRunnerIDs(*steps, globalRunnerID)
	if err != nil {
		return err
	}
	if len(runnerIDs) > 0 && !semaphoreManaged {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
		return workflowengine.NewAppError(
			errCode,
			"mobile-runner pipelines must be started via queue/semaphore",
		)
	}

	settedDevices := getOrCreateSettedDevices(runData)

	for i := range *steps {
		step := &(*steps)[i]

		if step.Use != mobileAutomationStepUse {
			continue
		}

		if err := processStep(processStepInput{
			ctx:              ctx,
			step:             step,
			config:           config,
			ao:               ao,
			settedDevices:    settedDevices,
			runData:          runData,
			httpActivity:     httpActivity,
			startEmuActivity: startEmuActivity,
			installActivity:  installActivity,
			logger:           logger,
			globalRunnerID:   globalRunnerID,
		}); err != nil {
			return err
		}
	}

	if err := startRecordingForDevices(startRecordingForDevicesInput{
		ctx:            ctx,
		settedDevices:  settedDevices,
		ao:             ao,
		recordActivity: recordActivity,
	}); err != nil {
		return err
	}

	SetRunDataValue(runData, "setted_devices", settedDevices)

	return nil
}

// validateRunnerIDConfiguration checks that either:
// - all mobile-automation steps have a defined runner_id, OR
// - there is a global_runner_id set
func validateRunnerIDConfiguration(steps *[]StepDefinition, globalRunnerID string) error {
	var mobileAutomationSteps []*StepDefinition
	for i := range *steps {
		if (*steps)[i].Use == mobileAutomationStepUse {
			mobileAutomationSteps = append(mobileAutomationSteps, &(*steps)[i])
		}
	}

	// If there are no mobile-automation steps, no validation needed
	if len(mobileAutomationSteps) == 0 {
		return nil
	}

	// Check if all mobile-automation steps have runner_id
	allStepsHaveRunnerID := true
	for _, step := range mobileAutomationSteps {
		if runnerID, ok := step.With.Payload["runner_id"].(string); !ok || runnerID == "" {
			allStepsHaveRunnerID = false
			break
		}
	}

	// Valid if either all steps have runner_id OR there's a global_runner_id
	if !allStepsHaveRunnerID && globalRunnerID == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
		return workflowengine.NewAppError(
			errCode,
			"mobile-automation steps require either a runner_id or a global_runner_id in the pipeline configuration",
		)
	}

	return nil
}

func getOrCreateSettedDevices(runData *map[string]any) map[string]any {
	settedDevices := make(map[string]any)
	if alreadyStartedDevices, ok := (*runData)["setted_devices"].(map[string]any); ok {
		settedDevices = alreadyStartedDevices
	}
	return settedDevices
}

func collectMobileRunnerIDs(steps []StepDefinition, globalID string) ([]string, error) {
	uniqueRunnerIDs := make(map[string]struct{})

	if globalID != "" {
		uniqueRunnerIDs[globalID] = struct{}{}
	}
	for i := range steps {
		step := &steps[i]
		if step.Use != mobileAutomationStepUse {
			continue
		}

		payload, err := decodeAndValidatePayload(step)
		if err != nil {
			return nil, err
		}
		if payload.RunnerID == "" {
			continue
		}
		uniqueRunnerIDs[payload.RunnerID] = struct{}{}
	}

	if len(uniqueRunnerIDs) == 0 {
		return nil, nil
	}

	runnerIDs := make([]string, 0, len(uniqueRunnerIDs))
	for runnerID := range uniqueRunnerIDs {
		runnerIDs = append(runnerIDs, runnerID)
	}
	sort.Strings(runnerIDs)

	return runnerIDs, nil
}

func hasRunnerPermit(runData *map[string]any, runnerID string) bool {
	if runData == nil || *runData == nil {
		return false
	}

	rawPermits, ok := (*runData)["mobile_runner_permits"]
	if !ok {
		return false
	}

	switch permits := rawPermits.(type) {
	case map[string]workflows.MobileRunnerSemaphorePermit:
		_, ok := permits[runnerID]
		return ok
	case map[string]any:
		permit, ok := permits[runnerID]
		if !ok {
			return false
		}
		_, err := workflowengine.DecodePayload[workflows.MobileRunnerSemaphorePermit](permit)
		return err == nil
	default:
		return false
	}
}

func getRunnerPermits(runData map[string]any) map[string]workflows.MobileRunnerSemaphorePermit {
	rawPermits, ok := runData["mobile_runner_permits"]
	if !ok {
		return nil
	}

	switch permits := rawPermits.(type) {
	case map[string]workflows.MobileRunnerSemaphorePermit:
		return permits
	case map[string]any:
		decoded := make(map[string]workflows.MobileRunnerSemaphorePermit, len(permits))
		for runnerID, rawPermit := range permits {
			permit, err := workflowengine.DecodePayload[workflows.MobileRunnerSemaphorePermit](rawPermit)
			if err != nil {
				continue
			}
			decoded[runnerID] = permit
		}
		return decoded
	default:
		return nil
	}
}

func releaseRunnerPermits(
	ctx workflow.Context,
	permits map[string]workflows.MobileRunnerSemaphorePermit,
	cleanupErrs *[]error,
) {
	if len(permits) == 0 {
		return
	}

	releaseActivity := activities.NewReleaseMobileRunnerPermitActivity()
	runnerIDs := make([]string, 0, len(permits))
	for runnerID := range permits {
		runnerIDs = append(runnerIDs, runnerID)
	}
	sort.Strings(runnerIDs)

	for _, runnerID := range runnerIDs {
		permit := permits[runnerID]
		req := workflowengine.ActivityInput{Payload: permit}
		if err := workflow.ExecuteActivity(ctx, releaseActivity.Name(), req).Get(ctx, nil); err != nil {
			*cleanupErrs = append(*cleanupErrs, err)
		}
	}
}

func processStep(
	input processStepInput,
) error {
	SetConfigValue(&input.step.With.Config, "app_url", input.config["app_url"])
	input.logger.Info("MobileAutomationSetupHook: processing step", "id", input.step.ID)

	payload, err := decodeAndValidatePayload(input.step)
	if err != nil {
		return err
	}

	// Use global_runner_id if step-level runner_id is not set
	if payload.RunnerID == "" && input.globalRunnerID != "" {
		payload.RunnerID = input.globalRunnerID
		// Update the step payload with the global runner_id for consistency
		SetPayloadValue(&input.step.With.Payload, "runner_id", input.globalRunnerID)
	}

	if !isSemaphoreManagedRun(input.config) && !hasRunnerPermit(input.runData, payload.RunnerID) {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("missing runner permit for step %s", input.step.ID),
			payload.RunnerID,
		)
	}

	taskqueue := fmt.Sprintf("%s-%s", payload.RunnerID, "TaskQueue")
	SetConfigValue(&input.step.With.Config, "taskqueue", taskqueue)
	mobileAo := *input.ao
	mobileAo.TaskQueue = taskqueue
	mobileCtx := workflow.WithActivityOptions(input.ctx, mobileAo)

	appURL, ok := input.config["app_url"].(string)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("missing or invalid app_url for step %s", input.step.ID),
		)
	}

	deviceMap, err := getOrCreateDeviceMap(getOrCreateDeviceMapInput{
		ctx:              input.ctx,
		mobileCtx:        mobileCtx,
		payload:          payload,
		settedDevices:    input.settedDevices,
		appURL:           appURL,
		stepID:           input.step.ID,
		httpActivity:     input.httpActivity,
		startEmuActivity: input.startEmuActivity,
	})
	if err != nil {
		return err
	}

	serial, ok := deviceMap["serial"].(string)
	if !ok {
		serial = ""
	}
	SetPayloadValue(&input.step.With.Payload, "serial", serial)

	runnerURL, ok := deviceMap["runner_url"].(string)
	if !ok || runnerURL == "" {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("missing or invalid runner_url for step %s", input.step.ID),
			deviceMap,
		)
	}

	SetRunDataValue(input.runData, "setted_devices", input.settedDevices)

	if err := fetchAndInstallAPK(fetchAndInstallAPKInput{
		ctx:             input.ctx,
		mobileCtx:       mobileCtx,
		step:            input.step,
		payload:         payload,
		deviceMap:       deviceMap,
		appURL:          appURL,
		runnerURL:       runnerURL,
		serial:          serial,
		httpActivity:    input.httpActivity,
		installActivity: input.installActivity,
	}); err != nil {
		return err
	}

	SetRunDataValue(input.runData, "setted_devices", input.settedDevices)

	return nil
}

func decodeAndValidatePayload(
	step *StepDefinition,
) (*workflows.MobileAutomationWorkflowPipelinePayload, error) {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	if len(step.With.Payload) == 0 {
		return nil, workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"missing payload for step %s: expected with.action_id or with.payload.action_id",
				step.ID,
			),
		)
	}
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
	input getOrCreateDeviceMapInput,
) (map[string]any, error) {
	deviceInfo, exists := input.settedDevices[input.payload.RunnerID]
	var deviceMap map[string]any
	if exists {
		deviceMap = deviceInfo.(map[string]any)
		return deviceMap, nil
	}

	deviceMap = map[string]any{
		"installed": make(map[string]string),
		"recording": false,
	}
	input.settedDevices[input.payload.RunnerID] = deviceMap

	if err := setupNewDevice(setupNewDeviceInput{
		ctx:              input.ctx,
		mobileCtx:        input.mobileCtx,
		payload:          input.payload,
		deviceMap:        deviceMap,
		appURL:           input.appURL,
		stepID:           input.stepID,
		httpActivity:     input.httpActivity,
		startEmuActivity: input.startEmuActivity,
	}); err != nil {
		return nil, err
	}

	return deviceMap, nil
}

func setupNewDevice(
	input setupNewDeviceInput,
) error {
	runnerURL, serial, err := fetchRunnerInfo(fetchRunnerInfoInput{
		ctx:          input.ctx,
		payload:      input.payload,
		appURL:       input.appURL,
		stepID:       input.stepID,
		httpActivity: input.httpActivity,
	})
	if err != nil {
		return err
	}

	if serial == "" {
		cloneName, newSerial, err := startEmulator(startEmulatorInput{
			ctx:              input.ctx,
			mobileCtx:        input.mobileCtx,
			payload:          input.payload,
			stepID:           input.stepID,
			startEmuActivity: input.startEmuActivity,
		})
		if err != nil {
			return err
		}
		serial = newSerial
		input.deviceMap["clone_name"] = cloneName
	}

	input.deviceMap["runner_url"] = runnerURL
	input.deviceMap["serial"] = serial

	return nil
}

func fetchRunnerInfo(
	input fetchRunnerInfoInput,
) (string, string, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]

	runnerReq := workflowengine.ActivityInput{
		Payload: map[string]any{
			"method":          http.MethodGet,
			"url":             utils.JoinURL(input.appURL, "api", "mobile-runner"),
			"expected_status": 200,
			"query_params": map[string]string{
				"runner_identifier": input.payload.RunnerID,
			},
		},
	}

	var runnerRes workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(input.ctx, input.httpActivity.Name(), runnerReq).
		Get(input.ctx, &runnerRes); err != nil {
		return "", "", err
	}

	body, ok := runnerRes.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		return "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("invalid HTTP response format for step %s", input.stepID),
			runnerRes.Output,
		)
	}

	runnerURL, ok := body["runner_url"].(string)
	if !ok || runnerURL == "" {
		return "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("missing or invalid runner_url for step %s", input.stepID),
			body,
		)
	}

	serial, ok := body["serial"].(string)
	if !ok {
		return "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("invalid device serial for step %s", input.stepID),
			body,
		)
	}

	return runnerURL, serial, nil
}

func startEmulator(
	input startEmulatorInput,
) (string, string, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]

	startResult := workflowengine.ActivityResult{}
	startInput := workflowengine.ActivityInput{
		Payload: map[string]any{"device_name": input.payload.RunnerID},
	}
	err := workflow.ExecuteActivity(input.mobileCtx, input.startEmuActivity.Name(), startInput).
		Get(input.ctx, &startResult)
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
				input.stepID,
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
				input.stepID,
			),
			startResult.Output,
		)
	}

	return cloneName, serial, nil
}

func fetchAndInstallAPK(
	input fetchAndInstallAPKInput,
) error {
	req := workflowengine.ActivityInput{
		Payload: map[string]any{
			"method": http.MethodPost,
			"url": utils.JoinURL(
				input.runnerURL,
				"credimi",
				"apk-action",
			),
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
			"body": map[string]any{
				"instance_url":       input.appURL,
				"version_identifier": input.payload.VersionID,
				"action_identifier":  input.payload.ActionID,
			},
			"expected_status": 200,
		},
	}

	var res workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(input.ctx, input.httpActivity.Name(), req).
		Get(input.ctx, &res); err != nil {
		return err
	}

	apkPath, versionIdentifier, actionCode, err := parseAPKResponse(
		res,
		input.payload,
		input.step,
	)
	if err != nil {
		return err
	}

	if err := installAPKIfNeeded(installAPKIfNeededInput{
		mobileCtx:       input.mobileCtx,
		deviceMap:       input.deviceMap,
		apkPath:         apkPath,
		versionID:       versionIdentifier,
		serial:          input.serial,
		stepID:          input.step.ID,
		installActivity: input.installActivity,
	}); err != nil {
		return err
	}

	if input.payload.ActionCode == "" {
		SetPayloadValue(&input.step.With.Payload, "action_code", actionCode)
		SetPayloadValue(&input.step.With.Payload, "stored_action_code", true)
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
	input installAPKIfNeededInput,
) error {
	installed, ok := input.deviceMap["installed"].(map[string]string)
	if !ok {
		installed = make(map[string]string)
	}

	if _, ok := installed[input.versionID]; !ok {
		installInput := workflowengine.ActivityInput{
			Payload: map[string]any{"apk": input.apkPath, "serial": input.serial},
		}
		installOutput := workflowengine.ActivityResult{}
		if err := workflow.ExecuteActivity(input.mobileCtx, input.installActivity.Name(), installInput).
			Get(input.mobileCtx, &installOutput); err != nil {
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
					input.stepID,
				),
				installOutput.Output,
			)
		}
		installed[input.versionID] = packageID
		input.deviceMap["installed"] = installed
	}

	return nil
}

func startRecordingForDevices(
	input startRecordingForDevicesInput,
) error {
	for runnerID, dev := range input.settedDevices {
		deviceMap := dev.(map[string]any)
		recording := deviceMap["recording"].(bool)
		if recording {
			continue
		}

		if err := startRecordingForDevice(startRecordingForDeviceInput{
			ctx:            input.ctx,
			runnerID:       runnerID,
			deviceMap:      deviceMap,
			ao:             input.ao,
			recordActivity: input.recordActivity,
		}); err != nil {
			return err
		}
	}
	return nil
}

func startRecordingForDevice(
	input startRecordingForDeviceInput,
) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]

	serial, ok := input.deviceMap["serial"].(string)
	if !ok || serial == "" {
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("missing serial for device %s", input.runnerID),
		)
	}

	mobileAO := *input.ao
	mobileAO.TaskQueue = fmt.Sprintf("%s-TaskQueue", input.runnerID)
	mobileCtx := workflow.WithActivityOptions(input.ctx, mobileAO)

	startRecordInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"serial":      serial,
			"workflow_id": workflow.GetInfo(mobileCtx).WorkflowExecution.ID,
		},
	}
	var recordResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(
		mobileCtx,
		input.recordActivity.Name(),
		startRecordInput,
	).Get(mobileCtx, &recordResult); err != nil {
		return err
	}

	if err := extractAndStoreRecordingInfo(
		recordResult,
		input.deviceMap,
		input.runnerID,
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

	logcatPath, ok := recordResult.Output.(map[string]any)["logcat_path"].(string)
	if !ok {
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing logcat_path in start record video response for device %s",
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
	deviceMap["logcat_path"] = logcatPath

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

	devices, _ := runData["setted_devices"].(map[string]any)

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
		if err := cleanupDevice(cleanupDeviceInput{
			ctx:           ctx,
			runnerID:      runnerID,
			raw:           raw,
			mobileAo:      &mobileAo,
			runIdentifier: runIdentifier,
			appURL:        appURL,
			output:        output,
			cleanupErrs:   &cleanupErrs,
			logger:        logger,
		}); err != nil {
			cleanupErrs = append(cleanupErrs, err)
		}
	}

	if !isSemaphoreManagedRun(config) {
		releaseRunnerPermits(ctx, getRunnerPermits(runData), &cleanupErrs)
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

func isSemaphoreManagedRun(config map[string]any) bool {
	if config == nil {
		return false
	}
	ticketID, ok := config[mobileRunnerSemaphoreTicketIDConfigKey].(string)
	return ok && ticketID != ""
}

func cleanupDevice(
	input cleanupDeviceInput,
) error {
	deviceMap, err := parseDeviceMap(input.runnerID, input.raw)
	if err != nil {
		*input.cleanupErrs = append(*input.cleanupErrs, err)
	}

	serial, cloneName, packages, err := extractDeviceInfo(input.runnerID, deviceMap)
	if err != nil {
		*input.cleanupErrs = append(*input.cleanupErrs, err)
	}

	input.mobileAo.TaskQueue = fmt.Sprintf("%s-%s", input.runnerID, "TaskQueue")
	mobileCtx := workflow.WithActivityOptions(input.ctx, *input.mobileAo)

	cleanupRecording(cleanupRecordingInput{
		ctx:         mobileCtx,
		runnerID:    input.runnerID,
		deviceInfo:  deviceMap,
		runID:       input.runIdentifier,
		output:      input.output,
		cleanupErrs: input.cleanupErrs,
		appURL:      input.appURL,
	})

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
	).Get(input.ctx, nil); err != nil {
		input.logger.Error(
			"failed ",
			"mobile device cleanup",
			input.runnerID,
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
	input cleanupRecordingInput,
) {
	logger := workflow.GetLogger(input.ctx)

	runner_url, ok := input.deviceInfo["runner_url"].(string)
	if !ok || runner_url == "" {
		*input.cleanupErrs = append(*input.cleanupErrs,
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
				"missing runner_url for device "+input.runnerID,
			),
		)
		return
	}

	recording, ok := input.deviceInfo["recording"].(bool)
	if !ok || !recording {
		return
	}

	recordingInfo, err := extractRecordingInfo(input.runnerID, input.deviceInfo)
	if err != nil {
		*input.cleanupErrs = append(*input.cleanupErrs, err)
	}

	lastFramePath, err := stopRecording(
		input.ctx,
		recordingInfo,
		logger,
	)
	if err != nil {
		*input.cleanupErrs = append(*input.cleanupErrs, err)
	}

	if err := storeRecordingResults(storeRecordingResultsInput{
		ctx:        input.ctx,
		runnerURL:  runner_url,
		videoPath:  recordingInfo.videoPath,
		lastFrame:  lastFramePath,
		logcatPath: recordingInfo.logcatPath,
		runID:      input.runID,
		runnerID:   input.runnerID,
		appURL:     input.appURL,
		output:     input.output,
		logger:     logger,
	}); err != nil {
		*input.cleanupErrs = append(*input.cleanupErrs, err)
	}
}

type recordingInfo struct {
	videoPath  string
	logcatPath string
	adbPid     int
	ffmpegPid  int
	logcatPid  int
}

func extractRecordingInfo(
	runnerID string,
	deviceInfo map[string]any,
) (*recordingInfo, error) {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]

	videoPath, ok := deviceInfo["video_path"].(string)
	if !ok || videoPath == "" {
		return nil, workflowengine.NewAppError(
			errCode,
			"missing video_path for device "+runnerID,
		)
	}

	logcatPath, ok := deviceInfo["logcat_path"].(string)
	if !ok || logcatPath == "" {
		return nil, workflowengine.NewAppError(
			errCode,
			"missing logcat_path for device "+runnerID,
		)
	}

	recordingAdbPid, ok := deviceInfo["recording_adb_pid"].(int)
	if !ok || recordingAdbPid == 0 {
		return nil, workflowengine.NewAppError(
			errCode,
			"missing recording_adb_pid for device "+runnerID,
		)
	}

	recordingFfmpegPid, ok := deviceInfo["recording_ffmpeg_pid"].(int)
	if !ok || recordingFfmpegPid == 0 {
		return nil, workflowengine.NewAppError(
			errCode,
			"missing recording_ffmpeg_pid for device "+runnerID,
		)
	}

	recordingLogcatPid, ok := deviceInfo["recording_logcat_pid"].(int)
	if !ok || recordingLogcatPid == 0 {
		return nil, workflowengine.NewAppError(
			errCode,
			"missing recording_logcat_pid for device "+runnerID,
		)
	}

	return &recordingInfo{
		videoPath:  videoPath,
		logcatPath: logcatPath,
		adbPid:     recordingAdbPid,
		ffmpegPid:  recordingFfmpegPid,
		logcatPid:  recordingLogcatPid,
	}, nil
}

func stopRecording(
	ctx workflow.Context,
	info *recordingInfo,
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
		return "", err
	}

	lastFramePath, ok := stopResult.Output.(map[string]any)["last_frame_path"].(string)
	if !ok || lastFramePath == "" {
		err := workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
			"missing last_frame_path in stop recording result",
			stopResult.Output,
		)
		return "", err
	}

	return lastFramePath, nil
}

func storeRecordingResults(
	input storeRecordingResultsInput,
) error {
	httpActivity := activities.NewHTTPActivity()
	var storeResult workflowengine.ActivityResult

	if err := workflow.ExecuteActivity(
		input.ctx,
		httpActivity.Name(),
		workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method: http.MethodPost,
				URL: utils.JoinURL(
					input.runnerURL,
					"credimi",
					"pipeline-result",
				),
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]any{
					"video_path":        input.videoPath,
					"last_frame_path":   input.lastFrame,
					"logcat_path":       input.logcatPath,
					"run_identifier":    input.runID,
					"runner_identifier": input.runnerID,
					"instance_url":      input.appURL,
				},
				ExpectedStatus: 200,
			},
		},
	).Get(input.ctx, &storeResult); err != nil {
		input.logger.Error("cleanup: store result failed", "error", err)
		return err
	}

	if err := extractAndStoreURLs(storeResult, input.output); err != nil {
		return err
	}

	return nil
}

func extractAndStoreURLs(
	storeResult workflowengine.ActivityResult,
	output *map[string]any,
) error {
	body, ok := storeResult.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		err := workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
			"missing body in store result",
			storeResult.Output,
		)
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
