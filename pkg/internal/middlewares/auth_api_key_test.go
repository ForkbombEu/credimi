// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package middlewares

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
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
		requireAPIErrorResponse(
			t,
			rec,
			http.StatusUnauthorized,
			"authentication_required",
			"Bearer token or Credimi-Api-Key is required",
		)
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
		requireAPIErrorResponse(
			t,
			rec,
			http.StatusUnauthorized,
			"invalid_api_key",
			"Invalid API key provided",
		)
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
		requireAPIErrorResponse(
			t,
			rec,
			http.StatusUnauthorized,
			"api_key_required",
			"Credimi-Api-Key is required",
		)
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
		requireAPIErrorResponse(
			t,
			rec,
			http.StatusForbidden,
			"insufficient_api_key_scope",
			"API key does not have required scope",
		)
	})

	t.Run("rejects internal admin scope owned by a user", func(t *testing.T) {
		plaintext := "user-owned-internal-admin-key"
		createAPIKeyRecord(t, app, apiKeyRecordInput{
			Plaintext: plaintext,
			UserID:    user.Id,
			Scope:     apiKeyScopeInternalAdmin,
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(apiKeyHeaderName, plaintext)
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}

		err := RequireInternalAdminAPIKey().Func(e)
		require.Error(t, err)
		require.Nil(t, e.Auth)
		writeMiddlewareErrorResponse(t, e, err)
		requireAPIErrorResponse(
			t,
			rec,
			http.StatusForbidden,
			"insufficient_api_key_scope",
			"API key does not have required scope",
		)
	})

	t.Run("accepts internal admin scope owned by a superuser", func(t *testing.T) {
		superuser, err := app.FindAuthRecordByEmail(
			core.CollectionNameSuperusers,
			"admin@example.org",
		)
		require.NoError(t, err)
		plaintext := "superuser-internal-admin-key"
		createAPIKeyRecord(t, app, apiKeyRecordInput{
			Plaintext:   plaintext,
			SuperuserID: superuser.Id,
			Scope:       apiKeyScopeInternalAdmin,
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(apiKeyHeaderName, plaintext)
		rec := httptest.NewRecorder()
		e := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: rec}}

		require.NoError(t, RequireInternalAdminAPIKey().Func(e))
		require.NotNil(t, e.Auth)
		require.True(t, e.Auth.IsSuperuser())
		require.Equal(t, superuser.Id, e.Auth.Id)
	})
}

func TestAPIKeysCollectionRules(t *testing.T) {
	app, err := tests.NewTestApp(middlewareTestDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	victim, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	attacker, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	victimToken, err := victim.NewAuthToken()
	require.NoError(t, err)
	attackerToken, err := attacker.NewAuthToken()
	require.NoError(t, err)

	const victimSecret = "victim-secret"
	const attackerSecret = "attacker-secret"
	victimKey := createAPIKeyRecord(t, app, apiKeyRecordInput{
		Plaintext: victimSecret,
		UserID:    victim.Id,
		Scope:     apiKeyScopeUser,
	})
	attackerHash, err := bcrypt.GenerateFromPassword([]byte(attackerSecret), bcrypt.MinCost)
	require.NoError(t, err)

	mux := buildMiddlewareTestMux(t, app)
	recordsURL := "/api/collections/api_keys/records"
	victimKeyURL := recordsURL + "/" + victimKey.Id
	listVictimKeysURL := recordsURL + "?filter=" + url.QueryEscape("user='"+victim.Id+"'")

	t.Run("attacker cannot list the victim's keys", func(t *testing.T) {
		rec := serveCollectionRequest(t, mux, http.MethodGet, listVictimKeysURL, attackerToken, "")
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, 0, decodeListItems(t, rec).TotalItems)
	})

	t.Run("attacker cannot view the victim's key", func(t *testing.T) {
		rec := serveCollectionRequest(t, mux, http.MethodGet, victimKeyURL, attackerToken, "")
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("attacker cannot create an internal admin key", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{
			"name":     "x",
			"key":      string(attackerHash),
			"user":     attacker.Id,
			"key_type": apiKeyScopeInternalAdmin,
		})
		require.NoError(t, err)
		rec := serveCollectionRequest(
			t,
			mux,
			http.MethodPost,
			recordsURL,
			attackerToken,
			string(body),
		)
		require.Equal(t, http.StatusForbidden, rec.Code)

		_, apiErr := authenticateAPIKeyByScope(app, attackerSecret, apiKeyScopeInternalAdmin)
		require.NotNil(t, apiErr)
		require.Equal(t, http.StatusUnauthorized, apiErr.Code)
		require.Equal(t, "invalid_api_key", apiErr.Reason)
	})

	t.Run("attacker cannot overwrite the victim's key hash", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{"key": string(attackerHash)})
		require.NoError(t, err)
		rec := serveCollectionRequest(
			t,
			mux,
			http.MethodPatch,
			victimKeyURL,
			attackerToken,
			string(body),
		)
		require.Equal(t, http.StatusForbidden, rec.Code)

		stored, err := app.FindRecordById("api_keys", victimKey.Id)
		require.NoError(t, err)
		require.NoError(t, bcrypt.CompareHashAndPassword(
			[]byte(stored.GetString("key")),
			[]byte(victimSecret),
		))
		_, apiErr := authenticateAPIKeyByScope(app, attackerSecret, apiKeyScopeUser)
		require.NotNil(t, apiErr)
		require.Equal(t, "invalid_api_key", apiErr.Reason)
	})

	t.Run("attacker cannot delete the victim's key", func(t *testing.T) {
		rec := serveCollectionRequest(t, mux, http.MethodDelete, victimKeyURL, attackerToken, "")
		require.Equal(t, http.StatusNotFound, rec.Code)

		_, err := app.FindRecordById("api_keys", victimKey.Id)
		require.NoError(t, err)
	})

	t.Run("owner cannot modify their own key", func(t *testing.T) {
		body, err := json.Marshal(map[string]any{"key_type": apiKeyScopeInternalAdmin})
		require.NoError(t, err)
		rec := serveCollectionRequest(
			t,
			mux,
			http.MethodPatch,
			victimKeyURL,
			victimToken,
			string(body),
		)
		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("owner lists their own keys without the key hash", func(t *testing.T) {
		rec := serveCollectionRequest(t, mux, http.MethodGet, listVictimKeysURL, victimToken, "")
		require.Equal(t, http.StatusOK, rec.Code)
		list := decodeListItems(t, rec)
		require.Equal(t, 1, list.TotalItems)
		require.Equal(t, victimKey.Id, list.Items[0]["id"])
		require.NotContains(t, list.Items[0], "key")
	})

	t.Run("owner revokes their own key", func(t *testing.T) {
		rec := serveCollectionRequest(t, mux, http.MethodDelete, victimKeyURL, victimToken, "")
		require.Equal(t, http.StatusNoContent, rec.Code)

		_, err := app.FindRecordById("api_keys", victimKey.Id)
		require.Error(t, err)
	})
}

type collectionListResponse struct {
	TotalItems int              `json:"totalItems"`
	Items      []map[string]any `json:"items"`
}

func decodeListItems(t *testing.T, rec *httptest.ResponseRecorder) collectionListResponse {
	t.Helper()
	var list collectionListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	return list
}

func serveCollectionRequest(
	t *testing.T,
	mux http.Handler,
	method string,
	target string,
	token string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", token)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// buildMiddlewareTestMux builds the app router once per app instance, because
// apis.NewRouter binds routes that cannot be registered twice.
func buildMiddlewareTestMux(t *testing.T, app *tests.TestApp) http.Handler {
	t.Helper()
	baseRouter, err := apis.NewRouter(app)
	require.NoError(t, err)

	var mux http.Handler
	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	require.NoError(t, app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		var err error
		mux, err = e.Router.BuildMux()
		return err
	}))
	require.NotNil(t, mux)
	return mux
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
		requireAPIErrorResponse(
			t,
			rec,
			http.StatusUnauthorized,
			"invalid_api_key",
			"Invalid API key provided",
		)
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

func createAPIKeyRecord(t *testing.T, app *tests.TestApp, input apiKeyRecordInput) *core.Record {
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
	return record
}
