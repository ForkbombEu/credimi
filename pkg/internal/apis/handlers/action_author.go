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
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
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
			Description:    "List the builds of the authenticated caller's organization (the user's first organization, as for pipeline runs) available for authoring, with the platforms each build supports and its source. A build is a wallet version with an Android APK or iOS simulator app (source installer, identifier organization/wallet/version), or a wallet with a Play Store app (google_app_id, or the id of its play.google.com playstore_url) or an App Store app (apple_app_id and an apps.apple.com appstore_url), installed from the store at session start (source store, identifier organization/wallet, version_tag \"store install\"). These are the only builds accepted by the session start endpoint. A caller without an organization gets an empty list.",
			ResponseSchema: ActionAuthorWalletVersionsResponse{},
		},
		{
			Method:         http.MethodGet,
			Path:           "/devices",
			Handler:        HandleActionAuthorDevices,
			OperationID:    "actionAuthor.listDevices",
			Summary:        "List devices available for authoring",
			Description:    "List the Android devices and iOS simulators the caller's organization can author on: devices of its own enabled runners, and of enabled runners of other organizations that offer the agent to other organizations (`agent_others_daily_tokens` above 0) and are shared with the caller's organization as for pipelines (`mobilerunner.SharedWith`: published, and admin-managed unless the caller's organization is published), with the authoring platform each device implies. namespace is the caller's organization namespace, where every session runs. agent_available reports whether the runner's action author polled its task queue in that namespace in the last two minutes (false when Temporal is unreachable); only devices with is_online and agent_available can start a session. Sorted by runner name, then device name. A caller without an organization gets an empty list.",
			ResponseSchema: ActionAuthorDevicesResponse{},
		},
		{
			Method:         http.MethodGet,
			Path:           "/wallet-actions",
			Handler:        HandleActionAuthorWalletActions,
			OperationID:    "actionAuthor.listWalletActions",
			Summary:        "List wallet actions usable as session prerequisites",
			Description:    "List the actions of the wallet of wallet_version_identifier that its organization owns or that are published, for a caller authorized to author on that version and platform (android or ios; both query parameters are required). Each action carries its canonified path, which a session's plan takes as a prerequisite, and the sorted parameters its code references, which the prerequisite must receive. Sorted by category, then name.",
			ResponseSchema: ActionAuthorWalletActionsResponse{},
			QuerySearchAttributes: []routing.QuerySearchAttribute{
				{
					Name:        "wallet_version_identifier",
					Required:    true,
					Description: "Build the session authors on: organization/wallet/version for an uploaded installer, or organization/wallet for a wallet installed from the Play Store or App Store at session start.",
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
			Description:    "Queue an authoring session for a member of an organization (the user's first organization, as for pipeline runs) on the selected device through the mobile-device semaphore. The session runs in the caller's organization namespace. The device type decides the platform (Android devices: android, iOS simulators: ios). wallet_version_identifier is organization/wallet/version for an uploaded installer of that platform of the caller's organization, or organization/wallet for its wallet installed from the Play Store or App Store at session start (the device needs the store with a signed-in account; otherwise the agent reports the failure in the chat and closes the session); GET /api/action-author/wallet-versions lists both; a wallet of another organization answers 403 forbidden. The device must belong to an enabled, online runner of the caller's organization, or to a published runner of another organization that offers the agent to other organizations; any other device answers 404 device_not_found, and a disabled runner answers like an offline one (503). A caller who is not a member of any organization answers 403 forbidden. A runner whose action author has not polled <runner_id>-ActionAuthor in the caller's namespace in the last two minutes answers 503 agent_unavailable. The agent sets each action's plan in the chat and Credimi checks it on every call that uses it; the session can save several actions. The runner owner sets two daily limits of model tokens (UTC day) in the runner's configuration: one for sessions of the runner's own organization (0: no limit) and one shared by every other organization (0: the agent is not offered to them). A start after the day's tokens of the caller's pool reach its limit answers 429 quota_exceeded. Poll GET /api/pipeline/queue/{ticket_id}?device_ids=<device_id>: once running, its workflow_id is the session and <workflow_id>-author is the chat workflow. This endpoint does not save an action: the session's user accepts or refuses each candidate through POST /api/action-author/reviews.",
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
			Description:    "Deliver the human decision on the pending save review tool_id of the agent workflow author_workflow_id in the organization namespace. Only the user who started the session can review it (403 forbidden otherwise); a namespace of an organization the caller is not a member of, or a workflow that is not the agent of a running session, answers 404 session_not_found. plan is the plan the save card shows; Credimi checks it again and binds the approval to it: for an acceptance, a missing or invalid plan answers 400 invalid_request; a refusal never reads the plan. An acceptance carries a Credimi-signed approval of candidate_sha256 and of the plan's hash (plan_sha256) that the save activity verifies (valid for 30 minutes); the agent cannot save without it, nor with another plan. A review that is no longer pending (unknown or settled tool_id) answers 409 review_not_pending.",
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
	// RunnerOwner is the name of the organization that owns the runner.
	RunnerOwner string `json:"runner_owner"`
	// Model is the runner's model as "<provider> / <model>", the provider
	// alone without a model, or empty when the runner reports neither.
	Model    string `json:"model"`
	Type     string `json:"type"`
	Platform string `json:"platform"`
	IsOnline bool   `json:"is_online"`
	// AgentAvailable reports whether the runner's action author polled its
	// task queue in the last two minutes; select only online devices with it.
	AgentAvailable bool `json:"agent_available"`
}

// runnerAgentModel describes the model a runner reports for its agent.
func runnerAgentModel(runner *core.Record) string {
	provider, model := runner.GetString("agent_provider"), runner.GetString("agent_model")
	if model == "" {
		return provider
	}
	return provider + " / " + model
}

// ActionAuthorDevicesResponse lists the devices accepted by the session start endpoint.
type ActionAuthorDevicesResponse struct {
	Devices []ActionAuthorDevice `json:"devices"`
}

// HandleActionAuthorDevices lists the supported devices of the caller's
// organization's runners and of other organizations' published runners that
// offer the agent. Every session runs in the caller's organization namespace.
func HandleActionAuthorDevices() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return authoringAPIError(e, actionauthor.ErrForbidden)
		}
		if apiErr := requireAuthoringUser(e.Auth); apiErr != nil {
			return apiErr
		}
		response := ActionAuthorDevicesResponse{Devices: []ActionAuthorDevice{}}
		org, err := pbutils.GetUserOrganization(e.App, e.Auth.Id)
		if err != nil {
			return e.JSON(http.StatusOK, response)
		}
		listed, err := listMobileRunnerRecords(e.App, org.Id, org.GetBool("published"))
		if err != nil {
			return authoringAPIError(e, fmt.Errorf("list authoring runners: %w", err))
		}
		runners := make([]*core.Record, 0, len(listed))
		for _, runner := range enabledMobileRunnerRecords(listed) {
			if actionAuthorRunnerOffered(runner, org.Id) {
				runners = append(runners, runner)
			}
		}
		namespace := org.GetString("canonified_name")
		agents := runnersActionAuthorAvailable(e, runners, namespace)
		now := time.Now()
		for _, runner := range runners {
			owner, err := e.App.FindRecordById("organizations", runner.GetString("owner"))
			if err != nil {
				return authoringAPIError(e, fmt.Errorf("find runner organization: %w", err))
			}
			// Heartbeat freshness, as GET /api/mobile-devices reports it.
			runnerOnline := mobilerunner.RecentlyAlive(runner, now) &&
				mobilerunner.URLUsable(mobilerunner.RunnerURL(runner))
			agentAvailable := agents[runner.Id]
			model := runnerAgentModel(runner)
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
					RunnerOwner:    owner.GetString("name"),
					Model:          model,
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

// ActionAuthorStartRequest selects the device and wallet version for a new session.
type ActionAuthorStartRequest struct {
	DeviceID                string `json:"device_id"                 validate:"required"`
	WalletVersionIdentifier string `json:"wallet_version_identifier" validate:"required"`
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

// authoringActor is the actor of an auth record that requireAuthoringUser accepted.
func authoringActor(auth *core.Record) actionauthor.Actor {
	return actionauthor.Actor{ActorID: auth.Id}
}

// requireAuthoringUser refuses auth records outside the users collection:
// sessions belong to organization members.
func requireAuthoringUser(auth *core.Record) *apierror.APIError {
	if auth.Collection().Name != "users" {
		return apierror.New(
			http.StatusForbidden,
			"action_author",
			"forbidden",
			"the action author is for organization members",
		)
	}
	return nil
}

// actionAuthorRunnerOffered reports whether a runner offers the agent to the
// organization: always to its own, to others when it has an allowance for them.
func actionAuthorRunnerOffered(runner *core.Record, orgID string) bool {
	return actionauthor.AllowanceFor(runner, orgID).Offered()
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

// HandleActionAuthorStart queues a session, in the caller's organization
// namespace, on a device of a runner that offers the agent to the caller's
// organization.
func HandleActionAuthorStart() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[ActionAuthorStartRequest](e)
		if err != nil {
			return authoringAPIError(e, actionauthor.ErrInvalid)
		}
		if e.Auth == nil {
			return authoringAPIError(e, actionauthor.ErrForbidden)
		}
		if apiErr := requireAuthoringUser(e.Auth); apiErr != nil {
			return apiErr
		}
		actor := authoringActor(e.Auth)
		org, err := pbutils.GetUserOrganization(e.App, actor.ActorID)
		if err != nil {
			return apierror.New(
				http.StatusForbidden,
				"action_author",
				"forbidden",
				"you are not a member of any organization",
			)
		}
		deviceID := canonify.NormalizePath(input.DeviceID)
		device, runner, err := mobilerunner.ResolveDevice(e.App, deviceID)
		switch {
		case errors.Is(err, mobilerunner.ErrDeviceNotFound),
			errors.Is(err, mobilerunner.ErrDeviceRunnerNotFound):
			return actionAuthorDeviceNotFound(deviceID)
		case err != nil:
			return authoringAPIError(e, err)
		}
		// Devices the organization cannot use, or whose runner does not offer
		// it the agent, answer as unknown devices before their type is revealed.
		err = mobilerunner.ValidateDeviceAccess(e.App, org.Id, []string{deviceID})
		switch {
		case errors.Is(err, mobilerunner.ErrDeviceNotAccessible):
			return actionAuthorDeviceNotFound(deviceID)
		case err != nil:
			return authoringAPIError(e, err)
		}
		if !actionAuthorRunnerOffered(runner, org.Id) {
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
		namespace := org.GetString("canonified_name")
		selection, err := actionauthor.Select(e.App, namespace, actionauthor.BuildRequest{
			Actor:                   actor,
			WalletVersionIdentifier: input.WalletVersionIdentifier,
			Platform:                platform,
		})
		if err != nil {
			return authoringAPIError(e, err)
		}
		if apiErr := requireMobileDeviceRunnersOnline(
			e.Request.Context(),
			e.App,
			[]string{deviceID},
		); apiErr != nil {
			return apiErr
		}
		if apiErr := requireActionAuthorAgent(e, runner, namespace); apiErr != nil {
			return apiErr
		}
		ticketID := uuid.NewString()
		// The ticket carries Credimi's signature of the enqueued session; the
		// start activity checks it and signs the session itself
		// (SessionInput.Grant), so no session grant reaches the semaphore's
		// history.
		sessionInput := authorsession.SessionInput{
			DeviceID:                deviceID,
			Platform:                platform,
			WalletVersionIdentifier: selection.Identifier,
			ActorID:                 actor.ActorID,
			AppID:                   input.AppID,
		}
		sessionInput.TicketGrant, err = actionauthor.SignTicket(e.App, actionauthor.SessionGrant{
			SessionWorkflowID:       activities.ActionAuthorSessionWorkflowIDPrefix + ticketID,
			Namespace:               namespace,
			DeviceID:                sessionInput.DeviceID,
			Platform:                sessionInput.Platform,
			WalletVersionIdentifier: sessionInput.WalletVersionIdentifier,
			ActorID:                 sessionInput.ActorID,
			AppID:                   sessionInput.AppID,
		})
		if err != nil {
			return authoringAPIError(e, err)
		}
		session, err := sessionInputMap(sessionInput)
		if err != nil {
			return authoringAPIError(e, err)
		}
		c, err := queueTemporalClient(workflowengine.MobileDeviceSemaphoreDefaultNamespace)
		if err != nil {
			return temporalUnavailableError()
		}
		// Sessions count at enqueue; a refused or failed enqueue gives it back.
		allowance := actionauthor.AllowanceFor(runner, org.Id)
		reservedAt := time.Now()
		if apiErr := reserveActionAuthorSession(e, allowance, reservedAt); apiErr != nil {
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
				MaxPipelinesInQueue: org.GetInt("max_pipelines_in_queue"),
				WaitTimeoutSeconds:  semaphoreclient.WaitTimeoutSeconds(),
				RunKind:             mobiledevicesemaphore.RunKindActionAuthor,
				ActionAuthor:        session,
			},
		)
		if err != nil {
			releaseActionAuthorSession(e, allowance, reservedAt)
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

// reserveActionAuthorSession counts a session against the runner's daily
// allowance for the caller's pool, answering 429 quota_exceeded when it is used up.
func reserveActionAuthorSession(
	e *core.RequestEvent,
	allowance actionauthor.Allowance,
	now time.Time,
) *apierror.APIError {
	err := actionauthor.ReserveSession(e.App, allowance, now)
	if errors.Is(err, actionauthor.ErrQuotaExceeded) {
		whose := "its organization"
		if allowance.Pool == actionauthor.PoolOthers {
			whose = "other organizations"
		}
		return apierror.New(
			http.StatusTooManyRequests,
			"action_author",
			"quota_exceeded",
			fmt.Sprintf(
				"this runner's daily action author allowance for %s is used (%d tokens); it resets at 00:00 UTC",
				whose,
				allowance.Tokens,
			),
		)
	}
	if err != nil {
		return authoringAPIError(e, err)
	}
	return nil
}

// releaseActionAuthorSession gives back a session whose enqueue failed.
func releaseActionAuthorSession(
	e *core.RequestEvent,
	allowance actionauthor.Allowance,
	reservedAt time.Time,
) {
	if err := actionauthor.ReleaseSession(e.App, allowance, reservedAt); err != nil {
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
