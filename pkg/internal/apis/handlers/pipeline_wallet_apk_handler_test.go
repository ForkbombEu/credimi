// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const walletAPKUserAPIKey = "wallet-apk-user-api-key"

func setupPipelineWalletAPKApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)

	canonify.RegisterCanonifyHooks(app)
	PipelineRoutes.Add(app)

	return app
}

func seedUserAPIKey(t testing.TB, app *tests.TestApp, plaintext string) {
	t.Helper()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	coll, err := app.FindCollectionByNameOrId("api_keys")
	require.NoError(t, err)
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("name", "wallet-apk-user-key")
	record.Set("key", string(hash))
	record.Set("user", user.Id)
	record.Set("superuser", "")
	record.Set("key_type", "user")
	record.Set("revoked", false)
	require.NoError(t, app.Save(record))
}

func createWalletAPITestPipeline(
	t testing.TB,
	app *tests.TestApp,
	orgID string,
	yaml string,
) *core.Record {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("owner", orgID)
	record.Set("name", "pipeline123")
	record.Set("description", "test pipeline")
	record.Set("steps", map[string]any{"rest-chain": map[string]any{"yaml": yaml}})
	record.Set("yaml", yaml)
	require.NoError(t, app.Save(record))

	return record
}

func blankWalletAPITestPipelineYAML(t testing.TB, app *tests.TestApp, pipelineID string) {
	t.Helper()

	_, err := app.DB().NewQuery(
		`UPDATE pipelines SET yaml = '' WHERE id = {:id}`,
	).Bind(dbx.Params{"id": pipelineID}).Execute()
	require.NoError(t, err)
}

func walletAPKMultipartBody(
	t testing.TB,
	fields map[string]string,
	includeFile bool,
) (*bytes.Reader, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		require.NoError(t, writer.WriteField(key, value))
	}
	if includeFile {
		fileWriter, err := writer.CreateFormFile(walletAPKFormFileField, "wallet.apk")
		require.NoError(t, err)
		_, err = fileWriter.Write([]byte("apk"))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	return bytes.NewReader(body.Bytes()), writer.FormDataContentType()
}

func TestPipelineRunWalletAPKRequestContract(t *testing.T) {
	validJSON := func() *bytes.Reader {
		return jsonBody(map[string]any{
			"pipeline_identifier": "usera-s-organization/pipeline123",
			"commit_sha":          "abc123",
			"apk_url":             "http://ci.example.test/wallet.apk",
		})
	}
	userKeyHeaders := map[string]string{
		"Content-Type":    "application/json",
		"Credimi-Api-Key": walletAPKUserAPIKey,
	}

	bothBody, bothContentType := walletAPKMultipartBody(t, map[string]string{
		"pipeline_identifier": "usera-s-organization/pipeline123",
		"commit_sha":          "abc123",
		"apk_url":             "http://ci.example.test/wallet.apk",
	}, true)

	scenarios := []tests.ApiScenario{
		{
			Name:   "requires auth",
			Method: http.MethodPost,
			URL:    "/api/pipeline/run-wallet-apk",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body:           validJSON(),
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				"authentication_required",
			},
			TestAppFactory: setupPipelineWalletAPKApp,
		},
		{
			Name:    "requires pipeline identifier",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"commit_sha": "abc123",
				"apk_url":    "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"pipeline_identifier",
				"missing pipeline_identifier",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedUserAPIKey(t, app, walletAPKUserAPIKey)
				return app
			},
		},
		{
			Name:    "requires commit sha",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"apk_url":             "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"commit_sha",
				"missing commit_sha",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedUserAPIKey(t, app, walletAPKUserAPIKey)
				return app
			},
		},
		{
			Name:    "requires one apk source",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"commit_sha":          "abc123",
			}),
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"apk_file or apk_url is required",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedUserAPIKey(t, app, walletAPKUserAPIKey)
				return app
			},
		},
		{
			Name:   "rejects both apk sources",
			Method: http.MethodPost,
			URL:    "/api/pipeline/run-wallet-apk",
			Headers: map[string]string{
				"Content-Type":    bothContentType,
				"Credimi-Api-Key": walletAPKUserAPIKey,
			},
			Body:           bothBody,
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"provide either apk_file or apk_url",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedUserAPIKey(t, app, walletAPKUserAPIKey)
				return app
			},
		},
		{
			Name:           "accepts user api key auth",
			Method:         http.MethodPost,
			URL:            "/api/pipeline/run-wallet-apk",
			Headers:        userKeyHeaders,
			Body:           validJSON(),
			ExpectedStatus: http.StatusNotImplemented,
			ExpectedContent: []string{
				"pipeline context validated",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedUserAPIKey(t, app, walletAPKUserAPIKey)
				orgID, err := getOrgIDfromName("userA's organization")
				require.NoError(t, err)
				createWalletAPITestPipeline(t, app, orgID, "name: test\nsteps: []\n")
				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestPipelineRunWalletAPKContextResolution(t *testing.T) {
	userKeyHeaders := map[string]string{
		"Content-Type":    "application/json",
		"Credimi-Api-Key": walletAPKUserAPIKey,
	}

	scenarios := []tests.ApiScenario{
		{
			Name:    "unknown pipeline",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/missing",
				"commit_sha":          "abc123",
				"apk_url":             "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				"pipeline not found",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedUserAPIKey(t, app, walletAPKUserAPIKey)
				return app
			},
		},
		{
			Name:    "pipeline yaml is required",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"commit_sha":          "abc123",
				"apk_url":             "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				"pipeline yaml is required",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedUserAPIKey(t, app, walletAPKUserAPIKey)
				orgID, err := getOrgIDfromName("userA's organization")
				require.NoError(t, err)
				pipeline := createWalletAPITestPipeline(t, app, orgID, "name: test\nsteps: []\n")
				blankWalletAPITestPipelineYAML(t, app, pipeline.Id)
				return app
			},
		},
		{
			Name:    "resolves namespace from user api key",
			Method:  http.MethodPost,
			URL:     "/api/pipeline/run-wallet-apk",
			Headers: userKeyHeaders,
			Body: jsonBody(map[string]any{
				"pipeline_identifier": "usera-s-organization/pipeline123",
				"commit_sha":          "abc123",
				"apk_url":             "http://ci.example.test/wallet.apk",
			}),
			ExpectedStatus: http.StatusNotImplemented,
			ExpectedContent: []string{
				"pipeline context validated",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedUserAPIKey(t, app, walletAPKUserAPIKey)
				orgID, err := getOrgIDfromName("userA's organization")
				require.NoError(t, err)
				createWalletAPITestPipeline(t, app, orgID, "name: test\nsteps: []\n")
				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestBuildPipelineRunWalletAPKResponse(t *testing.T) {
	enqueuedAt := time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC)
	position := 2
	lineLen := 5

	t.Run("queued response keeps queue fields and temp wallet metadata", func(t *testing.T) {
		response := buildPipelineRunWalletAPKResponse(
			PipelineQueueResponse{
				Status:     workflowengine.MobileRunnerSemaphoreRunQueued,
				TicketID:   "ticket-1",
				RunnerIDs:  []string{"runner-1"},
				EnqueuedAt: &enqueuedAt,
				Position:   &position,
				LineLen:    &lineLen,
			},
			"version-record-1",
			"usera-s-organization/wallet/abc123",
			"usera-s-organization/pipeline123",
		)

		require.Equal(t, workflowengine.MobileRunnerSemaphoreRunQueued, response.Status)
		require.Equal(t, "ticket-1", response.TicketID)
		require.Equal(t, []string{"runner-1"}, response.RunnerIDs)
		require.Equal(t, &position, response.Position)
		require.Equal(t, &lineLen, response.LineLen)
		require.Equal(t, "version-record-1", response.TempWalletVersionID)
		require.Equal(t, "usera-s-organization/wallet/abc123", response.TempWalletVersionIdentifier)
		require.Equal(t, "usera-s-organization/pipeline123", response.PipelineIdentifier)
	})

	t.Run("running response keeps workflow identifiers", func(t *testing.T) {
		response := buildPipelineRunWalletAPKResponse(
			PipelineQueueResponse{
				Status:     workflowengine.MobileRunnerSemaphoreRunRunning,
				WorkflowID: "workflow-1",
				RunID:      "run-1",
			},
			"version-record-1",
			"usera-s-organization/wallet/abc123",
			"usera-s-organization/pipeline123",
		)

		require.Equal(t, workflowengine.MobileRunnerSemaphoreRunRunning, response.Status)
		require.Equal(t, "workflow-1", response.WorkflowID)
		require.Equal(t, "run-1", response.RunID)
		require.Equal(t, "version-record-1", response.TempWalletVersionID)
	})
}
