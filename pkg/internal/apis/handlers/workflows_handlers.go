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
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/encoding/protojson"
)

var WorkflowsRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL: "/api/my/workflows",
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodGet,
			Path:           "/{workflowId}/runs",
			OperationID:    "workflowRuns.list",
			Handler:        HandleListMyWorkflowRuns,
			ResponseSchema: ListMyWorkflowRunsResponse{},
			Description:    "List all runs for a specific workflow",
			Summary:        "Get a list of all runs for a specific workflow",
		},
		{
			Method:         http.MethodGet,
			Path:           "/{workflowId}/runs/{runId}",
			OperationID:    "workflowRun.get",
			Handler:        HandleGetMyWorkflowRun,
			ResponseSchema: GetMyWorkflowRunResponse{},
			Description:    "Get details of a specific run for a workflow",
			Summary:        "Get details of a specific run for a workflow",
		},
		{
			Method:         http.MethodGet,
			Path:           "/{workflowId}/runs/{runId}/history",
			OperationID:    "workflowRun.history",
			Handler:        HandleGetMyWorkflowRunHistory,
			ResponseSchema: GetMyWorkflowRunHistoryResponse{},
			Description:    "Get the history of events for a specific run of a workflow",
			Summary:        "Get the history of events for a specific run of a workflow",
		},
		{
			Method:         http.MethodPost,
			Path:           "/{workflowId}/runs/{runId}/rerun",
			OperationID:    "workflowRun.rerun",
			Handler:        HandleRerunMyWorkflow,
			RequestSchema:  ReRunWorkflowRequest{},
			ResponseSchema: ReRunWorkflowResponse{},
			Description:    "Re-run a specific workflow run",
			Summary:        "Re-run a specific workflow run",
		},
		{
			Method:         http.MethodPost,
			Path:           "/{workflowId}/runs/{runId}/cancel",
			OperationID:    "workflowRun.cancel",
			Handler:        HandleCancelMyWorkflowRun,
			ResponseSchema: CancelMyWorkflowRunResponse{},
			Description:    "Cancel a specific workflow run",
			Summary:        "Cancel a specific workflow run",
		},
		{
			Method:         http.MethodGet,
			Path:           "/{workflowId}/runs/{runId}/export",
			OperationID:    "workflowRun.export",
			Handler:        HandleExportMyWorkflowRun,
			ResponseSchema: ExportMyWorkflowRunResponse{},
			Description:    "Export a specific workflow run",
			Summary:        "Export a specific workflow run",
		},
		{
			Method:         http.MethodGet,
			Path:           "/{workflowId}/runs/{runId}/logs",
			OperationID:    "workflowRun.logs",
			Handler:        HandleMyWorkflowLogs,
			ResponseSchema: WorkflowLogsResponse{},
			Description:    "Start or Stop logs for a specific workflow run and get the log channel",
			Summary:        "Start or Stop logs for a specific workflow run",
			QuerySearchAttributes: []routing.QuerySearchAttribute{
				{
					Name:        "action",
					Required:    false,
					Description: "Can be 'start' or 'stop' to control logging for the workflow run",
				},
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "/{workflowId}/runs/{runId}/terminate",
			OperationID:    "workflowRun.terminate",
			Handler:        HandleTerminateMyWorkflowRun,
			ResponseSchema: TerminateMyWorkflowRunResponse{},
			Description:    "Terminate a specific workflow run",
			Summary:        "Terminate a specific workflow run",
		},
	},
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	AuthenticationRequired: true,
}

var WorkflowListingRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL: "/api",
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodGet,
			Path:           "/list-workflows",
			OperationID:    "workflows.list",
			Handler:        HandleListMyWorkflows,
			ResponseSchema: ListMyWorkflowsResponse{},
			Description:    "List non-pipeline workflows for the authenticated user",
			Summary:        "Get a list of non-pipeline workflows for the authenticated user",
		},
	},
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	AuthenticationRequired: true,
}

type ReRunWorkflowRequest struct {
	Config map[string]interface{} `json:"config"`
}

var listWorkflowsTemporalClient = temporalclient.GetTemporalClientWithNamespace
var listWorkflows = listWorkflowsTemporal
var workflowTemporalClient = temporalclient.GetTemporalClientWithNamespace
var workflowRunInputGetter = getWorkflowInput
var workflowStartWithOptions = workflowengine.StartWorkflowWithOptions

func HandleListMyWorkflows() func(*core.RequestEvent) error {
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

		namespace, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}

		limit, pageNum := parsePaginationParams(e, 20, 0)
		offset := pageNum * limit

		c, err := listWorkflowsTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}
		statusParam := e.Request.URL.Query().Get("status")
		statusFilters, statusOK := parseWorkflowStatusFilters(statusParam)
		if statusParam != "" && !statusOK {
			return e.JSON(http.StatusOK, ListMyWorkflowsResponse{
				Executions: []*WorkflowExecutionSummary{},
			})
		}

		query := buildWorkflowStatusQuery(statusFilters)

		list, err := listWorkflows(context.Background(), c, namespace, query)
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
		var execs struct {
			Executions []*WorkflowExecution `json:"executions"`
		}
		err = json.Unmarshal(listJSON, &execs)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal workflow list",
				err.Error(),
			).JSON(e)
		}
		owner, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"unable to get user organization ca",
				err.Error(),
			).JSON(e)
		}

		filteredExecutions, err := filterNonPipelineExecutions(
			e.Request.Context(),
			c,
			execs.Executions,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to resolve workflow parents",
				err.Error(),
			).JSON(e)
		}

		hierarchy := buildExecutionHierarchy(
			e.App,
			filteredExecutions,
			owner,
			authRecord.GetString("Timezone"),
			c,
		)
		hierarchy = paginateWorkflowExecutionSummaries(hierarchy, limit, offset)

		resp := ListMyWorkflowsResponse{}
		resp.Executions = hierarchy
		return e.JSON(http.StatusOK, resp)
	}
}

func buildWorkflowStatusQuery(statusFilters []enums.WorkflowExecutionStatus) string {
	if len(statusFilters) == 0 {
		return ""
	}

	statusQueries := make([]string, 0, len(statusFilters))
	for _, status := range statusFilters {
		statusQueries = append(statusQueries, fmt.Sprintf("ExecutionStatus=%d", status))
	}

	return strings.Join(statusQueries, " or ")
}

func filterNonPipelineExecutions(
	ctx context.Context,
	temporalClient client.Client,
	executions []*WorkflowExecution,
) ([]*WorkflowExecution, error) {
	pipelineWorkflowName := pipeline.NewPipelineWorkflow().Name()
	executionByRunID := make(map[string]*WorkflowExecution, len(executions))
	for _, exec := range executions {
		if exec == nil || exec.Execution == nil {
			continue
		}
		executionByRunID[exec.Execution.RunID] = exec
	}

	filtered := make([]*WorkflowExecution, 0, len(executions))
	for _, exec := range executions {
		if exec == nil || exec.Execution == nil {
			continue
		}
		if exec.Type.Name == pipelineWorkflowName {
			continue
		}
		if exec.ParentExecution == nil {
			filtered = append(filtered, exec)
			continue
		}

		parent, ok := executionByRunID[exec.ParentExecution.RunID]
		if ok {
			if parent != nil && parent.Type.Name == pipelineWorkflowName {
				continue
			}
			filtered = append(filtered, exec)
			continue
		}

		parentType, err := getWorkflowTypeName(
			ctx,
			temporalClient,
			exec.ParentExecution.WorkflowID,
			exec.ParentExecution.RunID,
		)
		if err != nil {
			return nil, err
		}
		if parentType == pipelineWorkflowName {
			continue
		}

		filtered = append(filtered, exec)
	}

	return filtered, nil
}

func getWorkflowTypeName(
	ctx context.Context,
	temporalClient client.Client,
	workflowID string,
	runID string,
) (string, error) {
	if temporalClient == nil || workflowID == "" {
		return "", nil
	}

	description, err := temporalClient.DescribeWorkflowExecution(ctx, workflowID, runID)
	if err != nil {
		return "", err
	}
	if description == nil || description.GetWorkflowExecutionInfo() == nil ||
		description.GetWorkflowExecutionInfo().GetType() == nil {
		return "", nil
	}

	return description.GetWorkflowExecutionInfo().GetType().GetName(), nil
}

func paginateWorkflowExecutionSummaries(
	summaries []*WorkflowExecutionSummary,
	limit int,
	offset int,
) []*WorkflowExecutionSummary {
	if len(summaries) == 0 || limit <= 0 || offset >= len(summaries) {
		return []*WorkflowExecutionSummary{}
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

func listWorkflowsTemporal(
	ctx context.Context,
	c client.Client,
	namespace string,
	query string,
) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	return c.ListWorkflow(
		ctx,
		&workflowservice.ListWorkflowExecutionsRequest{
			Namespace: namespace,
			Query:     query,
		},
	)
}

func HandleGetMyWorkflowRun() func(*core.RequestEvent) error {
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

		workflowID := e.Request.PathValue("workflowId")
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

		c, err := workflowTemporalClient(namespace)
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
			workflowID,
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

func HandleGetMyWorkflowRunHistory() func(*core.RequestEvent) error {
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

		workflowID := e.Request.PathValue("workflowId")
		runID := e.Request.PathValue("runId")
		if workflowID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"workflowId and runId are required",
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

		c, err := workflowTemporalClient(namespace)
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
			workflowID,
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
			"history":    history,
			"count":      len(history),
			"time":       time.Now().Format(time.RFC3339),
			"workflowId": workflowID,
			"runId":      runID,
			"namespace":  namespace,
		})
	}
}

func HandleListMyWorkflowRuns() func(*core.RequestEvent) error {
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
		workflowID := e.Request.PathValue("workflowId")
		if workflowID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"workflowId",
				"workflowId is required",
				"missing workflowId parameter",
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
		c, err := workflowTemporalClient(namespace)
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
				Query:     fmt.Sprintf("WorkflowId = '%s'", workflowID),
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
		var execs struct {
			Executions []*WorkflowExecution `json:"executions"`
		}

		err = json.Unmarshal(listJSON, &execs)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to unmarshal workflow list",
				err.Error(),
			).JSON(e)
		}

		owner, err := GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"unable to get user organization canonified name",
				err.Error(),
			).JSON(e)
		}
		hierarchy := buildExecutionHierarchy(
			e.App,
			execs.Executions,
			owner,
			authRecord.GetString("Timezone"),
			c,
		)

		var resp ListMyWorkflowRunsResponse
		resp.Executions = hierarchy
		return e.JSON(http.StatusOK, resp)
	}
}

func HandleRerunMyWorkflow() func(*core.RequestEvent) error {
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

		workflowID := e.Request.PathValue("workflowId")
		runID := e.Request.PathValue("runId")
		if workflowID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"workflowId and runId are required",
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

		c, err := workflowTemporalClient(namespace)
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
			workflowID,
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

		workflowInput, err := workflowRunInputGetter(workflowID, runID, c)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow input",
				err.Error(),
			).JSON(e)
		}

		var req ReRunWorkflowRequest
		req, err = routing.GetValidatedInput[ReRunWorkflowRequest](e)
		if err != nil {
			return err
		}
		if req.Config != nil {
			for k, v := range req.Config {
				workflowInput.Config[k] = v
			}
		}

		result, err := workflowStartWithOptions(
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

func HandleCancelMyWorkflowRun() func(*core.RequestEvent) error {
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

		workflowID := e.Request.PathValue("workflowId")
		runID := e.Request.PathValue("runId")
		if workflowID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"workflowId and runId are required",
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

		c, err := workflowTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}

		err = c.CancelWorkflow(context.Background(), workflowID, runID)
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
			"message":    "Workflow execution canceled successfully",
			"workflowId": workflowID,
			"runId":      runID,
			"status":     statusStringCanceled,
			"time":       time.Now().Format(time.RFC3339),
			"namespace":  namespace,
		})
	}
}

func HandleExportMyWorkflowRun() func(*core.RequestEvent) error {
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

		workflowID := e.Request.PathValue("workflowId")
		runID := e.Request.PathValue("runId")
		if workflowID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"workflowId and runId are required",
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

		c, err := workflowTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}

		workflowInput, err := workflowRunInputGetter(workflowID, runID, c)
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
			"workflowId": workflowID,
			"runId":      runID,
			"input":      workflowInput.Payload,
			"config":     workflowInput.Config,
		}

		return e.JSON(http.StatusOK, map[string]interface{}{
			"export": exportData,
		})
	}
}

func getWorkflowInput(
	workflowID string,
	runID string,
	c client.Client,
) (workflowengine.WorkflowInput, error) {
	var workflowInput workflowengine.WorkflowInput
	historyIterator := c.GetWorkflowHistory(
		context.Background(),
		workflowID,
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

func HandleMyWorkflowLogs() func(*core.RequestEvent) error {
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

		workflowID := e.Request.PathValue("workflowId")
		runID := e.Request.PathValue("runId")
		if workflowID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"workflowId and runId are required",
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

		c, err := workflowTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}

		_, err = c.DescribeWorkflowExecution(context.Background(), workflowID, runID)
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
			err = c.SignalWorkflow(
				context.Background(),
				workflowID,
				runID,
				"start-logs",
				struct{}{},
			)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to send start logs signal",
					err.Error(),
				).JSON(e)
			}
		case "stop":
			err = c.SignalWorkflow(context.Background(), workflowID, runID, "stop-logs", struct{}{})
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"workflow",
					"failed to send stop logs signal",
					err.Error(),
				).JSON(e)
			}
		}

		logsChannel := fmt.Sprintf("%s-logs", workflowID)

		return e.JSON(http.StatusOK, map[string]interface{}{
			"channel":     logsChannel,
			"workflow_id": workflowID,
			"run_id":      runID,
			"message":     "Logs streaming started",
			"status":      "started",
			"time":        time.Now().Format(time.RFC3339),
			"namespace":   namespace,
		})
	}
}

func HandleTerminateMyWorkflowRun() func(*core.RequestEvent) error {
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

		workflowID := e.Request.PathValue("workflowId")
		runID := e.Request.PathValue("runId")
		if workflowID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"workflowId and runId are required",
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

		c, err := workflowTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create client",
				err.Error(),
			).JSON(e)
		}

		err = c.TerminateWorkflow(
			context.Background(),
			workflowID,
			runID,
			"Terminated by user",
			nil,
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
				"failed to terminate workflow execution",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"message":    "Workflow execution terminated successfully",
			"workflowId": workflowID,
			"runId":      runID,
			"status":     statusStringTerminated,
			"time":       time.Now().Format(time.RFC3339),
			"namespace":  namespace,
		})
	}
}

var uuidRegex = regexp.MustCompile(
	`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`,
)

func computeChildDisplayName(workflowID string) string {
	switch {
	case strings.HasPrefix(workflowID, "OpenIDNetCheckWorkflow"):
		return "View logs workflow"
	case strings.HasPrefix(workflowID, "EWCWorkflow"):
		return "View logs workflow"
	default:
		loc := uuidRegex.FindStringIndex(workflowID)
		if loc != nil && loc[1] < len(workflowID) {
			return workflowID[loc[1]+1:]
		}
		return workflowID
	}
}

func buildExecutionHierarchy(
	app core.App,
	executions []*WorkflowExecution,
	owner string,
	userTimezone string,
	c client.Client,
) []*WorkflowExecutionSummary {
	loc, err := time.LoadLocation(userTimezone)
	if err != nil {
		loc = time.Local
	}
	summaryMap := make(map[string]*WorkflowExecutionSummary)
	for _, exec := range executions {
		summary := &WorkflowExecutionSummary{
			Execution: exec.Execution,
			Type:      exec.Type,
			StartTime: exec.StartTime,
			EndTime:   exec.CloseTime,
			Duration:  calculateDuration(exec.StartTime, exec.CloseTime),
			Status:    normalizeTemporalStatus(exec.Status),
		}

		if enums.WorkflowExecutionStatus(
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

		summaryMap[exec.Execution.RunID] = summary
	}

	roots := make([]*WorkflowExecutionSummary, 0, len(executions))

	for _, exec := range executions {
		current := summaryMap[exec.Execution.RunID]

		if exec.ParentExecution != nil {
			if parent, ok := summaryMap[exec.ParentExecution.RunID]; ok {
				current.DisplayName = computeChildDisplayName(exec.Execution.WorkflowID)
				parent.Children = append(parent.Children, current)
				continue
			}
		}

		var parentDisplay string
		if exec.Memo != nil {
			if field, ok := exec.Memo.Fields["test"]; ok {
				parentDisplay = DecodeFromTemporalPayload(*field.Data)
			}
		}

		w := pipeline.PipelineWorkflow{}
		if current.Type.Name == w.Name() {
			results := computePipelineResults(
				app,
				owner,
				exec.Execution.WorkflowID,
				exec.Execution.RunID,
			)

			current.Results = results
		}
		current.DisplayName = parentDisplay
		roots = append(roots, current)
	}

	sortExecutionSummaries(roots, loc, false)

	return roots
}

func sortExecutionSummaries(list []*WorkflowExecutionSummary, loc *time.Location, ascending bool) {
	sortWorkflowExecutionSummaries(list, ascending)
	localizeWorkflowExecutionSummaries(list, loc)
}

func sortWorkflowExecutionSummaries(list []*WorkflowExecutionSummary, ascending bool) {
	slices.SortFunc(list, func(a, b *WorkflowExecutionSummary) int {
		t1, _ := time.Parse(time.RFC3339, a.StartTime)
		t2, _ := time.Parse(time.RFC3339, b.StartTime)
		if ascending {
			if t1.Before(t2) {
				return -1
			}
			if t1.After(t2) {
				return 1
			}
			return 0
		}
		if t1.After(t2) {
			return -1
		}
		if t1.Before(t2) {
			return 1
		}
		return 0
	})

	for _, e := range list {
		if len(e.Children) > 0 {
			sortWorkflowExecutionSummaries(e.Children, !ascending)
		}
	}
}

func localizeWorkflowExecutionSummaries(list []*WorkflowExecutionSummary, loc *time.Location) {
	for _, e := range list {
		if t, err := time.Parse(time.RFC3339, e.StartTime); err == nil {
			e.StartTime = t.In(loc).Format("02/01/2006, 15:04:05")
		}
		if t, err := time.Parse(time.RFC3339, e.EndTime); err == nil {
			e.EndTime = t.In(loc).Format("02/01/2006, 15:04:05")
		}

		if len(e.Children) > 0 {
			localizeWorkflowExecutionSummaries(e.Children, loc)
		}
	}
}

// resolveQueuedPipelineDisplayName picks the best available name for queued pipeline rows.
func resolveQueuedPipelineDisplayName(app core.App, identifier string) string {
	fallback := strings.TrimSpace(identifier)
	if fallback == "" {
		fallback = "pipeline-run"
	}

	if app == nil || identifier == "" {
		return fallback
	}

	record, err := canonify.Resolve(app, identifier)
	if err != nil {
		return fallback
	}

	yaml := record.GetString("yaml")
	if yaml != "" {
		wfDef, err := pipelineinternal.ParseWorkflow(yaml)
		if err == nil {
			if name := strings.TrimSpace(wfDef.Name); name != "" {
				return name
			}
		}
	}

	if name := strings.TrimSpace(record.GetString("name")); name != "" {
		return name
	}

	return fallback
}

func baseKey(filename, marker string) (string, bool) {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	idx := strings.LastIndex(name, marker)
	if idx == -1 {
		return "", false
	}

	return name[:idx], true
}
