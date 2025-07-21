// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
)

// WorkflowExecution represents a temporal workflow execution (check run)
type WorkflowExecution struct {
	WorkflowID    string                 `json:"workflow_id"`
	RunID         string                 `json:"run_id"`
	WorkflowType  string                 `json:"workflow_type"`
	Status        string                 `json:"status"`
	StartTime     time.Time              `json:"start_time"`
	CloseTime     *time.Time             `json:"close_time"`
	ExecutionTime *int64                 `json:"execution_time_ms"`
	Memo          map[string]interface{} `json:"memo"`
	Input         map[string]interface{} `json:"input"`
	Result        map[string]interface{} `json:"result"`
	TaskQueue     string                 `json:"task_queue"`
	Namespace     string                 `json:"namespace"`
}

// Check represents an available workflow type (check template)
type Check struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	IsActive    bool   `json:"is_active"`
	UserID      string `json:"user_id"`
	OrgID       string `json:"org_id"`
}

// Available workflow types in the system
var availableWorkflows = []Check{
	{
		ID:          "openidnet",
		Name:        "OpenID Net Conformance Check",
		Description: "Conformance check on https://www.certification.openid.net",
		Type:        "OpenIDNetWorkflow",
		IsActive:    true,
	},
	{
		ID:          "ewc",
		Name:        "EWC Conformance Check",
		Description: "Conformance check on EWC",
		Type:        "EWCWorkflow",
		IsActive:    true,
	},
	{
		ID:          "eudiw",
		Name:        "EUDIW Conformance Check",
		Description: "Conformance check on EUDIW",
		Type:        "EudiwWorkflow",
		IsActive:    true,
	},
	{
		ID:          "zenroom",
		Name:        "Zenroom Contract Execution",
		Description: "Run a Zenroom contract from the docker image",
		Type:        "ZenroomWorkflow",
		IsActive:    true,
	},
	{
		ID:          "custom",
		Name:        "Custom Check Workflow",
		Description: "Custom Check Workflow",
		Type:        "CustomCheckWorkflow",
		IsActive:    true,
	},
	{
		ID:          "credentials",
		Name:        "Credentials Issuers Workflow",
		Description: "Validate and import Credential Issuer metadata",
		Type:        "CredentialsIssuersWorkflow",
		IsActive:    true,
	},
}

// HandleListMyChecks returns all available check types for the authenticated user
func HandleListMyChecks() func(*core.RequestEvent) error {
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

		namespace, err := GetUserOrganizationID(e.App, authRecord.Id)
		if err != nil {
			return err
		}

		// Add user and org context to each check
		checks := make([]Check, len(availableWorkflows))
		for i, check := range availableWorkflows {
			checks[i] = check
			checks[i].UserID = authRecord.Id
			checks[i].OrgID = namespace
		}

		return e.JSON(http.StatusOK, map[string]interface{}{
			"checks": checks,
			"total":  len(checks),
		})
	}
}

// HandleListMyCheckRuns returns all workflow executions (runs) for a specific check type
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

		// Find the check type
		var checkType string
		for _, check := range availableWorkflows {
			if check.ID == checkID {
				checkType = check.Type
				break
			}
		}
		if checkType == "" {
			return apierror.New(
				http.StatusNotFound,
				"check",
				"check type not found",
				"invalid checkId",
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
		defer c.Close()

		// Query workflows of this type
		query := fmt.Sprintf("WorkflowType = '%s'", checkType)
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

		var runs []WorkflowExecution
		if list.Executions != nil {
			for _, exec := range list.Executions {
				run := WorkflowExecution{
					WorkflowID:   exec.Execution.WorkflowId,
					RunID:        exec.Execution.RunId,
					WorkflowType: exec.Type.Name,
					Status:       exec.Status.String(),
					TaskQueue:    exec.TaskQueue,
					Namespace:    namespace,
				}

				if exec.StartTime != nil {
					run.StartTime = exec.StartTime.AsTime()
				}
				if exec.CloseTime != nil {
					closeTime := exec.CloseTime.AsTime()
					run.CloseTime = &closeTime
					run.ExecutionTime = func() *int64 {
						duration := closeTime.Sub(run.StartTime).Milliseconds()
						return &duration
					}()
				}

				if exec.Memo != nil && exec.Memo.Fields != nil {
					run.Memo = make(map[string]interface{})
					for k, v := range exec.Memo.Fields {
						if v.GetStringValue() != "" {
							run.Memo[k] = v.GetStringValue()
						}
					}
				}

				runs = append(runs, run)
			}
		}

		return e.JSON(http.StatusOK, map[string]interface{}{
			"runs":  runs,
			"total": len(runs),
		})
	}
}

// HandleGetMyCheckRun returns details of a specific workflow execution (check run)
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
		defer c.Close()

		// Get workflow execution details
		workflowExecution, err := c.DescribeWorkflowExecution(
			context.Background(),
			runID, // In Temporal, the runId is actually the workflowId for this case
			"",    // Use empty string for runId to get the latest run
		)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"workflow",
				"workflow execution not found",
				err.Error(),
			)
		}

		// Convert to our format
		exec := workflowExecution.WorkflowExecutionInfo
		run := WorkflowExecution{
			WorkflowID:   exec.Execution.WorkflowId,
			RunID:        exec.Execution.RunId,
			WorkflowType: exec.Type.Name,
			Status:       exec.Status.String(),
			TaskQueue:    exec.TaskQueue,
			Namespace:    namespace,
		}

		if exec.StartTime != nil {
			run.StartTime = exec.StartTime.AsTime()
		}
		if exec.CloseTime != nil {
			closeTime := exec.CloseTime.AsTime()
			run.CloseTime = &closeTime
			run.ExecutionTime = func() *int64 {
				duration := closeTime.Sub(run.StartTime).Milliseconds()
				return &duration
			}()
		}

		if exec.Memo != nil && exec.Memo.Fields != nil {
			run.Memo = make(map[string]interface{})
			for k, v := range exec.Memo.Fields {
				if v.GetStringValue() != "" {
					run.Memo[k] = v.GetStringValue()
				}
			}
		}

		return e.JSON(http.StatusOK, run)
	}
}

// RunCheckRequest represents the request to run a check
type RunCheckRequest struct {
	Config map[string]interface{} `json:"config"`
	Input  map[string]interface{} `json:"input"`
}

// HandleRunMyCheck starts a new workflow execution for a specific check type
func HandleRunMyCheck() func(*core.RequestEvent) error {
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

		req, err := routing.GetValidatedInput[RunCheckRequest](e)
		if err != nil {
			return err
		}

		namespace, err := GetUserOrganizationID(e.App, authRecord.Id)
		if err != nil {
			return err
		}

		// Find the check type
		var check *Check
		for _, c := range availableWorkflows {
			if c.ID == checkID {
				check = &c
				break
			}
		}
		if check == nil {
			return apierror.New(
				http.StatusNotFound,
				"check",
				"check type not found",
				"invalid checkId",
			)
		}

		// Create workflow input
		workflowInput := workflowengine.WorkflowInput{
			Payload: req.Input,
			Config: map[string]any{
				"namespace": namespace,
				"app_url":   e.App.Settings().Meta.AppURL,
			},
		}

		// Add additional config from request
		for k, v := range req.Config {
			workflowInput.Config[k] = v
		}

		// Start the appropriate workflow based on check type
		var result workflowengine.WorkflowResult
		switch checkID {
		case "openidnet":
			workflow := workflows.OpenIDNetWorkflow{}
			result, err = workflow.Start(workflowInput)
		case "ewc":
			workflow := workflows.EWCWorkflow{}
			result, err = workflow.Start(workflowInput)
		case "eudiw":
			workflow := workflows.EudiwWorkflow{}
			result, err = workflow.Start(workflowInput)
		case "zenroom":
			workflow := workflows.ZenroomWorkflow{}
			result, err = workflow.Start(workflowInput)
		case "custom":
			workflow := workflows.CustomCheckWorkflow{}
			result, err = workflow.Start(workflowInput)
		case "credentials":
			workflow := workflows.CredentialsIssuersWorkflow{}
			result, err = workflow.Start(workflowInput)
		default:
			return apierror.New(
				http.StatusBadRequest,
				"check",
				"unsupported check type",
				"check type not implemented",
			)
		}

		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start workflow",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, map[string]interface{}{
			"message":     "Check started successfully",
			"workflow_id": result.WorkflowID,
			"run_id":      result.WorkflowRunID,
			"namespace":   namespace,
		})
	}
}

// HandleRerunMyCheck reruns a workflow execution based on a previous run
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
		defer c.Close()

		// Get the original workflow execution to extract input
		workflowExecution, err := c.DescribeWorkflowExecution(
			context.Background(),
			runID,
			"",
		)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"workflow",
				"original workflow execution not found",
				err.Error(),
			)
		}

		// Find the check type
		var check *Check
		for _, c := range availableWorkflows {
			if c.ID == checkID {
				check = &c
				break
			}
		}
		if check == nil {
			return apierror.New(
				http.StatusNotFound,
				"check",
				"check type not found",
				"invalid checkId",
			)
		}

		// Create workflow input based on original execution
		workflowInput := workflowengine.WorkflowInput{
			Payload: make(map[string]any),
			Config: map[string]any{
				"namespace": namespace,
				"app_url":   e.App.Settings().Meta.AppURL,
			},
		}

		// Extract memo and use as config
		if workflowExecution.WorkflowExecutionInfo.Memo != nil && workflowExecution.WorkflowExecutionInfo.Memo.Fields != nil {
			for k, v := range workflowExecution.WorkflowExecutionInfo.Memo.Fields {
				if v.GetStringValue() != "" {
					workflowInput.Config[k] = v.GetStringValue()
				}
			}
		}

		// Start the rerun workflow
		var result workflowengine.WorkflowResult
		switch checkID {
		case "openidnet":
			workflow := workflows.OpenIDNetWorkflow{}
			result, err = workflow.Start(workflowInput)
		case "ewc":
			workflow := workflows.EWCWorkflow{}
			result, err = workflow.Start(workflowInput)
		case "eudiw":
			workflow := workflows.EudiwWorkflow{}
			result, err = workflow.Start(workflowInput)
		case "zenroom":
			workflow := workflows.ZenroomWorkflow{}
			result, err = workflow.Start(workflowInput)
		case "custom":
			workflow := workflows.CustomCheckWorkflow{}
			result, err = workflow.Start(workflowInput)
		case "credentials":
			workflow := workflows.CredentialsIssuersWorkflow{}
			result, err = workflow.Start(workflowInput)
		default:
			return apierror.New(
				http.StatusBadRequest,
				"check",
				"unsupported check type",
				"check type not implemented",
			)
		}

		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start rerun workflow",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, map[string]interface{}{
			"message":     "Check rerun started successfully",
			"workflow_id": result.WorkflowID,
			"run_id":      result.WorkflowRunID,
			"rerun_from":  runID,
			"namespace":   namespace,
		})
	}
}

// HandleCancelMyCheckRun cancels a running workflow execution
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
		defer c.Close()

		// Cancel the workflow
		err = c.CancelWorkflow(context.Background(), runID, "")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to cancel workflow",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, map[string]interface{}{
			"message": "Check run cancelled successfully",
		})
	}
}

// HandleTailMyCheckLogs returns logs for a specific workflow execution
func HandleTailMyCheckLogs() func(*core.RequestEvent) error {
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
		defer c.Close()

		// Get workflow history for logs
		historyIterator := c.GetWorkflowHistory(
			context.Background(),
			runID,
			"",
			false,
			enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		)

		var logs []map[string]interface{}
		for historyIterator.HasNext() {
			event, err := historyIterator.Next()
			if err != nil {
				break
			}

			// Convert history event to log entry
			logEntry := map[string]interface{}{
				"timestamp":  event.EventTime.AsTime().Format(time.RFC3339),
				"event_type": event.EventType.String(),
				"event_id":   event.EventId,
			}

			// Add specific details based on event type
			switch event.EventType {
			case enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED:
				logEntry["message"] = "Workflow execution started"
			case enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
				logEntry["message"] = "Workflow execution completed"
			case enums.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
				logEntry["message"] = "Workflow execution failed"
			case enums.EVENT_TYPE_ACTIVITY_TASK_STARTED:
				logEntry["message"] = "Activity task started"
			case enums.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
				logEntry["message"] = "Activity task completed"
			case enums.EVENT_TYPE_ACTIVITY_TASK_FAILED:
				logEntry["message"] = "Activity task failed"
			default:
				logEntry["message"] = strings.ReplaceAll(event.EventType.String(), "_", " ")
			}

			logs = append(logs, logEntry)
		}

		// Set up Server-Sent Events for real-time logs
		e.Response.Header().Set("Content-Type", "text/event-stream")
		e.Response.Header().Set("Cache-Control", "no-cache")
		e.Response.Header().Set("Connection", "keep-alive")
		e.Response.Header().Set("Access-Control-Allow-Origin", "*")

		// Send existing logs
		for _, log := range logs {
			logJSON, _ := json.Marshal(log)
			fmt.Fprintf(e.Response, "data: %s\n\n", logJSON)
		}

		return nil
	}
}

// HandleExportMyCheckRun exports the workflow execution details
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
		defer c.Close()

		// Get workflow execution details
		workflowExecution, err := c.DescribeWorkflowExecution(
			context.Background(),
			runID,
			"",
		)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"workflow",
				"workflow execution not found",
				err.Error(),
			)
		}

		// Find the check info
		var checkInfo *Check
		for _, c := range availableWorkflows {
			if c.ID == checkID {
				checkInfo = &c
				break
			}
		}

		// Prepare export data
		exportData := map[string]interface{}{
			"check": map[string]interface{}{
				"id": checkID,
				"name": func() string {
					if checkInfo != nil {
						return checkInfo.Name
					}
					return "Unknown Check"
				}(),
				"type": workflowExecution.WorkflowExecutionInfo.Type.Name,
			},
			"workflow": map[string]interface{}{
				"workflow_id": workflowExecution.WorkflowExecutionInfo.Execution.WorkflowId,
				"run_id":      workflowExecution.WorkflowExecutionInfo.Execution.RunId,
				"status":      workflowExecution.WorkflowExecutionInfo.Status.String(),
				"start_time":  workflowExecution.WorkflowExecutionInfo.StartTime.AsTime().Format(time.RFC3339),
			},
			"exported_at": time.Now().Format(time.RFC3339),
			"namespace":   namespace,
		}

		if workflowExecution.WorkflowExecutionInfo.CloseTime != nil {
			exportData["workflow"].(map[string]interface{})["close_time"] = workflowExecution.WorkflowExecutionInfo.CloseTime.AsTime().Format(time.RFC3339)
		}

		// Add memo as config
		if workflowExecution.WorkflowExecutionInfo.Memo != nil && workflowExecution.WorkflowExecutionInfo.Memo.Fields != nil {
			config := make(map[string]interface{})
			for k, v := range workflowExecution.WorkflowExecutionInfo.Memo.Fields {
				if v.GetStringValue() != "" {
					config[k] = v.GetStringValue()
				}
			}
			exportData["config"] = config
		}

		// Set headers for file download
		filename := fmt.Sprintf("workflow-export-%s-%s.json", checkID, runID)
		e.Response.Header().Set("Content-Type", "application/json")
		e.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

		return e.JSON(http.StatusOK, exportData)
	}
}

// ScheduleCheckRequest represents the request to schedule a check
type ScheduleCheckRequest struct {
	CronExpression string                 `json:"cron_expression" validate:"required"`
	Config         map[string]interface{} `json:"config"`
	Input          map[string]interface{} `json:"input"`
	IsActive       bool                   `json:"is_active"`
}

// HandleScheduleMyCheck schedules a check to run at specified intervals (not implemented for Temporal workflows)
func HandleScheduleMyCheck() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return apierror.New(
			http.StatusNotImplemented,
			"schedule",
			"scheduling not yet implemented for workflow-based checks",
			"use temporal schedules directly",
		)
	}
}

// UpdateWorkflowRequest represents the request to update a workflow file
type UpdateWorkflowRequest struct {
	WorkflowFile   string                 `json:"workflow_file" validate:"required"`
	WorkflowConfig map[string]interface{} `json:"workflow_config"`
	Description    string                 `json:"description"`
}

// HandleUpdateMyCheckWorkflow updates the workflow file for a check (not implemented)
func HandleUpdateMyCheckWorkflow() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return apierror.New(
			http.StatusNotImplemented,
			"workflow",
			"workflow updates not yet implemented",
			"workflows are defined in code",
		)
	}
}
