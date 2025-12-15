// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
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
	CollectionName string
	AbsoluteFields []string
	UniqueFields   []string
	AutoFields     []string
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func makeUniqueValue(original string) string {
	randomDigits := fmt.Sprintf("%04d", rng.Intn(10000))
	return original + "_copy" + randomDigits
}

var CloneConfigs = map[string]CloneConfig{
	"credentials": {
		CollectionName: "credentials",
		AbsoluteFields: []string{"id", "created", "updated"},
		UniqueFields:   []string{"name"},
		AutoFields:     []string{"canonified_name"},
	},
}

func cloneRecord(app core.App, originalRecord *core.Record, config CloneConfig) (*core.Record, error) {
	clonedRecord := core.NewRecord(originalRecord.Collection())

	absoluteFields := make(map[string]bool)
	for _, field := range config.AbsoluteFields {
		absoluteFields[field] = true
	}

	autoFields := make(map[string]bool)
	for _, field := range config.AutoFields {
		autoFields[field] = true
	}

	uniqueFields := make(map[string]bool)
	for _, field := range config.UniqueFields {
		uniqueFields[field] = true
	}

	for key, value := range originalRecord.FieldsData() {
		if absoluteFields[key] || autoFields[key] {
			continue
		}
		if uniqueFields[key] {
			if strVal, ok := value.(string); ok && strVal != "" {
				uniqueValue := makeUniqueValue(strVal)
				clonedRecord.Set(key, uniqueValue)
				continue
			}
		}
		clonedRecord.Set(key, value)
	}
	if err := app.Save(clonedRecord); err != nil {
		return nil, err
	}

	return clonedRecord, nil
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
