// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
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
				"request contract validated",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := setupPipelineWalletAPKApp(t)
				seedUserAPIKey(t, app, walletAPKUserAPIKey)
				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
