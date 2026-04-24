// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var PipelineRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/pipeline",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/queue",
			Handler:       HandlePipelineQueueEnqueue,
			RequestSchema: PipelineQueueInput{},
			Description:   "Queue a pipeline workflow for the runner semaphore",
		},
		{
			Method:      http.MethodGet,
			Path:        "/queue/{ticket}",
			Handler:     HandlePipelineQueueStatus,
			Description: "Get queued pipeline status by ticket",
		},
		{
			Method:      http.MethodDelete,
			Path:        "/queue/{ticket}",
			Handler:     HandlePipelineQueueCancel,
			Description: "Cancel a queued pipeline ticket",
		},
		{
			Method:  http.MethodGet,
			Path:    "/list-executions",
			Handler: HandleGetPipelineDetails,
		},
		{
			Method:  http.MethodGet,
			Path:    "/list-executions/{id}",
			Handler: HandleGetPipelineSpecificDetails,
		},
	},
}

var PipelineTemporalInternalRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/pipeline",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/get-yaml",
			Handler:     HandleGetPipelineYAML,
			Description: "Get a pipeline YAML from a pipeline ID",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:        http.MethodPost,
			Path:          "/pipeline-execution-results",
			Handler:       HandleSetPipelineExecutionResults,
			RequestSchema: PipelineResultInput{},
			Description:   "Create pipeline execution results record",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/scoreboard/{namespace}",
			Handler: HandleGetPipelineScoreboard,
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "/scoreboard/aggregate/start",
			Handler:        HandleStartAggregateScoreboard,
			ResponseSchema: StartAggregateScoreboardResponse{},
			Description:    "Start the aggregate scoreboard workflow (use ?schedule=300 per scheduling every N seconds)",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:      http.MethodDelete,
			Path:        "/scoreboard/aggregate/schedule/{schedule_id}",
			Handler:     HandleCancelAggregateScoreboardSchedule,
			Description: "Cancel a scheduled aggregate scoreboard workflow",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:      http.MethodGet,
			Path:        "/execution-details/{namespace}/{workflow_id}/{run_id}",
			Handler:     HandleGetExecutionDetails,
			Description: "Get detailed information about a specific execution",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:      http.MethodPost,
			Path:        "/scoreboard/save-results",
			Handler:     HandleSaveScoreboardResults,
			Description: "Refresh the aggregate scoreboard",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
		{
			Method:        http.MethodPost,
			Path:          "/retention/delete-files",
			Handler:       HandleDeletePipelineResultFiles,
			RequestSchema: DeletePipelineResultFilesRequest{},
			Description:   "Delete retained pipeline result files older than the requested number of days",
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminAPIKey(),
			},
		},
	},
}

var pipelineTemporalClient = temporalclient.GetTemporalClientWithNamespace
var pipelineListQueuedRuns = listQueuedPipelineRuns

const pipelineListWorkflowsDefaultLimit = 1000

func HandleGetPipelineYAML() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		pipelineIdentifier := e.Request.URL.Query().Get("pipeline_identifier")
		if pipelineIdentifier == "" {
			return apierror.New(
				http.StatusBadRequest,
				"pipeline_identifier",
				"pipeline_identifier is required",
				"missing pipeline_identifier",
			).JSON(e)
		}

		record, err := canonify.Resolve(e.App, pipelineIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"pipeline_identifier",
				"pipeline not found",
				err.Error(),
			).JSON(e)
		}
		yaml := record.GetString("yaml")
		return e.String(http.StatusOK, yaml)
	}
}

type PipelineResultInput struct {
	Owner      string `json:"owner"`
	PipelineID string `json:"pipeline_id"`
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

func HandleSetPipelineExecutionResults() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[PipelineResultInput](e)
		if err != nil {
			return err
		}

		pipeline, err := canonify.Resolve(e.App, input.PipelineID)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"pipeline",
				"pipeline not found",
				err.Error(),
			).JSON(e)
		}

		owner, err := canonify.Resolve(e.App, input.Owner)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"owner",
				"owner not found",
				err.Error(),
			).JSON(e)
		}

		coll, err := e.App.FindCollectionByNameOrId("pipeline_results")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"collection",
				"failed to get collection",
				err.Error(),
			).JSON(e)
		}

		existing, err := e.App.FindFirstRecordByFilter(
			coll,
			"workflow_id = {:workflow_id} && run_id = {:run_id}",
			dbx.Params{
				"workflow_id": input.WorkflowID,
				"run_id":      input.RunID,
			},
		)
		if err == nil {
			if existing.GetString("owner") == owner.Id &&
				existing.GetString("pipeline") == pipeline.Id {
				return e.JSON(http.StatusOK, existing.FieldsData())
			}
			return apierror.New(
				http.StatusConflict,
				"pipeline",
				"pipeline execution result already exists",
				"pipeline execution result owner or pipeline mismatch",
			).JSON(e)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return apierror.New(
				http.StatusInternalServerError,
				"pipeline",
				"failed to lookup pipeline execution result",
				err.Error(),
			).JSON(e)
		}

		record := core.NewRecord(coll)
		record.Set("owner", owner.Id)
		record.Set("pipeline", pipeline.Id)
		record.Set("workflow_id", input.WorkflowID)
		record.Set("run_id", input.RunID)
		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"pipeline",
				"failed to save pipeline record",
				err.Error(),
			).JSON(e)
		}
		return e.JSON(http.StatusOK, record.FieldsData())
	}
}

func HandleGetPipelineSpecificDetails() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"auth",
				"authentication required",
				"user not authenticated",
			).JSON(e)
		}
		pipelineID := e.Request.PathValue("id")
		if pipelineID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"pipeline",
				"pipeline ID is required",
				"missing pipeline ID in path parameter",
			).JSON(e)
		}

		organization, err := GetUserOrganization(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error(),
			).JSON(e)
		}
		namespace := organization.GetString("canonified_name")
		limit, pageNum := parsePaginationParams(e, 20, 0)
		offset := pageNum * limit
		statusFilter := e.Request.URL.Query().Get("status")
		pipelineRecords, err := e.App.FindRecordsByFilter(
			"pipelines",
			"(id = {:id} && owner={:owner}) || (id = {:id} && published={:published})",
			"",
			-1,
			0,
			dbx.Params{
				"id":        pipelineID,
				"owner":     organization.Id,
				"published": true,
			},
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"pipelines",
				"failed to fetch pipelines",
				err.Error(),
			).JSON(e)
		}

		if len(pipelineRecords) == 0 {
			return e.JSON(http.StatusOK, []*pipelineWorkflowSummary{})
		}

		pipelineRecord := pipelineRecords[0]
		pipelinePath, err := canonify.BuildPath(
			e.App,
			pipelineRecord,
			canonify.CanonifyPaths["pipelines"],
			"",
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"pipeline",
				"failed to build pipeline identifier",
				err.Error(),
			).JSON(e)
		}
		pipelineIdentifier := strings.Trim(pipelinePath, "/")

		includeQueued := shouldIncludeQueuedPipelineExecutions(statusFilter)
		if includeQueued {
			queuedRuns, err := pipelineListQueuedRuns(e.Request.Context(), namespace)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to list queued runs",
					err.Error(),
				).JSON(e)
			}

			queuedByPipelineID := mapQueuedRunsToPipelines(
				e.App,
				[]*core.Record{pipelineRecord},
				queuedRuns,
			)
			queuedForPipeline := queuedByPipelineID[pipelineRecord.Id]

			if queuedOnlyPipelineExecutions(statusFilter) {
				return e.JSON(http.StatusOK, paginateQueuedPipelineSummaries(
					e.App,
					queuedForPipeline,
					authRecord.GetString("Timezone"),
					limit,
					offset,
				))
			}

			queuedCount := len(queuedForPipeline)
			if offset < queuedCount {
				queuedSummaries := paginateQueuedPipelineSummaries(
					e.App,
					queuedForPipeline,
					authRecord.GetString("Timezone"),
					limit,
					offset,
				)
				if len(queuedSummaries) >= limit {
					return e.JSON(http.StatusOK, queuedSummaries)
				}

				completedSummaries, apiErr := fetchCompletedWorkflowsWithPagination(
					e,
					map[string]*core.Record{pipelineRecord.Id: pipelineRecord},
					namespace,
					authRecord,
					organization.Id,
					pipelineIdentifier,
					statusFilter,
					limit-len(queuedSummaries),
					0,
					nil,
				)
				if apiErr != nil {
					return apiErr.JSON(e)
				}
				return e.JSON(http.StatusOK, append(queuedSummaries, completedSummaries...))
			}

			offset -= queuedCount
		}

		temporalClient, err := pipelineTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create temporal client",
				err.Error(),
			).JSON(e)
		}
		summaries, apiErr := fetchCompletedWorkflowsWithPagination(
			e,
			map[string]*core.Record{pipelineRecord.Id: pipelineRecord},
			namespace,
			authRecord,
			organization.Id,
			pipelineIdentifier,
			statusFilter,
			limit,
			offset,
			temporalClient,
		)
		if apiErr != nil {
			return apiErr.JSON(e)
		}

		return e.JSON(http.StatusOK, summaries)
	}
}

func shouldIncludeQueuedPipelineExecutions(statusFilter string) bool {
	if strings.TrimSpace(statusFilter) == "" {
		return true
	}

	for _, raw := range strings.Split(statusFilter, ",") {
		if strings.EqualFold(strings.TrimSpace(raw), statusStringQueued) {
			return true
		}
	}

	return false
}

func queuedOnlyPipelineExecutions(statusFilter string) bool {
	if strings.TrimSpace(statusFilter) == "" {
		return false
	}

	hasQueued := false
	hasNonQueued := false
	for _, raw := range strings.Split(statusFilter, ",") {
		normalized := strings.ToLower(strings.TrimSpace(raw))
		if normalized == "" {
			continue
		}
		if normalized == statusStringQueued {
			hasQueued = true
			continue
		}
		hasNonQueued = true
	}

	return hasQueued && !hasNonQueued
}

func paginateQueuedPipelineSummaries(
	app core.App,
	queuedRuns []QueuedPipelineRunAggregate,
	userTimezone string,
	limit int,
	offset int,
) []*pipelineWorkflowSummary {
	summaries := buildQueuedPipelineSummaries(
		app,
		queuedRuns,
		userTimezone,
		map[string]map[string]any{},
	)
	if len(summaries) == 0 || limit <= 0 || offset >= len(summaries) {
		return []*pipelineWorkflowSummary{}
	}
	if offset < 0 {
		offset = 0
	}

	end := offset + limit
	if end > len(summaries) {
		end = len(summaries)
	}

	return summaries[offset:end]
}

func HandleGetPipelineDetails() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"auth",
				"authentication required",
				"user not authenticated",
			).JSON(e)
		}

		organization, err := GetUserOrganization(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error(),
			).JSON(e)
		}
		namespace := organization.GetString("canonified_name")

		pipelineRecords, err := e.App.FindRecordsByFilter(
			"pipelines",
			"owner={:owner} || published={:published}",
			"",
			-1,
			0,
			dbx.Params{
				"owner":     organization.Id,
				"published": true,
			},
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"pipelines",
				"failed to fetch pipelines",
				err.Error(),
			).JSON(e)
		}

		if len(pipelineRecords) == 0 {
			return e.JSON(http.StatusOK, map[string][]*WorkflowExecutionSummary{})
		}

		queuedRuns, err := pipelineListQueuedRuns(e.Request.Context(), namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to list queued runs",
				err.Error(),
			).JSON(e)
		}
		queuedByPipelineID := mapQueuedRunsToPipelines(e.App, pipelineRecords, queuedRuns)

		pipelineMap := make(map[string]*core.Record, len(pipelineRecords))
		pipelineRunnerInfoMap := make(map[string]pipelineRunnerInfo, len(pipelineRecords))
		for _, pipelineRecord := range pipelineRecords {
			pipelineID := pipelineRecord.Id
			pipelineMap[pipelineID] = pipelineRecord

			runnerInfo, err := pipeline.ParsePipelineRunnerInfo(pipelineRecord.GetString("yaml"))
			if err != nil {
				e.App.Logger().Warn(fmt.Sprintf(
					"failed to parse pipeline yaml for runners (pipeline_id=%s): %v",
					pipelineID,
					err,
				))
			}
			pipelineRunnerInfoMap[pipelineID] = runnerInfo
		}

		pipelineIdentifierIndex := buildPipelineIdentifierIndex(e.App, pipelineMap)
		pipelineIdentifierByID := buildPipelineIdentifierByID(e.App, pipelineMap)

		temporalClient, err := pipelineTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create temporal client",
				err.Error(),
			).JSON(e)
		}

		executions, err := listPipelineWorkflowExecutions(
			context.Background(),
			temporalClient,
			namespace,
			nil,
			"",
			pipelineListWorkflowsDefaultLimit,
			0,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to list workflows",
				err.Error(),
			).JSON(e)
		}

		pipelineIdentifiers := resolvePipelineIdentifiersForExecutions(executions)

		pipelineExecutionsByID := make(map[string][]*WorkflowExecution)
		for _, exec := range executions {
			if exec == nil || exec.Execution == nil {
				continue
			}
			ref := workflowExecutionRef{
				WorkflowID: exec.Execution.WorkflowID,
				RunID:      exec.Execution.RunID,
			}
			pipelineIdentifier := pipelineIdentifiers[ref]
			if pipelineIdentifier == "" {
				continue
			}
			pipelineRecord := pipelineIdentifierIndex[pipelineIdentifier]
			if pipelineRecord == nil {
				continue
			}
			pipelineExecutionsByID[pipelineRecord.Id] = append(
				pipelineExecutionsByID[pipelineRecord.Id],
				exec,
			)
		}

		pipelineExecutionsMap := make(map[string][]*WorkflowExecutionSummary)
		for pipelineID, pipelineExecutions := range pipelineExecutionsByID {
			if len(pipelineExecutions) == 0 {
				continue
			}
			hierarchy := buildExecutionHierarchyRaw(
				e.App,
				pipelineExecutions,
				namespace,
				temporalClient,
			)
			if len(hierarchy) > 0 {
				pipelineExecutionsMap[pipelineID] = hierarchy
			}
		}

		if len(pipelineExecutionsMap) == 0 && len(queuedByPipelineID) == 0 {
			return e.JSON(http.StatusOK, map[string][]*WorkflowExecutionSummary{})
		}

		var allExecutions []struct {
			pipelineID string
			execution  *WorkflowExecutionSummary
		}

		for pipelineID, executions := range pipelineExecutionsMap {
			for _, exec := range executions {
				allExecutions = append(allExecutions, struct {
					pipelineID string
					execution  *WorkflowExecutionSummary
				}{
					pipelineID: pipelineID,
					execution:  exec,
				})
			}
		}

		selectedExecutions := selectTopExecutionsByPipeline(allExecutions, 5)

		response := make(map[string][]*pipelineWorkflowSummary, len(selectedExecutions))
		runnerCache := map[string]map[string]any{}
		loc, err := time.LoadLocation(authRecord.GetString("Timezone"))
		if err != nil {
			loc = time.Local
		}

		for pipelineID, executions := range selectedExecutions {
			info := pipelineRunnerInfoMap[pipelineID]
			annotated, err := attachRunnerInfoFromTemporalStartInput(
				attachRunnerInfoFromTemporalInputArgs{
					App:         e.App,
					Ctx:         context.Background(),
					Client:      temporalClient,
					Executions:  executions,
					Info:        info,
					RunnerCache: runnerCache,
				},
			)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"temporal",
					"failed to read workflow history",
					err.Error(),
				).JSON(e)
			}
			pipelineIdentifier := pipelineIdentifierByID[pipelineID]
			for _, summary := range annotated {
				summary.PipelineIdentifier = pipelineIdentifier
			}
			localizePipelineWorkflowSummaries(annotated, loc)
			response[pipelineID] = annotated
		}

		appendQueuedPipelineSummaries(
			e.App,
			response,
			queuedByPipelineID,
			authRecord.GetString("Timezone"),
			runnerCache,
		)

		return e.JSON(http.StatusOK, response)
	}
}

func selectTopExecutionsByPipeline(executions []struct {
	pipelineID string
	execution  *WorkflowExecutionSummary
}, limit int) map[string][]*WorkflowExecutionSummary {
	pipelineExecutions := make(map[string][]struct {
		pipelineID string
		execution  *WorkflowExecutionSummary
	})

	for _, exec := range executions {
		pipelineExecutions[exec.pipelineID] = append(pipelineExecutions[exec.pipelineID], exec)
	}

	result := make(map[string][]*WorkflowExecutionSummary)

	for pipelineID, execs := range pipelineExecutions {
		var runningExecs, otherExecs []*WorkflowExecutionSummary

		for _, exec := range execs {
			if exec.execution.Status == string(WorkflowStatusRunning) {
				runningExecs = append(runningExecs, exec.execution)
			} else {
				otherExecs = append(otherExecs, exec.execution)
			}
		}

		selected := runningExecs

		remainingSlots := limit - len(runningExecs)
		if remainingSlots > 0 && len(otherExecs) > 0 {
			sort.Slice(otherExecs, func(i, j int) bool {
				return utils.TimeStringAfter(otherExecs[i].StartTime, otherExecs[j].StartTime)
			})

			if remainingSlots > len(otherExecs) {
				remainingSlots = len(otherExecs)
			}

			selected = append(selected, otherExecs[:remainingSlots]...)
		}

		sort.Slice(selected, func(i, j int) bool {
			return utils.TimeStringAfter(selected[i].StartTime, selected[j].StartTime)
		})

		if len(selected) > 0 {
			result[pipelineID] = selected
		}
	}

	return result
}
