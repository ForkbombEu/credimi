// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
)

type fakeWorkflowCanceler struct {
	cancelErr      error
	lastWorkflowID string
	lastRunID      string
}

func (f *fakeWorkflowCanceler) CancelWorkflow(
	_ context.Context,
	workflowID string,
	runID string,
) error {
	f.lastWorkflowID = workflowID
	f.lastRunID = runID
	return f.cancelErr
}

func TestCancelWorkflowActivityMissingWorkflowID(t *testing.T) {
	act := NewCancelWorkflowActivity()
	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CancelWorkflowActivityInput{
			WorkflowNamespace: "tenant-1",
		},
	})
	require.ErrorContains(t, err, "workflow_id is required")
}

func TestCancelWorkflowActivityMissingNamespace(t *testing.T) {
	act := NewCancelWorkflowActivity()
	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CancelWorkflowActivityInput{
			WorkflowID: "wf-1",
		},
	})
	require.ErrorContains(t, err, "workflow_namespace is required")
}

func TestCancelWorkflowActivityNotFound(t *testing.T) {
	act := NewCancelWorkflowActivity()
	fakeClient := &fakeWorkflowCanceler{cancelErr: &serviceerror.NotFound{Message: "missing"}}
	act.temporalClientFactory = func(namespace string) (temporalWorkflowCanceler, error) {
		require.Equal(t, "tenant-1", namespace)
		return fakeClient, nil
	}

	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CancelWorkflowActivityInput{
			WorkflowID:        "wf-1",
			RunID:             "run-1",
			WorkflowNamespace: "tenant-1",
		},
	})
	require.NoError(t, err)
	output := result.Output.(CancelWorkflowActivityOutput)
	require.False(t, output.Canceled)
	require.Equal(t, "NOT_FOUND", output.Status)
	require.Equal(t, "wf-1", fakeClient.lastWorkflowID)
	require.Equal(t, "run-1", fakeClient.lastRunID)
}

func TestCancelWorkflowActivitySuccess(t *testing.T) {
	act := NewCancelWorkflowActivity()
	fakeClient := &fakeWorkflowCanceler{}
	act.temporalClientFactory = func(namespace string) (temporalWorkflowCanceler, error) {
		require.Equal(t, "tenant-2", namespace)
		return fakeClient, nil
	}

	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CancelWorkflowActivityInput{
			WorkflowID:        "wf-2",
			RunID:             "run-2",
			WorkflowNamespace: "tenant-2",
		},
	})
	require.NoError(t, err)
	output := result.Output.(CancelWorkflowActivityOutput)
	require.True(t, output.Canceled)
	require.Equal(t, "CANCELED", output.Status)
}

func TestCancelWorkflowActivityClientError(t *testing.T) {
	act := NewCancelWorkflowActivity()
	act.temporalClientFactory = func(namespace string) (temporalWorkflowCanceler, error) {
		return nil, errors.New("boom")
	}

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CancelWorkflowActivityInput{
			WorkflowID:        "wf-3",
			WorkflowNamespace: "tenant-3",
		},
	})
	require.ErrorContains(t, err, "boom")
}
