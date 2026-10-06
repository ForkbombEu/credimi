// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestStorePipelineStepScreenshots(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField(
		"run_identifier",
		"usera-s-organization/workflow123-run123",
	))
	require.NoError(t, writer.WriteField(
		"device_identifier",
		"usera-s-organization/test-runner/test-device",
	))
	require.NoError(t, writer.WriteField("step_id", "scan credential"))
	first, err := writer.CreateFormFile("screenshots", "checkout.png")
	require.NoError(t, err)
	_, err = first.Write([]byte("checkout screenshot"))
	require.NoError(t, err)
	second, err := writer.CreateFormFile("screenshots", "confirmation.png")
	require.NoError(t, err)
	_, err = second.Write([]byte("confirmation screenshot"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	scenario := tests.ApiScenario{
		Name:   "stores screenshots with step-prefixed names",
		Method: http.MethodPost,
		URL:    "/api/pipeline/store-step-screenshots",
		Body:   bytes.NewReader(body.Bytes()),
		Headers: map[string]string{
			"Content-Type":    writer.FormDataContentType(),
			"Credimi-Api-Key": "internal-test-api-key",
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"status":"success"`,
			`"step_id":"scan-credential"`,
			`scan_credential_checkout_`,
			`scan_credential_confirmation_`,
			`"screenshot_urls"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupWalletApp(t)
			PipelineTemporalInternalRoutes.Add(app)
			setupWalletPipelineTestRecords(t, app, orgID)
			reserveStepScreenshotDevice(t, app, orgID)
			return app
		},
	}
	scenario.Test(t)
}

// A complete FCAF validation stores a few screenshots for each of ~200 mobile
// steps on the same run record, so a run must hold well over 99 of them.
func TestStorePipelineStepScreenshotsBeyondNinetyNinePerRun(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField(
		"run_identifier",
		"usera-s-organization/workflow123-run123",
	))
	require.NoError(t, writer.WriteField(
		"device_identifier",
		"usera-s-organization/test-runner/test-device",
	))
	require.NoError(t, writer.WriteField("step_id", "present credential"))
	for i := range 100 {
		file, err := writer.CreateFormFile("screenshots", fmt.Sprintf("step-%03d.png", i))
		require.NoError(t, err)
		_, err = file.Write([]byte("screenshot"))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	scenario := tests.ApiScenario{
		Name:   "stores the 100th screenshot of a run",
		Method: http.MethodPost,
		URL:    "/api/pipeline/store-step-screenshots",
		Body:   bytes.NewReader(body.Bytes()),
		Headers: map[string]string{
			"Content-Type":    writer.FormDataContentType(),
			"Credimi-Api-Key": "internal-test-api-key",
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"status":"success"`,
			`present_credential_step_099_`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := setupWalletApp(t)
			PipelineTemporalInternalRoutes.Add(app)
			setupWalletPipelineTestRecords(t, app, orgID)
			reserveStepScreenshotDevice(t, app, orgID)
			return app
		},
	}
	scenario.Test(t)
}

func TestStorePipelineStepScreenshotsPublishedRunnerOwner(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)
	runnerUser, err := getUserRecordFromName("userB")
	require.NoError(t, err)
	runnerUserToken, err := runnerUser.NewAuthToken()
	require.NoError(t, err)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField(
		"run_identifier",
		"usera-s-organization/workflow123-run123",
	))
	require.NoError(t, writer.WriteField(
		"device_identifier",
		"userb-s-organization/public-runner/public-device",
	))
	require.NoError(t, writer.WriteField("step_id", "scan credential"))
	file, err := writer.CreateFormFile("screenshots", "checkout.png")
	require.NoError(t, err)
	_, err = file.Write([]byte("checkout screenshot"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	setupApp := func(reserved bool) func(t testing.TB) *tests.TestApp {
		return func(t testing.TB) *tests.TestApp {
			app := setupWalletApp(t)
			PipelineTemporalInternalRoutes.Add(app)
			setupWalletPipelineTestRecords(t, app, orgID)
			setOrganizationPublished(t, app, orgID, true)
			runnerOrgID, err := pbutils.GetUserOrganizationID(app, runnerUser.Id)
			require.NoError(t, err)
			runner := createWalletTestMobileRunner(t, app, runnerOrgID, "public-runner", true)
			device := createWalletTestMobileDevice(t, app, runnerOrgID, runner.Id, "public-device")
			if reserved {
				addWalletPipelineResultDevice(t, app, device.Id)
			}
			return app
		}
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "published runner owner stores screenshots from a reserved device",
			Method: http.MethodPost,
			URL:    "/api/pipeline/store-step-screenshots",
			Body:   bytes.NewReader(body.Bytes()),
			Headers: map[string]string{
				"Authorization": "Bearer " + runnerUserToken,
				"Content-Type":  writer.FormDataContentType(),
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"success"`,
				`scan_credential_checkout_`,
			},
			TestAppFactory: setupApp(true),
		},
		{
			Name:   "published runner owner cannot store screenshots from an unreserved device",
			Method: http.MethodPost,
			URL:    "/api/pipeline/store-step-screenshots",
			Body:   bytes.NewReader(body.Bytes()),
			Headers: map[string]string{
				"Authorization": "Bearer " + runnerUserToken,
				"Content-Type":  writer.FormDataContentType(),
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"forbidden"`,
				`device did not run this pipeline result`,
			},
			NotExpectedContent: []string{`"status":"success"`},
			TestAppFactory:     setupApp(false),
		},
	}
	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func reserveStepScreenshotDevice(t testing.TB, app *tests.TestApp, orgID string) {
	t.Helper()
	runners, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	runner := core.NewRecord(runners)
	runner.Set("owner", orgID)
	runner.Set("name", "test-runner")
	runner.Set("ip", "https://runner.example.test")
	runner.Set("type", "android_emulator")
	require.NoError(t, app.Save(runner))
	devices, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	device := core.NewRecord(devices)
	device.Set("owner", orgID)
	device.Set("runner", runner.Id)
	device.Set("name", "test-device")
	device.Set("canonified_name", "test-device")
	device.Set("type", "android_emulator")
	require.NoError(t, app.Save(device))
	result, err := app.FindFirstRecordByFilter(
		"pipeline_results",
		"workflow_id = 'workflow123' && run_id = 'run123'",
	)
	require.NoError(t, err)
	result.Set("devices", []string{device.Id})
	require.NoError(t, app.Save(result))
}
