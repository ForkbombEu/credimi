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
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
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
			Path:           "/{workflowId}/runs/{runId}",
			OperationID:    "workflowRun.get",
			Handler:        HandleGetMyWorkflowRun,
			ResponseSchema: GetMyWorkflowRunResponse{},
			Description:    "Get details of a specific run for a workflow",
			Summary:        "Get details of a specific run for a workflow",
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

var listWorkflowsTemporalClient = temporalclient.GetTemporalClientWithNamespace
var listWorkflows = listWorkflowsTemporal
var workflowTemporalClient = temporalclient.GetTemporalClientWithNamespace

func HandleListMyWorkflows() func(*core.RequestEvent) error {
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

		namespace, err := pbutils.GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organizations",
				"unable to get user organization canonified name",
				err.Error(),
			)
		}

		limit, page := parsePageParams(e, 20, 0)
		itemOffset := page * limit

		c, err := listWorkflowsTemporalClient(namespace)
		if err != nil {
			return temporalUnavailableError()
		}
		statusParam := e.Request.URL.Query().Get("status")
		statusFilters, statusOK := parseWorkflowStatusFilters(statusParam)
		if statusParam != "" && !statusOK {
			return e.JSON(http.StatusOK, ListMyWorkflowsResponse{
				Executions: []*WorkflowExecutionSummary{},
			})
		}

		query := buildWorkflowStatusQuery(statusFilters)

		list, err := listWorkflows(e.Request.Context(), c, namespace, query)
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
			)
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
			)
		}

		hierarchy := buildWorkflowExecutionHierarchy(
			e.Request.Context(),
			e.App,
			filteredExecutions,
			namespace,
			c,
		)
		hierarchy = paginateWorkflowExecutionSummaries(hierarchy, limit, itemOffset)

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
		// Action author sessions and their chats live on their own pages only.
		if isActionAuthorWorkflowID(exec.Execution.WorkflowID) {
			continue
		}
		if exec.Type.Name == pipelineWorkflowName &&
			!isPipelineExecuteWorkflowID(exec.Execution.WorkflowID) {
			continue
		}
		// Linked steps blocks and the steps they run belong to their pipeline run.
		if exec.Type.Name == pipelineinternal.LinkedStepsWorkflowName {
			continue
		}
		if exec.ParentExecution == nil {
			filtered = append(filtered, exec)
			continue
		}

		parent, ok := executionByRunID[exec.ParentExecution.RunID]
		if ok {
			if parent != nil &&
				parent.Type.Name == pipelineWorkflowName &&
				!isPipelineExecuteWorkflowID(parent.Execution.WorkflowID) {
				continue
			}
			if parent != nil && parent.Type.Name == pipelineinternal.LinkedStepsWorkflowName {
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
		if parentType == pipelineWorkflowName &&
			!isPipelineExecuteWorkflowID(exec.ParentExecution.WorkflowID) {
			continue
		}
		if parentType == pipelineinternal.LinkedStepsWorkflowName {
			continue
		}

		filtered = append(filtered, exec)
	}

	return filtered, nil
}

// isActionAuthorWorkflowID reports an action author session or its chat
// (<session>-author), which only the session's starter may see.
func isActionAuthorWorkflowID(workflowID string) bool {
	return strings.HasPrefix(workflowID, activities.ActionAuthorSessionWorkflowIDPrefix)
}

func isPipelineExecuteWorkflowID(workflowID string) bool {
	return strings.HasPrefix(workflowID, pipelineExecuteWorkflowIDPrefix)
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
	out := &workflowservice.ListWorkflowExecutionsResponse{}
	var pageToken []byte
	for {
		resp, err := c.ListWorkflow(
			ctx,
			&workflowservice.ListWorkflowExecutionsRequest{
				Namespace:     namespace,
				PageSize:      1000,
				NextPageToken: pageToken,
				Query:         query,
			},
		)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			break
		}
		out.Executions = append(out.Executions, resp.GetExecutions()...)
		if len(resp.GetNextPageToken()) == 0 {
			break
		}
		pageToken = resp.GetNextPageToken()
	}
	return out, nil
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
			)
		}

		workflowID := e.Request.PathValue("workflowId")
		runID := e.Request.PathValue("runId")
		if isActionAuthorWorkflowID(workflowID) {
			return apierror.New(
				http.StatusNotFound,
				"workflow",
				"workflow not found",
				"workflow not found",
			)
		}
		if runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"runId",
				"runId is required",
				"missing runId",
			)
		}
		namespace, err := pbutils.GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			)
		}
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization is empty",
				"missing organization",
			)
		}

		c, err := workflowTemporalClient(namespace)
		if err != nil {
			return temporalUnavailableError()
		}
		workflowExecution, err := c.DescribeWorkflowExecution(
			e.Request.Context(),
			workflowID,
			runID,
		)
		if err != nil {
			notFound := &serviceerror.NotFound{}
			if errors.As(err, &notFound) {
				if pipelineRunOwnedByAnotherOrganization(e.App, authRecord.Id, workflowID, runID) {
					return apierror.New(
						http.StatusForbidden,
						"workflow",
						"workflow belongs to another organization",
						"the run belongs to another organization",
					)
				}
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
		if info := workflowExecution.GetWorkflowExecutionInfo(); info != nil &&
			info.GetStatus() == enums.WORKFLOW_EXECUTION_STATUS_FAILED {
			if failure := fetchWorkflowFailure(
				e.Request.Context(),
				c,
				workflowID,
				runID,
			); failure != nil {
				finalJSON["failure_reason"] = *failure
			}
		}
		if devices := pipelineExecutionDevicesFromDescribe(
			e.Request.Context(),
			e.App,
			namespace,
			workflowExecution,
		); len(devices) > 0 {
			finalJSON["devices"] = devices
		}
		return e.JSON(http.StatusOK, finalJSON)
	}
}

// pipelineRunOwnedByAnotherOrganization reports whether a run missing from the
// caller's namespace is a pipeline run recorded for another organization, so
// the run page can ask the caller to switch account (e.g. after opening another
// user's notification) instead of reporting a missing run. It discloses only
// that the given workflow and run IDs exist; lookup failures report false.
func pipelineRunOwnedByAnotherOrganization(
	app core.App,
	userID, workflowID, runID string,
) bool {
	record, err := pipelineresults.FindByWorkflowRun(app, workflowID, runID)
	if err != nil {
		return false
	}
	orgID, err := pbutils.GetUserOrganizationID(app, userID)
	if err != nil {
		return false
	}
	return record.GetString("owner") != orgID
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
			)
		}

		workflowID := e.Request.PathValue("workflowId")
		runID := e.Request.PathValue("runId")
		if isActionAuthorWorkflowID(workflowID) {
			return apierror.New(
				http.StatusNotFound,
				"workflow",
				"workflow execution not found",
				"workflow not found",
			)
		}
		if workflowID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"workflowId and runId are required",
				"missing required parameters",
			)
		}

		namespace, err := pbutils.GetUserOrganizationCanonifiedName(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"unable to get user organization canonified name",
				err.Error(),
			)
		}

		c, err := workflowTemporalClient(namespace)
		if err != nil {
			return temporalUnavailableError()
		}

		err = c.CancelWorkflow(e.Request.Context(), workflowID, runID)
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
			"message":    "Workflow execution canceled successfully",
			"workflowId": workflowID,
			"runId":      runID,
			"status":     statusStringCanceled,
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
	case strings.HasPrefix(workflowID, "OpenID4VPWalletCheckWorkflow"):
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

func buildWorkflowExecutionHierarchy(
	ctx context.Context,
	app core.App,
	executions []*WorkflowExecution,
	namespace string,
	c client.Client,
) []*WorkflowExecutionSummary {
	summaryMap := make(map[string]*WorkflowExecutionSummary)
	for _, exec := range executions {
		summary := buildWorkflowExecutionSummary(ctx, exec, c)
		if summary == nil {
			continue
		}
		summaryMap[exec.Execution.RunID] = summary
	}

	roots := make([]*WorkflowExecutionSummary, 0, len(executions))

	for _, exec := range executions {
		if exec == nil || exec.Execution == nil {
			continue
		}
		current := summaryMap[exec.Execution.RunID]
		if current == nil {
			continue
		}

		if exec.ParentExecution != nil {
			if parent, ok := summaryMap[exec.ParentExecution.RunID]; ok {
				current.DisplayName = computeChildDisplayName(exec.Execution.WorkflowID)
				parent.Children = append(parent.Children, current)
				continue
			}
		}

		var parentDisplay string
		if exec.Memo != nil {
			if field, ok := exec.Memo.Fields["test"]; ok && field != nil && field.Data != nil {
				parentDisplay = workflowengine.DecodeStringPayload(field.temporalPayload())
			}
		}

		w := pipeline.PipelineWorkflow{}
		if current.Type.Name == w.Name() {
			artifacts := pipelineresults.ResolvePipelineExecutionArtifacts(
				app,
				namespace,
				exec.Execution.WorkflowID,
				exec.Execution.RunID,
			)
			current.Results = artifacts.Results
			current.MaestroScreenshots = artifacts.MaestroScreenshots
			current.Report = artifacts.Report
			current.FCAFReport = artifacts.FCAFReport
			current.FCAFReportPDF = artifacts.FCAFReportPDF
		}
		current.DisplayName = parentDisplay
		roots = append(roots, current)
	}

	sortWorkflowExecutionSummaries(roots, false)

	return roots
}

func sortExecutionSummaries(list []*WorkflowExecutionSummary, ascending bool) {
	sortWorkflowExecutionSummaries(list, ascending)
}

// sortWorkflowExecutionSummaries sorts roots in the caller's order and every nested level
// chronologically.
func sortWorkflowExecutionSummaries(list []*WorkflowExecutionSummary, ascending bool) {
	slices.SortFunc(list, func(a, b *WorkflowExecutionSummary) int {
		comparison := utils.CompareTimeStrings(a.StartTime, b.StartTime)
		if ascending {
			return comparison
		}
		return -comparison
	})

	for _, e := range list {
		if len(e.Children) > 0 {
			sortWorkflowExecutionSummaries(e.Children, true)
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
