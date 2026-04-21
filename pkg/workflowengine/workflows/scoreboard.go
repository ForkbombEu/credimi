// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const AggregateScoreboardTaskQueue = "AggregateScoreboardTaskQueue"

var aggregateScoreboardStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

type AggregatedPipelineStats struct {
	PipelineID          string                  `json:"pipeline_id"`
	PipelineName        string                  `json:"pipeline_name"`
	RunnerTypes         []string                `json:"runner_types"`
	Runners             []string                `json:"runners"`
	TotalRuns           int                     `json:"total_runs"`
	TotalSuccesses      int                     `json:"total_successes"`
	SuccessRate         float64                 `json:"success_rate"`
	ManualExecutions    int                     `json:"manual_executions"`
	ScheduledExecutions int                     `json:"scheduled_executions"`
	MinExecutionTime    string                  `json:"min_execution_time"`
	FirstExecutionDate  string                  `json:"first_execution_date"`
	LastExecutionDate   string                  `json:"last_execution_date"`
	LastExecution       *LatestExecutionDetails `json:"last_execution,omitempty"`
}

type LatestExecutionDetails struct {
	PipelineName         string   `json:"pipeline_name"`
	WorkflowID           string   `json:"workflow_id,omitempty"`
	RunID                string   `json:"run_id,omitempty"`
	OrgLogo              string   `json:"org_logo,omitempty"`
	Video                string   `json:"video,omitempty"`
	Screenshot           string   `json:"screenshots,omitempty"`
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

type AggregateScoreboardWorkflowOutput struct {
	AggregatedPipelines []AggregatedPipelineStats `json:"aggregated_pipelines"`
	NamespacesProcessed int                       `json:"namespaces_processed"`
	NamespacesFailed    int                       `json:"namespaces_failed"`
	FailedNamespaces    []string                  `json:"failed_namespaces,omitempty"`
}

type AggregateScoreboardWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

type pipelineRunRef struct {
	Namespace  string
	WorkflowID string
	RunID      string
	StartTime  string
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

func (w *AggregateScoreboardWorkflow) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "aggregate-scoreboard-" + uuid.NewString(),
		TaskQueue:                AggregateScoreboardTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}

	return aggregateScoreboardStartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
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
		ctx = workflow.WithActivityOptions(ctx, w.GetOptions())
	}

	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			input.RunMetadata,
		)
	}

	httpActivity := activities.NewInternalHTTPActivity()

	// 1. Get namespaces
	namespaces, err := w.getNamespaces(ctx, httpActivity, appURL)
	if err != nil {
		logger.Error("Failed to get namespaces", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, input.RunMetadata)
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

	// 2. Fetch all scoreboards and aggregate
	aggregatedMap, lastRunMap, failedNamespaces := w.fetchAndAggregateScoreboards(
		ctx, httpActivity, appURL, namespaces,
	)

	// 3. Calculate success rates and sort runners
	for _, stats := range aggregatedMap {
		if stats.TotalRuns > 0 {
			stats.SuccessRate = math.Round(float64(stats.TotalSuccesses)/float64(stats.TotalRuns)*10000) / 100
		}
		sort.Strings(stats.Runners)
		sort.Strings(stats.RunnerTypes)
	}

	// 4. Fetch last execution details
	w.fetchLastExecutionDetails(ctx, httpActivity, appURL, lastRunMap, aggregatedMap)

	// 5. Build output
	aggregatedPipelines := make([]AggregatedPipelineStats, 0, len(aggregatedMap))
	for _, stats := range aggregatedMap {
		aggregatedPipelines = append(aggregatedPipelines, *stats)
	}
	sort.Slice(aggregatedPipelines, func(i, j int) bool {
		return aggregatedPipelines[i].PipelineName < aggregatedPipelines[j].PipelineName
	})

	output := AggregateScoreboardWorkflowOutput{
		AggregatedPipelines: aggregatedPipelines,
		NamespacesProcessed: len(namespaces) - len(failedNamespaces),
		NamespacesFailed:    len(failedNamespaces),
		FailedNamespaces:    failedNamespaces,
	}

	// 6. Save results
	saveURL := utils.JoinURL(appURL, "api", "pipeline", "scoreboard", "save-results")
	savePayload := map[string]interface{}{
		"aggregated_pipelines": output.AggregatedPipelines,
	}
	saveRequest := workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method: http.MethodPost,
			URL:    saveURL,
			Body:   savePayload,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}
	var saveResult workflowengine.ActivityResult
	if err = workflow.ExecuteActivity(ctx, httpActivity.Name(), saveRequest).Get(ctx, &saveResult); err != nil {
		logger.Error("Failed to save results", "error", err)
	}

	return workflowengine.WorkflowResult{
		Message: "Successfully aggregated scoreboard across namespaces",
		Output:  output,
	}, nil
}

// getNamespaces retrieves all namespaces from the API
func (w *AggregateScoreboardWorkflow) getNamespaces(
	ctx workflow.Context,
	httpActivity *activities.InternalHTTPActivity,
	appURL string,
) ([]string, error) {
	var httpResult workflowengine.ActivityResult

	namespacesRequest := workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method:         http.MethodGet,
			URL:            utils.JoinURL(appURL, "api", "organizations", "namespaces"),
			ExpectedStatus: http.StatusOK,
		},
	}

	err := workflow.ExecuteActivity(ctx, httpActivity.Name(), namespacesRequest).Get(ctx, &httpResult)
	if err != nil {
		return nil, err
	}

	body, ok := httpResult.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("response body is not a map")
	}

	namespaces, ok := getRequiredStringSlice(body, "namespaces")
	if !ok {
		return nil, fmt.Errorf("namespaces field missing or invalid")
	}

	return uniqueStrings(namespaces), nil
}

func (w *AggregateScoreboardWorkflow) fetchAndAggregateScoreboards(
	ctx workflow.Context,
	httpActivity *activities.InternalHTTPActivity,
	appURL string,
	namespaces []string,
) (map[string]*AggregatedPipelineStats, map[string]*pipelineRunRef, []string) {
	aggregatedMap := make(map[string]*AggregatedPipelineStats)
	lastRunMap := make(map[string]*pipelineRunRef)
	var failedNamespaces []string

	// Start all parallel activities with preallocated slices
	scoreboardFutures := make([]workflow.Future, 0, len(namespaces))
	scoreboardNamespaces := make([]string, 0, len(namespaces))

	for _, namespace := range namespaces {
		scoreboardRequest := workflowengine.ActivityInput{
			Payload: activities.InternalHTTPActivityPayload{
				Method:         http.MethodGet,
				URL:            utils.JoinURL(appURL, "api", "pipeline", "scoreboard", namespace),
				ExpectedStatus: http.StatusOK,
			},
		}
		scoreboardFutures = append(scoreboardFutures, workflow.ExecuteActivity(ctx, httpActivity.Name(), scoreboardRequest))
		scoreboardNamespaces = append(scoreboardNamespaces, namespace)
	}

	// Process results
	for i, future := range scoreboardFutures {
		w.processScoreboardResponse(ctx, future, scoreboardNamespaces[i],
			aggregatedMap, lastRunMap, &failedNamespaces)
	}

	return aggregatedMap, lastRunMap, failedNamespaces
}

func (w *AggregateScoreboardWorkflow) processScoreboardResponse(
	ctx workflow.Context,
	future workflow.Future,
	namespace string,
	aggregatedMap map[string]*AggregatedPipelineStats,
	lastRunMap map[string]*pipelineRunRef,
	failedNamespaces *[]string,
) {
	logger := workflow.GetLogger(ctx)
	var result workflowengine.ActivityResult
	err := future.Get(ctx, &result)
	if err != nil {
		logger.Error("Failed to fetch scoreboard", "namespace", namespace, "error", err)
		*failedNamespaces = append(*failedNamespaces, namespace)
		return
	}

	respBody, ok := result.Output.(map[string]any)["body"]
	if !ok {
		logger.Error("Invalid response body", "namespace", namespace)
		*failedNamespaces = append(*failedNamespaces, namespace)
		return
	}

	pipelines, ok := respBody.([]any)
	if !ok {
		logger.Error("Response body is not an array", "namespace", namespace)
		*failedNamespaces = append(*failedNamespaces, namespace)
		return
	}

	for _, item := range pipelines {
		pipeline, ok := item.(map[string]any)
		if !ok {
			continue
		}
		w.aggregateSinglePipeline(pipeline, namespace, aggregatedMap, lastRunMap)
	}
}

func (w *AggregateScoreboardWorkflow) aggregateSinglePipeline(
	pipeline map[string]any,
	namespace string,
	aggregatedMap map[string]*AggregatedPipelineStats,
	lastRunMap map[string]*pipelineRunRef,
) {
	pipelineID, ok := pipeline["pipeline_id"].(string)
	if !ok || pipelineID == "" {
		return
	}

	// Get or create stats
	stats, exists := aggregatedMap[pipelineID]
	if !exists {
		stats = &AggregatedPipelineStats{
			PipelineID:   pipelineID,
			PipelineName: getString(pipeline, "pipeline_name"),
			RunnerTypes:  []string{},
			Runners:      []string{},
		}
		aggregatedMap[pipelineID] = stats
	}

	// Aggregate numeric stats
	w.aggregateNumericStats(stats, pipeline)

	// Aggregate runners and types
	w.aggregateRunners(stats, pipeline)
	w.aggregateRunnerTypes(stats, pipeline)

	// Update dates
	w.updateDates(stats, pipeline)

	// Track last successful run
	w.trackLastRun(pipeline, namespace, pipelineID, lastRunMap)
}

func (w *AggregateScoreboardWorkflow) aggregateNumericStats(
	stats *AggregatedPipelineStats,
	pipeline map[string]any,
) {
	if totalRuns, ok := pipeline["total_runs"].(float64); ok {
		stats.TotalRuns += int(totalRuns)
	}
	if totalSuccesses, ok := pipeline["total_successes"].(float64); ok {
		stats.TotalSuccesses += int(totalSuccesses)
	}
	if manual, ok := pipeline["manual_executions"].(float64); ok {
		stats.ManualExecutions += int(manual)
	}
	if scheduled, ok := pipeline["scheduled_executions"].(float64); ok {
		stats.ScheduledExecutions += int(scheduled)
	}
}

func (w *AggregateScoreboardWorkflow) aggregateRunners(
	stats *AggregatedPipelineStats,
	pipeline map[string]any,
) {
	if runners, ok := pipeline["runners"].([]any); ok {
		for _, runner := range runners {
			if runnerID, ok := runner.(string); ok {
				stats.Runners = appendUnique(stats.Runners, runnerID)
			}
		}
	}
}

func (w *AggregateScoreboardWorkflow) aggregateRunnerTypes(
	stats *AggregatedPipelineStats,
	pipeline map[string]any,
) {
	if runnerTypes, ok := pipeline["runner_types"].([]any); ok {
		for _, runnerType := range runnerTypes {
			if runnerTypeValue, ok := runnerType.(string); ok {
				stats.RunnerTypes = appendUnique(stats.RunnerTypes, runnerTypeValue)
			}
		}
	}
}

func (w *AggregateScoreboardWorkflow) updateDates(
	stats *AggregatedPipelineStats,
	pipeline map[string]any,
) {
	if firstDate, ok := pipeline["first_execution_date"].(string); ok && firstDate != "" {
		if stats.FirstExecutionDate == "" ||
			utils.TimeStringBefore(firstDate, stats.FirstExecutionDate) {
			stats.FirstExecutionDate = firstDate
		}
	}
	if lastDate, ok := pipeline["last_execution_date"].(string); ok && lastDate != "" {
		if stats.LastExecutionDate == "" ||
			utils.TimeStringAfter(lastDate, stats.LastExecutionDate) {
			stats.LastExecutionDate = lastDate
		}
	}
	if minTime, ok := pipeline["min_execution_time"].(string); ok && minTime != "" {
		if shouldReplaceMinExecutionTime(stats.MinExecutionTime, minTime) {
			stats.MinExecutionTime = minTime
		}
	}
}

func (w *AggregateScoreboardWorkflow) trackLastRun(
	pipeline map[string]any,
	namespace string,
	pipelineID string,
	lastRunMap map[string]*pipelineRunRef,
) {
	lastRunRaw, ok := pipeline["last_successful_run"]
	if !ok || lastRunRaw == nil {
		return
	}

	lastRunData, ok := lastRunRaw.(map[string]any)
	if !ok {
		return
	}

	startTime, _ := lastRunData["start_time"].(string)
	workflowID, _ := lastRunData["workflow_id"].(string)
	runID, _ := lastRunData["run_id"].(string)

	if startTime == "" || workflowID == "" || runID == "" {
		return
	}

	existingRun := lastRunMap[pipelineID]
	if existingRun == nil || utils.TimeStringAfter(startTime, existingRun.StartTime) {
		lastRunMap[pipelineID] = &pipelineRunRef{
			Namespace:  namespace,
			WorkflowID: workflowID,
			RunID:      runID,
			StartTime:  startTime,
		}
	}
}

// fetchLastExecutionDetails fetches details for the last successful runs
func (w *AggregateScoreboardWorkflow) fetchLastExecutionDetails(
	ctx workflow.Context,
	httpActivity *activities.InternalHTTPActivity,
	appURL string,
	lastRunMap map[string]*pipelineRunRef,
	aggregatedMap map[string]*AggregatedPipelineStats,
) {
	logger := workflow.GetLogger(ctx)
	for pipelineID, lastRun := range lastRunMap {
		if lastRun == nil {
			continue
		}
		stats := aggregatedMap[pipelineID]
		if stats == nil {
			continue
		}

		details, err := fetchExecutionDetails(ctx, httpActivity, appURL, lastRun)
		if err != nil {
			logger.Error(
				"Failed to fetch execution details",
				"pipeline_id", pipelineID,
				"workflow_id", lastRun.WorkflowID,
				"run_id", lastRun.RunID,
				"error", err,
			)
			continue
		}
		stats.LastExecution = details
	}
}

func fetchExecutionDetails(
	ctx workflow.Context,
	httpActivity *activities.InternalHTTPActivity,
	appURL string,
	run *pipelineRunRef,
) (*LatestExecutionDetails, error) {
	detailsRequest := workflowengine.ActivityInput{
		Payload: activities.InternalHTTPActivityPayload{
			Method: http.MethodGet,
			URL: utils.JoinURL(
				appURL,
				"api",
				"pipeline",
				"execution-details",
				run.Namespace,
				run.WorkflowID,
				run.RunID,
			),
			ExpectedStatus: http.StatusOK,
		},
	}

	var detailsResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(ctx, httpActivity.Name(), detailsRequest).Get(ctx, &detailsResult); err != nil {
		return nil, err
	}

	detailsBody, ok := detailsResult.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		return nil, workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.UnexpectedActivityOutput],
			"execution details body is not a map",
			detailsResult.Output,
		)
	}

	return &LatestExecutionDetails{
		PipelineName:         getString(detailsBody, "pipeline_name"),
		WorkflowID:           getString(detailsBody, "workflow_id"),
		RunID:                getString(detailsBody, "run_id"),
		OrgLogo:              getString(detailsBody, "org_logo"),
		Video:                getString(detailsBody, "video"),
		Screenshot:           getString(detailsBody, "screenshots"),
		Logs:                 getString(detailsBody, "logs"),
		WalletUsed:           getStringSlice(detailsBody, "wallet_used"),
		WalletVersionUsed:    getStringSlice(detailsBody, "wallet_version_used"),
		MaestroScripts:       getStringSlice(detailsBody, "maestro_scripts"),
		Credentials:          getStringSlice(detailsBody, "credentials"),
		Issuers:              getStringSlice(detailsBody, "issuers"),
		UseCaseVerifications: getStringSlice(detailsBody, "use_case_verifications"),
		Verifiers:            getStringSlice(detailsBody, "verifiers"),
		ConformanceTests:     getStringSlice(detailsBody, "conformance_tests"),
		CustomChecks:         getStringSlice(detailsBody, "custom_checks"),
	}, nil
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}

func getStringSlice(m map[string]any, key string) []string {
	if m == nil {
		return nil
	}
	if values, ok := m[key].([]any); ok {
		result := make([]string, 0, len(values))
		for _, item := range values {
			if value, ok := item.(string); ok {
				result = append(result, value)
			}
		}
		return result
	}
	if values, ok := m[key].([]string); ok {
		return append([]string(nil), values...)
	}
	return nil
}

func appendUnique(values []string, item string) []string {
	for _, existing := range values {
		if existing == item {
			return values
		}
	}
	return append(values, item)
}

func getRequiredStringSlice(m map[string]any, key string) ([]string, bool) {
	if m == nil {
		return nil, false
	}

	raw, ok := m[key]
	if !ok {
		return nil, false
	}

	switch values := raw.(type) {
	case []string:
		return append([]string(nil), values...), true
	case []any:
		result := make([]string, 0, len(values))
		for _, value := range values {
			item, ok := value.(string)
			if !ok {
				return nil, false
			}
			result = append(result, item)
		}
		return result, true
	default:
		return nil, false
	}
}

func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func shouldReplaceMinExecutionTime(current string, candidate string) bool {
	if current == "" {
		return true
	}

	currentDuration, currentErr := time.ParseDuration(current)
	candidateDuration, candidateErr := time.ParseDuration(candidate)

	switch {
	case currentErr == nil && candidateErr == nil:
		return candidateDuration < currentDuration
	case currentErr != nil && candidateErr == nil:
		return true
	case currentErr == nil && candidateErr != nil:
		return false
	default:
		return candidate < current
	}
}
