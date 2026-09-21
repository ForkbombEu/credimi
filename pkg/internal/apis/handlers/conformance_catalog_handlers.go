// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"

	"github.com/forkbombeu/credimi/pkg/conformancecatalog"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// ConformanceCatalogRoutes exposes catalog maintenance endpoints.
// List/get of checks use the native PocketBase collection API:
// GET /api/collections/conformance_checks/records
var ConformanceCatalogRoutes = routing.RouteGroup{
	BaseURL:                "/api/conformance-catalog",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:      http.MethodPost,
			Path:        "/rebuild",
			Handler:     HandleConformanceCatalogRebuild,
			Description: "Rebuild conformance_checks from config_templates (internal admin key)",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
	},
}

func HandleConformanceCatalogRebuild() func(*core.RequestEvent) error {
	return conformancecatalog.RebuildHTTP()
}
