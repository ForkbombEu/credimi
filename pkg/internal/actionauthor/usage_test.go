// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"os"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var usageTestNow = time.Date(2026, 11, 2, 23, 30, 0, 0, time.UTC)

func usageFixture(t *testing.T) (*tests.TestApp, string) {
	t.Helper()
	app, err := tests.NewTestApp("../../../test_pb_data")
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	org, err := app.FindFirstRecordByData("organizations", "name", "userA's organization")
	require.NoError(t, err)
	return app, org.Id
}

func usageRow(t *testing.T, app core.App, orgID string, now time.Time) *core.Record {
	t.Helper()
	record, err := app.FindFirstRecordByFilter(
		"action_author_usage",
		"organization = {:organization} && day = {:day}",
		dbx.Params{"organization": orgID, "day": now.UTC().Format("2006-01-02")},
	)
	require.NoError(t, err)
	return record
}

func TestReserveSessionUpToLimit(t *testing.T) {
	app, orgID := usageFixture(t)
	q := Quota{Sessions: 2, Tokens: 100}
	require.NoError(t, ReserveSession(app, orgID, usageTestNow, q))
	require.NoError(t, ReserveSession(app, orgID, usageTestNow, q))
	require.ErrorIs(t, ReserveSession(app, orgID, usageTestNow, q), ErrQuotaExceeded)
	assert.Equal(t, 2, usageRow(t, app, orgID, usageTestNow).GetInt("sessions"))

	require.NoError(t, ReleaseSession(app, orgID, usageTestNow))
	require.NoError(t, ReserveSession(app, orgID, usageTestNow, q))
	require.ErrorIs(t, ReserveSession(app, orgID, usageTestNow, q), ErrQuotaExceeded)
}

func TestReleaseSessionFloorsAtZero(t *testing.T) {
	app, orgID := usageFixture(t)
	require.NoError(t, ReleaseSession(app, orgID, usageTestNow))
	_, err := app.FindFirstRecordByFilter(
		"action_author_usage",
		"organization = {:organization}",
		dbx.Params{"organization": orgID},
	)
	require.Error(t, err, "releasing without usage must not create a row")

	require.NoError(t, ReserveSession(app, orgID, usageTestNow, Quota{Sessions: 5}))
	require.NoError(t, ReleaseSession(app, orgID, usageTestNow))
	require.NoError(t, ReleaseSession(app, orgID, usageTestNow))
	assert.Equal(t, 0, usageRow(t, app, orgID, usageTestNow).GetInt("sessions"))
}

func TestRecordUsageTokenExhaustion(t *testing.T) {
	app, orgID := usageFixture(t)
	q := Quota{Sessions: 10, Tokens: 1000}
	require.NoError(t, ReserveSession(app, orgID, usageTestNow, q))

	state, err := RecordUsage(
		app,
		orgID,
		usageTestNow,
		UsageDelta{Requests: 1, InputTokens: 400, OutputTokens: 100},
		q,
	)
	require.NoError(t, err)
	assert.Equal(t, UsageState{Exhausted: false, TokensRemaining: 500}, state)

	state, err = RecordUsage(
		app,
		orgID,
		usageTestNow,
		UsageDelta{Requests: 2, InputTokens: 450, OutputTokens: 200},
		q,
	)
	require.NoError(t, err)
	assert.Equal(t, UsageState{Exhausted: true, TokensRemaining: 0}, state)

	row := usageRow(t, app, orgID, usageTestNow)
	assert.Equal(t, 850, row.GetInt("input_tokens"))
	assert.Equal(t, 300, row.GetInt("output_tokens"))
	assert.Equal(t, 3, row.GetInt("model_calls"))

	require.ErrorIs(t, ReserveSession(app, orgID, usageTestNow, q), ErrQuotaExceeded)
	assert.Equal(t, 1, usageRow(t, app, orgID, usageTestNow).GetInt("sessions"))
}

func TestRecordUsageClampsNegativeDeltas(t *testing.T) {
	app, orgID := usageFixture(t)
	q := Quota{Tokens: 100}
	_, err := RecordUsage(
		app,
		orgID,
		usageTestNow,
		UsageDelta{Requests: 1, InputTokens: 10, OutputTokens: 10},
		q,
	)
	require.NoError(t, err)
	state, err := RecordUsage(
		app,
		orgID,
		usageTestNow,
		UsageDelta{Requests: -5, InputTokens: -50, OutputTokens: -50},
		q,
	)
	require.NoError(t, err)
	assert.Equal(t, UsageState{TokensRemaining: 80}, state)
	assert.Equal(t, 1, usageRow(t, app, orgID, usageTestNow).GetInt("model_calls"))
}

func TestRecordUsageUnlimitedQuota(t *testing.T) {
	app, orgID := usageFixture(t)
	q := Quota{Sessions: -1, Tokens: 0}
	state, err := RecordUsage(
		app,
		orgID,
		usageTestNow,
		UsageDelta{Requests: 1, InputTokens: 1 << 30, OutputTokens: 1 << 20},
		q,
	)
	require.NoError(t, err)
	assert.Equal(t, UsageState{Exhausted: false, TokensRemaining: -1}, state)
	for range 30 {
		require.NoError(t, ReserveSession(app, orgID, usageTestNow, q))
	}
}

func TestUsageNewUTCDayResets(t *testing.T) {
	app, orgID := usageFixture(t)
	q := Quota{Sessions: 1, Tokens: 10}
	require.NoError(t, ReserveSession(app, orgID, usageTestNow, q))
	_, err := RecordUsage(app, orgID, usageTestNow, UsageDelta{InputTokens: 10}, q)
	require.NoError(t, err)
	require.ErrorIs(t, ReserveSession(app, orgID, usageTestNow, q), ErrQuotaExceeded)

	// 23:30 UTC is already the next UTC day at 01:30 in UTC+2.
	nextDay := usageTestNow.Add(time.Hour).In(time.FixedZone("UTC+2", 2*60*60))
	require.NoError(t, ReserveSession(app, orgID, nextDay, q))
	state, err := RecordUsage(app, orgID, nextDay, UsageDelta{InputTokens: 3}, q)
	require.NoError(t, err)
	assert.Equal(t, UsageState{TokensRemaining: 7}, state)
	assert.Equal(t, "2026-11-03", usageRow(t, app, orgID, nextDay).GetString("day"))

	// A timestamp in another zone that is still the same UTC day hits the original row.
	sameDay := usageTestNow.In(time.FixedZone("UTC+2", 2*60*60))
	require.ErrorIs(t, ReserveSession(app, orgID, sameDay, q), ErrQuotaExceeded)
}

func TestQuotaFromEnv(t *testing.T) {
	cases := []struct {
		name     string
		sessions string
		tokens   string
		want     Quota
	}{
		{name: "defaults", want: Quota{Sessions: 20, Tokens: 5000000}},
		{
			name:     "overrides",
			sessions: "3",
			tokens:   "1234",
			want:     Quota{Sessions: 3, Tokens: 1234},
		},
		{
			name:     "unlimited",
			sessions: "0",
			tokens:   "-1",
			want:     Quota{Sessions: 0, Tokens: -1},
		},
		{
			name:     "invalid",
			sessions: "many",
			tokens:   "1.5",
			want:     Quota{Sessions: 20, Tokens: 5000000},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			usageSetenv(t, "CREDIMI_ACTION_AUTHOR_DAILY_SESSIONS", tc.sessions)
			usageSetenv(t, "CREDIMI_ACTION_AUTHOR_DAILY_TOKENS", tc.tokens)
			assert.Equal(t, tc.want, QuotaFromEnv())
		})
	}
}

// usageSetenv sets key to value for the test, or unsets it when value is empty.
func usageSetenv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
	if value == "" {
		require.NoError(t, os.Unsetenv(key))
	}
}
