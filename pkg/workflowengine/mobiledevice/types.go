// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package mobiledevice holds the names and types of the Nexus service through
// which a pipeline, in its organization namespace, runs its device work in the
// namespace of each device's runner: preparing the device, running each
// mobile-automation step and cleaning the device.
package mobiledevice

import (
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/temporalid"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

const (
	// ServiceName is the Nexus service every runner namespace serves.
	ServiceName = "mobile-device"

	// PrepareOperationName starts PrepareWorkflowName.
	PrepareOperationName = "Prepare a mobile device"
	// PrepareWorkflowName prepares one device for a pipeline run.
	PrepareWorkflowName = "Prepare a mobile device"
	// RunStepOperationName starts StepWorkflowName.
	RunStepOperationName = "Run a mobile automation step"
	// StepWorkflowName runs one mobile-automation step.
	StepWorkflowName = "Run a mobile automation workflow"
	// CleanUpOperationName starts CleanUpWorkflowName.
	CleanUpOperationName = "Clean up a mobile device"
	// CleanUpWorkflowName cleans one device after a pipeline run.
	CleanUpWorkflowName = "Clean up a mobile device"

	// PipelineRunMemoKey is the memo key naming the pipeline run a runner
	// namespace workflow belongs to.
	PipelineRunMemoKey = "credimi_pipeline_run"
)

// PrepareWorkflowID is the ID of the workflow that prepares deviceID for the
// pipeline run pipelineWorkflowID.
func PrepareWorkflowID(pipelineWorkflowID, deviceID string) string {
	return deviceWorkflowID(pipelineWorkflowID, deviceID, "prepare")
}

// CleanUpWorkflowID is the ID of the workflow that cleans deviceID after the
// pipeline run pipelineWorkflowID.
func CleanUpWorkflowID(pipelineWorkflowID, deviceID string) string {
	return deviceWorkflowID(pipelineWorkflowID, deviceID, "cleanup")
}

func deviceWorkflowID(pipelineWorkflowID, deviceID, kind string) string {
	return temporalid.Bounded(
		pipelineWorkflowID + "-device-" + canonify.CanonifyPlain(deviceID) + "-" + kind,
	)
}

// PipelineRun names the pipeline run that owns a runner-namespace workflow.
type PipelineRun struct {
	// Namespace is the pipeline's organization namespace.
	Namespace  string `json:"namespace"`
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

// MobileDeviceRequest names one device to prepare for a run.
type MobileDeviceRequest struct {
	DeviceID          string `json:"device_id"`          // canonified device path
	VersionIdentifier string `json:"version_identifier"` // wallet version identifier to install
	StepID            string `json:"step_id"`            // label used in errors and run identifiers
	RunIdentifier     string `json:"run_identifier"`
	// PackageID is the package of an app installed outside Credimi, with
	// VersionIdentifier MobileExternalSourceVersionID. Its `installed` entry
	// only provides PreparedMobileDevice.PackageID: cleanup removes the app by
	// the diff against the initial_installed_apps recorded at prepare.
	PackageID string `json:"package_id,omitempty"`
}

// PreparedMobileDevice is what a run needs to drive the prepared device.
type PreparedMobileDevice struct {
	RunnerID string `json:"runner_id"`
	// Serial is the adb serial, or the UDID of the run's simulator clone.
	Serial string `json:"serial"`
	Type   string `json:"type"`
	// PackageID is the Android package or the iOS bundle ID.
	PackageID string `json:"package_id"`
	// InstallerPath is the runner-host path of the installed APK or .app.
	InstallerPath string `json:"installer_path"`
}

// PrepareInput prepares one device for every mobile step of a pipeline run
// that uses it.
type PrepareInput struct {
	WorkflowID       string                `json:"workflow_id"`
	DeviceID         string                `json:"device_id"`
	Requests         []MobileDeviceRequest `json:"requests"`
	RunIdentifier    string                `json:"run_identifier"`
	DisablePlayStore bool                  `json:"disable_play_store"`
	NoVideo          bool                  `json:"no_video"`
	ExecutionTimeout time.Duration         `json:"execution_timeout"`
	Pipeline         PipelineRun           `json:"pipeline"`
	// Workflow carries the pipeline's activity options and config; its
	// payload is unused.
	Workflow workflowengine.WorkflowInput `json:"workflow"`
}

// PrepareResult is the prepared device.
type PrepareResult struct {
	// Device is the device's setted_devices entry.
	Device map[string]any `json:"device"`
	// Steps is the prepared device of each step, by step ID.
	Steps map[string]PreparedMobileDevice `json:"steps"`
}

// RunStepInput runs one mobile-automation step.
type RunStepInput struct {
	// WorkflowID is the step's task workflow ID.
	WorkflowID       string        `json:"workflow_id"`
	ExecutionTimeout time.Duration `json:"execution_timeout"`
	Pipeline         PipelineRun   `json:"pipeline"`
	// Input is the step workflow's input; its config holds the step ID.
	Input workflowengine.WorkflowInput `json:"input"`
}

// CleanUpInput cleans one device after a pipeline run.
type CleanUpInput struct {
	WorkflowID string `json:"workflow_id"`
	DeviceID   string `json:"device_id"`
	// Device is the setted_devices entry Prepare returned.
	Device           map[string]any               `json:"device"`
	RunIdentifier    string                       `json:"run_identifier"`
	ExecutionTimeout time.Duration                `json:"execution_timeout"`
	Pipeline         PipelineRun                  `json:"pipeline"`
	Workflow         workflowengine.WorkflowInput `json:"workflow"`
}

// CleanUpResult is the evidence the device cleanup stored and what went
// wrong.
type CleanUpResult struct {
	ResultVideoURLs []string `json:"result_video_urls"`
	ScreenshotURLs  []string `json:"screenshot_urls"`
	Warnings        []string `json:"warnings"`
	Errors          []string `json:"errors"`
}
