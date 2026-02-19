// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

type IdentifierValidateRequest struct {
	CanonifiedName string `json:"canonified_name"`
}

var CanonifyRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/canonify",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/identifier/validate",
			Handler:       HandleIdentifierValidate,
			RequestSchema: IdentifierValidateRequest{},
			Description:   "Validate an entity identifier",
		},
		{
			Method:      http.MethodGet,
			Path:        "/identifier/get",
			Handler:     HandleGetIdentifier,
			Description: "Get the canonical identifier path of a record by id",
		},
	},
}

func HandleIdentifierValidate() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[IdentifierValidateRequest](e)
		if err != nil {
			return err
		}
		record, err := canonify.Validate(e.App, req.CanonifiedName)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"identifier",
				"failed to validate identifier",
				err.Error(),
			).JSON(e)
		}
		return e.JSON(http.StatusOK, map[string]any{
			"message": "valid identifier",
			"record":  record,
		})
	}
}

func HandleGetIdentifier() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		collection := e.Request.URL.Query().Get("collection")
		if collection == "" {
			return apierror.New(
				http.StatusBadRequest,
				"collection",
				"collection is required",
				"missing collection",
			).JSON(e)
		}

		recID := e.Request.URL.Query().Get("id")
		if recID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"id",
				"record id is required",
				"missing id",
			).JSON(e)
		}

		rec, err := e.App.FindRecordById(collection, recID)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"id",
				"record not found",
				err.Error(),
			).JSON(e)
		}

		if collection == "marketplace_items" {
			colType := rec.GetString("type")
			collection = strings.Trim(colType, `"`)
			rec, err = e.App.FindRecordById(collection, recID)
			if err != nil {
				return apierror.New(
					http.StatusNotFound,
					"id",
					"record not found",
					err.Error(),
				).JSON(e)
			}
		}

		tpl, ok := canonify.CanonifyPaths[collection]
		if !ok {
			return apierror.New(
				http.StatusBadRequest,
				"collection",
				"no path template for collection",
				collection,
			).JSON(e)
		}

		identifier, err := canonify.BuildPath(e.App, rec, tpl, "")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"path",
				"failed to build identifier path",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"collection": collection,
			"id":         recID,
			"identifier": identifier,
		})
	}
}
