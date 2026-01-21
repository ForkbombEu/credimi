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
	handlers.SchedulesRoutes,
	//handlers.ScoreboardRoutes,
}

var RouteGroupsNotExported []routing.RouteGroup = []routing.RouteGroup{
	handlers.ConformanceRoutes,
	handlers.TemplateRoutes,
	handlers.IssuersRoutes,
	handlers.IssuerTemporalInternalRoutes,
	handlers.CredentialTemporalInternalRoutes,
	handlers.WalletRoutes,
	handlers.WalletTemporalInternalRoutes,
	handlers.VerifierTemporalInternalRoutes,
	handlers.DeepLinkRoutes,
	handlers.PipelineRoutes,
	handlers.PipelineTemporalInternalRoutes,
	handlers.CanonifyRoutes,
	handlers.DeepLinkCredential,
	handlers.DeepLinkVerifiers,
	handlers.ConformanceCheckRoutes,
	handlers.OrganizationRoutes,
	//handlers.ScoreboardPublicRoutes,
	handlers.CloneRecord,
}

func RegisterMyRoutes(app core.App) {
	for _, group := range RouteGroups {
		group.Add(app)
	}
	for _, group := range RouteGroupsNotExported {
		group.Add(app)
	}
}
