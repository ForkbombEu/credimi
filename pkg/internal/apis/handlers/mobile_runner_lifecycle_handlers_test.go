// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunnerlifecycle"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func createMobileDeviceForLifecycleTest(
	t *testing.T,
	app core.App,
	runner *core.Record,
	name string,
) string {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(collection)
	device.Set("owner", runner.GetString("owner"))
	device.Set("runner", runner.Id)
	device.Set("name", name)
	device.Set("canonified_name", name)
	device.Set("type", "android_emulator")
	require.NoError(t, app.Save(device))
	identifier, err := mobileDeviceIdentifier(app, device)
	require.NoError(t, err)
	return identifier
}

func TestHandleMobileRunnerLifecycleResume(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, orgID, "resume-runner", "https://runner.example", false)
	runner, err := canonify.Resolve(app, "/usera-s-organization/resume-runner")
	require.NoError(t, err)
	deviceID := createMobileDeviceForLifecycleTest(t, app, runner, "device-a")

	origNow := mobileRunnerLifecycleNow
	origLifecycleClient := mobileRunnerLifecycleTemporalClient
	origQueueClient := queueTemporalClient
	t.Cleanup(func() {
		mobileRunnerLifecycleNow = origNow
		mobileRunnerLifecycleTemporalClient = origLifecycleClient
		queueTemporalClient = origQueueClient
	})

	fixedNow := time.Date(2026, 6, 23, 12, 30, 0, 0, time.UTC)
	mobileRunnerLifecycleNow = func() time.Time { return fixedNow }
	mockClient := temporalmocks.NewClient(t)
	mockClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, workflows.MobileDeviceSemaphoreWorkflowName, mock.Anything).
		Return(&temporalmocks.WorkflowRun{}, nil)
	handle := temporalmocks.NewWorkflowUpdateHandle(t)
	mockClient.
		On(
			"UpdateWorkflow",
			mock.Anything,
			mock.MatchedBy(func(options client.UpdateWorkflowOptions) bool {
				req, ok := options.Args[0].(workflows.MobileDeviceSemaphoreResumeDeviceRequest)
				return ok &&
					options.WorkflowID == workflows.MobileDeviceSemaphoreWorkflowID(deviceID) &&
					options.UpdateName == workflows.MobileDeviceSemaphoreResumeDeviceUpdate &&
					options.UpdateID == "resume/"+deviceID+"/resume-1" &&
					req.Reason == "runner_startup"
			}),
		).
		Return(handle, nil).
		Once()
	mobileRunnerLifecycleTemporalClient = func(_ string) (client.Client, error) { return mockClient, nil }
	queueTemporalClient = func(_ string) (client.Client, error) { return mockClient, nil }

	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-runner/lifecycle/resume",
		MobileRunnerLifecycleRequest{
			RunnerID:  "/usera-s-organization/resume-runner",
			RequestID: "resume-1",
			Devices:   []MobileDeviceLifecycleState{{DeviceID: deviceID, Online: true}},
		},
	)

	err = HandleMobileRunnerLifecycleResume()(event)
	require.NoError(t, err)

	recorder := responseRecorder(t, event)
	require.Equal(t, http.StatusOK, recorder.Code)

	record, err := canonify.Resolve(app, "/usera-s-organization/resume-runner")
	require.NoError(t, err)
	require.True(t, record.GetBool("online"))
	require.Equal(
		t,
		fixedNow.Format("2006-01-02 15:04:05.000Z"),
		record.GetString("last_heartbeat_at"),
	)
}

func TestHandleMobileRunnerLifecycleHeartbeat(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, orgID, "heartbeat-runner", "https://runner.example", false)

	origNow := mobileRunnerLifecycleNow
	origQuerySemaphore := queryMobileDeviceSemaphoreState
	t.Cleanup(func() {
		mobileRunnerLifecycleNow = origNow
		queryMobileDeviceSemaphoreState = origQuerySemaphore
	})

	fixedNow := time.Date(2026, 6, 24, 10, 15, 30, 0, time.UTC)
	mobileRunnerLifecycleNow = func() time.Time { return fixedNow }
	queryMobileDeviceSemaphoreState = func(_ context.Context, _ string) (workflows.MobileDeviceSemaphoreStateView, error) {
		return workflows.MobileDeviceSemaphoreStateView{}, errSemaphoreNotFound
	}

	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-runner/lifecycle/heartbeat",
		MobileRunnerLifecycleRequest{RunnerID: "/usera-s-organization/heartbeat-runner"},
	)

	err = HandleMobileRunnerLifecycleHeartbeat()(event)
	require.NoError(t, err)

	recorder := responseRecorder(t, event)
	require.Equal(t, http.StatusOK, recorder.Code)

	response := decodeJSONBody(t, recorder)
	require.Equal(t, "usera-s-organization/heartbeat-runner", response["runner_id"])
	require.Equal(t, true, response["online"])
	require.Equal(
		t,
		float64(mobilerunnerlifecycle.DefaultHeartbeatTimeout/time.Second),
		response["heartbeat_timeout_seconds"],
	)

	record, err := canonify.Resolve(app, "/usera-s-organization/heartbeat-runner")
	require.NoError(t, err)
	require.True(t, record.GetBool("online"))
	require.Equal(
		t,
		fixedNow.Format("2006-01-02 15:04:05.000Z"),
		record.GetString("last_heartbeat_at"),
	)
}

func TestHandleMobileRunnerLifecycleHeartbeatResumesHeartbeatTimeoutPause(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(
		t,
		app,
		orgID,
		"heartbeat-resume-runner",
		"https://runner.example",
		false,
	)
	runner, err := canonify.Resolve(app, "/usera-s-organization/heartbeat-resume-runner")
	require.NoError(t, err)
	deviceID := createMobileDeviceForLifecycleTest(t, app, runner, "device-a")

	origNow := mobileRunnerLifecycleNow
	origLifecycleClient := mobileRunnerLifecycleTemporalClient
	origQueueClient := queueTemporalClient
	origQuerySemaphore := queryMobileDeviceSemaphoreState
	t.Cleanup(func() {
		mobileRunnerLifecycleNow = origNow
		mobileRunnerLifecycleTemporalClient = origLifecycleClient
		queueTemporalClient = origQueueClient
		queryMobileDeviceSemaphoreState = origQuerySemaphore
	})
	queryMobileDeviceSemaphoreState = func(_ context.Context, id string) (workflows.MobileDeviceSemaphoreStateView, error) {
		return workflows.MobileDeviceSemaphoreStateView{DeviceID: id, Paused: true}, nil
	}

	fixedNow := time.Date(2026, 6, 24, 10, 20, 30, 0, time.UTC)
	mobileRunnerLifecycleNow = func() time.Time { return fixedNow }
	mockClient := temporalmocks.NewClient(t)
	mockClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, workflows.MobileDeviceSemaphoreWorkflowName, mock.Anything).
		Return(&temporalmocks.WorkflowRun{}, nil)
	handle := temporalmocks.NewWorkflowUpdateHandle(t)
	mockClient.
		On(
			"UpdateWorkflow",
			mock.Anything,
			mock.MatchedBy(func(options client.UpdateWorkflowOptions) bool {
				req, ok := options.Args[0].(workflows.MobileDeviceSemaphoreResumeDeviceRequest)
				return ok &&
					options.WorkflowID == workflows.MobileDeviceSemaphoreWorkflowID(
						deviceID,
					) &&
					options.UpdateName == workflows.MobileDeviceSemaphoreResumeDeviceUpdate &&
					options.WaitForStage == client.WorkflowUpdateStageAccepted &&
					options.UpdateID == "resume/"+deviceID+"/heartbeat-1" &&
					req.Reason == "runner_heartbeat"
			}),
		).
		Return(handle, nil).
		Once()
	mobileRunnerLifecycleTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}
	queueTemporalClient = func(_ string) (client.Client, error) { return mockClient, nil }

	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-runner/lifecycle/heartbeat",
		MobileRunnerLifecycleRequest{
			RunnerID:  "/usera-s-organization/heartbeat-resume-runner",
			RequestID: "heartbeat-1",
			Devices:   []MobileDeviceLifecycleState{{DeviceID: deviceID, Online: true}},
		},
	)

	err = HandleMobileRunnerLifecycleHeartbeat()(event)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, responseRecorder(t, event).Code)
}

// A runner keeps reporting an offline device on every heartbeat, each with a
// fresh request ID. Re-sending the pause would add a distinct Temporal update
// per heartbeat until the semaphore hits the server's per-run update limit.
func TestHandleMobileRunnerLifecycleHeartbeatSkipsPauseForPausedSemaphore(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(
		t,
		app,
		orgID,
		"offline-device-runner",
		"https://runner.example",
		false,
	)
	runner, err := canonify.Resolve(app, "/usera-s-organization/offline-device-runner")
	require.NoError(t, err)
	offlineID := createMobileDeviceForLifecycleTest(t, app, runner, "offline-device")
	staleID := createMobileDeviceForLifecycleTest(t, app, runner, "stale-device")

	origLifecycleClient := mobileRunnerLifecycleTemporalClient
	origQuerySemaphore := queryMobileDeviceSemaphoreState
	t.Cleanup(func() {
		mobileRunnerLifecycleTemporalClient = origLifecycleClient
		queryMobileDeviceSemaphoreState = origQuerySemaphore
	})
	queried := map[string]bool{}
	queryMobileDeviceSemaphoreState = func(_ context.Context, id string) (workflows.MobileDeviceSemaphoreStateView, error) {
		queried[id] = true
		return workflows.MobileDeviceSemaphoreStateView{DeviceID: id, Paused: true}, nil
	}
	// No UpdateWorkflow expectation: any update call fails the test.
	mockClient := temporalmocks.NewClient(t)
	mobileRunnerLifecycleTemporalClient = func(_ string) (client.Client, error) { return mockClient, nil }

	for i := range 3 {
		event := performMobileRunnerRequest(
			t,
			app,
			user,
			"/api/mobile-runner/lifecycle/heartbeat",
			MobileRunnerLifecycleRequest{
				RunnerID:  "/usera-s-organization/offline-device-runner",
				RequestID: fmt.Sprintf("heartbeat-%d", i),
				Devices:   []MobileDeviceLifecycleState{{DeviceID: offlineID, Online: false}},
			},
		)
		require.NoError(t, HandleMobileRunnerLifecycleHeartbeat()(event))
		require.Equal(t, http.StatusOK, responseRecorder(t, event).Code)
	}
	require.True(t, queried[offlineID])
	require.True(t, queried[staleID])
}

func TestHandleMobileRunnerLifecycleHeartbeatPausesRunningSemaphore(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, orgID, "pausing-runner", "https://runner.example", false)
	runner, err := canonify.Resolve(app, "/usera-s-organization/pausing-runner")
	require.NoError(t, err)
	deviceID := createMobileDeviceForLifecycleTest(t, app, runner, "device-a")

	origLifecycleClient := mobileRunnerLifecycleTemporalClient
	origQuerySemaphore := queryMobileDeviceSemaphoreState
	t.Cleanup(func() {
		mobileRunnerLifecycleTemporalClient = origLifecycleClient
		queryMobileDeviceSemaphoreState = origQuerySemaphore
	})
	queryMobileDeviceSemaphoreState = func(_ context.Context, id string) (workflows.MobileDeviceSemaphoreStateView, error) {
		return workflows.MobileDeviceSemaphoreStateView{DeviceID: id, Paused: false}, nil
	}
	mockClient := temporalmocks.NewClient(t)
	mockClient.
		On(
			"UpdateWorkflow",
			mock.Anything,
			mock.MatchedBy(func(options client.UpdateWorkflowOptions) bool {
				return options.WorkflowID == workflows.MobileDeviceSemaphoreWorkflowID(deviceID) &&
					options.UpdateName == workflows.MobileDeviceSemaphorePauseDeviceUpdate &&
					options.UpdateID == "pause/"+deviceID+"/heartbeat-1"
			}),
		).
		Return(temporalmocks.NewWorkflowUpdateHandle(t), nil).
		Once()
	mobileRunnerLifecycleTemporalClient = func(_ string) (client.Client, error) { return mockClient, nil }

	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-runner/lifecycle/heartbeat",
		MobileRunnerLifecycleRequest{
			RunnerID:  "/usera-s-organization/pausing-runner",
			RequestID: "heartbeat-1",
			Devices:   []MobileDeviceLifecycleState{{DeviceID: deviceID, Online: false}},
		},
	)
	require.NoError(t, HandleMobileRunnerLifecycleHeartbeat()(event))
	require.Equal(t, http.StatusOK, responseRecorder(t, event).Code)
}

func TestLifecycleUpdateIDUsesRequestID(t *testing.T) {
	require.Equal(
		t,
		"resume/device-a/request-1",
		lifecycleUpdateID("resume", "device-a", "request-1"),
	)
}

func TestHandleMobileRunnerLifecycleHeartbeatRejectsEmptyDeviceID(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-runner/lifecycle/heartbeat",
		MobileRunnerLifecycleRequest{RunnerID: " "},
	)

	err = HandleMobileRunnerLifecycleHeartbeat()(event)
	recorder := responseRecorder(t, event)
	requireHandlerErrorHandled(t, recorder, err)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestHandleMobileRunnerLifecyclePauseMissingSemaphoreSucceeds(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, orgID, "pause-runner", "https://runner.example", false)
	runner, err := canonify.Resolve(app, "/usera-s-organization/pause-runner")
	require.NoError(t, err)
	createMobileDeviceForLifecycleTest(t, app, runner, "device-1")

	record, err := canonify.Resolve(app, "/usera-s-organization/pause-runner")
	require.NoError(t, err)
	record.Set("online", true)
	require.NoError(t, app.Save(record))

	origLifecycleClient := mobileRunnerLifecycleTemporalClient
	t.Cleanup(func() { mobileRunnerLifecycleTemporalClient = origLifecycleClient })

	mockClient := temporalmocks.NewClient(t)
	mockClient.
		On("UpdateWorkflow", mock.Anything, mock.Anything).
		Return(nil, &serviceerror.NotFound{Message: "missing"}).
		Once()
	mobileRunnerLifecycleTemporalClient = func(_ string) (client.Client, error) { return mockClient, nil }

	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-runner/lifecycle/pause",
		MobileRunnerLifecycleRequest{RunnerID: "/usera-s-organization/pause-runner"},
	)

	err = HandleMobileRunnerLifecyclePause()(event)
	require.NoError(t, err)

	recorder := responseRecorder(t, event)
	require.Equal(t, http.StatusOK, recorder.Code)

	record, err = canonify.Resolve(app, "/usera-s-organization/pause-runner")
	require.NoError(t, err)
	require.False(t, record.GetBool("online"))
}

func TestResolveLifecycleRunnerAllowsSuperuser(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
	require.NoError(t, err)
	otherOrg := createOtherWalletAPKOrganization(t, app)
	createMobileRunnerRecord(t, app, otherOrg.Id, "admin-runner", "https://runner.example", false)

	record, runnerID, apiErr := resolveLifecycleRunner(app, superuser, "/other-org/admin-runner")
	require.Nil(t, apiErr)
	require.Equal(t, "admin-runner", record.GetString("name"))
	require.Equal(t, "other-org/admin-runner", runnerID)
}

func TestUpdateRunnerSemaphoreReturnsNotFound(t *testing.T) {
	origLifecycleClient := mobileRunnerLifecycleTemporalClient
	t.Cleanup(func() { mobileRunnerLifecycleTemporalClient = origLifecycleClient })

	mockClient := temporalmocks.NewClient(t)
	mockClient.
		On("UpdateWorkflow", mock.Anything, mock.Anything).
		Return(nil, &serviceerror.NotFound{})
	mobileRunnerLifecycleTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	ok, err := updateRunnerSemaphore(
		t.Context(),
		"runner-1",
		workflows.MobileDeviceSemaphorePauseDeviceUpdate,
		workflows.MobileDeviceSemaphorePauseDeviceRequest{},
		nil,
		"pause/runner-1",
	)
	require.False(t, ok)
	require.ErrorIs(t, err, errSemaphoreNotFound)
}

func TestUpdateRunnerSemaphoreDecodesResponse(t *testing.T) {
	origLifecycleClient := mobileRunnerLifecycleTemporalClient
	t.Cleanup(func() { mobileRunnerLifecycleTemporalClient = origLifecycleClient })

	mockClient := temporalmocks.NewClient(t)
	handle := temporalmocks.NewWorkflowUpdateHandle(t)
	handle.On("Get", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		out := args.Get(1).(*workflows.MobileDeviceSemaphoreResumeDeviceResponse)
		*out = workflows.MobileDeviceSemaphoreResumeDeviceResponse{
			DeviceID: "runner-1",
			Paused:   false,
			QueueLen: 2,
		}
	}).Return(nil)
	mockClient.
		On("UpdateWorkflow", mock.Anything, mock.Anything).
		Return(handle, nil)
	mobileRunnerLifecycleTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	var out workflows.MobileDeviceSemaphoreResumeDeviceResponse
	ok, err := updateRunnerSemaphore(
		t.Context(),
		"runner-1",
		workflows.MobileDeviceSemaphoreResumeDeviceUpdate,
		workflows.MobileDeviceSemaphoreResumeDeviceRequest{},
		&out,
		"resume/runner-1",
	)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 2, out.QueueLen)
}

func TestHandleMobileRunnerLifecyclePauseSendsPauseUpdate(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, orgID, "pause-update-runner", "https://runner.example", false)
	runner, err := canonify.Resolve(app, "/usera-s-organization/pause-update-runner")
	require.NoError(t, err)
	deviceID := createMobileDeviceForLifecycleTest(t, app, runner, "device-1")

	origLifecycleClient := mobileRunnerLifecycleTemporalClient
	t.Cleanup(func() { mobileRunnerLifecycleTemporalClient = origLifecycleClient })

	mockClient := temporalmocks.NewClient(t)
	handle := temporalmocks.NewWorkflowUpdateHandle(t)
	mockClient.
		On(
			"UpdateWorkflow",
			mock.Anything,
			mock.MatchedBy(func(options client.UpdateWorkflowOptions) bool {
				req, ok := options.Args[0].(workflows.MobileDeviceSemaphorePauseDeviceRequest)
				return ok &&
					options.WorkflowID == workflows.MobileDeviceSemaphoreWorkflowID(deviceID) &&
					options.UpdateName == workflows.MobileDeviceSemaphorePauseDeviceUpdate &&
					options.WaitForStage == client.WorkflowUpdateStageAccepted &&
					req.CancelRunning &&
					req.Reason == "runner_shutdown" &&
					req.ShutdownAfterSeconds == int(
						mobilerunnerlifecycle.DefaultShutdownAfter/time.Second,
					)
			}),
		).
		Return(handle, nil).
		Once()
	mobileRunnerLifecycleTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-runner/lifecycle/pause",
		MobileRunnerLifecycleRequest{RunnerID: "/usera-s-organization/pause-update-runner"},
	)

	err = HandleMobileRunnerLifecyclePause()(event)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, responseRecorder(t, event).Code)
}

type lifecycleStubs struct {
	lifecycleClient client.Client
	lifecycleErr    error
	queueClient     client.Client
	queueErr        error
	query           func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error)
}

func stubLifecycleTemporal(t *testing.T, stubs lifecycleStubs) {
	t.Helper()

	origLifecycleClient := mobileRunnerLifecycleTemporalClient
	origQueueClient := queueTemporalClient
	origQuery := queryMobileDeviceSemaphoreState
	t.Cleanup(func() {
		mobileRunnerLifecycleTemporalClient = origLifecycleClient
		queueTemporalClient = origQueueClient
		queryMobileDeviceSemaphoreState = origQuery
	})
	mobileRunnerLifecycleTemporalClient = func(string) (client.Client, error) {
		if stubs.lifecycleErr != nil {
			return nil, stubs.lifecycleErr
		}
		if stubs.lifecycleClient == nil {
			t.Fatal("unexpected semaphore update")
		}
		return stubs.lifecycleClient, nil
	}
	queueTemporalClient = func(string) (client.Client, error) {
		if stubs.queueErr != nil {
			return nil, stubs.queueErr
		}
		if stubs.queueClient == nil {
			t.Fatal("unexpected semaphore workflow start")
		}
		return stubs.queueClient, nil
	}
	if stubs.query != nil {
		queryMobileDeviceSemaphoreState = stubs.query
	}
}

func semaphoreRunningClient(t *testing.T) *temporalmocks.Client {
	t.Helper()

	mockClient := temporalmocks.NewClient(t)
	mockClient.
		On(
			"ExecuteWorkflow",
			mock.Anything,
			mock.Anything,
			workflows.MobileDeviceSemaphoreWorkflowName,
			mock.Anything,
		).
		Return(&temporalmocks.WorkflowRun{}, nil)
	return mockClient
}

func TestMobileRunnerLifecycleRejectsInvalidRunners(t *testing.T) {
	handlers := map[string]func() func(*core.RequestEvent) error{
		"resume":    HandleMobileRunnerLifecycleResume,
		"heartbeat": HandleMobileRunnerLifecycleHeartbeat,
		"pause":     HandleMobileRunnerLifecyclePause,
	}
	cases := []struct {
		name       string
		runnerID   string
		wantStatus int
		wantReason string
	}{
		{
			name:       "unknown runner",
			runnerID:   "/usera-s-organization/missing-runner",
			wantStatus: http.StatusNotFound,
			wantReason: "mobile_runner_not_found",
		},
		{
			name:       "organization instead of runner",
			runnerID:   "/usera-s-organization",
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_runner_id",
		},
		{
			name:       "device instead of runner",
			runnerID:   "/usera-s-organization/own-runner/device-a",
			wantStatus: http.StatusBadRequest,
			wantReason: "invalid_runner_id",
		},
		{
			name:       "runner of another organization",
			runnerID:   "/other-org/foreign-runner",
			wantStatus: http.StatusForbidden,
			wantReason: "runner_owner_mismatch",
		},
	}

	for handlerName, handler := range handlers {
		for _, tc := range cases {
			t.Run(handlerName+"/"+tc.name, func(t *testing.T) {
				app := setupMobileRunnerApp(t)
				defer app.Cleanup()

				user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
				require.NoError(t, err)
				orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
				require.NoError(t, err)
				createMobileRunnerRecord(
					t,
					app,
					orgID,
					"own-runner",
					"https://runner.example",
					false,
				)
				ownRunner, err := canonify.Resolve(app, "/usera-s-organization/own-runner")
				require.NoError(t, err)
				createMobileDeviceForLifecycleTest(t, app, ownRunner, "device-a")
				otherOrg := createOtherWalletAPKOrganization(t, app)
				createMobileRunnerRecord(
					t,
					app,
					otherOrg.Id,
					"foreign-runner",
					"https://runner.example",
					false,
				)
				// Rejected requests must never reach Temporal.
				stubLifecycleTemporal(t, lifecycleStubs{})

				event := performMobileRunnerRequest(
					t,
					app,
					user,
					"/api/mobile-runner/lifecycle/"+handlerName,
					MobileRunnerLifecycleRequest{RunnerID: tc.runnerID},
				)
				err = handler()(event)
				recorder := responseRecorder(t, event)
				requireHandlerErrorHandled(t, recorder, err)
				require.Equal(t, tc.wantStatus, recorder.Code)
				require.Equal(
					t,
					tc.wantReason,
					decodeHandlerErrorResponse(t, recorder).Error.Reason,
				)

				foreign, err := canonify.Resolve(app, "/other-org/foreign-runner")
				require.NoError(t, err)
				require.False(t, foreign.GetBool("online"))
			})
		}
	}
}

func TestHandleMobileRunnerLifecycleResumeRejectsForeignDeviceAtomically(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, orgID, "runner-a", "https://runner-a.example", false)
	createMobileRunnerRecord(t, app, orgID, "runner-b", "https://runner-b.example", false)
	runnerA, err := canonify.Resolve(app, "/usera-s-organization/runner-a")
	require.NoError(t, err)
	runnerB, err := canonify.Resolve(app, "/usera-s-organization/runner-b")
	require.NoError(t, err)
	ownDeviceID := createMobileDeviceForLifecycleTest(t, app, runnerA, "own-device")
	siblingDeviceID := createMobileDeviceForLifecycleTest(t, app, runnerB, "sibling-device")
	stubLifecycleTemporal(t, lifecycleStubs{})

	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-runner/lifecycle/resume",
		MobileRunnerLifecycleRequest{
			RunnerID: "/usera-s-organization/runner-a",
			Devices: []MobileDeviceLifecycleState{
				{DeviceID: ownDeviceID, Online: true},
				{DeviceID: siblingDeviceID, Online: true},
			},
		},
	)
	err = HandleMobileRunnerLifecycleResume()(event)
	recorder := responseRecorder(t, event)
	requireHandlerErrorHandled(t, recorder, err)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	body := decodeHandlerErrorResponse(t, recorder)
	require.Equal(t, "failed_to_apply_device_resume", body.Error.Reason)
	require.Contains(t, body.Error.Message, siblingDeviceID)

	// The whole heartbeat transaction is rolled back.
	runner, err := app.FindRecordById("mobile_runners", runnerA.Id)
	require.NoError(t, err)
	require.False(t, runner.GetBool("online"))
	require.Empty(t, runner.GetString("last_heartbeat_at"))
	for _, deviceID := range []string{ownDeviceID, siblingDeviceID} {
		device, err := canonify.Resolve(app, "/"+deviceID)
		require.NoError(t, err)
		require.False(t, device.GetBool("online"), deviceID)
	}
}

func TestHandleMobileRunnerLifecycleResumeOnlyResumesReportedOnlineDevices(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, orgID, "resume-runner", "https://runner.example", false)
	runner, err := canonify.Resolve(app, "/usera-s-organization/resume-runner")
	require.NoError(t, err)
	onlineID := createMobileDeviceForLifecycleTest(t, app, runner, "online-device")
	offlineID := createMobileDeviceForLifecycleTest(t, app, runner, "offline-device")
	unreportedID := createMobileDeviceForLifecycleTest(t, app, runner, "unreported-device")
	unreported, err := canonify.Resolve(app, "/"+unreportedID)
	require.NoError(t, err)
	unreported.Set("online", true)
	require.NoError(t, app.Save(unreported))

	mockClient := semaphoreRunningClient(t)
	mockClient.
		On(
			"UpdateWorkflow",
			mock.Anything,
			mock.MatchedBy(func(options client.UpdateWorkflowOptions) bool {
				req, ok := options.Args[0].(workflows.MobileDeviceSemaphoreResumeDeviceRequest)
				return ok &&
					options.WorkflowID == workflows.MobileDeviceSemaphoreWorkflowID(onlineID) &&
					req.Reason == "manual restart"
			}),
		).
		// A semaphore that does not exist yet is not an error for resume.
		Return(nil, &serviceerror.NotFound{Message: "missing"}).
		Once()
	stubLifecycleTemporal(t, lifecycleStubs{lifecycleClient: mockClient, queueClient: mockClient})

	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-runner/lifecycle/resume",
		MobileRunnerLifecycleRequest{
			RunnerID: "/usera-s-organization/resume-runner",
			Reason:   " manual restart ",
			Devices: []MobileDeviceLifecycleState{
				{DeviceID: onlineID, Online: true},
				{DeviceID: offlineID, Online: false},
			},
		},
	)
	require.NoError(t, HandleMobileRunnerLifecycleResume()(event))
	require.Equal(t, http.StatusOK, responseRecorder(t, event).Code)

	want := map[string]bool{onlineID: true, offlineID: false, unreportedID: false}
	for deviceID, online := range want {
		device, err := canonify.Resolve(app, "/"+deviceID)
		require.NoError(t, err)
		require.Equal(t, online, device.GetBool("online"), deviceID)
	}
}

func TestMobileRunnerLifecycleSemaphoreFailures(t *testing.T) {
	pausedState := func(paused bool, err error) func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error) {
		return func(_ context.Context, id string) (workflows.MobileDeviceSemaphoreStateView, error) {
			return workflows.MobileDeviceSemaphoreStateView{DeviceID: id, Paused: paused}, err
		}
	}
	updateFails := func(t *testing.T, updateName string) *temporalmocks.Client {
		mockClient := temporalmocks.NewClient(t)
		mockClient.
			On(
				"UpdateWorkflow",
				mock.Anything,
				mock.MatchedBy(func(options client.UpdateWorkflowOptions) bool {
					return options.UpdateName == updateName
				}),
			).
			Return(nil, errors.New("temporal unavailable")).
			Once()
		return mockClient
	}

	cases := []struct {
		name       string
		handler    func() func(*core.RequestEvent) error
		online     bool
		stubs      func(t *testing.T) lifecycleStubs
		wantStatus int
		wantReason string
	}{
		{
			name:    "resume cannot start the device semaphore",
			handler: HandleMobileRunnerLifecycleResume,
			online:  true,
			stubs: func(*testing.T) lifecycleStubs {
				return lifecycleStubs{queueErr: errors.New("dial failed")}
			},
			wantStatus: http.StatusInternalServerError,
			wantReason: "failed_to_ensure_device_semaphore",
		},
		{
			name:    "resume update fails",
			handler: HandleMobileRunnerLifecycleResume,
			online:  true,
			stubs: func(t *testing.T) lifecycleStubs {
				return lifecycleStubs{
					lifecycleClient: updateFails(
						t,
						workflows.MobileDeviceSemaphoreResumeDeviceUpdate,
					),
					queueClient: semaphoreRunningClient(t),
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantReason: "failed_to_resume_device_semaphore",
		},
		{
			name:    "heartbeat cannot start the device semaphore",
			handler: HandleMobileRunnerLifecycleHeartbeat,
			online:  true,
			stubs: func(*testing.T) lifecycleStubs {
				return lifecycleStubs{queueErr: errors.New("dial failed")}
			},
			wantStatus: http.StatusInternalServerError,
			wantReason: "failed_to_ensure_device_semaphore",
		},
		{
			name:    "heartbeat resumes when the semaphore state is unknown and the update fails",
			handler: HandleMobileRunnerLifecycleHeartbeat,
			online:  true,
			stubs: func(t *testing.T) lifecycleStubs {
				return lifecycleStubs{
					lifecycleClient: updateFails(
						t,
						workflows.MobileDeviceSemaphoreResumeDeviceUpdate,
					),
					queueClient: semaphoreRunningClient(t),
					query:       pausedState(false, errors.New("query timeout")),
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantReason: "failed_to_resume_device_semaphore",
		},
		{
			name:    "heartbeat pause of an offline device fails",
			handler: HandleMobileRunnerLifecycleHeartbeat,
			online:  false,
			stubs: func(t *testing.T) lifecycleStubs {
				return lifecycleStubs{
					lifecycleClient: updateFails(
						t,
						workflows.MobileDeviceSemaphorePauseDeviceUpdate,
					),
					query: pausedState(false, nil),
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantReason: "failed_to_pause_device_semaphore",
		},
		{
			name:    "heartbeat leaves an already running semaphore alone",
			handler: HandleMobileRunnerLifecycleHeartbeat,
			online:  true,
			stubs: func(t *testing.T) lifecycleStubs {
				// Only the idempotent workflow start is expected; no update.
				return lifecycleStubs{
					queueClient: semaphoreRunningClient(t),
					query:       pausedState(false, nil),
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "heartbeat skips an offline device without semaphore",
			handler: HandleMobileRunnerLifecycleHeartbeat,
			online:  false,
			stubs: func(*testing.T) lifecycleStubs {
				return lifecycleStubs{query: pausedState(false, errSemaphoreNotFound)}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "pause cannot reach temporal",
			handler: HandleMobileRunnerLifecyclePause,
			stubs: func(*testing.T) lifecycleStubs {
				return lifecycleStubs{lifecycleErr: errors.New("dial failed")}
			},
			wantStatus: http.StatusInternalServerError,
			wantReason: "failed_to_pause_device_semaphore",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := setupMobileRunnerApp(t)
			defer app.Cleanup()

			user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)
			orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
			require.NoError(t, err)
			createMobileRunnerRecord(
				t,
				app,
				orgID,
				"failing-runner",
				"https://runner.example",
				false,
			)
			runner, err := canonify.Resolve(app, "/usera-s-organization/failing-runner")
			require.NoError(t, err)
			deviceID := createMobileDeviceForLifecycleTest(t, app, runner, "device-a")
			stubLifecycleTemporal(t, tc.stubs(t))

			event := performMobileRunnerRequest(
				t,
				app,
				user,
				"/api/mobile-runner/lifecycle",
				MobileRunnerLifecycleRequest{
					RunnerID:  "/usera-s-organization/failing-runner",
					RequestID: "req-1",
					Devices: []MobileDeviceLifecycleState{
						{DeviceID: deviceID, Online: tc.online},
					},
				},
			)
			err = tc.handler()(event)
			recorder := responseRecorder(t, event)
			requireHandlerErrorHandled(t, recorder, err)
			require.Equal(t, tc.wantStatus, recorder.Code)
			if tc.wantReason != "" {
				require.Equal(
					t,
					tc.wantReason,
					decodeHandlerErrorResponse(t, recorder).Error.Reason,
				)
			}
		})
	}
}
