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

var RouteGroupsNotExported []routing.RouteGroup = []routing.RouteGroup{
	handlers.ConformanceRoutes,
	handlers.TemplateRoutes,
	handlers.IssuersRoutes,
	handlers.IssuerTemporalInternalRoutes,
	handlers.WalletRoutes,
	handlers.DeepLinkRoutes,
	handlers.PipelineRoutes,
	handlers.WorkflowsRoutes,
}

func RegisterMyRoutes(app core.App) {
	for _, group := range RouteGroups {
		group.Add(app)
	}
	for _, group := range RouteGroupsNotExported {
		group.Add(app)
	}
}
