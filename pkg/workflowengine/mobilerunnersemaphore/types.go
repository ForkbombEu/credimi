// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package mobilerunnersemaphore

import (
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
)

const (
	TaskQueue    = "mobile-runner-semaphore-task-queue"
	WorkflowName = "mobile-runner-semaphore"
	StateQuery   = "GetState"

	ErrInvalidRequest     = "mobile-runner-semaphore-invalid-request"
	ErrQueueLimitExceeded = "mobile-runner-semaphore-queue-limit-exceeded"
)

const (
	EnqueueRunUpdate     = "EnqueueRun"
	RunStatusQuery       = "GetRunStatus"
	ListQueuedRunsQuery  = "ListQueuedRuns"
	RunDoneUpdate        = "RunDone"
	CancelRunUpdate      = "CancelRun"
	ShutdownRunnerUpdate = "ShutdownRunner"

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
	TicketID            string                                `json:"ticket_id"`
	OwnerNamespace      string                                `json:"owner_namespace"`
	EnqueuedAt          time.Time                             `json:"enqueued_at"`
	RunnerID            string                                `json:"runner_id"`
	RequiredRunnerIDs   []string                              `json:"required_runner_ids"`
	LeaderRunnerID      string                                `json:"leader_runner_id"`
	MaxPipelinesInQueue int                                   `json:"max_pipelines_in_queue,omitempty"`
	PipelineIdentifier  string                                `json:"pipeline_identifier,omitempty"`
	YAML                string                                `json:"yaml,omitempty"`
	PipelineConfig      map[string]any                        `json:"pipeline_config,omitempty"`
	Memo                map[string]any                        `json:"memo,omitempty"`
	Cleanup             *MobileRunnerSemaphoreCleanupMetadata `json:"cleanup,omitempty"`
	Notification        *MobileRunnerSemaphoreNotification    `json:"notification,omitempty"`
}

// MobileRunnerSemaphoreCleanupMetadata carries resources owned by a queued run
// that must be cleaned if the ticket is canceled before the workflow starts.
type MobileRunnerSemaphoreCleanupMetadata struct {
	TempWalletVersionID         string                                               `json:"temp_wallet_version_id,omitempty"`
	TempWalletVersionOwnerID    string                                               `json:"temp_wallet_version_owner_id,omitempty"`
	TempWalletVersionIdentifier string                                               `json:"temp_wallet_version_identifier,omitempty"`
	TempCredentials             []MobileRunnerSemaphoreTempCredentialCleanupMetadata `json:"temp_credentials,omitempty"`
	TempUseCaseVerifications    []MobileRunnerSemaphoreTempCredentialCleanupMetadata `json:"temp_use_case_verifications,omitempty"`
}

type MobileRunnerSemaphoreTempCredentialCleanupMetadata struct {
	RecordID   string `json:"record_id,omitempty"`
	OwnerID    string `json:"owner_id,omitempty"`
	Identifier string `json:"identifier,omitempty"`
}

type MobileRunnerSemaphoreNotification struct {
	GitHubPR *MobileRunnerSemaphoreGitHubPRNotification `json:"github_pr,omitempty"`
}

type MobileRunnerSemaphoreGitHubPRNotification struct {
	Repository         string            `json:"repository,omitempty"`
	PullRequestNumber  int               `json:"pull_request_number,omitempty"`
	CommitSHA          string            `json:"commit_sha,omitempty"`
	PipelineIdentifier string            `json:"pipeline_identifier,omitempty"`
	RunnerID           string            `json:"runner_id,omitempty"`   // Deprecated: use RunnerTypes for per-runner display metadata.
	RunnerType         string            `json:"runner_type,omitempty"` // Deprecated: use RunnerTypes for per-runner display metadata.
	RunnerTypes        map[string]string `json:"runner_types,omitempty"`
	PipelineURL        string            `json:"pipeline_url,omitempty"`
	AppURL             string            `json:"app_url,omitempty"`
	SectionTitle       string            `json:"section_title,omitempty"`
}

type MobileRunnerSemaphoreEnqueueRunResponse struct {
	TicketID string                         `json:"ticket_id"`
	Status   MobileRunnerSemaphoreRunStatus `json:"status"`
	Position int                            `json:"position"`
	LineLen  int                            `json:"line_len"`
}

type MobileRunnerSemaphoreRunStatusView struct {
	TicketID          string                                `json:"ticket_id"`
	Status            MobileRunnerSemaphoreRunStatus        `json:"status"`
	Position          int                                   `json:"position"`
	LineLen           int                                   `json:"line_len"`
	LeaderRunnerID    string                                `json:"leader_runner_id,omitempty"`
	RequiredRunnerIDs []string                              `json:"required_runner_ids,omitempty"`
	WorkflowID        string                                `json:"workflow_id,omitempty"`
	RunID             string                                `json:"run_id,omitempty"`
	WorkflowNamespace string                                `json:"workflow_namespace,omitempty"`
	ErrorMessage      string                                `json:"error_message,omitempty"`
	Cleanup           *MobileRunnerSemaphoreCleanupMetadata `json:"cleanup,omitempty"`
}

type MobileRunnerSemaphoreQueuedRunView struct {
	TicketID           string                                `json:"ticket_id"`
	OwnerNamespace     string                                `json:"owner_namespace"`
	PipelineIdentifier string                                `json:"pipeline_identifier,omitempty"`
	EnqueuedAt         time.Time                             `json:"enqueued_at"`
	LeaderRunnerID     string                                `json:"leader_runner_id,omitempty"`
	RequiredRunnerIDs  []string                              `json:"required_runner_ids,omitempty"`
	Status             MobileRunnerSemaphoreRunStatus        `json:"status"`
	Position           int                                   `json:"position"`
	LineLen            int                                   `json:"line_len"`
	Cleanup            *MobileRunnerSemaphoreCleanupMetadata `json:"cleanup,omitempty"`
}

type MobileRunnerSemaphoreRunDoneRequest struct {
	TicketID       string `json:"ticket_id"`
	OwnerNamespace string `json:"owner_namespace,omitempty"`
	WorkflowID     string `json:"workflow_id,omitempty"`
	RunID          string `json:"run_id,omitempty"`
	WorkflowResult string `json:"workflow_result,omitempty"`
}

type MobileRunnerSemaphoreRunCancelRequest struct {
	TicketID       string `json:"ticket_id"`
	OwnerNamespace string `json:"owner_namespace,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

type MobileRunnerSemaphoreShutdownRunnerRequest struct {
	Reason string `json:"reason,omitempty"`
}

type MobileRunnerSemaphoreShutdownRunnerResponse struct {
	RunnerID                 string   `json:"runner_id"`
	QueuedCanceled           int      `json:"queued_canceled"`
	StartingCanceled         int      `json:"starting_canceled"`
	RunningPipelinesCanceled int      `json:"running_pipelines_canceled"`
	FollowerSignalsSent      int      `json:"follower_signals_sent"`
	CleanupFailures          []string `json:"cleanup_failures,omitempty"`
	PipelineCancelFailures   []string `json:"pipeline_cancel_failures,omitempty"`
	FollowerSignalFailures   []string `json:"follower_signal_failures,omitempty"`
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
	TicketID       string `json:"ticket_id"`
	WorkflowID     string `json:"workflow_id,omitempty"`
	RunID          string `json:"run_id,omitempty"`
	WorkflowResult string `json:"workflow_result,omitempty"`
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
	runnerID = canonify.NormalizePath(runnerID)
	return fmt.Sprintf("mobile-runner-semaphore/%s", runnerID)
}
