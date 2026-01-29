// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"fmt"
	"sort"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	MobileRunnerSemaphoreTaskQueue        = "mobile-runner-semaphore-task-queue"
	MobileRunnerSemaphoreWorkflowName     = "mobile-runner-semaphore"
	MobileRunnerSemaphoreAcquireUpdate    = "Acquire"
	MobileRunnerSemaphoreReleaseUpdate    = "Release"
	MobileRunnerSemaphoreStateQuery       = "GetState"
	mobileRunnerSemaphoreMaxUpdateBatches = 1000
	queuePreviewLimit                     = 5
)

const (
	mobileRunnerSemaphoreErrInvalidRequest = "mobile-runner-semaphore-invalid-request"
	mobileRunnerSemaphoreErrTimeout        = "mobile-runner-semaphore-timeout"
)

type MobileRunnerSemaphoreWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

type MobileRunnerSemaphoreWorkflowInput struct {
	RunnerID string                            `json:"runner_id"`
	Capacity int                               `json:"capacity"`
	State    *MobileRunnerSemaphoreWorkflowState `json:"state,omitempty"`
}

type MobileRunnerSemaphoreWorkflowState struct {
	Capacity    int                                            `json:"capacity"`
	Holders     map[string]MobileRunnerSemaphoreHolder         `json:"holders,omitempty"`
	Queue       []string                                       `json:"queue,omitempty"`
	Requests    map[string]MobileRunnerSemaphoreRequestState   `json:"requests,omitempty"`
	LastGrantAt *time.Time                                     `json:"last_grant_at,omitempty"`
	UpdateCount int                                            `json:"update_count,omitempty"`
}

type MobileRunnerSemaphoreAcquireRequest struct {
	RequestID     string        `json:"request_id"`
	LeaseID       string        `json:"lease_id"`
	OwnerNamespace string       `json:"owner_namespace,omitempty"`
	OwnerWorkflowID string      `json:"owner_workflow_id,omitempty"`
	OwnerRunID     string       `json:"owner_run_id,omitempty"`
	WaitTimeout    time.Duration `json:"wait_timeout,omitempty"`
}

type MobileRunnerSemaphoreReleaseRequest struct {
	LeaseID string `json:"lease_id"`
}

type MobileRunnerSemaphorePermit struct {
	RunnerID    string    `json:"runner_id"`
	LeaseID     string    `json:"lease_id"`
	GrantedAt   time.Time `json:"granted_at"`
	QueueWaitMs int64     `json:"queue_wait_ms"`
}

type MobileRunnerSemaphoreReleaseResult struct {
	Released bool `json:"released"`
}

type MobileRunnerSemaphoreHolder struct {
	LeaseID        string    `json:"lease_id"`
	RequestID      string    `json:"request_id"`
	OwnerNamespace string    `json:"owner_namespace,omitempty"`
	OwnerWorkflowID string   `json:"owner_workflow_id,omitempty"`
	OwnerRunID     string    `json:"owner_run_id,omitempty"`
	GrantedAt      time.Time `json:"granted_at"`
	QueueWaitMs    int64     `json:"queue_wait_ms"`
}

type MobileRunnerSemaphoreQueueEntry struct {
	RequestID      string    `json:"request_id"`
	LeaseID        string    `json:"lease_id"`
	OwnerNamespace string    `json:"owner_namespace,omitempty"`
	OwnerWorkflowID string   `json:"owner_workflow_id,omitempty"`
	OwnerRunID     string    `json:"owner_run_id,omitempty"`
	RequestedAt    time.Time `json:"requested_at"`
}

type MobileRunnerSemaphoreStateView struct {
	RunnerID     string                             `json:"runner_id"`
	Capacity     int                                `json:"capacity"`
	CurrentHolder *MobileRunnerSemaphoreHolder      `json:"current_holder,omitempty"`
	Holders      []MobileRunnerSemaphoreHolder       `json:"holders"`
	QueueLen     int                                `json:"queue_len"`
	QueuePreview []MobileRunnerSemaphoreQueueEntry   `json:"queue_preview"`
	LastGrantAt  *time.Time                          `json:"last_grant_at,omitempty"`
}

type MobileRunnerSemaphoreRequestStatus string

const (
	mobileRunnerSemaphoreRequestQueued   MobileRunnerSemaphoreRequestStatus = "queued"
	mobileRunnerSemaphoreRequestGranted  MobileRunnerSemaphoreRequestStatus = "granted"
	mobileRunnerSemaphoreRequestTimedOut MobileRunnerSemaphoreRequestStatus = "timed_out"
)

type MobileRunnerSemaphoreRequestState struct {
	Request     MobileRunnerSemaphoreAcquireRequest `json:"request"`
	Status      MobileRunnerSemaphoreRequestStatus  `json:"status"`
	RequestedAt time.Time                           `json:"requested_at"`
	GrantedAt   time.Time                           `json:"granted_at,omitempty"`
	QueueWaitMs int64                               `json:"queue_wait_ms,omitempty"`
}

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

	runnerID := payload.RunnerID
	if runnerID == "" {
		return workflowengine.WorkflowResult{}, temporal.NewApplicationError(
			"runner_id is required",
			mobileRunnerSemaphoreErrInvalidRequest,
		)
	}

	capacity := payload.Capacity
	if capacity <= 0 {
		capacity = 1
	}

	holders := map[string]MobileRunnerSemaphoreHolder{}
	queue := []string{}
	requests := map[string]MobileRunnerSemaphoreRequestState{}
	var lastGrantAt *time.Time
	updateCount := 0

	if payload.State != nil {
		if payload.State.Capacity > 0 {
			capacity = payload.State.Capacity
		}
		holders = payload.State.Holders
		queue = payload.State.Queue
		requests = payload.State.Requests
		lastGrantAt = payload.State.LastGrantAt
		updateCount = payload.State.UpdateCount
	}

	if holders == nil {
		holders = map[string]MobileRunnerSemaphoreHolder{}
	}
	if requests == nil {
		requests = map[string]MobileRunnerSemaphoreRequestState{}
	}

	shouldContinue := false
	var continueInput workflowengine.WorkflowInput

	grantAvailable := func() {
		if capacity <= 0 {
			return
		}
		now := workflow.Now(ctx)
		for len(holders) < capacity && len(queue) > 0 {
			requestID := queue[0]
			queue = queue[1:]
			request, ok := requests[requestID]
			if !ok || request.Status != mobileRunnerSemaphoreRequestQueued {
				continue
			}

			request.Status = mobileRunnerSemaphoreRequestGranted
			request.GrantedAt = now
			request.QueueWaitMs = now.Sub(request.RequestedAt).Milliseconds()
			requests[requestID] = request

			holders[request.Request.LeaseID] = MobileRunnerSemaphoreHolder{
				LeaseID:        request.Request.LeaseID,
				RequestID:      requestID,
				OwnerNamespace: request.Request.OwnerNamespace,
				OwnerWorkflowID: request.Request.OwnerWorkflowID,
				OwnerRunID:     request.Request.OwnerRunID,
				GrantedAt:      request.GrantedAt,
				QueueWaitMs:    request.QueueWaitMs,
			}

			timeCopy := now
			lastGrantAt = &timeCopy
		}
	}

	maybeScheduleContinue := func() {
		if shouldContinue || updateCount < mobileRunnerSemaphoreMaxUpdateBatches {
			return
		}

		stateCopy := MobileRunnerSemaphoreWorkflowState{
			Capacity:    capacity,
			Holders:     copyHolders(holders),
			Queue:       copyQueue(queue),
			Requests:    copyRequests(requests),
			LastGrantAt: copyTimePtr(lastGrantAt),
			UpdateCount: 0,
		}

		continueInput = workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: runnerID,
				Capacity: capacity,
				State:    &stateCopy,
			},
		}
		shouldContinue = true
	}

	if err := workflow.SetQueryHandler(ctx, MobileRunnerSemaphoreStateQuery, func() (MobileRunnerSemaphoreStateView, error) {
		holdersView := buildHoldersView(holders)
		queuePreview := buildQueuePreview(queue, requests)

		var currentHolder *MobileRunnerSemaphoreHolder
		if capacity == 1 && len(holdersView) == 1 {
			holderCopy := holdersView[0]
			currentHolder = &holderCopy
		}

		return MobileRunnerSemaphoreStateView{
			RunnerID:      runnerID,
			Capacity:      capacity,
			CurrentHolder: currentHolder,
			Holders:       holdersView,
			QueueLen:      len(queue),
			QueuePreview:  queuePreview,
			LastGrantAt:   lastGrantAt,
		}, nil
	}); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := workflow.SetUpdateHandler(ctx, MobileRunnerSemaphoreAcquireUpdate,
		func(ctx workflow.Context, req MobileRunnerSemaphoreAcquireRequest) (MobileRunnerSemaphorePermit, error) {
			if req.RequestID == "" || req.LeaseID == "" {
				return MobileRunnerSemaphorePermit{}, temporal.NewApplicationError(
					"request_id and lease_id are required",
					mobileRunnerSemaphoreErrInvalidRequest,
				)
			}

			if existing, ok := requests[req.RequestID]; ok {
				switch existing.Status {
				case mobileRunnerSemaphoreRequestGranted:
					return buildPermit(runnerID, existing), nil
				case mobileRunnerSemaphoreRequestTimedOut:
					return MobileRunnerSemaphorePermit{}, temporal.NewApplicationError(
						"request timed out",
						mobileRunnerSemaphoreErrTimeout,
						req.RequestID,
					)
				}
			} else {
				requests[req.RequestID] = MobileRunnerSemaphoreRequestState{
					Request:     req,
					Status:      mobileRunnerSemaphoreRequestQueued,
					RequestedAt: workflow.Now(ctx),
				}
				queue = append(queue, req.RequestID)
				grantAvailable()
			}

			updateCount++
			maybeScheduleContinue()

			timeoutReached := false
			if req.WaitTimeout > 0 {
				timerCtx, cancel := workflow.WithCancel(ctx)
				workflow.Go(ctx, func(ctx workflow.Context) {
					if err := workflow.NewTimer(timerCtx, req.WaitTimeout).Get(timerCtx, nil); err == nil {
						timeoutReached = true
					}
				})

				_ = workflow.Await(ctx, func() bool {
					state, ok := requests[req.RequestID]
					if !ok {
						return timeoutReached
					}
					return state.Status == mobileRunnerSemaphoreRequestGranted || timeoutReached
				})
				cancel()

				state, ok := requests[req.RequestID]
				if !ok {
					return MobileRunnerSemaphorePermit{}, temporal.NewApplicationError(
						"request not found",
						mobileRunnerSemaphoreErrInvalidRequest,
						req.RequestID,
					)
				}

				if state.Status == mobileRunnerSemaphoreRequestGranted {
					return buildPermit(runnerID, state), nil
				}

				if timeoutReached {
					queue = removeFromQueue(queue, req.RequestID)
					state.Status = mobileRunnerSemaphoreRequestTimedOut
					requests[req.RequestID] = state
					grantAvailable()
					return MobileRunnerSemaphorePermit{}, temporal.NewApplicationError(
						"acquire timeout",
						mobileRunnerSemaphoreErrTimeout,
						req.RequestID,
					)
				}
			}

			err := workflow.Await(ctx, func() bool {
				state, ok := requests[req.RequestID]
				return ok && state.Status == mobileRunnerSemaphoreRequestGranted
			})
			if err != nil {
				return MobileRunnerSemaphorePermit{}, err
			}

			return buildPermit(runnerID, requests[req.RequestID]), nil
		},
	); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := workflow.SetUpdateHandler(ctx, MobileRunnerSemaphoreReleaseUpdate,
		func(ctx workflow.Context, req MobileRunnerSemaphoreReleaseRequest) (MobileRunnerSemaphoreReleaseResult, error) {
			if req.LeaseID == "" {
				return MobileRunnerSemaphoreReleaseResult{}, temporal.NewApplicationError(
					"lease_id is required",
					mobileRunnerSemaphoreErrInvalidRequest,
				)
			}

			if _, ok := holders[req.LeaseID]; !ok {
				return MobileRunnerSemaphoreReleaseResult{Released: false}, nil
			}

			delete(holders, req.LeaseID)
			grantAvailable()

			updateCount++
			maybeScheduleContinue()

			return MobileRunnerSemaphoreReleaseResult{Released: true}, nil
		},
	); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if err := workflow.Await(ctx, func() bool { return shouldContinue }); err != nil {
		return workflowengine.WorkflowResult{}, err
	}

	if shouldContinue {
		return workflowengine.WorkflowResult{}, workflow.NewContinueAsNewError(
			ctx,
			MobileRunnerSemaphoreWorkflowName,
			continueInput,
		)
	}

	return workflowengine.WorkflowResult{}, nil
}

func MobileRunnerSemaphoreWorkflowID(runnerID string) string {
	return fmt.Sprintf("mobile-runner-semaphore/%s", runnerID)
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
	copyQueue := make([]string, len(queue))
	copy(copyQueue, queue)
	return copyQueue
}

func copyHolders(holders map[string]MobileRunnerSemaphoreHolder) map[string]MobileRunnerSemaphoreHolder {
	if holders == nil {
		return nil
	}
	copyHolders := make(map[string]MobileRunnerSemaphoreHolder, len(holders))
	for key, value := range holders {
		copyHolders[key] = value
	}
	return copyHolders
}

func copyRequests(
	requests map[string]MobileRunnerSemaphoreRequestState,
) map[string]MobileRunnerSemaphoreRequestState {
	if requests == nil {
		return nil
	}
	copyRequests := make(map[string]MobileRunnerSemaphoreRequestState, len(requests))
	for key, value := range requests {
		copyRequests[key] = value
	}
	return copyRequests
}

func copyTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
