// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	usageCollection = "action_author_usage"
	usageDayLayout  = "2006-01-02"
	// usageReportIDs is how many report IDs a day's row remembers.
	usageReportIDs = 500
)

// ErrQuotaExceeded reports that a runner's daily action author allowance for
// a pool is used up, or that the runner does not offer the agent to the pool.
var ErrQuotaExceeded = errors.New("daily action author quota exceeded")

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

// UsageDelta is the model usage consumed by an authoring session since its last report.
type UsageDelta struct {
	Requests     int `json:"requests"`
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// UsageReport is one usage report of an agent turn. ReportID identifies the
// turn, so a retried report counts once.
type UsageReport struct {
	ReportID string `json:"report_id"`
	UsageDelta
}

// UsageState summarizes a pool's remaining daily tokens; TokensRemaining is
// nil when the pool has no limit.
type UsageState struct {
	Exhausted       bool `json:"exhausted"`
	TokensRemaining *int `json:"tokens_remaining"`
}

// ReserveSession counts a new authoring session in a's pool on now's UTC day,
// failing with ErrQuotaExceeded when the pool is not offered or its tokens are
// used up. Sessions are only recorded, never limited.
func ReserveSession(app core.App, a Allowance, now time.Time) error {
	if !a.Offered() {
		return ErrQuotaExceeded
	}
	return app.RunInTransaction(func(txApp core.App) error {
		record, err := usageFindOrNew(txApp, a, now)
		if err != nil {
			return err
		}
		if a.Limited() && usageTokens(record) >= a.Tokens {
			return ErrQuotaExceeded
		}
		record.Set("sessions", record.GetInt("sessions")+1)
		if err := txApp.Save(record); err != nil {
			return fmt.Errorf("failed to save action author usage: %w", err)
		}
		return nil
	})
}

// ReleaseSession returns a reserved session of a's pool on now's UTC day; it
// never drops below zero and is a no-op when the day has no usage.
func ReleaseSession(app core.App, a Allowance, now time.Time) error {
	return app.RunInTransaction(func(txApp core.App) error {
		record, err := usageFind(txApp, a, now)
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

// RecordUsage adds r's usage to a's pool on now's UTC day and reports the
// pool's remaining tokens. A report whose ID the day already counted only
// reports the state. Negative delta components are ignored.
func RecordUsage(app core.App, a Allowance, now time.Time, r UsageReport) (UsageState, error) {
	var state UsageState
	err := app.RunInTransaction(func(txApp core.App) error {
		record, err := usageFindOrNew(txApp, a, now)
		if err != nil {
			return err
		}
		var reportIDs []string
		if raw := record.GetString("report_ids"); raw != "" {
			if err := json.Unmarshal([]byte(raw), &reportIDs); err != nil {
				return fmt.Errorf("failed to read action author usage report IDs: %w", err)
			}
		}
		if slices.Contains(reportIDs, r.ReportID) {
			state = usageStateOf(usageTokens(record), a)
			return nil
		}
		reportIDs = append(reportIDs, r.ReportID)
		if len(reportIDs) > usageReportIDs {
			reportIDs = reportIDs[len(reportIDs)-usageReportIDs:]
		}
		record.Set("report_ids", reportIDs)
		record.Set("input_tokens", record.GetInt("input_tokens")+max(0, r.InputTokens))
		record.Set("output_tokens", record.GetInt("output_tokens")+max(0, r.OutputTokens))
		record.Set("model_calls", record.GetInt("model_calls")+max(0, r.Requests))
		if err := txApp.Save(record); err != nil {
			return fmt.Errorf("failed to save action author usage: %w", err)
		}
		state = usageStateOf(usageTokens(record), a)
		return nil
	})
	if err != nil {
		return UsageState{}, err
	}
	return state, nil
}

// ReadUsage reports a's pool's remaining tokens on now's UTC day.
func ReadUsage(app core.App, a Allowance, now time.Time) (UsageState, error) {
	used := 0
	record, err := usageFind(app, a, now)
	switch {
	case err == nil:
		used = usageTokens(record)
	case !errors.Is(err, sql.ErrNoRows):
		return UsageState{}, fmt.Errorf("failed to find action author usage: %w", err)
	}
	return usageStateOf(used, a), nil
}

// TodayUsage returns the tokens the runner's own and others pools used on
// now's UTC day.
func TodayUsage(app core.App, runnerID string, now time.Time) (own, others int, err error) {
	records, err := app.FindAllRecords(usageCollection, dbx.HashExp{
		"runner": runnerID,
		"day":    now.UTC().Format(usageDayLayout),
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list action author usage: %w", err)
	}
	for _, record := range records {
		switch Pool(record.GetString("pool")) {
		case PoolOwn:
			own += usageTokens(record)
		case PoolOthers:
			others += usageTokens(record)
		}
	}
	return own, others, nil
}

func usageStateOf(used int, a Allowance) UsageState {
	if !a.Offered() {
		none := 0
		return UsageState{Exhausted: true, TokensRemaining: &none}
	}
	if !a.Limited() {
		return UsageState{}
	}
	remaining := max(0, a.Tokens-used)
	return UsageState{Exhausted: used >= a.Tokens, TokensRemaining: &remaining}
}

func usageTokens(record *core.Record) int {
	return record.GetInt("input_tokens") + record.GetInt("output_tokens")
}

func usageFind(app core.App, a Allowance, now time.Time) (*core.Record, error) {
	return app.FindFirstRecordByFilter(
		usageCollection,
		"runner = {:runner} && pool = {:pool} && day = {:day}",
		dbx.Params{
			"runner": a.RunnerID,
			"pool":   string(a.Pool),
			"day":    now.UTC().Format(usageDayLayout),
		},
	)
}

// usageFindOrNew returns the day's usage row, or an unsaved one when the day has no usage yet.
func usageFindOrNew(app core.App, a Allowance, now time.Time) (*core.Record, error) {
	record, err := usageFind(app, a, now)
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
	record.Set("runner", a.RunnerID)
	record.Set("pool", string(a.Pool))
	record.Set("day", now.UTC().Format(usageDayLayout))
	return record, nil
}
