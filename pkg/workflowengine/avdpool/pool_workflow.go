// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package avdpool

import (
	"fmt"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/sdk/workflow"
)

const PoolManagerWorkflowName = "avd-pool-manager"

var defaultContinueAfter = 24 * time.Hour

// PoolManagerWorkflow manages AVD pool slots using workflow-as-semaphore.
type PoolManagerWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

func NewPoolManagerWorkflow() *PoolManagerWorkflow {
	w := &PoolManagerWorkflow{}
	w.WorkflowFunc = w.ExecuteWorkflow
	return w
}

func (PoolManagerWorkflow) Name() string {
	return PoolManagerWorkflowName
}

func (PoolManagerWorkflow) GetOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{}
}

func (w *PoolManagerWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *PoolManagerWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)

	config := DefaultPoolConfig()
	state := PoolState{
		AvailableSlots: config.MaxConcurrentEmulators,
		QueuedRequests: []PoolRequest{},
		ActiveLeases:   map[string]PoolLease{},
	}

	if payload, err := workflowengine.DecodePayload[PoolManagerWorkflowInput](input.Payload); err == nil {
		if payload.Config != (PoolConfig{}) {
			config = ApplyPoolConfigDefaults(payload.Config)
			state.AvailableSlots = config.MaxConcurrentEmulators
		}
		if len(payload.State.ActiveLeases) > 0 || len(payload.State.QueuedRequests) > 0 {
			state = payload.State
			if state.ActiveLeases == nil {
				state.ActiveLeases = map[string]PoolLease{}
			}
			if state.AvailableSlots == 0 {
				state.AvailableSlots = max(0, config.MaxConcurrentEmulators-len(state.ActiveLeases))
			}
		}
	}

	config = ApplyPoolConfigDefaults(config)

	if state.ActiveLeases == nil {
		state.ActiveLeases = map[string]PoolLease{}
	}

	startTime := workflow.Now(ctx)

	acquireCh := workflow.GetSignalChannel(ctx, PoolAcquireSignal)
	releaseCh := workflow.GetSignalChannel(ctx, PoolReleaseSignal)
	heartbeatCh := workflow.GetSignalChannel(ctx, PoolHeartbeatSignal)
	updateCapCh := workflow.GetSignalChannel(ctx, PoolUpdateCapacitySignal)

	if err := workflow.SetQueryHandler(ctx, PoolStatusQuery, func() (PoolStatus, error) {
		return PoolStatus{
			Available:   state.AvailableSlots,
			Queued:      len(state.QueuedRequests),
			Active:      len(state.ActiveLeases),
			MaxCapacity: config.MaxConcurrentEmulators,
		}, nil
	}); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	assignSlots := func(now time.Time) {
		for state.AvailableSlots > 0 && len(state.QueuedRequests) > 0 {
			request := state.QueuedRequests[0]
			state.QueuedRequests = state.QueuedRequests[1:]

			if isRequestTimedOut(now, request) {
				_ = sendPoolResponse(ctx, request, PoolSlotResponse{
					WorkflowID:   request.WorkflowID,
					RunID:        request.RunID,
					RequestID:    request.RequestID,
					Granted:      false,
					ErrorCode:    PoolTimeoutErrorCode,
					ErrorMessage: "pool slot request timed out",
				})
				continue
			}

			state.AvailableSlots--
			state.ActiveLeases[request.WorkflowID] = PoolLease{
				WorkflowID:    request.WorkflowID,
				RunID:         request.RunID,
				RequestID:     request.RequestID,
				AcquiredAt:    now,
				LastHeartbeat: now,
			}
			_ = sendPoolResponse(ctx, request, PoolSlotResponse{
				WorkflowID: request.WorkflowID,
				RunID:      request.RunID,
				RequestID:  request.RequestID,
				Granted:    true,
			})
		}
	}

	for {
		now := workflow.Now(ctx)
		if ctx.Err() != nil {
			return workflowengine.WorkflowResult{}, ctx.Err()
		}
		if now.Sub(startTime) >= defaultContinueAfter {
			continueInput := workflowengine.WorkflowInput{
				Payload: PoolManagerWorkflowInput{
					Config: config,
					State:  state,
				},
				Config: map[string]any{},
			}
			return workflowengine.WorkflowResult{}, workflow.NewContinueAsNewError(
				ctx,
				w.Workflow,
				continueInput,
			)
		}

		timer := workflow.NewTimer(ctx, config.HeartbeatInterval)
		selector := workflow.NewSelector(ctx)

		selector.AddReceive(acquireCh, func(c workflow.ReceiveChannel, _ bool) {
			var req PoolRequest
			c.Receive(ctx, &req)
			if req.Timeout <= 0 {
				req.Timeout = config.HeartbeatInterval
			}
			if lease, ok := state.ActiveLeases[req.WorkflowID]; ok {
				logger.Info("pool slot already acquired", "workflow_id", req.WorkflowID)
				_ = sendPoolResponse(ctx, req, PoolSlotResponse{
					WorkflowID: req.WorkflowID,
					RunID:      req.RunID,
					RequestID:  req.RequestID,
					Granted:    true,
				})
				lease.LastHeartbeat = now
				state.ActiveLeases[req.WorkflowID] = lease
				return
			}
			if state.AvailableSlots > 0 {
				state.AvailableSlots--
				state.ActiveLeases[req.WorkflowID] = PoolLease{
					WorkflowID:    req.WorkflowID,
					RunID:         req.RunID,
					RequestID:     req.RequestID,
					AcquiredAt:    now,
					LastHeartbeat: now,
				}
				_ = sendPoolResponse(ctx, req, PoolSlotResponse{
					WorkflowID: req.WorkflowID,
					RunID:      req.RunID,
					RequestID:  req.RequestID,
					Granted:    true,
				})
				return
			}
			if state.MaxQueueDepthReached(config) {
				_ = sendPoolResponse(ctx, req, PoolSlotResponse{
					WorkflowID:   req.WorkflowID,
					RunID:        req.RunID,
					RequestID:    req.RequestID,
					Granted:      false,
					ErrorCode:    PoolExhaustedErrorCode,
					ErrorMessage: fmt.Sprintf("AVD pool full, %d workflows queued", len(state.QueuedRequests)),
				})
				return
			}
			state.QueuedRequests = append(state.QueuedRequests, req)
		})

		selector.AddReceive(releaseCh, func(c workflow.ReceiveChannel, _ bool) {
			var rel PoolRelease
			c.Receive(ctx, &rel)
			if _, ok := state.ActiveLeases[rel.WorkflowID]; ok {
				delete(state.ActiveLeases, rel.WorkflowID)
				state.AvailableSlots++
			}
			assignSlots(now)
		})

		selector.AddReceive(heartbeatCh, func(c workflow.ReceiveChannel, _ bool) {
			var hb PoolHeartbeat
			c.Receive(ctx, &hb)
			if lease, ok := state.ActiveLeases[hb.WorkflowID]; ok {
				lease.LastHeartbeat = now
				state.ActiveLeases[hb.WorkflowID] = lease
			}
		})

		selector.AddReceive(updateCapCh, func(c workflow.ReceiveChannel, _ bool) {
			var update PoolCapacityUpdate
			c.Receive(ctx, &update)
			if update.MaxConcurrentEmulators <= 0 {
				return
			}
			config.MaxConcurrentEmulators = update.MaxConcurrentEmulators
			state.AvailableSlots = max(0, config.MaxConcurrentEmulators-len(state.ActiveLeases))
			assignSlots(now)
		})

		selector.AddFuture(timer, func(f workflow.Future) {
			_ = f.Get(ctx, nil)
			expireLeases(now, config.LeaseTimeout, state.ActiveLeases, func(workflowID string) {
				delete(state.ActiveLeases, workflowID)
				state.AvailableSlots++
			})
			state.QueuedRequests = expireQueuedRequests(ctx, now, state.QueuedRequests)
			assignSlots(workflow.Now(ctx))
		})

		selector.Select(ctx)
	}
}

func sendPoolResponse(
	ctx workflow.Context,
	request PoolRequest,
	response PoolSlotResponse,
) error {
	return workflow.SignalExternalWorkflow(
		ctx,
		request.WorkflowID,
		request.RunID,
		PoolResponseSignal,
		response,
	).Get(ctx, nil)
}

func expireQueuedRequests(
	ctx workflow.Context,
	now time.Time,
	queue []PoolRequest,
) []PoolRequest {
	remaining := make([]PoolRequest, 0, len(queue))
	for _, req := range queue {
		if isRequestTimedOut(now, req) {
			_ = sendPoolResponse(ctx, req, PoolSlotResponse{
				WorkflowID:   req.WorkflowID,
				RunID:        req.RunID,
				RequestID:    req.RequestID,
				Granted:      false,
				ErrorCode:    PoolTimeoutErrorCode,
				ErrorMessage: "pool slot request timed out",
			})
			continue
		}
		remaining = append(remaining, req)
	}
	return remaining
}

func expireLeases(
	now time.Time,
	leaseTimeout time.Duration,
	leases map[string]PoolLease,
	onExpire func(workflowID string),
) {
	for workflowID, lease := range leases {
		if now.Sub(lease.LastHeartbeat) > leaseTimeout {
			onExpire(workflowID)
		}
	}
}

func isRequestTimedOut(now time.Time, req PoolRequest) bool {
	return req.Timeout > 0 && now.Sub(req.RequestTime) >= req.Timeout
}

func (state PoolState) MaxQueueDepthReached(config PoolConfig) bool {
	return len(state.QueuedRequests) >= config.MaxQueueDepth
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
