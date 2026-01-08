// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflows provides implementations of workflows for Credentials Issuers.
// It includes the CredentialsIssuersWorkflow, which validates and imports credential issuer metadata.
// The workflow performs various steps including checking the issuer, parsing JSON responses,
// storing credentials, and cleaning up invalid credentials.
package workflows

import (
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/workflow"
)

// GetUseCaseVerificationDeeplinkWorkflow is a workflow that
// get the deeplink for a use case verification.
type GetUseCaseVerificationDeeplinkWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

// GetUseCaseVerificationDeeplinkWorkflowPayload is the payload
// for the GetUseCaseVerificationDeeplinkWorkflow.
type GetUseCaseVerificationDeeplinkWorkflowPayload struct {
	UseCaseIdentifier string `json:"use_case_id" yaml:"use_case_id" validate:"required"`
}

func NewGetUseCaseVerificationDeeplinkWorkflow() *GetUseCaseVerificationDeeplinkWorkflow {
	w := &GetUseCaseVerificationDeeplinkWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

// Name returns the name of the workflow.
func (w *GetUseCaseVerificationDeeplinkWorkflow) Name() string {
	return "Get use case verification deeplink"
}

// GetOptions returns the activity options for the workflow.
func (w *GetUseCaseVerificationDeeplinkWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *GetUseCaseVerificationDeeplinkWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *GetUseCaseVerificationDeeplinkWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, *input.ActivityOptions)

	payload, err := workflowengine.DecodePayload[GetUseCaseVerificationDeeplinkWorkflowPayload](
		input.Payload,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
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
	act := activities.NewHTTPActivity()
	var result workflowengine.ActivityResult
	request := workflowengine.ActivityInput{
		Payload: activities.HTTPActivityPayload{
			Method: http.MethodGet,
			URL: utils.JoinURL(
				input.Config["app_url"].(string),
				"api", "verifier", "get-use-case-verification-deeplink",
			),
			QueryParams: map[string]string{
				"use_case_identifier": payload.UseCaseIdentifier,
			},
			ExpectedStatus: 200,
		},
	}
	err = workflow.ExecuteActivity(ctx, act.Name(), request).Get(ctx, &result)
	if err != nil {
		logger.Error("HTTPActivity failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
	}
	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
	responseBody, ok := result.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		wErr := workflowengine.NewAppError(
			errCode,
			"output is not a map",
			result.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			wErr,
			input.RunMetadata,
		)
	}

	code, ok := responseBody["code"].(string)
	if !ok {
		wErr := workflowengine.NewAppError(
			errCode,
			"yaml code is not a string",
			result.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			wErr,
			input.RunMetadata,
		)
	}

	stepCIActivity := activities.NewStepCIWorkflowActivity()
	var stepCIResult workflowengine.ActivityResult
	stepCIInput := workflowengine.ActivityInput{
		Payload: activities.StepCIWorkflowActivityPayload{
			Yaml: code,
		},
	}

	err = workflow.ExecuteActivity(ctx, stepCIActivity.Name(), stepCIInput).Get(ctx, &stepCIResult)
	if err != nil {
		logger.Error("StepCIActivity failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
	}
	captures, ok := stepCIResult.Output.(map[string]any)["captures"].(map[string]any)
	if !ok {
		wErr := workflowengine.NewAppError(
			errCode,
			"captures is not a map",
			result.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			wErr,
			input.RunMetadata,
		)
	}
	deeplink, ok := captures["deeplink"].(string)
	if !ok {
		wErr := workflowengine.NewAppError(
			errCode,
			"deeplink missing or invalid from captures",
			result.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			wErr,
			input.RunMetadata,
		)
	}
	return workflowengine.WorkflowResult{
		Message: "Successfully retrieved  use case verification deeplink",
		Output:  deeplink,
	}, nil
}
