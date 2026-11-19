// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
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
		orgID, err := pbutils.GetUserOrganizationID(e.App, userID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"unable to get user organization ID",
				err.Error(),
			)
		}
		orgRecord, err := e.App.FindRecordById("organizations", orgID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"unable to get user organization record",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, orgRecord.FieldsData())
	}
}

// allOrganizationNamespaces returns the canonified name of every organization
// that has one.
func allOrganizationNamespaces(app core.App) ([]string, error) {
	records, err := app.FindAllRecords("organizations")
	if err != nil {
		return nil, fmt.Errorf("fetch organizations: %w", err)
	}

	namespaces := make([]string, 0, len(records))
	for _, record := range records {
		if canonified := record.GetString("canonified_name"); canonified != "" {
			namespaces = append(namespaces, canonified)
		}
	}
	return namespaces, nil
}
