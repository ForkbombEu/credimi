// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"archive/zip"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
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

type walletWorkflowStarter interface {
	Start(
		namespace string,
		input workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error)
}

var (
	walletTemporalClient       = temporalclient.GetTemporalClientWithNamespace
	walletWorkflowFactory      = func() walletWorkflowStarter { return workflows.NewWalletWorkflow() }
	walletWaitForPartialResult = workflowengine.WaitForPartialResult[map[string]any]
)

const (
	walletPlatformAndroid = "android"
	walletPlatformIOS     = "ios"
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
			Path:           "/get-installer-md5-or-etag",
			Handler:        HandleWalletGetInstallerMD5OrETag,
			RequestSchema:  WalletInstallerMD5OrETagRequest{},
			ResponseSchema: WalletInstallerMD5OrETagResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/store-pipeline-result",
			Handler: HandleWalletStorePipelineResult,
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
				apis.BodyLimit(500 << 20),
			},
		},
		{
			Method:  http.MethodDelete,
			Path:    "/temp-version/{record_id}",
			Handler: HandleWalletDeleteTempVersion,
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
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

func HandleWalletDeleteTempVersion() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		recordID := strings.TrimSpace(e.Request.PathValue("record_id"))
		if recordID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"record_id",
				"record_id is required",
				"missing record_id path parameter",
			).JSON(e)
		}

		var input walletDeleteTempVersionInput
		if e.Request.Body != nil {
			if err := json.NewDecoder(e.Request.Body).Decode(&input); err != nil &&
				!errors.Is(err, io.EOF) {
				return apierror.New(
					http.StatusBadRequest,
					"wallet_version",
					"invalid delete validation payload",
					err.Error(),
				).JSON(e)
			}
		}

		record, err := e.App.FindRecordById("wallet_versions", recordID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return e.JSON(http.StatusOK, map[string]any{"deleted": false})
			}
			return apierror.New(
				http.StatusInternalServerError,
				"wallet_version",
				"failed to find wallet version",
				err.Error(),
			).JSON(e)
		}

		if apiErr := validateTempWalletDeleteRequest(e.App, record, input); apiErr != nil {
			return apiErr.JSON(e)
		}

		if err := e.App.Delete(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"wallet_version",
				"failed to delete wallet version",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{"deleted": true})
	}
}

type walletDeleteTempVersionInput struct {
	ExpectedOwnerID    string `json:"expected_owner_id"`
	ExpectedIdentifier string `json:"expected_identifier"`
}

func validateTempWalletDeleteRequest(
	app core.App,
	record *core.Record,
	input walletDeleteTempVersionInput,
) *apierror.APIError {
	expectedOwnerID := strings.TrimSpace(input.ExpectedOwnerID)
	expectedIdentifier := strings.TrimSpace(input.ExpectedIdentifier)
	if expectedOwnerID == "" || expectedIdentifier == "" {
		return apierror.New(
			http.StatusBadRequest,
			"wallet_version",
			"delete validation payload is required",
			"expected_owner_id and expected_identifier are required",
		)
	}
	if record.GetString("owner") != expectedOwnerID {
		return apierror.New(
			http.StatusForbidden,
			"wallet_version",
			"temporary wallet version owner mismatch",
			"wallet version owner does not match expected_owner_id",
		)
	}
	resolved, err := canonify.Resolve(app, expectedIdentifier)
	if err != nil {
		return apierror.New(
			http.StatusForbidden,
			"wallet_version",
			"temporary wallet version identifier mismatch",
			err.Error(),
		)
	}
	if resolved.Id != record.Id {
		return apierror.New(
			http.StatusForbidden,
			"wallet_version",
			"temporary wallet version identifier mismatch",
			"expected_identifier does not resolve to the requested record",
		)
	}
	return nil
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
		w := walletWorkflowFactory()
		workflowInfo, err := w.Start(orgName, workflowInput)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start workflow",
				err.Error(),
			).JSON(e)
		}
		client, err := walletTemporalClient(
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
		result, err := walletWaitForPartialResult(
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

type WalletInstallerMD5OrETagRequest struct {
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
	WalletIdentifier        string `json:"wallet_identifier"`
	Platform                string `json:"platform"`
	SkipInstaller           bool   `json:"skip_installer,omitempty"`
}

type WalletInstallerMD5OrETagResponse struct {
	InstallerName       string `json:"installer_name"`
	InstallerIdentifier string `json:"installer_identifier"`
	RecordID            string `json:"record_id"`
	VersionIdentifier   string `json:"version_id"`
}

func HandleWalletGetInstallerMD5OrETag() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req WalletInstallerMD5OrETagRequest
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

		platform, err := normalizeWalletPlatform(req.Platform)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"platform",
				"invalid platform",
				err.Error(),
			).JSON(e)
		}
		if req.SkipInstaller {
			return e.JSON(http.StatusOK, WalletInstallerMD5OrETagResponse{
				VersionIdentifier: canonify.NormalizePath(req.WalletVersionIdentifier),
			})
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
		if apiErr := authorizeWalletInstallerAccess(e, versionRecord); apiErr != nil {
			return apiErr.JSON(e)
		}

		installerField := installerFieldForPlatform(platform)
		installer := versionRecord.GetString(installerField)
		if installer == "" {
			return apierror.New(
				http.StatusNotFound,
				installerField,
				fmt.Sprintf("no %s file found for this wallet version", installerField),
				fmt.Sprintf("%s field is empty", installerField),
			).JSON(e)
		}

		identifier, err := getFileMD5OrETagFromPocketBase(e.App, versionRecord, installer)
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

		return e.JSON(http.StatusOK, WalletInstallerMD5OrETagResponse{
			InstallerName:       installer,
			InstallerIdentifier: identifier,
			RecordID:            versionRecord.Id,
			VersionIdentifier:   versionIdentifier,
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

func normalizeWalletPlatform(platform string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case walletPlatformAndroid:
		return walletPlatformAndroid, nil
	case walletPlatformIOS:
		return walletPlatformIOS, nil
	default:
		return "", fmt.Errorf(
			"supported values are %s or %s",
			walletPlatformAndroid,
			walletPlatformIOS,
		)
	}
}

func installerFieldForPlatform(platform string) string {
	if platform == walletPlatformIOS {
		return "ios_installer"
	}

	return "android_installer"
}

func logRecordFieldForPlatform(platform string) string {
	if platform == walletPlatformIOS {
		return "ios_logstreams"
	}

	return "logcats"
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

		runnerIdentifier := e.Request.FormValue("runner_identifier")
		runIdentifier := e.Request.FormValue("run_identifier")
		platform, err := normalizeWalletPlatform(e.Request.FormValue("platform"))
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"platform",
				"invalid platform",
				err.Error(),
			).JSON(e)
		}

		resultRecord, err := canonify.Resolve(e.App, runIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"pipeline_results",
				"record not found",
				err.Error(),
			).JSON(e)
		}
		if apiErr := authorizeOwnerAccess(e, resultRecord.GetString("owner")); apiErr != nil {
			return apiErr.JSON(e)
		}

		versionName := strings.ReplaceAll(
			strings.Trim(runnerIdentifier, "/"),
			"/",
			"-",
		)
		filename := versionName + "_result_video"
		videoFilename, videoURLs, apierr := saveUploadedFileToRecord(
			e, resultRecord, "result_video", "video_results", filename, false,
		)
		if apierr != nil {
			return apierr.JSON(e)
		}

		filename = versionName + "_screenshot"
		frameFilename, frameURLs, apierr := saveUploadedFileToRecord(
			e, resultRecord, "last_frame", "screenshots", filename, false,
		)
		if apierr != nil {
			return apierr.JSON(e)
		}

		response := map[string]any{
			"status":               "success",
			"runner":               runnerIdentifier,
			"video_file_name":      videoFilename,
			"result_urls":          videoURLs,
			"last_frame_file_name": frameFilename,
			"screenshot_urls":      frameURLs,
		}

		filename = versionName + "_logfile"

		logFilename, logURLs, apierr := saveUploadedFileToRecord(
			e, resultRecord, "logfile", logRecordFieldForPlatform(platform), filename, true,
		)
		if apierr != nil {
			return apierr.JSON(e)
		}
		response["log_file_name"] = logFilename
		response["log_urls"] = logURLs

		return e.JSON(http.StatusOK, response)
	}
}

func authorizeOwnerAccess(e *core.RequestEvent, ownerID string) *apierror.APIError {
	if isInternalAdminPrincipal(e.Auth) {
		return nil
	}
	if ownerID == "" {
		return apierror.New(
			http.StatusInternalServerError,
			"owner",
			"owner missing",
			"record owner is required",
		)
	}

	allowed, err := belongsToAuthenticatedOrganization(e, ownerID)
	if err != nil {
		return apierror.New(
			http.StatusInternalServerError,
			"organization",
			"failed to get user organization",
			err.Error(),
		)
	}
	if !allowed {
		return apierror.New(
			http.StatusForbidden,
			"authorization",
			"forbidden",
			"record does not belong to the authenticated user's organization",
		)
	}

	return nil
}

func authorizeWalletInstallerAccess(
	e *core.RequestEvent,
	versionRecord *core.Record,
) *apierror.APIError {
	if versionRecord == nil {
		return apierror.New(
			http.StatusInternalServerError,
			"wallet_version",
			"wallet version missing",
			"wallet version is required",
		)
	}
	if isInternalAdminPrincipal(e.Auth) {
		return nil
	}
	if walletID := versionRecord.GetString("wallet"); walletID != "" {
		walletRecord, err := e.App.FindRecordById("wallets", walletID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"wallet",
				"failed to get wallet",
				err.Error(),
			)
		}
		if walletRecord.GetBool("published") {
			return nil
		}
	}

	return authorizeOwnerAccess(e, versionRecord.GetString("owner"))
}

func belongsToAuthenticatedOrganization(e *core.RequestEvent, ownerID string) (bool, error) {
	userOrgID, err := GetUserOrganizationID(e.App, e.Auth.Id)
	if err != nil {
		return false, err
	}
	return userOrgID == ownerID, nil
}

func isInternalAdminPrincipal(auth *core.Record) bool {
	if auth == nil || auth.Collection() == nil {
		return false
	}
	return auth.Collection().Name == "_superusers"
}

func saveUploadedFileToRecord(
	e *core.RequestEvent,
	record *core.Record,
	formField string,
	recordField string,
	filename string,
	zipUpload bool,
) (string, []string, *apierror.APIError) {
	file, fileHeader, err := e.Request.FormFile(formField)
	if err != nil {
		return "", nil, apierror.New(
			http.StatusBadRequest,
			"file",
			fmt.Sprintf("failed to read file for field %s", formField),
			err.Error(),
		)
	}
	defer file.Close()
	ext := filepath.Ext(fileHeader.Filename)
	if zipUpload {
		ext = ".zip"
	}
	tmp, err := os.CreateTemp("", fmt.Sprintf("%s_*%s", filename, ext))
	if err != nil {
		return "", nil, apierror.New(
			http.StatusInternalServerError,
			"filesystem",
			"failed to create temp file",
			err.Error(),
		)
	}
	defer tmp.Close()

	if zipUpload {
		zipWriter := zip.NewWriter(tmp)
		entryName := fileHeader.Filename
		if entryName == "" {
			entryName = filename
		}
		entryWriter, err := zipWriter.Create(entryName)
		if err != nil {
			zipWriter.Close()
			return "", nil, apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to create zip entry",
				err.Error(),
			)
		}

		if _, err := io.Copy(entryWriter, file); err != nil {
			zipWriter.Close()
			return "", nil, apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to write zip entry",
				err.Error(),
			)
		}

		if err := zipWriter.Close(); err != nil {
			return "", nil, apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to finalize zip file",
				err.Error(),
			)
		}
	} else {
		if _, err := io.Copy(tmp, file); err != nil {
			return "", nil, apierror.New(
				http.StatusInternalServerError,
				"filesystem",
				"failed to write temp file",
				err.Error(),
			)
		}
	}

	absPath, err := filepath.Abs(tmp.Name())
	if err != nil {
		return "", nil, apierror.New(
			http.StatusInternalServerError,
			"filesystem",
			"invalid temp file path",
			err.Error(),
		)
	}

	f, err := filesystem.NewFileFromPath(absPath)
	if err != nil {
		os.Remove(absPath)
		return "", nil, apierror.New(
			http.StatusInternalServerError,
			"filesystem",
			"failed to wrap file",
			err.Error(),
		)
	}

	existing := record.GetStringSlice(recordField)

	values := make([]any, 0, len(existing)+1)

	for _, name := range existing {
		values = append(values, name)
	}

	// add new file
	values = append(values, f)

	record.Set(recordField, values)

	if err := e.App.Save(record); err != nil {
		os.Remove(absPath)
		return "", nil, apierror.New(
			http.StatusInternalServerError,
			"pipeline_results",
			"failed to save record with uploaded file",
			err.Error(),
		)
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
	if zipUpload {
		return filename + ".zip", urls, nil
	}
	return fileHeader.Filename, urls, nil
}
