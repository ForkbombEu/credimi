// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var WalletRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/wallet",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/start-check",
			Handler:       HandleWalletStartCheck,
			RequestSchema: WalletURL{},
		},
	},
}

type WalletURL struct {
	URL string `json:"walletURL"`
}

func HandleWalletStartCheck() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req WalletURL

		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return apis.NewBadRequestError("invalid JSON input", err)
		}
		organization, err := GetUserOrganizationID(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error(),
			).JSON(e)
		}
		orgName, err := GetOrganizationCanonifiedName(e.App, organization)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get organization",
				err.Error(),
			).JSON(e)
		}
		// Start the workflow
		workflowInput := workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url": e.App.Settings().Meta.AppURL,
			},
			Payload: map[string]any{
				"url": req.URL,
			},
		}
		w := workflows.WalletWorkflow{}

		workflowInfo, err := w.Start(orgName, workflowInput)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start workflow",
				err.Error(),
			).JSON(e)
		}
		client, err := temporalclient.GetTemporalClientWithNamespace(
			orgName,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"failed to get temporal client",
				err.Error(),
			).JSON(e)
		}
		result, err := workflowengine.WaitForPartialResult[map[string]any](
			client,
			workflowInfo.WorkflowID,
			workflowInfo.WorkflowRunID,
			workflows.AppMetadataQuery,
			100*time.Millisecond,
			60*time.Second,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get partial workflow result",
				err.Error(),
			).JSON(e)
		}
		storeType := getStringFromMap(result, "storeType")
		metadata, ok := result["metadata"].(map[string]any)
		if !ok {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get partial workflow result",
				"failed to get metadata",
			).JSON(e)
		}
		var name, logo, appleAppID, googleAppID, playstoreURL, appstoreURL, homeURL string
		description := getStringFromMap(metadata, "description")
		switch storeType {
		case "google":
			name = getStringFromMap(metadata, "title")
			logo = getStringFromMap(metadata, "icon")
			googleAppID = getStringFromMap(metadata, "appId")
			homeURL = getStringFromMap(metadata, "developerWebsite")
			playstoreURL = req.URL

		case "apple":
			name = getStringFromMap(metadata, "trackName")
			logo = getStringFromMap(metadata, "artworkUrl100")
			appleAppID = getStringFromMap(metadata, "bundleId")
			homeURL = getStringFromMap(metadata, "sellerUrl")
			appstoreURL = req.URL
		}

		return e.JSON(http.StatusOK, map[string]any{
			"type":          storeType,
			"name":          name,
			"description":   description,
			"logo":          logo,
			"google_app_id": googleAppID,
			"apple_app_id":  appleAppID,
			"playstore_url": playstoreURL,
			"appstore_url":  appstoreURL,
			"home_url":      homeURL,
			"owner":         organization,
		})
	}
}
