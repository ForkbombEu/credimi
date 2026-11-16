// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func stubAuthoringChat(c *temporalmocks.Client, status enumspb.WorkflowExecutionStatus) {
	c.On("DescribeWorkflow", mock.Anything, authoringSessionChat, "").
		Return(temporalclient.Description(authoringExecution(activities.ActionAuthorWorkflowType, status, 5)), nil)
}

func replayScenario(
	t *testing.T,
	c *temporalmocks.Client,
	user string,
	status int,
	content string,
) *tests.ApiScenario {
	return &tests.ApiScenario{
		Name:            user,
		Method:          http.MethodGet,
		URL:             "/api/action-author/sessions/" + authoringSessionID + "/replay",
		Headers:         map[string]string{"Authorization": "Bearer " + authoringToken(t, user)},
		ExpectedStatus:  status,
		ExpectedContent: []string{content},
		TestAppFactory: func(testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			if user != "userA" {
				joinAgentRunnerOrganization(t, app, "member")
			}
			runningAuthoringSession(app, t, c, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED)
			return app
		},
	}
}

func TestActionAuthorSessionReplayAsksTheAgent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		page   map[string]any
		err    error
		expect string
	}{
		{
			name: "agent answers",
			page: map[string]any{
				"items": []any{
					map[string]any{"topic": "agent", "data": "AAAA", "offset": 0},
				},
				"next_offset": 1,
				"more_ready":  false,
			},
			expect: `{"replayable":true}`,
		},
		{
			name:   "no events",
			page:   map[string]any{"items": []any{}, "next_offset": 0, "more_ready": false},
			expect: `{"replayable":false}`,
		},
		{
			name:   "query fails",
			err:    errors.New("context deadline exceeded"),
			expect: `{"replayable":false}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &temporalmocks.Client{}
			namespaces := stubAuthoringSessionTemporal(t, c)
			stubAuthoringChat(c, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED)
			var value any
			if tc.err == nil {
				value = jsonEncodedValue{tc.page}
			}
			c.On(
				"QueryWorkflow",
				mock.Anything,
				authoringSessionChat,
				"",
				"__temporal_workflow_stream_poll",
				map[string]any{"topics": []string{}, "from_offset": 0},
			).
				Return(value, tc.err).
				Once()
			replayScenario(t, c, "userA", http.StatusOK, tc.expect).Test(t)
			// The chat is queried in the runner namespace the session runs in.
			require.Equal(t, []string{authoringSessionNamespace}, *namespaces)
			c.AssertExpectations(t)
		})
	}
}

func TestActionAuthorSessionReplayOfAChatThatCannotBeQueried(t *testing.T) {
	for _, tc := range []struct {
		name string
		stub func(c *temporalmocks.Client)
	}{
		{
			name: "running chat",
			stub: func(c *temporalmocks.Client) {
				stubAuthoringChat(c, enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING)
			},
		},
		{
			name: "missing chat",
			stub: func(c *temporalmocks.Client) {
				c.On("DescribeWorkflow", mock.Anything, authoringSessionChat, "").
					Return(nil, serviceerror.NewNotFound("no chat"))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &temporalmocks.Client{}
			stubAuthoringSessionTemporal(t, c)
			tc.stub(c)
			replayScenario(t, c, "userA", http.StatusOK, `{"replayable":false}`).Test(t)
			c.AssertNotCalled(
				t,
				"QueryWorkflow",
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
			)
		})
	}
}

func TestActionAuthorSessionReplayRefusesAnotherMember(t *testing.T) {
	c := &temporalmocks.Client{}
	stubAuthoringSessionTemporal(t, c)
	replayScenario(t, c, "userB", http.StatusNotFound, `"reason":"session_not_found"`).Test(t)
	c.AssertNotCalled(t, "DescribeWorkflow", mock.Anything, authoringSessionChat, "")
}
