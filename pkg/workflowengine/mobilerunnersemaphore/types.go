// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package mobilerunnersemaphore

import (
	"fmt"
	"time"
)

const (
	TaskQueue     = "mobile-runner-semaphore-task-queue"
	WorkflowName  = "mobile-runner-semaphore"
	AcquireUpdate = "Acquire"
	ReleaseUpdate = "Release"
	StateQuery    = "GetState"

	ErrInvalidRequest = "mobile-runner-semaphore-invalid-request"
	ErrTimeout        = "mobile-runner-semaphore-timeout"
)

const (
	acquireUpdateIDPrefix = "acquire/"
	releaseUpdateIDPrefix = "release/"
)

const (
	EnqueueRunUpdate = "EnqueueRun"
	RunStatusQuery   = "GetRunStatus"
	RunDoneUpdate    = "RunDone"
	CancelRunUpdate  = "CancelRun"

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
	Holders     map[string]MobileRunnerSemaphoreHolder         `json:"holders,omitempty"`
	Queue       []string                                       `json:"queue,omitempty"`
	Requests    map[string]MobileRunnerSemaphoreRequestState   `json:"requests,omitempty"`
	LastGrantAt *time.Time                                     `json:"last_grant_at,omitempty"`
	UpdateCount int                                            `json:"update_count,omitempty"`
	RunQueue    []string                                       `json:"run_queue,omitempty"`
	RunTickets  map[string]MobileRunnerSemaphoreRunTicketState `json:"run_tickets,omitempty"`
}

type MobileRunnerSemaphoreAcquireRequest struct {
	RequestID       string        `json:"request_id"`
	LeaseID         string        `json:"lease_id"`
	OwnerNamespace  string        `json:"owner_namespace,omitempty"`
	OwnerWorkflowID string        `json:"owner_workflow_id,omitempty"`
	OwnerRunID      string        `json:"owner_run_id,omitempty"`
	WaitTimeout     time.Duration `json:"wait_timeout,omitempty"`
}

type MobileRunnerSemaphoreReleaseRequest struct {
	LeaseID string `json:"lease_id"`
}

type MobileRunnerSemaphorePermit struct {
	RunnerID    string    `json:"runner_id"`
	LeaseID     string    `json:"lease_id"`
	GrantedAt   time.Time `json:"granted_at"`
	QueueWaitMs int64     `json:"queue_wait_ms"`
}

type MobileRunnerSemaphoreReleaseResult struct {
	Released bool `json:"released"`
}

type MobileRunnerSemaphoreHolder struct {
	LeaseID         string    `json:"lease_id"`
	RequestID       string    `json:"request_id"`
	OwnerNamespace  string    `json:"owner_namespace,omitempty"`
	OwnerWorkflowID string    `json:"owner_workflow_id,omitempty"`
	OwnerRunID      string    `json:"owner_run_id,omitempty"`
	GrantedAt       time.Time `json:"granted_at"`
	QueueWaitMs     int64     `json:"queue_wait_ms"`
}

type MobileRunnerSemaphoreQueueEntry struct {
	RequestID       string    `json:"request_id"`
	LeaseID         string    `json:"lease_id"`
	OwnerNamespace  string    `json:"owner_namespace,omitempty"`
	OwnerWorkflowID string    `json:"owner_workflow_id,omitempty"`
	OwnerRunID      string    `json:"owner_run_id,omitempty"`
	RequestedAt     time.Time `json:"requested_at"`
}

type MobileRunnerSemaphoreStateView struct {
	RunnerID      string                            `json:"runner_id"`
	Capacity      int                               `json:"capacity"`
	CurrentHolder *MobileRunnerSemaphoreHolder      `json:"current_holder,omitempty"`
	Holders       []MobileRunnerSemaphoreHolder     `json:"holders"`
	QueueLen      int                               `json:"queue_len"`
	QueuePreview  []MobileRunnerSemaphoreQueueEntry `json:"queue_preview"`
	LastGrantAt   *time.Time                        `json:"last_grant_at,omitempty"`
}

type MobileRunnerSemaphoreRequestStatus string

const (
	MobileRunnerSemaphoreRequestQueued   MobileRunnerSemaphoreRequestStatus = "queued"
	MobileRunnerSemaphoreRequestGranted  MobileRunnerSemaphoreRequestStatus = "granted"
	MobileRunnerSemaphoreRequestTimedOut MobileRunnerSemaphoreRequestStatus = "timed_out"
)

type MobileRunnerSemaphoreRequestState struct {
	Request     MobileRunnerSemaphoreAcquireRequest `json:"request"`
	Status      MobileRunnerSemaphoreRequestStatus  `json:"status"`
	RequestedAt time.Time                           `json:"requested_at"`
	GrantedAt   time.Time                           `json:"granted_at,omitempty"`
	QueueWaitMs int64                               `json:"queue_wait_ms,omitempty"`
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
	TicketID           string         `json:"ticket_id"`
	OwnerNamespace     string         `json:"owner_namespace"`
	EnqueuedAt         time.Time      `json:"enqueued_at"`
	RunnerID           string         `json:"runner_id"`
	RequiredRunnerIDs  []string       `json:"required_runner_ids"`
	LeaderRunnerID     string         `json:"leader_runner_id"`
	MaxPipelinesInQueue int           `json:"max_pipelines_in_queue,omitempty"`
	PipelineIdentifier string         `json:"pipeline_identifier,omitempty"`
	YAML               string         `json:"yaml,omitempty"`
	PipelineConfig     map[string]any `json:"pipeline_config,omitempty"`
	Memo               map[string]any `json:"memo,omitempty"`
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

func PermitLeaseID(workflowID, runID, runnerID string) string {
	return fmt.Sprintf("%s/%s/%s", workflowID, runID, runnerID)
}

func AcquireUpdateID(requestID string) string {
	return acquireUpdateIDPrefix + requestID
}

func ReleaseUpdateID(leaseID string) string {
	return releaseUpdateIDPrefix + leaseID
}
