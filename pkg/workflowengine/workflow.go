// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WorkflowInput represents the input data required to start a workflow.
type WorkflowInput struct {
	Payload         map[string]any            `json:"payload,omitempty"`
	Config          map[string]any            `json:"config,omitempty"`
	ActivityOptions *workflow.ActivityOptions `json:"activityOptions,omitempty"`
}

// WorkflowResult represents the result of a workflow execution, including a message, errors, and a log.
type WorkflowResult struct {
	WorkflowID    string `json:"workflowId,omitempty"`
	WorkflowRunID string `json:"workflowRunId,omitempty"`
	Author        string `json:"author,omitempty"`
	Message       string `json:"message,omitempty"`
	Errors        any    `json:"errors,omitempty"`
	Output        any    `json:"output,omitempty"`
	Log           any    `json:"log,omitempty"`
}

type WorkflowErrorMetadata struct {
	WorkflowName string `json:"workflowName,omitempty"`
	WorkflowID   string `json:"workflowId,omitempty"`
	Namespace    string `json:"namespace,omitempty"`
	TemporalUI   string `json:"temporalUI,omitempty"`
}

type WorkflowErrorDetails struct {
	WorkflowID string `json:"workflowID,omitempty"`
	RunID      string `json:"runID,omitempty"`
	Code       string `json:"code,omitempty"`
	Retryable  bool   `json:"retryable,omitempty"`
	Message    string `json:"message,omitempty"`
	Summary    string `json:"summary,omitempty"`
	Link       string `json:"link,omitempty"`
	Payload    any    `json:"payload,omitempty"`
}

type WorkflowRunInfo struct {
	Name      string         `json:"name"`
	TaskQueue string         `json:"taskQueue"`
	Input     WorkflowInput  `json:"input,omitempty"`
	Memo      map[string]any `json:"memo,omitempty"`
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
	namespace string,
	options client.StartWorkflowOptions,
	name string,
	input WorkflowInput,
) (result WorkflowResult, err error) {
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
		return WorkflowRunInfo{}, fmt.Errorf(
			"unable to create Temporal client for namespace %q: %w",
			namespace,
			err,
		)
	}

	describeResp, err := c.DescribeWorkflowExecution(context.Background(), workflowID, runID)
	if err != nil {
		return WorkflowRunInfo{}, fmt.Errorf(
			"unable to describe workflow execution (WorkflowID=%q, RunID=%q): %w",
			workflowID,
			runID,
			err,
		)
	}

	decodedMemo := make(map[string]any)
	for k, payload := range describeResp.GetWorkflowExecutionInfo().GetMemo().GetFields() {
		var v any
		if err := converter.GetDefaultDataConverter().FromPayload(payload, &v); err != nil {
			return WorkflowRunInfo{}, fmt.Errorf("failed to decode memo key %q: %w", k, err)
		}
		decodedMemo[k] = v
	}

	runInfo = WorkflowRunInfo{
		Name:      describeResp.GetWorkflowExecutionInfo().GetType().GetName(),
		TaskQueue: describeResp.GetWorkflowExecutionInfo().GetTaskQueue(),
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
		return runInfo, fmt.Errorf(
			"unable to get workflow history iterator (WorkflowID=%q, RunID=%q)",
			workflowID,
			runID,
		)
	}

	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return runInfo, fmt.Errorf("error reading workflow history: %w", err)
		}

		if event.GetEventType() == enums.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED {
			attr := event.GetWorkflowExecutionStartedEventAttributes()
			var wi WorkflowInput
			if err := converter.GetDefaultDataConverter().FromPayloads(attr.GetInput(), &wi); err != nil {
				return runInfo, fmt.Errorf("failed to decode workflow input payloads: %w", err)
			}
			runInfo.Input = wi
			break
		}
	}
	return runInfo, nil
}

func StartScheduledWorkflowWithOptions(
	runInfo WorkflowRunInfo,
	workflowID, namespace string,
	interval time.Duration,
) error {
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
		return nil, fmt.Errorf(
			"unable to create Temporal client for namespace %q: %w",
			namespace,
			err,
		)
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

// Wait for final workflow result
func WaitForWorkflowResult(c client.Client, workflowID, runID string) (WorkflowResult, error) {
	var result WorkflowResult
	we := c.GetWorkflow(context.Background(), workflowID, runID)
	if err := we.Get(context.Background(), &result); err != nil {
		return result, err
	}
	return result, nil
}

// ErrNotReady is returned by a workflow query when the requested data is not ready yet.
type NotReadyError struct{}

// Error implements the error interface for ErrNotReady.
func (e NotReadyError) Error() string {
	return "result not ready"
}

// Fetch partial workflow result via query (generic)
func WaitForPartialResult[T any](
	c client.Client,
	workflowID, runID, queryName string,
	pollInterval time.Duration,
	maxWait time.Duration, // 0 = no timeout
) (T, error) {
	var result T

	// Context with timeout if maxWait > 0
	ctx := context.Background()
	if maxWait > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, maxWait)
		defer cancel()
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return result, fmt.Errorf("timeout waiting for partial result: %w", ctx.Err())
		case <-ticker.C:
			queryResp, err := c.QueryWorkflow(ctx, workflowID, runID, queryName)
			if err != nil {
				if strings.Contains(err.Error(), "result not ready") {
					// Query not ready yet → keep polling
					continue
				}
				return result, err
			}

			// Got query result → decode into result
			if err := queryResp.Get(&result); err != nil {
				return result, err
			}
			return result, nil
		}
	}
}

func ParseWorkflowError(err error) WorkflowErrorDetails {
	msg := err.Error()
	details := WorkflowErrorDetails{}
	reIDs := regexp.MustCompile(`workflowID: ([^,]+), runID: ([^)]+)`)
	if matches := reIDs.FindStringSubmatch(msg); len(matches) == 3 {
		details.WorkflowID = matches[1]
		details.RunID = matches[2]
	}
	reCode := regexp.MustCompile(`\(type: ([^,]+), retryable: (true|false)\)`)
	if matches := reCode.FindStringSubmatch(msg); len(matches) == 3 {
		details.Code = matches[1]
		details.Retryable = matches[2] == "true"
	}
	reLink := regexp.MustCompile(`Further information at: (http[^\)]+)`)
	if matches := reLink.FindStringSubmatch(msg); len(matches) == 2 {
		details.Link = matches[1]
	}

	if details.Code != "" {
		// Full message = everything after "<code>:"
		parts := strings.SplitN(msg, details.Code+":", 2)
		if len(parts) == 2 {
			summaryPart := parts[1]

			if idx := strings.Index(summaryPart, "(Further information"); idx != -1 {
				summaryPart = summaryPart[:idx]
			}
			details.Message = strings.TrimSpace(summaryPart)

			reCompact := regexp.MustCompile(`\]:\s*(.*)$`)
			if matches := reCompact.FindStringSubmatch(details.Message); len(matches) == 2 {
				details.Summary = strings.TrimSpace(matches[1])
			} else {
				details.Summary = details.Message
			}
		}
	}
	details.Payload = extractAppErrorPayload(err)

	return details
}

func extractAppErrorPayload(err error) []any {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		var details []any
		derr := appErr.Details(&details)
		if derr == nil {
			return details
		}
		return nil
	}
	return nil
}
