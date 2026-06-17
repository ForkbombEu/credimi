// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pbutils

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data"

func TestGetUserOrganizationID(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
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
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	_, err = GetUserOrganizationID(app, "missing-user")
	require.Error(t, err)
}

func TestGetUserOrganizationCanonifiedName(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
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

func TestGetOrganizationCanonifiedName(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	org, err := app.FindFirstRecordByFilter(
		"organizations",
		"name={:name}",
		dbx.Params{"name": "userA's organization"},
	)
	require.NoError(t, err)

	name, err := GetOrganizationCanonifiedName(app, org.Id)
	require.NoError(t, err)
	require.Equal(t, org.GetString("canonified_name"), name)
}
