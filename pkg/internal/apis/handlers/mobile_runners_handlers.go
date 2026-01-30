// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/api/serviceerror"
)

type GetMobileRunnerResponseSchema struct {
	RunnerURL string `json:"runner_url"`
	Serial    string `json:"serial"`
}

type MobileRunnerSemaphoreResponseSchema struct {
	RunnerID    string                               `json:"runner_id"`
	Capacity    int                                  `json:"capacity"`
	InUse       bool                                 `json:"in_use"`
	Holder      *workflows.MobileRunnerSemaphoreHolder `json:"holder,omitempty"`
	QueueLen    int                                  `json:"queue_len"`
	LastGrantAt *time.Time                           `json:"last_grant_at,omitempty"`
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
			Path:           "/semaphore",
			Handler:        HandleGetMobileRunnerSemaphore,
			ResponseSchema: MobileRunnerSemaphoreResponseSchema{},
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
		url := record.GetString("ip")
		if port = record.GetString("port"); port != "" {
			url = fmt.Sprintf("%s:%s", url, port)
		}
		response.RunnerURL = url

		return e.JSON(http.StatusOK, response)
	}
}

var errSemaphoreNotFound = errors.New("semaphore not found")

var queryMobileRunnerSemaphoreState = queryMobileRunnerSemaphoreStateTemporal

func HandleGetMobileRunnerSemaphore() func(*core.RequestEvent) error {
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

		runnerID := record.GetString("name")
		if runnerID == "" {
			return apierror.New(
				http.StatusNotFound,
				"runner_identifier",
				"mobile runner not found",
				"runner name missing",
			).JSON(e)
		}

		state, err := queryMobileRunnerSemaphoreState(e.Request.Context(), runnerID)
		if err != nil {
			if errors.Is(err, errSemaphoreNotFound) {
				return apierror.New(
					http.StatusNotFound,
					"semaphore",
					"runner semaphore not found",
					err.Error(),
				).JSON(e)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"semaphore",
				"failed to query runner semaphore",
				err.Error(),
			).JSON(e)
		}

		response := MobileRunnerSemaphoreResponseSchema{
			RunnerID:    state.RunnerID,
			Capacity:    state.Capacity,
			InUse:       len(state.Holders) > 0,
			Holder:      state.CurrentHolder,
			QueueLen:    state.QueueLen,
			LastGrantAt: state.LastGrantAt,
		}

		return e.JSON(http.StatusOK, response)
	}
}

func queryMobileRunnerSemaphoreStateTemporal(
	ctx context.Context,
	runnerID string,
) (workflows.MobileRunnerSemaphoreStateView, error) {
	client, err := temporalclient.GetTemporalClientWithNamespace(defaultMobileRunnerSemaphoreNamespace)
	if err != nil {
		return workflows.MobileRunnerSemaphoreStateView{}, err
	}

	workflowID := workflows.MobileRunnerSemaphoreWorkflowID(runnerID)
	encoded, err := client.QueryWorkflow(
		ctx,
		workflowID,
		"",
		workflows.MobileRunnerSemaphoreStateQuery,
	)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return workflows.MobileRunnerSemaphoreStateView{}, errSemaphoreNotFound
		}
		return workflows.MobileRunnerSemaphoreStateView{}, err
	}

	var state workflows.MobileRunnerSemaphoreStateView
	if err := encoded.Get(&state); err != nil {
		return workflows.MobileRunnerSemaphoreStateView{}, err
	}

	return state, nil
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
			runnerURL := record.GetString("ip")
			if port = record.GetString("port"); port != "" {
				runnerURL = fmt.Sprintf("%s:%s", runnerURL, port)
			}

			response.Runners = append(response.Runners, runnerURL)
		}

		return e.JSON(http.StatusOK, response)
	}
}
