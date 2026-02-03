// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	mobileRunnerSemaphoreMaxUpdateBatches = 1000
	queuePreviewLimit                     = 5
	runCompletionCheckInterval            = 45 * time.Second
)

type MobileRunnerSemaphoreWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

const (
	mobileRunnerSemaphoreRequestQueued   MobileRunnerSemaphoreRequestStatus = workflowengine.MobileRunnerSemaphoreRequestQueued
	mobileRunnerSemaphoreRequestGranted  MobileRunnerSemaphoreRequestStatus = workflowengine.MobileRunnerSemaphoreRequestGranted
	mobileRunnerSemaphoreRequestTimedOut MobileRunnerSemaphoreRequestStatus = workflowengine.MobileRunnerSemaphoreRequestTimedOut
	mobileRunnerSemaphoreRunQueued       MobileRunnerSemaphoreRunStatus     = workflowengine.MobileRunnerSemaphoreRunQueued
	mobileRunnerSemaphoreRunStarting     MobileRunnerSemaphoreRunStatus     = workflowengine.MobileRunnerSemaphoreRunStarting
	mobileRunnerSemaphoreRunRunning      MobileRunnerSemaphoreRunStatus     = workflowengine.MobileRunnerSemaphoreRunRunning
	mobileRunnerSemaphoreRunFailed       MobileRunnerSemaphoreRunStatus     = workflowengine.MobileRunnerSemaphoreRunFailed
	mobileRunnerSemaphoreRunCanceled     MobileRunnerSemaphoreRunStatus     = workflowengine.MobileRunnerSemaphoreRunCanceled
	mobileRunnerSemaphoreRunNotFound     MobileRunnerSemaphoreRunStatus     = workflowengine.MobileRunnerSemaphoreRunNotFound
)

func NewMobileRunnerSemaphoreWorkflow() *MobileRunnerSemaphoreWorkflow {
	w := &MobileRunnerSemaphoreWorkflow{}
	w.WorkflowFunc = w.ExecuteWorkflow
	return w
}

func (MobileRunnerSemaphoreWorkflow) Name() string {
	return MobileRunnerSemaphoreWorkflowName
}

func (MobileRunnerSemaphoreWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *MobileRunnerSemaphoreWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.WorkflowFunc(ctx, input)
}

func (w *MobileRunnerSemaphoreWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	payload, err := workflowengine.DecodePayload[MobileRunnerSemaphoreWorkflowInput](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	runtime, err := newMobileRunnerSemaphoreRuntime(ctx, payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerQueryHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerRunStatusHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerAcquireHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerReleaseHandler(); err != nil {
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

	runtime.startRunSignalHandlers()
	runtime.startRunStarter()
	runtime.startRunSafetyNet()

	return runtime.awaitContinue()
}

type mobileRunnerSemaphoreRuntime struct {
	ctx                 workflow.Context
	runnerID            string
	capacity            int
	holders             map[string]MobileRunnerSemaphoreHolder
	queue               []string
	requests            map[string]MobileRunnerSemaphoreRequestState
	runQueue            []string
	runTickets          map[string]MobileRunnerSemaphoreRunTicketState
	lastGrantAt         *time.Time
	updateCount         int
	shouldContinue      bool
	continueInput       workflowengine.WorkflowInput
	runStarterRequested bool
}

func newMobileRunnerSemaphoreRuntime(
	ctx workflow.Context,
	payload MobileRunnerSemaphoreWorkflowInput,
) (*mobileRunnerSemaphoreRuntime, error) {
	if payload.RunnerID == "" {
		return nil, temporal.NewApplicationError(
			"runner_id is required",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}

	runtime := &mobileRunnerSemaphoreRuntime{
		ctx:        ctx,
		runnerID:   payload.RunnerID,
		capacity:   payload.Capacity,
		holders:    map[string]MobileRunnerSemaphoreHolder{},
		queue:      []string{},
		requests:   map[string]MobileRunnerSemaphoreRequestState{},
		runQueue:   []string{},
		runTickets: map[string]MobileRunnerSemaphoreRunTicketState{},
	}

	runtime.applyPayloadState(payload)
	runtime.normalizeState()

	return runtime, nil
}

func (r *mobileRunnerSemaphoreRuntime) applyPayloadState(payload MobileRunnerSemaphoreWorkflowInput) {
	if payload.Capacity <= 0 {
		r.capacity = 1
	}

	if payload.State == nil {
		return
	}

	if payload.State.Capacity > 0 {
		r.capacity = payload.State.Capacity
	}
	r.holders = payload.State.Holders
	r.queue = payload.State.Queue
	r.requests = payload.State.Requests
	r.runQueue = payload.State.RunQueue
	r.runTickets = payload.State.RunTickets
	r.lastGrantAt = payload.State.LastGrantAt
	r.updateCount = payload.State.UpdateCount
}

func (r *mobileRunnerSemaphoreRuntime) normalizeState() {
	if r.capacity <= 0 {
		r.capacity = 1
	}
	if r.holders == nil {
		r.holders = map[string]MobileRunnerSemaphoreHolder{}
	}
	if r.requests == nil {
		r.requests = map[string]MobileRunnerSemaphoreRequestState{}
	}
	if r.queue == nil {
		r.queue = []string{}
	}
	if r.runQueue == nil {
		r.runQueue = []string{}
	}
	if r.runTickets == nil {
		r.runTickets = map[string]MobileRunnerSemaphoreRunTicketState{}
	}
}

func (r *mobileRunnerSemaphoreRuntime) registerQueryHandler() error {
	return workflow.SetQueryHandler(
		r.ctx,
		MobileRunnerSemaphoreStateQuery,
		func() (MobileRunnerSemaphoreStateView, error) {
			holdersView := buildHoldersView(r.holders)
			queuePreview := buildQueuePreview(r.queue, r.requests)

			var currentHolder *MobileRunnerSemaphoreHolder
			if r.capacity == 1 && len(holdersView) == 1 {
				holderCopy := holdersView[0]
				currentHolder = &holderCopy
			}

			return MobileRunnerSemaphoreStateView{
				RunnerID:      r.runnerID,
				Capacity:      r.capacity,
				CurrentHolder: currentHolder,
				Holders:       holdersView,
				QueueLen:      len(r.queue),
				QueuePreview:  queuePreview,
				LastGrantAt:   r.lastGrantAt,
			}, nil
		},
	)
}

func (r *mobileRunnerSemaphoreRuntime) registerRunStatusHandler() error {
	return workflow.SetQueryHandler(
		r.ctx,
		MobileRunnerSemaphoreRunStatusQuery,
		func(ownerNamespace, ticketID string) (MobileRunnerSemaphoreRunStatusView, error) {
			return r.handleRunStatusQuery(ownerNamespace, ticketID)
		},
	)
}

func (r *mobileRunnerSemaphoreRuntime) registerAcquireHandler() error {
	return workflow.SetUpdateHandler(
		r.ctx,
		MobileRunnerSemaphoreAcquireUpdate,
		func(ctx workflow.Context, req MobileRunnerSemaphoreAcquireRequest) (MobileRunnerSemaphorePermit, error) {
			return r.handleAcquire(ctx, req)
		},
	)
}

func (r *mobileRunnerSemaphoreRuntime) registerReleaseHandler() error {
	return workflow.SetUpdateHandler(
		r.ctx,
		MobileRunnerSemaphoreReleaseUpdate,
		func(ctx workflow.Context, req MobileRunnerSemaphoreReleaseRequest) (MobileRunnerSemaphoreReleaseResult, error) {
			return r.handleRelease(ctx, req)
		},
	)
}

func (r *mobileRunnerSemaphoreRuntime) registerEnqueueRunHandler() error {
	return workflow.SetUpdateHandler(
		r.ctx,
		MobileRunnerSemaphoreEnqueueRunUpdate,
		func(_ workflow.Context, req MobileRunnerSemaphoreEnqueueRunRequest) (MobileRunnerSemaphoreEnqueueRunResponse, error) {
			return r.handleEnqueueRun(req)
		},
	)
}

func (r *mobileRunnerSemaphoreRuntime) registerCancelRunHandler() error {
	return workflow.SetUpdateHandler(
		r.ctx,
		MobileRunnerSemaphoreCancelRunUpdate,
		func(_ workflow.Context, req MobileRunnerSemaphoreRunCancelRequest) (MobileRunnerSemaphoreRunStatusView, error) {
			return r.handleCancelRun(req)
		},
	)
}

func (r *mobileRunnerSemaphoreRuntime) registerRunDoneHandler() error {
	return workflow.SetUpdateHandler(
		r.ctx,
		MobileRunnerSemaphoreRunDoneUpdate,
		func(ctx workflow.Context, req MobileRunnerSemaphoreRunDoneRequest) (MobileRunnerSemaphoreRunStatusView, error) {
			return r.handleRunDone(ctx, req)
		},
	)
}

func (r *mobileRunnerSemaphoreRuntime) startRunSignalHandlers() {
	startRunSignalHandler(
		r.ctx,
		MobileRunnerSemaphoreRunGrantedSignalName,
		func(ctx workflow.Context, signal MobileRunnerSemaphoreRunGrantedSignal) {
			r.handleRunGrantedSignal(signal)
		},
	)
	startRunSignalHandler(
		r.ctx,
		MobileRunnerSemaphoreRunStartedSignalName,
		r.handleRunStartedSignal,
	)
	startRunSignalHandler(
		r.ctx,
		MobileRunnerSemaphoreRunDoneSignalName,
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

func (r *mobileRunnerSemaphoreRuntime) startRunStarter() {
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
			if err := r.processRunQueue(ctx); err != nil {
				logger.Error("run starter failed", "error", err)
			}
		}
	})
	r.requestRunStart()
}

func (r *mobileRunnerSemaphoreRuntime) startRunSafetyNet() {
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

func (r *mobileRunnerSemaphoreRuntime) requestRunStart() {
	r.runStarterRequested = true
}

func (r *mobileRunnerSemaphoreRuntime) handleEnqueueRun(
	req MobileRunnerSemaphoreEnqueueRunRequest,
) (MobileRunnerSemaphoreEnqueueRunResponse, error) {
	if req.TicketID == "" || req.OwnerNamespace == "" {
		return MobileRunnerSemaphoreEnqueueRunResponse{}, temporal.NewApplicationError(
			"ticket_id and owner_namespace are required",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}
	if req.RunnerID == "" || req.RunnerID != r.runnerID {
		return MobileRunnerSemaphoreEnqueueRunResponse{}, temporal.NewApplicationError(
			"runner_id must match semaphore runner",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}
	if req.EnqueuedAt.IsZero() {
		return MobileRunnerSemaphoreEnqueueRunResponse{}, temporal.NewApplicationError(
			"enqueued_at is required",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}
	if len(req.RequiredRunnerIDs) == 0 || req.LeaderRunnerID == "" {
		return MobileRunnerSemaphoreEnqueueRunResponse{}, temporal.NewApplicationError(
			"required_runner_ids and leader_runner_id are required",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}
	if !containsString(req.RequiredRunnerIDs, req.LeaderRunnerID) {
		return MobileRunnerSemaphoreEnqueueRunResponse{}, temporal.NewApplicationError(
			"leader_runner_id must be included in required_runner_ids",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}

	if existing, ok := r.runTickets[req.TicketID]; ok {
		if existing.Request.OwnerNamespace != req.OwnerNamespace {
			return MobileRunnerSemaphoreEnqueueRunResponse{}, temporal.NewApplicationError(
				"ticket owner mismatch",
				MobileRunnerSemaphoreErrInvalidRequest,
			)
		}
		view := r.buildRunStatusView(req.TicketID, existing)
		if view.Status == mobileRunnerSemaphoreRunQueued {
			position, lineLen := r.runQueuePosition(req.TicketID)
			view.Position = position
			view.LineLen = lineLen
		}
		return MobileRunnerSemaphoreEnqueueRunResponse{
			TicketID: view.TicketID,
			Status:   view.Status,
			Position: view.Position,
			LineLen:  view.LineLen,
		}, nil
	}

	r.runTickets[req.TicketID] = MobileRunnerSemaphoreRunTicketState{
		Request: req,
		Status:  mobileRunnerSemaphoreRunQueued,
	}
	r.runQueue = insertRunQueue(r.runQueue, req.TicketID, r.runTickets)
	position, lineLen := r.runQueuePosition(req.TicketID)

	r.updateCount++
	r.maybeScheduleContinue()
	r.requestRunStart()

	return MobileRunnerSemaphoreEnqueueRunResponse{
		TicketID: req.TicketID,
		Status:   mobileRunnerSemaphoreRunQueued,
		Position: position,
		LineLen:  lineLen,
	}, nil
}

func (r *mobileRunnerSemaphoreRuntime) handleCancelRun(
	req MobileRunnerSemaphoreRunCancelRequest,
) (MobileRunnerSemaphoreRunStatusView, error) {
	if req.TicketID == "" || req.OwnerNamespace == "" {
		return MobileRunnerSemaphoreRunStatusView{}, temporal.NewApplicationError(
			"ticket_id and owner_namespace are required",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}

	state, ok := r.runTickets[req.TicketID]
	if !ok || state.Request.OwnerNamespace != req.OwnerNamespace {
		return MobileRunnerSemaphoreRunStatusView{
			TicketID: req.TicketID,
			Status:   mobileRunnerSemaphoreRunNotFound,
		}, nil
	}

	switch state.Status {
	case mobileRunnerSemaphoreRunQueued, mobileRunnerSemaphoreRunStarting:
		r.runQueue = removeFromQueue(r.runQueue, req.TicketID)
		delete(r.runTickets, req.TicketID)
		r.updateCount++
		r.maybeScheduleContinue()
		r.requestRunStart()
		return MobileRunnerSemaphoreRunStatusView{
			TicketID: req.TicketID,
			Status:   mobileRunnerSemaphoreRunNotFound,
		}, nil
	case mobileRunnerSemaphoreRunRunning:
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

func (r *mobileRunnerSemaphoreRuntime) handleRunDone(
	ctx workflow.Context,
	req MobileRunnerSemaphoreRunDoneRequest,
) (MobileRunnerSemaphoreRunStatusView, error) {
	if req.TicketID == "" || req.OwnerNamespace == "" {
		return MobileRunnerSemaphoreRunStatusView{}, temporal.NewApplicationError(
			"ticket_id and owner_namespace are required",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}

	state, ok := r.runTickets[req.TicketID]
	if !ok || state.Request.OwnerNamespace != req.OwnerNamespace {
		return MobileRunnerSemaphoreRunStatusView{
			TicketID: req.TicketID,
			Status:   mobileRunnerSemaphoreRunNotFound,
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
	signalFollowers := state.Request.LeaderRunnerID == r.runnerID
	r.finalizeRunTicket(ctx, req.TicketID, state, workflowID, runID, signalFollowers)

	return MobileRunnerSemaphoreRunStatusView{
		TicketID: req.TicketID,
		Status:   mobileRunnerSemaphoreRunNotFound,
	}, nil
}

func (r *mobileRunnerSemaphoreRuntime) handleRunStatusQuery(
	ownerNamespace,
	ticketID string,
) (MobileRunnerSemaphoreRunStatusView, error) {
	if ownerNamespace == "" || ticketID == "" {
		return MobileRunnerSemaphoreRunStatusView{
			TicketID: ticketID,
			Status:   mobileRunnerSemaphoreRunNotFound,
		}, nil
	}

	state, ok := r.runTickets[ticketID]
	if !ok || state.Request.OwnerNamespace != ownerNamespace {
		return MobileRunnerSemaphoreRunStatusView{
			TicketID: ticketID,
			Status:   mobileRunnerSemaphoreRunNotFound,
		}, nil
	}

	view := r.buildRunStatusView(ticketID, state)
	if view.Status == mobileRunnerSemaphoreRunQueued {
		position, lineLen := r.runQueuePosition(ticketID)
		view.Position = position
		view.LineLen = lineLen
	}

	return view, nil
}
func (r *mobileRunnerSemaphoreRuntime) handleAcquire(
	ctx workflow.Context,
	req MobileRunnerSemaphoreAcquireRequest,
) (MobileRunnerSemaphorePermit, error) {
	if req.RequestID == "" || req.LeaseID == "" {
		return MobileRunnerSemaphorePermit{}, temporal.NewApplicationError(
			"request_id and lease_id are required",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}

	if existing, ok := r.requests[req.RequestID]; ok {
		switch existing.Status {
		case mobileRunnerSemaphoreRequestGranted:
			return buildPermit(r.runnerID, existing), nil
		case mobileRunnerSemaphoreRequestTimedOut:
			return MobileRunnerSemaphorePermit{}, temporal.NewApplicationError(
				"request timed out",
				MobileRunnerSemaphoreErrTimeout,
				req.RequestID,
			)
		}
	} else {
		r.requests[req.RequestID] = MobileRunnerSemaphoreRequestState{
			Request:     req,
			Status:      mobileRunnerSemaphoreRequestQueued,
			RequestedAt: workflow.Now(ctx),
		}
		r.queue = append(r.queue, req.RequestID)
		r.grantAvailable(ctx)
	}

	r.updateCount++
	r.maybeScheduleContinue()
	r.requestRunStart()

	if req.WaitTimeout > 0 {
		granted, err := workflow.AwaitWithTimeout(ctx, req.WaitTimeout, func() bool {
			state, ok := r.requests[req.RequestID]
			return ok && state.Status == mobileRunnerSemaphoreRequestGranted
		})
		if err != nil {
			return MobileRunnerSemaphorePermit{}, err
		}

		state, ok := r.requests[req.RequestID]
		if !ok {
			return MobileRunnerSemaphorePermit{}, temporal.NewApplicationError(
				"request not found",
				MobileRunnerSemaphoreErrInvalidRequest,
				req.RequestID,
			)
		}

		if state.Status == mobileRunnerSemaphoreRequestGranted {
			return buildPermit(r.runnerID, state), nil
		}

		if !granted {
			r.queue = removeFromQueue(r.queue, req.RequestID)
			state.Status = mobileRunnerSemaphoreRequestTimedOut
			r.requests[req.RequestID] = state
			delete(r.holders, req.LeaseID)
			r.grantAvailable(ctx)
			return MobileRunnerSemaphorePermit{}, temporal.NewApplicationError(
				"acquire timeout",
				MobileRunnerSemaphoreErrTimeout,
				req.RequestID,
			)
		}
	}

	if err := workflow.Await(ctx, func() bool {
		state, ok := r.requests[req.RequestID]
		return ok && state.Status == mobileRunnerSemaphoreRequestGranted
	}); err != nil {
		return MobileRunnerSemaphorePermit{}, err
	}

	return buildPermit(r.runnerID, r.requests[req.RequestID]), nil
}

func (r *mobileRunnerSemaphoreRuntime) handleRelease(
	ctx workflow.Context,
	req MobileRunnerSemaphoreReleaseRequest,
) (MobileRunnerSemaphoreReleaseResult, error) {
	if req.LeaseID == "" {
		return MobileRunnerSemaphoreReleaseResult{}, temporal.NewApplicationError(
			"lease_id is required",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}

	// For idempotent acquires where the request already exists in a queued state,
	// we intentionally do not re-enqueue or short-circuit here. Instead, the call
	// falls through to the wait logic below, which will block until the existing
	// request is granted or times out.
	if _, ok := r.holders[req.LeaseID]; !ok {
		return MobileRunnerSemaphoreReleaseResult{Released: false}, nil
	}

	delete(r.holders, req.LeaseID)
	r.grantAvailable(ctx)

	r.updateCount++
	r.maybeScheduleContinue()
	r.requestRunStart()

	return MobileRunnerSemaphoreReleaseResult{Released: true}, nil
}

func (r *mobileRunnerSemaphoreRuntime) grantAvailable(ctx workflow.Context) {
	now := workflow.Now(ctx)
	for r.availableSlots() > 0 && len(r.queue) > 0 {
		requestID := r.queue[0]
		r.queue = r.queue[1:]
		request, ok := r.requests[requestID]
		if !ok || request.Status != mobileRunnerSemaphoreRequestQueued {
			continue
		}

		request.Status = mobileRunnerSemaphoreRequestGranted
		request.GrantedAt = now
		request.QueueWaitMs = now.Sub(request.RequestedAt).Milliseconds()
		r.requests[requestID] = request

		r.holders[request.Request.LeaseID] = MobileRunnerSemaphoreHolder{
			LeaseID:         request.Request.LeaseID,
			RequestID:       requestID,
			OwnerNamespace:  request.Request.OwnerNamespace,
			OwnerWorkflowID: request.Request.OwnerWorkflowID,
			OwnerRunID:      request.Request.OwnerRunID,
			GrantedAt:       request.GrantedAt,
			QueueWaitMs:     request.QueueWaitMs,
		}

		timeCopy := now
		r.lastGrantAt = &timeCopy
	}
}

func (r *mobileRunnerSemaphoreRuntime) maybeScheduleContinue() {
	if r.shouldContinue || r.updateCount < mobileRunnerSemaphoreMaxUpdateBatches {
		return
	}

	stateCopy := MobileRunnerSemaphoreWorkflowState{
		Capacity:    r.capacity,
		Holders:     copyHolders(r.holders),
		Queue:       copyQueue(r.queue),
		Requests:    copyRequests(r.requests),
		RunQueue:    copyQueue(r.runQueue),
		RunTickets:  copyRunTickets(r.runTickets),
		LastGrantAt: copyTimePtr(r.lastGrantAt),
		UpdateCount: 0,
	}

	r.continueInput = workflowengine.WorkflowInput{
		Payload: MobileRunnerSemaphoreWorkflowInput{
			RunnerID: r.runnerID,
			Capacity: r.capacity,
			State:    &stateCopy,
		},
	}
	r.shouldContinue = true
}

func (r *mobileRunnerSemaphoreRuntime) awaitContinue() (workflowengine.WorkflowResult, error) {
	if err := workflow.Await(r.ctx, func() bool { return r.shouldContinue }); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if r.shouldContinue {
		return workflowengine.WorkflowResult{}, workflow.NewContinueAsNewError(
			r.ctx,
			MobileRunnerSemaphoreWorkflowName,
			r.continueInput,
		)
	}

	return workflowengine.WorkflowResult{}, nil
}

func (r *mobileRunnerSemaphoreRuntime) processRunQueue(ctx workflow.Context) error {
	if err := r.startReadyRuns(ctx); err != nil {
		return err
	}

	for r.availableSlots() > 0 {
		ticketID, state, ok := r.nextQueuedRunTicket()
		if !ok {
			return nil
		}
		if err := r.grantRunTicket(ctx, ticketID, state); err != nil {
			return err
		}
		if err := r.startReadyRuns(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *mobileRunnerSemaphoreRuntime) startReadyRuns(ctx workflow.Context) error {
	ticketIDs := r.sortedRunTicketIDs()
	for _, ticketID := range ticketIDs {
		state, ok := r.runTickets[ticketID]
		if !ok || state.Status != mobileRunnerSemaphoreRunStarting {
			continue
		}
		if state.Request.LeaderRunnerID != r.runnerID {
			continue
		}
		if state.WorkflowID != "" {
			continue
		}
		if !r.allGrantsReceived(state) {
			continue
		}
		if err := r.startPipelineForTicket(ctx, ticketID, state); err != nil {
			return err
		}
	}
	return nil
}

func (r *mobileRunnerSemaphoreRuntime) grantRunTicket(
	ctx workflow.Context,
	ticketID string,
	state MobileRunnerSemaphoreRunTicketState,
) error {
	if state.Status != mobileRunnerSemaphoreRunQueued {
		return nil
	}

	r.runQueue = removeFromQueue(r.runQueue, ticketID)

	now := workflow.Now(ctx)
	state.Status = mobileRunnerSemaphoreRunStarting
	state.StartedAt = &now
	if state.GrantedRunnerIDs == nil {
		state.GrantedRunnerIDs = map[string]bool{}
	}
	state.GrantedRunnerIDs[r.runnerID] = true
	r.runTickets[ticketID] = state
	r.updateCount++
	r.maybeScheduleContinue()

	if state.Request.LeaderRunnerID != r.runnerID {
		if err := r.signalRunGranted(ctx, state.Request.LeaderRunnerID, ticketID); err != nil {
			r.markRunTicketFailed(ticketID, state, err)
		}
		return nil
	}

	if r.allGrantsReceived(state) {
		return r.startPipelineForTicket(ctx, ticketID, state)
	}

	return nil
}

func (r *mobileRunnerSemaphoreRuntime) startPipelineForTicket(
	ctx workflow.Context,
	ticketID string,
	state MobileRunnerSemaphoreRunTicketState,
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
			RequiredRunnerIDs:  state.Request.RequiredRunnerIDs,
			LeaderRunnerID:     state.Request.LeaderRunnerID,
			PipelineIdentifier: state.Request.PipelineIdentifier,
			YAML:               state.Request.YAML,
			PipelineConfig:     state.Request.PipelineConfig,
			Memo:               state.Request.Memo,
		},
	}

	if err := workflow.ExecuteActivity(activityCtx, startActivity.Name(), input).
		Get(activityCtx, &result); err != nil {
		r.markRunTicketFailed(ticketID, state, err)
		r.signalRunDone(ctx, ticketID, state.Request.RequiredRunnerIDs, "", "")
		return err
	}

	output, err := decodeStartQueuedPipelineOutput(result.Output)
	if err != nil {
		r.markRunTicketFailed(ticketID, state, err)
		r.signalRunDone(ctx, ticketID, state.Request.RequiredRunnerIDs, "", "")
		return err
	}

	state.Status = mobileRunnerSemaphoreRunRunning
	state.WorkflowID = output.WorkflowID
	state.RunID = output.RunID
	state.WorkflowNamespace = output.WorkflowNamespace
	startedAt := workflow.Now(ctx)
	state.StartedAt = &startedAt
	r.runTickets[ticketID] = state
	r.updateCount++
	r.maybeScheduleContinue()

	r.signalRunStarted(ctx, ticketID, state.Request.RequiredRunnerIDs, output)

	return nil
}

func (r *mobileRunnerSemaphoreRuntime) markRunTicketFailed(
	ticketID string,
	state MobileRunnerSemaphoreRunTicketState,
	err error,
) {
	state.Status = mobileRunnerSemaphoreRunFailed
	if err != nil {
		state.ErrorMessage = err.Error()
	}
	r.runTickets[ticketID] = state
	r.updateCount++
	r.maybeScheduleContinue()
	r.requestRunStart()
}

func (r *mobileRunnerSemaphoreRuntime) checkRunCompletion(ctx workflow.Context) {
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
		if !ok || state.Status != mobileRunnerSemaphoreRunRunning {
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

		signalFollowers := state.Request.LeaderRunnerID == r.runnerID
		r.finalizeRunTicket(ctx, ticketID, state, state.WorkflowID, state.RunID, signalFollowers)
	}
}

func (r *mobileRunnerSemaphoreRuntime) finalizeRunTicket(
	ctx workflow.Context,
	ticketID string,
	state MobileRunnerSemaphoreRunTicketState,
	workflowID string,
	runID string,
	signalFollowers bool,
) {
	if signalFollowers {
		r.signalRunDone(ctx, ticketID, state.Request.RequiredRunnerIDs, workflowID, runID)
	}
	r.runQueue = removeFromQueue(r.runQueue, ticketID)
	delete(r.runTickets, ticketID)
	r.updateCount++
	r.maybeScheduleContinue()
	r.requestRunStart()
}

func (r *mobileRunnerSemaphoreRuntime) signalRunGranted(
	ctx workflow.Context,
	leaderRunnerID string,
	ticketID string,
) error {
	future := workflow.SignalExternalWorkflow(
		ctx,
		MobileRunnerSemaphoreWorkflowID(leaderRunnerID),
		"",
		MobileRunnerSemaphoreRunGrantedSignalName,
		MobileRunnerSemaphoreRunGrantedSignal{
			TicketID: ticketID,
			RunnerID: r.runnerID,
		},
	)
	return future.Get(ctx, nil)
}

func (r *mobileRunnerSemaphoreRuntime) signalRunStarted(
	ctx workflow.Context,
	ticketID string,
	requiredRunnerIDs []string,
	output activities.StartQueuedPipelineActivityOutput,
) {
	for _, runnerID := range requiredRunnerIDs {
		if runnerID == r.runnerID {
			continue
		}
		future := workflow.SignalExternalWorkflow(
			ctx,
			MobileRunnerSemaphoreWorkflowID(runnerID),
			"",
			MobileRunnerSemaphoreRunStartedSignalName,
			MobileRunnerSemaphoreRunStartedSignal{
				TicketID:          ticketID,
				WorkflowID:        output.WorkflowID,
				RunID:             output.RunID,
				WorkflowNamespace: output.WorkflowNamespace,
			},
		)
		_ = future.Get(ctx, nil)
	}
}

func (r *mobileRunnerSemaphoreRuntime) signalRunDone(
	ctx workflow.Context,
	ticketID string,
	requiredRunnerIDs []string,
	workflowID string,
	runID string,
) {
	for _, runnerID := range requiredRunnerIDs {
		if runnerID == r.runnerID {
			continue
		}
		future := workflow.SignalExternalWorkflow(
			ctx,
			MobileRunnerSemaphoreWorkflowID(runnerID),
			"",
			MobileRunnerSemaphoreRunDoneSignalName,
			MobileRunnerSemaphoreRunDoneSignal{
				TicketID:   ticketID,
				WorkflowID: workflowID,
				RunID:      runID,
			},
		)
		_ = future.Get(ctx, nil)
	}
}

func (r *mobileRunnerSemaphoreRuntime) handleRunGrantedSignal(
	signal MobileRunnerSemaphoreRunGrantedSignal,
) {
	if signal.TicketID == "" || signal.RunnerID == "" {
		return
	}
	state, ok := r.runTickets[signal.TicketID]
	if !ok {
		return
	}
	if state.GrantedRunnerIDs == nil {
		state.GrantedRunnerIDs = map[string]bool{}
	}
	state.GrantedRunnerIDs[signal.RunnerID] = true
	r.runTickets[signal.TicketID] = state
	r.updateCount++
	r.maybeScheduleContinue()
	r.requestRunStart()
}

func (r *mobileRunnerSemaphoreRuntime) handleRunStartedSignal(
	ctx workflow.Context,
	signal MobileRunnerSemaphoreRunStartedSignal,
) {
	if signal.TicketID == "" {
		return
	}
	state, ok := r.runTickets[signal.TicketID]
	if !ok {
		return
	}
	state.Status = mobileRunnerSemaphoreRunRunning
	state.WorkflowID = signal.WorkflowID
	state.RunID = signal.RunID
	state.WorkflowNamespace = signal.WorkflowNamespace
	startedAt := workflow.Now(ctx)
	state.StartedAt = &startedAt
	r.runTickets[signal.TicketID] = state
	r.updateCount++
	r.maybeScheduleContinue()
}

func (r *mobileRunnerSemaphoreRuntime) handleRunDoneSignal(
	ctx workflow.Context,
	signal MobileRunnerSemaphoreRunDoneSignal,
) {
	if signal.TicketID == "" {
		return
	}
	state, ok := r.runTickets[signal.TicketID]
	if !ok {
		return
	}
	r.finalizeRunTicket(ctx, signal.TicketID, state, signal.WorkflowID, signal.RunID, false)
}

func (r *mobileRunnerSemaphoreRuntime) nextQueuedRunTicket() (string, MobileRunnerSemaphoreRunTicketState, bool) {
	for len(r.runQueue) > 0 {
		ticketID := r.runQueue[0]
		state, ok := r.runTickets[ticketID]
		if !ok || state.Status != mobileRunnerSemaphoreRunQueued {
			r.runQueue = r.runQueue[1:]
			r.updateCount++
			r.maybeScheduleContinue()
			continue
		}
		return ticketID, state, true
	}
	return "", MobileRunnerSemaphoreRunTicketState{}, false
}

func (r *mobileRunnerSemaphoreRuntime) allGrantsReceived(
	state MobileRunnerSemaphoreRunTicketState,
) bool {
	if len(state.Request.RequiredRunnerIDs) == 0 {
		return true
	}
	for _, runnerID := range state.Request.RequiredRunnerIDs {
		if !state.GrantedRunnerIDs[runnerID] {
			return false
		}
	}
	return true
}

func (r *mobileRunnerSemaphoreRuntime) sortedRunTicketIDs() []string {
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

func (r *mobileRunnerSemaphoreRuntime) hasRunningTickets() bool {
	for _, state := range r.runTickets {
		if state.Status == mobileRunnerSemaphoreRunRunning {
			return true
		}
	}
	return false
}

func (r *mobileRunnerSemaphoreRuntime) runSlotsUsed() int {
	used := 0
	for _, state := range r.runTickets {
		switch state.Status {
		case mobileRunnerSemaphoreRunStarting, mobileRunnerSemaphoreRunRunning:
			used++
		}
	}
	return used
}

func (r *mobileRunnerSemaphoreRuntime) availableSlots() int {
	if r.capacity <= 0 {
		return 0
	}
	used := len(r.holders) + r.runSlotsUsed()
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
		return activities.StartQueuedPipelineActivityOutput{}, temporal.NewApplicationError(
			"unexpected activity output",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}
}

func decodeStartQueuedPipelineOutputMap(
	value map[string]any,
) (activities.StartQueuedPipelineActivityOutput, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return activities.StartQueuedPipelineActivityOutput{}, temporal.NewApplicationError(
			"failed to encode activity output",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}
	var output activities.StartQueuedPipelineActivityOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		return activities.StartQueuedPipelineActivityOutput{}, temporal.NewApplicationError(
			"failed to decode activity output",
			MobileRunnerSemaphoreErrInvalidRequest,
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
		return activities.CheckWorkflowClosedActivityOutput{}, temporal.NewApplicationError(
			"unexpected activity output",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}
}

func decodeCheckWorkflowClosedOutputMap(
	value map[string]any,
) (activities.CheckWorkflowClosedActivityOutput, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return activities.CheckWorkflowClosedActivityOutput{}, temporal.NewApplicationError(
			"failed to encode activity output",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}
	var output activities.CheckWorkflowClosedActivityOutput
	if err := json.Unmarshal(raw, &output); err != nil {
		return activities.CheckWorkflowClosedActivityOutput{}, temporal.NewApplicationError(
			"failed to decode activity output",
			MobileRunnerSemaphoreErrInvalidRequest,
		)
	}
	return output, nil
}

func buildPermit(runnerID string, state MobileRunnerSemaphoreRequestState) MobileRunnerSemaphorePermit {
	return MobileRunnerSemaphorePermit{
		RunnerID:    runnerID,
		LeaseID:     state.Request.LeaseID,
		GrantedAt:   state.GrantedAt,
		QueueWaitMs: state.QueueWaitMs,
	}
}

func buildHoldersView(holders map[string]MobileRunnerSemaphoreHolder) []MobileRunnerSemaphoreHolder {
	if len(holders) == 0 {
		return nil
	}

	keys := make([]string, 0, len(holders))
	for leaseID := range holders {
		keys = append(keys, leaseID)
	}
	sort.Strings(keys)

	view := make([]MobileRunnerSemaphoreHolder, 0, len(keys))
	for _, leaseID := range keys {
		view = append(view, holders[leaseID])
	}

	return view
}

func (r *mobileRunnerSemaphoreRuntime) buildRunStatusView(
	ticketID string,
	state MobileRunnerSemaphoreRunTicketState,
) MobileRunnerSemaphoreRunStatusView {
	return MobileRunnerSemaphoreRunStatusView{
		TicketID:          ticketID,
		Status:            state.Status,
		LeaderRunnerID:    state.Request.LeaderRunnerID,
		RequiredRunnerIDs: copyStringSlice(state.Request.RequiredRunnerIDs),
		WorkflowID:        state.WorkflowID,
		RunID:             state.RunID,
		WorkflowNamespace: state.WorkflowNamespace,
		ErrorMessage:      state.ErrorMessage,
	}
}

func buildQueuePreview(
	queue []string,
	requests map[string]MobileRunnerSemaphoreRequestState,
) []MobileRunnerSemaphoreQueueEntry {
	limit := queuePreviewLimit
	if len(queue) < limit {
		limit = len(queue)
	}

	if limit == 0 {
		return nil
	}

	preview := make([]MobileRunnerSemaphoreQueueEntry, 0, limit)
	for _, requestID := range queue[:limit] {
		request, ok := requests[requestID]
		if !ok {
			continue
		}
		preview = append(preview, MobileRunnerSemaphoreQueueEntry{
			RequestID:       request.Request.RequestID,
			LeaseID:         request.Request.LeaseID,
			OwnerNamespace:  request.Request.OwnerNamespace,
			OwnerWorkflowID: request.Request.OwnerWorkflowID,
			OwnerRunID:      request.Request.OwnerRunID,
			RequestedAt:     request.RequestedAt,
		})
	}

	return preview
}

func removeFromQueue(queue []string, requestID string) []string {
	for i, queuedID := range queue {
		if queuedID == requestID {
			return append(queue[:i], queue[i+1:]...)
		}
	}
	return queue
}

func (r *mobileRunnerSemaphoreRuntime) runQueuePosition(ticketID string) (int, int) {
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
	tickets map[string]MobileRunnerSemaphoreRunTicketState,
) []string {
	queue = append(queue, ticketID)
	return sortRunQueue(queue, tickets)
}

func runTicketLess(
	left MobileRunnerSemaphoreEnqueueRunRequest,
	right MobileRunnerSemaphoreEnqueueRunRequest,
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
	tickets map[string]MobileRunnerSemaphoreRunTicketState,
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

func copyHolders(holders map[string]MobileRunnerSemaphoreHolder) map[string]MobileRunnerSemaphoreHolder {
	if holders == nil {
		return nil
	}
	result := make(map[string]MobileRunnerSemaphoreHolder, len(holders))
	for key, value := range holders {
		result[key] = value
	}
	return result
}

func copyRequests(
	requests map[string]MobileRunnerSemaphoreRequestState,
) map[string]MobileRunnerSemaphoreRequestState {
	if requests == nil {
		return nil
	}
	result := make(map[string]MobileRunnerSemaphoreRequestState, len(requests))
	for key, value := range requests {
		result[key] = value
	}
	return result
}

func copyRunTickets(
	tickets map[string]MobileRunnerSemaphoreRunTicketState,
) map[string]MobileRunnerSemaphoreRunTicketState {
	if tickets == nil {
		return nil
	}
	result := make(map[string]MobileRunnerSemaphoreRunTicketState, len(tickets))
	for key, value := range tickets {
		result[key] = copyRunTicketState(value)
	}
	return result
}

func copyRunTicketState(
	value MobileRunnerSemaphoreRunTicketState,
) MobileRunnerSemaphoreRunTicketState {
	copyValue := value
	copyValue.Request = copyRunTicketRequest(value.Request)
	copyValue.GrantedRunnerIDs = copyStringBoolMap(value.GrantedRunnerIDs)
	return copyValue
}

func copyRunTicketRequest(
	request MobileRunnerSemaphoreEnqueueRunRequest,
) MobileRunnerSemaphoreEnqueueRunRequest {
	copyRequest := request
	copyRequest.RequiredRunnerIDs = copyStringSlice(request.RequiredRunnerIDs)
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

func copyTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
