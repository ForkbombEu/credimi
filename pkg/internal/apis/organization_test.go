// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestCreateNewOrganizationForUser(t *testing.T) {
	t.Run("invalid email returns error", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		defer app.Cleanup()

		usersCollection, err := app.FindCollectionByNameOrId("users")
		require.NoError(t, err)

		user := core.NewRecord(usersCollection)
		user.Set("email", "invalid-email")

		err = createNewOrganizationForUser(app, user)
		require.Error(t, err)
		require.ErrorContains(t, err, "Invalid email format")
	})

	t.Run("creates organization and owner authorization", func(t *testing.T) {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		defer app.Cleanup()

		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)

		orgsBefore, err := app.FindRecordsByFilter("organizations", "", "", 0, 0)
		require.NoError(t, err)
		authBefore, err := app.FindRecordsByFilter(
			"orgAuthorizations",
			"user = {:user}",
			"",
			0,
			0,
			dbx.Params{"user": user.Id},
		)
		require.NoError(t, err)

		err = createNewOrganizationForUser(app, user)
		require.NoError(t, err)

		orgsAfter, err := app.FindRecordsByFilter("organizations", "", "", 0, 0)
		require.NoError(t, err)
		authAfter, err := app.FindRecordsByFilter(
			"orgAuthorizations",
			"user = {:user}",
			"-created",
			0,
			0,
			dbx.Params{"user": user.Id},
		)
		require.NoError(t, err)

		require.Len(t, orgsAfter, len(orgsBefore)+1)
		require.Len(t, authAfter, len(authBefore)+1)

		ownerRole, err := app.FindFirstRecordByFilter("orgRoles", "name='owner'")
		require.NoError(t, err)

		latestAuth := authAfter[0]
		require.Equal(t, ownerRole.Id, latestAuth.GetString("role"))
		require.Equal(t, user.Id, latestAuth.GetString("user"))

		organizationID := latestAuth.GetString("organization")
		require.NotEmpty(t, organizationID)
		newOrg, err := app.FindRecordById("organizations", organizationID)
		require.NoError(t, err)
		require.Contains(t, newOrg.GetString("name"), "'s organization")
		require.NotEmpty(t, newOrg.GetString("canonified_name"))
	})
}
