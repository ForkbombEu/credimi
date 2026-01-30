// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"sort"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	mobileRunnerSemaphoreMaxUpdateBatches = 1000
	queuePreviewLimit                     = 5
)

type MobileRunnerSemaphoreWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

const (
	mobileRunnerSemaphoreRequestQueued   MobileRunnerSemaphoreRequestStatus = "queued"
	mobileRunnerSemaphoreRequestGranted  MobileRunnerSemaphoreRequestStatus = "granted"
	mobileRunnerSemaphoreRequestTimedOut MobileRunnerSemaphoreRequestStatus = "timed_out"
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

	if err := runtime.registerAcquireHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := runtime.registerReleaseHandler(); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	return runtime.awaitContinue()
}

type mobileRunnerSemaphoreRuntime struct {
	ctx            workflow.Context
	runnerID       string
	capacity       int
	holders        map[string]MobileRunnerSemaphoreHolder
	queue          []string
	requests       map[string]MobileRunnerSemaphoreRequestState
	lastGrantAt    *time.Time
	updateCount    int
	shouldContinue bool
	continueInput  workflowengine.WorkflowInput
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
		ctx:      ctx,
		runnerID: payload.RunnerID,
		capacity: payload.Capacity,
		holders:  map[string]MobileRunnerSemaphoreHolder{},
		queue:    []string{},
		requests: map[string]MobileRunnerSemaphoreRequestState{},
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

	return MobileRunnerSemaphoreReleaseResult{Released: true}, nil
}

func (r *mobileRunnerSemaphoreRuntime) grantAvailable(ctx workflow.Context) {
	if r.capacity <= 0 {
		return
	}
	now := workflow.Now(ctx)
	for len(r.holders) < r.capacity && len(r.queue) > 0 {
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
			LeaseID:        request.Request.LeaseID,
			RequestID:      requestID,
			OwnerNamespace: request.Request.OwnerNamespace,
			OwnerWorkflowID: request.Request.OwnerWorkflowID,
			OwnerRunID:     request.Request.OwnerRunID,
			GrantedAt:      request.GrantedAt,
			QueueWaitMs:    request.QueueWaitMs,
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
			RequestID:      request.Request.RequestID,
			LeaseID:        request.Request.LeaseID,
			OwnerNamespace: request.Request.OwnerNamespace,
			OwnerWorkflowID: request.Request.OwnerWorkflowID,
			OwnerRunID:     request.Request.OwnerRunID,
			RequestedAt:    request.RequestedAt,
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

func copyQueue(queue []string) []string {
	if queue == nil {
		return nil
	}
	result := make([]string, len(queue))
	copy(result, queue)
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

func copyTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
