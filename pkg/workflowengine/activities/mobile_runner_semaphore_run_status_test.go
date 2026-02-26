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
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobilerunnersemaphore"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestQueryMobileRunnerSemaphoreRunStatusActivityMissingFields(t *testing.T) {
	activity := NewQueryMobileRunnerSemaphoreRunStatusActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: QueryMobileRunnerSemaphoreRunStatusInput{
			RunnerID: "runner-1",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code)
}

func TestQueryMobileRunnerSemaphoreRunStatusActivityDecodeError(t *testing.T) {
	activity := NewQueryMobileRunnerSemaphoreRunStatusActivity()
	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: "not-a-query-payload",
	})
	require.Error(t, err)
}

func TestQueryMobileRunnerSemaphoreRunStatusActivityName(t *testing.T) {
	require.Equal(
		t,
		"Query mobile runner semaphore run status",
		NewQueryMobileRunnerSemaphoreRunStatusActivity().Name(),
	)
}

type stubEncodedValue struct {
	value    mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView
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
	target, ok := valuePtr.(*mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView)
	if !ok {
		return fmt.Errorf("unexpected value pointer type %T", valuePtr)
	}
	*target = s.value
	return nil
}

func TestQueryMobileRunnerSemaphoreRunStatusActivityNotFound(t *testing.T) {
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
		"QueryWorkflow",
		mock.Anything,
		mobilerunnersemaphore.WorkflowID("runner-1"),
		"",
		mobilerunnersemaphore.RunStatusQuery,
		"owner-1",
		"ticket-1",
	).Return(nil, &serviceerror.NotFound{}).Once()

	activity := NewQueryMobileRunnerSemaphoreRunStatusActivity()
	result, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: QueryMobileRunnerSemaphoreRunStatusInput{
			RunnerID:       "runner-1",
			OwnerNamespace: "owner-1",
			TicketID:       "ticket-1",
		},
	})
	require.NoError(t, err)
	require.Equal(
		t,
		mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView{
			TicketID: "ticket-1",
			Status:   mobilerunnersemaphore.MobileRunnerSemaphoreRunNotFound,
		},
		result.Output,
	)
}

func TestQueryMobileRunnerSemaphoreRunStatusActivitySuccess(t *testing.T) {
	temporalclient.ShutdownClients()
	mockClient := &temporalmocks.Client{}
	mockClient.On("Close").Return(nil).Maybe()
	temporalclient.SetClientForTests(workflowengine.MobileRunnerSemaphoreDefaultNamespace, mockClient)
	t.Cleanup(func() {
		temporalclient.ClearTestClients()
		temporalclient.ShutdownClients()
		mockClient.AssertExpectations(t)
	})

	expected := mobilerunnersemaphore.MobileRunnerSemaphoreRunStatusView{
		TicketID: "ticket-1",
		Status:   mobilerunnersemaphore.MobileRunnerSemaphoreRunRunning,
	}

	mockClient.On(
		"QueryWorkflow",
		mock.Anything,
		mobilerunnersemaphore.WorkflowID("runner-1"),
		"",
		mobilerunnersemaphore.RunStatusQuery,
		"owner-1",
		"ticket-1",
	).Return(stubEncodedValue{
		value:    expected,
		hasValue: true,
	}, nil).Once()

	activity := NewQueryMobileRunnerSemaphoreRunStatusActivity()
	result, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: QueryMobileRunnerSemaphoreRunStatusInput{
			RunnerID:       "runner-1",
			OwnerNamespace: "owner-1",
			TicketID:       "ticket-1",
		},
	})
	require.NoError(t, err)
	require.Equal(t, expected, result.Output)
}
