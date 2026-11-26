// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFailureErrorsKeepEntriesWithoutAStep(t *testing.T) {
	failure := WorkflowError{Details: map[string]any{"errors": []any{
		map[string]any{"code": "CRE315", "details": map[string]any{"step_id": "wallet"}},
		map[string]any{"code": "CRE101", "message": "no step"},
		"not an error",
		map[string]any{"code": "CRE228", "details": map[string]any{"step_id": "offer"}},
	}}}

	all := FailureErrors(failure)
	require.Len(t, all, 3)
	require.Equal(t, "CRE101", all[1].Code)
	require.Nil(t, all[1].Details["step_id"])

	steps := StepFailures(failure)
	require.Len(t, steps, 2)
	require.Equal(t, "wallet", steps[0].Details["step_id"])
	require.Equal(t, "offer", steps[1].Details["step_id"])

	// Typed entries, as an in-process failure carries them, decode too.
	typed := WorkflowError{Details: map[string]any{"errors": []WorkflowError{
		{Code: "CRE315", Details: map[string]any{"step_id": "wallet"}},
	}}}
	require.Len(t, StepFailures(typed), 1)
	require.Empty(t, FailureErrors(WorkflowError{}))
}
