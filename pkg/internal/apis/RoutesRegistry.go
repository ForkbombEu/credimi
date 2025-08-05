// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later


package apis

import (
	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
)

var RouteGroups []routing.RouteGroup = []routing.RouteGroup{
	handlers.ChecksRoutes,
	handlers.ApiKeyRoutes,
	
}

var RouteGruoupsNotExported []routing.RouteGroup = []routing.RouteGroup{
	handlers.ConformanceRoutes,
	handlers.TemplateRoutes,
}

func RegisterMyRoutes(app core.App) {
	for _, group := range RouteGroups {
		group.Add(app)
	}
	for _, group := range RouteGruoupsNotExported {
		group.Add(app)
	}
}
