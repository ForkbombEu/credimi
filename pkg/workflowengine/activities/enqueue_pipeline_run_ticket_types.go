// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
)

// EnqueuePipelineRunTicketActivityName identifies the enqueue run ticket activity.
const EnqueuePipelineRunTicketActivityName = "Enqueue pipeline run ticket"

// EnqueuePipelineRunTicketActivityInput defines the payload for enqueuing a pipeline run ticket.
type EnqueuePipelineRunTicketActivityInput struct {
	TicketID            string         `json:"ticket_id"`
	OwnerNamespace      string         `json:"owner_namespace"`
	EnqueuedAt          time.Time      `json:"enqueued_at"`
	RunnerIDs           []string       `json:"runner_ids"`
	PipelineIdentifier  string         `json:"pipeline_identifier"`
	YAML                string         `json:"yaml"`
	PipelineConfig      map[string]any `json:"pipeline_config,omitempty"`
	Memo                map[string]any `json:"memo,omitempty"`
	MaxPipelinesInQueue int            `json:"max_pipelines_in_queue,omitempty"`
}

// EnqueuePipelineRunTicketRunnerStatus describes enqueue responses per runner.
type EnqueuePipelineRunTicketRunnerStatus struct {
	RunnerID          string                                               `json:"runner_id"`
	Status            mobilerunnersemaphore.MobileRunnerSemaphoreRunStatus `json:"status"`
	Position          int                                                  `json:"position"`
	LineLen           int                                                  `json:"line_len"`
	WorkflowID        string                                               `json:"workflow_id,omitempty"`
	RunID             string                                               `json:"run_id,omitempty"`
	WorkflowNamespace string                                               `json:"workflow_namespace,omitempty"`
	ErrorMessage      string                                               `json:"error_message,omitempty"`
}

// EnqueuePipelineRunTicketActivityOutput aggregates enqueue status across runners.
type EnqueuePipelineRunTicketActivityOutput struct {
	Status            mobilerunnersemaphore.MobileRunnerSemaphoreRunStatus `json:"status"`
	Position          int                                                  `json:"position"`
	LineLen           int                                                  `json:"line_len"`
	WorkflowID        string                                               `json:"workflow_id,omitempty"`
	RunID             string                                               `json:"run_id,omitempty"`
	WorkflowNamespace string                                               `json:"workflow_namespace,omitempty"`
	ErrorMessage      string                                               `json:"error_message,omitempty"`
	Runners           []EnqueuePipelineRunTicketRunnerStatus               `json:"runners"`
}
