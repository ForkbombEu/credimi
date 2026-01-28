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

const WorkerManagerTaskQueue = "worker-manager-task-queue"

type WorkerManagerWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

// WorkerManagerWorkflowPayload is the payload for the worker manager workflow.
type WorkerManagerWorkflowPayload struct {
	Namespace    string `json:"namespace"               yaml:"namespace"               validate:"required"`
	OldNamespace string `json:"old_namespace,omitempty" yaml:"old_namespace,omitempty"`
}

func NewWorkerManagerWorkflow() *WorkerManagerWorkflow {
	w := &WorkerManagerWorkflow{}
	w.WorkflowFunc = w.ExecuteWorkflow
	return w
}
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
	return w.WorkflowFunc(ctx, input)
}

func (w *WorkerManagerWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)

	opts := w.GetOptions()
	if input.ActivityOptions != nil {
		opts = *input.ActivityOptions
	}

	ctx = workflow.WithActivityOptions(ctx, opts)
	runMetadata := &workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
	}
	payload, err := workflowengine.DecodePayload[WorkerManagerWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			runMetadata,
		)
	}

	appURL, ok := input.Config["app_url"].(string)
	if !ok || appURL == "" {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingConfigError(
			"app_url",
			runMetadata,
		)
	}

	var HTTPActivity = activities.NewHTTPActivity()
	listReq := workflowengine.ActivityInput{
		Payload: activities.HTTPActivityPayload{
			Method: http.MethodGet,
			URL: utils.JoinURL(
				appURL,
				"api",
				"mobile-runner",
				"list-urls",
			),
			ExpectedStatus: 200,
		},
	}
	var resp workflowengine.ActivityResult
	err = workflow.ExecuteActivity(ctx, HTTPActivity.Name(), listReq).Get(ctx, &resp)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}

	errCode := errorcodes.Codes[errorcodes.UnexpectedActivityOutput]
	body, ok := resp.Output.(map[string]any)["body"].(map[string]any)
	if !ok {
		appErr :=
			workflowengine.NewAppError(
				errCode,
				"invalid HTTP response format",
				resp.Output,
			)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	runnerURLs, ok := body["runners"].([]any)
	if !ok {
		fmt.Println(body)
		appErr :=
			workflowengine.NewAppError(
				errCode,
				"invalid HTTP response body",
				body,
			)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	for _, runnerURL := range runnerURLs {
		err = workflow.ExecuteActivity(ctx, HTTPActivity.Name(), workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method: http.MethodPost,
				URL: utils.JoinURL(
					runnerURL.(string),
					"process",
					payload.Namespace,
				),
				Body: map[string]string{
					"old_namespace": payload.OldNamespace,
				},
				ExpectedStatus: 202,
			},
		}).Get(ctx, nil)

		if err != nil {
			logger.Error("Send namespaces names to start workers failed", "error", err)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				err,
				runMetadata,
			)
		}
	}
	return workflowengine.WorkflowResult{
		Message: fmt.Sprintf(
			"Send namespace '%s' to start workers successfully",
			payload.Namespace,
		),
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
