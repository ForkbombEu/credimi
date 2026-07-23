// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"fmt"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestQueryMobileDeviceSemaphoreRunStatusActivityMissingFields(t *testing.T) {
	activity := NewQueryMobileDeviceSemaphoreRunStatusActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: QueryMobileDeviceSemaphoreRunStatusInput{
			DeviceID: "runner-1",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code)
}

func TestQueryMobileDeviceSemaphoreRunStatusActivityDecodeError(t *testing.T) {
	activity := NewQueryMobileDeviceSemaphoreRunStatusActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: "not-a-query-payload",
	})
	require.Error(t, err)
}

func TestQueryMobileDeviceSemaphoreRunStatusActivityName(t *testing.T) {
	require.Equal(
		t,
		"Query mobile device semaphore run status",
		NewQueryMobileDeviceSemaphoreRunStatusActivity().Name(),
	)
}

type stubEncodedValue struct {
	value    mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView
	hasValue bool
	err      error
}

func (s stubEncodedValue) HasValue() bool {
	return s.hasValue
}

func (s stubEncodedValue) Get(valuePtr interface{}) error {
	if s.err != nil {
		return s.err
	}
	target, ok := valuePtr.(*mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView)
	if !ok {
		return fmt.Errorf("unexpected value pointer type %T", valuePtr)
	}
	*target = s.value
	return nil
}

func TestQueryMobileDeviceSemaphoreRunStatusActivityNotFound(t *testing.T) {
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
		"QueryWorkflow",
		mock.Anything,
		mobiledevicesemaphore.WorkflowID("runner-1"),
		"",
		mobiledevicesemaphore.RunStatusQuery,
		"owner-1",
		"ticket-1",
	).Return(nil, &serviceerror.NotFound{}).Once()

	activity := NewQueryMobileDeviceSemaphoreRunStatusActivity()
	result, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: QueryMobileDeviceSemaphoreRunStatusInput{
			DeviceID:       "runner-1",
			OwnerNamespace: "owner-1",
			TicketID:       "ticket-1",
		},
	})
	require.NoError(t, err)
	require.Equal(
		t,
		mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView{
			TicketID: "ticket-1",
			Status:   mobiledevicesemaphore.MobileDeviceSemaphoreRunNotFound,
		},
		result.Output,
	)
}

func TestQueryMobileDeviceSemaphoreRunStatusActivitySuccess(t *testing.T) {
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

	expected := mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView{
		TicketID: "ticket-1",
		Status:   mobiledevicesemaphore.MobileDeviceSemaphoreRunRunning,
	}

	mockClient.On(
		"QueryWorkflow",
		mock.Anything,
		mobiledevicesemaphore.WorkflowID("runner-1"),
		"",
		mobiledevicesemaphore.RunStatusQuery,
		"owner-1",
		"ticket-1",
	).Return(stubEncodedValue{
		value:    expected,
		hasValue: true,
	}, nil).Once()

	activity := NewQueryMobileDeviceSemaphoreRunStatusActivity()
	result, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: QueryMobileDeviceSemaphoreRunStatusInput{
			DeviceID:       "runner-1",
			OwnerNamespace: "owner-1",
			TicketID:       "ticket-1",
		},
	})
	require.NoError(t, err)
	require.Equal(t, expected, result.Output)
}
