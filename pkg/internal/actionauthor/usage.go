// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// usageDayLayout is the UTC day the runner reports its usage for.
const usageDayLayout = "2006-01-02"

// Pool is whose sessions a runner's usage counts: the runner's own
// organization, or every other organization together.
type Pool string

// Usage pools of a runner.
const (
	PoolOwn    Pool = "own"
	PoolOthers Pool = "others"
)

// Allowance is a runner's daily token limit for one pool: 0 is no limit for
// PoolOwn and not offered for PoolOthers.
type Allowance struct {
	RunnerID string
	Pool     Pool
	Tokens   int
}

// AllowanceFor returns the runner's allowance for sessions of orgID: the own
// pool when the runner belongs to orgID, the others pool otherwise.
func AllowanceFor(runner *core.Record, orgID string) Allowance {
	if runner.GetString("owner") == orgID {
		return Allowance{
			RunnerID: runner.Id,
			Pool:     PoolOwn,
			Tokens:   runner.GetInt("agent_own_daily_tokens"),
		}
	}
	return Allowance{
		RunnerID: runner.Id,
		Pool:     PoolOthers,
		Tokens:   runner.GetInt("agent_others_daily_tokens"),
	}
}

// Limited reports whether the pool has a daily token limit.
func (a Allowance) Limited() bool {
	return a.Tokens > 0
}

// Offered reports whether the runner offers the agent to the pool at all.
func (a Allowance) Offered() bool {
	return a.Pool == PoolOwn || a.Tokens > 0
}

// Exhausted reports whether the usage the runner reported for today's UTC day
// reached the pool's limit. A pool without a limit is never exhausted.
func (a Allowance) Exhausted(runner *core.Record, now time.Time) bool {
	if !a.Limited() || runner.GetString("agent_usage_day") != now.UTC().Format(usageDayLayout) {
		return false
	}
	field := "agent_own_tokens_used"
	if a.Pool == PoolOthers {
		field = "agent_others_tokens_used"
	}
	return runner.GetInt(field) >= a.Tokens
}
