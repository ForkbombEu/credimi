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
	"github.com/pocketbase/dbx"
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
	makeUnique   []string
	exclude      []string
	CanDuplicate func(e *core.RequestEvent, originalRecord *core.Record) (bool, error)
	BeforeSave   func(e *core.RequestEvent, newRecord *core.Record) error
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
		makeUnique:   []string{"name"},
		exclude:      []string{"canonified_name"},
		CanDuplicate: canDuplicateRecordIfRequestIsFromOwnerOrRecordIsPublic,
	},
	"wallet_actions": {
		makeUnique:   []string{"name"},
		exclude:      []string{"canonified_name"},
		CanDuplicate: canDuplicateRecordIfRequestIsFromOwnerOrRecordIsPublic,
	},
	"use_cases_verifications": {
		makeUnique:   []string{"name"},
		exclude:      []string{"canonified_name"},
		CanDuplicate: canDuplicateRecordIfRequestIsFromOwnerOrRecordIsPublic,
	},
	"pipelines": {
		makeUnique:   []string{"name"},
		exclude:      []string{"canonified_name"},
		CanDuplicate: canDuplicateRecordIfRequestIsFromOwnerOrRecordIsPublic,
		BeforeSave: UpdateOwnerField,
	},
	"custom_checks": {
		makeUnique:   []string{"name"},
		exclude:      []string{"canonified_name"},
		CanDuplicate: canDuplicateRecordIfRequestIsFromOwnerOrRecordIsPublic,
	},
}

func canDuplicateRecordIfRequestIsFromOwnerOrRecordIsPublic(e *core.RequestEvent, originalRecord *core.Record) (bool, error) {
	auth := e.Auth
	if auth == nil {
		return false, apis.NewUnauthorizedError("Authentication required", nil)
	}
	if originalRecord.GetBool("published") == true {
		return true, nil
	} else {
		return canDuplicateRecordWithOwnerFieldAndWithoutPublishedField(e, originalRecord)
	}
}

func canDuplicateRecordWithOwnerFieldAndWithoutPublishedField(e *core.RequestEvent, originalRecord *core.Record) (bool, error) {
	auth := e.Auth
	if auth == nil {
		return false, apis.NewUnauthorizedError("Authentication required", nil)
	}
	orgID := originalRecord.GetString("owner")
	if orgID == "" {
		return false, apis.NewForbiddenError("Record has no owner", nil)
	}
	authRecord, err := e.App.FindFirstRecordByFilter("orgAuthorizations", "user={:user} && organization={:org}",
		dbx.Params{"user": auth.Id, "org": orgID})
	if err != nil || authRecord == nil {
		return false, apis.NewForbiddenError("Not authorized for this organization", nil)
	}
	return true, nil
}

func UpdateOwnerField(e *core.RequestEvent, newRecord *core.Record) error {
    auth := e.Auth
    if auth == nil {
        return apis.NewUnauthorizedError("Authentication required", nil)
    }
    authRecord, err := e.App.FindFirstRecordByFilter("orgAuthorizations", "user={:user}",
        dbx.Params{"user": auth.Id})
    if err != nil {
        return fmt.Errorf("failed to find user organization: %w", err)
    }
    if authRecord == nil {
        return apis.NewForbiddenError("User not authorized for any organization", nil)
    }
    
    orgID := authRecord.GetString("organization")
    if orgID == "" {
        return fmt.Errorf("organization ID not found in authorization record")
    }
    
    newRecord.Set("owner", orgID)
    
    return nil
}

func cloneRecord(e *core.RequestEvent, originalRecord *core.Record, config CloneConfig) (*core.Record, error) {
	newRecord := core.NewRecord(originalRecord.Collection())
	collection := originalRecord.Collection()

	fileFields := make(map[string]bool)
	for _, field := range collection.Fields {
		if field.Type() == "file" {
			fileFields[field.GetName()] = true
		}
	}

	fileFieldValues := make(map[string]interface{})
	for fieldName := range fileFields {
		value := originalRecord.Get(fieldName)
		if value != nil {
			fileFieldValues[fieldName] = value
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

	if config.BeforeSave != nil {
    	if err := config.BeforeSave(e, newRecord); err != nil {
        	return nil, fmt.Errorf("failed in BeforeSave: %w", err)
    	}
		newRecord.Set("published", false)
	}
	if err := e.App.Save(newRecord); err != nil {
		return nil, fmt.Errorf("failed to save base record: %w", err)
	}

	if len(fileFieldValues) > 0 {
		if err := cloneFiles(e.App, originalRecord, newRecord, fileFieldValues); err != nil {
			e.App.Logger().Error(fmt.Sprintf("Error cloning files: %v", err))
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

		if config.CanDuplicate != nil {
			allowed, authErr := config.CanDuplicate(e, originalRecord)
			if authErr != nil {
				return authErr
			}
			if !allowed {
				return apis.NewForbiddenError(
					"Not authorized to clone this record",
					nil,
				)
			}
		} else if e.Auth == nil {
			return apis.NewUnauthorizedError("Authentication required", nil)
		}

		clonedRecord, err := cloneRecord(e, originalRecord, config)
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

func cloneFiles(app core.App, originalRecord, newRecord *core.Record, fileFieldValues map[string]interface{}) error {
	if len(fileFieldValues) == 0 {
		return nil
	}

	fs, err := app.NewFilesystem()
	if err != nil {
		return fmt.Errorf("failed to create filesystem: %w", err)
	}
	defer fs.Close()

	originalBasePath := originalRecord.BaseFilesPath()
	hasFiles := false

	for fieldName, value := range fileFieldValues {
		fileNames := extractFileNames(value)

		if len(fileNames) == 0 {
			newRecord.Set(fieldName, nil)
			continue
		}

		var files []*filesystem.File
		for _, fileName := range fileNames {
			file, err := cloneSingleFile(fs, originalBasePath, fileName)
			if err != nil {
				app.Logger().Warn(fmt.Sprintf("Failed to clone file %s: %v", fileName, err))
				continue
			}
			files = append(files, file)
		}

		if len(files) > 0 {
			if len(files) == 1 {
				newRecord.Set(fieldName, files[0])
			} else {
				newRecord.Set(fieldName, files)
			}
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

func extractFileNames(value interface{}) []string {
	switch v := value.(type) {
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	case []string:
		var result []string
		for _, s := range v {
			if s != "" {
				result = append(result, s)
			}
		}
		return result
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func cloneSingleFile(fs *filesystem.System, basePath, fileName string) (*filesystem.File, error) {
	originalPath := basePath + "/" + fileName

	reader, err := fs.GetFile(originalPath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}
	defer reader.Close()

	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	file, err := filesystem.NewFileFromBytes(fileBytes, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return file, nil
}
