// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/runqueue"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
)

// EnqueuePipelineRunTicketActivity enqueues run tickets into the mobile runner queue.
type EnqueuePipelineRunTicketActivity struct {
	workflowengine.BaseActivity
	temporalClientFactory func(namespace string) (temporalWorkflowUpdater, error)
}

// temporalWorkflowUpdater defines the Temporal client methods used by the enqueue activity.
type temporalWorkflowUpdater interface {
	ExecuteWorkflow(
		ctx context.Context,
		options client.StartWorkflowOptions,
		workflow interface{},
		args ...interface{},
	) (client.WorkflowRun, error)
	UpdateWorkflow(
		ctx context.Context,
		options client.UpdateWorkflowOptions,
	) (client.WorkflowUpdateHandle, error)
}

// NewEnqueuePipelineRunTicketActivity constructs the enqueue activity.
func NewEnqueuePipelineRunTicketActivity() *EnqueuePipelineRunTicketActivity {
	return &EnqueuePipelineRunTicketActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: EnqueuePipelineRunTicketActivityName,
		},
		temporalClientFactory: func(namespace string) (temporalWorkflowUpdater, error) {
			return temporalclient.GetTemporalClientWithNamespace(namespace)
		},
	}
}

// Name returns the activity name for enqueueing pipeline run tickets.
func (a *EnqueuePipelineRunTicketActivity) Name() string {
	return a.BaseActivity.Name
}

// Execute enqueues a run ticket across all runner semaphore workflows.
func (a *EnqueuePipelineRunTicketActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[EnqueuePipelineRunTicketActivityInput](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	ticketID := strings.TrimSpace(payload.TicketID)
	if ticketID == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, "ticket_id is required")
	}
	ownerNamespace := strings.TrimSpace(payload.OwnerNamespace)
	if ownerNamespace == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, "owner_namespace is required")
	}
	pipelineIdentifier := strings.TrimSpace(payload.PipelineIdentifier)
	if pipelineIdentifier == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, "pipeline_identifier is required")
	}
	yaml := strings.TrimSpace(payload.YAML)
	if yaml == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, "yaml is required")
	}

	runnerIDs := normalizeRunnerIDs(payload.RunnerIDs)
	if len(runnerIDs) == 0 {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(errCode.Code, "runner_ids are required")
	}

	config := payload.PipelineConfig
	if config == nil {
		config = map[string]any{}
	}
	memo := payload.Memo
	if memo == nil {
		memo = map[string]any{}
	}

	factory := a.temporalClientFactory
	if factory == nil {
		factory = func(namespace string) (temporalWorkflowUpdater, error) {
			return temporalclient.GetTemporalClientWithNamespace(namespace)
		}
	}
	temporalClient, err := factory(workflowengine.MobileRunnerSemaphoreDefaultNamespace)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		return result, a.NewActivityError(errCode.Code, err.Error())
	}

	for _, runnerID := range runnerIDs {
		if err := ensureRunQueueSemaphoreWorkflow(ctx, temporalClient, runnerID); err != nil {
			errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
			return result, a.NewActivityError(errCode.Code, err.Error())
		}
	}

	leaderRunnerID := runnerIDs[0]
	var logger log.Logger
	if activity.IsActivity(ctx) {
		logger = activity.GetLogger(ctx)
	}
	rollbackRunnerIDs := make([]string, 0, len(runnerIDs))
	runnerStatuses := make([]EnqueuePipelineRunTicketRunnerStatus, 0, len(runnerIDs))

	rollbackEnqueuedTickets := func(runnerIDs []string) {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for _, runnerID := range runnerIDs {
			status, err := cancelRunTicket(
				rollbackCtx,
				temporalClient,
				runnerID,
				mobilerunnersemaphore.MobileRunnerSemaphoreRunCancelRequest{
					TicketID:       ticketID,
					OwnerNamespace: ownerNamespace,
				},
			)
			if err != nil {
				if errors.Is(err, errRunTicketNotFound) {
					continue
				}
				if logger != nil {
					logger.Warn(fmt.Sprintf(
						"failed to rollback run ticket %s for runner %s: %v",
						ticketID,
						runnerID,
						err,
					))
				}
				continue
			}
			if status.Status == mobilerunnersemaphore.MobileRunnerSemaphoreRunNotFound {
				continue
			}
		}
	}

	for _, runnerID := range runnerIDs {
		rollbackRunnerIDs = append(rollbackRunnerIDs, runnerID)
		req := mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunRequest{
			TicketID:            ticketID,
			OwnerNamespace:      ownerNamespace,
			EnqueuedAt:          payload.EnqueuedAt,
			RunnerID:            runnerID,
			RequiredRunnerIDs:   runnerIDs,
			LeaderRunnerID:      leaderRunnerID,
			MaxPipelinesInQueue: payload.MaxPipelinesInQueue,
			PipelineIdentifier:  pipelineIdentifier,
			YAML:                yaml,
			PipelineConfig:      config,
			Memo:                memo,
		}
		resp, err := enqueueRunTicket(ctx, temporalClient, runnerID, req)
		if err != nil {
			rollbackEnqueuedTickets(rollbackRunnerIDs)
			if isQueueLimitExceeded(err) {
				return result, err
			}
			errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
			return result, a.NewActivityError(errCode.Code, err.Error())
		}
		runnerStatuses = append(runnerStatuses, EnqueuePipelineRunTicketRunnerStatus{
			RunnerID: runnerID,
			Status:   resp.Status,
			Position: resp.Position,
			LineLen:  resp.LineLen,
		})
	}

	aggregate := runqueue.AggregateRunnerStatuses(toRunQueueStatuses(runnerStatuses))
	result.Output = EnqueuePipelineRunTicketActivityOutput{
		Status:            aggregate.Status,
		Position:          aggregate.Position,
		LineLen:           aggregate.LineLen,
		WorkflowID:        aggregate.WorkflowID,
		RunID:             aggregate.RunID,
		WorkflowNamespace: aggregate.WorkflowNamespace,
		ErrorMessage:      aggregate.ErrorMessage,
		Runners:           runnerStatuses,
	}

	return result, nil
}

// errRunTicketNotFound signals that a run ticket could not be located in a runner queue.
var errRunTicketNotFound = errors.New("run ticket not found")

// isQueueLimitExceeded checks if the error reflects a queue limit rejection.
func isQueueLimitExceeded(err error) bool {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return appErr.Type() == mobilerunnersemaphore.ErrQueueLimitExceeded
	}
	return false
}

// ensureRunQueueSemaphoreWorkflow starts the runner semaphore workflow when missing.
func ensureRunQueueSemaphoreWorkflow(
	ctx context.Context,
	temporalClient temporalWorkflowUpdater,
	runnerID string,
) error {
	workflowID := mobilerunnersemaphore.WorkflowID(runnerID)
	input := workflowengine.WorkflowInput{
		Payload: mobilerunnersemaphore.MobileRunnerSemaphoreWorkflowInput{
			RunnerID: runnerID,
			Capacity: 1,
		},
	}

	_, err := temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: mobilerunnersemaphore.TaskQueue,
		},
		mobilerunnersemaphore.WorkflowName,
		input,
	)
	if err != nil && !temporal.IsWorkflowExecutionAlreadyStartedError(err) {
		return err
	}
	return nil
}

// enqueueRunTicket updates the runner semaphore workflow with a run ticket request.
func enqueueRunTicket(
	ctx context.Context,
	temporalClient temporalWorkflowUpdater,
	runnerID string,
	req mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunRequest,
) (mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunResponse, error) {
	workflowID := mobilerunnersemaphore.WorkflowID(runnerID)
	handle, err := temporalClient.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   workflowID,
		UpdateName:   mobilerunnersemaphore.EnqueueRunUpdate,
		UpdateID:     runQueueUpdateID("enqueue", runnerID, req.TicketID),
		Args:         []interface{}{req},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunResponse{}, err
	}

	var response mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunResponse
	if err := handle.Get(ctx, &response); err != nil {
		return mobilerunnersemaphore.MobileRunnerSemaphoreEnqueueRunResponse{}, err
	}
	return response, nil
}

// cancelRunTicket removes a run ticket from the runner semaphore workflow.
func cancelRunTicket(
	ctx context.Context,
	temporalClient temporalWorkflowUpdater,
	runnerID string,
	req mobilerunnersemaphore.MobileRunnerSemaphoreRunCancelRequest,
) (mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView, error) {
	workflowID := mobilerunnersemaphore.WorkflowID(runnerID)
	handle, err := temporalClient.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   workflowID,
		UpdateName:   mobilerunnersemaphore.CancelRunUpdate,
		UpdateID:     runQueueUpdateID("cancel", runnerID, req.TicketID),
		Args:         []interface{}{req},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView{}, errRunTicketNotFound
		}
		return mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView{}, err
	}

	var status mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView
	if err := handle.Get(ctx, &status); err != nil {
		return mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView{}, err
	}

	return status, nil
}

// runQueueUpdateID builds a stable update identifier for runner queue updates.
func runQueueUpdateID(prefix, runnerID, ticketID string) string {
	return prefix + "/" + runnerID + "/" + ticketID
}

// normalizeRunnerIDs trims and filters runner IDs while preserving order.
func normalizeRunnerIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		candidate := strings.TrimSpace(value)
		if candidate == "" {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

// toRunQueueStatuses converts activity runner statuses for aggregation.
func toRunQueueStatuses(
	statuses []EnqueuePipelineRunTicketRunnerStatus,
) []runqueue.RunnerStatus {
	if len(statuses) == 0 {
		return nil
	}
	out := make([]runqueue.RunnerStatus, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, runqueue.RunnerStatus{
			RunnerID:          status.RunnerID,
			Status:            status.Status,
			Position:          status.Position,
			LineLen:           status.LineLen,
			WorkflowID:        status.WorkflowID,
			RunID:             status.RunID,
			WorkflowNamespace: status.WorkflowNamespace,
			ErrorMessage:      status.ErrorMessage,
		})
	}
	return out
}
