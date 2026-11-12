// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"encoding/json"
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

// performMobileRunnerKeyRequest builds a request authenticated by a
// Credimi-Api-Key alone, as runners send them. The header value is not
// checked: the event carries auth directly.
func performMobileRunnerKeyRequest(
	t testing.TB,
	app *tests.TestApp,
	auth *core.Record,
	url string,
	validatedInput any,
) *core.RequestEvent {
	t.Helper()
	event := performMobileRunnerRequest(t, app, auth, url, validatedInput)
	event.Request.Header.Set(APIKeyHeaderName, "test-key")
	return event
}

// registerAgentRunner upserts userA's agent-runner as auth, by key or by
// token, with the given action author settings.
func registerAgentRunner(
	t *testing.T,
	app *tests.TestApp,
	auth *core.Record,
	byKey bool,
	settings *RunnerActionAuthorSettings,
) {
	t.Helper()
	perform := performMobileRunnerRequest
	if byKey {
		perform = performMobileRunnerKeyRequest
	}
	event := perform(
		t,
		app,
		auth,
		"/api/mobile-runner",
		UpsertMobileRunnerRequest{
			RunnerID:     agentRunnerID,
			Name:         "Agent runner",
			IP:           "https://runner.example",
			Type:         "android_emulator",
			Organization: "usera-s-organization",
			ActionAuthor: settings,
		},
	)
	require.NoError(t, HandleUpsertMobileRunner()(event))
	require.Equal(t, http.StatusOK, responseRecorder(t, event).Code)
}

// sendAgentRunnerHeartbeat sends a heartbeat of userA's agent-runner as auth,
// by key or by token, with the given action author settings and returns the
// decoded response.
func sendAgentRunnerHeartbeat(
	t *testing.T,
	app *tests.TestApp,
	auth *core.Record,
	byKey bool,
	now time.Time,
	settings *RunnerActionAuthorSettings,
) map[string]any {
	t.Helper()
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
	perform := performMobileRunnerRequest
	if byKey {
		perform = performMobileRunnerKeyRequest
	}
	event := perform(
		t,
		app,
		auth,
		"/api/mobile-runner/lifecycle/heartbeat",
		MobileRunnerLifecycleRequest{RunnerID: agentRunnerID, ActionAuthor: settings},
	)
	require.NoError(t, HandleMobileRunnerLifecycleHeartbeat()(event))
	recorder := responseRecorder(t, event)
	require.Equal(t, http.StatusOK, recorder.Code)
	return decodeJSONBody(t, recorder)
}

// heartbeatAgentRunner sends a heartbeat of userA's agent-runner by userA's
// key with the given action author settings and returns the decoded response.
func heartbeatAgentRunner(
	t *testing.T,
	app *tests.TestApp,
	now time.Time,
	settings *RunnerActionAuthorSettings,
) map[string]any {
	t.Helper()
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	return sendAgentRunnerHeartbeat(t, app, user, true, now, settings)
}

func agentRunnerRecord(t *testing.T, app core.App) *core.Record {
	t.Helper()
	record, err := canonify.Resolve(app, agentRunnerID)
	require.NoError(t, err)
	return record
}

// joinAgentRunnerOrganization makes userB a member of userA's organization
// only, with the given orgRoles name, and returns userB.
func joinAgentRunnerOrganization(t *testing.T, app core.App, role string) *core.Record {
	t.Helper()
	userA, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	member, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, userA.Id)
	require.NoError(t, err)
	roleRecord, err := app.FindFirstRecordByData("orgRoles", "name", role)
	require.NoError(t, err)
	authorization, err := app.FindFirstRecordByData("orgAuthorizations", "user", member.Id)
	require.NoError(t, err)
	authorization.Set("organization", orgID)
	authorization.Set("role", roleRecord.Id)
	require.NoError(t, app.Save(authorization))
	return member
}

func TestHeartbeatStoresActionAuthorSettingsAndReportsUsage(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	registerAgentRunner(t, app, user, true, nil)
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

func TestHeartbeatStoresActionAuthorEnabled(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	registerAgentRunner(t, app, user, true, nil)
	now := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)

	heartbeatAgentRunner(t, app, now, &RunnerActionAuthorSettings{Enabled: true, Model: "m"})
	require.True(t, agentRunnerRecord(t, app).GetBool("agent_enabled"))

	// Old runners omit enabled: their agent is off.
	var settings RunnerActionAuthorSettings
	require.NoError(t, json.Unmarshal([]byte(`{"own_daily_tokens":1,"model":"m"}`), &settings))
	heartbeatAgentRunner(t, app, now, &settings)
	runner := agentRunnerRecord(t, app)
	require.False(t, runner.GetBool("agent_enabled"))
	require.Equal(t, 1, runner.GetInt("agent_own_daily_tokens"))
}

// stubAgentNamespaceReady makes ready the only prepared agent namespace.
func stubAgentNamespaceReady(t *testing.T, ready *string) {
	t.Helper()
	orig := runnerAgentNamespaceReadyFn
	t.Cleanup(func() { runnerAgentNamespaceReadyFn = orig })
	runnerAgentNamespaceReadyFn = func(namespace string) bool { return namespace == *ready }
}

func TestLifecycleResponseCarriesReadyAgentNamespace(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	registerAgentRunner(t, app, user, true, nil)
	namespace := "runner-" + agentRunnerRecord(t, app).Id
	ready := ""
	stubAgentNamespaceReady(t, &ready)
	now := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	enabled := &RunnerActionAuthorSettings{Enabled: true}

	// Enabled but not prepared yet.
	response := heartbeatAgentRunner(t, app, now, enabled)
	require.NotContains(t, response, "agent_namespace")

	ready = namespace
	response = heartbeatAgentRunner(t, app, now, enabled)
	require.Equal(t, namespace, response["agent_namespace"])

	// The heartbeat that turns the agent off gets no namespace.
	response = heartbeatAgentRunner(t, app, now, &RunnerActionAuthorSettings{})
	require.NotContains(t, response, "agent_namespace")

	// A disabled runner gets no namespace.
	heartbeatAgentRunner(t, app, now, enabled)
	runner := agentRunnerRecord(t, app)
	runner.Set("disabled", true)
	require.NoError(t, app.Save(runner))
	require.Empty(t, runnerAgentNamespace(runner))
}

func TestUpsertResponseCarriesReadyAgentNamespace(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	ready := ""
	stubAgentNamespaceReady(t, &ready)

	upsert := func(settings *RunnerActionAuthorSettings) map[string]any {
		event := performMobileRunnerKeyRequest(
			t,
			app,
			user,
			"/api/mobile-runner",
			UpsertMobileRunnerRequest{
				RunnerID:     agentRunnerID,
				Name:         "Agent runner",
				IP:           "https://runner.example",
				Type:         "android_emulator",
				Organization: "usera-s-organization",
				ActionAuthor: settings,
			},
		)
		require.NoError(t, HandleUpsertMobileRunner()(event))
		recorder := responseRecorder(t, event)
		require.Equal(t, http.StatusOK, recorder.Code)
		return decodeJSONBody(t, recorder)
	}

	response := upsert(&RunnerActionAuthorSettings{Enabled: true})
	require.NotContains(t, response, "agent_namespace")

	ready = "runner-" + agentRunnerRecord(t, app).Id
	response = upsert(&RunnerActionAuthorSettings{Enabled: true})
	require.Equal(t, ready, response["agent_namespace"])

	response = upsert(&RunnerActionAuthorSettings{})
	require.NotContains(t, response, "agent_namespace")
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

	registerAgentRunner(t, app, user, true, &RunnerActionAuthorSettings{
		OwnDailyTokens:    0,
		OthersDailyTokens: 5000,
		Provider:          "openai_compatible",
		Model:             "gpt-5",
	})
	runner := agentRunnerRecord(t, app)
	require.Equal(t, "users:"+user.Id, runner.GetString("agent_settings_from"))
	require.Zero(t, runner.GetInt("agent_own_daily_tokens"))
	require.Equal(t, 5000, runner.GetInt("agent_others_daily_tokens"))
	require.Equal(t, "openai_compatible", runner.GetString("agent_provider"))
	require.Equal(t, "gpt-5", runner.GetString("agent_model"))

	registerAgentRunner(t, app, user, true, nil)
	runner = agentRunnerRecord(t, app)
	require.Equal(t, 5000, runner.GetInt("agent_others_daily_tokens"))
	require.Equal(t, "gpt-5", runner.GetString("agent_model"))

	// A token-authenticated upsert stores no settings and keeps the identity.
	registerAgentRunner(t, app, user, false, &RunnerActionAuthorSettings{
		OwnDailyTokens:    1,
		OthersDailyTokens: 999,
		Provider:          "anthropic",
		Model:             "other",
	})
	runner = agentRunnerRecord(t, app)
	require.Equal(t, "users:"+user.Id, runner.GetString("agent_settings_from"))
	require.Zero(t, runner.GetInt("agent_own_daily_tokens"))
	require.Equal(t, 5000, runner.GetInt("agent_others_daily_tokens"))
	require.Equal(t, "gpt-5", runner.GetString("agent_model"))
}

func TestUpsertMobileRunnerKeepsSettingsIdentityFromOtherMembers(t *testing.T) {
	registered := &RunnerActionAuthorSettings{OwnDailyTokens: 100, Model: "gpt-5"}
	claimed := &RunnerActionAuthorSettings{OthersDailyTokens: 999, Model: "claimed"}
	now := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		name  string
		auth  func(t *testing.T, app core.App) *core.Record
		takes bool
	}{
		{
			name: "a member who is not an owner",
			auth: func(t *testing.T, app core.App) *core.Record {
				return joinAgentRunnerOrganization(t, app, "member")
			},
		},
		{
			name: "an owner",
			auth: func(t *testing.T, app core.App) *core.Record {
				return joinAgentRunnerOrganization(t, app, "owner")
			},
			takes: true,
		},
		{
			name: "a superuser",
			auth: func(t *testing.T, app core.App) *core.Record {
				superuser, err := app.FindAuthRecordByEmail(
					core.CollectionNameSuperusers,
					"admin@example.org",
				)
				require.NoError(t, err)
				return superuser
			},
			takes: true,
		},
		{
			name: "the registering user",
			auth: func(t *testing.T, app core.App) *core.Record {
				user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
				require.NoError(t, err)
				return user
			},
			takes: true,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			app := setupMobileRunnerApp(t)
			defer app.Cleanup()
			user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
			require.NoError(t, err)
			registerAgentRunner(t, app, user, true, registered)
			auth := c.auth(t, app)

			registerAgentRunner(t, app, auth, true, claimed)
			runner := agentRunnerRecord(t, app)
			if !c.takes {
				require.Equal(t, "users:"+user.Id, runner.GetString("agent_settings_from"))
				require.Equal(t, 100, runner.GetInt("agent_own_daily_tokens"))
				require.Zero(t, runner.GetInt("agent_others_daily_tokens"))
				require.Equal(t, "gpt-5", runner.GetString("agent_model"))

				// The runner's own heartbeats still set its limits.
				heartbeatAgentRunner(t, app, now, &RunnerActionAuthorSettings{OwnDailyTokens: 7})
				require.Equal(t, 7, agentRunnerRecord(t, app).GetInt("agent_own_daily_tokens"))
				return
			}
			require.Equal(
				t,
				auth.Collection().Name+":"+auth.Id,
				runner.GetString("agent_settings_from"),
			)
			require.Zero(t, runner.GetInt("agent_own_daily_tokens"))
			require.Equal(t, 999, runner.GetInt("agent_others_daily_tokens"))
			require.Equal(t, "claimed", runner.GetString("agent_model"))
		})
	}
}

func TestUpsertMobileRunnerRejectsNegativeActionAuthorLimits(t *testing.T) {
	token := authoringToken(t, "userA")
	(&tests.ApiScenario{
		Method: http.MethodPost,
		URL:    "/api/mobile-runner",
		Body: strings.NewReader(
			`{"name":"Agent runner","ip":"https://runner.example","type":"android_emulator",` +
				`"action_author":{"own_daily_tokens":-1}}`,
		),
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusBadRequest,
		ExpectedContent: []string{`"request.validation"`, `failed on the 'min' tag`},
		TestAppFactory:  setupMobileRunnerApp,
	}).Test(t)
}

func TestHeartbeatIgnoresActionAuthorSettingsOfAnotherIdentity(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	registerAgentRunner(t, app, user, true, &RunnerActionAuthorSettings{
		OwnDailyTokens:    100,
		OthersDailyTokens: 0,
		Model:             "gpt-5",
	})

	member := joinAgentRunnerOrganization(t, app, "member")

	forged := &RunnerActionAuthorSettings{
		OwnDailyTokens:    0,
		OthersDailyTokens: 999,
		Model:             "forged",
	}
	now := time.Date(2026, 11, 5, 9, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		name  string
		auth  *core.Record
		byKey bool
	}{
		{"another member's key", member, true},
		{"the registering user's token", user, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			sendAgentRunnerHeartbeat(t, app, c.auth, c.byKey, now, forged)
			runner := agentRunnerRecord(t, app)
			require.Equal(t, 100, runner.GetInt("agent_own_daily_tokens"))
			require.Zero(t, runner.GetInt("agent_others_daily_tokens"))
			require.Equal(t, "gpt-5", runner.GetString("agent_model"))
		})
	}
}
