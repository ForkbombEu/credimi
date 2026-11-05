// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const agentRunnerID = "/usera-s-organization/agent-runner"

// heartbeatAgentRunner sends a heartbeat of userA's agent-runner with the
// given action author settings and returns the decoded response.
func heartbeatAgentRunner(
	t *testing.T,
	app *tests.TestApp,
	now time.Time,
	settings *RunnerActionAuthorSettings,
) map[string]any {
	t.Helper()
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	origNow := mobileRunnerLifecycleNow
	origQuerySemaphore := queryMobileDeviceSemaphoreState
	t.Cleanup(func() {
		mobileRunnerLifecycleNow = origNow
		queryMobileDeviceSemaphoreState = origQuerySemaphore
	})
	mobileRunnerLifecycleNow = func() time.Time { return now }
	queryMobileDeviceSemaphoreState = func(context.Context, string) (workflows.MobileDeviceSemaphoreStateView, error) {
		return workflows.MobileDeviceSemaphoreStateView{}, errSemaphoreNotFound
	}
	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-runner/lifecycle/heartbeat",
		MobileRunnerLifecycleRequest{RunnerID: agentRunnerID, ActionAuthor: settings},
	)
	require.NoError(t, HandleMobileRunnerLifecycleHeartbeat()(event))
	recorder := responseRecorder(t, event)
	require.Equal(t, http.StatusOK, recorder.Code)
	return decodeJSONBody(t, recorder)
}

func agentRunnerRecord(t *testing.T, app core.App) *core.Record {
	t.Helper()
	record, err := canonify.Resolve(app, agentRunnerID)
	require.NoError(t, err)
	return record
}

func TestHeartbeatStoresActionAuthorSettingsAndReportsUsage(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, orgID, "agent-runner", "https://runner.example", false)
	now := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)

	response := heartbeatAgentRunner(t, app, now, &RunnerActionAuthorSettings{
		OwnDailyTokens:    1000,
		OthersDailyTokens: 200000,
		Provider:          "anthropic",
		Model:             "claude-sonnet-5-5",
	})
	runner := agentRunnerRecord(t, app)
	require.Equal(t, 1000, runner.GetInt("agent_own_daily_tokens"))
	require.Equal(t, 200000, runner.GetInt("agent_others_daily_tokens"))
	require.Equal(t, "anthropic", runner.GetString("agent_provider"))
	require.Equal(t, "claude-sonnet-5-5", runner.GetString("agent_model"))
	require.Equal(t, map[string]any{
		"day":           "2026-11-05",
		"own_tokens":    float64(0),
		"others_tokens": float64(0),
	}, response["action_author_usage"])

	// Usage of both pools today; another day's usage does not count.
	for pool, tokens := range map[actionauthor.Pool]int{
		actionauthor.PoolOwn:    1234,
		actionauthor.PoolOthers: 56,
	} {
		_, err := actionauthor.RecordUsage(
			app,
			actionauthor.Allowance{RunnerID: runner.Id, Pool: pool},
			now,
			actionauthor.UsageReport{
				ReportID:   "wf/1",
				UsageDelta: actionauthor.UsageDelta{InputTokens: tokens},
			},
		)
		require.NoError(t, err)
	}
	_, err = actionauthor.RecordUsage(
		app,
		actionauthor.Allowance{RunnerID: runner.Id, Pool: actionauthor.PoolOwn},
		now.Add(-24*time.Hour),
		actionauthor.UsageReport{
			ReportID:   "wf/0",
			UsageDelta: actionauthor.UsageDelta{InputTokens: 99},
		},
	)
	require.NoError(t, err)

	// A heartbeat without settings keeps the stored ones.
	response = heartbeatAgentRunner(t, app, now, nil)
	runner = agentRunnerRecord(t, app)
	require.Equal(t, 1000, runner.GetInt("agent_own_daily_tokens"))
	require.Equal(t, "claude-sonnet-5-5", runner.GetString("agent_model"))
	require.Equal(t, map[string]any{
		"day":           "2026-11-05",
		"own_tokens":    float64(1234),
		"others_tokens": float64(56),
	}, response["action_author_usage"])

	// A disabled agent reports zeros and empty strings.
	heartbeatAgentRunner(t, app, now, &RunnerActionAuthorSettings{})
	runner = agentRunnerRecord(t, app)
	require.Zero(t, runner.GetInt("agent_own_daily_tokens"))
	require.Zero(t, runner.GetInt("agent_others_daily_tokens"))
	require.Empty(t, runner.GetString("agent_provider"))
	require.Empty(t, runner.GetString("agent_model"))
}

func TestLifecycleRejectsNegativeActionAuthorLimits(t *testing.T) {
	token := authoringToken(t, "userA")
	for _, body := range []string{
		`{"runner_id":"` + agentRunnerID + `","action_author":{"own_daily_tokens":-1}}`,
		`{"runner_id":"` + agentRunnerID + `","action_author":{"others_daily_tokens":-1}}`,
	} {
		(&tests.ApiScenario{
			Name:            body,
			Method:          http.MethodPost,
			URL:             "/api/mobile-runner/lifecycle/heartbeat",
			Body:            strings.NewReader(body),
			Headers:         map[string]string{"Authorization": "Bearer " + token},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"request.validation"`, `failed on the 'min' tag`},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupMobileRunnerApp(t)
				org, err := app.FindFirstRecordByData("organizations", "canonified_name", "usera-s-organization")
				require.NoError(t, err)
				createMobileRunnerRecord(t, app, org.Id, "agent-runner", "https://runner.example", false)
				return app
			},
		}).Test(t)
	}
}

func TestUpsertMobileRunnerStoresActionAuthorSettings(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	upsert := func(settings *RunnerActionAuthorSettings) {
		t.Helper()
		event := performMobileRunnerRequest(
			t,
			app,
			user,
			"/api/mobile-runner",
			UpsertMobileRunnerRequest{
				Name:         "Agent runner",
				IP:           "https://runner.example",
				Type:         "android_emulator",
				ActionAuthor: settings,
			},
		)
		require.NoError(t, HandleUpsertMobileRunner()(event))
		require.Equal(t, http.StatusOK, responseRecorder(t, event).Code)
	}

	upsert(&RunnerActionAuthorSettings{
		OwnDailyTokens:    0,
		OthersDailyTokens: 5000,
		Provider:          "openai_compatible",
		Model:             "gpt-5",
	})
	runner := agentRunnerRecord(t, app)
	require.Zero(t, runner.GetInt("agent_own_daily_tokens"))
	require.Equal(t, 5000, runner.GetInt("agent_others_daily_tokens"))
	require.Equal(t, "openai_compatible", runner.GetString("agent_provider"))
	require.Equal(t, "gpt-5", runner.GetString("agent_model"))

	upsert(nil)
	runner = agentRunnerRecord(t, app)
	require.Equal(t, 5000, runner.GetInt("agent_others_daily_tokens"))
	require.Equal(t, "gpt-5", runner.GetString("agent_model"))
}
