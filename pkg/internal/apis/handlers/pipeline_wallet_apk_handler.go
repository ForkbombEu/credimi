// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"gopkg.in/yaml.v3"
)

const walletAPKFormFileField = "apk_file"
const walletAPKExternalSourceVersionID = "installed_from_external_source"
const walletAPKCleanupConfigKey = "temp_wallet_version"
const walletAPKMaxBytes = int64(1000 << 20)
const walletAPKDownloadTimeout = 30 * time.Second

var walletAPKURLDownloader = downloadWalletAPKFromURL

type pipelineRunWalletAPKRequest struct {
	PipelineIdentifier string
	CommitSHA          string
	APKURL             string
	APKFile            *multipart.FileHeader
}

type PipelineRunWalletAPKResponse struct {
	PipelineQueueResponse
	TempWalletVersionID         string `json:"temp_wallet_version_id,omitempty"`
	TempWalletVersionIdentifier string `json:"temp_wallet_version_identifier,omitempty"`
	PipelineIdentifier          string `json:"pipeline_identifier,omitempty"`
}

type pipelineRunWalletAPKContext struct {
	input              pipelineRunWalletAPKRequest
	organizationRecord *core.Record
	namespace          string
	userID             string
	userName           string
	userEmail          string
	pipelineRecord     *core.Record
	pipelineYAML       string
	walletRecord       *core.Record
	versionReferences  []walletAPKVersionReference
	apkFile            *filesystem.File
}

type walletAPKVersionReference struct {
	StepID    string
	VersionID string
}

type tempWalletVersion struct {
	Record     *core.Record
	Identifier string
}

func buildPipelineRunWalletAPKResponse(
	queueResponse PipelineQueueResponse,
	tempWalletVersionID string,
	tempWalletVersionIdentifier string,
	pipelineIdentifier string,
) PipelineRunWalletAPKResponse {
	return PipelineRunWalletAPKResponse{
		PipelineQueueResponse:       queueResponse,
		TempWalletVersionID:         tempWalletVersionID,
		TempWalletVersionIdentifier: tempWalletVersionIdentifier,
		PipelineIdentifier:          pipelineIdentifier,
	}
}

func HandlePipelineRunWalletAPK() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, apiErr := parsePipelineRunWalletAPKRequest(e)
		if apiErr != nil {
			return apiErr.JSON(e)
		}
		if apiErr := validatePipelineRunWalletAPKRequest(input); apiErr != nil {
			return apiErr.JSON(e)
		}

		runContext, apiErr := resolvePipelineRunWalletAPKContext(e, input)
		if apiErr != nil {
			return apiErr.JSON(e)
		}
		tempVersion, apiErr := createPipelineRunWalletAPKTempVersion(e.App, runContext)
		if apiErr != nil {
			return apiErr.JSON(e)
		}
		rewrittenYAML, apiErr := rewritePipelineRunWalletAPKYAML(
			runContext.pipelineYAML,
			runContext.versionReferences,
			tempVersion.Identifier,
		)
		if apiErr != nil {
			return apiErr.JSON(e)
		}
		manipulatedYAML, apiErr := injectPipelineRunWalletAPKCleanupConfig(rewrittenYAML, tempVersion)
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		queueResponse, apiErr := enqueuePipelineRun(e, pipelineQueueRunContext{
			pipelineRecord:     runContext.pipelineRecord,
			pipelineIdentifier: input.PipelineIdentifier,
			organizationRecord: runContext.organizationRecord,
			userID:             runContext.userID,
			userName:           runContext.userName,
			userEmail:          runContext.userEmail,
			yaml:               manipulatedYAML,
		})
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		response := buildPipelineRunWalletAPKResponse(
			queueResponse,
			tempVersion.Record.Id,
			tempVersion.Identifier,
			input.PipelineIdentifier,
		)
		return e.JSON(http.StatusOK, response)
	}
}

func resolvePipelineRunWalletAPKContext(
	e *core.RequestEvent,
	input pipelineRunWalletAPKRequest,
) (pipelineRunWalletAPKContext, *apierror.APIError) {
	if e.Auth == nil {
		return pipelineRunWalletAPKContext{}, apierror.New(
			http.StatusUnauthorized,
			"auth",
			"authentication required",
			"user not authenticated",
		)
	}

	orgRecord, err := GetUserOrganization(e.App, e.Auth.Id)
	if err != nil {
		return pipelineRunWalletAPKContext{}, apierror.New(
			http.StatusInternalServerError,
			"organization",
			"unable to get user organization record",
			err.Error(),
		)
	}
	namespace := strings.TrimSpace(orgRecord.GetString("canonified_name"))
	if namespace == "" {
		return pipelineRunWalletAPKContext{}, apierror.New(
			http.StatusInternalServerError,
			"organization",
			"unable to get user organization canonified name",
			"missing organization canonified name",
		)
	}

	pipelineRecord, err := canonify.Resolve(e.App, input.PipelineIdentifier)
	if err != nil {
		return pipelineRunWalletAPKContext{}, apierror.New(
			http.StatusNotFound,
			"pipeline_identifier",
			"pipeline not found",
			err.Error(),
		)
	}
	pipelineYAML := strings.TrimSpace(pipelineRecord.GetString("yaml"))
	if pipelineYAML == "" {
		return pipelineRunWalletAPKContext{}, apierror.New(
			http.StatusBadRequest,
			"pipeline_yaml",
			"pipeline yaml is required",
			"pipeline has no yaml",
		)
	}

	walletRecord, versionReferences, apiErr := resolvePipelineRunWalletAPKWallet(
		e.App,
		pipelineYAML,
	)
	if apiErr != nil {
		return pipelineRunWalletAPKContext{}, apiErr
	}
	apkFile, apiErr := resolvePipelineRunWalletAPKFile(e.Request.Context(), input)
	if apiErr != nil {
		return pipelineRunWalletAPKContext{}, apiErr
	}

	return pipelineRunWalletAPKContext{
		input:              input,
		organizationRecord: orgRecord,
		namespace:          namespace,
		userID:             e.Auth.Id,
		userName:           e.Auth.GetString("name"),
		userEmail:          e.Auth.GetString("email"),
		pipelineRecord:     pipelineRecord,
		pipelineYAML:       pipelineYAML,
		walletRecord:       walletRecord,
		versionReferences:  versionReferences,
		apkFile:            apkFile,
	}, nil
}

func resolvePipelineRunWalletAPKFile(
	ctx context.Context,
	input pipelineRunWalletAPKRequest,
) (*filesystem.File, *apierror.APIError) {
	filename := walletAPKFilename(input.CommitSHA, "")
	if input.APKFile != nil {
		if input.APKFile.Size > walletAPKMaxBytes {
			return nil, apierror.New(
				http.StatusRequestEntityTooLarge,
				"apk_file",
				"apk file too large",
				fmt.Sprintf("apk_file exceeds %d bytes", walletAPKMaxBytes),
			)
		}
		file, err := filesystem.NewFileFromMultipart(input.APKFile)
		if err != nil {
			return nil, apierror.New(
				http.StatusBadRequest,
				"apk_file",
				"failed to read apk_file",
				err.Error(),
			)
		}
		file.Name = filename
		file.OriginalName = filename
		return file, nil
	}

	apkURL := strings.TrimSpace(input.APKURL)
	parsedURL, err := url.Parse(apkURL)
	if err != nil || parsedURL == nil || parsedURL.Host == "" {
		return nil, apierror.New(
			http.StatusBadRequest,
			"apk_url",
			"invalid apk_url",
			"apk_url must be an http or https URL",
		)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, apierror.New(
			http.StatusBadRequest,
			"apk_url",
			"invalid apk_url",
			"apk_url must use http or https",
		)
	}

	file, err := walletAPKURLDownloader(ctx, apkURL, walletAPKFilename(input.CommitSHA, path.Base(parsedURL.Path)))
	if err != nil {
		return nil, apierror.New(
			http.StatusBadRequest,
			"apk_url",
			"failed to download apk_url",
			err.Error(),
		)
	}
	if file.Size > walletAPKMaxBytes {
		return nil, apierror.New(
			http.StatusRequestEntityTooLarge,
			"apk_url",
			"apk file too large",
			fmt.Sprintf("apk_url exceeds %d bytes", walletAPKMaxBytes),
		)
	}

	return file, nil
}

func downloadWalletAPKFromURL(
	ctx context.Context,
	apkURL string,
	filename string,
) (*filesystem.File, error) {
	ctx, cancel := context.WithTimeout(ctx, walletAPKDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apkURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: walletAPKDownloadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode > 399 {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	if resp.ContentLength > walletAPKMaxBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", walletAPKMaxBytes)
	}

	limited := io.LimitReader(resp.Body, walletAPKMaxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > walletAPKMaxBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", walletAPKMaxBytes)
	}

	return filesystem.NewFileFromBytes(data, filename)
}

func walletAPKFilename(commitSHA string, originalName string) string {
	base := canonify.CanonifyPlain(commitSHA)
	if base == "" {
		base = "wallet"
	}
	if strings.EqualFold(path.Ext(originalName), ".apk") {
		return base + ".apk"
	}
	return base + ".apk"
}

func resolvePipelineRunWalletAPKWallet(
	app core.App,
	pipelineYAML string,
) (*core.Record, []walletAPKVersionReference, *apierror.APIError) {
	workflowDefinition, err := pipelineinternal.ParseWorkflow(pipelineYAML)
	if err != nil {
		return nil, nil, apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse pipeline yaml",
			err.Error(),
		)
	}

	references := collectWalletAPKVersionReferences(workflowDefinition)
	if len(references) == 0 {
		return nil, nil, apierror.New(
			http.StatusBadRequest,
			"wallet_version",
			"pipeline must reference exactly one wallet",
			"no mobile-automation wallet version references found",
		)
	}

	walletIDs := map[string]*core.Record{}
	for _, ref := range references {
		versionID := canonify.NormalizePath(ref.VersionID)
		if versionID == walletAPKExternalSourceVersionID {
			return nil, nil, apierror.New(
				http.StatusBadRequest,
				"wallet_version",
				"external source wallet versions are not supported",
				"installed_from_external_source cannot anchor a temporary wallet version",
			)
		}

		versionRecord, err := canonify.Resolve(app, versionID)
		if err != nil {
			return nil, nil, apierror.New(
				http.StatusBadRequest,
				"wallet_version",
				"wallet version not found",
				err.Error(),
			)
		}
		if versionRecord.Collection().Name != "wallet_versions" {
			return nil, nil, apierror.New(
				http.StatusBadRequest,
				"wallet_version",
				"wallet version identifier is invalid",
				"version_id must resolve to a wallet_versions record",
			)
		}

		walletID := versionRecord.GetString("wallet")
		walletRecord, err := app.FindRecordById("wallets", walletID)
		if err != nil {
			return nil, nil, apierror.New(
				http.StatusBadRequest,
				"wallet",
				"wallet not found",
				err.Error(),
			)
		}
		walletIDs[walletID] = walletRecord
	}

	if len(walletIDs) != 1 {
		return nil, nil, apierror.New(
			http.StatusBadRequest,
			"wallet",
			"pipeline must reference exactly one wallet",
			"multiple wallets found in mobile-automation version references",
		)
	}

	for _, walletRecord := range walletIDs {
		return walletRecord, references, nil
	}

	return nil, nil, apierror.New(
		http.StatusBadRequest,
		"wallet",
		"pipeline must reference exactly one wallet",
		"wallet could not be resolved",
	)
}

func createPipelineRunWalletAPKTempVersion(
	app core.App,
	runContext pipelineRunWalletAPKContext,
) (tempWalletVersion, *apierror.APIError) {
	if runContext.walletRecord.GetString("owner") != runContext.organizationRecord.Id {
		return tempWalletVersion{}, apierror.New(
			http.StatusForbidden,
			"wallet",
			"wallet must belong to caller organization",
			"temporary wallet versions can only be created for caller-owned wallets",
		)
	}

	tag := canonify.CanonifyPlain(runContext.input.CommitSHA)
	if tag == "" {
		return tempWalletVersion{}, apierror.New(
			http.StatusBadRequest,
			"commit_sha",
			"commit_sha is invalid",
			"commit_sha cannot be canonified",
		)
	}

	existing, err := app.FindFirstRecordByFilter(
		"wallet_versions",
		"wallet = {:wallet} && owner = {:owner} && canonified_tag = {:tag}",
		dbx.Params{
			"wallet": runContext.walletRecord.Id,
			"owner":  runContext.organizationRecord.Id,
			"tag":    tag,
		},
	)
	if err == nil && existing != nil {
		return tempWalletVersion{}, apierror.New(
			http.StatusConflict,
			"wallet_version",
			"temporary wallet version already exists",
			"wallet version with this commit_sha already exists",
		)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return tempWalletVersion{}, apierror.New(
			http.StatusInternalServerError,
			"wallet_version",
			"failed to check existing wallet version",
			err.Error(),
		)
	}

	collection, err := app.FindCollectionByNameOrId("wallet_versions")
	if err != nil {
		return tempWalletVersion{}, apierror.New(
			http.StatusInternalServerError,
			"wallet_version",
			"failed to get wallet_versions collection",
			err.Error(),
		)
	}

	record := core.NewRecord(collection)
	record.Set("wallet", runContext.walletRecord.Id)
	record.Set("owner", runContext.organizationRecord.Id)
	record.Set("tag", tag)
	record.Set("android_installer", []*filesystem.File{runContext.apkFile})

	if err := app.Save(record); err != nil {
		return tempWalletVersion{}, apierror.New(
			http.StatusInternalServerError,
			"wallet_version",
			"failed to create temporary wallet version",
			err.Error(),
		)
	}

	identifier := strings.Join([]string{
		runContext.namespace,
		runContext.walletRecord.GetString("canonified_name"),
		record.GetString("canonified_tag"),
	}, "/")

	return tempWalletVersion{Record: record, Identifier: identifier}, nil
}

func rewritePipelineRunWalletAPKYAML(
	pipelineYAML string,
	versionReferences []walletAPKVersionReference,
	tempVersionIdentifier string,
) (string, *apierror.APIError) {
	workflowDefinition, err := pipelineinternal.ParseWorkflow(pipelineYAML)
	if err != nil {
		return "", apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse pipeline yaml",
			err.Error(),
		)
	}

	referencedVersions := map[string]struct{}{}
	for _, ref := range versionReferences {
		referencedVersions[canonify.NormalizePath(ref.VersionID)] = struct{}{}
	}

	rewriteCount := 0
	for i := range workflowDefinition.Steps {
		rewriteCount += rewriteWalletAPKStepVersion(
			&workflowDefinition.Steps[i].StepSpec,
			referencedVersions,
			tempVersionIdentifier,
		)
		for _, onErr := range workflowDefinition.Steps[i].OnError {
			if onErr != nil {
				rewriteCount += rewriteWalletAPKStepVersion(
					&onErr.StepSpec,
					referencedVersions,
					tempVersionIdentifier,
				)
			}
		}
		for _, onSuccess := range workflowDefinition.Steps[i].OnSuccess {
			if onSuccess != nil {
				rewriteCount += rewriteWalletAPKStepVersion(
					&onSuccess.StepSpec,
					referencedVersions,
					tempVersionIdentifier,
				)
			}
		}
	}

	if rewriteCount != len(versionReferences) {
		return "", apierror.New(
			http.StatusInternalServerError,
			"wallet_version",
			"failed to rewrite all wallet version references",
			fmt.Sprintf("rewrote %d of %d references", rewriteCount, len(versionReferences)),
		)
	}

	rewrittenYAML, err := yaml.Marshal(workflowDefinition)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"yaml",
			"failed to marshal pipeline yaml",
			err.Error(),
		)
	}

	return string(rewrittenYAML), nil
}

func rewriteWalletAPKStepVersion(
	step *pipelineinternal.StepSpec,
	referencedVersions map[string]struct{},
	tempVersionIdentifier string,
) int {
	if step == nil || step.Use != "mobile-automation" || step.With.Payload == nil {
		return 0
	}
	versionID, ok := step.With.Payload["version_id"].(string)
	if !ok {
		return 0
	}
	if _, ok := referencedVersions[canonify.NormalizePath(versionID)]; !ok {
		return 0
	}

	step.With.Payload["version_id"] = tempVersionIdentifier
	return 1
}

func injectPipelineRunWalletAPKCleanupConfig(
	pipelineYAML string,
	tempVersion tempWalletVersion,
) (string, *apierror.APIError) {
	workflowDefinition, err := pipelineinternal.ParseWorkflow(pipelineYAML)
	if err != nil {
		return "", apierror.New(
			http.StatusBadRequest,
			"yaml",
			"failed to parse pipeline yaml",
			err.Error(),
		)
	}

	if workflowDefinition.Config == nil {
		workflowDefinition.Config = map[string]any{}
	}
	if _, ok := workflowDefinition.Config[walletAPKCleanupConfigKey]; ok {
		return "", apierror.New(
			http.StatusBadRequest,
			"config",
			"temp_wallet_version config already exists",
			"pipeline config already contains temp_wallet_version",
		)
	}

	recordID := ""
	if tempVersion.Record != nil {
		recordID = tempVersion.Record.Id
	}
	workflowDefinition.Config[walletAPKCleanupConfigKey] = map[string]any{
		"record_id":  recordID,
		"identifier": tempVersion.Identifier,
		"cleanup":    true,
	}

	rewrittenYAML, err := yaml.Marshal(workflowDefinition)
	if err != nil {
		return "", apierror.New(
			http.StatusInternalServerError,
			"yaml",
			"failed to marshal pipeline yaml",
			err.Error(),
		)
	}

	return string(rewrittenYAML), nil
}

func collectWalletAPKVersionReferences(
	workflowDefinition *pipelineinternal.WorkflowDefinition,
) []walletAPKVersionReference {
	if workflowDefinition == nil {
		return nil
	}

	var refs []walletAPKVersionReference
	var collect func(pipelineinternal.StepSpec)
	collect = func(step pipelineinternal.StepSpec) {
		if step.Use != "mobile-automation" || step.With.Payload == nil {
			return
		}
		versionID, ok := step.With.Payload["version_id"].(string)
		if !ok || strings.TrimSpace(versionID) == "" {
			return
		}
		refs = append(refs, walletAPKVersionReference{
			StepID:    step.ID,
			VersionID: canonify.NormalizePath(versionID),
		})
	}

	for _, step := range workflowDefinition.Steps {
		collect(step.StepSpec)
		for _, onErr := range step.OnError {
			if onErr != nil {
				collect(onErr.StepSpec)
			}
		}
		for _, onSuccess := range step.OnSuccess {
			if onSuccess != nil {
				collect(onSuccess.StepSpec)
			}
		}
	}

	return refs
}

func parsePipelineRunWalletAPKRequest(
	e *core.RequestEvent,
) (pipelineRunWalletAPKRequest, *apierror.APIError) {
	contentType := e.Request.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return parsePipelineRunWalletAPKMultipartRequest(e)
	}

	if strings.HasPrefix(contentType, "application/json") {
		return parsePipelineRunWalletAPKJSONRequest(e)
	}

	if err := e.Request.ParseForm(); err != nil {
		return pipelineRunWalletAPKRequest{}, apierror.New(
			http.StatusBadRequest,
			"request",
			"failed to parse form request",
			err.Error(),
		)
	}

	return pipelineRunWalletAPKRequest{
		PipelineIdentifier: strings.TrimSpace(e.Request.FormValue("pipeline_identifier")),
		CommitSHA:          strings.TrimSpace(e.Request.FormValue("commit_sha")),
		APKURL:             strings.TrimSpace(e.Request.FormValue("apk_url")),
	}, nil
}

func parsePipelineRunWalletAPKMultipartRequest(
	e *core.RequestEvent,
) (pipelineRunWalletAPKRequest, *apierror.APIError) {
	if err := e.Request.ParseMultipartForm(1000 << 20); err != nil {
		return pipelineRunWalletAPKRequest{}, apierror.New(
			http.StatusBadRequest,
			"request",
			"failed to parse multipart request",
			err.Error(),
		)
	}

	var apkFile *multipart.FileHeader
	if e.Request.MultipartForm != nil {
		files := e.Request.MultipartForm.File[walletAPKFormFileField]
		if len(files) > 0 {
			apkFile = files[0]
		}
	}

	return pipelineRunWalletAPKRequest{
		PipelineIdentifier: strings.TrimSpace(e.Request.FormValue("pipeline_identifier")),
		CommitSHA:          strings.TrimSpace(e.Request.FormValue("commit_sha")),
		APKURL:             strings.TrimSpace(e.Request.FormValue("apk_url")),
		APKFile:            apkFile,
	}, nil
}

func parsePipelineRunWalletAPKJSONRequest(
	e *core.RequestEvent,
) (pipelineRunWalletAPKRequest, *apierror.APIError) {
	var input struct {
		PipelineIdentifier string `json:"pipeline_identifier"`
		CommitSHA          string `json:"commit_sha"`
		APKURL             string `json:"apk_url"`
	}
	if err := json.NewDecoder(e.Request.Body).Decode(&input); err != nil {
		return pipelineRunWalletAPKRequest{}, apierror.New(
			http.StatusBadRequest,
			"request",
			"invalid JSON input",
			err.Error(),
		)
	}

	return pipelineRunWalletAPKRequest{
		PipelineIdentifier: strings.TrimSpace(input.PipelineIdentifier),
		CommitSHA:          strings.TrimSpace(input.CommitSHA),
		APKURL:             strings.TrimSpace(input.APKURL),
	}, nil
}

func validatePipelineRunWalletAPKRequest(input pipelineRunWalletAPKRequest) *apierror.APIError {
	if strings.TrimSpace(input.PipelineIdentifier) == "" {
		return apierror.New(
			http.StatusBadRequest,
			"pipeline_identifier",
			"pipeline_identifier is required",
			"missing pipeline_identifier",
		)
	}
	if strings.TrimSpace(input.CommitSHA) == "" {
		return apierror.New(
			http.StatusBadRequest,
			"commit_sha",
			"commit_sha is required",
			"missing commit_sha",
		)
	}

	hasFile := input.APKFile != nil
	hasURL := strings.TrimSpace(input.APKURL) != ""
	switch {
	case hasFile && hasURL:
		return apierror.New(
			http.StatusBadRequest,
			"apk",
			"provide either apk_file or apk_url, not both",
			"multiple APK sources provided",
		)
	case !hasFile && !hasURL:
		return apierror.New(
			http.StatusBadRequest,
			"apk",
			"apk_file or apk_url is required",
			"missing APK source",
		)
	}

	return nil
}

func isMissingMultipartFile(err error) bool {
	return errors.Is(err, http.ErrMissingFile)
}
