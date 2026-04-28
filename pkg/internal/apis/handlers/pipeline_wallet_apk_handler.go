// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
)

const walletAPKFormFileField = "apk_file"

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

		if _, apiErr := resolvePipelineRunWalletAPKContext(e, input); apiErr != nil {
			return apiErr.JSON(e)
		}

		return apierror.New(
			http.StatusNotImplemented,
			"wallet_apk_run",
			"wallet APK pipeline run is not implemented yet",
			"pipeline context validated",
		).JSON(e)
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

	return pipelineRunWalletAPKContext{
		input:              input,
		organizationRecord: orgRecord,
		namespace:          namespace,
		userID:             e.Auth.Id,
		userName:           e.Auth.GetString("name"),
		userEmail:          e.Auth.GetString("email"),
		pipelineRecord:     pipelineRecord,
		pipelineYAML:       pipelineYAML,
	}, nil
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
