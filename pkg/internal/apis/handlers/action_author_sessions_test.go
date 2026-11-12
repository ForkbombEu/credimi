// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
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
	authoringSessionID     = "ActionAuthorSession-0f8fad5b-d9cb-469f-a165-70867728950e"
	authoringSessionChat   = authoringSessionID + "-author"
	authoringSessionRun    = "session-run"
	authoringSessionDevice = authoringAdminRunner + "/android-emulator"
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

// recordAuthoringSession signs and records, as the start activity does, the
// session authoringSessionID that userA started in namespace on the admin
// runner's emulator, and returns its history.
func recordAuthoringSession(
	t testing.TB,
	app *tests.TestApp,
	namespace string,
) client.HistoryEventIterator {
	t.Helper()
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	grant, err := actionauthor.SignSession(app, actionauthor.SessionGrant{
		SessionWorkflowID:       authoringSessionID,
		Namespace:               namespace,
		DeviceID:                authoringSessionDevice,
		Platform:                actionauthor.PlatformAndroid,
		WalletVersionIdentifier: authoringAndroidVersion,
		ActorID:                 user.Id,
	})
	require.NoError(t, err)
	require.NoError(t, actionauthor.RecordSessionRun(
		app,
		namespace,
		authoringSessionID,
		authoringSessionRun,
		actionauthor.SessionDetails{
			WalletVersionIdentifier: authoringAndroidVersion,
			DeviceID:                authoringSessionDevice,
			ActorID:                 user.Id,
			Organization:            namespace,
		},
	))
	input, err := temporalcrypto.DataConverter().ToPayloads(workflowengine.WorkflowInput{
		Payload: map[string]any{
			"device_id":                 authoringSessionDevice,
			"platform":                  actionauthor.PlatformAndroid,
			"wallet_version_identifier": authoringAndroidVersion,
			"actor_id":                  user.Id,
			"grant":                     grant,
		},
	})
	require.NoError(t, err)
	return &fakeHistoryIterator{events: []*historypb.HistoryEvent{{
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
			WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
				Input: input,
			},
		},
	}}}
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

// runningAuthoringSession fakes the running session recorded in userA's
// namespace and its chat in its first turn, with a pending save review.
func runningAuthoringSession(
	app *tests.TestApp,
	t testing.TB,
	c *temporalmocks.Client,
	status enumspb.WorkflowExecutionStatus,
) {
	c.On("DescribeWorkflowExecution", mock.Anything, authoringSessionID, authoringSessionRun).
		Return(authoringExecution(activities.ActionAuthorSessionWorkflowName, status, 0), nil)
	c.On(
		"GetWorkflowHistory",
		mock.Anything,
		authoringSessionID,
		authoringSessionRun,
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(recordAuthoringSession(t, app, authoringNamespace))
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

	require.Equal(t, []string{authoringNamespace}, *namespaces)
	require.Equal(t, ActionAuthorSessionResponse{
		SessionID:               authoringSessionID,
		RunID:                   authoringSessionRun,
		ChatWorkflowID:          authoringSessionChat,
		Namespace:               authoringNamespace,
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
		// namespace the session is recorded in; empty records none.
		namespace string
		// user reading the session; empty is userA, who records it.
		user string
	}{
		{name: "malformed id", sessionID: "ActionAuthorSession-ticket"},
		{name: "unrecorded", sessionID: authoringSessionID},
		{
			name:      "recorded in another namespace",
			sessionID: authoringSessionID,
			namespace: authoringForeignNS,
		},
		{
			name:      "started by another member",
			sessionID: authoringSessionID,
			namespace: authoringNamespace,
			user:      "userB",
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
				if tc.user != "" {
					joinAgentRunnerOrganization(t, app, "member")
				}
				if tc.namespace != "" {
					recordAuthoringSession(t, app, tc.namespace)
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
	c.On("DescribeWorkflowExecution", mock.Anything, authoringSessionID, authoringSessionRun).
		Return(authoringExecution(
			"Dynamic Pipeline Workflow",
			enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
			0,
		), nil)
	(&tests.ApiScenario{
		Name:            "pipeline under a session id",
		Method:          http.MethodGet,
		URL:             "/api/action-author/sessions/" + authoringSessionID,
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusNotFound,
		ExpectedContent: []string{`"reason":"session_not_found"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			recordAuthoringSession(t, app, authoringNamespace)
			return app
		},
	}).Test(t)
}

func TestActionAuthorSessionsListsTheWalletSessions(t *testing.T) {
	c := &temporalmocks.Client{}
	namespaces := stubAuthoringSessionTemporal(t, c)
	describe := func(id string, status enumspb.WorkflowExecutionStatus, err error) {
		c.On("DescribeWorkflowExecution", mock.Anything, id, "run").
			Return(authoringExecution(activities.ActionAuthorSessionWorkflowName, status, 0), err)
	}
	describe("ActionAuthorSession-version", enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED, nil)
	describe("ActionAuthorSession-store", enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING, nil)
	c.On("DescribeWorkflowExecution", mock.Anything, "ActionAuthorSession-gone", "run").
		Return(nil, serviceerror.NewNotFound("gone"))
	var response ActionAuthorSessionsResponse
	(&tests.ApiScenario{
		Name:            "wallet sessions",
		Method:          http.MethodGet,
		URL:             "/api/action-author/sessions?wallet=usera-s-organization/authoring-handler-wallet",
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"sessions":[`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)
			for _, row := range []struct{ namespace, id, wallet, device, actor string }{
				{authoringNamespace, "ActionAuthorSession-store", "usera-s-organization/authoring-handler-wallet", authoringSessionDevice, user.Id},
				{authoringNamespace, "ActionAuthorSession-gone", authoringAndroidVersion, authoringSessionDevice, user.Id},
				{authoringNamespace, "ActionAuthorSession-version", authoringAndroidVersion, "usera-s-organization/removed-runner/phone", user.Id},
				{authoringNamespace, "ActionAuthorSession-other-wallet", "usera-s-organization/authoring-handler-wallet2/v1", authoringSessionDevice, user.Id},
				{authoringNamespace, "ActionAuthorSession-other-actor", authoringAndroidVersion, authoringSessionDevice, "another-user"},
				{authoringForeignNS, "ActionAuthorSession-other-namespace", authoringAndroidVersion, authoringSessionDevice, user.Id},
			} {
				require.NoError(t, actionauthor.RecordSessionRun(
					app,
					row.namespace,
					row.id,
					"run",
					actionauthor.SessionDetails{
						WalletVersionIdentifier: row.wallet,
						DeviceID:                row.device,
						ActorID:                 row.actor,
						Organization:            row.namespace,
					},
				))
				// Rows of one test share a millisecond: order them by name.
				_, err := app.DB().NewQuery(
					"UPDATE action_author_sessions SET created = {:created} WHERE workflow_id = {:id}",
				).Bind(map[string]any{
					"created": map[string]string{
						"ActionAuthorSession-store":   "2026-01-01 00:00:00.000Z",
						"ActionAuthorSession-gone":    "2026-01-02 00:00:00.000Z",
						"ActionAuthorSession-version": "2026-01-03 00:00:00.000Z",
					}[row.id],
					"id": row.id,
				}).Execute()
				require.NoError(t, err)
			}
			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
			decodeBody(t, res, &response)
		},
	}).Test(t)

	require.Equal(t, []string{authoringNamespace}, *namespaces)
	require.Equal(t, []ActionAuthorSessionListItem{
		{
			SessionID:               "ActionAuthorSession-version",
			Status:                  "Completed",
			StartTime:               "2026-01-02T03:04:05Z",
			WalletVersionIdentifier: authoringAndroidVersion,
			DeviceID:                "usera-s-organization/removed-runner/phone",
		},
		{
			SessionID:               "ActionAuthorSession-store",
			Status:                  "Running",
			StartTime:               "2026-01-02T03:04:05Z",
			WalletVersionIdentifier: "usera-s-organization/authoring-handler-wallet",
			DeviceID:                authoringSessionDevice,
			DeviceName:              "android-emulator",
		},
	}, response.Sessions)
}

func TestActionAuthorSessionsValidatesTheWallet(t *testing.T) {
	stubAuthoringSessionTemporal(t, &temporalmocks.Client{})
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
			require.Equal(t, signal{authoringNamespace, authoringSessionChat}, s, tc.name)
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
