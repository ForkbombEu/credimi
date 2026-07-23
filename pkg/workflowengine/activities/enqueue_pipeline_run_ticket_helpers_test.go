// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

type helperUpdateHandle struct {
	status mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView
	err    error
}

func (h *helperUpdateHandle) Get(_ context.Context, valuePtr interface{}) error {
	if h.err != nil {
		return h.err
	}
	if out, ok := valuePtr.(*mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView); ok {
		*out = h.status
	}
	return nil
}

func (h *helperUpdateHandle) WorkflowID() string { return "" }

func (h *helperUpdateHandle) RunID() string { return "" }

func (h *helperUpdateHandle) UpdateID() string { return "" }

type fakeUpdater struct {
	execErr   error
	updateErr error
	handle    client.WorkflowUpdateHandle
}

func (f *fakeUpdater) ExecuteWorkflow(
	_ context.Context,
	_ client.StartWorkflowOptions,
	_ interface{},
	_ ...interface{},
) (client.WorkflowRun, error) {
	return nil, f.execErr
}

func (f *fakeUpdater) UpdateWorkflow(
	_ context.Context,
	_ client.UpdateWorkflowOptions,
) (client.WorkflowUpdateHandle, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return f.handle, nil
}

func TestNormalizeDeviceIDs(t *testing.T) {
	out := normalizeDeviceIDs([]string{" /tenant/runner-1 ", "", "tenant/runner-2"})
	require.Equal(t, []string{"tenant/runner-1", "tenant/runner-2"}, out)
	require.Nil(t, normalizeDeviceIDs(nil))
}

func TestRunQueueUpdateID(t *testing.T) {
	require.Equal(
		t,
		"enqueue/tenant/runner/ticket",
		runQueueUpdateID("enqueue", "/tenant/runner", "ticket"),
	)
}

func TestIsQueueLimitExceeded(t *testing.T) {
	err := temporal.NewApplicationError("limit", mobiledevicesemaphore.ErrQueueLimitExceeded)
	require.True(t, isQueueLimitExceeded(err))
	require.False(t, isQueueLimitExceeded(nil))
}

func TestEnsureRunQueueSemaphoreWorkflow(t *testing.T) {
	updater := &fakeUpdater{execErr: &serviceerror.WorkflowExecutionAlreadyStarted{}}
	require.NoError(t, ensureRunQueueSemaphoreWorkflow(context.Background(), updater, "runner-1"))
}

func TestCancelRunTicketNotFound(t *testing.T) {
	updater := &fakeUpdater{updateErr: &serviceerror.NotFound{}}
	_, err := cancelRunTicket(
		context.Background(),
		updater,
		"runner-1",
		mobiledevicesemaphore.MobileDeviceSemaphoreRunCancelRequest{TicketID: "ticket-1"},
	)
	require.ErrorIs(t, err, errRunTicketNotFound)
}

func TestCancelRunTicketSuccess(t *testing.T) {
	handle := &helperUpdateHandle{
		status: mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView{
			TicketID: "ticket-1",
			Status:   mobiledevicesemaphore.MobileDeviceSemaphoreRunCanceled,
		},
	}
	updater := &fakeUpdater{handle: handle}
	status, err := cancelRunTicket(
		context.Background(),
		updater,
		"runner-1",
		mobiledevicesemaphore.MobileDeviceSemaphoreRunCancelRequest{TicketID: "ticket-1"},
	)
	require.NoError(t, err)
	require.Equal(t, "ticket-1", status.TicketID)
	require.Equal(t, mobiledevicesemaphore.MobileDeviceSemaphoreRunCanceled, status.Status)
}
