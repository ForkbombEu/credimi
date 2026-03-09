// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package mobilerunnersemaphore

import (
	"fmt"
	"time"
)

const (
	TaskQueue    = "mobile-runner-semaphore-task-queue"
	WorkflowName = "mobile-runner-semaphore"
	StateQuery   = "GetState"

	ErrInvalidRequest     = "mobile-runner-semaphore-invalid-request"
	ErrQueueLimitExceeded = "mobile-runner-semaphore-queue-limit-exceeded"
)

const (
	EnqueueRunUpdate    = "EnqueueRun"
	RunStatusQuery      = "GetRunStatus"
	ListQueuedRunsQuery = "ListQueuedRuns"
	RunDoneUpdate       = "RunDone"
	CancelRunUpdate     = "CancelRun"

	RunGrantedSignal = "RunGranted"
	RunStartedSignal = "RunStarted"
	RunDoneSignal    = "RunDoneSignal"
)

type MobileRunnerSemaphoreWorkflowInput struct {
	RunnerID string                              `json:"runner_id"`
	Capacity int                                 `json:"capacity"`
	State    *MobileRunnerSemaphoreWorkflowState `json:"state,omitempty"`
}

type MobileRunnerSemaphoreWorkflowState struct {
	Capacity    int                                            `json:"capacity"`
	UpdateCount int                                            `json:"update_count,omitempty"`
	RunQueue    []string                                       `json:"run_queue,omitempty"`
	RunTickets  map[string]MobileRunnerSemaphoreRunTicketState `json:"run_tickets,omitempty"`
}

type MobileRunnerSemaphoreStateView struct {
	RunnerID  string `json:"runner_id"`
	Capacity  int    `json:"capacity"`
	SlotsUsed int    `json:"slots_used"`
	QueueLen  int    `json:"queue_len"`
}

type MobileRunnerSemaphoreRunStatus string

const (
	MobileRunnerSemaphoreRunQueued   MobileRunnerSemaphoreRunStatus = "queued"
	MobileRunnerSemaphoreRunStarting MobileRunnerSemaphoreRunStatus = "starting"
	MobileRunnerSemaphoreRunRunning  MobileRunnerSemaphoreRunStatus = "running"
	MobileRunnerSemaphoreRunFailed   MobileRunnerSemaphoreRunStatus = "failed"
	MobileRunnerSemaphoreRunCanceled MobileRunnerSemaphoreRunStatus = "canceled"
	MobileRunnerSemaphoreRunNotFound MobileRunnerSemaphoreRunStatus = "not_found"
)

type MobileRunnerSemaphoreEnqueueRunRequest struct {
	TicketID            string         `json:"ticket_id"`
	OwnerNamespace      string         `json:"owner_namespace"`
	EnqueuedAt          time.Time      `json:"enqueued_at"`
	RunnerID            string         `json:"runner_id"`
	RequiredRunnerIDs   []string       `json:"required_runner_ids"`
	LeaderRunnerID      string         `json:"leader_runner_id"`
	MaxPipelinesInQueue int            `json:"max_pipelines_in_queue,omitempty"`
	PipelineIdentifier  string         `json:"pipeline_identifier,omitempty"`
	YAML                string         `json:"yaml,omitempty"`
	PipelineConfig      map[string]any `json:"pipeline_config,omitempty"`
	Memo                map[string]any `json:"memo,omitempty"`
}

type MobileRunnerSemaphoreEnqueueRunResponse struct {
	TicketID string                         `json:"ticket_id"`
	Status   MobileRunnerSemaphoreRunStatus `json:"status"`
	Position int                            `json:"position"`
	LineLen  int                            `json:"line_len"`
}

type MobileRunnerSemaphoreRunStatusView struct {
	TicketID          string                         `json:"ticket_id"`
	Status            MobileRunnerSemaphoreRunStatus `json:"status"`
	Position          int                            `json:"position"`
	LineLen           int                            `json:"line_len"`
	LeaderRunnerID    string                         `json:"leader_runner_id,omitempty"`
	RequiredRunnerIDs []string                       `json:"required_runner_ids,omitempty"`
	WorkflowID        string                         `json:"workflow_id,omitempty"`
	RunID             string                         `json:"run_id,omitempty"`
	WorkflowNamespace string                         `json:"workflow_namespace,omitempty"`
	ErrorMessage      string                         `json:"error_message,omitempty"`
}

type MobileRunnerSemaphoreQueuedRunView struct {
	TicketID           string                         `json:"ticket_id"`
	OwnerNamespace     string                         `json:"owner_namespace"`
	PipelineIdentifier string                         `json:"pipeline_identifier,omitempty"`
	EnqueuedAt         time.Time                      `json:"enqueued_at"`
	LeaderRunnerID     string                         `json:"leader_runner_id,omitempty"`
	RequiredRunnerIDs  []string                       `json:"required_runner_ids,omitempty"`
	Status             MobileRunnerSemaphoreRunStatus `json:"status"`
	Position           int                            `json:"position"`
	LineLen            int                            `json:"line_len"`
}

type MobileRunnerSemaphoreRunDoneRequest struct {
	TicketID       string `json:"ticket_id"`
	OwnerNamespace string `json:"owner_namespace,omitempty"`
	WorkflowID     string `json:"workflow_id,omitempty"`
	RunID          string `json:"run_id,omitempty"`
}

type MobileRunnerSemaphoreRunCancelRequest struct {
	TicketID       string `json:"ticket_id"`
	OwnerNamespace string `json:"owner_namespace,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

type MobileRunnerSemaphoreRunGrantedSignal struct {
	TicketID string `json:"ticket_id"`
	RunnerID string `json:"runner_id"`
}

type MobileRunnerSemaphoreRunStartedSignal struct {
	TicketID          string `json:"ticket_id"`
	WorkflowID        string `json:"workflow_id"`
	RunID             string `json:"run_id"`
	WorkflowNamespace string `json:"workflow_namespace"`
}

type MobileRunnerSemaphoreRunDoneSignal struct {
	TicketID   string `json:"ticket_id"`
	WorkflowID string `json:"workflow_id,omitempty"`
	RunID      string `json:"run_id,omitempty"`
}

type MobileRunnerSemaphoreRunTicketState struct {
	Request           MobileRunnerSemaphoreEnqueueRunRequest `json:"request"`
	Status            MobileRunnerSemaphoreRunStatus         `json:"status"`
	WorkflowID        string                                 `json:"workflow_id,omitempty"`
	RunID             string                                 `json:"run_id,omitempty"`
	WorkflowNamespace string                                 `json:"workflow_namespace,omitempty"`
	ErrorMessage      string                                 `json:"error_message,omitempty"`
	CancelRequested   bool                                   `json:"cancel_requested,omitempty"`
	GrantedRunnerIDs  map[string]bool                        `json:"granted_runner_ids,omitempty"`
	StartedAt         *time.Time                             `json:"started_at,omitempty"`
	DoneAt            *time.Time                             `json:"done_at,omitempty"`
}

func WorkflowID(runnerID string) string {
	return fmt.Sprintf("mobile-runner-semaphore/%s", runnerID)
}
