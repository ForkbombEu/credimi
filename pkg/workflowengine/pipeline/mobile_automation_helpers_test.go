// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
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
		&pipeline.StepDefinition{
			StepSpec: pipeline.StepSpec{ID: "step-1", With: pipeline.StepInputs{Payload: map[string]any{}}},
		},
	)
	require.Error(t, err)

	_, err = decodeAndValidatePayload(
		&pipeline.StepDefinition{StepSpec: pipeline.StepSpec{ID: "step-2", With: pipeline.StepInputs{Payload: map[string]any{
			"action_code": "code",
		}}}},
	)
	require.Error(t, err)

	_, err = decodeAndValidatePayload(
		&pipeline.StepDefinition{StepSpec: pipeline.StepSpec{ID: "step-3", With: pipeline.StepInputs{Payload: map[string]any{
			"runner_id": "runner-1",
		}}}},
	)
	require.Error(t, err)

	payload, err := decodeAndValidatePayload(
		&pipeline.StepDefinition{StepSpec: pipeline.StepSpec{ID: "step-4", With: pipeline.StepInputs{Payload: map[string]any{
			"action_id": "action-1",
			"runner_id": "runner-2",
		}}}},
	)
	require.NoError(t, err)
	require.Equal(t, "action-1", payload.ActionID)
	require.Equal(t, "runner-2", payload.RunnerID)
}

func TestCollectMobileRunnerIDs(t *testing.T) {
	steps := []pipeline.StepDefinition{
		{StepSpec: pipeline.StepSpec{Use: mobileAutomationStepUse, With: pipeline.StepInputs{Payload: map[string]any{
			"action_id": "action-1",
			"runner_id": "runner-b",
		}}}},
		{StepSpec: pipeline.StepSpec{Use: mobileAutomationStepUse, With: pipeline.StepInputs{Payload: map[string]any{
			"action_id": "action-2",
			"runner_id": "runner-a",
		}}}},
	}

	runnerIDs, err := collectMobileRunnerIDs(steps, "runner-global")
	require.NoError(t, err)
	require.Equal(t, []string{"runner-a", "runner-b", "runner-global"}, runnerIDs)
}

func TestCollectMobileRunnerIDsNormalizesLeadingSlash(t *testing.T) {
	steps := []pipeline.StepDefinition{
		{StepSpec: pipeline.StepSpec{Use: mobileAutomationStepUse, With: pipeline.StepInputs{Payload: map[string]any{
			"action_id": "action-1",
			"runner_id": "/tenant-a/runner-b",
		}}}},
	}

	runnerIDs, err := collectMobileRunnerIDs(steps, "/tenant-a/runner-a")
	require.NoError(t, err)
	require.Equal(t, []string{"tenant-a/runner-a", "tenant-a/runner-b"}, runnerIDs)
}

func TestParseAPKResponse(t *testing.T) {
	result := workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"installer_path": "path.apk",
			"version_id":     "ver-1",
			"code":           "action-code",
		},
	}}
	payload := &workflows.MobileAutomationWorkflowPipelinePayload{ActionID: "action-1"}
	step := &pipeline.StepDefinition{StepSpec: pipeline.StepSpec{ID: "step-1"}}
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

func TestMobileRunnerTaskQueueNormalizesLeadingSlash(t *testing.T) {
	require.Equal(t, "tenant-a/runner-1-TaskQueue", mobileRunnerTaskQueue("/tenant-a/runner-1"))
}

func TestParseDeviceMap(t *testing.T) {
	_, err := parseDeviceMap("runner-1", "bad")
	require.Error(t, err)

	deviceMap, err := parseDeviceMap("runner-1", map[string]any{"serial": "abc"})
	require.NoError(t, err)
	require.Equal(t, "abc", deviceMap["serial"])
}

func TestValidateRunnerIDConfigurationAdditional(t *testing.T) {
	steps := []pipeline.StepDefinition{
		{
			StepSpec: pipeline.StepSpec{
				Use: mobileAutomationStepUse,
				With: pipeline.StepInputs{
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

	nonMobile := []pipeline.StepDefinition{{StepSpec: pipeline.StepSpec{Use: "rest"}}}
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
		"recording_process_pid": float64(11),
		"ffmpeg_process_pid":    float64(12),
		"log_process_pid":       float64(13),
		"video_path":            "/tmp/video.mp4",
		"log_path":              "/tmp/log.txt",
	}}
	err = extractAndStoreRecordingInfo(okResult, deviceMap, "runner-1")
	require.NoError(t, err)
	require.Equal(t, true, deviceMap["recording"])
	require.Equal(t, "/tmp/video.mp4", deviceMap["video_path"])
}

func TestExtractDeviceInfoAdditional(t *testing.T) {
	deviceType, serial, name, packages, err := extractDeviceInfo("runner-1", map[string]any{})
	_ = deviceType
	_ = serial
	_ = name
	_ = packages
	require.Error(t, err)

	deviceType, serial, name, packages, err = extractDeviceInfo(
		"runner-1",
		map[string]any{"serial": "s1"},
	)
	_ = deviceType
	_ = serial
	_ = name
	_ = packages
	require.Error(t, err)

	deviceType, serial, name, packages, err = extractDeviceInfo("runner-1", map[string]any{
		"type":      "emulator",
		"serial":    "s1",
		"name":      "emulator-1",
		"installed": map[string]string{"v1": "pkg1", "v2": ""},
	})
	require.NoError(t, err)
	require.Equal(t, "emulator", deviceType)
	require.Equal(t, "s1", serial)
	require.Equal(t, "emulator-1", name)
	require.Equal(t, []string{"pkg1"}, packages)
}

func TestExtractRecordingInfoAdditional(t *testing.T) {
	_, err := extractRecordingInfo("runner-1", map[string]any{})
	require.Error(t, err)

	info, err := extractRecordingInfo("runner-1", map[string]any{
		"video_path":            "/tmp/video.mp4",
		"log_path":              "/tmp/log.txt",
		"recording_process_pid": 1,
		"recording_ffmpeg_pid":  2,
		"recording_log_pid":     3,
	})
	require.NoError(t, err)
	require.Equal(t, "/tmp/video.mp4", info.videoPath)
	require.Equal(t, 1, info.recordingPid)
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
	listAppsActivity := activities.NewListInstalledAppsActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		startEmuActivity.Execute,
		activity.RegisterOptions{Name: startEmuActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		listAppsActivity.Execute,
		activity.RegisterOptions{Name: listAppsActivity.Name()},
	)

	workflowName := "get-device-map-serial"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{RunnerID: "runner-1"}
			return getOrCreateDeviceMap(getOrCreateDeviceMapInput{
				ctx:                       ctx,
				mobileCtx:                 ctx,
				payload:                   payload,
				settedDevices:             map[string]any{},
				appURL:                    "http://localhost:8090",
				stepID:                    "step-1",
				trackInitialInstalledApps: true,
				httpActivity:              httpActivity,
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
			"type":       "physical",
			"serial":     "serial-1",
		},
	}}, nil)
	env.OnActivity(listAppsActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: []string{"com.android.settings"}}, nil)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "serial-1", result["serial"])
	require.Equal(t, "android_phone", result["type"])
	require.Equal(t, "http://runner", result["runner_url"])
}

func TestGetOrCreateDeviceMapStartsEmulator(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	startEmuActivity := activities.NewStartEmulatorActivity()
	listAppsActivity := activities.NewListInstalledAppsActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		startEmuActivity.Execute,
		activity.RegisterOptions{Name: startEmuActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		listAppsActivity.Execute,
		activity.RegisterOptions{Name: listAppsActivity.Name()},
	)

	workflowName := "get-device-map-emulator"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{RunnerID: "runner-1"}
			return getOrCreateDeviceMap(getOrCreateDeviceMapInput{
				ctx:                       ctx,
				mobileCtx:                 ctx,
				payload:                   payload,
				settedDevices:             map[string]any{},
				appURL:                    "http://localhost:8090",
				stepID:                    "step-1",
				trackInitialInstalledApps: true,
				httpActivity:              httpActivity,
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
			"type":       "android_emulator",
			"serial":     "serial-from-runner",
		},
	}}, nil)

	env.OnActivity(
		startEmuActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"serial": "emu-1",
		"name":   "device-1",
	}}, nil)
	env.OnActivity(listAppsActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: []string{"com.android.settings"}}, nil)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "emu-1", result["serial"])
	require.Equal(t, "device-1", result["name"])
	require.Equal(t, "android_emulator", result["type"])
}

func TestGetOrCreateDeviceMapSkipsInstalledAppsSnapshotWhenNotTracking(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	workflowName := "get-device-map-no-snapshot"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{RunnerID: "runner-1"}
			return getOrCreateDeviceMap(getOrCreateDeviceMapInput{
				ctx:           ctx,
				mobileCtx:     ctx,
				payload:       payload,
				settedDevices: map[string]any{},
				appURL:        "http://localhost:8090",
				stepID:        "step-1",
				httpActivity:  httpActivity,
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
			"type":       "physical",
			"serial":     "serial-1",
		},
	}}, nil)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "serial-1", result["serial"])
	require.Equal(t, "android_phone", result["type"])
	require.Equal(t, "http://runner", result["runner_url"])
	_, hasInitialSnapshot := result["initial_installed_apps"]
	require.False(t, hasInitialSnapshot)
}

func TestExtractDeviceInfo(t *testing.T) {
	deviceType, serial, name, packages, err := extractDeviceInfo("runner-1", map[string]any{})
	_ = deviceType
	_ = serial
	_ = name
	_ = packages
	require.Error(t, err)

	deviceType, serial, name, packages, err = extractDeviceInfo("runner-1", map[string]any{
		"serial": "serial-1",
	})
	_ = deviceType
	_ = serial
	_ = name
	_ = packages
	require.Error(t, err)

	deviceType, serial, name, packages, err = extractDeviceInfo("runner-1", map[string]any{
		"type": "redroid",
	})
	_ = deviceType
	_ = serial
	_ = name
	_ = packages
	require.Error(t, err)

	deviceType, serial, name, packages, err = extractDeviceInfo("runner-1", map[string]any{
		"type":      "redroid",
		"serial":    "serial-1",
		"installed": map[string]any{"app": "com.example"},
	})
	_ = deviceType
	_ = serial
	_ = name
	_ = packages
	require.Error(t, err)

	deviceType, serial, name, packages, err = extractDeviceInfo("runner-1", map[string]any{
		"type":   "redroid",
		"serial": "serial-1",
		"name":   "device-1",
		"installed": map[string]string{
			"app": "com.example",
			"bad": "",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "redroid", deviceType)
	require.Equal(t, "serial-1", serial)
	require.Equal(t, "device-1", name)
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
			runnerURL, deviceType, serial, err := fetchRunnerInfo(fetchRunnerInfoInput{
				ctx:          ctx,
				payload:      payload,
				appURL:       "http://localhost:8090",
				stepID:       "step-1",
				httpActivity: httpActivity,
			})
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"runner_url": runnerURL,
				"type":       deviceType.String(),
				"serial":     serial,
			}, nil
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
			"type":       "physical",
			"serial":     "serial-1",
		},
	}}, nil)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "http://runner", result["runner_url"])
	require.Equal(t, "android_phone", result["type"])
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
				"type":   "physical",
				"serial": "serial-1",
			},
			errSubstr: "runner_url",
		},
		{
			name: "missing type",
			body: map[string]any{
				"runner_url": "http://runner",
				"serial":     "serial-1",
			},
			errSubstr: "device type",
		},
		{
			name: "invalid serial",
			body: map[string]any{
				"runner_url": "http://runner",
				"type":       "physical",
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
					payload := &workflows.MobileAutomationWorkflowPipelinePayload{
						RunnerID: "runner-1",
					}
					_, _, _, err := fetchRunnerInfo(fetchRunnerInfoInput{
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

func TestStartManagedDeviceAndroid(t *testing.T) {
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
			name, serial, err := startManagedDevice(startManagedDeviceInput{
				ctx:        ctx,
				mobileCtx:  ctx,
				deviceType: deviceTypeAndroidEmulator,
				activities: activitiesForDeviceType(deviceTypeAndroidEmulator),
				payload:    payload,
				stepID:     "step-1",
			})
			if err != nil {
				return nil, err
			}
			return map[string]any{"serial": serial, "name": name}, nil
		},
		workflow.RegisterOptions{Name: workflowName},
	)

	env.OnActivity(
		startEmuActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			return payload["device_name"] == "runner-1" && payload["type"] == "android_emulator"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"serial": "emu-1",
		"name":   "device-1",
	}}, nil)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "emu-1", result["serial"])
	require.Equal(t, "device-1", result["name"])
}

func TestStartManagedDeviceIOS(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	startIOSActivity := activities.NewStartIOSSimulatorActivity()
	env.RegisterActivityWithOptions(
		startIOSActivity.Execute,
		activity.RegisterOptions{Name: startIOSActivity.Name()},
	)

	workflowName := "start-ios-simulator"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{RunnerID: "runner-1"}
			name, serial, err := startManagedDevice(startManagedDeviceInput{
				ctx:        ctx,
				mobileCtx:  ctx,
				deviceType: deviceTypeIOSSimulator,
				activities: activitiesForDeviceType(deviceTypeIOSSimulator),
				payload:    payload,
				stepID:     "step-1",
			})
			if err != nil {
				return nil, err
			}
			return map[string]any{"serial": serial, "name": name}, nil
		},
		workflow.RegisterOptions{Name: workflowName},
	)

	env.OnActivity(
		startIOSActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			return payload["device_name"] == "runner-1" && payload["type"] == "ios_simulator"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"serial": "ios-1",
		"name":   "ios-device-1",
	}}, nil)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "ios-1", result["serial"])
	require.Equal(t, "ios-device-1", result["name"])
}

func TestStartManagedDeviceMissingSerialDefaultsToEmptyString(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	startEmuActivity := activities.NewStartEmulatorActivity()
	env.RegisterActivityWithOptions(
		startEmuActivity.Execute,
		activity.RegisterOptions{Name: startEmuActivity.Name()},
	)

	workflowName := "start-emulator-missing-serial"
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{RunnerID: "runner-1"}
			name, serial, err := startManagedDevice(startManagedDeviceInput{
				ctx:        ctx,
				mobileCtx:  ctx,
				deviceType: deviceTypeAndroidEmulator,
				activities: activitiesForDeviceType(deviceTypeAndroidEmulator),
				payload:    payload,
				stepID:     "step-1",
			})
			if err != nil {
				return nil, err
			}
			return map[string]any{"serial": serial, "name": name}, nil
		},
		workflow.RegisterOptions{Name: workflowName},
	)

	env.OnActivity(
		startEmuActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"name": "device-1",
	}}, nil)

	env.ExecuteWorkflow(workflowName)
	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "", result["serial"])
	require.Equal(t, "device-1", result["name"])
}

func TestStartManagedDeviceErrors(t *testing.T) {
	tests := []struct {
		name      string
		output    map[string]any
		errSubstr string
	}{
		{
			name:      "missing name",
			output:    map[string]any{"serial": "emu-1"},
			errSubstr: "missing name",
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
					payload := &workflows.MobileAutomationWorkflowPipelinePayload{
						RunnerID: "runner-1",
					}
					_, _, err := startManagedDevice(startManagedDeviceInput{
						ctx:        ctx,
						mobileCtx:  ctx,
						deviceType: deviceTypeAndroidEmulator,
						activities: activitiesForDeviceType(deviceTypeAndroidEmulator),
						payload:    payload,
						stepID:     "step-1",
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
		"video_path": "video.mp4",
		"log_path":   "log.txt",
	})
	require.Error(t, err)

	info, err := extractRecordingInfo("runner-1", map[string]any{
		"video_path":            "video.mp4",
		"log_path":              "log.txt",
		"recording_process_pid": 1,
		"recording_ffmpeg_pid":  2,
		"recording_log_pid":     3,
	})
	require.NoError(t, err)
	require.Equal(t, "video.mp4", info.videoPath)
	require.Equal(t, "log.txt", info.logPath)
	require.Equal(t, 1, info.recordingPid)
	require.Equal(t, 2, info.ffmpegPid)
	require.Equal(t, 3, info.logPid)
}

func TestExtractRecordingInfoIOS(t *testing.T) {
	info, err := extractRecordingInfo("runner-1", map[string]any{
		"type":                  "ios_simulator",
		"video_path":            "video.mp4",
		"log_path":              "log.txt",
		"recording_process_pid": 1,
		"recording_log_pid":     3,
	})
	require.NoError(t, err)
	require.Equal(t, deviceTypeIOSSimulator, info.deviceType)
	require.Equal(t, 1, info.recordingPid)
	require.Equal(t, 0, info.ffmpegPid)
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

	var nilOutput map[string]any
	require.Panics(t, func() {
		_ = extractAndStoreURLs(workflowengine.ActivityResult{Output: map[string]any{
			"body": map[string]any{
				"result_urls":     []string{"video-2"},
				"screenshot_urls": []string{"frame-2"},
			},
		}}, &nilOutput)
	})
}
