// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package mobiledevicesession holds the names and types shared by the pipeline
// workflow, which runs in an organization namespace, and the device workflows
// it starts in a runner namespace through Credimi activities.
package mobiledevicesession

import "github.com/forkbombeu/credimi/pkg/workflowengine"

const (
	// WorkflowName is the workflow that holds one device for one pipeline run.
	WorkflowName = "Run a mobile device session"
	// MobileAutomationWorkflowName is the workflow of one mobile-automation step.
	MobileAutomationWorkflowName = "Run a mobile automation workflow"
	// ReadyUpdate answers once the device is prepared, or with the setup error.
	ReadyUpdate = "device_session_ready"
	// ReleaseUpdate cleans the device and ends the session.
	ReleaseUpdate = "device_session_release"
	// PipelineRunMemoKey is the memo key naming the pipeline run a runner
	// namespace workflow belongs to.
	PipelineRunMemoKey = "credimi_pipeline_run"
	// RunRunnerWorkflowActivityName runs a step's workflow in a runner
	// namespace and returns its result.
	RunRunnerWorkflowActivityName = "Run a workflow in a runner namespace"
	// StartDeviceSessionActivityName starts the device session of one device.
	StartDeviceSessionActivityName = "Start a mobile device session"
)

// RunRunnerWorkflowPayload starts a workflow in a runner namespace on behalf of
// a pipeline run and waits for its result.
type RunRunnerWorkflowPayload struct {
	Namespace    string                       `json:"namespace"`
	WorkflowID   string                       `json:"workflow_id"`
	WorkflowName string                       `json:"workflow_name"`
	Input        workflowengine.WorkflowInput `json:"input"`
	Memo         map[string]any               `json:"memo,omitempty"`
	Summary      string                       `json:"summary,omitempty"`
	Pipeline     PipelineRunRef               `json:"pipeline"`
}

// RunRunnerWorkflowResult is the output of RunRunnerWorkflowActivityName.
type RunRunnerWorkflowResult struct {
	Namespace  string                        `json:"namespace"`
	WorkflowID string                        `json:"workflow_id"`
	RunID      string                        `json:"run_id"`
	Result     workflowengine.WorkflowResult `json:"result"`
}

// StartDeviceSessionPayload starts the device session of one device of a
// pipeline run.
type StartDeviceSessionPayload struct {
	Namespace  string                       `json:"namespace"`
	WorkflowID string                       `json:"workflow_id"`
	DeviceID   string                       `json:"device_id"`
	Input      workflowengine.WorkflowInput `json:"input"`
	Pipeline   PipelineRunRef               `json:"pipeline"`
}

// PipelineRunRef names a pipeline run in its organization namespace.
type PipelineRunRef struct {
	Namespace  string `json:"namespace"`
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

// RunnerWorkflowRef names a workflow run in a runner namespace.
type RunnerWorkflowRef struct {
	Namespace  string `json:"namespace"`
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

// Release is the argument of ReleaseUpdate.
type Release struct {
	SkipCleanup bool `json:"skip_cleanup"`
}

// Cleanup is what ReleaseUpdate returns: the evidence the device cleanup
// stored and what went wrong.
type Cleanup struct {
	ResultVideoURLs []string `json:"result_video_urls,omitempty"`
	ScreenshotURLs  []string `json:"screenshot_urls,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	Errors          []string `json:"errors,omitempty"`
}
