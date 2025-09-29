// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"net/http"

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
	AuthenticationRequired: true,
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
			Method:        http.MethodPost,
			Path:          "/identifier/get",
			Handler:       HandleGetIdentifier,
			RequestSchema: IdentifierGetRequest{},
			Description:   "Get the canonical identifier path of a record by id",
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
			"record":  record.FieldsData(),
		})

	}
}

type IdentifierGetRequest struct {
	Collection string `json:"collection"`
	ID         string `json:"id"`
}

func HandleGetIdentifier() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[IdentifierGetRequest](e)
		if err != nil {
			return err
		}

		rec, err := e.App.FindRecordById(req.Collection, req.ID)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"id",
				"record not found",
				err.Error(),
			).JSON(e)
		}

		tpl, ok := canonify.CanonifyPaths[req.Collection]
		if !ok {
			return apierror.New(
				http.StatusBadRequest,
				"collection",
				"no path template for collection",
				req.Collection,
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
			"collection": req.Collection,
			"id":         req.ID,
			"identifier": identifier,
		})
	}
}
