// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var workflowTemporalClient = temporalclient.GetTemporalClientWithNamespace

// WorkflowInput represents the input data required to start a workflow.
type WorkflowInput struct {
	Payload         any                       `json:"payload,omitempty"`
	Config          map[string]any            `json:"config,omitempty"`
	ActivityOptions *workflow.ActivityOptions `json:"activityOptions,omitempty"`
	RunMetadata     *WorkflowRunMetadata      `json:"runMetadata,omitempty"`
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

type WorkflowRunMetadata struct {
	WorkflowName string `json:"workflowName,omitempty"`
	WorkflowID   string `json:"workflowId,omitempty"`
	RunID        string `json:"runId,omitempty"`
	Namespace    string `json:"namespace,omitempty"`
	TemporalUI   string `json:"temporalUI,omitempty"`
}

type WorkflowError struct {
	Code         string         `json:"code"`
	Summary      string         `json:"summary"`
	Message      string         `json:"message,omitempty"`
	ActivityName string         `json:"activityName,omitempty"`
	WorkflowName string         `json:"workflowName,omitempty"`
	WorkflowID   string         `json:"workflowId,omitempty"`
	RunID        string         `json:"runId,omitempty"`
	Namespace    string         `json:"namespace,omitempty"`
	TemporalUI   string         `json:"temporalUI,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
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
	ExecuteWorkflow(ctx workflow.Context, input WorkflowInput) (WorkflowResult, error)
	Name() string
	GetOptions() workflow.ActivityOptions
}

// WorkflowFn represents a function that executes a workflow.
type WorkflowFn func(ctx workflow.Context, input WorkflowInput) (WorkflowResult, error)

func BuildWorkflow(
	w Workflow,
) WorkflowFn {
	return func(ctx workflow.Context, input WorkflowInput) (WorkflowResult, error) {
		// ---- runtime metadata ----
		info := workflow.GetInfo(ctx)
		logger := workflow.GetLogger(ctx)
		input.RunMetadata = &WorkflowRunMetadata{
			WorkflowName: w.Name(),
			WorkflowID:   info.WorkflowExecution.ID,
			RunID:        info.WorkflowExecution.RunID,
			Namespace:    info.Namespace,
			TemporalUI: utils.JoinURL(
				input.Config["app_url"].(string),
				"my", "tests", "runs",
				info.WorkflowExecution.ID,
				info.WorkflowExecution.RunID,
			),
		}

		// ---- activity options composition ----
		ao := w.GetOptions()
		if input.ActivityOptions != nil {
			ao = *input.ActivityOptions
		}
		ctx = workflow.WithActivityOptions(ctx, ao)

		result, err := w.ExecuteWorkflow(ctx, input)

		if err != nil {
			if temporal.IsTimeoutError(err) {
				return result, err
			}

			if temporal.IsCanceledError(err) {
				logger.Info("Workflow was canceled", "WorkflowID", info.WorkflowExecution.ID)
				return result, NewWorkflowCancellationError(input.RunMetadata)
			}

			return result, NewWorkflowError(
				err,
				input.RunMetadata,
			)
		}
		return result, nil
	}
}

func NewWorkflowError(err error, metadata *WorkflowRunMetadata) error {
	var appErr *temporal.ApplicationError
	if !temporal.IsApplicationError(err) || !errors.As(err, &appErr) {
		return err
	}

	failure := workflowErrorFromApplicationError(appErr)
	if failure.Code == "" {
		failure.Code = appErr.Type()
	}
	if failure.Summary == "" {
		failure.Summary = appErr.Message()
	}
	applyWorkflowRunMetadata(&failure, metadata)

	return temporal.NewApplicationError(
		workflowFailureMessage(failure),
		failure.Code,
		failure,
	)
}

func workflowErrorFromApplicationError(appErr *temporal.ApplicationError) WorkflowError {
	var failure WorkflowError
	if decodeApplicationErrorDetails(appErr, &failure) && failure.Code != "" {
		return failure
	}

	var activityErr ActivityError
	if decodeApplicationErrorDetails(appErr, &activityErr) && activityErr.Code != "" {
		return WorkflowError{
			Code:         activityErr.Code,
			Summary:      activityErr.Summary,
			Message:      activityErr.Message,
			ActivityName: activityErr.ActivityName,
			Details:      activityErr.Details,
		}
	}

	return WorkflowError{
		Code:    appErr.Type(),
		Summary: appErr.Message(),
	}
}

func applyWorkflowRunMetadata(failure *WorkflowError, metadata *WorkflowRunMetadata) {
	if metadata == nil {
		return
	}
	if failure.WorkflowName == "" {
		failure.WorkflowName = metadata.WorkflowName
	}
	if failure.WorkflowID == "" {
		failure.WorkflowID = metadata.WorkflowID
	}
	if failure.RunID == "" {
		failure.RunID = metadata.RunID
	}
	if failure.Namespace == "" {
		failure.Namespace = metadata.Namespace
	}
	if failure.TemporalUI == "" {
		failure.TemporalUI = metadata.TemporalUI
	}
}

func decodeApplicationErrorDetails(appErr *temporal.ApplicationError, target any) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	return appErr.Details(target) == nil
}

func NewWorkflowCancellationError(metadata *WorkflowRunMetadata) error {
	errCode := errorcodes.Codes[errorcodes.WorkflowCancellationError]

	return temporal.NewCanceledError(errCode.Code, errCode.Description, metadata)
}

func NewAppError(failure WorkflowError) error {
	return temporal.NewApplicationError(workflowFailureMessage(failure), failure.Code, failure)
}

func workflowFailureMessage(failure WorkflowError) string {
	return errorMessage(failure.Summary, failure.Message)
}

func FormatWorkflowFailureReason(failure WorkflowError) string {
	message := workflowFailureMessage(failure)
	name := failure.ActivityName
	if name == "" {
		name = failure.WorkflowName
	}

	switch {
	case failure.Code != "" && name != "" && message != "":
		return fmt.Sprintf("%s: [%s] %s", failure.Code, name, message)
	case failure.Code != "" && message != "":
		return fmt.Sprintf("%s: %s", failure.Code, message)
	case name != "" && message != "":
		return fmt.Sprintf("[%s] %s", name, message)
	default:
		return message
	}
}

func errorMessage(summary string, message string) string {
	if message == "" || message == summary {
		return summary
	}
	return fmt.Sprintf("%s: %s", summary, message)
}

func NewMissingOrInvalidPayloadError(err error, runMetadata *WorkflowRunMetadata) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	appErr := NewAppError(WorkflowError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: err.Error(),
	})
	return NewWorkflowError(appErr, runMetadata)
}

func NewMissingConfigError(key string, metadata *WorkflowRunMetadata) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
	appErr := NewAppError(WorkflowError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: fmt.Sprintf("missing or invalid config key %q", key),
		Details: map[string]any{
			"key": key,
		},
	})
	return NewWorkflowError(appErr, metadata)
}

func NewStepCIOutputError(field string, output any, metadata *WorkflowRunMetadata) error {
	errCode := errorcodes.Codes[errorcodes.UnexpectedStepCIOutput]
	appErr := NewAppError(WorkflowError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: fmt.Sprintf("unexpected StepCI output field %q", field),
		Details: map[string]any{
			"field":  field,
			"output": output,
		},
	})
	return NewWorkflowError(appErr, metadata)
}

func StartWorkflowWithOptions(
	namespace string,
	options client.StartWorkflowOptions,
	name string,
	input WorkflowInput,
) (result WorkflowResult, err error) {
	c, err := workflowTemporalClient(
		namespace,
	)
	if err != nil {
		return WorkflowResult{}, fmt.Errorf("unable to create client: %w", err)
	}

	if input.Config["memo"] != nil {
		options.Memo = input.Config["memo"].(map[string]any)
	}

	if options.Memo == nil {
		options.Memo = make(map[string]any)
	}

	if options.Memo["test"] == nil {
		options.Memo["test"] = name
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

func ParseWorkflowError(err error) WorkflowError {
	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		return WorkflowError{}
	}
	return workflowErrorFromApplicationError(appErr)
}

func extractWorkflowError(err error) (WorkflowError, bool) {
	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		return WorkflowError{}, false
	}
	return workflowErrorFromApplicationError(appErr), true
}

func ExtractOutputFromError(err error) map[string]any {
	failure, ok := extractWorkflowError(err)
	if !ok || failure.Details == nil {
		return nil
	}

	out, ok := failure.Details["output"]
	if !ok {
		return nil
	}
	if outMap, ok := out.(map[string]any); ok {
		return outMap
	}

	return nil
}
