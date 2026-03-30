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
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type PipelineStatsResponse struct {
	PipelineID          string   `json:"pipeline_id"`
	PipelineName        string   `json:"pipeline_name"`
	PipelineIdentifier  string   `json:"pipeline_identifier"`
	RunnerTypes         []string `json:"runner_types"`          
	Runners             []string `json:"runners"`               
	TotalRuns           int      `json:"total_runs"`
	TotalSuccesses      int      `json:"total_successes"`
	SuccessRate         float64  `json:"success_rate"`          
	ManualExecutions    int      `json:"manual_executions"`
	ScheduledExecutions int      `json:"scheduled_executions"`
	MinExecutionTime    string   `json:"min_execution_time"`    
	FirstExecutionDate  string   `json:"first_execution_date"`
	LastExecutionDate   string   `json:"last_execution_date"`
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
            stats := calculateStatsFromExecutions(pipelineExecutions, e.App, runnerCache)
            
            response = append(response, PipelineStatsResponse{
                PipelineID:          pipelineID,
                PipelineName:        pipelineName,
                PipelineIdentifier:  fmt.Sprintf("%s/%s", namespace, pipelineRecord.GetString("canonified_name")),
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
            })
        }
		return e.JSON(http.StatusOK, response)
	}
}

func calculateStatsFromExecutions(
    executions []*WorkflowExecution,
    app core.App,
    runnerCache map[string]map[string]any,
) *PipelineStats {
    stats := &PipelineStats{
        Runners:     []string{},
        RunnerTypes: []string{},
        TotalRuns:   len(executions),
    }

    if len(executions) == 0 {
        return stats
    }

    runnerSet := make(map[string]struct{})
    var minDuration time.Duration
    var firstTime, lastTime string
    minDurationSet := false

    for _, exec := range executions {
        if exec == nil || exec.SearchAttributes == nil {
            continue
        }

        isCompleted := extractCompletionStatus(exec)
        if isCompleted {
            stats.TotalSuccesses++
        }

        isScheduled := isScheduledExecution(exec)
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

        updateMinDuration(exec, &minDuration, &minDurationSet)
    }

    stats.Runners = mapKeysToSlice(runnerSet)
    stats.RunnerTypes = resolveRunnerTypes(app, stats.Runners, runnerCache)

    if stats.TotalRuns > 0 {
        stats.SuccessRate = math.Round(float64(stats.TotalSuccesses) / float64(stats.TotalRuns) * 10000)/100
    }

    stats.FirstExecutionDate = firstTime
    stats.LastExecutionDate = lastTime
    stats.MinExecutionTime = formatDurationString(minDuration, minDurationSet)

    return stats
}

func extractCompletionStatus(exec *WorkflowExecution) bool {
    if statusVal, ok := (*exec.SearchAttributes)["ExecutionStatus"]; ok {
        if status, ok := statusVal.(string); ok && status == "Completed" {
            return true
        }
    }
    return false
}

func isScheduledExecution(exec *WorkflowExecution) bool {
    if scheduledVal, ok := (*exec.SearchAttributes)["TemporalScheduledById"]; ok {
        if s, ok := scheduledVal.(string); ok && s != "" {
            return true
        }
    }
    return false
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

func resolveRunnerTypes(app core.App, runnerIDs []string, runnerCache map[string]map[string]any) []string {
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
        if seconds > 0 {
            return fmt.Sprintf("%dm%ds", minutes, seconds)
        }
        return fmt.Sprintf("%dm", minutes)
    default:
        hours := int(d.Hours())
        minutes := int(d.Minutes()) % 60
        seconds := int(d.Seconds()) % 60
        if minutes > 0 && seconds > 0 {
            return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
        }
        if minutes > 0 {
            return fmt.Sprintf("%dh%dm", hours, minutes)
        }
        return fmt.Sprintf("%dh", hours)
    }
}
