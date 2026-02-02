// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunnersemaphore

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowID(t *testing.T) {
	workflowID := WorkflowID("runner-1")

	require.Equal(t, "mobile-runner-semaphore/runner-1", workflowID)
}

func TestPermitLeaseID(t *testing.T) {
	workflowID := "mobile-runner-semaphore/runner-1"
	leaseID := PermitLeaseID(workflowID, "run-1", "runner-1")

	require.Equal(t, "mobile-runner-semaphore/runner-1/run-1/runner-1", leaseID)

	otherLeaseID := PermitLeaseID(workflowID, "run-2", "runner-1")
	require.NotEqual(t, leaseID, otherLeaseID)
}
