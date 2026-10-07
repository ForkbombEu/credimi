// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
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

func TestSendPipelineCompletionNotificationActivityLookupFailure(t *testing.T) {
	app := newPipelineResultsTestApp(t)
	createTestPipelineResult(t, app, "wf-notify", "run-notify")
	_, err := app.DB().NewQuery("DROP TABLE pipelines").Execute()
	require.NoError(t, err)

	_, err = executeActivity(
		t,
		NewSendPipelineCompletionNotificationActivity(app),
		SendPipelineCompletionNotificationInput{
			WorkflowID: "wf-notify",
			RunID:      "run-notify",
			Result:     "success",
		},
	)
	requireActivityError(t, err, errorcodes.DatabaseOperationFailed, false)
	require.ErrorContains(t, err, "lookup pipeline")
}

func TestPipelineCompletionNotificationError(t *testing.T) {
	cases := []struct {
		name         string
		err          error
		wantCode     string
		nonRetryable bool
	}{
		{
			name:         "missing pipeline result",
			err:          fmt.Errorf("%w: workflow_id wf run_id run", pipelineresults.ErrNotFound),
			wantCode:     errorcodes.RecordNotFound,
			nonRetryable: true,
		},
		{
			name:         "missing record",
			err:          fmt.Errorf("find pipeline: %w", sql.ErrNoRows),
			wantCode:     errorcodes.RecordNotFound,
			nonRetryable: true,
		},
		{
			name:     "send failure",
			err:      fmt.Errorf("%w: push service down", errPipelineNotificationSend),
			wantCode: errorcodes.ExecuteHTTPRequestFailed,
		},
		{
			name:     "lookup failure",
			err:      errors.New("lookup organization: database is locked"),
			wantCode: errorcodes.DatabaseOperationFailed,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			act := NewSendPipelineCompletionNotificationActivity(nil)
			err := pipelineCompletionNotificationError(&act.BaseActivity, tc.err)
			requireActivityError(t, err, tc.wantCode, tc.nonRetryable)
		})
	}
}
