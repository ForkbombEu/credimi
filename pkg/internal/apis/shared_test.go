// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestGenerateToken(t *testing.T) {
	token, err := generateToken("users", "userA@example.org")
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestIsSuperUser(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	require.False(t, isSuperUser(app, user))

	superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
	require.NoError(t, err)

	require.True(t, isSuperUser(app, superuser))
}
