// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

func TestSessionWallet(t *testing.T) {
	require.Equal(t, "org/wallet", SessionWallet("org/wallet/v1"))
	require.Equal(t, "org/wallet", SessionWallet("org/wallet"))
}
