// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"testing"
	"time"

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

func TestAllowanceForPicksThePoolOfTheOrganization(t *testing.T) {
	app, runner, orgID, otherID := usageFixture(t)
	runner.Set("agent_own_daily_tokens", 100)
	runner.Set("agent_others_daily_tokens", 0)
	require.NoError(t, app.Save(runner))

	own := AllowanceFor(runner, orgID)
	assert.Equal(t, Allowance{Pool: PoolOwn, Tokens: 100}, own)
	assert.True(t, own.Offered())
	assert.True(t, own.Limited())

	others := AllowanceFor(runner, otherID)
	assert.Equal(t, Allowance{Pool: PoolOthers, Tokens: 0}, others)
	assert.False(t, others.Offered())
}

func TestAllowanceExhaustedReadsTodaysReportedUsage(t *testing.T) {
	_, runner, orgID, otherID := usageFixture(t)
	today := usageTestNow.Format("2006-01-02")
	yesterday := usageTestNow.Add(-24 * time.Hour).Format("2006-01-02")
	for name, tc := range map[string]struct {
		orgID       string
		limit       int
		day         string
		own, others int
		want        bool
	}{
		"own at its limit":             {orgID: orgID, limit: 100, day: today, own: 100, want: true},
		"own above its limit":          {orgID: orgID, limit: 100, day: today, own: 150, want: true},
		"own below its limit":          {orgID: orgID, limit: 100, day: today, own: 99},
		"own spends only its own pool": {orgID: orgID, limit: 100, day: today, others: 500},
		"own without a limit":          {orgID: orgID, day: today, own: 1 << 30},
		"own reported yesterday":       {orgID: orgID, limit: 100, day: yesterday, own: 500},
		"nothing reported":             {orgID: orgID, limit: 100},
		"others at its limit":          {orgID: otherID, limit: 100, day: today, others: 100, want: true},
		"others spends only its pool":  {orgID: otherID, limit: 100, day: today, own: 500},
		"others reported yesterday":    {orgID: otherID, limit: 100, day: yesterday, others: 500},
	} {
		t.Run(name, func(t *testing.T) {
			runner.Set("agent_own_daily_tokens", tc.limit)
			runner.Set("agent_others_daily_tokens", tc.limit)
			runner.Set("agent_usage_day", tc.day)
			runner.Set("agent_own_tokens_used", tc.own)
			runner.Set("agent_others_tokens_used", tc.others)

			// The day is UTC: a time in another zone reads the same day.
			now := usageTestNow.In(time.FixedZone("UTC+2", 2*60*60))
			assert.Equal(t, tc.want, AllowanceFor(runner, tc.orgID).Exhausted(runner, now))
		})
	}
}
