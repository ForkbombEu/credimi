// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/require"
)

func setupWalletApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	canonify.RegisterCanonifyHooks(app)
	WalletTemporalInternalRoutes.Add(app)
	return app
}

type readSeekNopCloser struct {
	*bytes.Reader
}

func (r *readSeekNopCloser) Close() error { return nil }

type mockFileReader struct {
	data []byte
}

func (m *mockFileReader) Open() (io.ReadSeekCloser, error) {
	return &readSeekNopCloser{bytes.NewReader(m.data)}, nil
}

func NewTestFile(name string, content []byte) *filesystem.File {
	return &filesystem.File{
		Reader:       &mockFileReader{data: content},
		Name:         name,
		OriginalName: name,
		Size:         int64(len(content)),
	}
}

func TestWalletGetAPKMD5(t *testing.T) {
	orgID, err := getOrgIDfromName("organizations", "userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:   "get APK MD5 with valid wallet identifier",
			Method: http.MethodPost,
			URL:    "/api/wallet/get-apk-md5",
			Body: jsonBody(map[string]any{
				"wallet_identifier":         "usera-s-organization/wallet123",
				"wallet_version_identifier": "",
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"apk_name"`,
				`"md5"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletApp(t)

				walletColl, err := app.FindCollectionByNameOrId("wallets")
				require.NoError(t, err)
				walletRecord := core.NewRecord(walletColl)
				walletRecord.Set("name", "wallet123")
				walletRecord.Set("owner", orgID)
				require.NoError(t, app.Save(walletRecord))

				walletVersionColl, err := app.FindCollectionByNameOrId("wallet_versions")
				require.NoError(t, err)
				versionRecord := core.NewRecord(walletVersionColl)
				versionRecord.Set("wallet", walletRecord.Id)
				versionRecord.Set("tag", "1.0.0")
				versionRecord.Set("owner", orgID)
				apkFile := NewTestFile("app.apk", []byte("dummy apk content"))
				versionRecord.Set("android_installer", []*filesystem.File{apkFile})
				require.NoError(t, app.Save(versionRecord))

				return app
			},
		},
		{
			Name:   "get APK MD5 with valid version identifier",
			Method: http.MethodPost,
			URL:    "/api/wallet/get-apk-md5",
			Body: jsonBody(map[string]any{
				"wallet_identifier":         "",
				"wallet_version_identifier": "usera-s-organization/wallet234/2-0-0",
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"apk_name"`,
				`"md5"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletApp(t)

				walletColl, err := app.FindCollectionByNameOrId("wallets")
				require.NoError(t, err)
				walletRecord := core.NewRecord(walletColl)
				walletRecord.Set("name", "wallet234")
				walletRecord.Set("owner", orgID)
				require.NoError(t, app.Save(walletRecord))

				walletVersionColl, err := app.FindCollectionByNameOrId("wallet_versions")
				require.NoError(t, err)
				versionRecord := core.NewRecord(walletVersionColl)
				versionRecord.Set("wallet", walletRecord.Id)
				versionRecord.Set("tag", "2.0.0")
				versionRecord.Set("owner", orgID)
				apkFile := NewTestFile("app.apk", []byte("dummy apk content"))
				versionRecord.Set("android_installer", []*filesystem.File{apkFile})
				require.NoError(t, app.Save(versionRecord))

				return app
			},
		},
		{
			Name:           "get APK MD5 with missing identifiers",
			Method:         http.MethodPost,
			URL:            "/api/wallet/get-apk-md5",
			Body:           jsonBody(map[string]any{}),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"identifier"`,
				`"no identifier provided"`,
			},
			TestAppFactory: setupWalletApp,
		},
		{
			Name:           "get APK MD5 with non-existent wallet",
			Method:         http.MethodPost,
			URL:            "/api/wallet/get-apk-md5",
			Body:           jsonBody(map[string]any{"wallet_identifier": "nonexistent"}),
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"wallet_version"`,
				`"wallet version not found"`,
			},
			TestAppFactory: setupWalletApp,
		},
		{
			Name:           "get APK MD5 with invalid JSON",
			Method:         http.MethodPost,
			URL:            "/api/wallet/get-apk-md5",
			Body:           bytes.NewReader([]byte(`{invalid json}`)),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"reason":"Invalid JSON format for the expected type"`,
			},
			TestAppFactory: setupWalletApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestWalletStoreActionResult(t *testing.T) {
	orgID, err := getOrgIDfromName("organizations", "userA's organization")
	require.NoError(t, err)

	successBodyBuf, successMp, err := tests.MockMultipartData(
		map[string]string{
			"wallet_identifier": "usera-s-organization/wallet123",
			"action_code":       "test_action",
		},
		"result",
	)
	require.NoError(t, err)
	fileWriter, err := successMp.CreateFormFile("result", "result.txt")
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte("dummy file content"))
	require.NoError(t, err)
	require.NoError(t, successMp.Close())

	missingFileBuf, missingFileMp, err := tests.MockMultipartData(
		map[string]string{
			"wallet_identifier": "usera-s-organization/wallet123",
			"action_code":       "test_action",
		},
	)
	require.NoError(t, err)
	require.NoError(t, missingFileMp.Close())

	scenarios := []tests.ApiScenario{
		{
			Name:   "store wallet action result successfully",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-action-result",
			Body:   bytes.NewReader(successBodyBuf.Bytes()),
			Headers: map[string]string{
				"Content-Type": successMp.FormDataContentType(),
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"success"`,
				`"fileName"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletApp(t)
				walletColl, err := app.FindCollectionByNameOrId("wallets")
				require.NoError(t, err)
				walletRecord := core.NewRecord(walletColl)
				walletRecord.Set("name", "wallet123")
				walletRecord.Set("owner", orgID)
				require.NoError(t, app.Save(walletRecord))
				return app
			},
		},
		{
			Name:   "store wallet action result missing file",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-action-result",
			Body:   bytes.NewReader(missingFileBuf.Bytes()),
			Headers: map[string]string{
				"Content-Type": missingFileMp.FormDataContentType(),
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"file"`,
				`"failed to read file"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletApp(t)
				walletColl, err := app.FindCollectionByNameOrId("wallets")
				require.NoError(t, err)
				walletRecord := core.NewRecord(walletColl)
				walletRecord.Set("name", "wallet123")
				walletRecord.Set("owner", orgID)
				require.NoError(t, app.Save(walletRecord))
				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
