// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func createLifecycleMonitorRunner(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	name string,
	online bool,
	lastHeartbeat time.Time,
) *core.Record {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("name", name)
	record.Set("ip", "https://runner.example")
	record.Set("type", "android_emulator")
	record.Set("online", online)
	if !lastHeartbeat.IsZero() {
		record.Set("last_heartbeat_at", lastHeartbeat.UTC().Format("2006-01-02 15:04:05.000Z"))
	}
	require.NoError(t, app.Save(record))
	return record
}

func ensureLifecycleMonitorFields(t testing.TB, app *tests.TestApp) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)

	if collection.Fields.GetByName("online") == nil {
		collection.Fields.Add(&core.BoolField{Name: "online"})
	}
	if collection.Fields.GetByName("last_heartbeat_at") == nil {
		collection.Fields.Add(&core.DateField{Name: "last_heartbeat_at"})
	}

	require.NoError(t, app.Save(collection))
}

func TestMarkStaleRunnersOfflineAndPauseSemaphores(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	ensureLifecycleMonitorFields(t, app)
	canonify.RegisterCanonifyHooks(app)

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)

	staleAt := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)
	freshAt := staleAt.Add(119 * time.Second)
	now := staleAt.Add(mobileRunnerLifecycleHeartbeatTimeout + time.Second)

	createLifecycleMonitorRunner(t, app, orgID, "stale-runner", true, staleAt)
	createLifecycleMonitorRunner(t, app, orgID, "fresh-runner", true, freshAt)

	origNow := mobileRunnerLifecycleMonitorNow
	origClient := mobileRunnerLifecycleMonitorTemporalClient
	t.Cleanup(func() {
		mobileRunnerLifecycleMonitorNow = origNow
		mobileRunnerLifecycleMonitorTemporalClient = origClient
	})
	mobileRunnerLifecycleMonitorNow = func() time.Time { return now }

	mockClient := temporalmocks.NewClient(t)
	handle := temporalmocks.NewWorkflowUpdateHandle(t)
	mockClient.
		On(
			"UpdateWorkflow",
			mock.Anything,
			mock.MatchedBy(func(options client.UpdateWorkflowOptions) bool {
				req, ok := options.Args[0].(workflows.MobileRunnerSemaphorePauseRunnerRequest)
				return ok &&
					options.WorkflowID == workflows.MobileRunnerSemaphoreWorkflowID("usera-s-organization/stale-runner") &&
					options.UpdateName == workflows.MobileRunnerSemaphorePauseRunnerUpdate &&
					options.WaitForStage == client.WorkflowUpdateStageAccepted &&
					req.Reason == "heartbeat timeout" &&
					req.CancelRunning &&
					req.ShutdownAfterSeconds == int(mobileRunnerLifecycleShutdownAfter/time.Second)
			}),
		).
		Return(handle, nil).
		Once()
	mobileRunnerLifecycleMonitorTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	require.NoError(t, markStaleRunnersOfflineAndPauseSemaphores(context.Background(), app))

	stale, err := canonify.Resolve(app, "/usera-s-organization/stale-runner")
	require.NoError(t, err)
	require.False(t, stale.GetBool("online"))

	fresh, err := canonify.Resolve(app, "/usera-s-organization/fresh-runner")
	require.NoError(t, err)
	require.True(t, fresh.GetBool("online"))
}

func TestMarkRunnerOfflineIfStillStaleSkipsFreshHeartbeat(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	ensureLifecycleMonitorFields(t, app)
	canonify.RegisterCanonifyHooks(app)

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	record := createLifecycleMonitorRunner(t, app, orgID, "fresh-cas-runner", true, now)

	shouldPause, err := markRunnerOfflineIfStillStale(app, record.Id, now.Add(-time.Minute))
	require.NoError(t, err)
	require.False(t, shouldPause)

	record, err = app.FindRecordById("mobile_runners", record.Id)
	require.NoError(t, err)
	require.True(t, record.GetBool("online"))
}

func TestPauseStaleRunnerSemaphoreIgnoresMissingWorkflow(t *testing.T) {
	origClient := mobileRunnerLifecycleMonitorTemporalClient
	t.Cleanup(func() { mobileRunnerLifecycleMonitorTemporalClient = origClient })

	mockClient := temporalmocks.NewClient(t)
	mockClient.
		On("UpdateWorkflow", mock.Anything, mock.Anything).
		Return(nil, &serviceerror.NotFound{})
	mobileRunnerLifecycleMonitorTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	err := pauseStaleRunnerSemaphore(context.Background(), "runner-1", time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
}

func TestMarkRunnerOfflineIfStillStaleAlreadyOffline(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	ensureLifecycleMonitorFields(t, app)
	canonify.RegisterCanonifyHooks(app)

	orgID, err := getOrgIDfromName(app)
	require.NoError(t, err)
	record := createLifecycleMonitorRunner(t, app, orgID, "offline-runner", false, time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC))

	shouldPause, err := markRunnerOfflineIfStillStale(app, record.Id, time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.True(t, shouldPause)
}

func TestRunMobileRunnerLifecycleMonitorStopsOnCanceledContext(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runMobileRunnerLifecycleMonitor(ctx, app)
}

func TestCancelMobileRunnerLifecycleMonitorRemovesStoredCancel(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	called := false
	app.Store().Set(mobileRunnerLifecycleMonitorCancelStoreKey, context.CancelFunc(func() {
		called = true
	}))

	cancelMobileRunnerLifecycleMonitor(app)
	require.True(t, called)
	require.Nil(t, app.Store().Get(mobileRunnerLifecycleMonitorCancelStoreKey))
}
