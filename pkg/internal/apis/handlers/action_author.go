// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/sdk/client"
)

// The Python action-author worker (agents/action-author) serves this workflow
// type on this task queue in the selected organization namespace.
const (
	actionAuthorWorkflowType = "ActionAuthor"
	actionAuthorTaskQueue    = "action-author"
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
			Description:    "List Android wallet versions in organizations where the authenticated caller is an owner. These are the only versions accepted by the session start endpoint.",
			ResponseSchema: ActionAuthorWalletVersionsResponse{},
		},
		{
			Method:         http.MethodPost,
			Path:           "/sessions",
			Handler:        HandleActionAuthorStart,
			OperationID:    "actionAuthor.startSession",
			Summary:        "Start an authenticated wallet-action authoring session",
			Description:    "Authorize the caller for the selected wallet organization and start a local authoring session in its Temporal namespace. Only the onboarding slot is supported. Review and explicit human confirmation happen in the local harness UI; this endpoint does not save an action.",
			ResponseStatus: http.StatusAccepted,
			RequestSchema:  ActionAuthorStartRequest{},
			ResponseSchema: ActionAuthorStartResponse{},
		},
	},
}

// ActionAuthorArtifactRoutes serves APKs authorized by build-bound capabilities.
var ActionAuthorArtifactRoutes = routing.RouteGroup{
	BaseURL:                "/api/action-author",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:              http.MethodGet,
			Path:                "/installer",
			Handler:             HandleActionAuthorInstaller,
			OperationID:         "actionAuthor.downloadInstaller",
			Summary:             "Download the selected authoring APK",
			Description:         "Stream the single Android APK authorized by the required Credimi-Artifact-Grant header. The short-lived capability is issued to the organization-scoped workflow, binds the selected build and actor, and is reauthorized at download. The response is not cached.",
			ResponseSchema:      "",
			ResponseContentType: "application/vnd.android.package-archive",
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
		versions, err := actionauthor.ListBuildOptions(e.App, actionauthor.Actor{
			ActorID:         e.Auth.Id,
			ActorCollection: e.Auth.Collection().Name,
		})
		if err != nil {
			return authoringAPIError(err)
		}
		return e.JSON(http.StatusOK, ActionAuthorWalletVersionsResponse{WalletVersions: versions})
	}
}

// ActionAuthorStartRequest selects the wallet version and action slot for a new session.
type ActionAuthorStartRequest struct {
	WalletVersionIdentifier string `json:"wallet_version_identifier" validate:"required"`
	Slot                    string `json:"slot"                      validate:"required,oneof=onboarding"`
	AppID                   string `json:"app_id,omitempty"`
}

// ActionAuthorStartResponse identifies the workflow in the authorized organization namespace.
type ActionAuthorStartResponse struct {
	WorkflowID              string `json:"workflow_id"`
	RunID                   string `json:"run_id"`
	WorkflowNamespace       string `json:"workflow_namespace"`
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
}

// ActionAuthorInitData binds trusted actor data to the requested slot and optional package ID.
type ActionAuthorInitData struct {
	actionauthor.BuildRequest
	Slot  string `json:"slot"`
	AppID string `json:"app_id,omitempty"`
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

// HandleActionAuthorStart starts an organization-scoped authoring workflow for the caller.
func HandleActionAuthorStart() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[ActionAuthorStartRequest](e)
		if err != nil {
			return authoringAPIError(actionauthor.ErrInvalid)
		}
		if e.Auth == nil {
			return authoringAPIError(actionauthor.ErrForbidden)
		}
		request := actionauthor.BuildRequest{
			WalletVersionIdentifier: input.WalletVersionIdentifier,
			Actor: actionauthor.Actor{
				ActorID:         e.Auth.Id,
				ActorCollection: e.Auth.Collection().Name,
			},
		}
		selection, err := actionauthor.Select(e.App, "", request)
		if err != nil {
			return authoringAPIError(err)
		}
		namespace := selection.Organization.GetString("canonified_name")
		request.WalletVersionIdentifier = selection.Identifier
		temporal, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusServiceUnavailable,
				"action_author",
				"temporal_unavailable",
				"cannot connect to Temporal",
			)
		}
		run, err := temporal.ExecuteWorkflow(
			e.Request.Context(),
			client.StartWorkflowOptions{
				ID:        actionAuthorWorkflowType + "-" + uuid.NewString(),
				TaskQueue: actionAuthorTaskQueue,
			},
			actionAuthorWorkflowType,
			// The Python harness agent takes its config, then the trusted session context.
			map[string]any{},
			ActionAuthorInitData{BuildRequest: request, Slot: input.Slot, AppID: input.AppID},
		)
		if err != nil {
			return apierror.New(
				http.StatusServiceUnavailable,
				"action_author",
				"start_failed",
				"cannot start authoring session",
			)
		}
		return e.JSON(http.StatusAccepted, ActionAuthorStartResponse{
			WorkflowID:              run.GetID(),
			RunID:                   run.GetRunID(),
			WorkflowNamespace:       namespace,
			WalletVersionIdentifier: selection.Identifier,
		})
	}
}

// HandleActionAuthorInstaller streams the single APK authorized by the supplied capability.
func HandleActionAuthorInstaller() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		file, err := actionauthor.OpenInstaller(
			e.App,
			e.Request.Header.Get(actionauthor.GrantHeader),
		)
		if err != nil {
			return authoringAPIError(err)
		}
		defer file.Close()
		e.Response.Header().Set("Cache-Control", "no-store")
		return e.Stream(http.StatusOK, "application/vnd.android.package-archive", file)
	}
}
