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
