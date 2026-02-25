// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const middlewareTestDataDir = "../../../test_pb_data"

func TestRequireAuthOrAPIKey_BearerAndFallbackContract(t *testing.T) {
	app, err := tests.NewTestApp(middlewareTestDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	t.Run("falls back to user api key when bearer absent", func(t *testing.T) {
		plaintext := "test-user-api-key"
		createAPIKeyRecord(t, app, apiKeyRecordInput{
			Plaintext: plaintext,
			UserID:    user.Id,
			Scope:     apiKeyScopeUser,
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(apiKeyHeaderName, plaintext)
		rec := httptest.NewRecorder()

		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}
		setNext(e, func() error { return nil })

		err := RequireAuthOrAPIKey().Func(e)
		require.NoError(t, err)
		require.Equal(t, user.Id, e.Auth.Id)
	})

	t.Run("missing both headers returns unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}
		err := RequireAuthOrAPIKey().Func(e)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid api key returns unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(apiKeyHeaderName, "not-a-real-key")
		rec := httptest.NewRecorder()

		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}
		err := RequireAuthOrAPIKey().Func(e)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("bearer takes precedence over api key fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()

		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}
		err := RequireAuthOrAPIKey().Func(e)
		require.Error(t, err)
	})
}

func TestRequireInternalAdminAPIKey(t *testing.T) {
	app, err := tests.NewTestApp(middlewareTestDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	t.Run("missing api key is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}
		err := RequireInternalAdminAPIKey().Func(e)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("accepts legacy user scoped key when internal scope metadata is absent", func(t *testing.T) {
		plaintext := "user-only-key"
		createAPIKeyRecord(t, app, apiKeyRecordInput{
			Plaintext: plaintext,
			UserID:    user.Id,
			Scope:     apiKeyScopeUser,
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(apiKeyHeaderName, plaintext)
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}

		err := RequireInternalAdminAPIKey().Func(e)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rec.Code)
	})

}

type apiKeyRecordInput struct {
	Plaintext   string
	UserID      string
	SuperuserID string
	Scope       string
	Revoked     bool
	ExpiresAt   *time.Time
}

func createAPIKeyRecord(t *testing.T, app *tests.TestApp, input apiKeyRecordInput) {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId("api_keys")
	require.NoError(t, err)

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Plaintext), bcrypt.DefaultCost)
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("name", "test-key")
	record.Set("key", string(hash))
	record.Set("user", input.UserID)
	record.Set("superuser", input.SuperuserID)
	record.Set("key_type", input.Scope)
	record.Set("revoked", input.Revoked)
	if input.ExpiresAt != nil {
		record.Set("expires_at", input.ExpiresAt.UTC().Format("2006-01-02 15:04:05.000Z"))
	}

	require.NoError(t, app.Save(record))
}
