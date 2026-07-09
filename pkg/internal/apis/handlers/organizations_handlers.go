// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

type OrganizationHasPublishedResponse struct {
	HasPublished bool `json:"has_published"`
}

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
		{
			Method:      http.MethodGet,
			Path:        "/visible-namespaces",
			Handler:     HandleGetVisibleOrganizationNamespaces,
			Description: "Get the caller organization namespace plus all published organization namespaces",
		},
		{
			Method:         http.MethodGet,
			Path:           "/{canonified_name}/has-published",
			Handler:        HandleGetOrganizationHasPublished,
			ResponseSchema: OrganizationHasPublishedResponse{},
			Description:    "Report whether the organization owns any published records",
		},
	},
}
var OrganizationTemporalInternalRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/organizations",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/namespaces",
			Handler:     HandleGetAllNamespaces,
			Description: "Get all organization namespaces (internal use)",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
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

func HandleGetAllNamespaces() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		records, err := e.App.FindAllRecords("organizations")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"failed to fetch organizations",
				err.Error(),
			)
		}

		namespaces := make([]string, 0, len(records))
		for _, record := range records {
			if canonified := record.GetString("canonified_name"); canonified != "" {
				namespaces = append(namespaces, canonified)
			}
		}

		return e.JSON(http.StatusOK, map[string]interface{}{
			"namespaces": namespaces,
		})
	}
}

func HandleGetOrganizationHasPublished() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		canonifiedName := strings.TrimSpace(e.Request.PathValue("canonified_name"))
		if canonifiedName == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organizations",
				"canonified_name is required",
				"missing organization canonified name",
			)
		}

		userOrg, err := pbutils.GetUserOrganization(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"unable to get user organization",
				err.Error(),
			)
		}

		if userOrg.GetString("canonified_name") != canonifiedName {
			return apierror.New(
				http.StatusForbidden,
				"authorization",
				"forbidden",
				"organization does not belong to the authenticated user",
			)
		}

		hasPublished, err := pbutils.OrganizationHasPublicEntities(e.App, userOrg.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"failed to check organization published records",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, OrganizationHasPublishedResponse{
			HasPublished: hasPublished,
		})
	}
}

func HandleGetVisibleOrganizationNamespaces() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		orgRecord, err := pbutils.GetUserOrganization(e.App, e.Auth.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"unable to get user organization",
				err.Error(),
			)
		}

		records, err := e.App.FindRecordsByFilter(
			"organizations",
			"published = true || id = {:id}",
			"name",
			-1,
			0,
			dbx.Params{"id": orgRecord.Id},
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"failed to fetch visible organizations",
				err.Error(),
			)
		}

		seen := make(map[string]struct{}, len(records))
		namespaces := make([]string, 0, len(records))
		for _, record := range records {
			namespace := record.GetString("canonified_name")
			if namespace == "" {
				continue
			}
			if _, ok := seen[namespace]; ok {
				continue
			}

			seen[namespace] = struct{}{}
			namespaces = append(namespaces, namespace)
		}
		sort.Strings(namespaces)

		return e.JSON(http.StatusOK, map[string]any{
			"namespaces": namespaces,
		})
	}
}
