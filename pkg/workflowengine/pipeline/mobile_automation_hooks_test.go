// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// TestMobileAutomationSetupHookSkipsPermitsWhenSemaphoreManaged ensures queue-managed runs continue.
func TestMobileAutomationSetupHookSkipsPermitsWhenSemaphoreManaged(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testSetupHookWorkflow,
		workflow.RegisterOptions{Name: "test-setup-hook"},
	)

	httpActivity := activities.NewHTTPActivity()
	installActivity := activities.NewApkInstallActivity()
	recordActivity := activities.NewStartRecordingActivity()

	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		installActivity.Execute,
		activity.RegisterOptions{Name: installActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		recordActivity.Execute,
		activity.RegisterOptions{Name: recordActivity.Name()},
	)

	mockSetupHookActivities(env, httpActivity, installActivity, recordActivity)

	env.ExecuteWorkflow("test-setup-hook", mobileAutomationSetupSteps(), true)

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

// TestMobileAutomationSetupHookFailsWithoutSemaphoreMetadata verifies non-semaphore runs fail fast.
func TestMobileAutomationSetupHookFailsWithoutSemaphoreMetadata(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testSetupHookWorkflow,
		workflow.RegisterOptions{Name: "test-setup-hook"},
	)

	env.ExecuteWorkflow("test-setup-hook", mobileAutomationSetupSteps(), false)

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "mobile-runner pipelines must be started via queue/semaphore")
}

func TestProcessStepFailsWithoutPermit(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testProcessStepWithoutPermitWorkflow,
		workflow.RegisterOptions{Name: "test-missing-permit"},
	)

	env.ExecuteWorkflow("test-missing-permit")

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing runner permit")
}

func TestMobileAutomationCleanupReleasesPermitsOnFailure(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testCleanupReleasesPermitsWorkflow,
		workflow.RegisterOptions{Name: "test-cleanup-release"},
	)

	cleanupActivity := activities.NewCleanupDeviceActivity()
	releaseActivity := activities.NewReleaseMobileRunnerPermitActivity()
	env.RegisterActivityWithOptions(
		cleanupActivity.Execute,
		activity.RegisterOptions{Name: cleanupActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		releaseActivity.Execute,
		activity.RegisterOptions{Name: releaseActivity.Name()},
	)

	env.OnActivity(cleanupActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, errors.New("cleanup failed"))
	env.OnActivity(releaseActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, nil).
		Times(2)

	env.ExecuteWorkflow("test-cleanup-release")

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "mobile automation cleanup")
	env.AssertExpectations(t)
}

func TestMobileAutomationCleanupSkipsPermitsWhenSemaphoreManaged(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testCleanupSkipsPermitsWorkflow,
		workflow.RegisterOptions{Name: "test-cleanup-skip"},
	)

	cleanupActivity := activities.NewCleanupDeviceActivity()

	env.RegisterActivityWithOptions(
		cleanupActivity.Execute,
		activity.RegisterOptions{Name: cleanupActivity.Name()},
	)

	env.OnActivity(cleanupActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, nil).
		Once()

	env.ExecuteWorkflow("test-cleanup-skip")

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestStartRecordingForDeviceMissingSerial(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testStartRecordingMissingSerialWorkflow,
		workflow.RegisterOptions{Name: "test-start-recording-missing-serial"},
	)

	env.ExecuteWorkflow("test-start-recording-missing-serial")

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing serial")
}

func TestStartRecordingForDevicesSkipsAlreadyRecording(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testStartRecordingSkipWorkflow,
		workflow.RegisterOptions{Name: "test-start-recording-skip"},
	)

	env.ExecuteWorkflow("test-start-recording-skip")

	require.NoError(t, env.GetWorkflowError())
}

func TestStartRecordingForDeviceSuccess(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testStartRecordingSuccessWorkflow,
		workflow.RegisterOptions{Name: "test-start-recording-success"},
	)

	recordActivity := activities.NewStartRecordingActivity()
	env.RegisterActivityWithOptions(
		recordActivity.Execute,
		activity.RegisterOptions{Name: recordActivity.Name()},
	)

	env.OnActivity(recordActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"adb_process_pid":    float64(1),
			"ffmpeg_process_pid": float64(2),
			"logcat_process_pid": float64(3),
			"video_path":         "/tmp/video.mp4",
			"logcat_path":        "/tmp/logcat.txt",
		}}, nil)

	env.ExecuteWorkflow("test-start-recording-success")

	require.NoError(t, env.GetWorkflowError())

	var device map[string]any
	require.NoError(t, env.GetWorkflowResult(&device))
	require.Equal(t, true, device["recording"])
	require.Equal(t, "/tmp/video.mp4", device["video_path"])
}

func TestCleanupRecordingSuccess(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testCleanupRecordingWorkflow,
		workflow.RegisterOptions{Name: "test-cleanup-recording"},
	)

	stopActivity := activities.NewStopRecordingActivity()
	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		stopActivity.Execute,
		activity.RegisterOptions{Name: stopActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	env.OnActivity(stopActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"last_frame_path": "/tmp/last.png",
		}}, nil)
	env.OnActivity(httpActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"body": map[string]any{
				"result_urls":     []string{"url1"},
				"screenshot_urls": []string{"shot1"},
			},
		}}, nil)

	env.ExecuteWorkflow("test-cleanup-recording")
	require.NoError(t, env.GetWorkflowError())

	var out map[string]any
	require.NoError(t, env.GetWorkflowResult(&out))
	require.Contains(t, workflowengine.AsSliceOfStrings(out["result_video_urls"]), "url1")
}

func TestCleanupRecordingMissingRunnerURL(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testCleanupRecordingMissingRunnerWorkflow,
		workflow.RegisterOptions{Name: "test-cleanup-recording-missing-runner"},
	)

	env.ExecuteWorkflow("test-cleanup-recording-missing-runner")
	require.NoError(t, env.GetWorkflowError())

	var errs int
	require.NoError(t, env.GetWorkflowResult(&errs))
	require.Equal(t, 1, errs)
}

func TestStopRecordingMissingLastFrame(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		testStopRecordingMissingLastFrameWorkflow,
		workflow.RegisterOptions{Name: "test-stop-recording-missing"},
	)

	stopActivity := activities.NewStopRecordingActivity()
	env.RegisterActivityWithOptions(
		stopActivity.Execute,
		activity.RegisterOptions{Name: stopActivity.Name()},
	)
	env.OnActivity(stopActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{}}, nil)

	env.ExecuteWorkflow("test-stop-recording-missing")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing last_frame_path")
}

func testProcessStepWithoutPermitWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{StartToCloseTimeout: time.Second},
	)
	step := StepDefinition{
		StepSpec: StepSpec{
			ID:  "step-1",
			Use: "mobile-automation",
			With: StepInputs{
				Payload: map[string]any{
					"runner_id": "runner-1",
					"action_id": "action-1",
				},
			},
		},
	}
	config := map[string]any{"app_url": "https://example.test"}
	settedDevices := map[string]any{}
	runData := map[string]any{}
	activityOptions := workflow.ActivityOptions{StartToCloseTimeout: time.Second}

	return processStep(
		processStepInput{
			ctx:              ctx,
			step:             &step,
			config:           config,
			ao:               &activityOptions,
			settedDevices:    settedDevices,
			runData:          &runData,
			httpActivity:     activities.NewHTTPActivity(),
			startEmuActivity: activities.NewStartEmulatorActivity(),
			installActivity:  activities.NewApkInstallActivity(),
			logger:           workflow.GetLogger(ctx),
			globalRunnerID:   "",
		},
	)
}

func testSetupHookWorkflow(
	ctx workflow.Context,
	steps []StepDefinition,
	semaphoreManaged bool,
) error {
	activityOptions := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
	config := map[string]any{"app_url": "https://example.test"}
	if semaphoreManaged {
		config["mobile_runner_semaphore_ticket_id"] = "ticket-1"
	}
	runData := map[string]any{}

	return MobileAutomationSetupHook(ctx, &steps, &activityOptions, config, &runData)
}

func testStartRecordingMissingSerialWorkflow(ctx workflow.Context) error {
	activityOptions := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
	return startRecordingForDevice(startRecordingForDeviceInput{
		ctx:            ctx,
		runnerID:       "runner-1",
		deviceMap:      map[string]any{},
		ao:             &activityOptions,
		recordActivity: activities.NewStartRecordingActivity(),
	})
}

func testStartRecordingSkipWorkflow(ctx workflow.Context) error {
	activityOptions := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
	setted := map[string]any{
		"runner-1": map[string]any{
			"serial":    "serial-1",
			"recording": true,
		},
	}
	return startRecordingForDevices(startRecordingForDevicesInput{
		ctx:            ctx,
		settedDevices:  setted,
		ao:             &activityOptions,
		recordActivity: activities.NewStartRecordingActivity(),
	})
}

func testStartRecordingSuccessWorkflow(ctx workflow.Context) (map[string]any, error) {
	activityOptions := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
	device := map[string]any{
		"serial": "serial-1",
	}
	err := startRecordingForDevice(startRecordingForDeviceInput{
		ctx:            ctx,
		runnerID:       "runner-1",
		deviceMap:      device,
		ao:             &activityOptions,
		recordActivity: activities.NewStartRecordingActivity(),
	})
	return device, err
}

func testCleanupRecordingWorkflow(ctx workflow.Context) (map[string]any, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Second})
	output := map[string]any{
		"result_video_urls": []string{},
		"screenshot_urls":   []string{},
	}
	errs := []error{}
	deviceInfo := map[string]any{
		"runner_url":           "https://runner",
		"recording":            true,
		"video_path":           "/tmp/video.mp4",
		"logcat_path":          "/tmp/logcat.txt",
		"recording_adb_pid":    1,
		"recording_ffmpeg_pid": 2,
		"recording_logcat_pid": 3,
	}
	cleanupRecording(cleanupRecordingInput{
		ctx:         ctx,
		runnerID:    "runner-1",
		deviceInfo:  deviceInfo,
		runID:       "run-1",
		output:      &output,
		cleanupErrs: &errs,
		appURL:      "https://app",
	})
	if len(errs) > 0 {
		return output, errs[0]
	}
	return output, nil
}

func testCleanupRecordingMissingRunnerWorkflow(ctx workflow.Context) (int, error) {
	errs := []error{}
	deviceInfo := map[string]any{
		"recording": true,
	}
	output := map[string]any{
		"result_video_urls": []string{},
		"screenshot_urls":   []string{},
	}
	cleanupRecording(cleanupRecordingInput{
		ctx:         ctx,
		runnerID:    "runner-1",
		deviceInfo:  deviceInfo,
		runID:       "run-1",
		output:      &output,
		cleanupErrs: &errs,
		appURL:      "https://app",
	})
	return len(errs), nil
}

func testStopRecordingMissingLastFrameWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Second})
	info := &recordingInfo{
		videoPath:  "/tmp/video.mp4",
		logcatPath: "/tmp/logcat.txt",
		adbPid:     1,
		ffmpegPid:  2,
		logcatPid:  3,
	}
	_, err := stopRecording(ctx, info, workflow.GetLogger(ctx))
	return err
}

func testCleanupReleasesPermitsWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{StartToCloseTimeout: time.Second},
	)
	config := map[string]any{"app_url": "https://example.test"}
	output := map[string]any{}
	runData := map[string]any{
		"setted_devices": map[string]any{
			"runner-1": map[string]any{
				"serial":     "serial-1",
				"clone_name": "clone-1",
				"installed":  map[string]string{},
				"runner_url": "https://runner.test",
				"recording":  false,
			},
		},
		"run_identifier": "run-1",
		"mobile_runner_permits": map[string]workflows.MobileRunnerSemaphorePermit{
			"runner-1": {RunnerID: "runner-1", LeaseID: "lease-1"},
			"runner-2": {RunnerID: "runner-2", LeaseID: "lease-2"},
		},
	}
	steps := []StepDefinition{}
	activityOptions := workflow.ActivityOptions{StartToCloseTimeout: time.Second}

	return MobileAutomationCleanupHook(ctx, steps, &activityOptions, config, runData, &output)
}

func testCleanupSkipsPermitsWorkflow(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{StartToCloseTimeout: time.Second},
	)
	config := map[string]any{
		"app_url":                           "https://example.test",
		"mobile_runner_semaphore_ticket_id": "ticket-1",
	}
	output := map[string]any{}
	runData := map[string]any{
		"setted_devices": map[string]any{
			"runner-1": map[string]any{
				"serial":     "serial-1",
				"clone_name": "clone-1",
				"installed":  map[string]string{},
				"runner_url": "https://runner.test",
				"recording":  false,
			},
		},
		"run_identifier": "run-1",
		"mobile_runner_permits": map[string]workflows.MobileRunnerSemaphorePermit{
			"runner-1": {RunnerID: "runner-1", LeaseID: "lease-1"},
		},
	}
	steps := []StepDefinition{}
	activityOptions := workflow.ActivityOptions{StartToCloseTimeout: time.Second}

	return MobileAutomationCleanupHook(ctx, steps, &activityOptions, config, runData, &output)
}

func mobileAutomationSetupSteps() []StepDefinition {
	return []StepDefinition{
		{
			StepSpec: StepSpec{
				ID:  "step-1",
				Use: "mobile-automation",
				With: StepInputs{
					Payload: map[string]any{
						"runner_id": "runner-1",
						"action_id": "action-1",
					},
				},
			},
		},
	}
}

func mockSetupHookActivities(
	env *testsuite.TestWorkflowEnvironment,
	httpActivity *activities.HTTPActivity,
	installActivity *activities.ApkInstallActivity,
	recordActivity *activities.StartRecordingActivity,
) {
	env.OnActivity(httpActivity.Name(), mock.Anything, mock.Anything).
		Return(
			workflowengine.ActivityResult{
				Output: map[string]any{
					"body": map[string]any{
						"runner_url": "https://runner.test",
						"serial":     "serial-1",
					},
				},
			},
			nil,
		).
		Once()
	env.OnActivity(httpActivity.Name(), mock.Anything, mock.Anything).
		Return(
			workflowengine.ActivityResult{
				Output: map[string]any{
					"body": map[string]any{
						"apk_path":   "/tmp/app.apk",
						"version_id": "version-1",
						"code":       "action-code",
					},
				},
			},
			nil,
		).
		Once()
	env.OnActivity(installActivity.Name(), mock.Anything, mock.Anything).
		Return(
			workflowengine.ActivityResult{
				Output: map[string]any{
					"package_id": "package-1",
				},
			},
			nil,
		).
		Once()
	env.OnActivity(recordActivity.Name(), mock.Anything, mock.Anything).
		Return(
			workflowengine.ActivityResult{
				Output: map[string]any{
					"video_path":         "/tmp/video.mp4",
					"logcat_path":        "/tmp/logcat.txt",
					"adb_process_pid":    float64(1),
					"ffmpeg_process_pid": float64(2),
					"logcat_process_pid": float64(3),
				},
			},
			nil,
		).
		Once()
}
