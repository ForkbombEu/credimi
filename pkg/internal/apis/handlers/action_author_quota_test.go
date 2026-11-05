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
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesemaphore"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

const authoringForeignVersion = "userb-s-organization/foreign-wallet/android"

// authoringAdminRunnerRecord returns userA's admin-managed runner.
func authoringAdminRunnerRecord(t testing.TB, app *tests.TestApp) *core.Record {
	t.Helper()
	runner, err := app.FindFirstRecordByData("mobile_runners", "name", "admin-runner")
	require.NoError(t, err)
	return runner
}

// authoringSessionsToday reads the admin runner's sessions of today in pool.
func authoringSessionsToday(t testing.TB, app *tests.TestApp, pool actionauthor.Pool) int {
	t.Helper()
	rows, err := app.FindAllRecords("action_author_usage", dbx.HashExp{
		"runner": authoringAdminRunnerRecord(t, app).Id,
		"pool":   string(pool),
		"day":    time.Now().UTC().Format("2006-01-02"),
	})
	require.NoError(t, err)
	if len(rows) == 0 {
		return 0
	}
	return rows[0].GetInt("sessions")
}

// setAuthoringLimits sets the admin runner's daily token limits.
func setAuthoringLimits(t testing.TB, app *tests.TestApp, own, others int) {
	t.Helper()
	runner := authoringAdminRunnerRecord(t, app)
	runner.Set("agent_own_daily_tokens", own)
	runner.Set("agent_others_daily_tokens", others)
	require.NoError(t, app.Save(runner))
}

// spendAuthoringTokens reserves a session in the admin runner's pool today,
// under the runner's stored limits, and records tokens of model usage for it.
func spendAuthoringTokens(t testing.TB, app *tests.TestApp, pool actionauthor.Pool, tokens int) {
	t.Helper()
	runner := authoringAdminRunnerRecord(t, app)
	orgID := runner.GetString("owner")
	if pool == actionauthor.PoolOthers {
		orgID = "another-organization"
	}
	a := actionauthor.AllowanceFor(runner, orgID)
	require.NoError(t, actionauthor.ReserveSession(app, a, time.Now()))
	_, err := actionauthor.RecordUsage(app, a, time.Now(), actionauthor.UsageReport{
		ReportID:   "wf/1",
		UsageDelta: actionauthor.UsageDelta{Requests: 1, InputTokens: tokens},
	})
	require.NoError(t, err)
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
	limitedApp := func(t testing.TB) *tests.TestApp {
		app := setupActionAuthorApp(t)
		setAuthoringLimits(t, app, 1000, 0)
		return app
	}

	enqueued := stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:            "first session of the day",
		Method:          http.MethodPost,
		URL:             "/api/action-author/sessions",
		Body:            strings.NewReader(body),
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"ticket_id"`},
		TestAppFactory:  limitedApp,
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			require.Equal(t, 1, authoringSessionsToday(t, app, actionauthor.PoolOwn))
		},
	}).Test(t)
	require.NotEmpty(t, enqueued.TicketID)

	enqueued = stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:            "another session while tokens remain",
		Method:          http.MethodPost,
		URL:             "/api/action-author/sessions",
		Body:            strings.NewReader(body),
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"ticket_id"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := limitedApp(t)
			spendAuthoringTokens(t, app, actionauthor.PoolOwn, 999)
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			require.Equal(t, 2, authoringSessionsToday(t, app, actionauthor.PoolOwn))
		},
	}).Test(t)
	require.NotEmpty(t, enqueued.TicketID)

	enqueued = stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:           "session after the day's tokens are used",
		Method:         http.MethodPost,
		URL:            "/api/action-author/sessions",
		Body:           strings.NewReader(body),
		Headers:        map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus: http.StatusTooManyRequests,
		ExpectedContent: []string{
			`"quota_exceeded"`,
			"this runner's daily action author allowance for its organization is used (1000 tokens); it resets at 00:00 UTC",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := limitedApp(t)
			spendAuthoringTokens(t, app, actionauthor.PoolOwn, 1000)
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			require.Equal(t, 1, authoringSessionsToday(t, app, actionauthor.PoolOwn))
		},
	}).Test(t)
	require.Empty(t, enqueued.TicketID)

	enqueued = stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:            "own pool without a limit",
		Method:          http.MethodPost,
		URL:             "/api/action-author/sessions",
		Body:            strings.NewReader(body),
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"ticket_id"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			spendAuthoringTokens(t, app, actionauthor.PoolOwn, 1_000_000)
			return app
		},
	}).Test(t)
	require.NotEmpty(t, enqueued.TicketID)
}

func TestActionAuthorStartCountsOtherOrganizationsInTheOthersPool(t *testing.T) {
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
		Name:            "offered runner",
		Method:          http.MethodPost,
		URL:             "/api/action-author/sessions",
		Body:            strings.NewReader(body),
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"workflow_namespace":"` + authoringForeignNS + `"`},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			shareAdminRunner(t, app)
			setAuthoringLimits(t, app, 0, 500)
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			require.Equal(t, 1, authoringSessionsToday(t, app, actionauthor.PoolOthers))
			require.Equal(t, 0, authoringSessionsToday(t, app, actionauthor.PoolOwn))
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
			spendAuthoringTokens(t, app, actionauthor.PoolOthers, 500)
			return app
		},
	}).Test(t)
	require.Empty(t, enqueued.TicketID)
}

func TestActionAuthorStartReleasesQuotaWhenEnqueueFails(t *testing.T) {
	token := authoringToken(t, "userA")
	for name, tc := range map[string]struct {
		enqueueErr error
		status     int
	}{
		"queue limit": {
			enqueueErr: temporal.NewApplicationError(
				"queue limit exceeded for runner admin-runner: 1 of 1",
				mobiledevicesemaphore.ErrQueueLimitExceeded,
			),
			status: http.StatusConflict,
		},
		"start failed": {
			enqueueErr: temporal.NewApplicationError("update failed", "test"),
			status:     http.StatusServiceUnavailable,
		},
	} {
		stubActionAuthorQueue(t, true)
		origCancel := cancelRunTicket
		t.Cleanup(func() { cancelRunTicket = origCancel })
		cancelRunTicket = func(
			context.Context,
			string,
			mobiledevicesemaphore.MobileDeviceSemaphoreRunCancelRequest,
		) (mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView, error) {
			return mobiledevicesemaphore.MobileDeviceSemaphoreRunStatusView{}, nil
		}
		enqueueActionAuthorTicket = func(
			context.Context,
			client.Client,
			string,
			mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunRequest,
		) (mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse, error) {
			return mobiledevicesemaphore.MobileDeviceSemaphoreEnqueueRunResponse{}, tc.enqueueErr
		}
		(&tests.ApiScenario{
			Name:   name,
			Method: http.MethodPost,
			URL:    "/api/action-author/sessions",
			Body: strings.NewReader(
				startSessionBody(authoringAdminRunner+"/android-emulator", authoringAndroidVersion),
			),
			Headers:         map[string]string{"Authorization": "Bearer " + token},
			ExpectedStatus:  tc.status,
			ExpectedContent: []string{`"error"`},
			TestAppFactory:  setupActionAuthorApp,
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
				require.Equal(t, 0, authoringSessionsToday(t, app, actionauthor.PoolOwn), name)
			},
		}).Test(t)
	}
}
