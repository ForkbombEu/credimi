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
	"sync"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/runners"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/encoding/protojson"
)

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

	batchSize := limit
	if statusFilter != "" && batchSize < statusFilteredPipelineResultBatchSize {
		batchSize = statusFilteredPipelineResultBatchSize
	}

	var allSummaries []*pipelineWorkflowSummary
	runnerCache := map[string]map[string]any{}
	runnerInfoByPipelineID := map[string]pipelineRunnerInfo{}

	temporalClient, err := temporalclient.GetTemporalClientWithNamespace(namespace)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"temporal",
			"unable to create temporal client",
			err.Error(),
		)
	}

	skipped := 0
	currentSkip := 0
	remainingLimit := limit

	// When no status filtering is requested, we can apply the offset directly at DB level.
	if statusFilter == "" {
		currentSkip = skip
		skipped = skip
	}

	for remainingLimit > 0 {
		resultsRecords, err := e.App.FindRecordsByFilter(
			"pipeline_results",
			"owner={:owner}",
			"-created",
			batchSize,
			currentSkip,
			dbx.Params{
				"owner": organizationId,
			},
		)

		if err != nil {
			return nil, apierror.New(
				http.StatusInternalServerError,
				"database",
				"failed to fetch pipeline results",
				err.Error(),
			)
		}

		if len(resultsRecords) == 0 {
			break
		}

		batchSummaries := fetchWorkflowBatch(
			e,
			pipelineMap,
			namespace,
			authRecord,
			temporalClient,
			resultsRecords,
			statusFilter,
			runnerCache,
			runnerInfoByPipelineID,
		)

		if skipped < skip {
			skipNow := skip - skipped
			if skipNow > len(batchSummaries) {
				skipNow = len(batchSummaries)
			}
			batchSummaries = batchSummaries[skipNow:]
			skipped += skipNow
		}

		take := remainingLimit
		if take > len(batchSummaries) {
			take = len(batchSummaries)
		}
		if take > 0 {
			allSummaries = append(allSummaries, batchSummaries[:take]...)
			remainingLimit -= take
		}

		currentSkip += batchSize
	}

	sort.Slice(allSummaries, func(i, j int) bool {
		return allSummaries[i].StartTime > allSummaries[j].StartTime
	})

	if allSummaries == nil {
		allSummaries = []*pipelineWorkflowSummary{}
	}

	return allSummaries, nil
}

func fetchWorkflowBatch(
	e *core.RequestEvent,
	pipelineMap map[string]*core.Record,
	namespace string,
	authRecord *core.Record,
	temporalClient client.Client,
	resultsRecords []*core.Record,
	statusFilter string,
	runnerCache map[string]map[string]any,
	runnerInfoByPipelineID map[string]pipelineRunnerInfo,
) []*pipelineWorkflowSummary {

	var batchSummaries []*pipelineWorkflowSummary

	type workflowInfo struct {
		resultRecord *core.Record
		workflowID   string
		runID        string
		pipelineID   string
	}

	workflowsToFetch := make([]workflowInfo, 0, len(resultsRecords))
	for _, resultRecord := range resultsRecords {
		workflowsToFetch = append(workflowsToFetch, workflowInfo{
			resultRecord: resultRecord,
			workflowID:   resultRecord.GetString("workflow_id"),
			runID:        resultRecord.GetString("run_id"),
			pipelineID:   resultRecord.GetString("pipeline"),
		})
	}

	parentRefs := make([]workflowExecutionRef, 0, len(workflowsToFetch))
	for _, wf := range workflowsToFetch {
		parentRefs = append(parentRefs, workflowExecutionRef{
			WorkflowID: wf.workflowID,
			RunID:      wf.runID,
		})
	}

	childWorkflowsByParent, err := getChildWorkflowsByParents(
		context.Background(),
		temporalClient,
		namespace,
		parentRefs,
	)
	if err != nil {
		e.App.Logger().Warn(fmt.Sprintf("failed to batch list child workflows: %v", err))
	}

	type fetchResult struct {
		workflowID string
		execution  *WorkflowExecution
		children   []*WorkflowExecution
		err        error
	}

	results := make([]*fetchResult, len(workflowsToFetch))
	sem := make(chan struct{}, pipelineResultsDescribeConcurrency)

	var wg sync.WaitGroup
	for i, wf := range workflowsToFetch {
		wg.Add(1)
		go func(idx int, wf workflowInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			execInfo, apiErr := describeWorkflowExecution(temporalClient, wf.workflowID, wf.runID)
			if apiErr != nil {
				results[idx] = &fetchResult{
					workflowID: wf.workflowID,
					err:        fmt.Errorf("%v", apiErr),
				}
				return
			}

			results[idx] = &fetchResult{
				workflowID: wf.workflowID,
				execution:  execInfo,
				children: childWorkflowsByParent[workflowExecutionRef{
					WorkflowID: wf.workflowID,
					RunID:      wf.runID,
				}],
				err: nil,
			}
		}(i, wf)
	}

	wg.Wait()

	for i, res := range results {
		if res == nil || res.err != nil {
			continue
		}

		pipelineRecord, ok := pipelineMap[workflowsToFetch[i].pipelineID]
		if !ok {
			continue
		}

		runnerInfo, ok := runnerInfoByPipelineID[workflowsToFetch[i].pipelineID]
		if !ok {
			runnerInfo, _ = runners.ParsePipelineRunnerInfo(pipelineRecord.GetString("yaml"))
			runnerInfoByPipelineID[workflowsToFetch[i].pipelineID] = runnerInfo
		}

		hierarchy := buildPipelineExecutionHierarchyFromResult(
			e.App,
			workflowsToFetch[i].resultRecord,
			res.execution,
			res.children,
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
			if statusFilter == "" || strings.EqualFold(summary.Status, statusFilter) {
				batchSummaries = append(batchSummaries, summary)
			}
		}
	}

	return batchSummaries
}

func parsePaginationParams(e *core.RequestEvent, defaultLimit, defaultOffset int) (limit, offset int) {
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

const (
	childWorkflowParentQueryChunkSize           = 25
	childWorkflowParentQueryPageSize      int32 = 1000
	statusFilteredPipelineResultBatchSize       = 50
	pipelineResultsDescribeConcurrency          = 12
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
