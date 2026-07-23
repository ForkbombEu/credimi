// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestIsMobileDeviceSemaphoreDisabled(t *testing.T) {
	t.Setenv("MOBILE_DEVICE_SEMAPHORE_DISABLED", "true")
	require.True(t, isMobileDeviceSemaphoreDisabled())

	t.Setenv("MOBILE_DEVICE_SEMAPHORE_DISABLED", "0")
	require.False(t, isMobileDeviceSemaphoreDisabled())
}

func TestIsNotFoundError(t *testing.T) {
	require.True(t, isNotFoundError(&serviceerror.NotFound{}))
	require.False(t, isNotFoundError(errors.New("nope")))
}

func TestReportMobileDeviceSemaphoreDoneActivityNotFound(t *testing.T) {
	t.Setenv("MOBILE_DEVICE_SEMAPHORE_DISABLED", "false")
	temporalclient.ShutdownClients()
	mockClient := &temporalmocks.Client{}
	mockClient.On("Close").Return(nil).Maybe()
	temporalclient.SetClientForTests(
		workflowengine.MobileDeviceSemaphoreDefaultNamespace,
		mockClient,
	)
	t.Cleanup(func() {
		temporalclient.ClearTestClients()
		temporalclient.ShutdownClients()
		mockClient.AssertExpectations(t)
	})

	mockClient.On(
		"UpdateWorkflow",
		mock.Anything,
		mock.MatchedBy(func(options tclient.UpdateWorkflowOptions) bool {
			return options.WorkflowID == mobiledevicesemaphore.WorkflowID("runner-1") &&
				options.UpdateName == mobiledevicesemaphore.RunDoneUpdate
		}),
	).Return(nil, &serviceerror.NotFound{}).Once()

	activity := NewReportMobileDeviceSemaphoreDoneActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: ReportMobileDeviceSemaphoreDoneInput{
			OwnerNamespace: "owner-1",
			LeaderDeviceID: "runner-1",
			TicketID:       "ticket-1",
		},
	})
	require.NoError(t, err)
}

func TestReportMobileDeviceSemaphoreDoneActivitySuccess(t *testing.T) {
	t.Setenv("MOBILE_DEVICE_SEMAPHORE_DISABLED", "false")
	temporalclient.ShutdownClients()
	mockClient := &temporalmocks.Client{}
	mockClient.On("Close").Return(nil).Maybe()
	temporalclient.SetClientForTests(
		workflowengine.MobileDeviceSemaphoreDefaultNamespace,
		mockClient,
	)
	t.Cleanup(func() {
		temporalclient.ClearTestClients()
		temporalclient.ShutdownClients()
		mockClient.AssertExpectations(t)
	})

	updateHandle := temporalmocks.NewWorkflowUpdateHandle(t)
	updateHandle.On("Get", mock.Anything, mock.Anything).Return(nil).Once()

	mockClient.On(
		"UpdateWorkflow",
		mock.Anything,
		mock.MatchedBy(func(options tclient.UpdateWorkflowOptions) bool {
			req, ok := options.Args[0].(mobiledevicesemaphore.MobileDeviceSemaphoreRunDoneRequest)
			return options.WorkflowID == mobiledevicesemaphore.WorkflowID("runner-1") &&
				options.UpdateName == mobiledevicesemaphore.RunDoneUpdate &&
				options.UpdateID == "run-done/ticket-1" &&
				ok &&
				req.WorkflowResult == "success"
		}),
	).Return(updateHandle, nil).Once()

	activity := NewReportMobileDeviceSemaphoreDoneActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: ReportMobileDeviceSemaphoreDoneInput{
			OwnerNamespace: "owner-1",
			LeaderDeviceID: "runner-1",
			TicketID:       "ticket-1",
			WorkflowResult: "success",
		},
	})
	require.NoError(t, err)
}
