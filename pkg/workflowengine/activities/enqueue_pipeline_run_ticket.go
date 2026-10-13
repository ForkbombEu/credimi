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

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/runqueue"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/forkbombeu/credimi/pkg/workflowengine/semaphoreclient"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
)

// EnqueuePipelineRunTicketActivity enqueues run tickets into the mobile runner queue.
type EnqueuePipelineRunTicketActivity struct {
	workflowengine.BaseActivity
	temporalClientFactory func(namespace string) (client.Client, error)
}

// NewEnqueuePipelineRunTicketActivity constructs the enqueue activity.
func NewEnqueuePipelineRunTicketActivity() *EnqueuePipelineRunTicketActivity {
	return &EnqueuePipelineRunTicketActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: EnqueuePipelineRunTicketActivityName,
		},
		temporalClientFactory: temporalclient.GetTemporalClientWithNamespace,
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
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: "ticket_id is required",
			},
		)
	}
	ownerNamespace := strings.TrimSpace(payload.OwnerNamespace)
	if ownerNamespace == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: "owner_namespace is required",
			},
		)
	}
	pipelineIdentifier := strings.TrimSpace(payload.PipelineIdentifier)
	if pipelineIdentifier == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: "pipeline_identifier is required",
			},
		)
	}
	yaml := strings.TrimSpace(payload.YAML)
	if yaml == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: "yaml is required",
			},
		)
	}

	deviceIDs := normalizeDeviceIDs(payload.DeviceIDs)
	if len(deviceIDs) == 0 {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: "device_ids are required",
			},
		)
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
		factory = temporalclient.GetTemporalClientWithNamespace
	}
	temporalClient, err := factory(workflowengine.MobileDeviceSemaphoreDefaultNamespace)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
			},
		)
	}

	leaderDeviceID := deviceIDs[0]
	var logger log.Logger
	if activity.IsActivity(ctx) {
		logger = activity.GetLogger(ctx)
	}
	rollbackDeviceIDs := make([]string, 0, len(deviceIDs))
	runnerStatuses := make([]EnqueuePipelineRunTicketDeviceStatus, 0, len(deviceIDs))

	rollbackEnqueuedTickets := func(deviceIDs []string) {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for _, deviceID := range deviceIDs {
			status, err := semaphoreclient.CancelRun(
				rollbackCtx,
				temporalClient,
				deviceID,
				mobiledevicesemaphore.MobileDeviceSemaphoreRunCancelRequest{
					TicketID:       ticketID,
					OwnerNamespace: ownerNamespace,
				},
			)
			if err != nil {
				if errors.Is(err, semaphoreclient.ErrRunTicketNotFound) {
					continue
				}
				if logger != nil {
					logger.Warn(fmt.Sprintf(
						"failed to rollback run ticket %s for runner %s: %v",
						ticketID,
						deviceID,
						err,
					))
				}
				continue
			}
			if status.Status == mobiledevicesemaphore.MobileDeviceSemaphoreRunNotFound {
				continue
			}
		}
	}

	for _, deviceID := range deviceIDs {
		rollbackDeviceIDs = append(rollbackDeviceIDs, deviceID)
		req := mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest{
			TicketID:            ticketID,
			OwnerNamespace:      ownerNamespace,
			EnqueuedAt:          payload.EnqueuedAt,
			DeviceID:            deviceID,
			RequiredDeviceIDs:   deviceIDs,
			LeaderDeviceID:      leaderDeviceID,
			MaxPipelinesInQueue: payload.MaxPipelinesInQueue,
			PipelineIdentifier:  pipelineIdentifier,
			YAML:                yaml,
			PipelineConfig:      config,
			Memo:                memo,
		}
		resp, err := semaphoreclient.EnqueueRun(ctx, temporalClient, deviceID, req)
		if err != nil {
			rollbackEnqueuedTickets(rollbackDeviceIDs)
			if isQueueLimitExceeded(err) {
				return result, err
			}
			errCode := errorcodes.Codes[errorcodes.PipelineExecutionError]
			return result, a.NewActivityError(
				workflowengine.ActivityError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: err.Error(),
				},
			)
		}
		runnerStatuses = append(runnerStatuses, EnqueuePipelineRunTicketDeviceStatus{
			DeviceID: deviceID,
			Status:   resp.Status,
			Position: resp.Position,
			LineLen:  resp.LineLen,
		})
	}

	aggregate := runqueue.AggregateDeviceStatuses(toRunQueueStatuses(runnerStatuses))
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

// isQueueLimitExceeded checks if the error reflects a queue limit rejection.
func isQueueLimitExceeded(err error) bool {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return appErr.Type() == mobiledevicesemaphore.ErrQueueLimitExceeded
	}
	return false
}

// normalizeDeviceIDs trims and filters runner IDs while preserving order.
func normalizeDeviceIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		candidate := canonify.NormalizePath(value)
		if candidate == "" {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

// toRunQueueStatuses converts activity runner statuses for aggregation.
func toRunQueueStatuses(
	statuses []EnqueuePipelineRunTicketDeviceStatus,
) []runqueue.DeviceStatus {
	if len(statuses) == 0 {
		return nil
	}
	out := make([]runqueue.DeviceStatus, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, runqueue.DeviceStatus{
			DeviceID:          status.DeviceID,
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
