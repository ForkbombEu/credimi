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

type fakeWorkflowSignaler struct {
	signalErr       error
	lastWorkflowID  string
	lastRunID       string
	lastSignalName  string
	lastSignalValue any
}

func (f *fakeWorkflowSignaler) SignalWorkflow(
	_ context.Context,
	workflowID string,
	runID string,
	signalName string,
	arg interface{},
) error {
	f.lastWorkflowID = workflowID
	f.lastRunID = runID
	f.lastSignalName = signalName
	f.lastSignalValue = arg
	return f.signalErr
}

func TestSignalWorkflowActivityValidation(t *testing.T) {
	tests := []struct {
		name    string
		payload any
		message string
	}{
		{
			name:    "invalid payload",
			payload: make(chan int),
			message: "Missing or invalid value in payload",
		},
		{
			name: "missing workflow ID",
			payload: SignalWorkflowActivityInput{
				WorkflowNamespace: "tenant-1",
				SignalName:        "start",
			},
			message: "workflow_id is required",
		},
		{
			name: "missing namespace",
			payload: SignalWorkflowActivityInput{
				WorkflowID: "wf-1",
				SignalName: "start",
			},
			message: "workflow_namespace is required",
		},
		{
			name: "missing signal name",
			payload: SignalWorkflowActivityInput{
				WorkflowID:        "wf-1",
				WorkflowNamespace: "tenant-1",
			},
			message: "signal_name is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			activity := NewSignalWorkflowActivity()
			_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
				Payload: test.payload,
			})
			require.ErrorContains(t, err, test.message)
		})
	}
}

func TestSignalWorkflowActivityClientError(t *testing.T) {
	activity := NewSignalWorkflowActivity()
	require.Equal(t, "Signal workflow", activity.Name())
	activity.temporalClientFactory = func(string) (temporalWorkflowSignaler, error) {
		return nil, errors.New("client unavailable")
	}

	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: SignalWorkflowActivityInput{
			WorkflowID:        "wf-1",
			WorkflowNamespace: "tenant-1",
			SignalName:        "start",
		},
	})
	require.ErrorContains(t, err, "client unavailable")
}

func TestSignalWorkflowActivityResult(t *testing.T) {
	tests := []struct {
		name           string
		signalErr      error
		expectedOutput SignalWorkflowActivityOutput
		expectedError  string
	}{
		{
			name:      "workflow not found",
			signalErr: &serviceerror.NotFound{Message: "missing"},
			expectedOutput: SignalWorkflowActivityOutput{
				Status: "NOT_FOUND",
			},
		},
		{
			name:          "signal failure",
			signalErr:     errors.New("signal failed"),
			expectedError: "signal failed",
		},
		{
			name: "success",
			expectedOutput: SignalWorkflowActivityOutput{
				Signaled: true,
				Status:   "SIGNALED",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			activity := NewSignalWorkflowActivity()
			fakeClient := &fakeWorkflowSignaler{signalErr: test.signalErr}
			activity.temporalClientFactory = func(namespace string) (temporalWorkflowSignaler, error) {
				require.Equal(t, "tenant-1", namespace)
				return fakeClient, nil
			}

			result, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
				Payload: SignalWorkflowActivityInput{
					WorkflowID:        " wf-1 ",
					RunID:             " run-1 ",
					WorkflowNamespace: " tenant-1 ",
					SignalName:        " start ",
					Payload:           map[string]any{"enabled": true},
				},
			})
			if test.expectedError != "" {
				require.ErrorContains(t, err, test.expectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expectedOutput, result.Output)
			require.Equal(t, "wf-1", fakeClient.lastWorkflowID)
			require.Equal(t, "run-1", fakeClient.lastRunID)
			require.Equal(t, "start", fakeClient.lastSignalName)
			require.Equal(t, map[string]any{"enabled": true}, fakeClient.lastSignalValue)
		})
	}
}
