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
			Handler:     conformancecatalog.RebuildHTTP,
			Description: "Rebuild ephemeral conformance catalog from config_templates (internal admin key)",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
	},
}

// ConformanceChecksRecordsRoutes owns the PocketBase-shaped collection URL for
// the ephemeral catalog (no durable data.db collection).
var ConformanceChecksRecordsRoutes = routing.RouteGroup{
	BaseURL:                "/api/collections/conformance_checks",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/records",
			Handler:     conformancecatalog.RecordsListHTTP,
			Description: "List conformance catalog checks (ephemeral; PB URL shape)",
		},
		{
			Method:      http.MethodGet,
			Path:        "/records/{id}",
			Handler:     conformancecatalog.RecordViewHTTP,
			Description: "Get one conformance catalog check by id",
		},
		{
			Method:      http.MethodPost,
			Path:        "/records",
			Handler:     conformancecatalog.RecordsWriteRejectHTTP,
			Description: "Reject creates on the read-only conformance catalog",
		},
		{
			Method:      http.MethodPatch,
			Path:        "/records/{id}",
			Handler:     conformancecatalog.RecordsWriteRejectHTTP,
			Description: "Reject updates on the read-only conformance catalog",
		},
		{
			Method:      http.MethodDelete,
			Path:        "/records/{id}",
			Handler:     conformancecatalog.RecordsWriteRejectHTTP,
			Description: "Reject deletes on the read-only conformance catalog",
		},
	},
}

// ConformanceSuitesRecordsRoutes owns the PocketBase-shaped suite projection URL.
var ConformanceSuitesRecordsRoutes = routing.RouteGroup{
	BaseURL:                "/api/collections/conformance_suites",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/records",
			Handler:     conformancecatalog.SuitesListHTTP,
			Description: "List conformance catalog suites (ephemeral; PB URL shape)",
		},
		{
			Method:      http.MethodGet,
			Path:        "/records/{id}",
			Handler:     conformancecatalog.SuiteViewHTTP,
			Description: "Get one conformance catalog suite by id",
		},
		{
			Method:      http.MethodPost,
			Path:        "/records",
			Handler:     conformancecatalog.SuitesWriteRejectHTTP,
			Description: "Reject creates on the read-only conformance suites catalog",
		},
		{
			Method:      http.MethodPatch,
			Path:        "/records/{id}",
			Handler:     conformancecatalog.SuitesWriteRejectHTTP,
			Description: "Reject updates on the read-only conformance suites catalog",
		},
		{
			Method:      http.MethodDelete,
			Path:        "/records/{id}",
			Handler:     conformancecatalog.SuitesWriteRejectHTTP,
			Description: "Reject deletes on the read-only conformance suites catalog",
		},
	},
}
