// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/stretchr/testify/require"
)

func TestRecordActionAuthorUsage(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	caller := authorTestCaller()
	caller.Namespace = namespace
	selection, err := actionauthor.Select(app, namespace, session)
	require.NoError(t, err)
	selection.Organization.Set("action_author_daily_tokens", 1000)
	require.NoError(t, app.Save(selection.Organization))
	a := func() *RecordActionAuthorUsageActivity {
		activity := NewRecordActionAuthorUsageActivity(app)
		activity.authorSessionLookup = specSessionLookup(
			t, session, actionauthor.Spec{Slot: "onboarding"},
		).lookup()
		return activity
	}
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	report := func(id string, d actionauthor.UsageDelta) actionauthor.UsageReport {
		return actionauthor.UsageReport{ReportID: id, UsageDelta: d}
	}

	state, err := a().record(t.Context(), caller, report("wf/1", actionauthor.UsageDelta{
		Requests: 2, InputTokens: 300, OutputTokens: 100,
	}), now)
	require.NoError(t, err)
	require.Equal(t, actionauthor.UsageState{TokensRemaining: 600}, state)

	// A retried report answers the same state and adds nothing.
	state, err = a().record(t.Context(), caller, report("wf/1", actionauthor.UsageDelta{
		Requests: 2, InputTokens: 300, OutputTokens: 100,
	}), now)
	require.NoError(t, err)
	require.Equal(t, actionauthor.UsageState{TokensRemaining: 600}, state)

	state, err = a().record(t.Context(), caller, report("wf/2", actionauthor.UsageDelta{
		Requests: 1, InputTokens: 500, OutputTokens: 200,
	}), now)
	require.NoError(t, err)
	require.Equal(t, actionauthor.UsageState{Exhausted: true, TokensRemaining: 0}, state)

	require.ErrorIs(
		t,
		actionauthor.ReserveSession(
			app,
			selection.Organization.Id,
			now,
			actionauthor.QuotaForOrganization(selection.Organization),
		),
		actionauthor.ErrQuotaExceeded,
	)

	_, err = a().record(t.Context(), caller, report("", actionauthor.UsageDelta{InputTokens: 1}), now)
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "report_id is required")

	// Callers outside a running session are refused before any usage counts.
	_, err = executeActivity(t, a(), report("wf/3", actionauthor.UsageDelta{InputTokens: 1}))
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "caller is not an action author session agent")
}

func TestRecordActionAuthorUsageWithoutAllowance(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	caller := authorTestCaller()
	caller.Namespace = namespace
	a := NewRecordActionAuthorUsageActivity(app)
	a.authorSessionLookup = specSessionLookup(
		t, session, actionauthor.Spec{Slot: "onboarding"},
	).lookup()

	state, err := a.record(t.Context(), caller, actionauthor.UsageReport{
		ReportID:   "wf/1",
		UsageDelta: actionauthor.UsageDelta{InputTokens: 1},
	}, time.Now())
	require.NoError(t, err)
	require.Equal(t, actionauthor.UsageState{Exhausted: true}, state)
}
