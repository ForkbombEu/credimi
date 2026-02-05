// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import "time"

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
