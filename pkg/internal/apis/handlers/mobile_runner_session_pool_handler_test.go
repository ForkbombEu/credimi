// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	temporalmocks "go.temporal.io/sdk/mocks"
)

const poolTestTicket = "2fe33961-f99b-422f-90f1-5353e201ae7a"

func (f registrationFixture) requestSessionPool(
	t *testing.T,
	auth *core.Record,
	sessionID string,
) (*core.RequestEvent, error) {
	t.Helper()

	event := performMobileRunnerRequest(
		t,
		f.app,
		auth,
		"/api/mobile-runner/action-author-session-pool",
		MobileRunnerSessionPoolRequest{RunnerID: tokenTestRunnerID, SessionID: sessionID},
	)
	return event, HandleMobileRunnerSessionPool()(event)
}

// startPoolSession fakes, in the namespace its ID names, a session that
// Credimi started for the organization orgID on the fixture's runner, in
// status, and returns its ID.
func (f registrationFixture) startPoolSession(
	t *testing.T,
	runnerID, orgID string,
	status enumspb.WorkflowExecutionStatus,
) string {
	t.Helper()

	sessionID := actionauthor.SessionWorkflowID(runnerID, poolTestTicket)
	c := &temporalmocks.Client{}
	stubSessionRuns(t, c, sessionID, authoringSessionRun)
	c.On(
		"GetWorkflowHistory",
		mock.Anything,
		sessionID,
		authoringSessionRun,
		false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(authoringSessionGrant{
		sessionID:    sessionID,
		organization: orgNamespace(t, f, orgID),
		device:       tokenTestRunnerID + "/phone",
		version:      orgNamespace(t, f, orgID) + "/wallet/v1",
		actorID:      f.user.Id,
	}.history(t, f.app, authoringSessionRun, true)).Maybe()
	c.On("DescribeWorkflowExecution", mock.Anything, sessionID, authoringSessionRun).
		Return(authoringExecution(activities.ActionAuthorSessionWorkflowName, status, 1), nil).
		Maybe()
	stubAuthoringSessionTemporal(t, c)
	return sessionID
}

func (f registrationFixture) poolTestRunner(t *testing.T) *core.Record {
	t.Helper()

	f.setRunnerField(t, "agent_enabled", true)
	record, err := canonify.Resolve(f.app, "/"+tokenTestRunnerID)
	require.NoError(t, err)
	return record
}

func requireSessionPool(t *testing.T, event *core.RequestEvent, want string) {
	t.Helper()

	recorder := responseRecorder(t, event)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, map[string]any{"pool": want}, decodeJSONBody(t, recorder))
}

func TestHandleMobileRunnerSessionPool(t *testing.T) {
	t.Run("a session of the runner's organization spends the own pool", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		runner := f.poolTestRunner(t)
		sessionID := f.startPoolSession(
			t,
			runner.Id,
			f.orgID,
			enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		)

		event, err := f.requestSessionPool(t, f.user, sessionID)
		require.NoError(t, err)
		requireSessionPool(t, event, "own")
	})

	t.Run("a session of another organization spends the others pool", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		runner := f.poolTestRunner(t)
		sessionID := f.startPoolSession(
			t,
			runner.Id,
			f.otherOrg,
			enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		)

		event, err := f.requestSessionPool(t, f.user, sessionID)
		require.NoError(t, err)
		requireSessionPool(t, event, "others")
	})

	t.Run("a session of another runner is not found", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		f.poolTestRunner(t)
		sessionID := f.startPoolSession(
			t,
			"otherrunner0001",
			f.orgID,
			enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		)

		_, err := f.requestSessionPool(t, f.user, sessionID)
		requireRegistrationAPIError(
			t,
			err,
			http.StatusNotFound,
			"action_author",
			"session_not_found",
		)
	})

	t.Run("a closed session is not found", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		runner := f.poolTestRunner(t)
		sessionID := f.startPoolSession(
			t,
			runner.Id,
			f.orgID,
			enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED,
		)

		_, err := f.requestSessionPool(t, f.user, sessionID)
		requireRegistrationAPIError(
			t,
			err,
			http.StatusNotFound,
			"action_author",
			"session_not_found",
		)
	})

	t.Run("a session Credimi did not start is not found", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		runner := f.poolTestRunner(t)
		c := &temporalmocks.Client{}
		sessionID := actionauthor.SessionWorkflowID(runner.Id, poolTestTicket)
		stubSessionRuns(t, c, sessionID)
		stubAuthoringSessionTemporal(t, c)

		_, err := f.requestSessionPool(t, f.user, sessionID)
		requireRegistrationAPIError(
			t,
			err,
			http.StatusNotFound,
			"action_author",
			"session_not_found",
		)
	})

	t.Run("only owners of the runner's organization ask", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		runner := f.poolTestRunner(t)
		sessionID := f.startPoolSession(
			t,
			runner.Id,
			f.orgID,
			enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
		)

		memberRole, err := f.app.FindFirstRecordByData("orgRoles", "name", "member")
		require.NoError(t, err)
		authorization, err := f.app.FindFirstRecordByData("orgAuthorizations", "user", f.user.Id)
		require.NoError(t, err)
		authorization.Set("role", memberRole.Id)
		require.NoError(t, f.app.Save(authorization))

		_, err = f.requestSessionPool(t, f.user, sessionID)
		requireRegistrationAPIError(
			t,
			err,
			http.StatusForbidden,
			"runner_id",
			"runner_owner_role_required",
		)
	})

	t.Run("the runner's agent must be enabled", func(t *testing.T) {
		f := setupRegistrationFixture(t)
		runner := f.poolTestRunner(t)
		f.setRunnerField(t, "agent_enabled", false)

		_, err := f.requestSessionPool(
			t,
			f.user,
			actionauthor.SessionWorkflowID(runner.Id, poolTestTicket),
		)
		requireRegistrationAPIError(t, err, http.StatusForbidden, "scope", "agent_not_enabled")
	})
}
