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

func TestHandleGetConformanceCheckDeeplinkMissingID(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/conformance-check/deeplink", nil)
	rec := httptest.NewRecorder()

	err = HandleGetConformanceCheckDeeplink()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var apiErr apierror.APIError
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&apiErr))
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "missing check id", apiErr.Reason)
}

func TestHandleGetConformanceCheckDeeplinkInvalidCheckName(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/conformance-check/deeplink?id=bad%20name",
		nil,
	)
	rec := httptest.NewRecorder()

	err = HandleGetConformanceCheckDeeplink()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var apiErr apierror.APIError
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&apiErr))
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "invalid check name", apiErr.Reason)
}

func TestHandleGetConformanceCheckDeeplinkPathTraversal(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/conformance-check/deeplink?id=openid4vp/draft-24/..",
		nil,
	)
	rec := httptest.NewRecorder()

	err = HandleGetConformanceCheckDeeplink()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var apiErr apierror.APIError
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&apiErr))
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
}

func TestHandleGetConformanceCheckDeeplinkUnsupportedSuite(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/conformance-check/deeplink?id=openid4vp/unknown/check-1",
		nil,
	)
	rec := httptest.NewRecorder()

	err = HandleGetConformanceCheckDeeplink()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var apiErr apierror.APIError
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&apiErr))
	require.Equal(t, http.StatusBadRequest, apiErr.Code)
	require.Equal(t, "unsupported suite", apiErr.Reason)
}
