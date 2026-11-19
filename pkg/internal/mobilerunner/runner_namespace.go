// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"fmt"
	"sync"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// RunnerNamespaceReadyField is the hidden mobile_runners field where Credimi
// records that it prepared the runner's namespace, so a restarted Credimi
// keeps answering heartbeats with the namespace.
const RunnerNamespaceReadyField = "runner_namespace_ready"

// RunnerNamespace is the Temporal namespace of the runner with record ID
// runnerRecordID: its device work and its action author sessions run there.
func RunnerNamespace(runnerRecordID string) string {
	return canonify.RunnerNamespacePrefix + runnerRecordID
}

// NamespaceEnabled reports whether runner owns a runner namespace: it is not
// disabled.
func NamespaceEnabled(runner *core.Record) bool {
	return !runner.GetBool("disabled")
}

// AgentEnabled reports whether runner serves action author sessions: its
// agent is enabled and the runner is not disabled.
func AgentEnabled(runner *core.Record) bool {
	return runner.GetBool("agent_enabled") && !runner.GetBool("disabled")
}

// servedRunnerNamespaces holds the runners whose namespace this process
// prepared: registered, with Credimi's search attributes and its worker
// running here.
var servedRunnerNamespaces sync.Map // runner record ID -> struct{}

// SetRunnerNamespaceServed records whether this process runs the workers of
// the namespace of the runner with record ID runnerID.
func SetRunnerNamespaceServed(runnerID string, served bool) {
	if served {
		servedRunnerNamespaces.Store(runnerID, struct{}{})
		return
	}
	servedRunnerNamespaces.Delete(runnerID)
}

// RunnerNamespaceServed reports whether this process runs the workers of the
// namespace of the runner with record ID runnerID.
func RunnerNamespaceServed(runnerID string) bool {
	_, ok := servedRunnerNamespaces.Load(runnerID)
	return ok
}

// RunnerNamespaceReady reports whether runner may serve its namespace: the
// runner is not disabled and Credimi prepared the namespace, in this process
// or in an earlier one (RunnerNamespaceReadyField).
func RunnerNamespaceReady(runner *core.Record) bool {
	if !NamespaceEnabled(runner) {
		return false
	}
	return runner.GetBool(RunnerNamespaceReadyField) || RunnerNamespaceServed(runner.Id)
}

// MarkRunnerNamespaceReady sets RunnerNamespaceReadyField on the runner with
// record ID runnerID while it is not disabled. It writes the column alone, so
// record hooks don't run again.
func MarkRunnerNamespaceReady(app core.App, runnerID string) error {
	_, err := app.DB().Update(
		"mobile_runners",
		dbx.Params{RunnerNamespaceReadyField: true},
		dbx.HashExp{"id": runnerID, "disabled": false},
	).Execute()
	if err != nil {
		return fmt.Errorf("mark runner %s namespace ready: %w", runnerID, err)
	}
	return nil
}
