// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
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

// signActionAuthorTicket signs payload's session as the start endpoint signs
// the tickets it enqueues.
func signActionAuthorTicket(
	t *testing.T,
	app core.App,
	payload *StartQueuedPipelineActivityInput,
) {
	t.Helper()
	start, err := workflowengine.DecodePayload[sessionStart](payload.ActionAuthor)
	require.NoError(t, err)
	token, err := actionauthor.SignTicket(app, start.grant(
		payload.OwnerNamespace,
		ActionAuthorSessionWorkflowIDPrefix+payload.TicketID,
	))
	require.NoError(t, err)
	payload.ActionAuthor["ticket_grant"] = token
}

func TestStartQueuedPipelineActivityStartsActionAuthorSession(t *testing.T) {
	act, captured := newCapturingQueuedPipelineActivity(
		t,
		"ActionAuthorSession-ticket-author",
		"run-author",
	)
	payload := actionAuthorTicketInput()
	signActionAuthorTicket(t, act.app, &payload)
	// A grant in the ticket is never trusted: the start overwrites it.
	payload.ActionAuthor["grant"] = "forged.grant"

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
	started, ok := input.Payload.(map[string]any)
	require.True(t, ok)
	grant, _ := started["grant"].(string)
	require.NoError(t, actionauthor.VerifySession(act.app, grant, actionauthor.SessionGrant{
		SessionWorkflowID: "ActionAuthorSession-ticket-author",
		Namespace:         testOrgANamespace,
		DeviceID:          "tenant/runner-1/device-1",
		Platform:          "android",
	}))
	delete(started, "grant")
	// The ticket's own grant does not reach the session.
	ticketSession := maps.Clone(payload.ActionAuthor)
	delete(ticketSession, "grant")
	delete(ticketSession, "ticket_grant")
	require.Equal(t, ticketSession, started)
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
	runID, err := actionauthor.SessionRun(
		act.app,
		testOrgANamespace,
		"ActionAuthorSession-ticket-author",
	)
	require.NoError(t, err)
	require.Equal(t, "run-author", runID)
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
	require.NoError(t, actionauthor.RecordSessionRun(
		act.app,
		testOrgANamespace,
		"ActionAuthorSession-ticket-author",
		"run-closed",
	))

	payload := actionAuthorTicketInput()
	signActionAuthorTicket(t, act.app, &payload)
	result, err := act.Execute(
		context.Background(),
		workflowengine.ActivityInput{Payload: payload},
	)
	require.NoError(t, err)

	require.Equal(t, []string{"ActionAuthorSession-ticket-author/"}, captured.described)
	output, ok := result.Output.(StartQueuedPipelineActivityOutput)
	require.True(t, ok)
	require.Equal(t, "ActionAuthorSession-ticket-author", output.WorkflowID)
	require.Equal(t, "run-closed", output.RunID)
}

func TestStartQueuedPipelineActivityRefusesAnotherActionAuthorSessionRun(t *testing.T) {
	act, _ := newCapturingQueuedPipelineActivity(
		t,
		"ActionAuthorSession-ticket-author",
		"run-restarted",
	)
	require.NoError(t, actionauthor.RecordSessionRun(
		act.app,
		testOrgANamespace,
		"ActionAuthorSession-ticket-author",
		"run-first",
	))

	payload := actionAuthorTicketInput()
	signActionAuthorTicket(t, act.app, &payload)
	_, err := act.Execute(
		context.Background(),
		workflowengine.ActivityInput{Payload: payload},
	)

	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "already ran as another run")
}

// A workflow started under the ticket's session ID before Credimi (attack c)
// is adopted by USE_EXISTING and recorded, but its input has no grant, so
// every lookup still refuses it.
func TestActionAuthorSessionStartedBeforeCredimiIsRefused(t *testing.T) {
	setFCAFTestEnv(t)
	act, _ := newCapturingQueuedPipelineActivity(t, authorTestSessionID, authorTestParentRun)
	payload := actionAuthorTicketInput()
	payload.TicketID = "ticket-1"
	payload.OwnerNamespace = authorTestNamespace
	signActionAuthorTicket(t, act.app, &payload)

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{Payload: payload})
	require.NoError(t, err)
	runID, err := actionauthor.SessionRun(act.app, authorTestNamespace, authorTestSessionID)
	require.NoError(t, err)
	require.Equal(t, authorTestParentRun, runID)

	session := authorTestSession()
	lookup := sessionLookup{
		app:     act.app,
		session: sessionDescription(),
		history: sessionHistory(t, map[string]any{
			"device_id":                 "tenant/runner-1/device-1",
			"platform":                  session.Platform,
			"wallet_version_identifier": session.WalletVersionIdentifier,
			"actor_id":                  session.ActorID,
		}),
	}
	_, err = lookupSessionStart(
		t.Context(),
		act.app,
		lookup.client(),
		authorTestNamespace,
		authorTestSessionID,
		authorTestParentRun,
	)
	require.ErrorIs(t, err, actionauthor.ErrForbidden)
	require.ErrorContains(t, err, "action author session was not started by Credimi")
}

// The semaphore does not authenticate who enqueues: a ticket without
// Credimi's signature of exactly its session never starts one.
func TestStartQueuedPipelineActivityRefusesUnsignedActionAuthorTickets(t *testing.T) {
	type ticketEdit func(t *testing.T, app core.App, p *StartQueuedPipelineActivityInput)
	for name, edit := range map[string]ticketEdit{
		"no ticket grant": func(*testing.T, core.App, *StartQueuedPipelineActivityInput) {},
		"forged ticket grant": func(_ *testing.T, _ core.App, p *StartQueuedPipelineActivityInput) {
			p.ActionAuthor["ticket_grant"] = "forged.grant"
		},
		"ticket grant of another session": func(
			t *testing.T,
			app core.App,
			p *StartQueuedPipelineActivityInput,
		) {
			signActionAuthorTicket(t, app, p)
			p.ActionAuthor["actor_id"] = "someone-else"
		},
		"ticket grant of another namespace": func(
			t *testing.T,
			app core.App,
			p *StartQueuedPipelineActivityInput,
		) {
			signActionAuthorTicket(t, app, p)
			p.OwnerNamespace = "userb-s-organization"
		},
		"session grant instead of a ticket grant": func(
			t *testing.T,
			app core.App,
			p *StartQueuedPipelineActivityInput,
		) {
			start, err := workflowengine.DecodePayload[sessionStart](p.ActionAuthor)
			require.NoError(t, err)
			token, err := actionauthor.SignSession(app, start.grant(
				p.OwnerNamespace,
				ActionAuthorSessionWorkflowIDPrefix+p.TicketID,
			))
			require.NoError(t, err)
			p.ActionAuthor["ticket_grant"] = token
		},
	} {
		t.Run(name, func(t *testing.T) {
			act, captured := newCapturingQueuedPipelineActivity(
				t,
				"ActionAuthorSession-ticket-author",
				"run-author",
			)
			payload := actionAuthorTicketInput()
			edit(t, act.app, &payload)

			_, err := act.Execute(
				context.Background(),
				workflowengine.ActivityInput{Payload: payload},
			)

			requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
			require.ErrorContains(t, err, "action author ticket was not enqueued by Credimi")
			require.Nil(t, captured.lastWorkflow)
			_, err = actionauthor.SessionRun(
				act.app,
				payload.OwnerNamespace,
				"ActionAuthorSession-ticket-author",
			)
			require.ErrorIs(t, err, actionauthor.ErrSessionNotRecorded)
		})
	}
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
