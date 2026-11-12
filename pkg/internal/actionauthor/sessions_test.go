// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestRecordSessionRunAcceptsOnlyOneRun(t *testing.T) {
	app, err := tests.NewTestApp("../../../test_pb_data")
	require.NoError(t, err)
	defer app.Cleanup()

	_, err = FindSession(app, "ActionAuthorSession-1")
	require.ErrorIs(t, err, ErrSessionNotRecorded)

	details := SessionDetails{Organization: "tenant"}
	require.NoError(
		t,
		RecordSessionRun(app, "runner-testrunner00001", "ActionAuthorSession-1", "run-1", details),
	)
	// A retried start records the same run again.
	require.NoError(
		t,
		RecordSessionRun(app, "runner-testrunner00001", "ActionAuthorSession-1", "run-1", details),
	)
	recorded, err := FindSession(app, "ActionAuthorSession-1")
	require.NoError(t, err)
	require.Equal(t, "run-1", recorded.RunID)
	require.Equal(t, "runner-testrunner00001", recorded.Namespace)
	require.Equal(t, "tenant", recorded.Organization)

	err = RecordSessionRun(app, "runner-testrunner00001", "ActionAuthorSession-1", "run-2", details)
	require.ErrorIs(t, err, ErrForbidden)
	require.ErrorContains(
		t,
		err,
		"action author session ActionAuthorSession-1 already ran as another run",
	)
	// Session workflow IDs are unique across namespaces.
	err = RecordSessionRun(app, "runner-other", "ActionAuthorSession-1", "run-2", details)
	require.ErrorIs(t, err, ErrForbidden)
	recorded, err = FindSession(app, "ActionAuthorSession-1")
	require.NoError(t, err)
	require.Equal(t, "run-1", recorded.RunID)
	require.Equal(t, "runner-testrunner00001", recorded.Namespace)
}

func TestRecordSessionRunKeepsTheFirstDetails(t *testing.T) {
	app, err := tests.NewTestApp("../../../test_pb_data")
	require.NoError(t, err)
	defer app.Cleanup()

	first := SessionDetails{
		WalletVersionIdentifier: "org/wallet/v1",
		DeviceID:                "runner/device",
		ActorID:                 "user-1",
		Organization:            "tenant",
	}
	require.NoError(
		t,
		RecordSessionRun(app, "runner-testrunner00001", "ActionAuthorSession-1", "run-1", first),
	)
	require.NoError(
		t,
		RecordSessionRun(
			app,
			"runner-testrunner00001",
			"ActionAuthorSession-1",
			"run-1",
			SessionDetails{},
		),
	)

	sessions, err := RecentWalletSessions(app, "tenant", "org/wallet", first.ActorID, 10)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.Equal(t, "ActionAuthorSession-1", sessions[0].WorkflowID)
	require.Equal(t, "run-1", sessions[0].RunID)
	require.Equal(t, first.WalletVersionIdentifier, sessions[0].WalletVersionIdentifier)
	require.Equal(t, first.DeviceID, sessions[0].DeviceID)
	require.Equal(t, first.ActorID, sessions[0].ActorID)
	require.Equal(t, "runner-testrunner00001", sessions[0].Namespace)
	require.Equal(t, "tenant", sessions[0].Organization)

	found, err := FindSession(app, "ActionAuthorSession-1")
	require.NoError(t, err)
	require.Equal(t, sessions[0], found)
}

func TestRecentWalletSessionsListsTheWalletNewestFirst(t *testing.T) {
	app, err := tests.NewTestApp("../../../test_pb_data")
	require.NoError(t, err)
	defer app.Cleanup()

	// Sessions of one organization run in its runners' namespaces.
	rows := []struct {
		namespace, organization, workflowID, wallet, actor string
	}{
		{"runner-a", "tenant", "ActionAuthorSession-store", "org/wallet", "user-1"},
		{"runner-b", "tenant", "ActionAuthorSession-version", "org/wallet/v1", "user-1"},
		{"runner-a", "tenant", "ActionAuthorSession-other-wallet", "org/wallet2/v1", "user-1"},
		{"runner-a", "other", "ActionAuthorSession-other-org", "org/wallet/v1", "user-1"},
		{"runner-a", "tenant", "ActionAuthorSession-other-actor", "org/wallet/v1", "user-2"},
		{"runner-a", "tenant", "ActionAuthorSession-latest", "org/wallet/v2", "user-1"},
	}
	base := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for i, row := range rows {
		require.NoError(t, RecordSessionRun(
			app,
			row.namespace,
			row.workflowID,
			"run",
			SessionDetails{
				WalletVersionIdentifier: row.wallet,
				DeviceID:                "device",
				ActorID:                 row.actor,
				Organization:            row.organization,
			},
		))
		// Records created in one test share a millisecond; set distinct times.
		_, err := app.DB().Update(
			sessionsCollection,
			dbx.Params{
				"created": base.Add(time.Duration(i) * time.Minute).
					Format("2006-01-02 15:04:05.000Z"),
			},
			dbx.HashExp{"workflow_id": row.workflowID},
		).
			Execute()
		require.NoError(t, err)
	}

	sessions, err := RecentWalletSessions(app, "tenant", "org/wallet", "user-1", 10)
	require.NoError(t, err)
	ids := make([]string, 0, len(sessions))
	for _, session := range sessions {
		ids = append(ids, session.WorkflowID)
	}
	require.Equal(t, []string{
		"ActionAuthorSession-latest",
		"ActionAuthorSession-version",
		"ActionAuthorSession-store",
	}, ids)
	require.Equal(t, base.Add(5*time.Minute), sessions[0].Created.UTC())

	limited, err := RecentWalletSessions(app, "tenant", "org/wallet", "user-1", 1)
	require.NoError(t, err)
	require.Len(t, limited, 1)
	require.Equal(t, "ActionAuthorSession-latest", limited[0].WorkflowID)
}

func TestIsSessionID(t *testing.T) {
	require.True(
		t,
		IsSessionID("ActionAuthorSession-testrunner00001-0f8fad5b-d9cb-469f-a165-70867728950e"),
	)
	for _, id := range []string{
		"",
		"ActionAuthorSession-",
		"ActionAuthorSession-0F8FAD5B-D9CB-469F-A165-70867728950E",
		"ActionAuthorSession-testrunner00001-0f8fad5b-d9cb-469f-a165-70867728950e-author",
		"x-ActionAuthorSession-testrunner00001-0f8fad5b-d9cb-469f-a165-70867728950e",
		"ActionAuthorSession-ticket",
		"ActionAuthorSession-0f8fad5b-d9cb-469f-a165-70867728950e",
		"ActionAuthorSession-TestRunner00001-0f8fad5b-d9cb-469f-a165-70867728950e",
		"ActionAuthorSession-testrunner0001-0f8fad5b-d9cb-469f-a165-70867728950e",
	} {
		require.False(t, IsSessionID(id), id)
	}
}

func TestSessionWorkflowIDRoundTrip(t *testing.T) {
	id := SessionWorkflowID("q432kn0bisw9pfm", "2fe33961-f99b-422f-90f1-5353e201ae7a")
	require.Equal(t, "ActionAuthorSession-q432kn0bisw9pfm-2fe33961-f99b-422f-90f1-5353e201ae7a", id)
	require.True(t, IsSessionID(id))
	runnerID, ok := SessionRunnerID(id)
	require.True(t, ok)
	require.Equal(t, "q432kn0bisw9pfm", runnerID)
	namespace, ok := SessionNamespace(id)
	require.True(t, ok)
	require.Equal(t, "runner-q432kn0bisw9pfm", namespace)

	for _, malformed := range []string{"", "ActionAuthorSession-ticket", id + "-author"} {
		_, ok = SessionRunnerID(malformed)
		require.False(t, ok, malformed)
		_, ok = SessionNamespace(malformed)
		require.False(t, ok, malformed)
	}
}
