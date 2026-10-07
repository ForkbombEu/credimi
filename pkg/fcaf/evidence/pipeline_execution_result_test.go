// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package evidence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodePipelineStepFailuresReadsStepIDFromWorkflowErrorDetails(t *testing.T) {
	failures, err := DecodePipelineStepFailures([]any{
		map[string]any{
			"code":    "CRE229",
			"summary": "assertion failed",
			"details": map[string]any{"step_id": "test-step"},
		},
	})

	require.NoError(t, err)
	require.Equal(t, []PipelineStepFailure{{
		StepID:  "test-step",
		Code:    "CRE229",
		Summary: "assertion failed",
	}}, failures)
}

func TestDecodePipelineExecutionResultPreservesStepFailures(t *testing.T) {
	result, err := DecodePipelineExecutionResult(map[string]any{
		"output": map[string]any{"step": map[string]any{"outputs": "ok"}},
		"stepFailures": []any{
			map[string]any{"step_id": "failed-step", "message": "failed"},
		},
	})

	require.NoError(t, err)
	require.Len(t, result.StepFailures, 1)
	require.Equal(t, "failed-step", result.StepFailures[0].StepID)
}

func TestPipelineExecutionResultRoundTripsThroughLegacyMap(t *testing.T) {
	original := PipelineExecutionResult{
		Output:        map[string]any{"step": "ok"},
		WorkflowID:    "workflow-1",
		WorkflowRunID: "run-1",
		StepFailures:  []PipelineStepFailure{{StepID: "step", Code: "CRE229"}},
	}

	legacy := original.LegacyMap()
	require.Equal(t, "workflow-1", legacy["workflowId"])
	require.Equal(t, "run-1", legacy["workflowRunId"])

	decoded, err := DecodePipelineExecutionResult(legacy)
	require.NoError(t, err)
	require.Equal(t, original, decoded)

	passthrough, err := DecodePipelineExecutionResult(original)
	require.NoError(t, err)
	require.Equal(t, original, passthrough)
}

func TestDecodePipelineExecutionResultRejectsInvalidShapes(t *testing.T) {
	_, err := DecodePipelineExecutionResult(map[string]any{"stepFailures": "failed"})
	require.ErrorContains(t, err, "decode pipeline execution result")

	_, err = DecodePipelineExecutionResult(map[string]any{"output": func() {}})
	require.ErrorContains(t, err, "marshal pipeline execution result")
}

func TestDecodePipelineStepFailuresHandlesEmptyAndInvalidInput(t *testing.T) {
	failures, err := DecodePipelineStepFailures(nil)
	require.NoError(t, err)
	require.Nil(t, failures)

	failures, err = DecodePipelineStepFailures([]any{
		map[string]any{"step_id": "top-level", "details": map[string]any{"step_id": "nested"}},
	})
	require.NoError(t, err)
	require.Equal(t, "top-level", failures[0].StepID, "explicit step_id wins over details")

	_, err = DecodePipelineStepFailures(map[string]any{"step_id": "not-a-list"})
	require.ErrorContains(t, err, "decode pipeline step failures")

	_, err = DecodePipelineStepFailures([]any{func() {}})
	require.ErrorContains(t, err, "marshal pipeline step failures")
}
