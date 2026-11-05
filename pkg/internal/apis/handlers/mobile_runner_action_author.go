// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/pocketbase/pocketbase/core"
)

// RunnerActionAuthorSettings are the action author settings a runner reports
// with registration and every heartbeat: its two daily token limits (0 own:
// no limit; 0 others: not offered to other organizations) and its model.
type RunnerActionAuthorSettings struct {
	OwnDailyTokens    int    `json:"own_daily_tokens"    validate:"min=0"`
	OthersDailyTokens int    `json:"others_daily_tokens" validate:"min=0"`
	Provider          string `json:"provider"`
	Model             string `json:"model"`
}

// RunnerActionAuthorUsage is the runner's action author usage on the UTC day,
// in tokens per pool.
type RunnerActionAuthorUsage struct {
	Day          string `json:"day"`
	OwnTokens    int    `json:"own_tokens"`
	OthersTokens int    `json:"others_tokens"`
}

// setRunnerActionAuthorSettings stores the reported settings; nil leaves the
// stored values unchanged.
func setRunnerActionAuthorSettings(record *core.Record, s *RunnerActionAuthorSettings) {
	if s == nil {
		return
	}
	record.Set("agent_own_daily_tokens", s.OwnDailyTokens)
	record.Set("agent_others_daily_tokens", s.OthersDailyTokens)
	record.Set("agent_provider", s.Provider)
	record.Set("agent_model", s.Model)
}

// runnerActionAuthorUsage returns the runner's usage today, or nil after a
// logged warning: a heartbeat never fails because of usage.
func runnerActionAuthorUsage(
	app core.App,
	runner *core.Record,
	now time.Time,
) *RunnerActionAuthorUsage {
	own, others, err := actionauthor.TodayUsage(app, runner.Id, now)
	if err != nil {
		app.Logger().
			Warn("failed to read runner action author usage", "runner", runner.Id, "error", err)
		return nil
	}
	return &RunnerActionAuthorUsage{
		Day:          now.UTC().Format(time.DateOnly),
		OwnTokens:    own,
		OthersTokens: others,
	}
}
