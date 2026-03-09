// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"context"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func newRuntimeForTests() *mobileRunnerSemaphoreRuntime {
	return &mobileRunnerSemaphoreRuntime{
		runnerID:   "runner-1",
		runQueue:   []string{},
		runTickets: map[string]MobileRunnerSemaphoreRunTicketState{},
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

func TestHandleEnqueueRunRunnerMismatch(t *testing.T) {
	rt := newRuntimeForTests()
	_, err := rt.handleEnqueueRun(MobileRunnerSemaphoreEnqueueRunRequest{
		TicketID:          "ticket-1",
		OwnerNamespace:    "ns-1",
		RunnerID:          "runner-2",
		EnqueuedAt:        time.Now(),
		RequiredRunnerIDs: []string{"runner-2"},
		LeaderRunnerID:    "runner-2",
	})
	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, MobileRunnerSemaphoreErrInvalidRequest, appErr.Type())
}

func TestHandleEnqueueRunMissingEnqueuedAt(t *testing.T) {
	rt := newRuntimeForTests()
	_, err := rt.handleEnqueueRun(MobileRunnerSemaphoreEnqueueRunRequest{
		TicketID:          "ticket-1",
		OwnerNamespace:    "ns-1",
		RunnerID:          "runner-1",
		RequiredRunnerIDs: []string{"runner-1"},
		LeaderRunnerID:    "runner-1",
	})
	require.Error(t, err)
}

func TestHandleEnqueueRunMissingRequiredRunnerIDs(t *testing.T) {
	rt := newRuntimeForTests()
	_, err := rt.handleEnqueueRun(MobileRunnerSemaphoreEnqueueRunRequest{
		TicketID:       "ticket-1",
		OwnerNamespace: "ns-1",
		RunnerID:       "runner-1",
		EnqueuedAt:     time.Now(),
	})
	require.Error(t, err)
}

func TestHandleEnqueueRunLeaderNotInRequired(t *testing.T) {
	rt := newRuntimeForTests()
	_, err := rt.handleEnqueueRun(MobileRunnerSemaphoreEnqueueRunRequest{
		TicketID:          "ticket-1",
		OwnerNamespace:    "ns-1",
		RunnerID:          "runner-1",
		EnqueuedAt:        time.Now(),
		RequiredRunnerIDs: []string{"runner-2"},
		LeaderRunnerID:    "runner-1",
	})
	require.Error(t, err)
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

func TestHandleEnqueueRunReturnsExistingTicket(t *testing.T) {
	rt := newRuntimeForTests()
	rt.runTickets["ticket-1"] = MobileRunnerSemaphoreRunTicketState{
		Request: MobileRunnerSemaphoreEnqueueRunRequest{
			TicketID:          "ticket-1",
			OwnerNamespace:    "ns-1",
			RunnerID:          "runner-1",
			EnqueuedAt:        time.Now().Add(-time.Minute),
			RequiredRunnerIDs: []string{"runner-1"},
			LeaderRunnerID:    "runner-1",
		},
		Status: mobileRunnerSemaphoreRunQueued,
	}
	rt.runQueue = []string{"ticket-1"}

	resp, err := rt.handleEnqueueRun(MobileRunnerSemaphoreEnqueueRunRequest{
		TicketID:          "ticket-1",
		OwnerNamespace:    "ns-1",
		RunnerID:          "runner-1",
		EnqueuedAt:        time.Now(),
		RequiredRunnerIDs: []string{"runner-1"},
		LeaderRunnerID:    "runner-1",
	})
	require.NoError(t, err)
	require.Equal(t, mobileRunnerSemaphoreRunQueued, resp.Status)
	require.Equal(t, 0, resp.Position)
	require.Equal(t, 1, resp.LineLen)
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

func TestApplyPayloadStateDefaults(t *testing.T) {
	rt := &mobileRunnerSemaphoreRuntime{capacity: 0}

	rt.applyPayloadState(MobileRunnerSemaphoreWorkflowInput{})

	require.Equal(t, 1, rt.capacity)
}

func TestApplyPayloadStateOverrides(t *testing.T) {
	rt := &mobileRunnerSemaphoreRuntime{
		capacity: 5,
	}
	state := &MobileRunnerSemaphoreWorkflowState{
		Capacity:    2,
		RunQueue:    []string{"ticket-1"},
		RunTickets:  map[string]MobileRunnerSemaphoreRunTicketState{"ticket-1": {}},
		UpdateCount: 7,
	}

	rt.applyPayloadState(MobileRunnerSemaphoreWorkflowInput{
		Capacity: 3,
		State:    state,
	})

	require.Equal(t, 2, rt.capacity)
	require.Equal(t, state.RunQueue, rt.runQueue)
	require.Equal(t, state.RunTickets, rt.runTickets)
	require.Equal(t, state.UpdateCount, rt.updateCount)
}

func TestNormalizeStateInitializesDefaults(t *testing.T) {
	rt := &mobileRunnerSemaphoreRuntime{}

	rt.normalizeState()

	require.Equal(t, 1, rt.capacity)
	require.NotNil(t, rt.runQueue)
	require.NotNil(t, rt.runTickets)
}

func TestHandleRunStartedSignalUpdatesState(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (struct {
			State       MobileRunnerSemaphoreRunTicketState
			UpdateCount int
		}, error) {
			rt := &mobileRunnerSemaphoreRuntime{
				runnerID: "runner-1",
				runTickets: map[string]MobileRunnerSemaphoreRunTicketState{
					"ticket-1": {
						Request: MobileRunnerSemaphoreEnqueueRunRequest{TicketID: "ticket-1"},
						Status:  mobileRunnerSemaphoreRunStarting,
					},
				},
				runQueue: []string{},
			}

			rt.handleRunStartedSignal(ctx, MobileRunnerSemaphoreRunStartedSignal{
				TicketID:          "ticket-1",
				WorkflowID:        "wf-1",
				RunID:             "run-1",
				WorkflowNamespace: "ns-1",
			})

			return struct {
				State       MobileRunnerSemaphoreRunTicketState
				UpdateCount int
			}{
				State:       rt.runTickets["ticket-1"],
				UpdateCount: rt.updateCount,
			}, nil
		},
		workflow.RegisterOptions{Name: "test-run-started-signal"},
	)

	env.ExecuteWorkflow("test-run-started-signal")
	require.NoError(t, env.GetWorkflowError())

	var result struct {
		State       MobileRunnerSemaphoreRunTicketState
		UpdateCount int
	}
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, mobileRunnerSemaphoreRunRunning, result.State.Status)
	require.Equal(t, "wf-1", result.State.WorkflowID)
	require.Equal(t, "run-1", result.State.RunID)
	require.Equal(t, "ns-1", result.State.WorkflowNamespace)
	require.NotNil(t, result.State.StartedAt)
	require.Equal(t, 1, result.UpdateCount)
}

func TestHandleRunDoneSignalRemovesTicket(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (struct {
			RunTickets          int
			RunQueue            []string
			UpdateCount         int
			RunStarterRequested bool
		}, error) {
			rt := &mobileRunnerSemaphoreRuntime{
				runnerID: "runner-1",
				runTickets: map[string]MobileRunnerSemaphoreRunTicketState{
					"ticket-1": {
						Request: MobileRunnerSemaphoreEnqueueRunRequest{TicketID: "ticket-1"},
						Status:  mobileRunnerSemaphoreRunRunning,
					},
				},
				runQueue: []string{"ticket-1"},
			}
			rt.handleRunDoneSignal(ctx, MobileRunnerSemaphoreRunDoneSignal{
				TicketID:   "ticket-1",
				WorkflowID: "wf-1",
				RunID:      "run-1",
			})

			return struct {
				RunTickets          int
				RunQueue            []string
				UpdateCount         int
				RunStarterRequested bool
			}{
				RunTickets:          len(rt.runTickets),
				RunQueue:            rt.runQueue,
				UpdateCount:         rt.updateCount,
				RunStarterRequested: rt.runStarterRequested,
			}, nil
		},
		workflow.RegisterOptions{Name: "test-run-done-signal"},
	)

	env.ExecuteWorkflow("test-run-done-signal")
	require.NoError(t, env.GetWorkflowError())

	var result struct {
		RunTickets          int
		RunQueue            []string
		UpdateCount         int
		RunStarterRequested bool
	}
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Zero(t, result.RunTickets)
	require.Empty(t, result.RunQueue)
	require.Equal(t, 1, result.UpdateCount)
	require.True(t, result.RunStarterRequested)
}

func TestAwaitContinueTriggersContinueAsNew(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			rt := &mobileRunnerSemaphoreRuntime{
				ctx:            ctx,
				runnerID:       "runner-1",
				shouldContinue: true,
				continueInput: workflowengine.WorkflowInput{
					Payload: MobileRunnerSemaphoreWorkflowInput{RunnerID: "runner-1"},
				},
			}
			_, err := rt.awaitContinue()
			return err
		},
		workflow.RegisterOptions{Name: "test-await-continue"},
	)

	env.ExecuteWorkflow("test-await-continue")
	err := env.GetWorkflowError()
	require.Error(t, err)
	require.True(t, workflow.IsContinueAsNewError(err))
}

func TestCheckRunCompletionFinalizesClosedRun(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	checkAct := activities.NewCheckWorkflowClosedActivity()
	env.RegisterActivityWithOptions(
		func(_ context.Context, _ workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{
				Output: activities.CheckWorkflowClosedActivityOutput{
					Closed: true,
					Status: "completed",
				},
			}, nil
		},
		activity.RegisterOptions{Name: checkAct.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (int, error) {
			rt := &mobileRunnerSemaphoreRuntime{
				runnerID: "runner-1",
				runTickets: map[string]MobileRunnerSemaphoreRunTicketState{
					"ticket-1": {
						Request: MobileRunnerSemaphoreEnqueueRunRequest{
							TicketID:          "ticket-1",
							OwnerNamespace:    "ns-1",
							LeaderRunnerID:    "runner-1",
							RequiredRunnerIDs: []string{"runner-1"},
						},
						Status:            mobileRunnerSemaphoreRunRunning,
						WorkflowID:        "wf-1",
						RunID:             "run-1",
						WorkflowNamespace: "ns-1",
					},
				},
				runQueue: []string{"ticket-1"},
			}
			rt.checkRunCompletion(ctx)
			return len(rt.runTickets), nil
		},
		workflow.RegisterOptions{Name: "test-check-run-completion"},
	)

	env.ExecuteWorkflow("test-check-run-completion")
	require.NoError(t, env.GetWorkflowError())

	var remaining int
	require.NoError(t, env.GetWorkflowResult(&remaining))
	require.Equal(t, 0, remaining)
}

func TestReconcileStartingTicketsUpdatesRunning(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	queryAct := activities.NewQueryMobileRunnerSemaphoreRunStatusActivity()
	env.RegisterActivityWithOptions(
		func(_ context.Context, _ workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{
				Output: MobileRunnerSemaphoreRunStatusView{
					TicketID:          "ticket-1",
					Status:            mobileRunnerSemaphoreRunRunning,
					WorkflowID:        "wf-1",
					RunID:             "run-1",
					WorkflowNamespace: "ns-1",
				},
			}, nil
		},
		activity.RegisterOptions{Name: queryAct.Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (MobileRunnerSemaphoreRunTicketState, error) {
			rt := &mobileRunnerSemaphoreRuntime{
				runnerID: "runner-1",
				runTickets: map[string]MobileRunnerSemaphoreRunTicketState{
					"ticket-1": {
						Request: MobileRunnerSemaphoreEnqueueRunRequest{
							TicketID:       "ticket-1",
							OwnerNamespace: "ns-1",
							LeaderRunnerID: "runner-2",
						},
						Status: mobileRunnerSemaphoreRunStarting,
					},
				},
			}
			rt.reconcileStartingTickets(ctx)
			return rt.runTickets["ticket-1"], nil
		},
		workflow.RegisterOptions{Name: "test-reconcile-starting"},
	)

	env.ExecuteWorkflow("test-reconcile-starting")
	require.NoError(t, env.GetWorkflowError())

	var state MobileRunnerSemaphoreRunTicketState
	require.NoError(t, env.GetWorkflowResult(&state))
	require.Equal(t, mobileRunnerSemaphoreRunRunning, state.Status)
	require.Equal(t, "wf-1", state.WorkflowID)
	require.Equal(t, "run-1", state.RunID)
	require.Equal(t, "ns-1", state.WorkflowNamespace)
	require.NotNil(t, state.StartedAt)
}

func TestExecuteWorkflowInvalidPayload(t *testing.T) {
	w := NewMobileRunnerSemaphoreWorkflow()
	_, err := w.ExecuteWorkflow(nil, workflowengine.WorkflowInput{
		Payload: make(chan int),
	})
	require.Error(t, err)
}

func TestExecuteWorkflowMissingRunnerID(t *testing.T) {
	w := NewMobileRunnerSemaphoreWorkflow()
	_, err := w.ExecuteWorkflow(nil, workflowengine.WorkflowInput{
		Payload: map[string]any{},
	})
	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, MobileRunnerSemaphoreErrInvalidRequest, appErr.Type())
}
