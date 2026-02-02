// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func decodeAPIError(t testing.TB, rec *httptest.ResponseRecorder) apierror.APIError {
	t.Helper()
	var apiErr apierror.APIError
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&apiErr))
	return apiErr
}

func TestHandleGetMyCheckRunRequiresAuth(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/my/checks/checks-1/runs/run-1",
		nil,
	)
	req.SetPathValue("checkId", "checks-1")
	req.SetPathValue("runId", "run-1")
	rec := httptest.NewRecorder()

	err = HandleGetMyCheckRun()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusUnauthorized, apiErr.Code)
	require.Equal(t, "authentication required", apiErr.Reason)
}

func TestHandleGetMyCheckRunMissingRunID(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/my/checks/checks-1/runs/",
		nil,
	)
	req.SetPathValue("checkId", "checks-1")
	rec := httptest.NewRecorder()

	err = HandleGetMyCheckRun()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "runId is required", apiErr.Reason)
}

func TestHandleListMyCheckRunsMissingCheckID(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks//runs", nil)
	rec := httptest.NewRecorder()

	err = HandleListMyCheckRuns()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	apiErr := decodeAPIError(t, rec)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "checkId is required", apiErr.Reason)
}

func TestHandleGetMyCheckRunHistoryMissingParams(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	authRecord, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/my/checks/checks-1/runs/", nil)
	req.SetPathValue("checkId", "")
	req.SetPathValue("runId", "")
	ar := httptest.NewRecorder()

	err = HandleGetMyCheckRunHistory()(&core.RequestEvent{
		App:  app,
		Auth: authRecord,
		Event: router.Event{
			Request:  req,
			Response: ar,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, ar.Code)

	apiErr := decodeAPIError(t, ar)
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "checkId and runId are required", apiErr.Reason)
}
