// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WorkflowInput represents the input data required to start a workflow.
type WorkflowInput struct {
	Payload map[string]any
	Config  map[string]any
}

// WorkflowResult represents the result of a workflow execution, including a message, errors, and a log.
type WorkflowResult struct {
	Message string
	Errors  []error
	Output  any
	Log     any
}

type WorkflowErrorMetadata struct {
	WorkflowName string
	WorkflowID   string
	Namespace    string
	TemporalUI   string
}

// Workflow defines the interface for a workflow, including its execution, name, and options.
type Workflow interface {
	Workflow(ctx workflow.Context, input WorkflowInput) (WorkflowResult, error)
	Name() string
	GetOptions() workflow.ActivityOptions
}

func NewWorkflowError(err error, metadata WorkflowErrorMetadata, extraPayload ...any) error {

	var appErr *temporal.ApplicationError
	if !temporal.IsApplicationError(err) || !errors.As(err, &appErr) {
		return err
	}

	var originalDetails any
	if err := appErr.Details(&originalDetails); err != nil {
		originalDetails = nil
	}

	credimiErr := utils.CredimiError{
		Code:      appErr.Type(),
		Component: "workflow engine",
		Location:  metadata.WorkflowName,
		Message:   appErr.Message(),
		Context:   []string{fmt.Sprintf("Further information at: %s", metadata.TemporalUI)},
	}

	newErr := temporal.NewApplicationError(
		credimiErr.Error(),
		appErr.Type(),
		originalDetails,
		extraPayload,
		metadata,
	)

	return newErr
}

func NewAppError(code errorcodes.Code, field string, payload ...any) error {
	return temporal.NewApplicationError(fmt.Sprintf("%s: '%s'", code.Description, field), code.Code, payload...)
}

// newMissingPayloadError returns a WorkflowError for a missing or invalid payload key.
func NewMissingPayloadError(key string, metadata WorkflowErrorMetadata) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	appErr := NewAppError(errCode, key)
	return NewWorkflowError(appErr, metadata)
}

// newMissingConfigError returns a WorkflowError for a missing or invalid config key.
func NewMissingConfigError(key string, metadata WorkflowErrorMetadata) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
	appErr := NewAppError(errCode, key)
	return NewWorkflowError(appErr, metadata)
}

// newStepCIOutputError returns a WorkflowError for unexpected or invalid StepCI output.
func NewStepCIOutputError(field string, output any, metadata WorkflowErrorMetadata) error {
	errCode := errorcodes.Codes[errorcodes.UnexpectedStepCIOutput]
	appErr := NewAppError(errCode, field, output)
	return NewWorkflowError(appErr, metadata)
}

func StartWorkflowWithOptions(
	options client.StartWorkflowOptions,
	name string,
	input WorkflowInput,
) (result WorkflowResult, err error) {
	// Load environment variables.
	err = godotenv.Load()
	if err != nil {
		return WorkflowResult{}, fmt.Errorf("failed to load .env file: %w", err)
	}
	namespace := "default"
	if input.Config["namespace"] != nil {
		namespace = input.Config["namespace"].(string)
	}
	c, err := temporalclient.GetTemporalClientWithNamespace(
		namespace,
	)
	if err != nil {
		return WorkflowResult{}, fmt.Errorf("unable to create client: %w", err)
	}
	defer c.Close()

	if input.Config["memo"] != nil {
		options.Memo = input.Config["memo"].(map[string]any)
	}

	// Start the workflow execution.
	_, err = c.ExecuteWorkflow(context.Background(), options, name, input)
	if err != nil {
		return WorkflowResult{}, fmt.Errorf("failed to start workflow: %w", err)
	}

	return WorkflowResult{}, nil
}
