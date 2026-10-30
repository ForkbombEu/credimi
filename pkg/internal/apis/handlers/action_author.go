// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/authorsession"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/forkbombeu/credimi/pkg/workflowengine/semaphoreclient"
	"github.com/google/uuid"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// ActionAuthorRoutes starts sessions after authenticating and authorizing their Credimi actor.
var ActionAuthorRoutes = routing.RouteGroup{
	BaseURL:                "/api/action-author",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodGet,
			Path:           "/wallet-versions",
			Handler:        HandleActionAuthorWalletVersions,
			OperationID:    "actionAuthor.listWalletVersions",
			Summary:        "List wallet versions available for authoring",
			Description:    "List wallet versions with an Android APK or iOS simulator app in organizations where the authenticated caller is an owner, with the platforms each version supports. These are the only versions accepted by the session start endpoint.",
			ResponseSchema: ActionAuthorWalletVersionsResponse{},
		},
		{
			Method:         http.MethodGet,
			Path:           "/devices",
			Handler:        HandleActionAuthorDevices,
			OperationID:    "actionAuthor.listDevices",
			Summary:        "List devices available for authoring",
			Description:    "List the Android devices and iOS simulators of enabled admin-managed runners owned by organizations where the authenticated caller is an owner (superusers: every admin-managed runner), with the authoring platform each device implies. Sorted by runner name, then device name.",
			ResponseSchema: ActionAuthorDevicesResponse{},
		},
		{
			Method:         http.MethodPost,
			Path:           "/sessions",
			Handler:        HandleActionAuthorStart,
			OperationID:    "actionAuthor.startSession",
			Summary:        "Queue a wallet-action authoring session on a runner device",
			Description:    "Authorize the caller for the selected wallet organization and queue an authoring session on the selected device through the mobile-device semaphore. The device type decides the platform (Android devices: android, iOS simulators: ios), and the version must have an installer for it. The device must belong to an enabled, online admin-managed runner of the wallet's organization; a device of an organization the caller does not own answers 404 device_not_found, and a disabled runner answers like an offline one (503). Only the onboarding slot is supported. Poll GET /api/pipeline/queue/{ticket_id}?device_ids=<device_id>: once running, its workflow_id is the session and <workflow_id>-author is the chat workflow. Review and explicit human confirmation happen in the harness UI; this endpoint does not save an action.",
			ResponseStatus: http.StatusAccepted,
			RequestSchema:  ActionAuthorStartRequest{},
			ResponseSchema: ActionAuthorStartResponse{},
		},
	},
}

// ActionAuthorWalletVersionsResponse lists builds selectable by the authenticated actor.
type ActionAuthorWalletVersionsResponse struct {
	WalletVersions []actionauthor.BuildOption `json:"wallet_versions"`
}

// HandleActionAuthorWalletVersions lists only versions the caller can use to start authoring.
func HandleActionAuthorWalletVersions() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return authoringAPIError(actionauthor.ErrForbidden)
		}
		versions, err := actionauthor.ListBuildOptions(e.App, authoringActor(e.Auth))
		if err != nil {
			return authoringAPIError(err)
		}
		return e.JSON(http.StatusOK, ActionAuthorWalletVersionsResponse{WalletVersions: versions})
	}
}

// ActionAuthorDevice is one runner device the caller can author on.
type ActionAuthorDevice struct {
	Path       string `json:"path"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	RunnerName string `json:"runner_name"`
	Type       string `json:"type"`
	Platform   string `json:"platform"`
	IsOnline   bool   `json:"is_online"`
}

// ActionAuthorDevicesResponse lists the devices accepted by the session start endpoint.
type ActionAuthorDevicesResponse struct {
	Devices []ActionAuthorDevice `json:"devices"`
}

// HandleActionAuthorDevices lists the supported devices of admin-managed
// runners owned by the caller's organizations.
func HandleActionAuthorDevices() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return authoringAPIError(actionauthor.ErrForbidden)
		}
		ownerFilter, err := actionauthor.OwnerFilter(e.App, authoringActor(e.Auth))
		if err != nil {
			return authoringAPIError(err)
		}
		runners, err := e.App.FindAllRecords(
			"mobile_runners",
			dbx.HashExp{"admin_managed": true},
			ownerFilter,
		)
		if err != nil {
			return authoringAPIError(fmt.Errorf("list authoring runners: %w", err))
		}
		response := ActionAuthorDevicesResponse{Devices: []ActionAuthorDevice{}}
		now := time.Now()
		for _, runner := range enabledMobileRunnerRecords(runners) {
			// Heartbeat freshness, as GET /api/mobile-devices reports it.
			runnerOnline := mobilerunner.RecentlyAlive(runner, now) &&
				mobilerunner.URLUsable(mobilerunner.RunnerURL(runner))
			owner, err := e.App.FindRecordById("organizations", runner.GetString("owner"))
			if err != nil {
				return authoringAPIError(fmt.Errorf("find runner organization: %w", err))
			}
			devices, err := e.App.FindAllRecords(
				"mobile_devices",
				dbx.HashExp{"runner": runner.Id},
			)
			if err != nil {
				return authoringAPIError(fmt.Errorf("list authoring devices: %w", err))
			}
			for _, device := range devices {
				platform := actionAuthorPlatform(device.GetString("type"))
				if platform == "" {
					continue
				}
				path, err := mobileDeviceIdentifier(e.App, device)
				if err != nil {
					return authoringAPIError(fmt.Errorf("build device path: %w", err))
				}
				response.Devices = append(response.Devices, ActionAuthorDevice{
					Path:       path,
					Namespace:  owner.GetString("canonified_name"),
					Name:       device.GetString("name"),
					RunnerName: runner.GetString("name"),
					Type:       device.GetString("type"),
					Platform:   platform,
					IsOnline:   runnerOnline && device.GetBool("online"),
				})
			}
		}
		sort.SliceStable(response.Devices, func(i, j int) bool {
			left, right := response.Devices[i], response.Devices[j]
			if left.RunnerName != right.RunnerName {
				return left.RunnerName < right.RunnerName
			}
			return left.Name < right.Name
		})
		return e.JSON(http.StatusOK, response)
	}
}

// ActionAuthorStartRequest selects the device, wallet version and action slot for a new session.
type ActionAuthorStartRequest struct {
	DeviceID                string `json:"device_id"                 validate:"required"`
	WalletVersionIdentifier string `json:"wallet_version_identifier" validate:"required"`
	Slot                    string `json:"slot"                      validate:"required,oneof=onboarding"`
	AppID                   string `json:"app_id,omitempty"`
}

// ActionAuthorStartResponse identifies the queued session ticket.
type ActionAuthorStartResponse struct {
	TicketID                string `json:"ticket_id"`
	DeviceID                string `json:"device_id"`
	Status                  string `json:"status"`
	Position                int    `json:"position"`
	WorkflowNamespace       string `json:"workflow_namespace"`
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
}

// enqueueActionAuthorTicket enqueues a session ticket and is stubbed in unit tests.
var enqueueActionAuthorTicket = semaphoreclient.EnqueueRun

func authoringActor(auth *core.Record) actionauthor.Actor {
	return actionauthor.Actor{ActorID: auth.Id, ActorCollection: auth.Collection().Name}
}

// actionAuthorPlatform maps a device type to its authoring platform; it is
// empty for types the action author does not support (ios_phone needs a
// simulator clone to reset).
func actionAuthorPlatform(deviceType string) string {
	switch deviceType {
	case "android_phone", "android_emulator", "redroid":
		return actionauthor.PlatformAndroid
	case "ios_simulator":
		return actionauthor.PlatformIOS
	default:
		return ""
	}
}

func authoringAPIError(err error) *apierror.APIError {
	switch {
	case errors.Is(err, actionauthor.ErrForbidden):
		return apierror.New(
			http.StatusForbidden,
			"action_author",
			"forbidden",
			"not authorized to author actions for this wallet",
		)
	case errors.Is(err, actionauthor.ErrInvalid):
		return apierror.New(http.StatusBadRequest, "action_author", "invalid_request", err.Error())
	case errors.Is(err, actionauthor.ErrConflict):
		return apierror.New(http.StatusConflict, "action_author", "conflict", err.Error())
	default:
		return apierror.New(
			http.StatusInternalServerError,
			"action_author",
			"operation_failed",
			"action author operation failed",
		)
	}
}

// HandleActionAuthorStart queues a session on an admin-managed runner device
// of the wallet's organization.
func HandleActionAuthorStart() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[ActionAuthorStartRequest](e)
		if err != nil {
			return authoringAPIError(actionauthor.ErrInvalid)
		}
		if e.Auth == nil {
			return authoringAPIError(actionauthor.ErrForbidden)
		}
		deviceID := canonify.NormalizePath(input.DeviceID)
		actor := authoringActor(e.Auth)
		device, runner, err := mobilerunner.ResolveDevice(e.App, deviceID)
		switch {
		case errors.Is(err, mobilerunner.ErrDeviceNotFound),
			errors.Is(err, mobilerunner.ErrDeviceRunnerNotFound):
			return actionAuthorDeviceNotFound(deviceID)
		case err != nil:
			return authoringAPIError(err)
		}
		// Devices of organizations the caller does not own answer as unknown
		// devices, before their type or runner is revealed.
		visible, err := actionAuthorRunnerVisible(e.App, actor, runner)
		if err != nil {
			return authoringAPIError(err)
		}
		if !visible {
			return actionAuthorDeviceNotFound(deviceID)
		}
		platform := actionAuthorPlatform(device.GetString("type"))
		if platform == "" {
			return apierror.New(
				http.StatusUnprocessableEntity,
				"action_author",
				"device_type_not_supported",
				"the action author supports Android devices and iOS simulators",
			)
		}
		// The device list hides disabled runners; starting on one answers as
		// starting on an offline runner.
		if runner.GetBool("disabled") {
			return mobileDeviceRunnerOfflineError(deviceID)
		}
		selection, err := actionauthor.Select(e.App, "", actionauthor.BuildRequest{
			Actor:                   actor,
			WalletVersionIdentifier: input.WalletVersionIdentifier,
			Platform:                platform,
		})
		if err != nil {
			return authoringAPIError(err)
		}
		if !runner.GetBool("admin_managed") {
			return apierror.New(
				http.StatusForbidden,
				"action_author",
				"runner_not_supported",
				"the action author runs only on admin-managed runners",
			)
		}
		if runner.GetString("owner") != selection.Organization.Id {
			return apierror.New(
				http.StatusForbidden,
				"action_author",
				"runner_not_owned",
				"the device's runner belongs to another organization",
			)
		}
		if apiErr := requireMobileDeviceRunnersOnline(
			e.Request.Context(),
			e.App,
			[]string{deviceID},
		); apiErr != nil {
			return apiErr
		}

		session, err := sessionInputMap(authorsession.SessionInput{
			DeviceID:                deviceID,
			Platform:                platform,
			WalletVersionIdentifier: selection.Identifier,
			ActorID:                 actor.ActorID,
			ActorCollection:         actor.ActorCollection,
			Slot:                    input.Slot,
			AppID:                   input.AppID,
		})
		if err != nil {
			return authoringAPIError(err)
		}
		namespace := selection.Organization.GetString("canonified_name")
		ticketID := uuid.NewString()
		c, err := queueTemporalClient(workflowengine.MobileDeviceSemaphoreDefaultNamespace)
		if err != nil {
			return temporalUnavailableError()
		}
		resp, err := enqueueActionAuthorTicket(
			e.Request.Context(),
			c,
			deviceID,
			mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:            ticketID,
				OwnerNamespace:      namespace,
				EnqueuedAt:          time.Now().UTC(),
				DeviceID:            deviceID,
				RequiredDeviceIDs:   []string{deviceID},
				LeaderDeviceID:      deviceID,
				MaxPipelinesInQueue: selection.Organization.GetInt("max_pipelines_in_queue"),
				RunKind:             mobiledevicesemaphore.RunKindActionAuthor,
				ActionAuthor:        session,
			},
		)
		if err != nil {
			if isQueueLimitExceeded(err) {
				return apierror.New(
					http.StatusConflict,
					"queue_limit",
					"queue limit exceeded",
					err.Error(),
				)
			}
			// A failed enqueue can still have reached the semaphore (timeouts).
			rollbackActionAuthorTicket(e, deviceID, ticketID, namespace)
			return apierror.New(
				http.StatusServiceUnavailable,
				"action_author",
				"start_failed",
				"cannot start authoring session",
			)
		}
		return e.JSON(http.StatusAccepted, ActionAuthorStartResponse{
			TicketID:                ticketID,
			DeviceID:                deviceID,
			Status:                  string(resp.Status),
			Position:                resp.Position,
			WorkflowNamespace:       namespace,
			WalletVersionIdentifier: selection.Identifier,
		})
	}
}

func actionAuthorDeviceNotFound(deviceID string) *apierror.APIError {
	return apierror.New(
		http.StatusNotFound,
		"action_author",
		"device_not_found",
		"mobile device "+deviceID+" not found",
	)
}

// actionAuthorRunnerVisible reports whether the runner belongs to an
// organization the actor owns; superusers see every runner.
func actionAuthorRunnerVisible(
	app core.App,
	actor actionauthor.Actor,
	runner *core.Record,
) (bool, error) {
	ownerFilter, err := actionauthor.OwnerFilter(app, actor)
	if err != nil {
		return false, err
	}
	owned, err := app.FindAllRecords(
		"mobile_runners",
		dbx.HashExp{"id": runner.Id},
		ownerFilter,
	)
	if err != nil {
		return false, fmt.Errorf("check authoring runner owner: %w", err)
	}
	return len(owned) == 1, nil
}

// sessionInputMap encodes the session as the semaphore ticket carries it.
func sessionInputMap(session authorsession.SessionInput) (map[string]any, error) {
	encoded, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("encode authoring session: %w", err)
	}
	var values map[string]any
	if err := json.Unmarshal(encoded, &values); err != nil {
		return nil, fmt.Errorf("decode authoring session: %w", err)
	}
	return values, nil
}

func rollbackActionAuthorTicket(e *core.RequestEvent, deviceID, ticketID, namespace string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := cancelRunTicket(
		ctx,
		deviceID,
		mobiledevicesemaphore.MobileDeviceSemaphoreRunCancelRequest{
			TicketID:       ticketID,
			OwnerNamespace: namespace,
		},
	); err != nil &&
		!errors.Is(err, semaphoreclient.ErrRunTicketNotFound) {
		e.App.Logger().Warn(fmt.Sprintf(
			"failed to rollback action author ticket %s for device %s: %v",
			ticketID,
			deviceID,
			err,
		))
	}
}
