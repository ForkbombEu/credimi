// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var DeepLinkCredential routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:  http.MethodPost,
			Path:    "/credential/deeplink",
			Handler: HandleGetCredentialDeeplink,
		},
	},
}

func HandleGetCredentialDeeplink() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.URL.Query().Get("id")
		if id == "" {
			return apierror.New(
				http.StatusBadRequest,
				"request",
				"missing credential id",
				"id parameter is required",
			).JSON(e)
		}

		rec, err := canonify.Resolve(e.App, id)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"resolve",
				"failed to resolve credential path",
				err.Error(),
			).JSON(e)
		}

		deeplink, ok := rec.Get("deeplink").(string)
		if !ok || deeplink == "" {
			return apierror.New(
				http.StatusInternalServerError,
				"credential",
				"deeplink not found",
				"field 'deeplink' is missing or empty",
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"deeplink": deeplink,
		})
	}
}
