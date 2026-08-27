// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func requireWorkflowLogsCapability(
	t *testing.T,
	input workflowengine.WorkflowInput,
	expected bool,
) {
	t.Helper()
	memo, ok := input.Config["memo"].(map[string]any)
	require.True(t, ok)
	require.Equal(
		t,
		workflowengine.CredimiCapabilities{Logs: expected},
		memo[workflowengine.CredimiCapabilitiesMemoKey],
	)
}

func TestConformanceSuiteHasLogs(t *testing.T) {
	tests := []struct {
		name     string
		suite    string
		expected bool
	}{
		{name: "OpenID", suite: OpenIDConformanceSuite, expected: true},
		{name: "EWC", suite: EWCSuite, expected: true},
		{name: "Webuild", suite: WebuildSuite, expected: true},
		{name: "unsupported suite", suite: EudiwSuite},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, ConformanceSuiteHasLogs(test.suite))
		})
	}
}
