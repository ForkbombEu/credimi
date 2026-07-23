// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
)

// EnqueuePipelineRunTicketActivityName identifies the enqueue run ticket activity.
const EnqueuePipelineRunTicketActivityName = "Enqueue pipeline run ticket"

// EnqueuePipelineRunTicketActivityInput defines the payload for enqueuing a pipeline run ticket.
type EnqueuePipelineRunTicketActivityInput struct {
	TicketID            string         `json:"ticket_id"`
	OwnerNamespace      string         `json:"owner_namespace"`
	EnqueuedAt          time.Time      `json:"enqueued_at"`
	DeviceIDs           []string       `json:"device_ids"`
	PipelineIdentifier  string         `json:"pipeline_identifier"`
	YAML                string         `json:"yaml"`
	PipelineConfig      map[string]any `json:"pipeline_config,omitempty"`
	Memo                map[string]any `json:"memo,omitempty"`
	MaxPipelinesInQueue int            `json:"max_pipelines_in_queue,omitempty"`
}

// EnqueuePipelineRunTicketDeviceStatus describes enqueue responses per runner.
type EnqueuePipelineRunTicketDeviceStatus struct {
	DeviceID          string                                               `json:"device_id"`
	Status            mobiledevicesemaphore.MobileDeviceSemaphoreRunStatus `json:"status"`
	Position          int                                                  `json:"position"`
	LineLen           int                                                  `json:"line_len"`
	WorkflowID        string                                               `json:"workflow_id,omitempty"`
	RunID             string                                               `json:"run_id,omitempty"`
	WorkflowNamespace string                                               `json:"workflow_namespace,omitempty"`
	ErrorMessage      string                                               `json:"error_message,omitempty"`
}

// EnqueuePipelineRunTicketActivityOutput aggregates enqueue status across runners.
type EnqueuePipelineRunTicketActivityOutput struct {
	Status            mobiledevicesemaphore.MobileDeviceSemaphoreRunStatus `json:"status"`
	Position          int                                                  `json:"position"`
	LineLen           int                                                  `json:"line_len"`
	WorkflowID        string                                               `json:"workflow_id,omitempty"`
	RunID             string                                               `json:"run_id,omitempty"`
	WorkflowNamespace string                                               `json:"workflow_namespace,omitempty"`
	ErrorMessage      string                                               `json:"error_message,omitempty"`
	Runners           []EnqueuePipelineRunTicketDeviceStatus               `json:"runners"`
}
