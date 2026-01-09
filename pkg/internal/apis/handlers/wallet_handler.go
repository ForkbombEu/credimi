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
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
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
			Path:           "/get-apk-md5-or-etag",
			Handler:        HandleWalletGetMD5,
			RequestSchema:  WalletMD5OrETagRequest{},
			ResponseSchema: WalletMD5OrETagResponse{},
		},
		{
			Method:  http.MethodPost,
			Path:    "/store-pipeline-result",
			Handler: HandleWalletStorePipelineResult,
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
			Payload: workflows.WalletWorkflowPayload{
				URL: req.URL,
			},
		}
		w := workflows.NewWalletWorkflow()
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

type WalletMD5OrETagRequest struct {
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
	WalletIdentifier        string `json:"wallet_identifier"`
}

type WalletMD5OrETagResponse struct {
	AndroidInstaller  string `json:"apk_name"`
	ApkIdentifier     string `json:"apk_identifier"`
	RecordID          string `json:"record_id"`
	VersionIdentifier string `json:"version_id"`
}

func HandleWalletGetMD5() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req WalletMD5OrETagRequest
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

		versionRecord, err := getVersionRecord(
			e.App,
			req.WalletVersionIdentifier,
			req.WalletIdentifier,
		)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"wallet_version",
				"wallet version not found",
				err.Error(),
			).JSON(e)
		}

		// Get android_installer field value
		androidInstaller := versionRecord.GetString("android_installer")
		if androidInstaller == "" {
			return apierror.New(
				http.StatusNotFound,
				"android_installer",
				"no android_installer file found for this wallet version",
				"android_installer field is empty",
			).JSON(e)
		}

		// Get MD5 or ETag from PocketBase's file metadata
		identifier, err := getFileMD5OrETagFromPocketBase(e.App, versionRecord, androidInstaller)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to get file MD5 or ETag",
				err.Error(),
			).JSON(e)
		}
		versionIdentifier := req.WalletVersionIdentifier
		if versionIdentifier == "" {
			versionIdentifier = fmt.Sprintf(
				"%s:%s",
				req.WalletIdentifier,
				versionRecord.GetString("canonified_tag"),
			)
		}

		return e.JSON(http.StatusOK, WalletMD5OrETagResponse{
			AndroidInstaller:  androidInstaller,
			RecordID:          versionRecord.Id,
			ApkIdentifier:     identifier,
			VersionIdentifier: versionIdentifier,
		})
	}
}

// getWalletAndVersionRecord retrieves a wallet_version record based on provided identifiers
func getVersionRecord(
	app core.App,
	versionIdentifier, walletIdentifier string,
) (*core.Record, error) {
	if versionIdentifier != "" {
		versionRecord, err := canonify.Resolve(app, versionIdentifier)
		if err != nil {
			return nil, err
		}
		return versionRecord, nil
	}
	walletRecord, err := canonify.Resolve(app, walletIdentifier)
	if err != nil {
		return nil, err
	}

	walletVersionColl, err := app.FindCollectionByNameOrId("wallet_versions")
	if err != nil {
		return nil, err
	}

	versionRecords, err := app.FindRecordsByFilter(
		walletVersionColl.Id,
		"wallet = {:walletID}",
		"-created",
		1,
		0,
		map[string]any{"walletID": walletRecord.Id},
	)
	if err != nil || len(versionRecords) == 0 {
		return nil, err
	}

	return versionRecords[0], nil
}

// getFileMD5orEtagFromPocketBase retrieves the MD5 hash or ETag from PocketBase's file metadata
func getFileMD5OrETagFromPocketBase(
	app core.App,
	record *core.Record,
	filename string,
) (string, error) {
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
	if attrs == nil {
		return "", fmt.Errorf("missing file attributes for %s", filePath)
	}
	if attrs.ETag != "" {
		return strings.Trim(attrs.ETag, `"`), nil
	}
	if attrs.MD5 != nil {
		return hex.EncodeToString(attrs.MD5), nil
	}
	return "", nil
}

func HandleWalletStorePipelineResult() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseMultipartForm(500 << 20); err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"multipart",
				"failed to parse multipart form",
				err.Error(),
			).JSON(e)
		}

		versionIdentifier := e.Request.FormValue("version_identifier")
		runIdentifier := e.Request.FormValue("run_identifier")

		resultRecord, err := canonify.Resolve(e.App, runIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"pipeline_results",
				"record not found",
				err.Error(),
			).JSON(e)
		}

		versionName := strings.ReplaceAll(
			strings.Trim(versionIdentifier, "/"),
			"/",
			"-",
		)
		filename := versionName + "_result_video"
		videoFilename, videoURLs, err := saveUploadedFileToRecord(
			e, resultRecord, "result_video", "video_results", filename,
		)
		if err != nil {
			return err
		}

		filename = versionName + "_screenshot"
		frameFilename, frameURLs, err := saveUploadedFileToRecord(
			e, resultRecord, "last_frame", "screenshots", filename,
		)
		if err != nil {
			return err
		}

		return e.JSON(http.StatusOK, map[string]any{
			"status":               "success",
			"version":              versionIdentifier,
			"video_file_name":      videoFilename,
			"result_urls":          videoURLs,
			"last_frame_file_name": frameFilename,
			"last_frame_urls":      frameURLs,
		})
	}
}

func saveUploadedFileToRecord(
	e *core.RequestEvent,
	record *core.Record,
	formField string,
	recordField string,
	filename string,
) (string, []string, error) {
	file, fileHeader, err := e.Request.FormFile(formField)
	if err != nil {
		return "", nil, apierror.New(
			http.StatusBadRequest,
			"file",
			fmt.Sprintf("failed to read file for field %s", formField),
			err.Error(),
		).JSON(e)
	}
	defer file.Close()
	tmp, err := os.CreateTemp(
		"",
		fmt.Sprintf("%s_*%s", filename, filepath.Ext(fileHeader.Filename)),
	)
	if err != nil {
		return "", nil, apierror.New(
			http.StatusInternalServerError,
			"filesystem",
			"failed to create temp file",
			err.Error(),
		).JSON(e)
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, file); err != nil {
		return "", nil, apierror.New(
			http.StatusInternalServerError,
			"filesystem",
			"failed to write temp file",
			err.Error(),
		).JSON(e)
	}

	absPath, err := filepath.Abs(tmp.Name())
	if err != nil {
		return "", nil, apierror.New(
			http.StatusInternalServerError,
			"filesystem",
			"invalid temp file path",
			err.Error(),
		).JSON(e)
	}

	f, err := filesystem.NewFileFromPath(absPath)
	if err != nil {
		os.Remove(absPath)
		return "", nil, apierror.New(
			http.StatusInternalServerError,
			"filesystem",
			"failed to wrap file",
			err.Error(),
		).JSON(e)
	}

	existing := record.Get(recordField)
	var files []*filesystem.File

	if existing != nil {
		if slice, ok := existing.([]*filesystem.File); ok {
			files = append(files, slice...)
		}
	}
	files = append(files, f)

	record.Set(recordField, files)

	if err := e.App.Save(record); err != nil {
		os.Remove(absPath)
		return "", nil, apierror.New(
			http.StatusInternalServerError,
			"pipeline_results",
			"failed to save record with uploaded file",
			err.Error(),
		).JSON(e)
	}

	names := record.GetStringSlice(recordField)
	urls := make([]string, 0, len(names))

	for _, fn := range names {
		url := utils.JoinURL(
			e.App.Settings().Meta.AppURL,
			"api", "files", "pipeline_results",
			record.Id,
			record.GetString(recordField),
			fn,
		)
		urls = append(urls, url)
	}

	os.Remove(absPath)
	return fileHeader.Filename, urls, nil
}
