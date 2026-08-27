// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflows provides the OpenID4VCIIssuerWorkflow for running
// OID4VCI issuer conformance checks via the OpenID Foundation Certification Suite.
package workflows

import (
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const (
	OpenID4VCIIssuerTaskQueue          = "OpenID4VCIIssuerTaskQueue"
	OpenID4VCIIssuerStepCITemplatePath = "pkg/workflowengine/workflows/openid4vci_issuer_config/stepci_issuer_template_v1_0.yaml"
	OpenID4VCIIssuerStartCheckSignal   = "start-openid4vci-issuer-log-update"
	OpenID4VCIIssuerStopCheckSignal    = "stop-openid4vci-issuer-log-update"
)

var openID4VCIIssuerStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

// OpenID4VCIIssuerWorkflowPayload is the input payload for OpenID4VCIIssuerWorkflow.
type OpenID4VCIIssuerWorkflowPayload = OpenIDConformanceWorkflowPayload

// OpenID4VCIIssuerWorkflow runs a fully automated OID4VCI issuer conformance check
// against the OpenID Foundation Certification Suite. Unlike the wallet workflow it
// requires no user interaction: StepCI handles plan creation, runner start, and log
// polling end-to-end. The user supplies StepCI parameters, including the credential-offer deeplink.
type OpenID4VCIIssuerWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

func NewOpenID4VCIIssuerWorkflow() *OpenID4VCIIssuerWorkflow {
	w := &OpenID4VCIIssuerWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

// Name returns the human-readable name of the workflow.
func (OpenID4VCIIssuerWorkflow) Name() string {
	return "OID4VCI Issuer conformance check on https://www.certification.openid.net"
}

// GetOptions returns activity options with extended timeouts for the StepCI polling run.
func (OpenID4VCIIssuerWorkflow) GetOptions() workflow.ActivityOptions {
	return openIDConformanceActivityOptions
}

func (w *OpenID4VCIIssuerWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

// ExecuteWorkflow is the main workflow function. It:
//  1. Decodes the payload (parameters + test name).
//  2. Runs StepCIWorkflowActivity using the issuer StepCI Go template to
//     resolve the credential issuer, create the test plan, and start the runner.
//  3. Polls the OpenID certification logs directly from Temporal until the
//     runner reaches a terminal state.
//  4. Returns a WorkflowResult with the full final logs.
//
// The OPENIDNET_TOKEN environment variable must be set with a valid bearer token
// for the OpenID Foundation Certification Suite API.
func (w *OpenID4VCIIssuerWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return runOpenIDConformanceWorkflow(ctx, input, w.GetOptions(), true)
}

// Start enqueues the workflow on the OpenID4VCIIssuerTaskQueue.
func (w *OpenID4VCIIssuerWorkflow) Start(
	input workflowengine.WorkflowInput,
) (result workflowengine.WorkflowResult, err error) {
	input = workflowengine.WithCredimiCapabilities(
		input,
		workflowengine.CredimiCapabilities{Logs: true},
	)
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "OpenID4VCIIssuerCheckWorkflow" + uuid.NewString(),
		TaskQueue:                OpenID4VCIIssuerTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}
	namespace := DefaultNamespace
	if input.Config["namespace"] != nil {
		namespace = input.Config["namespace"].(string)
	}
	return openID4VCIIssuerStartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}
