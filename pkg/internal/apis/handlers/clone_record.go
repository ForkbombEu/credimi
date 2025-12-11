// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
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
			Method:  http.MethodGet,
			Path:    "/clone-record",
			Handler: HandleGetCloneRecord,
		},
	},
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func HandleGetCloneRecord() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		recordID := e.Request.URL.Query().Get("id")

		if recordID == "" {
			return apis.NewBadRequestError(
				"Parameter 'id' is required",
				nil,
			)
		}

		originalRecord, err := canonify.Resolve(e.App, recordID)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"resolve",
				"failed to resolve collection path",
				err.Error(),
			).JSON(e)
		}

		clonedRecord, err := cloneRecord(e.App, originalRecord)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"clone",
				"failed to clone record",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"cloned_record": clonedRecord.FieldsData(),
		})
	}
}

func cloneRecord(app core.App, originalRecord *core.Record) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId(originalRecord.Collection().Id)
	if err != nil {
		return nil, err
	}

	clonedRecord := core.NewRecord(collection)

	for key, value := range originalRecord.FieldsData() {
		if isAbsoluteSystemField(key) {
			continue
		}

		if isAutoCalculatedField(key) {
			continue
		}

		if shouldMakeUnique(key) {
			if strVal, ok := value.(string); ok && strVal != "" {
				clonedRecord.Set(key, makeUniqueValue(strVal))
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

func isAbsoluteSystemField(fieldName string) bool {
	systemFields := []string{
		"id", "created", "updated",
	}

	for _, field := range systemFields {
		if field == fieldName {
			return true
		}
	}
	return false
}

func isAutoCalculatedField(fieldName string) bool {
	commonAutoFields := []string{
		"canonified_name",
		"deeplink",
		"logo",
	}

	for _, field := range commonAutoFields {
		if field == fieldName {
			return true
		}
	}

	return false
}

func shouldMakeUnique(fieldName string) bool {
	uniqueFields := []string{
		"name",
	}

	for _, field := range uniqueFields {
		if field == fieldName {
			return true
		}
	}
	return false
}

func makeUniqueValue(original string) string {
	randomDigits := fmt.Sprintf("%04d", rng.Intn(10000))
	return original + "_copy" + randomDigits
}
