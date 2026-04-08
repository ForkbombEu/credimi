// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"math"
	"net/http"
	"sort"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

const AggregateScoreboardTaskQueue = "aggregate-scoreboard-task-queue"

type AggregatedPipelineStats struct {
	PipelineID          string   `json:"pipeline_id"`
	PipelineName        string   `json:"pipeline_name"`
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
	LastExecution       *LatestExecutionDetails `json:"last_execution,omitempty"`
}

type LatestExecutionDetails struct {
	PipelineName          string   `json:"pipeline_name"`
	OrgLogo               string   `json:"org_logo,omitempty"`
	Video                 string   `json:"video,omitempty"`
	Screenshot            string   `json:"screenshots,omitempty"`
	Logs                  string   `json:"logs,omitempty"`
	WalletUsed            []string `json:"wallet_used,omitempty"`
	WalletVersionUsed     []string `json:"wallet_version_used,omitempty"`
	MaestroScripts        []string `json:"maestro_scripts,omitempty"`
	Credentials           []string `json:"credentials,omitempty"`
	Issuers               []string `json:"issuers,omitempty"`
	UseCaseVerifications  []string `json:"use_case_verifications,omitempty"`
	Verifiers             []string `json:"verifiers,omitempty"`
	ConformanceTests      []string `json:"conformance_tests,omitempty"`
	CustomChecks          []string `json:"custom_checks,omitempty"`
}

type AggregateScoreboardWorkflowOutput struct {
	AggregatedPipelines []AggregatedPipelineStats `json:"aggregated_pipelines"`
	NamespacesProcessed int      `json:"namespaces_processed"`
	NamespacesFailed    int      `json:"namespaces_failed"`
	FailedNamespaces    []string `json:"failed_namespaces,omitempty"`
}

type AggregateScoreboardWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

func NewAggregateScoreboardWorkflow() *AggregateScoreboardWorkflow {
	w := &AggregateScoreboardWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (w *AggregateScoreboardWorkflow) Name() string {
	return "AggregateScoreboardWorkflow"
}

func (w *AggregateScoreboardWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *AggregateScoreboardWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *AggregateScoreboardWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	
	if input.ActivityOptions != nil {
		ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)
	} else {
		defaultOpts := w.GetOptions()
		ctx = workflow.WithActivityOptions(ctx, defaultOpts)
	}

	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			input.RunMetadata,
		)
	}
	
	apiKey, _ := input.Config["api_key"].(string)

	httpActivity := activities.NewInternalHTTPActivity()
	var httpResult workflowengine.ActivityResult

	namespacesRequest := workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method:         http.MethodGet,
			URL:            utils.JoinURL(appURL, "api", "organizations", "namespaces"),
			ExpectedStatus: http.StatusOK,
			Headers: map[string]string{
				"Credimi-Api-Key": apiKey,
			},
		},
	}

	err := workflow.ExecuteActivity(ctx, httpActivity.Name(), namespacesRequest).Get(ctx, &httpResult)
	if err != nil {
		logger.Error("Failed to get namespaces", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, input.RunMetadata)
	}

	body, ok := httpResult.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		return workflowengine.WorkflowResult{}, workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
			"response body is not a map",
			httpResult.Output,
		)
	}

	namespacesRaw, ok := body["namespaces"].([]interface{})
	if !ok {
		return workflowengine.WorkflowResult{}, workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
			"namespaces field missing or invalid",
			body,
		)
	}

	namespaces := make([]string, len(namespacesRaw))
	for i, ns := range namespacesRaw {
		namespaces[i] = ns.(string)
	}

	if len(namespaces) == 0 {
		return workflowengine.WorkflowResult{
			Message: "No namespaces found",
			Output: AggregateScoreboardWorkflowOutput{
				AggregatedPipelines: []AggregatedPipelineStats{},
				NamespacesProcessed: 0,
			},
		}, nil
	}

	var scoreboardFutures []workflow.Future
	var scoreboardNamespaces []string

	for _, ns := range namespaces {
		scoreboardRequest := workflowengine.ActivityInput{
			Payload: activities.InternalHTTPActivityPayload{
				Method:         http.MethodGet,
				URL:            utils.JoinURL(appURL, "api", "pipeline", "scoreboard", ns),
				ExpectedStatus: http.StatusOK,
				Headers: map[string]string{
					"Credimi-Api-Key": apiKey,
				},
			},
		}
		future := workflow.ExecuteActivity(ctx, httpActivity.Name(), scoreboardRequest)
		scoreboardFutures = append(scoreboardFutures, future)
		scoreboardNamespaces = append(scoreboardNamespaces, ns)
	}

	type rawPipelineData struct {
		Namespace   string
		Pipeline    map[string]interface{}
		LastRun     *struct {
        	WorkflowID string
        	RunID      string
        	StartTime  string
    	}
	}
	
	var allRawPipelines []rawPipelineData
	var failedNamespaces []string

	for i, future := range scoreboardFutures {
		var result workflowengine.ActivityResult
		err := future.Get(ctx, &result)
		if err != nil {
			logger.Error("Failed to fetch scoreboard", "namespace", scoreboardNamespaces[i], "error", err)
			failedNamespaces = append(failedNamespaces, scoreboardNamespaces[i])
			continue
		}

		respBody, ok := result.Output.(map[string]any)["body"]
		
		if !ok {
			logger.Error("Invalid response body", "namespace", scoreboardNamespaces[i])
			failedNamespaces = append(failedNamespaces, scoreboardNamespaces[i])
			continue
		}

		pipelines, ok := respBody.([]interface{})
		if !ok {
			logger.Error("Response body is not an array", "namespace", scoreboardNamespaces[i])
			failedNamespaces = append(failedNamespaces, scoreboardNamespaces[i])
			continue
		}

		for _, p := range pipelines {
			pipeline, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			var lastRun *struct {
        		WorkflowID string
        		RunID      string
        		StartTime  string
    		}
			
			if lastRunRaw, ok := pipeline["last_successful_run"]; ok && lastRunRaw != nil {
				if lastRunMap, ok := lastRunRaw.(map[string]interface{}); ok {
					if startTime, ok := lastRunMap["start_time"].(string); ok {
						workflowID, _ := lastRunMap["workflow_id"].(string)
						runID, _ := lastRunMap["run_id"].(string)						
						lastRun = &struct {
                    		WorkflowID string
                    		RunID      string
                    		StartTime  string
                		}{
                    		WorkflowID: workflowID,
                    		RunID:      runID,
                    		StartTime:  startTime,
                		}
					}
				}
			}
			allRawPipelines = append(allRawPipelines, rawPipelineData{
        		Namespace: scoreboardNamespaces[i],
        		Pipeline:  pipeline,
        		LastRun:   lastRun,
    		})
		}
	}

	aggregatedMap := make(map[string]*AggregatedPipelineStats)

	lastRunMap := make(map[string]*struct {
    	WorkflowID string
   		RunID      string
		StartTime  string
    	Namespace  string
	})

	for _, raw := range allRawPipelines {
		p := raw.Pipeline
		pipelineName, ok := p["pipeline_name"].(string)
		if !ok || pipelineName == "" {
    		continue
		}		

		if _, exists := aggregatedMap[pipelineName]; !exists {
    		aggregatedMap[pipelineName] = &AggregatedPipelineStats{
        		PipelineName:       pipelineName,
        		RunnerTypes:        []string{},
        		Runners:            []string{},
    		}
		}

		stats := aggregatedMap[pipelineName]

		if totalRuns, ok := p["total_runs"].(float64); ok {
			stats.TotalRuns += int(totalRuns)
		}
		if totalSuccesses, ok := p["total_successes"].(float64); ok {
			stats.TotalSuccesses += int(totalSuccesses)
		}
		if manual, ok := p["manual_executions"].(float64); ok {
			stats.ManualExecutions += int(manual)
		}
		if scheduled, ok := p["scheduled_executions"].(float64); ok {
			stats.ScheduledExecutions += int(scheduled)
		}
		
		if runners, ok := p["runners"].([]interface{}); ok {
			for _, r := range runners {
				if rs, ok := r.(string); ok {
					stats.Runners = appendUnique(stats.Runners, rs)
				}
			}
		}
		
		if runnerTypes, ok := p["runner_types"].([]interface{}); ok {
			for _, rt := range runnerTypes {
				if rts, ok := rt.(string); ok {
					stats.RunnerTypes = appendUnique(stats.RunnerTypes, rts)
				}
			}
		}
		
		if firstDate, ok := p["first_execution_date"].(string); ok && firstDate != "" {
			if stats.FirstExecutionDate == "" || firstDate < stats.FirstExecutionDate {
				stats.FirstExecutionDate = firstDate
			}
		}
		if lastDate, ok := p["last_execution_date"].(string); ok && lastDate != "" {
			if stats.LastExecutionDate == "" || lastDate > stats.LastExecutionDate {
				stats.LastExecutionDate = lastDate
			}
		}
		
		if minTime, ok := p["min_execution_time"].(string); ok && minTime != "" {
			if stats.MinExecutionTime == "" || minTime < stats.MinExecutionTime {
				stats.MinExecutionTime = minTime
			}
		}

		if raw.LastRun != nil {
        	existingLastRun := lastRunMap[pipelineName]
        	if existingLastRun == nil || raw.LastRun.StartTime > existingLastRun.StartTime {
            	lastRunMap[pipelineName] = &struct {
                	WorkflowID string
                	RunID      string
                	StartTime  string
                	Namespace  string
            	}{
                	WorkflowID: raw.LastRun.WorkflowID,
                	RunID:      raw.LastRun.RunID,
                	StartTime:  raw.LastRun.StartTime,
                	Namespace:  raw.Namespace,
            	}
        	}
    	}
	}
	
	for _, stats := range aggregatedMap {
		if stats.TotalRuns > 0 {
			stats.SuccessRate = math.Round(float64(stats.TotalSuccesses) / float64(stats.TotalRuns) * 10000)/100
		}
		sort.Strings(stats.Runners)
		sort.Strings(stats.RunnerTypes)
	}
	
	for pipelineName, lastRun := range lastRunMap {
    	if lastRun == nil {
        	continue
    	}
		stats, exists := aggregatedMap[pipelineName]
    	if !exists {
        	logger.Error("Pipeline not found in aggregated map", "pipeline_name", pipelineName)
        	continue
    	}
    
    	detailsURL := utils.JoinURL(
        	appURL,
        	"api", "pipeline", "execution-details",
        	lastRun.Namespace,
        	lastRun.WorkflowID,
        	lastRun.RunID,
    	)
    
    	detailsRequest := workflowengine.ActivityInput{
        	Payload: activities.InternalHTTPActivityPayload{
            	Method:         http.MethodGet,
            	URL:            detailsURL,
            	ExpectedStatus: http.StatusOK,
            	Headers: map[string]string{
                	"Credimi-Api-Key": apiKey,
            	},
        	},
    	}
    
    	var detailsResult workflowengine.ActivityResult
    	err = workflow.ExecuteActivity(ctx, httpActivity.Name(), detailsRequest).Get(ctx, &detailsResult)
    	if err != nil {
        	logger.Error("Failed to fetch execution details", "pipeline_name", pipelineName, "error", err)
        	continue
    	}
    
    	if detailsBody, ok := detailsResult.Output.(map[string]any)["body"].(map[string]any); ok {
        	stats.LastExecution = &LatestExecutionDetails{
            	PipelineName:          getString(detailsBody, "pipeline_name"),
            	OrgLogo:               getString(detailsBody, "org_logo"),
            	Video:                 getString(detailsBody, "video"),
            	Screenshot:            getString(detailsBody, "screenshots"),
            	Logs:                  getString(detailsBody, "logs"),
            	WalletUsed:            getStringSlice(detailsBody, "wallet_used"),
            	WalletVersionUsed:     getStringSlice(detailsBody, "wallet_version_used"),
            	MaestroScripts:        getStringSlice(detailsBody, "maestro_scripts"),
            	Credentials:           getStringSlice(detailsBody, "credentials"),
            	Issuers:               getStringSlice(detailsBody, "issuers"),
            	UseCaseVerifications:  getStringSlice(detailsBody, "use_case_verifications"),
            	Verifiers:             getStringSlice(detailsBody, "verifiers"),
            	ConformanceTests:      getStringSlice(detailsBody, "conformance_tests"),
            	CustomChecks:          getStringSlice(detailsBody, "custom_checks"),
        	}
		} else {
			logger.Error("Invalid execution details response body", "pipeline_name", pipelineName)
		}
	}
	aggregatedPipelines := make([]AggregatedPipelineStats, 0, len(aggregatedMap))
	for _, stats := range aggregatedMap {
		aggregatedPipelines = append(aggregatedPipelines, *stats)
	}

	sort.Slice(aggregatedPipelines, func(i, j int) bool {
		return aggregatedPipelines[i].PipelineName < aggregatedPipelines[j].PipelineName
	})
	output := AggregateScoreboardWorkflowOutput{
		AggregatedPipelines:   aggregatedPipelines,
		NamespacesProcessed:   len(namespaces) - len(failedNamespaces),
		NamespacesFailed:      len(failedNamespaces),
		FailedNamespaces:      failedNamespaces,
	}

	return workflowengine.WorkflowResult{
		Message: "Successfully aggregated scoreboard across namespaces",
		Output:  output,
	}, nil
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStringSlice(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	if v, ok := m[key].([]interface{}); ok {
		result := make([]string, len(v))
		for i, item := range v {
			if s, ok := item.(string); ok {
				result[i] = s
			}
		}
		return result
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
