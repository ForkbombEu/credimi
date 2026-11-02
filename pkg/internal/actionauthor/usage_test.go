// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"fmt"
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

	require.NoError(t, ReserveSession(app, orgID, usageTestNow, Quota{Sessions: 5, Tokens: 5}))
	require.NoError(t, ReleaseSession(app, orgID, usageTestNow))
	require.NoError(t, ReleaseSession(app, orgID, usageTestNow))
	assert.Equal(t, 0, usageRow(t, app, orgID, usageTestNow).GetInt("sessions"))
}

// report returns a usage report of one agent turn.
func report(id string, d UsageDelta) UsageReport {
	return UsageReport{ReportID: id, UsageDelta: d}
}

func TestRecordUsageTokenExhaustion(t *testing.T) {
	app, orgID := usageFixture(t)
	q := Quota{Sessions: 10, Tokens: 1000}
	require.NoError(t, ReserveSession(app, orgID, usageTestNow, q))

	state, err := RecordUsage(
		app,
		orgID,
		usageTestNow,
		report("wf/1", UsageDelta{Requests: 1, InputTokens: 400, OutputTokens: 100}),
		q,
	)
	require.NoError(t, err)
	assert.Equal(t, UsageState{Exhausted: false, TokensRemaining: 500}, state)

	state, err = RecordUsage(
		app,
		orgID,
		usageTestNow,
		report("wf/2", UsageDelta{Requests: 2, InputTokens: 450, OutputTokens: 200}),
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

func TestRecordUsageCountsAReportOnce(t *testing.T) {
	app, orgID := usageFixture(t)
	q := Quota{Sessions: 10, Tokens: 1000}
	first, err := RecordUsage(app, orgID, usageTestNow,
		report("wf/1", UsageDelta{Requests: 1, InputTokens: 300, OutputTokens: 100}), q)
	require.NoError(t, err)
	assert.Equal(t, UsageState{TokensRemaining: 600}, first)

	repeat, err := RecordUsage(app, orgID, usageTestNow,
		report("wf/1", UsageDelta{Requests: 1, InputTokens: 300, OutputTokens: 100}), q)
	require.NoError(t, err)
	assert.Equal(t, first, repeat)
	row := usageRow(t, app, orgID, usageTestNow)
	assert.Equal(t, 300, row.GetInt("input_tokens"))
	assert.Equal(t, 1, row.GetInt("model_calls"))

	// The same report ID counts again on another day: IDs are kept per day.
	nextDay := usageTestNow.Add(time.Hour)
	state, err := RecordUsage(app, orgID, nextDay,
		report("wf/1", UsageDelta{InputTokens: 10}), q)
	require.NoError(t, err)
	assert.Equal(t, UsageState{TokensRemaining: 990}, state)
}

func TestRecordUsageKeepsTheLastReportIDs(t *testing.T) {
	app, orgID := usageFixture(t)
	q := Quota{Sessions: 10, Tokens: 1 << 30}
	for i := range usageReportIDs + 1 {
		_, err := RecordUsage(app, orgID, usageTestNow,
			report(fmt.Sprintf("wf/%d", i), UsageDelta{InputTokens: 1}), q)
		require.NoError(t, err)
	}
	var ids []string
	require.NoError(t, usageRow(t, app, orgID, usageTestNow).UnmarshalJSONField("report_ids", &ids))
	require.Len(t, ids, usageReportIDs)
	assert.Equal(t, "wf/1", ids[0])
	assert.Equal(t, fmt.Sprintf("wf/%d", usageReportIDs), ids[len(ids)-1])

	// wf/0 was forgotten, so it counts again.
	_, err := RecordUsage(app, orgID, usageTestNow, report("wf/0", UsageDelta{InputTokens: 1}), q)
	require.NoError(t, err)
	assert.Equal(t, usageReportIDs+2, usageRow(t, app, orgID, usageTestNow).GetInt("input_tokens"))
}

func TestRecordUsageClampsNegativeDeltas(t *testing.T) {
	app, orgID := usageFixture(t)
	q := Quota{Sessions: 1, Tokens: 100}
	_, err := RecordUsage(
		app,
		orgID,
		usageTestNow,
		report("wf/1", UsageDelta{Requests: 1, InputTokens: 10, OutputTokens: 10}),
		q,
	)
	require.NoError(t, err)
	state, err := RecordUsage(
		app,
		orgID,
		usageTestNow,
		report("wf/2", UsageDelta{Requests: -5, InputTokens: -50, OutputTokens: -50}),
		q,
	)
	require.NoError(t, err)
	assert.Equal(t, UsageState{TokensRemaining: 80}, state)
	assert.Equal(t, 1, usageRow(t, app, orgID, usageTestNow).GetInt("model_calls"))
}

func TestNoAllowance(t *testing.T) {
	for _, q := range []Quota{{}, {Sessions: 5}, {Tokens: 5}, {Sessions: -1, Tokens: -1}} {
		app, orgID := usageFixture(t)
		assert.False(t, q.HasAllowance())
		require.ErrorIs(t, ReserveSession(app, orgID, usageTestNow, q), ErrQuotaExceeded)
		state, err := RecordUsage(app, orgID, usageTestNow,
			report("wf/1", UsageDelta{InputTokens: 1}), q)
		require.NoError(t, err)
		assert.True(t, state.Exhausted)
		assert.Equal(t, 0, state.TokensRemaining)
	}
	assert.True(t, Quota{Sessions: 1, Tokens: 1}.HasAllowance())
}

func TestQuotaForOrganization(t *testing.T) {
	app, orgID := usageFixture(t)
	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	assert.Equal(t, Quota{}, QuotaForOrganization(org), "organizations start without allowance")
	org.Set("action_author_daily_sessions", 3)
	org.Set("action_author_daily_tokens", 1234)
	assert.Equal(t, Quota{Sessions: 3, Tokens: 1234}, QuotaForOrganization(org))
}

func TestUsageNewUTCDayResets(t *testing.T) {
	app, orgID := usageFixture(t)
	q := Quota{Sessions: 1, Tokens: 10}
	require.NoError(t, ReserveSession(app, orgID, usageTestNow, q))
	_, err := RecordUsage(app, orgID, usageTestNow, report("wf/1", UsageDelta{InputTokens: 10}), q)
	require.NoError(t, err)
	require.ErrorIs(t, ReserveSession(app, orgID, usageTestNow, q), ErrQuotaExceeded)

	// 23:30 UTC is already the next UTC day at 01:30 in UTC+2.
	nextDay := usageTestNow.Add(time.Hour).In(time.FixedZone("UTC+2", 2*60*60))
	require.NoError(t, ReserveSession(app, orgID, nextDay, q))
	state, err := RecordUsage(app, orgID, nextDay, report("wf/2", UsageDelta{InputTokens: 3}), q)
	require.NoError(t, err)
	assert.Equal(t, UsageState{TokensRemaining: 7}, state)
	assert.Equal(t, "2026-11-03", usageRow(t, app, orgID, nextDay).GetString("day"))

	// A timestamp in another zone that is still the same UTC day hits the original row.
	sameDay := usageTestNow.In(time.FixedZone("UTC+2", 2*60*60))
	require.ErrorIs(t, ReserveSession(app, orgID, sameDay, q), ErrQuotaExceeded)
}
