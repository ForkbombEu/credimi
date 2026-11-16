// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalclient

import (
	"context"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

// Describer is the part of client.Client that describes a workflow run.
type Describer interface {
	DescribeWorkflow(
		ctx context.Context,
		workflowID, runID string,
	) (*client.WorkflowExecutionDescription, error)
}

// Describe returns the server's full description of a workflow run.
func Describe(
	ctx context.Context,
	c Describer,
	workflowID, runID string,
) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	d, err := c.DescribeWorkflow(ctx, workflowID, runID)
	if err != nil {
		return nil, err
	}
	return d.RawResponse, nil
}
