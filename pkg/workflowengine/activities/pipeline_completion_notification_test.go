// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestSendPipelineCompletionNotificationActivityName(t *testing.T) {
	require.Equal(
		t,
		SendPipelineCompletionNotificationActivityName,
		NewSendPipelineCompletionNotificationActivity(nil).Name(),
	)
}

func TestSendPipelineCompletionNotificationActivity(t *testing.T) {
	app := newPipelineResultsTestApp(t)
	createTestPipelineResult(t, app, "wf-notify", "run-notify")

	tests := []struct {
		name        string
		payload     SendPipelineCompletionNotificationInput
		errContains []string
		wantSent    int
	}{
		{
			name:        "missing fields",
			payload:     SendPipelineCompletionNotificationInput{WorkflowID: "wf-notify"},
			errContains: []string{errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code},
		},
		{
			name: "missing pipeline result",
			payload: SendPipelineCompletionNotificationInput{
				WorkflowID: "wf-unknown",
				RunID:      "run-unknown",
				Result:     "success",
			},
			errContains: []string{
				errorcodes.Codes[errorcodes.RecordNotFound].Code,
				"pipeline result not found",
			},
		},
		{
			name: "no subscriptions",
			payload: SendPipelineCompletionNotificationInput{
				WorkflowID:   "wf-notify",
				RunID:        "run-notify",
				Result:       "failure",
				ErrorMessage: "step failed",
			},
			wantSent: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			act := NewSendPipelineCompletionNotificationActivity(app)
			env := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()
			env.RegisterActivityWithOptions(
				act.Execute,
				activity.RegisterOptions{Name: act.Name()},
			)

			encoded, err := env.ExecuteActivity(
				act.Name(),
				workflowengine.ActivityInput{Payload: tc.payload},
			)
			if len(tc.errContains) > 0 {
				require.Error(t, err)
				for _, want := range tc.errContains {
					require.ErrorContains(t, err, want)
				}
				return
			}
			require.NoError(t, err)
			var result struct {
				Output SendPipelineCompletionNotificationOutput `json:"output"`
			}
			require.NoError(t, encoded.Get(&result))
			require.Equal(t, tc.wantSent, result.Output.Sent)
		})
	}
}
