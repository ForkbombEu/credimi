// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

const authoringForeignVersion = "userb-s-organization/foreign-wallet/android"

// authoringAdminRunnerRecord returns userA's admin-managed runner.
func authoringAdminRunnerRecord(t testing.TB, app *tests.TestApp) *core.Record {
	t.Helper()
	runner, err := app.FindFirstRecordByData("mobile_runners", "name", "admin-runner")
	require.NoError(t, err)
	return runner
}

// setAuthoringLimits sets the admin runner's daily token limits.
func setAuthoringLimits(t testing.TB, app *tests.TestApp, own, others int) {
	t.Helper()
	runner := authoringAdminRunnerRecord(t, app)
	runner.Set("agent_own_daily_tokens", own)
	runner.Set("agent_others_daily_tokens", others)
	require.NoError(t, app.Save(runner))
}

// reportAuthoringUsage stores, as a trusted heartbeat does, the admin
// runner's usage of day in both pools.
func reportAuthoringUsage(t testing.TB, app *tests.TestApp, day string, own, others int) {
	t.Helper()
	runner := authoringAdminRunnerRecord(t, app)
	runner.Set("agent_usage_day", day)
	runner.Set("agent_own_tokens_used", own)
	runner.Set("agent_others_tokens_used", others)
	require.NoError(t, app.Save(runner))
}

func authoringToday() string {
	return time.Now().UTC().Format("2006-01-02")
}

// shareAdminRunner publishes the admin runner and gives userB's organization
// an Android build to author on it.
func shareAdminRunner(t testing.TB, app *tests.TestApp) {
	t.Helper()
	runner := authoringAdminRunnerRecord(t, app)
	runner.Set("published", true)
	require.NoError(t, app.Save(runner))
	org, err := app.FindFirstRecordByData("organizations", "name", "userB's organization")
	require.NoError(t, err)
	wallets, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(wallets)
	wallet.Set("owner", org.Id)
	wallet.Set("name", "Foreign wallet")
	require.NoError(t, app.Save(wallet))
	versions, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)
	version := core.NewRecord(versions)
	version.Set("owner", org.Id)
	version.Set("wallet", wallet.Id)
	version.Set("tag", "android")
	file, err := filesystem.NewFileFromBytes([]byte("apk"), "android_installer.zip")
	require.NoError(t, err)
	version.Set("android_installer", []*filesystem.File{file})
	require.NoError(t, app.Save(version))
}

func TestActionAuthorStartEnforcesOwnDailyLimit(t *testing.T) {
	token := authoringToken(t, "userA")
	body := startSessionBody(authoringAdminRunner+"/android-emulator", authoringAndroidVersion)
	for _, tc := range []struct {
		name         string
		limit        int
		day          string
		own          int
		status       int
		content      string
		wantEnqueued bool
	}{
		{
			name:         "tokens remain today",
			limit:        1000,
			day:          authoringToday(),
			own:          999,
			status:       http.StatusAccepted,
			content:      `"ticket_id"`,
			wantEnqueued: true,
		},
		{
			name:    "the runner reported today's tokens used",
			limit:   1000,
			day:     authoringToday(),
			own:     1000,
			status:  http.StatusTooManyRequests,
			content: "this runner's daily action author allowance for its organization is used (1000 tokens); it resets at 00:00 UTC",
		},
		{
			name:         "the usage reported is yesterday's",
			limit:        1000,
			day:          time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02"),
			own:          5000,
			status:       http.StatusAccepted,
			content:      `"ticket_id"`,
			wantEnqueued: true,
		},
		{
			name:         "own pool without a limit",
			day:          authoringToday(),
			own:          1_000_000,
			status:       http.StatusAccepted,
			content:      `"ticket_id"`,
			wantEnqueued: true,
		},
	} {
		enqueued := stubActionAuthorQueue(t, true)
		(&tests.ApiScenario{
			Name:            tc.name,
			Method:          http.MethodPost,
			URL:             "/api/action-author/sessions",
			Body:            strings.NewReader(body),
			Headers:         map[string]string{"Authorization": "Bearer " + token},
			ExpectedStatus:  tc.status,
			ExpectedContent: []string{tc.content},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupActionAuthorApp(t)
				setAuthoringLimits(t, app, tc.limit, 0)
				reportAuthoringUsage(t, app, tc.day, tc.own, 0)
				return app
			},
		}).Test(t)
		require.Equal(t, tc.wantEnqueued, enqueued.TicketID != "", tc.name)
	}
}

func TestActionAuthorStartChecksOtherOrganizationsInTheOthersPool(t *testing.T) {
	token := authoringToken(t, "userB")
	body := startSessionBody(authoringAdminRunner+"/android-emulator", authoringForeignVersion)

	enqueued := stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:           "published runner that does not offer the agent",
		Method:         http.MethodPost,
		URL:            "/api/action-author/sessions",
		Body:           strings.NewReader(body),
		Headers:        map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus: http.StatusNotFound,
		ExpectedContent: []string{
			`"device_not_found"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			shareAdminRunner(t, app)
			return app
		},
	}).Test(t)
	require.Empty(t, enqueued.TicketID)

	enqueued = stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:            "offered runner whose own pool is used up",
		Method:          http.MethodPost,
		URL:             "/api/action-author/sessions",
		Body:            strings.NewReader(body),
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"status":"queued"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			shareAdminRunner(t, app)
			setAuthoringLimits(t, app, 100, 500)
			reportAuthoringUsage(t, app, authoringToday(), 100, 499)
			return app
		},
	}).Test(t)
	require.Equal(t, authoringForeignNS, enqueued.OwnerNamespace)

	enqueued = stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:           "others pool used up",
		Method:         http.MethodPost,
		URL:            "/api/action-author/sessions",
		Body:           strings.NewReader(body),
		Headers:        map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus: http.StatusTooManyRequests,
		ExpectedContent: []string{
			`"quota_exceeded"`,
			"this runner's daily action author allowance for other organizations is used (500 tokens); it resets at 00:00 UTC",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			shareAdminRunner(t, app)
			setAuthoringLimits(t, app, 0, 500)
			reportAuthoringUsage(t, app, authoringToday(), 0, 500)
			return app
		},
	}).Test(t)
	require.Empty(t, enqueued.TicketID)
}
