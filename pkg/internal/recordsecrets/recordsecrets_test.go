// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package recordsecrets

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data/"

func TestHandleSecretsEnrichHidesSecretsForNonOwners(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	RegisterHooks(app)

	ownerOrg := findOrganizationByName(t, app, "userA's organization")
	ownerUser, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	otherUser, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)

	coll, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)
	ensureSecretsField(t, app, coll.Name)
	coll, err = app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)
	issuer := createCredentialIssuer(t, app, ownerOrg.Id)

	record := core.NewRecord(coll)
	record.Set("owner", ownerOrg.Id)
	record.Set("name", "secret credential")
	record.Set("credential_issuer", issuer.Id)
	record.Set("secrets", "secret1: value1\n")
	require.NoError(t, app.Save(record))

	t.Run("owner keeps secrets", func(t *testing.T) {
		enriched := enrichCredentialRecord(t, app, record.Id, ownerUser)
		require.Equal(t, "secret1: value1\n", enriched.PublicExport()["secrets"])
	})

	t.Run("anonymous hides secrets", func(t *testing.T) {
		enriched := enrichCredentialRecord(t, app, record.Id, nil)
		require.NotContains(t, enriched.PublicExport(), "secrets")
	})

	t.Run("other organization hides secrets", func(t *testing.T) {
		enriched := enrichCredentialRecord(t, app, record.Id, otherUser)
		require.NotContains(t, enriched.PublicExport(), "secrets")
	})
}

func enrichCredentialRecord(
	t testing.TB,
	app *tests.TestApp,
	recordID string,
	auth *core.Record,
) *core.Record {
	t.Helper()

	record, err := app.FindRecordById("credentials", recordID)
	require.NoError(t, err)

	event := &core.RecordEnrichEvent{
		App: app,
		RequestInfo: &core.RequestInfo{
			Auth: auth,
		},
	}
	event.Record = record

	require.NoError(t, app.OnRecordEnrich().Trigger(event, func(e *core.RecordEnrichEvent) error {
		return e.Next()
	}))

	return record
}

func findOrganizationByName(t testing.TB, app *tests.TestApp, name string) *core.Record {
	t.Helper()

	record, err := app.FindFirstRecordByFilter(
		"organizations",
		"name={:name}",
		dbx.Params{"name": name},
	)
	require.NoError(t, err)

	return record
}

func ensureSecretsField(t testing.TB, app *tests.TestApp, collectionName string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(collectionName)
	require.NoError(t, err)
	if collection.Fields.GetByName("secrets") != nil {
		return
	}

	collection.Fields.Add(&core.TextField{Name: "secrets"})
	require.NoError(t, app.Save(collection))
}

func createCredentialIssuer(t testing.TB, app *tests.TestApp, ownerID string) *core.Record {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", ownerID)
	record.Set("name", "secret credential issuer")
	record.Set("url", "https://issuer.example.com")
	require.NoError(t, app.Save(record))

	return record
}
