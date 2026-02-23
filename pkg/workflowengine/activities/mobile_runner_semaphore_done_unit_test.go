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
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	tclient "go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestIsMobileRunnerSemaphoreDisabled(t *testing.T) {
	t.Setenv("MOBILE_RUNNER_SEMAPHORE_DISABLED", "true")
	require.True(t, isMobileRunnerSemaphoreDisabled())

	t.Setenv("MOBILE_RUNNER_SEMAPHORE_DISABLED", "0")
	require.False(t, isMobileRunnerSemaphoreDisabled())
}

func TestIsNotFoundError(t *testing.T) {
	require.True(t, isNotFoundError(&serviceerror.NotFound{}))
	require.False(t, isNotFoundError(errors.New("nope")))
}

func TestReportMobileRunnerSemaphoreDoneActivityNotFound(t *testing.T) {
	t.Setenv("MOBILE_RUNNER_SEMAPHORE_DISABLED", "false")
	temporalclient.ShutdownClients()
	mockClient := &temporalmocks.Client{}
	mockClient.On("Close").Return(nil).Maybe()
	temporalclient.SetClientForTests(workflowengine.MobileRunnerSemaphoreDefaultNamespace, mockClient)
	t.Cleanup(func() {
		temporalclient.ClearTestClients()
		temporalclient.ShutdownClients()
		mockClient.AssertExpectations(t)
	})

	mockClient.On(
		"UpdateWorkflow",
		mock.Anything,
		mock.MatchedBy(func(options tclient.UpdateWorkflowOptions) bool {
			return options.WorkflowID == mobilerunnersemaphore.WorkflowID("runner-1") &&
				options.UpdateName == mobilerunnersemaphore.RunDoneUpdate
		}),
	).Return(nil, &serviceerror.NotFound{}).Once()

	activity := NewReportMobileRunnerSemaphoreDoneActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: ReportMobileRunnerSemaphoreDoneInput{
			OwnerNamespace: "owner-1",
			LeaderRunnerID: "runner-1",
			TicketID:       "ticket-1",
		},
	})
	require.NoError(t, err)
}

func TestReportMobileRunnerSemaphoreDoneActivitySuccess(t *testing.T) {
	t.Setenv("MOBILE_RUNNER_SEMAPHORE_DISABLED", "false")
	temporalclient.ShutdownClients()
	mockClient := &temporalmocks.Client{}
	mockClient.On("Close").Return(nil).Maybe()
	temporalclient.SetClientForTests(workflowengine.MobileRunnerSemaphoreDefaultNamespace, mockClient)
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
			return options.WorkflowID == mobilerunnersemaphore.WorkflowID("runner-1") &&
				options.UpdateName == mobilerunnersemaphore.RunDoneUpdate &&
				options.UpdateID == "run-done/ticket-1"
		}),
	).Return(updateHandle, nil).Once()

	activity := NewReportMobileRunnerSemaphoreDoneActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: ReportMobileRunnerSemaphoreDoneInput{
			OwnerNamespace: "owner-1",
			LeaderRunnerID: "runner-1",
			TicketID:       "ticket-1",
		},
	})
	require.NoError(t, err)
}
