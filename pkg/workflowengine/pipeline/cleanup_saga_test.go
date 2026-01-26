// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type cleanupSagaTestInput struct {
	Specs          []CleanupStepSpec
	RecordFailures bool
}

type cleanupSagaTestResult struct {
	ErrorCount int
	Recorded   int
}

func cleanupSagaTestWorkflow(
	ctx workflow.Context,
	input cleanupSagaTestInput,
) (cleanupSagaTestResult, error) {
	logger := workflow.GetLogger(ctx)
	output := map[string]any{}
	options := buildCleanupOptions(workflow.ActivityOptions{
		StartToCloseTimeout:    time.Minute,
		ScheduleToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	})

	recorded := 0
	var recordFn func(ctx workflow.Context, spec CleanupStepSpec, stepErr error, attempts int) error
	if input.RecordFailures {
		recordFn = func(_ workflow.Context, _ CleanupStepSpec, _ error, _ int) error {
			recorded++
			return nil
		}
	}

	errs := executeCleanupSpecs(ctx, logger, options, input.Specs, &output, recordFn)
	return cleanupSagaTestResult{
		ErrorCount: len(errs),
		Recorded:   recorded,
	}, nil
}

func TestCleanupSagaExecutesInReverseOrder(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stopEmulator := activities.NewStopEmulatorActivity()
	stopRecording := activities.NewStopRecordingActivity()
	httpActivity := activities.NewHTTPActivity()

	order := []string{}

	env.RegisterActivityWithOptions(
		func(ctx context.Context, _ workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			order = append(order, cleanupStepStopRecording)
			return workflowengine.ActivityResult{Output: map[string]any{"last_frame_path": "/tmp/frame.png"}}, nil
		},
		activity.RegisterOptions{Name: stopRecording.Name()},
	)
	env.RegisterActivityWithOptions(
		func(ctx context.Context, _ workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{
				Output: map[string]any{
					"body": map[string]any{
						"result_urls":     []string{"video-url"},
						"screenshot_urls": []string{"frame-url"},
					},
				},
			}, nil
		},
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		func(ctx context.Context, _ workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			order = append(order, cleanupStepStopEmulator)
			return workflowengine.ActivityResult{Output: "ok"}, nil
		},
		activity.RegisterOptions{Name: stopEmulator.Name()},
	)

	env.RegisterWorkflow(cleanupSagaTestWorkflow)

	specs := []CleanupStepSpec{
		cleanupSpecForEmulator("emu-1", "clone-1"),
		cleanupSpecForRecording(StopRecordingCleanupPayload{
			EmulatorSerial:   "emu-1",
			AdbProcessPid:    1,
			FfmpegProcessPid: 2,
			LogcatProcessPid: 3,
			VideoPath:        "/tmp/video.mp4",
			RunIdentifier:    "run-1",
			VersionID:        "version-1",
			AppURL:           "https://example.test",
		}),
	}

	env.ExecuteWorkflow(cleanupSagaTestWorkflow, cleanupSagaTestInput{Specs: specs})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result cleanupSagaTestResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 0, result.ErrorCount)
	require.Equal(t, []string{cleanupStepStopRecording, cleanupStepStopEmulator}, order)
}

func TestCleanupSagaRetriesBeforeSuccess(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stopEmulator := activities.NewStopEmulatorActivity()
	attempts := 0
	env.RegisterActivityWithOptions(
		func(ctx context.Context, _ workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			attempts++
			if attempts < 3 {
				return workflowengine.ActivityResult{}, errors.New("transient failure")
			}
			return workflowengine.ActivityResult{Output: "ok"}, nil
		},
		activity.RegisterOptions{Name: stopEmulator.Name()},
	)

	env.RegisterWorkflow(cleanupSagaTestWorkflow)

	spec := cleanupSpecForEmulator("emu-2", "clone-2")
	spec.MaxRetries = 3

	env.ExecuteWorkflow(cleanupSagaTestWorkflow, cleanupSagaTestInput{Specs: []CleanupStepSpec{spec}})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result cleanupSagaTestResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 0, result.ErrorCount)
	require.Equal(t, 3, attempts)
}

func TestCleanupSagaContinuesAfterFailure(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stopEmulator := activities.NewStopEmulatorActivity()
	stopRecording := activities.NewStopRecordingActivity()
	httpActivity := activities.NewHTTPActivity()

	order := []string{}
	env.RegisterActivityWithOptions(
		func(ctx context.Context, _ workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			order = append(order, cleanupStepStopRecording)
			return workflowengine.ActivityResult{}, errors.New("upload failure")
		},
		activity.RegisterOptions{Name: stopRecording.Name()},
	)
	env.RegisterActivityWithOptions(
		func(ctx context.Context, _ workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			order = append(order, cleanupStepStopEmulator)
			return workflowengine.ActivityResult{Output: "ok"}, nil
		},
		activity.RegisterOptions{Name: stopEmulator.Name()},
	)
	env.RegisterActivityWithOptions(
		func(ctx context.Context, _ workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{Output: map[string]any{}}, nil
		},
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	env.RegisterWorkflow(cleanupSagaTestWorkflow)

	recordSpec := cleanupSpecForRecording(StopRecordingCleanupPayload{
		EmulatorSerial:   "emu-3",
		AdbProcessPid:    1,
		FfmpegProcessPid: 2,
		LogcatProcessPid: 3,
		VideoPath:        "/tmp/video.mp4",
		RunIdentifier:    "run-3",
		VersionID:        "version-3",
		AppURL:           "https://example.test",
	})
	recordSpec.MaxRetries = 1
	stopSpec := cleanupSpecForEmulator("emu-3", "clone-3")
	stopSpec.MaxRetries = 1

	env.ExecuteWorkflow(cleanupSagaTestWorkflow, cleanupSagaTestInput{
		Specs:          []CleanupStepSpec{stopSpec, recordSpec},
		RecordFailures: true,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result cleanupSagaTestResult
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, 1, result.ErrorCount)
	require.Equal(t, 1, result.Recorded)
	require.Equal(t, []string{cleanupStepStopRecording, cleanupStepStopEmulator}, order)
}
