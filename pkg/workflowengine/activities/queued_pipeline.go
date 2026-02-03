// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"gopkg.in/yaml.v3"
)

type StartQueuedPipelineActivity struct {
	workflowengine.BaseActivity
}

type StartQueuedPipelineActivityInput struct {
	TicketID           string         `json:"ticket_id"`
	OwnerNamespace     string         `json:"owner_namespace"`
	PipelineIdentifier string         `json:"pipeline_identifier"`
	YAML               string         `json:"yaml"`
	PipelineConfig     map[string]any `json:"pipeline_config,omitempty"`
	Memo               map[string]any `json:"memo,omitempty"`
}

type StartQueuedPipelineActivityOutput struct {
	WorkflowID        string `json:"workflow_id"`
	RunID             string `json:"run_id"`
	WorkflowNamespace string `json:"workflow_namespace"`
}

const (
	pipelineTaskQueue              = "PipelineTaskQueue"
	pipelineWorkflowName           = "Dynamic Pipeline Workflow"
	defaultExecutionTimeout        = "24h"
	defaultActivityScheduleTimeout = "10m"
	defaultActivityStartTimeout    = "5m"
	defaultRetryMaxAttempts        = int32(5)
	defaultRetryInitialInterval    = "5s"
	defaultRetryMaxInterval        = "1m"
	defaultRetryBackoffCoefficient = 2.0
)

type queuedWorkflowDefinition struct {
	Name    string         `yaml:"name"`
	Runtime queuedRuntime  `yaml:"runtime,omitempty"`
	Config  map[string]any `yaml:"config,omitempty"`
}

type queuedRuntime struct {
	Debug          bool   `yaml:"debug,omitempty"`
	GlobalRunnerID string `yaml:"global_runner_id,omitempty"`
	Temporal       struct {
		ExecutionTimeout string                `yaml:"execution_timeout,omitempty"`
		ActivityOptions  queuedActivityOptions `yaml:"activity_options,omitempty"`
	} `yaml:"temporal,omitempty"`
}

type queuedActivityOptions struct {
	ScheduleToCloseTimeout string            `yaml:"schedule_to_close_timeout,omitempty"`
	StartToCloseTimeout    string            `yaml:"start_to_close_timeout,omitempty"`
	RetryPolicy            queuedRetryPolicy `yaml:"retry_policy,omitempty"`
}

type queuedRetryPolicy struct {
	MaximumAttempts    int32   `yaml:"maximum_attempts,omitempty"`
	InitialInterval    string  `yaml:"initial_interval,omitempty"`
	MaximumInterval    string  `yaml:"maximum_interval,omitempty"`
	BackoffCoefficient float64 `yaml:"backoff_coefficient,omitempty"`
}

type queuedWorkflowOptions struct {
	Options         client.StartWorkflowOptions
	ActivityOptions workflow.ActivityOptions
}

func NewStartQueuedPipelineActivity() *StartQueuedPipelineActivity {
	return &StartQueuedPipelineActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Start queued pipeline",
		},
	}
}

func (a *StartQueuedPipelineActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *StartQueuedPipelineActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[StartQueuedPipelineActivityInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	if strings.TrimSpace(payload.OwnerNamespace) == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, "owner_namespace is required")
	}
	if strings.TrimSpace(payload.PipelineIdentifier) == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, "pipeline_identifier is required")
	}
	if strings.TrimSpace(payload.YAML) == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, "yaml is required")
	}

	config := payload.PipelineConfig
	if config == nil {
		config = map[string]any{}
	}
	if namespace, ok := config["namespace"].(string); !ok || namespace == "" {
		config["namespace"] = payload.OwnerNamespace
	}

	appURL, ok := config["app_url"].(string)
	if !ok || strings.TrimSpace(appURL) == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, "app_url is required in pipeline_config")
	}

	memo := payload.Memo
	if memo == nil {
		memo = map[string]any{}
	}

	workflowDef, workflowDefMap, err := parseQueuedWorkflowDefinition(payload.YAML)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.PipelineParsingError]
		return result, a.NewActivityError(errCode.Code, err.Error())
	}

	for key, value := range workflowDef.Config {
		if _, exists := config[key]; !exists {
			config[key] = value
		}
	}

	if workflowDef.Runtime.GlobalRunnerID != "" {
		config["global_runner_id"] = workflowDef.Runtime.GlobalRunnerID
	}

	memo["test"] = workflowDef.Name
	options := prepareQueuedWorkflowOptions(workflowDef.Runtime)
	options.Options.ID = fmt.Sprintf(
		"Pipeline-%s-%s",
		canonify.CanonifyPlain(workflowDef.Name),
		uuid.NewString(),
	)
	options.Options.TaskQueue = pipelineTaskQueue
	options.Options.Memo = memo

	namespace := config["namespace"].(string)
	temporalClient, err := temporalclient.GetTemporalClientWithNamespace(namespace)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		return result, a.NewActivityError(errCode.Code, err.Error())
	}

	workflowInput := map[string]any{
		"workflow_definition": workflowDefMap,
		"workflow_input": workflowengine.WorkflowInput{
			Config:          config,
			ActivityOptions: &options.ActivityOptions,
		},
		"debug": workflowDef.Runtime.Debug,
	}

	workflowRun, err := temporalClient.ExecuteWorkflow(
		ctx,
		options.Options,
		pipelineWorkflowName,
		workflowInput,
	)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		return result, a.NewActivityError(errCode.Code, err.Error())
	}

	workflowID := workflowRun.GetID()
	runID := workflowRun.GetRunID()

	if err := createPipelineExecutionResult(
		ctx,
		appURL,
		payload.OwnerNamespace,
		payload.PipelineIdentifier,
		workflowID,
		runID,
	); err != nil {
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		return result, a.NewActivityError(errCode.Code, err.Error())
	}

	result.Output = StartQueuedPipelineActivityOutput{
		WorkflowID:        workflowID,
		RunID:             runID,
		WorkflowNamespace: namespace,
	}
	return result, nil
}

func parseQueuedWorkflowDefinition(
	yamlInput string,
) (queuedWorkflowDefinition, map[string]any, error) {
	var definition queuedWorkflowDefinition
	if err := yaml.Unmarshal([]byte(yamlInput), &definition); err != nil {
		return queuedWorkflowDefinition{}, nil, fmt.Errorf("parse workflow definition: %w", err)
	}
	workflowMap := map[string]any{}
	if err := yaml.Unmarshal([]byte(yamlInput), &workflowMap); err != nil {
		return queuedWorkflowDefinition{}, nil, fmt.Errorf("parse workflow definition map: %w", err)
	}
	return definition, workflowMap, nil
}

func prepareQueuedWorkflowOptions(rc queuedRuntime) queuedWorkflowOptions {
	rp := &temporal.RetryPolicy{
		MaximumAttempts: defaultRetryMaxAttempts,
		InitialInterval: parseDurationOrDefault(
			rc.Temporal.ActivityOptions.RetryPolicy.InitialInterval,
			defaultRetryInitialInterval,
		),
		MaximumInterval: parseDurationOrDefault(
			rc.Temporal.ActivityOptions.RetryPolicy.MaximumInterval,
			defaultRetryMaxInterval,
		),
		BackoffCoefficient: defaultRetryBackoffCoefficient,
	}

	if rc.Temporal.ActivityOptions.RetryPolicy.MaximumAttempts > 0 {
		rp.MaximumAttempts = rc.Temporal.ActivityOptions.RetryPolicy.MaximumAttempts
	}
	if rc.Temporal.ActivityOptions.RetryPolicy.BackoffCoefficient > 0 {
		rp.BackoffCoefficient = rc.Temporal.ActivityOptions.RetryPolicy.BackoffCoefficient
	}

	ao := workflow.ActivityOptions{
		ScheduleToCloseTimeout: parseDurationOrDefault(
			rc.Temporal.ActivityOptions.ScheduleToCloseTimeout,
			defaultActivityScheduleTimeout,
		),
		StartToCloseTimeout: parseDurationOrDefault(
			rc.Temporal.ActivityOptions.StartToCloseTimeout,
			defaultActivityStartTimeout,
		),
		RetryPolicy: rp,
	}

	return queuedWorkflowOptions{
		Options: client.StartWorkflowOptions{
			WorkflowExecutionTimeout: parseDurationOrDefault(
				rc.Temporal.ExecutionTimeout,
				defaultExecutionTimeout,
			),
		},
		ActivityOptions: ao,
	}
}

func parseDurationOrDefault(value, fallback string) time.Duration {
	if value == "" {
		value = fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return time.Minute * 5
	}
	return parsed
}

func createPipelineExecutionResult(
	ctx context.Context,
	appURL string,
	ownerNamespace string,
	pipelineID string,
	workflowID string,
	runID string,
) error {
	payload := map[string]any{
		"owner":       ownerNamespace,
		"pipeline_id": pipelineID,
		"workflow_id": workflowID,
		"run_id":      runID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal pipeline result payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		utils.JoinURL(appURL, "api", "pipeline", "pipeline-execution-results"),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build pipeline results request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post pipeline results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pipeline results status: %s", resp.Status)
	}
	return nil
}
