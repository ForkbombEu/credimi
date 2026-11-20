// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
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

// mobileRunnerWorkQueryTimeout bounds each device semaphore query of the
// runner work snapshot.
const mobileRunnerWorkQueryTimeout = 2 * time.Second

// mobileRunnerWorkTotalTimeout bounds all device queries of one runner work
// snapshot, below the runner's own 5s budget for the call. A var so tests
// can shorten it.
var mobileRunnerWorkTotalTimeout = 4 * time.Second

// mobileRunnerWorkMaxConcurrentQueries caps the device semaphore queries of
// one runner work snapshot that run at the same time.
const mobileRunnerWorkMaxConcurrentQueries = 8

var queryMobileDeviceSemaphoreRunTickets = queryMobileDeviceSemaphoreRunTicketsTemporal

type MobileRunnerLifecycleRequest struct {
	RunnerID  string                       `json:"runner_id"            validate:"required"`
	RequestID string                       `json:"request_id,omitempty"`
	Reason    string                       `json:"reason,omitempty"`
	Devices   []MobileDeviceLifecycleState `json:"devices,omitempty"`
	// ActionAuthor carries the runner's action author settings on heartbeat
	// and resume; pause ignores it, and an absent value keeps the stored ones.
	ActionAuthor *RunnerActionAuthorSettings `json:"action_author,omitempty"`
}

type MobileDeviceLifecycleState struct {
	DeviceID string `json:"device_id"        validate:"required"`
	Online   bool   `json:"online"`
	Reason   string `json:"reason,omitempty"`
}

type MobileRunnerLifecycleResponse struct {
	RunnerID                string `json:"runner_id"`
	Online                  bool   `json:"online"`
	SemaphoreWorkflowID     string `json:"semaphore_workflow_id"`
	HeartbeatTimeoutSeconds int    `json:"heartbeat_timeout_seconds"`
	ShutdownAfterSeconds    int    `json:"shutdown_after_seconds"`
	// RunnerCredential is the key Credimi currently presents to this runner.
	RunnerCredential string `json:"runner_credential"`
	// RunnerNamespace is the runner's Temporal namespace; absent until
	// Credimi has prepared it for a runner that is not disabled.
	RunnerNamespace string `json:"runner_namespace,omitempty"`
}

// MobileRunnerWorkResponse lists the active semaphore tickets of every device
// of a runner. Tickets of other organizations keep only their organization,
// kind, status, times and queue position.
type MobileRunnerWorkResponse struct {
	RunnerID    string                   `json:"runner_id"`
	GeneratedAt time.Time                `json:"generated_at"`
	Devices     []MobileRunnerDeviceWork `json:"devices"`
}

type MobileRunnerDeviceWork struct {
	DeviceID    string                                         `json:"device_id"`
	QueryFailed bool                                           `json:"query_failed"`
	Tickets     []workflows.MobileDeviceSemaphoreRunTicketView `json:"tickets"`
}

var MobileRunnerLifecycleRoutes = routing.RouteGroup{
	BaseURL:                "/api/mobile-runner/lifecycle",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		runnerPostRoute(
			"/resume",
			HandleMobileRunnerLifecycleResume,
			MobileRunnerLifecycleRequest{},
			MobileRunnerLifecycleResponse{},
		),
		runnerPostRoute(
			"/heartbeat",
			HandleMobileRunnerLifecycleHeartbeat,
			MobileRunnerLifecycleRequest{},
			MobileRunnerLifecycleResponse{},
		),
		runnerPostRoute(
			"/pause",
			HandleMobileRunnerLifecyclePause,
			MobileRunnerLifecycleRequest{},
			MobileRunnerLifecycleResponse{},
		),
		{
			Method:         http.MethodGet,
			Path:           "/work",
			Handler:        HandleMobileRunnerLifecycleWork,
			ResponseSchema: MobileRunnerWorkResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
		{
			Method:         http.MethodGet,
			Path:           "/live-view",
			Handler:        HandleMobileRunnerLifecycleLiveView,
			ResponseSchema: MobileRunnerLiveViewStatusResponse{},
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

		record, runnerID, apiErr := resolveLifecycleRunner(e.App, e.Auth, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}

		settings := trustedActionAuthorSettings(e, record, input.ActionAuthor)
		deviceStates, err := applyRunnerHeartbeatDevices(
			e.App,
			record,
			input.Devices,
			settings,
			mobileRunnerLifecycleNow(),
		)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_runner",
				"failed_to_apply_device_resume",
				err.Error(),
			)
		}
		for deviceID, online := range deviceStates {
			if !online {
				continue
			}
			if err := ensureRunQueueSemaphoreWorkflow(
				e.Request.Context(),
				deviceID,
			); err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"mobile_device",
					"failed_to_ensure_device_semaphore",
					err.Error(),
				)
			}
			_, err := updateRunnerSemaphore(
				e.Request.Context(),
				deviceID,
				workflows.MobileDeviceSemaphoreResumeDeviceUpdate,
				workflows.MobileDeviceSemaphoreResumeDeviceRequest{
					Reason: lifecycleReason(input.Reason, "runner_startup"),
				},
				nil,
				lifecycleUpdateID("resume", deviceID, input.RequestID),
			)
			if err != nil && !errors.Is(err, errSemaphoreNotFound) {
				return apierror.New(
					http.StatusInternalServerError,
					"mobile_device",
					"failed_to_resume_device_semaphore",
					err.Error(),
				)
			}
		}

		return lifecycleResponse(e, record, runnerID, true)
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

		record, runnerID, apiErr := resolveLifecycleRunner(e.App, e.Auth, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}

		settings := trustedActionAuthorSettings(e, record, input.ActionAuthor)
		deviceStates, err := applyRunnerHeartbeatDevices(
			e.App,
			record,
			input.Devices,
			settings,
			mobileRunnerLifecycleNow(),
		)
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
				if err := ensureRunQueueSemaphoreWorkflow(
					e.Request.Context(),
					deviceID,
				); err != nil {
					return apierror.New(
						http.StatusInternalServerError,
						"mobile_device",
						"failed_to_ensure_device_semaphore",
						err.Error(),
					)
				}
				if !heartbeatNeedsSemaphoreUpdate(e.Request.Context(), deviceID, true) {
					continue
				}
				if err := resumeRunnerSemaphore(
					e.Request.Context(),
					deviceID,
					lifecycleReason(input.Reason, "runner_heartbeat"),
					input.RequestID,
				); err != nil {
					return apierror.New(
						http.StatusInternalServerError,
						"mobile_device",
						"failed_to_resume_device_semaphore",
						err.Error(),
					)
				}
				continue
			}
			if !heartbeatNeedsSemaphoreUpdate(e.Request.Context(), deviceID, false) {
				continue
			}
			_, err := updateRunnerSemaphore(
				e.Request.Context(),
				deviceID,
				workflows.MobileDeviceSemaphorePauseDeviceUpdate,
				workflows.MobileDeviceSemaphorePauseDeviceRequest{
					Reason:               "device offline",
					CancelRunning:        true,
					ShutdownAfterSeconds: int(mobilerunnerlifecycle.ShutdownAfter() / time.Second),
				},
				nil,
				lifecycleUpdateID("pause", deviceID, input.RequestID),
			)
			if err != nil && !errors.Is(err, errSemaphoreNotFound) {
				return apierror.New(
					http.StatusInternalServerError,
					"mobile_device",
					"failed_to_pause_device_semaphore",
					err.Error(),
				)
			}
		}

		return lifecycleResponse(e, record, runnerID, true)
	}
}

// heartbeatNeedsSemaphoreUpdate reports whether a heartbeat must change the
// device semaphore. Heartbeats repeat every interval with a fresh request ID,
// so each pause or resume they send is a distinct Temporal update; the server
// caps those per workflow run, and a device that stays offline would otherwise
// add one every heartbeat until the semaphore rejects all updates. A failed
// query still sends the update so the semaphore cannot drift from the runner.
func heartbeatNeedsSemaphoreUpdate(ctx context.Context, deviceID string, online bool) bool {
	state, err := queryMobileDeviceSemaphoreState(ctx, deviceID)
	if errors.Is(err, errSemaphoreNotFound) {
		return false
	}
	if err != nil {
		return true
	}
	return state.Paused == online
}

func applyRunnerHeartbeatDevices(
	app core.App,
	runner *core.Record,
	reported []MobileDeviceLifecycleState,
	actionAuthor *RunnerActionAuthorSettings,
	now time.Time,
) (map[string]bool, error) {
	states := map[string]bool{}
	err := app.RunInTransaction(func(txApp core.App) error {
		currentRunner, err := txApp.FindRecordById("mobile_runners", runner.Id)
		if err != nil {
			return err
		}
		setRunnerHeartbeat(currentRunner, true, now)
		setRunnerActionAuthorSettings(currentRunner, actionAuthor)
		if err := txApp.Save(currentRunner); err != nil {
			return err
		}
		devices, err := txApp.FindRecordsByFilter(
			"mobile_devices",
			"runner = {:runner}",
			"",
			-1,
			0,
			dbx.Params{"runner": runner.Id},
		)
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
			// Resolve the canonical child directly. Relation-filter queries can
			// lag or behave differently across PocketBase relation storage
			// variants; the record itself remains the source of truth for the
			// ownership check performed by a heartbeat.
			device, err := canonify.Resolve(txApp, identifier)
			if err != nil || device == nil || device.Collection() == nil ||
				device.Collection().Name != mobileDevicesCollection ||
				device.GetString("runner") != currentRunner.Id {
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

		record, runnerID, apiErr := resolveLifecycleRunner(e.App, e.Auth, input.RunnerID)
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
				lifecycleUpdateID("pause", deviceID, input.RequestID),
			)
			if updateErr != nil && !errors.Is(updateErr, errSemaphoreNotFound) {
				return apierror.New(
					http.StatusInternalServerError,
					"mobile_device",
					"failed_to_pause_device_semaphore",
					updateErr.Error(),
				)
			}
		}

		return lifecycleResponse(e, record, runnerID, false)
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
		devices, err := txApp.FindRecordsByFilter(
			"mobile_devices",
			"runner = {:runner}",
			"",
			-1,
			0,
			dbx.Params{"runner": runner.Id},
		)
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

// runnerOwnerMismatchError refuses a non-superuser outside the runner's
// organization.
func runnerOwnerMismatchError() *apierror.APIError {
	return apierror.New(
		http.StatusForbidden,
		"runner_id",
		"runner_owner_mismatch",
		"runner_id does not belong to the authenticated organization",
	)
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", runnerOwnerMismatchError()
		}
		if err != nil {
			return nil, "", apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed_to_find_user_organization",
				err.Error(),
			)
		}
		if record.GetString("owner") != orgID {
			return nil, "", runnerOwnerMismatchError()
		}
	}

	canonicalDeviceID, err := mobilerunner.RunnerIdentifier(app, record)
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

func resumeRunnerSemaphore(ctx context.Context, deviceID, reason, requestID string) error {
	_, err := updateRunnerSemaphore(
		ctx,
		deviceID,
		workflows.MobileDeviceSemaphoreResumeDeviceUpdate,
		workflows.MobileDeviceSemaphoreResumeDeviceRequest{Reason: reason},
		nil,
		lifecycleUpdateID("resume", deviceID, requestID),
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

// lifecycleResponse writes the lifecycle response for runner, carrying the
// credential Credimi currently presents to it when the caller may hold it.
func lifecycleResponse(
	e *core.RequestEvent,
	runner *core.Record,
	runnerID string,
	online bool,
) error {
	credential, apiErr := mobileRunnerCredential(e.Auth, runner)
	if apiErr != nil {
		return apiErr
	}
	return e.JSON(http.StatusOK, MobileRunnerLifecycleResponse{
		RunnerID:                runnerID,
		Online:                  online,
		SemaphoreWorkflowID:     workflows.MobileDeviceSemaphoreWorkflowID(runnerID),
		HeartbeatTimeoutSeconds: int(mobilerunnerlifecycle.HeartbeatTimeout() / time.Second),
		ShutdownAfterSeconds:    int(mobilerunnerlifecycle.ShutdownAfter() / time.Second),
		RunnerCredential:        credential,
		RunnerNamespace:         runnerNamespace(runner),
	})
}

// mobileRunnerCredential returns the credential Credimi presents to runner;
// runners learn it from registration and lifecycle responses. An
// admin-managed runner is operator infrastructure used by every organization,
// so its credential goes only to superusers and the internal admin key it
// registers with, never to the members of the organization that owns the
// record; they get an empty value.
func mobileRunnerCredential(auth *core.Record, runner *core.Record) (string, *apierror.APIError) {
	if runner.GetBool("admin_managed") && !isSuperuserAuth(auth) {
		return "", nil
	}
	credential, err := mobilerunner.Credential(runner)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"mobile_runner",
			"runner_credential_unavailable",
			err.Error(),
		)
	}
	return credential, nil
}

func lifecycleReason(reason string, fallback string) string {
	reason = strings.TrimSpace(reason)
	if reason != "" {
		return reason
	}
	return fallback
}

func lifecycleUpdateID(action, runnerID, requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		requestID = strconv.FormatInt(mobileRunnerLifecycleNow().UnixNano(), 10)
	}
	return fmt.Sprintf("%s/%s/%s", action, runnerID, requestID)
}

// HandleMobileRunnerLifecycleWork returns the running and queued semaphore
// tickets of every device of a runner, redacting other organizations' runs.
// Like the runner credential, the work of an admin-managed runner, operator
// infrastructure used by every organization, goes only to superusers and the
// internal admin key.
func HandleMobileRunnerLifecycleWork() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		runner, runnerID, apiErr := resolveLifecycleRunner(
			e.App,
			e.Auth,
			e.Request.URL.Query().Get("runner_id"),
		)
		if apiErr != nil {
			return apiErr
		}
		if runner.GetBool("admin_managed") && !isSuperuserAuth(e.Auth) {
			return apierror.New(
				http.StatusForbidden,
				"mobile_runner",
				"admin_managed_runner_work_forbidden",
				"only superusers can list the work of an admin-managed runner",
			)
		}

		ownerNamespace, err := pbutils.GetOrganizationCanonifiedName(
			e.App,
			runner.GetString("owner"),
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"mobile_runner",
				"owner_lookup_failed",
				err.Error(),
			)
		}

		deviceIDs, err := runnerDeviceIdentifiers(e.App, runner)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"mobile_runner",
				"failed_to_list_devices",
				err.Error(),
			)
		}

		client, err := mobileRunnerLifecycleTemporalClient(
			workflowengine.MobileDeviceSemaphoreDefaultNamespace,
		)
		if err != nil {
			return temporalUnavailableError()
		}

		devices := mobileRunnerDevicesWork(
			e.Request.Context(),
			client,
			runnerID,
			deviceIDs,
			ownerNamespace,
		)

		return e.JSON(http.StatusOK, MobileRunnerWorkResponse{
			RunnerID:    runnerID,
			GeneratedAt: mobileRunnerLifecycleNow().UTC(),
			Devices:     devices,
		})
	}
}

// runnerDeviceIdentifiers lists the sorted device paths of a runner, skipping
// devices whose path cannot be built.
func runnerDeviceIdentifiers(app core.App, runner *core.Record) ([]string, error) {
	records, err := app.FindRecordsByFilter(
		"mobile_devices",
		"runner = {:runner}",
		"",
		-1,
		0,
		dbx.Params{"runner": runner.Id},
	)
	if err != nil {
		return nil, err
	}
	deviceIDs := make([]string, 0, len(records))
	for _, device := range records {
		identifier, err := mobileDeviceIdentifier(app, device)
		if err != nil {
			app.Logger().Warn(
				"mobile runner work skipped device",
				"device",
				device.Id,
				"error",
				err,
			)
			continue
		}
		deviceIDs = append(deviceIDs, identifier)
	}
	sort.Strings(deviceIDs)
	return deviceIDs, nil
}

// mobileRunnerDevicesWork queries the devices concurrently, at most
// mobileRunnerWorkMaxConcurrentQueries at a time, within
// mobileRunnerWorkTotalTimeout overall. The result keeps the order of
// deviceIDs; a device whose query did not answer in time is QueryFailed.
func mobileRunnerDevicesWork(
	ctx context.Context,
	client tclient.Client,
	runnerID string,
	deviceIDs []string,
	ownerNamespace string,
) []MobileRunnerDeviceWork {
	ctx, cancel := context.WithTimeout(ctx, mobileRunnerWorkTotalTimeout)
	defer cancel()

	devices := make([]MobileRunnerDeviceWork, len(deviceIDs))
	slots := make(chan struct{}, mobileRunnerWorkMaxConcurrentQueries)
	var wg sync.WaitGroup
	for i, deviceID := range deviceIDs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			case <-ctx.Done():
				devices[i] = MobileRunnerDeviceWork{
					DeviceID:    deviceID,
					QueryFailed: true,
					Tickets:     []workflows.MobileDeviceSemaphoreRunTicketView{},
				}
				return
			}
			devices[i] = mobileRunnerDeviceWork(ctx, client, runnerID, deviceID, ownerNamespace)
		}()
	}
	wg.Wait()
	return devices
}

func mobileRunnerDeviceWork(
	ctx context.Context,
	client tclient.Client,
	runnerID string,
	deviceID string,
	ownerNamespace string,
) MobileRunnerDeviceWork {
	ctx, cancel := context.WithTimeout(ctx, mobileRunnerWorkQueryTimeout)
	defer cancel()

	work := MobileRunnerDeviceWork{
		DeviceID: deviceID,
		Tickets:  []workflows.MobileDeviceSemaphoreRunTicketView{},
	}
	tickets, err := queryMobileDeviceSemaphoreRunTickets(ctx, client, deviceID)
	if err != nil {
		work.QueryFailed = true
		return work
	}
	for _, ticket := range tickets {
		if ticket.OwnerNamespace != ownerNamespace {
			ticket.TicketID = redactedMobileRunnerTicketID(runnerID, ticket.TicketID)
			ticket.PipelineIdentifier = ""
			ticket.WorkflowID = ""
			ticket.RunID = ""
			ticket.WorkflowNamespace = ""
		}
		work.Tickets = append(work.Tickets, ticket)
	}
	return work
}

// redactedMobileRunnerTicketID hides the ticket ID of another organization's
// run, which names its pipeline, behind an opaque ID that stays the same
// across the runner's devices and calls, so the runner can still dedupe it.
func redactedMobileRunnerTicketID(runnerID string, ticketID string) string {
	sum := sha256.Sum256([]byte(runnerID + "\x00" + ticketID))
	return "redacted-" + hex.EncodeToString(sum[:])[:16]
}

func queryMobileDeviceSemaphoreRunTicketsTemporal(
	ctx context.Context,
	client tclient.Client,
	deviceID string,
) ([]workflows.MobileDeviceSemaphoreRunTicketView, error) {
	encoded, err := client.QueryWorkflow(
		ctx,
		workflows.MobileDeviceSemaphoreWorkflowID(deviceID),
		"",
		workflows.MobileDeviceSemaphoreListRunTicketsQuery,
	)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return nil, nil
		}
		return nil, err
	}

	var tickets []workflows.MobileDeviceSemaphoreRunTicketView
	if err := encoded.Get(&tickets); err != nil {
		return nil, err
	}
	return tickets, nil
}
