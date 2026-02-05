// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"fmt"
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const CustomCheckTaskQueue = "custom-check-task-queue"

// CustomCheckWorkflow is a workflow that performs a custom check.
type CustomCheckWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

type CustomCheckWorkflowPayload struct {
	Yaml    string `json:"yaml,omitempty"     xoneof:"custom_check"`
	CheckID string `json:"check_id,omitempty" xoneof:"custom_check"`
}

func NewCustomCheckWorkflow() *CustomCheckWorkflow {
	w := &CustomCheckWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
}

func (CustomCheckWorkflow) Name() string {
	return "Custom Check Workflow"
}

func (CustomCheckWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *CustomCheckWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *CustomCheckWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	stepCIWorkflowActivity := activities.NewStepCIWorkflowActivity()
	logger := workflow.GetLogger(ctx)

	opts := w.GetOptions()
	if input.ActivityOptions != nil {
		opts = *input.ActivityOptions
	}
	ctx = workflow.WithActivityOptions(ctx, opts)
	payload, err := workflowengine.DecodePayload[CustomCheckWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}
	yaml := payload.Yaml
	if yaml == "" {
		if payload.CheckID == "" {
			return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
				fmt.Errorf("yaml or id must be provided"),
				input.RunMetadata,
			)
		}
		var HTTPActivity = activities.NewHTTPActivity()
		var HTTPResponse workflowengine.ActivityResult
		err := workflow.ExecuteActivity(ctx, HTTPActivity.Name(), workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method: http.MethodPost,
				URL: utils.JoinURL(
					input.Config["app_url"].(string),
					"api", "canonify", "identifier", "validate",
				),
				Body: map[string]any{
					"canonified_name": payload.CheckID,
				},
				ExpectedStatus: 200,
			},
		}).Get(ctx, &HTTPResponse)
		if err != nil {
			logger.Error(HTTPActivity.Name(), "error", err)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				input.RunMetadata,
			)
		}
		errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
		output, ok := HTTPResponse.Output.(map[string]any)
		if !ok {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: invalid output format", errCode.Description),
				HTTPResponse.Output,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				input.RunMetadata,
			)
		}

		body, ok := output["body"].(map[string]any)
		if !ok {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: missing body in output", errCode.Description),
				output,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				input.RunMetadata,
			)
		}

		record, ok := body["record"].(map[string]any)
		if !ok {
			appErr := workflowengine.NewAppError(
				errCode,
				fmt.Sprintf("%s: missing record in body", errCode.Description),
				body,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				input.RunMetadata,
			)
		}
		var storedYaml string
		storedYaml, ok = record["yaml"].(string)
		if !ok || storedYaml == "" {
			appErr := workflowengine.NewAppError(
				errCode,
				"missing yaml in custom check record",
				record,
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				input.RunMetadata,
			)
		}
		yaml = storedYaml
	}
	env, _ := input.Config["env"].(string)
	stepCIInput := workflowengine.ActivityInput{
		Payload: activities.StepCIWorkflowActivityPayload{
			Yaml: yaml,
			Env:  env,
		},
	}
	var stepCIResult workflowengine.ActivityResult

	err = workflow.ExecuteActivity(ctx, stepCIWorkflowActivity.Name(), stepCIInput).
		Get(ctx, &stepCIResult)

	if err != nil {
		logger.Error(stepCIWorkflowActivity.Name(), "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
	}
	result := stepCIResult.Output.(map[string]any)
	return workflowengine.WorkflowResult{
		Output: result["tests"].([]any),
	}, nil
}

func (w *CustomCheckWorkflow) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        "custom" + "-" + uuid.NewString(),
		TaskQueue: CustomCheckTaskQueue,
	}
	return workflowengine.StartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}
