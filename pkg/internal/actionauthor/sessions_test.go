// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestRecordSessionRunAcceptsOnlyOneRun(t *testing.T) {
	app, err := tests.NewTestApp("../../../test_pb_data")
	require.NoError(t, err)
	defer app.Cleanup()

	_, err = SessionRun(app, "tenant", "ActionAuthorSession-1")
	require.ErrorIs(t, err, ErrSessionNotRecorded)

	require.NoError(t, RecordSessionRun(app, "tenant", "ActionAuthorSession-1", "run-1"))
	// A retried start records the same run again.
	require.NoError(t, RecordSessionRun(app, "tenant", "ActionAuthorSession-1", "run-1"))
	runID, err := SessionRun(app, "tenant", "ActionAuthorSession-1")
	require.NoError(t, err)
	require.Equal(t, "run-1", runID)

	err = RecordSessionRun(app, "tenant", "ActionAuthorSession-1", "run-2")
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
	require.NoError(t, RecordSessionRun(app, "other", "ActionAuthorSession-1", "run-2"))
	runID, err = SessionRun(app, "other", "ActionAuthorSession-1")
	require.NoError(t, err)
	require.Equal(t, "run-2", runID)
}
