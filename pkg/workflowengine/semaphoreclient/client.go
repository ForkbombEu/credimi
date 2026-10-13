// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package semaphoreclient is the Temporal client side of the mobile-device
// semaphore workflow, shared by the HTTP handlers and the enqueue activity.
package semaphoreclient

import (
	"context"
	"errors"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

// ErrRunTicketNotFound signals that a run ticket could not be located in a device queue.
var ErrRunTicketNotFound = errors.New("run ticket not found")

// UpdateID builds the stable update ID of a semaphore update for a ticket.
func UpdateID(prefix, deviceID, ticketID string) string {
	deviceID = canonify.NormalizePath(deviceID)
	return prefix + "/" + deviceID + "/" + ticketID
}

// startOptions starts the device semaphore, or attaches to the running one.
func startOptions(deviceID string) client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		ID:                       mobiledevicesemaphore.WorkflowID(deviceID),
		TaskQueue:                mobiledevicesemaphore.TaskQueue,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
	}
}

func startInput(deviceID string) workflowengine.WorkflowInput {
	return workflowengine.WorkflowInput{
		Payload: mobiledevicesemaphore.MobileDeviceSemaphoreWorkflowInput{
			DeviceID: deviceID,
			Capacity: 1,
		},
	}
}

// EnsureWorkflow starts the device semaphore workflow unless it is already running.
func EnsureWorkflow(ctx context.Context, c client.Client, deviceID string) error {
	deviceID = canonify.NormalizePath(deviceID)
	_, err := c.ExecuteWorkflow(
		ctx,
		startOptions(deviceID),
		mobiledevicesemaphore.WorkflowName,
		startInput(deviceID),
	)
	return err
}

// EnqueueRun enqueues a run ticket on the device semaphore, starting the
// semaphore workflow in the same call when it is not running.
func EnqueueRun(
	ctx context.Context,
	c client.Client,
	deviceID string,
	req mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest,
) (mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse, error) {
	deviceID = canonify.NormalizePath(deviceID)
	req.DeviceID = canonify.NormalizePath(req.DeviceID)
	req.RequiredDeviceIDs = mobilerunner.NormalizeDeviceIDs(req.RequiredDeviceIDs)
	req.LeaderDeviceID = canonify.NormalizePath(req.LeaderDeviceID)

	// Start operations are single-use, so every call builds a new one.
	op := c.NewWithStartWorkflowOperation(
		startOptions(deviceID),
		mobiledevicesemaphore.WorkflowName,
		startInput(deviceID),
	)
	handle, err := c.UpdateWithStartWorkflow(ctx, client.UpdateWithStartWorkflowOptions{
		StartWorkflowOperation: op,
		UpdateOptions: client.UpdateWorkflowOptions{
			UpdateName:   mobiledevicesemaphore.EnqueueRunUpdate,
			UpdateID:     UpdateID("enqueue", deviceID, req.TicketID),
			Args:         []any{req},
			WaitForStage: client.WorkflowUpdateStageCompleted,
		},
	})
	if err != nil {
		return mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse{}, err
	}

	var resp mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse
	if err := handle.Get(ctx, &resp); err != nil {
		return mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse{}, err
	}
	return resp, nil
}

// CancelRun removes a run ticket from the device semaphore.
func CancelRun(
	ctx context.Context,
	c client.Client,
	deviceID string,
	req mobiledevicesemaphore.MobileDeviceSemaphoreRunCancelRequest,
) (mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView, error) {
	handle, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   mobiledevicesemaphore.WorkflowID(deviceID),
		UpdateName:   mobiledevicesemaphore.CancelRunUpdate,
		UpdateID:     UpdateID("cancel", deviceID, req.TicketID),
		Args:         []any{req},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView{}, ErrRunTicketNotFound
		}
		return mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView{}, err
	}

	var status mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView
	if err := handle.Get(ctx, &status); err != nil {
		return mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView{}, err
	}
	return status, nil
}
