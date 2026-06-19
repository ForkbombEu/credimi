// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"encoding/json"
	"io"
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
		require.Error(t, err)
		writeMiddlewareErrorResponse(t, e, err)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		requireAPIErrorResponse(t, rec, http.StatusUnauthorized, "authentication_required", "Bearer token or Credimi-Api-Key is required")
	})

	t.Run("invalid api key returns unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(apiKeyHeaderName, "not-a-real-key")
		rec := httptest.NewRecorder()

		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}
		err := RequireAuthOrAPIKey().Func(e)
		require.Error(t, err)
		writeMiddlewareErrorResponse(t, e, err)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		requireAPIErrorResponse(t, rec, http.StatusUnauthorized, "invalid_api_key", "Invalid API key provided")
	})

	t.Run("bearer takes precedence over api key fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()

		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}
		err := RequireAuthOrAPIKey().Func(e)
		require.Error(t, err)
	})

	t.Run("authorization scheme match is case sensitive", func(t *testing.T) {
		token, err := user.NewAuthToken()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "bearer "+token)
		rec := httptest.NewRecorder()

		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}
		err = RequireAuthOrAPIKey().Func(e)
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
		require.Error(t, err)
		writeMiddlewareErrorResponse(t, e, err)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		requireAPIErrorResponse(t, rec, http.StatusUnauthorized, "api_key_required", "Credimi-Api-Key is required")
	})

	t.Run("rejects key without explicit internal admin scope", func(t *testing.T) {
		plaintext := "legacy-user-key"
		createAPIKeyRecord(t, app, apiKeyRecordInput{
			Plaintext: plaintext,
			UserID:    user.Id,
			Scope:     "",
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(apiKeyHeaderName, plaintext)
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}

		err := RequireInternalAdminAPIKey().Func(e)
		require.Error(t, err)
		writeMiddlewareErrorResponse(t, e, err)
		require.Equal(t, http.StatusForbidden, rec.Code)
		requireAPIErrorResponse(t, rec, http.StatusForbidden, "insufficient_api_key_scope", "API key does not have required scope")
	})

}

func TestOptionalAuthOrAPIKey(t *testing.T) {
	app, err := tests.NewTestApp(middlewareTestDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)

	t.Run("missing api key continues anonymously", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}

		nextCalled := false
		setNext(e, func() error {
			nextCalled = true
			return nil
		})

		err := OptionalAuthOrAPIKey().Func(e)
		require.NoError(t, err)
		require.True(t, nextCalled)
		require.Nil(t, e.Auth)
	})

	t.Run("valid user api key sets auth", func(t *testing.T) {
		plaintext := "optional-user-api-key"
		createAPIKeyRecord(t, app, apiKeyRecordInput{
			Plaintext: plaintext,
			UserID:    user.Id,
			Scope:     apiKeyScopeUser,
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(apiKeyHeaderName, plaintext)
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}

		nextCalled := false
		setNext(e, func() error {
			nextCalled = true
			return nil
		})

		err := OptionalAuthOrAPIKey().Func(e)
		require.NoError(t, err)
		require.True(t, nextCalled)
		require.NotNil(t, e.Auth)
		require.Equal(t, user.Id, e.Auth.Id)
	})

	t.Run("invalid api key is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(apiKeyHeaderName, "not-a-real-key")
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}

		nextCalled := false
		setNext(e, func() error {
			nextCalled = true
			return nil
		})

		err := OptionalAuthOrAPIKey().Func(e)
		require.Error(t, err)
		writeMiddlewareErrorResponse(t, e, err)
		require.False(t, nextCalled)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		requireAPIErrorResponse(t, rec, http.StatusUnauthorized, "invalid_api_key", "Invalid API key provided")
	})
}

func writeMiddlewareErrorResponse(t *testing.T, e *core.RequestEvent, err error) {
	t.Helper()

	setNext(e, func() error { return err })
	require.NoError(t, ErrorHandlingMiddleware(e))
}

func requireAPIErrorResponse(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	code int,
	reason string,
	message string,
) {
	t.Helper()

	requireAPIErrorResponseFromReader(t, rec.Body, code, "request.validation", reason, message)
}

func requireAPIErrorResponseFromReader(
	t *testing.T,
	reader io.Reader,
	code int,
	domain string,
	reason string,
	message string,
) {
	t.Helper()

	var body errorMiddlewareResponse
	require.NoError(t, json.NewDecoder(reader).Decode(&body))
	require.Equal(t, "2.0", body.APIVersion)
	require.Equal(t, message, body.Message)
	require.Equal(t, code, body.Error.Code)
	require.Equal(t, domain, body.Error.Domain)
	require.Equal(t, reason, body.Error.Reason)
	require.Equal(t, message, body.Error.Message)
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
