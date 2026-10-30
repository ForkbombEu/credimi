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
	"github.com/pocketbase/dbx"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
)

func actionAuthorTicketInput() StartQueuedPipelineActivityInput {
	return StartQueuedPipelineActivityInput{
		TicketID:          "ticket-author",
		OwnerNamespace:    testOrgANamespace,
		RequiredDeviceIDs: []string{"tenant/runner-1/device-1"},
		LeaderDeviceID:    "tenant/runner-1/device-1",
		RunKind:           mobiledevicesemaphore.RunKindActionAuthor,
		ActionAuthor: map[string]any{
			"device_id": "tenant/runner-1/device-1",
			"platform":  "android",
		},
	}
}

func TestStartQueuedPipelineActivityStartsActionAuthorSession(t *testing.T) {
	act, captured := newCapturingQueuedPipelineActivity(
		t,
		"ActionAuthorSession-ticket-author",
		"run-author",
	)
	payload := actionAuthorTicketInput()

	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{Payload: payload})
	require.NoError(t, err)

	require.Equal(t, ActionAuthorSessionWorkflowName, captured.lastWorkflow)
	require.Equal(t, "ActionAuthorSession-ticket-author", captured.lastOptions.ID)
	require.Equal(t, pipelineTaskQueue, captured.lastOptions.TaskQueue)
	require.Equal(t, 3*time.Hour, captured.lastOptions.WorkflowExecutionTimeout)
	require.Equal(
		t,
		enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		captured.lastOptions.WorkflowIDReusePolicy,
	)
	require.Equal(
		t,
		enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		captured.lastOptions.WorkflowIDConflictPolicy,
	)
	require.Len(t, captured.lastArgs, 1)
	input, ok := captured.lastArgs[0].(workflowengine.WorkflowInput)
	require.True(t, ok)
	require.Equal(t, payload.ActionAuthor, input.Payload)
	require.Equal(t, map[string]any{
		mobileDeviceSemaphoreTicketIDConfigKey:       "ticket-author",
		mobileDeviceSemaphoreDeviceIDsConfigKey:      []string{"tenant/runner-1/device-1"},
		mobileDeviceSemaphoreLeaderDeviceIDConfigKey: "tenant/runner-1/device-1",
		mobileDeviceSemaphoreOwnerNamespaceConfigKey: testOrgANamespace,
	}, input.Config)

	require.Equal(t, StartQueuedPipelineActivityOutput{
		WorkflowID:        "ActionAuthorSession-ticket-author",
		RunID:             "run-author",
		WorkflowNamespace: testOrgANamespace,
	}, result.Output)
	records, err := act.app.FindAllRecords(
		"pipeline_results",
		dbx.HashExp{"workflow_id": "ActionAuthorSession-ticket-author"},
	)
	require.NoError(t, err)
	require.Empty(t, records)
}

func TestStartQueuedPipelineActivityReusesClosedActionAuthorSession(t *testing.T) {
	act, captured := newCapturingQueuedPipelineActivity(t, "unused", "unused")
	captured.executeErr = serviceerror.NewWorkflowExecutionAlreadyStarted("already started", "", "")
	captured.describeRun = "run-closed"

	result, err := act.Execute(
		context.Background(),
		workflowengine.ActivityInput{Payload: actionAuthorTicketInput()},
	)
	require.NoError(t, err)

	require.Equal(t, []string{"ActionAuthorSession-ticket-author/"}, captured.described)
	output, ok := result.Output.(StartQueuedPipelineActivityOutput)
	require.True(t, ok)
	require.Equal(t, "ActionAuthorSession-ticket-author", output.WorkflowID)
	require.Equal(t, "run-closed", output.RunID)
}

func TestStartQueuedPipelineActivityRejectsInvalidRunKinds(t *testing.T) {
	missingSession := actionAuthorTicketInput()
	missingSession.ActionAuthor = nil
	unknownKind := actionAuthorTicketInput()
	unknownKind.RunKind = "replay"
	missingTicket := actionAuthorTicketInput()
	missingTicket.TicketID = " "

	for name, tc := range map[string]struct {
		payload StartQueuedPipelineActivityInput
		message string
	}{
		"missing session": {payload: missingSession, message: "action_author is required"},
		"unknown kind":    {payload: unknownKind, message: `unsupported run_kind "replay"`},
		"missing ticket":  {payload: missingTicket, message: "ticket_id is required"},
	} {
		t.Run(name, func(t *testing.T) {
			act, captured := newCapturingQueuedPipelineActivity(t, "wf", "run")

			_, err := act.Execute(
				context.Background(),
				workflowengine.ActivityInput{Payload: tc.payload},
			)

			require.ErrorContains(t, err, tc.message)
			require.Nil(t, captured.lastWorkflow)
		})
	}
}
