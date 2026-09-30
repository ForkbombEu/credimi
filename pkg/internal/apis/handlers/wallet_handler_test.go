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
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase/apis"
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

func TestWalletDeleteTempVersion(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	t.Run("requires internal admin key", func(t *testing.T) {
		scenario := tests.ApiScenario{
			Name:           "missing key",
			Method:         http.MethodDelete,
			URL:            "/api/wallet/temp-version/missing",
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				"api_key_required",
			},
			TestAppFactory: setupWalletApp,
		}
		scenario.Test(t)
	})

	t.Run("deletes existing wallet version", func(t *testing.T) {
		app := setupWalletApp(t)
		defer app.Cleanup()

		versionID := createWalletAPKVersion(t, app, orgID, "wallet-temp-delete", "abc123")
		versionRecord, err := canonify.Resolve(app, versionID)
		require.NoError(t, err)

		baseRouter, err := apis.NewRouter(app)
		require.NoError(t, err)
		serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
		serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
			mux, err := e.Router.BuildMux()
			require.NoError(t, err)

			req := httptest.NewRequest(
				http.MethodDelete,
				"/api/wallet/temp-version/"+versionRecord.Id,
				jsonBody(map[string]any{
					"expected_owner_id":   orgID,
					"expected_identifier": versionID,
				}),
			)
			req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			require.Contains(t, rec.Body.String(), `"deleted":true`)
			return nil
		})
		require.NoError(t, serveErr)

		_, err = app.FindRecordById("wallet_versions", versionRecord.Id)
		require.Error(t, err)
	})

	t.Run("rejects existing wallet version without validation payload", func(t *testing.T) {
		app := setupWalletApp(t)
		defer app.Cleanup()

		versionID := createWalletAPKVersion(t, app, orgID, "wallet-temp-no-payload", "abc123")
		versionRecord, err := canonify.Resolve(app, versionID)
		require.NoError(t, err)

		baseRouter, err := apis.NewRouter(app)
		require.NoError(t, err)
		serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
		serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
			mux, err := e.Router.BuildMux()
			require.NoError(t, err)

			req := httptest.NewRequest(
				http.MethodDelete,
				"/api/wallet/temp-version/"+versionRecord.Id,
				nil,
			)
			req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Contains(t, rec.Body.String(), "delete validation payload is required")
			return nil
		})
		require.NoError(t, serveErr)

		_, err = app.FindRecordById("wallet_versions", versionRecord.Id)
		require.NoError(t, err)
	})

	t.Run("rejects owner mismatch", func(t *testing.T) {
		app := setupWalletApp(t)
		defer app.Cleanup()

		versionID := createWalletAPKVersion(t, app, orgID, "wallet-temp-owner-mismatch", "abc123")
		versionRecord, err := canonify.Resolve(app, versionID)
		require.NoError(t, err)

		baseRouter, err := apis.NewRouter(app)
		require.NoError(t, err)
		serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
		serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
			mux, err := e.Router.BuildMux()
			require.NoError(t, err)

			req := httptest.NewRequest(
				http.MethodDelete,
				"/api/wallet/temp-version/"+versionRecord.Id,
				jsonBody(map[string]any{
					"expected_owner_id":   "other-owner",
					"expected_identifier": versionID,
				}),
			)
			req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			require.Equal(t, http.StatusForbidden, rec.Code)
			require.Contains(t, rec.Body.String(), "owner mismatch")
			return nil
		})
		require.NoError(t, serveErr)

		_, err = app.FindRecordById("wallet_versions", versionRecord.Id)
		require.NoError(t, err)
	})

	t.Run("missing record is idempotent success", func(t *testing.T) {
		app := setupWalletApp(t)
		defer app.Cleanup()

		baseRouter, err := apis.NewRouter(app)
		require.NoError(t, err)
		serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
		serveErr := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
			mux, err := e.Router.BuildMux()
			require.NoError(t, err)

			req := httptest.NewRequest(
				http.MethodDelete,
				"/api/wallet/temp-version/missingrecord12",
				nil,
			)
			req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			require.Contains(t, rec.Body.String(), `"deleted":false`)
			return nil
		})
		require.NoError(t, serveErr)
	})
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
				walletRecord.Set("published", false)
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
	runnerUserRecord, err := getUserRecordFromName("userB")
	require.NoError(t, err)
	runnerUserToken, err := runnerUserRecord.NewAuthToken()
	require.NoError(t, err)

	const (
		reservedDevicePath = "usera-s-organization/test-runner/test-device"
		otherDevicePath    = "usera-s-organization/test-runner/other-device"
		publicDevicePath   = "userb-s-organization/public-runner/public-device"
	)

	successBody, successContentType := walletStorePipelineResultBody(
		t, reservedDevicePath, "android",
	)
	otherDeviceBody, otherDeviceContentType := walletStorePipelineResultBody(
		t, otherDevicePath, "android",
	)
	crossOrgRunnerBody, crossOrgRunnerContentType := walletStorePipelineResultBody(
		t, publicDevicePath, "android",
	)
	iosBody, iosContentType := walletStorePipelineResultBody(t, reservedDevicePath, "ios")

	// Prepare missing file multipart request
	var missingBody bytes.Buffer
	missingWriter := multipart.NewWriter(&missingBody)
	_ = missingWriter.WriteField("run_identifier", "usera-s-organization/workflow123-run123")
	_ = missingWriter.WriteField("device_identifier", reservedDevicePath)
	_ = missingWriter.WriteField("platform", "android")
	require.NoError(t, missingWriter.Close())

	var invalidPlatformBody bytes.Buffer
	invalidPlatformWriter := multipart.NewWriter(&invalidPlatformBody)
	_ = invalidPlatformWriter.WriteField(
		"run_identifier",
		"usera-s-organization/workflow123-run123",
	)
	_ = invalidPlatformWriter.WriteField("device_identifier", reservedDevicePath)
	_ = invalidPlatformWriter.WriteField("platform", "desktop")
	require.NoError(t, invalidPlatformWriter.Close())

	setupReservedDeviceApp := func(t testing.TB) *tests.TestApp {
		app := setupWalletApp(t)
		setupWalletPipelineTestRecords(t, app, orgID)
		runner := createWalletTestMobileRunner(t, app, orgID, "test-runner", false)
		device := createWalletTestMobileDevice(t, app, orgID, runner.Id, "test-device")
		createWalletTestMobileDevice(t, app, orgID, runner.Id, "other-device")
		addWalletPipelineResultDevice(t, app, device.Id)
		return app
	}
	setupPublicRunnerApp := func(t testing.TB, reserved bool) *tests.TestApp {
		app := setupReservedDeviceApp(t)
		setOrganizationPublished(t, app, orgID, true)

		runnerOrgID, err := pbutils.GetUserOrganizationID(app, runnerUserRecord.Id)
		require.NoError(t, err)
		runner := createWalletTestMobileRunner(t, app, runnerOrgID, "public-runner", true)
		device := createWalletTestMobileDevice(t, app, runnerOrgID, runner.Id, "public-device")
		if reserved {
			addWalletPipelineResultDevice(t, app, device.Id)
		}
		return app
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "store  pipeline result successfully",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-pipeline-result",
			Body:   bytes.NewReader(successBody),
			Headers: map[string]string{
				"Content-Type": successContentType,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"success"`,
				`"last_frame_file_name"`,
				`"screenshot_urls"`,
				`"video_file_name"`,
			},
			TestAppFactory: setupReservedDeviceApp,
		},
		{
			Name:   "store pipeline result successfully with authenticated user",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-pipeline-result",
			Body:   bytes.NewReader(successBody),
			Headers: map[string]string{
				"Authorization": "Bearer " + userToken,
				"Content-Type":  successContentType,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"success"`,
				`"last_frame_file_name"`,
				`"screenshot_urls"`,
				`"video_file_name"`,
			},
			TestAppFactory: setupReservedDeviceApp,
		},
		{
			Name:   "same organization user cannot store result from device outside the run",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-pipeline-result",
			Body:   bytes.NewReader(otherDeviceBody),
			Headers: map[string]string{
				"Authorization": "Bearer " + userToken,
				"Content-Type":  otherDeviceContentType,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"authorization"`,
				`"forbidden"`,
				`device did not run this pipeline result`,
			},
			TestAppFactory: setupReservedDeviceApp,
		},
		{
			Name:   "published runner owner can store result for published organization",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-pipeline-result",
			Body:   bytes.NewReader(crossOrgRunnerBody),
			Headers: map[string]string{
				"Authorization": "Bearer " + runnerUserToken,
				"Content-Type":  crossOrgRunnerContentType,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"success"`,
				`"device":"userb-s-organization/public-runner/public-device"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return setupPublicRunnerApp(t, true)
			},
		},
		{
			Name:   "published runner owner cannot store result for run its device did not execute",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-pipeline-result",
			Body:   bytes.NewReader(crossOrgRunnerBody),
			Headers: map[string]string{
				"Authorization": "Bearer " + runnerUserToken,
				"Content-Type":  crossOrgRunnerContentType,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"authorization"`,
				`"forbidden"`,
				`device did not run this pipeline result`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				return setupPublicRunnerApp(t, false)
			},
		},
		{
			Name:   "store ios pipeline result successfully",
			Method: http.MethodPost,
			URL:    "/api/wallet/store-pipeline-result",
			Body:   bytes.NewReader(iosBody),
			Headers: map[string]string{
				"Content-Type": iosContentType,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"success"`,
				`"last_frame_file_name"`,
				`"video_file_name"`,
				`"log_file_name"`,
			},
			TestAppFactory: setupReservedDeviceApp,
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
			TestAppFactory: setupReservedDeviceApp,
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
			TestAppFactory: setupReservedDeviceApp,
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

func walletStorePipelineResultBody(
	t testing.TB,
	deviceIdentifier string,
	platform string,
) ([]byte, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(
		t,
		writer.WriteField("run_identifier", "usera-s-organization/workflow123-run123"),
	)
	require.NoError(t, writer.WriteField("device_identifier", deviceIdentifier))
	require.NoError(t, writer.WriteField("platform", platform))

	videoHeader := textproto.MIMEHeader{}
	videoHeader.Set("Content-Disposition", `form-data; name="result_video"; filename="test.mp4"`)
	videoHeader.Set("Content-Type", "video/mp4")
	videoWriter, err := writer.CreatePart(videoHeader)
	require.NoError(t, err)
	// minimal valid MP4 header
	_, err = videoWriter.Write(
		[]byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'm', 'p', '4', '2'},
	)
	require.NoError(t, err)

	frameWriter, err := writer.CreateFormFile("last_frame", "frame.txt")
	require.NoError(t, err)
	_, err = frameWriter.Write([]byte("test frame content"))
	require.NoError(t, err)

	logWriter, err := writer.CreateFormFile("logfile", "log.txt")
	require.NoError(t, err)
	_, err = logWriter.Write([]byte("test log content"))
	require.NoError(t, err)

	require.NoError(t, writer.Close())
	return body.Bytes(), writer.FormDataContentType()
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
	requireHandlerErrorHandled(t, rec, err)
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
	requireHandlerErrorHandled(t, rec, err)
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
	requireHandlerErrorHandled(t, rec, err)
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
	requireHandlerErrorHandled(t, rec, err)
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

func createWalletTestMobileRunner(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	name string,
	published bool,
) *core.Record {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(coll)
	runner.Set("owner", orgID)
	runner.Set("name", name)
	runner.Set("ip", "http://127.0.0.1:1")
	runner.Set("type", "android_emulator")
	runner.Set("published", published)
	require.NoError(t, app.Save(runner))
	return runner
}

func createWalletTestMobileDevice(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	runnerID string,
	name string,
) *core.Record {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(coll)
	device.Set("owner", orgID)
	device.Set("runner", runnerID)
	device.Set("name", name)
	device.Set("canonified_name", name)
	device.Set("type", "android_emulator")
	require.NoError(t, app.Save(device))
	return device
}

func addWalletPipelineResultDevice(t testing.TB, app *tests.TestApp, deviceID string) {
	t.Helper()

	result, err := app.FindFirstRecordByFilter(
		"pipeline_results",
		"workflow_id = 'workflow123' && run_id = 'run123'",
	)
	require.NoError(t, err)
	result.Set("devices", append(result.GetStringSlice("devices"), deviceID))
	require.NoError(t, app.Save(result))
}
