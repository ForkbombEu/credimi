// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"errors"
	"time"

	"github.com/forkbombeu/credimi-extra/mobile"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const MobileAutomationTaskQueue = "MobileAutomationTaskQueue"

// MobileAutomationWorkflow is a workflow that runs a mobile automation flow
type MobileAutomationWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

// MobileAutomationWorkflowPayload is the payload for the mobile automation workflow
type MobileAutomationWorkflowPayload struct {
	RunIdentifier      string            `json:"run_identifier,omitempty"       yaml:"run_identifier,omitempty"`
	ActionID           string            `json:"action_id,omitempty"            yaml:"action_id,omitempty"`
	VersionID          string            `json:"version_id,omitempty"           yaml:"version_id,omitempty"`
	ActionCode         string            `json:"action_code,omitempty"          yaml:"action_code,omitempty"`
	StoredActionCode   bool              `json:"stored_action_code,omitempty"   yaml:"stored_action_code,omitempty"`
	EmulatorSerial     string            `json:"emulator_serial,omitempty"      yaml:"emulator_serial,omitempty"`
	CloneName          string            `json:"clone_name,omitempty"           yaml:"clone_name,omitempty"`
	Parameters         map[string]string `json:"parameters,omitempty"           yaml:"parameters,omitempty"`
	VideoPath          string            `json:"video_path,omitempty"           yaml:"video_path,omitempty"`
	RecordingAdbPid    int               `json:"recording_adb_pid,omitempty"    yaml:"recording_adb_pid,omitempty"`
	RecordingFfmpegPid int               `json:"recording_ffmpeg_pid,omitempty" yaml:"recording_ffmpeg_pid,omitempty"`
	RecordingLogcatPid int               `json:"recording_logcat_pid,omitempty" yaml:"recording_logcat_pid,omitempty"`
}

type MobileAutomationWorkflowPipelinePayload struct {
	ActionID   string            `json:"action_id,omitempty"   yaml:"action_id,omitempty"`
	VersionID  string            `json:"version_id,omitempty"  yaml:"version_id,omitempty"`
	ActionCode string            `json:"action_code,omitempty" yaml:"action_code,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"  yaml:"parameters,omitempty"`
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
	mobileAo := *input.ActivityOptions
	mobileAo.HeartbeatTimeout = time.Minute
	mobileAo.StartToCloseTimeout = 35 * time.Minute
	mobileAo.ScheduleToCloseTimeout = 35 * time.Minute
	ctx = workflow.WithActivityOptions(ctx, mobileAo)

	var output MobileWorkflowOutput
	status := "running"
	lastActivityName := ""
	lastActivityTime := time.Time{}
	forceCleanup := false
	recordingPaused := false
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

	recordingActive := payload.VideoPath != "" || payload.RecordingFfmpegPid != 0
	bootStatus := "unknown"

	forceCleanupCh := workflow.GetSignalChannel(ctx, workflowengine.ForceCleanupSignal)
	pauseRecordingCh := workflow.GetSignalChannel(ctx, workflowengine.PauseRecordingSignal)
	workflow.Go(ctx, func(ctx workflow.Context) {
		var signal struct{}
		for {
			forceCleanupCh.Receive(ctx, &signal)
			forceCleanup = true
			lastActivityName = "force_cleanup"
			lastActivityTime = workflow.Now(ctx)
		}
	})
	workflow.Go(ctx, func(ctx workflow.Context) {
		var signal struct{}
		for {
			pauseRecordingCh.Receive(ctx, &signal)
			recordingPaused = true
			lastActivityName = "pause_recording"
			lastActivityTime = workflow.Now(ctx)
		}
	})

	workflow.SetQueryHandler(ctx, workflowengine.PipelineStateQuery, func() (workflowengine.PipelineState, error) {
		info := workflow.GetInfo(ctx)
		return workflowengine.PipelineState{
			WorkflowID:       info.WorkflowExecution.ID,
			RunID:            info.WorkflowExecution.RunID,
			EmulatorSerial:   payload.EmulatorSerial,
			CloneName:        payload.CloneName,
			VersionID:        payload.VersionID,
			RecordingActive:  recordingActive && !recordingPaused,
			RecordingPaused:  recordingPaused,
			BootStatus:       bootStatus,
			LastActivity:     lastActivityName,
			LastActivityTime: lastActivityTime,
			Status:           status,
			ForceCleanup:     forceCleanup,
		}, nil
	})

	workflow.SetQueryHandler(ctx, workflowengine.ResourceUsageQuery, func() (workflowengine.ResourceUsage, error) {
		return workflowengine.ResourceUsage{}, nil
	})

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
		Payload: mobile.RunMobileFlowPayload{
			EmulatorSerial: payload.EmulatorSerial,
			Yaml:           payload.ActionCode,
			Parameters:     payload.Parameters,
			WorkflowId:     workflow.GetInfo(ctx).WorkflowExecution.ID,
		},
	}
	lastActivityName = mobileActivity.Name()
	lastActivityTime = workflow.Now(ctx)
	executeErr := workflow.ExecuteActivity(ctx, mobileActivity.Name(), mobileInput).
		Get(ctx, &mobileResponse)
	output.FlowOutput = mobileResponse.Output

	if executeErr != nil {
		details := map[string]any{
			"output": output,
		}
		var timeoutErr *temporal.TimeoutError
		if errors.As(executeErr, &timeoutErr) && timeoutErr.TimeoutType() == enumspb.TIMEOUT_TYPE_HEARTBEAT {
			var heartbeat mobile.ActivityHeartbeat
			if err := timeoutErr.LastHeartbeatDetails(&heartbeat); err == nil {
				details["heartbeat"] = heartbeat
			}
		}
		status = "failed"
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			executeErr,
			input.RunMetadata,
			details,
		)
	}

	status = "completed"
	return workflowengine.WorkflowResult{
		Output: output,
	}, nil
}
