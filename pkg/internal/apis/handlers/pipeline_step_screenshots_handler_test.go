// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"testing"

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
			ensureStepScreenshotField(t, app)
			PipelineTemporalInternalRoutes.Add(app)
			setupWalletPipelineTestRecords(t, app, orgID)
			reserveStepScreenshotDevice(t, app, orgID)
			return app
		},
	}
	scenario.Test(t)
}

func reserveStepScreenshotDevice(t testing.TB, app *tests.TestApp, orgID string) {
	t.Helper()
	ensureMobileDevicesCollection(t, app)
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

func ensureStepScreenshotField(t testing.TB, app *tests.TestApp) {
	t.Helper()
	ensureMobileDevicesCollection(t, app)
	collection, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)
	devices, err := app.FindCollectionByNameOrId("mobile_devices")
	require.NoError(t, err)
	if collection.Fields.GetByName("maestro_screenshots") == nil {
		collection.Fields.Add(&core.FileField{Name: "maestro_screenshots", MaxSelect: 99})
	}
	if collection.Fields.GetByName("devices") == nil {
		collection.Fields.Add(
			&core.RelationField{Name: "devices", CollectionId: devices.Id, MaxSelect: 999},
		)
	}
	require.NoError(t, app.Save(collection))
}
