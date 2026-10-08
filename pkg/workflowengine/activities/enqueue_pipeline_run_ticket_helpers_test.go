// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
)

func TestNormalizeDeviceIDs(t *testing.T) {
	out := normalizeDeviceIDs([]string{" /tenant/runner-1 ", "", "tenant/runner-2"})
	require.Equal(t, []string{"tenant/runner-1", "tenant/runner-2"}, out)
	require.Nil(t, normalizeDeviceIDs(nil))
}

func TestIsQueueLimitExceeded(t *testing.T) {
	err := temporal.NewApplicationError("limit", mobiledevicesemaphore.ErrQueueLimitExceeded)
	require.True(t, isQueueLimitExceeded(err))
	require.False(t, isQueueLimitExceeded(nil))
}

// TestEnqueuePipelineRunTicketActivityRollsBackAttemptedRunners verifies a failed
// enqueue cancels the ticket on every attempted runner and ignores missing tickets.
func TestEnqueuePipelineRunTicketActivityRollsBackAttemptedRunners(t *testing.T) {
	mockClient := temporalmocks.NewClient(t)
	mockClient.On("NewWithStartWorkflowOperation", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	mockClient.On("UpdateWithStartWorkflow", mock.Anything, mock.Anything).
		Return(nil, serviceerror.NewUnavailable("down")).
		Once()
	mockClient.
		On(
			"UpdateWorkflow",
			mock.Anything,
			mock.MatchedBy(func(o client.UpdateWorkflowOptions) bool {
				return o.WorkflowID == mobiledevicesemaphore.WorkflowID("runner-1") &&
					o.UpdateName == mobiledevicesemaphore.CancelRunUpdate &&
					o.UpdateID == "cancel/runner-1/ticket-1"
			}),
		).
		Return(nil, &serviceerror.NotFound{Message: "missing"}).
		Once()

	act := NewEnqueuePipelineRunTicketActivity()
	act.temporalClientFactory = func(string) (client.Client, error) { return mockClient, nil }

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: EnqueuePipelineRunTicketActivityInput{
			TicketID:           "ticket-1",
			OwnerNamespace:     "tenant-1",
			EnqueuedAt:         time.Now().UTC(),
			DeviceIDs:          []string{"runner-1", "runner-2"},
			PipelineIdentifier: "tenant-1/pipeline",
			YAML:               "name: test\nsteps: []\n",
		},
	})
	require.ErrorContains(t, err, "down")
}
