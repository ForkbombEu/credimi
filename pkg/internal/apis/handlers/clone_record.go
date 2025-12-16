// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"slices"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var CloneRecord routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:  http.MethodPost,
			Path:    "/clone-record",
			Handler: HandleCloneRecord,
		},
	},
}

type CloneConfig struct {
	makeUnique []string
	exclude    []string
}

var (
	rng          = rand.New(rand.NewSource(time.Now().UnixNano()))
	systemFields = []string{"id", "created", "updated"}
)

func makeUniqueValue(original string) string {
	randomDigits := fmt.Sprintf("%04d", rng.Intn(10000))
	return original + "_copy" + randomDigits
}

var CloneConfigs = map[string]CloneConfig{
	"credentials": {
		makeUnique: []string{"name"},
		exclude:    []string{"canonified_name"},
	},
}

func cloneRecord(app core.App, originalRecord *core.Record, config CloneConfig) (*core.Record, error) {
	newRecord := core.NewRecord(originalRecord.Collection())

	collection, err := app.FindCollectionByNameOrId(originalRecord.Collection().Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	fileFields := make(map[string]bool)
	for _, field := range collection.Fields {
		if field.Type() == "file" {
			fileFields[field.GetName()] = true
		}
	}

	fileFieldValues := make(map[string]string)
	for fieldName := range fileFields {
		if fileName := originalRecord.GetString(fieldName); fileName != "" {
			fileFieldValues[fieldName] = fileName
		}
	}

	for key, value := range originalRecord.FieldsData() {
		if slices.Contains(systemFields, key) {
			continue
		}
		if slices.Contains(config.exclude, key) {
			continue
		}

		if fileFields[key] {
			newRecord.Set(key, nil)
			continue
		}

		if slices.Contains(config.makeUnique, key) {
			if strVal, ok := value.(string); ok && strVal != "" {
				newRecord.Set(key, makeUniqueValue(strVal))
				continue
			}
		}
		newRecord.Set(key, value)
	}

	if err := app.Save(newRecord); err != nil {
		return nil, fmt.Errorf("failed to save base record: %w", err)
	}

	if len(fileFieldValues) > 0 {
		if err := cloneFiles(app, originalRecord, newRecord, fileFieldValues); err != nil {
			app.Logger().Error(fmt.Sprintf("Error cloning files: %v", err))
		}
	}

	return newRecord, nil
}

type CloneRequest struct {
	ID         string `json:"id"`
	Collection string `json:"collection"`
}

type CloneResponse struct {
	ClonedRecord map[string]interface{} `json:"cloned_record"`
	Message      string                 `json:"message"`
}

func HandleCloneRecord() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req CloneRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return apis.NewBadRequestError("Invalid JSON", err)
		}
		if req.ID == "" || req.Collection == "" {
			return apis.NewBadRequestError("id and collection are required", nil)
		}
		config, exists := CloneConfigs[req.Collection]
		if !exists {
			return apis.NewBadRequestError(
				fmt.Sprintf("Collection '%s' not supported for cloning", req.Collection), nil,
			)
		}
		originalRecord, err := e.App.FindRecordById(req.Collection, req.ID)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"not_found",
				fmt.Sprintf("Record '%s' not found in collection '%s'", req.ID, req.Collection),
				err.Error(),
			).JSON(e)
		}

		clonedRecord, err := cloneRecord(e.App, originalRecord, config)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"clone_failed",
				fmt.Sprintf("Failed to clone record in '%s'", req.Collection),
				err.Error(),
			).JSON(e)
		}
		response := CloneResponse{
			ClonedRecord: clonedRecord.FieldsData(),
			Message:      fmt.Sprintf("Record cloned from '%s'", req.Collection),
		}

		return e.JSON(http.StatusOK, response)
	}
}

func cloneFiles(app core.App, originalRecord, newRecord *core.Record, fileFieldValues map[string]string) error {
	if len(fileFieldValues) == 0 {
		return nil
	}

	fs, err := app.NewFilesystem()
	if err != nil {
		return fmt.Errorf("failed to create filesystem: %w", err)
	}
	defer fs.Close()

	originalBasePath := originalRecord.BaseFilesPath()
	filesMap := make(map[string]interface{})

	for fieldName, fileName := range fileFieldValues {
		originalPath := originalBasePath + "/" + fileName

		reader, err := fs.GetFile(originalPath)
		if err != nil {
			app.Logger().Warn(fmt.Sprintf("File not found, skipping: %s", originalPath))
			filesMap[fieldName] = nil
			continue
		}

		fileBytes, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			app.Logger().Warn(fmt.Sprintf("Failed to read file, skipping: %s", fileName))
			filesMap[fieldName] = nil
			continue
		}

		file, err := filesystem.NewFileFromBytes(fileBytes, fileName)
		if err != nil {
			app.Logger().Warn(fmt.Sprintf("Failed to create file, skipping: %s", fileName))
			filesMap[fieldName] = nil
			continue
		}

		filesMap[fieldName] = file
	}

	hasFiles := false
	for fieldName, file := range filesMap {
		if file != nil {
			newRecord.Set(fieldName, file)
			hasFiles = true
		} else {
			newRecord.Set(fieldName, nil)
		}
	}

	if hasFiles {
		if err := app.Save(newRecord); err != nil {
			return fmt.Errorf("failed to save with files: %w", err)
		}
	}

	return nil
}
