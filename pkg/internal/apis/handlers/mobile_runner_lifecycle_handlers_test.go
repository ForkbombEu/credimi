// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
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
		Return(nil, &serviceerror.WorkflowExecutionAlreadyStarted{})
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
	t.Cleanup(func() {
		mobileRunnerLifecycleNow = origNow
		mobileRunnerLifecycleTemporalClient = origLifecycleClient
		queueTemporalClient = origQueueClient
	})

	fixedNow := time.Date(2026, 6, 24, 10, 20, 30, 0, time.UTC)
	mobileRunnerLifecycleNow = func() time.Time { return fixedNow }
	mockClient := temporalmocks.NewClient(t)
	mockClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, workflows.MobileDeviceSemaphoreWorkflowName, mock.Anything).
		Return(nil, &serviceerror.WorkflowExecutionAlreadyStarted{})
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

func TestLifecycleUpdateIDUsesRequestID(t *testing.T) {
	require.Equal(t, "resume/device-a/request-1", lifecycleUpdateID("resume", "device-a", "request-1"))
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

func TestHandleMobileRunnerLifecyclePauseRejectsOtherOwner(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	otherOrg := createOtherWalletAPKOrganization(t, app)
	createMobileRunnerRecord(t, app, otherOrg.Id, "foreign-runner", "https://runner.example", false)

	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-runner/lifecycle/pause",
		MobileRunnerLifecycleRequest{RunnerID: "/other-org/foreign-runner"},
	)

	err = HandleMobileRunnerLifecyclePause()(event)
	recorder := responseRecorder(t, event)
	requireHandlerErrorHandled(t, recorder, err)
	require.Equal(t, http.StatusForbidden, recorder.Code)
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
