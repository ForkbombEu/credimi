// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

func AddTemplatingRoutes(app core.App) {
	routing.AddGroupRoutes(app, routing.RouteGroup{
		BaseUrl: "/api/compliance/configs",
		Routes: []routing.RouteDefinition{
			{Method: "GET", Path: "/templates", Handler: handlers.HandleGetConfigsTemplates, Input: nil},
			{Method: "POST", Path: "/placeholders", Handler: handlers.HandlePlaceholdersByFilenames, Input: handlers.GetPlaceholdersByFilenamesRequestInput{}},
		},
		Middlewares: []*hook.Handler[*core.RequestEvent]{apis.RequireAuth(), {Func: middlewares.ErrorHandlingMiddleware}},
		Validation:  true,
	})
}
