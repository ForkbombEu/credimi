//go:build credimi_extra

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"strings"
	"time"

	"github.com/forkbombeu/credimi-extra/mobile"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const MobileAutomationTaskQueue = "MobileAutomationTaskQueue"

const (
	externalInstallAppDetectionTimeout  = 20 * time.Second
	externalInstallAppDetectionInterval = 2 * time.Second
)

// MobileAutomationWorkflow is a workflow that runs a mobile automation flow
type MobileAutomationWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

// MobileExternalInstallWorkflow is a workflow that runs an external install-app step
// and performs post-install checks using the app detected on the device.
type MobileExternalInstallWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

// MobileAutomationWorkflowPayload is the payload for the mobile automation workflow
type MobileAutomationWorkflowPayload struct {
	RunIdentifier    string            `json:"run_identifier,omitempty"     yaml:"run_identifier,omitempty"`
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

func NewMobileExternalInstallWorkflow() *MobileExternalInstallWorkflow {
	w := &MobileExternalInstallWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (MobileAutomationWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (MobileExternalInstallWorkflow) GetOptions() workflow.ActivityOptions {
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

func (MobileExternalInstallWorkflow) Name() string {
	return "Run a mobile external install workflow"
}

func (w *MobileAutomationWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *MobileExternalInstallWorkflow) Workflow(
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
	ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)

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

	mobileActivity := activities.NewRunMobileFlowActivity()
	var mobileResponse workflowengine.ActivityResult
	mobileInput := workflowengine.ActivityInput{
		Config: workflowengine.ActivityTelemetryConfig(ctx, input.Config),
		Payload: mobile.RunMobileFlowPayload{
			Serial:     payload.Serial,
			Type:       payload.Type,
			Yaml:       payload.ActionCode,
			Parameters: payload.Parameters,
			WorkflowId: workflow.GetInfo(ctx).WorkflowExecution.ID,
		},
	}
	executeErr := workflow.ExecuteActivity(ctx, mobileActivity.Name(), mobileInput).
		Get(ctx, &mobileResponse)
	output.FlowOutput = mobileResponse.Output

	if executeErr != nil {
		if temporal.IsCanceledError(executeErr) {
			return workflowengine.WorkflowResult{}, executeErr
		}

		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			executeErr,
			input.RunMetadata,
			map[string]any{
				"output": output,
			},
		)
	}

	return workflowengine.WorkflowResult{
		Output: output,
	}, nil
}

func (w *MobileExternalInstallWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)

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

	runnerAO := *input.ActivityOptions
	runnerAO.TaskQueue = taskqueue
	runnerCtx := workflow.WithActivityOptions(ctx, runnerAO)

	beforeApps, err := executeListInstalledApps(runnerCtx, payload.Serial, payload.Type)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
			map[string]any{"output": output},
		)
	}

	mobileActivity := activities.NewRunMobileFlowActivity()
	var mobileResponse workflowengine.ActivityResult
	mobileInput := workflowengine.ActivityInput{
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
	output.FlowOutput = map[string]any{
		"mobile_flow": mobileResponse.Output,
	}

	if executeErr != nil {
		if temporal.IsCanceledError(executeErr) {
			return workflowengine.WorkflowResult{}, executeErr
		}

		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			executeErr,
			input.RunMetadata,
			map[string]any{
				"output": output,
			},
		)
	}

	addedApps, afterApps, attempts, err := waitForAddedInstalledApps(
		runnerCtx,
		payload.Serial,
		payload.Type,
		beforeApps,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
			map[string]any{
				"before_apps": beforeApps,
				"output":      output,
			},
		)
	}

	if len(addedApps) != 1 {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			fmt.Errorf("expected exactly one installed app, found %d", len(addedApps)),
			input.RunMetadata,
			map[string]any{
				"before_apps": beforeApps,
				"after_apps":  afterApps,
				"added_apps":  addedApps,
				"attempts":    attempts,
				"output":      output,
			},
		)
	}

	postInstallActivityName := activities.NewApkPostInstallChecksActivity().Name()
	postInstallPayload := map[string]any{
		"serial":     payload.Serial,
		"package_id": addedApps[0],
	}
	if isIOSWorkflowDeviceType(payload.Type) {
		postInstallActivityName = activities.NewIOSPostInstallChecksActivity().Name()
		postInstallPayload["type"] = payload.Type
	}

	var postInstallResponse workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(
		runnerCtx,
		postInstallActivityName,
		workflowengine.ActivityInput{Payload: postInstallPayload},
	).Get(runnerCtx, &postInstallResponse); err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
			map[string]any{"output": output},
		)
	}

	output.FlowOutput = map[string]any{
		"mobile_flow":  mobileResponse.Output,
		"post_install": postInstallResponse.Output,
	}

	return workflowengine.WorkflowResult{
		Output: output,
	}, nil
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
	attempts := make([][]string, 0, int(externalInstallAppDetectionTimeout/externalInstallAppDetectionInterval)+1)

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
