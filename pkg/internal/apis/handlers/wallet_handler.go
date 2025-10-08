// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
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
var WalletPublicRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/wallet",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/get-apk-and-action",
			Handler:       HandleWalletGetApkAndAction,
			RequestSchema: WalletApkRequest{},
		},
		{
			Method:        http.MethodPost,
			Path:          "/store-action-result",
			Handler:       HandleWalletStoreActionResult,
			RequestSchema: WalletStoreResult{},
		},
	},
}

type WalletURL struct {
	URL string `json:"walletURL"`
}

type WalletApkRequest struct {
	WalletIdentifier string `json:"wallet_identifier"`
	ActionIdentifier string `json:"action_identifier"`
	Org              string `json:"organization"`
}

type WalletStoreResult struct {
	ResultPath       string `json:"result_path"`
	ActionIdentifier string `json:"action_identifier"`
	Org              string `json:"organization"`
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
			30*time.Second,
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

func HandleWalletGetApkAndAction() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		app := e.App
		var req WalletApkRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return apis.NewBadRequestError("invalid JSON input", err)
		}

		walletRecord, err := canonify.Resolve(app, req.WalletIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"wallet",
				"wallet not found",
				err.Error(),
			).JSON(e)
		}
		actionRecord, err := canonify.Resolve(app, req.ActionIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"wallet_action",
				"wallet action not found",
				err.Error(),
			).JSON(e)
		}
		code := actionRecord.GetString("code")

		walletVersionColl, err := app.FindCollectionByNameOrId("wallet_versions")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"wallet_versions",
				"failed to find wallet_versions collection",
				err.Error(),
			).JSON(e)
		}
		versionRecord, err := app.FindFirstRecordByFilter(
			walletVersionColl.Id,
			"wallet = {:walletID} && owner = {:org}",
			map[string]any{"walletID": walletRecord.Id, "org": req.Org},
		)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"wallet_version",
				"wallet version not found",
				err.Error(),
			).JSON(e)
		}
		key := versionRecord.BaseFilesPath() + "/" + versionRecord.GetString("android_installer")

		fsys, err := app.NewFilesystem()
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to create pocketbase filesystem",
				err.Error(),
			).JSON(e)
		}
		defer fsys.Close()

		blob, err := fsys.GetFile(key)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to get apk file",
				err.Error(),
			).JSON(e)
		}
		defer blob.Close()

		tmpFile, err := os.CreateTemp("", "*.apk")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to create tmp apk file",
				err.Error(),
			).JSON(e)
		}
		defer tmpFile.Close()

		if _, err := io.Copy(tmpFile, blob); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to copy apk file",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"code":     code,
			"apk_path": tmpFile.Name(),
		})
	}
}

func HandleWalletStoreActionResult() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		app := e.App
		var req WalletStoreResult
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return apis.NewBadRequestError("invalid JSON input", err)
		}

		actionRecord, err := canonify.Resolve(app, req.ActionIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"wallet_action",
				"wallet action not found",
				err.Error(),
			).JSON(e)
		}

		const safeTmpDir = "/tmp/credimi/"
		absResultPath, err := filepath.Abs(req.ResultPath)
		if err != nil || !strings.HasPrefix(absResultPath, safeTmpDir) {
			return apierror.New(
				http.StatusBadRequest,
				"filesystem",
				"invalid result file path",
				"result file path is not allowed",
			).JSON(e)
		}

		f, err := filesystem.NewFileFromPath(absResultPath)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to open tmp file",
				err.Error(),
			).JSON(e)
		}

		actionRecord.Set("result", []*filesystem.File{f})
		if err := app.Save(actionRecord); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"wallet_action",
				"failed to save wallet action with result file",
				err.Error(),
			).JSON(e)
		}

		_ = os.Remove(absResultPath)

		return e.JSON(http.StatusOK, map[string]any{
			"status": "ok",
			"id":     actionRecord.Id,
		})
	}
}
