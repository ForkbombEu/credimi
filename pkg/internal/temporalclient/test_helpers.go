// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalclient

import (
	"sync"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

// SetClientForTests injects a client into the cache for unit tests.
func SetClientForTests(namespace string, c client.Client) {
	if testClientCache == nil {
		testClientCache = &sync.Map{}
	}
	testClientCache.Store(namespace, c)
}

// ClearTestClients clears the test cache.
func ClearTestClients() {
	testClientCache = nil
}

// Description wraps a raw describe response as DescribeWorkflow returns it.
func Description(
	resp *workflowservice.DescribeWorkflowExecutionResponse,
) *client.WorkflowExecutionDescription {
	if resp == nil {
		return nil
	}
	return &client.WorkflowExecutionDescription{RawResponse: resp}
}
