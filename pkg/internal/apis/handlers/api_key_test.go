// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

const apiKeyHandlerTestDataDir = "../../../../test_pb_data"

type tokenStub struct {
	token string
	err   error
}

func (t tokenStub) NewAuthToken() (string, error) {
	if t.err != nil {
		return "", t.err
	}
	return t.token, nil
}

type apiKeyGenerationServiceStub struct {
	userKey              string
	internalAdminKey     string
	userErr              error
	internalAdminErr     error
	userCalls            int
	internalAdminCalls   int
	lastUserID           string
	lastSuperuserID      string
	lastGeneratedKeyName string
}

func (s *apiKeyGenerationServiceStub) GenerateApiKey(userID, name string) (string, error) {
	s.userCalls++
	s.lastUserID = userID
	s.lastGeneratedKeyName = name
	if s.userErr != nil {
		return "", s.userErr
	}
	return s.userKey, nil
}

func (s *apiKeyGenerationServiceStub) GenerateInternalAdminAPIKey(
	superuserID, name string,
) (string, error) {
	s.internalAdminCalls++
	s.lastSuperuserID = superuserID
	s.lastGeneratedKeyName = name
	if s.internalAdminErr != nil {
		return "", s.internalAdminErr
	}
	return s.internalAdminKey, nil
}

func TestGenerateAuthenticateApiKeyResponse(t *testing.T) {
	t.Parallel()

	response, err := generateAuthenticateApiKeyResponse(tokenStub{token: "token-1"})
	require.NoError(t, err)
	require.Equal(t, "token-1", response.Token)

	_, err = generateAuthenticateApiKeyResponse(tokenStub{err: errors.New("boom")})
	require.Error(t, err)
}

func TestGenerateApiKeyInvalidInput(t *testing.T) {
	t.Parallel()

	handler := GenerateApiKey()
	req := httptest.NewRequest(http.MethodPost, "/api/apikey/generate", nil)
	req = req.WithContext(context.Background())
	rec := httptest.NewRecorder()

	auth := core.NewRecord(core.NewAuthCollection("users"))
	auth.Id = "user-1"

	e := &core.RequestEvent{
		Auth: auth,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}

	err := handler(e)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "name_required")
}

func TestGenerateAPIKeyForPrincipalUser(t *testing.T) {
	t.Parallel()

	stub := &apiKeyGenerationServiceStub{userKey: "user-key"}
	auth := core.NewRecord(core.NewAuthCollection("users"))
	auth.Id = "user-1"

	key, err := generateAPIKeyForPrincipal(stub, auth, "key-name")
	require.NoError(t, err)
	require.Equal(t, "user-key", key)
	require.Equal(t, 1, stub.userCalls)
	require.Equal(t, 0, stub.internalAdminCalls)
	require.Equal(t, "user-1", stub.lastUserID)
	require.Equal(t, "key-name", stub.lastGeneratedKeyName)
}

func TestGenerateAPIKeyForPrincipalSuperuser(t *testing.T) {
	t.Parallel()

	stub := &apiKeyGenerationServiceStub{internalAdminKey: "internal-key"}
	auth := core.NewRecord(core.NewAuthCollection("_superusers"))
	auth.Id = "superuser-1"

	key, err := generateAPIKeyForPrincipal(stub, auth, "key-name")
	require.NoError(t, err)
	require.Equal(t, "internal-key", key)
	require.Equal(t, 0, stub.userCalls)
	require.Equal(t, 1, stub.internalAdminCalls)
	require.Equal(t, "superuser-1", stub.lastSuperuserID)
	require.Equal(t, "key-name", stub.lastGeneratedKeyName)
}

func TestGenerateAPIKeyForPrincipalUnsupportedCollection(t *testing.T) {
	t.Parallel()

	stub := &apiKeyGenerationServiceStub{}
	auth := core.NewRecord(core.NewAuthCollection("admins"))
	auth.Id = "admin-1"

	_, err := generateAPIKeyForPrincipal(stub, auth, "key-name")
	require.Error(t, err)

	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusForbidden, apiErr.Code)
	require.Equal(t, "unsupported_auth_collection", apiErr.Reason)
	require.Equal(t, 0, stub.userCalls)
	require.Equal(t, 0, stub.internalAdminCalls)
}

func TestAuthenticateInternalAdminAPIKey(t *testing.T) {
	app, err := tests.NewTestApp(apiKeyHandlerTestDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	t.Run("valid internal admin key", func(t *testing.T) {
		seedInternalAdminKey(t, app)
		req := httptest.NewRequest(http.MethodGet, "/api/apikey/authenticate-internal-admin", nil)
		req.Header.Set("Credimi-Api-Key", "internal-test-api-key")
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}

		nextCalled := false
		setHandlerNext(e, func() error {
			nextCalled = true
			return nil
		})
		require.NoError(t, middlewares.RequireInternalAdminAPIKey().Func(e))
		require.True(t, nextCalled)

		require.NoError(t, AuthenticateInternalAdminAPIKey()(e))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), "Internal admin API key authenticated successfully")
	})

	t.Run("user scoped key is forbidden", func(t *testing.T) {
		user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
		require.NoError(t, err)
		seedHandlerAPIKey(t, app, handlerAPIKeySeed{
			Plaintext: "user-api-key",
			UserID:    user.Id,
			Scope:     "user",
		})

		req := httptest.NewRequest(http.MethodGet, "/api/apikey/authenticate-internal-admin", nil)
		req.Header.Set("Credimi-Api-Key", "user-api-key")
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}

		require.NoError(t, middlewares.RequireInternalAdminAPIKey().Func(e))
		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Contains(t, rec.Body.String(), "insufficient_api_key_scope")
	})

	t.Run("revoked internal admin key is unauthorized", func(t *testing.T) {
		superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
		require.NoError(t, err)
		seedHandlerAPIKey(t, app, handlerAPIKeySeed{
			Plaintext:   "revoked-internal-api-key",
			SuperuserID: superuser.Id,
			Scope:       "internal_admin",
			Revoked:     true,
		})

		req := httptest.NewRequest(http.MethodGet, "/api/apikey/authenticate-internal-admin", nil)
		req.Header.Set("Credimi-Api-Key", "revoked-internal-api-key")
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}

		require.NoError(t, middlewares.RequireInternalAdminAPIKey().Func(e))
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Contains(t, rec.Body.String(), "revoked_api_key")
	})

	t.Run("expired internal admin key is unauthorized", func(t *testing.T) {
		superuser, err := app.FindAuthRecordByEmail("_superusers", "admin@example.org")
		require.NoError(t, err)
		expiresAt := time.Now().Add(-time.Hour)
		seedHandlerAPIKey(t, app, handlerAPIKeySeed{
			Plaintext:   "expired-internal-api-key",
			SuperuserID: superuser.Id,
			Scope:       "internal_admin",
			ExpiresAt:   &expiresAt,
		})

		req := httptest.NewRequest(http.MethodGet, "/api/apikey/authenticate-internal-admin", nil)
		req.Header.Set("Credimi-Api-Key", "expired-internal-api-key")
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}

		require.NoError(t, middlewares.RequireInternalAdminAPIKey().Func(e))
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Contains(t, rec.Body.String(), "expired_api_key")
	})

	t.Run("missing key is unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/apikey/authenticate-internal-admin", nil)
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}

		require.NoError(t, middlewares.RequireInternalAdminAPIKey().Func(e))
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Contains(t, rec.Body.String(), "api_key_required")
	})
}

func setHandlerNext(e *core.RequestEvent, fn func() error) {
	eventField := reflect.ValueOf(e).Elem().FieldByName("Event")
	hookEvent := eventField.FieldByName("Event")
	nextField := hookEvent.FieldByName("next")
	reflect.NewAt(nextField.Type(), unsafe.Pointer(nextField.UnsafeAddr())).Elem().
		Set(reflect.ValueOf(fn))
}

type handlerAPIKeySeed struct {
	Plaintext   string
	UserID      string
	SuperuserID string
	Scope       string
	Revoked     bool
	ExpiresAt   *time.Time
}

func seedHandlerAPIKey(t testing.TB, app *tests.TestApp, seed handlerAPIKeySeed) {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId("api_keys")
	require.NoError(t, err)
	hash, err := bcrypt.GenerateFromPassword([]byte(seed.Plaintext), bcrypt.MinCost)
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("name", "handler-test-key")
	record.Set("key", string(hash))
	record.Set("user", seed.UserID)
	record.Set("superuser", seed.SuperuserID)
	record.Set("key_type", seed.Scope)
	record.Set("revoked", seed.Revoked)
	if seed.ExpiresAt != nil {
		record.Set("expires_at", seed.ExpiresAt.UTC().Format("2006-01-02 15:04:05.000Z"))
	}
	require.NoError(t, app.Save(record))
}
