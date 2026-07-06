// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/mobilerunnerlifecycle"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
)

const (
	mobileRunnerLifecycleMonitorCancelStoreKey = "mobile_runner_lifecycle_monitor_cancel"
)

var mobileRunnerLifecycleMonitorNow = func() time.Time {
	return time.Now().UTC()
}

var mobileRunnerLifecycleMonitorTemporalClient = temporalclient.GetTemporalClientWithNamespace

func bindMobileRunnerLifecycleMonitor(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		cancelMobileRunnerLifecycleMonitor(app)

		ctx, cancel := context.WithCancel(context.Background())
		app.Store().Set(mobileRunnerLifecycleMonitorCancelStoreKey, cancel)

		go runMobileRunnerLifecycleMonitor(ctx, app)
		return se.Next()
	})

	app.OnTerminate().BindFunc(func(_ *core.TerminateEvent) error {
		cancelMobileRunnerLifecycleMonitor(app)
		return nil
	})
}

func cancelMobileRunnerLifecycleMonitor(app core.App) {
	raw := app.Store().Get(mobileRunnerLifecycleMonitorCancelStoreKey)
	cancel, ok := raw.(context.CancelFunc)
	if !ok {
		return
	}
	cancel()
	app.Store().Remove(mobileRunnerLifecycleMonitorCancelStoreKey)
}

func runMobileRunnerLifecycleMonitor(ctx context.Context, app core.App) {
	ticker := time.NewTicker(mobilerunnerlifecycle.MonitorInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := markStaleRunnersOfflineAndPauseSemaphores(ctx, app); err != nil {
				app.Logger().Error("mobile runner lifecycle monitor failed", "error", err)
			}
		}
	}
}

func markStaleRunnersOfflineAndPauseSemaphores(ctx context.Context, app core.App) error {
	cutoff := mobileRunnerLifecycleMonitorNow().Add(-mobilerunnerlifecycle.HeartbeatTimeout())
	records, err := app.FindRecordsByFilter("mobile_runners", "online = true", "", -1, 0)
	if err != nil {
		return fmt.Errorf("list online mobile runners: %w", err)
	}

	for _, record := range records {
		lastHeartbeat := record.GetDateTime("last_heartbeat_at")
		if !lastHeartbeat.IsZero() && !lastHeartbeat.Time().UTC().Before(cutoff) {
			continue
		}

		runnerID, err := mobileRunnerRecordIdentifier(app, record)
		if err != nil {
			app.Logger().
				Error("build stale runner identifier failed", "record_id", record.Id, "error", err)
			continue
		}

		shouldPause, err := markRunnerOfflineIfStillStale(app, record.Id, cutoff)
		if err != nil {
			app.Logger().
				Error("mark stale runner offline failed", "runner_id", runnerID, "error", err)
			continue
		}
		if !shouldPause {
			continue
		}

		if err := pauseStaleRunnerSemaphore(ctx, runnerID, cutoff); err != nil {
			app.Logger().
				Error("pause stale runner semaphore failed", "runner_id", runnerID, "error", err)
		}
	}

	return nil
}

func markRunnerOfflineIfStillStale(app core.App, recordID string, cutoff time.Time) (bool, error) {
	shouldPause := false

	err := app.RunInTransaction(func(txApp core.App) error {
		record, err := txApp.FindRecordById("mobile_runners", recordID)
		if err != nil {
			return err
		}

		lastHeartbeat := record.GetDateTime("last_heartbeat_at")
		if !lastHeartbeat.IsZero() && !lastHeartbeat.Time().UTC().Before(cutoff) {
			return nil
		}
		if !record.GetBool("online") {
			shouldPause = true
			return nil
		}

		record.Set("online", false)
		if err := txApp.Save(record); err != nil {
			return err
		}
		shouldPause = true
		return nil
	})
	if err != nil {
		return false, err
	}

	return shouldPause, nil
}

func pauseStaleRunnerSemaphore(ctx context.Context, runnerID string, cutoff time.Time) error {
	client, err := mobileRunnerLifecycleMonitorTemporalClient(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
	if err != nil {
		return err
	}

	_, err = client.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID: workflows.MobileRunnerSemaphoreWorkflowID(runnerID),
		UpdateName: workflows.MobileRunnerSemaphorePauseRunnerUpdate,
		UpdateID:   heartbeatPauseUpdateID(runnerID, cutoff),
		Args: []any{workflows.MobileRunnerSemaphorePauseRunnerRequest{
			Reason:               "heartbeat timeout",
			CancelRunning:        true,
			ShutdownAfterSeconds: int(mobilerunnerlifecycle.ShutdownAfter() / time.Second),
		}},
		WaitForStage: tclient.WorkflowUpdateStageAccepted,
	})
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return nil
		}
		return err
	}

	return nil
}

func heartbeatPauseUpdateID(runnerID string, cutoff time.Time) string {
	return fmt.Sprintf("pause-heartbeat-timeout/%s/%d", runnerID, cutoff.Unix())
}
