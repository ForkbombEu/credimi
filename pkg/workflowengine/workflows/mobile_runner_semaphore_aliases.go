// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import "github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"

const (
	MobileRunnerSemaphoreTaskQueue            = mobilerunnersemaphore.TaskQueue
	MobileRunnerSemaphoreWorkflowName         = mobilerunnersemaphore.WorkflowName
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
	MobileRunnerSemaphoreErrQueueLimitExceeded = mobilerunnersemaphore.ErrQueueLimitExceeded
)

type MobileRunnerSemaphoreWorkflowInput = mobilerunnersemaphore.MobileRunnerSemaphoreWorkflowInput

type MobileRunnerSemaphoreWorkflowState = mobilerunnersemaphore.MobileRunnerSemaphoreWorkflowState

type MobileRunnerSemaphoreStateView = mobilerunnersemaphore.MobileRunnerSemaphoreStateView

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
