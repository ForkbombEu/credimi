// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/joho/godotenv"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WorkflowInput represents the input data required to start a workflow.
type WorkflowInput struct {
	Payload map[string]any
	Config  map[string]any
}

// WorkflowResult represents the result of a workflow execution, including a message, errors, and a log.
type WorkflowResult struct {
	WorkflowID    string
	WorkflowRunID string
	Author        string
	Message       string
	Errors        []error
	Output        any
	Log           any
}

type WorkflowErrorMetadata struct {
	WorkflowName string
	WorkflowID   string
	Namespace    string
	TemporalUI   string
}
type WorkflowRunInfo struct {
	Name      string
	TaskQueue string
	Input     WorkflowInput
	Memo      map[string]any
}

// Workflow defines the interface for a workflow, including its execution, name, and options.
type Workflow interface {
	Workflow(ctx workflow.Context, input WorkflowInput) (WorkflowResult, error)
	Name() string
	GetOptions() workflow.ActivityOptions
}

func NewWorkflowError(err error, metadata WorkflowErrorMetadata, extraPayload ...any) error {
	var appErr *temporal.ApplicationError
	if !temporal.IsApplicationError(err) || !errors.As(err, &appErr) {
		return err
	}

	var originalDetails any
	if err := appErr.Details(&originalDetails); err != nil {
		originalDetails = nil
	}

	credimiErr := utils.CredimiError{
		Code:      appErr.Type(),
		Component: "workflow engine",
		Location:  metadata.WorkflowName,
		Message:   appErr.Message(),
		Context:   []string{fmt.Sprintf("Further information at: %s", metadata.TemporalUI)},
	}

	newErr := temporal.NewApplicationError(
		credimiErr.Error(),
		appErr.Type(),
		originalDetails,
		extraPayload,
		metadata,
	)

	return newErr
}

func NewAppError(code errorcodes.Code, field string, payload ...any) error {
	return temporal.NewApplicationError(
		fmt.Sprintf("%s: '%s'", code.Description, field),
		code.Code,
		payload...)
}

// newMissingPayloadError returns a WorkflowError for a missing or invalid payload key.
func NewMissingPayloadError(key string, metadata WorkflowErrorMetadata) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	appErr := NewAppError(errCode, key)
	return NewWorkflowError(appErr, metadata)
}

// newMissingConfigError returns a WorkflowError for a missing or invalid config key.
func NewMissingConfigError(key string, metadata WorkflowErrorMetadata) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
	appErr := NewAppError(errCode, key)
	return NewWorkflowError(appErr, metadata)
}

// newStepCIOutputError returns a WorkflowError for unexpected or invalid StepCI output.
func NewStepCIOutputError(field string, output any, metadata WorkflowErrorMetadata) error {
	errCode := errorcodes.Codes[errorcodes.UnexpectedStepCIOutput]
	appErr := NewAppError(errCode, field, output)
	return NewWorkflowError(appErr, metadata)
}

func StartWorkflowWithOptions(
	options client.StartWorkflowOptions,
	name string,
	input WorkflowInput,
) (result WorkflowResult, err error) {
	// Load environment variables.
	err = godotenv.Load()
	if err != nil {
		return WorkflowResult{}, fmt.Errorf("failed to load .env file: %w", err)
	}
	namespace := "default"
	if input.Config["namespace"] != nil {
		namespace = input.Config["namespace"].(string)
	}
	c, err := temporalclient.GetTemporalClientWithNamespace(
		namespace,
	)
	if err != nil {
		return WorkflowResult{}, fmt.Errorf("unable to create client: %w", err)
	}

	if input.Config["memo"] != nil {
		options.Memo = input.Config["memo"].(map[string]any)
	}

	// Start the workflow execution.
	w, err := c.ExecuteWorkflow(context.Background(), options, name, input)
	if err != nil {
		return WorkflowResult{}, fmt.Errorf("failed to start workflow: %w", err)
	}

	return WorkflowResult{
		WorkflowID:    w.GetID(),
		WorkflowRunID: w.GetRunID(),
		Message:       fmt.Sprintf("Workflow %s started successfully with ID %s", name, w.GetID()),
	}, nil
}

func GetWorkflowRunInfo(workflowID, runID, namespace string) (WorkflowRunInfo, error) {
	runInfo := WorkflowRunInfo{}

	c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
	if err != nil {
		return WorkflowRunInfo{}, fmt.Errorf("unable to create Temporal client for namespace %q: %w", namespace, err)
	}

	describeResp, err := c.DescribeWorkflowExecution(context.Background(), workflowID, runID)
	if err != nil {
		return WorkflowRunInfo{}, fmt.Errorf("unable to describe workflow execution (WorkflowID=%q, RunID=%q): %w", workflowID, runID, err)
	}

	decodedMemo := make(map[string]any)
	for k, payload := range describeResp.WorkflowExecutionInfo.Memo.GetFields() {
		var v any
		if err := converter.GetDefaultDataConverter().FromPayload(payload, &v); err != nil {
			return WorkflowRunInfo{}, fmt.Errorf("failed to decode memo key %q: %w", k, err)
		}
		decodedMemo[k] = v
	}

	runInfo = WorkflowRunInfo{
		Name:      describeResp.WorkflowExecutionInfo.Type.GetName(),
		TaskQueue: describeResp.WorkflowExecutionInfo.GetTaskQueue(),
		Memo:      decodedMemo,
	}

	iter := c.GetWorkflowHistory(
		context.Background(),
		workflowID,
		runID,
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)
	if iter == nil {
		return runInfo, fmt.Errorf("unable to get workflow history iterator (WorkflowID=%q, RunID=%q)", workflowID, runID)
	}

	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return runInfo, fmt.Errorf("error reading workflow history: %w", err)
		}

		if event.GetEventType() == enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED {
			attr := event.GetWorkflowExecutionStartedEventAttributes()
			var wi WorkflowInput
			if err := converter.GetDefaultDataConverter().FromPayloads(attr.Input, &wi); err != nil {
				return runInfo, fmt.Errorf("failed to decode workflow input payloads: %w", err)
			}
			runInfo.Input = wi
			break
		}
	}
	return runInfo, nil
}

func StartScheduledWorkflowWithOptions(runInfo WorkflowRunInfo, workflowID, namespace string, interval time.Duration) error {
	c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
	if err != nil {
		return fmt.Errorf("unable to create Temporal client for namespace %q: %w", namespace, err)
	}
	ctx := context.Background()
	scheduleID := fmt.Sprintf("schedule_id_%s", workflowID)
	scheduleHandle, err := c.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{
				{
					Every: interval,
				},
			},
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        fmt.Sprintf("scheduled_%s", workflowID),
			Workflow:  runInfo.Name,
			TaskQueue: runInfo.TaskQueue,
			Args:      []any{runInfo.Input},
			Memo:      runInfo.Memo,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start scheduledID from workflowID: %s", workflowID)
	}
	_, _ = scheduleHandle.Describe(ctx)

	return nil
}
func ListScheduledWorkflows(namespace string) ([]string, error) {
	c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to create Temporal client for namespace %q: %w", namespace, err)
	}

	ctx := context.Background()

	iter, err := c.ScheduleClient().List(ctx, client.ScheduleListOptions{
		PageSize: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	var schedules []string
	for iter.HasNext() {
		sched, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to list schedules: %w", err)
		}
		schedules = append(schedules, sched.ID)
	}

	return schedules, nil
}
