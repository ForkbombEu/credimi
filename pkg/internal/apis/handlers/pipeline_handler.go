// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
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
	"github.com/forkbombeu/credimi/pkg/internal/runners"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/encoding/protojson"
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
			Path:    "/list-workflows",
			Handler: HandleGetPipelineDetails,
		},
		{
			Method:  http.MethodGet,
			Path:    "/list-workflows/{id}",
			Handler: HandleGetPipelineSpecificDetails,
		},
		{
			Method:  http.MethodGet,
			Path:    "/list-results",
			Handler: HandleGetPipelineResults,
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
		},
		{
			Method:        http.MethodPost,
			Path:          "/pipeline-execution-results",
			Handler:       HandleSetPipelineExecutionResults,
			RequestSchema: PipelineResultInput{},
			Description:   "Create pipeline execution results record",
		},
	},
}

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
		pipelineRecords, err := e.App.FindRecordsByFilter(
			"pipelines",
			"id = {:id} && owner={:owner}",
			"",
			-1,
			0,
			dbx.Params{
				"id":    pipelineID,
				"owner": organization.Id,
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
			return e.JSON(http.StatusOK, ListMyChecksResponse{
				Executions: []*WorkflowExecutionSummary{},
			})
		}

		var allExecutions []*WorkflowExecution
		var temporalClient client.Client

		pipelineRecord := pipelineRecords[0]
		pipelineID = pipelineRecord.Id
		queuedRuns, err := listQueuedPipelineRuns(e.Request.Context(), namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to list queued runs",
				err.Error(),
			).JSON(e)
		}
		queuedByPipelineID := mapQueuedRunsToPipelines(e.App, pipelineRecords, queuedRuns)
		queuedForPipeline := queuedByPipelineID[pipelineID]

		runnerInfo, err := runners.ParsePipelineRunnerInfo(pipelineRecord.GetString("yaml"))
		if err != nil {
			e.App.Logger().Warn(fmt.Sprintf(
				"failed to parse pipeline yaml for runners (pipeline_id=%s): %v",
				pipelineID,
				err,
			))
		}

		resultsRecords, err := e.App.FindRecordsByFilter(
			"pipeline_results",
			"pipeline={:pipeline} && owner={:owner}",
			"",
			-1,
			0,
			dbx.Params{
				"pipeline": pipelineID,
				"owner":    organization.Id,
			},
		)

		if err != nil {
			return e.JSON(http.StatusOK, ListMyChecksResponse{
				[]*WorkflowExecutionSummary{},
			})
		}

		allExecutions, err = processPipelineResults(namespace, resultsRecords, &temporalClient)
		if err != nil {
			return e.JSON(http.StatusOK, allExecutions)
		}

		if len(allExecutions) == 0 {
			queuedSummaries := buildQueuedPipelineSummaries(
				e.App,
				queuedForPipeline,
				authRecord.GetString("Timezone"),
				map[string]map[string]any{},
			)
			if len(queuedSummaries) == 0 {
				return e.JSON(http.StatusOK, ListMyChecksResponse{
					[]*WorkflowExecutionSummary{},
				})
			}
			return e.JSON(http.StatusOK, queuedSummaries)
		}

		if temporalClient == nil {
			temporalClient, err = temporalclient.GetTemporalClientWithNamespace(namespace)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"temporal",
					"unable to create temporal client",
					err.Error(),
				).JSON(e)
			}
		}

		hierarchy := buildExecutionHierarchy(
			e.App,
			allExecutions,
			namespace,
			authRecord.GetString("Timezone"),
			temporalClient,
		)

		sort.Slice(hierarchy, func(i, j int) bool {
			t1, _ := time.Parse(time.RFC3339, hierarchy[i].StartTime)
			t2, _ := time.Parse(time.RFC3339, hierarchy[j].StartTime)

			return t1.After(t2)
		})

		runnerCache := map[string]map[string]any{}
		annotated, err := attachRunnerInfoFromTemporalStartInput(
			attachRunnerInfoFromTemporalInputArgs{
				App:         e.App,
				Ctx:         context.Background(),
				Client:      temporalClient,
				Executions:  hierarchy,
				Info:        runnerInfo,
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

		if len(queuedForPipeline) > 0 {
			queuedSummaries := buildQueuedPipelineSummaries(
				e.App,
				queuedForPipeline,
				authRecord.GetString("Timezone"),
				runnerCache,
			)
			if len(queuedSummaries) > 0 {
				annotated = append(queuedSummaries, annotated...)
			}
		}

		return e.JSON(http.StatusOK, annotated)
	}
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

		queuedRuns, err := listQueuedPipelineRuns(e.Request.Context(), namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to list queued runs",
				err.Error(),
			).JSON(e)
		}
		queuedByPipelineID := mapQueuedRunsToPipelines(e.App, pipelineRecords, queuedRuns)

		pipelineExecutionsMap := make(map[string][]*WorkflowExecutionSummary)
		pipelineRunnerInfoMap := make(map[string]pipelineRunnerInfo)

		for _, pipelineRecord := range pipelineRecords {
			pipelineID := pipelineRecord.Id

			runnerInfo, err := runners.ParsePipelineRunnerInfo(pipelineRecord.GetString("yaml"))
			if err != nil {
				e.App.Logger().Warn(fmt.Sprintf(
					"failed to parse pipeline yaml for runners (pipeline_id=%s): %v",
					pipelineID,
					err,
				))
			}
			pipelineRunnerInfoMap[pipelineID] = runnerInfo

			resultsRecords, err := e.App.FindRecordsByFilter(
				"pipeline_results",
				"pipeline={:pipeline} && owner={:owner}",
				"",
				-1,
				0,
				dbx.Params{
					"pipeline": pipelineID,
					"owner":    organization.Id,
				},
			)

			if err == nil && len(resultsRecords) > 0 {
				var pipelineExecutions []*WorkflowExecution

				for _, resultRecord := range resultsRecords {
					c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
					if err != nil {
						return apierror.New(
							http.StatusInternalServerError,
							"temporal",
							"unable to create client",
							err.Error(),
						).JSON(e)
					}

					workflowExecution, err := c.DescribeWorkflowExecution(
						context.Background(),
						resultRecord.GetString("workflow_id"),
						resultRecord.GetString("run_id"),
					)
					if err != nil {
						notFound := &serviceerror.NotFound{}
						if errors.As(err, &notFound) {
							return apierror.New(
								http.StatusNotFound,
								"workflow",
								"workflow not found",
								err.Error(),
							).JSON(e)
						}
						invalidArgument := &serviceerror.InvalidArgument{}
						if errors.As(err, &invalidArgument) {
							return apierror.New(
								http.StatusBadRequest,
								"workflow",
								"invalid workflow ID",
								err.Error(),
							).JSON(e)
						}
						return apierror.New(
							http.StatusInternalServerError,
							"workflow",
							"failed to describe workflow execution",
							err.Error(),
						).JSON(e)
					}

					weJSON, err := protojson.Marshal(workflowExecution.GetWorkflowExecutionInfo())
					if err != nil {
						return apierror.New(
							http.StatusInternalServerError,
							"workflow",
							"failed to marshal workflow execution",
							err.Error(),
						).JSON(e)
					}

					var execInfo WorkflowExecution
					err = json.Unmarshal(weJSON, &execInfo)
					if err != nil {
						return apierror.New(
							http.StatusInternalServerError,
							"workflow",
							"failed to unmarshal workflow execution",
							err.Error(),
						).JSON(e)
					}

					if workflowExecution.GetWorkflowExecutionInfo().GetParentExecution() != nil {
						parentJSON, _ := protojson.Marshal(
							workflowExecution.GetWorkflowExecutionInfo().GetParentExecution(),
						)
						var parentInfo WorkflowIdentifier
						_ = json.Unmarshal(parentJSON, &parentInfo)
						execInfo.ParentExecution = &parentInfo
					}

					pipelineExecutions = append(pipelineExecutions, &execInfo)
				}

				if len(pipelineExecutions) > 0 {
					c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
					if err != nil {
						continue
					}

					hierarchy := buildExecutionHierarchy(
						e.App,
						pipelineExecutions,
						namespace,
						authRecord.GetString("Timezone"),
						c,
					)

					pipelineExecutionsMap[pipelineID] = hierarchy
				}
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

		tc, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			// fallback: if no temporal client, return without global_runner_id
			for pipelineID, executions := range selectedExecutions {
				info := pipelineRunnerInfoMap[pipelineID]
				response[pipelineID] = attachPipelineRunnerInfo(
					e.App,
					executions,
					"",
					info,
					runnerCache,
				)
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

		for pipelineID, executions := range selectedExecutions {
			info := pipelineRunnerInfoMap[pipelineID]
			annotated, err := attachRunnerInfoFromTemporalStartInput(
				attachRunnerInfoFromTemporalInputArgs{
					App:         e.App,
					Ctx:         context.Background(),
					Client:      tc,
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

func HandleGetPipelineResults() func(*core.RequestEvent) error {
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

		status := e.Request.URL.Query().Get("status")

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
			"published = {:published} || owner={:owner}",
			"",
			30,
			0,
			dbx.Params{
				"published": true,
				"owner":     organization.Id,
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

		pipelineMap := make(map[string]*core.Record)
		for _, p := range pipelineRecords {
			pipelineMap[p.Id] = p
		}

		var temporalClient client.Client

		queuedRuns, err := listQueuedPipelineRuns(e.Request.Context(), namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to list queued runs",
				err.Error(),
			).JSON(e)
		}
		queuedByPipelineID := mapQueuedRunsToPipelines(e.App, pipelineRecords, queuedRuns)

		allQueuedForAllPipelines := []QueuedPipelineRunAggregate{}
		for _, p := range pipelineRecords {
			queuedForThisPipeline := queuedByPipelineID[p.Id]
			allQueuedForAllPipelines = append(allQueuedForAllPipelines, queuedForThisPipeline...)
		}

		if strings.ToLower(status) == statusStringQueued {
			if len(allQueuedForAllPipelines) == 0 {
				return e.JSON(http.StatusOK, []*pipelineWorkflowSummary{})
			}

			queuedSummaries := buildQueuedPipelineSummaries(
				e.App,
				allQueuedForAllPipelines,
				authRecord.GetString("Timezone"),
				map[string]map[string]any{},
			)
			return e.JSON(http.StatusOK, queuedSummaries)
		}

		resultsRecords, err := e.App.FindRecordsByFilter(
			"pipeline_results",
			"owner={:owner}",
			"",
			-1,
			0,
			dbx.Params{
				"owner": organization.Id,
			},
		)

		if err != nil {
			return e.JSON(http.StatusOK, []*pipelineWorkflowSummary{})
		}

		var allSummaries []*pipelineWorkflowSummary
		runnerCache := map[string]map[string]any{}
		runnerInfoByPipelineID := map[string]pipelineRunnerInfo{}

		for _, resultRecord := range resultsRecords {
			workflowID := resultRecord.GetString("workflow_id")
			runID := resultRecord.GetString("run_id")
			pipelineID := resultRecord.GetString("pipeline")

			pipelineRecord, ok := pipelineMap[pipelineID]
			if !ok {
				e.App.Logger().Warn(fmt.Sprintf("pipeline not found for result: %s", pipelineID))
				continue
			}

			runnerInfo, ok := runnerInfoByPipelineID[pipelineID]
			if !ok {
				runnerInfo, err = runners.ParsePipelineRunnerInfo(pipelineRecord.GetString("yaml"))
				if err != nil {
					e.App.Logger().Warn(fmt.Sprintf(
						"failed to parse pipeline yaml for runners (pipeline_id=%s): %v",
						pipelineID,
						err,
					))
				}
				runnerInfoByPipelineID[pipelineID] = runnerInfo
			}

			if temporalClient == nil {
				temporalClient, err = temporalclient.GetTemporalClientWithNamespace(namespace)
				if err != nil {
					return apierror.New(
						http.StatusInternalServerError,
						"temporal",
						"unable to create temporal client",
						err.Error(),
					).JSON(e)
				}
			}

			execInfo, apiErr := describeWorkflowExecution(temporalClient, workflowID, runID)
			if apiErr != nil {
				return apiErr.JSON(e)
			}

			children, err := getChildWorkflows(
				context.Background(),
				temporalClient,
				namespace,
				workflowID,
				runID,
			)
			if err != nil {
				e.App.Logger().Warn(fmt.Sprintf("failed to get child workflows: %v", err))
			}

			hierarchy := buildPipelineExecutionHierarchyFromResult(
				e.App,
				resultRecord,
				execInfo,
				children,
				namespace,
				authRecord.GetString("Timezone"),
				temporalClient,
			)
			if len(hierarchy) == 0 {
				continue
			}

			annotated, err := attachRunnerInfoFromTemporalStartInput(
				attachRunnerInfoFromTemporalInputArgs{
					App:         e.App,
					Ctx:         context.Background(),
					Client:      temporalClient,
					Executions:  hierarchy,
					Info:        runnerInfo,
					RunnerCache: runnerCache,
				},
			)

			if err != nil {
				e.App.Logger().Warn(fmt.Sprintf("fallback to pipeline runner info: %v", err))
				annotated = attachPipelineRunnerInfo(
					e.App,
					hierarchy,
					"",
					runnerInfo,
					runnerCache,
				)
			}

			allSummaries = append(allSummaries, annotated...)
		}

		sort.Slice(allSummaries, func(i, j int) bool {
			return allSummaries[i].StartTime > allSummaries[j].StartTime
		})

		if status != "" {
			filtered := []*pipelineWorkflowSummary{}
			for _, summary := range allSummaries {
				if strings.EqualFold(summary.Status, status) {
					filtered = append(filtered, summary)
				}
			}
			allSummaries = filtered

			sort.Slice(allSummaries, func(i, j int) bool {
				return allSummaries[i].StartTime > allSummaries[j].StartTime
			})
		}

		finalSummaries := []*pipelineWorkflowSummary{}

		if status == "" {
			if len(allQueuedForAllPipelines) > 0 {
				queuedSummaries := buildQueuedPipelineSummaries(
					e.App,
					allQueuedForAllPipelines,
					authRecord.GetString("Timezone"),
					runnerCache,
				)
				if len(queuedSummaries) > 0 {
					finalSummaries = append(finalSummaries, queuedSummaries...)
				}
			}
		}

		finalSummaries = append(finalSummaries, allSummaries...)

		if finalSummaries == nil {
			finalSummaries = []*pipelineWorkflowSummary{}
		}

		return e.JSON(http.StatusOK, finalSummaries)
	}
}

func buildPipelineExecutionHierarchyFromResult(
	app core.App,
	resultRecord *core.Record,
	rootExecution *WorkflowExecution,
	childExecutions []*WorkflowExecution,
	namespace string,
	userTimezone string,
	c client.Client,
) []*WorkflowExecutionSummary {
	if rootExecution == nil || rootExecution.Execution == nil {
		return nil
	}

	loc, err := time.LoadLocation(userTimezone)
	if err != nil {
		loc = time.Local
	}

	rootSummary := buildWorkflowExecutionSummary(rootExecution, c)
	if rootSummary == nil {
		return nil
	}

	if rootExecution.Memo != nil {
		if field, ok := rootExecution.Memo.Fields["test"]; ok && field.Data != nil {
			rootSummary.DisplayName = DecodeFromTemporalPayload(*field.Data)
		}
	}

	w := pipeline.PipelineWorkflow{}
	if rootSummary.Type.Name == w.Name() {
		rootSummary.Results = computePipelineResultsFromRecord(app, resultRecord)
		if len(rootSummary.Results) == 0 {
			rootSummary.Results = computePipelineResults(
				app,
				namespace,
				rootExecution.Execution.WorkflowID,
				rootExecution.Execution.RunID,
			)
		}
	}

	for _, childExecution := range childExecutions {
		childSummary := buildWorkflowExecutionSummary(childExecution, c)
		if childSummary == nil || childExecution.Execution == nil {
			continue
		}
		childSummary.DisplayName = computeChildDisplayName(childExecution.Execution.WorkflowID)
		rootSummary.Children = append(rootSummary.Children, childSummary)
	}

	roots := []*WorkflowExecutionSummary{rootSummary}
	sortExecutionSummaries(roots, loc, false)
	return roots
}

func buildWorkflowExecutionSummary(
	exec *WorkflowExecution,
	c client.Client,
) *WorkflowExecutionSummary {
	if exec == nil || exec.Execution == nil {
		return nil
	}

	summary := &WorkflowExecutionSummary{
		Execution: exec.Execution,
		Type:      exec.Type,
		StartTime: exec.StartTime,
		EndTime:   exec.CloseTime,
		Status:    normalizeTemporalStatus(exec.Status),
	}

	if c != nil && enums.WorkflowExecutionStatus(
		enums.WorkflowExecutionStatus_value[exec.Status],
	) == enums.WORKFLOW_EXECUTION_STATUS_FAILED {
		if failure := fetchWorkflowFailure(
			context.Background(),
			c,
			exec.Execution.WorkflowID,
			exec.Execution.RunID,
		); failure != nil {
			summary.FailureReason = failure
		}
	}

	return summary
}

func computePipelineResultsFromRecord(app core.App, record *core.Record) []PipelineResults {
	if app == nil || record == nil {
		return nil
	}

	videos := record.GetStringSlice("video_results")
	screenshots := record.GetStringSlice("screenshots")
	if len(videos) == 0 || len(screenshots) == 0 {
		return nil
	}

	screenshotMap := make(map[string]string, len(screenshots))
	for _, name := range screenshots {
		if key, ok := baseKey(name, "_screenshot_"); ok {
			screenshotMap[key] = name
		}
	}

	results := make([]PipelineResults, 0, len(videos))
	for _, name := range videos {
		key, ok := baseKey(name, "_result_video_")
		if !ok {
			continue
		}

		screenshot, found := screenshotMap[key]
		if !found {
			continue
		}

		results = append(results, PipelineResults{
			Video: utils.JoinURL(
				app.Settings().Meta.AppURL,
				"api", "files", "pipeline_results",
				record.Id,
				record.GetString("video_results"),
				name,
			),
			Screenshot: utils.JoinURL(
				app.Settings().Meta.AppURL,
				"api", "files", "pipeline_results",
				record.Id,
				record.GetString("screenshots"),
				screenshot,
			),
		})
	}

	return results
}

func getChildWorkflows(
	ctx context.Context,
	temporalClient client.Client,
	namespace string,
	parentWorkflowID string,
	parentRunID string,
) ([]*WorkflowExecution, error) {
	childResp, err := temporalClient.ListWorkflow(
		ctx,
		&workflowservice.ListWorkflowExecutionsRequest{
			Namespace: namespace,
			Query: fmt.Sprintf(
				`ParentWorkflowId="%s" AND ParentRunId="%s"`,
				parentWorkflowID,
				parentRunID,
			),
		},
	)
	if err != nil {
		return nil, err
	}

	var children []*WorkflowExecution
	for _, childExec := range childResp.GetExecutions() {
		childDesc, err := temporalClient.DescribeWorkflowExecution(
			ctx,
			childExec.GetExecution().GetWorkflowId(),
			childExec.GetExecution().GetRunId(),
		)
		if err != nil {
			continue
		}

		childJSON, err := protojson.Marshal(childDesc.GetWorkflowExecutionInfo())
		if err != nil {
			continue
		}

		var childWorkflow WorkflowExecution
		err = json.Unmarshal(childJSON, &childWorkflow)
		if err != nil {
			continue
		}

		childWorkflow.ParentExecution = &WorkflowIdentifier{
			WorkflowID: parentWorkflowID,
			RunID:      parentRunID,
		}
		children = append(children, &childWorkflow)
	}
	return children, nil
}

func describeWorkflowExecution(
	temporalClient client.Client,
	workflowID string,
	runID string,
) (*WorkflowExecution, *apierror.APIError) {
	workflowExecution, err := temporalClient.DescribeWorkflowExecution(
		context.Background(),
		workflowID,
		runID,
	)
	if err != nil {
		notFound := &serviceerror.NotFound{}
		if errors.As(err, &notFound) {
			return nil, apierror.New(
				http.StatusNotFound,
				"workflow",
				"workflow not found",
				err.Error(),
			)
		}
		invalidArgument := &serviceerror.InvalidArgument{}
		if errors.As(err, &invalidArgument) {
			return nil, apierror.New(
				http.StatusBadRequest,
				"workflow",
				"invalid workflow ID",
				err.Error(),
			)
		}
		return nil, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to describe workflow execution",
			err.Error(),
		)
	}

	weJSON, err := protojson.Marshal(workflowExecution.GetWorkflowExecutionInfo())
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to marshal workflow execution",
			err.Error(),
		)
	}

	var execInfo WorkflowExecution
	err = json.Unmarshal(weJSON, &execInfo)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to unmarshal workflow execution",
			err.Error(),
		)
	}

	if workflowExecution.GetWorkflowExecutionInfo().GetParentExecution() != nil {
		parentJSON, _ := protojson.Marshal(
			workflowExecution.GetWorkflowExecutionInfo().GetParentExecution(),
		)
		var parentInfo WorkflowIdentifier
		_ = json.Unmarshal(parentJSON, &parentInfo)
		execInfo.ParentExecution = &parentInfo
	}

	return &execInfo, nil
}

type pipelineRunnerInfo = runners.PipelineRunnerInfo

type pipelineWorkflowSummary struct {
	WorkflowExecutionSummary
	GlobalRunnerID string           `json:"global_runner_id,omitempty"`
	RunnerIDs      []string         `json:"runner_ids,omitempty"`
	RunnerRecords  []map[string]any `json:"runner_records,omitempty"`
}

func appendQueuedPipelineSummaries(
	app core.App,
	response map[string][]*pipelineWorkflowSummary,
	queuedByPipelineID map[string][]QueuedPipelineRunAggregate,
	userTimezone string,
	runnerCache map[string]map[string]any,
) {
	for pipelineID, queuedRuns := range queuedByPipelineID {
		queuedSummaries := buildQueuedPipelineSummaries(
			app,
			queuedRuns,
			userTimezone,
			runnerCache,
		)
		if len(queuedSummaries) == 0 {
			continue
		}
		response[pipelineID] = append(queuedSummaries, response[pipelineID]...)
	}
}

func buildQueuedPipelineSummaries(
	app core.App,
	queuedRuns []QueuedPipelineRunAggregate,
	userTimezone string,
	runnerCache map[string]map[string]any,
) []*pipelineWorkflowSummary {
	if len(queuedRuns) == 0 {
		return nil
	}

	nameCache := map[string]string{}
	resolveName := func(identifier string) string {
		if cached, ok := nameCache[identifier]; ok {
			return cached
		}
		displayName := resolveQueuedPipelineDisplayName(app, identifier)
		nameCache[identifier] = displayName
		return displayName
	}

	sortedRuns := append([]QueuedPipelineRunAggregate(nil), queuedRuns...)
	sort.Slice(sortedRuns, func(i, j int) bool {
		return sortedRuns[i].EnqueuedAt.After(sortedRuns[j].EnqueuedAt)
	})

	summaries := make([]*pipelineWorkflowSummary, 0, len(sortedRuns))
	for _, queued := range sortedRuns {
		summaries = append(summaries, buildQueuedPipelineSummary(
			app,
			queued,
			userTimezone,
			runnerCache,
			resolveName(queued.PipelineIdentifier),
		))
	}

	return summaries
}

func buildQueuedPipelineSummary(
	app core.App,
	queued QueuedPipelineRunAggregate,
	userTimezone string,
	runnerCache map[string]map[string]any,
	displayName string,
) *pipelineWorkflowSummary {
	enqueuedAt := formatQueuedRunTime(queued.EnqueuedAt, userTimezone)
	queue := &WorkflowQueueSummary{
		TicketID:  queued.TicketID,
		Position:  queued.Position + 1,
		LineLen:   queued.LineLen,
		RunnerIDs: copyStringSlice(queued.RunnerIDs),
	}

	pipelineWorkflow := pipeline.NewPipelineWorkflow()
	exec := &WorkflowExecutionSummary{
		Execution: &WorkflowIdentifier{
			WorkflowID: "queue/" + queued.TicketID,
			RunID:      queued.TicketID,
		},
		Type: WorkflowType{
			Name: pipelineWorkflow.Name(),
		},
		EnqueuedAt:  enqueuedAt,
		Status:      string(WorkflowStatusQueued),
		DisplayName: displayName,
		Queue:       queue,
	}

	runnerIDs := copyStringSlice(queued.RunnerIDs)
	return &pipelineWorkflowSummary{
		WorkflowExecutionSummary: *exec,
		RunnerIDs:                runnerIDs,
		RunnerRecords:            runners.ResolveRunnerRecords(app, runnerIDs, runnerCache),
	}
}

func formatQueuedRunTime(enqueuedAt time.Time, userTimezone string) string {
	if enqueuedAt.IsZero() {
		return ""
	}
	loc, err := time.LoadLocation(userTimezone)
	if err != nil {
		loc = time.Local
	}
	return enqueuedAt.In(loc).Format("02/01/2006, 15:04:05")
}

func mapQueuedRunsToPipelines(
	app core.App,
	pipelineRecords []*core.Record,
	queuedRuns map[string]QueuedPipelineRunAggregate,
) map[string][]QueuedPipelineRunAggregate {
	if len(pipelineRecords) == 0 || len(queuedRuns) == 0 {
		return map[string][]QueuedPipelineRunAggregate{}
	}

	pipelineIdentifiers := make(map[string]string, len(pipelineRecords))
	for _, record := range pipelineRecords {
		pipelineID := record.Id
		pipelineIdentifiers[pipelineID] = pipelineID
		canonifiedName := record.GetString("canonified_name")
		org, err := app.FindRecordById("organizations", record.GetString("owner"))
		if err != nil {
			continue
		}
		orgName := org.GetString("canonified_name")
		if canonifiedName != "" && orgName != "" {
			pipelineIdentifiers[fmt.Sprintf("%s/%s", orgName, canonifiedName)] = pipelineID
		}
	}

	queuedByPipeline := make(map[string][]QueuedPipelineRunAggregate)
	for _, queued := range queuedRuns {
		pipelineID, ok := pipelineIdentifiers[strings.Trim(queued.PipelineIdentifier, "/")]
		if !ok {
			continue
		}
		queuedByPipeline[pipelineID] = append(queuedByPipeline[pipelineID], queued)
	}

	return queuedByPipeline
}

func attachPipelineRunnerInfo(
	app core.App,
	executions []*WorkflowExecutionSummary,
	globalRunnerID string,
	info pipelineRunnerInfo,
	runnerCache map[string]map[string]any,
) []*pipelineWorkflowSummary {
	if len(executions) == 0 {
		return []*pipelineWorkflowSummary{}
	}

	runnerIDs := runners.RunnerIDsWithGlobal(info, globalRunnerID)
	runnerRecords := runners.ResolveRunnerRecords(app, runnerIDs, runnerCache)

	annotated := make([]*pipelineWorkflowSummary, 0, len(executions))
	for _, exec := range executions {
		if exec == nil {
			continue
		}
		annotated = append(annotated, &pipelineWorkflowSummary{
			WorkflowExecutionSummary: *exec,
			GlobalRunnerID:           globalRunnerID,
			RunnerIDs:                runnerIDs,
			RunnerRecords:            runnerRecords,
		})
	}
	return annotated
}

type attachRunnerInfoFromTemporalInputArgs struct {
	App         core.App
	Ctx         context.Context
	Client      client.Client
	Executions  []*WorkflowExecutionSummary
	Info        pipelineRunnerInfo
	RunnerCache map[string]map[string]any
}

func attachRunnerInfoFromTemporalStartInput(
	args attachRunnerInfoFromTemporalInputArgs,
) ([]*pipelineWorkflowSummary, error) {
	if len(args.Executions) == 0 {
		return []*pipelineWorkflowSummary{}, nil
	}

	if args.RunnerCache == nil {
		args.RunnerCache = map[string]map[string]any{}
	}

	cache := map[string]string{}
	out := make([]*pipelineWorkflowSummary, 0, len(args.Executions))

	for _, exec := range args.Executions {
		if exec == nil {
			continue
		}

		key := exec.Execution.WorkflowID + ":" + exec.Execution.RunID

		globalRunnerID, ok := cache[key]
		if !ok {
			gr, err := readGlobalRunnerIDFromTemporalHistory(
				args.Ctx,
				args.Client,
				exec.Execution.WorkflowID,
				exec.Execution.RunID,
			)
			if err != nil {
				return nil, err
			}
			globalRunnerID = gr
			cache[key] = globalRunnerID
		}

		out = append(out, attachPipelineRunnerInfo(
			args.App,
			[]*WorkflowExecutionSummary{exec},
			globalRunnerID,
			args.Info,
			args.RunnerCache,
		)...)
	}

	return out, nil
}

func readGlobalRunnerIDFromTemporalHistory(
	ctx context.Context,
	c client.Client,
	workflowID, runID string,
) (string, error) {
	iter := c.GetWorkflowHistory(
		ctx,
		workflowID,
		runID,
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)

	dc := converter.GetDefaultDataConverter()

	for iter.HasNext() {
		ev, err := iter.Next()
		if err != nil {
			return "", err
		}

		if ev.GetEventType() != enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED {
			continue
		}

		attr := ev.GetWorkflowExecutionStartedEventAttributes()
		if attr == nil || attr.GetInput() == nil {
			return "", nil
		}

		var in pipeline.PipelineWorkflowInput
		if err := dc.FromPayload(attr.GetInput().GetPayloads()[0], &in); err != nil {
			// If decoding fails, omit (don’t fail the endpoint).
			return "", nil // nolint
		}

		return runners.GlobalRunnerIDFromConfig(in.WorkflowInput.Config), nil
	}

	return "", nil
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
				t1, _ := time.Parse(time.RFC3339, otherExecs[i].StartTime)
				t2, _ := time.Parse(time.RFC3339, otherExecs[j].StartTime)
				return t1.After(t2)
			})

			if remainingSlots > len(otherExecs) {
				remainingSlots = len(otherExecs)
			}

			selected = append(selected, otherExecs[:remainingSlots]...)
		}

		sort.Slice(selected, func(i, j int) bool {
			t1, _ := time.Parse(time.RFC3339, selected[i].StartTime)
			t2, _ := time.Parse(time.RFC3339, selected[j].StartTime)
			return t1.After(t2)
		})

		if len(selected) > 0 {
			result[pipelineID] = selected
		}
	}

	return result
}

func processPipelineResults(
	namespace string,
	resultsRecords []*core.Record,
	temporalClient *client.Client,
) ([]*WorkflowExecution, error) {
	var allExecutions []*WorkflowExecution

	for _, resultRecord := range resultsRecords {
		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return nil, apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
		}

		if *temporalClient == nil {
			*temporalClient = c
		}

		workflowExecution, err := c.DescribeWorkflowExecution(
			context.Background(),
			resultRecord.GetString("workflow_id"),
			resultRecord.GetString("run_id"),
		)
		if err != nil {
			notFound := &serviceerror.NotFound{}
			if errors.As(err, &notFound) {
				return nil, apierror.New(
					http.StatusNotFound,
					"workflow",
					"workflow not found",
					err.Error(),
				)
			}
			invalidArgument := &serviceerror.InvalidArgument{}
			if errors.As(err, &invalidArgument) {
				return nil, apierror.New(
					http.StatusBadRequest,
					"workflow",
					"invalid workflow ID",
					err.Error(),
				)
			}
			return nil, apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to describe workflow execution",
				err.Error(),
			)
		}

		weJSON, err := protojson.Marshal(workflowExecution.GetWorkflowExecutionInfo())
		if err != nil {
			return nil, apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to marshal workflow execution",
				err.Error(),
			)
		}

		var execInfo WorkflowExecution
		err = json.Unmarshal(weJSON, &execInfo)
		if err != nil {
			return nil, apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal workflow execution",
				err.Error(),
			)
		}

		if workflowExecution.GetWorkflowExecutionInfo().GetParentExecution() != nil {
			parentJSON, _ := protojson.Marshal(
				workflowExecution.GetWorkflowExecutionInfo().GetParentExecution(),
			)
			var parentInfo WorkflowIdentifier
			_ = json.Unmarshal(parentJSON, &parentInfo)
			execInfo.ParentExecution = &parentInfo
		}

		allExecutions = append(allExecutions, &execInfo)
	}

	return allExecutions, nil
}
