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

// AgentNamespaceReadyField is the hidden mobile_runners field where Credimi
// records that it prepared the runner's agent namespace, so a restarted
// Credimi keeps answering heartbeats with the namespace.
const AgentNamespaceReadyField = "agent_namespace_ready"

// AgentNamespace is the Temporal namespace where the action author sessions
// of the runner with record ID runnerRecordID run.
func AgentNamespace(runnerRecordID string) string {
	return canonify.RunnerNamespacePrefix + runnerRecordID
}

// AgentEnabled reports whether runner serves action author sessions: its
// agent is enabled and the runner is not disabled.
func AgentEnabled(runner *core.Record) bool {
	return runner.GetBool("agent_enabled") && !runner.GetBool("disabled")
}

// servedAgentNamespaces holds the runners whose agent namespace this process
// prepared: registered, with Credimi's search attributes and its worker
// running here.
var servedAgentNamespaces sync.Map // runner record ID -> struct{}

// SetAgentNamespaceServed records whether this process runs the workers of
// the agent namespace of the runner with record ID runnerID.
func SetAgentNamespaceServed(runnerID string, served bool) {
	if served {
		servedAgentNamespaces.Store(runnerID, struct{}{})
		return
	}
	servedAgentNamespaces.Delete(runnerID)
}

// AgentNamespaceServed reports whether this process runs the workers of the
// agent namespace of the runner with record ID runnerID.
func AgentNamespaceServed(runnerID string) bool {
	_, ok := servedAgentNamespaces.Load(runnerID)
	return ok
}

// AgentNamespaceReady reports whether runner's agent may serve its namespace:
// the agent is enabled and Credimi prepared the namespace, in this process or
// in an earlier one (AgentNamespaceReadyField).
func AgentNamespaceReady(runner *core.Record) bool {
	if !AgentEnabled(runner) {
		return false
	}
	return runner.GetBool(AgentNamespaceReadyField) || AgentNamespaceServed(runner.Id)
}

// MarkAgentNamespaceReady sets AgentNamespaceReadyField on the runner with
// record ID runnerID while its agent is enabled and it is not disabled. It
// writes the column alone, so record hooks don't run again.
func MarkAgentNamespaceReady(app core.App, runnerID string) error {
	_, err := app.DB().Update(
		"mobile_runners",
		dbx.Params{AgentNamespaceReadyField: true},
		dbx.HashExp{"id": runnerID, "agent_enabled": true, "disabled": false},
	).Execute()
	if err != nil {
		return fmt.Errorf("mark runner %s agent namespace ready: %w", runnerID, err)
	}
	return nil
}
