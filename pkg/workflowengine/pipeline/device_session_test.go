// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"encoding/json"
	"path"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/temporalid"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesession"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

const sessionDeviceID = "tenant/runner-1/device-1"

var sessionPipelineRun = mobiledevicesession.PipelineRunRef{
	Namespace:  "tenant",
	WorkflowID: "pipeline-1",
	RunID:      "run-1",
}

// deviceSessionAnswers records what the device session's updates and query
// answered.
type deviceSessionAnswers struct {
	readyDone   bool
	ready       DeviceSessionReady
	readyErr    error
	releaseDone bool
	cleanup     mobiledevicesession.Cleanup
	releaseErr  error
	queried     map[string]any
}

// decodeAs converts an update result to T through JSON, as a client would.
func decodeAs[T any](t *testing.T, v any) T {
	t.Helper()
	var out T
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

func newDeviceSessionEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	w := NewDeviceSessionWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})
	return env
}

func sendDeviceSessionReady(
	t *testing.T,
	env *testsuite.TestWorkflowEnvironment,
	answers *deviceSessionAnswers,
) {
	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflow(
			mobiledevicesession.ReadyUpdate,
			"ready",
			&testsuite.TestUpdateCallback{
				OnReject: func(err error) { answers.readyErr = err },
				OnComplete: func(result any, err error) {
					answers.readyDone = true
					answers.readyErr = err
					if err == nil {
						answers.ready = decodeAs[DeviceSessionReady](t, result)
					}
				},
			},
		)
	}, time.Second)
}

func sendDeviceSessionRelease(
	t *testing.T,
	env *testsuite.TestWorkflowEnvironment,
	answers *deviceSessionAnswers,
	after time.Duration,
	skipCleanup bool,
) {
	env.RegisterDelayedCallback(func() {
		env.UpdateWorkflow(
			mobiledevicesession.ReleaseUpdate,
			"release",
			&testsuite.TestUpdateCallback{
				OnReject: func(err error) { answers.releaseErr = err },
				OnComplete: func(result any, err error) {
					answers.releaseDone = true
					answers.releaseErr = err
					if err == nil {
						answers.cleanup = decodeAs[mobiledevicesession.Cleanup](t, result)
					}
				},
			},
			mobiledevicesession.Release{SkipCleanup: skipCleanup},
		)
	}, after)
}

func queryDeviceSession(
	t *testing.T,
	env *testsuite.TestWorkflowEnvironment,
	answers *deviceSessionAnswers,
	after time.Duration,
) {
	env.RegisterDelayedCallback(func() {
		value, err := env.QueryWorkflow(PipelineMobileDevicesQuery)
		require.NoError(t, err)
		require.NoError(t, value.Get(&answers.queried))
	}, after)
}

func deviceSessionInput(payload DeviceSessionPayload) workflowengine.WorkflowInput {
	return workflowengine.WorkflowInput{
		Payload: payload,
		Config:  map[string]any{"app_url": "https://app.example"},
		ActivityOptions: &workflow.ActivityOptions{
			StartToCloseTimeout: time.Second,
		},
	}
}

func sessionRequest(stepID string) MobileDeviceRequest {
	return MobileDeviceRequest{
		DeviceID:          sessionDeviceID,
		VersionIdentifier: "tenant/wallet/v1",
		StepID:            stepID,
		RunIdentifier:     "tenant/pipeline-run",
	}
}

// androidPhoneSession names the runner activities a session of a physical
// Android phone runs.
type androidPhoneSession struct {
	setup            string
	startLog         string
	record           string
	disablePlayStore string
	stopRecording    string
	cleanupPayloads  *[]map[string]any
	checks           *[]activities.CheckPipelineRunPayload
}

// mockAndroidPhoneSession mocks preparing, capturing and cleaning a physical
// Android phone whose evidence the runner stores as video-url and frame-url.
// It also answers the pipeline run checks with running, in order.
func mockAndroidPhoneSession(
	env *testsuite.TestWorkflowEnvironment,
	running []bool,
) androidPhoneSession {
	// Every activity is registered before the first mock.
	checks := recordPipelineRunChecks(env, running)
	registerStubActivity(env, activities.GetMobileDeviceActivityName)
	runnerHTTPActivity := registerMobileRunnerHTTPActivity(env)
	setupActivity := activities.NewSetupMobileDeviceActivity()
	installActivity := activities.NewApkInstallActivity()
	postInstallActivity := activities.NewApkPostInstallChecksActivity()
	listAppsActivity := activities.NewPreparePhysicalAndroidAppsActivity()
	recordActivity := activities.NewStartRecordingActivity()
	disablePlayStoreActivity := activities.NewDisableAndroidPlayStoreActivity()
	stopRecordingActivity := activities.NewStopRecordingActivity()
	startLogActivity := activities.NewStartAndroidDeviceLogActivity()
	stopLogActivity := activities.NewStopAndroidDeviceLogActivity()
	for _, a := range []workflowengine.ExecutableActivity{
		setupActivity,
		installActivity,
		postInstallActivity,
		listAppsActivity,
		recordActivity,
		disablePlayStoreActivity,
		stopRecordingActivity,
		startLogActivity,
		stopLogActivity,
	} {
		env.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: a.Name()})
	}
	cleanupPayloads := []map[string]any{}
	var cleanupQueues []string
	recordCleanupPayloads(env, &cleanupPayloads, &cleanupQueues)

	env.OnActivity(startLogActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"log_path":        "/tmp/log.txt",
			"log_process_pid": float64(13),
		}}, nil)
	env.OnActivity(stopLogActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{}}, nil)

	onGetMobileDevice(env, sessionDeviceID, map[string]any{
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
	env.OnActivity(runnerHTTPActivity.Name(), mock.Anything, runnerPathMatcher("/credimi/installer-action")).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"body": map[string]any{"installer_path": "/tmp/app.apk", "version_id": "ver-1"},
		}}, nil)
	env.OnActivity(runnerHTTPActivity.Name(), mock.Anything, runnerPathMatcher("/credimi/pipeline-result")).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"body": map[string]any{
				"result_urls":     []string{"video-url"},
				"screenshot_urls": []string{"frame-url"},
			},
		}}, nil)
	env.OnActivity(installActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{"package_id": "pkg-1"}}, nil)
	env.OnActivity(postInstallActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{"package_id": "pkg-1"}}, nil)
	env.OnActivity(recordActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"recording_process_pid": float64(11),
			"ffmpeg_process_pid":    float64(12),
			"video_path":            "/tmp/video.mp4",
		}}, nil)
	env.OnActivity(disablePlayStoreActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{"message": "disabled"}}, nil)
	env.OnActivity(stopRecordingActivity.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: map[string]any{
			"last_frame_path": "/tmp/last.png",
		}}, nil)

	return androidPhoneSession{
		setup:            setupActivity.Name(),
		startLog:         startLogActivity.Name(),
		record:           recordActivity.Name(),
		disablePlayStore: disablePlayStoreActivity.Name(),
		stopRecording:    stopRecordingActivity.Name(),
		cleanupPayloads:  &cleanupPayloads,
		checks:           checks,
	}
}

func runnerPathMatcher(path string) any {
	return mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
		payload, ok := input.Payload.(map[string]any)
		return ok && workflowengine.AsString(payload["path"]) == path
	})
}

// recordPipelineRunChecks answers the pipeline run checks with running, in
// order, and records their payloads.
func recordPipelineRunChecks(
	env *testsuite.TestWorkflowEnvironment,
	running []bool,
) *[]activities.CheckPipelineRunPayload {
	checks := []activities.CheckPipelineRunPayload{}
	env.RegisterActivityWithOptions(
		func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			payload, err := workflowengine.DecodePayload[activities.CheckPipelineRunPayload](
				input.Payload,
			)
			if err != nil {
				return workflowengine.ActivityResult{}, err
			}
			answer := running[min(len(checks), len(running)-1)]
			checks = append(checks, payload)
			return workflowengine.ActivityResult{
				Output: activities.CheckPipelineRunResult{Running: answer},
			}, nil
		},
		activity.RegisterOptions{Name: activities.CheckPipelineRunActivityName},
	)
	return &checks
}

func TestDeviceSessionWorkflowPreparesHoldsAndCleansDevice(t *testing.T) {
	tests := []struct {
		name string
		// release, when set, is when the pipeline releases the device.
		release            time.Duration
		skipCleanup        bool
		running            []bool
		wantChecks         int
		wantCleanups       int
		wantVideoURLs      []string
		wantScreenshotURLs []string
	}{
		{
			name:               "release cleans the device and returns its evidence",
			release:            time.Minute,
			running:            []bool{true},
			wantCleanups:       1,
			wantVideoURLs:      []string{"video-url"},
			wantScreenshotURLs: []string{"frame-url"},
		},
		{
			name:        "release with skip_cleanup leaves the device as it is",
			release:     time.Minute,
			skipCleanup: true,
			running:     []bool{true},
		},
		{
			name:               "pipeline run that ended releases and cleans the device",
			running:            []bool{true, false},
			wantChecks:         2,
			wantCleanups:       1,
			wantVideoURLs:      []string{"video-url"},
			wantScreenshotURLs: []string{"frame-url"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newDeviceSessionEnv(t)
			phone := mockAndroidPhoneSession(env, tt.running)
			checks := phone.checks

			answers := &deviceSessionAnswers{}
			sendDeviceSessionReady(t, env, answers)
			queryDeviceSession(t, env, answers, 30*time.Second)
			if tt.release > 0 {
				sendDeviceSessionRelease(t, env, answers, tt.release, tt.skipCleanup)
			}

			env.ExecuteWorkflow(
				mobiledevicesession.WorkflowName,
				deviceSessionInput(DeviceSessionPayload{
					DeviceID: sessionDeviceID,
					Requests: []MobileDeviceRequest{
						sessionRequest("step-1"),
						sessionRequest("step-2"),
					},
					RunIdentifier: "tenant/pipeline-run",
					Pipeline:      sessionPipelineRun,
				}),
			)
			require.True(t, env.IsWorkflowCompleted())
			require.NoError(t, env.GetWorkflowError())

			require.True(t, answers.readyDone)
			require.NoError(t, answers.readyErr)
			require.Len(t, answers.ready.Prepared, 2)
			for _, stepID := range []string{"step-1", "step-2"} {
				prepared := answers.ready.Prepared[stepID]
				assert.Equal(t, "tenant/runner-1", prepared.RunnerID, stepID)
				assert.Equal(t, "serial-1", prepared.Serial, stepID)
				assert.Equal(t, "android_phone", prepared.Type, stepID)
			}
			assert.Equal(t, "serial-1", answers.ready.Device["serial"])
			assert.Equal(t, true, answers.ready.Device["recording"])
			assert.Equal(t, true, answers.ready.Device["device_log"])

			// The live view reads the device through the query.
			require.Contains(t, answers.queried, sessionDeviceID)
			queried := workflowengine.AsMap(answers.queried[sessionDeviceID])
			assert.Equal(t, "serial-1", queried["serial"])
			assert.Equal(t, "/tmp/video.mp4", queried["video_path"])

			env.AssertActivityNumberOfCalls(t, phone.setup, 1)
			env.AssertActivityNumberOfCalls(t, phone.startLog, 1)
			env.AssertActivityNumberOfCalls(t, phone.record, 1)
			env.AssertActivityNumberOfCalls(t, phone.disablePlayStore, 0)

			require.Len(t, *checks, tt.wantChecks)
			for _, check := range *checks {
				assert.Equal(t, sessionPipelineRun, check.Pipeline)
				assert.Equal(t, "default-test-namespace", check.RunnerNamespace)
			}
			require.Len(t, *phone.cleanupPayloads, tt.wantCleanups)
			if tt.wantCleanups > 0 {
				assert.Equal(t, "serial-1", (*phone.cleanupPayloads)[0]["serial"])
				env.AssertActivityNumberOfCalls(t, phone.stopRecording, 1)
			} else {
				env.AssertActivityNumberOfCalls(t, phone.stopRecording, 0)
			}

			var result workflowengine.WorkflowResult
			require.NoError(t, env.GetWorkflowResult(&result))
			cleanup := decodeAs[mobiledevicesession.Cleanup](t, result.Output)
			assert.Equal(t, tt.wantVideoURLs, cleanup.ResultVideoURLs)
			assert.Equal(t, tt.wantScreenshotURLs, cleanup.ScreenshotURLs)
			assert.Empty(t, cleanup.Errors)
			if tt.release > 0 {
				require.True(t, answers.releaseDone)
				require.NoError(t, answers.releaseErr)
				assert.Equal(t, cleanup, answers.cleanup)
			}
		})
	}
}

func TestDeviceSessionWorkflowCaptureOptions(t *testing.T) {
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
			name:                 "disables the Play Store when configured",
			disablePlayStore:     true,
			wantRecordings:       1,
			wantPlayStoreDisable: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newDeviceSessionEnv(t)
			phone := mockAndroidPhoneSession(env, []bool{true})
			answers := &deviceSessionAnswers{}
			sendDeviceSessionReady(t, env, answers)
			sendDeviceSessionRelease(t, env, answers, time.Minute, true)

			env.ExecuteWorkflow(
				mobiledevicesession.WorkflowName,
				deviceSessionInput(DeviceSessionPayload{
					DeviceID:         sessionDeviceID,
					Requests:         []MobileDeviceRequest{sessionRequest("step-1")},
					RunIdentifier:    "tenant/pipeline-run",
					DisablePlayStore: tt.disablePlayStore,
					NoVideo:          tt.noVideo,
					Pipeline:         sessionPipelineRun,
				}),
			)
			require.NoError(t, env.GetWorkflowError())
			require.NoError(t, answers.readyErr)

			env.AssertActivityNumberOfCalls(t, phone.startLog, 1)
			env.AssertActivityNumberOfCalls(t, phone.record, tt.wantRecordings)
			env.AssertActivityNumberOfCalls(t, phone.disablePlayStore, tt.wantPlayStoreDisable)
			device := answers.ready.Device
			assert.Equal(t, true, device["device_log"])
			assert.Equal(t, "/tmp/log.txt", device["log_path"])
			assert.Equal(t, tt.wantRecordings == 1, device["recording"])
			if tt.noVideo {
				assert.NotContains(t, device, "video_path")
			}
			if tt.disablePlayStore {
				assert.Equal(t, true, device["play_store_disabled"])
			} else {
				assert.NotContains(t, device, "play_store_disabled")
			}
		})
	}
}

// TestDeviceSessionWorkflowCleansDeviceWhoseSetupFails covers emulators whose
// setup fails: the ready update returns the setup error, and the release
// still deletes a clone that started, or warns about one that never did.
func TestDeviceSessionWorkflowCleansDeviceWhoseSetupFails(t *testing.T) {
	tests := []struct {
		name         string
		startErr     error
		wantCleanups int
		wantWarnings []string
	}{
		{name: "deletes the started emulator", wantCleanups: 1},
		{
			name:     "warns about an emulator that never started",
			startErr: assert.AnError,
			wantWarnings: []string{
				"device cleanup skipped for runner-1/device-1 because it never started",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newDeviceSessionEnv(t)
			var payloads []map[string]any
			var taskQueues []string
			recordCleanupPayloads(env, &payloads, &taskQueues)
			recordPipelineRunChecks(env, []bool{true})
			registerStubActivity(env, activities.GetMobileDeviceActivityName)
			startActivity := activities.NewSetupMobileDeviceActivity()
			listAppsActivity := activities.NewListInstalledAppsActivity()
			for _, a := range []workflowengine.ExecutableActivity{startActivity, listAppsActivity} {
				env.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: a.Name()})
			}
			onGetMobileDevice(env, "runner-1/device-1", map[string]any{
				"runner_id":  "tenant/runner-1",
				"runner_url": "https://runner.example",
				"type":       "emulator",
			})
			env.OnActivity(startActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{Output: map[string]any{
					"serial": "emulator-5554",
					"name":   "clone-1",
				}}, tt.startErr)
			env.OnActivity(listAppsActivity.Name(), mock.Anything, mock.Anything).
				Return(workflowengine.ActivityResult{}, assert.AnError)

			answers := &deviceSessionAnswers{}
			sendDeviceSessionReady(t, env, answers)
			sendDeviceSessionRelease(t, env, answers, time.Minute, false)

			env.ExecuteWorkflow(
				mobiledevicesession.WorkflowName,
				deviceSessionInput(DeviceSessionPayload{
					DeviceID: "runner-1/device-1",
					Requests: []MobileDeviceRequest{{
						DeviceID:          "runner-1/device-1",
						VersionIdentifier: "tenant/wallet/v1",
						StepID:            "step-1",
						RunIdentifier:     "tenant/run-1",
					}},
					RunIdentifier: "tenant/run-1",
					Pipeline:      sessionPipelineRun,
				}),
			)
			require.NoError(t, env.GetWorkflowError())

			require.True(t, answers.readyDone)
			require.Error(t, answers.readyErr)
			assert.Contains(t, answers.readyErr.Error(), assert.AnError.Error())

			require.True(t, answers.releaseDone)
			require.NoError(t, answers.releaseErr)
			require.Len(t, payloads, tt.wantCleanups)
			if tt.wantCleanups > 0 {
				assert.Equal(t, "clone-1", payloads[0]["name"])
				assert.Equal(t, "emulator-5554", payloads[0]["serial"])
				assert.Equal(t, "android_emulator", payloads[0]["type"])
			}
			assert.Equal(t, tt.wantWarnings, answers.cleanup.Warnings)
			assert.Empty(t, answers.cleanup.Errors)
		})
	}
}

func TestDeviceSessionWorkflowReportsCleanupErrors(t *testing.T) {
	env := newDeviceSessionEnv(t)
	mockAndroidPhoneSession(env, []bool{true})
	answers := &deviceSessionAnswers{}
	sendDeviceSessionReady(t, env, answers)
	sendDeviceSessionRelease(t, env, answers, time.Minute, false)

	input := deviceSessionInput(DeviceSessionPayload{
		DeviceID: sessionDeviceID,
		Requests: []MobileDeviceRequest{sessionRequest("step-1")},
		// Without a run identifier the cleanup cannot store the evidence.
		Pipeline: sessionPipelineRun,
	})
	env.ExecuteWorkflow(mobiledevicesession.WorkflowName, input)
	require.NoError(t, env.GetWorkflowError())

	require.True(t, answers.releaseDone)
	require.NoError(t, answers.releaseErr)
	require.Len(t, answers.cleanup.Errors, 1)
	assert.Contains(
		t,
		answers.cleanup.Errors[0],
		"one or more errors occurred during mobile automation cleanup",
	)
}

func TestDeviceSessionWorkflowRejectsInvalidPayload(t *testing.T) {
	env := newDeviceSessionEnv(t)
	env.ExecuteWorkflow(mobiledevicesession.WorkflowName, workflowengine.WorkflowInput{
		Payload: map[string]any{"requests": "not a list"},
	})
	err := env.GetWorkflowError()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid device session payload")
}

// sessionActivities records the device session activities the pipeline runs.
type sessionActivities struct {
	starts   []mobiledevicesession.StartDeviceSessionPayload
	awaits   []mobiledevicesession.RunnerWorkflowRef
	releases []activities.ReleaseDeviceSessionPayload
}

// registerSessionActivities registers the device lookup and the device
// session activities. A device's runner namespace is "runner-ns-" plus its
// runner; awaitErr fails the wait of that device; release answers by
// session workflow ID.
func registerSessionActivities(
	env *testsuite.TestWorkflowEnvironment,
	awaitErr map[string]error,
	release map[string]mobiledevicesession.Cleanup,
	releaseErr map[string]error,
) *sessionActivities {
	recorded := &sessionActivities{}
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
				RunnerNamespace: "runner-ns-" + path.Base(runnerID),
			}}, nil
		},
		activity.RegisterOptions{Name: activities.GetMobileDeviceActivityName},
	)
	env.RegisterActivityWithOptions(
		func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			payload, err := workflowengine.DecodePayload[mobiledevicesession.StartDeviceSessionPayload](
				input.Payload,
			)
			if err != nil {
				return workflowengine.ActivityResult{}, err
			}
			recorded.starts = append(recorded.starts, payload)
			return workflowengine.ActivityResult{Output: mobiledevicesession.RunnerWorkflowRef{
				Namespace:  payload.Namespace,
				WorkflowID: payload.WorkflowID,
				RunID:      "session-run-" + payload.DeviceID,
			}}, nil
		},
		activity.RegisterOptions{Name: mobiledevicesession.StartDeviceSessionActivityName},
	)
	env.RegisterActivityWithOptions(
		func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			ref, err := workflowengine.DecodePayload[mobiledevicesession.RunnerWorkflowRef](
				input.Payload,
			)
			if err != nil {
				return workflowengine.ActivityResult{}, err
			}
			recorded.awaits = append(recorded.awaits, ref)
			start := recorded.starts[len(recorded.starts)-1]
			if err := awaitErr[start.DeviceID]; err != nil {
				return workflowengine.ActivityResult{}, err
			}
			session, err := workflowengine.DecodePayload[DeviceSessionPayload](start.Input.Payload)
			if err != nil {
				return workflowengine.ActivityResult{}, err
			}
			ready := DeviceSessionReady{
				Prepared: map[string]PreparedMobileDevice{},
				Device:   map[string]any{"serial": "serial-of-" + start.DeviceID},
			}
			for _, req := range session.Requests {
				ready.Prepared[req.StepID] = PreparedMobileDevice{
					RunnerID: path.Dir(req.DeviceID),
					Serial:   "serial-of-" + req.DeviceID,
					Type:     "android_phone",
				}
			}
			return workflowengine.ActivityResult{Output: ready}, nil
		},
		activity.RegisterOptions{Name: activities.AwaitDeviceSessionActivityName},
	)
	env.RegisterActivityWithOptions(
		func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			payload, err := workflowengine.DecodePayload[activities.ReleaseDeviceSessionPayload](
				input.Payload,
			)
			if err != nil {
				return workflowengine.ActivityResult{}, err
			}
			recorded.releases = append(recorded.releases, payload)
			if err := releaseErr[payload.Ref.WorkflowID]; err != nil {
				return workflowengine.ActivityResult{}, err
			}
			return workflowengine.ActivityResult{Output: release[payload.Ref.WorkflowID]}, nil
		},
		activity.RegisterOptions{Name: activities.ReleaseDeviceSessionActivityName},
	)
	return recorded
}

func sessionWorkflowID(deviceID string) string {
	return temporalid.Bounded("pipeline-1-device-" + canonify.CanonifyPlain(deviceID))
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

func TestMobileAutomationSetupHookStartsOneSessionPerDevice(t *testing.T) {
	const (
		deviceA = "tenant/runner-a/device-a"
		deviceB = "tenant/runner-b/device-b"
	)
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: "pipeline-1"})
	recorded := registerSessionActivities(env, nil, nil, nil)

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
		workflow.RegisterOptions{Name: "test-setup-device-sessions"},
	)

	env.ExecuteWorkflow("test-setup-device-sessions")
	require.NoError(t, env.GetWorkflowError())
	var result hookResult
	require.NoError(t, env.GetWorkflowResult(&result))

	pipelineRun := mobiledevicesession.PipelineRunRef{
		Namespace:  "default-test-namespace",
		WorkflowID: "pipeline-1",
		RunID:      "default-test-run-id",
	}
	// One session per device, in order of first use.
	require.Len(t, recorded.starts, 2)
	wantSteps := map[string][]string{deviceB: {"first", "third"}, deviceA: {"nested", "second"}}
	for i, deviceID := range []string{deviceB, deviceA} {
		start := recorded.starts[i]
		assert.Equal(t, deviceID, start.DeviceID)
		assert.Equal(t, "runner-ns-"+path.Base(path.Dir(deviceID)), start.Namespace)
		assert.Equal(t, sessionWorkflowID(deviceID), start.WorkflowID)
		assert.Equal(t, pipelineRun, start.Pipeline)
		session, err := workflowengine.DecodePayload[DeviceSessionPayload](start.Input.Payload)
		require.NoError(t, err)
		assert.Equal(t, deviceID, session.DeviceID)
		assert.Equal(t, "tenant/pipeline-run", session.RunIdentifier)
		assert.True(t, session.DisablePlayStore)
		assert.True(t, session.NoVideo)
		assert.Equal(t, pipelineRun, session.Pipeline)
		var stepIDs []string
		for _, req := range session.Requests {
			assert.Equal(t, deviceID, req.DeviceID)
			assert.Equal(t, "tenant/wallet/v1", req.VersionIdentifier)
			stepIDs = append(stepIDs, req.StepID)
		}
		assert.Equal(t, wantSteps[deviceID], stepIDs)
		assert.Equal(t, mobiledevicesession.RunnerWorkflowRef{
			Namespace:  start.Namespace,
			WorkflowID: start.WorkflowID,
			RunID:      "session-run-" + deviceID,
		}, recorded.awaits[i])
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
			"runner-ns-"+path.Base(runnerID),
			step.With.Config[runnerNamespaceConfigKey],
			step.ID,
		)
		assert.Equal(t, mobileRunnerTaskQueue(runnerID), step.With.Config["taskqueue"], step.ID)
		assert.Equal(t, runnerID, step.With.Config["runner_id"], step.ID)
		assert.Equal(t, step.ID, step.With.Config[workflowengine.StepIDConfigKey], step.ID)
		assert.Equal(t, "tenant/pipeline-run", step.With.Config["run_identifier"], step.ID)
		assert.Equal(t, map[string]any{
			"namespace":   pipelineRun.Namespace,
			"workflow_id": pipelineRun.WorkflowID,
			"run_id":      pipelineRun.RunID,
		}, step.With.Config[pipelineRunConfigKey], step.ID)
	}

	sessions := decodeAs[[]map[string]any](t, result.RunData[deviceSessionsRunDataKey])
	require.Len(t, sessions, 2)
	for i, deviceID := range []string{deviceB, deviceA} {
		assert.Equal(t, map[string]any{
			"device_id":   deviceID,
			"namespace":   "runner-ns-" + path.Base(path.Dir(deviceID)),
			"workflow_id": sessionWorkflowID(deviceID),
			"run_id":      "session-run-" + deviceID,
		}, sessions[i])
	}
	devices := workflowengine.AsMap(result.RunData["setted_devices"])
	require.Len(t, devices, 2)
	for _, deviceID := range []string{deviceA, deviceB} {
		assert.Equal(t, map[string]any{
			"serial":                     "serial-of-" + deviceID,
			"runner_namespace":           "runner-ns-" + path.Base(path.Dir(deviceID)),
			"device_session_workflow_id": sessionWorkflowID(deviceID),
			"device_session_run_id":      "session-run-" + deviceID,
		}, devices[deviceID])
	}
}

// TestMobileAutomationSetupHookReleasesDeviceWhoseSetupFails: a device whose
// session fails to prepare is still recorded, so cleanup releases it.
func TestMobileAutomationSetupHookReleasesDeviceWhoseSetupFails(t *testing.T) {
	const (
		deviceA = "tenant/runner-a/device-a"
		deviceB = "tenant/runner-b/device-b"
	)
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: "pipeline-1"})
	recorded := registerSessionActivities(
		env,
		map[string]error{
			deviceA: temporal.NewNonRetryableApplicationError("setup failed", "CRE1", nil),
		},
		map[string]mobiledevicesession.Cleanup{
			sessionWorkflowID(deviceA): {Warnings: []string{"device a cleaned"}},
		},
		nil,
	)

	type outcome struct {
		SetupErr   string         `json:"setup_err"`
		CleanupErr string         `json:"cleanup_err"`
		Output     map[string]any `json:"output"`
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
				}},
				map[string]any{
					"namespace":                            "tenant",
					mobileDeviceSemaphoreTicketIDConfigKey: "ticket-1",
				},
				&runData,
				&map[string]any{},
				log.Logger(noopLogger{}),
			); err != nil {
				out.SetupErr = err.Error()
			}
			output := map[string]any{}
			if err := MobileDeviceSessionsCleanupHook(
				ctx,
				nil,
				nil,
				nil,
				runData,
				&output,
			); err != nil {
				out.CleanupErr = err.Error()
			}
			out.Output = output
			return out, nil
		},
		workflow.RegisterOptions{Name: "test-setup-failure-released"},
	)

	env.ExecuteWorkflow("test-setup-failure-released")
	require.NoError(t, env.GetWorkflowError())
	var result outcome
	require.NoError(t, env.GetWorkflowResult(&result))

	assert.Contains(t, result.SetupErr, "setup failed")
	assert.Empty(t, result.CleanupErr)
	// The setup stops at the failing device: the next one never starts.
	require.Len(t, recorded.starts, 1)
	require.Len(t, recorded.releases, 1)
	assert.Equal(t, activities.ReleaseDeviceSessionPayload{
		Ref: mobiledevicesession.RunnerWorkflowRef{
			Namespace:  "runner-ns-runner-a",
			WorkflowID: sessionWorkflowID(deviceA),
			RunID:      "session-run-" + deviceA,
		},
	}, recorded.releases[0])
	assert.Equal(
		t,
		[]string{"device a cleaned"},
		workflowengine.AsSliceOfStrings(result.Output[cleanupWarningsOutputKey]),
	)
}

func TestMobileDeviceSessionsCleanupHook(t *testing.T) {
	sessionA := map[string]any{
		"device_id":   "tenant/runner-a/device-a",
		"namespace":   "runner-ns-a",
		"workflow_id": "session-a",
		"run_id":      "run-a",
	}
	sessionB := map[string]any{
		"device_id":   "tenant/runner-b/device-b",
		"namespace":   "runner-ns-b",
		"workflow_id": "session-b",
		"run_id":      "run-b",
	}
	evidence := map[string]mobiledevicesession.Cleanup{
		"session-a": {
			ResultVideoURLs: []string{"video-a", "shared-video"},
			ScreenshotURLs:  []string{"shot-a"},
			Warnings:        []string{"warning-a"},
		},
		"session-b": {
			ResultVideoURLs: []string{"shared-video", "video-b"},
			ScreenshotURLs:  []string{"shot-b"},
		},
	}
	cleanupErrMsg := "one or more errors occurred during mobile automation cleanup"

	tests := []struct {
		name         string
		sessions     []map[string]any
		policy       *pipeline.PipelineCancellationPolicy
		release      map[string]mobiledevicesession.Cleanup
		releaseErr   map[string]error
		wantSkip     []bool
		wantVideos   []string
		wantShots    []string
		wantWarnings []string
		wantErr      bool
	}{
		{name: "no device session releases nothing"},
		{
			name:       "releases every session and merges the evidence",
			sessions:   []map[string]any{sessionA, sessionB},
			release:    evidence,
			wantSkip:   []bool{false, false},
			wantVideos: []string{"video-a", "shared-video", "video-b"},
			wantShots:  []string{"shot-a", "shot-b"},
			wantWarnings: []string{
				"warning-a",
			},
		},
		{
			name:     "cancellation policy skips the cleanup of its devices",
			sessions: []map[string]any{sessionA, sessionB},
			policy: &pipeline.PipelineCancellationPolicy{
				Reason:               "runner paused",
				SkipDeviceCleanup:    true,
				SkipDeviceCleanupIDs: []string{"tenant/runner-a/device-a"},
			},
			release:    evidence,
			wantSkip:   []bool{true, false},
			wantVideos: []string{"video-a", "shared-video", "video-b"},
			wantShots:  []string{"shot-a", "shot-b"},
			wantWarnings: []string{
				"device cleanup skipped for tenant/runner-a/device-a because pipeline cancellation policy requested it",
				"warning-a",
			},
		},
		{
			name:     "release failures and cleanup errors become one error",
			sessions: []map[string]any{sessionA, sessionB},
			release: map[string]mobiledevicesession.Cleanup{
				"session-b": {
					ResultVideoURLs: []string{"video-b"},
					Errors:          []string{"uninstall failed"},
				},
			},
			releaseErr: map[string]error{
				"session-a": temporal.NewNonRetryableApplicationError(
					"release failed",
					"CRE1",
					nil,
				),
			},
			wantSkip:   []bool{false, false},
			wantVideos: []string{"video-b"},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			recorded := registerSessionActivities(env, nil, tt.release, tt.releaseErr)

			type outcome struct {
				Err     string         `json:"err"`
				Details map[string]any `json:"details"`
				Output  map[string]any `json:"output"`
			}
			env.RegisterWorkflowWithOptions(
				func(ctx workflow.Context) (outcome, error) {
					runData := map[string]any{"run_identifier": "tenant/pipeline-run"}
					if tt.sessions != nil {
						runData[deviceSessionsRunDataKey] = tt.sessions
					}
					if tt.policy != nil {
						runData[pipelineCancellationPolicyRunDataKey] = *tt.policy
					}
					output := map[string]any{}
					var out outcome
					if err := MobileDeviceSessionsCleanupHook(
						ctx,
						nil,
						nil,
						nil,
						runData,
						&output,
					); err != nil {
						out.Err = err.Error()
						out.Details = workflowengine.ParseWorkflowError(err).Details
					}
					out.Output = output
					return out, nil
				},
				workflow.RegisterOptions{Name: "test-device-sessions-cleanup"},
			)

			env.ExecuteWorkflow("test-device-sessions-cleanup")
			require.NoError(t, env.GetWorkflowError())
			var result outcome
			require.NoError(t, env.GetWorkflowResult(&result))

			require.Len(t, recorded.releases, len(tt.wantSkip))
			for i, release := range recorded.releases {
				session := tt.sessions[i]
				assert.Equal(t, mobiledevicesession.RunnerWorkflowRef{
					Namespace:  session["namespace"].(string),
					WorkflowID: session["workflow_id"].(string),
					RunID:      session["run_id"].(string),
				}, release.Ref)
				assert.Equal(t, tt.wantSkip[i], release.SkipCleanup, session["device_id"])
			}
			assert.Equal(
				t,
				tt.wantVideos,
				workflowengine.AsSliceOfStrings(result.Output["result_video_urls"]),
			)
			assert.Equal(
				t,
				tt.wantShots,
				workflowengine.AsSliceOfStrings(result.Output["screenshot_urls"]),
			)
			assert.Equal(
				t,
				tt.wantWarnings,
				workflowengine.AsSliceOfStrings(result.Output[cleanupWarningsOutputKey]),
			)
			if !tt.wantErr {
				assert.Empty(t, result.Err)
				return
			}
			assert.Contains(t, result.Err, cleanupErrMsg)
			payload := workflowengine.AsSliceOfStrings(result.Details["payload"])
			require.Len(t, payload, 2)
			assert.Contains(t, payload[0], "release failed")
			assert.Equal(t, "uninstall failed", payload[1])
		})
	}
}
