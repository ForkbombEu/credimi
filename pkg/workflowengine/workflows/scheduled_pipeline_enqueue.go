// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"gopkg.in/yaml.v3"
)

// ScheduledPipelineEnqueueWorkflowName identifies the scheduled enqueue workflow.
const ScheduledPipelineEnqueueWorkflowName = "Scheduled Pipeline Enqueue Workflow"

// ScheduledPipelineEnqueueWorkflow enqueues scheduled pipeline runs into the runner queue.
type ScheduledPipelineEnqueueWorkflow struct{}

// ScheduledPipelineEnqueueWorkflowInput defines the schedule enqueue payload and config.
type ScheduledPipelineEnqueueWorkflowInput struct {
	PipelineIdentifier  string         `json:"pipeline_identifier"`
	OwnerNamespace      string         `json:"owner_namespace"`
	PipelineConfig      map[string]any `json:"pipeline_config,omitempty"`
	GlobalRunnerID      string         `json:"global_runner_id,omitempty"`
	MaxPipelinesInQueue int            `json:"max_pipelines_in_queue,omitempty"`
}

// NewScheduledPipelineEnqueueWorkflow constructs a scheduled enqueue workflow.
func NewScheduledPipelineEnqueueWorkflow() *ScheduledPipelineEnqueueWorkflow {
	return &ScheduledPipelineEnqueueWorkflow{}
}

// Name returns the workflow name for scheduled enqueues.
func (ScheduledPipelineEnqueueWorkflow) Name() string {
	return ScheduledPipelineEnqueueWorkflowName
}

// GetOptions returns the default activity options for the scheduled enqueue workflow.
func (ScheduledPipelineEnqueueWorkflow) GetOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{}
}

// Workflow validates schedule input, fetches YAML, and enqueues a run ticket.
func (w *ScheduledPipelineEnqueueWorkflow) Workflow(
	ctx workflow.Context,
	input ScheduledPipelineEnqueueWorkflowInput,
) (workflowengine.WorkflowResult, error) {
	info := workflow.GetInfo(ctx)
	pipelineIdentifier := strings.TrimSpace(input.PipelineIdentifier)
	ownerNamespace := strings.TrimSpace(input.OwnerNamespace)

	config := input.PipelineConfig
	if config == nil {
		config = map[string]any{}
	}
	if ownerNamespace != "" {
		config["namespace"] = ownerNamespace
	}
	appURL, _ := config["app_url"].(string)

	runMetadata := &workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   info.WorkflowExecution.ID,
		Namespace:    info.Namespace,
		TemporalUI: utils.JoinURL(
			appURL,
			"my", "tests", "runs",
			info.WorkflowExecution.ID,
			info.WorkflowExecution.RunID,
		),
	}

	if pipelineIdentifier == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			fmt.Errorf("pipeline_identifier is required"),
			runMetadata,
		)
	}
	if ownerNamespace == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			fmt.Errorf("owner_namespace is required"),
			runMetadata,
		)
	}
	if strings.TrimSpace(appURL) == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			runMetadata,
		)
	}

	httpActivity := activities.NewHTTPActivity()
	httpCtx := workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{
			ScheduleToCloseTimeout: time.Minute,
			StartToCloseTimeout:    30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    time.Second,
				BackoffCoefficient: 1.0,
				MaximumInterval:    time.Minute,
				MaximumAttempts:    1,
			},
		},
	)

	request := workflowengine.ActivityInput{
		Payload: activities.HTTPActivityPayload{
			Method: http.MethodPost,
			URL: utils.JoinURL(
				appURL,
				"api", "canonify", "identifier", "validate",
			),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: map[string]any{
				"canonified_name": pipelineIdentifier,
			},
			ExpectedStatus: 200,
		},
	}

	var httpResult workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(httpCtx, httpActivity.Name(), request).
		Get(httpCtx, &httpResult); err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}

	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
	output, ok := httpResult.Output.(map[string]any)
	if !ok {
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: invalid output format", errCode.Description),
			httpResult.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	body, ok := output["body"].(map[string]any)
	if !ok {
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: missing body in output", errCode.Description),
			output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	record, ok := body["record"].(map[string]any)
	if !ok {
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: missing record in body", errCode.Description),
			body,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	pipelineYAML, ok := record["yaml"].(string)
	if !ok {
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("%s: missing yaml in record", errCode.Description),
			record,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	parsedPipeline, runnerInfo, err := parseScheduledPipelineDefinition(pipelineYAML)
	if err != nil {
		parseCode := errorcodes.Codes[errorcodes.PipelineParsingError]
		appErr := workflowengine.NewAppError(parseCode, err.Error())
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	globalRunnerID := ""
	if runnerInfo.NeedsGlobalRunner {
		globalRunnerID = strings.TrimSpace(parsedPipeline.Runtime.GlobalRunnerID)
		if globalRunnerID == "" {
			globalRunnerID = strings.TrimSpace(input.GlobalRunnerID)
		}
		if globalRunnerID == "" {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
				"global_runner_id",
				runMetadata,
			)
		}
	}
	if globalRunnerID != "" {
		config["global_runner_id"] = globalRunnerID
	}

	runnerIDs := runnerIDsWithGlobal(runnerInfo, globalRunnerID)
	sort.Strings(runnerIDs)
	if len(runnerIDs) == 0 {
		configErr := workflowengine.NewAppError(
			errorcodes.Codes[errorcodes.MissingOrInvalidConfig],
			"runner_ids",
			"no runner ids resolved from yaml",
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(configErr, runMetadata)
	}

	enqueuedAt := workflow.Now(ctx).UTC()
	ticketID := fmt.Sprintf(
		"sched/%s/%s",
		info.WorkflowExecution.ID,
		info.WorkflowExecution.RunID,
	)
	memo := map[string]any{
		"test": "pipeline-run",
	}

	enqueueInput := workflowengine.ActivityInput{
		Payload: activities.EnqueuePipelineRunTicketActivityInput{
			TicketID:            ticketID,
			OwnerNamespace:      ownerNamespace,
			EnqueuedAt:          enqueuedAt,
			RunnerIDs:           runnerIDs,
			PipelineIdentifier:  pipelineIdentifier,
			YAML:                pipelineYAML,
			PipelineConfig:      config,
			Memo:                memo,
			MaxPipelinesInQueue: input.MaxPipelinesInQueue,
		},
	}

	var enqueueResult workflowengine.ActivityResult
	enqueueCtx := workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{
			ScheduleToCloseTimeout: time.Minute,
			StartToCloseTimeout:    30 * time.Second,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:    time.Second,
				BackoffCoefficient: 1.0,
				MaximumInterval:    time.Minute,
				MaximumAttempts:    1,
			},
		},
	)
	if err := workflow.ExecuteActivity(
		enqueueCtx,
		activities.EnqueuePipelineRunTicketActivityName,
		enqueueInput,
	).Get(enqueueCtx, &enqueueResult); err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}

	return workflowengine.WorkflowResult{
		WorkflowID:    info.WorkflowExecution.ID,
		WorkflowRunID: info.WorkflowExecution.RunID,
		Message:       "scheduled pipeline enqueued",
		Output:        enqueueResult.Output,
	}, nil
}

// scheduledPipelineDefinition captures the YAML fields needed for runner resolution.
type scheduledPipelineDefinition struct {
	Runtime scheduledPipelineRuntime `yaml:"runtime,omitempty"`
	Steps   []scheduledPipelineStep  `yaml:"steps,omitempty"`
}

// scheduledPipelineRuntime stores global runner configuration from YAML.
type scheduledPipelineRuntime struct {
	GlobalRunnerID string `yaml:"global_runner_id,omitempty"`
}

// scheduledPipelineStep is a minimal step definition for runner lookup.
type scheduledPipelineStep struct {
	Use       string                 `yaml:"use,omitempty"`
	With      map[string]any         `yaml:"with,omitempty"`
	OnError   []scheduledPipelineStep `yaml:"on_error,omitempty"`
	OnSuccess []scheduledPipelineStep `yaml:"on_success,omitempty"`
}

// scheduledPipelineRunnerInfo describes runner IDs resolved from a pipeline YAML.
type scheduledPipelineRunnerInfo struct {
	RunnerIDs         []string
	NeedsGlobalRunner bool
}

// parseScheduledPipelineDefinition reads YAML and returns runner metadata for schedules.
func parseScheduledPipelineDefinition(
	yamlStr string,
) (scheduledPipelineDefinition, scheduledPipelineRunnerInfo, error) {
	def := scheduledPipelineDefinition{}
	if strings.TrimSpace(yamlStr) == "" {
		return def, scheduledPipelineRunnerInfo{}, nil
	}

	if err := yaml.Unmarshal([]byte(yamlStr), &def); err != nil {
		return def, scheduledPipelineRunnerInfo{}, err
	}

	runnerIDs := map[string]struct{}{}
	needsGlobal := false
	collectRunnerIDs(def.Steps, runnerIDs, &needsGlobal)

	info := scheduledPipelineRunnerInfo{
		NeedsGlobalRunner: needsGlobal,
	}
	if len(runnerIDs) == 0 {
		return def, info, nil
	}

	info.RunnerIDs = make([]string, 0, len(runnerIDs))
	for runnerID := range runnerIDs {
		info.RunnerIDs = append(info.RunnerIDs, runnerID)
	}
	sort.Strings(info.RunnerIDs)

	return def, info, nil
}

// collectRunnerIDs walks pipeline steps and collects runner IDs plus missing runner flags.
func collectRunnerIDs(
	steps []scheduledPipelineStep,
	runnerIDs map[string]struct{},
	needsGlobal *bool,
) {
	for _, step := range steps {
		runnerID := ""
		if step.With != nil {
			if rawRunnerID, ok := step.With["runner_id"]; ok {
				if id, ok := rawRunnerID.(string); ok {
					runnerID = strings.TrimSpace(id)
				}
			}
		}

		if runnerID != "" {
			runnerIDs[runnerID] = struct{}{}
		} else if step.Use == "mobile-automation" && needsGlobal != nil {
			*needsGlobal = true
		}

		if len(step.OnError) > 0 {
			collectRunnerIDs(step.OnError, runnerIDs, needsGlobal)
		}
		if len(step.OnSuccess) > 0 {
			collectRunnerIDs(step.OnSuccess, runnerIDs, needsGlobal)
		}
	}
}

// runnerIDsWithGlobal combines explicit runner IDs with a global runner override when needed.
func runnerIDsWithGlobal(info scheduledPipelineRunnerInfo, globalRunnerID string) []string {
	runnerIDs := append([]string{}, info.RunnerIDs...)
	globalRunnerID = strings.TrimSpace(globalRunnerID)
	if info.NeedsGlobalRunner && globalRunnerID != "" {
		found := false
		for _, id := range runnerIDs {
			if id == globalRunnerID {
				found = true
				break
			}
		}
		if !found {
			runnerIDs = append(runnerIDs, globalRunnerID)
			sort.Strings(runnerIDs)
		}
	}
	return runnerIDs
}
