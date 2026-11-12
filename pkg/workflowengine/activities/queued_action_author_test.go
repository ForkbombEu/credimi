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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
)

// testRunnerNamespace is the agent namespace of the runner of the
// action author tickets' device.
const testRunnerNamespace = "runner-" + testRunnerID

// testRunnerID is the record ID of that runner.
const testRunnerID = "testrunner00001"

func actionAuthorTicketInput() StartQueuedPipelineActivityInput {
	return StartQueuedPipelineActivityInput{
		TicketID:          "00000000-0000-4000-8000-0000000000aa",
		OwnerNamespace:    testOrgANamespace,
		RequiredDeviceIDs: []string{"tenant/runner-1/device-1"},
		LeaderDeviceID:    "tenant/runner-1/device-1",
		RunKind:           mobiledevicesemaphore.RunKindActionAuthor,
		ActionAuthor: map[string]any{
			"device_id":                 "tenant/runner-1/device-1",
			"platform":                  "android",
			"wallet_version_identifier": testOrgANamespace + "/wallet/v1",
			"actor_id":                  "user-1",
			"namespace":                 testRunnerNamespace,
			"organization":              testOrgANamespace,
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
		start.Namespace,
		actionauthor.SessionWorkflowID(testRunnerID, payload.TicketID),
	))
	require.NoError(t, err)
	payload.ActionAuthor["ticket_grant"] = token
}

func TestStartQueuedPipelineActivityStartsActionAuthorSession(t *testing.T) {
	act, captured := newCapturingQueuedPipelineActivity(
		t,
		queuedSessionID,
		"run-author",
	)
	payload := actionAuthorTicketInput()
	signActionAuthorTicket(t, act.app, &payload)
	// A grant in the ticket is never trusted: the start overwrites it.
	payload.ActionAuthor["grant"] = "forged.grant"

	result, err := act.Execute(context.Background(), workflowengine.ActivityInput{Payload: payload})
	require.NoError(t, err)

	require.Equal(t, ActionAuthorSessionWorkflowName, captured.lastWorkflow)
	// The session runs in the runner's agent namespace.
	require.Equal(t, []string{testRunnerNamespace}, captured.namespaces)
	require.Equal(t, queuedSessionID, captured.lastOptions.ID)
	require.Equal(t, pipelineTaskQueue, captured.lastOptions.TaskQueue)
	require.Equal(t, 3*time.Hour, captured.lastOptions.WorkflowExecutionTimeout)
	require.Equal(
		t,
		enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		captured.lastOptions.WorkflowIDReusePolicy,
	)
	require.Equal(
		t,
		enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
		captured.lastOptions.WorkflowIDConflictPolicy,
	)
	attributes := captured.lastOptions.TypedSearchAttributes
	actor, ok := attributes.GetKeyword(
		temporal.NewSearchAttributeKeyKeyword(workflowengine.ActionAuthorActorSearchAttribute),
	)
	require.True(t, ok)
	require.Equal(t, "user-1", actor)
	wallet, ok := attributes.GetKeyword(
		temporal.NewSearchAttributeKeyKeyword(workflowengine.ActionAuthorWalletSearchAttribute),
	)
	require.True(t, ok)
	require.Equal(t, testOrgANamespace+"/wallet", wallet)
	require.Len(t, captured.lastArgs, 1)
	input, ok := captured.lastArgs[0].(workflowengine.WorkflowInput)
	require.True(t, ok)
	started, ok := input.Payload.(map[string]any)
	require.True(t, ok)
	grant, _ := started["grant"].(string)
	require.NoError(t, actionauthor.VerifySession(act.app, grant, actionauthor.SessionGrant{
		SessionWorkflowID:       queuedSessionID,
		Namespace:               testRunnerNamespace,
		Organization:            testOrgANamespace,
		DeviceID:                "tenant/runner-1/device-1",
		Platform:                "android",
		WalletVersionIdentifier: testOrgANamespace + "/wallet/v1",
		ActorID:                 "user-1",
	}))
	delete(started, "grant")
	// The ticket's own grant does not reach the session.
	ticketSession := maps.Clone(payload.ActionAuthor)
	delete(ticketSession, "grant")
	delete(ticketSession, "ticket_grant")
	require.Equal(t, ticketSession, started)
	require.Equal(t, map[string]any{
		mobileDeviceSemaphoreTicketIDConfigKey:       "00000000-0000-4000-8000-0000000000aa",
		mobileDeviceSemaphoreDeviceIDsConfigKey:      []string{"tenant/runner-1/device-1"},
		mobileDeviceSemaphoreLeaderDeviceIDConfigKey: "tenant/runner-1/device-1",
		mobileDeviceSemaphoreOwnerNamespaceConfigKey: testOrgANamespace,
	}, input.Config)

	require.Equal(t, StartQueuedPipelineActivityOutput{
		WorkflowID:        queuedSessionID,
		RunID:             "run-author",
		WorkflowNamespace: testRunnerNamespace,
	}, result.Output)
	records, err := act.app.FindAllRecords(
		"pipeline_results",
		dbx.HashExp{"workflow_id": queuedSessionID},
	)
	require.NoError(t, err)
	require.Empty(t, records)
	requireRunGrantSignal(t, act.app, captured, "run-author")
}

// requireRunGrantSignal requires that the start sent exactly one signal: the
// run grant of the session run runID.
func requireRunGrantSignal(
	t *testing.T,
	app core.App,
	captured *capturingTemporalClient,
	runID string,
) {
	t.Helper()
	require.Len(t, captured.signals, 1)
	signal := captured.signals[0]
	require.Equal(t, queuedSessionID, signal.workflowID)
	require.Equal(t, runID, signal.runID)
	require.Equal(t, ActionAuthorRunGrantSignal, signal.name)
	token, ok := signal.arg.(string)
	require.True(t, ok)
	require.NoError(t, actionauthor.VerifyRun(app, token, actionauthor.RunGrant{
		SessionWorkflowID: queuedSessionID,
		RunID:             runID,
	}))
}

// startedSessionClient fakes the runner namespace of a session the ticket's
// start finds already started, whose runs (oldest first) start with
// histories[run].
func startedSessionClient(
	t *testing.T,
	act *StartQueuedPipelineActivity,
	histories map[string]*fcafHistoryIterator,
	runs ...string,
) *[]string {
	t.Helper()
	mockClient := &temporalmocks.Client{}
	stubSessionRuns(mockClient, queuedSessionID, runs...)
	for _, run := range runs {
		mockClient.On(
			"GetWorkflowHistory",
			mock.Anything,
			queuedSessionID,
			run,
			false,
			enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).Return(replayHistory(histories[run])).Maybe()
	}
	var namespaces []string
	act.sessionClientFactory = func(namespace string) (client.Client, error) {
		namespaces = append(namespaces, namespace)
		return mockClient, nil
	}
	return &namespaces
}

// queuedSessionID is the session of actionAuthorTicketInput.
const queuedSessionID = "ActionAuthorSession-testrunner00001-00000000-0000-4000-8000-0000000000aa"

// queuedSessionHistory is the start of the session of payload, with
// Credimi's grant of it when signed.
func queuedSessionHistory(
	t *testing.T,
	app core.App,
	payload StartQueuedPipelineActivityInput,
	signed bool,
) *fcafHistoryIterator {
	t.Helper()
	input := maps.Clone(payload.ActionAuthor)
	delete(input, "ticket_grant")
	if signed {
		start, err := workflowengine.DecodePayload[sessionStart](payload.ActionAuthor)
		require.NoError(t, err)
		grant, err := actionauthor.SignSession(app, start.grant(start.Namespace, queuedSessionID))
		require.NoError(t, err)
		input["grant"] = grant
	}
	return sessionHistory(t, input)
}

// A retried start whose session already started reuses the run Credimi
// started first, and signals it its run grant: the previous attempt may have
// ended before it signaled, so the run may have none yet.
func TestStartQueuedPipelineActivityReusesItsOwnActionAuthorSession(t *testing.T) {
	setFCAFTestEnv(t)
	act, captured := newCapturingQueuedPipelineActivity(t, "unused", "unused")
	captured.executeErr = serviceerror.NewWorkflowExecutionAlreadyStarted("already started", "", "")
	payload := actionAuthorTicketInput()
	signActionAuthorTicket(t, act.app, &payload)
	namespaces := startedSessionClient(t, act, map[string]*fcafHistoryIterator{
		"run-first":  queuedSessionHistory(t, act.app, payload, true),
		"run-replay": queuedSessionHistory(t, act.app, payload, true),
	}, "run-first", "run-replay")

	result, err := act.Execute(
		context.Background(),
		workflowengine.ActivityInput{Payload: payload},
	)
	require.NoError(t, err)

	require.Equal(t, []string{testRunnerNamespace}, *namespaces)
	require.Equal(t, StartQueuedPipelineActivityOutput{
		WorkflowID:        queuedSessionID,
		RunID:             "run-first",
		WorkflowNamespace: testRunnerNamespace,
	}, result.Output)
	requireRunGrantSignal(t, act.app, captured, "run-first")
}

// A start whose run grant signal fails is retried: the retry reuses the run
// and signals it again.
func TestStartQueuedPipelineActivityRetriesAFailedRunGrantSignal(t *testing.T) {
	act, captured := newCapturingQueuedPipelineActivity(t, queuedSessionID, "run-author")
	captured.signalErr = serviceerror.NewUnavailable("temporal unavailable")
	payload := actionAuthorTicketInput()
	signActionAuthorTicket(t, act.app, &payload)

	_, err := act.Execute(context.Background(), workflowengine.ActivityInput{Payload: payload})

	requireActivityError(t, err, errorcodes.PipelineExecutionError, false)
	require.ErrorContains(t, err, "signal action author session run grant")
	require.Len(t, captured.signals, 1)
	require.Equal(t, "run-author", captured.signals[0].runID)
}

// A workflow started under the ticket's session ID before Credimi is never
// adopted: the start fails for good.
func TestStartQueuedPipelineActivityRefusesAForeignEarlierSession(t *testing.T) {
	setFCAFTestEnv(t)
	act, captured := newCapturingQueuedPipelineActivity(t, "unused", "unused")
	captured.executeErr = serviceerror.NewWorkflowExecutionAlreadyStarted("already started", "", "")
	payload := actionAuthorTicketInput()
	signActionAuthorTicket(t, act.app, &payload)
	startedSessionClient(t, act, map[string]*fcafHistoryIterator{
		"run-foreign": queuedSessionHistory(t, act.app, payload, false),
	}, "run-foreign")

	_, err := act.Execute(
		context.Background(),
		workflowengine.ActivityInput{Payload: payload},
	)

	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(
		t,
		err,
		"action author session "+queuedSessionID+" was not started by Credimi",
	)
	require.Empty(t, captured.signals)
}

// A started session Temporal does not list yet is retried.
func TestStartQueuedPipelineActivityRetriesAnInvisibleActionAuthorSession(t *testing.T) {
	act, captured := newCapturingQueuedPipelineActivity(t, "unused", "unused")
	captured.executeErr = serviceerror.NewWorkflowExecutionAlreadyStarted("already started", "", "")
	payload := actionAuthorTicketInput()
	signActionAuthorTicket(t, act.app, &payload)
	startedSessionClient(t, act, nil)

	_, err := act.Execute(
		context.Background(),
		workflowengine.ActivityInput{Payload: payload},
	)

	requireActivityError(t, err, errorcodes.PipelineExecutionError, false)
	require.ErrorContains(t, err, "action author session is not visible in Temporal yet")
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
			p.ActionAuthor["namespace"] = "runner-other"
		},
		"ticket grant of another organization": func(
			t *testing.T,
			app core.App,
			p *StartQueuedPipelineActivityInput,
		) {
			signActionAuthorTicket(t, app, p)
			p.OwnerNamespace = "userb-s-organization"
			p.ActionAuthor["organization"] = "userb-s-organization"
		},
		"session grant instead of a ticket grant": func(
			t *testing.T,
			app core.App,
			p *StartQueuedPipelineActivityInput,
		) {
			start, err := workflowengine.DecodePayload[sessionStart](p.ActionAuthor)
			require.NoError(t, err)
			token, err := actionauthor.SignSession(app, start.grant(
				start.Namespace,
				actionauthor.SessionWorkflowID(testRunnerID, p.TicketID),
			))
			require.NoError(t, err)
			p.ActionAuthor["ticket_grant"] = token
		},
	} {
		t.Run(name, func(t *testing.T) {
			act, captured := newCapturingQueuedPipelineActivity(
				t,
				queuedSessionID,
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
		})
	}
}

// A ticket starts a session only in a runner namespace and only for the
// organization that owns the ticket, even when Credimi signed it.
func TestStartQueuedPipelineActivityRefusesActionAuthorTicketScope(t *testing.T) {
	for name, tc := range map[string]struct {
		namespace    string
		organization string
		message      string
	}{
		"organization namespace": {
			namespace:    testOrgANamespace,
			organization: testOrgANamespace,
			message:      `namespace "usera-s-organization" is not a runner namespace`,
		},
		"organization other than the ticket owner": {
			namespace:    testRunnerNamespace,
			organization: "userb-s-organization",
			message: `organization "userb-s-organization" is not the ticket owner ` +
				`"usera-s-organization"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			act, captured := newCapturingQueuedPipelineActivity(
				t,
				queuedSessionID,
				"run-author",
			)
			payload := actionAuthorTicketInput()
			payload.ActionAuthor["namespace"] = tc.namespace
			payload.ActionAuthor["organization"] = tc.organization
			signActionAuthorTicket(t, act.app, &payload)

			_, err := act.Execute(
				context.Background(),
				workflowengine.ActivityInput{Payload: payload},
			)

			requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
			require.ErrorContains(t, err, tc.message)
			require.Nil(t, captured.lastWorkflow)
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
