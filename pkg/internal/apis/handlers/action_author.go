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
	"regexp"
	"slices"
	"sort"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/authorsession"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/forkbombeu/credimi/pkg/workflowengine/semaphoreclient"
	"github.com/google/uuid"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
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
			Description:    "List the builds available for authoring to the authenticated caller, member of an organization (the user's first organization, as for pipeline runs): builds of the caller's organization's wallets and of other organizations' published wallets (their Play Store app, and their versions whose installer is downloadable), with the platforms each build supports and its source. A build is a wallet version with an Android APK or iOS simulator app (source installer, identifier organization/wallet/version), or, for Android only, a wallet with a Play Store app (google_app_id, or the id of its play.google.com playstore_url), installed from the Play Store at session start (source store, identifier organization/wallet, version_tag \"store install\", platforms [android]). The iOS simulator has no App Store, so iOS builds are only wallet versions with an iOS simulator app. These are the only builds accepted by the session start endpoint. A caller without an organization gets an empty list.",
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
			Method:         http.MethodPost,
			Path:           "/sessions",
			Handler:        HandleActionAuthorStart,
			OperationID:    "actionAuthor.startSession",
			Summary:        "Queue a wallet-action authoring session on a runner device",
			Description:    "Queue an authoring session for a member of an organization (the user's first organization, as for pipeline runs) on the selected device through the mobile-device semaphore. The session runs in the caller's organization namespace. The device type decides the platform (Android devices: android, iOS simulators: ios). wallet_version_identifier is organization/wallet/version for an uploaded installer of that platform of a wallet of the caller's organization or a published wallet of another organization (downloadable versions only), or, on an Android device only, organization/wallet for its wallet installed from the Play Store at session start (the device needs the Play Store with a signed-in account; otherwise the agent reports the failure in the chat and closes the session); GET /api/action-author/wallet-versions lists both. Saved actions are owned by the caller's organization. The iOS simulator has no App Store: organization/wallet on an iOS simulator answers 400 invalid_request before anything is queued. An unpublished wallet of another organization, or a non-downloadable version of one, answers 403 forbidden. The device must belong to an enabled, online runner of the caller's organization, or to a published runner of another organization that offers the agent to other organizations; any other device answers 404 device_not_found, and a disabled runner answers like an offline one (503). A caller who is not a member of any organization answers 403 forbidden. A runner whose action author has not polled <runner_id>-ActionAuthor in the caller's namespace in the last two minutes answers 503 agent_unavailable. The agent sets each action's plan in the chat and Credimi checks it on every call that uses it; the session can save several actions. The runner owner sets two daily limits of model tokens (UTC day) in the runner's configuration: one for sessions of the runner's own organization (0: no limit) and one shared by every other organization (0: the agent is not offered to them). A start after the day's tokens of the caller's pool reach its limit answers 429 quota_exceeded. Poll GET /api/pipeline/queue/{ticket_id}?device_ids=<device_id>: once running, its workflow_id is the session and <workflow_id>-author is the chat workflow. This endpoint does not save an action: the session's user accepts or refuses each candidate through POST /api/action-author/reviews.",
			ResponseStatus: http.StatusAccepted,
			RequestSchema:  ActionAuthorStartRequest{},
			ResponseSchema: ActionAuthorStartResponse{},
		},
		{
			Method:         http.MethodGet,
			Path:           "/sessions",
			Handler:        HandleActionAuthorSessions,
			OperationID:    "actionAuthor.listSessions",
			Summary:        "List a wallet's recent authoring sessions",
			Description:    "List the ten latest authoring sessions the caller started in their organization namespace on the wallet organization/wallet (its Play Store build and every version), newest first, with their Temporal status (for example Running or Completed), start time (RFC 3339), build, device path and device name (empty when the device no longer exists). A session Temporal cannot describe is left out.",
			ResponseSchema: ActionAuthorSessionsResponse{},
			QuerySearchAttributes: []routing.QuerySearchAttribute{
				{
					Name:        "wallet",
					Required:    true,
					Description: "Wallet of the sessions: organization/wallet (canonified names).",
				},
			},
		},
		{
			Method:         http.MethodGet,
			Path:           "/sessions/{session_id}",
			Handler:        HandleActionAuthorSession,
			OperationID:    "actionAuthor.getSession",
			Summary:        "Read an authoring session",
			Description:    "Read the authoring session session_id (ActionAuthorSession-<runner record id>-<ticket uuid>, the session_id of the start response) that the caller started in their organization namespace, running or closed: its run, agent chat workflow (chat_workflow_id), Temporal status, times, build, device, runner and model. live_view is true while a session runs on an Android device that streams its screen (POST /api/pipeline/live-view opens it with the session ID and run). For a running session, chat_ready reports whether the agent chat has started its first turn, closes_at (RFC 3339, empty while a turn runs or when unknown) when the idle session closes to free the device, and pending_review the candidate action waiting for the caller's review (POST /api/action-author/reviews), null otherwise. A session that has not started yet, an unknown session and one another user started all answer 404 session_not_found (follow a queued session's ticket in the device queue).",
			ResponseSchema: ActionAuthorSessionResponse{},
		},
		{
			Method:         http.MethodPost,
			Path:           "/sessions/{session_id}/close",
			Handler:        HandleActionAuthorSessionClose,
			OperationID:    "actionAuthor.closeSession",
			Summary:        "End an authoring session",
			Description:    "End the running authoring session session_id that the caller started: Credimi asks the agent chat to close, or cancels the session while its device is still being prepared. The session then cleans and releases its device. A session that has already ended answers 409 session_not_running; an unknown session, or one another user started, answers 404 session_not_found as GET /api/action-author/sessions/{session_id} does.",
			ResponseStatus: http.StatusAccepted,
			ResponseSchema: ActionAuthorSessionCloseResponse{},
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

// ActionAuthorDevice is one runner device the caller can author on.
type ActionAuthorDevice struct {
	Path       string `json:"path"`
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
	// QueueLength is the number of tickets in the device's queue, reported
	// only for online devices whose queue could be read.
	QueueLength *int `json:"queue_length,omitempty"`
}

// runnerAgentModel describes the model a runner reports for its agent.
func runnerAgentModel(runner *core.Record) string {
	provider, model := runner.GetString("agent_provider"), runner.GetString("agent_model")
	if model == "" {
		return provider
	}
	return provider + " / " + model
}

// actionAuthorRunnerDetails returns the name of the organization that owns
// runner and the model its agent reports.
func actionAuthorRunnerDetails(app core.App, runner *core.Record) (owner, model string, err error) {
	organization, err := app.FindRecordById("organizations", runner.GetString("owner"))
	if err != nil {
		return "", "", fmt.Errorf("find runner organization: %w", err)
	}
	return organization.GetString("name"), runnerAgentModel(runner), nil
}

// ActionAuthorDevicesResponse lists the devices accepted by the session start endpoint.
type ActionAuthorDevicesResponse struct {
	Devices []ActionAuthorDevice `json:"devices"`
}

// HandleActionAuthorDevices lists the supported devices of the caller's
// organization's own enabled runners (admin-managed or not) and of runners of
// other organizations that offer the agent to other organizations
// (agent_others_daily_tokens above 0) and are shared with the caller's
// organization as for pipelines (mobilerunner.SharedWith). Every session runs
// in its runner's agent namespace (mobilerunner.AgentNamespace).
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
		agents := runnersActionAuthorAvailable(e, runners)
		now := time.Now()
		for _, runner := range runners {
			owner, model, err := actionAuthorRunnerDetails(e.App, runner)
			if err != nil {
				return authoringAPIError(e, err)
			}
			// Heartbeat freshness, as GET /api/mobile-devices reports it.
			runnerOnline := mobilerunner.RecentlyAlive(runner, now) &&
				mobilerunner.URLUsable(mobilerunner.RunnerURL(runner))
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
				entry := ActionAuthorDevice{
					Path:           path,
					Name:           device.GetString("name"),
					RunnerName:     runner.GetString("name"),
					RunnerOwner:    owner,
					Model:          model,
					Type:           device.GetString("type"),
					Platform:       platform,
					IsOnline:       runnerOnline && device.GetBool("online"),
					AgentAvailable: agentAvailable,
				}
				if entry.IsOnline {
					// One unreadable queue never fails the list.
					queueLength, apiErr := mobileDeviceQueueLen(e.Request.Context(), path)
					if apiErr != nil {
						e.App.Logger().
							Warn("action author device queue", "device", path, "error", apiErr)
					} else {
						entry.QueueLength = &queueLength
					}
				}
				response.Devices = append(response.Devices, entry)
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
}

// ActionAuthorStartResponse identifies the queued session ticket.
type ActionAuthorStartResponse struct {
	// SessionID is the session workflow the ticket starts once granted,
	// ActionAuthorSession-<runner record id>-<ticket_id>.
	SessionID               string `json:"session_id"`
	TicketID                string `json:"ticket_id"`
	DeviceID                string `json:"device_id"`
	Status                  string `json:"status"`
	Position                int    `json:"position"`
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

// HandleActionAuthorStart queues a session, in the runner's agent namespace
// for the caller's organization, on a device of a runner that offers the agent to the caller's
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
		orgNamespace := org.GetString("canonified_name")
		runnerNamespace := mobilerunner.AgentNamespace(runner.Id)
		selection, err := actionauthor.Select(e.App, orgNamespace, actionauthor.BuildRequest{
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
		if apiErr := requireActionAuthorAgent(e, runner); apiErr != nil {
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
			Namespace:               runnerNamespace,
			Organization:            orgNamespace,
		}
		sessionInput.TicketGrant, err = actionauthor.SignTicket(e.App, actionauthor.SessionGrant{
			SessionWorkflowID:       actionauthor.SessionWorkflowID(runner.Id, ticketID),
			Namespace:               sessionInput.Namespace,
			Organization:            sessionInput.Organization,
			DeviceID:                sessionInput.DeviceID,
			Platform:                sessionInput.Platform,
			WalletVersionIdentifier: sessionInput.WalletVersionIdentifier,
			ActorID:                 sessionInput.ActorID,
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
		// The runner meters its model's usage and reports it in heartbeats;
		// a pool it reports used up today refuses new sessions.
		if allowance := actionauthor.AllowanceFor(runner, org.Id); allowance.Exhausted(
			runner,
			time.Now(),
		) {
			return actionAuthorQuotaExceeded(allowance)
		}
		resp, err := enqueueActionAuthorTicket(
			e.Request.Context(),
			c,
			deviceID,
			mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:            ticketID,
				OwnerNamespace:      orgNamespace,
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
			if isQueueLimitExceeded(err) {
				return apierror.New(
					http.StatusConflict,
					"queue_limit",
					"queue limit exceeded",
					err.Error(),
				)
			}
			// A failed enqueue can still have reached the semaphore (timeouts).
			rollbackActionAuthorTicket(e, deviceID, ticketID, orgNamespace)
			return apierror.New(
				http.StatusServiceUnavailable,
				"action_author",
				"start_failed",
				"cannot start authoring session",
			)
		}
		return e.JSON(http.StatusAccepted, ActionAuthorStartResponse{
			SessionID:               actionauthor.SessionWorkflowID(runner.Id, ticketID),
			TicketID:                ticketID,
			DeviceID:                deviceID,
			Status:                  string(resp.Status),
			Position:                resp.Position,
			WalletVersionIdentifier: selection.Identifier,
		})
	}
}

// actionAuthorQuotaExceeded refuses a session whose pool the runner reported
// used up today.
func actionAuthorQuotaExceeded(allowance actionauthor.Allowance) *apierror.APIError {
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

// Agent chat workflow queries and signal of the Temporal Agent Harness that
// the session page reads and sends.
const (
	actionAuthorIdleStateQuery   = "idle_state"
	actionAuthorAgentStatusQuery = "agent_status"
	actionAuthorCloseSignal      = "close"
	actionAuthorSaveReviewTool   = "confirm_wallet_action_save"
	actionAuthorRecentSessions   = 10
)

// actionAuthorSessionTemporalClient connects to a session's namespace and is
// stubbed in unit tests.
var actionAuthorSessionTemporalClient = temporalclient.GetTemporalClientWithNamespace

var actionAuthorWalletPattern = regexp.MustCompile(`^[^/]+/[^/]+$`)

// ActionAuthorSessionResponse is a session Credimi started, running or closed.
type ActionAuthorSessionResponse struct {
	SessionID      string `json:"session_id"`
	RunID          string `json:"run_id"`
	ChatWorkflowID string `json:"chat_workflow_id"`
	// Status is the Temporal status of the session, for example Running.
	Status string `json:"status"`
	// StartTime and CloseTime are RFC 3339; CloseTime is empty while the session runs.
	StartTime               string `json:"start_time"`
	CloseTime               string `json:"close_time,omitempty"`
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
	Platform                string `json:"platform"`
	DeviceID                string `json:"device_id"`
	DeviceName              string `json:"device_name"`
	RunnerName              string `json:"runner_name"`
	RunnerOwner             string `json:"runner_owner"`
	Model                   string `json:"model"`
	// LiveView reports a running session on an Android device that streams its screen.
	LiveView bool `json:"live_view"`
	// ChatReady reports that the agent chat has started its first turn; it
	// stays true once the chat or the session ended.
	ChatReady bool `json:"chat_ready"`
	// ChatRunning reports that the agent chat still runs.
	ChatRunning bool `json:"chat_running"`
	// ClosesAt is when the idle session closes, empty when unknown or while a turn runs.
	ClosesAt      string                     `json:"closes_at"`
	PendingReview *ActionAuthorPendingReview `json:"pending_review"`
}

// ActionAuthorPendingReview is the candidate action waiting for the caller's review.
type ActionAuthorPendingReview struct {
	ToolID          string `json:"tool_id"`
	Name            string `json:"name"`
	Build           string `json:"build"`
	Code            string `json:"code"`
	CandidateSHA256 string `json:"candidate_sha256"`
	// Plan is nil when the candidate has no object plan.
	Plan map[string]any `json:"plan"`
	// Replaces is the path of the action the candidate replaces, empty when none.
	Replaces string `json:"replaces"`
	// Error is why the previous save of this candidate failed, empty when none.
	Error string `json:"error"`
	// UsedBy counts the pipelines of the caller's organization or published
	// that use the replaced action, 0 without a replacement.
	UsedBy int `json:"used_by"`
}

// ActionAuthorSessionsResponse lists a wallet's recent sessions.
type ActionAuthorSessionsResponse struct {
	Sessions []ActionAuthorSessionListItem `json:"sessions"`
}

// ActionAuthorSessionListItem is one recent session of a wallet.
type ActionAuthorSessionListItem struct {
	SessionID               string `json:"session_id"`
	Status                  string `json:"status"`
	StartTime               string `json:"start_time"`
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
	DeviceID                string `json:"device_id"`
	DeviceName              string `json:"device_name"`
}

// ActionAuthorSessionCloseResponse names the session asked to end.
type ActionAuthorSessionCloseResponse struct {
	SessionID string `json:"session_id"`
}

func actionAuthorSessionMissing() *apierror.APIError {
	return apierror.New(
		http.StatusNotFound,
		"action_author",
		"session_not_found",
		"This authoring session does not exist.",
	)
}

// actionAuthorCallerNamespace is the organization canonified_name of the
// authenticated organization member.
func actionAuthorCallerNamespace(e *core.RequestEvent) (string, *apierror.APIError) {
	if e.Auth == nil {
		return "", authoringAPIError(e, actionauthor.ErrForbidden)
	}
	if apiErr := requireAuthoringUser(e.Auth); apiErr != nil {
		return "", apiErr
	}
	namespace, err := pbutils.GetUserOrganizationCanonifiedName(e.App, e.Auth.Id)
	if err != nil {
		return "", apierror.New(
			http.StatusForbidden,
			"action_author",
			"forbidden",
			"you are not a member of any organization",
		)
	}
	return namespace, nil
}

// readCallerActionAuthorSession reads the session session_id of the path
// that the caller started for their organization, and returns a client of
// the namespace the session runs in.
func readCallerActionAuthorSession(
	e *core.RequestEvent,
) (client.Client, activities.ActionAuthorSessionInfo, *apierror.APIError) {
	var info activities.ActionAuthorSessionInfo
	organization, apiErr := actionAuthorCallerNamespace(e)
	if apiErr != nil {
		return nil, info, apiErr
	}
	sessionID := e.Request.PathValue("session_id")
	if !actionauthor.IsSessionID(sessionID) {
		return nil, info, actionAuthorSessionMissing()
	}
	var c client.Client
	var clientErr error
	clientFor := func(namespace string) (client.Client, error) {
		c, clientErr = actionAuthorSessionTemporalClient(namespace)
		return c, clientErr
	}
	info, err := activities.ReadActionAuthorSession(
		e.Request.Context(),
		e.App,
		clientFor,
		organization,
		sessionID,
		e.Auth.Id,
	)
	switch {
	case errors.Is(err, actionauthor.ErrSessionNotVisible),
		errors.Is(err, actionauthor.ErrForbidden):
		return nil, info, actionAuthorSessionMissing()
	case clientErr != nil:
		return nil, info, temporalUnavailableError()
	case err != nil:
		return nil, info, authoringAPIError(e, err)
	}
	return c, info, nil
}

// HandleActionAuthorSession reads a session the caller started and,
// while it runs, the state of its agent chat.
func HandleActionAuthorSession() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		c, info, apiErr := readCallerActionAuthorSession(e)
		if apiErr != nil {
			return apiErr
		}
		running := info.Status == enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING
		response := ActionAuthorSessionResponse{
			SessionID:               info.SessionWorkflowID,
			RunID:                   info.SessionRunID,
			ChatWorkflowID:          info.SessionWorkflowID + activities.ActionAuthorWorkflowIDSuffix,
			Status:                  info.Status.String(),
			StartTime:               info.StartTime.UTC().Format(time.RFC3339),
			WalletVersionIdentifier: info.Session.WalletVersionIdentifier,
			Platform:                info.Session.Platform,
			DeviceID:                info.DeviceID,
		}
		if !info.CloseTime.IsZero() {
			response.CloseTime = info.CloseTime.UTC().Format(time.RFC3339)
		}
		device, runner, err := mobilerunner.ResolveDevice(e.App, info.DeviceID)
		if err == nil {
			owner, model, err := actionAuthorRunnerDetails(e.App, runner)
			if err != nil {
				return authoringAPIError(e, err)
			}
			response.DeviceName = device.GetString("name")
			response.RunnerName = runner.GetString("name")
			response.RunnerOwner = owner
			response.Model = model
			response.LiveView = running &&
				info.Session.Platform == actionauthor.PlatformAndroid &&
				device.GetBool("live_stream")
		}
		readActionAuthorChat(e, c, response.ChatWorkflowID, running, &response)
		return e.JSON(http.StatusOK, response)
	}
}

// readActionAuthorChat fills the chat state of a session in any state. The
// idle state and the pending review are read only while the session and its
// started chat run. A missing chat, or a query that fails, leaves them empty.
func readActionAuthorChat(
	e *core.RequestEvent,
	c client.Client,
	chatID string,
	sessionRunning bool,
	response *ActionAuthorSessionResponse,
) {
	ctx := e.Request.Context()
	described, err := c.DescribeWorkflowExecution(ctx, chatID, "")
	if err != nil {
		var notFound *serviceerror.NotFound
		if !errors.As(err, &notFound) {
			e.App.Logger().Warn("action author chat describe", "chat", chatID, "error", err)
		}
		return
	}
	chat := described.GetWorkflowExecutionInfo()
	// The chat runs its first workflow task once the agent's worker picked it up.
	response.ChatReady = chat.GetHistoryLength() > 2
	response.ChatRunning = chat.GetStatus() == enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING
	if !response.ChatReady || !response.ChatRunning || !sessionRunning {
		return
	}
	var idle struct {
		ClosesAt string `json:"closes_at"`
	}
	if err := queryActionAuthorChat(ctx, c, chatID, actionAuthorIdleStateQuery, &idle); err != nil {
		e.App.Logger().Warn("action author chat idle state", "chat", chatID, "error", err)
	} else {
		response.ClosesAt = idle.ClosesAt
	}
	var status struct {
		PendingCallbacks []struct {
			ToolID    string         `json:"tool_id"`
			ToolName  string         `json:"tool_name"`
			ToolInput map[string]any `json:"tool_input"`
		} `json:"pending_callbacks"`
	}
	if err := queryActionAuthorChat(
		ctx,
		c,
		chatID,
		actionAuthorAgentStatusQuery,
		&status,
	); err != nil {
		e.App.Logger().Warn("action author chat status", "chat", chatID, "error", err)
		return
	}
	for _, callback := range status.PendingCallbacks {
		if callback.ToolName != actionAuthorSaveReviewTool {
			continue
		}
		text := func(key string) string {
			value, _ := callback.ToolInput[key].(string)
			return value
		}
		plan, _ := callback.ToolInput["plan"].(map[string]any)
		response.PendingReview = &ActionAuthorPendingReview{
			ToolID:          callback.ToolID,
			Name:            text("name"),
			Build:           text("build"),
			Code:            text("code"),
			CandidateSHA256: text("candidate_sha256"),
			Plan:            plan,
			Replaces:        text("replaces"),
			Error:           text("error"),
		}
		if response.PendingReview.Replaces != "" {
			response.PendingReview.UsedBy = actionAuthorUsedBy(e, response.PendingReview.Replaces)
		}
		return
	}
}

// actionAuthorUsedBy counts the pipelines owned by the caller's organization
// or published whose YAML names action; 0, with a warning, on error.
func actionAuthorUsedBy(e *core.RequestEvent, action string) int {
	orgID, err := pbutils.GetUserOrganizationID(e.App, e.Auth.Id)
	if err != nil {
		e.App.Logger().Warn("action author used by", "action", action, "error", err)
		return 0
	}
	count, err := e.App.CountRecords(
		"pipelines",
		dbx.And(
			dbx.Or(dbx.HashExp{"owner": orgID}, dbx.HashExp{"published": true}),
			dbx.Like("yaml", action),
		),
	)
	if err != nil {
		e.App.Logger().Warn("action author used by", "action", action, "error", err)
		return 0
	}
	return int(count)
}

func queryActionAuthorChat(
	ctx context.Context,
	c client.Client,
	chatID, query string,
	value any,
) error {
	encoded, err := c.QueryWorkflow(ctx, chatID, "", query)
	if err != nil {
		return err
	}
	return encoded.Get(value)
}

// HandleActionAuthorSessions lists the latest sessions of a wallet that the
// caller started for their organization: Temporal lists them by search
// attributes in the ready agent namespaces of the runners that offer the
// caller's organization the agent, and each listed session is kept only when
// the listed ID has a trusted run of the caller's. Search attributes are
// written by whoever starts a workflow, so the newest listed IDs are read
// until actionAuthorRecentSessions are trusted: forged runs never hide them.
func HandleActionAuthorSessions() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		organization, apiErr := actionAuthorCallerNamespace(e)
		if apiErr != nil {
			return apiErr
		}
		wallet := e.Request.URL.Query().Get("wallet")
		if !actionAuthorWalletPattern.MatchString(wallet) {
			return authoringAPIError(e, fmt.Errorf(
				"%w: wallet must be organization/wallet",
				actionauthor.ErrInvalid,
			))
		}
		org, err := pbutils.GetUserOrganization(e.App, e.Auth.Id)
		if err != nil {
			return authoringAPIError(e, actionauthor.ErrForbidden)
		}
		listed, err := listMobileRunnerRecords(e.App, org.Id, org.GetBool("published"))
		if err != nil {
			return authoringAPIError(e, fmt.Errorf("list authoring runners: %w", err))
		}
		ctx := e.Request.Context()
		query := fmt.Sprintf(
			`WorkflowType = "%s" and %s = "%s" and %s = "%s"`,
			escapeTemporalQueryValue(activities.ActionAuthorSessionWorkflowName),
			workflowengine.ActionAuthorActorSearchAttribute,
			escapeTemporalQueryValue(e.Auth.Id),
			workflowengine.ActionAuthorWalletSearchAttribute,
			escapeTemporalQueryValue(wallet),
		)
		var executions []*workflowpb.WorkflowExecutionInfo
		namespaces, failed := 0, 0
		for _, runner := range enabledMobileRunnerRecords(listed) {
			if !actionAuthorRunnerOffered(runner, org.Id) ||
				!mobilerunner.AgentNamespaceReady(runner) {
				continue
			}
			namespace := mobilerunner.AgentNamespace(runner.Id)
			namespaces++
			listed, err := listRunnerActionAuthorSessions(e, namespace, query)
			if err != nil {
				failed++
				e.App.Logger().Warn(
					"action author sessions namespace skipped",
					"namespace", namespace,
					"error", err,
				)
				continue
			}
			executions = append(executions, listed...)
		}
		if failed > 0 && failed == namespaces {
			return temporalUnavailableError()
		}
		response := ActionAuthorSessionsResponse{Sessions: []ActionAuthorSessionListItem{}}
		for _, execution := range newestSessionRuns(executions) {
			if len(response.Sessions) == actionAuthorRecentSessions {
				break
			}
			sessionID := execution.GetExecution().GetWorkflowId()
			// Only the trusted run of a session the caller started counts.
			info, err := activities.ReadActionAuthorSession(
				ctx,
				e.App,
				actionAuthorSessionTemporalClient,
				organization,
				sessionID,
				e.Auth.Id,
			)
			if err != nil {
				e.App.Logger().Warn(
					"action author session skipped",
					"session", sessionID,
					"error", err,
				)
				continue
			}
			item := ActionAuthorSessionListItem{
				SessionID:               sessionID,
				Status:                  info.Status.String(),
				StartTime:               info.StartTime.UTC().Format(time.RFC3339),
				WalletVersionIdentifier: info.Session.WalletVersionIdentifier,
				DeviceID:                info.DeviceID,
			}
			if device, _, err := mobilerunner.ResolveDevice(e.App, info.DeviceID); err == nil {
				item.DeviceName = device.GetString("name")
			}
			response.Sessions = append(response.Sessions, item)
		}
		return e.JSON(http.StatusOK, response)
	}
}

// newestSessionRuns keeps the newest listed run of each session ID, newest
// first.
func newestSessionRuns(
	executions []*workflowpb.WorkflowExecutionInfo,
) []*workflowpb.WorkflowExecutionInfo {
	slices.SortStableFunc(executions, func(a, b *workflowpb.WorkflowExecutionInfo) int {
		return b.GetStartTime().AsTime().Compare(a.GetStartTime().AsTime())
	})
	seen := make(map[string]bool, len(executions))
	return slices.DeleteFunc(executions, func(execution *workflowpb.WorkflowExecutionInfo) bool {
		id := execution.GetExecution().GetWorkflowId()
		if seen[id] {
			return true
		}
		seen[id] = true
		return false
	})
}

// listRunnerActionAuthorSessions returns the first page of the sessions query
// lists in a runner's agent namespace. A namespace without a client, not
// registered or without the search attributes yet is skipped; other list
// failures are returned.
func listRunnerActionAuthorSessions(
	e *core.RequestEvent,
	namespace, query string,
) ([]*workflowpb.WorkflowExecutionInfo, error) {
	c, err := actionAuthorSessionTemporalClient(namespace)
	if err != nil {
		e.App.Logger().Warn(
			"action author sessions namespace skipped",
			"namespace", namespace,
			"error", err,
		)
		return nil, nil
	}
	resp, err := c.ListWorkflow(e.Request.Context(), &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: namespace,
		PageSize:  actionAuthorSessionsPageSize,
		Query:     query,
	})
	var notFound *serviceerror.NotFound
	var namespaceNotFound *serviceerror.NamespaceNotFound
	var invalid *serviceerror.InvalidArgument
	switch {
	case errors.As(err, &notFound),
		errors.As(err, &namespaceNotFound),
		errors.As(err, &invalid):
		e.App.Logger().Warn(
			"action author sessions namespace skipped",
			"namespace", namespace,
			"error", err,
		)
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("list action author sessions in %s: %w", namespace, err)
	}
	return resp.GetExecutions(), nil
}

// actionAuthorSessionsPageSize is how many sessions one runner namespace
// lists: forged runs listed under the caller's search attributes must not
// hide the caller's actionAuthorRecentSessions.
const actionAuthorSessionsPageSize = 50

// HandleActionAuthorSessionClose asks the agent chat of a running session to
// close, or cancels a session whose chat has not started; the session then
// cleans and releases its device. A chat that has already ended leaves the
// session releasing its device on its own.
func HandleActionAuthorSessionClose() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		c, info, apiErr := readCallerActionAuthorSession(e)
		if apiErr != nil {
			return apiErr
		}
		if info.Status != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
			return apierror.New(
				http.StatusConflict,
				"action_author",
				"session_not_running",
				"The session has already ended.",
			)
		}
		ctx := e.Request.Context()
		chatID := info.SessionWorkflowID + activities.ActionAuthorWorkflowIDSuffix
		err := signalChatClose(ctx, c, info.Namespace, chatID)
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			// Signals reach only running workflows: tell a chat that never
			// started (the device is still being prepared) from one that ended,
			// or one that started since the signal.
			var described *workflowservice.DescribeWorkflowExecutionResponse
			described, err = c.DescribeWorkflowExecution(ctx, chatID, "")
			switch {
			case errors.As(err, &notFound):
				err = c.CancelWorkflow(ctx, info.SessionWorkflowID, info.SessionRunID)
			case err == nil && described.GetWorkflowExecutionInfo().GetStatus() ==
				enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:
				err = signalChatClose(ctx, c, info.Namespace, chatID)
				if errors.As(err, &notFound) {
					// It ended in between.
					err = nil
				}
			}
		}
		if err != nil {
			e.App.Logger().Error(
				"action author session close",
				"session", info.SessionWorkflowID,
				"error", err,
			)
			return temporalUnavailableError()
		}
		return e.JSON(
			http.StatusAccepted,
			ActionAuthorSessionCloseResponse{SessionID: info.SessionWorkflowID},
		)
	}
}

// signalChatClose sends the harness close signal without input: the agent's
// handler takes no argument, and client.SignalWorkflow always sends one.
var signalChatClose = func(ctx context.Context, c client.Client, namespace, chatID string) error {
	_, err := c.WorkflowService().SignalWorkflowExecution(ctx, chatCloseRequest(namespace, chatID))
	return err
}

func chatCloseRequest(namespace, chatID string) *workflowservice.SignalWorkflowExecutionRequest {
	return &workflowservice.SignalWorkflowExecutionRequest{
		Namespace:         namespace,
		WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: chatID},
		SignalName:        actionAuthorCloseSignal,
		Identity:          "credimi",
		RequestId:         uuid.NewString(),
	}
}
