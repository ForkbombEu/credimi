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

	_, err = SessionRun(app, "tenant", "ActionAuthorSession-1")
	require.ErrorIs(t, err, ErrSessionNotRecorded)

	require.NoError(
		t,
		RecordSessionRun(app, "tenant", "ActionAuthorSession-1", "run-1", SessionDetails{}),
	)
	// A retried start records the same run again.
	require.NoError(
		t,
		RecordSessionRun(app, "tenant", "ActionAuthorSession-1", "run-1", SessionDetails{}),
	)
	runID, err := SessionRun(app, "tenant", "ActionAuthorSession-1")
	require.NoError(t, err)
	require.Equal(t, "run-1", runID)

	err = RecordSessionRun(app, "tenant", "ActionAuthorSession-1", "run-2", SessionDetails{})
	require.ErrorIs(t, err, ErrForbidden)
	require.ErrorContains(
		t,
		err,
		"action author session ActionAuthorSession-1 already ran as another run",
	)
	runID, err = SessionRun(app, "tenant", "ActionAuthorSession-1")
	require.NoError(t, err)
	require.Equal(t, "run-1", runID)

	// The same workflow ID in another namespace is another session.
	require.NoError(
		t,
		RecordSessionRun(app, "other", "ActionAuthorSession-1", "run-2", SessionDetails{}),
	)
	runID, err = SessionRun(app, "other", "ActionAuthorSession-1")
	require.NoError(t, err)
	require.Equal(t, "run-2", runID)
}

func TestRecordSessionRunKeepsTheFirstDetails(t *testing.T) {
	app, err := tests.NewTestApp("../../../test_pb_data")
	require.NoError(t, err)
	defer app.Cleanup()

	first := SessionDetails{WalletVersionIdentifier: "org/wallet/v1", DeviceID: "runner/device"}
	require.NoError(t, RecordSessionRun(app, "tenant", "ActionAuthorSession-1", "run-1", first))
	require.NoError(
		t,
		RecordSessionRun(app, "tenant", "ActionAuthorSession-1", "run-1", SessionDetails{}),
	)

	sessions, err := RecentWalletSessions(app, "tenant", "org/wallet", 10)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	require.Equal(t, "ActionAuthorSession-1", sessions[0].WorkflowID)
	require.Equal(t, "run-1", sessions[0].RunID)
	require.Equal(t, first.WalletVersionIdentifier, sessions[0].WalletVersionIdentifier)
	require.Equal(t, first.DeviceID, sessions[0].DeviceID)
}

func TestRecentWalletSessionsListsTheWalletNewestFirst(t *testing.T) {
	app, err := tests.NewTestApp("../../../test_pb_data")
	require.NoError(t, err)
	defer app.Cleanup()

	rows := []struct {
		namespace, workflowID, wallet string
	}{
		{"tenant", "ActionAuthorSession-store", "org/wallet"},
		{"tenant", "ActionAuthorSession-version", "org/wallet/v1"},
		{"tenant", "ActionAuthorSession-other-wallet", "org/wallet2/v1"},
		{"other", "ActionAuthorSession-other-namespace", "org/wallet/v1"},
		{"tenant", "ActionAuthorSession-latest", "org/wallet/v2"},
	}
	base := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for i, row := range rows {
		require.NoError(t, RecordSessionRun(
			app,
			row.namespace,
			row.workflowID,
			"run",
			SessionDetails{WalletVersionIdentifier: row.wallet, DeviceID: "device"},
		))
		// Records created in one test share a millisecond; set distinct times.
		_, err := app.DB().Update(
			sessionsCollection,
			dbx.Params{
				"created": base.Add(time.Duration(i) * time.Minute).
					Format("2006-01-02 15:04:05.000Z"),
			},
			dbx.HashExp{"namespace": row.namespace, "workflow_id": row.workflowID},
		).
			Execute()
		require.NoError(t, err)
	}

	sessions, err := RecentWalletSessions(app, "tenant", "org/wallet", 10)
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
	require.Equal(t, base.Add(4*time.Minute), sessions[0].Created.UTC())

	limited, err := RecentWalletSessions(app, "tenant", "org/wallet", 1)
	require.NoError(t, err)
	require.Len(t, limited, 1)
	require.Equal(t, "ActionAuthorSession-latest", limited[0].WorkflowID)
}

func TestIsSessionID(t *testing.T) {
	require.True(t, IsSessionID("ActionAuthorSession-0f8fad5b-d9cb-469f-a165-70867728950e"))
	for _, id := range []string{
		"",
		"ActionAuthorSession-",
		"ActionAuthorSession-0F8FAD5B-D9CB-469F-A165-70867728950E",
		"ActionAuthorSession-0f8fad5b-d9cb-469f-a165-70867728950e-author",
		"x-ActionAuthorSession-0f8fad5b-d9cb-469f-a165-70867728950e",
		"ActionAuthorSession-ticket",
	} {
		require.False(t, IsSessionID(id), id)
	}
}
