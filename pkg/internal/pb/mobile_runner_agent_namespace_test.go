// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestRegisterMobileRunnerAgentNamespaceHooks(t *testing.T) {
	app := newRulesTestApp(t)
	origEnsure := ensureRunnerAgentNamespaceFn
	origStop := stopRunnerAgentWorkersFn
	t.Cleanup(func() {
		ensureRunnerAgentNamespaceFn = origEnsure
		stopRunnerAgentWorkersFn = origStop
	})
	calls := make(chan string, 4)
	ensureRunnerAgentNamespaceFn = func(_ core.App, namespace string) {
		calls <- "ensure " + namespace
	}
	stopRunnerAgentWorkersFn = func(namespace string) {
		calls <- "stop " + namespace
	}
	RegisterMobileRunnerAgentNamespaceHooks(app)

	next := func() string {
		t.Helper()
		select {
		case call := <-calls:
			return call
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for an agent namespace hook call")
			return ""
		}
	}
	none := func() {
		t.Helper()
		select {
		case call := <-calls:
			t.Fatalf("unexpected agent namespace hook call %q", call)
		case <-time.After(50 * time.Millisecond):
		}
	}

	collection, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(collection)
	runner.Set("owner", jsHooksUserAOrg)
	runner.Set("name", "agent-hooks-runner")
	runner.Set("ip", "127.0.0.1")
	runner.Set("agent_enabled", true)
	require.NoError(t, app.Save(runner))
	namespace := "runner-" + runner.Id
	require.Equal(t, "ensure "+namespace, next())

	// update saves a freshly read runner, so its Original is the stored one.
	update := func(field string, value any) {
		t.Helper()
		var err error
		runner, err = app.FindRecordById("mobile_runners", runner.Id)
		require.NoError(t, err)
		runner.Set(field, value)
		require.NoError(t, app.Save(runner))
	}

	// Disabling the runner stops the namespace's workers.
	update("disabled", true)
	require.Equal(t, "stop "+namespace, next())

	// An update of a runner whose agent was already off does nothing.
	update("name", "renamed")
	none()

	update("disabled", false)
	require.Equal(t, "ensure "+namespace, next())

	update("agent_enabled", false)
	require.Equal(t, "stop "+namespace, next())

	require.NoError(t, app.Delete(runner))
	require.Equal(t, "stop "+namespace, next())

	// A runner created with its agent off is left alone.
	off := core.NewRecord(collection)
	off.Set("owner", jsHooksUserAOrg)
	off.Set("name", "agent-off-runner")
	off.Set("ip", "127.0.0.1")
	require.NoError(t, app.Save(off))
	none()
}
