// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func TestHandleIdentifierValidateSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	org, err := app.FindFirstRecordByFilter("organizations", `name="userA's organization"`)
	require.NoError(t, err)

	tpl := canonify.CanonifyPaths["organizations"]
	path, err := canonify.BuildPath(app, org, tpl, "")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/canonify/identifier/validate", nil)
	req = req.WithContext(
		context.WithValue(req.Context(), middlewares.ValidatedInputKey, IdentifierValidateRequest{
			CanonifiedName: path,
		}),
	)
	rec := httptest.NewRecorder()

	err = HandleIdentifierValidate()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "valid identifier")
}

func TestHandleGetIdentifierMissingParams(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/canonify/identifier/get", nil)
	rec := httptest.NewRecorder()

	err = HandleGetIdentifier()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleGetIdentifierSuccess(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	org, err := app.FindFirstRecordByFilter("organizations", `name="userA's organization"`)
	require.NoError(t, err)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/canonify/identifier/get?collection=organizations&id="+org.Id,
		nil,
	)
	rec := httptest.NewRecorder()

	err = HandleGetIdentifier()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "\"identifier\"")
}
