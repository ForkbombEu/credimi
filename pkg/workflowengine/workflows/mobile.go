// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"github.com/forkbombeu/credimi-extra/mobile"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

const MobileAutomationTaskQueue = "MobileAutomationTaskQueue"

// MobileAutomationWorkflow is a workflow that runs a mobile automation flow
type MobileAutomationWorkflow struct{}

// MobileAutomationWorkflowPayload is the payload for the mobile automation workflow
type MobileAutomationWorkflowPayload struct {
	RunIdentifier      string            `json:"run_identifier,omitempty"       yaml:"run_identifier,omitempty"`
	ActionID           string            `json:"action_id,omitempty"            yaml:"action_id,omitempty"`
	VersionID          string            `json:"version_id,omitempty"           yaml:"version_id,omitempty"`
	ActionCode         string            `json:"action_code,omitempty"          yaml:"action_code,omitempty"`
	StoredActionCode   bool              `json:"stored_action_code,omitempty"   yaml:"stored_action_code,omitempty"`
	EmulatorSerial     string            `json:"emulator_serial,omitempty"      yaml:"emulator_serial,omitempty"`
	Parameters         map[string]string `json:"parameters,omitempty"           yaml:"parameters,omitempty"`
	VideoPath          string            `json:"video_path,omitempty"           yaml:"video_path,omitempty"`
	RecordingAdbPid    int               `json:"recording_adb_pid,omitempty"    yaml:"recording_adb_pid,omitempty"`
	RecordingFfmpegPid int               `json:"recording_ffmpeg_pid,omitempty" yaml:"recording_ffmpeg_pid,omitempty"`
}

type MobileAutomationWorkflowPipelinePayload struct {
	ActionID   string            `json:"action_id,omitempty"   yaml:"action_id,omitempty"`
	VersionID  string            `json:"version_id,omitempty"  yaml:"version_id,omitempty"`
	ActionCode string            `json:"action_code,omitempty" yaml:"action_code,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"  yaml:"parameters,omitempty"`
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
	ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)

	var output MobileWorkflowOutput
	testRunURL := utils.JoinURL(
		input.Config["app_url"].(string),
		"my", "tests", "runs",
		workflow.GetInfo(ctx).WorkflowExecution.ID,
		workflow.GetInfo(ctx).WorkflowExecution.RunID,
	)
	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI:   testRunURL,
	}
	output.TestRunURL = testRunURL

	payload, err := workflowengine.DecodePayload[MobileAutomationWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			runMetadata,
		)
	}

	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			runMetadata,
		)
	}

	mobileActivity := activities.NewRunMobileFlowActivity()
	var mobileResponse workflowengine.ActivityResult
	mobileInput := workflowengine.ActivityInput{
		Payload: mobile.RunMobileFlowPayload{
			EmulatorSerial: payload.EmulatorSerial,
			Yaml:           payload.ActionCode,
			Parameters:     payload.Parameters,
			WorkflowId:     workflow.GetInfo(ctx).WorkflowExecution.ID,
		},
	}
	executeErr := workflow.ExecuteActivity(ctx, mobileActivity.Name(), mobileInput).
		Get(ctx, &mobileResponse)
	output.FlowOutput = mobileResponse.Output

	if executeErr != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			executeErr,
			runMetadata,
			map[string]any{
				"output": output,
			},
		)
	}

	return workflowengine.WorkflowResult{
		Output: output,
	}, nil
}
