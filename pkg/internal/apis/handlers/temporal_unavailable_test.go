// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

func TestTemporalClientFailureAnswersTemporalUnavailable(t *testing.T) {
	unavailable := func(string) (client.Client, error) {
		return nil, errors.New("dial tcp 127.0.0.1:7233: connection refused")
	}

	cases := []struct {
		name    string
		method  string
		path    string
		handler func() func(*core.RequestEvent) error
		stub    func(t *testing.T)
	}{
		{
			name:    "get workflow run",
			method:  http.MethodGet,
			path:    "/api/my/workflows/wf-1/runs/run-1",
			handler: HandleGetMyWorkflowRun,
			stub:    stubWorkflowTemporalClient(unavailable),
		},
		{
			name:    "get workflow run history",
			method:  http.MethodGet,
			path:    "/api/my/workflows/wf-1/runs/run-1/history",
			handler: HandleGetMyWorkflowRunHistory,
			stub:    stubWorkflowTemporalClient(unavailable),
		},
		{
			name:    "cancel workflow run",
			method:  http.MethodPost,
			path:    "/api/my/workflows/wf-1/runs/run-1/cancel",
			handler: HandleCancelMyWorkflowRun,
			stub:    stubWorkflowTemporalClient(unavailable),
		},
		{
			name:    "export workflow run",
			method:  http.MethodGet,
			path:    "/api/my/workflows/wf-1/runs/run-1/export",
			handler: HandleExportMyWorkflowRun,
			stub:    stubWorkflowTemporalClient(unavailable),
		},
		{
			name:    "compliance deeplink",
			method:  http.MethodGet,
			path:    "/api/compliance/deeplink/wf-1/run-1",
			handler: HandleDeeplink,
			stub: func(t *testing.T) {
				orig := complianceTemporalClient
				t.Cleanup(func() { complianceTemporalClient = orig })
				complianceTemporalClient = unavailable
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, err := tests.NewTestApp(testDataDir)
			require.NoError(t, err)
			defer app.Cleanup()

			authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)

			tc.stub(t)

			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.SetPathValue("workflowId", "wf-1")
			req.SetPathValue("runId", "run-1")
			rec := httptest.NewRecorder()

			err = tc.handler()(&core.RequestEvent{
				App:  app,
				Auth: authRecord,
				Event: router.Event{
					Request:  req,
					Response: rec,
				},
			})
			requireHandlerErrorHandled(t, rec, err)

			require.Equal(t, http.StatusServiceUnavailable, rec.Code)
			body := decodeHandlerErrorResponse(t, rec)
			assert.Equal(t, http.StatusServiceUnavailable, body.Error.Code)
			assert.Equal(t, "temporal", body.Error.Domain)
			assert.Equal(t, "temporal_unavailable", body.Error.Reason)
			assert.Equal(t, "cannot connect to Temporal", body.Error.Message)
		})
	}
}

func stubWorkflowTemporalClient(
	fn func(string) (client.Client, error),
) func(t *testing.T) {
	return func(t *testing.T) {
		orig := workflowTemporalClient
		t.Cleanup(func() { workflowTemporalClient = orig })
		workflowTemporalClient = fn
	}
}
