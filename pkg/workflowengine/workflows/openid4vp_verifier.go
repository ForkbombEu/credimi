// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const (
	OpenID4VPVerifierTaskQueue          = "OpenID4VPVerifierTaskQueue"
	OpenID4VPVerifierStepCITemplatePath = "pkg/workflowengine/workflows/openid4vp_verifier_config/stepci_verifier_template_v1_0.yaml"
	OpenID4VPVerifierStartCheckSignal   = "start-openid4vp-verifier-log-update"
	OpenID4VPVerifierStopCheckSignal    = "stop-openid4vp-verifier-log-update"
)

// OpenID4VPVerifierWorkflowPayload is the input payload for OpenID4VPVerifierWorkflow.
type OpenID4VPVerifierWorkflowPayload = OpenIDConformanceWorkflowPayload

// OpenID4VPVerifierWorkflow runs a fully automated OID4VP verifier conformance check
// against the OpenID Foundation Certification Suite.
type OpenID4VPVerifierWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

var openID4VPVerifierStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

func NewOpenID4VPVerifierWorkflow() *OpenID4VPVerifierWorkflow {
	w := &OpenID4VPVerifierWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (OpenID4VPVerifierWorkflow) Name() string {
	return "OID4VP Verifier conformance check on https://www.certification.openid.net"
}

func (OpenID4VPVerifierWorkflow) GetOptions() workflow.ActivityOptions {
	return openIDConformanceActivityOptions
}

func (w *OpenID4VPVerifierWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *OpenID4VPVerifierWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return runOpenIDConformanceWorkflow(ctx, input, w.GetOptions(), true)
}

func (w *OpenID4VPVerifierWorkflow) Start(
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "OpenID4VPVerifierCheckWorkflow" + uuid.NewString(),
		TaskQueue:                OpenID4VPVerifierTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}
	namespace := DefaultNamespace
	if input.Config["namespace"] != nil {
		namespace = input.Config["namespace"].(string)
	}
	return openID4VPVerifierStartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}
