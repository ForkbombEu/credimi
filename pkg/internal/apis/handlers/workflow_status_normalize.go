// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

// normalizeTemporalStatus maps Temporal enum strings into API-friendly CamelCase labels.
func normalizeTemporalStatus(raw string) string {
	switch raw {
	case string(WorkflowStatusRunning),
		string(WorkflowStatusCompleted),
		string(WorkflowStatusFailed),
		string(WorkflowStatusCanceled),
		string(WorkflowStatusTerminated),
		string(WorkflowStatusTimedOut),
		string(WorkflowStatusContinuedAsNew),
		"Queued",
		string(WorkflowStatusUnspecified):
		return raw
	case "WORKFLOW_EXECUTION_STATUS_RUNNING":
		return string(WorkflowStatusRunning)
	case "WORKFLOW_EXECUTION_STATUS_COMPLETED":
		return string(WorkflowStatusCompleted)
	case "WORKFLOW_EXECUTION_STATUS_FAILED":
		return string(WorkflowStatusFailed)
	case "WORKFLOW_EXECUTION_STATUS_CANCELED":
		return string(WorkflowStatusCanceled)
	case "WORKFLOW_EXECUTION_STATUS_TERMINATED":
		return string(WorkflowStatusTerminated)
	case "WORKFLOW_EXECUTION_STATUS_TIMED_OUT":
		return string(WorkflowStatusTimedOut)
	case "WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW":
		return string(WorkflowStatusContinuedAsNew)
	case "WORKFLOW_EXECUTION_STATUS_UNSPECIFIED":
		return string(WorkflowStatusUnspecified)
	default:
		return string(WorkflowStatusUnspecified)
	}
}
