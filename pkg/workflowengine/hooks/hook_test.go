// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package hooks

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data"

func TestFetchNamespacesIncludesDefaultAndOrganizations(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	namespaces, err := FetchNamespaces(app)
	require.NoError(t, err)

	collection, err := app.FindCollectionByNameOrId("organizations")
	require.NoError(t, err)

	records, err := app.FindRecordsByFilter(collection, "", "-created", 0, 0)
	require.NoError(t, err)

	require.Equal(t, "default", namespaces[0])
	require.Len(t, namespaces, len(records)+1)

	for i, record := range records {
		require.Equal(t, record.GetString("canonified_name"), namespaces[i+1])
	}
}

func TestFetchNamespacesHandlesEmptyOrganizations(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	_, err = app.DB().NewQuery("PRAGMA foreign_keys = OFF").Execute()
	require.NoError(t, err)
	_, err = app.DB().NewQuery("DELETE FROM organizations").Execute()
	require.NoError(t, err)
	_, err = app.DB().NewQuery("PRAGMA foreign_keys = ON").Execute()
	require.NoError(t, err)

	namespaces, err := FetchNamespaces(app)
	require.NoError(t, err)
	require.Equal(t, []string{"default"}, namespaces)
}
