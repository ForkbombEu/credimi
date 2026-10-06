// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

func covRunnerWalletRequest(
	t *testing.T,
	app *tests.TestApp,
	auth *core.Record,
	handler func(*core.RequestEvent) error,
	contentType string,
	body []byte,
) (int, map[string]any, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/wallet", bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	err := handler(
		&core.RequestEvent{App: app, Auth: auth, Event: router.Event{Request: req, Response: rec}},
	)
	if err != nil {
		requireHandlerErrorHandled(t, rec, err)
	}
	raw := rec.Body.String()
	if rec.Code != http.StatusOK {
		return rec.Code, nil, raw
	}
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &decoded))
	return rec.Code, decoded, raw
}

// covRunnerSeedWallet stores a wallet owned by orgID with one version tagged
// 1.0.0 that only has an Android installer.
func covRunnerSeedWallet(
	t *testing.T,
	app *tests.TestApp,
	orgID, name string,
	published, downloadable bool,
) {
	t.Helper()
	walletColl, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(walletColl)
	wallet.Set("name", name)
	wallet.Set("owner", orgID)
	wallet.Set("published", published)
	require.NoError(t, app.Save(wallet))

	versionColl, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)
	version := core.NewRecord(versionColl)
	version.Set("wallet", wallet.Id)
	version.Set("tag", "1.0.0")
	version.Set("owner", orgID)
	version.Set("downloadable", downloadable)
	version.Set("android_installer", []*filesystem.File{NewTestFile("app.apk", []byte("apk"))})
	require.NoError(t, app.Save(version))
}

func TestWalletGetInstallerMD5OrETagValidationAndAccess(t *testing.T) {
	app := setupWalletApp(t)
	defer app.Cleanup()
	userA, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	userB, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
	require.NoError(t, err)
	orgA, err := pbutils.GetUserOrganizationID(app, userA.Id)
	require.NoError(t, err)
	covRunnerSeedWallet(t, app, orgA, "private-wallet", false, true)
	covRunnerSeedWallet(t, app, orgA, "public-locked", true, false)

	testCases := []struct {
		name        string
		auth        *core.Record
		body        string
		wantStatus  int
		wantContain string
		wantVersion string
	}{
		{
			name:        "no identifier",
			auth:        userA,
			body:        `{"platform":"android"}`,
			wantStatus:  http.StatusBadRequest,
			wantContain: "no identifier provided",
		},
		{
			name:        "unsupported platform",
			auth:        userA,
			body:        `{"wallet_identifier":"usera-s-organization/private-wallet","platform":"windows"}`,
			wantStatus:  http.StatusBadRequest,
			wantContain: "invalid platform",
		},
		{
			name:        "skip installer returns the normalized version without lookup",
			auth:        userB,
			body:        `{"wallet_version_identifier":"/nowhere/ghost/1-0-0","platform":" IOS ","skip_installer":true}`,
			wantStatus:  http.StatusOK,
			wantVersion: "nowhere/ghost/1-0-0",
		},
		{
			name:        "unknown wallet",
			auth:        userA,
			body:        `{"wallet_identifier":"usera-s-organization/ghost","platform":"android"}`,
			wantStatus:  http.StatusNotFound,
			wantContain: "wallet version not found",
		},
		{
			name:        "unknown version",
			auth:        userA,
			body:        `{"wallet_version_identifier":"usera-s-organization/private-wallet/9-9-9","platform":"android"}`,
			wantStatus:  http.StatusNotFound,
			wantContain: "wallet version not found",
		},
		{
			name:        "platform without installer",
			auth:        superuser,
			body:        `{"wallet_identifier":"usera-s-organization/private-wallet","platform":"ios"}`,
			wantStatus:  http.StatusNotFound,
			wantContain: "no ios_installer file found",
		},
		{
			name:        "foreign user and private wallet",
			auth:        userB,
			body:        `{"wallet_identifier":"usera-s-organization/private-wallet","platform":"android"}`,
			wantStatus:  http.StatusForbidden,
			wantContain: "record does not belong",
		},
		{
			name:        "foreign user and published wallet with non downloadable version",
			auth:        userB,
			body:        `{"wallet_identifier":"usera-s-organization/public-locked","platform":"android"}`,
			wantStatus:  http.StatusForbidden,
			wantContain: "record does not belong",
		},
		{
			name:        "superuser by wallet identifier gets wallet:tag version id",
			auth:        superuser,
			body:        `{"wallet_identifier":"usera-s-organization/private-wallet","platform":"Android"}`,
			wantStatus:  http.StatusOK,
			wantVersion: "usera-s-organization/private-wallet:1-0-0",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status, body, raw := covRunnerWalletRequest(
				t,
				app,
				tc.auth,
				HandleWalletGetInstallerMD5OrETag(),
				"application/json",
				[]byte(tc.body),
			)
			require.Equal(t, tc.wantStatus, status, raw)
			if tc.wantContain != "" {
				assert.Contains(t, raw, tc.wantContain)
			}
			if tc.wantVersion != "" {
				assert.Equal(t, tc.wantVersion, body["version_id"])
			}
		})
	}
}

func TestWalletStorePipelineResultRejections(t *testing.T) {
	setup := func(t *testing.T, publishResultOrg bool) (*tests.TestApp, map[string]*core.Record) {
		t.Helper()
		app := setupWalletApp(t)
		t.Cleanup(app.Cleanup)
		userA, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)
		userB, err := app.FindAuthRecordByEmail("users", "userB@example.org")
		require.NoError(t, err)
		orgA, err := pbutils.GetUserOrganizationID(app, userA.Id)
		require.NoError(t, err)
		orgB, err := pbutils.GetUserOrganizationID(app, userB.Id)
		require.NoError(t, err)

		setupWalletPipelineTestRecords(t, app, orgA)
		runner := createWalletTestMobileRunner(t, app, orgA, "test-runner", false)
		device := createWalletTestMobileDevice(t, app, orgA, runner.Id, "test-device")
		addWalletPipelineResultDevice(t, app, device.Id)
		privateRunner := createWalletTestMobileRunner(t, app, orgB, "private-runner", false)
		privateDevice := createWalletTestMobileDevice(
			t,
			app,
			orgB,
			privateRunner.Id,
			"private-device",
		)
		addWalletPipelineResultDevice(t, app, privateDevice.Id)
		if publishResultOrg {
			setOrganizationPublished(t, app, orgA, true)
		}
		return app, map[string]*core.Record{"userA": userA, "userB": userB}
	}

	testCases := []struct {
		name        string
		publish     bool
		auth        string
		device      string
		run         string
		platform    string
		omitLog     bool
		rawBody     string
		saveFails   bool
		wantStatus  int
		wantContain string
	}{
		{
			name:        "body is not multipart",
			auth:        "userA",
			rawBody:     "not multipart",
			wantStatus:  http.StatusBadRequest,
			wantContain: "failed to parse multipart form",
		},
		{
			name:        "unknown run",
			auth:        "userA",
			device:      "usera-s-organization/test-runner/test-device",
			run:         "usera-s-organization/ghost-run",
			wantStatus:  http.StatusNotFound,
			wantContain: "record not found",
		},
		{
			name:        "unknown device",
			auth:        "userA",
			device:      "usera-s-organization/test-runner/ghost",
			wantStatus:  http.StatusBadRequest,
			wantContain: "failed_to_resolve_device_identifier",
		},
		{
			name:        "runner identifier instead of device",
			auth:        "userA",
			device:      "usera-s-organization/test-runner",
			wantStatus:  http.StatusForbidden,
			wantContain: "device did not run this pipeline result",
		},
		{
			name:        "foreign runner owner and unpublished result organization",
			auth:        "userB",
			device:      "userb-s-organization/private-runner/private-device",
			wantStatus:  http.StatusForbidden,
			wantContain: "record does not belong",
		},
		{
			name:        "foreign private runner and published result organization",
			publish:     true,
			auth:        "userB",
			device:      "userb-s-organization/private-runner/private-device",
			wantStatus:  http.StatusForbidden,
			wantContain: "record does not belong",
		},
		{
			name:        "foreign user reporting another organization device",
			publish:     true,
			auth:        "userB",
			device:      "usera-s-organization/test-runner/test-device",
			wantStatus:  http.StatusForbidden,
			wantContain: "record does not belong",
		},
		{
			name:        "missing log file",
			auth:        "userA",
			device:      "usera-s-organization/test-runner/test-device",
			omitLog:     true,
			wantStatus:  http.StatusBadRequest,
			wantContain: "failed to read file for field logfile",
		},
		{
			name:        "record save failure",
			auth:        "userA",
			device:      "usera-s-organization/test-runner/test-device",
			saveFails:   true,
			wantStatus:  http.StatusInternalServerError,
			wantContain: "failed to save record with uploaded file",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app, users := setup(t, tc.publish)
			if tc.saveFails {
				app.OnRecordUpdate("pipeline_results").BindFunc(func(*core.RecordEvent) error {
					return errors.New("boom")
				})
			}
			body, contentType := []byte(tc.rawBody), "text/plain"
			if tc.rawBody == "" {
				run := tc.run
				if run == "" {
					run = "usera-s-organization/workflow123-run123"
				}
				body, contentType = covRunnerStoreBody(t, run, tc.device, "android", !tc.omitLog)
			}
			status, _, raw := covRunnerWalletRequest(
				t, app, users[tc.auth], HandleWalletStorePipelineResult(), contentType, body,
			)
			require.Equal(t, tc.wantStatus, status, raw)
			assert.Contains(t, raw, tc.wantContain)
		})
	}
}

func TestWalletStorePipelineResultAppendsArtifacts(t *testing.T) {
	app := setupWalletApp(t)
	defer app.Cleanup()
	userA, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgA, err := pbutils.GetUserOrganizationID(app, userA.Id)
	require.NoError(t, err)
	setupWalletPipelineTestRecords(t, app, orgA)
	runner := createWalletTestMobileRunner(t, app, orgA, "test-runner", false)
	device := createWalletTestMobileDevice(t, app, orgA, runner.Id, "test-device")
	addWalletPipelineResultDevice(t, app, device.Id)

	for i := range 2 {
		body, contentType := covRunnerStoreBody(
			t, "usera-s-organization/workflow123-run123",
			"/usera-s-organization/test-runner/test-device/", "ios", true,
		)
		status, decoded, raw := covRunnerWalletRequest(
			t, app, userA, HandleWalletStorePipelineResult(), contentType, body,
		)
		require.Equal(t, http.StatusOK, status, raw)
		assert.Equal(
			t,
			"usera-s-organization-test-runner-test-device_logfile.zip",
			decoded["log_file_name"],
		)
		assert.Len(t, decoded["result_urls"], i+1)
		assert.Len(t, decoded["log_urls"], i+1)
	}

	result, err := app.FindFirstRecordByFilter(
		"pipeline_results", "workflow_id = 'workflow123' && run_id = 'run123'",
	)
	require.NoError(t, err)
	assert.Len(t, result.GetStringSlice("video_results"), 2)
	assert.Len(t, result.GetStringSlice("screenshots"), 2)
	assert.Len(t, result.GetStringSlice("ios_logstreams"), 2)
	assert.Empty(t, result.GetStringSlice("logcats"))
}

func covRunnerStoreBody(t *testing.T, run, device, platform string, withLog bool) ([]byte, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("run_identifier", run))
	require.NoError(t, writer.WriteField("device_identifier", device))
	require.NoError(t, writer.WriteField("platform", platform))
	parts := map[string]string{"result_video": "video.mp4", "last_frame": "frame.png"}
	if withLog {
		parts["logfile"] = "device.log"
	}
	for field, filename := range parts {
		part, err := writer.CreateFormFile(field, filename)
		require.NoError(t, err)
		_, err = part.Write([]byte(strings.Repeat("x", 16)))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return body.Bytes(), writer.FormDataContentType()
}

func TestHandleWalletStartCheckAppleStoreMetadata(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	userA, err := app.FindAuthRecordByEmail("users", "userA@example.org")
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
			startFn: func(string, workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
				return workflowengine.WorkflowResult{WorkflowID: "wf", WorkflowRunID: "run"}, nil
			},
		}
	}
	walletTemporalClient = func(string) (client.Client, error) { return temporalmocks.NewClient(t), nil }
	walletWaitForPartialResult = func(client.Client, string, string, string, time.Duration, time.Duration) (map[string]any, error) {
		return map[string]any{
			"storeType": "apple",
			"metadata": map[string]any{
				"trackName":     "iWallet",
				"artworkUrl100": "icon.png",
				"bundleId":      "com.apple.wallet",
				"sellerUrl":     "https://seller.example",
				"description":   "d",
			},
		}, nil
	}

	status, body, raw := covRunnerWalletRequest(
		t, app, userA, HandleWalletStartCheck(), "application/json",
		[]byte(`{"walletURL":"https://apps.apple.com/app/id1"}`),
	)
	require.Equal(t, http.StatusOK, status, raw)
	assert.Equal(t, "apple", body["type"])
	assert.Equal(t, "iWallet", body["name"])
	assert.Equal(t, "icon.png", body["logo"])
	assert.Equal(t, "com.apple.wallet", body["apple_app_id"])
	assert.Equal(t, "https://seller.example", body["home_url"])
	assert.Equal(t, "https://apps.apple.com/app/id1", body["appstore_url"])
	assert.Empty(t, body["playstore_url"])
	assert.Empty(t, body["google_app_id"])
}
