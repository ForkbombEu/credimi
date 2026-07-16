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
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
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
	"google.golang.org/protobuf/encoding/protojson"
)

// pipelineResultsTemporalClient resolves Temporal clients for pipeline results handlers.
var pipelineResultsTemporalClient = temporalclient.GetTemporalClientWithNamespace

type pipelineExecutionHistoryRequest struct {
	App                core.App
	TemporalClient     client.Client
	Namespace          string
	OwnerID            string
	UserTimezone       string
	PipelineRecord     *core.Record
	PipelineIdentifier string
	StatusFilter       string
	Limit              int
	Skip               int
}

func listPipelineExecutionHistory(
	ctx context.Context,
	request pipelineExecutionHistoryRequest,
) ([]*pipelineWorkflowSummary, error) {
	if request.Limit <= 0 {
		return []*pipelineWorkflowSummary{}, nil
	}

	statusFilters, statusOK := parseWorkflowStatusFilters(request.StatusFilter)
	if request.StatusFilter != "" && !statusOK {
		return []*pipelineWorkflowSummary{}, nil
	}

	executions, err := listPipelineWorkflowExecutions(
		ctx,
		request.TemporalClient,
		request.Namespace,
		statusFilters,
		request.PipelineIdentifier,
		request.Limit,
		request.Skip,
	)
	if err != nil {
		return nil, fmt.Errorf("list pipeline workflows: %w", err)
	}
	sort.Slice(executions, func(i, j int) bool {
		return utils.TimeStringAfter(executions[i].StartTime, executions[j].StartTime)
	})

	executionRefs := workflowExecutionRefs(executions)
	if len(executionRefs) == 0 {
		return []*pipelineWorkflowSummary{}, nil
	}

	childrenByParent, err := getChildWorkflowsByParents(
		ctx,
		request.TemporalClient,
		request.Namespace,
		executionRefs,
	)
	if err != nil {
		return nil, fmt.Errorf("list child workflows: %w", err)
	}

	resultRecords, err := fetchPipelineResultRecords(
		request.App,
		request.OwnerID,
		executionRefs,
	)
	if err != nil {
		return nil, fmt.Errorf("fetch pipeline results: %w", err)
	}

	builder := newPipelineExecutionSummaryBuilder(
		request.App,
		request.TemporalClient,
		request.UserTimezone,
	)
	summaries := make([]*pipelineWorkflowSummary, 0, len(executions))
	for _, execution := range executions {
		ref, ok := workflowExecutionReference(execution)
		if !ok {
			continue
		}

		summary, err := builder.Build(
			ctx,
			request.PipelineRecord,
			request.PipelineIdentifier,
			execution,
			childrenByParent[ref],
			resultRecords[ref],
		)
		if err != nil {
			return nil, err
		}
		if summary != nil {
			summaries = append(summaries, summary)
		}
	}

	return summaries, nil
}

func parsePageParams(
	e *core.RequestEvent,
	defaultLimit, defaultPage int,
) (limit, page int) {
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

	pageValue := query.Get("page")
	if pageValue == "" {
		pageValue = query.Get("offset") // Legacy alias.
	}
	if pageValue == "" {
		page = defaultPage
	} else {
		if parsed, err := strconv.Atoi(pageValue); err == nil && parsed >= 0 {
			page = parsed
		} else {
			page = defaultPage
		}
	}

	return limit, page
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

	if limit > pipelineListWorkflowsDefaultLimit {
		limit = pipelineListWorkflowsDefaultLimit
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
		resp, err := temporalClient.ListWorkflow(
			ctx,
			&workflowservice.ListWorkflowExecutionsRequest{
				Namespace:     namespace,
				PageSize:      pageSize,
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

	for i := 0; i < len(executions); i += workflowExecutionFilterChunkSize {
		chunkEnd := i + workflowExecutionFilterChunkSize
		if chunkEnd > len(executions) {
			chunkEnd = len(executions)
		}

		filter, params := buildWorkflowExecutionFilter(executions[i:chunkEnd])
		if filter == "" {
			continue
		}
		if ownerID != "" {
			filter = fmt.Sprintf("owner={:owner} && (%s)", filter)
			params["owner"] = ownerID
		}

		chunkRecords, err := app.FindRecordsByFilter(
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

		for _, record := range chunkRecords {
			ref := workflowExecutionRef{
				WorkflowID: record.GetString("workflow_id"),
				RunID:      record.GetString("run_id"),
			}
			if ref.WorkflowID == "" || ref.RunID == "" {
				continue
			}
			resultRecords[ref] = record
		}
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
	dataConverter := temporalcrypto.DataConverter()
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
// Executions missing the Temporal search attribute are ignored.
func resolvePipelineIdentifiersForExecutions(
	executions []*WorkflowExecution,
) map[workflowExecutionRef]string {
	identifiers := make(map[workflowExecutionRef]string, len(executions))

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

		if identifier := pipelineIdentifierFromSearchAttributes(
			exec.SearchAttributes,
		); identifier != "" {
			identifiers[ref] = identifier
		}
	}

	return identifiers
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
	// Keep PocketBase workflow-id filters below expression and length limits.
	workflowExecutionFilterChunkSize = 40
)

type workflowExecutionRef struct {
	WorkflowID string
	RunID      string
}

func workflowExecutionReference(execution *WorkflowExecution) (workflowExecutionRef, bool) {
	if execution == nil || execution.Execution == nil {
		return workflowExecutionRef{}, false
	}
	ref := workflowExecutionRef{
		WorkflowID: execution.Execution.WorkflowID,
		RunID:      execution.Execution.RunID,
	}
	return ref, ref.WorkflowID != "" && ref.RunID != ""
}

func workflowExecutionRefs(executions []*WorkflowExecution) []workflowExecutionRef {
	refs := make([]workflowExecutionRef, 0, len(executions))
	for _, execution := range executions {
		if ref, ok := workflowExecutionReference(execution); ok {
			refs = append(refs, ref)
		}
	}
	return refs
}

type pipelineExecutionSummaryBuilder struct {
	app         core.App
	client      client.Client
	location    *time.Location
	runnerCache map[string]map[string]any
	runnerInfo  map[string]pipeline.PipelineRunnerInfo
}

func newPipelineExecutionSummaryBuilder(
	app core.App,
	temporalClient client.Client,
	userTimezone string,
) *pipelineExecutionSummaryBuilder {
	location, err := time.LoadLocation(userTimezone)
	if err != nil {
		location = time.Local
	}
	return &pipelineExecutionSummaryBuilder{
		app:         app,
		client:      temporalClient,
		location:    location,
		runnerCache: map[string]map[string]any{},
		runnerInfo:  map[string]pipeline.PipelineRunnerInfo{},
	}
}

func (b *pipelineExecutionSummaryBuilder) Build(
	ctx context.Context,
	pipelineRecord *core.Record,
	pipelineIdentifier string,
	rootExecution *WorkflowExecution,
	childExecutions []*WorkflowExecution,
	resultRecord *core.Record,
) (*pipelineWorkflowSummary, error) {
	rootSummary := buildWorkflowExecutionSummary(ctx, rootExecution, b.client)
	if rootSummary == nil {
		return nil, nil
	}

	if rootExecution.Memo != nil {
		if field, ok := rootExecution.Memo.Fields["test"]; ok && field.Data != nil {
			rootSummary.DisplayName = DecodeFromTemporalPayload(*field.Data)
		}
	}

	if resultRecord != nil {
		attachPipelineArtifactsToSummary(rootSummary, b.app, resultRecord)
	}

	for _, childExecution := range childExecutions {
		childSummary := buildWorkflowExecutionSummary(ctx, childExecution, b.client)
		if childSummary == nil || childExecution.Execution == nil {
			continue
		}
		childSummary.DisplayName = computeChildDisplayName(childExecution.Execution.WorkflowID)
		childSummary.HasLogs = workflowExecutionHasLogs(childExecution)
		rootSummary.Children = append(rootSummary.Children, childSummary)
	}
	sortWorkflowExecutionSummaries(rootSummary.Children, true)

	runnerInfo := b.pipelineRunnerInfo(pipelineRecord)
	globalRunnerID := ""
	if b.client != nil && runnerInfo.NeedsGlobalRunner {
		var err error
		globalRunnerID, err = readGlobalRunnerIDFromTemporalHistory(
			ctx,
			b.client,
			rootSummary.Execution.WorkflowID,
			rootSummary.Execution.RunID,
		)
		if err != nil {
			return nil, fmt.Errorf("read pipeline workflow start input: %w", err)
		}
	}

	runnerIDs := pipeline.RunnerIDsWithGlobal(runnerInfo, globalRunnerID)
	summary := &pipelineWorkflowSummary{
		WorkflowExecutionSummary: *rootSummary,
		GlobalRunnerID:           globalRunnerID,
		RunnerIDs:                runnerIDs,
		RunnerRecords: pipeline.ResolveRunnerRecords(
			b.app,
			runnerIDs,
			b.runnerCache,
		),
	}
	summary.PipelineIdentifier = pipelineIdentifier
	summary.PipelineName = resolvePipelineNameFromRecord(pipelineRecord, pipelineIdentifier)
	localizePipelineWorkflowSummaries([]*pipelineWorkflowSummary{summary}, b.location)
	return summary, nil
}

func workflowExecutionHasLogs(exec *WorkflowExecution) bool {
	if exec == nil || exec.Memo == nil {
		return false
	}
	payload := exec.Memo.Fields[workflowengine.CredimiCapabilitiesMemoKey]
	if payload == nil || payload.Data == nil {
		return false
	}
	data, err := base64.StdEncoding.DecodeString(*payload.Data)
	if err != nil {
		return false
	}
	var capabilities workflowengine.CredimiCapabilities
	if err := json.Unmarshal(data, &capabilities); err != nil {
		return false
	}
	return capabilities.Logs
}

func (b *pipelineExecutionSummaryBuilder) pipelineRunnerInfo(
	pipelineRecord *core.Record,
) pipeline.PipelineRunnerInfo {
	if pipelineRecord == nil {
		return pipeline.PipelineRunnerInfo{}
	}
	if info, ok := b.runnerInfo[pipelineRecord.Id]; ok {
		return info
	}
	info, _ := pipeline.ParsePipelineRunnerInfo(pipelineRecord.GetString("yaml"))
	b.runnerInfo[pipelineRecord.Id] = info
	return info
}

func localizePipelineWorkflowSummaries(list []*pipelineWorkflowSummary, loc *time.Location) {
	for _, summary := range list {
		if summary == nil {
			continue
		}

		if t, err := utils.ParseTimeString(summary.StartTime); err == nil {
			summary.StartTime = t.In(loc).Format("02/01/2006, 15:04:05")
		}
		if t, err := utils.ParseTimeString(summary.EndTime); err == nil {
			summary.EndTime = t.In(loc).Format("02/01/2006, 15:04:05")
		}
		if len(summary.Children) > 0 {
			localizeWorkflowExecutionSummaries(summary.Children, loc)
		}
	}
}

func buildWorkflowExecutionSummary(
	ctx context.Context,
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

	if c != nil && summary.Status == string(WorkflowStatusFailed) {
		if failure := fetchWorkflowFailure(
			ctx,
			c,
			exec.Execution.WorkflowID,
			exec.Execution.RunID,
		); failure != nil {
			summary.FailureReason = failure
		}
	}

	return summary
}

func attachPipelineArtifactsToSummary(
	summary *WorkflowExecutionSummary,
	app core.App,
	record *core.Record,
) {
	if summary == nil || record == nil {
		return
	}
	artifacts := pipelineresults.BuildPipelineExecutionArtifacts(app, record)
	summary.Results = artifacts.Results
	summary.Report = artifacts.Report
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
	ctx context.Context,
	temporalClient client.Client,
	workflowID string,
	runID string,
) (*WorkflowExecution, *apierror.APIError) {
	workflowExecution, err := temporalClient.DescribeWorkflowExecution(
		ctx,
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
	decodedSearchAttributes, err := decodeWorkflowSearchAttributes(
		workflowExecution.GetWorkflowExecutionInfo().GetSearchAttributes(),
	)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"workflow",
			"failed to decode workflow search attributes",
			err.Error(),
		)
	}
	if len(decodedSearchAttributes) > 0 {
		execInfo.SearchAttributes = &decodedSearchAttributes
	}

	return &execInfo, nil
}

type pipelineWorkflowSummary struct {
	WorkflowExecutionSummary
	PipelineIdentifier string           `json:"pipeline_identifier,omitempty"`
	PipelineName       string           `json:"pipeline_name,omitempty"`
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
		PipelineIdentifier: workflowengine.NormalizePipelineIdentifier(
			queued.PipelineIdentifier,
		),
		PipelineName:  displayName,
		RunnerIDs:     runnerIDs,
		RunnerRecords: pipeline.ResolveRunnerRecords(app, runnerIDs, runnerCache),
	}
}

func resolvePipelineNameFromRecord(pipelineRecord *core.Record, fallback string) string {
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		fallback = "pipeline-run"
	}

	if pipelineRecord == nil {
		return fallback
	}

	yaml := pipelineRecord.GetString("yaml")
	if yaml != "" {
		wfDef, err := pipelineinternal.ParseWorkflow(yaml)
		if err == nil {
			if name := strings.TrimSpace(wfDef.Name); name != "" {
				return name
			}
		}
	}

	if name := strings.TrimSpace(pipelineRecord.GetString("name")); name != "" {
		return name
	}

	return fallback
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

	dc := temporalcrypto.DataConverter()

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

		return pipeline.GlobalRunnerIDFromConfig(in.WorkflowInput.Config), nil
	}

	return "", nil
}
