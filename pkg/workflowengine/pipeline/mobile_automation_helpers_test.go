// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
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

func TestDecodeAndValidatePayload(t *testing.T) {
	_, err := decodeAndValidatePayload(
		&StepDefinition{
			StepSpec: StepSpec{ID: "step-1", With: StepInputs{Payload: map[string]any{}}},
		},
	)
	require.Error(t, err)

	_, err = decodeAndValidatePayload(
		&StepDefinition{StepSpec: StepSpec{ID: "step-2", With: StepInputs{Payload: map[string]any{
			"action_code": "code",
		}}}},
	)
	require.Error(t, err)

	_, err = decodeAndValidatePayload(
		&StepDefinition{StepSpec: StepSpec{ID: "step-3", With: StepInputs{Payload: map[string]any{
			"runner_id": "runner-1",
		}}}},
	)
	require.Error(t, err)

	payload, err := decodeAndValidatePayload(
		&StepDefinition{StepSpec: StepSpec{ID: "step-4", With: StepInputs{Payload: map[string]any{
			"action_id": "action-1",
			"runner_id": "runner-2",
		}}}},
	)
	require.NoError(t, err)
	require.Equal(t, "action-1", payload.ActionID)
	require.Equal(t, "runner-2", payload.RunnerID)
}

func TestCollectMobileRunnerIDs(t *testing.T) {
	steps := []StepDefinition{
		{StepSpec: StepSpec{Use: mobileAutomationStepUse, With: StepInputs{Payload: map[string]any{
			"action_id": "action-1",
			"runner_id": "runner-b",
		}}}},
		{StepSpec: StepSpec{Use: mobileAutomationStepUse, With: StepInputs{Payload: map[string]any{
			"action_id": "action-2",
			"runner_id": "runner-a",
		}}}},
	}

	runnerIDs, err := collectMobileRunnerIDs(steps, "runner-global")
	require.NoError(t, err)
	require.Equal(t, []string{"runner-a", "runner-b", "runner-global"}, runnerIDs)
}

func TestHasRunnerPermit(t *testing.T) {
	runData := map[string]any{
		"mobile_runner_permits": map[string]workflows.MobileRunnerSemaphorePermit{
			"runner-1": {RunnerID: "runner-1"},
		},
	}
	permit := hasRunnerPermit(&runData, "runner-1")
	require.True(t, permit)
	require.False(t, hasRunnerPermit(&runData, "runner-2"))
}

func TestGetRunnerPermits(t *testing.T) {
	runData := map[string]any{
		"mobile_runner_permits": map[string]any{
			"runner-1": map[string]any{"runner_id": "runner-1", "lease_id": "lease-1"},
			"bad":      "nope",
		},
	}
	permits := getRunnerPermits(runData)
	require.Len(t, permits, 1)
	require.Equal(t, "runner-1", permits["runner-1"].RunnerID)
}

func TestParseAPKResponse(t *testing.T) {
	result := workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"apk_path":   "path.apk",
			"version_id": "ver-1",
			"code":       "action-code",
		},
	}}
	payload := &workflows.MobileAutomationWorkflowPipelinePayload{ActionID: "action-1"}
	step := &StepDefinition{StepSpec: StepSpec{ID: "step-1"}}
	apkPath, versionID, actionCode, err := parseAPKResponse(result, payload, step)
	require.NoError(t, err)
	require.Equal(t, "path.apk", apkPath)
	require.Equal(t, "ver-1", versionID)
	require.Equal(t, "action-code", actionCode)

	badResult := workflowengine.ActivityResult{Output: map[string]any{"body": map[string]any{}}}
	apkPath, versionID, actionCode, err = parseAPKResponse(badResult, payload, step)
	require.Error(t, err)
	require.Empty(t, apkPath)
	require.Empty(t, versionID)
	require.Empty(t, actionCode)
}

func TestGetOrCreateSettedDevices(t *testing.T) {
	runData := map[string]any{}
	devices := getOrCreateSettedDevices(&runData)
	require.NotNil(t, devices)
	require.Empty(t, devices)

	runData["setted_devices"] = map[string]any{"runner": map[string]any{"serial": "1"}}
	restored := getOrCreateSettedDevices(&runData)
	require.Contains(t, restored, "runner")
}

func TestIsSemaphoreManagedRun(t *testing.T) {
	require.False(t, isSemaphoreManagedRun(nil))
	require.False(t, isSemaphoreManagedRun(map[string]any{}))
	require.True(
		t,
		isSemaphoreManagedRun(map[string]any{mobileRunnerSemaphoreTicketIDConfigKey: "ticket"}),
	)
}

func TestParseDeviceMap(t *testing.T) {
	_, err := parseDeviceMap("runner-1", "bad")
	require.Error(t, err)

	deviceMap, err := parseDeviceMap("runner-1", map[string]any{"serial": "abc"})
	require.NoError(t, err)
	require.Equal(t, "abc", deviceMap["serial"])
}

func TestValidateRunnerIDConfigurationAdditional(t *testing.T) {
	steps := []StepDefinition{
		{
			StepSpec: StepSpec{
				Use: mobileAutomationStepUse,
				With: StepInputs{
					Payload: map[string]any{
						"action_id": "action-1",
					},
				},
			},
		},
	}
	err := validateRunnerIDConfiguration(&steps, "")
	require.Error(t, err)

	err = validateRunnerIDConfiguration(&steps, "global-runner")
	require.NoError(t, err)

	steps[0].With.Payload["runner_id"] = "runner-1"
	err = validateRunnerIDConfiguration(&steps, "")
	require.NoError(t, err)

	nonMobile := []StepDefinition{{StepSpec: StepSpec{Use: "rest"}}}
	err = validateRunnerIDConfiguration(&nonMobile, "")
	require.NoError(t, err)
}

func TestExtractAndStoreRecordingInfoAdditional(t *testing.T) {
	deviceMap := map[string]any{}
	err := extractAndStoreRecordingInfo(
		workflowengine.ActivityResult{Output: map[string]any{}},
		deviceMap,
		"runner-1",
	)
	require.Error(t, err)

	okResult := workflowengine.ActivityResult{Output: map[string]any{
		"adb_process_pid":    float64(11),
		"ffmpeg_process_pid": float64(12),
		"logcat_process_pid": float64(13),
		"video_path":         "/tmp/video.mp4",
		"logcat_path":        "/tmp/logcat.txt",
	}}
	err = extractAndStoreRecordingInfo(okResult, deviceMap, "runner-1")
	require.NoError(t, err)
	require.Equal(t, true, deviceMap["recording"])
	require.Equal(t, "/tmp/video.mp4", deviceMap["video_path"])
}

func TestExtractDeviceInfoAdditional(t *testing.T) {
	_, _, _, err := extractDeviceInfo("runner-1", map[string]any{})
	require.Error(t, err)

	_, _, _, err = extractDeviceInfo("runner-1", map[string]any{"serial": "s1"})
	require.Error(t, err)

	serial, clone, packages, err := extractDeviceInfo("runner-1", map[string]any{
		"serial":     "s1",
		"clone_name": "clone",
		"installed":  map[string]string{"v1": "pkg1", "v2": ""},
	})
	require.NoError(t, err)
	require.Equal(t, "s1", serial)
	require.Equal(t, "clone", clone)
	require.Equal(t, []string{"pkg1"}, packages)
}

func TestExtractRecordingInfoAdditional(t *testing.T) {
	_, err := extractRecordingInfo("runner-1", map[string]any{})
	require.Error(t, err)

	info, err := extractRecordingInfo("runner-1", map[string]any{
		"video_path":           "/tmp/video.mp4",
		"logcat_path":          "/tmp/logcat.txt",
		"recording_adb_pid":    1,
		"recording_ffmpeg_pid": 2,
		"recording_logcat_pid": 3,
	})
	require.NoError(t, err)
	require.Equal(t, "/tmp/video.mp4", info.videoPath)
	require.Equal(t, 1, info.adbPid)
}

func TestExtractAndStoreURLsAdditional(t *testing.T) {
	output := map[string]any{
		"result_video_urls": []string{},
		"screenshot_urls":   []string{},
	}
	err := extractAndStoreURLs(workflowengine.ActivityResult{Output: map[string]any{}}, &output)
	require.Error(t, err)

	err = extractAndStoreURLs(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"result_urls":     []string{"url1"},
			"screenshot_urls": []string{"shot1"},
		},
	}}, &output)
	require.NoError(t, err)
	require.Contains(t, output["result_video_urls"].([]string), "url1")
	require.Contains(t, output["screenshot_urls"].([]string), "shot1")
}

func TestGetOrCreateDeviceMapExisting(t *testing.T) {
	setted := map[string]any{
		"runner-1": map[string]any{"serial": "serial-1"},
	}
	payload := &workflows.MobileAutomationWorkflowPipelinePayload{RunnerID: "runner-1"}
	device, err := getOrCreateDeviceMap(getOrCreateDeviceMapInput{
		payload:       payload,
		settedDevices: setted,
	})
	require.NoError(t, err)
	require.Equal(t, "serial-1", device["serial"])
}

func TestGetOrCreateDeviceMapUsesRunnerSerial(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	startEmuActivity := activities.NewStartEmulatorActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		startEmuActivity.Execute,
		activity.RegisterOptions{Name: startEmuActivity.Name()},
	)

	workflowName := "get-device-map-serial"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{RunnerID: "runner-1"}
			return getOrCreateDeviceMap(getOrCreateDeviceMapInput{
				ctx:              ctx,
				mobileCtx:        ctx,
				payload:          payload,
				settedDevices:    map[string]any{},
				appURL:           "http://localhost:8090",
				stepID:           "step-1",
				httpActivity:     httpActivity,
				startEmuActivity: startEmuActivity,
			})
		},
		workflow.RegisterOptions{Name: workflowName},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"runner_url": "http://runner",
			"serial":     "serial-1",
		},
	}}, nil)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "serial-1", result["serial"])
	require.Equal(t, "http://runner", result["runner_url"])
}

func TestGetOrCreateDeviceMapStartsEmulator(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	startEmuActivity := activities.NewStartEmulatorActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		startEmuActivity.Execute,
		activity.RegisterOptions{Name: startEmuActivity.Name()},
	)

	workflowName := "get-device-map-emulator"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{RunnerID: "runner-1"}
			return getOrCreateDeviceMap(getOrCreateDeviceMapInput{
				ctx:              ctx,
				mobileCtx:        ctx,
				payload:          payload,
				settedDevices:    map[string]any{},
				appURL:           "http://localhost:8090",
				stepID:           "step-1",
				httpActivity:     httpActivity,
				startEmuActivity: startEmuActivity,
			})
		},
		workflow.RegisterOptions{Name: workflowName},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"runner_url": "http://runner",
			"serial":     "",
		},
	}}, nil)

	env.OnActivity(
		startEmuActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"serial":     "emu-1",
		"clone_name": "clone-1",
	}}, nil)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "emu-1", result["serial"])
	require.Equal(t, "clone-1", result["clone_name"])
}

func TestExtractDeviceInfo(t *testing.T) {
	_, _, _, err := extractDeviceInfo("runner-1", map[string]any{})
	require.Error(t, err)

	_, _, _, err = extractDeviceInfo("runner-1", map[string]any{
		"serial": "serial-1",
	})
	require.Error(t, err)

	serial, clone, packages, err := extractDeviceInfo("runner-1", map[string]any{
		"serial":     "serial-1",
		"clone_name": "clone-1",
		"installed": map[string]string{
			"app": "com.example",
			"bad": "",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "serial-1", serial)
	require.Equal(t, "clone-1", clone)
	require.Equal(t, []string{"com.example"}, packages)
}

func TestFetchRunnerInfo(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	workflowName := "fetch-runner-info"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{RunnerID: "runner-1"}
			runnerURL, serial, err := fetchRunnerInfo(fetchRunnerInfoInput{
				ctx:          ctx,
				payload:      payload,
				appURL:       "http://localhost:8090",
				stepID:       "step-1",
				httpActivity: httpActivity,
			})
			if err != nil {
				return nil, err
			}
			return map[string]any{"runner_url": runnerURL, "serial": serial}, nil
		},
		workflow.RegisterOptions{Name: workflowName},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"runner_url": "http://runner",
			"serial":     "serial-1",
		},
	}}, nil)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "http://runner", result["runner_url"])
	require.Equal(t, "serial-1", result["serial"])
}

func TestFetchRunnerInfoErrors(t *testing.T) {
	tests := []struct {
		name      string
		body      any
		errSubstr string
	}{
		{
			name:      "invalid body type",
			body:      "bad",
			errSubstr: "invalid HTTP response format",
		},
		{
			name: "missing runner url",
			body: map[string]any{
				"serial": "serial-1",
			},
			errSubstr: "runner_url",
		},
		{
			name: "invalid serial",
			body: map[string]any{
				"runner_url": "http://runner",
				"serial":     123,
			},
			errSubstr: "invalid device serial",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()

			httpActivity := activities.NewHTTPActivity()
			env.RegisterActivityWithOptions(
				httpActivity.Execute,
				activity.RegisterOptions{Name: httpActivity.Name()},
			)

			workflowName := "fetch-runner-info-error"
			env.RegisterWorkflowWithOptions(
				func(ctx workflow.Context) error {
					ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
					ctx = workflow.WithActivityOptions(ctx, ao)
					payload := &workflows.MobileAutomationWorkflowPipelinePayload{RunnerID: "runner-1"}
					_, _, err := fetchRunnerInfo(fetchRunnerInfoInput{
						ctx:          ctx,
						payload:      payload,
						appURL:       "http://localhost:8090",
						stepID:       "step-1",
						httpActivity: httpActivity,
					})
					return err
				},
				workflow.RegisterOptions{Name: workflowName},
			)

			env.OnActivity(
				httpActivity.Name(),
				mock.Anything,
				mock.Anything,
			).Return(workflowengine.ActivityResult{Output: map[string]any{"body": tc.body}}, nil)

			env.ExecuteWorkflow(workflowName)
			require.True(t, env.IsWorkflowCompleted())
			err := env.GetWorkflowError()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errSubstr)
		})
	}
}

func TestStartEmulator(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	startEmuActivity := activities.NewStartEmulatorActivity()
	env.RegisterActivityWithOptions(
		startEmuActivity.Execute,
		activity.RegisterOptions{Name: startEmuActivity.Name()},
	)

	workflowName := "start-emulator"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{RunnerID: "runner-1"}
			cloneName, serial, err := startEmulator(startEmulatorInput{
				ctx:              ctx,
				mobileCtx:        ctx,
				payload:          payload,
				stepID:           "step-1",
				startEmuActivity: startEmuActivity,
			})
			if err != nil {
				return nil, err
			}
			return map[string]any{"serial": serial, "clone_name": cloneName}, nil
		},
		workflow.RegisterOptions{Name: workflowName},
	)

	env.OnActivity(
		startEmuActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"serial":     "emu-1",
		"clone_name": "clone-1",
	}}, nil)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "emu-1", result["serial"])
	require.Equal(t, "clone-1", result["clone_name"])
}

func TestStartEmulatorErrors(t *testing.T) {
	tests := []struct {
		name      string
		output    map[string]any
		errSubstr string
	}{
		{
			name:      "missing serial",
			output:    map[string]any{"clone_name": "clone-1"},
			errSubstr: "missing serial",
		},
		{
			name:      "missing clone_name",
			output:    map[string]any{"serial": "emu-1"},
			errSubstr: "missing clone_name",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()

			startEmuActivity := activities.NewStartEmulatorActivity()
			env.RegisterActivityWithOptions(
				startEmuActivity.Execute,
				activity.RegisterOptions{Name: startEmuActivity.Name()},
			)

			workflowName := "start-emulator-error"
			env.RegisterWorkflowWithOptions(
				func(ctx workflow.Context) error {
					ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
					ctx = workflow.WithActivityOptions(ctx, ao)
					payload := &workflows.MobileAutomationWorkflowPipelinePayload{RunnerID: "runner-1"}
					_, _, err := startEmulator(startEmulatorInput{
						ctx:              ctx,
						mobileCtx:        ctx,
						payload:          payload,
						stepID:           "step-1",
						startEmuActivity: startEmuActivity,
					})
					return err
				},
				workflow.RegisterOptions{Name: workflowName},
			)

			env.OnActivity(
				startEmuActivity.Name(),
				mock.Anything,
				mock.Anything,
			).Return(workflowengine.ActivityResult{Output: tc.output}, nil)

			env.ExecuteWorkflow(workflowName)
			require.True(t, env.IsWorkflowCompleted())
			err := env.GetWorkflowError()
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.errSubstr)
		})
	}
}

func TestExtractRecordingInfo(t *testing.T) {
	_, err := extractRecordingInfo("runner-1", map[string]any{})
	require.Error(t, err)

	_, err = extractRecordingInfo("runner-1", map[string]any{
		"video_path": "video.mp4",
	})
	require.Error(t, err)

	_, err = extractRecordingInfo("runner-1", map[string]any{
		"video_path":  "video.mp4",
		"logcat_path": "logcat.txt",
	})
	require.Error(t, err)

	info, err := extractRecordingInfo("runner-1", map[string]any{
		"video_path":           "video.mp4",
		"logcat_path":          "logcat.txt",
		"recording_adb_pid":    1,
		"recording_ffmpeg_pid": 2,
		"recording_logcat_pid": 3,
	})
	require.NoError(t, err)
	require.Equal(t, "video.mp4", info.videoPath)
	require.Equal(t, "logcat.txt", info.logcatPath)
	require.Equal(t, 1, info.adbPid)
	require.Equal(t, 2, info.ffmpegPid)
	require.Equal(t, 3, info.logcatPid)
}

func TestExtractAndStoreURLs(t *testing.T) {
	output := map[string]any{
		"result_video_urls": []string{},
		"screenshot_urls":   []string{},
	}

	require.Panics(t, func() {
		_ = extractAndStoreURLs(workflowengine.ActivityResult{Output: "bad"}, &output)
	})

	err := extractAndStoreURLs(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{},
	}}, &output)
	require.Error(t, err)

	err = extractAndStoreURLs(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"result_urls":     []string{"video-1"},
			"screenshot_urls": []string{"frame-1"},
		},
	}}, &output)
	require.NoError(t, err)
	require.Equal(t, []string{"video-1"}, output["result_video_urls"])
	require.Equal(t, []string{"frame-1"}, output["screenshot_urls"])
}
