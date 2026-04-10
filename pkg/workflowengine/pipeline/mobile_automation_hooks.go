// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

const (
	mobileAutomationStepUse                 = "mobile-automation"
	mobileExternalInstallStepUse            = "mobile-external-install"
	mobileRunnerSemaphoreTicketIDConfigKey  = "mobile_runner_semaphore_ticket_id"
	mobileDisableAndroidPlayStoreConfigKey  = "disable_android_play_store"
	mobilePendingPlayStoreDisableRunDataKey = "mobile_pending_play_store_disable"
	mobileSpecialInstallMetadataKey         = "mobile_special_install"
	mobileSkipInstallerRequestKey           = "skip_installer"
	mobilePlatformAndroid                   = "android"
	mobilePlatformIOS                       = "ios"
	mobileExternalSourceVersionID           = "installed_from_external_source"
	walletActionCategoryInstallApp          = "install-app"
)

type mobileDeviceType string

const (
	deviceTypeAndroidEmulator mobileDeviceType = "android_emulator"
	deviceTypeAndroidPhone    mobileDeviceType = "android_phone"
	deviceTypeIOSSimulator    mobileDeviceType = "ios_simulator"
	deviceTypeIOSPhone        mobileDeviceType = "ios_phone"
	deviceTypeRedroid         mobileDeviceType = "redroid"
)

type platformActivities struct {
	Start             string
	Install           string
	PostInstall       string
	StartRecording    string
	StopRecording     string
	InstallAssetField string
}

var (
	androidPlatformActivities = platformActivities{
		Start:             activities.NewStartEmulatorActivity().Name(),
		Install:           activities.NewApkInstallActivity().Name(),
		PostInstall:       activities.NewApkPostInstallChecksActivity().Name(),
		StartRecording:    activities.NewStartRecordingActivity().Name(),
		StopRecording:     activities.NewStopRecordingActivity().Name(),
		InstallAssetField: "apk",
	}
	iosPlatformActivities = platformActivities{
		Start:             activities.NewStartIOSSimulatorActivity().Name(),
		Install:           activities.NewInstallIOSAppActivity().Name(),
		PostInstall:       activities.NewIOSPostInstallChecksActivity().Name(),
		StartRecording:    activities.NewStartIOSRecordingActivity().Name(),
		StopRecording:     activities.NewStopIOSRecordingActivity().Name(),
		InstallAssetField: "app",
	}
)

type processStepInput struct {
	ctx            workflow.Context
	step           *pipeline.StepDefinition
	config         map[string]any
	ao             *workflow.ActivityOptions
	settedDevices  map[string]any
	runData        *map[string]any
	httpActivity   *activities.HTTPActivity
	logger         log.Logger
	globalRunnerID string
}

type fetchAndInstallAPKInput struct {
	ctx             workflow.Context
	mobileCtx       workflow.Context
	step            *pipeline.StepDefinition
	payload         *workflows.MobileAutomationWorkflowPipelinePayload
	deviceMap       map[string]any
	deviceType      mobileDeviceType
	activities      platformActivities
	appURL          string
	runnerURL       string
	serial          string
	skipInstaller   bool
	externalInstall bool
	httpActivity    *activities.HTTPActivity
}

type getOrCreateDeviceMapInput struct {
	ctx                       workflow.Context
	mobileCtx                 workflow.Context
	payload                   *workflows.MobileAutomationWorkflowPipelinePayload
	settedDevices             map[string]any
	appURL                    string
	stepID                    string
	trackInitialInstalledApps bool
	httpActivity              *activities.HTTPActivity
}

type setupNewDeviceInput struct {
	ctx                       workflow.Context
	mobileCtx                 workflow.Context
	payload                   *workflows.MobileAutomationWorkflowPipelinePayload
	deviceMap                 map[string]any
	appURL                    string
	stepID                    string
	trackInitialInstalledApps bool
	httpActivity              *activities.HTTPActivity
}

type fetchRunnerInfoInput struct {
	ctx          workflow.Context
	payload      *workflows.MobileAutomationWorkflowPipelinePayload
	appURL       string
	stepID       string
	httpActivity *activities.HTTPActivity
}

type startManagedDeviceInput struct {
	ctx        workflow.Context
	mobileCtx  workflow.Context
	payload    *workflows.MobileAutomationWorkflowPipelinePayload
	deviceType mobileDeviceType
	activities platformActivities
	stepID     string
}

type installAppIfNeededInput struct {
	mobileCtx  workflow.Context
	deviceMap  map[string]any
	appPath    string
	versionID  string
	serial     string
	stepID     string
	deviceType mobileDeviceType
	activities platformActivities
}

type startRecordingForDevicesInput struct {
	ctx           workflow.Context
	settedDevices map[string]any
	ao            *workflow.ActivityOptions
}

type disablePlayStoreForDevicesInput struct {
	ctx           workflow.Context
	settedDevices map[string]any
	ao            *workflow.ActivityOptions
}

type startRecordingForDeviceInput struct {
	ctx       workflow.Context
	runnerID  string
	deviceMap map[string]any
	ao        *workflow.ActivityOptions
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
	mobileCtx   workflow.Context
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
	logPath    string
	deviceType mobileDeviceType
	runID      string
	runnerID   string
	appURL     string
	output     *map[string]any
	logger     log.Logger
}

func MobileAutomationSetupHook(
	ctx workflow.Context,
	steps *[]pipeline.StepDefinition,
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
	if hasExternalSourceMobileSteps(*steps) {
		appURL, ok := config["app_url"].(string)
		if !ok || appURL == "" {
			errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
			return workflowengine.NewAppError(
				errCode,
				"missing or invalid app_url in workflow input config",
			)
		}

		if err := prepareMobileAutomationSteps(ctx, steps, appURL, httpActivity); err != nil {
			return err
		}
	}

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
			ctx:            ctx,
			step:           step,
			config:         config,
			ao:             ao,
			settedDevices:  settedDevices,
			runData:        runData,
			httpActivity:   httpActivity,
			logger:         logger,
			globalRunnerID: globalRunnerID,
		}); err != nil {
			return err
		}
	}

	shouldDisablePlayStore := workflowengine.AsBool(config[mobileDisableAndroidPlayStoreConfigKey])
	if shouldDisablePlayStore && hasExternalInstallWorkflowSteps(*steps) {
		SetRunDataValue(runData, mobilePendingPlayStoreDisableRunDataKey, true)
	}

	if shouldDisablePlayStore && !hasPendingPlayStoreDisable(*runData) {
		if err := disablePlayStoreForDevices(disablePlayStoreForDevicesInput{
			ctx:           ctx,
			settedDevices: settedDevices,
			ao:            ao,
		}); err != nil {
			return err
		}
	}

	if err := startRecordingForDevices(startRecordingForDevicesInput{
		ctx:           ctx,
		settedDevices: settedDevices,
		ao:            ao,
	}); err != nil {
		return err
	}

	SetRunDataValue(runData, "setted_devices", settedDevices)

	return nil
}

// validateRunnerIDConfiguration checks that either:
// - all mobile-automation steps have a defined runner_id, OR
// - there is a global_runner_id set
func validateRunnerIDConfiguration(steps *[]pipeline.StepDefinition, globalRunnerID string) error {
	var mobileAutomationSteps []*pipeline.StepDefinition
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
		runnerID, ok := step.With.Payload["runner_id"].(string)
		if !ok || canonify.NormalizePath(runnerID) == "" {
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

func collectMobileRunnerIDs(steps []pipeline.StepDefinition, globalID string) ([]string, error) {
	uniqueRunnerIDs := make(map[string]struct{})

	globalID = canonify.NormalizePath(globalID)
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
		payload.RunnerID = canonify.NormalizePath(payload.RunnerID)
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

func prepareMobileAutomationSteps(
	ctx workflow.Context,
	steps *[]pipeline.StepDefinition,
	appURL string,
	httpActivity *activities.HTTPActivity,
) error {
	specialSteps := make([]pipeline.StepDefinition, 0)
	remainingSteps := make([]pipeline.StepDefinition, 0, len(*steps))

	for i := range *steps {
		step := (*steps)[i]
		if step.Use != mobileAutomationStepUse {
			remainingSteps = append(remainingSteps, step)
			continue
		}

		payload, err := decodeAndValidatePayload(&step)
		if err != nil {
			return err
		}

		if payload.VersionID != mobileExternalSourceVersionID {
			remainingSteps = append(remainingSteps, step)
			continue
		}

		category, err := fetchMobileActionCategory(
			ctx,
			appURL,
			payload.ActionID,
			step.ID,
			httpActivity,
		)
		if err != nil {
			return err
		}
		if category != walletActionCategoryInstallApp {
			remainingSteps = append(remainingSteps, step)
			continue
		}

		if step.Metadata == nil {
			step.Metadata = map[string]any{}
		}
		step.Metadata[mobileSpecialInstallMetadataKey] = true
		specialSteps = append(specialSteps, step)
	}

	if len(specialSteps) == 0 {
		return nil
	}

	reordered := make([]pipeline.StepDefinition, 0, len(*steps))
	reordered = append(reordered, specialSteps...)
	reordered = append(reordered, remainingSteps...)
	*steps = reordered

	return nil
}

func fetchMobileActionCategory(
	ctx workflow.Context,
	appURL string,
	actionID string,
	stepID string,
	httpActivity *activities.HTTPActivity,
) (string, error) {
	if strings.TrimSpace(actionID) == "" {
		return "", nil
	}

	validatePayload := map[string]any{
		"canonified_name": actionID,
	}
	validateURL := utils.JoinURL(appURL, "api", "canonify", "identifier", "validate")

	internalReq := workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method:         http.MethodPost,
			URL:            validateURL,
			ExpectedStatus: 200,
			Body:           validatePayload,
		},
	}

	internalHTTPActivity := activities.NewInternalHTTPActivity()
	var result workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(ctx, internalHTTPActivity.Name(), internalReq).Get(ctx, &result); err != nil {
		if !isMissingPipelineInternalHTTPActivity(err) {
			return "", err
		}

		fallbackReq := workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method:         http.MethodPost,
				URL:            validateURL,
				ExpectedStatus: 200,
				Body:           validatePayload,
			},
		}
		if fbErr := workflow.ExecuteActivity(
			ctx,
			httpActivity.Name(),
			fallbackReq,
		).Get(ctx, &result); fbErr != nil {
			return "", fbErr
		}
	}

	body, ok := result.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		return "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("invalid action validation response for step %s", stepID),
			result.Output,
		)
	}

	record, ok := body["record"].(map[string]any)
	if !ok {
		return "", nil
	}

	category, ok := record["category"].(string)
	if !ok || strings.TrimSpace(category) == "" {
		return "", nil
	}

	return strings.TrimSpace(category), nil
}

func hasExternalSourceMobileSteps(steps []pipeline.StepDefinition) bool {
	for i := range steps {
		if steps[i].Use != mobileAutomationStepUse {
			continue
		}
		if workflowengine.AsString(
			steps[i].With.Payload["version_id"],
		) == mobileExternalSourceVersionID {
			return true
		}
	}

	return false
}

func hasExternalInstallWorkflowSteps(steps []pipeline.StepDefinition) bool {
	for i := range steps {
		if steps[i].Use == mobileExternalInstallStepUse {
			return true
		}
	}

	return false
}

func hasPendingPlayStoreDisable(runData map[string]any) bool {
	return workflowengine.AsBool(runData[mobilePendingPlayStoreDisableRunDataKey])
}

func runPendingPlayStoreDisableIfNeeded(
	ctx workflow.Context,
	step pipeline.StepDefinition,
	ao *workflow.ActivityOptions,
	config map[string]any,
	runData *map[string]any,
) error {
	if !workflowengine.AsBool(config[mobileDisableAndroidPlayStoreConfigKey]) {
		return nil
	}
	if !hasPendingPlayStoreDisable(*runData) {
		return nil
	}
	if step.Use == mobileExternalInstallStepUse {
		return nil
	}

	if err := disablePlayStoreForDevices(disablePlayStoreForDevicesInput{
		ctx:           ctx,
		settedDevices: getOrCreateSettedDevices(runData),
		ao:            ao,
	}); err != nil {
		return err
	}

	SetRunDataValue(runData, mobilePendingPlayStoreDisableRunDataKey, false)

	return nil
}

func runPendingPlayStoreDisableAfterSteps(
	ctx workflow.Context,
	ao *workflow.ActivityOptions,
	config map[string]any,
	runData *map[string]any,
) error {
	if !workflowengine.AsBool(config[mobileDisableAndroidPlayStoreConfigKey]) {
		return nil
	}
	if !hasPendingPlayStoreDisable(*runData) {
		return nil
	}

	if err := disablePlayStoreForDevices(disablePlayStoreForDevicesInput{
		ctx:           ctx,
		settedDevices: getOrCreateSettedDevices(runData),
		ao:            ao,
	}); err != nil {
		return err
	}

	SetRunDataValue(runData, mobilePendingPlayStoreDisableRunDataKey, false)

	return nil
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
	payload.RunnerID = canonify.NormalizePath(payload.RunnerID)
	if payload.RunnerID == "" && input.globalRunnerID != "" {
		payload.RunnerID = input.globalRunnerID
		// Update the step payload with the global runner_id for consistency
		SetPayloadValue(&input.step.With.Payload, "runner_id", input.globalRunnerID)
	}

	taskqueue := mobileRunnerTaskQueue(payload.RunnerID)
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
		ctx:                       input.ctx,
		mobileCtx:                 mobileCtx,
		payload:                   payload,
		settedDevices:             input.settedDevices,
		appURL:                    appURL,
		stepID:                    input.step.ID,
		trackInitialInstalledApps: isSpecialMobileInstallStep(input.step),
		httpActivity:              input.httpActivity,
	})
	if err != nil {
		return err
	}

	deviceType := deviceTypeFromMap(deviceMap)
	deviceActivities := activitiesForDeviceType(deviceType)

	serial, ok := deviceMap["serial"].(string)
	if !ok {
		serial = ""
	}
	SetPayloadValue(&input.step.With.Payload, "serial", serial)
	if deviceType != "" {
		SetPayloadValue(&input.step.With.Payload, "type", deviceType.String())
	}

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
		deviceType:      deviceType,
		activities:      deviceActivities,
		appURL:          appURL,
		runnerURL:       runnerURL,
		serial:          serial,
		skipInstaller:   payload.VersionID == mobileExternalSourceVersionID,
		externalInstall: isSpecialMobileInstallStep(input.step),
		httpActivity:    input.httpActivity,
	}); err != nil {
		return err
	}

	SetRunDataValue(input.runData, "setted_devices", input.settedDevices)

	return nil
}

func isSpecialMobileInstallStep(step *pipeline.StepDefinition) bool {
	if step == nil || step.Metadata == nil {
		return false
	}

	special, ok := step.Metadata[mobileSpecialInstallMetadataKey].(bool)
	return ok && special
}

func decodeAndValidatePayload(
	step *pipeline.StepDefinition,
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
		ctx:                       input.ctx,
		mobileCtx:                 input.mobileCtx,
		payload:                   input.payload,
		deviceMap:                 deviceMap,
		appURL:                    input.appURL,
		stepID:                    input.stepID,
		trackInitialInstalledApps: input.trackInitialInstalledApps,
		httpActivity:              input.httpActivity,
	}); err != nil {
		return nil, err
	}

	return deviceMap, nil
}

func setupNewDevice(
	input setupNewDeviceInput,
) error {
	runnerURL, deviceType, serial, err := fetchRunnerInfo(fetchRunnerInfoInput{
		ctx:          input.ctx,
		payload:      input.payload,
		appURL:       input.appURL,
		stepID:       input.stepID,
		httpActivity: input.httpActivity,
	})
	if err != nil {
		return err
	}

	deviceActivities := activitiesForDeviceType(deviceType)
	if deviceType.IsManagedEmulator() {
		name, newSerial, err := startManagedDevice(startManagedDeviceInput{
			ctx:        input.ctx,
			mobileCtx:  input.mobileCtx,
			deviceType: deviceType,
			activities: deviceActivities,
			payload:    input.payload,
			stepID:     input.stepID,
		})
		if err != nil {
			return err
		}
		if newSerial != "" {
			serial = newSerial
		}
		input.deviceMap["name"] = name
	}

	input.deviceMap["type"] = deviceType.String()
	input.deviceMap["runner_url"] = runnerURL
	input.deviceMap["serial"] = serial

	if input.trackInitialInstalledApps {
		initialInstalledApps, err := listInstalledAppsOnRunner(
			input.mobileCtx,
			serial,
			deviceType.String(),
		)
		if err != nil {
			return err
		}
		input.deviceMap["initial_installed_apps"] = initialInstalledApps
	}

	return nil
}

func fetchRunnerInfo(
	input fetchRunnerInfoInput,
) (string, mobileDeviceType, string, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]

	runnerReq := workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method:         http.MethodGet,
			URL:            utils.JoinURL(input.appURL, "api", "mobile-runner"),
			ExpectedStatus: 200,
			QueryParams: map[string]string{
				"runner_identifier": input.payload.RunnerID,
			},
		},
	}
	internalHTTPActivity := activities.NewInternalHTTPActivity()

	var runnerRes workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(input.ctx, internalHTTPActivity.Name(), runnerReq).
		Get(input.ctx, &runnerRes); err != nil {
		if isMissingPipelineInternalHTTPActivity(err) {
			fallbackReq := workflowengine.ActivityInput{
				Payload: activities.HTTPActivityPayload{
					Method:         http.MethodGet,
					URL:            utils.JoinURL(input.appURL, "api", "mobile-runner"),
					ExpectedStatus: 200,
					QueryParams: map[string]string{
						"runner_identifier": input.payload.RunnerID,
					},
				},
			}
			if fbErr := workflow.ExecuteActivity(
				input.ctx,
				activities.NewHTTPActivity().Name(),
				fallbackReq,
			).Get(input.ctx, &runnerRes); fbErr != nil {
				return "", "", "", fbErr
			}
		} else {
			return "", "", "", err
		}
	}

	body, ok := runnerRes.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		return "", "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("invalid HTTP response format for step %s", input.stepID),
			runnerRes.Output,
		)
	}

	runnerURL, ok := body["runner_url"].(string)
	if !ok || runnerURL == "" {
		return "", "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("missing or invalid runner_url for step %s", input.stepID),
			body,
		)
	}

	rawDeviceType, ok := body["type"].(string)
	if !ok {
		return "", "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("missing or invalid device type for step %s", input.stepID),
			body,
		)
	}
	deviceType := normalizeDeviceType(rawDeviceType)
	if deviceType == "" {
		return "", "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("missing or invalid device type for step %s", input.stepID),
			body,
		)
	}

	serial, ok := body["serial"].(string)
	if !ok {
		return "", "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("invalid device serial for step %s", input.stepID),
			body,
		)
	}

	return runnerURL, deviceType, serial, nil
}

func startManagedDevice(
	input startManagedDeviceInput,
) (string, string, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]

	startResult := workflowengine.ActivityResult{}
	startInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"device_name": input.payload.RunnerID,
			"type":        input.deviceType.String(),
		},
		Config: workflowengine.ActivityTelemetryConfig(input.mobileCtx, nil),
	}
	err := workflow.ExecuteActivity(input.mobileCtx, input.activities.Start, startInput).
		Get(input.ctx, &startResult)
	if err != nil {
		return "", "", err
	}

	var serial string

	body, ok := startResult.Output.(map[string]any)
	if !ok {
		return "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: invalid response format for step %s",
				errCode.Description,
				input.stepID,
			),
			startResult.Output,
		)
	}

	if serialValue, exists := body["serial"]; exists && serialValue != nil {
		serial, ok = serialValue.(string)
		if !ok {
			return "", "", workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: invalid serial in response for step %s",
					errCode.Description,
					input.stepID,
				),
				startResult.Output,
			)
		}
	}

	name, ok := body["name"].(string)
	if !ok {
		return "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing name in response for step %s",
				errCode.Description,
				input.stepID,
			),
			startResult.Output,
		)
	}

	return name, serial, nil
}

func listInstalledAppsOnRunner(
	mobileCtx workflow.Context,
	serial string,
	deviceType string,
) ([]string, error) {
	var result workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(
		mobileCtx,
		activities.NewListInstalledAppsActivity().Name(),
		workflowengine.ActivityInput{
			Payload: map[string]any{
				"serial": serial,
				"type":   deviceType,
			},
		},
	).Get(mobileCtx, &result); err != nil {
		return nil, err
	}

	return workflowengine.AsSliceOfStrings(result.Output), nil
}

func fetchAndInstallAPK(
	input fetchAndInstallAPKInput,
) error {
	body := map[string]any{
		"instance_url":       input.appURL,
		"version_identifier": input.payload.VersionID,
		"action_identifier":  input.payload.ActionID,
		"platform":           installerPlatformForDeviceType(input.deviceType),
	}
	if input.skipInstaller {
		body[mobileSkipInstallerRequestKey] = true
	}

	req := workflowengine.ActivityInput{
		Payload: activities.HTTPActivityPayload{
			Method: http.MethodPost,
			URL: utils.JoinURL(
				input.runnerURL,
				"credimi",
				"installer-action",
			),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body:           body,
			Timeout:        "300",
			ExpectedStatus: 200,
		},
	}

	var res workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(input.ctx, input.httpActivity.Name(), req).
		Get(input.ctx, &res); err != nil {
		return err
	}

	responseBody, err := parseInstallerActionResponseBody(res, input.step)
	if err != nil {
		return err
	}

	actionCode, err := parseInstallerActionCode(responseBody, input.payload, input.step)
	if err != nil {
		return err
	}
	if input.payload.ActionCode == "" {
		SetPayloadValue(&input.step.With.Payload, "action_code", actionCode)
		SetPayloadValue(&input.step.With.Payload, "stored_action_code", true)
	}
	if input.externalInstall {
		input.step.Use = mobileExternalInstallStepUse
		return nil
	}
	if input.skipInstaller {
		return nil
	}

	apkPath, versionIdentifier, err := parseInstallerResponse(responseBody, input.step)
	if err != nil {
		return err
	}

	if err := installAppIfNeeded(installAppIfNeededInput{
		mobileCtx:  input.mobileCtx,
		deviceMap:  input.deviceMap,
		appPath:    apkPath,
		versionID:  versionIdentifier,
		serial:     input.serial,
		stepID:     input.step.ID,
		deviceType: input.deviceType,
		activities: input.activities,
	}); err != nil {
		return err
	}

	return nil
}

func parseInstallerActionResponseBody(
	res workflowengine.ActivityResult,
	step *pipeline.StepDefinition,
) (map[string]any, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]

	body, ok := res.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		return nil, workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("invalid HTTP response format for step %s", step.ID),
			res.Output,
		)
	}

	return body, nil
}

func parseInstallerResponse(
	body map[string]any,
	step *pipeline.StepDefinition,
) (string, string, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]

	apkPath, ok := body["installer_path"].(string)
	if !ok {
		return "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing installer_path in response for step %s",
				errCode.Description,
				step.ID,
			),
			body,
		)
	}

	versionIdentifier, ok := body["version_id"].(string)
	if !ok {
		return "", "", workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing version_id in response for step %s",
				errCode.Description,
				step.ID,
			),
			body,
		)
	}

	return apkPath, versionIdentifier, nil
}

func parseInstallerActionCode(
	body map[string]any,
	payload *workflows.MobileAutomationWorkflowPipelinePayload,
	step *pipeline.StepDefinition,
) (string, error) {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
	actionCode := payload.ActionCode
	if actionCode == "" {
		var ok bool
		actionCode, ok = body["code"].(string)
		if !ok || actionCode == "" {
			return "", workflowengine.NewAppError(
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

	return actionCode, nil
}

func parseAPKResponse(
	res workflowengine.ActivityResult,
	payload *workflows.MobileAutomationWorkflowPipelinePayload,
	step *pipeline.StepDefinition,
) (string, string, string, error) {
	body, err := parseInstallerActionResponseBody(res, step)
	if err != nil {
		return "", "", "", err
	}

	apkPath, versionIdentifier, err := parseInstallerResponse(body, step)
	if err != nil {
		return "", "", "", err
	}

	actionCode, err := parseInstallerActionCode(body, payload, step)
	if err != nil {
		return "", "", "", err
	}

	return apkPath, versionIdentifier, actionCode, nil
}

func installAppIfNeeded(
	input installAppIfNeededInput,
) error {
	installed, ok := input.deviceMap["installed"].(map[string]string)
	if !ok {
		installed = make(map[string]string)
	}

	if _, ok := installed[input.versionID]; !ok {
		installPayload := map[string]any{
			input.activities.InstallAssetField: input.appPath,
			"serial":                           input.serial,
			"type":                             input.deviceType.String(),
		}
		installInput := workflowengine.ActivityInput{
			Payload: installPayload,
			Config:  workflowengine.ActivityTelemetryConfig(input.mobileCtx, nil),
		}
		installOutput := workflowengine.ActivityResult{}
		if err := workflow.ExecuteActivity(input.mobileCtx, input.activities.Install, installInput).
			Get(input.mobileCtx, &installOutput); err != nil {
			return err
		}

		finalOutput := installOutput
		if input.activities.PostInstall != "" {
			postInstallOutput := workflowengine.ActivityResult{}
			if err := workflow.ExecuteActivity(input.mobileCtx, input.activities.PostInstall, installInput).
				Get(input.mobileCtx, &postInstallOutput); err != nil {
				return err
			}
			finalOutput = postInstallOutput
		}

		packageID, ok := finalOutput.Output.(map[string]any)["package_id"].(string)
		if !ok {
			errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf(
					"%s: missing package_id in response for step %s",
					errCode.Description,
					input.stepID,
				),
				finalOutput.Output,
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
			ctx:       input.ctx,
			runnerID:  runnerID,
			deviceMap: deviceMap,
			ao:        input.ao,
		}); err != nil {
			return err
		}
	}
	return nil
}

func disablePlayStoreForDevices(
	input disablePlayStoreForDevicesInput,
) error {
	for runnerID, dev := range input.settedDevices {
		deviceMap := dev.(map[string]any)
		if wasPlayStoreDisabled(deviceMap) {
			continue
		}

		deviceType := deviceTypeFromMap(deviceMap)
		if deviceType.IsIOS() {
			continue
		}

		serial, ok := deviceMap["serial"].(string)
		if !ok || serial == "" {
			errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
			return workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("missing serial for device %s", runnerID),
			)
		}

		mobileAO := *input.ao
		mobileAO.TaskQueue = mobileRunnerTaskQueue(runnerID)
		mobileCtx := workflow.WithActivityOptions(input.ctx, mobileAO)

		if err := workflow.ExecuteActivity(
			mobileCtx,
			activities.NewDisableAndroidPlayStoreActivity().Name(),
			workflowengine.ActivityInput{
				Payload: map[string]any{
					"serial": serial,
				},
			},
		).Get(mobileCtx, nil); err != nil {
			return err
		}

		deviceMap["play_store_disabled"] = true
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
	mobileAO.TaskQueue = mobileRunnerTaskQueue(input.runnerID)
	mobileCtx := workflow.WithActivityOptions(input.ctx, mobileAO)
	deviceType := deviceTypeFromMap(input.deviceMap)
	deviceActivities := activitiesForDeviceType(deviceType)

	startRecordInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"serial":      serial,
			"workflow_id": workflow.GetInfo(mobileCtx).WorkflowExecution.ID,
		},
		Config: workflowengine.ActivityTelemetryConfig(mobileCtx, nil),
	}
	var recordResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(
		mobileCtx,
		deviceActivities.StartRecording,
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
	deviceType := deviceTypeFromMap(deviceMap)
	output, ok := recordResult.Output.(map[string]any)
	if !ok {
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: invalid start record video response for device %s",
				errCode.Description,
				runnerID,
			),
			recordResult.Output,
		)
	}

	recordingProcessPID, ok := output["recording_process_pid"].(float64)
	if !ok {
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing recording_process in start record video response for device %s",
				errCode.Description,
				runnerID,
			),
			recordResult.Output,
		)
	}

	ffmpegPID := float64(0)
	logPID, ok := output["log_process_pid"].(float64)
	if !ok {
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing log_process in start record video response for device %s",
				errCode.Description,
				runnerID,
			),
			recordResult.Output,
		)
	}
	if !deviceType.IsIOS() {
		ffmpegPID, ok = output["ffmpeg_process_pid"].(float64)
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
	}

	videoPath, ok := output["video_path"].(string)
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

	logPath, hasLogPath := output["log_path"].(string)
	if !hasLogPath || logPath == "" {
		return workflowengine.NewAppError(
			errCode,
			fmt.Sprintf(
				"%s: missing log_path in start record video response for device %s",
				errCode.Description,
				runnerID,
			),
			recordResult.Output,
		)
	}

	deviceMap["recording_process_pid"] = int(recordingProcessPID)
	deviceMap["recording_ffmpeg_pid"] = int(ffmpegPID)
	deviceMap["recording_log_pid"] = int(logPID)
	deviceMap["recording"] = true
	deviceMap["video_path"] = videoPath
	if hasLogPath {
		deviceMap["log_path"] = logPath
	}

	return nil
}

func MobileAutomationCleanupHook(
	ctx workflow.Context,
	steps []pipeline.StepDefinition,
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

	deviceType, serial, name, packages, err := extractDeviceInfo(input.runnerID, deviceMap)
	if err != nil {
		*input.cleanupErrs = append(*input.cleanupErrs, err)
	}
	initialInstalledApps := extractInitialInstalledApps(deviceMap)
	reenablePlayStore := wasPlayStoreDisabled(deviceMap)

	input.mobileAo.TaskQueue = mobileRunnerTaskQueue(input.runnerID)
	mobileCtx := workflow.WithActivityOptions(input.ctx, *input.mobileAo)

	cleanupRecording(cleanupRecordingInput{

		ctx:         input.ctx,
		mobileCtx:   mobileCtx,
		runnerID:    input.runnerID,
		deviceInfo:  deviceMap,
		runID:       input.runIdentifier,
		output:      input.output,
		cleanupErrs: input.cleanupErrs,
		appURL:      input.appURL,
	})

	cleanupPayload := map[string]any{
		"serial":                 serial,
		"type":                   deviceType,
		"name":                   name,
		"apk_packages":           packages,
		"initial_installed_apps": initialInstalledApps,
		"reenable_play_store":    reenablePlayStore,
	}

	if err := workflow.ExecuteActivity(
		mobileCtx,
		activities.NewCleanupDeviceActivity().Name(),
		workflowengine.ActivityInput{
			Payload: cleanupPayload,
			Config:  workflowengine.ActivityTelemetryConfig(mobileCtx, nil),
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

func mobileRunnerTaskQueue(runnerID string) string {
	return fmt.Sprintf("%s-TaskQueue", canonify.NormalizePath(runnerID))
}

func normalizeDeviceType(raw string) mobileDeviceType {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "emulator", string(deviceTypeAndroidEmulator):
		return deviceTypeAndroidEmulator
	case "physical", string(deviceTypeAndroidPhone):
		return deviceTypeAndroidPhone
	case mobilePlatformIOS, string(deviceTypeIOSSimulator):
		return deviceTypeIOSSimulator
	case string(deviceTypeIOSPhone):
		return deviceTypeIOSPhone
	case string(deviceTypeRedroid):
		return deviceTypeRedroid
	default:
		return mobileDeviceType(strings.TrimSpace(strings.ToLower(raw)))
	}
}

func installerPlatformForDeviceType(deviceType mobileDeviceType) string {
	if deviceType.IsIOS() {
		return mobilePlatformIOS
	}

	return mobilePlatformAndroid
}

func (d mobileDeviceType) String() string {
	return string(d)
}

func (d mobileDeviceType) IsIOS() bool {
	switch d {
	case deviceTypeIOSSimulator, deviceTypeIOSPhone:
		return true
	default:
		return false
	}
}

func (d mobileDeviceType) IsManagedEmulator() bool {
	switch d {
	case deviceTypeAndroidEmulator, deviceTypeRedroid, deviceTypeIOSSimulator:
		return true
	default:
		return false
	}
}

func activitiesForDeviceType(deviceType mobileDeviceType) platformActivities {
	if deviceType.IsIOS() {
		return iosPlatformActivities
	}
	return androidPlatformActivities
}

func deviceTypeFromMap(deviceMap map[string]any) mobileDeviceType {
	return normalizeDeviceType(workflowengine.AsString(deviceMap["type"]))
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
) (string, string, string, []string, error) {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]

	deviceType, ok := deviceMap["type"].(string)
	if !ok || deviceType == "" {
		return "", "", "", nil, workflowengine.NewAppError(
			errCode,
			"error decoding payload for device "+runnerID,
			deviceMap,
		)
	}

	serial, ok := deviceMap["serial"].(string)
	if !ok || serial == "" {
		return "", "", "", nil, workflowengine.NewAppError(
			errCode,
			"error decoding payload for device "+runnerID,
			deviceMap,
		)
	}

	name, _ := deviceMap["name"].(string)

	var packages []string

	if installed, ok := deviceMap["installed"].(map[string]string); ok {
		for _, pkg := range installed {
			if pkg != "" {
				packages = append(packages, pkg)
			}
		}
	} else {
		return "", "", "", nil, workflowengine.NewAppError(
			errCode,
			"error decoding payload for device "+runnerID,
			deviceMap,
		)
	}

	return deviceType, serial, name, packages, nil
}

func extractInitialInstalledApps(deviceMap map[string]any) []string {
	return workflowengine.AsSliceOfStrings(deviceMap["initial_installed_apps"])
}

func wasPlayStoreDisabled(deviceMap map[string]any) bool {
	return workflowengine.AsBool(deviceMap["play_store_disabled"])
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
		input.mobileCtx,
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
		logPath:    recordingInfo.logPath,
		deviceType: recordingInfo.deviceType,
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
	deviceType   mobileDeviceType
	activities   platformActivities
	videoPath    string
	logPath      string
	recordingPid int
	ffmpegPid    int
	logPid       int
}

func extractRecordingInfo(
	runnerID string,
	deviceInfo map[string]any,
) (*recordingInfo, error) {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	deviceType := deviceTypeFromMap(deviceInfo)

	videoPath, ok := deviceInfo["video_path"].(string)
	if !ok || videoPath == "" {
		return nil, workflowengine.NewAppError(
			errCode,
			"missing video_path for device "+runnerID,
		)
	}

	logPath, hasLogPath := deviceInfo["log_path"].(string)
	if !hasLogPath || logPath == "" {
		return nil, workflowengine.NewAppError(
			errCode,
			"missing log_path for device "+runnerID,
		)
	}

	recordingPid, ok := deviceInfo["recording_process_pid"].(int)
	if !ok || recordingPid == 0 {
		return nil, workflowengine.NewAppError(
			errCode,
			"missing recording_process_pid for device "+runnerID,
		)
	}

	if deviceType.IsIOS() {
		logPid, ok := deviceInfo["recording_log_pid"].(int)
		if !ok || logPid == 0 {
			return nil, workflowengine.NewAppError(
				errCode,
				"missing recording_log_pid for device "+runnerID,
			)
		}
		return &recordingInfo{
			deviceType:   deviceType,
			activities:   activitiesForDeviceType(deviceType),
			videoPath:    videoPath,
			logPath:      logPath,
			recordingPid: recordingPid,
			logPid:       logPid,
		}, nil
	}

	recordingFfmpegPid, ok := deviceInfo["recording_ffmpeg_pid"].(int)
	if !ok || recordingFfmpegPid == 0 {
		return nil, workflowengine.NewAppError(
			errCode,
			"missing recording_ffmpeg_pid for device "+runnerID,
		)
	}

	recordingLogPid, ok := deviceInfo["recording_log_pid"].(int)
	if !ok || recordingLogPid == 0 {
		return nil, workflowengine.NewAppError(
			errCode,
			"missing recording_log_pid for device "+runnerID,
		)
	}

	return &recordingInfo{
		deviceType:   deviceType,
		activities:   activitiesForDeviceType(deviceType),
		videoPath:    videoPath,
		logPath:      logPath,
		recordingPid: recordingPid,
		ffmpegPid:    recordingFfmpegPid,
		logPid:       recordingLogPid,
	}, nil
}

func stopRecording(
	ctx workflow.Context,
	info *recordingInfo,
	logger log.Logger,
) (string, error) {
	var stopResult workflowengine.ActivityResult

	stopPayload := map[string]any{
		"video_path":            info.videoPath,
		"recording_process_pid": info.recordingPid,
		"ffmpeg_process_pid":    info.ffmpegPid,
		"log_process_pid":       info.logPid,
	}
	stopActivityName := info.activities.StopRecording
	if stopActivityName == "" {
		stopActivityName = activitiesForDeviceType(info.deviceType).StopRecording
	}
	if info.deviceType.IsIOS() {
		stopPayload = map[string]any{
			"recording_process_pid": info.recordingPid,
			"video_path":            info.videoPath,
			"log_process_pid":       info.logPid,
		}
	}

	if err := workflow.ExecuteActivity(
		ctx,
		stopActivityName,
		workflowengine.ActivityInput{
			Payload: stopPayload,
			Config:  workflowengine.ActivityTelemetryConfig(ctx, nil),
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
	body := map[string]any{
		"video_path":        input.videoPath,
		"last_frame_path":   input.lastFrame,
		"run_identifier":    input.runID,
		"runner_identifier": input.runnerID,
		"instance_url":      input.appURL,
		"platform":          installerPlatformForDeviceType(input.deviceType),
	}
	if input.logPath != "" {
		body["log_path"] = input.logPath
	}

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
				Body:           body,
				Timeout:        "300",
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
