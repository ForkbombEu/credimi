// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// HookRefuseAPIKeyOAuth2Linking refuses auth-with-oauth2 on every auth
// collection when the request's Authorization token was exchanged for an API
// key (handlers.RequestHasAPIKeyToken): PocketBase links the provider account
// to the signed-in record and returns a fresh token without the api_key
// claim, which would launder a leaked key into a browser session.
func HookRefuseAPIKeyOAuth2Linking(app core.App) {
	app.OnRecordAuthWithOAuth2Request().BindFunc(
		func(e *core.RecordAuthWithOAuth2RequestEvent) error {
			if handlers.RequestHasAPIKeyToken(e.Request) {
				return apis.NewForbiddenError(
					"OAuth2 sign-in is not allowed with a token exchanged for an API key",
					nil,
				)
			}
			return e.Next()
		},
	)
}
