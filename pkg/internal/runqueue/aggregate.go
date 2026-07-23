// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package runqueue

import "github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"

// DeviceStatus represents the run status for a single runner in the queue.
type DeviceStatus struct {
	DeviceID          string
	Status            mobiledevicesemaphore.MobileDeviceSemaphoreRunStatus
	Position          int
	LineLen           int
	WorkflowID        string
	RunID             string
	WorkflowNamespace string
	ErrorMessage      string
}

// AggregateStatus summarizes runner statuses for a queued run ticket.
type AggregateStatus struct {
	Status            mobiledevicesemaphore.MobileDeviceSemaphoreRunStatus
	Position          int
	LineLen           int
	WorkflowID        string
	RunID             string
	WorkflowNamespace string
	ErrorMessage      string
}

// AggregateDeviceStatuses computes the aggregate view for a set of runner statuses.
func AggregateDeviceStatuses(statuses []DeviceStatus) AggregateStatus {
	aggregateStatus := mobiledevicesemaphore.MobileDeviceSemaphoreRunNotFound
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
		if status.Status == mobiledevicesemaphore.MobileDeviceSemaphoreRunRunning &&
			workflowID == "" {
			workflowID = status.WorkflowID
			runID = status.RunID
			workflowNamespace = status.WorkflowNamespace
		}
		if status.Status == mobiledevicesemaphore.MobileDeviceSemaphoreRunFailed &&
			errorMessage == "" {
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
func runStatusPriority(status mobiledevicesemaphore.MobileDeviceSemaphoreRunStatus) int {
	switch status {
	case mobiledevicesemaphore.MobileDeviceSemaphoreRunFailed:
		return 4
	case mobiledevicesemaphore.MobileDeviceSemaphoreRunCanceled:
		return 4
	case mobiledevicesemaphore.MobileDeviceSemaphoreRunRunning:
		return 3
	case mobiledevicesemaphore.MobileDeviceSemaphoreRunStarting:
		return 2
	case mobiledevicesemaphore.MobileDeviceSemaphoreRunQueued:
		return 1
	case mobiledevicesemaphore.MobileDeviceSemaphoreRunNotFound:
		return 0
	default:
		return 0
	}
}
