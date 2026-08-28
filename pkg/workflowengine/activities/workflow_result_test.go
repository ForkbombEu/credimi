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
	temporalmocks "go.temporal.io/sdk/mocks"
)

func TestGetWorkflowResultActivityDoesNotCloseSharedClient(t *testing.T) {
	temporalclient.ShutdownClients()
	mockClient := &temporalmocks.Client{}
	temporalclient.SetClientForTests("ns-1", mockClient)
	t.Cleanup(func() {
		temporalclient.ClearTestClients()
		temporalclient.ShutdownClients()
		mockClient.AssertExpectations(t)
	})

	mockRun := &temporalmocks.WorkflowRun{}
	mockClient.On("GetWorkflow", mock.Anything, "wf-1", "run-1").Return(mockRun).Once()
	mockRun.
		On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
		Run(func(args mock.Arguments) {
			resultPtr := args.Get(1).(*workflowengine.WorkflowResult)
			*resultPtr = workflowengine.WorkflowResult{
				WorkflowID:    "wf-1",
				WorkflowRunID: "run-1",
				Output:        map[string]any{"status": "ok"},
			}
		}).
		Return(nil).
		Once()

	activity := NewGetWorkflowResultActivity()
	result, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: GetWorkflowResultActivityInput{
			WorkflowID:        "wf-1",
			RunID:             "run-1",
			WorkflowNamespace: "ns-1",
		},
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"status": "ok"}, result.Output.(workflowengine.WorkflowResult).Output)
	mockClient.AssertNotCalled(t, "Close")
}

func TestGetWorkflowResultActivityMissingFields(t *testing.T) {
	activity := NewGetWorkflowResultActivity()

	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: "not-a-workflow-result-payload",
	})
	require.Error(t, err)

	_, err = activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: GetWorkflowResultActivityInput{
			WorkflowNamespace: "ns-1",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code)

	_, err = activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: GetWorkflowResultActivityInput{
			WorkflowID: "wf-1",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "workflow_namespace is required")
}

func TestGetWorkflowResultActivityReturnsStructuredFailedWorkflowDetails(t *testing.T) {
	temporalclient.ShutdownClients()
	mockClient := &temporalmocks.Client{}
	temporalclient.SetClientForTests("ns-1", mockClient)
	t.Cleanup(func() {
		temporalclient.ClearTestClients()
		temporalclient.ShutdownClients()
		mockClient.AssertExpectations(t)
	})

	mockRun := &temporalmocks.WorkflowRun{}
	mockClient.On("GetWorkflow", mock.Anything, "wf-1", "run-1").Return(mockRun).Once()
	mockRun.On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
		Return(workflowengine.NewWorkflowError(
			workflowengine.NewAppError(workflowengine.WorkflowError{
				Code:    "CRE229",
				Summary: "Pipeline failed",
				Details: map[string]any{
					"output": map[string]any{
						"successful-step": map[string]any{"outputs": "ok"},
					},
					"errors": []workflowengine.WorkflowError{{
						Code:    "CRE228",
						Summary: "test step failed",
						Details: map[string]any{"step_id": "failed-step"},
					}},
				},
			}),
			&workflowengine.WorkflowRunMetadata{},
		)).
		Once()

	activity := NewGetWorkflowResultActivity()
	result, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: GetWorkflowResultActivityInput{
			WorkflowID:           "wf-1",
			RunID:                "run-1",
			WorkflowNamespace:    "ns-1",
			ReturnFailureDetails: true,
		},
	})

	require.NoError(t, err)
	workflowResult, ok := result.Output.(workflowengine.WorkflowResult)
	require.True(t, ok)
	require.Equal(t, "ok", workflowResult.Output.(map[string]any)["successful-step"].(map[string]any)["outputs"])
	require.NotNil(t, workflowResult.Errors)
}
