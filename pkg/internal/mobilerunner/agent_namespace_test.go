// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestAgentNamespace(t *testing.T) {
	require.Equal(t, "runner-4g6a8f4c4m5441k", AgentNamespace("4g6a8f4c4m5441k"))
}

func TestAgentEnabled(t *testing.T) {
	coll := core.NewBaseCollection("mobile_runners")
	for _, tc := range []struct {
		enabled, disabled, want bool
	}{
		{enabled: true, want: true},
		{enabled: true, disabled: true},
		{},
		{disabled: true},
	} {
		record := core.NewRecord(coll)
		record.Set("agent_enabled", tc.enabled)
		record.Set("disabled", tc.disabled)
		require.Equal(t, tc.want, AgentEnabled(record), tc)
	}
}

func TestAgentNamespaceReady(t *testing.T) {
	coll := core.NewBaseCollection("mobile_runners")
	runner := core.NewRecord(coll)
	runner.Id = "readytest"
	runner.Set("agent_enabled", true)
	require.False(t, AgentNamespaceReady(runner))

	// Prepared by an earlier Credimi process.
	runner.Set(AgentNamespaceReadyField, true)
	require.True(t, AgentNamespaceReady(runner))

	// Prepared by this process.
	runner.Set(AgentNamespaceReadyField, false)
	SetAgentNamespaceServed(runner.Id, true)
	require.True(t, AgentNamespaceServed(runner.Id))
	require.True(t, AgentNamespaceReady(runner))

	// Never ready while the agent is off or the runner disabled.
	runner.Set(AgentNamespaceReadyField, true)
	runner.Set("disabled", true)
	require.False(t, AgentNamespaceReady(runner))
	runner.Set("disabled", false)
	runner.Set("agent_enabled", false)
	require.False(t, AgentNamespaceReady(runner))

	SetAgentNamespaceServed(runner.Id, false)
	require.False(t, AgentNamespaceServed(runner.Id))
}
