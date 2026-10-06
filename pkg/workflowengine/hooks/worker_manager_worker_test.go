// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package hooks

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

// The worker-manager workflow must find every activity it schedules on the
// worker that polls its task queue; Temporal fails the run with
// ActivityNotRegisteredError otherwise.
func TestWorkerManagerWorkerRegistersWorkflowActivities(t *testing.T) {
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "test-admin-key")

	var workerStarts atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/mobile-runner/list-urls":
			_, _ = w.Write([]byte(`{"runners":["` + server.URL + `"]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/worker/org-1":
			workerStarts.Add(1)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv(workflowengine.InternalAppURLConfigKeyEnv, server.URL)

	var config *workerConfig
	for i := range DefaultWorkers {
		if DefaultWorkers[i].TaskQueue == workflows.WorkerManagerTaskQueue {
			config = &DefaultWorkers[i]
		}
	}
	require.NotNil(t, config)

	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	for _, act := range config.Activities {
		env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
	}

	env.ExecuteWorkflow(workflows.NewWorkerManagerWorkflow().Workflow, workflowengine.WorkflowInput{
		Payload: workflows.WorkerManagerWorkflowPayload{Namespace: "org-1"},
		Config:  map[string]any{"app_url": server.URL},
	})

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 1, output["successful_runners"], output)
	require.EqualValues(t, 1, workerStarts.Load())
}
