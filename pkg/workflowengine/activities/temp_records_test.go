// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

// tempRecordSeeders create one temporary record per supported collection and
// return it with its canonified identifier.
var tempRecordSeeders = map[string]func(t *testing.T, app *tests.TestApp) (*core.Record, string){
	"credentials": func(t *testing.T, app *tests.TestApp) (*core.Record, string) {
		issuer := createTestIssuer(t, app, testOrgAID, "Temp Issuer", "https://temp-issuer.example")
		credential := createTestCredential(t, app, testOrgAID, issuer.Id, "tempcred")
		return credential, testOrgANamespace + "/" + issuer.GetString("canonified_name") + "/" +
			credential.GetString("canonified_name")
	},
	"use_cases_verifications": func(t *testing.T, app *tests.TestApp) (*core.Record, string) {
		verifierColl, err := app.FindCollectionByNameOrId("verifiers")
		require.NoError(t, err)
		verifier := core.NewRecord(verifierColl)
		verifier.Set("owner", testOrgAID)
		verifier.Set("name", "Temp Verifier")
		verifier.Set("url", "https://verifier.example")
		verifier.Set("standard_and_version", "testsuite/draft-01")
		verifier.Set("format", []string{"SD-JWT"})
		verifier.Set("signing_algorithms", []string{"ES256"})
		verifier.Set("cryptographic_binding_methods", []string{"jwk"})
		verifier.Set("description", "temporary verifier")
		require.NoError(t, app.Save(verifier))

		coll, err := app.FindCollectionByNameOrId("use_cases_verifications")
		require.NoError(t, err)
		useCase := core.NewRecord(coll)
		useCase.Set("owner", testOrgAID)
		useCase.Set("verifier", verifier.Id)
		useCase.Set("name", "tempusecase")
		useCase.Set("yaml", "code")
		require.NoError(t, app.Save(useCase))
		return useCase, testOrgANamespace + "/" + verifier.GetString("canonified_name") + "/" +
			useCase.GetString("canonified_name")
	},
	"wallet_versions": func(t *testing.T, app *tests.TestApp) (*core.Record, string) {
		walletColl, err := app.FindCollectionByNameOrId("wallets")
		require.NoError(t, err)
		wallet := core.NewRecord(walletColl)
		wallet.Set("owner", testOrgAID)
		wallet.Set("name", "wallet-temp-delete")
		require.NoError(t, app.Save(wallet))

		versionColl, err := app.FindCollectionByNameOrId("wallet_versions")
		require.NoError(t, err)
		version := core.NewRecord(versionColl)
		version.Set("owner", testOrgAID)
		version.Set("wallet", wallet.Id)
		version.Set("tag", "abc123")
		installer, err := filesystem.NewFileFromBytes([]byte("dummy apk content"), "app.apk")
		require.NoError(t, err)
		version.Set("android_installer", []*filesystem.File{installer})
		require.NoError(t, app.Save(version))
		return version, testOrgANamespace + "/" + wallet.GetString("canonified_name") + "/" +
			version.GetString("canonified_tag")
	},
}

func TestDeleteTempRecordActivity(t *testing.T) {
	cases := []struct {
		name        string
		recordID    func(record *core.Record) string
		ownerID     string
		identifier  func(identifier string) string
		wantDeleted bool
		wantErr     string
	}{
		{
			name:        "deletes the matching record",
			ownerID:     testOrgAID,
			wantDeleted: true,
		},
		{
			name:     "missing record is an idempotent success",
			recordID: func(*core.Record) string { return "missingrecord12" },
			ownerID:  testOrgAID,
		},
		{
			name:    "owner mismatch",
			ownerID: "other-owner",
			wantErr: errorcodes.RecordNotAccessible,
		},
		{
			name:       "identifier of another record",
			ownerID:    testOrgAID,
			identifier: func(string) string { return testOrgANamespace + "/missing/record" },
			wantErr:    errorcodes.RecordNotAccessible,
		},
	}

	for collection, seed := range tempRecordSeeders {
		for _, tc := range cases {
			t.Run(collection+"/"+tc.name, func(t *testing.T) {
				app := newCredimiTestApp(t)
				record, identifier := seed(t, app)
				recordID := record.Id
				if tc.recordID != nil {
					recordID = tc.recordID(record)
				}
				if tc.identifier != nil {
					identifier = tc.identifier(identifier)
				}

				result, err := executeActivity(
					t,
					NewDeleteTempRecordActivity(app),
					DeleteTempRecordInput{
						Collection:         collection,
						RecordID:           recordID,
						ExpectedOwnerID:    tc.ownerID,
						ExpectedIdentifier: identifier,
					},
				)
				_, findErr := app.FindRecordById(collection, record.Id)
				if tc.wantErr != "" {
					requireActivityError(t, err, tc.wantErr, true)
					require.NoError(t, findErr)
					return
				}
				require.NoError(t, err)
				require.Equal(t, map[string]any{"deleted": tc.wantDeleted}, result.Output)
				if tc.wantDeleted {
					require.Error(t, findErr)
				} else {
					require.NoError(t, findErr)
				}
			})
		}
	}
}

func TestDeleteTempRecordActivityRejectsUnsupportedCollection(t *testing.T) {
	app := newCredimiTestApp(t)
	_, err := executeActivity(t, NewDeleteTempRecordActivity(app), DeleteTempRecordInput{
		Collection:         "pipelines",
		RecordID:           "record",
		ExpectedOwnerID:    testOrgAID,
		ExpectedIdentifier: "identifier",
	})
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
}
