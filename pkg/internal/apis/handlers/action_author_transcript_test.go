// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	updatepb "go.temporal.io/api/update/v1"
	temporalmocks "go.temporal.io/sdk/mocks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// stubAuthoringChatHistory makes c answer the latest run of the session's
// chat with history.
func stubAuthoringChatHistory(c *temporalmocks.Client, history *fakeHistoryIterator) {
	c.On(
		"GetWorkflowHistory",
		mock.Anything,
		authoringSessionChat,
		"",
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(history)
}

func TestActionAuthorSessionTranscriptReadsTheChatHistory(t *testing.T) {
	c := &temporalmocks.Client{}
	namespaces := stubAuthoringSessionTemporal(t, c)
	args, err := temporalcrypto.DataConverter().ToPayloads(map[string]any{
		"type":    "ask",
		"payload": map[string]any{"text": "Write the onboarding"},
	})
	require.NoError(t, err)
	stubAuthoringChatHistory(c, &fakeHistoryIterator{events: []*historypb.HistoryEvent{{
		EventId:   1,
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		EventTime: timestamppb.New(authoringSessionStart),
	}, {
		EventId:   2,
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED,
		EventTime: timestamppb.New(authoringSessionStart.Add(time.Minute)),
		Attributes: &historypb.HistoryEvent_WorkflowExecutionUpdateAcceptedEventAttributes{
			WorkflowExecutionUpdateAcceptedEventAttributes: &historypb.WorkflowExecutionUpdateAcceptedEventAttributes{
				AcceptedRequest: &updatepb.Request{Input: &updatepb.Input{
					Name: "send_agent_message",
					Args: args,
				}},
			},
		},
	}, {
		EventId:   3,
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_UPDATE_ACCEPTED,
		EventTime: timestamppb.New(authoringSessionStart.Add(2 * time.Minute)),
		Attributes: &historypb.HistoryEvent_WorkflowExecutionUpdateAcceptedEventAttributes{
			WorkflowExecutionUpdateAcceptedEventAttributes: &historypb.WorkflowExecutionUpdateAcceptedEventAttributes{
				AcceptedRequest: &updatepb.Request{Input: &updatepb.Input{
					Name: "send_agent_message",
					Args: &commonpb.Payloads{Payloads: []*commonpb.Payload{{
						Metadata: map[string][]byte{"encoding": []byte("binary/unknown")},
					}}},
				}},
			},
		},
	}}})
	var response ActionAuthorTranscriptResponse
	(&tests.ApiScenario{
		Name:           "starter",
		Method:         http.MethodGet,
		URL:            "/api/action-author/sessions/" + authoringSessionID + "/transcript",
		Headers:        map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"entries":[`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			runningAuthoringSession(app, t, c, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED)
			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
			decodeBody(t, res, &response)
		},
	}).Test(t)

	// The chat is read in the runner namespace the session runs in.
	require.Equal(t, []string{authoringSessionNamespace}, *namespaces)
	require.Equal(t, []activities.TranscriptEntry{
		{
			Kind: activities.TranscriptUser,
			Time: "2026-01-02T03:05:05Z",
			Text: "Write the onboarding",
		},
		{Kind: activities.TranscriptUnavailable, Time: "2026-01-02T03:06:05Z"},
	}, response.Entries)
}

// An empty chat answers an empty list, not null.
func TestActionAuthorSessionTranscriptOfAnEmptyChat(t *testing.T) {
	c := &temporalmocks.Client{}
	stubAuthoringSessionTemporal(t, c)
	stubAuthoringChatHistory(c, &fakeHistoryIterator{})
	(&tests.ApiScenario{
		Name:            "empty chat",
		Method:          http.MethodGet,
		URL:             "/api/action-author/sessions/" + authoringSessionID + "/transcript",
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, "userA")},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`{"entries":[]}`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			runningAuthoringSession(app, t, c, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED)
			return app
		},
	}).Test(t)
}

func TestActionAuthorSessionTranscriptAnswersNotFound(t *testing.T) {
	for _, tc := range []struct {
		name  string
		user  string
		chat  *fakeHistoryIterator
		setup func(t *testing.T, app *tests.TestApp)
	}{
		{
			name: "another member",
			user: "userB",
			setup: func(t *testing.T, app *tests.TestApp) {
				joinAgentRunnerOrganization(t, app, "member")
			},
		},
		{
			name: "missing chat",
			user: "userA",
			chat: &fakeHistoryIterator{nextErr: serviceerror.NewNotFound("no chat")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &temporalmocks.Client{}
			stubAuthoringSessionTemporal(t, c)
			if tc.chat != nil {
				stubAuthoringChatHistory(c, tc.chat)
			}
			(&tests.ApiScenario{
				Name:   tc.name,
				Method: http.MethodGet,
				URL:    "/api/action-author/sessions/" + authoringSessionID + "/transcript",
				Headers: map[string]string{
					"Authorization": "Bearer " + authoringToken(t, tc.user),
				},
				ExpectedStatus:  http.StatusNotFound,
				ExpectedContent: []string{`"reason":"session_not_found"`},
				TestAppFactory: func(testing.TB) *tests.TestApp {
					app := setupActionAuthorApp(t)
					if tc.setup != nil {
						tc.setup(t, app)
					}
					runningAuthoringSession(app, t, c, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED)
					return app
				},
			}).Test(t)

			if tc.chat == nil {
				c.AssertNotCalled(
					t,
					"GetWorkflowHistory",
					mock.Anything,
					authoringSessionChat,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				)
			}
		})
	}
}
