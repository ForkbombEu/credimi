// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"context"
	"errors"
	"testing"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestMobileDeviceSemaphoreWorkflowGetOptions(t *testing.T) {
	w := NewMobileDeviceSemaphoreWorkflow()
	require.Equal(t, DefaultActivityOptions, w.GetOptions())
}

func TestMobileDeviceSemaphoreWorkflowRunQueuePositions(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	errCh := make(chan error, 4)
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 3)

	enqueuedAt := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	runningAt := enqueuedAt.Add(-time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:          "ticket-b",
				OwnerNamespace:    "tenant-1",
				EnqueuedAt:        enqueuedAt,
				DeviceID:          "runner-1",
				RequiredDeviceIDs: []string{"runner-1"},
				LeaderDeviceID:    "runner-1",
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
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:          "ticket-a",
				OwnerNamespace:    "tenant-1",
				EnqueuedAt:        enqueuedAt,
				DeviceID:          "runner-1",
				RequiredDeviceIDs: []string{"runner-1"},
				LeaderDeviceID:    "runner-1",
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
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
				State: &MobileDeviceSemaphoreWorkflowState{
					Capacity: 1,
					RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
						"ticket-running": {
							Request: MobileDeviceSemaphoreEnqueueRunRequest{
								TicketID:          "ticket-running",
								OwnerNamespace:    "tenant-1",
								EnqueuedAt:        runningAt,
								DeviceID:          "runner-1",
								RequiredDeviceIDs: []string{"runner-1"},
								LeaderDeviceID:    "runner-1",
							},
							Status:    mobileDeviceSemaphoreRunRunning,
							StartedAt: &runningAt,
						},
					},
				},
			},
		})
		close(done)
	}()

	<-done

	var ticketAStatuses []MobileDeviceSemaphoreRunStatusView
	var ticketBStatuses []MobileDeviceSemaphoreRunStatusView
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

	require.Equal(t, mobileDeviceSemaphoreRunQueued, ticketAStatuses[0].Status)
	require.Equal(t, 0, ticketAStatuses[0].Position)
	require.Equal(t, 2, ticketAStatuses[0].LineLen)

	var sawInitial bool
	var sawReordered bool
	for _, status := range ticketBStatuses {
		require.Equal(t, mobileDeviceSemaphoreRunQueued, status.Status)
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

	require.Empty(t, drainErrors(errCh))
}

func TestMobileDeviceSemaphoreWorkflowNotifiesQueuedPositionForFollowerRunner(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	updateAct := activities.NewUpdateGitHubPRCommentActivity()
	env.RegisterActivityWithOptions(
		updateAct.Execute,
		activity.RegisterOptions{Name: updateAct.Name()},
	)
	env.OnActivity(
		updateAct.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			update, err := workflowengine.DecodePayload[activities.UpdateGitHubPRCommentInput](
				input.Payload,
			)
			return err == nil &&
				update.Repository == "forkbombeu/wallet" &&
				update.PullRequestNumber == 17 &&
				update.CommitSHA == "abcdef1234567890" &&
				update.PipelineID == "tenant-a/pipeline-a" &&
				update.DeviceID == "runner-2" &&
				update.RunnerType == "android_emulator" &&
				update.Status == string(mobileDeviceSemaphoreRunQueued) &&
				update.Position != nil &&
				*update.Position == 1
		}),
	).Return(workflowengine.ActivityResult{}, nil).Once()

	errCh := make(chan error, 1)
	enqueuedAt := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	runningAt := enqueuedAt.Add(-time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-follower-notification",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-1",
				OwnerNamespace:     "tenant-a",
				EnqueuedAt:         enqueuedAt,
				DeviceID:           "runner-2",
				RequiredDeviceIDs:  []string{"runner-1", "runner-2"},
				LeaderDeviceID:     "runner-1",
				PipelineIdentifier: "tenant-a/pipeline-a",
				Notification: &MobileDeviceSemaphoreNotification{
					GitHubPR: &MobileDeviceSemaphoreGitHubPRNotification{
						Repository:         "forkbombeu/wallet",
						PullRequestNumber:  17,
						CommitSHA:          "abcdef1234567890",
						PipelineIdentifier: "tenant-a/pipeline-a",
						DeviceID:           "runner-1",
						RunnerType:         "android_phone",
						RunnerTypes: map[string]string{
							"runner-1": "android_phone",
							"runner-2": "android_emulator",
						},
					},
				},
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*3)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-2",
				Capacity: 1,
				State: &MobileDeviceSemaphoreWorkflowState{
					Capacity: 1,
					RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
						"ticket-running": {
							Request: MobileDeviceSemaphoreEnqueueRunRequest{
								TicketID:          "ticket-running",
								OwnerNamespace:    "tenant-a",
								EnqueuedAt:        runningAt,
								DeviceID:          "runner-2",
								RequiredDeviceIDs: []string{"runner-2"},
								LeaderDeviceID:    "runner-2",
							},
							Status:    mobileDeviceSemaphoreRunRunning,
							StartedAt: &runningAt,
						},
					},
				},
			},
		})
		close(done)
	}()

	<-done

	require.Empty(t, drainErrors(errCh))
	env.AssertExpectations(t)
}

func TestMobileDeviceSemaphoreWorkflowListQueuedRuns(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	errCh := make(chan error, 4)
	queuedCh := make(chan []MobileDeviceSemaphoreQueuedRunView, 1)

	enqueuedAt := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	runningAt := enqueuedAt.Add(-time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-1",
				OwnerNamespace:     "tenant-a",
				EnqueuedAt:         enqueuedAt,
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				LeaderDeviceID:     "runner-1",
				PipelineIdentifier: "tenant-a/pipeline-a",
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-2",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-2",
				OwnerNamespace:     "tenant-b",
				EnqueuedAt:         enqueuedAt.Add(time.Minute),
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				LeaderDeviceID:     "runner-1",
				PipelineIdentifier: "tenant-b/pipeline-b",
			},
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-3",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-3",
				OwnerNamespace:     "tenant-a",
				EnqueuedAt:         enqueuedAt.Add(2 * time.Minute),
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				LeaderDeviceID:     "runner-1",
				PipelineIdentifier: "tenant-a/pipeline-a",
			},
			errCh,
		)
	}, time.Second*3)

	env.RegisterDelayedCallback(func() {
		queryQueuedRuns(env, "tenant-a", queuedCh, errCh)
	}, time.Second*4)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*5)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
				State: &MobileDeviceSemaphoreWorkflowState{
					Capacity: 1,
					RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
						"ticket-running": {
							Request: MobileDeviceSemaphoreEnqueueRunRequest{
								TicketID:          "ticket-running",
								OwnerNamespace:    "tenant-a",
								EnqueuedAt:        runningAt,
								DeviceID:          "runner-1",
								RequiredDeviceIDs: []string{"runner-1"},
								LeaderDeviceID:    "runner-1",
							},
							Status:    mobileDeviceSemaphoreRunRunning,
							StartedAt: &runningAt,
						},
					},
				},
			},
		})
		close(done)
	}()

	<-done

	var queued []MobileDeviceSemaphoreQueuedRunView
	select {
	case queued = <-queuedCh:
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for queued runs")
	}

	require.Len(t, queued, 2)
	require.Equal(t, "ticket-1", queued[0].TicketID)
	require.Equal(t, "tenant-a", queued[0].OwnerNamespace)
	require.Equal(t, "tenant-a/pipeline-a", queued[0].PipelineIdentifier)
	require.Equal(t, enqueuedAt, queued[0].EnqueuedAt)
	require.Equal(t, "runner-1", queued[0].LeaderDeviceID)
	require.Equal(t, []string{"runner-1"}, queued[0].RequiredDeviceIDs)
	require.Equal(t, mobileDeviceSemaphoreRunQueued, queued[0].Status)
	require.Equal(t, 0, queued[0].Position)
	require.Equal(t, 3, queued[0].LineLen)

	require.Equal(t, "ticket-3", queued[1].TicketID)
	require.Equal(t, 2, queued[1].Position)
	require.Equal(t, 3, queued[1].LineLen)

	require.Empty(t, drainErrors(errCh))
}

func TestMobileDeviceSemaphoreWorkflowQueueLimitEnforced(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	errCh := make(chan error, 2)

	enqueuedAt := time.Date(2026, 2, 3, 10, 30, 0, 0, time.UTC)
	runningAt := enqueuedAt.Add(-time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-limit-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:            "ticket-limit-1",
				OwnerNamespace:      "tenant-1",
				EnqueuedAt:          enqueuedAt,
				DeviceID:            "runner-1",
				RequiredDeviceIDs:   []string{"runner-1"},
				LeaderDeviceID:      "runner-1",
				MaxPipelinesInQueue: 1,
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-limit-2",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:            "ticket-limit-2",
				OwnerNamespace:      "tenant-1",
				EnqueuedAt:          enqueuedAt.Add(time.Second),
				DeviceID:            "runner-1",
				RequiredDeviceIDs:   []string{"runner-1"},
				LeaderDeviceID:      "runner-1",
				MaxPipelinesInQueue: 1,
			},
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*3)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
				State: &MobileDeviceSemaphoreWorkflowState{
					Capacity: 1,
					RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
						"ticket-running": {
							Request: MobileDeviceSemaphoreEnqueueRunRequest{
								TicketID:          "ticket-running",
								OwnerNamespace:    "tenant-1",
								EnqueuedAt:        runningAt,
								DeviceID:          "runner-1",
								RequiredDeviceIDs: []string{"runner-1"},
								LeaderDeviceID:    "runner-1",
							},
							Status:    mobileDeviceSemaphoreRunRunning,
							StartedAt: &runningAt,
						},
					},
				},
			},
		})
		close(done)
	}()

	<-done

	errs := drainErrors(errCh)
	require.NotEmpty(t, errs)
	for _, err := range errs {
		var appErr *temporal.ApplicationError
		require.True(t, errors.As(err, &appErr))
		require.Equal(t, MobileDeviceSemaphoreErrQueueLimitExceeded, appErr.Type())
	}
}

func TestMobileDeviceSemaphoreWorkflowQueueLimitCountsOwnerNamespaceOnly(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	errCh := make(chan error, 2)
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 1)

	enqueuedAt := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	runningAt := enqueuedAt.Add(-time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-tenant-a",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:            "ticket-tenant-a",
				OwnerNamespace:      "tenant-a",
				EnqueuedAt:          enqueuedAt,
				DeviceID:            "runner-1",
				RequiredDeviceIDs:   []string{"runner-1"},
				LeaderDeviceID:      "runner-1",
				MaxPipelinesInQueue: 1,
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-tenant-b",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:            "ticket-tenant-b",
				OwnerNamespace:      "tenant-b",
				EnqueuedAt:          enqueuedAt.Add(time.Second),
				DeviceID:            "runner-1",
				RequiredDeviceIDs:   []string{"runner-1"},
				LeaderDeviceID:      "runner-1",
				MaxPipelinesInQueue: 1,
			},
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-b", "ticket-tenant-b", statusCh, errCh)
	}, time.Second*3)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*4)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
				State: &MobileDeviceSemaphoreWorkflowState{
					Capacity: 1,
					RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
						"ticket-running": {
							Request: MobileDeviceSemaphoreEnqueueRunRequest{
								TicketID:          "ticket-running",
								OwnerNamespace:    "tenant-a",
								EnqueuedAt:        runningAt,
								DeviceID:          "runner-1",
								RequiredDeviceIDs: []string{"runner-1"},
								LeaderDeviceID:    "runner-1",
							},
							Status:    mobileDeviceSemaphoreRunRunning,
							StartedAt: &runningAt,
						},
					},
				},
			},
		})
		close(done)
	}()

	<-done

	deadline := time.After(2 * time.Second)
	for {
		select {
		case status := <-statusCh:
			require.Equal(t, mobileDeviceSemaphoreRunQueued, status.Status)
			goto done
		case err := <-errCh:
			var appErr *temporal.ApplicationError
			if errors.As(err, &appErr) &&
				appErr.Type() == MobileDeviceSemaphoreErrQueueLimitExceeded {
				continue
			}
			require.NoError(t, err)
		case <-deadline:
			require.Fail(t, "timed out waiting for status")
		}
	}

done:
	errs := drainErrors(errCh)
	for _, err := range errs {
		var appErr *temporal.ApplicationError
		require.True(t, errors.As(err, &appErr))
		require.Equal(t, MobileDeviceSemaphoreErrQueueLimitExceeded, appErr.Type())
	}
}

func TestMobileDeviceSemaphoreWorkflowQueueLimitUnlimitedWhenNonPositive(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	errCh := make(chan error, 2)
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 1)

	enqueuedAt := time.Date(2026, 2, 3, 11, 30, 0, 0, time.UTC)
	runningAt := enqueuedAt.Add(-time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-unlimited-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:            "ticket-unlimited-1",
				OwnerNamespace:      "tenant-1",
				EnqueuedAt:          enqueuedAt,
				DeviceID:            "runner-1",
				RequiredDeviceIDs:   []string{"runner-1"},
				LeaderDeviceID:      "runner-1",
				MaxPipelinesInQueue: 0,
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-unlimited-2",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:            "ticket-unlimited-2",
				OwnerNamespace:      "tenant-1",
				EnqueuedAt:          enqueuedAt.Add(time.Second),
				DeviceID:            "runner-1",
				RequiredDeviceIDs:   []string{"runner-1"},
				LeaderDeviceID:      "runner-1",
				MaxPipelinesInQueue: 0,
			},
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-unlimited-2", statusCh, errCh)
	}, time.Second*3)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*4)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
				State: &MobileDeviceSemaphoreWorkflowState{
					Capacity: 1,
					RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
						"ticket-running": {
							Request: MobileDeviceSemaphoreEnqueueRunRequest{
								TicketID:          "ticket-running",
								OwnerNamespace:    "tenant-1",
								EnqueuedAt:        runningAt,
								DeviceID:          "runner-1",
								RequiredDeviceIDs: []string{"runner-1"},
								LeaderDeviceID:    "runner-1",
							},
							Status:    mobileDeviceSemaphoreRunRunning,
							StartedAt: &runningAt,
						},
					},
				},
			},
		})
		close(done)
	}()

	<-done

	select {
	case status := <-statusCh:
		require.Equal(t, mobileDeviceSemaphoreRunQueued, status.Status)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for status")
	}

	require.Empty(t, drainErrors(errCh))
}

func TestMobileDeviceSemaphoreWorkflowQueueLimitIgnoresFailedTickets(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	startAct := activities.NewStartQueuedPipelineActivity()
	env.RegisterActivityWithOptions(
		startAct.Execute,
		activity.RegisterOptions{Name: startAct.Name()},
	)
	env.OnActivity(startAct.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, errors.New("start failed")).
		Once()
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

	errCh := make(chan error, 3)
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 2)

	enqueuedAt := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
	pipelineConfig := map[string]any{
		"app_url":   "https://example.com",
		"namespace": "tenant-1",
	}

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-failed-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:            "ticket-failed-1",
				OwnerNamespace:      "tenant-1",
				EnqueuedAt:          enqueuedAt,
				DeviceID:            "runner-1",
				RequiredDeviceIDs:   []string{"runner-1"},
				LeaderDeviceID:      "runner-1",
				PipelineIdentifier:  "pipelines/test",
				YAML:                "name: test\nsteps: []\n",
				PipelineConfig:      pipelineConfig,
				MaxPipelinesInQueue: 1,
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-failed-1", statusCh, errCh)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-after-fail",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:            "ticket-after-fail",
				OwnerNamespace:      "tenant-1",
				EnqueuedAt:          enqueuedAt.Add(2 * time.Second),
				DeviceID:            "runner-1",
				RequiredDeviceIDs:   []string{"runner-1"},
				LeaderDeviceID:      "runner-1",
				PipelineIdentifier:  "pipelines/test",
				YAML:                "name: test\nsteps: []\n",
				PipelineConfig:      pipelineConfig,
				MaxPipelinesInQueue: 1,
			},
			errCh,
		)
	}, time.Second*3)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-after-fail", statusCh, errCh)
	}, time.Second*4)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*5)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	var failedStatus *MobileDeviceSemaphoreRunStatusView
	var nextStatus *MobileDeviceSemaphoreRunStatusView
	for i := 0; i < 2; i++ {
		select {
		case status := <-statusCh:
			switch status.TicketID {
			case "ticket-failed-1":
				failedStatus = &status
			case "ticket-after-fail":
				nextStatus = &status
			}
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			require.Fail(t, "timed out waiting for status")
		}
	}

	require.NotNil(t, failedStatus)
	require.Equal(t, mobileDeviceSemaphoreRunFailed, failedStatus.Status)
	require.NotNil(t, nextStatus)
	require.NotEqual(t, mobileDeviceSemaphoreRunNotFound, nextStatus.Status)

	require.Empty(t, drainErrors(errCh))
}

func TestMobileDeviceSemaphoreWorkflowRunCancelQueued(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	errCh := make(chan error, 3)
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 2)

	enqueuedAt := time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
	runningAt := enqueuedAt.Add(-time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-cancel-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:          "ticket-cancel-1",
				OwnerNamespace:    "tenant-1",
				EnqueuedAt:        enqueuedAt,
				DeviceID:          "runner-1",
				RequiredDeviceIDs: []string{"runner-1"},
				LeaderDeviceID:    "runner-1",
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-cancel-2",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:          "ticket-cancel-2",
				OwnerNamespace:    "tenant-1",
				EnqueuedAt:        enqueuedAt.Add(2 * time.Second),
				DeviceID:          "runner-1",
				RequiredDeviceIDs: []string{"runner-1"},
				LeaderDeviceID:    "runner-1",
			},
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		enqueueCancelRunUpdate(
			env,
			"cancel-ticket-1",
			MobileDeviceSemaphoreRunCancelRequest{
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
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
				State: &MobileDeviceSemaphoreWorkflowState{
					Capacity: 1,
					RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
						"ticket-running": {
							Request: MobileDeviceSemaphoreEnqueueRunRequest{
								TicketID:          "ticket-running",
								OwnerNamespace:    "tenant-1",
								EnqueuedAt:        runningAt,
								DeviceID:          "runner-1",
								RequiredDeviceIDs: []string{"runner-1"},
								LeaderDeviceID:    "runner-1",
							},
							Status:    mobileDeviceSemaphoreRunRunning,
							StartedAt: &runningAt,
						},
					},
				},
			},
		})
		close(done)
	}()

	<-done

	var canceledStatus *MobileDeviceSemaphoreRunStatusView
	var queuedStatus *MobileDeviceSemaphoreRunStatusView
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
	require.Equal(t, mobileDeviceSemaphoreRunNotFound, canceledStatus.Status)
	require.NotNil(t, queuedStatus)
	require.Equal(t, mobileDeviceSemaphoreRunQueued, queuedStatus.Status)
	require.Equal(t, 0, queuedStatus.Position)
	require.Equal(t, 1, queuedStatus.LineLen)

	require.Empty(t, drainErrors(errCh))
}

func TestMobileDeviceSemaphoreWorkflowRunStartsWhenCapacityAvailable(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	startAct := activities.NewStartQueuedPipelineActivity()
	env.RegisterActivityWithOptions(
		startAct.Execute,
		activity.RegisterOptions{Name: startAct.Name()},
	)
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
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 1)

	enqueuedAt := time.Date(2026, 2, 3, 13, 0, 0, 0, time.UTC)
	pipelineConfig := map[string]any{
		"app_url":   "https://example.com",
		"namespace": "tenant-1",
	}

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-start-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-start-1",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt,
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				LeaderDeviceID:     "runner-1",
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
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	select {
	case status := <-statusCh:
		require.Equal(t, mobileDeviceSemaphoreRunRunning, status.Status)
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

func TestMobileDeviceSemaphoreWorkflowRunStartedSignalFailureDoesNotAbort(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	startAct := activities.NewStartQueuedPipelineActivity()
	env.RegisterActivityWithOptions(
		startAct.Execute,
		activity.RegisterOptions{Name: startAct.Name()},
	)
	env.OnActivity(startAct.Name(), mock.Anything, mock.Anything).Return(
		workflowengine.ActivityResult{
			Output: activities.StartQueuedPipelineActivityOutput{
				WorkflowID:        "wf-signal-1",
				RunID:             "run-signal-1",
				WorkflowNamespace: "tenant-1",
			},
		},
		nil,
	)

	env.OnSignalExternalWorkflow(
		mock.Anything,
		MobileDeviceSemaphoreWorkflowID("runner-2"),
		"",
		MobileDeviceSemaphoreRunStartedSignalName,
		mock.Anything,
	).Return(errors.New("signal failed")).Once()

	errCh := make(chan error, 2)
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 1)

	enqueuedAt := time.Date(2026, 2, 3, 13, 30, 0, 0, time.UTC)
	pipelineConfig := map[string]any{
		"app_url":   "https://example.com",
		"namespace": "tenant-1",
	}

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-signal-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-signal-1",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt,
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1", "runner-2"},
				LeaderDeviceID:     "runner-1",
				PipelineIdentifier: "pipelines/test",
				YAML:               "name: test\nsteps: []\n",
				PipelineConfig:     pipelineConfig,
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(
			MobileDeviceSemaphoreRunGrantedSignalName,
			MobileDeviceSemaphoreRunGrantedSignal{
				TicketID: "ticket-signal-1",
				DeviceID: "runner-2",
			},
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-signal-1", statusCh, errCh)
	}, time.Second*3)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*4)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	select {
	case status := <-statusCh:
		require.Equal(t, mobileDeviceSemaphoreRunRunning, status.Status)
		require.Equal(t, "wf-signal-1", status.WorkflowID)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for running status")
	}

	require.Empty(t, drainErrors(errCh))
}

func TestMobileDeviceSemaphoreWorkflowReconcilesStartingFollower(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	queryAct := activities.NewQueryMobileDeviceSemaphoreRunStatusActivity()
	env.RegisterActivityWithOptions(
		queryAct.Execute,
		activity.RegisterOptions{Name: queryAct.Name()},
	)
	env.OnActivity(queryAct.Name(), mock.Anything, mock.Anything).Return(
		workflowengine.ActivityResult{
			Output: MobileDeviceSemaphoreRunStatusView{
				TicketID:          "ticket-reconcile-1",
				Status:            mobileDeviceSemaphoreRunRunning,
				WorkflowID:        "wf-reconcile-1",
				RunID:             "run-reconcile-1",
				WorkflowNamespace: "tenant-1",
			},
		},
		nil,
	).Maybe()

	env.OnSignalExternalWorkflow(
		mock.Anything,
		MobileDeviceSemaphoreWorkflowID("runner-1"),
		"",
		MobileDeviceSemaphoreRunGrantedSignalName,
		mock.Anything,
	).Return(nil).Maybe()

	errCh := make(chan error, 2)
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 1)

	enqueuedAt := time.Date(2026, 2, 3, 13, 45, 0, 0, time.UTC)
	pipelineConfig := map[string]any{
		"app_url":   "https://example.com",
		"namespace": "tenant-1",
	}

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-reconcile-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-reconcile-1",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt,
				DeviceID:           "runner-2",
				RequiredDeviceIDs:  []string{"runner-1", "runner-2"},
				LeaderDeviceID:     "runner-1",
				PipelineIdentifier: "pipelines/test",
				YAML:               "name: test\nsteps: []\n",
				PipelineConfig:     pipelineConfig,
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-reconcile-1", statusCh, errCh)
	}, runStartingReconcileInterval+5*time.Second)

	env.RegisterDelayedCallback(env.CancelWorkflow, runStartingReconcileInterval+8*time.Second)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-2",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	select {
	case status := <-statusCh:
		require.Equal(t, mobileDeviceSemaphoreRunRunning, status.Status)
		require.Equal(t, "wf-reconcile-1", status.WorkflowID)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for reconciled status")
	}

	require.Empty(t, drainErrors(errCh))
}

func TestMobileDeviceSemaphoreWorkflowRunStartFailureAdvancesQueue(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	startAct := activities.NewStartQueuedPipelineActivity()
	env.RegisterActivityWithOptions(
		startAct.Execute,
		activity.RegisterOptions{Name: startAct.Name()},
	)
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
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 2)

	enqueuedAt := time.Date(2026, 2, 3, 14, 0, 0, 0, time.UTC)
	pipelineConfig := map[string]any{
		"app_url":   "https://example.com",
		"namespace": "tenant-1",
	}

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-fail-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-fail-1",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt,
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				LeaderDeviceID:     "runner-1",
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
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-fail-2",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt.Add(2 * time.Second),
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				LeaderDeviceID:     "runner-1",
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
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	var failedStatus *MobileDeviceSemaphoreRunStatusView
	var runningStatus *MobileDeviceSemaphoreRunStatusView
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
	require.Equal(t, mobileDeviceSemaphoreRunFailed, failedStatus.Status)
	require.NotNil(t, runningStatus)
	require.Equal(t, mobileDeviceSemaphoreRunRunning, runningStatus.Status)
	require.Equal(t, "wf-2", runningStatus.WorkflowID)

	require.Empty(t, drainErrors(errCh))
}

func TestMobileDeviceSemaphoreWorkflowRunStartFailureContinuesQueue(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	startAct := activities.NewStartQueuedPipelineActivity()
	env.RegisterActivityWithOptions(
		startAct.Execute,
		activity.RegisterOptions{Name: startAct.Name()},
	)
	env.OnActivity(startAct.Name(), mock.Anything, mock.Anything).
		Return(workflowengine.ActivityResult{}, errors.New("start failed")).
		Once()
	env.OnActivity(startAct.Name(), mock.Anything, mock.Anything).Return(
		workflowengine.ActivityResult{
			Output: activities.StartQueuedPipelineActivityOutput{
				WorkflowID:        "wf-continue-1",
				RunID:             "run-continue-1",
				WorkflowNamespace: "tenant-1",
			},
		},
		nil,
	)

	errCh := make(chan error, 3)
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 2)

	enqueuedAt := time.Date(2026, 2, 3, 14, 15, 0, 0, time.UTC)
	pipelineConfig := map[string]any{
		"app_url":   "https://example.com",
		"namespace": "tenant-1",
	}

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-continue-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-continue-1",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt,
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				LeaderDeviceID:     "runner-1",
				PipelineIdentifier: "pipelines/test",
				YAML:               "name: test\nsteps: []\n",
				PipelineConfig:     pipelineConfig,
			},
			errCh,
		)
		enqueueRunUpdate(
			env,
			"enqueue-continue-2",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-continue-2",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt.Add(time.Second),
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				LeaderDeviceID:     "runner-1",
				PipelineIdentifier: "pipelines/test",
				YAML:               "name: test\nsteps: []\n",
				PipelineConfig:     pipelineConfig,
			},
			errCh,
		)
	}, time.Second)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-continue-1", statusCh, errCh)
		queryRunStatus(env, "tenant-1", "ticket-continue-2", statusCh, errCh)
	}, time.Second*2)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*3)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	var failedStatus *MobileDeviceSemaphoreRunStatusView
	var runningStatus *MobileDeviceSemaphoreRunStatusView
	for i := 0; i < 2; i++ {
		select {
		case status := <-statusCh:
			switch status.TicketID {
			case "ticket-continue-1":
				failedStatus = &status
			case "ticket-continue-2":
				runningStatus = &status
			}
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			require.Fail(t, "timed out waiting for continued status")
		}
	}

	require.NotNil(t, failedStatus)
	require.Equal(t, mobileDeviceSemaphoreRunFailed, failedStatus.Status)
	require.NotNil(t, runningStatus)
	require.Equal(t, mobileDeviceSemaphoreRunRunning, runningStatus.Status)
	require.Equal(t, "wf-continue-1", runningStatus.WorkflowID)

	require.Empty(t, drainErrors(errCh))
}

func TestMobileDeviceSemaphoreWorkflowRunDoneAdvancesQueue(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	startAct := activities.NewStartQueuedPipelineActivity()
	env.RegisterActivityWithOptions(
		startAct.Execute,
		activity.RegisterOptions{Name: startAct.Name()},
	)
	env.OnActivity(startAct.Name(), mock.Anything, mock.Anything).Return(
		workflowengine.ActivityResult{
			Output: activities.StartQueuedPipelineActivityOutput{
				WorkflowID:        "wf-4",
				RunID:             "run-4",
				WorkflowNamespace: "tenant-1",
			},
		},
		nil,
	).Once()
	env.OnActivity(startAct.Name(), mock.Anything, mock.Anything).Return(
		workflowengine.ActivityResult{
			Output: activities.StartQueuedPipelineActivityOutput{
				WorkflowID:        "wf-5",
				RunID:             "run-5",
				WorkflowNamespace: "tenant-1",
			},
		},
		nil,
	).Once()

	errCh := make(chan error, 3)
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 1)

	enqueuedAt := time.Date(2026, 2, 3, 16, 0, 0, 0, time.UTC)
	pipelineConfig := map[string]any{
		"app_url":   "https://example.com",
		"namespace": "tenant-1",
	}

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-done-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-done-1",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt,
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				LeaderDeviceID:     "runner-1",
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
			"enqueue-done-2",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-done-2",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt.Add(2 * time.Second),
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				LeaderDeviceID:     "runner-1",
				PipelineIdentifier: "pipelines/test",
				YAML:               "name: test\nsteps: []\n",
				PipelineConfig:     pipelineConfig,
			},
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		enqueueRunDoneUpdate(
			env,
			"done-1",
			MobileDeviceSemaphoreRunDoneRequest{
				TicketID:       "ticket-done-1",
				OwnerNamespace: "tenant-1",
				WorkflowID:     "wf-4",
				RunID:          "run-4",
			},
			errCh,
		)
	}, time.Second*3)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-done-2", statusCh, errCh)
	}, time.Second*4)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second*5)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	select {
	case status := <-statusCh:
		require.Equal(t, mobileDeviceSemaphoreRunRunning, status.Status)
		require.Equal(t, "wf-5", status.WorkflowID)
		require.Equal(t, "run-5", status.RunID)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for run done status")
	}

	require.Empty(t, drainErrors(errCh))
}

func TestMobileDeviceSemaphoreWorkflowSafetyNetAdvancesQueue(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	startAct := activities.NewStartQueuedPipelineActivity()
	env.RegisterActivityWithOptions(
		startAct.Execute,
		activity.RegisterOptions{Name: startAct.Name()},
	)
	checkAct := activities.NewCheckWorkflowClosedActivity()
	env.RegisterActivityWithOptions(
		checkAct.Execute,
		activity.RegisterOptions{Name: checkAct.Name()},
	)
	env.OnActivity(startAct.Name(), mock.Anything, mock.Anything).Return(
		workflowengine.ActivityResult{
			Output: activities.StartQueuedPipelineActivityOutput{
				WorkflowID:        "wf-6",
				RunID:             "run-6",
				WorkflowNamespace: "tenant-1",
			},
		},
		nil,
	).Once()
	env.OnActivity(startAct.Name(), mock.Anything, mock.Anything).Return(
		workflowengine.ActivityResult{
			Output: activities.StartQueuedPipelineActivityOutput{
				WorkflowID:        "wf-7",
				RunID:             "run-7",
				WorkflowNamespace: "tenant-1",
			},
		},
		nil,
	).Once()
	env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).Return(
		workflowengine.ActivityResult{
			Output: activities.CheckWorkflowClosedActivityOutput{
				Closed: true,
				Status: "COMPLETED",
			},
		},
		nil,
	).Once()
	env.OnActivity(checkAct.Name(), mock.Anything, mock.Anything).Return(
		workflowengine.ActivityResult{
			Output: activities.CheckWorkflowClosedActivityOutput{
				Closed: false,
				Status: "RUNNING",
			},
		},
		nil,
	)

	errCh := make(chan error, 3)
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 1)

	enqueuedAt := time.Date(2026, 2, 3, 17, 0, 0, 0, time.UTC)
	pipelineConfig := map[string]any{
		"app_url":   "https://example.com",
		"namespace": "tenant-1",
	}

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-safety-1",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-safety-1",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt,
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				LeaderDeviceID:     "runner-1",
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
			"enqueue-safety-2",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:           "ticket-safety-2",
				OwnerNamespace:     "tenant-1",
				EnqueuedAt:         enqueuedAt.Add(2 * time.Second),
				DeviceID:           "runner-1",
				RequiredDeviceIDs:  []string{"runner-1"},
				LeaderDeviceID:     "runner-1",
				PipelineIdentifier: "pipelines/test",
				YAML:               "name: test\nsteps: []\n",
				PipelineConfig:     pipelineConfig,
			},
			errCh,
		)
	}, time.Second*2)

	env.RegisterDelayedCallback(func() {
		queryRunStatus(env, "tenant-1", "ticket-safety-2", statusCh, errCh)
	}, runCompletionCheckInterval+5*time.Second)

	env.RegisterDelayedCallback(env.CancelWorkflow, runCompletionCheckInterval+10*time.Second)

	done := make(chan struct{})
	go func() {
		env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
			},
		})
		close(done)
	}()

	<-done

	select {
	case status := <-statusCh:
		require.Equal(t, mobileDeviceSemaphoreRunRunning, status.Status)
		require.Equal(t, "wf-7", status.WorkflowID)
		require.Equal(t, "run-7", status.RunID)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for safety net status")
	}

	require.Empty(t, drainErrors(errCh))
}

func TestMobileDeviceSemaphoreWorkflowRunStatusOwnerMismatch(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	errCh := make(chan error, 2)
	statusCh := make(chan MobileDeviceSemaphoreRunStatusView, 1)

	enqueuedAt := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	runningAt := enqueuedAt.Add(-time.Second)

	env.RegisterDelayedCallback(func() {
		enqueueRunUpdate(
			env,
			"enqueue-owner",
			MobileDeviceSemaphoreEnqueueRunRequest{
				TicketID:          "ticket-owner",
				OwnerNamespace:    "tenant-1",
				EnqueuedAt:        enqueuedAt,
				DeviceID:          "runner-1",
				RequiredDeviceIDs: []string{"runner-1"},
				LeaderDeviceID:    "runner-1",
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
			Payload: MobileDeviceSemaphoreWorkflowInput{
				DeviceID: "runner-1",
				Capacity: 1,
				State: &MobileDeviceSemaphoreWorkflowState{
					Capacity: 1,
					RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
						"ticket-running": {
							Request: MobileDeviceSemaphoreEnqueueRunRequest{
								TicketID:          "ticket-running",
								OwnerNamespace:    "tenant-1",
								EnqueuedAt:        runningAt,
								DeviceID:          "runner-1",
								RequiredDeviceIDs: []string{"runner-1"},
								LeaderDeviceID:    "runner-1",
							},
							Status:    mobileDeviceSemaphoreRunRunning,
							StartedAt: &runningAt,
						},
					},
				},
			},
		})
		close(done)
	}()

	<-done

	select {
	case status := <-statusCh:
		require.Equal(t, mobileDeviceSemaphoreRunNotFound, status.Status)
		require.Zero(t, status.LineLen)
		require.Zero(t, status.Position)
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.Fail(t, "timed out waiting for owner mismatch status")
	}

	require.Empty(t, drainErrors(errCh))
}

func enqueueCancelRunUpdate(
	env *testsuite.TestWorkflowEnvironment,
	updateID string,
	req MobileDeviceSemaphoreRunCancelRequest,
	errCh chan<- error,
) {
	env.UpdateWorkflow(
		MobileDeviceSemaphoreCancelRunUpdate,
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
	req MobileDeviceSemaphoreEnqueueRunRequest,
	errCh chan<- error,
) {
	env.UpdateWorkflow(
		MobileDeviceSemaphoreEnqueueRunUpdate,
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

func enqueueRunDoneUpdate(
	env *testsuite.TestWorkflowEnvironment,
	updateID string,
	req MobileDeviceSemaphoreRunDoneRequest,
	errCh chan<- error,
) {
	env.UpdateWorkflow(
		MobileDeviceSemaphoreRunDoneUpdate,
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
	statusCh chan<- MobileDeviceSemaphoreRunStatusView,
	errCh chan<- error,
) {
	encoded, err := env.QueryWorkflow(MobileDeviceSemaphoreRunStatusQuery, ownerNamespace, ticketID)
	if err != nil {
		errCh <- err
		return
	}
	var status MobileDeviceSemaphoreRunStatusView
	if decodeErr := encoded.Get(&status); decodeErr != nil {
		errCh <- decodeErr
		return
	}
	statusCh <- status
}

func queryQueuedRuns(
	env *testsuite.TestWorkflowEnvironment,
	ownerNamespace string,
	queuedCh chan<- []MobileDeviceSemaphoreQueuedRunView,
	errCh chan<- error,
) {
	encoded, err := env.QueryWorkflow(MobileDeviceSemaphoreListQueuedRunsQuery, ownerNamespace)
	if err != nil {
		errCh <- err
		return
	}
	var queued []MobileDeviceSemaphoreQueuedRunView
	if decodeErr := encoded.Get(&queued); decodeErr != nil {
		errCh <- decodeErr
		return
	}
	queuedCh <- queued
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

func TestMobileDeviceSemaphoreWorkflowCancellationTriggersShutdownCleanup(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	w := NewMobileDeviceSemaphoreWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{Name: w.Name()})

	cancelAct := activities.NewCancelWorkflowActivity()
	signalAct := activities.NewSignalWorkflowActivity()
	env.RegisterActivityWithOptions(
		func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			payload, err := workflowengine.DecodePayload[activities.SignalWorkflowActivityInput](
				input.Payload,
			)
			require.NoError(t, err)
			require.Equal(t, "wf-1", payload.WorkflowID)
			require.Equal(t, pipelineinternal.PipelineCancellationPolicySignal, payload.SignalName)
			policy := workflowengine.AsMap(payload.Payload)
			require.True(t, workflowengine.AsBool(policy["skip_runner_cleanup"]))
			require.Equal(
				t,
				[]string{"runner-1"},
				workflowengine.AsSliceOfStrings(policy["skip_runner_cleanup_ids"]),
			)
			return workflowengine.ActivityResult{
				Output: activities.SignalWorkflowActivityOutput{Signaled: true, Status: "SIGNALED"},
			}, nil
		},
		activity.RegisterOptions{Name: signalAct.Name()},
	)
	env.RegisterActivityWithOptions(
		func(_ context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
			payload, err := workflowengine.DecodePayload[activities.CancelWorkflowActivityInput](
				input.Payload,
			)
			require.NoError(t, err)
			require.Equal(t, "wf-1", payload.WorkflowID)
			require.Equal(t, "tenant-1", payload.WorkflowNamespace)
			return workflowengine.ActivityResult{
				Output: activities.CancelWorkflowActivityOutput{Canceled: true, Status: "CANCELED"},
			}, nil
		},
		activity.RegisterOptions{Name: cancelAct.Name()},
	)

	env.RegisterDelayedCallback(env.CancelWorkflow, time.Second)

	env.OnSignalExternalWorkflow(
		mock.Anything,
		MobileDeviceSemaphoreWorkflowID("runner-2"),
		"",
		MobileDeviceSemaphoreRunDoneSignalName,
		mock.MatchedBy(func(signal MobileDeviceSemaphoreRunDoneSignal) bool {
			return signal.TicketID == "ticket-1" &&
				signal.WorkflowResult == "canceled"
		}),
	).Return(nil).Once()

	env.ExecuteWorkflow(w.Name(), workflowengine.WorkflowInput{
		Payload: MobileDeviceSemaphoreWorkflowInput{
			DeviceID: "runner-1",
			Capacity: 1,
			State: &MobileDeviceSemaphoreWorkflowState{
				Capacity: 1,
				RunTickets: map[string]MobileDeviceSemaphoreRunTicketState{
					"ticket-1": {
						Request: MobileDeviceSemaphoreEnqueueRunRequest{
							TicketID:          "ticket-1",
							OwnerNamespace:    "tenant-1",
							RequiredDeviceIDs: []string{"runner-1", "runner-2"},
							LeaderDeviceID:    "runner-1",
						},
						Status: mobileDeviceSemaphoreRunQueued,
					},
				},
			},
		},
	})

	require.Error(t, env.GetWorkflowError())
	require.Contains(t, env.GetWorkflowError().Error(), "canceled")
}
