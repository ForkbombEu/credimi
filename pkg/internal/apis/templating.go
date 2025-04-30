// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	// "github.com/pocketbase/pocketbase/apis"
	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

func AddTemplatingRoutes(app core.App) {
	routing.AddGroupRoutes(app, routing.RouteGroup{
		BaseURL: "/api/template",
		Routes: []routing.RouteDefinition{
			{
				Method:  "GET",
				Path:    "/blueprints",
				Handler: handlers.HandleGetConfigsTemplates,
				Input:   nil,
			},
			{
				Method:  "POST",
				Path:    "/placeholders",
				Handler: handlers.HandlePlaceholdersByFilenames,
				Input:   handlers.GetPlaceholdersByFilenamesRequestInput{},
			},
		},
		Middlewares: []*hook.Handler[*core.RequestEvent]{
			// TODO: uncomment when new configs templates feature is ready
			// apis.RequireAuth(),
			{Func: middlewares.ErrorHandlingMiddleware},
		},
		Validation: true,
	})
}
