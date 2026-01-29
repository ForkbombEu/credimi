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

type MobileRunnerSemaphoreWorkflowInput struct {
	RunnerID string                              `json:"runner_id"`
	Capacity int                                 `json:"capacity"`
	State    *MobileRunnerSemaphoreWorkflowState `json:"state,omitempty"`
}

type MobileRunnerSemaphoreWorkflowState struct {
	Capacity    int                                          `json:"capacity"`
	Holders     map[string]MobileRunnerSemaphoreHolder       `json:"holders,omitempty"`
	Queue       []string                                     `json:"queue,omitempty"`
	Requests    map[string]MobileRunnerSemaphoreRequestState `json:"requests,omitempty"`
	LastGrantAt *time.Time                                   `json:"last_grant_at,omitempty"`
	UpdateCount int                                          `json:"update_count,omitempty"`
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
	LeaseID        string    `json:"lease_id"`
	RequestID      string    `json:"request_id"`
	OwnerNamespace string    `json:"owner_namespace,omitempty"`
	OwnerWorkflowID string   `json:"owner_workflow_id,omitempty"`
	OwnerRunID     string    `json:"owner_run_id,omitempty"`
	GrantedAt      time.Time `json:"granted_at"`
	QueueWaitMs    int64     `json:"queue_wait_ms"`
}

type MobileRunnerSemaphoreQueueEntry struct {
	RequestID      string    `json:"request_id"`
	LeaseID        string    `json:"lease_id"`
	OwnerNamespace string    `json:"owner_namespace,omitempty"`
	OwnerWorkflowID string   `json:"owner_workflow_id,omitempty"`
	OwnerRunID     string    `json:"owner_run_id,omitempty"`
	RequestedAt    time.Time `json:"requested_at"`
}

type MobileRunnerSemaphoreStateView struct {
	RunnerID      string                            `json:"runner_id"`
	Capacity      int                               `json:"capacity"`
	CurrentHolder *MobileRunnerSemaphoreHolder      `json:"current_holder,omitempty"`
	Holders       []MobileRunnerSemaphoreHolder      `json:"holders"`
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

func WorkflowID(runnerID string) string {
	return fmt.Sprintf("mobile-runner-semaphore/%s", runnerID)
}

func PermitLeaseID(workflowID, runID, runnerID string) string {
	return fmt.Sprintf("%s/%s/%s", workflowID, runID, runnerID)
}
