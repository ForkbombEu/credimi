// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/webpush"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var WebPushRoutes = routing.RouteGroup{
	BaseURL:                "/api/web-push",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodGet,
			Path:           "/vapid-public-key",
			Handler:        HandleGetWebPushVAPIDPublicKey,
			ResponseSchema: WebPushVAPIDPublicKeyResponse{},
			Description:    "Get the public VAPID key used for web push subscriptions",
		},
	},
}

type WebPushVAPIDPublicKeyResponse struct {
	PublicKey string `json:"public_key" validate:"required"`
}

func HandleGetWebPushVAPIDPublicKey() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		publicKey, _, err := webpush.GetVAPIDKeyPair(e.App)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"web_push",
				"failed to get VAPID public key",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, WebPushVAPIDPublicKeyResponse{PublicKey: publicKey})
	}
}
