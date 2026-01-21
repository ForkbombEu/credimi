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
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
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
			Path:    "/details",
			Handler: HandleGetPipelineDetails,
		},
		{
			Method:  http.MethodGet,
			Path:    "/specific-details",
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

		pipelineID := e.Request.URL.Query().Get("pipelineId")
		if pipelineID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"pipeline",
				"pipeline ID is required",
				"missing pipelineId query parameter",
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
			return e.JSON(http.StatusOK, map[string]any{
				"pipelines": []any{},
				"count":     0,
			})
		}

		results := make(map[string][]WorkflowDescriptionInfoSummary)

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

			results[pipelineID] = make([]WorkflowDescriptionInfoSummary, 0, len(resultsRecords))
			if err == nil {
				for _, resultRecord := range resultsRecords {

					c, err := temporalclient.GetTemporalClientWithNamespace(organization.GetString("canonified_name"))
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

					var execInfo WorkflowExecutionInfo
					err = json.Unmarshal(weJSON, &execInfo)
					if err != nil {
						return apierror.New(
							http.StatusInternalServerError,
							"workflow",
							"failed to unmarshal workflow execution",
							err.Error(),
						).JSON(e)
					}

					summary := WorkflowDescriptionInfoSummary{
						Execution: execInfo.Execution,
						Type:      execInfo.Type,
						StartTime: execInfo.StartTime,
						EndTime:   execInfo.CloseTime,
						Status:    execInfo.Status,
					}

					if enums.WorkflowExecutionStatus(
						enums.WorkflowExecutionStatus_value[summary.Status],
					) == enums.WORKFLOW_EXECUTION_STATUS_FAILED {
						if failure := fetchWorkflowFailure(
							context.Background(),
							c,
							summary.Execution.WorkflowID,
							summary.Execution.RunID,
						); failure != nil {
							summary.FailureReason = failure
						}
					}

					summary.DisplayName = pipelineRecord.GetString("name")

					runResults := computePipelineResults(
						e.App,
						organization.GetString("canonified_name"),
						summary.Execution.WorkflowID,
						summary.Execution.RunID,
					)
					summary.Results = runResults

					results[pipelineID] = append(results[pipelineID], summary)
				}
			}

		}

		return e.JSON(http.StatusOK, map[string]any{
			"pipelines": results,
		})
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
			return e.JSON(http.StatusOK, map[string]any{
				"pipelines": []any{},
				"count":     0,
			})
		}

		allWorkflowsByPipeline := make(map[string][]WorkflowDescriptionInfoSummary)
		runningWorkflowsByPipeline := make(map[string][]WorkflowDescriptionInfoSummary)

		runningCount := 0

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

			if err == nil {
				for _, resultRecord := range resultsRecords {
					c, err := temporalclient.GetTemporalClientWithNamespace(organization.GetString("canonified_name"))
					if err != nil {
						continue
					}

					workflowExecution, err := c.DescribeWorkflowExecution(
						context.Background(),
						resultRecord.GetString("workflow_id"),
						resultRecord.GetString("run_id"),
					)
					if err != nil {
						continue
					}

					weJSON, err := protojson.Marshal(workflowExecution.WorkflowExecutionInfo)
					if err != nil {
						continue
					}

					var execInfo WorkflowExecutionInfo
					err = json.Unmarshal(weJSON, &execInfo)
					if err != nil {
						continue
					}

					summary := WorkflowDescriptionInfoSummary{
						Execution: execInfo.Execution,
						Type:      execInfo.Type,
						StartTime: execInfo.StartTime,
						EndTime:   execInfo.CloseTime,
						Status:    execInfo.Status,
					}

					if enums.WorkflowExecutionStatus(
						enums.WorkflowExecutionStatus_value[summary.Status],
					) == enums.WORKFLOW_EXECUTION_STATUS_FAILED {
						if failure := fetchWorkflowFailure(
							context.Background(),
							c,
							summary.Execution.WorkflowID,
							summary.Execution.RunID,
						); failure != nil {
							summary.FailureReason = failure
						}
					}

					summary.DisplayName = pipelineRecord.GetString("name")

					runResults := computePipelineResults(
						e.App,
						organization.GetString("canonified_name"),
						summary.Execution.WorkflowID,
						summary.Execution.RunID,
					)
					summary.Results = runResults

					allWorkflowsByPipeline[pipelineID] = append(allWorkflowsByPipeline[pipelineID], summary)

					if summary.Status == "RUNNING" {
						runningWorkflowsByPipeline[pipelineID] = append(runningWorkflowsByPipeline[pipelineID], summary)
						runningCount++
					}
				}
			}
		}

		results := make(map[string][]WorkflowDescriptionInfoSummary)

		for pipelineID, workflows := range runningWorkflowsByPipeline {
			results[pipelineID] = append(results[pipelineID], workflows...)
		}

		if runningCount < 5 {
			needed := 5 - runningCount

			type workflowItem struct {
				pipelineID string
				summary    WorkflowDescriptionInfoSummary
			}

			var nonRunningList []workflowItem
			for pipelineID, workflows := range allWorkflowsByPipeline {
				for _, workflow := range workflows {
					if workflow.Status != "RUNNING" {
						nonRunningList = append(nonRunningList, workflowItem{
							pipelineID: pipelineID,
							summary:    workflow,
						})
					}
				}
			}

			sort.Slice(nonRunningList, func(i, j int) bool {
				return nonRunningList[i].summary.EndTime > nonRunningList[j].summary.EndTime
			})

			added := 0
			for _, item := range nonRunningList {
				if added >= needed {
					break
				}
				results[item.pipelineID] = append(results[item.pipelineID], item.summary)
				added++
			}
		}

		return e.JSON(http.StatusOK, map[string]any{
			"pipelines": results,
		})
	}
}
