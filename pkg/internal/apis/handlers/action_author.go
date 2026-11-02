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
			Description:    "List the Android devices and iOS simulators of enabled admin-managed runners owned by organizations where the authenticated caller is an owner (superusers: every admin-managed runner), with the authoring platform each device implies. agent_available reports whether the runner's action author polled its task queue in the last two minutes (false when Temporal is unreachable); only devices with is_online and agent_available can start a session. Sorted by runner name, then device name.",
			ResponseSchema: ActionAuthorDevicesResponse{},
		},
		{
			Method:         http.MethodGet,
			Path:           "/wallet-actions",
			Handler:        HandleActionAuthorWalletActions,
			OperationID:    "actionAuthor.listWalletActions",
			Summary:        "List wallet actions usable as session prerequisites",
			Description:    "List the actions of the wallet of wallet_version_identifier that its organization owns or that are published, for a caller authorized to author on that version and platform (android or ios; both query parameters are required). Each action carries its canonified path, which the session start endpoint takes as a prerequisite, and the sorted parameters its code references, which the prerequisite must receive. Sorted by category, then name.",
			ResponseSchema: ActionAuthorWalletActionsResponse{},
			QuerySearchAttributes: []routing.QuerySearchAttribute{
				{
					Name:        "wallet_version_identifier",
					Required:    true,
					Description: "Wallet version the session authors on, organization/wallet/version.",
				},
				{
					Name:        "platform",
					Required:    true,
					Description: "Authoring platform of the session device: android or ios.",
				},
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "/sessions",
			Handler:        HandleActionAuthorStart,
			OperationID:    "actionAuthor.startSession",
			Summary:        "Queue a wallet-action authoring session on a runner device",
			Description:    "Authorize the caller for the selected wallet organization and queue an authoring session on the selected device through the mobile-device semaphore. The device type decides the platform (Android devices: android, iOS simulators: ios), and the version must have an installer for it. The device must belong to an enabled, online admin-managed runner of the wallet's organization; a device of an organization the caller does not own answers 404 device_not_found, and a disabled runner answers like an offline one (503). A runner whose action author has not polled <runner_id>-ActionAuthor in the last two minutes answers 503 agent_unavailable. The slot selects the action category: onboarding takes no prerequisites; get-credential, get-credential-specific and verify-credential need at least one; other takes optional prerequisites. outcome is success (default) or rejection (not for onboarding), and browser flows (not for onboarding) need an Android device. Prerequisites (at most 5) are wallet actions of the selected wallet, published or owned by its organization, listed by GET /api/action-author/wallet-actions; each must receive exactly the parameters its code references. Parameters (at most 8 per list) take fresh values from a credential_offer or use_case_verification (published or owned records), an https GET or POST without credential headers (http), or a fixed value (literal); only the new action's http parameters may check a protocol status. An invalid spec answers 400 invalid_request. Poll GET /api/pipeline/queue/{ticket_id}?device_ids=<device_id>: once running, its workflow_id is the session and <workflow_id>-author is the chat workflow. This endpoint does not save an action: the session's user accepts or refuses each candidate through POST /api/action-author/reviews.",
			ResponseStatus: http.StatusAccepted,
			RequestSchema:  ActionAuthorStartRequest{},
			ResponseSchema: ActionAuthorStartResponse{},
		},
		{
			Method:         http.MethodPost,
			Path:           "/reviews",
			Handler:        HandleActionAuthorReview,
			OperationID:    "actionAuthor.reviewCandidate",
			Summary:        "Accept or refuse a candidate action of an authoring session",
			Description:    "Deliver the human decision on the pending save review tool_id of the agent workflow author_workflow_id in the organization namespace. Only the user who started the session can review it (403 forbidden otherwise); a namespace the caller does not own, or a workflow that is not the agent of a running session, answers 404 session_not_found. An acceptance carries a Credimi-signed approval of candidate_sha256 that the save activity verifies (valid for 30 minutes); the agent cannot save without it. A review that is no longer pending (unknown or settled tool_id) answers 409 review_not_pending.",
			RequestSchema:  ActionAuthorReviewRequest{},
			ResponseSchema: ActionAuthorReviewResponse{},
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
			return authoringAPIError(e, actionauthor.ErrForbidden)
		}
		versions, err := actionauthor.ListBuildOptions(e.App, authoringActor(e.Auth))
		if err != nil {
			return authoringAPIError(e, err)
		}
		return e.JSON(http.StatusOK, ActionAuthorWalletVersionsResponse{WalletVersions: versions})
	}
}

// ActionAuthorWalletAction is one action a session can run as a prerequisite.
type ActionAuthorWalletAction struct {
	Path       string   `json:"path"`
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Published  bool     `json:"published"`
	Parameters []string `json:"parameters"`
}

// ActionAuthorWalletActionsResponse lists the prerequisite candidates of a wallet.
type ActionAuthorWalletActionsResponse struct {
	Actions []ActionAuthorWalletAction `json:"actions"`
}

// HandleActionAuthorWalletActions lists the wallet's owned and published
// actions for a caller who can author on the selected version.
func HandleActionAuthorWalletActions() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return authoringAPIError(e, actionauthor.ErrForbidden)
		}
		query := e.Request.URL.Query()
		identifier := query.Get("wallet_version_identifier")
		platform := query.Get("platform")
		if identifier == "" ||
			(platform != actionauthor.PlatformAndroid && platform != actionauthor.PlatformIOS) {
			return authoringAPIError(e, fmt.Errorf(
				"%w: wallet_version_identifier and platform (android or ios) are required",
				actionauthor.ErrInvalid,
			))
		}
		selection, err := actionauthor.Select(e.App, "", actionauthor.BuildRequest{
			Actor:                   authoringActor(e.Auth),
			WalletVersionIdentifier: identifier,
			Platform:                platform,
		})
		if err != nil {
			return authoringAPIError(e, err)
		}
		records, err := e.App.FindRecordsByFilter(
			"wallet_actions",
			"wallet = {:wallet} && (owner = {:org} || published = true)",
			"category,name",
			0,
			0,
			dbx.Params{"wallet": selection.Wallet.Id, "org": selection.Organization.Id},
		)
		if err != nil {
			return authoringAPIError(e, fmt.Errorf("list wallet actions: %w", err))
		}
		actions := make([]ActionAuthorWalletAction, 0, len(records))
		for _, record := range records {
			path, err := canonify.BuildPath(
				e.App,
				record,
				canonify.CanonifyPaths["wallet_actions"],
				record.GetString("canonified_name"),
			)
			if err != nil {
				return authoringAPIError(e, fmt.Errorf("build wallet action path: %w", err))
			}
			actions = append(actions, ActionAuthorWalletAction{
				Path:       path,
				Name:       record.GetString("name"),
				Category:   record.GetString("category"),
				Published:  record.GetBool("published"),
				Parameters: actionauthor.ActionParameters(record.GetString("code")),
			})
		}
		return e.JSON(http.StatusOK, ActionAuthorWalletActionsResponse{Actions: actions})
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
	// AgentAvailable reports whether the runner's action author polled its
	// task queue in the last two minutes; select only online devices with it.
	AgentAvailable bool `json:"agent_available"`
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
			return authoringAPIError(e, actionauthor.ErrForbidden)
		}
		ownerFilter, err := actionauthor.OwnerFilter(e.App, authoringActor(e.Auth))
		if err != nil {
			return authoringAPIError(e, err)
		}
		runners, err := e.App.FindAllRecords(
			"mobile_runners",
			dbx.HashExp{"admin_managed": true},
			ownerFilter,
		)
		if err != nil {
			return authoringAPIError(e, fmt.Errorf("list authoring runners: %w", err))
		}
		runners = enabledMobileRunnerRecords(runners)
		namespaces := make([]string, len(runners))
		for i, runner := range runners {
			owner, err := e.App.FindRecordById("organizations", runner.GetString("owner"))
			if err != nil {
				return authoringAPIError(e, fmt.Errorf("find runner organization: %w", err))
			}
			namespaces[i] = owner.GetString("canonified_name")
		}
		agents := runnersActionAuthorAvailable(e, runners, namespaces)
		response := ActionAuthorDevicesResponse{Devices: []ActionAuthorDevice{}}
		now := time.Now()
		for i, runner := range runners {
			// Heartbeat freshness, as GET /api/mobile-devices reports it.
			runnerOnline := mobilerunner.RecentlyAlive(runner, now) &&
				mobilerunner.URLUsable(mobilerunner.RunnerURL(runner))
			namespace := namespaces[i]
			agentAvailable := agents[runner.Id]
			devices, err := e.App.FindAllRecords(
				"mobile_devices",
				dbx.HashExp{"runner": runner.Id},
			)
			if err != nil {
				return authoringAPIError(e, fmt.Errorf("list authoring devices: %w", err))
			}
			for _, device := range devices {
				platform := actionAuthorPlatform(device.GetString("type"))
				if platform == "" {
					continue
				}
				path, err := mobileDeviceIdentifier(e.App, device)
				if err != nil {
					return authoringAPIError(e, fmt.Errorf("build device path: %w", err))
				}
				response.Devices = append(response.Devices, ActionAuthorDevice{
					Path:           path,
					Namespace:      namespace,
					Name:           device.GetString("name"),
					RunnerName:     runner.GetString("name"),
					Type:           device.GetString("type"),
					Platform:       platform,
					IsOnline:       runnerOnline && device.GetBool("online"),
					AgentAvailable: agentAvailable,
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

// ActionAuthorStartRequest selects the device, wallet version and action spec for a new session.
type ActionAuthorStartRequest struct {
	DeviceID                string `json:"device_id"                 validate:"required"`
	WalletVersionIdentifier string `json:"wallet_version_identifier" validate:"required"`
	AppID                   string `json:"app_id,omitempty"`
	actionauthor.Spec
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

// authoringAPIError maps authoring errors to API errors; it logs the cause of
// internal errors, which the response hides.
func authoringAPIError(e *core.RequestEvent, err error) *apierror.APIError {
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
		e.App.Logger().Error("action author operation failed", "error", err)
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
			return authoringAPIError(e, actionauthor.ErrInvalid)
		}
		if e.Auth == nil {
			return authoringAPIError(e, actionauthor.ErrForbidden)
		}
		deviceID := canonify.NormalizePath(input.DeviceID)
		actor := authoringActor(e.Auth)
		device, runner, err := mobilerunner.ResolveDevice(e.App, deviceID)
		switch {
		case errors.Is(err, mobilerunner.ErrDeviceNotFound),
			errors.Is(err, mobilerunner.ErrDeviceRunnerNotFound):
			return actionAuthorDeviceNotFound(deviceID)
		case err != nil:
			return authoringAPIError(e, err)
		}
		// Devices of organizations the caller does not own answer as unknown
		// devices, before their type or runner is revealed.
		visible, err := actionAuthorRunnerVisible(e.App, actor, runner)
		if err != nil {
			return authoringAPIError(e, err)
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
			return authoringAPIError(e, err)
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
		namespace := selection.Organization.GetString("canonified_name")
		if apiErr := requireActionAuthorAgent(e, runner, namespace); apiErr != nil {
			return apiErr
		}
		spec := input.Spec
		if err := spec.Normalize(platform); err != nil {
			return authoringAPIError(e, err)
		}
		if _, err := actionauthor.CheckSpecRecords(e.App, selection, &spec); err != nil {
			return authoringAPIError(e, err)
		}

		session, err := sessionInputMap(authorsession.SessionInput{
			DeviceID:                deviceID,
			Platform:                platform,
			WalletVersionIdentifier: selection.Identifier,
			ActorID:                 actor.ActorID,
			ActorCollection:         actor.ActorCollection,
			Spec:                    spec,
			AppID:                   input.AppID,
		})
		if err != nil {
			return authoringAPIError(e, err)
		}
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
