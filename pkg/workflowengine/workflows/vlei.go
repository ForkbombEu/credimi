// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"reflect"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const VLEIValidationTaskQueue = "VLEIValidationTaskQueue"

type VLEIValidationWorkflow struct{}

func (w *VLEIValidationWorkflow) Name() string {
	return "Validate vLEI against a schema"
}

func (w *VLEIValidationWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *VLEIValidationWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
	}

	schema, ok := input.Config["schema"].(string)
	if !ok || schema == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError("schema", runMetadata)
	}

	vLEIType, ok := input.Payload["vLEI_type"].(string)
	if !ok || vLEIType == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError("vLEI_type", runMetadata)
	}

	var jsonData map[string]any
	var err error

	raw, ok := input.Payload["rawJSON"].(string)
	if !ok || raw == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError("rawJSON", runMetadata)
	}

	parseJSON := activities.NewJSONActivity(map[string]reflect.Type{"map": reflect.TypeOf(map[string]any{})})
	var parsedResult workflowengine.ActivityResult

	err = workflow.ExecuteActivity(ctx, parseJSON.Name(), workflowengine.ActivityInput{
		Payload: map[string]any{
			"rawJSON":    raw,
			"structType": "map",
		},
	}).Get(ctx, &parsedResult)

	if err != nil {
		logger.Error("ParseJSON failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}

	jsonData, err = decodeToMap(parsedResult.Output, runMetadata)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	validateJSON := activities.NewSchemaValidationActivity()
	err = workflow.ExecuteActivity(ctx, validateJSON.Name(), workflowengine.ActivityInput{
		Payload: map[string]any{
			"data":   jsonData,
			"schema": schema,
		},
	}).Get(ctx, nil)

	if err != nil {
		logger.Error("SchemaValidation failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}

	return workflowengine.WorkflowResult{
		Message: "vLEI is valid according to the schema for " + vLEIType,
	}, nil
}

func (w *VLEIValidationWorkflow) Start(input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                       "VLEIValidation-" + uuid.NewString(),
		TaskQueue:                VLEIValidationTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}
	return workflowengine.StartWorkflowWithOptions(workflowOptions, w.Name(), input)
}
