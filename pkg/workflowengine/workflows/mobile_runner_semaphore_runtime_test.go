// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func newRuntimeForTests() *mobileRunnerSemaphoreRuntime {
	return &mobileRunnerSemaphoreRuntime{
		runnerID:   "runner-1",
		runQueue:   []string{},
		runTickets: map[string]MobileRunnerSemaphoreRunTicketState{},
		requests:   map[string]MobileRunnerSemaphoreRequestState{},
		holders:    map[string]MobileRunnerSemaphoreHolder{},
	}
}

func TestHandleEnqueueRunValidation(t *testing.T) {
	rt := newRuntimeForTests()
	_, err := rt.handleEnqueueRun(MobileRunnerSemaphoreEnqueueRunRequest{})
	require.Error(t, err)

	var appErr *temporal.ApplicationError
	require.True(t, temporal.IsApplicationError(err))
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, MobileRunnerSemaphoreErrInvalidRequest, appErr.Type())
}

func TestHandleEnqueueRunOwnerMismatch(t *testing.T) {
	rt := newRuntimeForTests()
	rt.runTickets["ticket-1"] = MobileRunnerSemaphoreRunTicketState{
		Request: MobileRunnerSemaphoreEnqueueRunRequest{
			TicketID:       "ticket-1",
			OwnerNamespace: "ns-1",
		},
		Status: mobileRunnerSemaphoreRunQueued,
	}

	_, err := rt.handleEnqueueRun(MobileRunnerSemaphoreEnqueueRunRequest{
		TicketID:          "ticket-1",
		OwnerNamespace:    "ns-2",
		RunnerID:          "runner-1",
		EnqueuedAt:        time.Now(),
		RequiredRunnerIDs: []string{"runner-1"},
		LeaderRunnerID:    "runner-1",
	})
	require.Error(t, err)
}

func TestHandleEnqueueRunQueueLimit(t *testing.T) {
	rt := newRuntimeForTests()
	rt.runTickets["ticket-1"] = MobileRunnerSemaphoreRunTicketState{
		Request: MobileRunnerSemaphoreEnqueueRunRequest{
			TicketID:       "ticket-1",
			OwnerNamespace: "ns-1",
		},
		Status: mobileRunnerSemaphoreRunQueued,
	}

	_, err := rt.handleEnqueueRun(MobileRunnerSemaphoreEnqueueRunRequest{
		TicketID:            "ticket-2",
		OwnerNamespace:      "ns-1",
		RunnerID:            "runner-1",
		EnqueuedAt:          time.Now(),
		RequiredRunnerIDs:   []string{"runner-1"},
		LeaderRunnerID:      "runner-1",
		MaxPipelinesInQueue: 1,
	})
	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, MobileRunnerSemaphoreErrQueueLimitExceeded, appErr.Type())
}

func TestHandleEnqueueRunSuccess(t *testing.T) {
	rt := newRuntimeForTests()
	now := time.Now()
	resp, err := rt.handleEnqueueRun(MobileRunnerSemaphoreEnqueueRunRequest{
		TicketID:          "ticket-1",
		OwnerNamespace:    "ns-1",
		RunnerID:          "runner-1",
		EnqueuedAt:        now,
		RequiredRunnerIDs: []string{"runner-1"},
		LeaderRunnerID:    "runner-1",
	})
	require.NoError(t, err)
	require.Equal(t, mobileRunnerSemaphoreRunQueued, resp.Status)
	require.Len(t, rt.runQueue, 1)
}

func TestHandleCancelRunPaths(t *testing.T) {
	rt := newRuntimeForTests()
	view, err := rt.handleCancelRun(MobileRunnerSemaphoreRunCancelRequest{
		TicketID:       "missing",
		OwnerNamespace: "ns-1",
	})
	require.NoError(t, err)
	require.Equal(t, mobileRunnerSemaphoreRunNotFound, view.Status)

	rt.runTickets["ticket-1"] = MobileRunnerSemaphoreRunTicketState{
		Request: MobileRunnerSemaphoreEnqueueRunRequest{
			TicketID:       "ticket-1",
			OwnerNamespace: "ns-1",
		},
		Status: mobileRunnerSemaphoreRunQueued,
	}
	rt.runQueue = []string{"ticket-1"}
	view, err = rt.handleCancelRun(MobileRunnerSemaphoreRunCancelRequest{
		TicketID:       "ticket-1",
		OwnerNamespace: "ns-1",
	})
	require.NoError(t, err)
	require.Equal(t, mobileRunnerSemaphoreRunNotFound, view.Status)
	require.Empty(t, rt.runTickets)

	rt.runTickets["ticket-2"] = MobileRunnerSemaphoreRunTicketState{
		Request: MobileRunnerSemaphoreEnqueueRunRequest{
			TicketID:       "ticket-2",
			OwnerNamespace: "ns-1",
		},
		Status: mobileRunnerSemaphoreRunRunning,
	}
	view, err = rt.handleCancelRun(MobileRunnerSemaphoreRunCancelRequest{
		TicketID:       "ticket-2",
		OwnerNamespace: "ns-1",
	})
	require.NoError(t, err)
	require.Equal(t, mobileRunnerSemaphoreRunRunning, view.Status)
	require.True(t, rt.runTickets["ticket-2"].CancelRequested)
}

func TestHandleRunDoneRemovesTicket(t *testing.T) {
	rt := newRuntimeForTests()
	rt.runnerID = "runner-2"
	rt.runTickets["ticket-1"] = MobileRunnerSemaphoreRunTicketState{
		Request: MobileRunnerSemaphoreEnqueueRunRequest{
			TicketID:       "ticket-1",
			OwnerNamespace: "ns-1",
			LeaderRunnerID: "runner-1",
		},
		Status:     mobileRunnerSemaphoreRunRunning,
		WorkflowID: "wf-1",
		RunID:      "run-1",
	}
	rt.runQueue = []string{"ticket-1"}

	var ctx workflow.Context
	view, err := rt.handleRunDone(ctx, MobileRunnerSemaphoreRunDoneRequest{
		TicketID:       "ticket-1",
		OwnerNamespace: "ns-1",
	})
	require.NoError(t, err)
	require.Equal(t, mobileRunnerSemaphoreRunNotFound, view.Status)
	require.Empty(t, rt.runTickets)
}

func TestHandleRunStatusQueryQueued(t *testing.T) {
	rt := newRuntimeForTests()
	rt.runTickets["ticket-1"] = MobileRunnerSemaphoreRunTicketState{
		Request: MobileRunnerSemaphoreEnqueueRunRequest{
			TicketID:       "ticket-1",
			OwnerNamespace: "ns-1",
		},
		Status: mobileRunnerSemaphoreRunQueued,
	}
	rt.runQueue = []string{"ticket-1"}

	view, err := rt.handleRunStatusQuery("ns-1", "ticket-1")
	require.NoError(t, err)
	require.Equal(t, 0, view.Position)
	require.Equal(t, 1, view.LineLen)
}

func TestHandleListQueuedRunsQuery(t *testing.T) {
	rt := newRuntimeForTests()
	rt.runQueue = []string{"ticket-1", "ticket-2"}
	rt.runTickets["ticket-1"] = MobileRunnerSemaphoreRunTicketState{
		Request: MobileRunnerSemaphoreEnqueueRunRequest{
			TicketID:       "ticket-1",
			OwnerNamespace: "ns-1",
		},
		Status: mobileRunnerSemaphoreRunQueued,
	}
	rt.runTickets["ticket-2"] = MobileRunnerSemaphoreRunTicketState{
		Request: MobileRunnerSemaphoreEnqueueRunRequest{
			TicketID:       "ticket-2",
			OwnerNamespace: "ns-2",
		},
		Status: mobileRunnerSemaphoreRunQueued,
	}

	views := rt.handleListQueuedRunsQuery("ns-1")
	require.Len(t, views, 1)
	require.Equal(t, "ticket-1", views[0].TicketID)
}

func TestHandleAcquireImmediateGrant(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (MobileRunnerSemaphorePermit, error) {
			rt := &mobileRunnerSemaphoreRuntime{
				runnerID: "runner-1",
				capacity: 1,
				requests: map[string]MobileRunnerSemaphoreRequestState{},
				holders:  map[string]MobileRunnerSemaphoreHolder{},
				queue:    []string{},
			}
			return rt.handleAcquire(ctx, MobileRunnerSemaphoreAcquireRequest{
				RequestID:   "req-1",
				LeaseID:     "lease-1",
				WaitTimeout: time.Second,
			})
		},
		workflow.RegisterOptions{Name: "test-acquire-grant"},
	)

	env.ExecuteWorkflow("test-acquire-grant")
	require.NoError(t, env.GetWorkflowError())

	var permit MobileRunnerSemaphorePermit
	require.NoError(t, env.GetWorkflowResult(&permit))
	require.Equal(t, "lease-1", permit.LeaseID)
}

func TestHandleAcquireTimeout(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			rt := &mobileRunnerSemaphoreRuntime{
				runnerID: "runner-1",
				capacity: 0,
				requests: map[string]MobileRunnerSemaphoreRequestState{},
				holders:  map[string]MobileRunnerSemaphoreHolder{},
				queue:    []string{},
			}
			_, err := rt.handleAcquire(ctx, MobileRunnerSemaphoreAcquireRequest{
				RequestID:   "req-1",
				LeaseID:     "lease-1",
				WaitTimeout: time.Second,
			})
			return err
		},
		workflow.RegisterOptions{Name: "test-acquire-timeout"},
	)

	env.ExecuteWorkflow("test-acquire-timeout")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), MobileRunnerSemaphoreErrTimeout)
}

func TestHandleReleaseResults(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (struct {
			Released MobileRunnerSemaphoreReleaseResult
			Missing  MobileRunnerSemaphoreReleaseResult
		}, error) {
			rt := &mobileRunnerSemaphoreRuntime{
				runnerID: "runner-1",
				capacity: 1,
				requests: map[string]MobileRunnerSemaphoreRequestState{},
				holders: map[string]MobileRunnerSemaphoreHolder{
					"lease-1": {LeaseID: "lease-1"},
				},
				queue: []string{},
			}
			released, err := rt.handleRelease(ctx, MobileRunnerSemaphoreReleaseRequest{LeaseID: "lease-1"})
			if err != nil {
				return struct {
					Released MobileRunnerSemaphoreReleaseResult
					Missing  MobileRunnerSemaphoreReleaseResult
				}{}, err
			}
			missing, err := rt.handleRelease(ctx, MobileRunnerSemaphoreReleaseRequest{LeaseID: "missing"})
			if err != nil {
				return struct {
					Released MobileRunnerSemaphoreReleaseResult
					Missing  MobileRunnerSemaphoreReleaseResult
				}{}, err
			}
			return struct {
				Released MobileRunnerSemaphoreReleaseResult
				Missing  MobileRunnerSemaphoreReleaseResult
			}{
				Released: released,
				Missing:  missing,
			}, nil
		},
		workflow.RegisterOptions{Name: "test-release"},
	)

	env.ExecuteWorkflow("test-release")
	require.NoError(t, env.GetWorkflowError())

	var result struct {
		Released MobileRunnerSemaphoreReleaseResult
		Missing  MobileRunnerSemaphoreReleaseResult
	}
	require.NoError(t, env.GetWorkflowResult(&result))
	require.True(t, result.Released.Released)
	require.False(t, result.Missing.Released)
}
