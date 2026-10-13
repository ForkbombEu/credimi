// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package semaphoreclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

type fakeStartOperation struct{}

func (fakeStartOperation) Get(context.Context) (client.WorkflowRun, error) { return nil, nil }

func TestUpdateIDNormalizesDevice(t *testing.T) {
	require.Equal(
		t,
		"enqueue/tenant/runner-1/ticket-1",
		UpdateID("enqueue", "/tenant/runner-1", "ticket-1"),
	)
}

func TestEnsureWorkflowUsesExistingRun(t *testing.T) {
	c := temporalmocks.NewClient(t)
	c.On(
		"ExecuteWorkflow",
		mock.Anything,
		mock.MatchedBy(func(o client.StartWorkflowOptions) bool {
			return o.ID == mobiledevicesemaphore.WorkflowID("runner-1") &&
				o.TaskQueue == mobiledevicesemaphore.TaskQueue &&
				o.WorkflowIDConflictPolicy == enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING
		}),
		mobiledevicesemaphore.WorkflowName,
		mock.Anything,
	).Return(&temporalmocks.WorkflowRun{}, nil).Once()

	require.NoError(t, EnsureWorkflow(context.Background(), c, " /runner-1 "))
}

func TestEnsureWorkflowReturnsErrors(t *testing.T) {
	c := temporalmocks.NewClient(t)
	c.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("boom")).Once()

	require.ErrorContains(t, EnsureWorkflow(context.Background(), c, "runner-1"), "boom")
}

func TestEnqueueRunUsesUpdateWithStart(t *testing.T) {
	c := temporalmocks.NewClient(t)
	op := fakeStartOperation{}
	var startOpts client.StartWorkflowOptions
	var startArg any
	c.On(
		"NewWithStartWorkflowOperation",
		mock.Anything,
		mobiledevicesemaphore.WorkflowName,
		mock.Anything,
	).Run(func(args mock.Arguments) {
		startOpts = args.Get(0).(client.StartWorkflowOptions)
		startArg = args.Get(2)
	}).Return(op).Once()

	handle := temporalmocks.NewWorkflowUpdateHandle(t)
	handle.On("Get", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			out := args.Get(1).(*mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse)
			*out = mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse{
				TicketID: "ticket-1",
				Status:   mobiledevicesemaphore.MobileDeviceSemaphoreRunQueued,
				Position: 1,
				LineLen:  2,
			}
		}).
		Return(nil).Once()

	var updateOpts client.UpdateWithStartWorkflowOptions
	c.On("UpdateWithStartWorkflow", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			updateOpts = args.Get(1).(client.UpdateWithStartWorkflowOptions)
		}).
		Return(handle, nil).Once()

	req := mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest{
		TicketID:          "ticket-1",
		OwnerNamespace:    "tenant-1",
		EnqueuedAt:        time.Now().UTC(),
		DeviceID:          "/runner-1",
		RequiredDeviceIDs: []string{"runner-2", "/runner-1", "runner-2"},
		LeaderDeviceID:    "/runner-1",
	}
	resp, err := EnqueueRun(context.Background(), c, "/runner-1", req)
	require.NoError(t, err)
	require.Equal(t, "ticket-1", resp.TicketID)
	require.Equal(t, 1, resp.Position)

	require.Equal(t, mobiledevicesemaphore.WorkflowID("runner-1"), startOpts.ID)
	require.Equal(t, mobiledevicesemaphore.TaskQueue, startOpts.TaskQueue)
	require.Equal(
		t,
		enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		startOpts.WorkflowIDConflictPolicy,
	)
	require.Equal(t, workflowengine.WorkflowInput{
		Payload: mobiledevicesemaphore.MobileDeviceSemaphoreWorkflowInput{
			DeviceID: "runner-1",
			Capacity: 1,
		},
	}, startArg)

	require.Equal(t, op, updateOpts.StartWorkflowOperation)
	require.Equal(t, mobiledevicesemaphore.EnqueueRunUpdate, updateOpts.UpdateOptions.UpdateName)
	require.Equal(t, "enqueue/runner-1/ticket-1", updateOpts.UpdateOptions.UpdateID)
	require.Equal(t, client.WorkflowUpdateStageCompleted, updateOpts.UpdateOptions.WaitForStage)
	require.Len(t, updateOpts.UpdateOptions.Args, 1)
	sent := updateOpts.UpdateOptions.Args[0].(mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest)
	require.Equal(t, "runner-1", sent.DeviceID)
	require.Equal(t, "runner-1", sent.LeaderDeviceID)
	require.Equal(t, []string{"runner-1", "runner-2"}, sent.RequiredDeviceIDs)
}

func TestEnqueueRunReturnsUpdateErrors(t *testing.T) {
	c := temporalmocks.NewClient(t)
	c.On("NewWithStartWorkflowOperation", mock.Anything, mock.Anything, mock.Anything).
		Return(fakeStartOperation{}).Once()
	c.On("UpdateWithStartWorkflow", mock.Anything, mock.Anything).
		Return(nil, errors.New("rejected")).Once()

	_, err := EnqueueRun(
		context.Background(),
		c,
		"runner-1",
		mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest{TicketID: "ticket-1"},
	)
	require.ErrorContains(t, err, "rejected")
}

func TestCancelRun(t *testing.T) {
	t.Run("not found maps to ErrRunTicketNotFound", func(t *testing.T) {
		c := temporalmocks.NewClient(t)
		c.On("UpdateWorkflow", mock.Anything, mock.Anything).
			Return(nil, &serviceerror.NotFound{Message: "missing"}).Once()

		_, err := CancelRun(
			context.Background(),
			c,
			"runner-1",
			mobiledevicesemaphore.MobileDeviceSemaphoreRunCancelRequest{TicketID: "ticket-1"},
		)
		require.ErrorIs(t, err, ErrRunTicketNotFound)
	})

	t.Run("returns status", func(t *testing.T) {
		c := temporalmocks.NewClient(t)
		handle := temporalmocks.NewWorkflowUpdateHandle(t)
		handle.On("Get", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				out := args.Get(1).(*mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView)
				*out = mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView{
					TicketID: "ticket-3",
					Status:   mobiledevicesemaphore.MobileDeviceSemaphoreRunCanceled,
				}
			}).
			Return(nil).Once()
		c.On(
			"UpdateWorkflow",
			mock.Anything,
			mock.MatchedBy(func(o client.UpdateWorkflowOptions) bool {
				return o.WorkflowID == mobiledevicesemaphore.WorkflowID("runner-3") &&
					o.UpdateName == mobiledevicesemaphore.CancelRunUpdate &&
					o.UpdateID == "cancel/runner-3/ticket-3"
			}),
		).Return(handle, nil).Once()

		status, err := CancelRun(
			context.Background(),
			c,
			"runner-3",
			mobiledevicesemaphore.MobileDeviceSemaphoreRunCancelRequest{TicketID: "ticket-3"},
		)
		require.NoError(t, err)
		require.Equal(t, mobiledevicesemaphore.MobileDeviceSemaphoreRunCanceled, status.Status)
	})
}
