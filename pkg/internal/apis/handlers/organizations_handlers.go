// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var OrganizationRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/organizations",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/my",
			Handler:     HandleGetMyOrganization,
			Description: "Get the current user's organization info",
		},
	},
}

func HandleGetMyOrganization() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		userID := e.Auth.Id
		orgID, err := GetUserOrganizationID(e.App, userID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"unable to get user organization ID",
				err.Error(),
			).JSON(e)
		}
		orgRecord, err := e.App.FindRecordById("organizations", orgID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"unable to get user organization record",
				err.Error(),
			).JSON(e)
		}
		return e.JSON(http.StatusOK, orgRecord.FieldsData())
	}
}
