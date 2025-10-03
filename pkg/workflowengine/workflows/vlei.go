// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const (
	VLEIValidationTaskQueue      = "VLEIValidationTaskQueue"
	VLEIValidationLocalTaskQueue = "VLEIValidationLocalTaskQueue"
)

type VLEIValidationWorkflow struct{}

func (w *VLEIValidationWorkflow) Name() string {
	return "Validate vLEI credential from server request"
}

func (w *VLEIValidationWorkflow) GetOptions() workflow.ActivityOptions {
	ao := DefaultActivityOptions
	ao.RetryPolicy.MaximumAttempts = 1
	return ao
}

func (w *VLEIValidationWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI: fmt.Sprintf("%s/my/tests/runs/%s/%s",
			input.Config["app_url"],
			workflow.GetInfo(ctx).WorkflowExecution.ID,
			workflow.GetInfo(ctx).WorkflowExecution.RunID,
		),
	}

	serverURL, ok := input.Config["server_url"].(string)
	if !ok || serverURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"server_url",
			runMetadata,
		)
	}

	credentialID, ok := input.Payload["credentialID"].(string)
	if !ok || credentialID == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"credentialID",
			runMetadata,
		)
	}

	// Fetch raw CESR from server
	HTTPActivity := activities.NewHTTPActivity()
	var serverResponse workflowengine.ActivityResult
	request := workflowengine.ActivityInput{
		Payload: map[string]any{
			"method":          "GET",
			"url":             fmt.Sprintf("%s/oobi/%s", serverURL, credentialID),
			"expected_status": 200,
		},
	}
	if err := workflow.ExecuteActivity(ctx, HTTPActivity.Name(), request).
		Get(ctx, &serverResponse); err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			runMetadata,
		)
	}

	outputMap, ok := serverResponse.Output.(map[string]any)
	if !ok {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.UnexpectedHTTPResponse],
				"invalid output type",
				serverResponse.Output,
			),
			runMetadata,
		)
	}

	cesrStr, ok := outputMap["body"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			workflowengine.NewAppError(
				errorcodes.Codes[errorcodes.UnexpectedHTTPResponse],
				"missing 'body'",
				serverResponse.Output,
			),
			runMetadata,
		)
	}

	result, err := validateCESRFromString(ctx, cesrStr, runMetadata)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	result.Message += fmt.Sprintf(" for credential: '%s'", credentialID)
	return result, nil
}

func (w *VLEIValidationWorkflow) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "VLEIValidation-" + uuid.NewString(),
		TaskQueue:                VLEIValidationTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}
	return workflowengine.StartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}

type VLEIValidationLocalWorkflow struct{}

func (w *VLEIValidationLocalWorkflow) Name() string {
	return "Validate vLEI from cesr file"
}

func (w *VLEIValidationLocalWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *VLEIValidationLocalWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI: fmt.Sprintf("%s/my/tests/runs/%s/%s",
			input.Config["app_url"],
			workflow.GetInfo(ctx).WorkflowExecution.ID,
			workflow.GetInfo(ctx).WorkflowExecution.RunID,
		),
	}

	cesrStr, ok := input.Payload["rawCESR"].(string)
	if !ok || cesrStr == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"rawCESR",
			runMetadata,
		)
	}

	return validateCESRFromString(ctx, cesrStr, runMetadata)
}

func (w *VLEIValidationLocalWorkflow) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "VLEILocalValidation-" + uuid.NewString(),
		TaskQueue:                VLEIValidationLocalTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}
	return workflowengine.StartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}

// validateCESRFromString runs CESR parsing + validation inside a workflow.
func validateCESRFromString(
	ctx workflow.Context,
	rawCESR string,
	runMetadata workflowengine.WorkflowErrorMetadata,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)

	parseCESR := activities.NewCESRParsingActivity()
	var parsedResult workflowengine.ActivityResult
	err := workflow.ExecuteActivity(ctx, parseCESR.Name(), workflowengine.ActivityInput{
		Payload: map[string]any{"rawCESR": rawCESR},
	}).Get(ctx, &parsedResult)
	if err != nil {
		logger.Error("ParseCESR failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}

	eventsBytes, err := json.Marshal(parsedResult.Output)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
		appErr := workflowengine.NewAppError(errCode, err.Error(), parsedResult.Output)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	validateCESR := activities.NewCESRValidateActivity()
	var validateResult workflowengine.ActivityResult
	err = workflow.ExecuteActivity(ctx, validateCESR.Name(), workflowengine.ActivityInput{
		Payload: map[string]any{"events": string(eventsBytes)},
	}).Get(ctx, &validateResult)
	if err != nil {
		logger.Error("CESRValidation failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}

	resultMessage, ok := validateResult.Output.(string)
	if !ok {
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		appErr := workflowengine.NewAppError(
			errCode,
			fmt.Sprintf("unexpected output type: %T", validateResult.Output),
			validateResult.Output,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	return workflowengine.WorkflowResult{
		Message: resultMessage,
		Log:     validateResult.Log,
	}, nil
}
