// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
)

func TestCheckWorkflowClosedActivityMissingFields(t *testing.T) {
	activity := NewCheckWorkflowClosedActivity()
	require.Equal(t, "Check workflow closed", activity.Name())

	_, err := activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: "not-a-workflow-status-payload",
	})
	require.Error(t, err)

	_, err = activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CheckWorkflowClosedActivityInput{
			WorkflowNamespace: "ns",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code)

	_, err = activity.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: CheckWorkflowClosedActivityInput{
			WorkflowID: "wf-1",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "workflow_namespace is required")
}
