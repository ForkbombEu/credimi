// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	usageCollection      = "action_author_usage"
	usageDayLayout       = "2006-01-02"
	usageDefaultSessions = 20
	usageDefaultTokens   = 5000000
	usageSessionsEnv     = "CREDIMI_ACTION_AUTHOR_DAILY_SESSIONS"
	usageTokensEnv       = "CREDIMI_ACTION_AUTHOR_DAILY_TOKENS"
)

// ErrQuotaExceeded reports that an organization used up its daily action author quota.
var ErrQuotaExceeded = errors.New("daily action author quota exceeded")

// Quota bounds an organization's daily action author usage; a value <= 0 is unlimited.
type Quota struct {
	Sessions int
	Tokens   int
}

// UsageDelta is the model usage consumed by an authoring session since its last report.
type UsageDelta struct {
	Requests     int `json:"requests"`
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// UsageState summarizes an organization's remaining daily token budget.
type UsageState struct {
	Exhausted bool `json:"exhausted"`
	// TokensRemaining is -1 when the token quota is unlimited.
	TokensRemaining int `json:"tokens_remaining"`
}

// QuotaFromEnv reads the daily quota from CREDIMI_ACTION_AUTHOR_DAILY_SESSIONS and
// CREDIMI_ACTION_AUTHOR_DAILY_TOKENS, using the defaults when unset or not an integer.
func QuotaFromEnv() Quota {
	return Quota{
		Sessions: usageEnvInt(usageSessionsEnv, usageDefaultSessions),
		Tokens:   usageEnvInt(usageTokensEnv, usageDefaultTokens),
	}
}

func usageEnvInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// ReserveSession counts a new authoring session for orgID on now's UTC day, failing with
// ErrQuotaExceeded when the session or token quota is already used up.
func ReserveSession(app core.App, orgID string, now time.Time, q Quota) error {
	return app.RunInTransaction(func(txApp core.App) error {
		record, err := usageFindOrNew(txApp, orgID, now)
		if err != nil {
			return err
		}
		if q.Sessions > 0 && record.GetInt("sessions") >= q.Sessions {
			return ErrQuotaExceeded
		}
		if q.Tokens > 0 && usageTokens(record) >= q.Tokens {
			return ErrQuotaExceeded
		}
		record.Set("sessions", record.GetInt("sessions")+1)
		if err := txApp.Save(record); err != nil {
			return fmt.Errorf("failed to save action author usage: %w", err)
		}
		return nil
	})
}

// ReleaseSession returns a reserved session for orgID on now's UTC day; it never drops
// below zero and is a no-op when the day has no usage.
func ReleaseSession(app core.App, orgID string, now time.Time) error {
	return app.RunInTransaction(func(txApp core.App) error {
		record, err := usageFind(txApp, orgID, now)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to find action author usage: %w", err)
		}
		record.Set("sessions", max(0, record.GetInt("sessions")-1))
		if err := txApp.Save(record); err != nil {
			return fmt.Errorf("failed to save action author usage: %w", err)
		}
		return nil
	})
}

// RecordUsage adds d to orgID's usage on now's UTC day and reports the remaining token budget.
// Negative delta components are ignored.
func RecordUsage(
	app core.App,
	orgID string,
	now time.Time,
	d UsageDelta,
	q Quota,
) (UsageState, error) {
	var state UsageState
	err := app.RunInTransaction(func(txApp core.App) error {
		record, err := usageFindOrNew(txApp, orgID, now)
		if err != nil {
			return err
		}
		record.Set("input_tokens", record.GetInt("input_tokens")+max(0, d.InputTokens))
		record.Set("output_tokens", record.GetInt("output_tokens")+max(0, d.OutputTokens))
		record.Set("model_calls", record.GetInt("model_calls")+max(0, d.Requests))
		if err := txApp.Save(record); err != nil {
			return fmt.Errorf("failed to save action author usage: %w", err)
		}
		state = usageStateOf(usageTokens(record), q)
		return nil
	})
	if err != nil {
		return UsageState{}, err
	}
	return state, nil
}

func usageStateOf(used int, q Quota) UsageState {
	if q.Tokens <= 0 {
		return UsageState{TokensRemaining: -1}
	}
	return UsageState{
		Exhausted:       used >= q.Tokens,
		TokensRemaining: max(0, q.Tokens-used),
	}
}

func usageTokens(record *core.Record) int {
	return record.GetInt("input_tokens") + record.GetInt("output_tokens")
}

func usageFind(app core.App, orgID string, now time.Time) (*core.Record, error) {
	return app.FindFirstRecordByFilter(
		usageCollection,
		"organization = {:organization} && day = {:day}",
		dbx.Params{"organization": orgID, "day": now.UTC().Format(usageDayLayout)},
	)
}

// usageFindOrNew returns the day's usage row, or an unsaved one when the day has no usage yet.
func usageFindOrNew(app core.App, orgID string, now time.Time) (*core.Record, error) {
	record, err := usageFind(app, orgID, now)
	if err == nil {
		return record, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to find action author usage: %w", err)
	}
	collection, err := app.FindCollectionByNameOrId(usageCollection)
	if err != nil {
		return nil, fmt.Errorf("failed to find action author usage collection: %w", err)
	}
	record = core.NewRecord(collection)
	record.Set("organization", orgID)
	record.Set("day", now.UTC().Format(usageDayLayout))
	return record, nil
}
