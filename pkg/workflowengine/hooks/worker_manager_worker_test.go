// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package hooks

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

// The worker-manager workflow must find every activity it schedules on the
// worker that polls its task queue; Temporal fails the run with
// ActivityNotRegisteredError otherwise. The runner is called with its own
// credential, never with the internal admin key.
func TestWorkerManagerWorkerRegistersWorkflowActivities(t *testing.T) {
	t.Setenv("CREDIMI_INTERNAL_ADMIN_KEY", "test-admin-key")
	t.Setenv(mobilerunner.CredentialSecretEnvVar, "test-runner-secret")

	var workerStarts atomic.Int32
	var gotKey atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/worker/org-1" {
			workerStarts.Add(1)
			gotKey.Store(r.Header.Get("Credimi-Api-Key"))
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	organizations, err := workerManagerAllOrganizationRecords(app)
	require.NoError(t, err)
	require.NotEmpty(t, organizations)
	collection, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(collection)
	runner.Set("owner", organizations[0].Id)
	runner.Set("name", "manager-runner")
	runner.Set("canonified_name", "manager-runner")
	runner.Set("type", "android_emulator")
	runner.Set("ip", server.URL)
	runner.Set("admin_managed", true)
	require.NoError(t, app.Save(runner))
	runnerID, err := mobilerunner.RunnerIdentifier(app, runner)
	require.NoError(t, err)
	wantKey, err := mobilerunner.Credential(runner)
	require.NoError(t, err)

	var config *workerConfig
	workers := defaultWorkers(app)
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
			Namespace: "org-1",
			RunnerIDs: []string{runnerID},
		},
	})

	var result workflowengine.WorkflowResult
	require.NoError(t, env.GetWorkflowResult(&result))
	output, ok := result.Output.(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 1, output["successful_runners"], output)
	require.EqualValues(t, 1, workerStarts.Load())
	require.Equal(t, wantKey, gotKey.Load())
	require.NotEqual(t, "test-admin-key", gotKey.Load())
}
