// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package logo

import (
	"fmt"
	"io"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data/"

func setupTestApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir) // qui la metti che punta a test_pb_data con il path relativo rispetto al tuo test file
	require.NoError(t, err)
	LogoHooks(app)

	return app
}

func getOrgIDfromName(collectionNameOrID string, name string) (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	filter := fmt.Sprintf(`name="%s"`, name)

	record, err := app.FindFirstRecordByFilter(collectionNameOrID, filter)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func TestLogoHooks_Valid(t *testing.T) {

	app := setupTestApp(t)
	defer app.Cleanup()
	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	record := core.NewRecord(coll)
	own, _ := getOrgIDfromName("organizations", "userA's organization")
	record.Set("owner", own)
	record.Set("logo", "")

	record.Set("logo_url", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/25/de/dc/25dedc8b-0e66-70cd-1530-082a007ef642/AppIcon-0-0-1x_U007ephone-0-1-85-220.png/100x100bb.jpg")

	require.Empty(t, record.GetString("logo"), "The logo field should have been set by LogoHooks after saving")

	err = app.Save(record)
	require.NoError(t, err)

	logoField := record.GetString("logo")
	require.NotEmpty(t, logoField, "The logo field shouldn't have been set by LogoHooks after saving")

	logokey := record.BaseFilesPath() + "/" + record.GetString("logo")

	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}
	defer fsys.Close()

	r, err := fsys.GetFile(logokey)
	if err != nil {
		t.Fatalf("Failed to get file: %v", err)
	}
	defer r.Close()

	downloadedData, err := io.ReadAll(r)
	require.NoError(t, err)

	require.NotEmpty(t, downloadedData, "The downloaded logo file should not be empty")

	t.Logf("Test complete: LogoHooks worked correctly")
}

func TestLogoHooks_UpdateAddLogoURL(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	own, _ := getOrgIDfromName("organizations", "userA's organization")
	record.Set("owner", own)
	record.Set("logo", "")
	record.Set("logo_url", "")

	err = app.Save(record)
	require.NoError(t, err)

	require.Empty(t, record.GetString("logo"), "Logo field should be empty initially")

	record.Set("logo_url", "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/25/de/dc/25dedc8b-0e66-70cd-1530-082a007ef642/AppIcon-0-0-1x_U007ephone-0-1-85-220.png/100x100bb.jpg")

	err = app.Save(record)
	require.NoError(t, err)

	logoField := record.GetString("logo")
	require.NotEmpty(t, logoField, "Logo field should have been set by LogoHooks after update")

	logokey := record.BaseFilesPath() + "/" + logoField
	fsys, err := app.NewFilesystem()
	require.NoError(t, err)
	defer fsys.Close()

	r, err := fsys.GetFile(logokey)
	require.NoError(t, err)
	defer r.Close()

	downloadedData, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NotEmpty(t, downloadedData, "Downloaded logo file should not be empty")

	t.Logf("Test complete: LogoHooks worked correctly on record update")
}

func TestLogoHooks_InvalidURL(t *testing.T) {
	app := setupTestApp(t)
	defer app.Cleanup()

	coll, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	own, _ := getOrgIDfromName("organizations", "userA's organization")
	record.Set("owner", own)
	record.Set("logo", "")
	record.Set("logo_url", "https://invalid-domain-that-does-not-exist-12345.com/logo.png")

	err = app.Save(record)
	require.NoError(t, err, "Save should succeed even if logo download fails")

	logoField := record.GetString("logo")

	require.Empty(t, logoField, "Logo field should remain empty when URL is invalid")

	t.Logf("Test complete: LogoHooks handled invalid URL gracefully")
}
