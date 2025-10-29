// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/encoding/protojson"
)

var ChecksRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL: "/api/my/checks",
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodGet,
			Handler:        HandleListMyChecks,
			ResponseSchema: ListMyChecksResponse{},
			Description:    "List all checks for the authenticated user",
			Summary:        "Get a list of all checks for the authenticated user",
		},
		{
			Method:         http.MethodGet,
			Path:           "/{checkId}/runs",
			Handler:        HandleListMyCheckRuns,
			ResponseSchema: ListMyCheckRunsResponse{},
			Description:    "List all runs for a specific check",
			Summary:        "Get a list of all runs for a specific check",
		},
		{
			Method:         http.MethodGet,
			Path:           "/{checkId}/runs/{runId}",
			Handler:        HandleGetMyCheckRun,
			ResponseSchema: GetMyCheckRunResponse{},
			Description:    "Get details of a specific run for a check",
			Summary:        "Get details of a specific run for a check",
		},
		{
			Method:         http.MethodGet,
			Path:           "/{checkId}/runs/{runId}/history",
			Handler:        HandleGetMyCheckRunHistory,
			ResponseSchema: GetMyCheckRunHistoryResponse{},
			Description:    "Get the history of events for a specific run of a check",
			Summary:        "Get the history of events for a specific run of a check",
		},
		{
			Method:         http.MethodPost,
			Path:           "/{checkId}/runs/{runId}/rerun",
			Handler:        HandleRerunMyCheck,
			RequestSchema:  ReRunCheckRequest{},
			ResponseSchema: ReRunCheckResponse{},
			Description:    "Re-run a specific check run",
			Summary:        "Re-run a specific check run",
		},
		{
			Method:         http.MethodPost,
			Path:           "/{checkId}/runs/{runId}/cancel",
			Handler:        HandleCancelMyCheckRun,
			ResponseSchema: CancelMyCheckRunResponse{},
			Description:    "Cancel a specific check run",
			Summary:        "Cancel a specific check run",
		},
		{
			Method:         http.MethodGet,
			Path:           "/{checkId}/runs/{runId}/export",
			Handler:        HandleExportMyCheckRun,
			ResponseSchema: ExportMyCheckRunResponse{},
			Description:    "Export a specific check run",
			Summary:        "Export a specific check run",
		},
		{
			Method:         http.MethodGet,
			Path:           "/{checkId}/runs/{runId}/logs",
			Handler:        HandleMyCheckLogs,
			ResponseSchema: ChecksLogsResponse{},
			Description:    "Start or Stop logs for a specific check run and get the log channel",
			Summary:        "Start or Stop logs for a specific check run",
			QuerySearchAttributes: []routing.QuerySearchAttribute{
				{
					Name:        "action",
					Required:    false,
					Description: "Can be 'start' or 'stop' to control logging for the check run",
				},
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "/{checkId}/runs/{runId}/terminate",
			Handler:        HandleTerminateMyCheckRun,
			ResponseSchema: TerminateMyCheckRunResponse{},
			Description:    "Terminate a specific check run",
			Summary:        "Terminate a specific check run",
		},
	},
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	AuthenticationRequired: true,
}

type ReRunCheckRequest struct {
	Config map[string]interface{} `json:"config"`
}

func HandleListMyChecks() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}
		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}
		statusParam := e.Request.URL.Query().Get("status")
		var statusFilters []enums.WorkflowExecutionStatus
		if statusParam != "" {
			statusStrings := strings.SplitSeq(statusParam, ",")
			for s := range statusStrings {
				switch strings.ToLower(strings.TrimSpace(s)) {
				case "running":
					statusFilters = append(statusFilters, enums.WORKFLOW_EXECUTION_STATUS_RUNNING)
				case "completed":
					statusFilters = append(statusFilters, enums.WORKFLOW_EXECUTION_STATUS_COMPLETED)
				case "failed":
					statusFilters = append(statusFilters, enums.WORKFLOW_EXECUTION_STATUS_FAILED)
				case "terminated":
					statusFilters = append(
						statusFilters,
						enums.WORKFLOW_EXECUTION_STATUS_TERMINATED,
					)
				case "canceled":
					statusFilters = append(statusFilters, enums.WORKFLOW_EXECUTION_STATUS_CANCELED)
				case "timed_out":
					statusFilters = append(statusFilters, enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT)
				case "continued_as_new":
					statusFilters = append(
						statusFilters,
						enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW,
					)
				case "unspecified":
					statusFilters = append(
						statusFilters,
						enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED,
					)
				}
			}
		}

		var query string
		if len(statusFilters) > 0 {
			var statusQueries []string
			for _, s := range statusFilters {
				statusQueries = append(statusQueries, fmt.Sprintf("ExecutionStatus=%d", s))
			}
			query = strings.Join(statusQueries, " or ")
		}

		list, err := c.ListWorkflow(
			context.Background(),
			&workflowservice.ListWorkflowExecutionsRequest{
				Namespace: namespace,
				Query:     query,
			},
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to list workflows",
				err.Error(),
			).JSON(e)
		}
		listJSON, err := protojson.Marshal(list)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to marshal workflow list",
				err.Error(),
			).JSON(e)
		}
		var resp ListMyChecksResponse
		err = json.Unmarshal(listJSON, &resp)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal workflow list",
				err.Error(),
			).JSON(e)
		}
		hierarchy := buildExecutionHierarchy(resp.Executions)
		resp.Executions = hierarchy
		return e.JSON(http.StatusOK, resp)
	}
}

func HandleGetMyCheckRun() func(*core.RequestEvent) error {
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

		checkID := e.Request.PathValue("checkId")
		runID := e.Request.PathValue("runId")
		if runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"runId",
				"runId is required",
				"missing runId",
			).JSON(e)
		}
		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization is empty",
				"missing organization",
			).JSON(e)
		}

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
			checkID,
			runID,
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
		weJSON, err := protojson.Marshal(workflowExecution)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to marshal workflow execution",
				err.Error(),
			).JSON(e)
		}
		finalJSON := make(map[string]interface{})
		err = json.Unmarshal(weJSON, &finalJSON)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal workflow execution",
				err.Error(),
			).JSON(e)
		}
		return e.JSON(http.StatusOK, finalJSON)
	}
}

func HandleGetMyCheckRunHistory() func(*core.RequestEvent) error {
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

		checkID := e.Request.PathValue("checkId")
		runID := e.Request.PathValue("runId")
		if checkID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"checkId and runId are required",
				"missing required parameters",
			).JSON(e)
		}

		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}

		historyIterator := c.GetWorkflowHistory(
			context.Background(),
			checkID,
			runID,
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		)

		var history []map[string]interface{}
		for historyIterator.HasNext() {
			event, err := historyIterator.Next()
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to get workflow history",
					err.Error(),
				).JSON(e)
			}
			eventJSON, err := protojson.Marshal(event)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to marshal workflow event",
					err.Error(),
				).JSON(e)
			}
			var eventMap map[string]interface{}
			err = json.Unmarshal(eventJSON, &eventMap)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to unmarshal workflow event",
					err.Error(),
				).JSON(e)
			}
			history = append(history, eventMap)
		}

		return e.JSON(http.StatusOK, map[string]interface{}{
			"history":   history,
			"count":     len(history),
			"time":      time.Now().Format(time.RFC3339),
			"checkId":   checkID,
			"runId":     runID,
			"namespace": namespace,
		})
	}
}

func HandleListMyCheckRuns() func(*core.RequestEvent) error {
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
		checkID := e.Request.PathValue("checkId")
		if checkID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"checkId",
				"checkId is required",
				"missing checkId parameter",
			).JSON(e)
		}
		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization is empty",
				"missing organization",
			).JSON(e)
		}
		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}

		list, err := c.ListWorkflow(
			context.Background(),
			&workflowservice.ListWorkflowExecutionsRequest{
				Namespace: namespace,
				Query:     fmt.Sprintf("WorkflowId = '%s'", checkID),
			},
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to list workflow executions",
				err.Error(),
			).JSON(e)
		}
		listJSON, err := protojson.Marshal(list)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to marshal workflow list",
				err.Error(),
			).JSON(e)
		}
		var resp ListMyCheckRunsResponse
		err = json.Unmarshal(listJSON, &resp)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal workflow list",
				err.Error(),
			).JSON(e)
		}
		hierarchy := buildExecutionHierarchy(resp.Executions)
		resp.Executions = hierarchy
		return e.JSON(http.StatusOK, resp)
	}
}

func HandleRerunMyCheck() func(*core.RequestEvent) error {
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

		checkID := e.Request.PathValue("checkId")
		runID := e.Request.PathValue("runId")
		if checkID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"checkId and runId are required",
				"missing required parameters",
			).JSON(e)
		}

		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}

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
			checkID,
			runID,
		)
		if err != nil {
			notFound := &serviceerror.NotFound{}
			if errors.As(err, &notFound) {
				return apierror.New(
					http.StatusNotFound,
					"workflow",
					"workflow execution not found",
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

		workflowName := workflowExecution.GetWorkflowExecutionInfo().GetType().GetName()

		workflowOptions := client.StartWorkflowOptions{
			TaskQueue: workflowExecution.GetWorkflowExecutionInfo().GetTaskQueue(),
			WorkflowRunTimeout: workflowExecution.GetExecutionConfig().
				GetWorkflowRunTimeout().
				AsDuration(),
			WorkflowExecutionTimeout: workflowExecution.GetExecutionConfig().
				GetWorkflowExecutionTimeout().
				AsDuration(),
		}

		workflowInput, err := getWorkflowInput(checkID, runID, c)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow input",
				err.Error(),
			).JSON(e)
		}

		var req ReRunCheckRequest
		req, err = routing.GetValidatedInput[ReRunCheckRequest](e)
		if err != nil {
			return err
		}
		if req.Config != nil {
			for k, v := range req.Config {
				workflowInput.Config[k] = v
			}
		}

		result, err := workflowengine.StartWorkflowWithOptions(
			namespace,
			workflowOptions,
			workflowName,
			workflowInput,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start workflow",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"workflow_id": result.WorkflowID,
			"run_id":      result.WorkflowRunID,
		})
	}
}

func HandleCancelMyCheckRun() func(*core.RequestEvent) error {
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

		checkID := e.Request.PathValue("checkId")
		runID := e.Request.PathValue("runId")
		if checkID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"checkId and runId are required",
				"missing required parameters",
			).JSON(e)
		}

		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}

		err = c.CancelWorkflow(context.Background(), checkID, runID)
		if err != nil {
			notFound := &serviceerror.NotFound{}
			if errors.As(err, &notFound) {
				return apierror.New(
					http.StatusNotFound,
					"workflow",
					"workflow execution not found",
					err.Error(),
				).JSON(e)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to cancel workflow execution",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"message":   "Workflow execution canceled successfully",
			"checkId":   checkID,
			"runId":     runID,
			"status":    "canceled",
			"time":      time.Now().Format(time.RFC3339),
			"namespace": namespace,
		})
	}
}

func HandleExportMyCheckRun() func(*core.RequestEvent) error {
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

		checkID := e.Request.PathValue("checkId")
		runID := e.Request.PathValue("runId")
		if checkID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"checkId and runId are required",
				"missing required parameters",
			).JSON(e)
		}

		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}

		workflowInput, err := getWorkflowInput(checkID, runID, c)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow input",
				err.Error(),
			).JSON(e)
		}
		if workflowInput.Config == nil {
			workflowInput.Config = make(map[string]interface{})
		}
		if workflowInput.Payload == nil {
			workflowInput.Payload = make(map[string]interface{})
		}
		exportData := map[string]interface{}{
			"checkId": checkID,
			"runId":   runID,
			"input":   workflowInput.Payload,
			"config":  workflowInput.Config,
		}

		return e.JSON(http.StatusOK, map[string]interface{}{
			"export": exportData,
		})
	}
}

func getWorkflowInput(
	checkID string,
	runID string,
	c client.Client,
) (workflowengine.WorkflowInput, error) {
	var workflowInput workflowengine.WorkflowInput
	historyIterator := c.GetWorkflowHistory(
		context.Background(),
		checkID,
		runID,
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)

	for historyIterator.HasNext() {
		event, err := historyIterator.Next()
		if err != nil {
			return workflowengine.WorkflowInput{}, fmt.Errorf(
				"failed to get workflow history: %w",
				err,
			)
		}

		if event.GetEventType() == enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED {
			startedAttributes := event.GetWorkflowExecutionStartedEventAttributes()
			if startedAttributes.GetInput() != nil {
				// Unmarshal the input payload
				inputJSON, err := protojson.Marshal(startedAttributes.GetInput())
				if err != nil {
					return workflowengine.WorkflowInput{}, fmt.Errorf(
						"failed to marshal workflow input: %w",
						err,
					)
				}
				var inputMap map[string]interface{}
				err = json.Unmarshal(inputJSON, &inputMap)
				if err != nil {
					return workflowengine.WorkflowInput{}, fmt.Errorf(
						"failed to unmarshal workflow input: %w",
						err,
					)
				}
				if payloads, ok := inputMap["payloads"]; ok {
					if payloadsSlice, ok := payloads.([]interface{}); ok && len(payloadsSlice) > 0 {
						if payloadMap, ok := payloadsSlice[0].(map[string]interface{}); ok {
							if data, ok := payloadMap["data"]; ok {
								if dataStr, ok := data.(string); ok {
									decodedData, err := base64.StdEncoding.DecodeString(dataStr)
									if err != nil {
										return workflowengine.WorkflowInput{}, fmt.Errorf(
											"failed to decode workflow input payload: %w", err,
										)
									}
									var payloadData map[string]interface{}
									err = json.Unmarshal(decodedData, &payloadData)
									if err != nil {
										return workflowengine.WorkflowInput{}, fmt.Errorf(
											"failed to unmarshal workflow input payload: %w", err,
										)
									}
									if payload, ok := payloadData["Payload"]; ok {
										if payloadMap, ok := payload.(map[string]interface{}); ok {
											workflowInput.Payload = payloadMap
										} else {
											return workflowengine.WorkflowInput{}, fmt.Errorf(
												"invalid workflow input payload format: payload is not a map",
											)
										}
									} else {
										return workflowengine.WorkflowInput{}, fmt.Errorf(
											"missing workflow input payload: payload field is missing in input data",
										)
									}
									if config, ok := payloadData["Config"]; ok {
										log.Println("Rerun workflow input config:", config)
										if configMap, ok := config.(map[string]interface{}); ok {
											workflowInput.Config = configMap
										} else {
											return workflowengine.WorkflowInput{}, fmt.Errorf(
												"invalid workflow input config format: config is not a map",
											)
										}
									} else {
										return workflowengine.WorkflowInput{}, fmt.Errorf(
											"missing workflow input config: config field is missing in input data",
										)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return workflowInput, nil
}

func HandleMyCheckLogs() func(*core.RequestEvent) error {
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

		checkID := e.Request.PathValue("checkId")
		runID := e.Request.PathValue("runId")
		if checkID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"checkId and runId are required",
				"missing required parameters",
			).JSON(e)
		}

		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization is empty",
				"missing organization",
			).JSON(e)
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}

		_, err = c.DescribeWorkflowExecution(context.Background(), checkID, runID)
		if err != nil {
			notFound := &serviceerror.NotFound{}
			if errors.As(err, &notFound) {
				return apierror.New(
					http.StatusNotFound,
					"workflow",
					"workflow execution not found",
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

		action := e.Request.URL.Query().Get("action")

		switch action {
		case "start":
			err = c.SignalWorkflow(context.Background(), checkID, runID, "start-logs", struct{}{})
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to send start logs signal",
					err.Error(),
				).JSON(e)
			}
		case "stop":
			err = c.SignalWorkflow(context.Background(), checkID, runID, "stop-logs", struct{}{})
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to send stop logs signal",
					err.Error(),
				).JSON(e)
			}
		}

		logsChannel := fmt.Sprintf("%s-logs", checkID)

		return e.JSON(http.StatusOK, map[string]interface{}{
			"channel":     logsChannel,
			"workflow_id": checkID,
			"run_id":      runID,
			"message":     "Logs streaming started",
			"status":      "started",
			"time":        time.Now().Format(time.RFC3339),
			"namespace":   namespace,
		})
	}
}

func HandleTerminateMyCheckRun() func(*core.RequestEvent) error {
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

		checkID := e.Request.PathValue("checkId")
		runID := e.Request.PathValue("runId")
		if checkID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"checkId and runId are required",
				"missing required parameters",
			).JSON(e)
		}

		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}

		err = c.TerminateWorkflow(context.Background(), checkID, runID, "Terminated by user", nil)
		if err != nil {
			notFound := &serviceerror.NotFound{}
			if errors.As(err, &notFound) {
				return apierror.New(
					http.StatusNotFound,
					"workflow",
					"workflow execution not found",
					err.Error(),
				).JSON(e)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to terminate workflow execution",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"message":   "Workflow execution terminated successfully",
			"checkId":   checkID,
			"runId":     runID,
			"status":    "terminated",
			"time":      time.Now().Format(time.RFC3339),
			"namespace": namespace,
		})
	}
}

var uuidRegex = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

func computeDisplayName(workflowID string) string {
	switch {
	case strings.HasPrefix(workflowID, "OpenIDNetCheckWorkflow"):
		return "View logs workflow"
	case strings.HasPrefix(workflowID, "EWCWorkflow"):
		return "View logs workflow"
	default:
		cleanID := uuidRegex.ReplaceAllString(workflowID, "")
		cleanID = strings.ReplaceAll(cleanID, "--", "-")
		cleanID = strings.Trim(cleanID, "-")

		return cleanID
	}
}

func buildExecutionHierarchy(executions []*WorkflowExecution) []*WorkflowExecution {
	// Create a map from RunID to *WorkflowExecution
	execMap := make(map[string]*WorkflowExecution)
	for i := range executions {
		execMap[executions[i].Execution.RunID] = executions[i]
	}

	var roots []*WorkflowExecution

	for _, exec := range execMap {
		if exec.ParentExecution != nil {
			if parent, ok := execMap[exec.ParentExecution.RunID]; ok {
				exec.DisplayName = computeDisplayName(exec.Execution.WorkflowID)
				parent.Children = append(parent.Children, exec)
				continue
			}
		}
		roots = append(roots, exec)
	}

	// Recursive sort by start time
	var sortFn func(list []*WorkflowExecution)
	sortFn = func(list []*WorkflowExecution) {
		sortExecutionsByStartTime(list)
		for _, e := range list {
			if len(e.Children) > 0 {
				sortFn(e.Children)
			}
		}
	}
	sortFn(roots)

	return roots
}

func sortExecutionsByStartTime(executions []*WorkflowExecution) {
	slices.SortFunc(executions, func(a, b *WorkflowExecution) int {
		return strings.Compare(b.StartTime, a.StartTime)
	})
}
