// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflows provides the OpenID4VCIIssuerWorkflow for running
// OID4VCI issuer conformance checks via the OpenID Foundation Certification Suite.
package workflows

import (
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const (
	OpenID4VCIIssuerTaskQueue              = "OpenID4VCIIssuerTaskQueue"
	OpenID4VCIIssuerStepCITemplatePath     = "pkg/workflowengine/workflows/openid4vci_issuer_config/stepci_issuer_template_v1_0.yaml"
)

// issuerActivityOptions extends DefaultActivityOptions with longer timeouts to
// accommodate the StepCI polling loop (up to 30 retries × 10 s = 5 minutes).
var issuerActivityOptions = workflow.ActivityOptions{
	ScheduleToCloseTimeout: time.Hour,
	StartToCloseTimeout:    time.Minute * 30,
	RetryPolicy:            retryPolicy,
}

var openID4VCIIssuerStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

// OpenID4VCIIssuerWorkflowPayload is the input payload for OpenID4VCIIssuerWorkflow.
type OpenID4VCIIssuerWorkflowPayload struct {
	CredentialOffer string `json:"credential_offer" yaml:"credential_offer" validate:"required"`
	UserMail        string `json:"user_mail"        yaml:"user_mail"        validate:"required"`
	TestName        string `json:"test"             yaml:"test"             validate:"required"`
}

// OpenID4VCIIssuerWorkflow runs a fully automated OID4VCI issuer conformance check
// against the OpenID Foundation Certification Suite. Unlike the wallet workflow it
// requires no user interaction: StepCI handles plan creation, runner start, and log
// polling end-to-end. The user only supplies the credential-offer deeplink.
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
	return issuerActivityOptions
}

func (w *OpenID4VCIIssuerWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

// ExecuteWorkflow is the main workflow function. It:
//  1. Decodes the payload (credential_offer + test name).
//  2. Runs StepCIWorkflowActivity using the issuer StepCI Go template, which
//     resolves the credential issuer, creates a certification test plan, starts
//     the runner, and polls logs until the test is FINISHED.
//  3. Returns a WorkflowResult with the captured logs.
//
// The OPENIDNET_TOKEN environment variable must be set with a valid bearer token
// for the OpenID Foundation Certification Suite API.
func (w *OpenID4VCIIssuerWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	payload, err := workflowengine.DecodePayload[OpenID4VCIIssuerWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}

	template, ok := input.Config["template"].(string)
	if !ok || template == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"template",
			input.RunMetadata,
		)
	}

	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			input.RunMetadata,
		)
	}

	stepCIPayload := activities.StepCIWorkflowActivityPayload{
		Data: map[string]any{
			"credential_offer": payload.CredentialOffer,
			"test":             payload.TestName,
		},
		Secrets: map[string]string{
			"token": utils.GetEnvironmentVariable("OPENIDNET_TOKEN", nil, true),
		},
	}

	cfg := StepCIAndEmailConfig{
		AppURL:        appURL,
		AppName:       input.Config["app_name"].(string),
		AppLogo:       input.Config["app_logo"].(string),
		UserName:      input.Config["user_name"].(string),
		UserMail:      payload.UserMail,
		Template:      template,
		StepCIPayload: stepCIPayload,
		Namespace:     input.Config["namespace"].(string),
		RunMetadata:   input.RunMetadata,
		Suite:         OpenIDConformanceSuite,
		SendMail:      false,
	}

	result, err := RunStepCIAndSendMail(ctx, cfg)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	// The StepCI template polls until the test is FINISHED; any failure causes
	// StepCI to exit with an error (handled above). If we reach here the test
	// completed successfully.
	testResult, _ := result.Captures["result"].([]any)
	if len(testResult) > 0 {
		if r, ok := testResult[0].(string); ok && r == "FAILED" {
			errCode := errorcodes.Codes[errorcodes.OpenID4VCIIssuerCheckFailed]
			appErr := workflowengine.NewAppError(errCode, errCode.Description, result.Captures["logs"])
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				input.RunMetadata,
			)
		}
	}

	return workflowengine.WorkflowResult{
		Message: "Check completed successfully",
		Log:     result.Captures["logs"],
	}, nil
}

// Start enqueues the workflow on the OpenID4VCIIssuerTaskQueue.
func (w *OpenID4VCIIssuerWorkflow) Start(
	input workflowengine.WorkflowInput,
) (result workflowengine.WorkflowResult, err error) {
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
