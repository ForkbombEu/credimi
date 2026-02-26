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

var workerManagerStartWorkflowWithOptions = workflowengine.StartWorkflowWithOptions

// WorkerManagerWorkflowPayload is the payload for the worker manager workflow.
type WorkerManagerWorkflowPayload struct {
	Namespace    string `json:"namespace"               yaml:"namespace"               validate:"required"`
	OldNamespace string `json:"old_namespace,omitempty" yaml:"old_namespace,omitempty"`
}

type WorkerManagerRunnerResult struct {
	RunnerURL string `json:"runner_url"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
}

type WorkerManagerWorkflowOutput struct {
	Namespace         string                      `json:"namespace"`
	RunnerResults     []WorkerManagerRunnerResult `json:"runner_results"`
	TotalRunners      int                         `json:"total_runners"`
	SuccessfulRunners int                         `json:"successful_runners"`
	FailedRunners     int                         `json:"failed_runners"`
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

	runnerURLs, ok := parseRunnerURLs(body["runners"])
	if !ok {
		appErr :=
			workflowengine.NewAppError(
				errCode,
				"invalid HTTP response body",
				body,
			)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}

	runnerResults := make([]WorkerManagerRunnerResult, 0, len(runnerURLs))
	successfulRunners := 0

	for _, runnerURL := range runnerURLs {
		runnerResult := WorkerManagerRunnerResult{
			RunnerURL: runnerURL,
		}

		err = workflow.ExecuteActivity(ctx, HTTPActivity.Name(), workflowengine.ActivityInput{
			Payload: activities.HTTPActivityPayload{
				Method: http.MethodPost,
				URL: utils.JoinURL(
					runnerURL,
					"worker",
					payload.Namespace,
				),
				Body: map[string]string{
					"old_namespace": payload.OldNamespace,
				},
				ExpectedStatus: 202,
			},
		}).Get(ctx, nil)

		if err != nil {
			logger.Error(
				"Send namespaces names to start workers failed for runner",
				"runner_url",
				runnerURL,
				"error",
				err,
			)
			runnerResult.Error = err.Error()
		} else {
			runnerResult.Success = true
			successfulRunners++
		}

		runnerResults = append(runnerResults, runnerResult)
	}

	failedRunners := len(runnerResults) - successfulRunners

	return workflowengine.WorkflowResult{
		Message: fmt.Sprintf(
			"Send namespace '%s' to start workers finished: %d/%d succeeded (%d failed)",
			payload.Namespace,
			successfulRunners,
			len(runnerResults),
			failedRunners,
		),
		Output: WorkerManagerWorkflowOutput{
			Namespace:         payload.Namespace,
			RunnerResults:     runnerResults,
			TotalRunners:      len(runnerResults),
			SuccessfulRunners: successfulRunners,
			FailedRunners:     failedRunners,
		},
	}, nil
}

func parseRunnerURLs(rawRunners any) ([]string, bool) {
	switch runners := rawRunners.(type) {
	case []string:
		return runners, true
	case []any:
		runnerURLs := make([]string, 0, len(runners))
		for _, runner := range runners {
			runnerURL, ok := runner.(string)
			if !ok {
				return nil, false
			}
			runnerURLs = append(runnerURLs, runnerURL)
		}
		return runnerURLs, true
	default:
		return nil, false
	}
}

func (w *WorkerManagerWorkflow) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        "worker-manager" + "-" + uuid.NewString(),
		TaskQueue: WorkerManagerTaskQueue,
	}
	return workerManagerStartWorkflowWithOptions(namespace, workflowOptions, w.Name(), input)
}
