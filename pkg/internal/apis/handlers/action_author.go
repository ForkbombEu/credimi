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
			Description:    "Authorize the caller for the selected wallet organization and queue an authoring session on the selected device through the mobile-device semaphore. The device type decides the platform (Android devices: android, iOS simulators: ios), and the version must have an installer for it. The device must belong to an enabled, online admin-managed runner of the wallet's organization; a device of an organization the caller does not own answers 404 device_not_found, and a disabled runner answers like an offline one (503). A runner whose action author has not polled <runner_id>-ActionAuthor in the last two minutes answers 503 agent_unavailable. The spec fields (slot, outcome, wait_seconds, prerequisites, parameters) are optional: without them the agent proposes the plan in the chat and the user confirms it through POST /api/action-author/plans, once per action, and the session can save several actions; with them the plan is fixed for the whole session, which closes after its save. The slot selects the action category: onboarding takes no prerequisites; get-credential, get-credential-specific and verify-credential need at least one; other takes optional prerequisites. outcome is success (default), rejection (not for onboarding) or wait (only for other): a wait action keeps the device idle for wait_seconds (1-600, only for wait) and takes no parameters: its code ends with extendedWaitUntil on a visible condition, with optional: true and a plain integer timeout of wait_seconds * 1000. Prerequisites (at most 5) are wallet actions of the selected wallet, published or owned by its organization, listed by GET /api/action-author/wallet-actions; each must receive exactly the parameters its code references. Parameters (at most 8 per list) take fresh values from a credential_offer or use_case_verification (published or owned records), an https GET or POST whose only headers are Content-Type, Accept and Accept-Language (http), or a fixed value (literal); only the new action's http parameters may check a protocol status. An invalid spec answers 400 invalid_request; a spec without a slot is invalid. Each organization has a daily allowance of model tokens (UTC day), set by a superuser on the organization (action_author_daily_tokens); new organizations have none. A start without an allowance or after the day's tokens reach it answers 429 quota_exceeded. Poll GET /api/pipeline/queue/{ticket_id}?device_ids=<device_id>: once running, its workflow_id is the session and <workflow_id>-author is the chat workflow. This endpoint does not save an action: the session's user accepts or refuses each candidate through POST /api/action-author/reviews.",
			ResponseStatus: http.StatusAccepted,
			RequestSchema:  ActionAuthorStartRequest{},
			ResponseSchema: ActionAuthorStartResponse{},
		},
		{
			Method:         http.MethodPost,
			Path:           "/plans",
			Handler:        HandleActionAuthorPlan,
			OperationID:    "actionAuthor.confirmPlan",
			Summary:        "Confirm or refuse the plan an authoring session's agent proposed",
			Description:    "Deliver the human decision on the pending plan card tool_id of the agent workflow author_workflow_id in the organization namespace. Only the user who started the session can confirm its plans (403 forbidden otherwise); a namespace the caller does not own, or a workflow that is not the agent of a running session, answers 404 session_not_found. A session started with a spec has a fixed plan and answers 409 plan_fixed. An acceptance validates plan as the session start endpoint validates a spec (400 invalid_request, and nothing is delivered), then delivers to the agent the normalized plan, its resolved prerequisites, its parameter descriptions and a Credimi-signed plan_token, which the agent sends with every activity of that plan; the payload segment of the token is the base64url JSON grant whose plan field is the confirmed plan. A plan card that is no longer pending (unknown or settled tool_id) answers 409 plan_not_pending.",
			RequestSchema:  ActionAuthorPlanRequest{},
			ResponseSchema: ActionAuthorPlanResponse{},
		},
		{
			Method:         http.MethodPost,
			Path:           "/reviews",
			Handler:        HandleActionAuthorReview,
			OperationID:    "actionAuthor.reviewCandidate",
			Summary:        "Accept or refuse a candidate action of an authoring session",
			Description:    "Deliver the human decision on the pending save review tool_id of the agent workflow author_workflow_id in the organization namespace. Only the user who started the session can review it (403 forbidden otherwise); a namespace the caller does not own, or a workflow that is not the agent of a running session, answers 404 session_not_found. plan_token is the session's confirmed plan token (omit it for a session started with a spec); for an acceptance, an invalid plan answers 400 invalid_request and a plan token that Credimi did not sign for this session and user answers 403 forbidden with the reason (plan not signed by Credimi, plan for another session or plan confirmed by another user); a refusal never reads the plan. An acceptance carries a Credimi-signed approval of candidate_sha256 and of the plan's hash (plan_sha256) that the save activity verifies (valid for 30 minutes); the agent cannot save without it, nor with another plan. A review that is no longer pending (unknown or settled tool_id) answers 409 review_not_pending.",
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

// ActionAuthorWalletActionsResponse lists the prerequisite candidates of a wallet.
type ActionAuthorWalletActionsResponse struct {
	Actions []actionauthor.WalletAction `json:"actions"`
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
		actions, err := actionauthor.ListWalletActions(e.App, selection, "")
		if err != nil {
			return authoringAPIError(e, err)
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
		// Without a spec the agent proposes the plan in the chat.
		spec := input.Spec
		if !spec.IsEmpty() {
			if err := spec.Normalize(); err != nil {
				return authoringAPIError(e, err)
			}
			if _, err := actionauthor.CheckSpecRecords(e.App, selection, &spec); err != nil {
				return authoringAPIError(e, err)
			}
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
		// Sessions count at enqueue; a refused or failed enqueue gives it back.
		orgID := selection.Organization.Id
		reservedAt := time.Now()
		if apiErr := reserveActionAuthorSession(
			e,
			selection.Organization,
			reservedAt,
		); apiErr != nil {
			return apiErr
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
			releaseActionAuthorSession(e, orgID, reservedAt)
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

// reserveActionAuthorSession counts a session against the organization's
// daily quota, answering 429 quota_exceeded when the organization has no
// allowance or used it up.
func reserveActionAuthorSession(
	e *core.RequestEvent,
	org *core.Record,
	now time.Time,
) *apierror.APIError {
	quota := actionauthor.QuotaForOrganization(org)
	err := actionauthor.ReserveSession(e.App, org.Id, now, quota)
	if errors.Is(err, actionauthor.ErrQuotaExceeded) {
		message := fmt.Sprintf(
			"your organization used today's action author allowance (%d tokens); it resets at 00:00 UTC",
			quota.Tokens,
		)
		if !quota.HasAllowance() {
			message = "your organization has no action author allowance; ask a Credimi administrator to set one"
		}
		return apierror.New(
			http.StatusTooManyRequests,
			"action_author",
			"quota_exceeded",
			message,
		)
	}
	if err != nil {
		return authoringAPIError(e, err)
	}
	return nil
}

// releaseActionAuthorSession gives back a session whose enqueue failed.
func releaseActionAuthorSession(e *core.RequestEvent, orgID string, reservedAt time.Time) {
	if err := actionauthor.ReleaseSession(e.App, orgID, reservedAt); err != nil {
		e.App.Logger().Warn("failed to release action author session quota", "error", err)
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
