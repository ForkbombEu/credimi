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

// runnerSettingsPrincipal names the caller that may report a runner's action
// author settings: "<auth collection>:<record id>", or "" for a request that
// is not authenticated by a Credimi-Api-Key alone (runners always are).
func runnerSettingsPrincipal(e *core.RequestEvent) string {
	if e.Auth == nil || !authenticatedByAPIKeyOnly(e) {
		return ""
	}
	return e.Auth.Collection().Name + ":" + e.Auth.Id
}

// trustedActionAuthorSettings returns settings only when the caller is the
// identity that last registered the runner; any other caller's values are
// ignored, so the runner's config.toml stays their only source.
func trustedActionAuthorSettings(
	e *core.RequestEvent,
	runner *core.Record,
	s *RunnerActionAuthorSettings,
) *RunnerActionAuthorSettings {
	if s == nil {
		return nil
	}
	principal := runnerSettingsPrincipal(e)
	if principal == "" || principal != runner.GetString("agent_settings_from") {
		return nil
	}
	return s
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
