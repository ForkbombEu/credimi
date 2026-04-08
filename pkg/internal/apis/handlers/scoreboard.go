// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/sdk/client"
)

const aggregateScoreboardNamespace = "default"

var aggregateScoreboardWorkflowStart = func(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	w := workflows.NewAggregateScoreboardWorkflow()
	return w.Start(namespace, input)
}

type StartAggregateScoreboardResponse struct {
	WorkflowID        string `json:"workflowId"`
	WorkflowRunID     string `json:"workflowRunId"`
	Message           string `json:"message"`
	WorkflowNamespace string `json:"workflowNamespace"`
}

type PipelineStatsResponse struct {
	PipelineID          string             `json:"pipeline_id"`
	PipelineName        string             `json:"pipeline_name"`
	PipelineIdentifier  string             `json:"pipeline_identifier"`
	RunnerTypes         []string           `json:"runner_types"`
	Runners             []string           `json:"runners"`
	TotalRuns           int                `json:"total_runs"`
	TotalSuccesses      int                `json:"total_successes"`
	SuccessRate         float64            `json:"success_rate"`
	ManualExecutions    int                `json:"manual_executions"`
	ScheduledExecutions int                `json:"scheduled_executions"`
	MinExecutionTime    string             `json:"min_execution_time"`
	FirstExecutionDate  string             `json:"first_execution_date"`
	LastExecutionDate   string             `json:"last_execution_date"`
	LastSuccessfulRun   *LastSuccessfulRun `json:"last_successful_run,omitempty"`
}

type LastSuccessfulRun struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	StartTime  string `json:"start_time"`
}

type PipelineStats struct {
	PipelineName        string
	Runners             []string
	RunnerTypes         []string
	TotalRuns           int
	TotalSuccesses      int
	SuccessRate         float64
	ManualExecutions    int
	ScheduledExecutions int
	MinExecutionTime    string
	FirstExecutionDate  string
	LastExecutionDate   string
}

type LastExecutionDetails struct {
	PipelineName         string   `json:"pipeline_name"`
	OrgLogo              string   `json:"org_logo,omitempty"`
	Video                string   `json:"video_results,omitempty"`
	Screenshots          string   `json:"screenshots,omitempty"`
	Logs                 string   `json:"logs,omitempty"`
	WalletUsed           []string `json:"wallet_used,omitempty"`
	WalletVersionUsed    []string `json:"wallet_version_used,omitempty"`
	MaestroScripts       []string `json:"maestro_scripts,omitempty"`
	Credentials          []string `json:"credentials,omitempty"`
	Issuers              []string `json:"issuers,omitempty"`
	UseCaseVerifications []string `json:"use_case_verifications,omitempty"`
	Verifiers            []string `json:"verifiers,omitempty"`
	ConformanceTests     []string `json:"conformance_tests,omitempty"`
	CustomChecks         []string `json:"custom_checks,omitempty"`
}

func HandleStartAggregateScoreboard() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		workflowResult, err := aggregateScoreboardWorkflowStart(
			aggregateScoreboardNamespace,
			workflowengine.WorkflowInput{
				Config: map[string]any{
					"app_url": e.App.Settings().Meta.AppURL,
				},
			},
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to start aggregate scoreboard workflow",
				err.Error(),
			).JSON(e)
		}

		return e.JSON(http.StatusOK, StartAggregateScoreboardResponse{
			WorkflowID:        workflowResult.WorkflowID,
			WorkflowRunID:     workflowResult.WorkflowRunID,
			Message:           workflowResult.Message,
			WorkflowNamespace: aggregateScoreboardNamespace,
		})
	}
}

func HandleGetPipelineScoreboard() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		namespace := e.Request.PathValue("namespace")
		if namespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"namespace",
				"namespace is required",
				"please provide a namespace in the path",
			).JSON(e)
		}
		pipelineRecords, err := e.App.FindRecordsByFilter(
			"pipelines",
			"published={:published}",
			"",
			-1,
			0,
			dbx.Params{
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
			return e.JSON(http.StatusOK, []PipelineStatsResponse{})
		}
		pipelineMap := make(map[string]*core.Record)
		for _, record := range pipelineRecords {
			pipelineMap[record.Id] = record
		}

		pipelineIdentifierIndex := buildPipelineIdentifierIndex(e.App, pipelineMap)

		temporalClient, err := pipelineResultsTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create temporal client",
				err.Error(),
			).JSON(e)
		}

		executions, err := listPipelineWorkflowExecutions(
			context.Background(),
			temporalClient,
			namespace,
			nil,
			"",
			pipelineListWorkflowsDefaultLimit,
			0,
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to list workflows",
				err.Error(),
			).JSON(e)
		}
		pipelineIdentifiers := resolvePipelineIdentifiersForExecutions(executions)

		executionsByPipelineID := make(map[string][]*WorkflowExecution)

		for _, exec := range executions {
			if exec == nil || exec.Execution == nil {
				continue
			}
			ref := workflowExecutionRef{
				WorkflowID: exec.Execution.WorkflowID,
				RunID:      exec.Execution.RunID,
			}
			pipelineIdentifier := pipelineIdentifiers[ref]
			if pipelineIdentifier == "" {
				continue
			}
			pipelineRecord := pipelineIdentifierIndex[pipelineIdentifier]
			if pipelineRecord == nil {
				continue
			}
			executionsByPipelineID[pipelineRecord.Id] = append(
				executionsByPipelineID[pipelineRecord.Id],
				exec,
			)
		}
		response := make([]PipelineStatsResponse, 0, len(executionsByPipelineID))

		for pipelineID, pipelineExecutions := range executionsByPipelineID {
			pipelineRecord := pipelineMap[pipelineID]
			if pipelineRecord == nil {
				continue
			}

			pipelineName := pipelineRecord.GetString("name")

			runnerCache := make(map[string]map[string]any)
			stats, lastSuccessfulRun := calculateStatsFromExecutions(
				pipelineExecutions,
				e.App,
				runnerCache,
			)

			response = append(response, PipelineStatsResponse{
				PipelineID:   pipelineID,
				PipelineName: pipelineName,
				PipelineIdentifier: fmt.Sprintf(
					"%s/%s",
					namespace,
					pipelineRecord.GetString("canonified_name"),
				),
				RunnerTypes:         stats.RunnerTypes,
				Runners:             stats.Runners,
				TotalRuns:           stats.TotalRuns,
				TotalSuccesses:      stats.TotalSuccesses,
				SuccessRate:         stats.SuccessRate,
				ManualExecutions:    stats.ManualExecutions,
				ScheduledExecutions: stats.ScheduledExecutions,
				MinExecutionTime:    stats.MinExecutionTime,
				FirstExecutionDate:  stats.FirstExecutionDate,
				LastExecutionDate:   stats.LastExecutionDate,
				LastSuccessfulRun:   lastSuccessfulRun,
			})
		}
		return e.JSON(http.StatusOK, response)
	}
}

func HandleGetExecutionDetails() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		namespace := e.Request.PathValue("namespace")
		workflowID := e.Request.PathValue("workflow_id")
		runID := e.Request.PathValue("run_id")

		if namespace == "" || workflowID == "" || runID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"params",
				"namespace, workflow_id and run_id are required",
				"").JSON(e)
		}

		temporalClient, err := pipelineResultsTemporalClient(namespace)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"temporal",
				"unable to create temporal client",
				err.Error()).JSON(e)
		}
		exec, apiErr := getWorkflowExecutionWithDecodedAttrs(temporalClient, workflowID, runID)
		if apiErr != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"workflow",
				"failed to get workflow execution",
				apiErr.Error()).JSON(e)
		}
		pipelineIdentifier := pipelineIdentifierFromSearchAttributes(exec.SearchAttributes)

		parts := strings.SplitN(pipelineIdentifier, "/", 2)
		pipelineName := ""
		if len(parts) == 2 {
			pipelineName = parts[1]
		}

		resultRecord, _ := e.App.FindFirstRecordByFilter(
			"pipeline_results",
			"workflow_id={:workflow_id} && run_id={:run_id}",
			dbx.Params{
				"workflow_id": workflowID,
				"run_id":      runID,
			},
		)

		video, screenshot, logs := getPipelineResultFromRecord(e.App, resultRecord)
		entityDetails := extractEntityDetailsFromExecution(exec)

		response := LastExecutionDetails{
			PipelineName:         pipelineName,
			OrgLogo:              getOrgLogo(e.App, namespace),
			Video:                video,
			Screenshots:          screenshot,
			Logs:                 logs,
			WalletUsed:           entityDetails.WalletUsed,
			WalletVersionUsed:    entityDetails.WalletVersionUsed,
			MaestroScripts:       entityDetails.MaestroScripts,
			Credentials:          entityDetails.Credentials,
			Issuers:              entityDetails.Issuers,
			UseCaseVerifications: entityDetails.UseCaseVerifications,
			Verifiers:            entityDetails.Verifiers,
			ConformanceTests:     entityDetails.ConformanceTests,
			CustomChecks:         entityDetails.CustomChecks,
		}

		return e.JSON(http.StatusOK, response)
	}
}

func getWorkflowExecutionWithDecodedAttrs(
	temporalClient client.Client,
	workflowID string,
	runID string,
) (*WorkflowExecution, error) {
	resp, err := temporalClient.DescribeWorkflowExecution(
		context.Background(),
		workflowID,
		runID,
	)
	if err != nil {
		return nil, err
	}

	execInfo := resp.GetWorkflowExecutionInfo()
	var decodedAttrs DecodedWorkflowSearchAttributes
	if execInfo.GetSearchAttributes() != nil {
		decodedAttrs, err = decodeWorkflowSearchAttributes(execInfo.GetSearchAttributes())
		if err != nil {
			return nil, err
		}
	}

	return &WorkflowExecution{
		Execution: &WorkflowIdentifier{
			WorkflowID: execInfo.GetExecution().GetWorkflowId(),
			RunID:      execInfo.GetExecution().GetRunId(),
		},
		Type:             WorkflowType{Name: execInfo.GetType().GetName()},
		SearchAttributes: &decodedAttrs,
	}, nil
}

func calculateStatsFromExecutions(
	executions []*WorkflowExecution,
	app core.App,
	runnerCache map[string]map[string]any,
) (*PipelineStats, *LastSuccessfulRun) {
	stats := &PipelineStats{
		Runners:     []string{},
		RunnerTypes: []string{},
		TotalRuns:   len(executions),
	}

	if len(executions) == 0 {
		return stats, nil
	}

	runnerSet := make(map[string]struct{})
	var minDuration time.Duration
	var firstTime, lastTime string
	minDurationSet := false

	var lastSuccessfulExec *WorkflowExecution

	for _, exec := range executions {
		if exec == nil || exec.SearchAttributes == nil {
			continue
		}

		isCompleted := extractCompletionStatus(exec)
		if isCompleted {
			stats.TotalSuccesses++

			if lastSuccessfulExec == nil || exec.StartTime > lastSuccessfulExec.StartTime {
				lastSuccessfulExec = exec
			}
		}

		isScheduled := strings.HasPrefix(exec.Execution.WorkflowID, "Pipeline-Sched-")
		if isScheduled {
			stats.ScheduledExecutions++
		} else {
			stats.ManualExecutions++
		}

		runnerIDs := extractRunnerIDsFromExec(exec)
		for _, id := range runnerIDs {
			runnerSet[id] = struct{}{}
		}

		updateDateRange(exec.StartTime, &firstTime, &lastTime)

		if isCompleted {
			updateMinDuration(exec, &minDuration, &minDurationSet)
		}
	}

	stats.Runners = mapKeysToSlice(runnerSet)
	stats.RunnerTypes = resolveRunnerTypes(app, stats.Runners, runnerCache)

	if stats.TotalRuns > 0 {
		stats.SuccessRate = math.Round(
			float64(stats.TotalSuccesses)/float64(stats.TotalRuns)*10000,
		) / 100
	}

	stats.FirstExecutionDate = firstTime
	stats.LastExecutionDate = lastTime
	stats.MinExecutionTime = formatDurationString(minDuration, minDurationSet)

	var lastSuccessfulRun *LastSuccessfulRun
	if lastSuccessfulExec != nil {
		lastSuccessfulRun = &LastSuccessfulRun{
			WorkflowID: lastSuccessfulExec.Execution.WorkflowID,
			RunID:      lastSuccessfulExec.Execution.RunID,
			StartTime:  lastSuccessfulExec.StartTime,
		}
	}

	return stats, lastSuccessfulRun
}

func extractCompletionStatus(exec *WorkflowExecution) bool {
	if exec == nil {
		return false
	}

	return normalizeTemporalStatus(exec.Status) == string(WorkflowStatusCompleted)
}

func extractRunnerIDsFromExec(exec *WorkflowExecution) []string {
	if runnerVal, ok := (*exec.SearchAttributes)[workflowengine.RunnerIdentifiersSearchAttribute]; ok {
		switch v := runnerVal.(type) {
		case []string:
			return v
		case []interface{}:
			runnerIDs := make([]string, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); ok {
					runnerIDs = append(runnerIDs, s)
				}
			}
			return runnerIDs
		}
	}
	return nil
}

func updateDateRange(startTimeStr string, firstTime, lastTime *string) {
	if startTimeStr == "" {
		return
	}
	if *firstTime == "" || startTimeStr < *firstTime {
		*firstTime = startTimeStr
	}
	if *lastTime == "" || startTimeStr > *lastTime {
		*lastTime = startTimeStr
	}
}

func updateMinDuration(exec *WorkflowExecution, minDuration *time.Duration, minDurationSet *bool) {
	if exec.StartTime == "" || exec.CloseTime == "" {
		return
	}
	startTime, err1 := time.Parse(time.RFC3339, exec.StartTime)
	closeTime, err2 := time.Parse(time.RFC3339, exec.CloseTime)
	if err1 != nil || err2 != nil {
		return
	}
	duration := closeTime.Sub(startTime)
	if !*minDurationSet || duration < *minDuration {
		*minDuration = duration
		*minDurationSet = true
	}
}

func mapKeysToSlice(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func resolveRunnerTypes(
	app core.App,
	runnerIDs []string,
	runnerCache map[string]map[string]any,
) []string {
	if len(runnerIDs) == 0 || app == nil {
		return []string{}
	}
	runnerRecords := pipeline.ResolveRunnerRecords(app, runnerIDs, runnerCache)
	types := make([]string, 0, len(runnerRecords))
	for _, record := range runnerRecords {
		if runnerType, ok := record["type"].(string); ok && runnerType != "" {
			types = append(types, runnerType)
		}
	}
	sort.Strings(types)
	return types
}

func formatDurationString(d time.Duration, set bool) string {
	if !set {
		return ""
	}

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%.0fs", d.Seconds())
	case d < time.Hour:
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	default:
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	}
}

func extractFirstTwoParts(fullPath string) string {
	parts := strings.Split(fullPath, "/")
	if len(parts) >= 2 {
		return strings.Join(parts[:len(parts)-1], "/")
	}
	return fullPath
}

func getPipelineResultFromRecord(
	app core.App,
	record *core.Record,
) (video, screenshot, logs string) {
	if record == nil {
		return "", "", ""
	}

	results := computePipelineResultsFromRecord(app, record)
	if len(results) == 0 {
		return "", "", ""
	}

	first := results[0]
	return first.Video, first.Screenshot, first.Log
}

func extractEntityDetailsFromExecution(exec *WorkflowExecution) *LastExecutionDetails {
	if exec == nil || exec.SearchAttributes == nil {
		return &LastExecutionDetails{}
	}

	attrs := *exec.SearchAttributes

	details := &LastExecutionDetails{}

	// version_id
	details.WalletVersionUsed = getStringListFromAttrs(attrs, "VersionsID")
	for _, v := range details.WalletVersionUsed {
		walletUsed := extractFirstTwoParts(v)
		details.WalletUsed = appendUnique(details.WalletUsed, walletUsed)
	}

	// action_id
	details.MaestroScripts = getStringListFromAttrs(attrs, "ActionsID")

	// credential_id
	details.Credentials = getStringListFromAttrs(attrs, "CredentialsID")
	for _, cred := range details.Credentials {
		issuer := extractFirstTwoParts(cred)
		details.Issuers = appendUnique(details.Issuers, issuer)
	}

	// use_case_id
	details.UseCaseVerifications = getStringListFromAttrs(attrs, "UseCaseID")
	for _, uc := range details.UseCaseVerifications {
		verifier := extractFirstTwoParts(uc)
		details.Verifiers = appendUnique(details.Verifiers, verifier)
	}

	// check_id (conformance e custom)
	details.ConformanceTests = getStringListFromAttrs(attrs, "ConformanceCheckID")
	details.CustomChecks = getStringListFromAttrs(attrs, "CustomCheckID")

	return details
}

func getStringListFromAttrs(attrs DecodedWorkflowSearchAttributes, key string) []string {
	if val, ok := attrs[key]; ok {
		switch v := val.(type) {
		case []string:
			return v
		case []interface{}:
			result := make([]string, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

func appendUnique(slice []string, item string) []string {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}

func getOrgLogo(app core.App, namespace string) string {
	if namespace == "" {
		return ""
	}

	org, err := app.FindFirstRecordByFilter(
		"organizations",
		"canonified_name = {:canonified_name}",
		dbx.Params{"canonified_name": namespace},
	)
	if err != nil {
		return ""
	}

	logo := org.GetString("logo")
	if logo == "" {
		return ""
	}

	return utils.JoinURL(
		app.Settings().Meta.AppURL,
		"api", "files", "organizations",
		org.Id, "logo", logo,
	)
}
