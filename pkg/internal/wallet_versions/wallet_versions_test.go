// SPDX-FileCopyrightText: 2025 Your Company
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package walletversions

import (
	"net/http"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data/"

func getTestOrgID() (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	filter := `name="userA's organization"`

	record, err := app.FindFirstRecordByFilter("organizations", filter)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func setupWalletVersionsApp(orgID string) func(t testing.TB) *tests.TestApp {
	return func(t testing.TB) *tests.TestApp {
		app, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		WalletVersionHooks(app)

		testFileContent := []byte("fake apk content for testing")
		testFile, err := filesystem.NewFileFromBytes(testFileContent, "test_app.apk")
		require.NoError(t, err)
		walletsColl, _ := app.FindCollectionByNameOrId("wallets")
		walletRecord := core.NewRecord(walletsColl)
		walletRecord.Set("owner", orgID)
		require.NoError(t, app.Save(walletRecord))

		coll, _ := app.FindCollectionByNameOrId("wallet_versions")
		record := core.NewRecord(coll)
		record.Set("id", "record123456789")
		record.Set("owner", orgID)
		record.Set("downloadable", false)
		record.Set("tag", "mytag")
		record.Set("wallet", walletRecord.Id)
		record.Set("android_installer", []*filesystem.File{testFile})
		require.NoError(t, app.Save(record))

		return app
	}
}

func TestWalletVersions(t *testing.T) {
	orgID, err := getTestOrgID()
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:   "test false",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/collections/wallet_versions/records/record123456789"
			}(),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":"record123456789"`,
			},
			NotExpectedContent: []string{
				`"android_installer"`,
			},
			TestAppFactory: setupWalletVersionsApp(orgID),
		},
		{
			Name:   "test true",
			Method: http.MethodGet,
			URL: func() string {
				return "/api/collections/wallet_versions/records/record123456789"
			}(),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":"record123456789"`,
				`"android_installer"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletVersionsApp(orgID)(t)

				record, err := app.FindRecordById("wallet_versions", "record123456789", nil)
				require.NoError(t, err)

				record.Set("downloadable", true)
				require.NoError(t, app.Save(record))

				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
