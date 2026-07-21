//go:build credimi_extra

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi-extra/mobile"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const MobileAutomationTaskQueue = "MobileAutomationTaskQueue"

const pipelineTaskQueue = "PipelineTaskQueue"

const (
	externalInstallDetectionConfigKey    = "detect_external_install"
	externalInstallAppDetectionTimeout   = 20 * time.Second
	externalInstallAppDetectionInterval  = 2 * time.Second
	mobileActivityHeartbeatTimeout       = 30 * time.Second
	mobileActivityScheduleToStartTimeout = 30 * time.Second
)

// MobileAutomationWorkflow is a workflow that runs a mobile automation flow
type MobileAutomationWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

// MobileAutomationWorkflowPayload is the payload for the mobile automation workflow
type MobileAutomationWorkflowPayload struct {
	ActionID         string            `json:"action_id,omitempty"          yaml:"action_id,omitempty"`
	VersionID        string            `json:"version_id,omitempty"         yaml:"version_id,omitempty"`
	ActionCode       string            `json:"action_code,omitempty"        yaml:"action_code,omitempty"`
	StoredActionCode bool              `json:"stored_action_code,omitempty" yaml:"stored_action_code,omitempty"`
	Serial           string            `json:"serial,omitempty"             yaml:"serial,omitempty"`
	Type             string            `json:"type,omitempty"               yaml:"type,omitempty"`
	RunnerID         string            `json:"runner_id,omitempty"          yaml:"runner_id,omitempty"`
	Parameters       map[string]string `json:"parameters,omitempty"         yaml:"parameters,omitempty"`
}

type MobileAutomationWorkflowPipelinePayload struct {
	ActionID   string            `json:"action_id,omitempty"   yaml:"action_id,omitempty"`
	VersionID  string            `json:"version_id,omitempty"  yaml:"version_id,omitempty"`
	ActionCode string            `json:"action_code,omitempty" yaml:"action_code,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"  yaml:"parameters,omitempty"`
	RunnerID   string            `json:"runner_id,omitempty"   yaml:"runner_id,omitempty"`
}

func NewMobileAutomationWorkflow() *MobileAutomationWorkflow {
	w := &MobileAutomationWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (MobileAutomationWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

type MobileWorkflowOutput struct {
	TestRunURL     string `json:"test_run_url"`
	ResultVideoURL string `json:"result_video_url,omitempty"`
	FlowOutput     any    `json:"flow_output,omitempty"`
}

func (MobileAutomationWorkflow) Name() string {
	return "Run a mobile automation workflow"
}

func (w *MobileAutomationWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

// ExecuteWorkflow executes a mobile automation workflow, given the input payload.
// It first creates a test run URL and then executes the RunMobileFlowActivity with the given payload.
// If the video flag is set, it checks if the video file exists and if it does, stores the result in the mobile server.
// Finally, it returns a WorkflowResult containing the test run URL and the result video URL.
func (w *MobileAutomationWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, mobileActivityOptions(input.ActivityOptions, ""))

	var output MobileWorkflowOutput
	testRunURL := utils.JoinURL(
		input.Config["app_url"].(string),
		"my", "tests", "runs",
		workflow.GetInfo(ctx).WorkflowExecution.ID,
		workflow.GetInfo(ctx).WorkflowExecution.RunID,
	)

	output.TestRunURL = testRunURL

	payload, err := workflowengine.DecodePayload[MobileAutomationWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}

	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			input.RunMetadata,
		)
	}
	_ = appURL

	taskqueue, ok := input.Config["taskqueue"].(string)
	if !ok || strings.TrimSpace(taskqueue) == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"taskqueue",
			input.RunMetadata,
		)
	}

	runnerCtx := workflow.WithActivityOptions(ctx, mobileActivityOptions(input.ActivityOptions, taskqueue))
	externalInstall := workflowengine.AsBool(input.Config[externalInstallDetectionConfigKey])
	var beforeApps []string
	if externalInstall {
		beforeApps, err = executeListInstalledApps(runnerCtx, payload.Serial, payload.Type)
		if err != nil {
			if temporal.IsCanceledError(err) {
				return workflowengine.WorkflowResult{}, err
			}
			return workflowengine.WorkflowResult{}, newMobileWorkflowError(
				err,
				input.RunMetadata,
				map[string]any{"output": output},
			)
		}
	}

	mobileActivity := activities.NewRunMobileFlowActivity()
	var mobileResponse workflowengine.ActivityResult
	mobileInput := workflowengine.ActivityInput{
		Config: workflowengine.ActivityTelemetryConfig(runnerCtx, input.Config),
		Payload: mobile.RunMobileFlowPayload{
			Serial:     payload.Serial,
			Type:       payload.Type,
			Yaml:       payload.ActionCode,
			Parameters: payload.Parameters,
			WorkflowId: workflow.GetInfo(ctx).WorkflowExecution.ID,
		},
	}
	executeErr := workflow.ExecuteActivity(runnerCtx, mobileActivity.Name(), mobileInput).
		Get(runnerCtx, &mobileResponse)

	if executeErr != nil {
		if temporal.IsCanceledError(executeErr) {
			return workflowengine.WorkflowResult{}, executeErr
		}

		return workflowengine.WorkflowResult{}, newMobileWorkflowError(
			executeErr,
			input.RunMetadata,
			map[string]any{
				"output": output,
			},
		)
	}
	flowOutput, err := storeMobileFlowScreenshots(
		ctx,
		input,
		payload,
		mobileResponse.Output,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, newMobileWorkflowError(
			err,
			input.RunMetadata,
			map[string]any{"output": output},
		)
	}
	if externalInstall {
		postInstallOutput, err := runExternalInstallPostChecks(
			runnerCtx,
			payload,
			beforeApps,
			input.RunMetadata,
			output,
		)
		if err != nil {
			return workflowengine.WorkflowResult{}, err
		}
		output.FlowOutput = map[string]any{
			"mobile_flow":  flowOutput,
			"post_install": postInstallOutput,
		}
	} else {
		output.FlowOutput = flowOutput
	}

	return workflowengine.WorkflowResult{
		Output: output,
	}, nil
}

func runExternalInstallPostChecks(
	ctx workflow.Context,
	payload MobileAutomationWorkflowPayload,
	beforeApps []string,
	runMetadata *workflowengine.WorkflowRunMetadata,
	output MobileWorkflowOutput,
) (any, error) {
	addedApps, afterApps, attempts, err := waitForAddedInstalledApps(ctx, payload.Serial, payload.Type, beforeApps)
	if err != nil {
		if temporal.IsCanceledError(err) {
			return nil, err
		}
		return nil, newMobileWorkflowError(err, runMetadata, map[string]any{
			"before_apps": beforeApps,
			"output":      output,
		})
	}
	if len(addedApps) != 1 {
		return nil, newMobileWorkflowError(
			fmt.Errorf("expected exactly one installed app, found %d", len(addedApps)),
			runMetadata,
			map[string]any{
				"before_apps": beforeApps,
				"after_apps":  afterApps,
				"added_apps":  addedApps,
				"attempts":    attempts,
				"output":      output,
			},
		)
	}

	activityName := activities.NewApkPostInstallChecksActivity().Name()
	payloadMap := map[string]any{"serial": payload.Serial, "package_id": addedApps[0]}
	if isIOSWorkflowDeviceType(payload.Type) {
		activityName = activities.NewIOSPostInstallChecksActivity().Name()
		payloadMap["type"] = payload.Type
	}

	var result workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(ctx, activityName, workflowengine.ActivityInput{Payload: payloadMap}).Get(ctx, &result); err != nil {
		if temporal.IsCanceledError(err) {
			return nil, err
		}
		return nil, newMobileWorkflowError(err, runMetadata, map[string]any{"output": output})
	}
	return result.Output, nil
}

func storeMobileFlowScreenshots(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
	payload MobileAutomationWorkflowPayload,
	rawFlowOutput any,
) (map[string]any, error) {
	flowOutput := workflowengine.AsMap(rawFlowOutput)
	screenshotPaths := workflowengine.AsSliceOfStrings(flowOutput["maestro_screenshot_paths"])
	if len(screenshotPaths) == 0 {
		return flowOutput, nil
	}

	runnerURL := workflowengine.AsString(input.Config["runner_url"])
	stepID := workflowengine.AsString(input.Config["step_id"])
	runIdentifier := workflowengine.AsString(input.Config["run_identifier"])
	if runnerURL == "" || stepID == "" || runIdentifier == "" || payload.RunnerID == "" {
		return nil, workflowengine.NewMissingConfigError(
			"runner_url, step_id, run_identifier, or runner_id",
			input.RunMetadata,
		)
	}

	activityOptions := mobileActivityOptions(input.ActivityOptions, pipelineTaskQueue)
	storageCtx := workflow.WithActivityOptions(ctx, activityOptions)
	httpActivity := activities.NewInternalHTTPActivity()
	var storeResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(
		storageCtx,
		httpActivity.Name(),
		workflowengine.ActivityInput{Payload: activities.InternalHTTPActivityPayload{
			Method: http.MethodPost,
			URL: utils.JoinURL(
				runnerURL,
				"credimi",
				"execution-screenshots",
			),
			Headers: map[string]string{
				workflowengine.HTTPHeaderContentType: workflowengine.MIMEApplicationJSON,
			},
			Body: map[string]any{
				"run_identifier":    runIdentifier,
				"runner_identifier": payload.RunnerID,
				"step_id":           stepID,
				"screenshot_paths":  screenshotPaths,
			},
			Timeout:        "300",
			ExpectedStatus: http.StatusOK,
		}},
	).Get(storageCtx, &storeResult); err != nil {
		return nil, err
	}

	responseBody := workflowengine.AsMap(workflowengine.AsMap(storeResult.Output)["body"])
	screenshotURLs := workflowengine.AsSliceOfStrings(responseBody["screenshot_urls"])
	if len(screenshotURLs) == 0 {
		return nil, workflowengine.NewMissingOrInvalidPayloadError(
			fmt.Errorf("runner response is missing screenshot_urls"),
			input.RunMetadata,
		)
	}

	delete(flowOutput, "maestro_screenshot_paths")
	flowOutput["maestro_screenshot_urls"] = screenshotURLs
	return flowOutput, nil
}

func mobileActivityOptions(
	input *workflow.ActivityOptions,
	taskQueue string,
) workflow.ActivityOptions {
	var options workflow.ActivityOptions
	if input != nil {
		options = *input
	}
	if options.HeartbeatTimeout == 0 {
		options.HeartbeatTimeout = mobileActivityHeartbeatTimeout
	}
	if taskQueue != "" {
		if options.ScheduleToStartTimeout == 0 {
			options.ScheduleToStartTimeout = mobileActivityScheduleToStartTimeout
		}
		options.TaskQueue = taskQueue
	}
	return options
}

func executeListInstalledApps(
	ctx workflow.Context,
	serial string,
	deviceType string,
) ([]string, error) {
	listActivity := activities.NewListInstalledAppsActivity()
	var result workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(
		ctx,
		listActivity.Name(),
		workflowengine.ActivityInput{
			Payload: mobile.ListInstalledAppsPayload{
				Serial: serial,
				Type:   deviceType,
			},
		},
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	return workflowengine.AsSliceOfStrings(result.Output), nil
}

func waitForAddedInstalledApps(
	ctx workflow.Context,
	serial string,
	deviceType string,
	beforeApps []string,
) ([]string, []string, [][]string, error) {
	deadline := workflow.Now(ctx).Add(externalInstallAppDetectionTimeout)
	attempts := make(
		[][]string,
		0,
		int(externalInstallAppDetectionTimeout/externalInstallAppDetectionInterval)+1,
	)

	for {
		afterApps, err := executeListInstalledApps(ctx, serial, deviceType)
		if err != nil {
			return nil, nil, attempts, err
		}

		attempts = append(attempts, append([]string(nil), afterApps...))
		addedApps := diffAddedApps(beforeApps, afterApps)
		if len(addedApps) == 1 {
			return addedApps, afterApps, attempts, nil
		}

		if !workflow.Now(ctx).Before(deadline) {
			return addedApps, afterApps, attempts, nil
		}

		if err := workflow.Sleep(ctx, externalInstallAppDetectionInterval); err != nil {
			return nil, nil, attempts, err
		}
	}
}

func diffAddedApps(before, after []string) []string {
	beforeSet := make(map[string]struct{}, len(before))
	for _, appID := range before {
		trimmed := strings.TrimSpace(appID)
		if trimmed == "" {
			continue
		}
		beforeSet[trimmed] = struct{}{}
	}

	var added []string
	for _, appID := range after {
		trimmed := strings.TrimSpace(appID)
		if trimmed == "" {
			continue
		}
		if _, exists := beforeSet[trimmed]; exists {
			continue
		}
		added = append(added, trimmed)
	}

	return added
}

func isIOSWorkflowDeviceType(deviceType string) bool {
	switch strings.TrimSpace(strings.ToLower(deviceType)) {
	case "ios", "ios_phone", "ios_simulator":
		return true
	default:
		return false
	}
}

func newMobileWorkflowError(
	err error,
	metadata *workflowengine.WorkflowRunMetadata,
	details map[string]any,
) error {
	failure := workflowengine.ParseWorkflowError(err)
	if failure.Code == "" {
		errCode := errorcodes.Codes[errorcodes.ChildWorkflowExecutionError]
		failure = workflowengine.WorkflowError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: err.Error(),
		}
	}
	if len(details) > 0 {
		if failure.Details == nil {
			failure.Details = map[string]any{}
		}
		for key, value := range details {
			failure.Details[key] = value
		}
	}

	return workflowengine.NewWorkflowError(workflowengine.NewAppError(failure), metadata)
}
