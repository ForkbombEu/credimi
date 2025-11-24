// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var DeepLinkVerifiers routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:  http.MethodGet,
			Path:    "/verification/deeplink",
			Handler: HandleVerificationDeeplink,
		},
	},
}

func HandleVerificationDeeplink() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.URL.Query().Get("id")
		if id == "" {
			return apierror.New(
				http.StatusBadRequest,
				"request",
				"missing record id",
				"id parameter is required",
			).JSON(e)
		}

		rec, err := canonify.Resolve(e.App, id)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"resolve",
				"failed to resolve verification path",
				err.Error(),
			).JSON(e)
		}

		yamlStr := rec.GetString("yaml")
		if yamlStr == "" {
			return apierror.New(
				http.StatusInternalServerError,
				"yaml",
				"yaml not found in record",
				"field 'yaml' is missing or empty",
			).JSON(e)
		}

		bodyData, err := json.Marshal(map[string]string{"yaml": yamlStr})
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"marshal",
				"failed to encode yaml body",
				err.Error(),
			).JSON(e)
		}

		baseURL := e.App.Settings().Meta.AppURL
		url := utils.JoinURL(baseURL, "api", "get-deeplink")

		ctx := e.Request.Context()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyData))
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"request",
				"failed to create request to /api/get-deeplink",
				err.Error(),
			).JSON(e)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"request",
				"failed to call internal /api/get-deeplink endpoint",
				err.Error(),
			).JSON(e)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return apierror.New(
				resp.StatusCode,
				"get-deeplink",
				"internal endpoint returned an error",
				string(respBody),
			).JSON(e)
		}

		var result map[string]any
		if err := json.Unmarshal(respBody, &result); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"json",
				"failed to parse /api/get-deeplink response",
				err.Error(),
			).JSON(e)
		}

		deeplink, ok := result["deeplink"].(string)
		if !ok || deeplink == "" {
			return apierror.New(
				http.StatusInternalServerError,
				"deeplink",
				"deeplink missing in response",
				"field 'deeplink' is not present or empty",
			).JSON(e)
		}

		return e.String(http.StatusOK, deeplink)
	}
}
