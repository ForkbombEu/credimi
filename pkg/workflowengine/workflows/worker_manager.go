// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const WorkerManagerTaskQueue = "worker-manager-task-queue"

type WorkerManagerWorkflow struct{}

func (WorkerManagerWorkflow) Name() string {
	return "Send namespaces names to start workers"
}

func (WorkerManagerWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *WorkerManagerWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)

	opts := w.GetOptions()
	if input.ActivityOptions != nil {
		opts = *input.ActivityOptions
	}
	ctx = workflow.WithActivityOptions(ctx, opts)
	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
	}
	namespace, ok := input.Payload["namespace"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"namespace",
			runMetadata,
		)
	}
	oldNamespace, ok := input.Payload["old_namespace"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingPayloadError(
			"old_namespace",
			runMetadata,
		)
	}
	serverURL, ok := input.Config["server_url"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"server_url",
			runMetadata,
		)
	}

	var HTTPActivity = activities.NewHTTPActivity()
	var HTTPResponse workflowengine.ActivityResult
	err := workflow.ExecuteActivity(ctx, HTTPActivity.Name(), workflowengine.ActivityInput{
		Payload: map[string]any{
			"method": "POST",
			"url": fmt.Sprintf(
				"%s/%s",
				serverURL,
				"process",
			),
			"body": map[string]string{
				"namespace":     namespace,
				"old_namespace": oldNamespace,
			},
			"expected_status": 200,
		},
	}).Get(ctx, &HTTPResponse)

	if err != nil {
		logger.Error("Send namespaces names to start workers failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}
	return workflowengine.WorkflowResult{
		Message: fmt.Sprintf("Send namespace '%s' to start workers successfully", namespace),
	}, nil
}

func (w *WorkerManagerWorkflow) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        "worker-manager" + "-" + uuid.NewString(),
		TaskQueue: WorkerManagerTaskQueue,
	}
	return workflowengine.StartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}
