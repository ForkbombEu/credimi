// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// registerOpenPipelineCheck makes the safety net see every pipeline running.
func registerOpenPipelineCheck(env *testsuite.TestWorkflowEnvironment) {
	env.RegisterActivityWithOptions(
		func(context.Context, workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{
				Output: activities.CheckWorkflowClosedActivityOutput{
					Closed: false,
					Status: "Running",
				},
			}, nil
		},
		activity.RegisterOptions{Name: activities.NewCheckWorkflowClosedActivity().Name()},
	)
}

// registerPipelineCancelStubs lets the cancel-time shutdown of a running
// ticket succeed.
func registerPipelineCancelStubs(env *testsuite.TestWorkflowEnvironment) {
	env.RegisterActivityWithOptions(
		func(context.Context, workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{
				Output: activities.SignalWorkflowActivityOutput{Signaled: true, Status: "SIGNALED"},
			}, nil
		},
		activity.RegisterOptions{Name: activities.NewSignalWorkflowActivity().Name()},
	)
	env.RegisterActivityWithOptions(
		func(context.Context, workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{
				Output: activities.CancelWorkflowActivityOutput{Canceled: true, Status: "CANCELED"},
			}, nil
		},
		activity.RegisterOptions{Name: activities.NewCancelWorkflowActivity().Name()},
	)
}

func queuedTimeoutTicket(
	ticketID string,
	enqueuedAt time.Time,
	deviceIDs []string,
) MobileDeviceSemaphoreRunTicketState {
	return MobileDeviceSemaphoreRunTicketState{
		Request: MobileDeviceSemaphoreEnqueueRunRequest{
			TicketID:           ticketID,
			OwnerNamespace:     "tenant-1",
			EnqueuedAt:         enqueuedAt,
			DeviceID:           "runner-1",
			RequiredDeviceIDs:  deviceIDs,
			LeaderDeviceID:     "runner-1",
			WaitTimeoutSeconds: 60,
		},
		Status: mobileDeviceSemaphoreRunQueued,
	}
}

func TestMobileDeviceSemaphoreWorkflowExpiresQueuedTicket(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	enqueuedAt := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	env.SetStartTime(enqueuedAt)

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})
	registerOpenPipelineCheck(env)
	registerPipelineCancelStubs(env)

	var cleanedWallets []string
	env.RegisterActivityWithOptions(
		func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			payload, err := workflowengine.DecodePayload[activities.CleanupMobileDeviceSemaphoreResourcesActivityInput](
				input.Payload,
			)
			require.NoError(t, err)
			cleanedWallets = append(cleanedWallets, payload.Cleanup.TempWalletVersionID)
			return workflowengine.ActivityResult{
				Output: activities.CleanupMobileDeviceSemaphoreResourcesActivityOutput{},
			}, nil
		},
		activity.RegisterOptions{
			Name: activities.CleanupMobileDeviceSemaphoreResourcesActivityName,
		},
	)
	env.OnSignalExternalWorkflow(
		mock.Anything,
		MobileDeviceSemaphoreWorkflowID("runner-2"),
		"",
		MobileDeviceSemaphoreRunDoneSignalName,
		mock.MatchedBy(func(signal MobileDeviceSemaphoreRunDoneSignal) bool {
			return signal.TicketID == "ticket-queued" &&
				signal.WorkflowResult == "failed" &&
				signal.ErrorMessage != ""
		}),
	).Return(nil).Once()

	queued := queuedTimeoutTicket("ticket-queued", enqueuedAt, []string{"runner-1", "runner-2"})
	queued.Request.Cleanup = &MobileDeviceSemaphoreCleanupMetadata{TempWalletVersionID: "wallet-1"}
	// A running ticket past its wait deadline is not expired.
	running := queuedTimeoutTicket(
		"ticket-running",
		enqueuedAt.Add(-time.Hour),
		[]string{"runner-1"},
	)
	running.Status = mobileDeviceSemaphoreRunRunning
	running.WorkflowID = "wf-1"
	running.RunID = "run-1"
	running.WorkflowNamespace = "tenant-1"

	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 2)
	errCh := make(chan error, 2)
	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-queued", statusCh, errCh)
		queryRunStatus(env, "tenant-1", "ticket-running", statusCh, errCh)
	}, 90*time.Second)
	env.RegisterDelayedCallback(env.CancelWorkflow, 100*time.Second)

	env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
		Payload: MobileDeviceSemaphoreWorkflowInput{
			DeviceID: "runner-1",
			Capacity: 1,
			State: &MobileDeviceSemaphoreWorkflowState{
				Capacity: 1,
				RunQueue: []string{"ticket-queued"},
				RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
					"ticket-queued":  queued,
					"ticket-running": running,
				},
			},
		},
	})

	require.Empty(t, drainErrors(errCh))
	require.Len(t, statusCh, 2)
	expired := <-statusCh
	require.Equal(t, mobileDeviceSemaphoreRunFailed, expired.Status)
	require.Contains(t, expired.ErrorMessage, "queue wait timeout")
	require.Contains(t, expired.ErrorMessage, "1m0s")
	stillRunning := <-statusCh
	require.Equal(t, mobileDeviceSemaphoreRunRunning, stillRunning.Status)
	require.Equal(t, []string{"wallet-1"}, cleanedWallets)
	env.AssertExpectations(t)
}

func TestMobileDeviceSemaphoreWorkflowKeepsQueuedTicketBeforeTimeoutChange(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	enqueuedAt := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	env.SetStartTime(enqueuedAt)

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})
	env.OnGetVersion(mobileDeviceSemaphoreQueueWaitTimeoutChange, workflow.DefaultVersion, 1).
		Return(workflow.DefaultVersion)

	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 1)
	errCh := make(chan error, 1)
	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-queued", statusCh, errCh)
	}, 90*time.Second)
	env.RegisterDelayedCallback(env.CancelWorkflow, 100*time.Second)

	env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
		Payload: MobileDeviceSemaphoreWorkflowInput{
			DeviceID: "runner-1",
			Capacity: 1,
			State: &MobileDeviceSemaphoreWorkflowState{
				Capacity: 1,
				// Paused, so the ticket is not granted while it waits.
				Paused:   true,
				RunQueue: []string{"ticket-queued"},
				RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
					"ticket-queued": queuedTimeoutTicket(
						"ticket-queued",
						enqueuedAt,
						[]string{"runner-1"},
					),
				},
			},
		},
	})

	require.Empty(t, drainErrors(errCh))
	require.Len(t, statusCh, 1)
	require.Equal(t, mobileDeviceSemaphoreRunQueued, (<-statusCh).Status)
}

func TestHandleRunDoneSignalAppliesPeerExpiry(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) (map[string]MobileDeviceSemaphoreRunTicketState, error) {
			rt := newRuntimeForTests()
			rt.runTickets["ticket-starting"] = MobileDeviceSemaphoreRunTicketState{
				Request: MobileDeviceSemaphoreEnqueueRunRequest{TicketID: "ticket-starting"},
				Status:  mobileDeviceSemaphoreRunStarting,
			}
			doneAt := workflow.Now(ctx)
			rt.runTickets["ticket-expired"] = MobileDeviceSemaphoreRunTicketState{
				Request:      MobileDeviceSemaphoreEnqueueRunRequest{TicketID: "ticket-expired"},
				Status:       mobileDeviceSemaphoreRunFailed,
				ErrorMessage: "own expiry",
				DoneAt:       &doneAt,
			}
			for _, ticketID := range []string{"ticket-starting", "ticket-expired"} {
				rt.handleRunDoneSignal(ctx, MobileDeviceSemaphoreRunDoneSignal{
					TicketID:       ticketID,
					WorkflowResult: "failed",
					ErrorMessage:   "queue wait timeout: peer",
				})
			}
			return rt.runTickets, nil
		},
		workflow.RegisterOptions{Name: "test-run-done-signal-peer-expiry"},
	)

	env.ExecuteWorkflow("test-run-done-signal-peer-expiry")
	require.NoError(t, env.GetWorkflowError())
	var tickets map[string]MobileDeviceSemaphoreRunTicketState
	require.NoError(t, env.GetWorkflowResult(&tickets))
	require.Equal(t, mobileDeviceSemaphoreRunFailed, tickets["ticket-starting"].Status)
	require.Equal(t, "queue wait timeout: peer", tickets["ticket-starting"].ErrorMessage)
	require.Equal(t, "own expiry", tickets["ticket-expired"].ErrorMessage)
}

func TestHandleCancelRunCancelsRunningPipelineGracefully(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	var cancels atomic.Int32
	env.RegisterActivityWithOptions(
		func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			payload, err := workflowengine.DecodePayload[activities.CancelWorkflowActivityInput](
				input.Payload,
			)
			require.NoError(t, err)
			require.Equal(t, "wf-1", payload.WorkflowID)
			require.Equal(t, "run-1", payload.RunID)
			require.Equal(t, "tenant-1", payload.WorkflowNamespace)
			if cancels.Add(1) == 1 {
				// The first cancel fails; the next DELETE must still cancel.
				return workflowengine.ActivityResult{}, assert.AnError
			}
			return workflowengine.ActivityResult{
				Output: activities.CancelWorkflowActivityOutput{Canceled: true, Status: "CANCELED"},
			}, nil
		},
		activity.RegisterOptions{Name: activities.NewCancelWorkflowActivity().Name()},
	)
	// A graceful cancel sends no skip-cleanup policy.
	env.RegisterActivityWithOptions(
		func(context.Context, workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			assert.Fail(t, "graceful cancel must not send a cancellation policy")
			return workflowengine.ActivityResult{}, nil
		},
		activity.RegisterOptions{Name: activities.NewSignalWorkflowActivity().Name()},
	)

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) ([]MobileDeviceSemaphoreRunStatusView, error) {
			rt := newRuntimeForTests()
			rt.cancelRunningPipelines = true
			rt.runTickets["ticket-1"] = MobileDeviceSemaphoreRunTicketState{
				Request: MobileDeviceSemaphoreEnqueueRunRequest{
					TicketID:       "ticket-1",
					OwnerNamespace: "tenant-1",
				},
				Status:            mobileDeviceSemaphoreRunRunning,
				WorkflowID:        "wf-1",
				RunID:             "run-1",
				WorkflowNamespace: "tenant-1",
			}
			req := MobileDeviceSemaphoreRunCancelRequest{
				TicketID:       "ticket-1",
				OwnerNamespace: "tenant-1",
			}
			first, err := rt.handleCancelRun(ctx, req)
			if err != nil {
				return nil, err
			}
			// A second cancel retries the pipeline cancel.
			second, err := rt.handleCancelRun(ctx, req)
			return []MobileDeviceSemaphoreRunStatusView{first, second}, err
		},
		workflow.RegisterOptions{Name: "test-cancel-running-graceful"},
	)

	env.ExecuteWorkflow("test-cancel-running-graceful")
	require.NoError(t, env.GetWorkflowError())
	var views []MobileDeviceSemaphoreRunStatusView
	require.NoError(t, env.GetWorkflowResult(&views))
	require.Len(t, views, 2)
	// The ticket keeps the device until the pipeline reports RunDone.
	require.Equal(t, mobileDeviceSemaphoreRunRunning, views[0].Status)
	require.Equal(t, mobileDeviceSemaphoreRunRunning, views[1].Status)
	require.Equal(t, int32(2), cancels.Load())
}

// runningTicket is a single-device ticket whose pipeline is running.
func runningTicket() MobileDeviceSemaphoreRunTicketState {
	return MobileDeviceSemaphoreRunTicketState{
		Request: MobileDeviceSemaphoreEnqueueRunRequest{
			TicketID:          "ticket-1",
			OwnerNamespace:    "tenant-1",
			DeviceID:          "runner-1",
			RequiredDeviceIDs: []string{"runner-1"},
			LeaderDeviceID:    "runner-1",
		},
		Status:            mobileDeviceSemaphoreRunRunning,
		WorkflowID:        "wf-1",
		RunID:             "run-1",
		WorkflowNamespace: "tenant-1",
	}
}

func TestMobileDeviceSemaphoreWorkflowCancelRunning(t *testing.T) {
	tests := []struct {
		name         string
		beforeChange bool
		wantCancels  int32
	}{
		{name: "cancels the pipeline", wantCancels: 1},
		{name: "only flags the ticket before the change", beforeChange: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			w := NewMobileDeviceSemaphoreWorkflow()
			env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})
			registerOpenPipelineCheck(env)
			var cancels atomic.Int32
			env.RegisterActivityWithOptions(
				func(context.Context, workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
					cancels.Add(1)
					return workflowengine.ActivityResult{
						Output: activities.CancelWorkflowActivityOutput{
							Canceled: true,
							Status:   "CANCELED",
						},
					}, nil
				},
				activity.RegisterOptions{Name: activities.NewCancelWorkflowActivity().Name()},
			)
			env.RegisterActivityWithOptions(
				func(context.Context, workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
					return workflowengine.ActivityResult{
						Output: activities.SignalWorkflowActivityOutput{
							Signaled: true,
							Status:   "SIGNALED",
						},
					}, nil
				},
				activity.RegisterOptions{Name: activities.NewSignalWorkflowActivity().Name()},
			)
			if tt.beforeChange {
				env.OnGetVersion(mobileDeviceSemaphoreCancelRunningChange, workflow.DefaultVersion, 1).
					Return(workflow.DefaultVersion)
			}

			errCh := make(chan error, 2)
			statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 1)
			var cancelsBeforeShutdown int32
			env.RegisterDelayedCallback(func() {
				enqueueCancelRunUpdate(env, "cancel-1", MobileDeviceSemaphoreRunCancelRequest{
					TicketID:       "ticket-1",
					OwnerNamespace: "tenant-1",
				}, errCh)
			}, time.Second)
			env.RegisterDelayedCallback(func() {
				cancelsBeforeShutdown = cancels.Load()
				queryRunStatus(env, "tenant-1", "ticket-1", statusCh, errCh)
			}, 2*time.Second)
			env.RegisterDelayedCallback(env.CancelWorkflow, 3*time.Second)

			env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
				Payload: MobileDeviceSemaphoreWorkflowInput{
					DeviceID: "runner-1",
					Capacity: 1,
					State: &MobileDeviceSemaphoreWorkflowState{
						Capacity: 1,
						RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
							"ticket-1": runningTicket(),
						},
					},
				},
			})

			require.Empty(t, drainErrors(errCh))
			require.Equal(t, tt.wantCancels, cancelsBeforeShutdown)
			require.Len(t, statusCh, 1)
			status := <-statusCh
			require.Equal(t, mobileDeviceSemaphoreRunRunning, status.Status)
		})
	}
}
