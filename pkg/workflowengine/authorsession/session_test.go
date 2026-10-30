// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package authorsession

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

const (
	testDeviceID = "tenant/runner-1/device-1"
	testRunnerID = "tenant/runner-1"
	testVersion  = "tenant/wallet/v1"
	testPackage  = "eu.wallet"
	testWorkflow = "ActionAuthorSession-ticket-1"
)

// activityCall is one recorded activity execution.
type activityCall struct {
	name      string
	taskQueue string
	payload   any
}

// sessionHarness fakes the Credimi, runner and agent activities a session
// uses and records them in call order.
type sessionHarness struct {
	t   *testing.T
	env *testsuite.TestWorkflowEnvironment

	mu    sync.Mutex
	calls []activityCall

	deviceType    string
	installerPath string
	installErr    error

	childErr       error
	childTaskQueue string
	childTimeout   time.Duration
	childContext   *AuthoringContext
}

func newSessionHarness(t *testing.T, deviceType string) *sessionHarness {
	t.Helper()
	suite := &testsuite.WorkflowTestSuite{}
	h := &sessionHarness{
		t:             t,
		env:           suite.NewTestWorkflowEnvironment(),
		deviceType:    deviceType,
		installerPath: "/runner/wallet.apk",
	}
	if deviceType == deviceTypeIOSSimulator {
		h.installerPath = "/runner/wallet.app"
	}
	h.env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: testWorkflow})
	session := New()
	h.env.RegisterWorkflowWithOptions(
		session.Workflow,
		workflow.RegisterOptions{Name: WorkflowName},
	)
	h.env.RegisterWorkflowWithOptions(h.child, workflow.RegisterOptions{Name: AuthorWorkflowType})

	h.register(activities.ResolveActionAuthorBuildActivityName, func(any) (any, error) {
		return actionauthor.Build{
			WalletID:          "wallet-1",
			VersionID:         "version-1",
			VersionIdentifier: testVersion,
			OwnerNamespace:    "tenant",
			Platform:          h.platform(),
			InstallerSHA256:   "digest",
		}, nil
	})
	h.register(activities.GetMobileDeviceActivityName, func(any) (any, error) {
		return map[string]any{
			"device_id":  testDeviceID,
			"runner_id":  testRunnerID,
			"runner_url": "https://runner.example",
			"type":       h.deviceType,
		}, nil
	})
	startDevice := func(any) (any, error) {
		return map[string]any{"name": "device-clone", "serial": h.serial()}, nil
	}
	h.register(activities.NewSetupMobileDeviceActivity().Name(), startDevice)
	h.register(activities.NewStartIOSSimulatorActivity().Name(), startDevice)
	h.register(activities.NewListInstalledAppsActivity().Name(), func(any) (any, error) {
		return []string{}, nil
	})
	h.register(activities.NewMobileRunnerHTTPActivity(nil).Name(), func(any) (any, error) {
		return map[string]any{"body": map[string]any{
			"installer_path": h.installerPath,
			"version_id":     testVersion,
		}}, nil
	})
	install := func(any) (any, error) {
		if h.installErr != nil {
			return nil, h.installErr
		}
		return map[string]any{"package_id": testPackage}, nil
	}
	for _, name := range []string{
		activities.NewApkInstallActivity().Name(),
		activities.NewApkPostInstallChecksActivity().Name(),
		activities.NewInstallIOSAppActivity().Name(),
		activities.NewIOSPostInstallChecksActivity().Name(),
	} {
		h.register(name, install)
	}
	h.register(activities.NewCleanupDeviceActivity().Name(), func(any) (any, error) {
		return nil, nil
	})
	h.register(
		activities.NewReportMobileDeviceSemaphoreDoneActivity().Name(),
		func(any) (any, error) {
			return nil, nil
		},
	)
	h.env.RegisterActivityWithOptions(
		func(ctx context.Context, payload map[string]any) error {
			h.record(ReleaseDeviceActivityName, activity.GetInfo(ctx).TaskQueue, payload)
			return nil
		},
		activity.RegisterOptions{Name: ReleaseDeviceActivityName},
	)
	return h
}
func (h *sessionHarness) platform() string {
	if h.deviceType == deviceTypeIOSSimulator {
		return actionauthor.PlatformIOS
	}
	return actionauthor.PlatformAndroid
}

func (h *sessionHarness) serial() string {
	if h.deviceType == deviceTypeIOSSimulator {
		return "UDID-S"
	}
	return "emulator-5554"
}

func (h *sessionHarness) register(name string, fn func(any) (any, error)) {
	h.env.RegisterActivityWithOptions(
		func(ctx context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			h.record(name, activity.GetInfo(ctx).TaskQueue, input.Payload)
			output, err := fn(input.Payload)
			return workflowengine.ActivityResult{Output: output}, err
		},
		activity.RegisterOptions{Name: name},
	)
}

func (h *sessionHarness) record(name, taskQueue string, payload any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls = append(h.calls, activityCall{name: name, taskQueue: taskQueue, payload: payload})
}

func (h *sessionHarness) child(
	ctx workflow.Context,
	_ map[string]any,
	authoring AuthoringContext,
) error {
	info := workflow.GetInfo(ctx)
	h.childTaskQueue = info.TaskQueueName
	h.childTimeout = info.WorkflowExecutionTimeout
	h.childContext = &authoring
	return h.childErr
}

func (h *sessionHarness) run(session SessionInput) error {
	h.env.ExecuteWorkflow(WorkflowName, workflowengine.WorkflowInput{
		Payload: session,
		Config: map[string]any{
			"mobile_device_semaphore_ticket_id":        "ticket-1",
			"mobile_device_semaphore_device_ids":       []string{testDeviceID},
			"mobile_device_semaphore_leader_device_id": testDeviceID,
			"mobile_device_semaphore_owner_namespace":  "tenant",
		},
	})
	require.True(h.t, h.env.IsWorkflowCompleted())
	return h.env.GetWorkflowError()
}

func (h *sessionHarness) names() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	names := make([]string, 0, len(h.calls))
	for _, call := range h.calls {
		names = append(names, call.name)
	}
	return names
}

func (h *sessionHarness) call(name string) (activityCall, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, call := range h.calls {
		if call.name == name {
			return call, true
		}
	}
	return activityCall{}, false
}

func (h *sessionHarness) requireReleasedAfter(previous string, result string) {
	h.t.Helper()
	names := h.names()
	cleanup := indexOf(names, activities.NewCleanupDeviceActivity().Name())
	report := indexOf(names, activities.NewReportMobileDeviceSemaphoreDoneActivity().Name())
	require.GreaterOrEqual(h.t, cleanup, 0, "cleanup must run: %v", names)
	require.Greater(h.t, report, cleanup, "RunDone must follow cleanup: %v", names)
	if previous != "" {
		require.Greater(h.t, cleanup, indexOf(names, previous), "%v", names)
	}
	reportCall, _ := h.call(activities.NewReportMobileDeviceSemaphoreDoneActivity().Name())
	reported, err := workflowengine.DecodePayload[activities.ReportMobileDeviceSemaphoreDoneInput](
		reportCall.payload,
	)
	require.NoError(h.t, err)
	require.Equal(h.t, activities.ReportMobileDeviceSemaphoreDoneInput{
		OwnerNamespace: "tenant",
		LeaderDeviceID: testDeviceID,
		TicketID:       "ticket-1",
		WorkflowID:     testWorkflow,
		RunID:          reported.RunID,
		WorkflowResult: result,
	}, reported)
}

func indexOf(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}

func sessionInput(platform string) SessionInput {
	return SessionInput{
		DeviceID:                testDeviceID,
		Platform:                platform,
		WalletVersionIdentifier: testVersion,
		ActorID:                 "user-1",
		ActorCollection:         "users",
		Slot:                    "onboarding",
	}
}

func TestSessionStartsAuthorOnRunnerQueueWithAuthoringContext(t *testing.T) {
	h := newSessionHarness(t, "android_emulator")

	require.NoError(t, h.run(sessionInput(actionauthor.PlatformAndroid)))

	require.Equal(t, testRunnerID+"-ActionAuthor", h.childTaskQueue)
	require.Equal(t, 2*time.Hour, h.childTimeout)
	require.NotNil(t, h.childContext)
	require.Equal(t, AuthoringContext{
		WalletVersionIdentifier: testVersion,
		ActorID:                 "user-1",
		ActorCollection:         "users",
		Slot:                    "onboarding",
		AppID:                   "",
		Platform:                actionauthor.PlatformAndroid,
		Build: AuthoringBuild{
			WalletID:          "wallet-1",
			VersionID:         "version-1",
			VersionIdentifier: testVersion,
			OwnerNamespace:    "tenant",
			Platform:          actionauthor.PlatformAndroid,
			InstallerSHA256:   "digest",
		},
		Device: AuthoringDevice{
			DeviceID:      testDeviceID,
			RunnerID:      testRunnerID,
			Serial:        "emulator-5554",
			Type:          "android_emulator",
			PackageID:     testPackage,
			InstallerPath: "/runner/wallet.apk",
		},
	}, *h.childContext)

	resolveCall, ok := h.call(activities.ResolveActionAuthorBuildActivityName)
	require.True(t, ok)
	request, err := workflowengine.DecodePayload[actionauthor.BuildRequest](resolveCall.payload)
	require.NoError(t, err)
	require.Equal(t, actionauthor.BuildRequest{
		Actor:                   actionauthor.Actor{ActorID: "user-1", ActorCollection: "users"},
		WalletVersionIdentifier: testVersion,
		Platform:                actionauthor.PlatformAndroid,
	}, request)

	h.requireReleasedAfter("", "success")
	require.NotContains(t, h.names(), ReleaseDeviceActivityName)
	cleanupCall, _ := h.call(activities.NewCleanupDeviceActivity().Name())
	require.Equal(t, testRunnerID+"-TaskQueue", cleanupCall.taskQueue)
}

func TestSessionCleansAndReleasesOnEveryAuthorOutcome(t *testing.T) {
	outcomes := map[string]error{
		"failed":     temporal.NewApplicationError("agent crashed", "test"),
		"timed out":  temporal.NewTimeoutError(enumspb.TIMEOUT_TYPE_START_TO_CLOSE, nil),
		"terminated": &temporal.TerminatedError{},
	}
	for name, childErr := range outcomes {
		t.Run(name, func(t *testing.T) {
			h := newSessionHarness(t, "android_emulator")
			h.childErr = childErr

			require.NoError(t, h.run(sessionInput(actionauthor.PlatformAndroid)))

			h.requireReleasedAfter("", "success")
		})
	}
}

func TestSessionCleansDeviceWhenSetupFailsAfterDeviceIsKnown(t *testing.T) {
	h := newSessionHarness(t, "android_emulator")
	h.installErr = temporal.NewNonRetryableApplicationError("install failed", "test", nil)

	err := h.run(sessionInput(actionauthor.PlatformAndroid))

	require.ErrorContains(t, err, "install failed")
	require.Nil(t, h.childContext)
	h.requireReleasedAfter(activities.NewApkInstallActivity().Name(), "failed")
	cleanupCall, _ := h.call(activities.NewCleanupDeviceActivity().Name())
	cleanupPayload, ok := cleanupCall.payload.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "emulator-5554", cleanupPayload["serial"])
}

func TestSessionRejectsAppIDMismatchBeforeAuthorStarts(t *testing.T) {
	h := newSessionHarness(t, "android_emulator")
	session := sessionInput(actionauthor.PlatformAndroid)
	session.AppID = "com.other"

	err := h.run(session)

	require.ErrorContains(t, err, "selected build is eu.wallet, not com.other")
	require.Nil(t, h.childContext)
	h.requireReleasedAfter("", "failed")
}

func TestSessionReleasesSimulatorClonesBeforeCleanup(t *testing.T) {
	outcomes := map[string]error{
		"completed":  nil,
		"terminated": &temporal.TerminatedError{},
	}
	for name, childErr := range outcomes {
		t.Run(name, func(t *testing.T) {
			h := newSessionHarness(t, deviceTypeIOSSimulator)
			h.childErr = childErr

			require.NoError(t, h.run(sessionInput(actionauthor.PlatformIOS)))

			require.Equal(t, "UDID-S", h.childContext.Device.Serial)
			require.Equal(t, "/runner/wallet.app", h.childContext.Device.InstallerPath)
			release, ok := h.call(ReleaseDeviceActivityName)
			require.True(t, ok, "%v", h.names())
			require.Equal(t, testRunnerID+"-ActionAuthor", release.taskQueue)
			require.Equal(
				t,
				map[string]any{"platform": "ios", "template": "UDID-S"},
				release.payload,
			)
			h.requireReleasedAfter(ReleaseDeviceActivityName, "success")
		})
	}
}

func TestSessionSkipsSimulatorReleaseWhenDeviceWasNotPrepared(t *testing.T) {
	h := newSessionHarness(t, deviceTypeIOSSimulator)
	h.installErr = temporal.NewNonRetryableApplicationError("install failed", "test", nil)

	require.Error(t, h.run(sessionInput(actionauthor.PlatformIOS)))

	require.NotContains(t, h.names(), ReleaseDeviceActivityName)
	h.requireReleasedAfter("", "failed")
}

func TestSessionRejectsInvalidInputAndStillReleasesTicket(t *testing.T) {
	h := newSessionHarness(t, "android_emulator")

	err := h.run(SessionInput{Slot: "onboarding"})

	require.ErrorContains(t, err, "device_id, platform and wallet_version_identifier are required")
	require.NotContains(t, h.names(), activities.NewCleanupDeviceActivity().Name())
	require.Contains(t, h.names(), activities.NewReportMobileDeviceSemaphoreDoneActivity().Name())
}

func TestSessionFailsOnInvalidBuild(t *testing.T) {
	h := newSessionHarness(t, "android_emulator")
	h.env.OnActivity(activities.ResolveActionAuthorBuildActivityName, mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{Output: "not a build"}, nil)

	err := h.run(sessionInput(actionauthor.PlatformAndroid))

	require.ErrorContains(t, err, "invalid action author build")
}
