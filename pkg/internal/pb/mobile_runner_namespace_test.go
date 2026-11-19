// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestRegisterMobileRunnerNamespaceHooks(t *testing.T) {
	app := newRulesTestApp(t)
	origEnsure := ensureRunnerNamespaceFn
	origStop := stopRunnerNamespaceWorkersFn
	origDelete := deleteRunnerNexusEndpointFn
	t.Cleanup(func() {
		ensureRunnerNamespaceFn = origEnsure
		stopRunnerNamespaceWorkersFn = origStop
		deleteRunnerNexusEndpointFn = origDelete
	})
	calls := make(chan string, 4)
	ensureRunnerNamespaceFn = func(_ core.App, runnerID string) {
		calls <- "ensure " + runnerID
	}
	stopRunnerNamespaceWorkersFn = func(_ core.App, runnerID string) {
		calls <- "stop " + runnerID
	}
	deleteRunnerNexusEndpointFn = func(runnerID string) error {
		calls <- "delete endpoint " + runnerID
		return nil
	}
	RegisterMobileRunnerNamespaceHooks(app)

	next := func() string {
		t.Helper()
		select {
		case call := <-calls:
			return call
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for a runner namespace hook call")
			return ""
		}
	}
	none := func() {
		t.Helper()
		select {
		case call := <-calls:
			t.Fatalf("unexpected runner namespace hook call %q", call)
		case <-time.After(50 * time.Millisecond):
		}
	}

	collection, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(collection)
	runner.Set("owner", jsHooksUserAOrg)
	runner.Set("name", "namespace-hooks-runner")
	runner.Set("ip", "127.0.0.1")
	require.NoError(t, app.Save(runner))
	id := runner.Id
	require.Equal(t, "ensure "+id, next())

	// update saves a freshly read runner, so its Original is the stored one.
	update := func(field string, value any) {
		t.Helper()
		var err error
		runner, err = app.FindRecordById("mobile_runners", runner.Id)
		require.NoError(t, err)
		runner.Set(field, value)
		require.NoError(t, app.Save(runner))
	}
	// markReady marks the namespace ready as EnsureRunnerNamespace does.
	markReady := func() {
		t.Helper()
		require.NoError(t, mobilerunner.MarkRunnerNamespaceReady(app, id))
		stored, err := app.FindRecordById("mobile_runners", id)
		require.NoError(t, err)
		require.True(t, stored.GetBool(mobilerunner.RunnerNamespaceReadyField))
	}
	readyCleared := func() {
		t.Helper()
		stored, err := app.FindRecordById("mobile_runners", id)
		require.NoError(t, err)
		require.False(t, stored.GetBool(mobilerunner.RunnerNamespaceReadyField))
	}

	// Disabling the runner clears its readiness and stops the namespace's
	// workers.
	markReady()
	update("disabled", true)
	readyCleared()
	require.Equal(t, "stop "+id, next())

	// An update of a runner that was already disabled does nothing.
	update("name", "renamed")
	none()

	update("disabled", false)
	require.Equal(t, "ensure "+id, next())

	// The agent does not own the namespace: turning it off keeps the
	// namespace ready and prepared.
	markReady()
	update("agent_enabled", false)
	stored, err := app.FindRecordById("mobile_runners", id)
	require.NoError(t, err)
	require.True(t, stored.GetBool(mobilerunner.RunnerNamespaceReadyField))
	require.Equal(t, "ensure "+id, next())

	require.NoError(t, app.Delete(runner))
	require.Equal(t, "stop "+id, next())
	require.Equal(t, "delete endpoint "+id, next())

	// A runner created disabled is left alone.
	off := core.NewRecord(collection)
	off.Set("owner", jsHooksUserAOrg)
	off.Set("name", "disabled-runner")
	off.Set("ip", "127.0.0.1")
	off.Set("disabled", true)
	require.NoError(t, app.Save(off))
	none()
}
