// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
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

type ListMobileRunnersPublicResponseSchema struct {
	Runners []MobileRunnerListItem `json:"runners"`
}

type MobileRunnerListItem struct {
	Name        string                     `json:"name"`
	Path        string                     `json:"path"`
	URL         string                     `json:"url,omitempty"`
	Description string                     `json:"description,omitempty"`
	Type        string                     `json:"type,omitempty"`
	IsPublished bool                       `json:"is_published"`
	IsOwned     bool                       `json:"is_owned"`
	IsOnline    bool                       `json:"is_online"`
	Devices     []MobileRunnerHealthDevice `json:"devices,omitempty"`
	QueueLength *int                       `json:"queue_length,omitempty"`
}

type MobileRunnerHealthDevice struct {
	Serial      string `json:"serial,omitempty"`
	State       string `json:"state,omitempty"`
	Product     string `json:"product,omitempty"`
	Model       string `json:"model,omitempty"`
	Device      string `json:"device,omitempty"`
	TransportID string `json:"transport_id,omitempty"`
}

type mobileRunnerHealthResponse struct {
	Status  string                     `json:"status"`
	Devices []MobileRunnerHealthDevice `json:"devices,omitempty"`
}

var MobileRunnersPublicRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/mobile-runners",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodGet,
			Path:           "",
			OperationID:    "listMobileRunners",
			Handler:        HandleListMobileRunners,
			ResponseSchema: ListMobileRunnersPublicResponseSchema{},
			Summary:        "List available mobile runners",
			Description:    "Lists mobile runners visible to the caller, including health, devices, and queue length for online runners.",
			QuerySearchAttributes: []routing.QuerySearchAttribute{
				{
					Name:        "view",
					Required:    false,
					Description: "Optional response view. Use \"selector\" to return the lightweight runner selector shape and skip queue/device details.",
				},
			},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
	},
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

var checkMobileRunnerHealth = checkMobileRunnerHealthHTTP

func HandleListMobileRunners() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		callerOrgID := ""
		if e.Auth == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"auth",
				"authentication_required",
				"authentication is required",
			)
		}
		if !isSuperuserAuth(e.Auth) {
			orgID, err := pbutils.GetUserOrganizationID(e.App, e.Auth.Id)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"organization",
					"failed_to_find_user_organization",
					err.Error(),
				)
			}
			callerOrgID = orgID
		}

		records, err := listMobileRunnerRecords(e.App, callerOrgID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"mobile_runners",
				"failed_to_list_mobile_runners",
				err.Error(),
			)
		}

		response := ListMobileRunnersPublicResponseSchema{
			Runners: make([]MobileRunnerListItem, 0, len(records)),
		}
		includeDetails := e.Request.URL.Query().Get("view") != "selector"
		for _, record := range records {
			item, apiErr := mobileRunnerListItem(
				e.Request.Context(),
				e.App,
				record,
				callerOrgID,
				includeDetails,
			)
			if apiErr != nil {
				return apiErr
			}
			response.Runners = append(response.Runners, item)
		}

		sort.SliceStable(response.Runners, func(i, j int) bool {
			left := response.Runners[i]
			right := response.Runners[j]
			if left.IsOwned != right.IsOwned {
				return left.IsOwned
			}
			if left.IsOnline != right.IsOnline {
				return left.IsOnline
			}
			return left.Path < right.Path
		})

		return e.JSON(http.StatusOK, response)
	}
}

func listMobileRunnerRecords(app core.App, callerOrgID string) ([]*core.Record, error) {
	if callerOrgID == "" {
		return app.FindRecordsByFilter("mobile_runners", "", "name", -1, 0)
	}

	return app.FindRecordsByFilter(
		"mobile_runners",
		"owner = {:owner} || published = true",
		"name",
		-1,
		0,
		dbx.Params{"owner": callerOrgID},
	)
}

func mobileRunnerListItem(
	ctx context.Context,
	app core.App,
	record *core.Record,
	callerOrgID string,
	includeDetails bool,
) (MobileRunnerListItem, *apierror.APIError) {
	runnerID, err := mobileRunnerIdentifier(app, record)
	if err != nil {
		return MobileRunnerListItem{}, apierror.New(
			http.StatusInternalServerError,
			"runner_id",
			"failed_to_build_runner_id",
			err.Error(),
		)
	}

	runnerURL := mobileRunnerURL(record)
	online, devices, err := checkMobileRunnerHealth(ctx, runnerURL)
	if err != nil {
		return MobileRunnerListItem{}, apierror.New(
			http.StatusInternalServerError,
			"mobile_runner",
			"failed_to_check_runner_health",
			err.Error(),
		)
	}

	item := MobileRunnerListItem{
		Name:        record.GetString("name"),
		Path:        runnerID,
		Description: record.GetString("description"),
		IsPublished: record.GetBool("published"),
		IsOwned:     callerOrgID != "" && record.GetString("owner") == callerOrgID,
		IsOnline:    online,
	}

	if includeDetails {
		item.URL = runnerURL
		item.Type = record.GetString("type")
		item.Devices = devices
	}

	if includeDetails && online {
		queueLen, apiErr := mobileRunnerQueueLen(ctx, runnerID)
		if apiErr != nil {
			return MobileRunnerListItem{}, apiErr
		}
		item.QueueLength = &queueLen
	}

	return item, nil
}

func checkMobileRunnerHealthHTTP(
	ctx context.Context,
	runnerURL string,
) (bool, []MobileRunnerHealthDevice, error) {
	if strings.TrimSpace(runnerURL) == "" {
		return false, nil, nil
	}

	healthURL, err := url.JoinPath(runnerURL, "health")
	if err != nil {
		return false, nil, err
	}

	healthCtx, cancel := context.WithTimeout(ctx, walletAPKRunnerHealthTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(healthCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		return false, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil, nil
	}

	var health mobileRunnerHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return true, nil, nil
	}

	return true, health.Devices, nil
}

func mobileRunnerQueueLen(ctx context.Context, runnerID string) (int, *apierror.APIError) {
	state, err := queryMobileRunnerSemaphoreState(ctx, runnerID)
	if err != nil {
		if errors.Is(err, errSemaphoreNotFound) {
			return 0, nil
		}
		return 0, apierror.New(
			http.StatusInternalServerError,
			"mobile_runner",
			"failed_to_query_runner_queue",
			err.Error(),
		)
	}

	return state.QueueLen, nil
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
			)
		}
		record, err := canonify.Resolve(e.App, runnerIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"runner_identifier",
				"mobile runner not found",
				err.Error(),
			)
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
			)
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
			)
		}
		if apiErr := validatePipelineRunnerAccess(
			e.App,
			ownerRecord.Id,
			input.RunnerIDs,
		); apiErr != nil {
			return apiErr
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
			)
		}

		record, err := canonify.Resolve(e.App, runnerIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"runner_identifier",
				"mobile runner not found",
				err.Error(),
			)
		}

		runnerID := record.GetString("name")
		if runnerID == "" {
			return apierror.New(
				http.StatusNotFound,
				"runner_identifier",
				"mobile runner not found",
				"runner name missing",
			)
		}

		state, err := queryMobileRunnerSemaphoreState(e.Request.Context(), runnerID)
		if err != nil {
			if errors.Is(err, errSemaphoreNotFound) {
				return apierror.New(
					http.StatusNotFound,
					"semaphore",
					"runner semaphore not found",
					err.Error(),
				)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"semaphore",
				"failed to query runner semaphore",
				err.Error(),
			)
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
			)
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
			)
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
