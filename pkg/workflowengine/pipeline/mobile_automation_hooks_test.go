// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"context"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

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
			"recording_process_pid": float64(1),
			"ffmpeg_process_pid":    float64(2),
			"log_process_pid":       float64(3),
			"video_path":            "/tmp/video.mp4",
			"log_path":              "/tmp/log.txt",
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

func TestInstallAppIfNeededUsesPlatformActivity(t *testing.T) {
	tests := []struct {
		name           string
		deviceType     mobileDeviceType
		register       func(env *testsuite.TestWorkflowEnvironment) (string, string)
		assetFieldName string
	}{
		{
			name:       "android",
			deviceType: deviceTypeAndroidPhone,
			register: func(env *testsuite.TestWorkflowEnvironment) (string, string) {
				installActivity := activities.NewApkInstallActivity()
				postInstallActivity := activities.NewApkPostInstallChecksActivity()
				env.RegisterActivityWithOptions(
					installActivity.Execute,
					activity.RegisterOptions{Name: installActivity.Name()},
				)
				env.RegisterActivityWithOptions(
					postInstallActivity.Execute,
					activity.RegisterOptions{Name: postInstallActivity.Name()},
				)
				return installActivity.Name(), postInstallActivity.Name()
			},
			assetFieldName: "apk",
		},
		{
			name:       "ios",
			deviceType: deviceTypeIOSSimulator,
			register: func(env *testsuite.TestWorkflowEnvironment) (string, string) {
				installActivity := activities.NewInstallIOSAppActivity()
				postInstallActivity := activities.NewIOSPostInstallChecksActivity()
				env.RegisterActivityWithOptions(
					installActivity.Execute,
					activity.RegisterOptions{Name: installActivity.Name()},
				)
				env.RegisterActivityWithOptions(
					postInstallActivity.Execute,
					activity.RegisterOptions{Name: postInstallActivity.Name()},
				)
				return installActivity.Name(), postInstallActivity.Name()
			},
			assetFieldName: "app",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()

			installActivityName, postInstallActivityName := tc.register(env)

			env.RegisterWorkflowWithOptions(
				func(ctx workflow.Context) (map[string]string, error) {
					ctx = workflow.WithActivityOptions(
						ctx,
						workflow.ActivityOptions{StartToCloseTimeout: time.Second},
					)

					deviceMap := map[string]any{
						"installed": map[string]string{},
					}

					input := installAppIfNeededInput{
						mobileCtx:  ctx,
						deviceMap:  deviceMap,
						appPath:    "/tmp/app.bin",
						versionID:  "ver-1",
						serial:     "serial-1",
						stepID:     "step-1",
						activities: activitiesForDeviceType(tc.deviceType),
					}

					if err := installAppIfNeeded(input); err != nil {
						return nil, err
					}
					if err := installAppIfNeeded(input); err != nil {
						return nil, err
					}

					return deviceMap["installed"].(map[string]string), nil
				},
				workflow.RegisterOptions{Name: "test-install-app-if-needed"},
			)

			env.OnActivity(
				installActivityName,
				mock.Anything,
				mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
					payload, ok := input.Payload.(map[string]any)
					if !ok {
						return false
					}
					return payload[tc.assetFieldName] == "/tmp/app.bin" &&
						payload["serial"] == "serial-1"
				}),
			).Return(workflowengine.ActivityResult{Output: map[string]any{
				"package_id": "pkg-1",
			}}, nil).Once()
			if postInstallActivityName != "" {
				env.OnActivity(
					postInstallActivityName,
					mock.Anything,
					mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
						payload, ok := input.Payload.(map[string]any)
						if !ok {
							return false
						}
						return payload[tc.assetFieldName] == "/tmp/app.bin" &&
							payload["serial"] == "serial-1"
					}),
				).Return(workflowengine.ActivityResult{Output: map[string]any{
					"package_id": "pkg-1",
				}}, nil).Once()
			}

			env.ExecuteWorkflow("test-install-app-if-needed")
			require.NoError(t, env.GetWorkflowError())

			var installed map[string]string
			require.NoError(t, env.GetWorkflowResult(&installed))
			require.Equal(t, "pkg-1", installed["ver-1"])
		})
	}
}

func TestFetchAndInstallAPKStoresActionCode(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	installActivity := activities.NewApkInstallActivity()
	postInstallActivity := activities.NewApkPostInstallChecksActivity()
	listAppsActivity := activities.NewListInstalledAppsActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		installActivity.Execute,
		activity.RegisterOptions{Name: installActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		postInstallActivity.Execute,
		activity.RegisterOptions{Name: postInstallActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		listAppsActivity.Execute,
		activity.RegisterOptions{Name: listAppsActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ctx = workflow.WithActivityOptions(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			)

			step := &pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID: "step-1",
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"action_id": "action-1",
							"runner_id": "runner-1",
						},
					},
				},
			}
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{
				ActionID: "action-1",
				RunnerID: "runner-1",
			}
			deviceMap := map[string]any{
				"installed": map[string]string{},
			}

			err := fetchAndInstallAPK(fetchAndInstallAPKInput{
				ctx:          ctx,
				mobileCtx:    ctx,
				step:         step,
				payload:      payload,
				deviceMap:    deviceMap,
				deviceType:   deviceTypeAndroidPhone,
				activities:   activitiesForDeviceType(deviceTypeAndroidPhone),
				appURL:       "https://app.example",
				runnerURL:    "https://runner.example",
				serial:       "serial-1",
				httpActivity: httpActivity,
			})
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"action_code": step.With.Payload["action_code"],
				"stored":      step.With.Payload["stored_action_code"],
				"installed":   deviceMap["installed"].(map[string]string)["ver-1"],
			}, nil
		},
		workflow.RegisterOptions{Name: "test-fetch-and-install-apk"},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			body, ok := payload["body"].(map[string]any)
			return ok &&
				workflowengine.AsString(
					payload["url"],
				) == "https://runner.example/credimi/installer-action" &&
				workflowengine.AsString(body["platform"]) == "android"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"installer_path": "/tmp/app.apk",
			"version_id":     "ver-1",
			"code":           "code-1",
		},
	}}, nil)
	env.OnActivity(listAppsActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: []string{"com.android.settings"}}, nil)

	env.OnActivity(
		installActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			return payload["apk"] == "/tmp/app.apk" && payload["serial"] == "serial-1"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"package_id": "pkg-1",
	}}, nil)
	env.OnActivity(
		postInstallActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			return payload["apk"] == "/tmp/app.apk" && payload["serial"] == "serial-1"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"package_id": "pkg-1",
	}}, nil)

	env.ExecuteWorkflow("test-fetch-and-install-apk")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "code-1", result["action_code"])
	require.Equal(t, true, result["stored"])
	require.Equal(t, "pkg-1", result["installed"])
}

func TestFetchAndInstallAPKExternalInstallSkipsInstallerAndMutatesStepUse(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ctx = workflow.WithActivityOptions(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			)

			step := &pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID:  "step-1",
					Use: mobileAutomationStepUse,
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"action_id":  "action-1",
							"runner_id":  "runner-1",
							"version_id": mobileExternalSourceVersionID,
						},
					},
					Metadata: map[string]any{
						mobileSpecialInstallMetadataKey: true,
					},
				},
			}
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{
				ActionID:  "action-1",
				RunnerID:  "runner-1",
				VersionID: mobileExternalSourceVersionID,
			}
			deviceMap := map[string]any{
				"installed": map[string]string{},
			}

			err := fetchAndInstallAPK(fetchAndInstallAPKInput{
				ctx:             ctx,
				mobileCtx:       ctx,
				step:            step,
				payload:         payload,
				deviceMap:       deviceMap,
				deviceType:      deviceTypeAndroidPhone,
				activities:      activitiesForDeviceType(deviceTypeAndroidPhone),
				appURL:          "https://app.example",
				runnerURL:       "https://runner.example",
				serial:          "serial-1",
				skipInstaller:   true,
				externalInstall: true,
				httpActivity:    httpActivity,
			})
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"use":         step.Use,
				"action_code": step.With.Payload["action_code"],
				"stored":      step.With.Payload["stored_action_code"],
				"installed":   len(deviceMap["installed"].(map[string]string)),
			}, nil
		},
		workflow.RegisterOptions{Name: "test-fetch-and-install-apk-external"},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			body, ok := payload["body"].(map[string]any)
			return ok &&
				workflowengine.AsString(
					payload["url"],
				) == "https://runner.example/credimi/installer-action" &&
				workflowengine.AsBool(body["skip_installer"])
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"version_id": mobileExternalSourceVersionID,
			"code":       "code-1",
		},
	}}, nil)

	env.ExecuteWorkflow("test-fetch-and-install-apk-external")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, mobileExternalInstallStepUse, result["use"])
	require.Equal(t, "code-1", result["action_code"])
	require.Equal(t, true, result["stored"])
	require.Equal(t, float64(0), result["installed"])
}

func TestFetchAndInstallAPKExternalSourceNonInstallStepSkipsInstallerWithoutMutatingUse(
	t *testing.T,
) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ctx = workflow.WithActivityOptions(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			)

			step := &pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID:  "step-1",
					Use: mobileAutomationStepUse,
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"action_id":  "action-1",
							"runner_id":  "runner-1",
							"version_id": mobileExternalSourceVersionID,
						},
					},
				},
			}
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{
				ActionID:  "action-1",
				RunnerID:  "runner-1",
				VersionID: mobileExternalSourceVersionID,
			}
			deviceMap := map[string]any{
				"installed": map[string]string{},
			}

			err := fetchAndInstallAPK(fetchAndInstallAPKInput{
				ctx:           ctx,
				mobileCtx:     ctx,
				step:          step,
				payload:       payload,
				deviceMap:     deviceMap,
				deviceType:    deviceTypeAndroidPhone,
				activities:    activitiesForDeviceType(deviceTypeAndroidPhone),
				appURL:        "https://app.example",
				runnerURL:     "https://runner.example",
				serial:        "serial-1",
				skipInstaller: true,
				httpActivity:  httpActivity,
			})
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"use":         step.Use,
				"action_code": step.With.Payload["action_code"],
				"stored":      step.With.Payload["stored_action_code"],
				"installed":   len(deviceMap["installed"].(map[string]string)),
			}, nil
		},
		workflow.RegisterOptions{Name: "test-fetch-and-install-apk-external-source"},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			body, ok := payload["body"].(map[string]any)
			return ok &&
				workflowengine.AsString(
					payload["url"],
				) == "https://runner.example/credimi/installer-action" &&
				workflowengine.AsBool(body["skip_installer"])
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"version_id": mobileExternalSourceVersionID,
			"code":       "code-1",
		},
	}}, nil)

	env.ExecuteWorkflow("test-fetch-and-install-apk-external-source")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, mobileAutomationStepUse, result["use"])
	require.Equal(t, "code-1", result["action_code"])
	require.Equal(t, true, result["stored"])
	require.Equal(t, float64(0), result["installed"])
}

func TestPrepareMobileAutomationStepsHoistsExternalInstallSteps(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) ([]pipeline.StepDefinition, error) {
			ctx = workflow.WithActivityOptions(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			)

			steps := []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{
						ID:  "normal-http",
						Use: "http-request",
					},
				},
				{
					StepSpec: pipeline.StepSpec{
						ID:  "special-install",
						Use: mobileAutomationStepUse,
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"action_id":  "wallet/install",
								"version_id": mobileExternalSourceVersionID,
							},
						},
					},
				},
				{
					StepSpec: pipeline.StepSpec{
						ID:  "external-regular",
						Use: mobileAutomationStepUse,
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"action_id":  "wallet/open",
								"version_id": mobileExternalSourceVersionID,
							},
						},
					},
				},
			}

			if err := prepareMobileAutomationSteps(ctx, &steps, "https://app.example", httpActivity); err != nil {
				return nil, err
			}

			return steps, nil
		},
		workflow.RegisterOptions{Name: "test-prepare-mobile-automation-steps"},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.Anything,
	).Return(func(
		_ context.Context,
		input workflowengine.ActivityInput,
	) (workflowengine.ActivityResult, error) {
		payload, ok := input.Payload.(map[string]any)
		require.True(t, ok)
		require.Equal(
			t,
			"https://app.example/api/canonify/identifier/validate",
			workflowengine.AsString(payload["url"]),
		)

		body, ok := payload["body"].(map[string]any)
		require.True(t, ok)
		actionID := workflowengine.AsString(body["canonified_name"])

		category := "open-app"
		if actionID == "wallet/install" {
			category = walletActionCategoryInstallApp
		}

		return workflowengine.ActivityResult{Output: map[string]any{
			"body": map[string]any{
				"record": map[string]any{
					"category": category,
				},
			},
		}}, nil
	})

	env.ExecuteWorkflow("test-prepare-mobile-automation-steps")
	require.NoError(t, env.GetWorkflowError())

	var result []pipeline.StepDefinition
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Len(t, result, 3)
	require.Equal(t, "special-install", result[0].ID)
	require.Equal(t, "normal-http", result[1].ID)
	require.Equal(t, "external-regular", result[2].ID)
	require.Equal(t, true, result[0].Metadata[mobileSpecialInstallMetadataKey])
	require.Nil(t, result[2].Metadata)
}

func TestProcessStepAddsNormalizedDeviceTypeAndTaskQueue(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	installActivity := activities.NewApkInstallActivity()
	postInstallActivity := activities.NewApkPostInstallChecksActivity()
	listAppsActivity := activities.NewListInstalledAppsActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		installActivity.Execute,
		activity.RegisterOptions{Name: installActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		postInstallActivity.Execute,
		activity.RegisterOptions{Name: postInstallActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		listAppsActivity.Execute,
		activity.RegisterOptions{Name: listAppsActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)

			step := &pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID:  "step-1",
					Use: mobileAutomationStepUse,
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"action_id": "action-1",
						},
					},
				},
			}
			runData := map[string]any{}
			settedDevices := map[string]any{}

			err := processStep(processStepInput{
				ctx:            ctx,
				step:           step,
				config:         map[string]any{"app_url": "https://app.example"},
				ao:             &ao,
				settedDevices:  settedDevices,
				runData:        &runData,
				httpActivity:   httpActivity,
				logger:         workflow.GetLogger(ctx),
				globalRunnerID: "tenant/runner-1",
			})
			if err != nil {
				return nil, err
			}

			deviceMap := runData["setted_devices"].(map[string]any)["tenant/runner-1"].(map[string]any)
			return map[string]any{
				"runner_id":    step.With.Payload["runner_id"],
				"serial":       step.With.Payload["serial"],
				"type":         step.With.Payload["type"],
				"action_code":  step.With.Payload["action_code"],
				"taskqueue":    step.With.Config["taskqueue"],
				"device_type":  deviceMap["type"],
				"package_id":   deviceMap["installed"].(map[string]string)["ver-1"],
				"stored_code":  step.With.Payload["stored_action_code"],
				"runner_count": len(runData["setted_devices"].(map[string]any)),
			}, nil
		},
		workflow.RegisterOptions{Name: "test-process-step"},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			return workflowengine.AsString(
				payload["url"],
			) == "https://app.example/api/mobile-runner"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"runner_url": "https://runner.example",
			"type":       "physical",
			"serial":     "serial-1",
		},
	}}, nil)
	env.OnActivity(listAppsActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: []string{"com.android.settings"}}, nil)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			body, ok := payload["body"].(map[string]any)
			return ok &&
				workflowengine.AsString(
					payload["url"],
				) == "https://runner.example/credimi/installer-action" &&
				workflowengine.AsString(body["platform"]) == "android"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"installer_path": "/tmp/app.apk",
			"version_id":     "ver-1",
			"code":           "code-1",
		},
	}}, nil)

	env.OnActivity(
		installActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			return payload["apk"] == "/tmp/app.apk" && payload["serial"] == "serial-1"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"package_id": "pkg-1",
	}}, nil)
	env.OnActivity(
		postInstallActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			return payload["apk"] == "/tmp/app.apk" && payload["serial"] == "serial-1"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"package_id": "pkg-1",
	}}, nil)

	env.ExecuteWorkflow("test-process-step")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "tenant/runner-1", result["runner_id"])
	require.Equal(t, "serial-1", result["serial"])
	require.Equal(t, "android_phone", result["type"])
	require.Equal(t, "code-1", result["action_code"])
	require.Equal(t, true, result["stored_code"])
	require.Equal(t, "tenant/runner-1-TaskQueue", result["taskqueue"])
	require.Equal(t, "android_phone", result["device_type"])
	require.Equal(t, "pkg-1", result["package_id"])
	require.Equal(t, float64(1), result["runner_count"])
}

func TestCleanupDeviceMarksDeviceCleaned(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	cleanupActivity := activities.NewCleanupDeviceActivity()
	env.RegisterActivityWithOptions(
		cleanupActivity.Execute,
		activity.RegisterOptions{Name: cleanupActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)

			deviceMap := map[string]any{
				"type":       "android_phone",
				"serial":     "serial-1",
				"runner_url": "https://runner.example",
				"recording":  false,
				"installed": map[string]string{
					"ver-1": "pkg-1",
				},
			}
			output := map[string]any{}
			cleanupErrs := []error{}

			err := cleanupDevice(cleanupDeviceInput{
				ctx:           ctx,
				runnerID:      "tenant/runner-1",
				raw:           deviceMap,
				mobileAo:      &ao,
				runIdentifier: "run-1",
				appURL:        "https://app.example",
				output:        &output,
				cleanupErrs:   &cleanupErrs,
				logger:        workflow.GetLogger(ctx),
			})
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"cleaned": deviceMap["cleaned"],
				"errors":  len(cleanupErrs),
			}, nil
		},
		workflow.RegisterOptions{Name: "test-cleanup-device"},
	)

	env.OnActivity(
		cleanupActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			packages, ok := payload["apk_packages"].([]any)
			if !ok {
				return false
			}
			return payload["serial"] == "serial-1" &&
				payload["type"] == "android_phone" &&
				len(packages) == 1 &&
				packages[0] == "pkg-1"
		}),
	).Return(workflowengine.ActivityResult{}, nil)

	env.ExecuteWorkflow("test-cleanup-device")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, true, result["cleaned"])
	require.Equal(t, float64(0), result["errors"])
}

func TestNormalizeDeviceTypeMappings(t *testing.T) {
	require.Equal(t, deviceTypeAndroidEmulator, normalizeDeviceType(" emulator "))
	require.Equal(t, deviceTypeAndroidPhone, normalizeDeviceType("physical"))
	require.Equal(t, deviceTypeIOSSimulator, normalizeDeviceType("ios"))
	require.Equal(t, deviceTypeIOSPhone, normalizeDeviceType("ios_phone"))
	require.Equal(t, deviceTypeRedroid, normalizeDeviceType("redroid"))
	require.Equal(t, mobileDeviceType("custom"), normalizeDeviceType(" custom "))
}

func TestExtractAndStoreRecordingInfoIOSUsesRecordingPID(t *testing.T) {
	deviceMap := map[string]any{
		"type": "ios_simulator",
	}

	err := extractAndStoreRecordingInfo(
		workflowengine.ActivityResult{Output: map[string]any{
			"recording_process_pid": float64(7),
			"video_path":            "/tmp/video.mp4",
			"log_path":              "/tmp/log.txt",
			"log_process_pid":       float64(8),
		}},
		deviceMap,
		"runner-1",
	)
	require.NoError(t, err)
	require.Equal(t, 7, deviceMap["recording_process_pid"])
	require.Equal(t, 0, deviceMap["recording_ffmpeg_pid"])
	require.Equal(t, 8, deviceMap["recording_log_pid"])
	require.Equal(t, 7, deviceMap["recording_process_pid"])
}

func TestFetchAndInstallAPKKeepsExistingActionCode(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	installActivity := activities.NewApkInstallActivity()
	postInstallActivity := activities.NewApkPostInstallChecksActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		installActivity.Execute,
		activity.RegisterOptions{Name: installActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		postInstallActivity.Execute,
		activity.RegisterOptions{Name: postInstallActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ctx = workflow.WithActivityOptions(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			)

			step := &pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID: "step-1",
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"action_id":   "action-1",
							"action_code": "preset-code",
							"runner_id":   "runner-1",
						},
					},
				},
			}
			payload := &workflows.MobileAutomationWorkflowPipelinePayload{
				ActionID:   "action-1",
				ActionCode: "preset-code",
				RunnerID:   "runner-1",
			}
			deviceMap := map[string]any{}

			err := fetchAndInstallAPK(fetchAndInstallAPKInput{
				ctx:          ctx,
				mobileCtx:    ctx,
				step:         step,
				payload:      payload,
				deviceMap:    deviceMap,
				deviceType:   deviceTypeAndroidPhone,
				activities:   activitiesForDeviceType(deviceTypeAndroidPhone),
				appURL:       "https://app.example",
				runnerURL:    "https://runner.example",
				serial:       "serial-1",
				httpActivity: httpActivity,
			})
			if err != nil {
				return nil, err
			}

			_, stored := step.With.Payload["stored_action_code"]
			return map[string]any{
				"action_code": step.With.Payload["action_code"],
				"stored":      stored,
			}, nil
		},
		workflow.RegisterOptions{Name: "test-fetch-and-install-apk-keep-code"},
	)

	env.OnActivity(httpActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"body": map[string]any{
				"installer_path": "/tmp/app.apk",
				"version_id":     "ver-1",
				"code":           "ignored-code",
			},
		}}, nil)

	env.OnActivity(installActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"package_id": "pkg-1",
		}}, nil)
	env.OnActivity(postInstallActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"package_id": "pkg-1",
		}}, nil)

	env.ExecuteWorkflow("test-fetch-and-install-apk-keep-code")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "preset-code", result["action_code"])
	require.Equal(t, false, result["stored"])
}

func TestStoreRecordingResultsSuccess(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ctx = workflow.WithActivityOptions(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			)

			output := map[string]any{
				"result_video_urls": []string{},
				"screenshot_urls":   []string{},
			}
			err := storeRecordingResults(storeRecordingResultsInput{
				ctx:        ctx,
				runnerURL:  "https://runner.example",
				videoPath:  "/tmp/video.mp4",
				lastFrame:  "/tmp/frame.png",
				logPath:    "/tmp/log.txt",
				deviceType: deviceTypeAndroidPhone,
				runID:      "run-1",
				runnerID:   "runner-1",
				appURL:     "https://app.example",
				output:     &output,
				logger:     workflow.GetLogger(ctx),
			})
			if err != nil {
				return nil, err
			}

			return output, nil
		},
		workflow.RegisterOptions{Name: "test-store-recording-results"},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			body, ok := payload["body"].(map[string]any)
			if !ok {
				return false
			}
			return workflowengine.AsString(
				payload["url"],
			) == "https://runner.example/credimi/pipeline-result" &&
				workflowengine.AsString(body["platform"]) == "android" &&
				workflowengine.AsString(body["log_path"]) == "/tmp/log.txt"
		}),
	).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"body": map[string]any{
				"result_urls":     []string{"video-url"},
				"screenshot_urls": []string{"frame-url"},
			},
		}}, nil)

	env.ExecuteWorkflow("test-store-recording-results")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, []any{"video-url"}, result["result_video_urls"])
	require.Equal(t, []any{"frame-url"}, result["screenshot_urls"])
}

func TestStoreRecordingResultsIOSSendsLogPath(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ctx = workflow.WithActivityOptions(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			)

			output := map[string]any{
				"result_video_urls": []string{},
				"screenshot_urls":   []string{},
			}
			return storeRecordingResults(storeRecordingResultsInput{
				ctx:        ctx,
				runnerURL:  "https://runner.example",
				videoPath:  "/tmp/video.mp4",
				lastFrame:  "/tmp/frame.png",
				logPath:    "/tmp/log.txt",
				deviceType: deviceTypeIOSSimulator,
				runID:      "run-1",
				runnerID:   "runner-1",
				appURL:     "https://app.example",
				output:     &output,
				logger:     workflow.GetLogger(ctx),
			})
		},
		workflow.RegisterOptions{Name: "test-store-recording-results-ios"},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			body, ok := payload["body"].(map[string]any)
			if !ok {
				return false
			}
			return workflowengine.AsString(
				payload["url"],
			) == "https://runner.example/credimi/pipeline-result" &&
				workflowengine.AsString(body["platform"]) == "ios" &&
				workflowengine.AsString(body["log_path"]) == "/tmp/log.txt"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"result_urls":     []string{"video-url"},
			"screenshot_urls": []string{"frame-url"},
		},
	}}, nil)

	env.ExecuteWorkflow("test-store-recording-results-ios")
	require.NoError(t, env.GetWorkflowError())
}

func TestMobileAutomationCleanupHookSuccess(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	cleanupActivity := activities.NewCleanupDeviceActivity()
	env.RegisterActivityWithOptions(
		cleanupActivity.Execute,
		activity.RegisterOptions{Name: cleanupActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (bool, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)

			deviceMap := map[string]any{
				"type":       "android_phone",
				"serial":     "serial-1",
				"runner_url": "https://runner.example",
				"recording":  false,
				"installed":  map[string]string{"ver-1": "pkg-1"},
			}

			err := MobileAutomationCleanupHook(
				ctx,
				nil,
				&ao,
				map[string]any{"app_url": "https://app.example"},
				map[string]any{
					"run_identifier": "run-1",
					"setted_devices": map[string]any{"tenant/runner-1": deviceMap},
				},
				&map[string]any{},
			)
			if err != nil {
				return false, err
			}

			cleaned, _ := deviceMap["cleaned"].(bool)
			return cleaned, nil
		},
		workflow.RegisterOptions{Name: "test-mobile-cleanup-hook-success"},
	)

	env.OnActivity(cleanupActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, nil)

	env.ExecuteWorkflow("test-mobile-cleanup-hook-success")
	require.NoError(t, env.GetWorkflowError())

	var cleaned bool
	require.NoError(t, env.GetWorkflowResult(&cleaned))
	require.True(t, cleaned)
}

func TestMobileAutomationCleanupHookMissingAppURL(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			return MobileAutomationCleanupHook(
				ctx,
				nil,
				&ao,
				map[string]any{},
				map[string]any{},
				&map[string]any{},
			)
		},
		workflow.RegisterOptions{Name: "test-mobile-cleanup-hook-missing-app-url"},
	)

	env.ExecuteWorkflow("test-mobile-cleanup-hook-missing-app-url")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing or invalid app_url")
}

func TestMobileAutomationSetupHookSuccess(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	installActivity := activities.NewApkInstallActivity()
	postInstallActivity := activities.NewApkPostInstallChecksActivity()
	recordActivity := activities.NewStartRecordingActivity()
	listAppsActivity := activities.NewListInstalledAppsActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		installActivity.Execute,
		activity.RegisterOptions{Name: installActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		postInstallActivity.Execute,
		activity.RegisterOptions{Name: postInstallActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		recordActivity.Execute,
		activity.RegisterOptions{Name: recordActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		listAppsActivity.Execute,
		activity.RegisterOptions{Name: listAppsActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)

			steps := []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step-1",
						Use: mobileAutomationStepUse,
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"action_id": "action-1",
							},
						},
					},
				},
			}
			runData := map[string]any{}

			err := MobileAutomationSetupHook(
				ctx,
				&steps,
				&ao,
				map[string]any{
					"app_url":                              "https://app.example",
					"global_runner_id":                     "tenant/runner-1",
					mobileRunnerSemaphoreTicketIDConfigKey: "ticket-1",
				},
				&runData,
			)
			if err != nil {
				return nil, err
			}

			deviceMap := runData["setted_devices"].(map[string]any)["tenant/runner-1"].(map[string]any)
			return map[string]any{
				"runner_id":     steps[0].With.Payload["runner_id"],
				"serial":        steps[0].With.Payload["serial"],
				"type":          steps[0].With.Payload["type"],
				"action_code":   steps[0].With.Payload["action_code"],
				"taskqueue":     steps[0].With.Config["taskqueue"],
				"recording":     deviceMap["recording"],
				"video_path":    deviceMap["video_path"],
				"log_path":      deviceMap["log_path"],
				"recording_pid": deviceMap["recording_process_pid"],
			}, nil
		},
		workflow.RegisterOptions{Name: "test-mobile-automation-setup-success"},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			return ok &&
				workflowengine.AsString(payload["url"]) == "https://app.example/api/mobile-runner"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"runner_url": "https://runner.example",
			"type":       "physical",
			"serial":     "serial-1",
		},
	}}, nil)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			body, ok := payload["body"].(map[string]any)
			return ok &&
				workflowengine.AsString(
					payload["url"],
				) == "https://runner.example/credimi/installer-action" &&
				workflowengine.AsString(body["platform"]) == "android"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"installer_path": "/tmp/app.apk",
			"version_id":     "ver-1",
			"code":           "code-1",
		},
	}}, nil)

	env.OnActivity(installActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"package_id": "pkg-1",
		}}, nil)
	env.OnActivity(postInstallActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"package_id": "pkg-1",
		}}, nil)
	env.OnActivity(listAppsActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: []string{"com.android.settings"}}, nil)

	env.OnActivity(recordActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"recording_process_pid": float64(11),
			"ffmpeg_process_pid":    float64(12),
			"log_process_pid":       float64(13),
			"video_path":            "/tmp/video.mp4",
			"log_path":              "/tmp/log.txt",
		}}, nil)

	env.ExecuteWorkflow("test-mobile-automation-setup-success")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "tenant/runner-1", result["runner_id"])
	require.Equal(t, "serial-1", result["serial"])
	require.Equal(t, "android_phone", result["type"])
	require.Equal(t, "code-1", result["action_code"])
	require.Equal(t, "tenant/runner-1-TaskQueue", result["taskqueue"])
	require.Equal(t, true, result["recording"])
	require.Equal(t, "/tmp/video.mp4", result["video_path"])
	require.Equal(t, "/tmp/log.txt", result["log_path"])
	require.Equal(t, float64(11), result["recording_pid"])
}

func TestMobileAutomationSetupHookDisablesPlayStoreWhenConfigured(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	installActivity := activities.NewApkInstallActivity()
	postInstallActivity := activities.NewApkPostInstallChecksActivity()
	disablePlayStoreActivity := activities.NewDisableAndroidPlayStoreActivity()
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
		postInstallActivity.Execute,
		activity.RegisterOptions{Name: postInstallActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		disablePlayStoreActivity.Execute,
		activity.RegisterOptions{Name: disablePlayStoreActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		recordActivity.Execute,
		activity.RegisterOptions{Name: recordActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)

			steps := []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step-1",
						Use: mobileAutomationStepUse,
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"action_id": "action-1",
							},
						},
					},
				},
			}
			runData := map[string]any{}

			err := MobileAutomationSetupHook(
				ctx,
				&steps,
				&ao,
				map[string]any{
					"app_url":                              "https://app.example",
					"global_runner_id":                     "tenant/runner-1",
					mobileRunnerSemaphoreTicketIDConfigKey: "ticket-1",
					mobileDisableAndroidPlayStoreConfigKey: true,
				},
				&runData,
			)
			if err != nil {
				return nil, err
			}

			deviceMap := runData["setted_devices"].(map[string]any)["tenant/runner-1"].(map[string]any)
			return map[string]any{
				"play_store_disabled": deviceMap["play_store_disabled"],
				"recording":           deviceMap["recording"],
			}, nil
		},
		workflow.RegisterOptions{Name: "test-mobile-automation-setup-disable-play-store"},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			return ok &&
				workflowengine.AsString(payload["url"]) == "https://app.example/api/mobile-runner"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"runner_url": "https://runner.example",
			"type":       "physical",
			"serial":     "serial-1",
		},
	}}, nil)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}
			body, ok := payload["body"].(map[string]any)
			return ok &&
				workflowengine.AsString(
					payload["url"],
				) == "https://runner.example/credimi/installer-action" &&
				workflowengine.AsString(body["platform"]) == "android"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"body": map[string]any{
			"installer_path": "/tmp/app.apk",
			"version_id":     "ver-1",
			"code":           "code-1",
		},
	}}, nil)

	env.OnActivity(installActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"package_id": "pkg-1",
		}}, nil)
	env.OnActivity(postInstallActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"package_id": "pkg-1",
		}}, nil)
	env.OnActivity(
		disablePlayStoreActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			return ok && workflowengine.AsString(payload["serial"]) == "serial-1"
		}),
	).Return(workflowengine.ActivityResult{Output: map[string]any{
		"message": "disabled",
	}}, nil).Once()
	env.OnActivity(recordActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"recording_process_pid": float64(11),
			"ffmpeg_process_pid":    float64(12),
			"log_process_pid":       float64(13),
			"video_path":            "/tmp/video.mp4",
			"log_path":              "/tmp/log.txt",
		}}, nil)

	env.ExecuteWorkflow("test-mobile-automation-setup-disable-play-store")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, true, result["play_store_disabled"])
	require.Equal(t, true, result["recording"])
}

func TestMobileAutomationSetupHookDefersPlayStoreDisableForExternalInstallSteps(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	listAppsActivity := activities.NewListInstalledAppsActivity()
	recordActivity := activities.NewStartRecordingActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		listAppsActivity.Execute,
		activity.RegisterOptions{Name: listAppsActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		recordActivity.Execute,
		activity.RegisterOptions{Name: recordActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)

			steps := []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step-1",
						Use: mobileAutomationStepUse,
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"action_id":  "wallet/install",
								"version_id": mobileExternalSourceVersionID,
							},
						},
					},
				},
			}
			runData := map[string]any{}

			err := MobileAutomationSetupHook(
				ctx,
				&steps,
				&ao,
				map[string]any{
					"app_url":                              "https://app.example",
					"global_runner_id":                     "tenant/runner-1",
					mobileRunnerSemaphoreTicketIDConfigKey: "ticket-1",
					mobileDisableAndroidPlayStoreConfigKey: true,
				},
				&runData,
			)
			if err != nil {
				return nil, err
			}

			deviceMap := runData["setted_devices"].(map[string]any)["tenant/runner-1"].(map[string]any)
			_, hasPlayStoreDisabled := deviceMap["play_store_disabled"]
			return map[string]any{
				"use":                        steps[0].Use,
				"pending_play_store_disable": runData[mobilePendingPlayStoreDisableRunDataKey],
				"play_store_disabled":        hasPlayStoreDisabled,
				"recording":                  deviceMap["recording"],
			}, nil
		},
		workflow.RegisterOptions{Name: "test-mobile-automation-setup-defer-disable-play-store"},
	)

	env.OnActivity(
		httpActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(map[string]any)
			if !ok {
				return false
			}

			switch workflowengine.AsString(payload["url"]) {
			case "https://app.example/api/canonify/identifier/validate":
				body, ok := payload["body"].(map[string]any)
				return ok && workflowengine.AsString(body["canonified_name"]) == "wallet/install"
			case "https://app.example/api/mobile-runner":
				return true
			case "https://runner.example/credimi/installer-action":
				body, ok := payload["body"].(map[string]any)
				return ok && workflowengine.AsBool(body["skip_installer"])
			default:
				return false
			}
		}),
	).Return(func(
		_ context.Context,
		input workflowengine.ActivityInput,
	) (workflowengine.ActivityResult, error) {
		payload := input.Payload.(map[string]any)
		switch workflowengine.AsString(payload["url"]) {
		case "https://app.example/api/canonify/identifier/validate":
			return workflowengine.ActivityResult{Output: map[string]any{
				"body": map[string]any{
					"record": map[string]any{
						"category": walletActionCategoryInstallApp,
					},
				},
			}}, nil
		case "https://app.example/api/mobile-runner":
			return workflowengine.ActivityResult{Output: map[string]any{
				"body": map[string]any{
					"runner_url": "https://runner.example",
					"type":       "physical",
					"serial":     "serial-1",
				},
			}}, nil
		default:
			return workflowengine.ActivityResult{Output: map[string]any{
				"body": map[string]any{
					"version_id": mobileExternalSourceVersionID,
					"code":       "code-1",
				},
			}}, nil
		}
	})

	env.OnActivity(listAppsActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: []string{"com.example.old"}}, nil)
	env.OnActivity(recordActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"recording_process_pid": float64(11),
			"ffmpeg_process_pid":    float64(12),
			"log_process_pid":       float64(13),
			"video_path":            "/tmp/video.mp4",
			"log_path":              "/tmp/log.txt",
		}}, nil)

	env.ExecuteWorkflow("test-mobile-automation-setup-defer-disable-play-store")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, mobileExternalInstallStepUse, result["use"])
	require.Equal(t, true, result["pending_play_store_disable"])
	require.Equal(t, false, result["play_store_disabled"])
	require.Equal(t, true, result["recording"])
}

func TestCleanupDeviceWithRecordingSuccess(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	stopActivity := activities.NewStopRecordingActivity()
	httpActivity := activities.NewHTTPActivity()
	cleanupActivity := activities.NewCleanupDeviceActivity()
	env.RegisterActivityWithOptions(
		stopActivity.Execute,
		activity.RegisterOptions{Name: stopActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		cleanupActivity.Execute,
		activity.RegisterOptions{Name: cleanupActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]any, error) {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)

			deviceMap := map[string]any{
				"type":                  "android_phone",
				"serial":                "serial-1",
				"runner_url":            "https://runner.example",
				"recording":             true,
				"video_path":            "/tmp/video.mp4",
				"log_path":              "/tmp/log.txt",
				"recording_process_pid": 1,
				"recording_ffmpeg_pid":  2,
				"recording_log_pid":     3,
				"installed":             map[string]string{"ver-1": "pkg-1"},
			}
			output := map[string]any{
				"result_video_urls": []string{},
				"screenshot_urls":   []string{},
			}
			cleanupErrs := []error{}

			err := cleanupDevice(cleanupDeviceInput{
				ctx:           ctx,
				runnerID:      "tenant/runner-1",
				raw:           deviceMap,
				mobileAo:      &ao,
				runIdentifier: "run-1",
				appURL:        "https://app.example",
				output:        &output,
				cleanupErrs:   &cleanupErrs,
				logger:        workflow.GetLogger(ctx),
			})
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"cleaned":      deviceMap["cleaned"],
				"result_urls":  output["result_video_urls"],
				"frame_urls":   output["screenshot_urls"],
				"cleanup_errs": len(cleanupErrs),
			}, nil
		},
		workflow.RegisterOptions{Name: "test-cleanup-device-with-recording"},
	)

	env.OnActivity(stopActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"last_frame_path": "/tmp/last.png",
		}}, nil)

	env.OnActivity(httpActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"body": map[string]any{
				"result_urls":     []string{"video-url"},
				"screenshot_urls": []string{"frame-url"},
			},
		}}, nil)

	env.OnActivity(cleanupActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, nil)

	env.ExecuteWorkflow("test-cleanup-device-with-recording")
	require.NoError(t, env.GetWorkflowError())

	var result map[string]any
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, true, result["cleaned"])
	require.Equal(t, []any{"video-url"}, result["result_urls"])
	require.Equal(t, []any{"frame-url"}, result["frame_urls"])
	require.Equal(t, float64(0), result["cleanup_errs"])
}

func TestProcessStepMissingAppURL(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			step := &pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID:  "step-1",
					Use: mobileAutomationStepUse,
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"action_id": "action-1",
							"runner_id": "runner-1",
						},
					},
				},
			}
			runData := map[string]any{}

			return processStep(processStepInput{
				ctx:           workflow.WithActivityOptions(ctx, ao),
				step:          step,
				config:        map[string]any{},
				ao:            &ao,
				settedDevices: map[string]any{},
				runData:       &runData,
				httpActivity:  activities.NewHTTPActivity(),
				logger:        workflow.GetLogger(ctx),
			})
		},
		workflow.RegisterOptions{Name: "test-process-step-missing-app-url"},
	)

	env.ExecuteWorkflow("test-process-step-missing-app-url")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing or invalid app_url")
}

func TestStoreRecordingResultsReturnsActivityError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ctx = workflow.WithActivityOptions(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			)

			output := map[string]any{
				"result_video_urls": []string{},
				"screenshot_urls":   []string{},
			}
			return storeRecordingResults(storeRecordingResultsInput{
				ctx:        ctx,
				runnerURL:  "https://runner.example",
				videoPath:  "/tmp/video.mp4",
				lastFrame:  "/tmp/frame.png",
				logPath:    "/tmp/log.txt",
				deviceType: deviceTypeAndroidPhone,
				runID:      "run-1",
				runnerID:   "runner-1",
				appURL:     "https://app.example",
				output:     &output,
				logger:     workflow.GetLogger(ctx),
			})
		},
		workflow.RegisterOptions{Name: "test-store-recording-results-error"},
	)

	env.OnActivity(httpActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, assert.AnError)

	env.ExecuteWorkflow("test-store-recording-results-error")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), assert.AnError.Error())
}

func TestMobileAutomationCleanupHookMissingRunIdentifier(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	cleanupActivity := activities.NewCleanupDeviceActivity()
	env.RegisterActivityWithOptions(
		cleanupActivity.Execute,
		activity.RegisterOptions{Name: cleanupActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)

			return MobileAutomationCleanupHook(
				ctx,
				nil,
				&ao,
				map[string]any{"app_url": "https://app.example"},
				map[string]any{
					"setted_devices": map[string]any{
						"tenant/runner-1": map[string]any{
							"type":       "android_phone",
							"serial":     "serial-1",
							"runner_url": "https://runner.example",
							"recording":  false,
							"installed":  map[string]string{"ver-1": "pkg-1"},
						},
					},
				},
				&map[string]any{},
			)
		},
		workflow.RegisterOptions{Name: "test-mobile-cleanup-hook-missing-run-id"},
	)

	env.OnActivity(cleanupActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, nil)

	env.ExecuteWorkflow("test-mobile-cleanup-hook-missing-run-id")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "one or more errors occurred during mobile automation cleanup")
}

func TestMobileAutomationSetupHookCollectRunnerIDsError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			steps := []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step-1",
						Use: mobileAutomationStepUse,
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"action_code": "code-without-version",
								"runner_id":   "runner-1",
							},
						},
					},
				},
			}
			runData := map[string]any{}
			return MobileAutomationSetupHook(
				ctx,
				&steps,
				&ao,
				map[string]any{mobileRunnerSemaphoreTicketIDConfigKey: "ticket-1"},
				&runData,
			)
		},
		workflow.RegisterOptions{Name: "test-mobile-setup-collect-runner-error"},
	)

	env.ExecuteWorkflow("test-mobile-setup-collect-runner-error")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing or invalid version_id")
}

func TestMobileAutomationSetupHookProcessStepError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			steps := []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step-1",
						Use: mobileAutomationStepUse,
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"runner_id": "runner-1",
							},
						},
					},
				},
			}
			runData := map[string]any{}
			return MobileAutomationSetupHook(
				ctx,
				&steps,
				&ao,
				map[string]any{
					"app_url":                              "https://app.example",
					mobileRunnerSemaphoreTicketIDConfigKey: "ticket-1",
				},
				&runData,
			)
		},
		workflow.RegisterOptions{Name: "test-mobile-setup-process-step-error"},
	)

	env.ExecuteWorkflow("test-mobile-setup-process-step-error")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing or invalid action_id")
}

func TestProcessStepMissingRunnerURL(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			step := &pipeline.StepDefinition{
				StepSpec: pipeline.StepSpec{
					ID:  "step-1",
					Use: mobileAutomationStepUse,
					With: pipeline.StepInputs{
						Payload: map[string]any{
							"action_id": "action-1",
							"runner_id": "runner-1",
						},
					},
				},
			}
			runData := map[string]any{}

			return processStep(processStepInput{
				ctx:  ctx,
				step: step,
				config: map[string]any{
					"app_url": "https://app.example",
				},
				ao: &ao,
				settedDevices: map[string]any{
					"runner-1": map[string]any{
						"type":   "android_phone",
						"serial": "serial-1",
					},
				},
				runData:        &runData,
				httpActivity:   activities.NewHTTPActivity(),
				logger:         workflow.GetLogger(ctx),
				globalRunnerID: "",
			})
		},
		workflow.RegisterOptions{Name: "test-process-step-missing-runner-url"},
	)

	env.ExecuteWorkflow("test-process-step-missing-runner-url")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing or invalid runner_url")
}

func TestFetchRunnerInfoRejectsEmptyDeviceType(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	httpActivity := activities.NewHTTPActivity()
	env.RegisterActivityWithOptions(
		httpActivity.Execute,
		activity.RegisterOptions{Name: httpActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ctx = workflow.WithActivityOptions(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			)
			_, _, _, err := fetchRunnerInfo(fetchRunnerInfoInput{
				ctx: ctx,
				payload: &workflows.MobileAutomationWorkflowPipelinePayload{
					RunnerID: "runner-1",
				},
				appURL:       "https://app.example",
				stepID:       "step-1",
				httpActivity: httpActivity,
			})
			return err
		},
		workflow.RegisterOptions{Name: "test-fetch-runner-empty-type"},
	)

	env.OnActivity(httpActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"body": map[string]any{
				"runner_url": "https://runner.example",
				"type":       "",
				"serial":     "serial-1",
			},
		}}, nil)

	env.ExecuteWorkflow("test-fetch-runner-empty-type")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "device type")
}

func TestInstallAppIfNeededRequiresPackageID(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	installActivity := activities.NewApkInstallActivity()
	postInstallActivity := activities.NewApkPostInstallChecksActivity()
	env.RegisterActivityWithOptions(
		installActivity.Execute,
		activity.RegisterOptions{Name: installActivity.Name()},
	)
	env.RegisterActivityWithOptions(
		postInstallActivity.Execute,
		activity.RegisterOptions{Name: postInstallActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ctx = workflow.WithActivityOptions(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			)
			return installAppIfNeeded(installAppIfNeededInput{
				mobileCtx:  ctx,
				deviceMap:  map[string]any{},
				appPath:    "/tmp/app.apk",
				versionID:  "ver-1",
				serial:     "serial-1",
				stepID:     "step-1",
				activities: activitiesForDeviceType(deviceTypeAndroidPhone),
			})
		},
		workflow.RegisterOptions{Name: "test-install-app-missing-package-id"},
	)

	env.OnActivity(installActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{}}, nil)
	env.OnActivity(postInstallActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{}}, nil)

	env.ExecuteWorkflow("test-install-app-missing-package-id")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing package_id")
}

func TestStartRecordingForDevicesPropagatesError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			return startRecordingForDevices(startRecordingForDevicesInput{
				ctx: workflow.WithActivityOptions(ctx, ao),
				settedDevices: map[string]any{
					"runner-1": map[string]any{
						"recording": false,
					},
				},
				ao: &ao,
			})
		},
		workflow.RegisterOptions{Name: "test-start-recording-for-devices-error"},
	)

	env.ExecuteWorkflow("test-start-recording-for-devices-error")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing serial")
}

func TestExtractAndStoreRecordingInfoErrorBranches(t *testing.T) {
	tests := []struct {
		name    string
		output  map[string]any
		message string
	}{
		{
			name: "missing ffmpeg pid",
			output: map[string]any{
				"recording_process_pid": float64(1),
				"log_process_pid":       float64(3),
				"video_path":            "video.mp4",
				"log_path":              "log.txt",
			},
			message: "missing ffmpeg_process",
		},
		{
			name: "missing log pid",
			output: map[string]any{
				"recording_process_pid": float64(1),
				"ffmpeg_process_pid":    float64(2),
				"video_path":            "video.mp4",
				"log_path":              "log.txt",
			},
			message: "missing log_process",
		},
		{
			name: "missing video path",
			output: map[string]any{
				"recording_process_pid": float64(1),
				"ffmpeg_process_pid":    float64(2),
				"log_process_pid":       float64(3),
				"log_path":              "log.txt",
			},
			message: "missing video_path",
		},
		{
			name: "missing log path",
			output: map[string]any{
				"recording_process_pid": float64(1),
				"ffmpeg_process_pid":    float64(2),
				"log_process_pid":       float64(3),
				"video_path":            "video.mp4",
			},
			message: "missing log_path",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := extractAndStoreRecordingInfo(
				workflowengine.ActivityResult{Output: tc.output},
				map[string]any{"type": "android_phone"},
				"runner-1",
			)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.message)
		})
	}
}

func TestCleanupDeviceReturnsCleanupActivityError(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	cleanupActivity := activities.NewCleanupDeviceActivity()
	env.RegisterActivityWithOptions(
		cleanupActivity.Execute,
		activity.RegisterOptions{Name: cleanupActivity.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			output := map[string]any{}
			cleanupErrs := []error{}

			return cleanupDevice(cleanupDeviceInput{
				ctx:      ctx,
				runnerID: "runner-1",
				raw: map[string]any{
					"type":       "android_phone",
					"serial":     "serial-1",
					"runner_url": "https://runner",
					"recording":  false,
					"installed":  map[string]string{},
				},
				mobileAo:      &ao,
				runIdentifier: "run-1",
				appURL:        "https://app.example",
				output:        &output,
				cleanupErrs:   &cleanupErrs,
				logger:        workflow.GetLogger(ctx),
			})
		},
		workflow.RegisterOptions{Name: "test-cleanup-device-error"},
	)

	env.OnActivity(cleanupActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, assert.AnError)

	env.ExecuteWorkflow("test-cleanup-device-error")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), assert.AnError.Error())
}

func testSetupHookWorkflow(
	ctx workflow.Context,
	steps []pipeline.StepDefinition,
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
		ctx:       ctx,
		runnerID:  "runner-1",
		deviceMap: map[string]any{},
		ao:        &activityOptions,
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
		ctx:           ctx,
		settedDevices: setted,
		ao:            &activityOptions,
	})
}

func testStartRecordingSuccessWorkflow(ctx workflow.Context) (map[string]any, error) {
	activityOptions := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
	device := map[string]any{
		"serial": "serial-1",
	}
	err := startRecordingForDevice(startRecordingForDeviceInput{
		ctx:       ctx,
		runnerID:  "runner-1",
		deviceMap: device,
		ao:        &activityOptions,
	})
	return device, err
}

func testCleanupRecordingWorkflow(ctx workflow.Context) (map[string]any, error) {
	ctx = workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{StartToCloseTimeout: time.Second},
	)
	output := map[string]any{
		"result_video_urls": []string{},
		"screenshot_urls":   []string{},
	}
	errs := []error{}
	deviceInfo := map[string]any{
		"runner_url":            "https://runner",
		"recording":             true,
		"video_path":            "/tmp/video.mp4",
		"log_path":              "/tmp/log.txt",
		"recording_process_pid": 1,
		"recording_ffmpeg_pid":  2,
		"recording_log_pid":     3,
	}
	cleanupRecording(cleanupRecordingInput{
		mobileCtx:   ctx,
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
	ctx = workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{StartToCloseTimeout: time.Second},
	)
	info := &recordingInfo{
		videoPath:    "/tmp/video.mp4",
		logPath:      "/tmp/log.txt",
		recordingPid: 1,
		ffmpegPid:    2,
		logPid:       3,
	}
	_, err := stopRecording(ctx, info, workflow.GetLogger(ctx))
	return err
}

func mobileAutomationSetupSteps() []pipeline.StepDefinition {
	return []pipeline.StepDefinition{
		{
			StepSpec: pipeline.StepSpec{
				ID:  "step-1",
				Use: "mobile-automation",
				With: pipeline.StepInputs{
					Payload: map[string]any{
						"runner_id": "runner-1",
						"action_id": "action-1",
					},
				},
			},
		},
	}
}
