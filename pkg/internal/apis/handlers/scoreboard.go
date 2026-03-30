// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"context"
	"fmt"
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
		//success
		if statusVal, ok := (*exec.SearchAttributes)["ExecutionStatus"]; ok {
            if status, ok := statusVal.(string); ok && status == "Completed" {
                stats.TotalSuccesses++
            }
        }
		//scheduled
		scheduledBy := ""
        if scheduledVal, ok := (*exec.SearchAttributes)["TemporalScheduledById"]; ok {
            if s, ok := scheduledVal.(string); ok {
                scheduledBy = s
            }
        }
        isScheduled := scheduledBy != ""

		if isScheduled {
            stats.ScheduledExecutions++
        } else {
            stats.ManualExecutions++
        }
		//runnerID
		var runnerIDs []string
        if runnerVal, ok := (*exec.SearchAttributes)[workflowengine.RunnerIdentifiersSearchAttribute]; ok {
            switch v := runnerVal.(type) {
            case []string:
                runnerIDs = v
            case []interface{}:
                for _, item := range v {
                    if s, ok := item.(string); ok {
                        runnerIDs = append(runnerIDs, s)
                    }
                }
            }
        }
		for _, id := range runnerIDs {
        	runnerSet[id] = struct{}{}  
   		}
		//date
		if exec.StartTime != "" {
            startTimeStr := exec.StartTime
            if firstTime == "" || startTimeStr < firstTime {
                firstTime = startTimeStr
            }
            if lastTime == "" || startTimeStr > lastTime {
                lastTime = startTimeStr
            }
        }
        
		// minimum duration
        if exec.StartTime != "" && exec.CloseTime != "" {
            startTime, err1 := time.Parse(time.RFC3339, exec.StartTime)
            closeTime, err2 := time.Parse(time.RFC3339, exec.CloseTime)
            if err1 == nil && err2 == nil {
                duration := closeTime.Sub(startTime)
                if !minDurationSet || duration < minDuration {
                    minDuration = duration
                    minDurationSet = true
                }
            }
        }
    }
	stats.Runners = make([]string, 0, len(runnerSet))
    for id := range runnerSet {
        stats.Runners = append(stats.Runners, id)
    }
    sort.Strings(stats.Runners)
	runnerRecords := pipeline.ResolveRunnerRecords(app, stats.Runners, runnerCache)

	runnerTypes := make([]string, 0, len(runnerRecords))
	for _, record := range runnerRecords {
    	if runnerType, ok := record["type"].(string); ok && runnerType != "" {
        	runnerTypes = append(runnerTypes, runnerType)
    	}
	}
	sort.Strings(runnerTypes)

	stats.RunnerTypes = runnerTypes
	
	if stats.TotalRuns > 0 {
        stats.SuccessRate = float64(stats.TotalSuccesses) / float64(stats.TotalRuns) * 100
    }
	stats.FirstExecutionDate = firstTime
    stats.LastExecutionDate = lastTime
    
    if minDurationSet {
    if minDuration < time.Minute {
        stats.MinExecutionTime = fmt.Sprintf("%.0fs", minDuration.Seconds())
    } else if minDuration < time.Hour {
        minutes := int(minDuration.Minutes())
        seconds := int(minDuration.Seconds()) % 60
        stats.MinExecutionTime = fmt.Sprintf("%dm%ds", minutes, seconds)
    } else {
        hours := int(minDuration.Hours())
        minutes := int(minDuration.Minutes()) % 60
		seconds := int(minDuration.Seconds()) % 60
        stats.MinExecutionTime = fmt.Sprintf("%dh%dm%ds", hours, minutes,seconds)
    }
}
    
    return stats
}

