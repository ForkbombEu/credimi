// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/api/serviceerror"
)

type GetMobileRunnerResponseSchema struct {
	Type      string `json:"type"`
	RunnerURL string `json:"runner_url"`
	Serial    string `json:"serial"`
}

type MobileRunnerSemaphoreResponseSchema struct {
	RunnerID  string `json:"runner_id"`
	Capacity  int    `json:"capacity"`
	SlotsUsed int    `json:"slots_used"`
	InUse     bool   `json:"in_use"`
	QueueLen  int    `json:"queue_len"`
}

type ValidateMobileRunnerAccessRequest struct {
	OwnerNamespace string   `json:"owner_namespace"`
	RunnerIDs      []string `json:"runner_ids"`
}

var MobileRunnersTemporalInternalRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/mobile-runner",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		middlewares.RequireInternalAdminAPIKey(),
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
		{
			Method:        http.MethodPost,
			Path:          "/validate-access",
			Handler:       HandleValidateMobileRunnerAccess,
			RequestSchema: ValidateMobileRunnerAccessRequest{},
			Description:   "Validate that runner IDs are accessible to an owner namespace",
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
		response.Type = record.GetString("type")
		response.Serial = record.GetString("serial")
		response.RunnerURL = mobileRunnerURL(record)

		return e.JSON(http.StatusOK, response)
	}
}

func HandleValidateMobileRunnerAccess() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[ValidateMobileRunnerAccessRequest](e)
		if err != nil {
			return err
		}
		ownerNamespace := strings.TrimSpace(input.OwnerNamespace)
		if ownerNamespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"owner_namespace",
				"owner_namespace is required",
				"missing owner_namespace",
			).JSON(e)
		}

		ownerRecord, err := e.App.FindFirstRecordByFilter(
			"organizations",
			"canonified_name = {:namespace}",
			map[string]any{"namespace": ownerNamespace},
		)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"owner_namespace",
				"owner namespace not found",
				err.Error(),
			).JSON(e)
		}
		if apiErr := validatePipelineRunnerAccess(e.App, ownerRecord.Id, input.RunnerIDs); apiErr != nil {
			return apiErr.JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{"valid": true})
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
			RunnerID:  state.RunnerID,
			Capacity:  state.Capacity,
			SlotsUsed: state.SlotsUsed,
			InUse:     state.SlotsUsed > 0,
			QueueLen:  state.QueueLen,
		}

		return e.JSON(http.StatusOK, response)
	}
}

func queryMobileRunnerSemaphoreStateTemporal(
	ctx context.Context,
	runnerID string,
) (workflows.MobileRunnerSemaphoreStateView, error) {
	client, err := temporalclient.GetTemporalClientWithNamespace(
		workflowengine.MobileRunnerSemaphoreDefaultNamespace,
	)
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
			response.Runners = append(response.Runners, mobileRunnerURL(record))
		}

		return e.JSON(http.StatusOK, response)
	}
}

func mobileRunnerURL(record *core.Record) string {
	runnerURL := strings.TrimSpace(record.GetString("ip"))
	if runnerURL == "" {
		return ""
	}
	if port := strings.TrimSpace(record.GetString("port")); port != "" {
		runnerURL = fmt.Sprintf("%s:%s", strings.TrimRight(runnerURL, "/"), port)
	}

	return runnerURL
}

func mobileRunnerIdentifier(app core.App, record *core.Record) (string, error) {
	runnerID, err := canonify.BuildPath(
		app,
		record,
		canonify.CanonifyPaths["mobile_runners"],
		"",
	)
	if err != nil {
		return "", err
	}

	return canonify.NormalizePath(runnerID), nil
}
