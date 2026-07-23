// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package mobiledevicesemaphore

import (
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
)

const (
	TaskQueue    = "mobile-device-semaphore-task-queue"
	WorkflowName = "mobile-device-semaphore"
	StateQuery   = "GetState"

	ErrInvalidRequest     = "mobile-device-semaphore-invalid-request"
	ErrQueueLimitExceeded = "mobile-device-semaphore-queue-limit-exceeded"
)

const (
	EnqueueRunUpdate     = "EnqueueRun"
	RunStatusQuery       = "GetRunStatus"
	ListQueuedRunsQuery  = "ListQueuedRuns"
	RunDoneUpdate        = "RunDone"
	CancelRunUpdate      = "CancelRun"
	PauseDeviceUpdate    = "PauseDevice"
	ResumeDeviceUpdate   = "ResumeDevice"
	ShutdownDeviceUpdate = "ShutdownDevice"

	RunGrantedSignal = "RunGranted"
	RunStartedSignal = "RunStarted"
	RunDoneSignal    = "RunDoneSignal"
)

type MobileDeviceSemaphoreWorkflowInput struct {
	DeviceID string                              `json:"device_id"`
	Capacity int                                 `json:"capacity"`
	State    *MobileDeviceSemaphoreWorkflowState `json:"state,omitempty"`
}

type MobileDeviceSemaphoreWorkflowState struct {
	Capacity             int                                            `json:"capacity"`
	UpdateCount          int                                            `json:"update_count,omitempty"`
	RunQueue             []string                                       `json:"run_queue,omitempty"`
	RunTickets           map[string]MobileDeviceSemaphoreRunTicketState `json:"run_tickets,omitempty"`
	Paused               bool                                           `json:"paused,omitempty"`
	PausedAt             time.Time                                      `json:"paused_at,omitempty"`
	PauseReason          string                                         `json:"pause_reason,omitempty"`
	PauseGeneration      int                                            `json:"pause_generation,omitempty"`
	ShutdownAfterSeconds int                                            `json:"shutdown_after_seconds,omitempty"`
}

type MobileDeviceSemaphoreStateView struct {
	DeviceID             string    `json:"device_id"`
	Capacity             int       `json:"capacity"`
	SlotsUsed            int       `json:"slots_used"`
	QueueLen             int       `json:"queue_len"`
	Paused               bool      `json:"paused,omitempty"`
	PausedAt             time.Time `json:"paused_at,omitempty"`
	PauseReason          string    `json:"pause_reason,omitempty"`
	PauseGeneration      int       `json:"pause_generation,omitempty"`
	ShutdownAfterSeconds int       `json:"shutdown_after_seconds,omitempty"`
}

type MobileDeviceSemaphoreRunStatus string

const (
	MobileDeviceSemaphoreRunQueued   MobileDeviceSemaphoreRunStatus = "queued"
	MobileDeviceSemaphoreRunStarting MobileDeviceSemaphoreRunStatus = "starting"
	MobileDeviceSemaphoreRunRunning  MobileDeviceSemaphoreRunStatus = "running"
	MobileDeviceSemaphoreRunFailed   MobileDeviceSemaphoreRunStatus = "failed"
	MobileDeviceSemaphoreRunCanceled MobileDeviceSemaphoreRunStatus = "canceled"
	MobileDeviceSemaphoreRunNotFound MobileDeviceSemaphoreRunStatus = "not_found"
)

type MobileDeviceSemaphoreEnqueueRunRequest struct {
	TicketID            string                                `json:"ticket_id"`
	OwnerNamespace      string                                `json:"owner_namespace"`
	EnqueuedAt          time.Time                             `json:"enqueued_at"`
	DeviceID            string                                `json:"device_id"`
	RequiredDeviceIDs   []string                              `json:"required_device_ids"`
	LeaderDeviceID      string                                `json:"leader_device_id"`
	MaxPipelinesInQueue int                                   `json:"max_pipelines_in_queue,omitempty"`
	PipelineIdentifier  string                                `json:"pipeline_identifier,omitempty"`
	YAML                string                                `json:"yaml,omitempty"`
	PipelineConfig      map[string]any                        `json:"pipeline_config,omitempty"`
	Memo                map[string]any                        `json:"memo,omitempty"`
	Cleanup             *MobileDeviceSemaphoreCleanupMetadata `json:"cleanup,omitempty"`
	Notification        *MobileDeviceSemaphoreNotification    `json:"notification,omitempty"`
}

// MobileDeviceSemaphoreCleanupMetadata carries resources owned by a queued run
// that must be cleaned if the ticket is canceled before the workflow starts.
type MobileDeviceSemaphoreCleanupMetadata struct {
	TempWalletVersionID         string                                               `json:"temp_wallet_version_id,omitempty"`
	TempWalletVersionOwnerID    string                                               `json:"temp_wallet_version_owner_id,omitempty"`
	TempWalletVersionIdentifier string                                               `json:"temp_wallet_version_identifier,omitempty"`
	TempCredentials             []MobileDeviceSemaphoreTempCredentialCleanupMetadata `json:"temp_credentials,omitempty"`
	TempUseCaseVerifications    []MobileDeviceSemaphoreTempCredentialCleanupMetadata `json:"temp_use_case_verifications,omitempty"`
}

type MobileDeviceSemaphoreTempCredentialCleanupMetadata struct {
	RecordID   string `json:"record_id,omitempty"`
	OwnerID    string `json:"owner_id,omitempty"`
	Identifier string `json:"identifier,omitempty"`
}

type MobileDeviceSemaphoreNotification struct {
	GitHubPR *MobileDeviceSemaphoreGitHubPRNotification `json:"github_pr,omitempty"`
}

type MobileDeviceSemaphoreGitHubPRNotification struct {
	Repository         string            `json:"repository,omitempty"`
	PullRequestNumber  int               `json:"pull_request_number,omitempty"`
	CommitSHA          string            `json:"commit_sha,omitempty"`
	PipelineIdentifier string            `json:"pipeline_identifier,omitempty"`
	DeviceID           string            `json:"runner_id,omitempty"`   // Deprecated: use RunnerTypes for per-runner display metadata.
	RunnerType         string            `json:"runner_type,omitempty"` // Deprecated: use RunnerTypes for per-runner display metadata.
	RunnerTypes        map[string]string `json:"runner_types,omitempty"`
	PipelineURL        string            `json:"pipeline_url,omitempty"`
	AppURL             string            `json:"app_url,omitempty"`
	SectionTitle       string            `json:"section_title,omitempty"`
}

type MobileDeviceSemaphoreEnqueueRunResponse struct {
	TicketID string                         `json:"ticket_id"`
	Status   MobileDeviceSemaphoreRunStatus `json:"status"`
	Position int                            `json:"position"`
	LineLen  int                            `json:"line_len"`
}

type MobileDeviceSemaphoreRunStatusView struct {
	TicketID          string                                `json:"ticket_id"`
	Status            MobileDeviceSemaphoreRunStatus        `json:"status"`
	Position          int                                   `json:"position"`
	LineLen           int                                   `json:"line_len"`
	LeaderDeviceID    string                                `json:"leader_device_id,omitempty"`
	RequiredDeviceIDs []string                              `json:"required_device_ids,omitempty"`
	WorkflowID        string                                `json:"workflow_id,omitempty"`
	RunID             string                                `json:"run_id,omitempty"`
	WorkflowNamespace string                                `json:"workflow_namespace,omitempty"`
	ErrorMessage      string                                `json:"error_message,omitempty"`
	Cleanup           *MobileDeviceSemaphoreCleanupMetadata `json:"cleanup,omitempty"`
}

type MobileDeviceSemaphoreQueuedRunView struct {
	TicketID           string                                `json:"ticket_id"`
	OwnerNamespace     string                                `json:"owner_namespace"`
	PipelineIdentifier string                                `json:"pipeline_identifier,omitempty"`
	EnqueuedAt         time.Time                             `json:"enqueued_at"`
	LeaderDeviceID     string                                `json:"leader_device_id,omitempty"`
	RequiredDeviceIDs  []string                              `json:"required_device_ids,omitempty"`
	Status             MobileDeviceSemaphoreRunStatus        `json:"status"`
	Position           int                                   `json:"position"`
	LineLen            int                                   `json:"line_len"`
	Cleanup            *MobileDeviceSemaphoreCleanupMetadata `json:"cleanup,omitempty"`
}

type MobileDeviceSemaphoreRunDoneRequest struct {
	TicketID       string `json:"ticket_id"`
	OwnerNamespace string `json:"owner_namespace,omitempty"`
	WorkflowID     string `json:"workflow_id,omitempty"`
	RunID          string `json:"run_id,omitempty"`
	WorkflowResult string `json:"workflow_result,omitempty"`
}

type MobileDeviceSemaphoreRunCancelRequest struct {
	TicketID       string `json:"ticket_id"`
	OwnerNamespace string `json:"owner_namespace,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

type MobileDeviceSemaphoreShutdownDeviceRequest struct {
	Reason string `json:"reason,omitempty"`
}

type MobileDeviceSemaphorePauseDeviceRequest struct {
	Reason               string `json:"reason,omitempty"`
	CancelRunning        bool   `json:"cancel_running"`
	ShutdownAfterSeconds int    `json:"shutdown_after_seconds,omitempty"`
}

type MobileDeviceSemaphorePauseDeviceResponse struct {
	DeviceID                 string   `json:"device_id"`
	Paused                   bool     `json:"paused"`
	RunningPipelinesCanceled int      `json:"running_pipelines_canceled"`
	PipelineCancelFailures   []string `json:"pipeline_cancel_failures,omitempty"`
	ShutdownAfterSeconds     int      `json:"shutdown_after_seconds,omitempty"`
}

type MobileDeviceSemaphoreResumeDeviceRequest struct {
	Reason string `json:"reason,omitempty"`
}

type MobileDeviceSemaphoreResumeDeviceResponse struct {
	DeviceID string `json:"device_id"`
	Paused   bool   `json:"paused"`
	QueueLen int    `json:"queue_len"`
}

type MobileDeviceSemaphoreShutdownDeviceResponse struct {
	DeviceID                 string   `json:"device_id"`
	QueuedCanceled           int      `json:"queued_canceled"`
	StartingCanceled         int      `json:"starting_canceled"`
	RunningPipelinesCanceled int      `json:"running_pipelines_canceled"`
	FollowerSignalsSent      int      `json:"follower_signals_sent"`
	CleanupFailures          []string `json:"cleanup_failures,omitempty"`
	PipelineCancelFailures   []string `json:"pipeline_cancel_failures,omitempty"`
	FollowerSignalFailures   []string `json:"follower_signal_failures,omitempty"`
}

type MobileDeviceSemaphoreRunGrantedSignal struct {
	TicketID string `json:"ticket_id"`
	DeviceID string `json:"device_id"`
}

type MobileDeviceSemaphoreRunStartedSignal struct {
	TicketID          string `json:"ticket_id"`
	WorkflowID        string `json:"workflow_id"`
	RunID             string `json:"run_id"`
	WorkflowNamespace string `json:"workflow_namespace"`
}

type MobileDeviceSemaphoreRunDoneSignal struct {
	TicketID       string `json:"ticket_id"`
	WorkflowID     string `json:"workflow_id,omitempty"`
	RunID          string `json:"run_id,omitempty"`
	WorkflowResult string `json:"workflow_result,omitempty"`
}

type MobileDeviceSemaphoreRunTicketState struct {
	Request           MobileDeviceSemaphoreEnqueueRunRequest `json:"request"`
	Status            MobileDeviceSemaphoreRunStatus         `json:"status"`
	WorkflowID        string                                 `json:"workflow_id,omitempty"`
	RunID             string                                 `json:"run_id,omitempty"`
	WorkflowNamespace string                                 `json:"workflow_namespace,omitempty"`
	ErrorMessage      string                                 `json:"error_message,omitempty"`
	CancelRequested   bool                                   `json:"cancel_requested,omitempty"`
	GrantedDeviceIDs  map[string]bool                        `json:"granted_device_ids,omitempty"`
	StartedAt         *time.Time                             `json:"started_at,omitempty"`
	DoneAt            *time.Time                             `json:"done_at,omitempty"`
}

func WorkflowID(deviceID string) string {
	deviceID = canonify.NormalizePath(deviceID)
	return fmt.Sprintf("mobile-device-semaphore/%s", deviceID)
}
