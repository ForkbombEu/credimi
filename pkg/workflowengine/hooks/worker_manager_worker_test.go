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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/worker/org-1" {
			workerStarts.Add(1)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	var config *workerConfig
	workers := defaultWorkers(nil)
	for i := range workers {
		if workers[i].TaskQueue == workflows.WorkerManagerTaskQueue {
			config = &workers[i]
		}
	}
	require.NotNil(t, config)

	env := (&testsuite.WorkflowTestSuite{}).NewTestWorkflowEnvironment()
	for _, act := range config.Activities {
		env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
	}

	env.ExecuteWorkflow(workflows.NewWorkerManagerWorkflow().Workflow, workflowengine.WorkflowInput{
		Payload: workflows.WorkerManagerWorkflowPayload{
			Namespace:  "org-1",
			RunnerURLs: []string{server.URL},
		},
	})

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 1, output["successful_runners"], output)
	require.EqualValues(t, 1, workerStarts.Load())
}
