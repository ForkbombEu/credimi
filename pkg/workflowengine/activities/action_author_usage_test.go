// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

// usageSessionLookup seeds a runner of the session's organization with one
// device and returns lookups of a session on that device, and the runner.
func usageSessionLookup(
	t *testing.T,
	app core.App,
	session actionauthor.BuildRequest,
	ownTokens int,
) (func() authorSessionLookup, *core.Record) {
	t.Helper()
	selection, err := actionauthor.Select(app, "", session)
	require.NoError(t, err)
	runners, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(runners)
	runner.Set("owner", selection.Organization.Id)
	runner.Set("name", "usage-runner")
	runner.Set("ip", "https://usage-runner.example.test")
	runner.Set("agent_own_daily_tokens", ownTokens)
	require.NoError(t, app.Save(runner))
	devices, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(devices)
	device.Set("owner", selection.Organization.Id)
	device.Set("runner", runner.Id)
	device.Set("name", "usage-device")
	device.Set("type", "android_emulator")
	device.Set("serial", "emulator-5554")
	require.NoError(t, app.Save(device))
	deviceID := selection.Organization.GetString("canonified_name") + "/" +
		runner.GetString("canonified_name") + "/" + device.GetString("canonified_name")
	// Each lookup reads the session's history afresh.
	return func() authorSessionLookup {
		lookup := unsignedSessionLookup(app)
		lookup.history = signedSessionHistory(t, app, map[string]any{
			"device_id":                 deviceID,
			"platform":                  session.Platform,
			"wallet_version_identifier": session.WalletVersionIdentifier,
			"actor_id":                  session.ActorID,
		})
		return lookup.lookup()
	}, runner
}

func remainingTokens(n int) *int { return &n }

func TestRecordAndReadActionAuthorUsage(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, _ := authorSaveFixture(t)
	caller := authorTestCaller()
	lookup, runner := usageSessionLookup(t, app, session, 1000)
	record := func() *RecordActionAuthorUsageActivity {
		activity := NewRecordActionAuthorUsageActivity(app)
		activity.authorSessionLookup = lookup()
		return activity
	}
	read := func() *ReadActionAuthorUsageActivity {
		activity := NewReadActionAuthorUsageActivity(app)
		activity.authorSessionLookup = lookup()
		return activity
	}
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	report := func(id string, d actionauthor.UsageDelta) actionauthor.UsageReport {
		return actionauthor.UsageReport{ReportID: id, UsageDelta: d}
	}

	state, err := read().read(t.Context(), caller, now)
	require.NoError(t, err)
	require.Equal(t, actionauthor.UsageState{TokensRemaining: remainingTokens(1000)}, state)

	state, err = record().record(t.Context(), caller, report("wf/1", actionauthor.UsageDelta{
		Requests: 2, InputTokens: 300, OutputTokens: 100,
	}), now)
	require.NoError(t, err)
	require.Equal(t, actionauthor.UsageState{TokensRemaining: remainingTokens(600)}, state)

	// A retried report answers the same state and adds nothing.
	state, err = record().record(t.Context(), caller, report("wf/1", actionauthor.UsageDelta{
		Requests: 2, InputTokens: 300, OutputTokens: 100,
	}), now)
	require.NoError(t, err)
	require.Equal(t, actionauthor.UsageState{TokensRemaining: remainingTokens(600)}, state)

	state, err = record().record(t.Context(), caller, report("wf/2", actionauthor.UsageDelta{
		Requests: 1, InputTokens: 500, OutputTokens: 200,
	}), now)
	require.NoError(t, err)
	exhausted := actionauthor.UsageState{Exhausted: true, TokensRemaining: remainingTokens(0)}
	require.Equal(t, exhausted, state)
	state, err = read().read(t.Context(), caller, now)
	require.NoError(t, err)
	require.Equal(t, exhausted, state)

	// The usage counts in the runner's own pool.
	own, others, err := actionauthor.TodayUsage(app, runner.Id, now)
	require.NoError(t, err)
	require.Equal(t, 1100, own)
	require.Zero(t, others)

	_, err = record().record(t.Context(), caller, report("", actionauthor.UsageDelta{InputTokens: 1}), now)
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "report_id is required")

	// Callers outside a running session are refused before any usage counts.
	_, err = executeActivity(t, record(), report("wf/3", actionauthor.UsageDelta{InputTokens: 1}))
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
	require.ErrorContains(t, err, "caller is not an action author session agent")
	_, err = executeActivity(t, read(), map[string]any{})
	requireActivityError(t, err, errorcodes.RecordNotAccessible, true)
}

func TestReadActionAuthorUsageWithoutLimit(t *testing.T) {
	setFCAFTestEnv(t)
	app, session, _ := authorSaveFixture(t)
	caller := authorTestCaller()
	lookup, _ := usageSessionLookup(t, app, session, 0)
	a := NewReadActionAuthorUsageActivity(app)
	a.authorSessionLookup = lookup()

	state, err := a.read(t.Context(), caller, time.Now())
	require.NoError(t, err)
	require.Equal(t, actionauthor.UsageState{}, state, "no limit: tokens_remaining null")
}
