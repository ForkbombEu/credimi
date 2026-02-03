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
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/encoding/protojson"
)

type PipelineInput struct {
	Yaml               string `json:"yaml"`
	PipelineIdentifier string `json:"pipeline_identifier"`
}

var PipelineRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/pipeline",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/start",
			Handler:       HandlePipelineStart,
			RequestSchema: PipelineInput{},
			Description:   "Start a pipeline workflow from a YAML file",
		},
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

func HandlePipelineStart() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[PipelineInput](e)
		if err != nil {
			return err
		}
		appURL := e.App.Settings().Meta.AppURL
		appName := e.App.Settings().Meta.AppName
		logoURL := utils.JoinURL(
			appURL,
			"logos",
			fmt.Sprintf("%s_logo-transp_emblem.png", strings.ToLower(appName)),
		)
		if e.Auth == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"auth",
				"authentication required",
				"user not authenticated",
			).JSON(e)
		}

		userID := e.Auth.Id
		userMail := e.Auth.GetString("email")
		userName := e.Auth.GetString("name")
		orgID, err := GetUserOrganizationID(e.App, userID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization ID",
				err.Error(),
			).JSON(e)
		}
		namespace, err := GetUserOrganizationCanonifiedName(e.App, userID)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
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

		pipelineRecord, err := canonify.Resolve(e.App, input.PipelineIdentifier)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"pipeline_identifier",
				"pipeline not found",
				err.Error(),
			).JSON(e)
		}
		record := core.NewRecord(coll)
		record.Set("owner", orgID)
		record.Set("pipeline", pipelineRecord.Id)

		memo := map[string]any{
			"test":   "pipeline-run",
			"userID": userID,
		}
		config := map[string]any{
			"namespace": namespace,
			"app_url":   appURL,
			"app_name":  appName,
			"app_logo":  logoURL,
			"user_name": userName,
			"user_mail": userMail,
		}
		w := pipeline.NewPipelineWorkflow()
		result, err := w.Start(input.Yaml, config, memo)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start workflow",
				err.Error(),
			).JSON(e)
		}

		record.Set("workflow_id", result.WorkflowID)
		record.Set("run_id", result.WorkflowRunID)

		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"pipeline",
				"failed to save pipeline record",
				err.Error(),
			).JSON(e)
		}
		return e.JSON(http.StatusOK, map[string]any{
			"message": "Workflow started successfully",
			"result":  result,
		})
	}
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
		namespace := organization.GetString("canonified_name")
		var temporalClient client.Client

		pipelineRecord := pipelineRecords[0]
		pipelineID = pipelineRecord.Id

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

			if temporalClient == nil {
				temporalClient = c
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

			allExecutions = append(allExecutions, &execInfo)
		}

		if len(allExecutions) == 0 {
			return e.JSON(http.StatusOK, ListMyChecksResponse{
				[]*WorkflowExecutionSummary{},
			})
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

		pipelineExecutionsMap := make(map[string][]*WorkflowExecutionSummary)
		pipelineRunnerInfoMap := make(map[string]pipelineRunnerInfo)
		namespace := organization.GetString("canonified_name")

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

		if len(pipelineExecutionsMap) == 0 {
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

		return e.JSON(http.StatusOK, response)
	}
}

type pipelineRunnerInfo = runners.PipelineRunnerInfo

type pipelineWorkflowSummary struct {
	WorkflowExecutionSummary
	GlobalRunnerID string           `json:"global_runner_id,omitempty"`
	RunnerIDs      []string         `json:"runner_ids,omitempty"`
	RunnerRecords  []map[string]any `json:"runner_records,omitempty"`
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
			if exec.execution.Status == "running" {
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
