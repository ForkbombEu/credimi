// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestMobileRunnerSemaphoreWorkflowFIFO(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	permitsCh := make(chan MobileRunnerSemaphorePermit, 3)
	errCh := make(chan error, 3)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-1", LeaseID: "lease-1"},
			permitsCh,
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-2", LeaseID: "lease-2"},
			permitsCh,
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-3", LeaseID: "lease-3"},
			permitsCh,
			errCh,
		)
	}, time.Second*3)

	env.RegisterDelayedCallback(func() {
		enqueueReleaseUpdate(env, MobileRunnerSemaphoreReleaseRequest{LeaseID: "lease-1"}, errCh)
	}, time.Second*4)

	env.RegisterDelayedCallback(func() {
		enqueueReleaseUpdate(env, MobileRunnerSemaphoreReleaseRequest{LeaseID: "lease-2"}, errCh)
	}, time.Second*5)

	env.RegisterDelayedCallback(func() {
		enqueueReleaseUpdate(env, MobileRunnerSemaphoreReleaseRequest{LeaseID: "lease-3"}, errCh)
	}, time.Second*6)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*7)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	permits := map[string]MobileRunnerSemaphorePermit{}
	for len(permits) < 3 {
		select {
		case err := <-errCh:
			require.NoError(t, err)
		case permit := <-permitsCh:
			permits[permit.LeaseID] = permit
		case <-time.After(2 * time.Second):
			require.Fail(t, "timed out waiting for permits")
		}
	}

	require.Empty(t, drainErrors(errCh))

	p1 := permits["lease-1"]
	p2 := permits["lease-2"]
	p3 := permits["lease-3"]

	require.True(t, p1.GrantedAt.Before(p2.GrantedAt))
	require.True(t, p2.GrantedAt.Before(p3.GrantedAt))
	require.Zero(t, p1.QueueWaitMs)
	require.Greater(t, p2.QueueWaitMs, int64(0))
	require.Greater(t, p3.QueueWaitMs, int64(0))
}

func TestMobileRunnerSemaphoreWorkflowAcquireIdempotent(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	permitsCh := make(chan MobileRunnerSemaphorePermit, 2)
	errCh := make(chan error, 2)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-1", LeaseID: "lease-1"},
			permitsCh,
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-1", LeaseID: "lease-1"},
			permitsCh,
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*3)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	var first MobileRunnerSemaphorePermit
	var second MobileRunnerSemaphorePermit

	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			require.NoError(t, err)
		case permit := <-permitsCh:
			if i == 0 {
				first = permit
			} else {
				second = permit
			}
		case <-time.After(2 * time.Second):
			require.Fail(t, "timed out waiting for permits")
		}
	}

	require.Empty(t, drainErrors(errCh))
	require.Equal(t, first, second)
}

func TestMobileRunnerSemaphoreWorkflowReleaseUnknownLease(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	errCh := make(chan error, 1)
	releaseCh := make(chan MobileRunnerSemaphoreReleaseResult, 1)

	env.RegisterDelayedCallback(func() {
		enqueueReleaseUpdateWithResult(
			env,
			MobileRunnerSemaphoreReleaseRequest{LeaseID: "missing-lease"},
			releaseCh,
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*2)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case result := <-releaseCh:
		require.False(t, result.Released)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for release")
	}

	require.Empty(t, drainErrors(errCh))
}

func TestMobileRunnerSemaphoreWorkflowQueueAdvancesAfterRelease(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	permit1Ch := make(chan MobileRunnerSemaphorePermit, 1)
	permit2Ch := make(chan MobileRunnerSemaphorePermit, 1)
	errCh := make(chan error, 2)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "lease-1", LeaseID: "lease-1"},
			permit1Ch,
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "lease-2", LeaseID: "lease-2"},
			permit2Ch,
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		enqueueReleaseUpdate(env, MobileRunnerSemaphoreReleaseRequest{LeaseID: "lease-1"}, errCh)
	}, time.Second*3)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*4)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case permit := <-permit1Ch:
		require.Equal(t, "lease-1", permit.LeaseID)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for first permit")
	}

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case permit := <-permit2Ch:
		require.Equal(t, "lease-2", permit.LeaseID)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for second permit")
	}

	require.Empty(t, drainErrors(errCh))
}

func TestMobileRunnerSemaphoreWorkflowAcquireTimeout(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	errCh := make(chan error, 2)
	timeoutCh := make(chan error, 1)
	stateCh := make(chan MobileRunnerSemaphoreStateView, 1)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-1", LeaseID: "lease-1"},
			make(chan MobileRunnerSemaphorePermit, 1),
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdateExpectError(
			env,
			MobileRunnerSemaphoreAcquireRequest{
				RequestID:   "req-2",
				LeaseID:     "lease-2",
				WaitTimeout: time.Second * 2,
			},
			timeoutCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		encoded, err := env.QueryWorkflow(MobileRunnerSemaphoreStateQuery)
		if err != nil {
			stateCh <- MobileRunnerSemaphoreStateView{RunnerID: "query-error"}
			errCh <- err
			return
		}
		var state MobileRunnerSemaphoreStateView
		if decodeErr := encoded.Get(&state); decodeErr != nil {
			errCh <- decodeErr
			return
		}
		stateCh <- state
	}, time.Second*5)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*6)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	select {
	case err := <-timeoutCh:
		var appErr *temporal.ApplicationError
		require.True(t, errors.As(err, &appErr))
		require.Equal(t, MobileRunnerSemaphoreErrTimeout, appErr.Type())
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for timeout error")
	}

	select {
	case state := <-stateCh:
		require.Equal(t, 0, state.QueueLen)
		require.NotNil(t, state.CurrentHolder)
		require.Equal(t, "lease-1", state.CurrentHolder.LeaseID)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for state")
	}

	require.Empty(t, drainErrors(errCh))
}

func TestMobileRunnerSemaphoreWorkflowLateGrantAfterTimeout(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	permit1Ch := make(chan MobileRunnerSemaphorePermit, 1)
	permit2Ch := make(chan MobileRunnerSemaphorePermit, 1)
	acquireErr1Ch := make(chan error, 1)
	acquireErr2Ch := make(chan error, 1)
	errCh := make(chan error, 2)
	stateCh := make(chan MobileRunnerSemaphoreStateView, 1)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-1", LeaseID: "lease-1"},
			permit1Ch,
			acquireErr1Ch,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{
				RequestID:   "req-2",
				LeaseID:     "lease-2",
				WaitTimeout: time.Second,
			},
			permit2Ch,
			acquireErr2Ch,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		enqueueReleaseUpdate(env, MobileRunnerSemaphoreReleaseRequest{LeaseID: "lease-1"}, errCh)
	}, time.Second*3)

	env.RegisterDelayedCallback(func() {
		encoded, err := env.QueryWorkflow(MobileRunnerSemaphoreStateQuery)
		if err != nil {
			errCh <- err
			return
		}
		var state MobileRunnerSemaphoreStateView
		if decodeErr := encoded.Get(&state); decodeErr != nil {
			errCh <- decodeErr
			return
		}
		stateCh <- state
	}, time.Second*4)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*5)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	select {
	case err := <-acquireErr1Ch:
		require.NoError(t, err)
	case permit := <-permit1Ch:
		require.Equal(t, "lease-1", permit.LeaseID)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for first permit")
	}

	var acquireErr error
	var permit *MobileRunnerSemaphorePermit
	select {
	case err := <-acquireErr2Ch:
		acquireErr = err
	case result := <-permit2Ch:
		permit = &result
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for second acquire")
	}

	var state MobileRunnerSemaphoreStateView
	select {
	case state = <-stateCh:
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for state")
	}

	if acquireErr != nil {
		var appErr *temporal.ApplicationError
		require.True(t, errors.As(acquireErr, &appErr))
		require.Equal(t, MobileRunnerSemaphoreErrTimeout, appErr.Type())
		require.Nil(t, state.CurrentHolder)
		require.Empty(t, state.Holders)
		require.Zero(t, state.QueueLen)
	} else if permit != nil {
		require.Equal(t, "lease-2", permit.LeaseID)
		require.NotNil(t, state.CurrentHolder)
		require.Equal(t, "lease-2", state.CurrentHolder.LeaseID)
	}

	require.Empty(t, drainErrors(acquireErr1Ch))
	require.Empty(t, drainErrors(acquireErr2Ch))
	require.Empty(t, drainErrors(errCh))
}

func TestMobileRunnerSemaphoreWorkflowRunQueuePositions(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	permitCh := make(chan MobileRunnerSemaphorePermit, 1)
	errCh := make(chan error, 4)
	statusCh := make(chan MobileRunnerSemaphoreRunStatusView, 3)

	enqueuedAt := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-1", LeaseID: "lease-1"},
			permitCh,
			errCh,
		)
	}, time.Millisecond)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-1",
			MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:          "ticket-b",
				OwnerNamespace:    "tenant-1",
				EnqueuedAt:        enqueuedAt,
				RunnerID:          "runner-1",
				RequiredRunnerIDs: []string{"runner-1"},
				LeaderRunnerID:    "runner-1",
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-b", statusCh, errCh)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-2",
			MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:          "ticket-a",
				OwnerNamespace:    "tenant-1",
				EnqueuedAt:        enqueuedAt,
				RunnerID:          "runner-1",
				RequiredRunnerIDs: []string{"runner-1"},
				LeaderRunnerID:    "runner-1",
			},
			errCh,
		)
	}, time.Second*3)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-a", statusCh, errCh)
	}, time.Second*4)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-b", statusCh, errCh)
	}, time.Second*5)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*6)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	var ticketAStatuses []MobileRunnerSemaphoreRunStatusView
	var ticketBStatuses []MobileRunnerSemaphoreRunStatusView
	for i := 0; i < 3; i++ {
		select {
		case status := <-statusCh:
			switch status.TicketID {
			case "ticket-a":
				ticketAStatuses = append(ticketAStatuses, status)
			case "ticket-b":
				ticketBStatuses = append(ticketBStatuses, status)
			}
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			require.Fail(t, "timed out waiting for run status")
		}
	}

	require.Len(t, ticketAStatuses, 1)
	require.Len(t, ticketBStatuses, 2)

	require.Equal(t, mobileRunnerSemaphoreRunQueued, ticketAStatuses[0].Status)
	require.Equal(t, 0, ticketAStatuses[0].Position)
	require.Equal(t, 2, ticketAStatuses[0].LineLen)

	var sawInitial bool
	var sawReordered bool
	for _, status := range ticketBStatuses {
		require.Equal(t, mobileRunnerSemaphoreRunQueued, status.Status)
		switch status.LineLen {
		case 1:
			sawInitial = true
			require.Equal(t, 0, status.Position)
		case 2:
			sawReordered = true
			require.Equal(t, 1, status.Position)
		default:
			require.Failf(t, "unexpected line_len", "line_len=%d", status.LineLen)
		}
	}

	require.True(t, sawInitial)
	require.True(t, sawReordered)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-permitCh:
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for permit")
	}

	require.Empty(t, drainErrors(errCh))
}

func TestMobileRunnerSemaphoreWorkflowRunCancelQueued(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	permitCh := make(chan MobileRunnerSemaphorePermit, 1)
	errCh := make(chan error, 3)
	statusCh := make(chan MobileRunnerSemaphoreRunStatusView, 2)

	enqueuedAt := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-1", LeaseID: "lease-1"},
			permitCh,
			errCh,
		)
	}, time.Millisecond)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-cancel-1",
			MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:          "ticket-cancel-1",
				OwnerNamespace:    "tenant-1",
				EnqueuedAt:        enqueuedAt,
				RunnerID:          "runner-1",
				RequiredRunnerIDs: []string{"runner-1"},
				LeaderRunnerID:    "runner-1",
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-cancel-2",
			MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:          "ticket-cancel-2",
				OwnerNamespace:    "tenant-1",
				EnqueuedAt:        enqueuedAt.Add(2 * time.Second),
				RunnerID:          "runner-1",
				RequiredRunnerIDs: []string{"runner-1"},
				LeaderRunnerID:    "runner-1",
			},
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		enqueueCancelRunUpdate(
			env,
			"cancel-ticket-1",
			MobileRunnerSemaphoreRunCancelRequest{
				TicketID:       "ticket-cancel-1",
				OwnerNamespace: "tenant-1",
			},
			errCh,
		)
	}, time.Second*3)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-cancel-1", statusCh, errCh)
	}, time.Second*4)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-cancel-2", statusCh, errCh)
	}, time.Second*5)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*6)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	var canceledStatus *MobileRunnerSemaphoreRunStatusView
	var queuedStatus *MobileRunnerSemaphoreRunStatusView
	for i := 0; i < 2; i++ {
		select {
		case status := <-statusCh:
			switch status.TicketID {
			case "ticket-cancel-1":
				canceledStatus = &status
			case "ticket-cancel-2":
				queuedStatus = &status
			}
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			require.Fail(t, "timed out waiting for cancel status")
		}
	}

	require.NotNil(t, canceledStatus)
	require.Equal(t, mobileRunnerSemaphoreRunNotFound, canceledStatus.Status)
	require.NotNil(t, queuedStatus)
	require.Equal(t, mobileRunnerSemaphoreRunQueued, queuedStatus.Status)
	require.Equal(t, 0, queuedStatus.Position)
	require.Equal(t, 1, queuedStatus.LineLen)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-permitCh:
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for permit")
	}

	require.Empty(t, drainErrors(errCh))
}

func TestMobileRunnerSemaphoreWorkflowRunStartsWhenCapacityAvailable(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	startAct := activities.NewStartQueuedPipelineActivity()
	env.RegisterActivityWithOptions(startAct.Execute, activity.RegisterOptions{Name: startAct.Name()})
	env.OnActivity(startAct.Name(), mock.Anything, mock.Anything).Return(
		workflowengine.ActivityResult{
			Output: activities.StartQueuedPipelineActivityOutput{
				WorkflowID:        "wf-1",
				RunID:             "run-1",
				WorkflowNamespace: "tenant-1",
			},
		},
		nil,
	)

	errCh := make(chan error, 2)
	statusCh := make(chan MobileRunnerSemaphoreRunStatusView, 1)

	enqueuedAt := time.Date(2026, 2, 3, 13, 0, 0, 0, time.UTC)
	pipelineConfig := map[string]any{
		"app_url":   "https://example.com",
		"namespace": "tenant-1",
	}

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-start-1",
			MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-start-1",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt,
				RunnerID:           "runner-1",
				RequiredRunnerIDs:  []string{"runner-1"},
				LeaderRunnerID:     "runner-1",
				PipelineIdentifier: "pipelines/test",
				YAML:               "name: test\nsteps: []\n",
				PipelineConfig:     pipelineConfig,
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-start-1", statusCh, errCh)
	}, time.Second*2)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*3)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	select {
	case status := <-statusCh:
		require.Equal(t, mobileRunnerSemaphoreRunRunning, status.Status)
		require.Equal(t, "wf-1", status.WorkflowID)
		require.Equal(t, "run-1", status.RunID)
		require.Equal(t, "tenant-1", status.WorkflowNamespace)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for running status")
	}

	require.Empty(t, drainErrors(errCh))
}

func TestMobileRunnerSemaphoreWorkflowRunStartFailureAdvancesQueue(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	startAct := activities.NewStartQueuedPipelineActivity()
	env.RegisterActivityWithOptions(startAct.Execute, activity.RegisterOptions{Name: startAct.Name()})
	env.OnActivity(startAct.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, errors.New("start failed")).
		Once()
	env.OnActivity(startAct.Name(), mock.Anything, mock.Anything).Return(
		workflowengine.ActivityResult{
			Output: activities.StartQueuedPipelineActivityOutput{
				WorkflowID:        "wf-2",
				RunID:             "run-2",
				WorkflowNamespace: "tenant-1",
			},
		},
		nil,
	)

	errCh := make(chan error, 3)
	statusCh := make(chan MobileRunnerSemaphoreRunStatusView, 2)

	enqueuedAt := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)
	pipelineConfig := map[string]any{
		"app_url":   "https://example.com",
		"namespace": "tenant-1",
	}

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-fail-1",
			MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-fail-1",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt,
				RunnerID:           "runner-1",
				RequiredRunnerIDs:  []string{"runner-1"},
				LeaderRunnerID:     "runner-1",
				PipelineIdentifier: "pipelines/test",
				YAML:               "name: test\nsteps: []\n",
				PipelineConfig:     pipelineConfig,
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-fail-2",
			MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-fail-2",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt.Add(2 * time.Second),
				RunnerID:           "runner-1",
				RequiredRunnerIDs:  []string{"runner-1"},
				LeaderRunnerID:     "runner-1",
				PipelineIdentifier: "pipelines/test",
				YAML:               "name: test\nsteps: []\n",
				PipelineConfig:     pipelineConfig,
			},
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-fail-1", statusCh, errCh)
	}, time.Second*3)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-fail-2", statusCh, errCh)
	}, time.Second*4)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*5)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	var failedStatus *MobileRunnerSemaphoreRunStatusView
	var runningStatus *MobileRunnerSemaphoreRunStatusView
	for i := 0; i < 2; i++ {
		select {
		case status := <-statusCh:
			switch status.TicketID {
			case "ticket-fail-1":
				failedStatus = &status
			case "ticket-fail-2":
				runningStatus = &status
			}
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			require.Fail(t, "timed out waiting for failure status")
		}
	}

	require.NotNil(t, failedStatus)
	require.Equal(t, mobileRunnerSemaphoreRunFailed, failedStatus.Status)
	require.NotNil(t, runningStatus)
	require.Equal(t, mobileRunnerSemaphoreRunRunning, runningStatus.Status)
	require.Equal(t, "wf-2", runningStatus.WorkflowID)

	require.Empty(t, drainErrors(errCh))
}

func TestMobileRunnerSemaphoreWorkflowRunWaitsForPermitRelease(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	startAct := activities.NewStartQueuedPipelineActivity()
	env.RegisterActivityWithOptions(startAct.Execute, activity.RegisterOptions{Name: startAct.Name()})
	env.OnActivity(startAct.Name(), mock.Anything, mock.Anything).Return(
		workflowengine.ActivityResult{
			Output: activities.StartQueuedPipelineActivityOutput{
				WorkflowID:        "wf-3",
				RunID:             "run-3",
				WorkflowNamespace: "tenant-1",
			},
		},
		nil,
	)

	permitCh := make(chan MobileRunnerSemaphorePermit, 1)
	errCh := make(chan error, 3)
	statusCh := make(chan MobileRunnerSemaphoreRunStatusView, 2)

	enqueuedAt := time.Date(2026, 2, 3, 15, 0, 0, 0, time.UTC)
	pipelineConfig := map[string]any{
		"app_url":   "https://example.com",
		"namespace": "tenant-1",
	}

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-1", LeaseID: "lease-1"},
			permitCh,
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-wait-1",
			MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-wait-1",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt,
				RunnerID:           "runner-1",
				RequiredRunnerIDs:  []string{"runner-1"},
				LeaderRunnerID:     "runner-1",
				PipelineIdentifier: "pipelines/test",
				YAML:               "name: test\nsteps: []\n",
				PipelineConfig:     pipelineConfig,
			},
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-wait-1", statusCh, errCh)
	}, time.Second*3)

	env.RegisterDelayedCallback(func() {
		enqueueReleaseUpdate(env, MobileRunnerSemaphoreReleaseRequest{LeaseID: "lease-1"}, errCh)
	}, time.Second*4)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-wait-1", statusCh, errCh)
	}, time.Second*5)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*6)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	var queuedStatus *MobileRunnerSemaphoreRunStatusView
	var runningStatus *MobileRunnerSemaphoreRunStatusView
	for i := 0; i < 2; i++ {
		select {
		case status := <-statusCh:
			if status.Status == mobileRunnerSemaphoreRunQueued {
				queuedStatus = &status
			} else if status.Status == mobileRunnerSemaphoreRunRunning {
				runningStatus = &status
			}
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			require.Fail(t, "timed out waiting for permit release status")
		}
	}

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-permitCh:
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for permit")
	}

	require.NotNil(t, queuedStatus)
	require.Equal(t, mobileRunnerSemaphoreRunQueued, queuedStatus.Status)
	require.NotNil(t, runningStatus)
	require.Equal(t, mobileRunnerSemaphoreRunRunning, runningStatus.Status)
	require.Equal(t, "wf-3", runningStatus.WorkflowID)

	require.Empty(t, drainErrors(errCh))
}

func TestMobileRunnerSemaphoreWorkflowRunStatusOwnerMismatch(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileRunnerSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	permitCh := make(chan MobileRunnerSemaphorePermit, 1)
	errCh := make(chan error, 2)
	statusCh := make(chan MobileRunnerSemaphoreRunStatusView, 1)

	enqueuedAt := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)

	env.RegisterDelayedCallback(func() {
		enqueueAcquireUpdate(
			env,
			MobileRunnerSemaphoreAcquireRequest{RequestID: "req-1", LeaseID: "lease-1"},
			permitCh,
			errCh,
		)
	}, time.Millisecond)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-owner",
			MobileRunnerSemaphoreEnqueueRunRequest{
				TicketID:          "ticket-owner",
				OwnerNamespace:    "tenant-1",
				EnqueuedAt:        enqueuedAt,
				RunnerID:          "runner-1",
				RequiredRunnerIDs: []string{"runner-1"},
				LeaderRunnerID:    "runner-1",
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-2", "ticket-owner", statusCh, errCh)
	}, time.Second*2)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*3)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileRunnerSemaphoreWorkflowInput{
				RunnerID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	select {
	case status := <-statusCh:
		require.Equal(t, mobileRunnerSemaphoreRunNotFound, status.Status)
		require.Zero(t, status.LineLen)
		require.Zero(t, status.Position)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for owner mismatch status")
	}

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-permitCh:
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for permit")
	}

	require.Empty(t, drainErrors(errCh))
}

func enqueueAcquireUpdate(
	env *testsuite.TestWorkflowEnvironment,
	req MobileRunnerSemaphoreAcquireRequest,
	permitsCh chan<- MobileRunnerSemaphorePermit,
	errCh chan<- error,
) {
	env.UpdateWorkflow(
		MobileRunnerSemaphoreAcquireUpdate,
		MobileRunnerSemaphoreAcquireUpdateID(req.RequestID),
		&testsuite.TestUpdateCallback{
			OnReject: func(err error) {
				errCh <- err
			},
			OnComplete: func(response interface{}, err error) {
				if err != nil {
					errCh <- err
					return
				}
				permit, ok := response.(MobileRunnerSemaphorePermit)
				if !ok {
					errCh <- fmt.Errorf("unexpected permit type: %T", response)
					return
				}
				permitsCh <- permit
			},
		},
		req,
	)
}

func enqueueReleaseUpdate(
	env *testsuite.TestWorkflowEnvironment,
	req MobileRunnerSemaphoreReleaseRequest,
	errCh chan<- error,
) {
	env.UpdateWorkflow(
		MobileRunnerSemaphoreReleaseUpdate,
		MobileRunnerSemaphoreReleaseUpdateID(req.LeaseID),
		&testsuite.TestUpdateCallback{
			OnReject: func(err error) {
				errCh <- err
			},
			OnComplete: func(_ interface{}, err error) {
				if err != nil {
					errCh <- err
				}
			},
		},
		req,
	)
}

func enqueueReleaseUpdateWithResult(
	env *testsuite.TestWorkflowEnvironment,
	req MobileRunnerSemaphoreReleaseRequest,
	releaseCh chan<- MobileRunnerSemaphoreReleaseResult,
	errCh chan<- error,
) {
	env.UpdateWorkflow(
		MobileRunnerSemaphoreReleaseUpdate,
		MobileRunnerSemaphoreReleaseUpdateID(req.LeaseID),
		&testsuite.TestUpdateCallback{
			OnReject: func(err error) {
				errCh <- err
			},
			OnComplete: func(response interface{}, err error) {
				if err != nil {
					errCh <- err
					return
				}
				result, ok := response.(MobileRunnerSemaphoreReleaseResult)
				if !ok {
					errCh <- fmt.Errorf("unexpected release type: %T", response)
					return
				}
				releaseCh <- result
			},
		},
		req,
	)
}

func enqueueAcquireUpdateExpectError(
	env *testsuite.TestWorkflowEnvironment,
	req MobileRunnerSemaphoreAcquireRequest,
	timeoutCh chan<- error,
) {
	env.UpdateWorkflow(
		MobileRunnerSemaphoreAcquireUpdate,
		MobileRunnerSemaphoreAcquireUpdateID(req.RequestID),
		&testsuite.TestUpdateCallback{
			OnReject: func(err error) {
				timeoutCh <- err
			},
			OnComplete: func(_ interface{}, err error) {
				if err == nil {
					timeoutCh <- fmt.Errorf("expected timeout error")
					return
				}
				timeoutCh <- err
			},
		},
		req,
	)
}

func enqueueCancelRunUpdate(
	env *testsuite.TestWorkflowEnvironment,
	updateID string,
	req MobileRunnerSemaphoreRunCancelRequest,
	errCh chan<- error,
) {
	env.UpdateWorkflow(
		MobileRunnerSemaphoreCancelRunUpdate,
		updateID,
		&testsuite.TestUpdateCallback{
			OnReject: func(err error) {
				errCh <- err
			},
			OnComplete: func(_ interface{}, err error) {
				if err != nil {
					errCh <- err
				}
			},
		},
		req,
	)
}

func enqueueRunUpdate(
	env *testsuite.TestWorkflowEnvironment,
	updateID string,
	req MobileRunnerSemaphoreEnqueueRunRequest,
	errCh chan<- error,
) {
	env.UpdateWorkflow(
		MobileRunnerSemaphoreEnqueueRunUpdate,
		updateID,
		&testsuite.TestUpdateCallback{
			OnReject: func(err error) {
				errCh <- err
			},
			OnComplete: func(_ interface{}, err error) {
				if err != nil {
					errCh <- err
				}
			},
		},
		req,
	)
}

func queryRunStatus(
	env *testsuite.TestWorkflowEnvironment,
	ownerNamespace string,
	ticketID string,
	statusCh chan<- MobileRunnerSemaphoreRunStatusView,
	errCh chan<- error,
) {
	encoded, err := env.QueryWorkflow(MobileRunnerSemaphoreRunStatusQuery, ownerNamespace, ticketID)
	if err != nil {
		errCh <- err
		return
	}
	var status MobileRunnerSemaphoreRunStatusView
	if decodeErr := encoded.Get(&status); decodeErr != nil {
		errCh <- decodeErr
		return
	}
	statusCh <- status
}

func drainErrors(errCh <-chan error) []error {
	var errs []error
	for {
		select {
		case err := <-errCh:
			errs = append(errs, err)
		default:
			return errs
		}
	}
}
