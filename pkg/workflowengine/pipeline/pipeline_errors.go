// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

type pipelineStepFailure struct {
	StepID  string
	Failure workflowengine.WorkflowError
}

func newPipelineStepFailure(stepID string, err error) pipelineStepFailure {
	failure := workflowengine.ParseWorkflowError(err)
	if isEmptyWorkflowError(failure) {
		failure = workflowengine.WorkflowError{
			Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
			Summary: "Pipeline step failed",
		}
		if err != nil {
			failure.Message = strings.TrimSpace(err.Error())
		}
	}

	failure.Details = mergeStepFailureDetails(failure.Details, stepID)

	return pipelineStepFailure{
		StepID:  stepID,
		Failure: failure,
	}
}

func isEmptyWorkflowError(failure workflowengine.WorkflowError) bool {
	return failure.Code == "" && failure.Summary == "" && failure.Message == ""
}

func newPipelineExecutionError(
	failures []pipelineStepFailure,
	output map[string]any,
	runMetadata *workflowengine.WorkflowRunMetadata,
) error {
	errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
	appErr := workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    errCode.Code,
		Summary: buildPipelineFailureSummary(failures),
		Details: map[string]any{
			"errors": buildPipelineFailureErrors(failures),
			"output": output,
		},
	})

	return workflowengine.NewWorkflowError(appErr, runMetadata)
}

func buildPipelineFailureSummary(failures []pipelineStepFailure) string {
	var base string
	if len(failures) == 1 {
		base = "Pipeline failed: 1 step failed"
	} else {
		base = fmt.Sprintf("Pipeline failed: %d steps failed", len(failures))
	}

	if len(failures) == 0 {
		return base
	}

	displayCount := len(failures)
	if displayCount > 3 {
		displayCount = 3
	}

	causes := make([]string, 0, displayCount)
	for _, failure := range failures[:displayCount] {
		cause := summarizePipelineStepFailure(failure)
		if cause == "" {
			continue
		}
		causes = append(causes, cause)
	}

	if len(causes) == 0 {
		return base
	}

	summary := fmt.Sprintf("%s; %s", base, strings.Join(causes, "; "))
	if len(failures) > displayCount {
		summary = fmt.Sprintf("%s; +%d more", summary, len(failures)-displayCount)
	}

	return summary
}

func buildPipelineFailureErrors(stepFailures []pipelineStepFailure) []workflowengine.WorkflowError {
	failures := make([]workflowengine.WorkflowError, 0, len(stepFailures))
	for _, failure := range stepFailures {
		failures = append(failures, failure.Failure)
	}
	return failures
}

func summarizePipelineStepFailure(failure pipelineStepFailure) string {
	cause := stepFailureCause(failure.Failure)
	if cause == "" {
		cause = "Pipeline step failed"
	}

	code := strings.TrimSpace(failure.Failure.Code)
	stepID := strings.TrimSpace(failure.StepID)

	switch {
	case stepID != "" && code != "":
		return fmt.Sprintf("%s failed with %s %s", stepID, code, cause)
	case stepID != "":
		return fmt.Sprintf("%s failed with %s", stepID, cause)
	case code != "":
		return fmt.Sprintf("%s %s", code, cause)
	default:
		return cause
	}
}

func mergeStepFailureDetails(details map[string]any, stepID string) map[string]any {
	merged := make(map[string]any, len(details)+1)
	for key, value := range details {
		merged[key] = value
	}

	merged["step_id"] = stepID
	return merged
}

func stepFailureCause(failure workflowengine.WorkflowError) string {
	if summary := strings.TrimSpace(failure.Summary); summary != "" {
		return summary
	}
	return strings.TrimSpace(failure.Message)
}
