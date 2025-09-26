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
	Identifier string `json:"identifier"`
	Collection string `json:"collection"`
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
	},
}

func HandleIdentifierValidate() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[IdentifierValidateRequest](e)
		if err != nil {
			return err
		}
		record, err := canonify.Validate(e.App, req.Collection, req.Identifier)
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
