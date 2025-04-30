// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"context"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
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

// Workflow defines the interface for a workflow, including its execution, name, and options.
type Workflow interface {
	Workflow(ctx workflow.Context, input WorkflowInput) (WorkflowResult, error)
	Name() string
	GetOptions() workflow.ActivityOptions
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
