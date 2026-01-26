// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import "time"

const (
	PipelineStateQuery   = "GetPipelineState"
	ResourceUsageQuery   = "GetResourceUsage"
	ForceCleanupSignal   = "ForceCleanup"
	PauseRecordingSignal = "PauseRecording"
)

// PipelineState exposes high-level workflow state for debugging.
type PipelineState struct {
	WorkflowID       string    `json:"workflow_id"`
	RunID            string    `json:"run_id"`
	EmulatorSerial   string    `json:"emulator_serial,omitempty"`
	CloneName        string    `json:"clone_name,omitempty"`
	VersionID        string    `json:"version_id,omitempty"`
	RecordingActive  bool      `json:"recording_active"`
	RecordingPaused  bool      `json:"recording_paused"`
	BootStatus       string    `json:"boot_status,omitempty"`
	LastActivity     string    `json:"last_activity,omitempty"`
	LastActivityTime time.Time `json:"last_activity_time,omitempty"`
	Status           string    `json:"status,omitempty"`
	ForceCleanup     bool      `json:"force_cleanup"`
}

// ResourceUsage provides resource-related metrics for debugging.
type ResourceUsage struct {
	PoolSlotAcquired   bool  `json:"pool_slot_acquired"`
	PoolWaitTimeMs     int64 `json:"pool_wait_time_ms"`
	EmulatorBootTimeMs int64 `json:"emulator_boot_time_ms"`
	VideoSizeBytes     int64 `json:"video_size_bytes"`
	MaestroStepCount   int   `json:"maestro_step_count"`
}
