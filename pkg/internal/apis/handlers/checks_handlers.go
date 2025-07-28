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
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/workflowengine"

	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/encoding/protojson"
)

func HandleListMyChecks() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		namespace, err := GetUserOrganizationID(e.App, authRecord.Id)
		if err != nil {
			return err
		}
		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
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
					statusFilters = append(statusFilters, enums.WORKFLOW_EXECUTION_STATUS_TERMINATED)
				case "canceled":
					statusFilters = append(statusFilters, enums.WORKFLOW_EXECUTION_STATUS_CANCELED)
				case "timed_out":
					statusFilters = append(statusFilters, enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT)
				case "continued_as_new":
					statusFilters = append(statusFilters, enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW)
				case "unspecified":
					statusFilters = append(statusFilters, enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED)
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
			)
		}
		listJSON, err := protojson.Marshal(list)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to marshal workflow list",
				err.Error(),
			)
		}
		finalJSON := make(map[string]interface{})
		err = json.Unmarshal(listJSON, &finalJSON)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal workflow list",
				err.Error(),
			)
		}
		if finalJSON["executions"] == nil {
			finalJSON["executions"] = []any{}
		}
		executions, ok := (finalJSON["executions"]).([]any)
		if !ok {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"invalid executions data type",
				"executions field is not of expected type",
			)
		}
		finalJSON["executions"] = sortExecutionsByStartTime(executions)
		return e.JSON(http.StatusOK, finalJSON)
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
			)
		}

		checkID := e.Request.PathValue("checkId")
		runID := e.Request.PathValue("runId")
		if runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"runId",
				"runId is required",
				"missing runId",
			)
		}
		namespace, err := GetUserOrganizationID(e.App, authRecord.Id)
		if err != nil {
			return err
		}
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization is empty",
				"missing organization",
			)
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
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
				)
			}
			invalidArgument := &serviceerror.InvalidArgument{}
			if errors.As(err, &invalidArgument) {
				return apierror.New(
					http.StatusBadRequest,
					"workflow",
					"invalid workflow ID",
					err.Error(),
				)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to describe workflow execution",
				err.Error(),
			)
		}
		weJSON, err := protojson.Marshal(workflowExecution)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to marshal workflow execution",
				err.Error(),
			)
		}
		finalJSON := make(map[string]interface{})
		err = json.Unmarshal(weJSON, &finalJSON)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal workflow execution",
				err.Error(),
			)
		}
		return e.JSON(http.StatusOK, finalJSON)
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
			)
		}
		checkID := e.Request.PathValue("checkId")
		if checkID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"checkId",
				"checkId is required",
				"missing checkId parameter",
			)
		}
		namespace, err := GetUserOrganizationID(e.App, authRecord.Id)
		if err != nil {
			return err
		}
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization is empty",
				"missing organization",
			)
		}
		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
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
			)
		}
		listJSON, err := protojson.Marshal(list)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to marshal workflow list",
				err.Error(),
			)
		}
		finalJSON := make(map[string]interface{})
		err = json.Unmarshal(listJSON, &finalJSON)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal workflow list",
				err.Error(),
			)
		}
		if finalJSON["executions"] == nil {
			finalJSON["executions"] = []any{}
		}
		executions, ok := (finalJSON["executions"]).([]any)
		if !ok {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"invalid executions data type",
				"executions field is not of expected type",
			)
		}
		finalJSON["executions"] = sortExecutionsByStartTime(executions)
		return e.JSON(http.StatusOK, finalJSON)
	}
}

type ReRunCheckRequest struct {
	Config map[string]interface{} `json:"config"`
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
			)
		}

		checkID := e.Request.PathValue("checkId")
		runID := e.Request.PathValue("runId")
		if checkID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"checkId and runId are required",
				"missing required parameters",
			)
		}

		namespace, err := GetUserOrganizationID(e.App, authRecord.Id)
		if err != nil {
			return err
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
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
				)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to describe workflow execution",
				err.Error(),
			)
		}

		workflowName := workflowExecution.WorkflowExecutionInfo.Type.Name

		workflowOptions := client.StartWorkflowOptions{
			TaskQueue:                workflowExecution.WorkflowExecutionInfo.TaskQueue,
			WorkflowRunTimeout:       workflowExecution.ExecutionConfig.WorkflowRunTimeout.AsDuration(),
			WorkflowExecutionTimeout: workflowExecution.ExecutionConfig.WorkflowExecutionTimeout.AsDuration(),
		}

		workflowInput, err := getWorkflowInput(checkID, runID, c)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow input",
				err.Error(),
			)
		}

		var req ReRunCheckRequest
		req, err = routing.GetValidatedInput[ReRunCheckRequest](e)
		if err != nil {
			return err
		}
		if req.Config == nil {
			for k, v := range req.Config {
				workflowInput.Config[k] = v
			}
		}

		result, err := workflowengine.StartWorkflowWithOptions(workflowOptions, workflowName, workflowInput)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start workflow",
				err.Error(),
			)
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
			)
		}

		checkID := e.Request.PathValue("checkId")
		runID := e.Request.PathValue("runId")
		if checkID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"checkId and runId are required",
				"missing required parameters",
			)
		}

		namespace, err := GetUserOrganizationID(e.App, authRecord.Id)
		if err != nil {
			return err
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
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
				)
			}
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to cancel workflow execution",
				err.Error(),
			)
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
			)
		}

		checkID := e.Request.PathValue("checkId")
		runID := e.Request.PathValue("runId")
		if checkID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"checkId and runId are required",
				"missing required parameters",
			)
		}

		namespace, err := GetUserOrganizationID(e.App, authRecord.Id)
		if err != nil {
			return err
		}

		c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			)
		}

		workflowInput, err := getWorkflowInput(checkID, runID, c)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow input",
				err.Error(),
			)
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
		exportJSON, err := json.Marshal(exportData)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"export",
				"failed to marshal export data",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, map[string]interface{}{
			"export": string(exportJSON),
		})
	}
}

func getWorkflowInput(checkID string, runID string, c client.Client) (workflowengine.WorkflowInput, error) {
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
			return workflowengine.WorkflowInput{}, fmt.Errorf("failed to get workflow history: %w", err)
		}

		if event.EventType == enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED {
			startedAttributes := event.GetWorkflowExecutionStartedEventAttributes()
			if startedAttributes.Input != nil {
				// Unmarshal the input payload
				inputJSON, err := protojson.Marshal(startedAttributes.Input)
				if err != nil {
					return workflowengine.WorkflowInput{}, fmt.Errorf("failed to marshal workflow input: %w", err)
				}
				var inputMap map[string]interface{}
				err = json.Unmarshal(inputJSON, &inputMap)
				if err != nil {
					return workflowengine.WorkflowInput{}, fmt.Errorf("failed to unmarshal workflow input: %w", err)
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
