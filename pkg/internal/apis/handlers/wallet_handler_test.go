// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
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
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:   "get APK MD5 with valid wallet identifier",
			Method: http.MethodPost,
			URL:    "/api/wallet/get-apk-md5-or-etag",
			Body: jsonBody(map[string]any{
				"wallet_identifier":         "usera-s-organization/wallet123",
				"wallet_version_identifier": "",
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"apk_name"`,
				`"apk_identifier"`,
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
			URL:    "/api/wallet/get-apk-md5-or-etag",
			Body: jsonBody(map[string]any{
				"wallet_identifier":         "",
				"wallet_version_identifier": "usera-s-organization/wallet234/2-0-0",
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"apk_name"`,
				`"apk_identifier"`,
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
			URL:            "/api/wallet/get-apk-md5-or-etag",
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
			URL:            "/api/wallet/get-apk-md5-or-etag",
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
			URL:            "/api/wallet/get-apk-md5-or-etag",
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

func TestWalletStorePipelineResult(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	// Prepare the success multipart request with MP4
	var successBody bytes.Buffer
	successWriter := multipart.NewWriter(&successBody)

	// add form fields
	_ = successWriter.WriteField("run_identifier", "usera-s-organization/workflow123-run123")
	_ = successWriter.WriteField("runner_identifier", "usera-s-organization/test-runner")

	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition", `form-data; name="result_video"; filename="test.mp4"`)
	partHeader.Set("Content-Type", "video/mp4")

	fileWriter, err := successWriter.CreatePart(partHeader)
	require.NoError(t, err)

	// write minimal valid MP4 header
	mp4Header := []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'm', 'p', '4', '2'}
	_, err = fileWriter.Write(mp4Header)
	require.NoError(t, err)

	frameWriter, err := successWriter.CreateFormFile("last_frame", "frame.txt")
	require.NoError(t, err)

	_, err = frameWriter.Write([]byte("test frame content"))
	require.NoError(t, err)

	require.NoError(t, successWriter.Close())

	// Prepare missing file multipart request
	var missingBody bytes.Buffer
	missingWriter := multipart.NewWriter(&missingBody)
	_ = missingWriter.WriteField("run_identifier", "usera-s-organization/workflow123-run123")
	_ = missingWriter.WriteField("runner_identifier", "usera-s-organization/test-runner")
	require.NoError(t, missingWriter.Close())

	scenarios := []tests.ApiScenario{
		{
			Name:   "store  pipeline result successfully",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-pipeline-result",
			Body:   bytes.NewReader(successBody.Bytes()),
			Headers: map[string]string{
				"Content-Type": successWriter.FormDataContentType(),
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"success"`,
				`"last_frame_file_name"`,
				`"video_file_name"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletApp(t)
				setupWalletPipelineTestRecords(t, app, orgID)
				return app
			},
		},
		{
			Name:   "store pipeline result missing files",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-pipeline-result",
			Body:   bytes.NewReader(missingBody.Bytes()),
			Headers: map[string]string{
				"Content-Type": missingWriter.FormDataContentType(),
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"file"`,
				`failed to read file for field result_video"`,
				`failed to read file for field last_frame"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletApp(t)
				setupWalletPipelineTestRecords(t, app, orgID)
				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func setupWalletPipelineTestRecords(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
) {
	t.Helper()

	// Wallet
	walletColl, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(walletColl)
	wallet.Set("name", "wallet123")
	wallet.Set("owner", orgID)
	require.NoError(t, app.Save(wallet))

	// Pipeline
	pipelineColl, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	pipeline := core.NewRecord(pipelineColl)
	pipeline.Set("name", "pipeline123")
	pipeline.Set("owner", orgID)
	pipeline.Set("description", "Test pipeline")
	pipeline.Set("steps", map[string]string{"step1": "do something"})
	pipeline.Set("yaml", "name: Test Pipeline")
	require.NoError(t, app.Save(pipeline))

	// Pipeline Results
	pipelineResultsColl, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	run := core.NewRecord(pipelineResultsColl)
	run.Set("workflow_id", "workflow123")
	run.Set("run_id", "run123")
	run.Set("owner", orgID)
	run.Set("pipeline", pipeline.Id)
	require.NoError(t, app.Save(run))
}
