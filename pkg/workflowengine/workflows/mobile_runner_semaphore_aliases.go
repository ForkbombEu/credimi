// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import "github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"

const (
	MobileRunnerSemaphoreTaskQueue     = mobilerunnersemaphore.TaskQueue
	MobileRunnerSemaphoreWorkflowName  = mobilerunnersemaphore.WorkflowName
	MobileRunnerSemaphoreAcquireUpdate = mobilerunnersemaphore.AcquireUpdate
	MobileRunnerSemaphoreReleaseUpdate = mobilerunnersemaphore.ReleaseUpdate
	MobileRunnerSemaphoreStateQuery    = mobilerunnersemaphore.StateQuery

	MobileRunnerSemaphoreErrInvalidRequest = mobilerunnersemaphore.ErrInvalidRequest
	MobileRunnerSemaphoreErrTimeout        = mobilerunnersemaphore.ErrTimeout
)

type MobileRunnerSemaphoreWorkflowInput = mobilerunnersemaphore.MobileRunnerSemaphoreWorkflowInput

type MobileRunnerSemaphoreWorkflowState = mobilerunnersemaphore.MobileRunnerSemaphoreWorkflowState

type MobileRunnerSemaphoreAcquireRequest = mobilerunnersemaphore.MobileRunnerSemaphoreAcquireRequest

type MobileRunnerSemaphoreReleaseRequest = mobilerunnersemaphore.MobileRunnerSemaphoreReleaseRequest

type MobileRunnerSemaphorePermit = mobilerunnersemaphore.MobileRunnerSemaphorePermit

type MobileRunnerSemaphoreReleaseResult = mobilerunnersemaphore.MobileRunnerSemaphoreReleaseResult

type MobileRunnerSemaphoreHolder = mobilerunnersemaphore.MobileRunnerSemaphoreHolder

type MobileRunnerSemaphoreQueueEntry = mobilerunnersemaphore.MobileRunnerSemaphoreQueueEntry

type MobileRunnerSemaphoreStateView = mobilerunnersemaphore.MobileRunnerSemaphoreStateView

type MobileRunnerSemaphoreRequestStatus = mobilerunnersemaphore.MobileRunnerSemaphoreRequestStatus

type MobileRunnerSemaphoreRequestState = mobilerunnersemaphore.MobileRunnerSemaphoreRequestState

func MobileRunnerSemaphoreWorkflowID(runnerID string) string {
	return mobilerunnersemaphore.WorkflowID(runnerID)
}

func MobileRunnerSemaphorePermitLeaseID(workflowID, runID, runnerID string) string {
	return mobilerunnersemaphore.PermitLeaseID(workflowID, runID, runnerID)
}

func MobileRunnerSemaphoreAcquireUpdateID(requestID string) string {
	return mobilerunnersemaphore.AcquireUpdateID(requestID)
}

func MobileRunnerSemaphoreReleaseUpdateID(leaseID string) string {
	return mobilerunnersemaphore.ReleaseUpdateID(leaseID)
}
