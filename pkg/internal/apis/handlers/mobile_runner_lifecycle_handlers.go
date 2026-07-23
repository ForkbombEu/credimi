// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunnerlifecycle"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
)

var mobileRunnerLifecycleNow = func() time.Time {
	return time.Now().UTC()
}

var mobileRunnerLifecycleTemporalClient = temporalclient.GetTemporalClientWithNamespace

type MobileRunnerLifecycleRequest struct {
	DeviceID string                       `json:"runner_id"        validate:"required"`
	Reason   string                       `json:"reason,omitempty"`
	Devices  []MobileDeviceLifecycleState `json:"devices,omitempty"`
}

type MobileDeviceLifecycleState struct {
	DeviceID string `json:"device_id" validate:"required"`
	Online   bool   `json:"online"`
	Reason   string `json:"reason,omitempty"`
}

type MobileRunnerLifecycleResponse struct {
	DeviceID                string `json:"runner_id"`
	Online                  bool   `json:"online"`
	SemaphoreWorkflowID     string `json:"semaphore_workflow_id"`
	HeartbeatTimeoutSeconds int    `json:"heartbeat_timeout_seconds"`
	ShutdownAfterSeconds    int    `json:"shutdown_after_seconds"`
}

var MobileRunnerLifecycleRoutes = routing.RouteGroup{
	BaseURL:                "/api/mobile-runner/lifecycle",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodPost,
			Path:           "/resume",
			Handler:        HandleMobileRunnerLifecycleResume,
			RequestSchema:  MobileRunnerLifecycleRequest{},
			ResponseSchema: MobileRunnerLifecycleResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "/heartbeat",
			Handler:        HandleMobileRunnerLifecycleHeartbeat,
			RequestSchema:  MobileRunnerLifecycleRequest{},
			ResponseSchema: MobileRunnerLifecycleResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "/pause",
			Handler:        HandleMobileRunnerLifecyclePause,
			RequestSchema:  MobileRunnerLifecycleRequest{},
			ResponseSchema: MobileRunnerLifecycleResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
	},
}

func HandleMobileRunnerLifecycleResume() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[MobileRunnerLifecycleRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_runner",
				"invalid_request",
				err.Error(),
			)
		}

		record, runnerID, apiErr := resolveLifecycleRunner(e.App, e.Auth, input.DeviceID)
		if apiErr != nil {
			return apiErr
		}

		now := mobileRunnerLifecycleNow()
		setRunnerHeartbeat(record, true, now)
		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"mobile_runner",
				"failed_to_save_mobile_runner",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, lifecycleResponse(runnerID, true))
	}
}

func HandleMobileRunnerLifecycleHeartbeat() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[MobileRunnerLifecycleRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_runner",
				"invalid_request",
				err.Error(),
			)
		}

		record, runnerID, apiErr := resolveLifecycleRunner(e.App, e.Auth, input.DeviceID)
		if apiErr != nil {
			return apiErr
		}

		deviceStates, err := applyRunnerHeartbeatDevices(e.App, record, input.Devices, mobileRunnerLifecycleNow())
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_runner",
				"failed_to_apply_device_heartbeat",
				err.Error(),
			)
		}
		for deviceID, online := range deviceStates {
			if online {
				if err := ensureRunQueueSemaphoreWorkflowTemporal(e.Request.Context(), deviceID); err != nil {
					return apierror.New(http.StatusInternalServerError, "mobile_device", "failed_to_ensure_device_semaphore", err.Error())
				}
				if err := resumeHeartbeatPausedRunnerSemaphore(e.Request.Context(), deviceID); err != nil {
					return apierror.New(http.StatusInternalServerError, "mobile_device", "failed_to_resume_device_semaphore", err.Error())
				}
				continue
			}
			_, err := updateRunnerSemaphore(e.Request.Context(), deviceID, workflows.MobileDeviceSemaphorePauseDeviceUpdate, workflows.MobileDeviceSemaphorePauseDeviceRequest{Reason: "device offline", CancelRunning: true, ShutdownAfterSeconds: int(mobilerunnerlifecycle.ShutdownAfter() / time.Second)}, nil, lifecycleUpdateID("pause", deviceID))
			if err != nil && !errors.Is(err, errSemaphoreNotFound) {
				return apierror.New(http.StatusInternalServerError, "mobile_device", "failed_to_pause_device_semaphore", err.Error())
			}
		}

		return e.JSON(http.StatusOK, lifecycleResponse(runnerID, true))
	}
}

func applyRunnerHeartbeatDevices(app core.App, runner *core.Record, reported []MobileDeviceLifecycleState, now time.Time) (map[string]bool, error) {
	states := map[string]bool{}
	err := app.RunInTransaction(func(txApp core.App) error {
		currentRunner, err := txApp.FindRecordById("mobile_runners", runner.Id)
		if err != nil {
			return err
		}
		setRunnerHeartbeat(currentRunner, true, now)
		if err := txApp.Save(currentRunner); err != nil {
			return err
		}
		devices, err := txApp.FindRecordsByFilter("mobile_devices", "runner = {:runner}", "", -1, 0, dbx.Params{"runner": runner.Id})
		if err != nil {
			return err
		}
		byID := map[string]*core.Record{}
		for _, device := range devices {
			identifier, err := mobileDeviceIdentifier(txApp, device)
			if err != nil {
				return err
			}
			byID[identifier] = device
		}
		for _, report := range reported {
			identifier := canonify.NormalizePath(report.DeviceID)
			device, ok := byID[identifier]
			if !ok {
				return fmt.Errorf("device_id %q does not belong to runner", report.DeviceID)
			}
			device.Set("online", report.Online)
			if err := txApp.Save(device); err != nil {
				return err
			}
			states[identifier] = report.Online
		}
		for identifier, device := range byID {
			if _, ok := states[identifier]; ok {
				continue
			}
			device.Set("online", false)
			if err := txApp.Save(device); err != nil {
				return err
			}
			states[identifier] = false
		}
		return nil
	})
	return states, err
}

func HandleMobileRunnerLifecyclePause() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[MobileRunnerLifecycleRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_runner",
				"invalid_request",
				err.Error(),
			)
		}

		record, runnerID, apiErr := resolveLifecycleRunner(e.App, e.Auth, input.DeviceID)
		if apiErr != nil {
			return apiErr
		}

		deviceIDs, err := markRunnerAndDevicesOffline(e.App, record)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"mobile_runner",
				"failed_to_pause_runner_devices",
				err.Error(),
			)
		}
		for _, deviceID := range deviceIDs {
			_, updateErr := updateRunnerSemaphore(
				e.Request.Context(),
				deviceID,
				workflows.MobileDeviceSemaphorePauseDeviceUpdate,
				workflows.MobileDeviceSemaphorePauseDeviceRequest{
					Reason:               lifecycleReason(input.Reason, "runner_shutdown"),
					CancelRunning:        true,
					ShutdownAfterSeconds: int(mobilerunnerlifecycle.ShutdownAfter() / time.Second),
				},
				nil,
				lifecycleUpdateID("pause", deviceID),
			)
			if updateErr != nil && !errors.Is(updateErr, errSemaphoreNotFound) {
				return apierror.New(http.StatusInternalServerError, "mobile_device", "failed_to_pause_device_semaphore", updateErr.Error())
			}
		}

		return e.JSON(http.StatusOK, lifecycleResponse(runnerID, false))
	}
}

func markRunnerAndDevicesOffline(app core.App, runner *core.Record) ([]string, error) {
	deviceIDs := []string{}
	err := app.RunInTransaction(func(txApp core.App) error {
		currentRunner, err := txApp.FindRecordById("mobile_runners", runner.Id)
		if err != nil {
			return err
		}
		currentRunner.Set("online", false)
		if err := txApp.Save(currentRunner); err != nil {
			return err
		}
		devices, err := txApp.FindRecordsByFilter("mobile_devices", "runner = {:runner}", "", -1, 0, dbx.Params{"runner": runner.Id})
		if err != nil {
			return err
		}
		for _, device := range devices {
			device.Set("online", false)
			if err := txApp.Save(device); err != nil {
				return err
			}
			deviceID, err := mobileDeviceIdentifier(txApp, device)
			if err != nil {
				return err
			}
			deviceIDs = append(deviceIDs, deviceID)
		}
		sort.Strings(deviceIDs)
		return nil
	})
	return deviceIDs, err
}

func resolveLifecycleRunner(
	app core.App,
	auth *core.Record,
	runnerID string,
) (*core.Record, string, *apierror.APIError) {
	normalizedDeviceID := canonify.NormalizePath(runnerID)
	if normalizedDeviceID == "" {
		return nil, "", apierror.New(
			http.StatusBadRequest,
			"runner_id",
			"runner_id_required",
			"runner_id is required",
		)
	}

	record, err := canonify.Resolve(app, normalizedDeviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", apierror.New(
				http.StatusNotFound,
				"runner_id",
				"mobile_runner_not_found",
				"mobile runner not found",
			)
		}
		return nil, "", apierror.New(
			http.StatusInternalServerError,
			"runner_id",
			"failed_to_resolve_runner_id",
			err.Error(),
		)
	}
	if record.Collection() == nil || record.Collection().Name != "mobile_runners" {
		return nil, "", apierror.New(
			http.StatusBadRequest,
			"runner_id",
			"invalid_runner_id",
			"runner_id does not reference a mobile runner",
		)
	}

	if !isSuperuserAuth(auth) {
		orgID, err := pbutils.GetUserOrganizationID(app, auth.Id)
		if err != nil {
			return nil, "", apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed_to_find_user_organization",
				err.Error(),
			)
		}
		if record.GetString("owner") != orgID {
			return nil, "", apierror.New(
				http.StatusForbidden,
				"runner_id",
				"runner_owner_mismatch",
				"runner_id does not belong to the authenticated organization",
			)
		}
	}

	canonicalDeviceID, err := mobileRunnerIdentifier(app, record)
	if err != nil {
		return nil, "", apierror.New(
			http.StatusInternalServerError,
			"runner_id",
			"failed_to_build_runner_id",
			err.Error(),
		)
	}

	return record, canonicalDeviceID, nil
}

func resumeHeartbeatPausedRunnerSemaphore(ctx context.Context, runnerID string) error {
	state, err := queryMobileDeviceSemaphoreState(ctx, runnerID)
	if err != nil {
		if errors.Is(err, errSemaphoreNotFound) {
			return nil
		}
		return err
	}
	if !state.Paused || strings.TrimSpace(state.PauseReason) != "heartbeat timeout" {
		return nil
	}

	_, err = updateRunnerSemaphore(
		ctx,
		runnerID,
		workflows.MobileDeviceSemaphoreResumeDeviceUpdate,
		workflows.MobileDeviceSemaphoreResumeDeviceRequest{Reason: "heartbeat_recovered"},
		nil,
		lifecycleUpdateID("heartbeat-resume", runnerID),
	)
	if errors.Is(err, errSemaphoreNotFound) {
		return nil
	}
	return err
}

func updateRunnerSemaphore(
	ctx context.Context,
	runnerID string,
	updateName string,
	req any,
	out any,
	updateID string,
) (bool, error) {
	client, err := mobileRunnerLifecycleTemporalClient(
		workflowengine.MobileDeviceSemaphoreDefaultNamespace,
	)
	if err != nil {
		return false, err
	}

	handle, err := client.UpdateWorkflow(ctx, tclient.UpdateWorkflowOptions{
		WorkflowID:   workflows.MobileDeviceSemaphoreWorkflowID(runnerID),
		UpdateName:   updateName,
		UpdateID:     updateID,
		Args:         []any{req},
		WaitForStage: tclient.WorkflowUpdateStageAccepted,
	})
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return false, errSemaphoreNotFound
		}
		return false, err
	}
	if out != nil {
		if err := handle.Get(ctx, out); err != nil {
			return true, err
		}
		return true, nil
	}
	return true, nil
}

func setRunnerHeartbeat(record *core.Record, online bool, now time.Time) {
	record.Set("online", online)
	record.Set("last_heartbeat_at", now.UTC().Format("2006-01-02 15:04:05.000Z"))
}

func lifecycleResponse(
	runnerID string,
	online bool,
) MobileRunnerLifecycleResponse {
	return MobileRunnerLifecycleResponse{
		DeviceID:                runnerID,
		Online:                  online,
		SemaphoreWorkflowID:     workflows.MobileDeviceSemaphoreWorkflowID(runnerID),
		HeartbeatTimeoutSeconds: int(mobilerunnerlifecycle.HeartbeatTimeout() / time.Second),
		ShutdownAfterSeconds:    int(mobilerunnerlifecycle.ShutdownAfter() / time.Second),
	}
}

func lifecycleReason(reason string, fallback string) string {
	reason = strings.TrimSpace(reason)
	if reason != "" {
		return reason
	}
	return fallback
}

func lifecycleUpdateID(action string, runnerID string) string {
	return fmt.Sprintf("%s/%s/%d", action, runnerID, mobileRunnerLifecycleNow().UnixNano())
}
