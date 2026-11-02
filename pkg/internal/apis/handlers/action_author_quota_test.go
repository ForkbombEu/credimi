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
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

// authoringSessionsToday reads userA's organization's sessions of today.
func authoringSessionsToday(t testing.TB, app *tests.TestApp) int {
	t.Helper()
	org, err := app.FindFirstRecordByData("organizations", "name", "userA's organization")
	require.NoError(t, err)
	rows, err := app.FindAllRecords("action_author_usage", dbx.HashExp{
		"organization": org.Id,
		"day":          time.Now().UTC().Format("2006-01-02"),
	})
	require.NoError(t, err)
	if len(rows) == 0 {
		return 0
	}
	return rows[0].GetInt("sessions")
}

// setAuthoringAllowance sets userA's organization's daily action author token allowance.
func setAuthoringAllowance(t testing.TB, app *tests.TestApp, tokens int) {
	t.Helper()
	org, err := app.FindFirstRecordByData("organizations", "name", "userA's organization")
	require.NoError(t, err)
	org.Set("action_author_daily_tokens", tokens)
	require.NoError(t, app.Save(org))
}

// spendAuthoringTokens reserves a session for userA's organization today and records
// tokens of model usage for it.
func spendAuthoringTokens(t testing.TB, app *tests.TestApp, tokens int) {
	t.Helper()
	org, err := app.FindFirstRecordByData("organizations", "name", "userA's organization")
	require.NoError(t, err)
	quota := actionauthor.QuotaForOrganization(org)
	require.NoError(t, actionauthor.ReserveSession(app, org.Id, time.Now(), quota))
	_, err = actionauthor.RecordUsage(app, org.Id, time.Now(), actionauthor.UsageReport{
		ReportID:   "wf/1",
		UsageDelta: actionauthor.UsageDelta{Requests: 1, InputTokens: tokens},
	}, quota)
	require.NoError(t, err)
}

func TestActionAuthorStartEnforcesDailyQuota(t *testing.T) {
	token := authoringToken(t, "userA")
	body := startSessionBody(authoringAdminRunner+"/android-emulator", authoringAndroidVersion)
	limitedApp := func(t testing.TB) *tests.TestApp {
		app := setupActionAuthorApp(t)
		setAuthoringAllowance(t, app, 1000)
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
			require.Equal(t, 1, authoringSessionsToday(t, app))
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
			spendAuthoringTokens(t, app, 999)
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			require.Equal(t, 2, authoringSessionsToday(t, app))
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
			"your organization used today's action author allowance (1000 tokens); it resets at 00:00 UTC",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := limitedApp(t)
			spendAuthoringTokens(t, app, 1000)
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			require.Equal(t, 1, authoringSessionsToday(t, app))
		},
	}).Test(t)
	require.Empty(t, enqueued.TicketID)
}

func TestActionAuthorStartWithoutAllowance(t *testing.T) {
	token := authoringToken(t, "userA")
	enqueued := stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:   "organization left at 0",
		Method: http.MethodPost,
		URL:    "/api/action-author/sessions",
		Body: strings.NewReader(
			startSessionBody(authoringAdminRunner+"/android-emulator", authoringAndroidVersion),
		),
		Headers:        map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus: http.StatusTooManyRequests,
		ExpectedContent: []string{
			`"action_author"`,
			`"quota_exceeded"`,
			"your organization has no action author allowance; ask a Credimi administrator to set one",
		},
		NotExpectedContent: []string{"unlimited", "tokens)"},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			setAuthoringAllowance(t, app, 0)
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			require.Equal(t, 0, authoringSessionsToday(t, app))
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
				require.Equal(t, 0, authoringSessionsToday(t, app), name)
			},
		}).Test(t)
	}
}
