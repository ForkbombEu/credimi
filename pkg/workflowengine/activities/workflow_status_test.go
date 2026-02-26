// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build unit

package activities

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestCheckWorkflowClosedActivityMissingFields(t *testing.T) {
	activity := NewCheckWorkflowClosedActivity()
	require.Equal(t, "Check workflow closed", activity.Name())

	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: "not-a-workflow-status-payload",
	})
	require.Error(t, err)

	_, err = activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CheckWorkflowClosedActivityInput{
			WorkflowNamespace: "ns",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code)

	_, err = activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CheckWorkflowClosedActivityInput{
			WorkflowID: "wf-1",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "workflow_namespace is required")
}

func TestCheckWorkflowClosedActivityNotFound(t *testing.T) {
	temporalclient.ShutdownClients()
	mockClient := &temporalmocks.Client{}
	mockClient.On("Close").Return(nil).Maybe()
	temporalclient.SetClientForTests("ns-1", mockClient)
	t.Cleanup(func() {
		temporalclient.ClearTestClients()
		temporalclient.ShutdownClients()
		mockClient.AssertExpectations(t)
	})

	mockClient.On(
		"DescribeWorkflowExecution",
		mock.Anything,
		"wf-1",
		"",
	).Return(nil, &serviceerror.NotFound{}).Once()

	activity := NewCheckWorkflowClosedActivity()
	result, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CheckWorkflowClosedActivityInput{
			WorkflowID:        "wf-1",
			WorkflowNamespace: "ns-1",
		},
	})
	require.NoError(t, err)
	require.Equal(
		t,
		CheckWorkflowClosedActivityOutput{
			Closed: true,
			Status: "NOT_FOUND",
		},
		result.Output,
	)
}

func TestCheckWorkflowClosedActivityRunning(t *testing.T) {
	temporalclient.ShutdownClients()
	mockClient := &temporalmocks.Client{}
	mockClient.On("Close").Return(nil).Maybe()
	temporalclient.SetClientForTests("ns-1", mockClient)
	t.Cleanup(func() {
		temporalclient.ClearTestClients()
		temporalclient.ShutdownClients()
		mockClient.AssertExpectations(t)
	})

	response := &workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
			Status: enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		},
	}

	mockClient.On(
		"DescribeWorkflowExecution",
		mock.Anything,
		"wf-1",
		"",
	).Return(response, nil).Once()

	activity := NewCheckWorkflowClosedActivity()
	result, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CheckWorkflowClosedActivityInput{
			WorkflowID:        "wf-1",
			WorkflowNamespace: "ns-1",
		},
	})
	require.NoError(t, err)
	require.Equal(
		t,
		CheckWorkflowClosedActivityOutput{
			Closed: false,
			Status: enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING.String(),
		},
		result.Output,
	)
}
