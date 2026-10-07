// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const AggregateScoreboardTaskQueue = "AggregateScoreboardTaskQueue"

// Activity names of the scoreboard activities. The activities live in the
// handlers package because they reuse its Temporal listing code.
const (
	ListScoreboardNamespacesActivityName      = "List organization namespaces for the scoreboard"
	GetNamespaceScoreboardActivityName        = "Get the pipeline scoreboard of a namespace"
	GetScoreboardExecutionDetailsActivityName = "Get scoreboard execution details"
	SaveScoreboardResultsActivityName         = "Save aggregated scoreboard results"
)

// ScoreboardNamespaceInput is the payload of GetNamespaceScoreboardActivityName.
type ScoreboardNamespaceInput struct {
	Namespace string `json:"namespace" validate:"required"`
}

// ScoreboardExecutionInput is the payload of
// GetScoreboardExecutionDetailsActivityName.
type ScoreboardExecutionInput struct {
	Namespace  string `json:"namespace"   validate:"required"`
	WorkflowID string `json:"workflow_id" validate:"required"`
	RunID      string `json:"run_id"      validate:"required"`
}

var aggregateScoreboardStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

type AggregatedPipelineStats struct {
	PipelineID              string                  `json:"pipeline_id"`
	PipelineName            string                  `json:"pipeline_name"`
	DeviceTypes             []string                `json:"device_types"`
	DeviceIDs               []string                `json:"device_ids"`
	TotalRuns               int                     `json:"total_runs"`
	TotalSuccesses          int                     `json:"total_successes"`
	SuccessRate             float64                 `json:"success_rate"`
	ManualExecutions        int                     `json:"manual_executions"`
	ScheduledExecutions     int                     `json:"scheduled_executions"`
	CIExecutions            int                     `json:"ci_executions"`
	MinExecutionTime        string                  `json:"min_execution_time"`
	MinExecutionTimeSeconds int                     `json:"min_execution_time_seconds"`
	FirstExecutionDate      string                  `json:"first_execution_date"`
	LastExecutionDate       string                  `json:"last_execution_date"`
	LastExecution           *LatestExecutionDetails `json:"last_execution,omitempty"`
}

type LatestExecutionDetails struct {
	WorkflowID           string   `json:"workflow_id,omitempty"`
	RunID                string   `json:"run_id,omitempty"`
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

// namespacePipelineStats mirrors the JSON fields the workflow reads from each
// entry of the GetNamespaceScoreboardActivityName output
// (handlers.PipelineStatsResponse, which this package cannot import).
type namespacePipelineStats struct {
	PipelineID              string                    `json:"pipeline_id"`
	PipelineName            string                    `json:"pipeline_name"`
	DeviceTypes             []string                  `json:"device_types"`
	DeviceIDs               []string                  `json:"device_ids"`
	TotalRuns               int                       `json:"total_runs"`
	TotalSuccesses          int                       `json:"total_successes"`
	ManualExecutions        int                       `json:"manual_executions"`
	ScheduledExecutions     int                       `json:"scheduled_executions"`
	CIExecutions            int                       `json:"ci_executions"`
	MinExecutionTime        string                    `json:"min_execution_time"`
	MinExecutionTimeSeconds *int                      `json:"min_execution_time_seconds"`
	FirstExecutionDate      string                    `json:"first_execution_date"`
	LastExecutionDate       string                    `json:"last_execution_date"`
	LastRun                 *namespacePipelineLastRun `json:"last_run"`
}

type namespacePipelineLastRun struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	StartTime  string `json:"start_time"`
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

	// 1. Get namespaces
	namespaces, err := w.getNamespaces(ctx)
	if err != nil {
		logger.Error("Failed to get namespaces", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
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
	aggregatedMap, lastRunMap, failedNamespaces := w.fetchAndAggregateScoreboards(ctx, namespaces)

	// 3. Calculate success rates and sort runners
	for _, stats := range aggregatedMap {
		if stats.TotalRuns > 0 {
			stats.SuccessRate = math.Round(
				float64(stats.TotalSuccesses)/float64(stats.TotalRuns)*10000,
			) / 100
		}
		sort.Strings(stats.DeviceIDs)
		sort.Strings(stats.DeviceTypes)
	}

	// 4. Fetch last execution details
	w.fetchLastExecutionDetails(ctx, lastRunMap, aggregatedMap)

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
	var saveResult workflowengine.ActivityResult
	if err = workflow.ExecuteActivity(ctx, SaveScoreboardResultsActivityName, workflowengine.ActivityInput{
		Payload: AggregateScoreboardWorkflowOutput{AggregatedPipelines: output.AggregatedPipelines},
	}).
		Get(ctx, &saveResult); err != nil {
		logger.Error("Failed to save results", "error", err)
	}

	return workflowengine.WorkflowResult{
		Message: "Successfully aggregated scoreboard across namespaces",
		Output:  output,
	}, nil
}

// getNamespaces retrieves all organization namespaces.
func (w *AggregateScoreboardWorkflow) getNamespaces(ctx workflow.Context) ([]string, error) {
	var result workflowengine.ActivityResult
	err := workflow.ExecuteActivity(
		ctx,
		ListScoreboardNamespacesActivityName,
		workflowengine.ActivityInput{},
	).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	if result.Output == nil {
		return nil, errors.New("namespaces output missing")
	}
	namespaces, err := workflowengine.DecodeOutput[[]string](result.Output)
	if err != nil {
		return nil, fmt.Errorf("namespaces output invalid: %w", err)
	}

	return uniqueStrings(namespaces), nil
}

func (w *AggregateScoreboardWorkflow) fetchAndAggregateScoreboards(
	ctx workflow.Context,
	namespaces []string,
) (map[string]*AggregatedPipelineStats, map[string]*pipelineRunRef, []string) {
	aggregatedMap := make(map[string]*AggregatedPipelineStats)
	lastRunMap := make(map[string]*pipelineRunRef)
	var failedNamespaces []string

	// Start all parallel activities with preallocated slices
	scoreboardFutures := make([]workflow.Future, 0, len(namespaces))
	scoreboardNamespaces := make([]string, 0, len(namespaces))

	for _, namespace := range namespaces {
		scoreboardFutures = append(
			scoreboardFutures,
			workflow.ExecuteActivity(
				ctx,
				GetNamespaceScoreboardActivityName,
				workflowengine.ActivityInput{
					Payload: ScoreboardNamespaceInput{Namespace: namespace},
				},
			),
		)
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

	pipelines, err := workflowengine.DecodeOutput[[]namespacePipelineStats](result.Output)
	if err != nil {
		logger.Error("Invalid scoreboard output", "namespace", namespace, "error", err)
		*failedNamespaces = append(*failedNamespaces, namespace)
		return
	}

	for i := range pipelines {
		w.aggregateSinglePipeline(&pipelines[i], namespace, aggregatedMap, lastRunMap)
	}
}

func (w *AggregateScoreboardWorkflow) aggregateSinglePipeline(
	pipeline *namespacePipelineStats,
	namespace string,
	aggregatedMap map[string]*AggregatedPipelineStats,
	lastRunMap map[string]*pipelineRunRef,
) {
	pipelineID := pipeline.PipelineID
	if pipelineID == "" {
		return
	}

	// Get or create stats
	stats, exists := aggregatedMap[pipelineID]
	if !exists {
		stats = &AggregatedPipelineStats{
			PipelineID:   pipelineID,
			PipelineName: pipeline.PipelineName,
			DeviceTypes:  []string{},
			DeviceIDs:    []string{},
		}
		aggregatedMap[pipelineID] = stats
	}

	// Aggregate numeric stats
	w.aggregateNumericStats(stats, pipeline)

	// Aggregate execution devices and types.
	w.aggregateDeviceIDs(stats, pipeline)
	w.aggregateDeviceTypes(stats, pipeline)

	// Update dates
	w.updateDates(stats, pipeline)

	// Track last run (failed runs included: the scoreboard shows why pipelines are broken)
	w.trackLastRun(pipeline, namespace, pipelineID, lastRunMap)
}

func (w *AggregateScoreboardWorkflow) aggregateNumericStats(
	stats *AggregatedPipelineStats,
	pipeline *namespacePipelineStats,
) {
	stats.TotalRuns += pipeline.TotalRuns
	stats.TotalSuccesses += pipeline.TotalSuccesses
	stats.ManualExecutions += pipeline.ManualExecutions
	stats.ScheduledExecutions += pipeline.ScheduledExecutions
	stats.CIExecutions += pipeline.CIExecutions
}

func (w *AggregateScoreboardWorkflow) aggregateDeviceIDs(
	stats *AggregatedPipelineStats,
	pipeline *namespacePipelineStats,
) {
	for _, deviceID := range pipeline.DeviceIDs {
		stats.DeviceIDs = appendUnique(stats.DeviceIDs, deviceID)
	}
}

func (w *AggregateScoreboardWorkflow) aggregateDeviceTypes(
	stats *AggregatedPipelineStats,
	pipeline *namespacePipelineStats,
) {
	for _, deviceType := range pipeline.DeviceTypes {
		stats.DeviceTypes = appendUnique(stats.DeviceTypes, deviceType)
	}
}

func (w *AggregateScoreboardWorkflow) updateDates(
	stats *AggregatedPipelineStats,
	pipeline *namespacePipelineStats,
) {
	if firstDate := pipeline.FirstExecutionDate; firstDate != "" {
		if stats.FirstExecutionDate == "" ||
			utils.TimeStringBefore(firstDate, stats.FirstExecutionDate) {
			stats.FirstExecutionDate = firstDate
		}
	}
	if lastDate := pipeline.LastExecutionDate; lastDate != "" {
		if stats.LastExecutionDate == "" ||
			utils.TimeStringAfter(lastDate, stats.LastExecutionDate) {
			stats.LastExecutionDate = lastDate
		}
	}
	if minTime := pipeline.MinExecutionTime; minTime != "" {
		if shouldReplaceMinExecutionTime(stats.MinExecutionTime, minTime) {
			stats.MinExecutionTime = minTime
			if pipeline.MinExecutionTimeSeconds != nil {
				stats.MinExecutionTimeSeconds = *pipeline.MinExecutionTimeSeconds
			} else if parsed, err := time.ParseDuration(minTime); err == nil {
				stats.MinExecutionTimeSeconds = int(math.Round(parsed.Seconds()))
			}
		}
	}
}

func (w *AggregateScoreboardWorkflow) trackLastRun(
	pipeline *namespacePipelineStats,
	namespace string,
	pipelineID string,
	lastRunMap map[string]*pipelineRunRef,
) {
	lastRun := pipeline.LastRun
	if lastRun == nil ||
		lastRun.StartTime == "" || lastRun.WorkflowID == "" || lastRun.RunID == "" {
		return
	}

	existingRun := lastRunMap[pipelineID]
	if existingRun == nil || utils.TimeStringAfter(lastRun.StartTime, existingRun.StartTime) {
		lastRunMap[pipelineID] = &pipelineRunRef{
			Namespace:  namespace,
			WorkflowID: lastRun.WorkflowID,
			RunID:      lastRun.RunID,
			StartTime:  lastRun.StartTime,
		}
	}
}

// fetchLastExecutionDetails fetches details for the last runs
func (w *AggregateScoreboardWorkflow) fetchLastExecutionDetails(
	ctx workflow.Context,
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

		details, err := fetchExecutionDetails(ctx, lastRun)
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
	run *pipelineRunRef,
) (*LatestExecutionDetails, error) {
	var detailsResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(
		ctx,
		GetScoreboardExecutionDetailsActivityName,
		workflowengine.ActivityInput{
			Payload: ScoreboardExecutionInput{
				Namespace:  run.Namespace,
				WorkflowID: run.WorkflowID,
				RunID:      run.RunID,
			},
		},
	).Get(ctx, &detailsResult); err != nil {
		return nil, err
	}

	if detailsResult.Output == nil {
		return nil, unexpectedExecutionDetailsError("execution details output is missing", nil)
	}
	details, err := workflowengine.DecodeOutput[LatestExecutionDetails](detailsResult.Output)
	if err != nil {
		return nil, unexpectedExecutionDetailsError(
			"decode execution details: "+err.Error(),
			detailsResult.Output,
		)
	}

	return &details, nil
}

func unexpectedExecutionDetailsError(message string, output any) error {
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
	return workflowengine.NewAppError(
		workflowengine.WorkflowError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: message,
			Details: map[string]any{"payload": output},
		},
	)
}

func appendUnique(values []string, item string) []string {
	for _, existing := range values {
		if existing == item {
			return values
		}
	}
	return append(values, item)
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
