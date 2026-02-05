// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package runqueue

import "github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"

// RunnerStatus represents the run status for a single runner in the queue.
type RunnerStatus struct {
	RunnerID          string
	Status            mobilerunnersemaphore.MobileRunnerSemaphoreRunStatus
	Position          int
	LineLen           int
	WorkflowID        string
	RunID             string
	WorkflowNamespace string
	ErrorMessage      string
}

// AggregateStatus summarizes runner statuses for a queued run ticket.
type AggregateStatus struct {
	Status            mobilerunnersemaphore.MobileRunnerSemaphoreRunStatus
	Position          int
	LineLen           int
	WorkflowID        string
	RunID             string
	WorkflowNamespace string
	ErrorMessage      string
}

// AggregateRunnerStatuses computes the aggregate view for a set of runner statuses.
func AggregateRunnerStatuses(statuses []RunnerStatus) AggregateStatus {
	aggregateStatus := mobilerunnersemaphore.MobileRunnerSemaphoreRunNotFound
	aggregatePriority := runStatusPriority(aggregateStatus)
	maxPosition := 0
	maxLineLen := 0
	workflowID := ""
	runID := ""
	workflowNamespace := ""
	errorMessage := ""

	for _, status := range statuses {
		if status.Position > maxPosition {
			maxPosition = status.Position
		}
		if status.LineLen > maxLineLen {
			maxLineLen = status.LineLen
		}
		priority := runStatusPriority(status.Status)
		if priority > aggregatePriority {
			aggregateStatus = status.Status
			aggregatePriority = priority
		}
		if status.Status == mobilerunnersemaphore.MobileRunnerSemaphoreRunRunning && workflowID == "" {
			workflowID = status.WorkflowID
			runID = status.RunID
			workflowNamespace = status.WorkflowNamespace
		}
		if status.Status == mobilerunnersemaphore.MobileRunnerSemaphoreRunFailed && errorMessage == "" {
			errorMessage = status.ErrorMessage
		}
	}

	return AggregateStatus{
		Status:            aggregateStatus,
		Position:          maxPosition,
		LineLen:           maxLineLen,
		WorkflowID:        workflowID,
		RunID:             runID,
		WorkflowNamespace: workflowNamespace,
		ErrorMessage:      errorMessage,
	}
}

// runStatusPriority assigns comparison weights to runner status values.
func runStatusPriority(status mobilerunnersemaphore.MobileRunnerSemaphoreRunStatus) int {
	switch status {
	case mobilerunnersemaphore.MobileRunnerSemaphoreRunFailed:
		return 4
	case mobilerunnersemaphore.MobileRunnerSemaphoreRunCanceled:
		return 4
	case mobilerunnersemaphore.MobileRunnerSemaphoreRunRunning:
		return 3
	case mobilerunnersemaphore.MobileRunnerSemaphoreRunStarting:
		return 2
	case mobilerunnersemaphore.MobileRunnerSemaphoreRunQueued:
		return 1
	case mobilerunnersemaphore.MobileRunnerSemaphoreRunNotFound:
		return 0
	default:
		return 0
	}
}
