// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/sdk/workflow"
)

const WebuildTemplateFolderPath = "pkg/workflowengine/workflows/webuild_config"

type WebuildWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

var webuildStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

func NewWebuildWorkflow() *WebuildWorkflow {
	w := &WebuildWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (WebuildWorkflow) Name() string {
	return "Conformance check on WEBUILD"
}

func (WebuildWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *WebuildWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *WebuildWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return executeEWCLikeWorkflow(ctx, input, w.GetOptions())
}

func (w *WebuildWorkflow) Start(
	input workflowengine.WorkflowInput,
) (result workflowengine.WorkflowResult, err error) {
	return startEWCLikeWorkflow(
		input,
		w.Name(),
		"WebuildWorkflow",
		webuildStartWorkflowWithOptions,
	)
}

type WebuildStatusWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

func NewWebuildStatusWorkflow() *WebuildStatusWorkflow {
	w := &WebuildStatusWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (WebuildStatusWorkflow) Name() string {
	return "Drain WEBUILD check status conformance endpoint"
}

func (WebuildStatusWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *WebuildStatusWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *WebuildStatusWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return executeEWCLikeStatusWorkflow(ctx, input, w.GetOptions())
}
