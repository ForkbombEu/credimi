// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

type GetMobileRunnerResponseSchema struct {
	RunnerURL string `json:"runner_url"`
	Serial    string `json:"serial"`
}

var MobileRunnersTemporalInternalRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/mobile-runner",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodGet,
			Path:           "",
			Handler:        HandleGetMobileRunner,
			ResponseSchema: GetMobileRunnerResponseSchema{},
		},
		{
			Method:         http.MethodGet,
			Path:           "/list-urls",
			Handler:        HandleListMobileRunnerURLs,
			ResponseSchema: ListMobileRunnersResponseSchema{},
		},
	},
}

func HandleGetMobileRunner() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		runnerIdentifier := e.Request.URL.Query().Get("runner_identifier")
		if runnerIdentifier == "" {
			return apierror.New(
				http.StatusBadRequest,
				"runner_identifier",
				"runner_identifier is required",
				"missing runner_identifier",
			).JSON(e)
		}
		record, err := canonify.Resolve(e.App, runnerIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"runner_identifier",
				"mobile runner not found",
				err.Error(),
			).JSON(e)
		}

		var response GetMobileRunnerResponseSchema
		response.Serial = record.GetString("serial")
		var port string
		if port = record.GetString("port"); port == "" {
			port = "8050"
		}
		response.RunnerURL = fmt.Sprintf("%s:%s", record.GetString("ip"), port)

		return e.JSON(http.StatusOK, response)
	}
}

type ListMobileRunnersResponseSchema struct {
	Runners []string `json:"runners"`
}

func HandleListMobileRunnerURLs() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		collection, err := e.App.FindCollectionByNameOrId("mobile_runners")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"collection",
				"mobile_runners collection not found",
				err.Error(),
			).JSON(e)
		}

		var records []*core.Record
		err = e.App.RecordQuery(collection).
			All(&records)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"records",
				"failed to fetch mobile runners",
				err.Error(),
			).JSON(e)
		}

		response := ListMobileRunnersResponseSchema{
			Runners: make([]string, 0, len(records)),
		}

		for _, record := range records {
			var port string
			if port = record.GetString("port"); port == "" {
				port = "8050"
			}

			runnerURL := fmt.Sprintf(
				"%s:%s",
				record.GetString("ip"),
				port,
			)

			response.Runners = append(response.Runners, runnerURL)
		}

		return e.JSON(http.StatusOK, response)
	}
}
