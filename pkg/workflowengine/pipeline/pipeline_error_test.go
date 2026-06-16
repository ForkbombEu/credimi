// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"errors"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	internalpipeline "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestBuildPipelineFailureSummary(t *testing.T) {
	summary := buildPipelineFailureSummary([]pipelineStepFailure{
		{
			StepID: "get-credential",
			Failure: workflowengine.WorkflowError{
				Code:    "CRE302",
				Summary: "StepCI checks failed",
			},
		},
		{
			StepID: "mobile",
			Failure: workflowengine.WorkflowError{
				Code:    "CRE228",
				Summary: "Failed to resolve pipeline inputs",
			},
		},
	})

	require.Equal(
		t,
		"Pipeline failed: 2 steps failed; get-credential failed with CRE302 StepCI checks failed; mobile failed with CRE228 Failed to resolve pipeline inputs",
		summary,
	)
}

func TestBuildPipelineFailureSummaryLimitsDisplayedCauses(t *testing.T) {
	summary := buildPipelineFailureSummary([]pipelineStepFailure{
		{StepID: "step-1", Failure: workflowengine.WorkflowError{Code: "CRE-1", Summary: "err-1"}},
		{StepID: "step-2", Failure: workflowengine.WorkflowError{Code: "CRE-2", Summary: "err-2"}},
		{StepID: "step-3", Failure: workflowengine.WorkflowError{Code: "CRE-3", Summary: "err-3"}},
		{StepID: "step-4", Failure: workflowengine.WorkflowError{Code: "CRE-4", Summary: "err-4"}},
	})

	require.Equal(
		t,
		"Pipeline failed: 4 steps failed; step-1 failed with CRE-1 err-1; step-2 failed with CRE-2 err-2; step-3 failed with CRE-3 err-3; +1 more",
		summary,
	)
}

func TestNewPipelineStepFailureStructuredFallback(t *testing.T) {
	failure := newPipelineStepFailure("step-1", errors.New("capture failed"))

	require.Equal(t, "step-1", failure.StepID)
	require.Equal(t, errorcodes.Codes[errorcodes.PipelineExecutionError].Code, failure.Failure.Code)
	require.Equal(t, "Pipeline step failed", failure.Failure.Summary)
	require.Equal(t, "capture failed", failure.Failure.Message)
	require.Equal(t, "step-1", failure.Failure.Details["step_id"])
}

func TestNewPipelineStepFailurePlainFallbackKeepsOriginalMessage(t *testing.T) {
	message := "child workflow execution error (type: Dynamic Pipeline Workflow, workflowID: wf, runID: run): StepCI checks failed: One or more StepCI assertions failed. (type: CRE302, retryable: true)"
	failure := newPipelineStepFailure("child-step", errors.New(message))

	require.Equal(t, errorcodes.Codes[errorcodes.PipelineExecutionError].Code, failure.Failure.Code)
	require.Equal(t, "Pipeline step failed", failure.Failure.Summary)
	require.Equal(t, message, failure.Failure.Message)
	require.Equal(t, "child-step", failure.Failure.Details["step_id"])
}

func TestNewPipelineStepFailureCopiesDetailsBeforeAddingStepID(t *testing.T) {
	details := map[string]any{
		"output": map[string]any{"step": "login"},
		"raw":    "raw StepCI result",
	}
	err := workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    "CRE302",
		Summary: "StepCI checks failed",
		Message: "One or more StepCI assertions failed.",
		Details: details,
	})

	failure := newPipelineStepFailure("child-step", err)

	require.Equal(t, "CRE302", failure.Failure.Code)
	require.Equal(t, "StepCI checks failed", failure.Failure.Summary)
	require.Equal(t, "One or more StepCI assertions failed.", failure.Failure.Message)
	require.Equal(t, map[string]any{"step": "login"}, failure.Failure.Details["output"])
	require.Equal(t, "raw StepCI result", failure.Failure.Details["raw"])
	require.Equal(t, "child-step", failure.Failure.Details["step_id"])
	require.NotContains(t, details, "step_id")
}

func TestDecodePayloadReportsInvalidParameterValueType(t *testing.T) {
	step := &internalpipeline.StepDefinition{
		StepSpec: internalpipeline.StepSpec{
			ID:  "mobile-step",
			Use: "mobile-automation",
			With: internalpipeline.StepInputs{
				Payload: map[string]any{
					"runner_id": "ios-runner",
					"parameters": map[string]any{
						"offer": map[string]any{"url": "https://example.test"},
					},
				},
			},
		},
	}

	_, err := DecodePayload(step)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode payload for mobile-step")
	require.Contains(t, err.Error(), "invalid payload for mobile-automation")
	require.Contains(t, err.Error(), "with.payload.parameters must contain only string values")
	require.Contains(t, err.Error(), `parameter "offer" resolved to object`)
}
