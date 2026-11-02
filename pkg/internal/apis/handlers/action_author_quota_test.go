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

func TestActionAuthorStartEnforcesDailyQuota(t *testing.T) {
	t.Setenv("CREDIMI_ACTION_AUTHOR_DAILY_SESSIONS", "1")
	t.Setenv("CREDIMI_ACTION_AUTHOR_DAILY_TOKENS", "1000")
	token := authoringToken(t, "userA")
	body := startSessionBody(authoringAdminRunner+"/android-emulator", authoringAndroidVersion)

	enqueued := stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:            "first session of the day",
		Method:          http.MethodPost,
		URL:             "/api/action-author/sessions",
		Body:            strings.NewReader(body),
		Headers:         map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus:  http.StatusAccepted,
		ExpectedContent: []string{`"ticket_id"`},
		TestAppFactory:  setupActionAuthorApp,
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			require.Equal(t, 1, authoringSessionsToday(t, app))
		},
	}).Test(t)
	require.NotEmpty(t, enqueued.TicketID)

	enqueued = stubActionAuthorQueue(t, true)
	(&tests.ApiScenario{
		Name:           "second session of the day",
		Method:         http.MethodPost,
		URL:            "/api/action-author/sessions",
		Body:           strings.NewReader(body),
		Headers:        map[string]string{"Authorization": "Bearer " + token},
		ExpectedStatus: http.StatusTooManyRequests,
		ExpectedContent: []string{
			`"quota_exceeded"`,
			"your organization reached today's action author limit (1 sessions, 1000 tokens); it resets at 00:00 UTC",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupActionAuthorApp(t)
			org, err := app.FindFirstRecordByData("organizations", "name", "userA's organization")
			require.NoError(t, err)
			require.NoError(t, actionauthor.ReserveSession(
				app, org.Id, time.Now(), actionauthor.QuotaFromEnv(),
			))
			return app
		},
	}).Test(t)
	require.Empty(t, enqueued.TicketID)
}

func TestActionAuthorStartReleasesQuotaWhenEnqueueFails(t *testing.T) {
	t.Setenv("CREDIMI_ACTION_AUTHOR_DAILY_SESSIONS", "1")
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
