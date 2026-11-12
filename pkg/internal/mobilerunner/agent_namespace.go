// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
)

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
