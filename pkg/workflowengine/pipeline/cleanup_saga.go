// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

const (
	cleanupStepStopEmulator  = "stop-emulator"
	cleanupStepStopRecording = "stop-recording"
)

const cleanupStepSpecsKey = "cleanup_step_specs"

// CleanupStepSpec is a serializable cleanup step description.
type CleanupStepSpec struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	Payload        any    `json:"payload"`
	MaxRetries     int    `json:"max_retries"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	Idempotent     bool   `json:"idempotent"`
}

// CleanupStep is an executable cleanup step.
type CleanupStep struct {
	Spec    CleanupStepSpec
	Execute func(ctx workflow.Context) error
}

// CleanupActivityOptions groups activity options for cleanup operations.
type CleanupActivityOptions struct {
	Pipeline workflow.ActivityOptions
	Mobile   workflow.ActivityOptions
	Record   workflow.ActivityOptions
}

// StopEmulatorCleanupPayload defines the payload for stopping emulators in cleanup.
type StopEmulatorCleanupPayload struct {
	EmulatorSerial string `json:"emulator_serial" yaml:"emulator_serial" validate:"required"`
	CloneName      string `json:"clone_name"      yaml:"clone_name"      validate:"required"`
}

// StopRecordingCleanupPayload defines the payload for stopping recordings in cleanup.
type StopRecordingCleanupPayload struct {
	EmulatorSerial   string `json:"emulator_serial"    yaml:"emulator_serial"    validate:"required"`
	AdbProcessPid    int    `json:"adb_process_pid"    yaml:"adb_process_pid"    validate:"required"`
	FfmpegProcessPid int    `json:"ffmpeg_process_pid" yaml:"ffmpeg_process_pid" validate:"required"`
	LogcatProcessPid int    `json:"logcat_process_pid" yaml:"logcat_process_pid" validate:"required"`
	VideoPath        string `json:"video_path"        yaml:"video_path"        validate:"required"`
	RunIdentifier    string `json:"run_identifier"    yaml:"run_identifier"    validate:"required"`
	VersionID        string `json:"version_id"        yaml:"version_id"        validate:"required"`
	AppURL           string `json:"app_url"           yaml:"app_url"           validate:"required"`
}

func appendCleanupStepSpec(runData *map[string]any, spec CleanupStepSpec) {
	if runData == nil {
		return
	}
	if spec.MaxRetries == 0 {
		spec.MaxRetries = 3
	}
	if spec.Type == "" {
		spec.Type = cleanupStepType(spec.Name)
	}

	existing, _ := (*runData)[cleanupStepSpecsKey].([]CleanupStepSpec)
	existing = append(existing, spec)
	(*runData)[cleanupStepSpecsKey] = existing
}

func cleanupStepSpecs(runData map[string]any) []CleanupStepSpec {
	if runData == nil {
		return nil
	}
	if specs, ok := runData[cleanupStepSpecsKey].([]CleanupStepSpec); ok {
		return specs
	}
	if rawSpecs, ok := runData[cleanupStepSpecsKey].([]any); ok {
		out := make([]CleanupStepSpec, 0, len(rawSpecs))
		for _, raw := range rawSpecs {
			if spec, ok := raw.(CleanupStepSpec); ok {
				out = append(out, spec)
				continue
			}
			decoded, err := workflowengine.DecodePayload[CleanupStepSpec](raw)
			if err == nil {
				out = append(out, decoded)
			}
		}
		return out
	}
	return nil
}

func cleanupStepType(name string) string {
	if name == "" {
		return ""
	}
	parts := strings.SplitN(name, ":", 2)
	return parts[0]
}

func buildCleanupOptions(ao workflow.ActivityOptions) CleanupActivityOptions {
	pipelineAo := ao
	mobileAo := ao
	mobileAo.TaskQueue = workflows.MobileAutomationTaskQueue
	recordAo := mobileAo
	recordAo.HeartbeatTimeout = time.Minute
	recordAo.StartToCloseTimeout = 35 * time.Minute
	recordAo.ScheduleToCloseTimeout = 35 * time.Minute

	return CleanupActivityOptions{
		Pipeline: pipelineAo,
		Mobile:   mobileAo,
		Record:   recordAo,
	}
}

func executeCleanupSpecs(
	ctx workflow.Context,
	logger log.Logger,
	options CleanupActivityOptions,
	specs []CleanupStepSpec,
	output *map[string]any,
	recordFailure func(ctx workflow.Context, spec CleanupStepSpec, err error, attempt int) error,
) []error {
	steps := make([]CleanupStep, 0, len(specs))
	for _, spec := range specs {
		step, err := buildCleanupStep(spec, options, output)
		if err != nil {
			logger.Error("failed to build cleanup step", "step", spec.Name, "error", err)
			if recordFailure != nil {
				_ = recordFailure(ctx, spec, err, 0)
			}
			steps = append(steps, CleanupStep{
				Spec: spec,
				Execute: func(_ workflow.Context) error {
					return err
				},
			})
			continue
		}
		steps = append(steps, step)
	}

	var errorsOut []error
	for i := len(steps) - 1; i >= 0; i-- {
		step := steps[i]
		maxRetries := step.Spec.MaxRetries
		if maxRetries <= 0 {
			maxRetries = 1
		}

		var err error
		for attempt := 1; attempt <= maxRetries; attempt++ {
			err = step.Execute(ctx)
			if err == nil {
				break
			}
			if attempt < maxRetries {
				backoff := time.Duration(1<<uint(attempt-1)) * time.Second
				workflow.Sleep(ctx, backoff)
			}
		}

		if err != nil {
			logger.Error("cleanup step failed", "step", step.Spec.Name, "error", err)
			errorsOut = append(errorsOut, err)
			if recordFailure != nil {
				if recErr := recordFailure(ctx, step.Spec, err, maxRetries); recErr != nil {
					logger.Error("failed to record cleanup failure", "step", step.Spec.Name, "error", recErr)
				}
			}
		}
	}

	return errorsOut
}

func buildCleanupStep(
	spec CleanupStepSpec,
	options CleanupActivityOptions,
	output *map[string]any,
) (CleanupStep, error) {
	stepType := spec.Type
	if stepType == "" {
		stepType = cleanupStepType(spec.Name)
	}
	switch stepType {
	case cleanupStepStopEmulator:
		payload, err := workflowengine.DecodePayload[StopEmulatorCleanupPayload](spec.Payload)
		if err != nil {
			return CleanupStep{}, err
		}
		return CleanupStep{
			Spec: spec,
			Execute: func(ctx workflow.Context) error {
				return executeStopEmulatorCleanup(ctx, options, payload, spec.TimeoutSeconds)
			},
		}, nil
	case cleanupStepStopRecording:
		payload, err := workflowengine.DecodePayload[StopRecordingCleanupPayload](spec.Payload)
		if err != nil {
			return CleanupStep{}, err
		}
		return CleanupStep{
			Spec: spec,
			Execute: func(ctx workflow.Context) error {
				return executeStopRecordingCleanup(ctx, options, payload, output, spec.TimeoutSeconds)
			},
		}, nil
	default:
		return CleanupStep{}, fmt.Errorf("unknown cleanup step type: %s", spec.Type)
	}
}

func executeStopEmulatorCleanup(
	ctx workflow.Context,
	options CleanupActivityOptions,
	payload StopEmulatorCleanupPayload,
	timeoutSeconds int,
) error {
	stopEmulatorActivity := activities.NewStopEmulatorActivity()
	activityInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"emulator_serial": payload.EmulatorSerial,
			"clone_name":      payload.CloneName,
		},
	}

	ao := options.Mobile
	if timeoutSeconds > 0 {
		timeout := time.Duration(timeoutSeconds) * time.Second
		ao.StartToCloseTimeout = timeout
		ao.ScheduleToCloseTimeout = timeout
	}
	stepCtx := workflow.WithActivityOptions(ctx, ao)
	return workflow.ExecuteActivity(stepCtx, stopEmulatorActivity.Name(), activityInput).Get(stepCtx, nil)
}

func executeStopRecordingCleanup(
	ctx workflow.Context,
	options CleanupActivityOptions,
	payload StopRecordingCleanupPayload,
	output *map[string]any,
	timeoutSeconds int,
) error {
	stopRecordingActivity := activities.NewStopRecordingActivity()
	recordAo := options.Record
	if timeoutSeconds > 0 {
		timeout := time.Duration(timeoutSeconds) * time.Second
		recordAo.StartToCloseTimeout = timeout
		recordAo.ScheduleToCloseTimeout = timeout
	}
	recordCtx := workflow.WithActivityOptions(ctx, recordAo)

	stopInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"emulator_serial":    payload.EmulatorSerial,
			"video_path":         payload.VideoPath,
			"adb_process_pid":    payload.AdbProcessPid,
			"ffmpeg_process_pid": payload.FfmpegProcessPid,
			"logcat_process_pid": payload.LogcatProcessPid,
		},
	}

	var stopResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(recordCtx, stopRecordingActivity.Name(), stopInput).Get(recordCtx, &stopResult); err != nil {
		return err
	}

	stopOutput, ok := stopResult.Output.(map[string]any)
	if !ok {
		return workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
			"missing stop recording result",
			stopResult.Output,
		)
	}
	lastFramePath, ok := stopOutput["last_frame_path"].(string)
	if !ok || lastFramePath == "" {
		return workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
			"missing last_frame_path in stop recording result",
			stopResult.Output,
		)
	}

	httpActivity := activities.NewHTTPActivity()
	httpCtx := workflow.WithActivityOptions(ctx, options.Pipeline)
	var storeResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(
		httpCtx,
		httpActivity.Name(),
		workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method: "POST",
				URL:    utils.JoinURL(payload.AppURL, "store-pipeline-result"),
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: map[string]any{
					"video_path":         payload.VideoPath,
					"last_frame_path":    lastFramePath,
					"run_identifier":     payload.RunIdentifier,
					"version_identifier": payload.VersionID,
					"instance_url":       payload.AppURL,
				},
				ExpectedStatus: 200,
			},
		},
	).Get(httpCtx, &storeResult); err != nil {
		return err
	}

	body, ok := storeResult.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		return workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
			"missing body in store result",
			storeResult.Output,
		)
	}
	resultURLs := workflowengine.AsSliceOfStrings(body["result_urls"])
	frameURLs := workflowengine.AsSliceOfStrings(body["screenshot_urls"])
	if len(resultURLs) == 0 || len(frameURLs) == 0 {
		return workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
			"missing result or screenshot URLs",
			storeResult.Output,
		)
	}

	if output != nil {
		if *output == nil {
			*output = make(map[string]any)
		}
		existingVideos, _ := (*output)["result_video_urls"].([]string)
		(*output)["result_video_urls"] = append(existingVideos, resultURLs...)
		existingScreens, _ := (*output)["screenshot_urls"].([]string)
		(*output)["screenshot_urls"] = append(existingScreens, frameURLs...)
	}

	return nil
}

func cleanupSpecForEmulator(serial, cloneName string) CleanupStepSpec {
	return CleanupStepSpec{
		Name:           fmt.Sprintf("%s:%s", cleanupStepStopEmulator, serial),
		Type:           cleanupStepStopEmulator,
		Payload:        map[string]any{"emulator_serial": serial, "clone_name": cloneName},
		MaxRetries:     3,
		Idempotent:     true,
		TimeoutSeconds: int((2 * time.Minute).Seconds()),
	}
}

func cleanupSpecForRecording(payload StopRecordingCleanupPayload) CleanupStepSpec {
	return CleanupStepSpec{
		Name:           fmt.Sprintf("%s:%s", cleanupStepStopRecording, payload.EmulatorSerial),
		Type:           cleanupStepStopRecording,
		Payload:        payload,
		MaxRetries:     3,
		Idempotent:     true,
		TimeoutSeconds: int((35 * time.Minute).Seconds()),
	}
}

func validateCleanupSpecs(specs []CleanupStepSpec) error {
	if len(specs) == 0 {
		return nil
	}
	for _, spec := range specs {
		if spec.Name == "" {
			return errors.New("cleanup step missing name")
		}
		stepType := spec.Type
		if stepType == "" {
			stepType = cleanupStepType(spec.Name)
		}
		if stepType == "" {
			return fmt.Errorf("cleanup step %s has unknown type", spec.Name)
		}
	}
	return nil
}

func recordFailedCleanup(
	ctx workflow.Context,
	options CleanupActivityOptions,
	spec CleanupStepSpec,
	stepErr error,
	attempts int,
	workflowID string,
) error {
	pipelineCtx := workflow.WithActivityOptions(ctx, options.Pipeline)
	recordActivity := activities.NewRecordFailedCleanupActivity()
	payload := activities.RecordFailedCleanupPayload{
		WorkflowID: workflowID,
		StepName:   spec.Name,
		Error:      stepErr.Error(),
		RetryCount: attempts,
		Payload:    spec.Payload,
	}
	return workflow.ExecuteActivity(
		pipelineCtx,
		recordActivity.Name(),
		workflowengine.ActivityInput{Payload: payload},
	).Get(pipelineCtx, nil)
}

func startCleanupVerificationWorkflow(
	ctx workflow.Context,
	specs []CleanupStepSpec,
	appURL string,
	runData map[string]any,
) error {
	if len(specs) == 0 {
		return nil
	}
	info := workflow.GetInfo(ctx)
	runIdentifier, _ := runData["run_identifier"].(string)

	payload := CleanupVerificationPayload{
		WorkflowID:    info.WorkflowExecution.ID,
		RunID:         info.WorkflowExecution.RunID,
		RunIdentifier: runIdentifier,
		DelaySeconds:  int(time.Minute.Seconds()),
		StepSpecs:     specs,
	}

	childOpts := workflow.ChildWorkflowOptions{
		WorkflowID: fmt.Sprintf("cleanup-verify-%s-%s", info.WorkflowExecution.ID, info.WorkflowExecution.RunID),
		TaskQueue:  PipelineTaskQueue,
	}
	childCtx := workflow.WithChildOptions(ctx, childOpts)
	future := workflow.ExecuteChildWorkflow(
		childCtx,
		NewCleanupVerificationWorkflow().Name(),
		workflowengine.WorkflowInput{
			Payload: payload,
			Config: map[string]any{
				"app_url": appURL,
			},
		},
	)

	var execution workflow.Execution
	return future.GetChildWorkflowExecution().Get(ctx, &execution)
}
