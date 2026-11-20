// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const pipelineStepScreenshotFormField = "screenshots"

func HandleStorePipelineStepScreenshots() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := e.Request.ParseMultipartForm(500 << 20); err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"multipart",
				"failed to parse multipart form",
				err.Error(),
			)
		}

		runIdentifier := e.Request.FormValue("run_identifier")
		deviceIdentifier := canonify.NormalizePath(e.Request.FormValue("device_identifier"))
		stepID := canonify.CanonifyPlain(e.Request.FormValue("step_id"))
		if stepID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"step_id",
				"step_id is required",
				"missing step_id",
			)
		}

		resultRecord, err := canonify.Resolve(e.App, runIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"pipeline_results",
				"record not found",
				err.Error(),
			)
		}
		if apiErr := authorizePipelineResultDeviceStoreAccess(
			e,
			resultRecord,
			deviceIdentifier,
		); apiErr != nil {
			return apiErr
		}

		filenames, urls, apiErr := storePipelineStepScreenshotFiles(
			e,
			resultRecord,
			stepID,
		)
		if apiErr != nil {
			return apiErr
		}

		return e.JSON(http.StatusOK, map[string]any{
			"status":                "success",
			"step_id":               stepID,
			"screenshot_file_names": filenames,
			"screenshot_urls":       urls,
		})
	}
}

func storePipelineStepScreenshotFiles(
	e *core.RequestEvent,
	record *core.Record,
	stepID string,
) ([]string, []string, *apierror.APIError) {
	if e.Request.MultipartForm == nil {
		return nil, nil, apierror.New(
			http.StatusBadRequest,
			"screenshots",
			"screenshots are required",
			"missing multipart files",
		)
	}
	headers := e.Request.MultipartForm.File[pipelineStepScreenshotFormField]
	if len(headers) == 0 {
		return nil, nil, apierror.New(
			http.StatusBadRequest,
			"screenshots",
			"screenshots are required",
			"missing multipart files",
		)
	}

	field, ok := record.Collection().Fields.GetByName("maestro_screenshots").(*core.FileField)
	if !ok {
		return nil, nil, apierror.New(
			http.StatusInternalServerError,
			"pipeline_results",
			"maestro_screenshots is not a file field",
			"invalid collection schema",
		)
	}
	files := make([]*filesystem.File, 0, len(headers))
	for _, header := range headers {
		file, err := header.Open()
		if err != nil {
			return nil, nil, uploadedStepScreenshotError("open", err)
		}
		data, readErr := io.ReadAll(file)
		closeErr := file.Close()
		if readErr != nil {
			return nil, nil, uploadedStepScreenshotError("read", readErr)
		}
		if closeErr != nil {
			return nil, nil, uploadedStepScreenshotError("close", closeErr)
		}

		originalName := filepath.Base(header.Filename)
		if originalName == "." || originalName == "" {
			return nil, nil, apierror.New(
				http.StatusBadRequest,
				"screenshots",
				"invalid screenshot filename",
				"filename is required",
			)
		}
		requestedName := stepID + "-" + originalName
		storedFile, err := filesystem.NewFileFromBytes(data, requestedName)
		if err != nil {
			return nil, nil, uploadedStepScreenshotError("prepare", err)
		}
		files = append(files, storedFile)
	}

	// Append to a record reloaded under the per-record lock: concurrent uploads
	// to the same result would otherwise overwrite the field from a stale copy.
	unlock := lockPipelineResult(record.Id)
	defer unlock()
	fresh, err := e.App.FindRecordById("pipeline_results", record.Id)
	if err != nil {
		return nil, nil, apierror.New(
			http.StatusInternalServerError,
			"pipeline_results",
			"failed to save Maestro screenshots",
			err.Error(),
		)
	}
	before := fresh.GetStringSlice("maestro_screenshots")
	if field.MaxSelect > 0 && len(before)+len(files) > field.MaxSelect {
		return nil, nil, apierror.New(
			http.StatusBadRequest,
			"screenshots",
			"too many Maestro screenshots",
			fmt.Sprintf("maximum %d files allowed", field.MaxSelect),
		)
	}
	fresh.Set("maestro_screenshots+", files)
	if err := e.App.Save(fresh); err != nil {
		return nil, nil, apierror.New(
			http.StatusInternalServerError,
			"pipeline_results",
			"failed to save Maestro screenshots",
			err.Error(),
		)
	}

	newNames := fresh.GetStringSlice("maestro_screenshots")[len(before):]
	urls := make([]string, 0, len(newNames))
	for _, filename := range newNames {
		urls = append(urls, utils.JoinURL(
			e.App.Settings().Meta.AppURL,
			"api", "files", "pipeline_results", fresh.Id, filename,
		))
	}
	return newNames, urls, nil
}

func uploadedStepScreenshotError(operation string, err error) *apierror.APIError {
	return apierror.New(
		http.StatusBadRequest,
		"screenshots",
		fmt.Sprintf("failed to %s screenshot", strings.TrimSpace(operation)),
		err.Error(),
	)
}
