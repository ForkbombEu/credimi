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

// usageFixture returns a runner of userA's organization, and the ids of
// userA's and userB's organizations.
func usageFixture(t *testing.T) (*tests.TestApp, *core.Record, string, string) {
	t.Helper()
	app, err := tests.NewTestApp("../../../test_pb_data")
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	org, err := app.FindFirstRecordByData("organizations", "name", "userA's organization")
	require.NoError(t, err)
	other, err := app.FindFirstRecordByData("organizations", "name", "userB's organization")
	require.NoError(t, err)
	runners, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(runners)
	runner.Set("owner", org.Id)
	runner.Set("name", "usage-runner")
	runner.Set("ip", "https://usage-runner.example.test")
	require.NoError(t, app.Save(runner))
	return app, runner, org.Id, other.Id
}

func usageRow(t *testing.T, app core.App, a Allowance, now time.Time) *core.Record {
	t.Helper()
	record, err := app.FindFirstRecordByFilter(
		"action_author_usage",
		"runner = {:runner} && pool = {:pool} && day = {:day}",
		dbx.Params{
			"runner": a.RunnerID,
			"pool":   string(a.Pool),
			"day":    now.UTC().Format("2006-01-02"),
		},
	)
	require.NoError(t, err)
	return record
}

func remaining(n int) *int { return &n }

// report returns a usage report of one agent turn.
func report(id string, d UsageDelta) UsageReport {
	return UsageReport{ReportID: id, UsageDelta: d}
}

func TestAllowanceForPicksThePoolOfTheOrganization(t *testing.T) {
	app, runner, orgID, otherID := usageFixture(t)
	runner.Set("agent_own_daily_tokens", 100)
	runner.Set("agent_others_daily_tokens", 0)
	require.NoError(t, app.Save(runner))

	own := AllowanceFor(runner, orgID)
	assert.Equal(t, Allowance{RunnerID: runner.Id, Pool: PoolOwn, Tokens: 100}, own)
	assert.True(t, own.Offered())
	assert.True(t, own.Limited())

	others := AllowanceFor(runner, otherID)
	assert.Equal(t, Allowance{RunnerID: runner.Id, Pool: PoolOthers, Tokens: 0}, others)
	assert.False(t, others.Offered())
}

func TestOwnPoolWithoutLimitIsUnlimited(t *testing.T) {
	app, runner, orgID, _ := usageFixture(t)
	a := AllowanceFor(runner, orgID)
	require.False(t, a.Limited())
	require.True(t, a.Offered())

	require.NoError(t, ReserveSession(app, a, usageTestNow))
	state, err := RecordUsage(app, a, usageTestNow,
		report("wf/1", UsageDelta{InputTokens: 1 << 20, OutputTokens: 1 << 20}))
	require.NoError(t, err)
	assert.Equal(t, UsageState{}, state, "no limit: never exhausted, tokens_remaining null")
	require.NoError(t, ReserveSession(app, a, usageTestNow))

	read, err := ReadUsage(app, a, usageTestNow)
	require.NoError(t, err)
	assert.Equal(t, UsageState{}, read)
}

func TestOthersPoolWithoutLimitIsNotOffered(t *testing.T) {
	app, runner, _, otherID := usageFixture(t)
	a := AllowanceFor(runner, otherID)
	require.ErrorIs(t, ReserveSession(app, a, usageTestNow), ErrQuotaExceeded)
	_, err := app.FindFirstRecordByFilter(
		"action_author_usage", "runner = {:runner}", dbx.Params{"runner": runner.Id},
	)
	require.Error(t, err, "a refused reservation creates no row")

	state, err := ReadUsage(app, a, usageTestNow)
	require.NoError(t, err)
	assert.Equal(t, UsageState{Exhausted: true, TokensRemaining: remaining(0)}, state)
	state, err = RecordUsage(app, a, usageTestNow, report("wf/1", UsageDelta{InputTokens: 1}))
	require.NoError(t, err)
	assert.Equal(t, UsageState{Exhausted: true, TokensRemaining: remaining(0)}, state)
}

func TestReserveSessionCountsSessionsWithoutLimitingThem(t *testing.T) {
	app, runner, orgID, _ := usageFixture(t)
	a := Allowance{RunnerID: runner.Id, Pool: PoolOwn, Tokens: 100}
	for range 5 {
		require.NoError(t, ReserveSession(app, a, usageTestNow))
	}
	assert.Equal(t, 5, usageRow(t, app, a, usageTestNow).GetInt("sessions"))

	require.NoError(t, ReleaseSession(app, a, usageTestNow))
	assert.Equal(t, 4, usageRow(t, app, a, usageTestNow).GetInt("sessions"))
	assert.Equal(t, orgID, runner.GetString("owner"))
}

func TestLimitedPoolExhaustion(t *testing.T) {
	app, runner, _, _ := usageFixture(t)
	a := Allowance{RunnerID: runner.Id, Pool: PoolOthers, Tokens: 1000}
	require.NoError(t, ReserveSession(app, a, usageTestNow))

	state, err := RecordUsage(app, a, usageTestNow,
		report("wf/1", UsageDelta{Requests: 1, InputTokens: 400, OutputTokens: 100}))
	require.NoError(t, err)
	assert.Equal(t, UsageState{TokensRemaining: remaining(500)}, state)
	read, err := ReadUsage(app, a, usageTestNow)
	require.NoError(t, err)
	assert.Equal(t, state, read)
	require.NoError(t, ReserveSession(app, a, usageTestNow), "500 of 1000 tokens leave room")

	state, err = RecordUsage(app, a, usageTestNow,
		report("wf/2", UsageDelta{Requests: 2, InputTokens: 450, OutputTokens: 200}))
	require.NoError(t, err)
	assert.Equal(t, UsageState{Exhausted: true, TokensRemaining: remaining(0)}, state)

	row := usageRow(t, app, a, usageTestNow)
	assert.Equal(t, 850, row.GetInt("input_tokens"))
	assert.Equal(t, 300, row.GetInt("output_tokens"))
	assert.Equal(t, 3, row.GetInt("model_calls"))

	require.ErrorIs(t, ReserveSession(app, a, usageTestNow), ErrQuotaExceeded)
	assert.Equal(t, 2, usageRow(t, app, a, usageTestNow).GetInt("sessions"))
}

func TestPoolsAreCountedApart(t *testing.T) {
	app, runner, _, _ := usageFixture(t)
	own := Allowance{RunnerID: runner.Id, Pool: PoolOwn, Tokens: 100}
	others := Allowance{RunnerID: runner.Id, Pool: PoolOthers, Tokens: 100}
	_, err := RecordUsage(app, own, usageTestNow, report("wf/1", UsageDelta{InputTokens: 100}))
	require.NoError(t, err)
	require.ErrorIs(t, ReserveSession(app, own, usageTestNow), ErrQuotaExceeded)
	require.NoError(t, ReserveSession(app, others, usageTestNow))
	_, err = RecordUsage(app, others, usageTestNow, report("wf/1", UsageDelta{OutputTokens: 7}))
	require.NoError(t, err, "report IDs are kept per pool")

	ownTokens, othersTokens, err := TodayUsage(app, runner.Id, usageTestNow)
	require.NoError(t, err)
	assert.Equal(t, 100, ownTokens)
	assert.Equal(t, 7, othersTokens)

	ownTokens, othersTokens, err = TodayUsage(app, runner.Id, usageTestNow.Add(time.Hour))
	require.NoError(t, err)
	assert.Zero(t, ownTokens)
	assert.Zero(t, othersTokens)
}

func TestReleaseSessionFloorsAtZero(t *testing.T) {
	app, runner, _, _ := usageFixture(t)
	a := Allowance{RunnerID: runner.Id, Pool: PoolOwn, Tokens: 5}
	require.NoError(t, ReleaseSession(app, a, usageTestNow))
	_, err := app.FindFirstRecordByFilter(
		"action_author_usage", "runner = {:runner}", dbx.Params{"runner": runner.Id},
	)
	require.Error(t, err, "releasing without usage must not create a row")

	require.NoError(t, ReserveSession(app, a, usageTestNow))
	require.NoError(t, ReleaseSession(app, a, usageTestNow))
	require.NoError(t, ReleaseSession(app, a, usageTestNow))
	assert.Equal(t, 0, usageRow(t, app, a, usageTestNow).GetInt("sessions"))
}

func TestRecordUsageCountsAReportOnce(t *testing.T) {
	app, runner, _, _ := usageFixture(t)
	a := Allowance{RunnerID: runner.Id, Pool: PoolOwn, Tokens: 1000}
	first, err := RecordUsage(app, a, usageTestNow,
		report("wf/1", UsageDelta{Requests: 1, InputTokens: 300, OutputTokens: 100}))
	require.NoError(t, err)
	assert.Equal(t, UsageState{TokensRemaining: remaining(600)}, first)

	repeat, err := RecordUsage(app, a, usageTestNow,
		report("wf/1", UsageDelta{Requests: 1, InputTokens: 300, OutputTokens: 100}))
	require.NoError(t, err)
	assert.Equal(t, first, repeat)
	row := usageRow(t, app, a, usageTestNow)
	assert.Equal(t, 300, row.GetInt("input_tokens"))
	assert.Equal(t, 1, row.GetInt("model_calls"))

	// The same report ID counts again on another day: IDs are kept per day.
	nextDay := usageTestNow.Add(time.Hour)
	state, err := RecordUsage(app, a, nextDay, report("wf/1", UsageDelta{InputTokens: 10}))
	require.NoError(t, err)
	assert.Equal(t, UsageState{TokensRemaining: remaining(990)}, state)
}

func TestRecordUsageKeepsTheLastReportIDs(t *testing.T) {
	app, runner, _, _ := usageFixture(t)
	a := Allowance{RunnerID: runner.Id, Pool: PoolOwn}
	for i := range usageReportIDs + 1 {
		_, err := RecordUsage(app, a, usageTestNow,
			report(fmt.Sprintf("wf/%d", i), UsageDelta{InputTokens: 1}))
		require.NoError(t, err)
	}
	var ids []string
	require.NoError(t, usageRow(t, app, a, usageTestNow).UnmarshalJSONField("report_ids", &ids))
	require.Len(t, ids, usageReportIDs)
	assert.Equal(t, "wf/1", ids[0])
	assert.Equal(t, fmt.Sprintf("wf/%d", usageReportIDs), ids[len(ids)-1])

	// wf/0 was forgotten, so it counts again.
	_, err := RecordUsage(app, a, usageTestNow, report("wf/0", UsageDelta{InputTokens: 1}))
	require.NoError(t, err)
	assert.Equal(t, usageReportIDs+2, usageRow(t, app, a, usageTestNow).GetInt("input_tokens"))
}

func TestRecordUsageClampsNegativeDeltas(t *testing.T) {
	app, runner, _, _ := usageFixture(t)
	a := Allowance{RunnerID: runner.Id, Pool: PoolOwn, Tokens: 100}
	_, err := RecordUsage(app, a, usageTestNow,
		report("wf/1", UsageDelta{Requests: 1, InputTokens: 10, OutputTokens: 10}))
	require.NoError(t, err)
	state, err := RecordUsage(app, a, usageTestNow,
		report("wf/2", UsageDelta{Requests: -5, InputTokens: -50, OutputTokens: -50}))
	require.NoError(t, err)
	assert.Equal(t, UsageState{TokensRemaining: remaining(80)}, state)
	assert.Equal(t, 1, usageRow(t, app, a, usageTestNow).GetInt("model_calls"))
}

func TestUsageNewUTCDayResets(t *testing.T) {
	app, runner, _, _ := usageFixture(t)
	a := Allowance{RunnerID: runner.Id, Pool: PoolOwn, Tokens: 10}
	require.NoError(t, ReserveSession(app, a, usageTestNow))
	_, err := RecordUsage(app, a, usageTestNow, report("wf/1", UsageDelta{InputTokens: 10}))
	require.NoError(t, err)
	require.ErrorIs(t, ReserveSession(app, a, usageTestNow), ErrQuotaExceeded)

	// 23:30 UTC is already the next UTC day at 01:30 in UTC+2.
	nextDay := usageTestNow.Add(time.Hour).In(time.FixedZone("UTC+2", 2*60*60))
	require.NoError(t, ReserveSession(app, a, nextDay))
	state, err := RecordUsage(app, a, nextDay, report("wf/2", UsageDelta{InputTokens: 3}))
	require.NoError(t, err)
	assert.Equal(t, UsageState{TokensRemaining: remaining(7)}, state)
	assert.Equal(t, "2026-11-03", usageRow(t, app, a, nextDay).GetString("day"))

	// A timestamp in another zone that is still the same UTC day hits the original row.
	sameDay := usageTestNow.In(time.FixedZone("UTC+2", 2*60*60))
	require.ErrorIs(t, ReserveSession(app, a, sameDay), ErrQuotaExceeded)
}
