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
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
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

			weJSON, err := protojson.Marshal(workflowExecution.WorkflowExecutionInfo)
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

			if workflowExecution.WorkflowExecutionInfo.ParentExecution != nil {
				parentJSON, _ := protojson.Marshal(workflowExecution.WorkflowExecutionInfo.ParentExecution)
				var parentInfo WorkflowIdentifier
				json.Unmarshal(parentJSON, &parentInfo)
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

		return e.JSON(http.StatusOK, hierarchy)
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
		namespace := organization.GetString("canonified_name")

		for _, pipelineRecord := range pipelineRecords {
			pipelineID := pipelineRecord.Id
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

					weJSON, err := protojson.Marshal(workflowExecution.WorkflowExecutionInfo)
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

					if workflowExecution.WorkflowExecutionInfo.ParentExecution != nil {
						parentJSON, _ := protojson.Marshal(workflowExecution.WorkflowExecutionInfo.ParentExecution)
						var parentInfo WorkflowIdentifier
						json.Unmarshal(parentJSON, &parentInfo)
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

		return e.JSON(http.StatusOK, selectedExecutions)
	}
}

func selectTopExecutionsByPipeline(executions []struct {
	pipelineID string
	execution  *WorkflowExecutionSummary
}, limit int) map[string][]*WorkflowExecutionSummary {

	result := make(map[string][]*WorkflowExecutionSummary)

	if len(executions) == 0 {
		return result
	}

	var runningExecutions []struct {
		pipelineID string
		execution  *WorkflowExecutionSummary
	}
	var otherExecutions []struct {
		pipelineID string
		execution  *WorkflowExecutionSummary
	}

	for _, exec := range executions {
		if exec.execution.Status == "running" {
			runningExecutions = append(runningExecutions, exec)
		} else {
			otherExecutions = append(otherExecutions, exec)
		}
	}

	var finalSelections []struct {
		pipelineID string
		execution  *WorkflowExecutionSummary
	}
	finalSelections = runningExecutions

	remainingSlots := limit - len(runningExecutions)

	if remainingSlots > 0 && len(otherExecutions) > 0 {
		sort.Slice(otherExecutions, func(i, j int) bool {
			iTimeStr := otherExecutions[i].execution.StartTime
			jTimeStr := otherExecutions[j].execution.StartTime

			if iTimeStr == "" && jTimeStr == "" {
				return false
			}
			if iTimeStr == "" {
				return false
			}
			if jTimeStr == "" {
				return true
			}

			iTime, err1 := time.Parse(time.RFC3339, iTimeStr)
			jTime, err2 := time.Parse(time.RFC3339, jTimeStr)

			if err1 == nil && err2 == nil {
				return iTime.After(jTime)
			}

			return iTimeStr > jTimeStr
		})

		if remainingSlots > len(otherExecutions) {
			remainingSlots = len(otherExecutions)
		}
		finalSelections = append(finalSelections, otherExecutions[:remainingSlots]...)
	}

	for _, selection := range finalSelections {
		result[selection.pipelineID] = append(result[selection.pipelineID], selection.execution)
	}

	for pipelineID, execs := range result {
		sort.Slice(execs, func(i, j int) bool {
			iTimeStr := execs[i].StartTime
			jTimeStr := execs[j].StartTime

			if iTimeStr == "" && jTimeStr == "" {
				return false
			}
			if iTimeStr == "" {
				return false
			}
			if jTimeStr == "" {
				return true
			}

			iTime, err1 := time.Parse(time.RFC3339, iTimeStr)
			jTime, err2 := time.Parse(time.RFC3339, jTimeStr)

			if err1 == nil && err2 == nil {
				return iTime.After(jTime)
			}

			return iTimeStr > jTimeStr
		})
		
		result[pipelineID] = execs
	}

	return result
}
