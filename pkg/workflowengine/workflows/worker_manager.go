// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"net/http"
	"strings"

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
	Namespace    string   `json:"namespace"               yaml:"namespace"               validate:"required"`
	OldNamespace string   `json:"old_namespace,omitempty" yaml:"old_namespace,omitempty"`
	RunnerIDs    []string `json:"runner_ids"              yaml:"runner_ids"`
}

type WorkerManagerRunnerResult struct {
	RunnerID string `json:"runner_id"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
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
	runMetadata := &workflowengine.WorkflowRunMetadata{
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

	// Runner-directed calls get their own activity: it reaches the runner at
	// its stored address with the runner's own credential.
	runnerHTTPActivity := activities.NewMobileRunnerHTTPActivity(nil)
	runnerIDs := normalizeWorkerManagerRunnerIDs(payload.RunnerIDs)

	runnerResults := make([]WorkerManagerRunnerResult, 0, len(runnerIDs))
	successfulRunners := 0

	for _, runnerID := range runnerIDs {
		runnerResult := WorkerManagerRunnerResult{
			RunnerID: runnerID,
		}

		err = workflow.ExecuteActivity(ctx, runnerHTTPActivity.Name(), workflowengine.ActivityInput{
			Payload: activities.MobileRunnerHTTPActivityPayload{
				Method:   http.MethodPost,
				RunnerID: runnerID,
				Path:     "/worker/" + payload.Namespace,
				Body: map[string]string{
					"old_namespace": payload.OldNamespace,
				},
				ExpectedStatus: 202,
			},
		}).
			Get(ctx, nil)

		if err != nil {
			logger.Error(
				"Send namespaces names to start workers failed for runner",
				"runner_id",
				runnerID,
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

func normalizeWorkerManagerRunnerIDs(runnerIDs []string) []string {
	seen := make(map[string]struct{}, len(runnerIDs))
	result := make([]string, 0, len(runnerIDs))
	for _, runnerID := range runnerIDs {
		trimmed := strings.TrimSpace(runnerID)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}

		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
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
