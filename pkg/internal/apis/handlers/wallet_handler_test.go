// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func setupWalletApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	canonify.RegisterCanonifyHooks(app)
	WalletTemporalInternalRoutes.Add(app)
	seedInternalAdminKey(t, app)
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

type walletWorkflowStub struct {
	startFn func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error)
}

func (w walletWorkflowStub) Start(
	namespace string,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return w.startFn(namespace, input)
}

func TestWalletGetInstallerMD5OrETag(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	userToken, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:   "get Android installer MD5 with valid wallet identifier",
			Method: http.MethodPost,
			URL:    "/api/wallet/get-installer-md5-or-etag",
			Body: jsonBody(map[string]any{
				"wallet_identifier":         "usera-s-organization/wallet123",
				"wallet_version_identifier": "",
				"platform":                  "android",
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"installer_name"`,
				`"installer_identifier"`,
				`"app.apk"`,
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
			Name:   "get Android installer MD5 with valid version identifier",
			Method: http.MethodPost,
			URL:    "/api/wallet/get-installer-md5-or-etag",
			Body: jsonBody(map[string]any{
				"wallet_identifier":         "",
				"wallet_version_identifier": "usera-s-organization/wallet234/2-0-0",
				"platform":                  "android",
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"installer_name"`,
				`"installer_identifier"`,
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
			Name:   "authenticated user can get installer for own organization",
			Method: http.MethodPost,
			URL:    "/api/wallet/get-installer-md5-or-etag",
			Body: jsonBody(map[string]any{
				"wallet_identifier":         "usera-s-organization/wallet-user-auth",
				"wallet_version_identifier": "",
				"platform":                  "android",
			}),
			Headers: map[string]string{
				"Authorization": "Bearer " + userToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"installer_name"`,
				`"installer_identifier"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletApp(t)

				walletColl, err := app.FindCollectionByNameOrId("wallets")
				require.NoError(t, err)
				walletRecord := core.NewRecord(walletColl)
				walletRecord.Set("name", "wallet-user-auth")
				walletRecord.Set("owner", orgID)
				require.NoError(t, app.Save(walletRecord))

				walletVersionColl, err := app.FindCollectionByNameOrId("wallet_versions")
				require.NoError(t, err)
				versionRecord := core.NewRecord(walletVersionColl)
				versionRecord.Set("wallet", walletRecord.Id)
				versionRecord.Set("tag", "4.0.0")
				versionRecord.Set("owner", orgID)
				apkFile := NewTestFile("app.apk", []byte("dummy apk content"))
				versionRecord.Set("android_installer", []*filesystem.File{apkFile})
				require.NoError(t, app.Save(versionRecord))

				return app
			},
		},
		{
			Name:   "authenticated user can get installer for published wallet from another organization",
			Method: http.MethodPost,
			URL:    "/api/wallet/get-installer-md5-or-etag",
			Body: jsonBody(map[string]any{
				"wallet_identifier":         "other-org/wallet-published",
				"wallet_version_identifier": "",
				"platform":                  "android",
			}),
			Headers: map[string]string{
				"Authorization": "Bearer " + userToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"installer_name"`,
				`"installer_identifier"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletApp(t)

				orgColl, err := app.FindCollectionByNameOrId("organizations")
				require.NoError(t, err)
				otherOrg := core.NewRecord(orgColl)
				otherOrg.Set("name", "Other Org")
				otherOrg.Set("canonified_name", "other-org")
				require.NoError(t, app.Save(otherOrg))

				walletColl, err := app.FindCollectionByNameOrId("wallets")
				require.NoError(t, err)
				walletRecord := core.NewRecord(walletColl)
				walletRecord.Set("name", "wallet-published")
				walletRecord.Set("owner", otherOrg.Id)
				walletRecord.Set("published", true)
				require.NoError(t, app.Save(walletRecord))

				walletVersionColl, err := app.FindCollectionByNameOrId("wallet_versions")
				require.NoError(t, err)
				versionRecord := core.NewRecord(walletVersionColl)
				versionRecord.Set("wallet", walletRecord.Id)
				versionRecord.Set("tag", "1.0.0")
				versionRecord.Set("owner", otherOrg.Id)
				apkFile := NewTestFile("app.apk", []byte("dummy apk content"))
				versionRecord.Set("android_installer", []*filesystem.File{apkFile})
				require.NoError(t, app.Save(versionRecord))

				return app
			},
		},
		{
			Name:   "authenticated user cannot get installer for another organization",
			Method: http.MethodPost,
			URL:    "/api/wallet/get-installer-md5-or-etag",
			Body: jsonBody(map[string]any{
				"wallet_identifier":         "other-org/wallet-forbidden",
				"wallet_version_identifier": "",
				"platform":                  "android",
			}),
			Headers: map[string]string{
				"Authorization": "Bearer " + userToken,
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"authorization"`,
				`"forbidden"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletApp(t)

				orgColl, err := app.FindCollectionByNameOrId("organizations")
				require.NoError(t, err)
				otherOrg := core.NewRecord(orgColl)
				otherOrg.Set("name", "Other Org")
				otherOrg.Set("canonified_name", "other-org")
				require.NoError(t, app.Save(otherOrg))

				walletColl, err := app.FindCollectionByNameOrId("wallets")
				require.NoError(t, err)
				walletRecord := core.NewRecord(walletColl)
				walletRecord.Set("name", "wallet-forbidden")
				walletRecord.Set("owner", otherOrg.Id)
				require.NoError(t, app.Save(walletRecord))

				walletVersionColl, err := app.FindCollectionByNameOrId("wallet_versions")
				require.NoError(t, err)
				versionRecord := core.NewRecord(walletVersionColl)
				versionRecord.Set("wallet", walletRecord.Id)
				versionRecord.Set("tag", "1.0.0")
				versionRecord.Set("owner", otherOrg.Id)
				apkFile := NewTestFile("app.apk", []byte("dummy apk content"))
				versionRecord.Set("android_installer", []*filesystem.File{apkFile})
				require.NoError(t, app.Save(versionRecord))

				return app
			},
		},
		{
			Name:   "get iOS installer MD5 with valid wallet identifier",
			Method: http.MethodPost,
			URL:    "/api/wallet/get-installer-md5-or-etag",
			Body: jsonBody(map[string]any{
				"wallet_identifier":         "usera-s-organization/walletios",
				"wallet_version_identifier": "",
				"platform":                  "ios",
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"installer_name"`,
				`"installer_identifier"`,
				`"app.ipa"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletApp(t)

				walletColl, err := app.FindCollectionByNameOrId("wallets")
				require.NoError(t, err)
				walletRecord := core.NewRecord(walletColl)
				walletRecord.Set("name", "walletios")
				walletRecord.Set("owner", orgID)
				require.NoError(t, app.Save(walletRecord))

				walletVersionColl, err := app.FindCollectionByNameOrId("wallet_versions")
				require.NoError(t, err)
				versionRecord := core.NewRecord(walletVersionColl)
				versionRecord.Set("wallet", walletRecord.Id)
				versionRecord.Set("tag", "3.0.0")
				versionRecord.Set("owner", orgID)
				apkFile := NewTestFile("android.apk", []byte("dummy android installer content"))
				versionRecord.Set("android_installer", []*filesystem.File{apkFile})
				iosFile := NewTestFile("app.ipa", []byte("dummy ios installer content"))
				versionRecord.Set("ios_installer", []*filesystem.File{iosFile})
				require.NoError(t, app.Save(versionRecord))

				return app
			},
		},
		{
			Name:           "get installer MD5 with missing identifiers",
			Method:         http.MethodPost,
			URL:            "/api/wallet/get-installer-md5-or-etag",
			Body:           jsonBody(map[string]any{"platform": "android"}),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"identifier"`,
				`"no identifier provided"`,
			},
			TestAppFactory: setupWalletApp,
		},
		{
			Name:   "get installer MD5 with non-existent wallet",
			Method: http.MethodPost,
			URL:    "/api/wallet/get-installer-md5-or-etag",
			Body: jsonBody(map[string]any{
				"wallet_identifier": "nonexistent",
				"platform":          "android",
			}),
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"wallet_version"`,
				`"wallet version not found"`,
			},
			TestAppFactory: setupWalletApp,
		},
		{
			Name:   "get installer MD5 with invalid platform",
			Method: http.MethodPost,
			URL:    "/api/wallet/get-installer-md5-or-etag",
			Body: jsonBody(
				map[string]any{"wallet_identifier": "nonexistent", "platform": "desktop"},
			),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"platform"`,
				`"invalid platform"`,
			},
			TestAppFactory: setupWalletApp,
		},
		{
			Name:           "get installer MD5 with invalid JSON",
			Method:         http.MethodPost,
			URL:            "/api/wallet/get-installer-md5-or-etag",
			Body:           bytes.NewReader([]byte(`{invalid json}`)),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"reason":"Invalid JSON format for the expected type"`,
			},
			TestAppFactory: setupWalletApp,
		},
		{
			Name:   "skip installer returns version id without lookup",
			Method: http.MethodPost,
			URL:    "/api/wallet/get-installer-md5-or-etag",
			Body: jsonBody(map[string]any{
				"wallet_version_identifier": "installed_from_external_source",
				"platform":                  "android",
				"skip_installer":            true,
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"installer_name":""`,
				`"installer_identifier":""`,
				`"version_id":"installed_from_external_source"`,
			},
			TestAppFactory: setupWalletApp,
		},
	}

	for _, scenario := range scenarios {
		if scenario.Headers == nil {
			scenario.Headers = map[string]string{}
		}
		if _, ok := scenario.Headers["Authorization"]; !ok {
			scenario.Headers["Credimi-Api-Key"] = "internal-test-api-key"
		}
		scenario.Test(t)
	}
}

func TestWalletStorePipelineResult(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	userToken, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	// Prepare the success multipart request with MP4
	var successBody bytes.Buffer
	successWriter := multipart.NewWriter(&successBody)

	// add form fields
	_ = successWriter.WriteField("run_identifier", "usera-s-organization/workflow123-run123")
	_ = successWriter.WriteField("runner_identifier", "usera-s-organization/test-runner")
	_ = successWriter.WriteField("platform", "android")

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

	logWriter, err := successWriter.CreateFormFile("logfile", "log.txt")
	require.NoError(t, err)

	_, err = logWriter.Write([]byte("test log content"))
	require.NoError(t, err)

	require.NoError(t, successWriter.Close())

	var iosBody bytes.Buffer
	iosWriter := multipart.NewWriter(&iosBody)
	_ = iosWriter.WriteField("run_identifier", "usera-s-organization/workflow123-run123")
	_ = iosWriter.WriteField("runner_identifier", "usera-s-organization/test-runner")
	_ = iosWriter.WriteField("platform", "ios")

	iosVideoWriter, err := iosWriter.CreatePart(partHeader)
	require.NoError(t, err)
	_, err = iosVideoWriter.Write(mp4Header)
	require.NoError(t, err)

	iosFrameWriter, err := iosWriter.CreateFormFile("last_frame", "frame.txt")
	require.NoError(t, err)
	_, err = iosFrameWriter.Write([]byte("test frame content"))
	require.NoError(t, err)

	iosLogWriter, err := iosWriter.CreateFormFile("logfile", "ios-log.txt")
	require.NoError(t, err)
	_, err = iosLogWriter.Write([]byte("test ios log content"))
	require.NoError(t, err)

	require.NoError(t, iosWriter.Close())

	// Prepare missing file multipart request
	var missingBody bytes.Buffer
	missingWriter := multipart.NewWriter(&missingBody)
	_ = missingWriter.WriteField("run_identifier", "usera-s-organization/workflow123-run123")
	_ = missingWriter.WriteField("runner_identifier", "usera-s-organization/test-runner")
	_ = missingWriter.WriteField("platform", "android")
	require.NoError(t, missingWriter.Close())

	var invalidPlatformBody bytes.Buffer
	invalidPlatformWriter := multipart.NewWriter(&invalidPlatformBody)
	_ = invalidPlatformWriter.WriteField(
		"run_identifier",
		"usera-s-organization/workflow123-run123",
	)
	_ = invalidPlatformWriter.WriteField("runner_identifier", "usera-s-organization/test-runner")
	_ = invalidPlatformWriter.WriteField("platform", "desktop")
	require.NoError(t, invalidPlatformWriter.Close())

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
			Name:   "store pipeline result successfully with authenticated user",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-pipeline-result",
			Body:   bytes.NewReader(successBody.Bytes()),
			Headers: map[string]string{
				"Authorization": "Bearer " + userToken,
				"Content-Type":  successWriter.FormDataContentType(),
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
			Name:   "store ios pipeline result successfully",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-pipeline-result",
			Body:   bytes.NewReader(iosBody.Bytes()),
			Headers: map[string]string{
				"Content-Type": iosWriter.FormDataContentType(),
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"success"`,
				`"last_frame_file_name"`,
				`"video_file_name"`,
				`"log_file_name"`,
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
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletApp(t)
				setupWalletPipelineTestRecords(t, app, orgID)
				return app
			},
		},
		{
			Name:   "store pipeline result with invalid platform",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-pipeline-result",
			Body:   bytes.NewReader(invalidPlatformBody.Bytes()),
			Headers: map[string]string{
				"Content-Type": invalidPlatformWriter.FormDataContentType(),
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"platform"`,
				`"invalid platform"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupWalletApp(t)
				setupWalletPipelineTestRecords(t, app, orgID)
				return app
			},
		},
	}

	for _, scenario := range scenarios {
		if scenario.Headers == nil {
			scenario.Headers = map[string]string{}
		}
		if _, ok := scenario.Headers["Authorization"]; !ok {
			scenario.Headers["Credimi-Api-Key"] = "internal-test-api-key"
		}
		scenario.Test(t)
	}
}

func TestHandleWalletStartCheckInvalidJSON(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/wallet/start-check",
		bytes.NewBufferString("{"),
	)
	rec := httptest.NewRecorder()

	err = HandleWalletStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.Error(t, err)
}

func TestHandleWalletStartCheckWorkflowStartError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origFactory := walletWorkflowFactory
	origClient := walletTemporalClient
	origWait := walletWaitForPartialResult
	t.Cleanup(func() {
		walletWorkflowFactory = origFactory
		walletTemporalClient = origClient
		walletWaitForPartialResult = origWait
	})

	walletWorkflowFactory = func() walletWorkflowStarter {
		return walletWorkflowStub{
			startFn: func(_ string, _ workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
				return workflowengine.WorkflowResult{}, errors.New("start failed")
			},
		}
	}
	walletTemporalClient = func(_ string) (client.Client, error) {
		return temporalmocks.NewClient(t), nil
	}
	walletWaitForPartialResult = func(
		_ client.Client,
		_, _, _ string,
		_ time.Duration,
		_ time.Duration,
	) (map[string]any, error) {
		return nil, errors.New("not reached")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/wallet/start-check",
		bytes.NewBufferString(`{"walletURL":"https://example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleWalletStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleWalletStartCheckTemporalClientError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origFactory := walletWorkflowFactory
	origClient := walletTemporalClient
	origWait := walletWaitForPartialResult
	t.Cleanup(func() {
		walletWorkflowFactory = origFactory
		walletTemporalClient = origClient
		walletWaitForPartialResult = origWait
	})

	walletWorkflowFactory = func() walletWorkflowStarter {
		return walletWorkflowStub{
			startFn: func(_ string, _ workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
				return workflowengine.WorkflowResult{
					WorkflowID:    "wf-1",
					WorkflowRunID: "run-1",
				}, nil
			},
		}
	}
	walletTemporalClient = func(_ string) (client.Client, error) {
		return nil, errors.New("no client")
	}
	walletWaitForPartialResult = func(
		_ client.Client,
		_, _, _ string,
		_ time.Duration,
		_ time.Duration,
	) (map[string]any, error) {
		return nil, errors.New("not reached")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/wallet/start-check",
		bytes.NewBufferString(`{"walletURL":"https://example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleWalletStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleWalletStartCheckPartialResultError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origFactory := walletWorkflowFactory
	origClient := walletTemporalClient
	origWait := walletWaitForPartialResult
	t.Cleanup(func() {
		walletWorkflowFactory = origFactory
		walletTemporalClient = origClient
		walletWaitForPartialResult = origWait
	})

	walletWorkflowFactory = func() walletWorkflowStarter {
		return walletWorkflowStub{
			startFn: func(_ string, _ workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
				return workflowengine.WorkflowResult{
					WorkflowID:    "wf-1",
					WorkflowRunID: "run-1",
				}, nil
			},
		}
	}
	walletTemporalClient = func(_ string) (client.Client, error) {
		return temporalmocks.NewClient(t), nil
	}
	walletWaitForPartialResult = func(
		_ client.Client,
		_, _, _ string,
		_ time.Duration,
		_ time.Duration,
	) (map[string]any, error) {
		return nil, errors.New("query failed")
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/wallet/start-check",
		bytes.NewBufferString(`{"walletURL":"https://example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleWalletStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleWalletStartCheckMetadataError(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	origFactory := walletWorkflowFactory
	origClient := walletTemporalClient
	origWait := walletWaitForPartialResult
	t.Cleanup(func() {
		walletWorkflowFactory = origFactory
		walletTemporalClient = origClient
		walletWaitForPartialResult = origWait
	})

	walletWorkflowFactory = func() walletWorkflowStarter {
		return walletWorkflowStub{
			startFn: func(_ string, _ workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
				return workflowengine.WorkflowResult{
					WorkflowID:    "wf-1",
					WorkflowRunID: "run-1",
				}, nil
			},
		}
	}
	walletTemporalClient = func(_ string) (client.Client, error) {
		return temporalmocks.NewClient(t), nil
	}
	walletWaitForPartialResult = func(
		_ client.Client,
		_, _, _ string,
		_ time.Duration,
		_ time.Duration,
	) (map[string]any, error) {
		return map[string]any{
			"storeType": "google",
			"metadata":  "not-a-map",
		}, nil
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/wallet/start-check",
		bytes.NewBufferString(`{"walletURL":"https://example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleWalletStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandleWalletStartCheckSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://app.example.com"

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	origFactory := walletWorkflowFactory
	origClient := walletTemporalClient
	origWait := walletWaitForPartialResult
	t.Cleanup(func() {
		walletWorkflowFactory = origFactory
		walletTemporalClient = origClient
		walletWaitForPartialResult = origWait
	})

	walletWorkflowFactory = func() walletWorkflowStarter {
		return walletWorkflowStub{
			startFn: func(_ string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
				payload := input.Payload.(workflows.WalletWorkflowPayload)
				require.Equal(t, "https://example.com", payload.URL)
				return workflowengine.WorkflowResult{
					WorkflowID:    "wf-1",
					WorkflowRunID: "run-1",
				}, nil
			},
		}
	}
	walletTemporalClient = func(_ string) (client.Client, error) {
		return temporalmocks.NewClient(t), nil
	}
	walletWaitForPartialResult = func(
		_ client.Client,
		_, _, _ string,
		_ time.Duration,
		_ time.Duration,
	) (map[string]any, error) {
		return map[string]any{
			"storeType": "google",
			"metadata": map[string]any{
				"title":            "Wallet",
				"icon":             "logo.png",
				"appId":            "com.example",
				"developerWebsite": "https://example.com",
				"description":      "desc",
			},
		}, nil
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/wallet/start-check",
		bytes.NewBufferString(`{"walletURL":"https://example.com"}`),
	)
	rec := httptest.NewRecorder()

	err = HandleWalletStartCheck()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&payload))
	require.Equal(t, "google", payload["type"])
	require.Equal(t, "Wallet", payload["name"])
	require.Equal(t, "logo.png", payload["logo"])
	require.Equal(t, "com.example", payload["google_app_id"])
	require.Equal(t, "https://example.com", payload["playstore_url"])
	require.Equal(t, orgID, payload["owner"])
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
