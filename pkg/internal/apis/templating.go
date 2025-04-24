// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"reflect"

	"github.com/forkbombeu/didimo/pkg/internal/apis/handlers"
	"github.com/forkbombeu/didimo/pkg/internal/middlewares"
	"github.com/forkbombeu/didimo/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

func DefineTemplatingRoutes(app core.App) []routing.RouteDefinition {

	placeholdersByFilenamesInputType := reflect.TypeOf(handlers.GetPlaceholdersByFilenamesRequestInput{})
	
	return []routing.RouteDefinition{
		{Method: "GET", Path: "/templates", Handler: handlers.HandleGetConfigsTemplates(app), InputType: nil},
		{Method: "POST", Path: "/placeholders", Handler: handlers.HandlePlaceholdersByFilenames(app), InputType: placeholdersByFilenamesInputType},
	}
}

func AddTemplatingRoutes(app core.App) {
	templatingRoutes := DefineTemplatingRoutes(app)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		basePath := "/api/compliance/configs"

		g := se.Router.Group(basePath)
		g.Bind(apis.RequireAuth())
		g.Bind(&hook.Handler[*core.RequestEvent]{Func: middlewares.ErrorHandlingMiddleware})
		middlewares.RegisterRoutesWithValidation(app, g, templatingRoutes)

		return se.Next()
	})
}
