// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestRunnerNamespace(t *testing.T) {
	require.Equal(t, "runner-4g6a8f4c4m5441k", RunnerNamespace("4g6a8f4c4m5441k"))
}

func TestNamespaceEnabled(t *testing.T) {
	coll := core.NewBaseCollection("mobile_runners")
	record := core.NewRecord(coll)
	require.True(t, NamespaceEnabled(record))
	record.Set("disabled", true)
	require.False(t, NamespaceEnabled(record))
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

func TestRunnerNamespaceReady(t *testing.T) {
	coll := core.NewBaseCollection("mobile_runners")
	runner := core.NewRecord(coll)
	runner.Id = "readytest"
	require.False(t, RunnerNamespaceReady(runner))

	// Prepared by an earlier Credimi process.
	runner.Set(RunnerNamespaceReadyField, true)
	require.True(t, RunnerNamespaceReady(runner))

	// Prepared by this process.
	runner.Set(RunnerNamespaceReadyField, false)
	SetRunnerNamespaceServed(runner.Id, true)
	require.True(t, RunnerNamespaceServed(runner.Id))
	require.True(t, RunnerNamespaceReady(runner))

	// Never ready while the runner is disabled; the agent does not matter.
	runner.Set(RunnerNamespaceReadyField, true)
	runner.Set("disabled", true)
	require.False(t, RunnerNamespaceReady(runner))
	runner.Set("disabled", false)
	require.True(t, RunnerNamespaceReady(runner))

	SetRunnerNamespaceServed(runner.Id, false)
	require.False(t, RunnerNamespaceServed(runner.Id))
}
