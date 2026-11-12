// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
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
	authoringSessionID     = "ActionAuthorSession-testrunner00001-0f8fad5b-d9cb-469f-a165-70867728950e"
	authoringSessionChat   = authoringSessionID + "-author"
	authoringSessionRun    = "session-run"
	authoringSessionDevice = authoringAdminRunner + "/android-emulator"
	// authoringSessionNamespace is the runner namespace sessions run in.
	authoringSessionNamespace = "runner-testrunner00001"
)

var authoringSessionStart = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// jsonEncodedValue answers a workflow query with value.
type jsonEncodedValue struct{ value any }

func (v jsonEncodedValue) HasValue() bool { return v.value != nil }

func (v jsonEncodedValue) Get(valuePtr any) error {
	encoded, err := json.Marshal(v.value)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, valuePtr)
}

// authoringSessionGrant is the start input of a session as the start
// activity signs it: Credimi's grant binds it to sessionID, the namespace
// its ID names (or namespace, when set), and the rest of the input.
type authoringSessionGrant struct {
	sessionID, organization, device, version, actorID string
	namespace                                         string
}

// history returns the history of the session's run: its start input, and
// when signed is true Credimi's grant in it and Credimi's run grant signal
// for run.
func (g authoringSessionGrant) history(
	t testing.TB,
	app *tests.TestApp,
	run string,
	signed bool,
) client.HistoryEventIterator {
	t.Helper()
	namespace := g.namespace
	if namespace == "" {
		var ok bool
		namespace, ok = actionauthor.SessionNamespace(g.sessionID)
		require.True(t, ok, g.sessionID)
	}
	payload := map[string]any{
		"device_id":                 g.device,
		"platform":                  actionauthor.PlatformAndroid,
		"wallet_version_identifier": g.version,
		"actor_id":                  g.actorID,
		"namespace":                 namespace,
		"organization":              g.organization,
	}
	if signed {
		grant, err := actionauthor.SignSession(app, actionauthor.SessionGrant{
			SessionWorkflowID:       g.sessionID,
			Namespace:               namespace,
			Organization:            g.organization,
			DeviceID:                g.device,
			Platform:                actionauthor.PlatformAndroid,
			WalletVersionIdentifier: g.version,
			ActorID:                 g.actorID,
		})
		require.NoError(t, err)
		payload["grant"] = grant
	}
	input, err := temporalcrypto.DataConverter().ToPayloads(workflowengine.WorkflowInput{
		Payload: payload,
	})
	require.NoError(t, err)
	events := []*historypb.HistoryEvent{{
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
			WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
				Input: input,
			},
		},
	}}
	if signed {
		events = append(events, runGrantSignal(t, app, g.sessionID, run))
	}
	return &fakeHistoryIterator{events: events}
}

// runGrantSignal is the run grant signal the start activity sends to the run
// of sessionID.
func runGrantSignal(
	t testing.TB,
	app *tests.TestApp,
	sessionID, run string,
) *historypb.HistoryEvent {
	t.Helper()
	token, err := actionauthor.SignRun(app, actionauthor.RunGrant{
		SessionWorkflowID: sessionID,
		RunID:             run,
	})
	require.NoError(t, err)
	input, err := temporalcrypto.DataConverter().ToPayloads(token)
	require.NoError(t, err)
	return &historypb.HistoryEvent{
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionSignaledEventAttributes{
			WorkflowExecutionSignaledEventAttributes: &historypb.WorkflowExecutionSignaledEventAttributes{
				SignalName: activities.ActionAuthorRunGrantSignal,
				Input:      input,
			},
		},
	}
}

// startAuthoringSession fakes, as the start activity leaves it in Temporal,
// the session authoringSessionID that userA started for organization on the
// admin runner's emulator: its run authoringSessionRun, listed as its only
// run, and its signed start input.
func startAuthoringSession(
	t testing.TB,
	app *tests.TestApp,
	c *temporalmocks.Client,
	organization string,
) {
	t.Helper()
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	stubSessionRuns(t, c, authoringSessionID, authoringSessionRun)
	c.On(
		"GetWorkflowHistory",
		mock.Anything,
		authoringSessionID,
		authoringSessionRun,
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(authoringSessionGrant{
		sessionID:    authoringSessionID,
		organization: organization,
		device:       authoringSessionDevice,
		version:      authoringAndroidVersion,
		actorID:      user.Id,
	}.history(t, app, authoringSessionRun, true))
}

// sessionRunExecutions are runs (oldest first) of sessionID of workflowType,
// newest first as Temporal's visibility lists them.
func sessionRunExecutions(
	sessionID, workflowType string,
	runs ...string,
) []*workflowpb.WorkflowExecutionInfo {
	executions := make([]*workflowpb.WorkflowExecutionInfo, 0, len(runs))
	for i := len(runs) - 1; i >= 0; i-- {
		executions = append(executions, &workflowpb.WorkflowExecutionInfo{
			Execution: &commonpb.WorkflowExecution{WorkflowId: sessionID, RunId: runs[i]},
			Type:      &commonpb.WorkflowType{Name: workflowType},
			StartTime: timestamppb.New(authoringSessionStart.Add(time.Duration(i) * time.Minute)),
		})
	}
	return executions
}

// stubSessionWorkflowRuns makes c list executions as the runs of sessionID.
func stubSessionWorkflowRuns(
	c *temporalmocks.Client,
	sessionID string,
	executions []*workflowpb.WorkflowExecutionInfo,
) {
	c.On("ListWorkflow", mock.Anything, mock.MatchedBy(
		func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
			return req.GetQuery() == fmt.Sprintf("WorkflowId = %q", sessionID)
		},
	)).Return(&workflowservice.ListWorkflowExecutionsResponse{Executions: executions}, nil).Maybe()
}

// stubSessionRuns makes c list runs (oldest first) as the runs of sessionID,
// newest first as Temporal's visibility lists them.
func stubSessionRuns(t testing.TB, c *temporalmocks.Client, sessionID string, runs ...string) {
	t.Helper()
	stubSessionWorkflowRuns(
		c,
		sessionID,
		sessionRunExecutions(sessionID, activities.ActionAuthorSessionWorkflowName, runs...),
	)
}

func authoringExecution(
	workflowType string,
	status enumspb.WorkflowExecutionStatus,
	historyLength int64,
) *workflowservice.DescribeWorkflowExecutionResponse {
	info := &workflowpb.WorkflowExecutionInfo{
		Type:          &commonpb.WorkflowType{Name: workflowType},
		Status:        status,
		StartTime:     timestamppb.New(authoringSessionStart),
		HistoryLength: historyLength,
	}
	if status != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		info.CloseTime = timestamppb.New(authoringSessionStart.Add(time.Hour))
	}
	return &workflowservice.DescribeWorkflowExecutionResponse{WorkflowExecutionInfo: info}
}

// stubAuthoringSessionTemporal answers every namespace with c and records the
// namespaces asked for.
func stubAuthoringSessionTemporal(t testing.TB, c client.Client) *[]string {
	t.Helper()
	setAuthoringSessionKey(t)
	namespaces := []string{}
	orig := actionAuthorSessionTemporalClient
	t.Cleanup(func() { actionAuthorSessionTemporalClient = orig })
	actionAuthorSessionTemporalClient = func(namespace string) (client.Client, error) {
		namespaces = append(namespaces, namespace)
		return c, nil
	}
	return &namespaces
}

func setAuthoringSessionKey(t testing.TB) {
	t.Helper()
	t.Setenv(
		temporalcrypto.SecretsEncryptionKeyEnv,
		"MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA=",
	)
}

// runningAuthoringSession fakes the session userA started for their
// organization, in status.
func runningAuthoringSession(
	app *tests.TestApp,
	t testing.TB,
	c *temporalmocks.Client,
	status enumspb.WorkflowExecutionStatus,
) {
	c.On("DescribeWorkflowExecution", mock.Anything, authoringSessionID, authoringSessionRun).
		Return(authoringExecution(activities.ActionAuthorSessionWorkflowName, status, 0), nil)
	startAuthoringSession(t, app, c, authoringNamespace)
}

func decodeBody[T any](t testing.TB, res *http.Response, value *T) {
	t.Helper()
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, value), string(raw))
}

func TestActionAuthorSessionReadsTheRunningChat(t *testing.T) {
	c := &temporalmocks.Client{}
	namespaces := stubAuthoringSessionTemporal(t, c)
	c.On("DescribeWorkflowExecution", mock.Anything, authoringSessionChat, "").
		Return(authoringExecution(
			activities.ActionAuthorWorkflowType,
			enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
			5,
		), nil)
	c.On("QueryWorkflow", mock.Anything, authoringSessionChat, "", "idle_state").
		Return(jsonEncodedValue{map[string]any{"closes_at": "2026-01-02T04:00:00Z"}}, nil)
	c.On("QueryWorkflow", mock.Anything, authoringSessionChat, "", "agent_status").
		Return(jsonEncodedValue{map[string]any{"pending_callbacks": []map[string]any{
			{"tool_id": "other", "tool_name": "ask_user", "tool_input": map[string]any{}},
			{
				"tool_id":   "tool-1",
				"tool_name": "confirm_wallet_action_save",
				"tool_input": map[string]any{
					"name":             "install",
					"build":            authoringAndroidVersion,
					"code":             "appId: eu.wallet",
					"candidate_sha256": "abc",
					"plan":             map[string]any{"slot": "install-app"},
					"replaces":         "usera-s-organization/authoring-handler-wallet/install",
					"error":            "save failed: try again",
				},
			},
		}}}, nil)
	var response ActionAuthorSessionResponse
	(&tests.ApiScenario{
		Name:           "running session",
		Method:         http.MethodGet,
		URL:            "/api/action-author/sessions/" + authoringSessionID,
		Headers:        map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"chat_ready":true`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			device, err := app.FindFirstRecordByFilter(
				"mobile_devices",
				"name = 'android-emulator' && runner.name = 'admin-runner'",
			)
			require.NoError(t, err)
			device.Set("live_stream", true)
			require.NoError(t, app.Save(device))
			runner, err := app.FindFirstRecordByData("mobile_runners", "name", "admin-runner")
			require.NoError(t, err)
			runner.Set("agent_provider", "anthropic")
			runner.Set("agent_model", "claude")
			require.NoError(t, app.Save(runner))
			runningAuthoringSession(app, t, c, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING)
			replaced := "usera-s-organization/authoring-handler-wallet/install"
			seedUsedByPipeline(t, app, "userA's organization", "Own user", replaced, false)
			seedUsedByPipeline(t, app, "userA's organization", "Own other", "acme/x/y", false)
			seedUsedByPipeline(t, app, "userB's organization", "Published user", replaced, true)
			seedUsedByPipeline(t, app, "userB's organization", "Private user", replaced, false)
			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
			decodeBody(t, res, &response)
		},
	}).Test(t)

	// The session and its chat are read in the runner namespace it runs in.
	require.Equal(
		t,
		[]string{authoringSessionNamespace},
		*namespaces,
	)
	require.Equal(t, ActionAuthorSessionResponse{
		SessionID:               authoringSessionID,
		RunID:                   authoringSessionRun,
		ChatWorkflowID:          authoringSessionChat,
		Status:                  "Running",
		StartTime:               "2026-01-02T03:04:05Z",
		WalletVersionIdentifier: authoringAndroidVersion,
		Platform:                actionauthor.PlatformAndroid,
		DeviceID:                authoringSessionDevice,
		DeviceName:              "android-emulator",
		RunnerName:              "admin-runner",
		RunnerOwner:             "userA's organization",
		Model:                   "anthropic / claude",
		LiveView:                true,
		ChatReady:               true,
		ChatRunning:             true,
		ClosesAt:                "2026-01-02T04:00:00Z",
		PendingReview: &ActionAuthorPendingReview{
			ToolID:          "tool-1",
			Name:            "install",
			Build:           authoringAndroidVersion,
			Code:            "appId: eu.wallet",
			CandidateSHA256: "abc",
			Plan:            map[string]any{"slot": "install-app"},
			Replaces:        "usera-s-organization/authoring-handler-wallet/install",
			Error:           "save failed: try again",
			UsedBy:          2,
		},
	}, response)
}

// seedUsedByPipeline adds a pipeline of the organization whose step runs action.
func seedUsedByPipeline(
	t testing.TB,
	app *tests.TestApp,
	organization, name, action string,
	published bool,
) {
	t.Helper()
	org, err := app.FindFirstRecordByData("organizations", "name", organization)
	require.NoError(t, err)
	collection, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("owner", org.Id)
	record.Set("name", name)
	record.Set("description", name)
	record.Set("yaml", "name: "+name+"\nsteps:\n  - id: run\n    use: mobile-automation\n"+
		"    with:\n      action_id: "+action+"\n")
	record.Set("published", published)
	require.NoError(t, app.Save(record))
}

// A completed chat is still shown, of a running session that releases its
// device as of a completed one, but its idle state and reviews are not read.
func TestActionAuthorSessionReadsACompletedChat(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    enumspb.WorkflowExecutionStatus
		closeTime string
	}{
		{name: "running session", status: enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING},
		{
			name:      "completed session",
			status:    enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			closeTime: "2026-01-02T04:04:05Z",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &temporalmocks.Client{}
			stubAuthoringSessionTemporal(t, c)
			c.On("DescribeWorkflowExecution", mock.Anything, authoringSessionChat, "").
				Return(authoringExecution(
					activities.ActionAuthorWorkflowType,
					enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					10,
				), nil)
			var response ActionAuthorSessionResponse
			(&tests.ApiScenario{
				Name:   tc.name,
				Method: http.MethodGet,
				URL:    "/api/action-author/sessions/" + authoringSessionID,
				Headers: map[string]string{
					"Authorization": "Bearer " + authoringToken(t, "userA"),
				},
				ExpectedStatus:  http.StatusOK,
				ExpectedContent: []string{`"chat_ready":true`, `"chat_running":false`},
				TestAppFactory: func(t testing.TB) *tests.TestApp {
					app := setupActionAuthorApp(t)
					runningAuthoringSession(app, t, c, tc.status)
					return app
				},
				AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
					decodeBody(t, res, &response)
				},
			}).Test(t)

			require.Equal(t, tc.closeTime, response.CloseTime)
			require.True(t, response.ChatReady)
			require.False(t, response.ChatRunning)
			require.False(t, response.LiveView)
			require.Empty(t, response.ClosesAt)
			require.Nil(t, response.PendingReview)
			c.AssertNotCalled(
				t,
				"QueryWorkflow",
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
			)
		})
	}
}

// A session that ended before its chat started has no chat to show.
func TestActionAuthorSessionReadsAClosedSessionWithoutItsChat(t *testing.T) {
	c := &temporalmocks.Client{}
	stubAuthoringSessionTemporal(t, c)
	c.On("DescribeWorkflowExecution", mock.Anything, authoringSessionChat, "").
		Return(nil, serviceerror.NewNotFound("no chat"))
	var response ActionAuthorSessionResponse
	(&tests.ApiScenario{
		Name:            "completed session",
		Method:          http.MethodGet,
		URL:             "/api/action-author/sessions/" + authoringSessionID,
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"status":"Completed"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			runningAuthoringSession(app, t, c, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED)
			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
			decodeBody(t, res, &response)
		},
	}).Test(t)

	require.Equal(t, "2026-01-02T04:04:05Z", response.CloseTime)
	require.False(t, response.ChatReady)
	require.False(t, response.ChatRunning)
	require.False(t, response.LiveView)
	require.Nil(t, response.PendingReview)
}

// Another member of the organization gets the answer of a session that does not exist.
func TestActionAuthorSessionIsHiddenFromAnotherMember(t *testing.T) {
	c := &temporalmocks.Client{}
	stubAuthoringSessionTemporal(t, c)
	(&tests.ApiScenario{
		Name:            "another member",
		Method:          http.MethodGet,
		URL:             "/api/action-author/sessions/" + authoringSessionID,
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userB")},
		ExpectedStatus:  http.StatusNotFound,
		ExpectedContent: []string{`"reason":"session_not_found"`},
		TestAppFactory: func(testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			joinAgentRunnerOrganization(t, app, "member")
			runningAuthoringSession(app, t, c, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED)
			return app
		},
	}).Test(t)

	c.AssertNotCalled(t, "DescribeWorkflowExecution", mock.Anything, mock.Anything, mock.Anything)
}

func TestActionAuthorSessionCloseRefusesAnotherMember(t *testing.T) {
	c := &temporalmocks.Client{}
	stubAuthoringSessionTemporal(t, c)
	signalled := false
	orig := signalChatClose
	t.Cleanup(func() { signalChatClose = orig })
	signalChatClose = func(context.Context, client.Client, string, string) error {
		signalled = true
		return nil
	}
	(&tests.ApiScenario{
		Name:            "another member",
		Method:          http.MethodPost,
		URL:             "/api/action-author/sessions/" + authoringSessionID + "/close",
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userB")},
		ExpectedStatus:  http.StatusNotFound,
		ExpectedContent: []string{`"reason":"session_not_found"`},
		TestAppFactory: func(testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			joinAgentRunnerOrganization(t, app, "member")
			runningAuthoringSession(app, t, c, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING)
			return app
		},
	}).Test(t)

	require.False(t, signalled)
	c.AssertNotCalled(t, "CancelWorkflow", mock.Anything, mock.Anything, mock.Anything)
}

// An unknown session and one another member started give the same answer.
func TestActionAuthorSessionAnswersNotFound(t *testing.T) {
	for _, tc := range []struct {
		name      string
		sessionID string
		// organization the session is started for; empty starts none.
		organization string
		// user reading the session; empty is userA, who starts it.
		user string
		// member makes user a member of userA's organization.
		member bool
	}{
		{name: "malformed id", sessionID: "ActionAuthorSession-ticket"},
		{name: "not visible in Temporal", sessionID: authoringSessionID},
		{
			name:         "started for another organization",
			sessionID:    authoringSessionID,
			organization: authoringForeignNS,
		},
		{
			name:         "started by another member",
			sessionID:    authoringSessionID,
			organization: authoringNamespace,
			user:         "userB",
			member:       true,
		},
		{
			name:         "read by a member of another organization",
			sessionID:    authoringSessionID,
			organization: authoringNamespace,
			user:         "userB",
		},
	} {
		c := &temporalmocks.Client{}
		stubAuthoringSessionTemporal(t, c)
		user := tc.user
		if user == "" {
			user = "userA"
		}
		(&tests.ApiScenario{
			Name:            tc.name,
			Method:          http.MethodGet,
			URL:             "/api/action-author/sessions/" + tc.sessionID,
			Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, user)},
			ExpectedStatus:  http.StatusNotFound,
			ExpectedContent: []string{`"reason":"session_not_found"`},
			TestAppFactory: func(testing.TB) *tests.TestApp {
				app := setupActionAuthorApp(t)
				if tc.member {
					joinAgentRunnerOrganization(t, app, "member")
				}
				if tc.organization != "" {
					startAuthoringSession(t, app, c, tc.organization)
				} else {
					stubSessionRuns(t, c, tc.sessionID)
				}
				return app
			},
		}).Test(t)
		c.AssertNotCalled(
			t,
			"DescribeWorkflowExecution",
			mock.Anything,
			mock.Anything,
			mock.Anything,
		)
	}
}

func TestActionAuthorSessionRefusesOtherWorkflows(t *testing.T) {
	c := &temporalmocks.Client{}
	stubAuthoringSessionTemporal(t, c)
	stubSessionWorkflowRuns(c, authoringSessionID, sessionRunExecutions(
		authoringSessionID,
		"Dynamic Pipeline Workflow",
		authoringSessionRun,
	))
	(&tests.ApiScenario{
		Name:            "pipeline under a session id",
		Method:          http.MethodGet,
		URL:             "/api/action-author/sessions/" + authoringSessionID,
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusNotFound,
		ExpectedContent: []string{`"reason":"session_not_found"`},
		TestAppFactory:  setupActionAuthorApp,
	}).Test(t)

	c.AssertNotCalled(t, "DescribeWorkflowExecution", mock.Anything, mock.Anything, mock.Anything)
	c.AssertNotCalled(
		t,
		"GetWorkflowHistory",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

// listedAuthoringSession is a session the wallet sessions query lists in a
// runner's agent namespace.
type listedAuthoringSession struct {
	authoringSessionGrant
	run     string
	started time.Time
	// signed signs the session's start input with Credimi's grant.
	signed bool
	// status is the described status of a session the handler keeps.
	status enumspb.WorkflowExecutionStatus
}

func (s listedAuthoringSession) execution() *workflowpb.WorkflowExecutionInfo {
	return &workflowpb.WorkflowExecutionInfo{
		Execution: &commonpb.WorkflowExecution{WorkflowId: s.sessionID, RunId: s.run},
		Type:      &commonpb.WorkflowType{Name: activities.ActionAuthorSessionWorkflowName},
		StartTime: timestamppb.New(s.started),
	}
}

// stub makes c answer the session's trusted run lookup: the listed run is
// its only run.
func (s listedAuthoringSession) stub(t testing.TB, app *tests.TestApp, c *temporalmocks.Client) {
	t.Helper()
	stubSessionWorkflowRuns(c, s.sessionID, []*workflowpb.WorkflowExecutionInfo{s.execution()})
	c.On(
		"GetWorkflowHistory",
		mock.Anything,
		s.sessionID,
		s.run,
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(s.history(t, app, s.run, s.signed))
	if s.status != enumspb.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED {
		described := authoringExecution(activities.ActionAuthorSessionWorkflowName, s.status, 0)
		described.WorkflowExecutionInfo.StartTime = timestamppb.New(s.started)
		c.On("DescribeWorkflowExecution", mock.Anything, s.sessionID, s.run).
			Return(described, nil)
	}
}

// The sessions of the agent namespaces of the runners offered to the caller's
// organization are merged newest first; listed runs that are not trusted
// runs of the caller's sessions are dropped.
func TestActionAuthorSessionsListsTheWalletSessions(t *testing.T) {
	c := &temporalmocks.Client{}
	namespaces := stubAuthoringSessionTemporal(t, c)
	const wallet = "usera-s-organization/authoring-handler-wallet"
	var adminNamespace, tenantNamespace string
	var oldest, newest string
	var response ActionAuthorSessionsResponse
	(&tests.ApiScenario{
		Name:            "wallet sessions",
		Method:          http.MethodGet,
		URL:             "/api/action-author/sessions?wallet=" + wallet,
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"sessions":[`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)
			other, err := app.FindAuthRecordByEmail("users", "userB@example.org")
			require.NoError(t, err)
			admin, err := app.FindFirstRecordByData("mobile_runners", "name", "admin-runner")
			require.NoError(t, err)
			tenant, err := app.FindFirstRecordByData("mobile_runners", "name", "tenant-runner")
			require.NoError(t, err)
			// Only runners whose agent namespace is ready are listed.
			require.NoError(t, mobilerunner.MarkAgentNamespaceReady(app, admin.Id))
			require.NoError(t, mobilerunner.MarkAgentNamespaceReady(app, tenant.Id))
			adminNamespace = mobilerunner.AgentNamespace(admin.Id)
			tenantNamespace = mobilerunner.AgentNamespace(tenant.Id)
			sessionID := func(runner *core.Record, ticket string) string {
				return actionauthor.SessionWorkflowID(runner.Id, ticket)
			}
			oldest = sessionID(admin, "00000000-0000-4000-8000-000000000001")
			newest = sessionID(tenant, "00000000-0000-4000-8000-000000000002")
			day := func(d int) time.Time { return time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC) }
			sessions := map[string][]listedAuthoringSession{
				adminNamespace: {
					// Search attributes copied onto a workflow Credimi did not sign.
					{
						authoringSessionGrant: authoringSessionGrant{
							sessionID:    sessionID(admin, "00000000-0000-4000-8000-000000000003"),
							organization: authoringNamespace,
							device:       authoringSessionDevice,
							version:      authoringAndroidVersion,
							actorID:      user.Id,
						},
						run:     "run-forged",
						started: day(4),
					},
					// Another user's session listed under the caller's actor.
					{
						authoringSessionGrant: authoringSessionGrant{
							sessionID:    sessionID(admin, "00000000-0000-4000-8000-000000000004"),
							organization: authoringNamespace,
							device:       authoringSessionDevice,
							version:      authoringAndroidVersion,
							actorID:      other.Id,
						},
						run:     "run-other",
						started: day(2),
						signed:  true,
					},
					{
						authoringSessionGrant: authoringSessionGrant{
							sessionID:    oldest,
							organization: authoringNamespace,
							device:       "usera-s-organization/removed-runner/phone",
							version:      authoringAndroidVersion,
							actorID:      user.Id,
						},
						run:     "run-oldest",
						started: day(1),
						signed:  true,
						status:  enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED,
					},
				},
				tenantNamespace: {{
					authoringSessionGrant: authoringSessionGrant{
						sessionID:    newest,
						organization: authoringNamespace,
						device:       authoringSessionDevice,
						version:      wallet + "/v2",
						actorID:      user.Id,
					},
					run:     "run-newest",
					started: day(3),
					signed:  true,
					status:  enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
				}},
			}
			query := fmt.Sprintf(
				`WorkflowType = "Action author session" and ActionAuthorActor = "%s" and `+
					`ActionAuthorWallet = "%s"`,
				user.Id,
				wallet,
			)
			for namespace, listed := range sessions {
				executions := []*workflowpb.WorkflowExecutionInfo{}
				for _, session := range listed {
					session.stub(t, app, c)
					executions = append(executions, session.execution())
				}
				c.On("ListWorkflow", mock.Anything, mock.MatchedBy(
					func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
						return req.GetNamespace() == namespace && req.GetQuery() == query &&
							req.GetPageSize() == actionAuthorSessionsPageSize
					},
				)).Return(
					&workflowservice.ListWorkflowExecutionsResponse{Executions: executions},
					nil,
				).Once()
			}
			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
			decodeBody(t, res, &response)
		},
	}).Test(t)

	// Each runner's agent namespace is listed.
	require.Subset(t, *namespaces, []string{adminNamespace, tenantNamespace})
	c.AssertNumberOfCalls(t, "DescribeWorkflowExecution", 2)
	require.Equal(t, []ActionAuthorSessionListItem{
		{
			SessionID:               newest,
			Status:                  "Running",
			StartTime:               "2026-01-03T00:00:00Z",
			WalletVersionIdentifier: wallet + "/v2",
			DeviceID:                authoringSessionDevice,
			DeviceName:              "android-emulator",
		},
		{
			SessionID:               oldest,
			Status:                  "Completed",
			StartTime:               "2026-01-01T00:00:00Z",
			WalletVersionIdentifier: authoringAndroidVersion,
			DeviceID:                "usera-s-organization/removed-runner/phone",
		},
	}, response.Sessions)
}

// walletSessionsFixture is the caller userA, the agent namespaces of the
// ready admin and tenant runners, and the sessions query of userA's sessions
// of authoringSessionsWallet.
type walletSessionsFixture struct {
	app           *tests.TestApp
	user          *core.Record
	admin, tenant string
	query         string
}

const authoringSessionsWallet = "usera-s-organization/authoring-handler-wallet"

// session is a signed session of the caller on runner (an agent namespace),
// listed with run, started on day d of January 2026.
func (f walletSessionsFixture) session(
	runnerNamespace, ticket, run string,
	d int,
) listedAuthoringSession {
	return listedAuthoringSession{
		authoringSessionGrant: authoringSessionGrant{
			sessionID: actionauthor.SessionWorkflowID(
				strings.TrimPrefix(runnerNamespace, "runner-"),
				ticket,
			),
			organization: authoringNamespace,
			device:       authoringSessionDevice,
			version:      authoringAndroidVersion,
			actorID:      f.user.Id,
		},
		run:     run,
		started: time.Date(2026, 1, d, 0, 0, 0, 0, time.UTC),
		signed:  true,
		status:  enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
	}
}

// list makes c answer the sessions query in namespace with executions, or
// with err.
func (f walletSessionsFixture) list(
	c *temporalmocks.Client,
	namespace string,
	err error,
	executions ...*workflowpb.WorkflowExecutionInfo,
) {
	var resp *workflowservice.ListWorkflowExecutionsResponse
	if err == nil {
		resp = &workflowservice.ListWorkflowExecutionsResponse{Executions: executions}
	}
	c.On("ListWorkflow", mock.Anything, mock.MatchedBy(
		func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
			return req.GetNamespace() == namespace && req.GetQuery() == f.query &&
				req.GetPageSize() == 50 && len(req.GetNextPageToken()) == 0
		},
	)).Return(resp, err).Once()
}

// listWalletSessions lists userA's sessions of authoringSessionsWallet after
// setup stubs c, expecting status, and returns the listed sessions and the
// namespaces the handler asked a client for.
func listWalletSessions(
	t *testing.T,
	status int,
	setup func(t testing.TB, f walletSessionsFixture, c *temporalmocks.Client),
) ([]ActionAuthorSessionListItem, []string) {
	t.Helper()
	c := &temporalmocks.Client{}
	namespaces := stubAuthoringSessionTemporal(t, c)
	content := `"sessions":[`
	if status != http.StatusOK {
		content = `"reason":"temporal_unavailable"`
	}
	var response ActionAuthorSessionsResponse
	(&tests.ApiScenario{
		Name:            "wallet sessions",
		Method:          http.MethodGet,
		URL:             "/api/action-author/sessions?wallet=" + authoringSessionsWallet,
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  status,
		ExpectedContent: []string{content},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)
			f := walletSessionsFixture{
				app:  app,
				user: user,
				query: fmt.Sprintf(
					`WorkflowType = "Action author session" and ActionAuthorActor = "%s" and `+
						`ActionAuthorWallet = "%s"`,
					user.Id,
					authoringSessionsWallet,
				),
			}
			for name, namespace := range map[string]*string{
				"admin-runner":  &f.admin,
				"tenant-runner": &f.tenant,
			} {
				runner, err := app.FindFirstRecordByData("mobile_runners", "name", name)
				require.NoError(t, err)
				require.NoError(t, mobilerunner.MarkAgentNamespaceReady(app, runner.Id))
				*namespace = mobilerunner.AgentNamespace(runner.Id)
			}
			setup(t, f, c)
			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
			if status == http.StatusOK {
				decodeBody(t, res, &response)
			}
		},
	}).Test(t)
	return response.Sessions, *namespaces
}

// A runner whose agent namespace is not ready is not listed, a namespace
// Temporal does not know or that fails to list is skipped, and the request
// fails only when every listed namespace fails.
func TestActionAuthorSessionsSkipsUnavailableNamespaces(t *testing.T) {
	unavailable := serviceerror.NewUnavailable("temporal down")
	for _, tc := range []struct {
		name                string
		adminErr, tenantErr error
		status              int
	}{
		{
			name:     "missing namespace",
			adminErr: serviceerror.NewNamespaceNotFound("runner-admin"),
			status:   http.StatusOK,
		},
		{name: "failing namespace", adminErr: unavailable, status: http.StatusOK},
		{
			name:      "every namespace fails",
			adminErr:  unavailable,
			tenantErr: unavailable,
			status:    http.StatusServiceUnavailable,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var unready, sessionID string
			sessions, namespaces := listWalletSessions(t, tc.status, func(
				t testing.TB,
				f walletSessionsFixture,
				c *temporalmocks.Client,
			) {
				tenant, err := f.app.FindFirstRecordByData(
					"mobile_runners",
					"name",
					"tenant-runner",
				)
				require.NoError(t, err)
				runner := createAuthoringRunner(
					t,
					f.app,
					tenant.GetString("owner"),
					"unready",
					false,
				)
				unready = mobilerunner.AgentNamespace(runner.Id)
				f.list(c, f.admin, tc.adminErr)
				if tc.tenantErr != nil {
					f.list(c, f.tenant, tc.tenantErr)
					return
				}
				listed := f.session(f.tenant, "00000000-0000-4000-8000-000000000001", "run", 1)
				listed.stub(t, f.app, c)
				sessionID = listed.sessionID
				f.list(c, f.tenant, nil, listed.execution())
			})

			require.NotContains(t, namespaces, unready)
			if tc.status != http.StatusOK {
				return
			}
			require.Len(t, sessions, 1)
			require.Equal(t, sessionID, sessions[0].SessionID)
		})
	}
}

// Runs forged under the caller's search attributes are listed newer than the
// caller's sessions, but never push them out of the list.
func TestActionAuthorSessionsKeepsTrustedSessionsBehindForgedRuns(t *testing.T) {
	var want []string
	sessions, _ := listWalletSessions(t, http.StatusOK, func(
		t testing.TB,
		f walletSessionsFixture,
		c *temporalmocks.Client,
	) {
		f.list(c, f.admin, nil)
		executions := []*workflowpb.WorkflowExecutionInfo{}
		for i := range actionAuthorRecentSessions + 2 {
			forged := f.session(
				f.tenant,
				fmt.Sprintf("00000000-0000-4000-8000-%012d", 100+i),
				"run-forged",
				10+i,
			)
			forged.signed = false
			forged.status = enumspb.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED
			forged.stub(t, f.app, c)
			executions = append(executions, forged.execution())
		}
		for i := range 2 {
			listed := f.session(
				f.tenant,
				fmt.Sprintf("00000000-0000-4000-8000-%012d", 1+i),
				"run",
				2-i,
			)
			listed.stub(t, f.app, c)
			want = append(want, listed.sessionID)
			executions = append(executions, listed.execution())
		}
		f.list(c, f.tenant, nil, executions...)
	})

	got := make([]string, 0, len(sessions))
	for _, session := range sessions {
		got = append(got, session.SessionID)
	}
	require.Equal(t, want, got)
}

// A session listed with several runs is listed once.
func TestActionAuthorSessionsListsASessionOnce(t *testing.T) {
	var sessionID string
	sessions, _ := listWalletSessions(t, http.StatusOK, func(
		t testing.TB,
		f walletSessionsFixture,
		c *temporalmocks.Client,
	) {
		listed := f.session(f.tenant, "00000000-0000-4000-8000-000000000001", "run", 1)
		listed.stub(t, f.app, c)
		sessionID = listed.sessionID
		replayed := listed.execution()
		replayed.Execution = &commonpb.WorkflowExecution{WorkflowId: sessionID, RunId: "run-2"}
		replayed.StartTime = timestamppb.New(listed.started.Add(time.Hour))
		f.list(c, f.admin, nil)
		f.list(c, f.tenant, nil, listed.execution(), replayed)
	})

	require.Len(t, sessions, 1)
	require.Equal(t, sessionID, sessions[0].SessionID)
}

func TestActionAuthorSessionsValidatesTheWallet(t *testing.T) {
	c := &temporalmocks.Client{}
	stubAuthoringSessionTemporal(t, c)
	c.On("ListWorkflow", mock.Anything, mock.Anything).
		Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil).Maybe()
	for _, wallet := range []string{"", "org", "org/wallet/v1", "/wallet"} {
		(&tests.ApiScenario{
			Name:            "wallet " + wallet,
			Method:          http.MethodGet,
			URL:             "/api/action-author/sessions?wallet=" + wallet,
			Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"reason":"invalid_request"`},
			TestAppFactory:  setupActionAuthorApp,
		}).Test(t)
	}
	// No session: an empty list, never null.
	(&tests.ApiScenario{
		Name:            "no sessions",
		Method:          http.MethodGet,
		URL:             "/api/action-author/sessions?wallet=usera-s-organization/none",
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"sessions":[]`},
		TestAppFactory:  setupActionAuthorApp,
	}).Test(t)
}

func TestActionAuthorSessionCloseSignalsTheChatOrCancelsTheSession(t *testing.T) {
	notFound := serviceerror.NewNotFound("no chat")
	for _, tc := range []struct {
		name   string
		status enumspb.WorkflowExecutionStatus
		// signalErrs answer the close signals in turn; missing ones succeed.
		signalErrs []error
		// chatErr is the chat's describe answer after a NotFound signal.
		chatErr error
		// chatRunning makes that describe find the chat running.
		chatRunning bool
		// signals is the number of close signals sent to a running session's chat.
		signals int
		code    int
		content string
		cancels bool
	}{
		{
			name:    "chat runs",
			status:  enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
			signals: 1,
			code:    http.StatusAccepted,
			content: `"session_id":"` + authoringSessionID + `"`,
		},
		{
			name:       "device still being prepared",
			status:     enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
			signalErrs: []error{notFound},
			chatErr:    notFound,
			signals:    1,
			code:       http.StatusAccepted,
			content:    `"session_id":"` + authoringSessionID + `"`,
			cancels:    true,
		},
		{
			name:       "chat already ended",
			status:     enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
			signalErrs: []error{serviceerror.NewNotFound("chat completed")},
			signals:    1,
			code:       http.StatusAccepted,
			content:    `"session_id":"` + authoringSessionID + `"`,
		},
		{
			name:        "chat started after the signal",
			status:      enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
			signalErrs:  []error{notFound},
			chatRunning: true,
			signals:     2,
			code:        http.StatusAccepted,
			content:     `"session_id":"` + authoringSessionID + `"`,
		},
		{
			name:        "chat started and ended after the signal",
			status:      enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
			signalErrs:  []error{notFound, serviceerror.NewNotFound("chat completed")},
			chatRunning: true,
			signals:     2,
			code:        http.StatusAccepted,
			content:     `"session_id":"` + authoringSessionID + `"`,
		},
		{
			name:    "session ended",
			status:  enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			code:    http.StatusConflict,
			content: `"reason":"session_not_running"`,
		},
	} {
		c := &temporalmocks.Client{}
		stubAuthoringSessionTemporal(t, c)
		type signal struct{ namespace, chatID string }
		signals := []signal{}
		orig := signalChatClose
		t.Cleanup(func() { signalChatClose = orig })
		signalChatClose = func(_ context.Context, _ client.Client, namespace, chatID string) error {
			signals = append(signals, signal{namespace, chatID})
			if len(signals) <= len(tc.signalErrs) {
				return tc.signalErrs[len(signals)-1]
			}
			return nil
		}
		c.On("CancelWorkflow", mock.Anything, authoringSessionID, authoringSessionRun).Return(nil)
		var chat *workflowservice.DescribeWorkflowExecutionResponse
		if tc.chatErr == nil {
			chatStatus := enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED
			if tc.chatRunning {
				chatStatus = enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING
			}
			chat = authoringExecution("ActionAuthor", chatStatus, 0)
		}
		c.On("DescribeWorkflowExecution", mock.Anything, authoringSessionChat, "").
			Return(chat, tc.chatErr)
		(&tests.ApiScenario{
			Name:            tc.name,
			Method:          http.MethodPost,
			URL:             "/api/action-author/sessions/" + authoringSessionID + "/close",
			Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
			ExpectedStatus:  tc.code,
			ExpectedContent: []string{tc.content},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupActionAuthorApp(t)
				runningAuthoringSession(app, t, c, tc.status)
				return app
			},
		}).Test(t)

		require.Len(t, signals, tc.signals, tc.name)
		for _, s := range signals {
			require.Equal(t, signal{authoringSessionNamespace, authoringSessionChat}, s, tc.name)
		}
		if tc.cancels {
			c.AssertNumberOfCalls(t, "CancelWorkflow", 1)
			c.AssertCalled(
				t,
				"CancelWorkflow",
				mock.Anything,
				authoringSessionID,
				authoringSessionRun,
			)
		} else {
			c.AssertNotCalled(t, "CancelWorkflow", mock.Anything, mock.Anything, mock.Anything)
		}
	}
}

func TestChatCloseRequestHasNoInput(t *testing.T) {
	request := chatCloseRequest("org", "x-author")
	require.Nil(t, request.GetInput())
	require.Equal(t, "close", request.GetSignalName())
	require.Equal(t, "x-author", request.GetWorkflowExecution().GetWorkflowId())
	require.Equal(t, "org", request.GetNamespace())
}
