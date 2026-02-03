// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplySemaphoreTicketMetadata(t *testing.T) {
	config := map[string]any{}
	payload := StartQueuedPipelineActivityInput{
		TicketID:          "ticket-1",
		OwnerNamespace:    "tenant-1",
		RequiredRunnerIDs: []string{"runner-1", "runner-2"},
		LeaderRunnerID:    "runner-1",
	}

	applySemaphoreTicketMetadata(config, payload)

	require.Equal(t, "ticket-1", config[mobileRunnerSemaphoreTicketIDConfigKey])
	require.Equal(t, []string{"runner-1", "runner-2"}, config[mobileRunnerSemaphoreRunnerIDsConfigKey])
	require.Equal(t, "runner-1", config[mobileRunnerSemaphoreLeaderRunnerIDConfigKey])
	require.Equal(t, "tenant-1", config[mobileRunnerSemaphoreOwnerNamespaceConfigKey])
}
