// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"errors"
	"maps"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// authorTestNamespace is the runner namespace sessions run in;
	// authorTestOrganization is the organization of authorTestSession.
	authorTestNamespace    = "runner-testrunner00001"
	authorTestOrganization = "tenant"
	authorTestSessionID    = "ActionAuthorSession-testrunner00001-00000000-0000-4000-8000-000000000001"
	authorTestAuthorID     = authorTestSessionID + "-author"
	authorTestAuthorRun    = "author-run"
	authorTestParentRun    = "session-run"
)

func authorTestCaller() authorCaller {
	return authorCaller{
		Namespace:    authorTestNamespace,
		WorkflowType: ActionAuthorWorkflowType,
		WorkflowID:   authorTestAuthorID,
		RunID:        authorTestAuthorRun,
	}
}

func authorTestSession() actionauthor.BuildRequest {
	return actionauthor.BuildRequest{
		Actor:                   actionauthor.Actor{ActorID: "user-1"},
		WalletVersionIdentifier: "tenant/wallet/v1",
		Platform:                actionauthor.PlatformAndroid,
	}
}

func describeExecution(
	workflowType string,
	status enumspb.WorkflowExecutionStatus,
	parent *commonpb.WorkflowExecution,
) *workflowservice.DescribeWorkflowExecutionResponse {
	return &workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
			Type:            &commonpb.WorkflowType{Name: workflowType},
			Status:          status,
			ParentExecution: parent,
		},
	}
}

func authorDescription() *workflowservice.DescribeWorkflowExecutionResponse {
	return describeExecution(
		ActionAuthorWorkflowType,
		enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		&commonpb.WorkflowExecution{WorkflowId: authorTestSessionID, RunId: authorTestParentRun},
	)
}

func sessionDescription() *workflowservice.DescribeWorkflowExecutionResponse {
	return describeExecution(
		ActionAuthorSessionWorkflowName,
		enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		nil,
	)
}

// sessionHistory is the session's history starting with its start input.
func sessionHistory(t *testing.T, payload any) *fcafHistoryIterator {
	t.Helper()
	input, err := temporalcrypto.DataConverter().ToPayloads(workflowengine.WorkflowInput{
		Payload: payload,
		Config:  map[string]any{"mobile_device_semaphore_ticket_id": "ticket-1"},
	})
	require.NoError(t, err)
	return &fcafHistoryIterator{events: []*historypb.HistoryEvent{{
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
			WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
				Input: input,
			},
		},
	}}}
}

// signedSessionHistory is the history of the session authorTestSessionID
// that Credimi started with input as run authorTestParentRun in
// authorTestNamespace, for the wallet's organization (the first segment of
// its identifier), signed and recorded as the start activity signs and
// records it.
func signedSessionHistory(
	t *testing.T,
	app core.App,
	input map[string]any,
) *fcafHistoryIterator {
	t.Helper()
	field := func(name string) string {
		value, _ := input[name].(string)
		return value
	}
	organization, _, _ := strings.Cut(field("wallet_version_identifier"), "/")
	grant, err := actionauthor.SignSession(app, actionauthor.SessionGrant{
		SessionWorkflowID:       authorTestSessionID,
		Namespace:               authorTestNamespace,
		Organization:            organization,
		DeviceID:                field("device_id"),
		Platform:                field("platform"),
		WalletVersionIdentifier: field("wallet_version_identifier"),
		ActorID:                 field("actor_id"),
	})
	require.NoError(t, err)
	require.NoError(t, actionauthor.RecordSessionRun(
		app,
		authorTestNamespace,
		authorTestSessionID,
		authorTestParentRun,
		actionauthor.SessionDetails{ActorID: field("actor_id"), Organization: organization},
	))
	signed := maps.Clone(input)
	signed["namespace"] = authorTestNamespace
	signed["organization"] = organization
	signed["grant"] = grant
	return sessionHistory(t, signed)
}

// sessionLookup fakes the caller's namespace with the given executions; app
// checks the session's grant.
type sessionLookup struct {
	app        core.App
	clientErr  error
	author     *workflowservice.DescribeWorkflowExecutionResponse
	authorErr  error
	session    *workflowservice.DescribeWorkflowExecutionResponse
	sessionErr error
	history    client.HistoryEventIterator
}

// unsignedSessionLookup fakes the running session without a history: tests
// set one with signedSessionHistory, which records the session once.
func unsignedSessionLookup(app core.App) sessionLookup {
	return sessionLookup{
		app:     app,
		author:  authorDescription(),
		session: sessionDescription(),
	}
}

// validSessionLookup fakes the running, Credimi-signed session of
// authorTestSession in authorTestNamespace.
func validSessionLookup(t *testing.T, app core.App) sessionLookup {
	t.Helper()
	session := authorTestSession()
	lookup := unsignedSessionLookup(app)
	if app != nil {
		lookup.history = signedSessionHistory(t, app, map[string]any{
			"device_id":                 "tenant/runner-1/device-1",
			"platform":                  session.Platform,
			"wallet_version_identifier": session.WalletVersionIdentifier,
			"actor_id":                  session.ActorID,
		})
	}
	return lookup
}

// client fakes the caller's namespace client.
func (l sessionLookup) client() *temporalmocks.Client {
	mockClient := &temporalmocks.Client{}
	mockClient.On("DescribeWorkflowExecution", mock.Anything, authorTestAuthorID, authorTestAuthorRun).
		Return(l.author, l.authorErr).
		Maybe()
	mockClient.On("DescribeWorkflowExecution", mock.Anything, authorTestSessionID, authorTestParentRun).
		Return(l.session, l.sessionErr).
		Maybe()
	mockClient.On(
		"GetWorkflowHistory",
		mock.Anything,
		authorTestSessionID,
		authorTestParentRun,
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(l.history).Maybe()
	return mockClient
}

// lookup resolves callers against the faked namespace.
func (l sessionLookup) lookup() authorSessionLookup {
	mockClient := l.client()
	return authorSessionLookup{app: l.app, temporalClient: func(string) (client.Client, error) {
		if l.clientErr != nil {
			return nil, l.clientErr
		}
		return mockClient, nil
	}}
}

func (l sessionLookup) activity(t *testing.T, app core.App) *SaveAuthoredWalletActionActivity {
	t.Helper()
	a := NewSaveAuthoredWalletActionActivity(app)
	a.authorSessionLookup = l.lookup()
	return a
}

func TestSaveAuthoredWalletActionRejectsCallersOutsideARunningSession(t *testing.T) {
	setFCAFTestEnv(t)
	app := newCredimiTestApp(t)
	notFound := serviceerror.NewNotFound("not found")
	tests := []struct {
		name      string
		caller    func(*authorCaller)
		lookup    func(*sessionLookup)
		payload   func(*actionauthor.SaveRequest)
		code      string
		retryable bool
		message   string
	}{
		{
			name:    "missing namespace",
			caller:  func(c *authorCaller) { c.Namespace = "" },
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not an action author session agent",
		},
		{
			name:    "another workflow type",
			caller:  func(c *authorCaller) { c.WorkflowType = "Some pipeline" },
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not an action author session agent",
		},
		{
			name:    "workflow ID without author suffix",
			caller:  func(c *authorCaller) { c.WorkflowID = authorTestSessionID },
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not an action author session agent",
		},
		{
			name:    "workflow ID without session prefix",
			caller:  func(c *authorCaller) { c.WorkflowID = "Pipeline-1-author" },
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not an action author session agent",
		},
		{
			name:    "workflow ID without ticket",
			caller:  func(c *authorCaller) { c.WorkflowID = ActionAuthorSessionWorkflowIDPrefix + "-author" },
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not an action author session agent",
		},
		{
			name:      "namespace client failure is retried",
			lookup:    func(l *sessionLookup) { l.clientErr = errors.New("temporal down") },
			code:      errorcodes.UnexpectedWorkflowError,
			retryable: true,
			message:   "temporal down",
		},
		{
			name:    "unknown caller execution",
			lookup:  func(l *sessionLookup) { l.authorErr = notFound },
			code:    errorcodes.RecordNotAccessible,
			message: "action author session not found",
		},
		{
			name:      "caller describe failure is retried",
			lookup:    func(l *sessionLookup) { l.authorErr = errors.New("unavailable") },
			code:      errorcodes.UnexpectedWorkflowError,
			retryable: true,
			message:   "unavailable",
		},
		{
			name: "caller is not a child workflow",
			lookup: func(l *sessionLookup) {
				l.author.WorkflowExecutionInfo.ParentExecution = nil
			},
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not the child of its action author session",
		},
		{
			name: "caller is the child of another workflow",
			lookup: func(l *sessionLookup) {
				l.author.WorkflowExecutionInfo.ParentExecution.WorkflowId = "ActionAuthorSession-other"
			},
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not the child of its action author session",
		},
		{
			name: "described caller has another type",
			lookup: func(l *sessionLookup) {
				l.author.WorkflowExecutionInfo.Type.Name = "Other"
			},
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not the child of its action author session",
		},
		{
			name:    "session not found",
			lookup:  func(l *sessionLookup) { l.sessionErr = notFound },
			code:    errorcodes.RecordNotAccessible,
			message: "action author session not found",
		},
		{
			name: "session closed",
			lookup: func(l *sessionLookup) {
				l.session.WorkflowExecutionInfo.Status = enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED
			},
			code:    errorcodes.RecordNotAccessible,
			message: "action author session is not running",
		},
		{
			name: "parent is not a session",
			lookup: func(l *sessionLookup) {
				l.session.WorkflowExecutionInfo.Type.Name = "Some pipeline"
			},
			code:    errorcodes.RecordNotAccessible,
			message: "action author session is not running",
		},
		{
			name:    "session history is empty",
			lookup:  func(l *sessionLookup) { l.history = &fcafHistoryIterator{} },
			code:    errorcodes.RecordNotAccessible,
			message: "action author session has no start input",
		},
		{
			name: "session history read failure is retried",
			lookup: func(l *sessionLookup) {
				l.history = &failingHistoryIterator{err: errors.New("history unavailable")}
			},
			code:      errorcodes.UnexpectedWorkflowError,
			retryable: true,
			message:   "history unavailable",
		},
		{
			name: "first event is not the start",
			lookup: func(l *sessionLookup) {
				l.history = &fcafHistoryIterator{events: []*historypb.HistoryEvent{
					{EventType: enumspb.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED},
				}}
			},
			code:    errorcodes.RecordNotAccessible,
			message: "action author session has no start input",
		},
		{
			name: "session input without actor",
			lookup: func(l *sessionLookup) {
				l.history = sessionHistory(t, map[string]any{
					"platform":                  actionauthor.PlatformAndroid,
					"wallet_version_identifier": "tenant/wallet/v1",
				})
			},
			code:    errorcodes.RecordNotAccessible,
			message: "action author session has no actor or build",
		},
		{
			name:    "payload names another actor",
			payload: func(p *actionauthor.SaveRequest) { p.ActorID = "user-2" },
			code:    errorcodes.RecordNotAccessible,
			message: "save request does not match the session",
		},
		{
			name: "payload names another build",
			payload: func(p *actionauthor.SaveRequest) {
				p.WalletVersionIdentifier = "other/wallet/v1"
			},
			code:    errorcodes.RecordNotAccessible,
			message: "save request does not match the session",
		},
		{
			name:    "payload names another platform",
			payload: func(p *actionauthor.SaveRequest) { p.Platform = actionauthor.PlatformIOS },
			code:    errorcodes.RecordNotAccessible,
			message: "save request does not match the session",
		},
		{
			name: "unsigned session",
			lookup: func(l *sessionLookup) {
				l.history = sessionHistory(t, map[string]any{
					"device_id":                 "tenant/runner-1/device-1",
					"platform":                  actionauthor.PlatformAndroid,
					"wallet_version_identifier": "tenant/wallet/v1",
					"actor_id":                  "user-1",
				})
			},
			code:    errorcodes.RecordNotAccessible,
			message: "action author session was not started by Credimi",
		},
		{
			name: "grant for another session",
			lookup: func(l *sessionLookup) {
				l.history = grantForAnotherSession(t, app)
			},
			code:    errorcodes.RecordNotAccessible,
			message: "action author session was not started by Credimi",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lookup := validSessionLookup(t, app)
			if tc.lookup != nil {
				tc.lookup(&lookup)
			}
			caller := authorTestCaller()
			if tc.caller != nil {
				tc.caller(&caller)
			}
			payload := actionauthor.SaveRequest{BuildRequest: authorTestSession()}
			if tc.payload != nil {
				tc.payload(&payload)
			}

			_, err := lookup.activity(t, app).save(t.Context(), caller, payload)

			requireActivityError(t, err, tc.code, !tc.retryable)
			require.ErrorContains(t, err, tc.message)
		})
	}
}

type failingHistoryIterator struct{ err error }

func (f *failingHistoryIterator) HasNext() bool { return true }

func (f *failingHistoryIterator) Next() (*historypb.HistoryEvent, error) {
	return nil, f.err
}

// grantForAnotherSession is the history of authorTestSession's start input
// carrying Credimi's grant of another session.
func grantForAnotherSession(t *testing.T, app core.App) *fcafHistoryIterator {
	t.Helper()
	session := authorTestSession()
	input := map[string]any{
		"device_id":                 "tenant/runner-1/device-1",
		"platform":                  session.Platform,
		"wallet_version_identifier": session.WalletVersionIdentifier,
		"actor_id":                  session.ActorID,
		"organization":              authorTestOrganization,
	}
	grant, err := actionauthor.SignSession(app, actionauthor.SessionGrant{
		SessionWorkflowID:       "ActionAuthorSession-other",
		Namespace:               authorTestNamespace,
		Organization:            authorTestOrganization,
		DeviceID:                "tenant/runner-1/device-1",
		Platform:                session.Platform,
		WalletVersionIdentifier: session.WalletVersionIdentifier,
		ActorID:                 session.ActorID,
	})
	require.NoError(t, err)
	input["grant"] = grant
	return sessionHistory(t, input)
}

// authorSaveFixture seeds an Android build of userA's organization and
// returns the session that authors it.
func authorSaveFixture(t *testing.T) (core.App, actionauthor.BuildRequest, string) {
	t.Helper()
	app := newCredimiTestApp(t)
	actor, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	org, err := app.FindFirstRecordByData("organizations", "name", "userA's organization")
	require.NoError(t, err)
	wallets, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(wallets)
	wallet.Set("owner", org.Id)
	wallet.Set("name", "Authoring activity wallet")
	require.NoError(t, app.Save(wallet))
	versions, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)
	version := core.NewRecord(versions)
	version.Set("owner", org.Id)
	version.Set("wallet", wallet.Id)
	version.Set("tag", "build-1")
	apk, err := filesystem.NewFileFromBytes([]byte("selected APK bytes"), "selected.apk")
	require.NoError(t, err)
	version.Set("android_installer", []*filesystem.File{apk})
	require.NoError(t, app.Save(version))
	actions, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	if actions.Fields.GetByName("authoring_provenance") == nil {
		actions.Fields.Add(
			&core.JSONField{Name: "authoring_provenance", Hidden: true, MaxSize: 65536},
		)
		require.NoError(t, app.Save(actions))
	}
	return app, actionauthor.BuildRequest{
		Actor:    actionauthor.Actor{ActorID: actor.Id},
		Platform: actionauthor.PlatformAndroid,
		WalletVersionIdentifier: strings.Join([]string{
			org.GetString("canonified_name"),
			wallet.GetString("canonified_name"),
			version.GetString("canonified_tag"),
		}, "/"),
	}, org.GetString("canonified_name")
}

func TestSaveAuthoredWalletActionSavesForTheSessionActor(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	build, err := actionauthor.Resolve(app, namespace, session)
	require.NoError(t, err)
	code := "appId: eu.wallet\n---\n- launchApp\n- assertVisible: Home\n- takeScreenshot: home\n"
	// Each attempt reads the session's history afresh.
	a := func() *SaveAuthoredWalletActionActivity {
		lookup := unsignedSessionLookup(app)
		lookup.history = signedSessionHistory(t, app, map[string]any{
			"platform":                  session.Platform,
			"wallet_version_identifier": session.WalletVersionIdentifier,
			"actor_id":                  session.ActorID,
		})
		return lookup.activity(t, app)
	}
	caller := authorTestCaller()
	approval := func(edit func(*actionauthor.Approval)) string {
		approval := actionauthor.Approval{
			AuthorWorkflowID: caller.WorkflowID,
			CandidateSHA256:  actionauthor.CodeSHA256(code),
			ApproverID:       session.ActorID,
			ApprovedAt:       time.Now().Unix(),
			PlanSHA256: actionauthor.PlanSHA256(actionauthor.Spec{
				Slot:    "other",
				Outcome: actionauthor.OutcomeSuccess,
			}),
		}
		if edit != nil {
			edit(&approval)
		}
		token, err := actionauthor.SignApproval(app, approval)
		require.NoError(t, err)
		return token
	}
	// The payload omits the session's actor and build: the session supplies them.
	payload := actionauthor.SaveRequest{
		Name:            "Reviewed onboarding",
		Code:            code,
		CandidateSHA256: actionauthor.CodeSHA256(code),
		InstallerSHA256: build.InstallerSHA256,
		PackageID:       "eu.wallet",
		ReplayCount:     1,
		Approval:        approval(nil),
		Spec:            actionauthor.Spec{Slot: "other"},
	}

	for name, token := range map[string]string{
		"missing approval": "",
		"forged approval":  "forged.x",
		"another approver": approval(func(a *actionauthor.Approval) { a.ApproverID = "someone-else" }),
		"expired approval": approval(func(a *actionauthor.Approval) {
			a.ApprovedAt = time.Now().Add(-time.Hour).Unix()
		}),
		"another plan": approval(func(a *actionauthor.Approval) {
			a.PlanSHA256 = actionauthor.PlanSHA256(actionauthor.Spec{Slot: "other"})
		}),
	} {
		rejected := payload
		rejected.Approval = token
		_, err := a().save(t.Context(), caller, rejected)
		requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
		require.ErrorContains(t, err, actionauthor.ErrForbidden.Error(), name)
	}

	disallowed := payload
	disallowed.Code = "appId: eu.wallet\n---\n- runScript: x.js\n- assertVisible: Home\n- takeScreenshot: home\n"
	disallowed.CandidateSHA256 = actionauthor.CodeSHA256(disallowed.Code)
	disallowed.Approval = approval(func(a *actionauthor.Approval) {
		a.CandidateSHA256 = disallowed.CandidateSHA256
	})
	_, err = a().save(t.Context(), caller, disallowed)
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "runScript")

	receipt, err := a().save(t.Context(), caller, payload)

	require.NoError(t, err)
	require.True(t, receipt.Created)
	require.Equal(t, authorTestAuthorID, receipt.Provenance.WorkflowID)
	require.Equal(t, authorTestAuthorRun, receipt.Provenance.RunID)
	require.Equal(t, session.WalletVersionIdentifier, receipt.Provenance.WalletVersionIdentifier)
	require.Equal(t, "other", receipt.Provenance.Slot)
	require.Equal(t, session.ActorID, receipt.Provenance.ApproverID)
	record, err := app.FindRecordById("wallet_actions", receipt.RecordID)
	require.NoError(t, err)
	require.Equal(t, "other", record.GetString("category"))

	// A retry carrying the session's own values returns the same record.
	payload.BuildRequest = session
	retried, err := a().save(t.Context(), caller, payload)
	require.NoError(t, err)
	require.Equal(t, receipt.RecordID, retried.RecordID)

	// Save failures keep their coded classification.
	payload.InstallerSHA256 = strings.Repeat("0", 64)
	_, err = a().save(t.Context(), caller, payload)
	requireActivityError(t, err, errorcodes.DatabaseOperationFailed, true)
	require.ErrorContains(t, err, "selected installer changed")
}

func TestSaveAuthoredWalletActionExecuteUsesActivityCaller(t *testing.T) {
	a := NewSaveAuthoredWalletActionActivity(nil)
	a.temporalClient = func(string) (client.Client, error) {
		t.Fatal("a caller that is not an author must not be looked up")
		return nil, nil
	}

	// The test environment's caller is a workflow of another type.
	_, err := executeActivity(t, a, actionauthor.SaveRequest{Name: "Reviewed onboarding"})
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "caller is not an action author session agent")

	_, err = executeActivity(t, a, "not a request")
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
}

func TestResolveActionAuthorBuildAcceptsOnlyCredimiSignedSessions(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	input := map[string]any{
		"device_id":                 namespace + "/runner-1/device-1",
		"platform":                  session.Platform,
		"wallet_version_identifier": session.WalletVersionIdentifier,
		"actor_id":                  session.ActorID,
	}
	sessionCaller := authorCaller{
		Namespace:    authorTestNamespace,
		WorkflowType: ActionAuthorSessionWorkflowName,
		WorkflowID:   authorTestSessionID,
		RunID:        authorTestParentRun,
	}
	resolver := func(history *fcafHistoryIterator) *ResolveActionAuthorBuildActivity {
		lookup := validSessionLookup(t, app)
		lookup.history = history
		a := NewResolveActionAuthorBuildActivity(app)
		a.authorSessionLookup = lookup.lookup()
		return a
	}

	build, err := resolver(signedSessionHistory(t, app, input)).
		resolve(t.Context(), sessionCaller, session)
	require.NoError(t, err)
	require.Equal(t, session.WalletVersionIdentifier, build.VersionIdentifier)

	for name, tc := range map[string]struct {
		caller  func(*authorCaller)
		history *fcafHistoryIterator
		payload func(*actionauthor.BuildRequest)
		message string
	}{
		"caller of another type": {
			caller:  func(c *authorCaller) { c.WorkflowType = ActionAuthorWorkflowType },
			message: "caller is not an action author session",
		},
		"caller without session prefix": {
			caller:  func(c *authorCaller) { c.WorkflowID = "Pipeline-1" },
			message: "caller is not an action author session",
		},
		"caller without namespace": {
			caller:  func(c *authorCaller) { c.Namespace = "" },
			message: "caller is not an action author session",
		},
		"unsigned session": {
			history: sessionHistory(t, input),
			message: "action author session was not started by Credimi",
		},
		"payload of another actor": {
			payload: func(p *actionauthor.BuildRequest) { p.ActorID = "someone-else" },
			message: "build request does not match the session",
		},
		"payload of another platform": {
			payload: func(p *actionauthor.BuildRequest) { p.Platform = actionauthor.PlatformIOS },
			message: "build request does not match the session",
		},
		"payload of another wallet version": {
			payload: func(p *actionauthor.BuildRequest) {
				p.WalletVersionIdentifier = namespace + "/wallet/other"
			},
			message: "build request does not match the session",
		},
	} {
		t.Run(name, func(t *testing.T) {
			caller := sessionCaller
			if tc.caller != nil {
				tc.caller(&caller)
			}
			history := tc.history
			if history == nil {
				history = signedSessionHistory(t, app, input)
			}
			payload := session
			if tc.payload != nil {
				tc.payload(&payload)
			}
			_, err := resolver(history).resolve(t.Context(), caller, payload)
			requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
			require.ErrorContains(t, err, tc.message)
		})
	}

	// The test environment's caller is not a session workflow.
	a := NewResolveActionAuthorBuildActivity(app)
	a.temporalClient = func(string) (client.Client, error) {
		t.Fatal("a caller that is not a session must not be looked up")
		return nil, nil
	}
	_, err = executeActivity(t, a, session)
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "caller is not an action author session")
}

// A grant copied from a session's history does not open another run of the
// same workflow ID: a replay after the session closed, or a restart after it
// was terminated.
func TestLookupSessionStartAcceptsOnlyTheRecordedRun(t *testing.T) {
	setFCAFTestEnv(t)
	app := newCredimiTestApp(t)
	session := authorTestSession()
	input := map[string]any{
		"device_id":                 "tenant/runner-1/device-1",
		"platform":                  session.Platform,
		"wallet_version_identifier": session.WalletVersionIdentifier,
		"actor_id":                  session.ActorID,
	}
	lookup := func(namespace, runID string) (ActionAuthorSessionRef, error) {
		mockClient := &temporalmocks.Client{}
		mockClient.On("DescribeWorkflowExecution", mock.Anything, authorTestSessionID, runID).
			Return(sessionDescription(), nil)
		mockClient.On(
			"GetWorkflowHistory",
			mock.Anything,
			authorTestSessionID,
			runID,
			false,
			enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
		).Return(signedSessionHistory(t, app, input)).Maybe()
		return lookupSessionStart(
			t.Context(),
			app,
			mockClient,
			namespace,
			authorTestSessionID,
			runID,
		)
	}

	ref, err := lookup(authorTestNamespace, authorTestParentRun)
	require.NoError(t, err)
	require.Equal(t, authorTestParentRun, ref.SessionRunID)
	require.Equal(t, session, ref.Session)

	_, err = lookup(authorTestNamespace, "replayed-run")
	require.ErrorIs(t, err, actionauthor.ErrForbidden)
	require.ErrorContains(t, err, "action author session was not started by Credimi")

	// A session recorded in another namespace is refused.
	_, err = lookup("runner-other", authorTestParentRun)
	require.ErrorIs(t, err, actionauthor.ErrForbidden)
	require.ErrorContains(t, err, "action author session was not started by Credimi")

	// The session's first activity can run before its start was recorded.
	_, err = app.DB().NewQuery("DELETE FROM action_author_sessions").Execute()
	require.NoError(t, err)
	mockClient := &temporalmocks.Client{}
	mockClient.On("DescribeWorkflowExecution", mock.Anything, authorTestSessionID, authorTestParentRun).
		Return(sessionDescription(), nil)
	_, err = lookupSessionStart(
		t.Context(),
		app,
		mockClient,
		authorTestNamespace,
		authorTestSessionID,
		authorTestParentRun,
	)
	require.ErrorIs(t, err, actionauthor.ErrSessionNotRecorded)
	require.NotErrorIs(t, err, actionauthor.ErrForbidden)

	a := NewResolveActionAuthorBuildActivity(app)
	a.authorSessionLookup = validSessionLookup(t, app).lookup()
	_, err = app.DB().NewQuery("DELETE FROM action_author_sessions").Execute()
	require.NoError(t, err)
	_, err = a.resolve(t.Context(), authorCaller{
		Namespace:    authorTestNamespace,
		WorkflowType: ActionAuthorSessionWorkflowName,
		WorkflowID:   authorTestSessionID,
		RunID:        authorTestParentRun,
	}, session)
	requireActivityError(t, err, errorcodes.UnexpectedWorkflowError, false)
	require.ErrorContains(t, err, "action author session is not recorded yet")
}

// The recorded row's actor must be the start input's: a row of another
// user, or one without an actor, refuses the session.
func TestLookupSessionStartRequiresTheRecordedActor(t *testing.T) {
	setFCAFTestEnv(t)
	session := authorTestSession()
	input := map[string]any{
		"device_id":                 "tenant/runner-1/device-1",
		"platform":                  session.Platform,
		"wallet_version_identifier": session.WalletVersionIdentifier,
		"actor_id":                  session.ActorID,
	}
	require.Equal(t, "user-1", session.ActorID)
	for name, recordedActor := range map[string]string{
		"another user": "user-2",
		"no actor":     "",
	} {
		t.Run(name, func(t *testing.T) {
			app := newCredimiTestApp(t)
			// Recorded first: signedSessionHistory records the same run again,
			// which keeps this row.
			require.NoError(t, actionauthor.RecordSessionRun(
				app,
				authorTestNamespace,
				authorTestSessionID,
				authorTestParentRun,
				actionauthor.SessionDetails{ActorID: recordedActor},
			))
			mockClient := &temporalmocks.Client{}
			mockClient.On(
				"DescribeWorkflowExecution",
				mock.Anything,
				authorTestSessionID,
				authorTestParentRun,
			).Return(sessionDescription(), nil)
			mockClient.On(
				"GetWorkflowHistory",
				mock.Anything,
				authorTestSessionID,
				authorTestParentRun,
				false,
				enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
			).Return(signedSessionHistory(t, app, input))

			_, err := lookupSessionStart(
				t.Context(),
				app,
				mockClient,
				authorTestNamespace,
				authorTestSessionID,
				authorTestParentRun,
			)
			require.ErrorIs(t, err, actionauthor.ErrForbidden)
			require.ErrorContains(t, err, "action author session was not started by Credimi")
		})
	}
}

func TestReadActionAuthorSessionReadsARecordedSessionInAnyState(t *testing.T) {
	setFCAFTestEnv(t)
	app := newCredimiTestApp(t)
	session := authorTestSession()
	history := signedSessionHistory(t, app, map[string]any{
		"device_id":                 "tenant/runner-1/device-1",
		"platform":                  session.Platform,
		"wallet_version_identifier": session.WalletVersionIdentifier,
		"actor_id":                  session.ActorID,
	})
	started := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	closed := started.Add(time.Hour)
	completed := describeExecution(
		ActionAuthorSessionWorkflowName,
		enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED,
		nil,
	)
	completed.WorkflowExecutionInfo.StartTime = timestamppb.New(started)
	completed.WorkflowExecutionInfo.CloseTime = timestamppb.New(closed)
	mockClient := &temporalmocks.Client{}
	mockClient.On("DescribeWorkflowExecution", mock.Anything, authorTestSessionID, authorTestParentRun).
		Return(completed, nil)
	mockClient.On(
		"GetWorkflowHistory",
		mock.Anything,
		authorTestSessionID,
		authorTestParentRun,
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(history)

	var clientNamespaces []string
	clientFor := func(c client.Client) func(string) (client.Client, error) {
		return func(namespace string) (client.Client, error) {
			clientNamespaces = append(clientNamespaces, namespace)
			return c, nil
		}
	}
	info, err := ReadActionAuthorSession(
		t.Context(),
		app,
		clientFor(mockClient),
		authorTestOrganization,
		authorTestSessionID,
		session.ActorID,
	)
	require.NoError(t, err)
	// The session is read in the namespace it was recorded in.
	require.Equal(t, []string{authorTestNamespace}, clientNamespaces)
	require.Equal(t, authorTestNamespace, info.Namespace)
	require.Equal(t, authorTestOrganization, info.Organization)
	require.Equal(t, authorTestSessionID, info.SessionWorkflowID)
	require.Equal(t, authorTestParentRun, info.SessionRunID)
	require.Equal(t, session, info.Session)
	require.Equal(t, "tenant/runner-1/device-1", info.DeviceID)
	require.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED, info.Status)
	require.Equal(t, started, info.StartTime)
	require.Equal(t, closed, info.CloseTime)

	// Another user, or a member of another organization, gets the answer of
	// an unknown session, before any Temporal call.
	clientNamespaces = nil
	for _, caller := range [][2]string{
		{authorTestOrganization, "another-user"},
		{"other-org", session.ActorID},
		{authorTestNamespace, session.ActorID},
	} {
		_, err = ReadActionAuthorSession(
			t.Context(),
			app,
			clientFor(&temporalmocks.Client{}),
			caller[0],
			authorTestSessionID,
			caller[1],
		)
		require.ErrorIs(t, err, actionauthor.ErrForbidden, caller)
		require.ErrorContains(t, err, "action author session belongs to another user", caller)
	}
	require.Empty(t, clientNamespaces)
}

func TestReadActionAuthorSessionRefusesUnrecordedAndOtherWorkflows(t *testing.T) {
	app := newCredimiTestApp(t)
	mockClient := &temporalmocks.Client{}

	clientFor := func(string) (client.Client, error) { return mockClient, nil }

	_, err := ReadActionAuthorSession(
		t.Context(),
		app,
		clientFor,
		authorTestOrganization,
		authorTestSessionID,
		"user-1",
	)
	require.ErrorIs(t, err, actionauthor.ErrSessionNotRecorded)

	require.NoError(t, actionauthor.RecordSessionRun(
		app,
		authorTestNamespace,
		authorTestSessionID,
		authorTestParentRun,
		actionauthor.SessionDetails{ActorID: "user-1", Organization: authorTestOrganization},
	))
	mockClient.On("DescribeWorkflowExecution", mock.Anything, authorTestSessionID, authorTestParentRun).
		Return(describeExecution(
			"Dynamic Pipeline Workflow",
			enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
			nil,
		), nil)
	_, err = ReadActionAuthorSession(
		t.Context(),
		app,
		clientFor,
		authorTestOrganization,
		authorTestSessionID,
		"user-1",
	)
	require.ErrorIs(t, err, actionauthor.ErrForbidden)
	require.ErrorContains(t, err, "workflow is not an action author session")
}
