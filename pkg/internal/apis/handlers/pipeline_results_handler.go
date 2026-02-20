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
	"strconv"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/runners"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	workflowengine "github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/encoding/protojson"
)

// pipelineResultsTemporalClient resolves Temporal clients for pipeline results handlers.
var pipelineResultsTemporalClient = temporalclient.GetTemporalClientWithNamespace

// pipelineResultsListQueuedRuns allows tests to stub queued run aggregation.
var pipelineResultsListQueuedRuns = listQueuedPipelineRuns

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

		// Parse pagination parameters
		// Frontend sends offset as a 0-based page number, convert to skip count
		limit, pageNum := parsePaginationParams(e, 20, 0)
		skip := pageNum * limit

		status := e.Request.URL.Query().Get("status")
		statusLower := strings.ToLower(status)

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
			-1,
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

		// Handle non-queued status filter
		if status != "" && statusLower != statusStringQueued {
			summaries, apiErr := fetchCompletedWorkflowsWithPagination(
				e,
				pipelineMap,
				namespace,
				authRecord,
				organization.Id,
				status,
				limit,
				skip,
			)
			if apiErr != nil {
				return apiErr.JSON(e)
			}
			return e.JSON(http.StatusOK, summaries)
		}

		// Get queued runs only when needed (status=queued or no status filter).
		queuedRuns, err := pipelineResultsListQueuedRuns(e.Request.Context(), namespace)
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

		queuedCount := len(allQueuedForAllPipelines)

		// Handle queued-only case
		if statusLower == statusStringQueued {
			if len(allQueuedForAllPipelines) == 0 {
				return e.JSON(http.StatusOK, []*pipelineWorkflowSummary{})
			}

			startIdx := skip
			if startIdx >= len(allQueuedForAllPipelines) {
				return e.JSON(http.StatusOK, []*pipelineWorkflowSummary{})
			}

			endIdx := startIdx + limit
			if endIdx > len(allQueuedForAllPipelines) {
				endIdx = len(allQueuedForAllPipelines)
			}

			queuedToShow := allQueuedForAllPipelines[startIdx:endIdx]
			queuedSummaries := buildQueuedPipelineSummaries(
				e.App,
				queuedToShow,
				authRecord.GetString("Timezone"),
				map[string]map[string]any{},
			)
			return e.JSON(http.StatusOK, queuedSummaries)
		}

		// No status filter - include both queued and completed
		var finalSummaries []*pipelineWorkflowSummary

		if skip < queuedCount {
			queuedStartIdx := skip
			queuedEndIdx := queuedStartIdx + limit
			if queuedEndIdx > queuedCount {
				queuedEndIdx = queuedCount
			}

			queuedToShow := allQueuedForAllPipelines[queuedStartIdx:queuedEndIdx]
			queuedSummaries := buildQueuedPipelineSummaries(
				e.App,
				queuedToShow,
				authRecord.GetString("Timezone"),
				map[string]map[string]any{},
			)
			finalSummaries = append(finalSummaries, queuedSummaries...)

			if queuedEndIdx >= queuedCount && len(finalSummaries) < limit {
				remainingLimit := limit - len(finalSummaries)
				completedSkip := 0
				completedSummaries, apiErr := fetchCompletedWorkflowsWithPagination(
					e,
					pipelineMap,
					namespace,
					authRecord,
					organization.Id,
					"",
					remainingLimit,
					completedSkip,
				)
				if apiErr != nil {
					return apiErr.JSON(e)
				}
				finalSummaries = append(finalSummaries, completedSummaries...)
			}
		} else {
			completedSkip := skip - queuedCount

			completedSummaries, apiErr := fetchCompletedWorkflowsWithPagination(
				e,
				pipelineMap,
				namespace,
				authRecord,
				organization.Id,
				"",
				limit,
				completedSkip,
			)
			if apiErr != nil {
				return apiErr.JSON(e)
			}
			finalSummaries = append(finalSummaries, completedSummaries...)
		}

		if finalSummaries == nil {
			finalSummaries = []*pipelineWorkflowSummary{}
		}

		return e.JSON(http.StatusOK, finalSummaries)
	}
}

func fetchCompletedWorkflowsWithPagination(
	e *core.RequestEvent,
	pipelineMap map[string]*core.Record,
	namespace string,
	authRecord *core.Record,
	organizationId string,
	statusFilter string,
	limit int,
	skip int,
) ([]*pipelineWorkflowSummary, *apierror.APIError) {
	if limit <= 0 {
		return []*pipelineWorkflowSummary{}, nil
	}

	temporalClient, err := pipelineResultsTemporalClient(namespace)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"temporal",
			"unable to create temporal client",
			err.Error(),
		)
	}

	statusFilters, statusOk := parseWorkflowStatusFilters(statusFilter)
	if statusFilter != "" && !statusOk {
		return []*pipelineWorkflowSummary{}, nil
	}

	executions, err := listPipelineWorkflowExecutions(
		context.Background(),
		temporalClient,
		namespace,
		statusFilters,
		"",
		limit,
		skip,
	)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to list workflows",
			err.Error(),
		)
	}
	if len(executions) == 0 {
		return []*pipelineWorkflowSummary{}, nil
	}

	pipelineIdentifiers, err := resolvePipelineIdentifiersForExecutions(
		e.App,
		executions,
		organizationId,
	)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"pipeline",
			"failed to resolve pipeline identifiers",
			err.Error(),
		)
	}

	pipelineIdentifierIndex := buildPipelineIdentifierIndex(e.App, pipelineMap)
	filteredExecutions := make([]*WorkflowExecution, 0, len(executions))
	executionRefs := make([]workflowExecutionRef, 0, len(executions))
	pipelineByExecution := make(map[workflowExecutionRef]*core.Record, len(executions))
	pipelineIdentifierByExecution := make(map[workflowExecutionRef]string, len(executions))

	for _, exec := range executions {
		if exec == nil || exec.Execution == nil {
			continue
		}
		ref := workflowExecutionRef{
			WorkflowID: exec.Execution.WorkflowID,
			RunID:      exec.Execution.RunID,
		}
		if ref.WorkflowID == "" || ref.RunID == "" {
			continue
		}
		pipelineIdentifier := pipelineIdentifiers[ref]
		if pipelineIdentifier == "" {
			continue
		}
		pipelineRecord := pipelineIdentifierIndex[pipelineIdentifier]
		if pipelineRecord == nil {
			continue
		}

		filteredExecutions = append(filteredExecutions, exec)
		executionRefs = append(executionRefs, ref)
		pipelineByExecution[ref] = pipelineRecord
		pipelineIdentifierByExecution[ref] = pipelineIdentifier
	}

	if len(filteredExecutions) == 0 {
		return []*pipelineWorkflowSummary{}, nil
	}

	childWorkflowsByParent, err := getChildWorkflowsByParents(
		context.Background(),
		temporalClient,
		namespace,
		executionRefs,
	)
	if err != nil {
		e.App.Logger().Warn(fmt.Sprintf("failed to batch list child workflows: %v", err))
	}

	resultRecordsByExecution, err := fetchPipelineResultRecords(
		e.App,
		organizationId,
		executionRefs,
	)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"database",
			"failed to fetch pipeline results",
			err.Error(),
		)
	}

	var allSummaries []*pipelineWorkflowSummary
	runnerCache := map[string]map[string]any{}
	runnerInfoByPipelineID := map[string]pipelineRunnerInfo{}

	for _, exec := range filteredExecutions {
		ref := workflowExecutionRef{
			WorkflowID: exec.Execution.WorkflowID,
			RunID:      exec.Execution.RunID,
		}
		pipelineRecord := pipelineByExecution[ref]
		if pipelineRecord == nil {
			continue
		}

		runnerInfo, ok := runnerInfoByPipelineID[pipelineRecord.Id]
		if !ok {
			runnerInfo, _ = runners.ParsePipelineRunnerInfo(pipelineRecord.GetString("yaml"))
			runnerInfoByPipelineID[pipelineRecord.Id] = runnerInfo
		}

		hierarchy := buildPipelineExecutionHierarchyFromResult(
			e.App,
			resultRecordsByExecution[ref],
			exec,
			childWorkflowsByParent[ref],
			namespace,
			authRecord.GetString("Timezone"),
			temporalClient,
		)

		if len(hierarchy) == 0 {
			continue
		}

		annotated, _ := attachRunnerInfoFromTemporalStartInput(
			attachRunnerInfoFromTemporalInputArgs{
				App:         e.App,
				Ctx:         context.Background(),
				Client:      temporalClient,
				Executions:  hierarchy,
				Info:        runnerInfo,
				RunnerCache: runnerCache,
			},
		)

		for _, summary := range annotated {
			if summary == nil || summary.Execution == nil {
				continue
			}
			summaryRef := workflowExecutionRef{
				WorkflowID: summary.Execution.WorkflowID,
				RunID:      summary.Execution.RunID,
			}
			summary.PipelineIdentifier = pipelineIdentifierByExecution[summaryRef]
		}

		allSummaries = append(allSummaries, annotated...)
	}

	sort.Slice(allSummaries, func(i, j int) bool {
		return allSummaries[i].StartTime > allSummaries[j].StartTime
	})

	if allSummaries == nil {
		allSummaries = []*pipelineWorkflowSummary{}
	}

	return allSummaries, nil
}

func parsePaginationParams(
	e *core.RequestEvent,
	defaultLimit, defaultOffset int,
) (limit, offset int) {
	query := e.Request.URL.Query()

	limitStr := query.Get("limit")
	if limitStr == "" {
		limit = defaultLimit
	} else {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		} else {
			limit = defaultLimit
		}
	}

	offsetStr := query.Get("offset")
	if offsetStr == "" {
		offset = defaultOffset
	} else {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		} else {
			offset = defaultOffset
		}
	}

	return limit, offset
}

// parseWorkflowStatusFilters converts status filters into Temporal enum filters.
func parseWorkflowStatusFilters(
	statusFilter string,
) ([]enums.WorkflowExecutionStatus, bool) {
	if strings.TrimSpace(statusFilter) == "" {
		return nil, true
	}

	statuses := []enums.WorkflowExecutionStatus{}
	for _, raw := range strings.Split(statusFilter, ",") {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case statusStringRunning:
			statuses = append(statuses, enums.WORKFLOW_EXECUTION_STATUS_RUNNING)
		case statusStringCompleted:
			statuses = append(statuses, enums.WORKFLOW_EXECUTION_STATUS_COMPLETED)
		case statusStringFailed:
			statuses = append(statuses, enums.WORKFLOW_EXECUTION_STATUS_FAILED)
		case statusStringTerminated:
			statuses = append(statuses, enums.WORKFLOW_EXECUTION_STATUS_TERMINATED)
		case statusStringCanceled:
			statuses = append(statuses, enums.WORKFLOW_EXECUTION_STATUS_CANCELED)
		case statusStringTimedOut:
			statuses = append(statuses, enums.WORKFLOW_EXECUTION_STATUS_TIMED_OUT)
		case statusStringContinuedAsNew:
			statuses = append(statuses, enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW)
		case statusStringUnspecified:
			statuses = append(statuses, enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED)
		}
	}

	if len(statuses) == 0 {
		return nil, false
	}
	return statuses, true
}

// buildPipelineWorkflowsQuery builds a Temporal visibility query for pipeline workflows.
func buildPipelineWorkflowsQuery(
	statusFilters []enums.WorkflowExecutionStatus,
	pipelineIdentifier string,
) string {
	workflowName := pipeline.NewPipelineWorkflow().Name()
	parts := []string{
		fmt.Sprintf(`WorkflowType="%s"`, escapeTemporalQueryValue(workflowName)),
	}

	if len(statusFilters) > 0 {
		statusQueries := make([]string, 0, len(statusFilters))
		for _, status := range statusFilters {
			statusQueries = append(statusQueries, fmt.Sprintf("ExecutionStatus=%d", status))
		}
		parts = append(parts, "("+strings.Join(statusQueries, " or ")+")")
	}

	normalizedPipelineID := workflowengine.NormalizePipelineIdentifier(pipelineIdentifier)
	if normalizedPipelineID != "" {
		parts = append(
			parts,
			fmt.Sprintf(
				`%s="%s"`,
				workflowengine.PipelineIdentifierSearchAttribute,
				escapeTemporalQueryValue(normalizedPipelineID),
			),
		)
	}

	return strings.Join(parts, " and ")
}

// listPipelineWorkflowExecutions lists pipeline workflows using Temporal visibility with pagination.
func listPipelineWorkflowExecutions(
	ctx context.Context,
	temporalClient client.Client,
	namespace string,
	statusFilters []enums.WorkflowExecutionStatus,
	pipelineIdentifier string,
	limit int,
	offset int,
) ([]*WorkflowExecution, error) {
	if limit <= 0 {
		return []*WorkflowExecution{}, nil
	}
	if offset < 0 {
		offset = 0
	}

	query := buildPipelineWorkflowsQuery(statusFilters, pipelineIdentifier)
	pageSize := int32(limit)
	if pageSize <= 0 {
		pageSize = 1
	}

	results := make([]*WorkflowExecution, 0, limit)
	skipped := 0
	var pageToken []byte

	for len(results) < limit {
		resp, err := temporalClient.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Namespace:     namespace,
			PageSize:      pageSize,
			NextPageToken: pageToken,
			Query:         query,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			break
		}

		for _, execInfo := range resp.GetExecutions() {
			if execInfo == nil {
				continue
			}
			if skipped < offset {
				skipped++
				continue
			}

			execJSON, err := protojson.Marshal(execInfo)
			if err != nil {
				return nil, fmt.Errorf("marshal workflow execution: %w", err)
			}
			var exec WorkflowExecution
			if err := json.Unmarshal(execJSON, &exec); err != nil {
				return nil, fmt.Errorf("unmarshal workflow execution: %w", err)
			}
			if decodedSearchAttributes, err := decodeWorkflowSearchAttributes(
				execInfo.GetSearchAttributes(),
			); err != nil {
				return nil, err
			} else if len(decodedSearchAttributes) > 0 {
				exec.SearchAttributes = &decodedSearchAttributes
			}
			results = append(results, &exec)
			if len(results) >= limit {
				break
			}
		}

		if len(resp.GetNextPageToken()) == 0 {
			break
		}
		pageToken = resp.GetNextPageToken()
	}

	return results, nil
}

func buildPipelineIdentifierIndex(
	app core.App,
	pipelineMap map[string]*core.Record,
) map[string]*core.Record {
	index := make(map[string]*core.Record, len(pipelineMap))
	if app == nil {
		return index
	}

	for _, record := range pipelineMap {
		if record == nil {
			continue
		}
		path, err := canonify.BuildPath(app, record, canonify.CanonifyPaths["pipelines"], "")
		if err != nil {
			continue
		}
		index[strings.Trim(path, "/")] = record
	}

	return index
}

func buildPipelineIdentifierByID(
	app core.App,
	pipelineMap map[string]*core.Record,
) map[string]string {
	identifiers := make(map[string]string, len(pipelineMap))
	if app == nil {
		return identifiers
	}

	for pipelineID, record := range pipelineMap {
		if record == nil {
			continue
		}
		path, err := canonify.BuildPath(app, record, canonify.CanonifyPaths["pipelines"], "")
		if err != nil {
			continue
		}
		identifiers[pipelineID] = strings.Trim(path, "/")
	}

	return identifiers
}

func fetchPipelineResultRecords(
	app core.App,
	ownerID string,
	executions []workflowExecutionRef,
) (map[workflowExecutionRef]*core.Record, error) {
	resultRecords := map[workflowExecutionRef]*core.Record{}
	if app == nil || len(executions) == 0 {
		return resultRecords, nil
	}

	filter, params := buildWorkflowExecutionFilter(executions)
	if filter == "" {
		return resultRecords, nil
	}
	if ownerID != "" {
		filter = fmt.Sprintf("owner={:owner} && (%s)", filter)
		params["owner"] = ownerID
	}

	records, err := app.FindRecordsByFilter(
		"pipeline_results",
		filter,
		"",
		-1,
		0,
		params,
	)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		ref := workflowExecutionRef{
			WorkflowID: record.GetString("workflow_id"),
			RunID:      record.GetString("run_id"),
		}
		if ref.WorkflowID == "" || ref.RunID == "" {
			continue
		}
		resultRecords[ref] = record
	}

	return resultRecords, nil
}

// decodeWorkflowSearchAttributes converts Temporal payloads into native search attribute values.
func decodeWorkflowSearchAttributes(
	searchAttributes *commonpb.SearchAttributes,
) (DecodedWorkflowSearchAttributes, error) {
	if searchAttributes == nil {
		return nil, nil
	}
	fields := searchAttributes.GetIndexedFields()
	if len(fields) == 0 {
		return nil, nil
	}

	decoded := DecodedWorkflowSearchAttributes{}
	dataConverter := converter.GetDefaultDataConverter()
	for key, payload := range fields {
		if payload == nil {
			continue
		}
		var value any
		if err := dataConverter.FromPayload(payload, &value); err != nil {
			return nil, fmt.Errorf("decode search attribute %s: %w", key, err)
		}
		decoded[key] = value
	}
	return decoded, nil
}

// resolvePipelineIdentifiersForExecutions maps workflow execution refs to pipeline identifiers.
func resolvePipelineIdentifiersForExecutions(
	app core.App,
	executions []*WorkflowExecution,
	ownerID string,
) (map[workflowExecutionRef]string, error) {
	identifiers := make(map[workflowExecutionRef]string, len(executions))
	missing := make([]workflowExecutionRef, 0, len(executions))

	for _, exec := range executions {
		if exec == nil || exec.Execution == nil {
			continue
		}
		ref := workflowExecutionRef{
			WorkflowID: exec.Execution.WorkflowID,
			RunID:      exec.Execution.RunID,
		}
		if ref.WorkflowID == "" || ref.RunID == "" {
			continue
		}

		if identifier := pipelineIdentifierFromSearchAttributes(exec.SearchAttributes); identifier != "" {
			identifiers[ref] = identifier
			continue
		}

		missing = append(missing, ref)
	}

	if len(missing) == 0 || app == nil {
		return identifiers, nil
	}

	filter, params := buildWorkflowExecutionFilter(missing)
	if filter == "" {
		return identifiers, nil
	}
	if ownerID != "" {
		filter = fmt.Sprintf("owner={:owner} && (%s)", filter)
		params["owner"] = ownerID
	}

	records, err := app.FindRecordsByFilter(
		"pipeline_results",
		filter,
		"",
		-1,
		0,
		params,
	)
	if err != nil {
		return identifiers, err
	}
	if len(records) == 0 {
		return identifiers, nil
	}

	pipelineIDs := map[string]struct{}{}
	for _, record := range records {
		if pipelineID := record.GetString("pipeline"); pipelineID != "" {
			pipelineIDs[pipelineID] = struct{}{}
		}
	}

	pipelineIDList := make([]string, 0, len(pipelineIDs))
	for pipelineID := range pipelineIDs {
		pipelineIDList = append(pipelineIDList, pipelineID)
	}

	pipelineRecords, err := app.FindRecordsByIds("pipelines", pipelineIDList)
	if err != nil {
		return identifiers, err
	}

	pipelineByID := map[string]*core.Record{}
	for _, record := range pipelineRecords {
		pipelineByID[record.Id] = record
	}

	for _, record := range records {
		ref := workflowExecutionRef{
			WorkflowID: record.GetString("workflow_id"),
			RunID:      record.GetString("run_id"),
		}
		if _, exists := identifiers[ref]; exists {
			continue
		}

		pipelineRecord := pipelineByID[record.GetString("pipeline")]
		if pipelineRecord == nil {
			continue
		}
		path, err := canonify.BuildPath(
			app,
			pipelineRecord,
			canonify.CanonifyPaths["pipelines"],
			"",
		)
		if err != nil {
			continue
		}
		identifiers[ref] = strings.Trim(path, "/")
	}

	return identifiers, nil
}

func pipelineIdentifierFromSearchAttributes(
	attributes *DecodedWorkflowSearchAttributes,
) string {
	if attributes == nil {
		return ""
	}
	value, ok := (*attributes)[workflowengine.PipelineIdentifierSearchAttribute]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return workflowengine.NormalizePipelineIdentifier(typed)
	case []string:
		if len(typed) > 0 {
			return workflowengine.NormalizePipelineIdentifier(typed[0])
		}
	case fmt.Stringer:
		return workflowengine.NormalizePipelineIdentifier(typed.String())
	}
	return ""
}

func buildWorkflowExecutionFilter(
	executions []workflowExecutionRef,
) (string, dbx.Params) {
	clauses := make([]string, 0, len(executions))
	params := dbx.Params{}

	for idx, exec := range executions {
		if exec.WorkflowID == "" || exec.RunID == "" {
			continue
		}
		workflowKey := fmt.Sprintf("workflow_id_%d", idx)
		runKey := fmt.Sprintf("run_id_%d", idx)
		params[workflowKey] = exec.WorkflowID
		params[runKey] = exec.RunID
		clauses = append(
			clauses,
			fmt.Sprintf(
				"(workflow_id = {:%s} && run_id = {:%s})",
				workflowKey,
				runKey,
			),
		)
	}

	return strings.Join(clauses, " || "), params
}

const (
	childWorkflowParentQueryChunkSize       = 25
	childWorkflowParentQueryPageSize  int32 = 1000
)

type workflowExecutionRef struct {
	WorkflowID string
	RunID      string
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
		Duration:  calculateDuration(exec.StartTime, exec.CloseTime),
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

func getChildWorkflowsByParents(
	ctx context.Context,
	temporalClient client.Client,
	namespace string,
	parents []workflowExecutionRef,
) (map[workflowExecutionRef][]*WorkflowExecution, error) {
	childrenByParent := make(map[workflowExecutionRef][]*WorkflowExecution, len(parents))
	if len(parents) == 0 || temporalClient == nil {
		return childrenByParent, nil
	}

	uniqueParents := make([]workflowExecutionRef, 0, len(parents))
	parentSet := make(map[workflowExecutionRef]struct{}, len(parents))
	for _, parent := range parents {
		if parent.WorkflowID == "" || parent.RunID == "" {
			continue
		}
		if _, exists := parentSet[parent]; exists {
			continue
		}
		parentSet[parent] = struct{}{}
		uniqueParents = append(uniqueParents, parent)
		childrenByParent[parent] = nil
	}
	if len(uniqueParents) == 0 {
		return childrenByParent, nil
	}

	var firstErr error
	for i := 0; i < len(uniqueParents); i += childWorkflowParentQueryChunkSize {
		chunkEnd := i + childWorkflowParentQueryChunkSize
		if chunkEnd > len(uniqueParents) {
			chunkEnd = len(uniqueParents)
		}

		query := buildChildWorkflowParentQuery(uniqueParents[i:chunkEnd])
		if query == "" {
			continue
		}

		pageToken := []byte(nil)
		for {
			resp, err := temporalClient.ListWorkflow(
				ctx,
				&workflowservice.ListWorkflowExecutionsRequest{
					Namespace:     namespace,
					Query:         query,
					PageSize:      childWorkflowParentQueryPageSize,
					NextPageToken: pageToken,
				},
			)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				break
			}

			for _, childExec := range resp.GetExecutions() {
				if childExec == nil || childExec.GetExecution() == nil {
					continue
				}
				parentExec := childExec.GetParentExecution()
				if parentExec == nil {
					continue
				}

				parentRef := workflowExecutionRef{
					WorkflowID: parentExec.GetWorkflowId(),
					RunID:      parentExec.GetRunId(),
				}
				if _, ok := parentSet[parentRef]; !ok {
					continue
				}

				childJSON, err := protojson.Marshal(childExec)
				if err != nil {
					continue
				}

				var childWorkflow WorkflowExecution
				if err := json.Unmarshal(childJSON, &childWorkflow); err != nil {
					continue
				}

				childWorkflow.ParentExecution = &WorkflowIdentifier{
					WorkflowID: parentRef.WorkflowID,
					RunID:      parentRef.RunID,
				}
				childrenByParent[parentRef] = append(childrenByParent[parentRef], &childWorkflow)
			}

			if len(resp.GetNextPageToken()) == 0 {
				break
			}
			pageToken = resp.GetNextPageToken()
		}
	}

	return childrenByParent, firstErr
}

func buildChildWorkflowParentQuery(parents []workflowExecutionRef) string {
	if len(parents) == 0 {
		return ""
	}

	clauses := make([]string, 0, len(parents))
	for _, parent := range parents {
		if parent.WorkflowID == "" || parent.RunID == "" {
			continue
		}
		clauses = append(clauses, fmt.Sprintf(
			`(ParentWorkflowId="%s" AND ParentRunId="%s")`,
			escapeTemporalQueryValue(parent.WorkflowID),
			escapeTemporalQueryValue(parent.RunID),
		))
	}
	return strings.Join(clauses, " OR ")
}

func escapeTemporalQueryValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
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
	PipelineIdentifier string           `json:"pipeline_identifier,omitempty"`
	GlobalRunnerID     string           `json:"global_runner_id,omitempty"`
	RunnerIDs          []string         `json:"runner_ids,omitempty"`
	RunnerRecords      []map[string]any `json:"runner_records,omitempty"`
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
		PipelineIdentifier:       workflowengine.NormalizePipelineIdentifier(queued.PipelineIdentifier),
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
	orgNameByOwnerID := make(map[string]string)
	for _, record := range pipelineRecords {
		pipelineID := record.Id
		pipelineIdentifiers[pipelineID] = pipelineID

		canonifiedName := record.GetString("canonified_name")

		ownerID := record.GetString("owner")
		orgName, ok := orgNameByOwnerID[ownerID]
		if !ok {
			orgName = ""
			if ownerID != "" {
				org, err := app.FindRecordById("organizations", ownerID)
				if err == nil {
					orgName = org.GetString("canonified_name")
				}
			}
			orgNameByOwnerID[ownerID] = orgName
		}

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
