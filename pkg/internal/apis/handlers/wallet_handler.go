// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
var WalletTemporalInternalRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/wallet",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodPost,
			Path:           "/get-apk-md5",
			Handler:        HandleWalletGetMD5,
			RequestSchema:  WalletMD5Request{},
			ResponseSchema: WalletMD5Response{},
		},
		{
			Method:  http.MethodPost,
			Path:    "/store-action-result",
			Handler: HandleWalletStoreActionResult,
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				apis.BodyLimit(500 << 20),
			},
		},
	},
}

type WalletURL struct {
	URL string `json:"walletURL"`
}

type WalletApkRequest struct {
	WalletIdentifier string `json:"wallet_identifier"`
	ActionIdentifier string `json:"action_identifier"`
}

type WalletStoreResult struct {
	ResultPath       string `json:"result_path"`
	ActionIdentifier string `json:"action_identifier"`
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

type WalletMD5Request struct {
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
	WalletIdentifier        string `json:"wallet_identifier"`
}

type WalletMD5Response struct {
	AndroidInstaller string `json:"apk_name"`
	MD5              string `json:"md5"`
	RecordID         string `json:"record_id"`
}

func HandleWalletGetMD5() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req WalletMD5Request
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return apis.NewBadRequestError("invalid JSON input", err)
		}

		// Validate that at least one identifier is provided
		if req.WalletVersionIdentifier == "" && req.WalletIdentifier == "" {
			return apierror.New(
				http.StatusBadRequest,
				"identifier",
				"no identifier provided",
				"at least one identifier must be provided",
			).JSON(e)
		}

		record, err := getWalletVersionRecord(e.App, req.WalletVersionIdentifier, req.WalletIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"wallet_version",
				"wallet version not found",
				err.Error(),
			).JSON(e)
		}

		// Get android_installer field value
		androidInstaller := record.GetString("android_installer")
		if androidInstaller == "" {
			return apierror.New(
				http.StatusNotFound,
				"android_installer",
				"no android_installer file found for this wallet version",
				"android_installer field is empty",
			).JSON(e)
		}

		// Get MD5 from PocketBase's file metadata
		md5Hash, err := getFileMD5FromPocketBase(e.App, record, androidInstaller)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to get file MD5",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, WalletMD5Response{
			AndroidInstaller: androidInstaller,
			RecordID:         record.Id,
			MD5:              md5Hash,
		})
	}
}

// getWalletVersionRecord retrieves a wallet_version record based on provided identifiers
func getWalletVersionRecord(app core.App, versionIdentifier, walletIdentifier string) (*core.Record, error) {
	if versionIdentifier != "" {
		record, err := canonify.Resolve(app, versionIdentifier)
		if err != nil {
			return nil, err
		}
		return record, nil
	}
	walletRecord, err := canonify.Resolve(app, walletIdentifier)
	if err != nil {
		return nil, err
	}

	walletVersionColl, err := app.FindCollectionByNameOrId("wallet_versions")
	if err != nil {
		return nil, err
	}

	records, err := app.FindRecordsByFilter(
		walletVersionColl.Id,
		"wallet = {:walletID}",
		"-created",
		1,
		0,
		map[string]any{"walletID": walletRecord.Id},
	)
	if err != nil || len(records) == 0 {
		return nil, err
	}

	return records[0], nil
}

// getFileMD5FromPocketBase retrieves the MD5 hash from PocketBase's file metadata
func getFileMD5FromPocketBase(app core.App, record *core.Record, filename string) (string, error) {
	fsys, err := app.NewFilesystem()
	if err != nil {
		return "", err
	}
	defer fsys.Close()

	filePath := record.BaseFilesPath() + "/" + filename
	attrs, err := fsys.Attributes(filePath)
	if err != nil {
		return "", err
	}
	if attrs.MD5 != nil {
		return hex.EncodeToString(attrs.MD5), nil
	}
	return "", err
}

func HandleWalletStoreActionResult() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {

		if err := e.Request.ParseMultipartForm(500 << 20); err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"multipart",
				"failed to parse multipart form",
				err.Error(),
			).JSON(e)
		}
		var actionRecord *core.Record
		var err error
		ActionIdentifier := e.Request.FormValue("action_identifier")
		if ActionIdentifier == "" {
			collection, err := e.App.FindCollectionByNameOrId("wallet_actions")
			if err != nil {
				return apierror.New(
					http.StatusNotFound,
					"wallet_action",
					"collection not found",
					err.Error(),
				).JSON(e)
			}
			actionRecord = core.NewRecord(collection)
			walletRecord, err := canonify.Resolve(e.App, e.Request.FormValue("wallet_identifier"))
			if err != nil {
				return apierror.New(
					http.StatusNotFound,
					"wallet",
					"wallet not found",
					err.Error(),
				).JSON(e)
			}
			actionRecord.Set("name", "pipeline_result")
			actionRecord.Set("wallet", walletRecord.Id)
			actionRecord.Set("owner", walletRecord.GetString("owner"))
			actionRecord.Set("code", e.Request.FormValue("action_code"))

			if err := e.App.Save(actionRecord); err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"database",
					"failed to save record",
					err.Error(),
				).JSON(e)
			}
		} else {
			actionRecord, err = canonify.Resolve(e.App, ActionIdentifier)
			if err != nil {
				return apierror.New(
					http.StatusNotFound,
					"wallet_action",
					"record not found",
					err.Error(),
				).JSON(e)
			}
		}
		action := actionRecord.GetString("canonified_name")
		file, header, err := e.Request.FormFile("result")
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"file",
				"failed to read file",
				err.Error(),
			).JSON(e)
		}
		defer file.Close()
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("result_%s_*%s", action, filepath.Ext(header.Filename)))
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to create temp file",
				err.Error(),
			).JSON(e)
		}
		defer tmpFile.Close()

		if _, err := io.Copy(tmpFile, file); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to write temp file",
				err.Error(),
			).JSON(e)
		}

		absResultPath, err := filepath.Abs(tmpFile.Name())
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"invalid temp file path",
				err.Error(),
			).JSON(e)
		}

		f, err := filesystem.NewFileFromPath(absResultPath)
		if err != nil {
			_ = os.Remove(absResultPath)
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to wrap file",
				err.Error(),
			).JSON(e)
		}

		actionRecord.Set("result", []*filesystem.File{f})
		if err := e.App.Save(actionRecord); err != nil {
			_ = os.Remove(absResultPath)
			return apierror.New(
				http.StatusInternalServerError,
				"wallet_action",
				"failed to save wallet action with result file",
				err.Error(),
			).JSON(e)
		}
		resultURL := fmt.Sprintf(
			"%s/api/files/wallet_actions/%s/%s",
			e.App.Settings().Meta.AppURL,
			actionRecord.Id,
			actionRecord.GetString("result"),
		)
		err = os.Remove(absResultPath)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to remove temp file",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"status":     "success",
			"action":     action,
			"fileName":   header.Filename,
			"result_url": resultURL,
		})
	}
}
