// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package avdpool

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	PoolAcquireSignal          = "PoolAcquireSlot"
	PoolReleaseSignal          = "PoolReleaseSlot"
	PoolHeartbeatSignal        = "PoolHeartbeat"
	PoolUpdateCapacitySignal   = "PoolUpdateCapacity"
	PoolResponseSignal         = "PoolSlotResponse"
	PoolStatusQuery            = "GetPoolStatus"
	DefaultPoolWorkflowID      = "avd-pool-manager"
	PoolTimeoutErrorCode       = "PoolTimeout"
	PoolExhaustedErrorCode     = "PoolExhausted"
	PoolAlreadyLeasedErrorCode = "PoolAlreadyLeased"
)

func AcquireSlot(ctx workflow.Context, poolWorkflowID string, timeout time.Duration) error {
	info := workflow.GetInfo(ctx)
	requestID := fmt.Sprintf("%s-%d", info.WorkflowExecution.ID, workflow.Now(ctx).UnixNano())

	request := PoolRequest{
		WorkflowID:  info.WorkflowExecution.ID,
		RunID:       info.WorkflowExecution.RunID,
		RequestID:   requestID,
		RequestTime: workflow.Now(ctx),
		Timeout:     timeout,
	}

	if err := workflow.SignalExternalWorkflow(
		ctx,
		poolWorkflowID,
		"",
		PoolAcquireSignal,
		request,
	).Get(ctx, nil); err != nil {
		return err
	}

	responseCh := workflow.GetSignalChannel(ctx, PoolResponseSignal)
	for {
		var resp PoolSlotResponse
		responseCh.Receive(ctx, &resp)
		if resp.RequestID != requestID {
			continue
		}
		if resp.Granted {
			return nil
		}
		if resp.ErrorCode == "" {
			return temporal.NewNonRetryableApplicationError(
				"pool slot request failed",
				PoolExhaustedErrorCode,
				nil,
			)
		}
		return temporal.NewNonRetryableApplicationError(resp.ErrorMessage, resp.ErrorCode, nil)
	}
}

func ReleaseSlot(ctx workflow.Context, poolWorkflowID string) error {
	info := workflow.GetInfo(ctx)
	return workflow.SignalExternalWorkflow(
		ctx,
		poolWorkflowID,
		"",
		PoolReleaseSignal,
		PoolRelease{
			WorkflowID: info.WorkflowExecution.ID,
			RunID:      info.WorkflowExecution.RunID,
		},
	).Get(ctx, nil)
}

func SendHeartbeat(ctx workflow.Context, poolWorkflowID string) error {
	info := workflow.GetInfo(ctx)
	return workflow.SignalExternalWorkflow(
		ctx,
		poolWorkflowID,
		"",
		PoolHeartbeatSignal,
		PoolHeartbeat{
			WorkflowID: info.WorkflowExecution.ID,
			RunID:      info.WorkflowExecution.RunID,
		},
	).Get(ctx, nil)
}

func UpdateCapacity(ctx workflow.Context, poolWorkflowID string, maxConcurrent int) error {
	return workflow.SignalExternalWorkflow(
		ctx,
		poolWorkflowID,
		"",
		PoolUpdateCapacitySignal,
		PoolCapacityUpdate{MaxConcurrentEmulators: maxConcurrent},
	).Get(ctx, nil)
}
