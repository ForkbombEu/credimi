// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"errors"
	"strings"
	"testing"

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
)

const (
	authorTestNamespace = "tenant"
	authorTestSessionID = "ActionAuthorSession-ticket-1"
	authorTestAuthorID  = authorTestSessionID + "-author"
	authorTestAuthorRun = "author-run"
	authorTestParentRun = "session-run"
)

func authorTestCaller() saveCaller {
	return saveCaller{
		Namespace:    authorTestNamespace,
		WorkflowType: ActionAuthorWorkflowType,
		WorkflowID:   authorTestAuthorID,
		RunID:        authorTestAuthorRun,
	}
}

func authorTestSession() actionauthor.BuildRequest {
	return actionauthor.BuildRequest{
		Actor:                   actionauthor.Actor{ActorID: "user-1", ActorCollection: "users"},
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

// sessionLookup fakes the caller's namespace with the given executions.
type sessionLookup struct {
	clientErr  error
	author     *workflowservice.DescribeWorkflowExecutionResponse
	authorErr  error
	session    *workflowservice.DescribeWorkflowExecutionResponse
	sessionErr error
	history    client.HistoryEventIterator
}

func validSessionLookup(t *testing.T) sessionLookup {
	session := authorTestSession()
	return sessionLookup{
		author:  authorDescription(),
		session: sessionDescription(),
		history: sessionHistory(t, map[string]any{
			"device_id":                 "tenant/runner-1/device-1",
			"platform":                  session.Platform,
			"wallet_version_identifier": session.WalletVersionIdentifier,
			"actor_id":                  session.ActorID,
			"actor_collection":          session.ActorCollection,
			"slot":                      "onboarding",
		}),
	}
}

func (l sessionLookup) activity(t *testing.T, app core.App) *SaveAuthoredWalletActionActivity {
	t.Helper()
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
	a := NewSaveAuthoredWalletActionActivity(app)
	a.temporalClient = func(string) (client.Client, error) {
		if l.clientErr != nil {
			return nil, l.clientErr
		}
		return mockClient, nil
	}
	return a
}

func TestSaveAuthoredWalletActionRejectsCallersOutsideARunningSession(t *testing.T) {
	setFCAFTestEnv(t)
	notFound := serviceerror.NewNotFound("not found")
	tests := []struct {
		name      string
		caller    func(*saveCaller)
		lookup    func(*sessionLookup)
		payload   func(*actionauthor.SaveRequest)
		code      string
		retryable bool
		message   string
	}{
		{
			name:    "missing namespace",
			caller:  func(c *saveCaller) { c.Namespace = "" },
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not an action author session agent",
		},
		{
			name:    "another workflow type",
			caller:  func(c *saveCaller) { c.WorkflowType = "Some pipeline" },
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not an action author session agent",
		},
		{
			name:    "workflow ID without author suffix",
			caller:  func(c *saveCaller) { c.WorkflowID = authorTestSessionID },
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not an action author session agent",
		},
		{
			name:    "workflow ID without session prefix",
			caller:  func(c *saveCaller) { c.WorkflowID = "Pipeline-1-author" },
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not an action author session agent",
		},
		{
			name:    "workflow ID without ticket",
			caller:  func(c *saveCaller) { c.WorkflowID = ActionAuthorSessionWorkflowIDPrefix + "-author" },
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
			name:    "payload names another collection",
			payload: func(p *actionauthor.SaveRequest) { p.ActorCollection = "_superusers" },
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
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lookup := validSessionLookup(t)
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

			_, err := lookup.activity(t, nil).save(t.Context(), caller, payload)

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
		Actor:    actionauthor.Actor{ActorID: actor.Id, ActorCollection: "users"},
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
	code := "appId: eu.wallet\n---\n- launchApp\n"
	// Each attempt reads the session's history afresh.
	a := func() *SaveAuthoredWalletActionActivity {
		lookup := validSessionLookup(t)
		lookup.history = sessionHistory(t, map[string]any{
			"platform":                  session.Platform,
			"wallet_version_identifier": session.WalletVersionIdentifier,
			"actor_id":                  session.ActorID,
			"actor_collection":          session.ActorCollection,
		})
		return lookup.activity(t, app)
	}
	caller := authorTestCaller()
	caller.Namespace = namespace
	// The payload omits the session's actor and build: the session supplies them.
	payload := actionauthor.SaveRequest{
		Name:            "Reviewed onboarding",
		Code:            code,
		CandidateSHA256: actionauthor.CodeSHA256(code),
		InstallerSHA256: build.InstallerSHA256,
		PackageID:       "eu.wallet",
		Slot:            "onboarding",
		ReplayCount:     2,
	}

	receipt, err := a().save(t.Context(), caller, payload)

	require.NoError(t, err)
	require.True(t, receipt.Created)
	require.Equal(t, authorTestAuthorID, receipt.Provenance.WorkflowID)
	require.Equal(t, authorTestAuthorRun, receipt.Provenance.RunID)
	require.Equal(t, session.WalletVersionIdentifier, receipt.Provenance.WalletVersionIdentifier)

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
