// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"errors"
	"maps"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevice"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/temporalnexus"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// testPipelineRun is the pipeline run of a test workflow started with ID
// "pipeline-1".
var testPipelineRun = mobiledevice.PipelineRun{
	Namespace:  "default-test-namespace",
	WorkflowID: "pipeline-1",
	RunID:      "default-test-run-id",
}

func deviceWorkflowInput() workflowengine.WorkflowInput {
	return workflowengine.WorkflowInput{
		Config: map[string]any{"app_url": "https://app.example"},
		ActivityOptions: &workflow.ActivityOptions{
			StartToCloseTimeout: time.Second,
		},
	}
}

// recordNexusOptions records the options of every Nexus operation the
// workflows of env start.
func recordNexusOptions(env *testsuite.TestWorkflowEnvironment) *[]workflow.NexusOperationOptions {
	recorded := &[]workflow.NexusOperationOptions{}
	env.SetWorkerOptions(worker.Options{
		Interceptors: []interceptor.WorkerInterceptor{&nexusOptionsRecorder{recorded: recorded}},
	})
	return recorded
}

type nexusOptionsRecorder struct {
	interceptor.WorkerInterceptorBase
	recorded *[]workflow.NexusOperationOptions
}

func (r *nexusOptionsRecorder) InterceptWorkflow(
	_ workflow.Context,
	next interceptor.WorkflowInboundInterceptor,
) interceptor.WorkflowInboundInterceptor {
	return &nexusOptionsInbound{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
		recorded:                       r.recorded,
	}
}

type nexusOptionsInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	recorded *[]workflow.NexusOperationOptions
}

func (i *nexusOptionsInbound) Init(outbound interceptor.WorkflowOutboundInterceptor) error {
	return i.Next.Init(&nexusOptionsOutbound{
		WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{
			Next: outbound,
		},
		recorded: i.recorded,
	})
}

type nexusOptionsOutbound struct {
	interceptor.WorkflowOutboundInterceptorBase
	recorded *[]workflow.NexusOperationOptions
}

func (o *nexusOptionsOutbound) ExecuteNexusOperation(
	ctx workflow.Context,
	input interceptor.ExecuteNexusOperationInput,
) workflow.NexusOperationFuture {
	*o.recorded = append(*o.recorded, input.Options)
	return o.Next.ExecuteNexusOperation(ctx, input)
}

// TestPrepareDeviceWorkflowCleansDeviceWhoseSetupFails: an emulator whose
// setup fails after it started is deleted before Prepare returns the setup
// error, which also names a failed cleanup.
func TestPrepareDeviceWorkflowCleansDeviceWhoseSetupFails(t *testing.T) {
	const deviceID = "runner-1/device-1"
	tests := []struct {
		name       string
		cleanupErr error
	}{
		{name: "deletes the started emulator"},
		{
			name: "appends the cleanup error",
			cleanupErr: temporal.NewNonRetryableApplicationError(
				"runner unreachable",
				"CRE1",
				nil,
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			env.RegisterWorkflowWithOptions(
				PrepareDeviceWorkflow,
				workflow.RegisterOptions{Name: mobiledevice.PrepareWorkflowName},
			)
			var cleanups []map[string]any
			env.RegisterActivityWithOptions(
				func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
					cleanups = append(cleanups, workflowengine.AsMap(input.Payload))
					return workflowengine.ActivityResult{Output: "cleaned"}, tt.cleanupErr
				},
				activity.RegisterOptions{Name: activities.NewCleanupDeviceActivity().Name()},
			)
			registerStubActivity(env, activities.GetMobileDeviceActivityName)
			startActivity := activities.NewSetupMobileDeviceActivity()
			listAppsActivity := activities.NewListInstalledAppsActivity()
			for _, a := range []workflowengine.ExecutableActivity{startActivity, listAppsActivity} {
				env.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: a.Name()})
			}
			onGetMobileDevice(env, deviceID, map[string]any{
				"runner_id":  "tenant/runner-1",
				"runner_url": "https://runner.example",
				"type":       "emulator",
			})
			env.OnActivity(startActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: map[string]any{
					"serial": "emulator-5554",
					"name":   "clone-1",
				}}, nil)
			env.OnActivity(listAppsActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{}, temporal.NewNonRetryableApplicationError(
					"adb offline",
					"CRE2",
					nil,
				))

			env.ExecuteWorkflow(mobiledevice.PrepareWorkflowName, mobiledevice.PrepareInput{
				WorkflowID: "pipeline-1-device-runner-1-device-1-prepare",
				DeviceID:   deviceID,
				Requests: []mobiledevice.MobileDeviceRequest{{
					DeviceID:          deviceID,
					VersionIdentifier: "tenant/wallet/v1",
					StepID:            "step-1",
					RunIdentifier:     "tenant/run-1",
				}},
				RunIdentifier: "tenant/run-1",
				Pipeline:      testPipelineRun,
				Workflow:      deviceWorkflowInput(),
			})

			err := env.GetWorkflowError()
			require.Error(t, err)
			message := err.Error()
			assert.Contains(t, message, "adb offline")
			require.NotEmpty(t, cleanups)
			assert.Equal(t, "clone-1", cleanups[0]["name"])
			assert.Equal(t, "emulator-5554", cleanups[0]["serial"])
			if tt.cleanupErr == nil {
				assert.Len(t, cleanups, 1)
				assert.NotContains(t, message, "; cleanup: ")
				return
			}
			cleanupAt := strings.Index(message, "; cleanup: ")
			require.Positive(t, cleanupAt, message)
			assert.Less(t, strings.Index(message, "adb offline"), cleanupAt)
			assert.Contains(
				t,
				message[cleanupAt:],
				"one or more errors occurred during mobile automation cleanup",
			)
		})
	}
}

func TestCleanUpDeviceWorkflowMapsCleanupOutput(t *testing.T) {
	const deviceID = "tenant/runner-1/device-1"
	tests := []struct {
		name          string
		device        map[string]any
		runIdentifier string
		want          mobiledevice.CleanUpResult
		wantError     string
	}{
		{
			name: "returns the stored evidence",
			device: map[string]any{
				"type":                  "android_phone",
				"serial":                "serial-1",
				"runner_id":             "tenant/runner-1",
				"recording":             true,
				"video_path":            "/tmp/video.mp4",
				"log_path":              "/tmp/log.txt",
				"recording_process_pid": 1,
				"recording_ffmpeg_pid":  2,
				"device_log":            true,
				"device_log_pid":        3,
				"installed":             map[string]string{"ver-1": "pkg-1"},
			},
			runIdentifier: "tenant/run-1",
			want: mobiledevice.CleanUpResult{
				ResultVideoURLs: []string{"video-url"},
				ScreenshotURLs:  []string{"frame-url"},
			},
		},
		{
			name: "returns the cleanup warnings",
			device: map[string]any{
				"type":      "android_emulator",
				"runner_id": "tenant/runner-1",
				"installed": map[string]string{},
			},
			runIdentifier: "tenant/run-1",
			want: mobiledevice.CleanUpResult{
				Warnings: []string{
					"device cleanup skipped for " + deviceID + " because it never started",
				},
			},
		},
		{
			name: "returns the cleanup error",
			// Without a run identifier the cleanup cannot store the evidence.
			device: map[string]any{
				"type":      "android_phone",
				"serial":    "serial-1",
				"runner_id": "tenant/runner-1",
				"recording": false,
				"installed": map[string]string{"ver-1": "pkg-1"},
			},
			wantError: "one or more errors occurred during mobile automation cleanup",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			env.RegisterWorkflowWithOptions(
				CleanUpDeviceWorkflow,
				workflow.RegisterOptions{Name: mobiledevice.CleanUpWorkflowName},
			)
			stopActivity := activities.NewStopRecordingActivity()
			httpActivity := activities.NewMobileRunnerHTTPActivity(nil)
			cleanupActivity := activities.NewCleanupDeviceActivity()
			for _, a := range []workflowengine.ExecutableActivity{
				stopActivity,
				httpActivity,
				cleanupActivity,
			} {
				env.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: a.Name()})
			}
			onStopAndroidDeviceLog(env, map[string]any{})
			env.OnActivity(stopActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: map[string]any{
					"last_frame_path": "/tmp/last.png",
				}}, nil).Maybe()
			env.OnActivity(httpActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: map[string]any{
					"body": map[string]any{
						"result_urls":     []string{"video-url"},
						"screenshot_urls": []string{"frame-url"},
					},
				}}, nil).Maybe()
			env.OnActivity(cleanupActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{}, nil).Maybe()

			env.ExecuteWorkflow(mobiledevice.CleanUpWorkflowName, mobiledevice.CleanUpInput{
				WorkflowID:    "pipeline-1-device-tenant-runner-1-device-1-cleanup",
				DeviceID:      deviceID,
				Device:        tt.device,
				RunIdentifier: tt.runIdentifier,
				Pipeline:      testPipelineRun,
				Workflow:      deviceWorkflowInput(),
			})
			require.NoError(t, env.GetWorkflowError())
			var result mobiledevice.CleanUpResult
			require.NoError(t, env.GetWorkflowResult(&result))

			if tt.wantError != "" {
				require.Len(t, result.Errors, 1)
				assert.Contains(t, result.Errors[0], tt.wantError)
				result.Errors = nil
			}
			assert.Equal(t, tt.want, result)
		})
	}
}

// fakeDeviceService answers the Prepare and Clean up operations like a
// runner namespace would, each starting a workflow, and records their inputs.
type fakeDeviceService struct {
	prepares   []mobiledevice.PrepareInput
	cleanups   []mobiledevice.CleanUpInput
	prepareErr map[string]error
	cleanup    map[string]mobiledevice.CleanUpResult
	cleanupErr map[string]error
}

func (f *fakeDeviceService) register(t *testing.T, env *testsuite.TestWorkflowEnvironment) {
	t.Helper()
	env.RegisterWorkflowWithOptions(
		func(_ workflow.Context, in mobiledevice.PrepareInput) (mobiledevice.PrepareResult, error) {
			if err := f.prepareErr[in.DeviceID]; err != nil {
				return mobiledevice.PrepareResult{}, err
			}
			result := mobiledevice.PrepareResult{
				Device: map[string]any{"serial": "serial-of-" + in.DeviceID},
				Steps:  map[string]mobiledevice.PreparedMobileDevice{},
			}
			for _, req := range in.Requests {
				result.Steps[req.StepID] = mobiledevice.PreparedMobileDevice{
					RunnerID: path.Dir(req.DeviceID),
					Serial:   "serial-of-" + req.DeviceID,
					Type:     "android_phone",
				}
			}
			return result, nil
		},
		workflow.RegisterOptions{Name: mobiledevice.PrepareWorkflowName},
	)
	env.RegisterWorkflowWithOptions(
		func(_ workflow.Context, in mobiledevice.CleanUpInput) (mobiledevice.CleanUpResult, error) {
			if err := f.cleanupErr[in.DeviceID]; err != nil {
				return mobiledevice.CleanUpResult{}, err
			}
			return f.cleanup[in.DeviceID], nil
		},
		workflow.RegisterOptions{Name: mobiledevice.CleanUpWorkflowName},
	)
	service := nexus.NewService(mobiledevice.ServiceName)
	require.NoError(t, service.Register(
		temporalnexus.MustNewWorkflowRunOperationWithOptions(
			temporalnexus.WorkflowRunOperationOptions[
				mobiledevice.PrepareInput,
				mobiledevice.PrepareResult,
			]{
				Name: mobiledevice.PrepareOperationName,
				Handler: func(
					ctx context.Context,
					in mobiledevice.PrepareInput,
					options nexus.StartOperationOptions,
				) (temporalnexus.WorkflowHandle[mobiledevice.PrepareResult], error) {
					f.prepares = append(f.prepares, in)
					return temporalnexus.ExecuteUntypedWorkflow[mobiledevice.PrepareResult](
						ctx,
						options,
						client.StartWorkflowOptions{ID: in.WorkflowID},
						mobiledevice.PrepareWorkflowName,
						in,
					)
				},
			},
		),
		temporalnexus.MustNewWorkflowRunOperationWithOptions(
			temporalnexus.WorkflowRunOperationOptions[
				mobiledevice.CleanUpInput,
				mobiledevice.CleanUpResult,
			]{
				Name: mobiledevice.CleanUpOperationName,
				Handler: func(
					ctx context.Context,
					in mobiledevice.CleanUpInput,
					options nexus.StartOperationOptions,
				) (temporalnexus.WorkflowHandle[mobiledevice.CleanUpResult], error) {
					f.cleanups = append(f.cleanups, in)
					return temporalnexus.ExecuteUntypedWorkflow[mobiledevice.CleanUpResult](
						ctx,
						options,
						client.StartWorkflowOptions{ID: in.WorkflowID},
						mobiledevice.CleanUpWorkflowName,
						in,
					)
				},
			},
		),
	))
	env.RegisterNexusService(service)
}

// registerRunnerNamespaceLookup answers the device lookup with runner
// namespace "runner-" plus the device's runner name.
func registerRunnerNamespaceLookup(env *testsuite.TestWorkflowEnvironment) {
	env.RegisterActivityWithOptions(
		func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			req, err := workflowengine.DecodePayload[activities.GetMobileDeviceInput](input.Payload)
			if err != nil {
				return workflowengine.ActivityResult{}, err
			}
			runnerID := path.Dir(req.DeviceIdentifier)
			return workflowengine.ActivityResult{Output: activities.MobileDeviceInfo{
				DeviceID:        req.DeviceIdentifier,
				RunnerID:        runnerID,
				RunnerNamespace: testRunnerNamespace(req.DeviceIdentifier),
			}}, nil
		},
		activity.RegisterOptions{Name: activities.GetMobileDeviceActivityName},
	)
}

func testRunnerNamespace(deviceID string) string {
	return "runner-" + path.Base(path.Dir(deviceID))
}

func mobileStep(id, deviceID string) pipeline.StepSpec {
	return pipeline.StepSpec{
		ID:  id,
		Use: mobileAutomationStepUse,
		With: pipeline.StepInputs{Payload: map[string]any{
			"device_id":   deviceID,
			"action_code": "appId: wallet",
			"version_id":  "tenant/wallet/v1",
		}},
	}
}

func TestMobileAutomationSetupHookPreparesEachDeviceOnce(t *testing.T) {
	const (
		deviceA = "tenant/runner-a/device-a"
		deviceB = "tenant/runner-b/device-b"
	)
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: "pipeline-1"})
	registerRunnerNamespaceLookup(env)
	service := &fakeDeviceService{}
	service.register(t, env)
	nexusOptions := recordNexusOptions(env)

	type hookResult struct {
		Steps   []pipeline.StepSpec `json:"steps"`
		RunData map[string]any      `json:"run_data"`
	}
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (hookResult, error) {
			first := pipeline.StepDefinition{StepSpec: mobileStep("first", deviceB)}
			first.OnSuccess = []*pipeline.OnSuccessStepDefinition{
				{StepSpec: mobileStep("nested", deviceA)},
			}
			steps := []pipeline.StepDefinition{
				first,
				{StepSpec: mobileStep("second", deviceA)},
				{StepSpec: mobileStep("third", deviceB)},
			}
			runData := map[string]any{"run_identifier": "tenant/pipeline-run"}
			err := MobileAutomationSetupHook(
				ctx,
				&pipeline.WorkflowDefinition{Steps: steps},
				map[string]any{
					"namespace":                            "tenant",
					mobileDeviceSemaphoreTicketIDConfigKey: "ticket-1",
					// After JSON the ticket's device list is []any.
					mobileDeviceSemaphoreDeviceIDsKey:      []any{deviceA, deviceB},
					mobileDisableAndroidPlayStoreConfigKey: true,
					mobileNoVideoConfigKey:                 true,
				},
				&runData,
				&map[string]any{},
				log.Logger(noopLogger{}),
			)
			if err != nil {
				return hookResult{}, err
			}
			return hookResult{
				Steps: []pipeline.StepSpec{
					steps[0].StepSpec,
					steps[0].OnSuccess[0].StepSpec,
					steps[1].StepSpec,
					steps[2].StepSpec,
				},
				RunData: runData,
			}, nil
		},
		workflow.RegisterOptions{Name: "test-setup-prepares-devices"},
	)

	env.ExecuteWorkflow("test-setup-prepares-devices")
	require.NoError(t, env.GetWorkflowError())
	var result hookResult
	require.NoError(t, env.GetWorkflowResult(&result))

	executionTimeout := PrepareWorkflowOptions(pipeline.RuntimeConfig{}).
		Options.WorkflowExecutionTimeout
	require.Len(t, *nexusOptions, 2)
	for i, deviceID := range []string{deviceB, deviceA} {
		options := (*nexusOptions)[i]
		assert.Equal(t, deviceOperationScheduleToStart, options.ScheduleToStartTimeout)
		assert.Equal(t, executionTimeout, options.ScheduleToCloseTimeout)
		assert.Equal(t, mobiledevice.PrepareSummary(deviceID), options.Summary)
	}
	// One Prepare per device, in order of first use.
	require.Len(t, service.prepares, 2)
	wantSteps := map[string][]string{deviceB: {"first", "third"}, deviceA: {"nested", "second"}}
	for i, deviceID := range []string{deviceB, deviceA} {
		in := service.prepares[i]
		assert.Equal(t, deviceID, in.DeviceID)
		assert.Equal(t, mobiledevice.PrepareWorkflowID("pipeline-1", deviceID), in.WorkflowID)
		assert.Equal(t, "tenant/pipeline-run", in.RunIdentifier)
		assert.True(t, in.DisablePlayStore)
		assert.True(t, in.NoVideo)
		assert.Equal(t, executionTimeout, in.ExecutionTimeout)
		assert.Equal(t, testPipelineRun, in.Pipeline)
		assert.Empty(t, in.Workflow.Config)
		require.NotNil(t, in.Workflow.ActivityOptions)
		var stepIDs []string
		for _, req := range in.Requests {
			assert.Equal(t, deviceID, req.DeviceID)
			assert.Equal(t, "tenant/wallet/v1", req.VersionIdentifier)
			stepIDs = append(stepIDs, req.StepID)
		}
		assert.Equal(t, wantSteps[deviceID], stepIDs)
	}

	stepDevices := []string{deviceB, deviceA, deviceA, deviceB}
	for i, step := range result.Steps {
		deviceID := stepDevices[i]
		runnerID := path.Dir(deviceID)
		assert.Equal(t, deviceID, step.With.Payload["device_id"], step.ID)
		assert.Equal(t, "serial-of-"+deviceID, step.With.Payload["serial"], step.ID)
		assert.Equal(t, "android_phone", step.With.Payload["type"], step.ID)
		assert.Equal(
			t,
			testRunnerNamespace(deviceID),
			step.With.Config[runnerNamespaceConfigKey],
			step.ID,
		)
		assert.Equal(t, map[string]any{
			"namespace":   testPipelineRun.Namespace,
			"workflow_id": testPipelineRun.WorkflowID,
			"run_id":      testPipelineRun.RunID,
		}, step.With.Config[pipelineRunConfigKey], step.ID)
		assert.Equal(
			t,
			executionTimeout.String(),
			step.With.Config[pipelineExecutionTimeoutConfigKey],
			step.ID,
		)
		assert.Equal(t, mobileRunnerTaskQueue(runnerID), step.With.Config["taskqueue"], step.ID)
		assert.Equal(t, runnerID, step.With.Config["runner_id"], step.ID)
		assert.Equal(t, step.ID, step.With.Config[workflowengine.StepIDConfigKey], step.ID)
		assert.Equal(t, "tenant/pipeline-run", step.With.Config["run_identifier"], step.ID)
	}

	devices := workflowengine.AsMap(result.RunData["setted_devices"])
	require.Len(t, devices, 2)
	for _, deviceID := range []string{deviceA, deviceB} {
		assert.Equal(t, map[string]any{
			"serial":                 "serial-of-" + deviceID,
			runnerNamespaceConfigKey: testRunnerNamespace(deviceID),
		}, devices[deviceID])
	}
}

// TestMobileAutomationSetupHookStopsAtFailedPrepare: a device whose Prepare
// fails was cleaned by it; the devices prepared before it stay recorded for
// MobileDevicesCleanupHook.
func TestMobileAutomationSetupHookStopsAtFailedPrepare(t *testing.T) {
	const (
		deviceA = "tenant/runner-a/device-a"
		deviceB = "tenant/runner-b/device-b"
		deviceC = "tenant/runner-c/device-c"
	)
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: "pipeline-1"})
	registerRunnerNamespaceLookup(env)
	service := &fakeDeviceService{prepareErr: map[string]error{
		deviceB: temporal.NewNonRetryableApplicationError("setup failed", "CRE1", nil),
	}}
	service.register(t, env)

	type outcome struct {
		Err     string         `json:"err"`
		Devices map[string]any `json:"devices"`
	}
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (outcome, error) {
			runData := map[string]any{"run_identifier": "tenant/pipeline-run"}
			var out outcome
			if err := MobileAutomationSetupHook(
				ctx,
				&pipeline.WorkflowDefinition{Steps: []pipeline.StepDefinition{
					{StepSpec: mobileStep("first", deviceA)},
					{StepSpec: mobileStep("second", deviceB)},
					{StepSpec: mobileStep("third", deviceC)},
				}},
				map[string]any{
					"namespace":                            "tenant",
					mobileDeviceSemaphoreTicketIDConfigKey: "ticket-1",
					mobileDeviceSemaphoreDeviceIDsKey:      []string{deviceA, deviceB, deviceC},
				},
				&runData,
				&map[string]any{},
				log.Logger(noopLogger{}),
			); err != nil {
				out.Err = err.Error()
			}
			out.Devices = workflowengine.AsMap(runData["setted_devices"])
			return out, nil
		},
		workflow.RegisterOptions{Name: "test-setup-failed-prepare"},
	)

	env.ExecuteWorkflow("test-setup-failed-prepare")
	require.NoError(t, env.GetWorkflowError())
	var out outcome
	require.NoError(t, env.GetWorkflowResult(&out))

	assert.Contains(t, out.Err, "setup failed")
	require.Len(t, service.prepares, 2)
	assert.Equal(t, deviceA, service.prepares[0].DeviceID)
	assert.Equal(t, deviceB, service.prepares[1].DeviceID)
	assert.Equal(t, []string{deviceA}, slices.Sorted(maps.Keys(out.Devices)))
}

// TestMobileAutomationSetupHookRefusesDeviceOutsideTicket: a child pipeline
// inherits its parent's ticket, but the queue locked only the parent's devices.
func TestMobileAutomationSetupHookRefusesDeviceOutsideTicket(t *testing.T) {
	const (
		lockedDevice = "tenant/runner-a/device-a"
		otherDevice  = "other-org/runner-b/device-b"
	)
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: "pipeline-1"})
	registerRunnerNamespaceLookup(env)
	service := &fakeDeviceService{}
	service.register(t, env)
	nexusOptions := recordNexusOptions(env)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (string, error) {
			runData := map[string]any{"run_identifier": "tenant/pipeline-run"}
			err := MobileAutomationSetupHook(
				ctx,
				&pipeline.WorkflowDefinition{Steps: []pipeline.StepDefinition{
					{StepSpec: mobileStep("child-step", otherDevice)},
				}},
				map[string]any{
					"namespace":                            "tenant",
					mobileDeviceSemaphoreTicketIDConfigKey: "ticket-1",
					mobileDeviceSemaphoreDeviceIDsKey:      []any{lockedDevice},
				},
				&runData,
				&map[string]any{},
				log.Logger(noopLogger{}),
			)
			if err == nil {
				return "", nil
			}
			var appErr *temporal.ApplicationError
			if !errors.As(err, &appErr) || !appErr.NonRetryable() {
				return "retryable: " + err.Error(), nil
			}
			return appErr.Type() + ": " + err.Error(), nil
		},
		workflow.RegisterOptions{Name: "test-setup-device-outside-ticket"},
	)

	env.ExecuteWorkflow("test-setup-device-outside-ticket")
	require.NoError(t, env.GetWorkflowError())
	var got string
	require.NoError(t, env.GetWorkflowResult(&got))

	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
	assert.True(t, strings.HasPrefix(got, errCode.Code+": "), got)
	assert.Contains(
		t,
		got,
		"device "+otherDevice+" is not covered by the run's queue ticket: "+
			"a child pipeline may use only devices its parent pipeline lists",
	)
	assert.Empty(t, *nexusOptions)
	assert.Empty(t, service.prepares)
}

func TestMobileDevicesCleanupHookCleansEachPreparedDevice(t *testing.T) {
	const (
		deviceA       = "tenant/runner-a/device-a"
		deviceB       = "tenant/runner-b/device-b"
		deviceSkipped = "tenant/runner-c/device-c"
		deviceLocal   = "tenant/runner-d/device-d"
	)
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: "pipeline-1"})
	service := &fakeDeviceService{
		cleanup: map[string]mobiledevice.CleanUpResult{
			deviceA: {
				ResultVideoURLs: []string{"video-a", "video-shared"},
				ScreenshotURLs:  []string{"frame-a"},
				Warnings:        []string{"device a warning"},
				Errors:          []string{"device a stop failed"},
			},
		},
		cleanupErr: map[string]error{
			deviceB: temporal.NewNonRetryableApplicationError("runner gone", "CRE1", nil),
		},
	}
	service.register(t, env)
	nexusOptions := recordNexusOptions(env)

	type outcome struct {
		Err     string         `json:"err"`
		Details map[string]any `json:"details"`
		Output  map[string]any `json:"output"`
	}
	device := func(deviceID string) map[string]any {
		return map[string]any{
			"serial":                 "serial-of-" + deviceID,
			runnerNamespaceConfigKey: testRunnerNamespace(deviceID),
		}
	}
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (outcome, error) {
			output := map[string]any{"result_video_urls": []string{"video-shared"}}
			err := MobileDevicesCleanupHook(
				ctx,
				nil,
				&workflow.ActivityOptions{StartToCloseTimeout: time.Second},
				map[string]any{"namespace": "tenant"},
				map[string]any{
					"run_identifier": "tenant/pipeline-run",
					"setted_devices": map[string]any{
						deviceA:       device(deviceA),
						deviceB:       device(deviceB),
						deviceSkipped: device(deviceSkipped),
						// A device without runner namespace was never prepared.
						deviceLocal: map[string]any{"serial": "local"},
					},
					pipelineCancellationPolicyRunDataKey: pipeline.PipelineCancellationPolicy{
						SkipDeviceCleanup:    true,
						SkipDeviceCleanupIDs: []string{deviceSkipped},
					},
				},
				&output,
			)
			out := outcome{Output: output}
			if err != nil {
				out.Err = err.Error()
				out.Details = workflowengine.AsMap(workflowengine.ParseWorkflowError(err).Details)
			}
			return out, nil
		},
		workflow.RegisterOptions{Name: "test-devices-cleanup-hook"},
	)

	env.ExecuteWorkflow("test-devices-cleanup-hook")
	require.NoError(t, env.GetWorkflowError())
	var out outcome
	require.NoError(t, env.GetWorkflowResult(&out))

	executionTimeout := PrepareWorkflowOptions(pipeline.RuntimeConfig{}).
		Options.WorkflowExecutionTimeout
	require.Len(t, *nexusOptions, 2)
	for i, deviceID := range []string{deviceA, deviceB} {
		options := (*nexusOptions)[i]
		assert.Equal(t, deviceOperationScheduleToStart, options.ScheduleToStartTimeout)
		assert.Equal(t, executionTimeout, options.ScheduleToCloseTimeout)
		assert.Equal(t, mobiledevice.CleanUpSummary(deviceID), options.Summary)
	}
	require.Len(t, service.cleanups, 2)
	for i, deviceID := range []string{deviceA, deviceB} {
		in := service.cleanups[i]
		assert.Equal(t, deviceID, in.DeviceID)
		assert.Equal(t, mobiledevice.CleanUpWorkflowID("pipeline-1", deviceID), in.WorkflowID)
		assert.Equal(t, device(deviceID), in.Device)
		assert.Equal(t, "tenant/pipeline-run", in.RunIdentifier)
		assert.Equal(t, executionTimeout, in.ExecutionTimeout)
		assert.Equal(t, testPipelineRun, in.Pipeline)
		assert.Empty(t, in.Workflow.Config)
	}

	assert.Equal(
		t,
		[]any{"video-shared", "video-a"},
		out.Output["result_video_urls"],
	)
	assert.Equal(t, []any{"frame-a"}, out.Output["screenshot_urls"])
	assert.Equal(t, []any{
		"device a warning",
		"device cleanup skipped for " + deviceSkipped +
			" because pipeline cancellation policy requested it",
	}, out.Output[cleanupWarningsOutputKey])
	assert.Contains(t, out.Err, "one or more errors occurred during mobile automation cleanup")
	errs := workflowengine.AsSliceOfStrings(out.Details["payload"])
	require.Len(t, errs, 2)
	assert.Equal(t, "device a stop failed", errs[0])
	assert.Contains(t, errs[1], "runner gone")
}

// TestPrepareDeviceWorkflowCaptureOptions: Prepare sets up a physical
// Android phone for every step, always starts the device log, records the
// screen unless no_video is set and disables the Play Store only when asked.
func TestPrepareDeviceWorkflowCaptureOptions(t *testing.T) {
	const deviceID = "tenant/runner-1/device-1"
	tests := []struct {
		name                 string
		disablePlayStore     bool
		noVideo              bool
		wantRecordings       int
		wantPlayStoreDisable int
	}{
		{name: "records the screen by default", wantRecordings: 1},
		{name: "no_video skips the recording", noVideo: true},
		{
			name:                 "disables the Play Store when asked",
			disablePlayStore:     true,
			wantRecordings:       1,
			wantPlayStoreDisable: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			env.RegisterWorkflowWithOptions(
				PrepareDeviceWorkflow,
				workflow.RegisterOptions{Name: mobiledevice.PrepareWorkflowName},
			)
			registerStubActivity(env, activities.GetMobileDeviceActivityName)
			runnerHTTPActivity := registerMobileRunnerHTTPActivity(env)
			setupActivity := activities.NewSetupMobileDeviceActivity()
			installActivity := activities.NewApkInstallActivity()
			postInstallActivity := activities.NewApkPostInstallChecksActivity()
			listAppsActivity := activities.NewPreparePhysicalAndroidAppsActivity()
			recordActivity := activities.NewStartRecordingActivity()
			disablePlayStoreActivity := activities.NewDisableAndroidPlayStoreActivity()
			startLogActivity := activities.NewStartAndroidDeviceLogActivity()
			for _, a := range []workflowengine.ExecutableActivity{
				setupActivity,
				installActivity,
				postInstallActivity,
				listAppsActivity,
				recordActivity,
				disablePlayStoreActivity,
				startLogActivity,
			} {
				env.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: a.Name()})
			}

			onGetMobileDevice(env, deviceID, map[string]any{
				"runner_id":  "tenant/runner-1",
				"runner_url": "https://runner.example",
				"type":       "physical",
				"serial":     "serial-1",
			})
			env.OnActivity(setupActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: map[string]any{
					"screen_prepared":     true,
					"original_stay_awake": "0",
				}}, nil)
			env.OnActivity(listAppsActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: []string{"com.android.settings"}}, nil)
			env.OnActivity(runnerHTTPActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: map[string]any{
					"body": map[string]any{
						"installer_path": "/tmp/app.apk",
						"version_id":     "tenant/wallet/v1",
					},
				}}, nil)
			installed := workflowengine.ActivityResult{
				Output: map[string]any{"package_id": "pkg-1"},
			}
			env.OnActivity(installActivity.Name(), mock.Anything, mock.Anything).
				Return(installed, nil)
			env.OnActivity(postInstallActivity.Name(), mock.Anything, mock.Anything).
				Return(installed, nil)
			env.OnActivity(startLogActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: map[string]any{
					"log_path":        "/tmp/log.txt",
					"log_process_pid": float64(13),
				}}, nil)
			env.OnActivity(recordActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: map[string]any{
					"recording_process_pid": float64(11),
					"ffmpeg_process_pid":    float64(12),
					"video_path":            "/tmp/video.mp4",
				}}, nil)
			env.OnActivity(disablePlayStoreActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: map[string]any{"message": "disabled"}}, nil)

			request := func(stepID string) mobiledevice.MobileDeviceRequest {
				return mobiledevice.MobileDeviceRequest{
					DeviceID:          deviceID,
					VersionIdentifier: "tenant/wallet/v1",
					StepID:            stepID,
					RunIdentifier:     "tenant/pipeline-run",
				}
			}
			env.ExecuteWorkflow(mobiledevice.PrepareWorkflowName, mobiledevice.PrepareInput{
				WorkflowID: mobiledevice.PrepareWorkflowID("pipeline-1", deviceID),
				DeviceID:   deviceID,
				Requests: []mobiledevice.MobileDeviceRequest{
					request("step-1"),
					request("step-2"),
				},
				RunIdentifier:    "tenant/pipeline-run",
				DisablePlayStore: tt.disablePlayStore,
				NoVideo:          tt.noVideo,
				Pipeline:         testPipelineRun,
				Workflow:         deviceWorkflowInput(),
			})
			require.NoError(t, env.GetWorkflowError())
			var result mobiledevice.PrepareResult
			require.NoError(t, env.GetWorkflowResult(&result))

			env.AssertActivityNumberOfCalls(t, setupActivity.Name(), 1)
			env.AssertActivityNumberOfCalls(t, startLogActivity.Name(), 1)
			env.AssertActivityNumberOfCalls(t, recordActivity.Name(), tt.wantRecordings)
			env.AssertActivityNumberOfCalls(
				t,
				disablePlayStoreActivity.Name(),
				tt.wantPlayStoreDisable,
			)

			require.Len(t, result.Steps, 2)
			for _, stepID := range []string{"step-1", "step-2"} {
				prepared := result.Steps[stepID]
				assert.Equal(t, "tenant/runner-1", prepared.RunnerID, stepID)
				assert.Equal(t, "serial-1", prepared.Serial, stepID)
				assert.Equal(t, "android_phone", prepared.Type, stepID)
			}
			device := result.Device
			assert.Equal(t, "serial-1", device["serial"])
			assert.Equal(t, true, device["device_log"])
			assert.Equal(t, "/tmp/log.txt", device["log_path"])
			assert.Equal(t, tt.wantRecordings == 1, device["recording"])
			if tt.noVideo {
				assert.NotContains(t, device, "video_path")
			} else {
				assert.Equal(t, "/tmp/video.mp4", device["video_path"])
			}
			if tt.disablePlayStore {
				assert.Equal(t, true, device["play_store_disabled"])
			} else {
				assert.NotContains(t, device, "play_store_disabled")
			}
		})
	}
}
