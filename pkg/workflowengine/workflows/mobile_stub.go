//go:build !credimi_extra

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const MobileAutomationTaskQueue = "MobileAutomationTaskQueue"

// MobileAutomationWorkflow is a workflow that runs a mobile automation flow.
type MobileAutomationWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

// MobileAutomationWorkflowPayload is the payload for the mobile automation workflow.
type MobileAutomationWorkflowPayload struct {
	RunIdentifier    string            `json:"run_identifier,omitempty"     yaml:"run_identifier,omitempty"`
	ActionID         string            `json:"action_id,omitempty"          yaml:"action_id,omitempty"`
	VersionID        string            `json:"version_id,omitempty"         yaml:"version_id,omitempty"`
	ActionCode       string            `json:"action_code,omitempty"        yaml:"action_code,omitempty"`
	StoredActionCode bool              `json:"stored_action_code,omitempty" yaml:"stored_action_code,omitempty"`
	Serial           string            `json:"serial,omitempty"             yaml:"serial,omitempty"`
	RunnerID         string            `json:"runner_id,omitempty"          yaml:"runner_id,omitempty"`
	Parameters       map[string]string `json:"parameters,omitempty"         yaml:"parameters,omitempty"`
}

type MobileAutomationWorkflowPipelinePayload struct {
	ActionID   string            `json:"action_id,omitempty"   yaml:"action_id,omitempty"`
	VersionID  string            `json:"version_id,omitempty"  yaml:"version_id,omitempty"`
	ActionCode string            `json:"action_code,omitempty" yaml:"action_code,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"  yaml:"parameters,omitempty"`
	RunnerID   string            `json:"runner_id,omitempty"   yaml:"runner_id,omitempty"`
}

func NewMobileAutomationWorkflow() *MobileAutomationWorkflow {
	w := &MobileAutomationWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (MobileAutomationWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

type MobileWorkflowOutput struct {
	TestRunURL     string `json:"test_run_url"`
	ResultVideoURL string `json:"result_video_url,omitempty"`
	FlowOutput     any    `json:"flow_output,omitempty"`
}

func (MobileAutomationWorkflow) Name() string {
	return "Run a mobile automation workflow"
}

func (w *MobileAutomationWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

// ExecuteWorkflow returns an error when the mobile automation module is disabled.
func (w *MobileAutomationWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	_ = ctx
	return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
		temporal.NewApplicationError(
			"mobile automation is disabled; build with -tags=credimi_extra",
			errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
		),
		input.RunMetadata,
	)
}
