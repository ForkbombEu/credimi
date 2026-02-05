// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import "github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"

const (
	MobileRunnerSemaphoreTaskQueue            = mobilerunnersemaphore.TaskQueue
	MobileRunnerSemaphoreWorkflowName         = mobilerunnersemaphore.WorkflowName
	MobileRunnerSemaphoreAcquireUpdate        = mobilerunnersemaphore.AcquireUpdate
	MobileRunnerSemaphoreReleaseUpdate        = mobilerunnersemaphore.ReleaseUpdate
	MobileRunnerSemaphoreStateQuery           = mobilerunnersemaphore.StateQuery
	MobileRunnerSemaphoreEnqueueRunUpdate     = mobilerunnersemaphore.EnqueueRunUpdate
	MobileRunnerSemaphoreRunStatusQuery       = mobilerunnersemaphore.RunStatusQuery
	MobileRunnerSemaphoreListQueuedRunsQuery  = mobilerunnersemaphore.ListQueuedRunsQuery
	MobileRunnerSemaphoreRunDoneUpdate        = mobilerunnersemaphore.RunDoneUpdate
	MobileRunnerSemaphoreCancelRunUpdate      = mobilerunnersemaphore.CancelRunUpdate
	MobileRunnerSemaphoreRunGrantedSignalName = mobilerunnersemaphore.RunGrantedSignal
	MobileRunnerSemaphoreRunStartedSignalName = mobilerunnersemaphore.RunStartedSignal
	MobileRunnerSemaphoreRunDoneSignalName    = mobilerunnersemaphore.RunDoneSignal

	MobileRunnerSemaphoreErrInvalidRequest     = mobilerunnersemaphore.ErrInvalidRequest
	MobileRunnerSemaphoreErrTimeout            = mobilerunnersemaphore.ErrTimeout
	MobileRunnerSemaphoreErrQueueLimitExceeded = mobilerunnersemaphore.ErrQueueLimitExceeded
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

type MobileRunnerSemaphoreRunStatus = mobilerunnersemaphore.MobileRunnerSemaphoreRunStatus

type MobileRunnerSemaphoreEnqueueRunRequest = mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunRequest

type MobileRunnerSemaphoreEnqueueRunResponse = mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunResponse

type MobileRunnerSemaphoreRunStatusView = mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView

type MobileRunnerSemaphoreQueuedRunView = mobilerunnersemaphore.MobileRunnerSemaphoreQueuedRunView

type MobileRunnerSemaphoreRunDoneRequest = mobilerunnersemaphore.MobileRunnerSemaphoreRunDoneRequest

type MobileRunnerSemaphoreRunCancelRequest = mobilerunnersemaphore.MobileRunnerSemaphoreRunCancelRequest

type MobileRunnerSemaphoreRunGrantedSignal = mobilerunnersemaphore.MobileRunnerSemaphoreRunGrantedSignal

type MobileRunnerSemaphoreRunStartedSignal = mobilerunnersemaphore.MobileRunnerSemaphoreRunStartedSignal

type MobileRunnerSemaphoreRunDoneSignal = mobilerunnersemaphore.MobileRunnerSemaphoreRunDoneSignal

type MobileRunnerSemaphoreRunTicketState = mobilerunnersemaphore.MobileRunnerSemaphoreRunTicketState

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
