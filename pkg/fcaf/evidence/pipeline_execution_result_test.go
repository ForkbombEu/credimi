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
