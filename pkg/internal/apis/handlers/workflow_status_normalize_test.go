// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeTemporalStatus(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{raw: "WORKFLOW_EXECUTION_STATUS_RUNNING", want: "Running"},
		{raw: "WORKFLOW_EXECUTION_STATUS_COMPLETED", want: "Completed"},
		{raw: "WORKFLOW_EXECUTION_STATUS_FAILED", want: "Failed"},
		{raw: "WORKFLOW_EXECUTION_STATUS_CANCELED", want: "Canceled"},
		{raw: "WORKFLOW_EXECUTION_STATUS_TERMINATED", want: "Terminated"},
		{raw: "WORKFLOW_EXECUTION_STATUS_TIMED_OUT", want: "TimedOut"},
		{raw: "WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW", want: "ContinuedAsNew"},
		{raw: "WORKFLOW_EXECUTION_STATUS_UNSPECIFIED", want: "Unspecified"},
		{raw: "Running", want: "Running"},
		{raw: "Queued", want: "Queued"},
		{raw: "Unspecified", want: "Unspecified"},
		{raw: "", want: "Unspecified"},
		{raw: "UNKNOWN_STATUS", want: "Unspecified"},
	}

	for _, tc := range cases {
		require.Equal(t, tc.want, normalizeTemporalStatus(tc.raw))
	}
}

func TestBuildExecutionHierarchyNormalizesStatus(t *testing.T) {
	exec := &WorkflowExecution{
		Execution: &WorkflowIdentifier{
			WorkflowID: "wf-1",
			RunID:      "run-1",
		},
		Type: WorkflowType{
			Name: "OtherWorkflow",
		},
		StartTime: time.Now().UTC().Format(time.RFC3339),
		Status:    "WORKFLOW_EXECUTION_STATUS_RUNNING",
	}

	summaries := buildExecutionHierarchy(nil, []*WorkflowExecution{exec}, "owner", "UTC", nil)
	require.Len(t, summaries, 1)
	require.Equal(t, "Running", summaries[0].Status)
}
