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

// ErrQuotaExceeded reports that an organization used up its daily action author quota.
var ErrQuotaExceeded = errors.New("daily action author quota exceeded")

// Quota bounds an organization's daily action author usage. A limit <= 0 is
// no allowance at all: there is no unlimited value.
type Quota struct {
	Sessions int
	Tokens   int
}

// HasAllowance reports whether the organization may use the action author at all.
func (q Quota) HasAllowance() bool {
	return q.Sessions > 0 && q.Tokens > 0
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

// UsageState summarizes an organization's remaining daily token budget.
type UsageState struct {
	Exhausted       bool `json:"exhausted"`
	TokensRemaining int  `json:"tokens_remaining"`
}

// QuotaForOrganization returns the daily quota a superuser set on the
// organization record; new organizations have none.
func QuotaForOrganization(org *core.Record) Quota {
	return Quota{
		Sessions: org.GetInt("action_author_daily_sessions"),
		Tokens:   org.GetInt("action_author_daily_tokens"),
	}
}

// ReserveSession counts a new authoring session for orgID on now's UTC day, failing with
// ErrQuotaExceeded when the session or token quota is already used up.
func ReserveSession(app core.App, orgID string, now time.Time, q Quota) error {
	return app.RunInTransaction(func(txApp core.App) error {
		record, err := usageFindOrNew(txApp, orgID, now)
		if err != nil {
			return err
		}
		if q.Sessions <= 0 || record.GetInt("sessions") >= q.Sessions {
			return ErrQuotaExceeded
		}
		if q.Tokens <= 0 || usageTokens(record) >= q.Tokens {
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

// RecordUsage adds r's usage to orgID's usage on now's UTC day and reports the remaining
// token budget. A report whose ID the day already counted only reports the state.
// Negative delta components are ignored.
func RecordUsage(
	app core.App,
	orgID string,
	now time.Time,
	r UsageReport,
	q Quota,
) (UsageState, error) {
	var state UsageState
	err := app.RunInTransaction(func(txApp core.App) error {
		record, err := usageFindOrNew(txApp, orgID, now)
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
			state = usageStateOf(usageTokens(record), q)
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
		state = usageStateOf(usageTokens(record), q)
		return nil
	})
	if err != nil {
		return UsageState{}, err
	}
	return state, nil
}

func usageStateOf(used int, q Quota) UsageState {
	if !q.HasAllowance() {
		return UsageState{Exhausted: true}
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
