// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func setupUtilityTestApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	return app
}

func TestGetUserOrganizationID(t *testing.T) {
	app := setupUtilityTestApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	orgAuth, err := app.FindFirstRecordByFilter(
		"orgAuthorizations",
		"user={:user}",
		dbx.Params{"user": user.Id},
	)
	require.NoError(t, err)

	orgID, err := GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	require.Equal(t, orgAuth.GetString("organization"), orgID)
}

func TestGetUserOrganizationIDMissingUser(t *testing.T) {
	app := setupUtilityTestApp(t)
	defer app.Cleanup()

	_, err := GetUserOrganizationID(app, "missing-user")
	require.Error(t, err)
}

func TestGetUserOrganizationCanonifiedName(t *testing.T) {
	app := setupUtilityTestApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	orgAuth, err := app.FindFirstRecordByFilter(
		"orgAuthorizations",
		"user={:user}",
		dbx.Params{"user": user.Id},
	)
	require.NoError(t, err)

	org, err := app.FindRecordById("organizations", orgAuth.GetString("organization"))
	require.NoError(t, err)

	name, err := GetUserOrganizationCanonifiedName(app, user.Id)
	require.NoError(t, err)
	require.Equal(t, org.GetString("canonified_name"), name)
}

func TestGetUserOrganizationCanonifiedNameMissingUser(t *testing.T) {
	app := setupUtilityTestApp(t)
	defer app.Cleanup()

	_, err := GetUserOrganizationCanonifiedName(app, "missing-user")
	require.Error(t, err)
}
