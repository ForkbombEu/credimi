// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	mobileDeviceSemaphoreMaxUpdateBatches = 1000
	runCompletionCheckInterval            = 45 * time.Second
	runStartingReconcileInterval          = 20 * time.Second
)

type MobileDeviceSemaphoreWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

const (
	mobileDeviceSemaphoreRunQueued   MobileDeviceSemaphoreRunStatus = workflowengine.MobileDeviceSemaphoreRunQueued
	mobileDeviceSemaphoreRunStarting MobileDeviceSemaphoreRunStatus = workflowengine.MobileDeviceSemaphoreRunStarting
	mobileDeviceSemaphoreRunRunning  MobileDeviceSemaphoreRunStatus = workflowengine.MobileDeviceSemaphoreRunRunning
	mobileDeviceSemaphoreRunFailed   MobileDeviceSemaphoreRunStatus = workflowengine.MobileDeviceSemaphoreRunFailed
	mobileDeviceSemaphoreRunCanceled MobileDeviceSemaphoreRunStatus = workflowengine.MobileDeviceSemaphoreRunCanceled
	mobileDeviceSemaphoreRunNotFound MobileDeviceSemaphoreRunStatus = workflowengine.MobileDeviceSemaphoreRunNotFound
)

func NewMobileDeviceSemaphoreWorkflow() *MobileDeviceSemaphoreWorkflow {
	w := &MobileDeviceSemaphoreWorkflow{}
	w.WorkflowFunc = w.ExecuteWorkflow
	return w
}

func newSemaphoreApplicationError(message string, code string) error {
	return workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    code,
		Summary: message,
	})
}

func (MobileDeviceSemaphoreWorkflow) Name() string {
	return MobileDeviceSemaphoreWorkflowName
}

func (MobileDeviceSemaphoreWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *MobileDeviceSemaphoreWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *MobileDeviceSemaphoreWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	payload, err := workflowengine.DecodePayload[MobileDeviceSemaphoreWorkflowInput](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	runtime, err := newMobileDeviceSemaphoreRuntime(ctx, payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	defer func() {
		if runtime.shouldContinue || runtime.shutdownCompleted {
			return
		}
		cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
		if _, shutdownErr := runtime.shutdownRunnerWithOptions(
			cleanupCtx,
			"semaphore workflow canceled",
			true,
			false,
		); shutdownErr != nil {
			workflow.GetLogger(ctx).Error("semaphore shutdown cleanup failed", "error", shutdownErr)
		}
	}()

	if err := runtime.registerQueryHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerRunStatusHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerListQueuedRunsHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerEnqueueRunHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerCancelRunHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerRunDoneHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerShutdownDeviceHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerPauseDeviceHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerResumeDeviceHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	runtime.startRunSignalHandlers()
	runtime.startRunStarter()
	runtime.startRunSafetyNet()
	runtime.startRunReconciler()
	runtime.startPauseTimeoutWatcher()

	if err := runtime.awaitContinue(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}
	return workflowengine.WorkflowResult{}, nil
}

type mobileDeviceSemaphoreRuntime struct {
	ctx                  workflow.Context
	deviceID             string
	capacity             int
	runQueue             []string
	runTickets           map[string]MobileDeviceSemaphoreRunTicketState
	paused               bool
	pausedAt             time.Time
	pauseReason          string
	pauseGeneration      int
	shutdownAfterSeconds int
	updateCount          int
	shouldContinue       bool
	continueInput        workflowengine.WorkflowInput
	runStarterRequested  bool
	queuePositionsDirty  bool
	shutdownRequested    bool
	shutdownCompleted    bool
}

func newMobileDeviceSemaphoreRuntime(
	ctx workflow.Context,
	payload MobileDeviceSemaphoreWorkflowInput,
) (*mobileDeviceSemaphoreRuntime, error) {
	if payload.DeviceID == "" {
		return nil, newSemaphoreApplicationError(
			"device_id is required",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}

	runtime := &mobileDeviceSemaphoreRuntime{
		ctx:        ctx,
		deviceID:   payload.DeviceID,
		capacity:   payload.Capacity,
		runQueue:   []string{},
		runTickets: map[string]MobileDeviceSemaphoreRunTicketState{},
	}

	runtime.applyPayloadState(payload)
	runtime.normalizeState()

	return runtime, nil
}

func (r *mobileDeviceSemaphoreRuntime) applyPayloadState(
	payload MobileDeviceSemaphoreWorkflowInput,
) {
	if payload.Capacity <= 0 {
		r.capacity = 1
	}

	if payload.State == nil {
		return
	}

	if payload.State.Capacity > 0 {
		r.capacity = payload.State.Capacity
	}
	r.runQueue = payload.State.RunQueue
	r.runTickets = payload.State.RunTickets
	r.paused = payload.State.Paused
	r.pausedAt = payload.State.PausedAt
	r.pauseReason = payload.State.PauseReason
	r.pauseGeneration = payload.State.PauseGeneration
	r.shutdownAfterSeconds = payload.State.ShutdownAfterSeconds
	r.updateCount = payload.State.UpdateCount
}

func (r *mobileDeviceSemaphoreRuntime) normalizeState() {
	if r.capacity <= 0 {
		r.capacity = 1
	}
	if r.runQueue == nil {
		r.runQueue = []string{}
	}
	if r.runTickets == nil {
		r.runTickets = map[string]MobileDeviceSemaphoreRunTicketState{}
	}
	if r.shutdownAfterSeconds < 0 {
		r.shutdownAfterSeconds = 0
	}
}

func (r *mobileDeviceSemaphoreRuntime) registerQueryHandler() error {
	return workflow.SetQueryHandler(
		r.ctx,
		MobileDeviceSemaphoreStateQuery,
		func() (MobileDeviceSemaphoreStateView, error) {
			return MobileDeviceSemaphoreStateView{
				DeviceID:             r.deviceID,
				Capacity:             r.capacity,
				SlotsUsed:            r.runSlotsUsed(),
				QueueLen:             len(r.runQueue),
				Paused:               r.paused,
				PausedAt:             r.pausedAt,
				PauseReason:          r.pauseReason,
				PauseGeneration:      r.pauseGeneration,
				ShutdownAfterSeconds: r.shutdownAfterSeconds,
			}, nil
		},
	)
}

func (r *mobileDeviceSemaphoreRuntime) registerRunStatusHandler() error {
	return workflow.SetQueryHandler(
		r.ctx,
		MobileDeviceSemaphoreRunStatusQuery,
		func(ownerNamespace, ticketID string) (MobileDeviceSemaphoreRunStatusView, error) {
			return r.handleRunStatusQuery(ownerNamespace, ticketID)
		},
	)
}

func (r *mobileDeviceSemaphoreRuntime) registerListQueuedRunsHandler() error {
	return workflow.SetQueryHandler(
		r.ctx,
		MobileDeviceSemaphoreListQueuedRunsQuery,
		func(ownerNamespace string) ([]MobileDeviceSemaphoreQueuedRunView, error) {
			return r.handleListQueuedRunsQuery(ownerNamespace), nil
		},
	)
}

func (r *mobileDeviceSemaphoreRuntime) registerEnqueueRunHandler() error {
	return workflow.SetUpdateHandler(
		r.ctx,
		MobileDeviceSemaphoreEnqueueRunUpdate,
		func(_ workflow.Context, req MobileDeviceSemaphoreEnqueueRunRequest) (MobileDeviceSemaphoreEnqueueRunResponse, error) {
			return r.handleEnqueueRun(req)
		},
	)
}

func (r *mobileDeviceSemaphoreRuntime) registerCancelRunHandler() error {
	return workflow.SetUpdateHandler(
		r.ctx,
		MobileDeviceSemaphoreCancelRunUpdate,
		func(_ workflow.Context, req MobileDeviceSemaphoreRunCancelRequest) (MobileDeviceSemaphoreRunStatusView, error) {
			return r.handleCancelRun(req)
		},
	)
}

func (r *mobileDeviceSemaphoreRuntime) registerRunDoneHandler() error {
	return workflow.SetUpdateHandler(
		r.ctx,
		MobileDeviceSemaphoreRunDoneUpdate,
		func(ctx workflow.Context, req MobileDeviceSemaphoreRunDoneRequest) (MobileDeviceSemaphoreRunStatusView, error) {
			return r.handleRunDone(ctx, req)
		},
	)
}

func (r *mobileDeviceSemaphoreRuntime) registerShutdownDeviceHandler() error {
	return workflow.SetUpdateHandler(
		r.ctx,
		MobileDeviceSemaphoreShutdownDeviceUpdate,
		func(ctx workflow.Context, req MobileDeviceSemaphoreShutdownDeviceRequest) (MobileDeviceSemaphoreShutdownDeviceResponse, error) {
			return r.shutdownRunner(ctx, req.Reason)
		},
	)
}

func (r *mobileDeviceSemaphoreRuntime) registerPauseDeviceHandler() error {
	return workflow.SetUpdateHandler(
		r.ctx,
		MobileDeviceSemaphorePauseDeviceUpdate,
		func(ctx workflow.Context, req MobileDeviceSemaphorePauseDeviceRequest) (MobileDeviceSemaphorePauseDeviceResponse, error) {
			return r.handlePauseDevice(ctx, req)
		},
	)
}

func (r *mobileDeviceSemaphoreRuntime) registerResumeDeviceHandler() error {
	return workflow.SetUpdateHandler(
		r.ctx,
		MobileDeviceSemaphoreResumeDeviceUpdate,
		func(_ workflow.Context, _ MobileDeviceSemaphoreResumeDeviceRequest) (MobileDeviceSemaphoreResumeDeviceResponse, error) {
			r.paused = false
			r.pausedAt = time.Time{}
			r.pauseReason = ""
			r.pauseGeneration++
			r.shutdownAfterSeconds = 0
			r.updateCount++
			r.maybeScheduleContinue()
			r.requestRunStart()

			return MobileDeviceSemaphoreResumeDeviceResponse{
				DeviceID: r.deviceID,
				Paused:   false,
				QueueLen: len(r.runQueue),
			}, nil
		},
	)
}

func (r *mobileDeviceSemaphoreRuntime) startRunSignalHandlers() {
	startRunSignalHandler(
		r.ctx,
		MobileDeviceSemaphoreRunGrantedSignalName,
		func(ctx workflow.Context, signal MobileDeviceSemaphoreRunGrantedSignal) {
			r.handleRunGrantedSignal(signal)
		},
	)
	startRunSignalHandler(
		r.ctx,
		MobileDeviceSemaphoreRunStartedSignalName,
		r.handleRunStartedSignal,
	)
	startRunSignalHandler(
		r.ctx,
		MobileDeviceSemaphoreRunDoneSignalName,
		r.handleRunDoneSignal,
	)
}

func startRunSignalHandler[T any](
	ctx workflow.Context,
	signalName string,
	handler func(workflow.Context, T),
) {
	signalChan := workflow.GetSignalChannel(ctx, signalName)
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			var signal T
			if ok := signalChan.Receive(ctx, &signal); !ok {
				return
			}
			handler(ctx, signal)
		}
	})
}

func (r *mobileDeviceSemaphoreRuntime) startRunStarter() {
	workflow.Go(r.ctx, func(ctx workflow.Context) {
		logger := workflow.GetLogger(ctx)
		for {
			if err := workflow.Await(ctx, func() bool {
				return r.runStarterRequested || r.shouldContinue
			}); err != nil {
				logger.Error("run starter await failed", "error", err)
				return
			}
			if r.shouldContinue {
				return
			}
			r.runStarterRequested = false
			r.processRunQueue(ctx)
		}
	})
	r.requestRunStart()
}

func (r *mobileDeviceSemaphoreRuntime) startRunSafetyNet() {
	workflow.Go(r.ctx, func(ctx workflow.Context) {
		logger := workflow.GetLogger(ctx)
		for {
			if err := workflow.Await(ctx, func() bool {
				return r.shouldContinue || r.hasRunningTickets()
			}); err != nil {
				logger.Error("run safety net await failed", "error", err)
				return
			}
			if r.shouldContinue {
				return
			}
			if err := workflow.Sleep(ctx, runCompletionCheckInterval); err != nil {
				return
			}
			if r.shouldContinue {
				return
			}
			r.checkRunCompletion(ctx)
		}
	})
}

func (r *mobileDeviceSemaphoreRuntime) startRunReconciler() {
	workflow.Go(r.ctx, func(ctx workflow.Context) {
		logger := workflow.GetLogger(ctx)
		for {
			if err := workflow.Await(ctx, func() bool {
				return r.shouldContinue || r.hasFollowerStartingTickets()
			}); err != nil {
				logger.Error("run reconciler await failed", "error", err)
				return
			}
			if r.shouldContinue {
				return
			}
			if err := workflow.Sleep(ctx, runStartingReconcileInterval); err != nil {
				return
			}
			if r.shouldContinue {
				return
			}
			r.reconcileStartingTickets(ctx)
		}
	})
}

func (r *mobileDeviceSemaphoreRuntime) startPauseTimeoutWatcher() {
	workflow.Go(r.ctx, func(ctx workflow.Context) {
		logger := workflow.GetLogger(ctx)
		for {
			if err := workflow.Await(ctx, func() bool {
				return r.shouldContinue || (r.paused && r.shutdownAfterSeconds > 0)
			}); err != nil {
				logger.Error("pause timeout await failed", "error", err)
				return
			}
			if r.shouldContinue {
				return
			}

			generation := r.pauseGeneration
			timeout := time.Duration(r.shutdownAfterSeconds) * time.Second
			if err := workflow.Sleep(ctx, timeout); err != nil {
				return
			}
			if r.shouldContinue {
				return
			}
			if !r.paused || r.pauseGeneration != generation {
				continue
			}

			if _, err := r.shutdownRunner(ctx, "runner pause timeout"); err != nil {
				logger.Error("pause timeout shutdown failed", "error", err)
			}
			return
		}
	})
}

func (r *mobileDeviceSemaphoreRuntime) requestRunStart() {
	r.runStarterRequested = true
}

func (r *mobileDeviceSemaphoreRuntime) handleEnqueueRun(
	req MobileDeviceSemaphoreEnqueueRunRequest,
) (MobileDeviceSemaphoreEnqueueRunResponse, error) {
	if r.shutdownRequested {
		return MobileDeviceSemaphoreEnqueueRunResponse{}, newSemaphoreApplicationError(
			"runner shutdown in progress",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
	if req.TicketID == "" || req.OwnerNamespace == "" {
		return MobileDeviceSemaphoreEnqueueRunResponse{}, newSemaphoreApplicationError(
			"ticket_id and owner_namespace are required",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
	if req.DeviceID == "" || req.DeviceID != r.deviceID {
		return MobileDeviceSemaphoreEnqueueRunResponse{}, newSemaphoreApplicationError(
			"device_id must match semaphore runner",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
	if req.EnqueuedAt.IsZero() {
		return MobileDeviceSemaphoreEnqueueRunResponse{}, newSemaphoreApplicationError(
			"enqueued_at is required",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
	if len(req.RequiredDeviceIDs) == 0 || req.LeaderDeviceID == "" {
		return MobileDeviceSemaphoreEnqueueRunResponse{}, newSemaphoreApplicationError(
			"required_device_ids and leader_device_id are required",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
	if !containsString(req.RequiredDeviceIDs, req.LeaderDeviceID) {
		return MobileDeviceSemaphoreEnqueueRunResponse{}, newSemaphoreApplicationError(
			"leader_device_id must be included in required_device_ids",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}

	if existing, ok := r.runTickets[req.TicketID]; ok {
		if existing.Request.OwnerNamespace != req.OwnerNamespace {
			return MobileDeviceSemaphoreEnqueueRunResponse{}, newSemaphoreApplicationError(
				"ticket owner mismatch",
				MobileDeviceSemaphoreErrInvalidRequest,
			)
		}
		view := r.buildRunStatusView(req.TicketID, existing)
		if view.Status == mobileDeviceSemaphoreRunQueued {
			position, lineLen := r.runQueuePosition(req.TicketID)
			view.Position = position
			view.LineLen = lineLen
		}
		return MobileDeviceSemaphoreEnqueueRunResponse{
			TicketID: view.TicketID,
			Status:   view.Status,
			Position: view.Position,
			LineLen:  view.LineLen,
		}, nil
	}

	if req.MaxPipelinesInQueue > 0 {
		inFlight := r.inFlightRunCount(req.OwnerNamespace)
		if inFlight >= req.MaxPipelinesInQueue {
			return MobileDeviceSemaphoreEnqueueRunResponse{}, newSemaphoreApplicationError(
				fmt.Sprintf(
					"queue limit exceeded for runner %s: %d of %d",
					r.deviceID,
					inFlight,
					req.MaxPipelinesInQueue,
				),
				MobileDeviceSemaphoreErrQueueLimitExceeded,
			)
		}
	}

	r.runTickets[req.TicketID] = MobileDeviceSemaphoreRunTicketState{
		Request: req,
		Status:  mobileDeviceSemaphoreRunQueued,
	}
	r.runQueue = insertRunQueue(r.runQueue, req.TicketID, r.runTickets)
	position, lineLen := r.runQueuePosition(req.TicketID)

	r.updateCount++
	r.maybeScheduleContinue()
	if !r.paused {
		r.requestRunStart()
	}
	r.markQueuePositionsDirty()
	r.flushQueuedPositionUpdates(r.ctx)

	return MobileDeviceSemaphoreEnqueueRunResponse{
		TicketID: req.TicketID,
		Status:   mobileDeviceSemaphoreRunQueued,
		Position: position,
		LineLen:  lineLen,
	}, nil
}

func (r *mobileDeviceSemaphoreRuntime) handlePauseDevice(
	ctx workflow.Context,
	req MobileDeviceSemaphorePauseDeviceRequest,
) (MobileDeviceSemaphorePauseDeviceResponse, error) {
	resp := MobileDeviceSemaphorePauseDeviceResponse{
		DeviceID:             r.deviceID,
		Paused:               true,
		ShutdownAfterSeconds: req.ShutdownAfterSeconds,
	}
	if resp.ShutdownAfterSeconds < 0 {
		resp.ShutdownAfterSeconds = 0
	}
	if r.paused {
		resp.ShutdownAfterSeconds = r.shutdownAfterSeconds
		return resp, nil
	}

	r.paused = true
	r.pausedAt = workflow.Now(ctx)
	r.pauseReason = strings.TrimSpace(req.Reason)
	r.pauseGeneration++
	r.shutdownAfterSeconds = req.ShutdownAfterSeconds
	if r.shutdownAfterSeconds < 0 {
		r.shutdownAfterSeconds = 0
	}
	r.runStarterRequested = false

	if req.CancelRunning {
		ticketIDs := append([]string(nil), r.sortedRunTicketIDs()...)
		for _, ticketID := range ticketIDs {
			state, ok := r.runTickets[ticketID]
			if !ok {
				continue
			}
			if state.Status != mobileDeviceSemaphoreRunStarting &&
				state.Status != mobileDeviceSemaphoreRunRunning {
				continue
			}

			shutdownResp := &MobileDeviceSemaphoreShutdownDeviceResponse{}
			if r.cancelTrackedWorkflow(
				ctx,
				ticketID,
				state,
				func() string {
					if r.pauseReason == "" {
						return "runner paused"
					}
					return "runner paused: " + r.pauseReason
				}(),
				shutdownResp,
			) {
				resp.RunningPipelinesCanceled++
			}
			resp.PipelineCancelFailures = append(
				resp.PipelineCancelFailures,
				shutdownResp.PipelineCancelFailures...,
			)

			if state.Request.LeaderDeviceID == r.deviceID {
				signalResp := &MobileDeviceSemaphoreShutdownDeviceResponse{}
				r.signalRunCanceledForShutdown(ctx, ticketID, state, signalResp)
			}

			r.runQueue = removeFromQueue(r.runQueue, ticketID)
			delete(r.runTickets, ticketID)
			r.updateCount++
			r.markQueuePositionsDirty()
		}
	}

	r.maybeScheduleContinue()
	r.flushQueuedPositionUpdates(ctx)
	return resp, nil
}

func (r *mobileDeviceSemaphoreRuntime) handleCancelRun(
	req MobileDeviceSemaphoreRunCancelRequest,
) (MobileDeviceSemaphoreRunStatusView, error) {
	if req.TicketID == "" || req.OwnerNamespace == "" {
		return MobileDeviceSemaphoreRunStatusView{}, newSemaphoreApplicationError(
			"ticket_id and owner_namespace are required",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}

	state, ok := r.runTickets[req.TicketID]
	if !ok || state.Request.OwnerNamespace != req.OwnerNamespace {
		return MobileDeviceSemaphoreRunStatusView{
			TicketID: req.TicketID,
			Status:   mobileDeviceSemaphoreRunNotFound,
		}, nil
	}

	switch state.Status {
	case mobileDeviceSemaphoreRunQueued, mobileDeviceSemaphoreRunStarting:
		view := r.buildRunStatusView(req.TicketID, state)
		view.Status = mobileDeviceSemaphoreRunNotFound
		r.runQueue = removeFromQueue(r.runQueue, req.TicketID)
		delete(r.runTickets, req.TicketID)
		r.updateCount++
		r.maybeScheduleContinue()
		r.requestRunStart()
		r.markQueuePositionsDirty()
		r.flushQueuedPositionUpdates(r.ctx)
		return view, nil
	case mobileDeviceSemaphoreRunRunning:
		state.CancelRequested = true
		r.runTickets[req.TicketID] = state
		r.updateCount++
		r.maybeScheduleContinue()
		r.requestRunStart()
		return r.buildRunStatusView(req.TicketID, state), nil
	default:
		return r.buildRunStatusView(req.TicketID, state), nil
	}
}

func (r *mobileDeviceSemaphoreRuntime) handleRunDone(
	ctx workflow.Context,
	req MobileDeviceSemaphoreRunDoneRequest,
) (MobileDeviceSemaphoreRunStatusView, error) {
	if req.TicketID == "" || req.OwnerNamespace == "" {
		return MobileDeviceSemaphoreRunStatusView{}, newSemaphoreApplicationError(
			"ticket_id and owner_namespace are required",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}

	state, ok := r.runTickets[req.TicketID]
	if !ok || state.Request.OwnerNamespace != req.OwnerNamespace {
		return MobileDeviceSemaphoreRunStatusView{
			TicketID: req.TicketID,
			Status:   mobileDeviceSemaphoreRunNotFound,
		}, nil
	}

	workflowID := req.WorkflowID
	if workflowID == "" {
		workflowID = state.WorkflowID
	}
	runID := req.RunID
	if runID == "" {
		runID = state.RunID
	}
	signalFollowers := state.Request.LeaderDeviceID == r.deviceID
	workflowResult := strings.TrimSpace(req.WorkflowResult)
	if workflowResult == "" {
		workflowResult = "completed"
	}
	r.finalizeRunTicket(
		ctx,
		req.TicketID,
		state,
		workflowID,
		runID,
		workflowResult,
		signalFollowers,
	)

	return MobileDeviceSemaphoreRunStatusView{
		TicketID: req.TicketID,
		Status:   mobileDeviceSemaphoreRunNotFound,
	}, nil
}

func (r *mobileDeviceSemaphoreRuntime) handleRunStatusQuery(
	ownerNamespace,
	ticketID string,
) (MobileDeviceSemaphoreRunStatusView, error) {
	if ownerNamespace == "" || ticketID == "" {
		return MobileDeviceSemaphoreRunStatusView{
			TicketID: ticketID,
			Status:   mobileDeviceSemaphoreRunNotFound,
		}, nil
	}

	state, ok := r.runTickets[ticketID]
	if !ok || state.Request.OwnerNamespace != ownerNamespace {
		return MobileDeviceSemaphoreRunStatusView{
			TicketID: ticketID,
			Status:   mobileDeviceSemaphoreRunNotFound,
		}, nil
	}

	view := r.buildRunStatusView(ticketID, state)
	if view.Status == mobileDeviceSemaphoreRunQueued {
		position, lineLen := r.runQueuePosition(ticketID)
		view.Position = position
		view.LineLen = lineLen
	}

	return view, nil
}

func (r *mobileDeviceSemaphoreRuntime) handleListQueuedRunsQuery(
	ownerNamespace string,
) []MobileDeviceSemaphoreQueuedRunView {
	if ownerNamespace == "" || len(r.runQueue) == 0 {
		return nil
	}

	views := make([]MobileDeviceSemaphoreQueuedRunView, 0)
	for _, ticketID := range r.runQueue {
		state, ok := r.runTickets[ticketID]
		if !ok || state.Status != mobileDeviceSemaphoreRunQueued {
			continue
		}
		if state.Request.OwnerNamespace != ownerNamespace {
			continue
		}
		position, lineLen := r.runQueuePosition(ticketID)
		views = append(views, MobileDeviceSemaphoreQueuedRunView{
			TicketID:           ticketID,
			OwnerNamespace:     state.Request.OwnerNamespace,
			PipelineIdentifier: state.Request.PipelineIdentifier,
			EnqueuedAt:         state.Request.EnqueuedAt,
			LeaderDeviceID:     state.Request.LeaderDeviceID,
			RequiredDeviceIDs:  copyStringSlice(state.Request.RequiredDeviceIDs),
			Status:             state.Status,
			Position:           position,
			LineLen:            lineLen,
			Cleanup:            state.Request.Cleanup,
		})
	}

	return views
}

func (r *mobileDeviceSemaphoreRuntime) maybeScheduleContinue() {
	if r.shutdownRequested || r.shouldContinue ||
		r.updateCount < mobileDeviceSemaphoreMaxUpdateBatches {
		return
	}

	stateCopy := MobileDeviceSemaphoreWorkflowState{
		Capacity:             r.capacity,
		RunQueue:             copyQueue(r.runQueue),
		RunTickets:           copyRunTickets(r.runTickets),
		Paused:               r.paused,
		PausedAt:             r.pausedAt,
		PauseReason:          r.pauseReason,
		PauseGeneration:      r.pauseGeneration,
		ShutdownAfterSeconds: r.shutdownAfterSeconds,
		UpdateCount:          0,
	}

	r.continueInput = workflowengine.WorkflowInput{
		Payload: MobileDeviceSemaphoreWorkflowInput{
			DeviceID: r.deviceID,
			Capacity: r.capacity,
			State:    &stateCopy,
		},
	}
	r.shouldContinue = true
}

func (r *mobileDeviceSemaphoreRuntime) awaitContinue() error {
	if err := workflow.Await(r.ctx, func() bool {
		return r.shouldContinue || r.shutdownCompleted
	}); err != nil {
		return err
	}

	if r.shouldContinue {
		return workflow.NewContinueAsNewError(
			r.ctx,
			MobileDeviceSemaphoreWorkflowName,
			r.continueInput,
		)
	}

	return nil
}

func (r *mobileDeviceSemaphoreRuntime) processRunQueue(ctx workflow.Context) {
	defer r.flushQueuedPositionUpdates(ctx)
	if r.shutdownRequested || r.paused {
		return
	}

	r.startReadyRuns(ctx)

	for r.availableSlots() > 0 {
		ticketID, state, ok := r.nextQueuedRunTicket()
		if !ok {
			return
		}
		r.grantRunTicket(ctx, ticketID, state)
		r.startReadyRuns(ctx)
	}
}

func (r *mobileDeviceSemaphoreRuntime) startReadyRuns(ctx workflow.Context) {
	if r.shutdownRequested || r.paused {
		return
	}
	logger := workflow.GetLogger(ctx)
	ticketIDs := r.sortedRunTicketIDs()
	for _, ticketID := range ticketIDs {
		state, ok := r.runTickets[ticketID]
		if !ok || state.Status != mobileDeviceSemaphoreRunStarting {
			continue
		}
		if state.Request.LeaderDeviceID != r.deviceID {
			continue
		}
		if state.WorkflowID != "" {
			continue
		}
		if !r.allGrantsReceived(state) {
			continue
		}
		if err := r.startPipelineForTicket(ctx, ticketID, state); err != nil {
			logger.Error("start pipeline failed", "ticket_id", ticketID, "error", err)
			continue
		}
	}
}

func (r *mobileDeviceSemaphoreRuntime) grantRunTicket(
	ctx workflow.Context,
	ticketID string,
	state MobileDeviceSemaphoreRunTicketState,
) {
	if state.Status != mobileDeviceSemaphoreRunQueued {
		return
	}

	r.runQueue = removeFromQueue(r.runQueue, ticketID)

	now := workflow.Now(ctx)
	state.Status = mobileDeviceSemaphoreRunStarting
	state.StartedAt = &now
	if state.GrantedDeviceIDs == nil {
		state.GrantedDeviceIDs = map[string]bool{}
	}
	state.GrantedDeviceIDs[r.deviceID] = true
	r.runTickets[ticketID] = state
	r.updateCount++
	r.maybeScheduleContinue()
	r.markQueuePositionsDirty()

	if state.Request.LeaderDeviceID != r.deviceID {
		if err := r.signalRunGranted(ctx, state.Request.LeaderDeviceID, ticketID); err != nil {
			r.markRunTicketFailed(ticketID, state, err)
		}
		return
	}

	if r.allGrantsReceived(state) {
		if err := r.startPipelineForTicket(ctx, ticketID, state); err != nil {
			logger := workflow.GetLogger(ctx)
			logger.Error("start pipeline failed", "ticket_id", ticketID, "error", err)
		}
		return
	}
}

func (r *mobileDeviceSemaphoreRuntime) startPipelineForTicket(
	ctx workflow.Context,
	ticketID string,
	state MobileDeviceSemaphoreRunTicketState,
) error {
	startActivity := activities.NewStartQueuedPipelineActivity()
	activityOptions := DefaultActivityOptions
	activityOptions.RetryPolicy = &temporal.RetryPolicy{MaximumAttempts: 1}
	activityCtx := workflow.WithActivityOptions(ctx, activityOptions)
	var result workflowengine.ActivityResult
	input := workflowengine.ActivityInput{
		Payload: activities.StartQueuedPipelineActivityInput{
			TicketID:           ticketID,
			OwnerNamespace:     state.Request.OwnerNamespace,
			RequiredDeviceIDs:  state.Request.RequiredDeviceIDs,
			LeaderDeviceID:     state.Request.LeaderDeviceID,
			PipelineIdentifier: state.Request.PipelineIdentifier,
			YAML:               state.Request.YAML,
			PipelineConfig:     state.Request.PipelineConfig,
			Memo:               state.Request.Memo,
		},
	}

	if err := workflow.ExecuteActivity(activityCtx, startActivity.Name(), input).
		Get(activityCtx, &result); err != nil {
		r.markRunTicketFailed(ticketID, state, err)
		r.signalRunDone(ctx, ticketID, state.Request.RequiredDeviceIDs, "", "", "failed")
		return err
	}

	output, err := decodeStartQueuedPipelineOutput(result.Output)
	if err != nil {
		r.markRunTicketFailed(ticketID, state, err)
		r.signalRunDone(ctx, ticketID, state.Request.RequiredDeviceIDs, "", "", "failed")
		return err
	}

	state.Status = mobileDeviceSemaphoreRunRunning
	state.WorkflowID = output.WorkflowID
	state.RunID = output.RunID
	state.WorkflowNamespace = output.WorkflowNamespace
	startedAt := workflow.Now(ctx)
	state.StartedAt = &startedAt
	r.runTickets[ticketID] = state
	r.updateCount++
	r.maybeScheduleContinue()
	r.notifyGitHubPRComment(ctx, ticketID, state, mobileDeviceSemaphoreRunRunning, nil, "", "")

	r.signalRunStarted(ctx, ticketID, state.Request.RequiredDeviceIDs, output)

	return nil
}

func (r *mobileDeviceSemaphoreRuntime) markRunTicketFailed(
	ticketID string,
	state MobileDeviceSemaphoreRunTicketState,
	err error,
) {
	state.Status = mobileDeviceSemaphoreRunFailed
	if err != nil {
		state.ErrorMessage = err.Error()
	}
	r.runTickets[ticketID] = state
	r.updateCount++
	r.maybeScheduleContinue()
	r.requestRunStart()
	r.notifyGitHubPRComment(
		r.ctx,
		ticketID,
		state,
		mobileDeviceSemaphoreRunFailed,
		nil,
		"",
		state.ErrorMessage,
	)
}

func (r *mobileDeviceSemaphoreRuntime) checkRunCompletion(ctx workflow.Context) {
	if len(r.runTickets) == 0 {
		return
	}

	logger := workflow.GetLogger(ctx)
	checkActivity := activities.NewCheckWorkflowClosedActivity()
	activityOptions := DefaultActivityOptions
	activityOptions.RetryPolicy = &temporal.RetryPolicy{MaximumAttempts: 1}
	activityCtx := workflow.WithActivityOptions(ctx, activityOptions)

	ticketIDs := r.sortedRunTicketIDs()
	for _, ticketID := range ticketIDs {
		state, ok := r.runTickets[ticketID]
		if !ok || state.Status != mobileDeviceSemaphoreRunRunning {
			continue
		}
		if state.WorkflowID == "" || state.WorkflowNamespace == "" {
			continue
		}

		input := workflowengine.ActivityInput{
			Payload: activities.CheckWorkflowClosedActivityInput{
				WorkflowID:        state.WorkflowID,
				RunID:             state.RunID,
				WorkflowNamespace: state.WorkflowNamespace,
			},
		}
		var result workflowengine.ActivityResult
		if err := workflow.ExecuteActivity(activityCtx, checkActivity.Name(), input).
			Get(activityCtx, &result); err != nil {
			logger.Error("run completion check failed", "ticket_id", ticketID, "error", err)
			continue
		}

		output, err := decodeCheckWorkflowClosedOutput(result.Output)
		if err != nil {
			logger.Error("run completion decode failed", "ticket_id", ticketID, "error", err)
			continue
		}
		if !output.Closed {
			continue
		}

		signalFollowers := state.Request.LeaderDeviceID == r.deviceID
		r.finalizeRunTicket(
			ctx,
			ticketID,
			state,
			state.WorkflowID,
			state.RunID,
			output.Status,
			signalFollowers,
		)
	}
}

func (r *mobileDeviceSemaphoreRuntime) reconcileStartingTickets(ctx workflow.Context) {
	logger := workflow.GetLogger(ctx)
	ticketIDs := r.sortedRunTicketIDs()
	for _, ticketID := range ticketIDs {
		state, ok := r.runTickets[ticketID]
		if !ok || state.Status != mobileDeviceSemaphoreRunStarting {
			continue
		}
		if state.Request.LeaderDeviceID == r.deviceID {
			continue
		}

		status, err := r.queryLeaderRunStatus(ctx, ticketID, state)
		if err != nil {
			logger.Error(
				"run reconciliation failed",
				"ticket_id",
				ticketID,
				"leader_device_id",
				state.Request.LeaderDeviceID,
				"error",
				err,
			)
			continue
		}

		switch status.Status {
		case mobileDeviceSemaphoreRunRunning:
			if status.WorkflowID != "" {
				state.WorkflowID = status.WorkflowID
			}
			if status.RunID != "" {
				state.RunID = status.RunID
			}
			if status.WorkflowNamespace != "" {
				state.WorkflowNamespace = status.WorkflowNamespace
			}
			state.Status = mobileDeviceSemaphoreRunRunning
			startedAt := workflow.Now(ctx)
			state.StartedAt = &startedAt
			r.runTickets[ticketID] = state
			r.updateCount++
			r.maybeScheduleContinue()
		case mobileDeviceSemaphoreRunFailed,
			mobileDeviceSemaphoreRunCanceled,
			mobileDeviceSemaphoreRunNotFound:
			workflowID := status.WorkflowID
			if workflowID == "" {
				workflowID = state.WorkflowID
			}
			runID := status.RunID
			if runID == "" {
				runID = state.RunID
			}
			r.finalizeRunTicket(
				ctx,
				ticketID,
				state,
				workflowID,
				runID,
				string(status.Status),
				false,
			)
		}
	}
}

func (r *mobileDeviceSemaphoreRuntime) finalizeRunTicket(
	ctx workflow.Context,
	ticketID string,
	state MobileDeviceSemaphoreRunTicketState,
	workflowID string,
	runID string,
	workflowStatus string,
	signalFollowers bool,
) {
	if signalFollowers {
		r.signalRunDone(
			ctx,
			ticketID,
			state.Request.RequiredDeviceIDs,
			workflowID,
			runID,
			workflowStatus,
		)
	}
	if workflowID != "" {
		state.WorkflowID = workflowID
	}
	if runID != "" {
		state.RunID = runID
	}
	r.notifyGitHubPRComment(
		ctx,
		ticketID,
		state,
		MobileDeviceSemaphoreRunStatus("terminated"),
		nil,
		workflowStatus,
		state.ErrorMessage,
	)
	r.runQueue = removeFromQueue(r.runQueue, ticketID)
	delete(r.runTickets, ticketID)
	r.updateCount++
	r.maybeScheduleContinue()
	r.requestRunStart()
	r.markQueuePositionsDirty()
	r.flushQueuedPositionUpdates(ctx)
}

func (r *mobileDeviceSemaphoreRuntime) signalRunGranted(
	ctx workflow.Context,
	leaderDeviceID string,
	ticketID string,
) error {
	future := workflow.SignalExternalWorkflow(
		ctx,
		MobileDeviceSemaphoreWorkflowID(leaderDeviceID),
		"",
		MobileDeviceSemaphoreRunGrantedSignalName,
		MobileDeviceSemaphoreRunGrantedSignal{
			TicketID: ticketID,
			DeviceID: r.deviceID,
		},
	)
	return future.Get(ctx, nil)
}

func (r *mobileDeviceSemaphoreRuntime) signalRunStarted(
	ctx workflow.Context,
	ticketID string,
	requiredDeviceIDs []string,
	output activities.StartQueuedPipelineActivityOutput,
) {
	logger := workflow.GetLogger(ctx)
	for _, deviceID := range requiredDeviceIDs {
		if deviceID == r.deviceID {
			continue
		}
		future := workflow.SignalExternalWorkflow(
			ctx,
			MobileDeviceSemaphoreWorkflowID(deviceID),
			"",
			MobileDeviceSemaphoreRunStartedSignalName,
			MobileDeviceSemaphoreRunStartedSignal{
				TicketID:          ticketID,
				WorkflowID:        output.WorkflowID,
				RunID:             output.RunID,
				WorkflowNamespace: output.WorkflowNamespace,
			},
		)
		if err := future.Get(ctx, nil); err != nil {
			logger.Error(
				"signal run started failed",
				"ticket_id",
				ticketID,
				"target_device_id",
				deviceID,
				"signal",
				MobileDeviceSemaphoreRunStartedSignalName,
				"error",
				err,
			)
		}
	}
}

func (r *mobileDeviceSemaphoreRuntime) signalRunDone(
	ctx workflow.Context,
	ticketID string,
	requiredDeviceIDs []string,
	workflowID string,
	runID string,
	workflowResult string,
) {
	logger := workflow.GetLogger(ctx)
	for _, deviceID := range requiredDeviceIDs {
		if deviceID == r.deviceID {
			continue
		}
		future := workflow.SignalExternalWorkflow(
			ctx,
			MobileDeviceSemaphoreWorkflowID(deviceID),
			"",
			MobileDeviceSemaphoreRunDoneSignalName,
			MobileDeviceSemaphoreRunDoneSignal{
				TicketID:       ticketID,
				WorkflowID:     workflowID,
				RunID:          runID,
				WorkflowResult: workflowResult,
			},
		)
		if err := future.Get(ctx, nil); err != nil {
			logger.Error(
				"signal run done failed",
				"ticket_id",
				ticketID,
				"target_device_id",
				deviceID,
				"signal",
				MobileDeviceSemaphoreRunDoneSignalName,
				"error",
				err,
			)
		}
	}
}

func (r *mobileDeviceSemaphoreRuntime) handleRunGrantedSignal(
	signal MobileDeviceSemaphoreRunGrantedSignal,
) {
	if signal.TicketID == "" || signal.DeviceID == "" {
		return
	}
	state, ok := r.runTickets[signal.TicketID]
	if !ok {
		return
	}
	if state.GrantedDeviceIDs == nil {
		state.GrantedDeviceIDs = map[string]bool{}
	}
	state.GrantedDeviceIDs[signal.DeviceID] = true
	r.runTickets[signal.TicketID] = state
	r.updateCount++
	r.maybeScheduleContinue()
	r.requestRunStart()
}

func (r *mobileDeviceSemaphoreRuntime) handleRunStartedSignal(
	ctx workflow.Context,
	signal MobileDeviceSemaphoreRunStartedSignal,
) {
	if signal.TicketID == "" {
		return
	}
	state, ok := r.runTickets[signal.TicketID]
	if !ok {
		return
	}
	state.Status = mobileDeviceSemaphoreRunRunning
	state.WorkflowID = signal.WorkflowID
	state.RunID = signal.RunID
	state.WorkflowNamespace = signal.WorkflowNamespace
	startedAt := workflow.Now(ctx)
	state.StartedAt = &startedAt
	r.runTickets[signal.TicketID] = state
	r.updateCount++
	r.maybeScheduleContinue()
	r.notifyGitHubPRComment(
		ctx,
		signal.TicketID,
		state,
		mobileDeviceSemaphoreRunRunning,
		nil,
		"",
		"",
	)
}

func (r *mobileDeviceSemaphoreRuntime) handleRunDoneSignal(
	ctx workflow.Context,
	signal MobileDeviceSemaphoreRunDoneSignal,
) {
	if signal.TicketID == "" {
		return
	}
	state, ok := r.runTickets[signal.TicketID]
	if !ok {
		return
	}
	workflowResult := strings.TrimSpace(signal.WorkflowResult)
	if workflowResult == "" {
		workflowResult = "completed"
	}
	r.finalizeRunTicket(
		ctx,
		signal.TicketID,
		state,
		signal.WorkflowID,
		signal.RunID,
		workflowResult,
		false,
	)
}

func (r *mobileDeviceSemaphoreRuntime) shutdownRunner(
	ctx workflow.Context,
	reason string,
) (MobileDeviceSemaphoreShutdownDeviceResponse, error) {
	return r.shutdownRunnerWithOptions(ctx, reason, true, true)
}

func (r *mobileDeviceSemaphoreRuntime) shutdownRunnerWithOptions(
	ctx workflow.Context,
	reason string,
	signalPeers bool,
	signalRunningPeers bool,
) (MobileDeviceSemaphoreShutdownDeviceResponse, error) {
	response := MobileDeviceSemaphoreShutdownDeviceResponse{
		DeviceID: r.deviceID,
	}
	if r.shutdownCompleted {
		return response, nil
	}

	r.shutdownRequested = true
	r.shouldContinue = false

	ticketIDs := append([]string(nil), r.sortedRunTicketIDs()...)
	for _, ticketID := range ticketIDs {
		state, ok := r.runTickets[ticketID]
		if !ok {
			continue
		}

		switch state.Status {
		case mobileDeviceSemaphoreRunQueued:
			r.runQueue = removeFromQueue(r.runQueue, ticketID)
			response.QueuedCanceled++
			response.CleanupFailures = append(
				response.CleanupFailures,
				r.cleanupRunTicketResources(ctx, state)...,
			)
			if signalPeers {
				response.FollowerSignalsSent += r.signalRunCanceledForShutdown(
					ctx,
					ticketID,
					state,
					&response,
				)
			}
			delete(r.runTickets, ticketID)
			r.updateCount++
			r.markQueuePositionsDirty()
		case mobileDeviceSemaphoreRunStarting:
			r.runQueue = removeFromQueue(r.runQueue, ticketID)
			response.StartingCanceled++
			if !ticketHasStartedWorkflow(state) {
				response.CleanupFailures = append(
					response.CleanupFailures,
					r.cleanupRunTicketResources(ctx, state)...,
				)
			} else if r.cancelTrackedWorkflow(ctx, ticketID, state, reason, &response) {
				response.RunningPipelinesCanceled++
			}
			if signalPeers {
				response.FollowerSignalsSent += r.signalRunCanceledForShutdown(
					ctx,
					ticketID,
					state,
					&response,
				)
			}
			delete(r.runTickets, ticketID)
			r.updateCount++
			r.markQueuePositionsDirty()
		case mobileDeviceSemaphoreRunRunning:
			if r.cancelTrackedWorkflow(ctx, ticketID, state, reason, &response) {
				response.RunningPipelinesCanceled++
			}
			if signalPeers && signalRunningPeers {
				response.FollowerSignalsSent += r.signalRunCanceledForShutdown(
					ctx,
					ticketID,
					state,
					&response,
				)
			}
			delete(r.runTickets, ticketID)
			r.updateCount++
			r.markQueuePositionsDirty()
		case mobileDeviceSemaphoreRunFailed,
			mobileDeviceSemaphoreRunCanceled,
			mobileDeviceSemaphoreRunNotFound:
			r.runQueue = removeFromQueue(r.runQueue, ticketID)
			delete(r.runTickets, ticketID)
			r.updateCount++
			r.markQueuePositionsDirty()
		}
	}

	r.flushQueuedPositionUpdates(ctx)
	r.shutdownCompleted = true
	return response, nil
}

func (r *mobileDeviceSemaphoreRuntime) notifyQueuedPositionUpdates(ctx workflow.Context) {
	for _, ticketID := range r.runQueue {
		state, ok := r.runTickets[ticketID]
		if !ok || state.Status != mobileDeviceSemaphoreRunQueued {
			continue
		}
		position, _ := r.runQueuePosition(ticketID)
		humanPosition := position + 1
		r.notifyGitHubPRComment(
			ctx,
			ticketID,
			state,
			mobileDeviceSemaphoreRunQueued,
			&humanPosition,
			"",
			"",
		)
	}
}

func (r *mobileDeviceSemaphoreRuntime) markQueuePositionsDirty() {
	r.queuePositionsDirty = true
}

func (r *mobileDeviceSemaphoreRuntime) flushQueuedPositionUpdates(ctx workflow.Context) {
	if !r.queuePositionsDirty {
		return
	}
	r.queuePositionsDirty = false
	r.notifyQueuedPositionUpdates(ctx)
}

func (r *mobileDeviceSemaphoreRuntime) cleanupRunTicketResources(
	ctx workflow.Context,
	state MobileDeviceSemaphoreRunTicketState,
) []string {
	if state.Request.Cleanup == nil {
		return nil
	}

	cleanupActivity := activities.NewCleanupMobileDeviceSemaphoreResourcesActivity()
	activityOptions := DefaultActivityOptions
	activityOptions.RetryPolicy = &temporal.RetryPolicy{MaximumAttempts: 1}
	activityCtx := workflow.WithActivityOptions(ctx, activityOptions)

	var result workflowengine.ActivityResult
	err := workflow.ExecuteActivity(activityCtx, cleanupActivity.Name(), workflowengine.ActivityInput{
		Payload: activities.CleanupMobileDeviceSemaphoreResourcesActivityInput{
			AppURL:  runTicketAppURL(state),
			Cleanup: state.Request.Cleanup,
		},
	}).
		Get(activityCtx, &result)
	if err != nil {
		return []string{err.Error()}
	}

	output, decodeErr := decodeCleanupMobileDeviceSemaphoreResourcesOutput(result.Output)
	if decodeErr != nil {
		return []string{decodeErr.Error()}
	}
	return output.CleanupFailures
}

func (r *mobileDeviceSemaphoreRuntime) cancelTrackedWorkflow(
	ctx workflow.Context,
	ticketID string,
	state MobileDeviceSemaphoreRunTicketState,
	reason string,
	response *MobileDeviceSemaphoreShutdownDeviceResponse,
) bool {
	if strings.TrimSpace(state.WorkflowID) == "" ||
		strings.TrimSpace(state.WorkflowNamespace) == "" {
		return false
	}

	cancelActivity := activities.NewCancelWorkflowActivity()
	activityOptions := DefaultActivityOptions
	activityOptions.RetryPolicy = &temporal.RetryPolicy{MaximumAttempts: 1}
	activityCtx := workflow.WithActivityOptions(ctx, activityOptions)

	signalActivity := activities.NewSignalWorkflowActivity()
	var signalResult workflowengine.ActivityResult
	signalErr := workflow.ExecuteActivity(activityCtx, signalActivity.Name(), workflowengine.ActivityInput{
		Payload: activities.SignalWorkflowActivityInput{
			WorkflowID:        state.WorkflowID,
			RunID:             state.RunID,
			WorkflowNamespace: state.WorkflowNamespace,
			SignalName:        pipelineinternal.PipelineCancellationPolicySignal,
			Payload: pipelineinternal.PipelineCancellationPolicy{
				Reason:               reason,
				SkipDeviceCleanup:    true,
				SkipDeviceCleanupIDs: []string{r.deviceID},
			},
		},
	}).
		Get(activityCtx, &signalResult)
	if signalErr != nil {
		response.PipelineCancelFailures = append(
			response.PipelineCancelFailures,
			fmt.Sprintf(
				"ticket %s pipeline cancellation policy signal failed: %v",
				ticketID,
				signalErr,
			),
		)
	} else {
		signalOutput, decodeErr := decodeSignalWorkflowOutput(signalResult.Output)
		if decodeErr != nil {
			response.PipelineCancelFailures = append(
				response.PipelineCancelFailures,
				fmt.Sprintf(
					"ticket %s pipeline cancellation policy signal decode failed: %v",
					ticketID,
					decodeErr,
				),
			)
		} else if signalOutput.Status != "SIGNALED" && signalOutput.Status != "NOT_FOUND" {
			response.PipelineCancelFailures = append(
				response.PipelineCancelFailures,
				fmt.Sprintf(
					"ticket %s pipeline cancellation policy signal returned unexpected status: %s",
					ticketID,
					signalOutput.Status,
				),
			)
		}
	}

	var result workflowengine.ActivityResult
	err := workflow.ExecuteActivity(activityCtx, cancelActivity.Name(), workflowengine.ActivityInput{
		Payload: activities.CancelWorkflowActivityInput{
			WorkflowID:        state.WorkflowID,
			RunID:             state.RunID,
			WorkflowNamespace: state.WorkflowNamespace,
			Reason:            reason,
		},
	}).
		Get(activityCtx, &result)
	if err != nil {
		response.PipelineCancelFailures = append(
			response.PipelineCancelFailures,
			fmt.Sprintf("ticket %s pipeline cancel failed: %v", ticketID, err),
		)
		return false
	}

	output, decodeErr := decodeCancelWorkflowOutput(result.Output)
	if decodeErr != nil {
		response.PipelineCancelFailures = append(
			response.PipelineCancelFailures,
			fmt.Sprintf("ticket %s pipeline cancel decode failed: %v", ticketID, decodeErr),
		)
		return false
	}

	return output.Canceled || output.Status == "NOT_FOUND"
}

func (r *mobileDeviceSemaphoreRuntime) signalRunCanceledForShutdown(
	ctx workflow.Context,
	ticketID string,
	state MobileDeviceSemaphoreRunTicketState,
	response *MobileDeviceSemaphoreShutdownDeviceResponse,
) int {
	if len(state.Request.RequiredDeviceIDs) == 0 || state.Request.LeaderDeviceID != r.deviceID {
		return 0
	}

	count := 0
	for _, deviceID := range sortedDeviceIDs(state.Request.RequiredDeviceIDs) {
		if deviceID == r.deviceID {
			continue
		}
		future := workflow.SignalExternalWorkflow(
			ctx,
			MobileDeviceSemaphoreWorkflowID(deviceID),
			"",
			MobileDeviceSemaphoreRunDoneSignalName,
			MobileDeviceSemaphoreRunDoneSignal{
				TicketID:       ticketID,
				WorkflowID:     state.WorkflowID,
				RunID:          state.RunID,
				WorkflowResult: "canceled",
			},
		)
		if err := future.Get(ctx, nil); err != nil {
			response.FollowerSignalFailures = append(
				response.FollowerSignalFailures,
				fmt.Sprintf("ticket %s signal to %s failed: %v", ticketID, deviceID, err),
			)
			continue
		}
		count++
	}
	return count
}

func (r *mobileDeviceSemaphoreRuntime) notifyGitHubPRComment(
	ctx workflow.Context,
	ticketID string,
	state MobileDeviceSemaphoreRunTicketState,
	status MobileDeviceSemaphoreRunStatus,
	position *int,
	workflowStatus string,
	errorMessage string,
) {
	notification := state.Request.Notification
	if notification == nil || notification.GitHubPR == nil {
		return
	}
	updateActivity := activities.NewUpdateGitHubPRCommentActivity()
	activityOptions := DefaultActivityOptions
	runnerType := ""
	if value := strings.TrimSpace(notification.GitHubPR.DeviceTypes[r.deviceID]); value != "" {
		runnerType = value
	} else if notification.GitHubPR.DeviceID == r.deviceID {
		runnerType = notification.GitHubPR.DeviceType
	}
	input := workflowengine.ActivityInput{
		Payload: activities.UpdateGitHubPRCommentInput{
			Repository:        notification.GitHubPR.Repository,
			PullRequestNumber: notification.GitHubPR.PullRequestNumber,
			CommitSHA:         notification.GitHubPR.CommitSHA,
			TicketID:          ticketID,
			Status:            string(status),
			Position:          position,
			PipelineID:        notification.GitHubPR.PipelineIdentifier,
			DeviceID:          r.deviceID,
			DeviceType:        runnerType,
			PipelineURL:       notification.GitHubPR.PipelineURL,
			AppURL:            notification.GitHubPR.AppURL,
			WorkflowID:        state.WorkflowID,
			RunID:             state.RunID,
			WorkflowStatus:    workflowStatus,
			ErrorMessage:      errorMessage,
			SectionTitle:      notification.GitHubPR.SectionTitle,
		},
	}
	workflow.Go(ctx, func(ctx workflow.Context) {
		logger := workflow.GetLogger(ctx)
		activityCtx := workflow.WithActivityOptions(ctx, activityOptions)
		if err := workflow.ExecuteActivity(activityCtx, updateActivity.Name(), input).
			Get(activityCtx, nil); err != nil {
			logger.Error(
				"failed to update github pr comment",
				"ticket_id",
				ticketID,
				"error",
				err,
			)
		}
	})
}

func (r *mobileDeviceSemaphoreRuntime) nextQueuedRunTicket() (string, MobileDeviceSemaphoreRunTicketState, bool) {
	for len(r.runQueue) > 0 {
		ticketID := r.runQueue[0]
		state, ok := r.runTickets[ticketID]
		if !ok || state.Status != mobileDeviceSemaphoreRunQueued {
			r.runQueue = r.runQueue[1:]
			r.updateCount++
			r.maybeScheduleContinue()
			r.markQueuePositionsDirty()
			continue
		}
		return ticketID, state, true
	}
	return "", MobileDeviceSemaphoreRunTicketState{}, false
}

func (r *mobileDeviceSemaphoreRuntime) allGrantsReceived(
	state MobileDeviceSemaphoreRunTicketState,
) bool {
	if len(state.Request.RequiredDeviceIDs) == 0 {
		return true
	}
	for _, deviceID := range state.Request.RequiredDeviceIDs {
		if !state.GrantedDeviceIDs[deviceID] {
			return false
		}
	}
	return true
}

func (r *mobileDeviceSemaphoreRuntime) sortedRunTicketIDs() []string {
	if len(r.runTickets) == 0 {
		return nil
	}
	ids := make([]string, 0, len(r.runTickets))
	for ticketID := range r.runTickets {
		ids = append(ids, ticketID)
	}
	sort.SliceStable(ids, func(i, j int) bool {
		left := r.runTickets[ids[i]]
		right := r.runTickets[ids[j]]
		return runTicketLess(left.Request, right.Request)
	})
	return ids
}

func ticketHasStartedWorkflow(state MobileDeviceSemaphoreRunTicketState) bool {
	return strings.TrimSpace(state.WorkflowID) != "" &&
		strings.TrimSpace(state.WorkflowNamespace) != ""
}

func runTicketAppURL(state MobileDeviceSemaphoreRunTicketState) string {
	if appURL, ok := state.Request.PipelineConfig["app_url"].(string); ok {
		return strings.TrimSpace(appURL)
	}
	if state.Request.Notification != nil && state.Request.Notification.GitHubPR != nil {
		return strings.TrimSpace(state.Request.Notification.GitHubPR.AppURL)
	}
	return ""
}

func sortedDeviceIDs(deviceIDs []string) []string {
	out := append([]string(nil), deviceIDs...)
	sort.Strings(out)
	return out
}

func (r *mobileDeviceSemaphoreRuntime) hasRunningTickets() bool {
	for _, state := range r.runTickets {
		if state.Status == mobileDeviceSemaphoreRunRunning {
			return true
		}
	}
	return false
}

func (r *mobileDeviceSemaphoreRuntime) hasFollowerStartingTickets() bool {
	for _, state := range r.runTickets {
		if state.Status == mobileDeviceSemaphoreRunStarting &&
			state.Request.LeaderDeviceID != r.deviceID {
			return true
		}
	}
	return false
}

func (r *mobileDeviceSemaphoreRuntime) runSlotsUsed() int {
	used := 0
	for _, state := range r.runTickets {
		switch state.Status {
		case mobileDeviceSemaphoreRunStarting, mobileDeviceSemaphoreRunRunning:
			used++
		}
	}
	return used
}

func (r *mobileDeviceSemaphoreRuntime) inFlightRunCount(ownerNamespace string) int {
	inFlight := 0
	for _, state := range r.runTickets {
		if state.Request.OwnerNamespace != ownerNamespace {
			continue
		}
		switch state.Status {
		case mobileDeviceSemaphoreRunQueued,
			mobileDeviceSemaphoreRunStarting,
			mobileDeviceSemaphoreRunRunning:
			inFlight++
		}
	}
	return inFlight
}

func (r *mobileDeviceSemaphoreRuntime) availableSlots() int {
	if r.capacity <= 0 {
		return 0
	}
	used := r.runSlotsUsed()
	if used >= r.capacity {
		return 0
	}
	return r.capacity - used
}

func decodeStartQueuedPipelineOutput(
	output any,
) (activities.StartQueuedPipelineActivityOutput, error) {
	switch value := output.(type) {
	case activities.StartQueuedPipelineActivityOutput:
		return value, nil
	case map[string]any:
		return decodeStartQueuedPipelineOutputMap(value)
	default:
		return activities.StartQueuedPipelineActivityOutput{}, newSemaphoreApplicationError(
			"unexpected activity output",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
}

func decodeStartQueuedPipelineOutputMap(
	value map[string]any,
) (activities.StartQueuedPipelineActivityOutput, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return activities.StartQueuedPipelineActivityOutput{}, newSemaphoreApplicationError(
			"failed to encode activity output",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
	var output activities.StartQueuedPipelineActivityOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		return activities.StartQueuedPipelineActivityOutput{}, newSemaphoreApplicationError(
			"failed to decode activity output",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
	return output, nil
}

func decodeCheckWorkflowClosedOutput(
	output any,
) (activities.CheckWorkflowClosedActivityOutput, error) {
	switch value := output.(type) {
	case activities.CheckWorkflowClosedActivityOutput:
		return value, nil
	case map[string]any:
		return decodeCheckWorkflowClosedOutputMap(value)
	default:
		return activities.CheckWorkflowClosedActivityOutput{}, newSemaphoreApplicationError(
			"unexpected activity output",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
}

func decodeCancelWorkflowOutput(output any) (activities.CancelWorkflowActivityOutput, error) {
	switch value := output.(type) {
	case activities.CancelWorkflowActivityOutput:
		return value, nil
	case map[string]any:
		raw, err := json.Marshal(value)
		if err != nil {
			return activities.CancelWorkflowActivityOutput{}, newSemaphoreApplicationError(
				"failed to encode cancel workflow output",
				MobileDeviceSemaphoreErrInvalidRequest,
			)
		}
		var decoded activities.CancelWorkflowActivityOutput
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return activities.CancelWorkflowActivityOutput{}, newSemaphoreApplicationError(
				"failed to decode cancel workflow output",
				MobileDeviceSemaphoreErrInvalidRequest,
			)
		}
		return decoded, nil
	default:
		return activities.CancelWorkflowActivityOutput{}, newSemaphoreApplicationError(
			"unexpected cancel workflow output",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
}

func decodeSignalWorkflowOutput(output any) (activities.SignalWorkflowActivityOutput, error) {
	switch value := output.(type) {
	case activities.SignalWorkflowActivityOutput:
		return value, nil
	case map[string]any:
		encoded, err := json.Marshal(value)
		if err != nil {
			return activities.SignalWorkflowActivityOutput{}, newSemaphoreApplicationError(
				fmt.Sprintf("failed to marshal signal workflow output: %v", err),
				MobileDeviceSemaphoreErrInvalidRequest,
			)
		}
		var decoded activities.SignalWorkflowActivityOutput
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			return activities.SignalWorkflowActivityOutput{}, newSemaphoreApplicationError(
				fmt.Sprintf("failed to decode signal workflow output: %v", err),
				MobileDeviceSemaphoreErrInvalidRequest,
			)
		}
		return decoded, nil
	default:
		return activities.SignalWorkflowActivityOutput{}, newSemaphoreApplicationError(
			fmt.Sprintf("unsupported signal workflow output type %T", output),
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
}

func decodeCleanupMobileDeviceSemaphoreResourcesOutput(
	output any,
) (activities.CleanupMobileDeviceSemaphoreResourcesActivityOutput, error) {
	switch value := output.(type) {
	case activities.CleanupMobileDeviceSemaphoreResourcesActivityOutput:
		return value, nil
	case map[string]any:
		raw, err := json.Marshal(value)
		if err != nil {
			return activities.CleanupMobileDeviceSemaphoreResourcesActivityOutput{}, newSemaphoreApplicationError(
				"failed to encode cleanup output",
				MobileDeviceSemaphoreErrInvalidRequest,
			)
		}
		var decoded activities.CleanupMobileDeviceSemaphoreResourcesActivityOutput
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return activities.CleanupMobileDeviceSemaphoreResourcesActivityOutput{}, newSemaphoreApplicationError(
				"failed to decode cleanup output",
				MobileDeviceSemaphoreErrInvalidRequest,
			)
		}
		return decoded, nil
	default:
		return activities.CleanupMobileDeviceSemaphoreResourcesActivityOutput{}, newSemaphoreApplicationError(
			"unexpected cleanup output",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
}

func (r *mobileDeviceSemaphoreRuntime) queryLeaderRunStatus(
	ctx workflow.Context,
	ticketID string,
	state MobileDeviceSemaphoreRunTicketState,
) (MobileDeviceSemaphoreRunStatusView, error) {
	queryActivity := activities.NewQueryMobileDeviceSemaphoreRunStatusActivity()
	activityOptions := DefaultActivityOptions
	activityOptions.RetryPolicy = &temporal.RetryPolicy{MaximumAttempts: 1}
	activityCtx := workflow.WithActivityOptions(ctx, activityOptions)

	input := workflowengine.ActivityInput{
		Payload: activities.QueryMobileDeviceSemaphoreRunStatusInput{
			DeviceID:       state.Request.LeaderDeviceID,
			OwnerNamespace: state.Request.OwnerNamespace,
			TicketID:       ticketID,
		},
	}
	var result workflowengine.ActivityResult
	if err := workflow.ExecuteActivity(activityCtx, queryActivity.Name(), input).
		Get(activityCtx, &result); err != nil {
		return MobileDeviceSemaphoreRunStatusView{}, err
	}

	return decodeRunStatusView(result.Output)
}

func decodeRunStatusView(output any) (MobileDeviceSemaphoreRunStatusView, error) {
	switch value := output.(type) {
	case MobileDeviceSemaphoreRunStatusView:
		return value, nil
	case map[string]any:
		return decodeRunStatusViewMap(value)
	default:
		return MobileDeviceSemaphoreRunStatusView{}, newSemaphoreApplicationError(
			"unexpected activity output",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
}

func decodeRunStatusViewMap(value map[string]any) (MobileDeviceSemaphoreRunStatusView, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return MobileDeviceSemaphoreRunStatusView{}, newSemaphoreApplicationError(
			"failed to encode run status",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
	var output MobileDeviceSemaphoreRunStatusView
	if err := json.Unmarshal(raw, &output); err != nil {
		return MobileDeviceSemaphoreRunStatusView{}, newSemaphoreApplicationError(
			"failed to decode run status",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
	return output, nil
}

func decodeCheckWorkflowClosedOutputMap(
	value map[string]any,
) (activities.CheckWorkflowClosedActivityOutput, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return activities.CheckWorkflowClosedActivityOutput{}, newSemaphoreApplicationError(
			"failed to encode activity output",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
	var output activities.CheckWorkflowClosedActivityOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		return activities.CheckWorkflowClosedActivityOutput{}, newSemaphoreApplicationError(
			"failed to decode activity output",
			MobileDeviceSemaphoreErrInvalidRequest,
		)
	}
	return output, nil
}

func (r *mobileDeviceSemaphoreRuntime) buildRunStatusView(
	ticketID string,
	state MobileDeviceSemaphoreRunTicketState,
) MobileDeviceSemaphoreRunStatusView {
	return MobileDeviceSemaphoreRunStatusView{
		TicketID:          ticketID,
		Status:            state.Status,
		LeaderDeviceID:    state.Request.LeaderDeviceID,
		RequiredDeviceIDs: copyStringSlice(state.Request.RequiredDeviceIDs),
		WorkflowID:        state.WorkflowID,
		RunID:             state.RunID,
		WorkflowNamespace: state.WorkflowNamespace,
		ErrorMessage:      state.ErrorMessage,
		Cleanup:           state.Request.Cleanup,
	}
}

func removeFromQueue(queue []string, requestID string) []string {
	for i, queuedID := range queue {
		if queuedID == requestID {
			return append(queue[:i], queue[i+1:]...)
		}
	}
	return queue
}

func (r *mobileDeviceSemaphoreRuntime) runQueuePosition(ticketID string) (int, int) {
	lineLen := len(r.runQueue)
	for i, queuedID := range r.runQueue {
		if queuedID == ticketID {
			return i, lineLen
		}
	}
	return 0, lineLen
}

func insertRunQueue(
	queue []string,
	ticketID string,
	tickets map[string]MobileDeviceSemaphoreRunTicketState,
) []string {
	queue = append(queue, ticketID)
	return sortRunQueue(queue, tickets)
}

func runTicketLess(
	left MobileDeviceSemaphoreEnqueueRunRequest,
	right MobileDeviceSemaphoreEnqueueRunRequest,
) bool {
	if left.EnqueuedAt.Before(right.EnqueuedAt) {
		return true
	}
	if right.EnqueuedAt.Before(left.EnqueuedAt) {
		return false
	}
	return left.TicketID < right.TicketID
}

func sortRunQueue(
	queue []string,
	tickets map[string]MobileDeviceSemaphoreRunTicketState,
) []string {
	sort.SliceStable(queue, func(i, j int) bool {
		leftID := queue[i]
		rightID := queue[j]
		left, leftOk := tickets[leftID]
		right, rightOk := tickets[rightID]
		if leftOk && rightOk {
			return runTicketLess(left.Request, right.Request)
		}
		if leftOk != rightOk {
			return leftOk
		}
		return leftID < rightID
	})
	return queue
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func copyQueue(queue []string) []string {
	if queue == nil {
		return nil
	}
	result := make([]string, len(queue))
	copy(result, queue)
	return result
}

func copyStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	result := make([]string, len(values))
	copy(result, values)
	return result
}

func copyRunTickets(
	tickets map[string]MobileDeviceSemaphoreRunTicketState,
) map[string]MobileDeviceSemaphoreRunTicketState {
	if tickets == nil {
		return nil
	}
	result := make(map[string]MobileDeviceSemaphoreRunTicketState, len(tickets))
	for key, value := range tickets {
		result[key] = copyRunTicketState(value)
	}
	return result
}

func copyRunTicketState(
	value MobileDeviceSemaphoreRunTicketState,
) MobileDeviceSemaphoreRunTicketState {
	copyValue := value
	copyValue.Request = copyRunTicketRequest(value.Request)
	copyValue.GrantedDeviceIDs = copyStringBoolMap(value.GrantedDeviceIDs)
	return copyValue
}

func copyRunTicketRequest(
	request MobileDeviceSemaphoreEnqueueRunRequest,
) MobileDeviceSemaphoreEnqueueRunRequest {
	copyRequest := request
	copyRequest.RequiredDeviceIDs = copyStringSlice(request.RequiredDeviceIDs)
	copyRequest.PipelineConfig = copyStringAnyMap(request.PipelineConfig)
	copyRequest.Memo = copyStringAnyMap(request.Memo)
	return copyRequest
}

func copyStringAnyMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func copyStringBoolMap(values map[string]bool) map[string]bool {
	if values == nil {
		return nil
	}
	result := make(map[string]bool, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
